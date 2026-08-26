package app

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

func branchCreateBaseForActiveSection(m model) string {
	switch m.activeSection {
	case sectionGraph:
		focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash == "" || focus.Hash == "VIRTUAL_CONFLICT_HASH" {
			return ""
		}
		return focus.Hash
	case sectionCurrent:
		return activeSectionTarget(m)
	default:
		return ""
	}
}

func startBranchCreateInput(m model, base string) (model, bool) {
	if err := branchCreateBaseValidationError(m.repoStatus, base); err != nil {
		m.status = branchCreateBlockedStatusFromError(err)
		return m, false
	}
	m.branchBase = base
	m.branchOpen = true
	m.branchDraft = ""
	m.status = loadingToast("Enter a branch name.")
	return m, true
}

func (m model) handleBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.awaitingGoTop && msg.String() != "g" {
		m.awaitingGoTop = false
	}
	if handled, nextM, cmd := m.handleBrowseGlobalKey(msg); handled {
		return nextM, cmd
	}
	switch m.activeSection {
	case sectionGraph:
		return m.handleBrowseGraphKey(msg)
	case sectionCurrent, sectionRemote, sectionTags:
		return m.handleBrowseSectionKey(msg)
	default:
		return m, nil
	}
}

func (m model) handleBrowseGlobalKey(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return true, m, tea.Quit
	case "1":
		m = switchBrowseSection(m, sectionGraph)
		return true, m, nil
	case "2":
		m = switchBrowseSection(m, sectionCurrent)
		return true, m, nil
	case "3":
		m = switchBrowseSection(m, sectionRemote)
		return true, m, nil
	case "4":
		m = switchBrowseSection(m, sectionTags)
		return true, m, nil
	case "f":
		m.status = operationLoadingStatusFor(progressFetchSources, "Fetching sources...", state.ActionNone)
		return true, m, fetchRepoState(m.repo, m.commitLimit)
	case "F":
		m.status = operationLoadingStatusFor(progressFetchTags, "Fetching tags...", state.ActionNone)
		return true, m, fetchTagsRepoState(m.repo, m.commitLimit, m.tagProvenance)
	case "P":
		if m.activeSection == sectionTags {
			item, ok := activeSectionTargetItem(m)
			if !ok || item.Ref == "" {
				m.status = state.New().WithBlocked(state.BlockTargetEmpty, "No tag selected.", "Choose a tag row.")
				return true, m, nil
			}
			m.status = operationLoadingStatusFor(progressPushTag, "Pushing tag...", state.ActionPushTag)
			return true, m, executePushTag(m.repo, item.Ref, m.commitLimit, m.tagProvenance)
		}
		if m.repoStatus.Root == "" || m.repoStatus.Detached || m.repoStatus.EmptyRepo {
			return true, m, nil
		}
		m.status = operationLoadingStatusFor(progressFetch, "Fetching for push...", state.ActionPush)
		return true, m, executeFetchForPush(m.repo, m.commitLimit)
	case "S":
		m.stashPopupOpen = true
		if m.stashPopupCursor < 0 {
			m.stashPopupCursor = 0
		}
		if maxCursor := len(m.stashEntries) - 1; maxCursor >= 0 && m.stashPopupCursor > maxCursor {
			m.stashPopupCursor = maxCursor
		}
		return true, m, nil
	case "?":
		m.hiddenHotkeysOpen = true
		m.hiddenHotkeysScroll = 0
		return true, m, nil
	case "tab":
		m.activeSection = nextGraphSection(m.activeSection)
		return true, m, nil
	case "shift+tab":
		m.activeSection = prevGraphSection(m.activeSection)
		return true, m, nil
	case "up", "k":
		if m.status.Mode == state.ModeTargetPick {
			return false, m, nil
		}
		m = moveBrowseCursor(m, -1)
		return true, m, nil
	case "down", "j":
		if m.status.Mode == state.ModeTargetPick {
			return false, m, nil
		}
		m = moveBrowseCursor(m, 1)
		return true, m, nil
	case "left", "h":
		if m.activeSection == sectionGraph {
			m = moveGraphLane(m, -1)
			return true, m, nil
		}
		return true, m, nil
	case "right", "l":
		if m.activeSection == sectionGraph {
			m = moveGraphLane(m, 1)
			return true, m, nil
		}
		return true, m, nil
	case "g":
		if m.activeSection == sectionGraph {
			if m.awaitingGoTop {
				m.sectionCursor[sectionGraph] = 0
				m.graphScroll = 0
				rows := graphRows(m.repoStatus)
				if len(rows) > 0 {
					m.graphLaneCursor = graph.PointerLane(rows[0])
				}
				m.awaitingGoTop = false
				return true, m, nil
			}
			m.awaitingGoTop = true
		}
		return true, m, nil
	case "G":
		if m.activeSection == sectionGraph {
			rows := graphRows(m.repoStatus)
			if len(rows) > 0 {
				last := len(rows) - 1
				m.sectionCursor[sectionGraph] = last
				m.graphScroll = clampScroll(last, len(rows), graphPageSize(&m))
				m.graphLaneCursor = graph.PointerLane(rows[last])
			}
			m.awaitingGoTop = false
			return true, m, nil
		}
		return true, m, nil
	case "H":
		if m.activeSection == sectionGraph {
			rows := graphRows(m.repoStatus)
			rowIdx := graph.FindRowByHash(rows, m.repoStatus.Head)
			if rowIdx >= 0 {
				m.sectionCursor[sectionGraph] = rowIdx
				m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
				m.graphLaneCursor = graph.PointerLane(rows[rowIdx])
			}
			m.awaitingGoTop = false
		}
		return true, m, nil
	case "ctrl+u":
		if m.activeSection == sectionGraph {
			m = pageBrowseGraph(m, -1)
			return true, m, nil
		}
		m.contextScroll -= 4
		if m.contextScroll < 0 {
			m.contextScroll = 0
		}
		return true, m, nil
	case "ctrl+d":
		if m.activeSection == sectionGraph {
			m = pageBrowseGraph(m, 1)
			return true, m, nil
		}
		m.contextScroll += 4
		return true, m, nil
	default:
		return false, m, nil
	}
}

func (m model) handleBrowseGraphKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash == "" || focus.Hash == "VIRTUAL_CONFLICT_HASH" {
			return m, nil
		}
		m.commitInspectorOpen = true
		m.commitInspectorLoading = true
		m.commitInspectorMetadataLoading = true
		m.commitInspectorDiffLoading = false
		m.commitInspectorError = ""
		m.commitInspectorDiffError = ""
		m.commitInspectorSnapshot = CommitSnapshot{}
		m.commitInspectorDiffWindow = DiffWindow{}
		m.commitInspectorWindowRequest = DiffWindowRequest{}
		m.commitInspectorStale = false
		m.commitInspectorRevalidating = false
		m.commitInspectorSelectedFileID = ""
		m.commitInspectorSelectedCanonicalKey = ""
		m.commitInspectorContinuationPending = false
		m.commitInspectorLines = nil
		m = m.cancelInspector()
		m.commitInspectorRequest++
		m.commitInspectorEpoch = m.repositoryEpoch
		m.commitInspectorRequestedCommit = focus.Hash
		m.commitInspectorRequestedParent = ""
		m.commitInspectorContext, m.commitInspectorCancel = context.WithCancel(context.Background())
		return m, inspectCommitCommand(m.commitInspectorContext, m, CommitRequest{Commit: focus.Hash, RequestID: m.commitInspectorRequest, RepositoryEpoch: m.commitInspectorEpoch})
	case "esc":
		if strings.TrimSpace(m.graphSearchQuery) != "" || m.graphSearchOpen {
			m.graphSearchOpen = false
			m.graphSearchDraft = ""
			m.graphSearchQuery = ""
			m.graphSearchCursor = 0
			m.graphSearchError = ""
			return m, nil
		}
		return m, nil
	case "t":
		focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash == "" || focus.Hash == "VIRTUAL_CONFLICT_HASH" {
			return m, nil
		}
		m.tagPopupOpen = true
		m.tagPopupDraft = ""
		m.tagPopupError = ""
		m.tagPopupTarget = focus.Hash
		return m, nil
	case "o":
		entries := graphStashPopEntriesForFocus(m)
		if len(entries) == 0 {
			if !graphFocusIsHead(m) {
				m.status = state.New().WithBlocked(state.BlockUnknown, "Stash pop unavailable.", "Focus HEAD before popping stash.")
			} else {
				m.status = state.New().WithBlocked(state.BlockTargetEmpty, "No stash available.", "Add a stash at HEAD first.")
			}
			return m, nil
		}
		m = openGraphStashPop(m)
		return m, nil
	case "x":
		m.status = state.New().WithBlocked(state.BlockUnknown, "Cherry-pick is disabled.", "This mode is temporarily unavailable.")
		return m, nil
	case "m":
		if !isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Merge unavailable.", "Select a local branch.")
			return m, nil
		}
		focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash == "" || focus.Hash == "VIRTUAL_CONFLICT_HASH" {
			return m, nil
		}
		m.status = operationLoadingStatusFor(progressGraphTargetAnalysis, "Analyzing graph target...", state.ActionMerge)
		return m, checkGraphActionTarget(m.repo, state.ActionMerge, focus.Hash, m.repoStatus)
	case "r":
		if !isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Rebase unavailable.", "Select a local branch.")
			return m, nil
		}
		focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash == "" || focus.Hash == "VIRTUAL_CONFLICT_HASH" {
			return m, nil
		}
		m.status = operationLoadingStatusFor(progressGraphTargetAnalysis, "Analyzing graph target...", state.ActionRebase)
		return m, checkGraphActionTarget(m.repo, state.ActionRebase, focus.Hash, m.repoStatus)
	case "s":
		focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash == "" {
			m.status = state.New().WithBlocked(state.BlockUnknown, "No reset target.", "Move to a commit line.")
			return m, nil
		}
		m.status = operationLoadingStatusFor(progressPrepareReset, "Preparing reset...", state.ActionReset)
		return m, previewSelection(m.repo, m.repoStatus, state.ActionReset, focus.Hash)
	case "space", " ":
		targets := graphCheckoutTargets(m)
		if len(targets) == 0 {
			return m, nil
		}
		if m.repoStatus.WorktreeDirty {
			m.status = state.New().WithBlocked(state.BlockDirtyTree, "Working tree is dirty.", "Commit or stash changes first.")
			return m, nil
		}
		if len(targets) == 1 {
			m.status = checkoutConfirmStatus(targets[0].Ref)
			return m, nil
		}
		status := state.New().WithTargetPick(state.ActionCheckout, targets)
		status.Title = "Checkout branch"
		status.Message = "Choose a local branch."
		status.Detail = "Enter confirms. Esc returns."
		m.status = status
		return m, nil
	case "/":
		return openGraphSearch(m), nil
	case "n":
		if strings.TrimSpace(m.graphSearchQuery) != "" {
			return applyGraphSearchRepeat(m, 1), nil
		}
		base := branchCreateBaseForActiveSection(m)
		m, _ = startBranchCreateInput(m, base)
		return m, nil
	case "N":
		if strings.TrimSpace(m.graphSearchQuery) != "" {
			return applyGraphSearchRepeat(m, -1), nil
		}
		return m, nil
	case "d":
		targets := graphBranchDeleteTargets(m)
		if len(targets) > 1 {
			status := state.New().WithTargetPick(state.ActionDeleteBranch, targets)
			status.Title = "Delete branch"
			status.Message = "Choose a branch to delete."
			status.Detail = "Enter confirms. Esc returns."
			m.status = status
			return m, nil
		}
		selection := deleteBranchSelection(m)
		if !selection.ok {
			m.status = selection.blocked
			return m, nil
		}
		m.status = deleteBranchConfirmStatus(selection)
		return m, nil
	default:
		return m, nil
	}
}

func (m model) handleBrowseSectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.activeSection != sectionTags {
			return m, nil
		}
		item, ok := activeSectionTargetItem(m)
		if !ok || item.CommitHash == "" {
			m.status = state.New().WithBlocked(state.BlockTargetEmpty, "No tag selected.", "Move to a tag row.")
			return m, nil
		}
		rows := graphRows(m.repoStatus)
		row := graph.FindRowByHash(rows, item.CommitHash)
		if row < 0 {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Tag target is missing.", "Refresh the repo and try again.")
			return m, nil
		}
		m.activeSection = sectionGraph
		m.sectionCursor[sectionGraph] = row
		m.graphLaneCursor = graph.PointerLane(rows[row])
		hint := repositoryStateHintForModel(&m)
		m.graphScroll = clampScroll(row, len(rows), graphPageSizeForRowsWithHint(&m, rows, row, graphContentHeightForModel(&m), hint != ""))
		m.awaitingGoTop = false
		return m, nil
	case "space", " ":
		if m.activeSection == sectionCurrent || m.activeSection == sectionRemote {
			if target := activeSectionTarget(m); target != "" {
				if m.repoStatus.WorktreeDirty {
					m.status = state.New().WithBlocked(state.BlockDirtyTree, "Working tree is dirty.", "Commit or stash changes first.")
					return m, nil
				}
				titleMsg := "Checkout branch?"
				m.status = state.New().WithConfirm(state.ActionCheckout, titleMsg, "Switch to "+target+".")
				m.status.Title = titleMsg
				m.status.Selected = target
				return m, nil
			}
			m.status = state.New().WithBlocked(state.BlockUnknown, "No checkout target.", "Move to a local or remote branch.")
			return m, nil
		}
		m.status = state.New().WithBlocked(state.BlockUnknown, "Checkout unavailable here.", "Use the Context or Remote section.")
		return m, nil
	case "s":
		if m.activeSection == sectionCurrent {
			if !m.repoStatus.WorktreeDirty {
				m.status = state.New().WithBlocked(state.BlockDirtyTree, "Working tree is clean.", "Nothing to stash.")
				return m, nil
			}
			m.stashMessageOpen = true
			m.stashMessageDraft = ""
			m.stashMessageError = ""
			m.status = deriveStatus(m.repoStatus)
			return m, nil
		}
		return m, nil
	case "c":
		if m.activeSection == sectionCurrent {
			if !m.repoStatus.WorktreeDirty {
				m.status = state.New().WithBlocked(state.BlockDirtyTree, "Working tree is clean.", "Nothing to clean.")
				return m, nil
			}
			m.status = confirmCleanWorkingTree()
			return m, nil
		}
		return m, nil
	case "a":
		if m.activeSection == sectionCurrent && (m.repoStatus.MergeInProgress || m.repoStatus.RebaseInProgress) {
			m.status = operationLoadingStatusFor(progressAbort, "Aborting...", state.ActionAbort)
			return m, executeAbort(m.repo, m.commitLimit)
		}
		return m, nil
	case "p":
		if m.activeSection == sectionCurrent {
			if pullReady(m.repoStatus) {
				request := beginPullRequest(&m)
				m.status = operationLoadingStatusFor(progressFetch, "Fetching upstream...", state.ActionPull)
				if m.pull != nil {
					return m, startPullPreview(&m, request)
				}
				return m, executeFetchForPull(m.repo, m.commitLimit, request)
			}
			m.status = actionPull(m.repoStatus)
			return m, nil
		}
		if m.activeSection == sectionGraph {
			if !pullReady(m.repoStatus) {
				m.status = actionPull(m.repoStatus)
				return m, nil
			}
			m.status = operationLoadingStatusFor(progressFetch, "Fetching upstream...", state.ActionPull)
			request := beginPullRequest(&m)
			if m.pull != nil {
				return m, startPullPreview(&m, request)
			}
			return m, executeFetchForPull(m.repo, m.commitLimit, request)
		}
		return m, nil
	case "n":
		if m.activeSection == sectionGraph && m.status.Mode == state.ModeBrowse && m.graphSearchOpen == false {
			if strings.TrimSpace(m.graphSearchQuery) != "" {
				return applyGraphSearchRepeat(m, 1), nil
			}
		}
		if m.activeSection == sectionCurrent {
			base := branchCreateBaseForActiveSection(m)
			m, _ = startBranchCreateInput(m, base)
			return m, nil
		}
		return m, nil
	case "/":
		if m.activeSection != sectionGraph || m.status.Mode != state.ModeBrowse {
			return m, nil
		}
		return openGraphSearch(m), nil
	case "d":
		if m.activeSection == sectionTags {
			selection := deleteTagSelection(m)
			if !selection.ok {
				m.status = selection.blocked
				return m, nil
			}
			m.status = deleteConfirmStatus(state.ActionDeleteTag, selection.title, selection.detail, selection.target)
			return m, nil
		}
		if m.activeSection == sectionGraph || m.activeSection == sectionCurrent || m.activeSection == sectionRemote {
			selection := deleteBranchSelection(m)
			if !selection.ok {
				m.status = selection.blocked
				return m, nil
			}
			m.status = deleteBranchConfirmStatus(selection)
			return m, nil
		}
		return m, nil
	case "D":
		if m.activeSection == sectionTags {
			selection := deleteRemoteTagSelection(m)
			if !selection.ok {
				m.status = selection.blocked
				return m, nil
			}
			m.status = deleteConfirmStatus(state.ActionDeleteRemoteTag, selection.title, selection.detail, selection.target)
			return m, nil
		}
		return m, nil
	default:
		return m, nil
	}
}

func confirmCleanWorkingTree() state.Status {
	status := state.New().WithConfirm(
		state.ActionCleanWorkingTree,
		"Clean working tree?",
		"This will discard local changes and untracked files.",
	)
	status.Title = "Clean working tree?"
	return status
}
