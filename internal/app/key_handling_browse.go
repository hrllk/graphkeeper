package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

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
	case sectionCurrent, sectionLocal, sectionRemote, sectionTags:
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
		m = switchBrowseSection(m, sectionCurrent)
		return true, m, nil
	case "2":
		m = switchBrowseSection(m, sectionRemote)
		return true, m, nil
	case "3":
		m = switchBrowseSection(m, sectionTags)
		return true, m, nil
	case "4":
		m = switchBrowseSection(m, sectionGraph)
		return true, m, nil
	case "f":
		m.status.Message = "Fetching remotes..."
		m.status.Detail = "Refreshing remote refs and branch tracking in the background."
		return true, m, fetchRepoState(m.repo, m.commitLimit)
	case "P":
		if m.repoStatus.Root == "" || m.repoStatus.Detached || m.repoStatus.EmptyRepo {
			return true, m, nil
		}
		m.status = state.New().WithLoading("Fetching before push...")
		return true, m, executeFetchForPush(m.repo, m.commitLimit)
	case "p":
		if pullReady(m.repoStatus) {
			m.status = state.New().WithLoading("Fetching upstream before pull...")
			return true, m, executeFetchForPull(m.repo, m.commitLimit)
		}
		m.status = actionPull(m.repoStatus)
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
		var cmd tea.Cmd
		m, cmd = maybeLoadMoreGraph(m)
		return true, m, cmd
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
				rows := graph.Rows(m.repoStatus)
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
			rows := graph.Rows(m.repoStatus)
			if len(rows) > 0 {
				last := len(rows) - 1
				m.sectionCursor[sectionGraph] = last
				m.graphScroll = clampScroll(last, len(rows), graphPageSize(&m))
				m.graphLaneCursor = graph.PointerLane(rows[last])
			}
			m.awaitingGoTop = false
			var cmd tea.Cmd
			m, cmd = maybeLoadMoreGraph(m)
			return true, m, cmd
		}
		return true, m, nil
	case "H":
		if m.activeSection == sectionGraph {
			rows := graph.Rows(m.repoStatus)
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
		return true, m, nil
	case "ctrl+d":
		if m.activeSection == sectionGraph {
			m = pageBrowseGraph(m, 1)
			var cmd tea.Cmd
			m, cmd = maybeLoadMoreGraph(m)
			return true, m, cmd
		}
		return true, m, nil
	default:
		return false, m, nil
	}
}

func (m model) handleBrowseGraphKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "m":
		if !isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Merge not available.", "Move the lane cursor onto a local branch to enable merge.")
			return m, nil
		}
		focus := graph.CurrentFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash == "" || focus.Hash == "VIRTUAL_CONFLICT_HASH" {
			return m, nil
		}
		titleMsg := "Merge into current branch?"
		detailMsg := fmt.Sprintf("This will merge commit %s into %s.\nA merge commit will be created if histories have diverged.\n\nContinue? (y: yes  •  n: no)",
			shorten(focus.Hash, 7), m.repoStatus.Branch)
		m.status = m.status.WithConfirm(state.ActionMerge, titleMsg, detailMsg)
		m.status.Title = titleMsg
		m.status.Selected = focus.Hash
		return m, nil
	case "r":
		if !isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Rebase not available.", "Move the lane cursor onto a local branch to enable rebase.")
			return m, nil
		}
		focus := graph.CurrentFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash == "" || focus.Hash == "VIRTUAL_CONFLICT_HASH" {
			return m, nil
		}
		titleMsg := "Rebase onto this commit?"
		detailMsg := fmt.Sprintf("This will rebase %s onto commit %s.\nLocal commits will be replayed on top of the target.\n\n⚠️ Conflicts may occur during rebase.\n\nContinue? (y: yes  •  n: no)",
			m.repoStatus.Branch, shorten(focus.Hash, 7))
		m.status = m.status.WithConfirm(state.ActionRebase, titleMsg, detailMsg)
		m.status.Title = titleMsg
		m.status.Selected = focus.Hash
		return m, nil
	case "s":
		focus := graph.CurrentFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash == "" {
			m.status = state.New().WithBlocked(state.BlockUnknown, "No reset target.", "Move the pointer onto a commit line.")
			return m, nil
		}
		titleMsg := "Hard reset to commit?"
		detailMsg := fmt.Sprintf("This will reset your HEAD, index, and working tree. Any uncommitted changes will be lost. Target commit: %s. Continue?", focus.Hash)
		if m.repoStatus.WorktreeDirty {
			detailMsg = fmt.Sprintf("⚠️ WARNING: You have uncommitted changes in your working tree! Hard reset will permanently OVERWRITE and LOSE all uncommitted changes. Target commit: %s. Continue?", focus.Hash)
		}
		m.status = m.status.WithConfirm(state.ActionReset, titleMsg, detailMsg)
		m.status.Title = titleMsg
		m.status.Selected = focus.Hash
		return m, nil
	case "n":
		if !canCreateBranch(m.repoStatus) {
			m.status = state.New().WithBlocked(state.BlockDirtyTree, "Working tree is not clean.", "Commit or stash local changes before creating and checking out a new branch.")
			return m, nil
		}
		base := activeSectionTarget(m)
		if base == "" {
			focus := graph.CurrentFocus(m.repoStatus, m.sectionCursor[sectionGraph])
			base = focus.Hash
		}
		m.branchBase = base
		m.branchOpen = true
		m.branchDraft = ""
		m.status = state.New().WithLoading("Type a new branch name and press enter.")
		return m, nil
	default:
		return m, nil
	}
}

func (m model) handleBrowseSectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "space", " ":
		if m.activeSection == sectionCurrent || m.activeSection == sectionLocal || m.activeSection == sectionRemote {
			if target := activeSectionTarget(m); target != "" {
				m.status = state.New().WithLoading("Checking out " + target + "...")
				return m, executeCheckout(m.repo, target, initialGraphCommitLimit)
			}
			m.status = state.New().WithBlocked(state.BlockUnknown, "No checkout target.", "Move the pointer onto a local or remote branch.")
			return m, nil
		}
		if m.activeSection == sectionGraph {
			return m, nil
		}
		m.status = state.New().WithBlocked(state.BlockUnknown, "Checkout unavailable in this section.", "Use the Local or Remote sections to switch branches.")
		return m, nil
	case "a":
		if (m.activeSection == sectionCurrent || m.activeSection == sectionLocal) && (m.repoStatus.MergeInProgress || m.repoStatus.RebaseInProgress) {
			m.status = state.New().WithLoading("Aborting merge/rebase...")
			return m, executeAbort(m.repo, m.commitLimit)
		}
		return m, nil
	case "n":
		if m.activeSection == sectionCurrent || m.activeSection == sectionLocal {
			if !canCreateBranch(m.repoStatus) {
				m.status = state.New().WithBlocked(state.BlockDirtyTree, "Working tree is not clean.", "Commit or stash local changes before creating and checking out a new branch.")
				return m, nil
			}
			base := activeSectionTarget(m)
			if base == "" {
				focus := graph.CurrentFocus(m.repoStatus, m.sectionCursor[sectionGraph])
				base = focus.Hash
			}
			m.branchBase = base
			m.branchOpen = true
			m.branchDraft = ""
			m.status = state.New().WithLoading("Type a new branch name and press enter.")
			return m, nil
		}
		return m, nil
	default:
		return m, nil
	}
}
