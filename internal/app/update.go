package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/git-graph-tui/internal/graph"
	"hrllk/git-graph-tui/internal/state"
	"hrllk/git-graph-tui/internal/telemetry"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case loadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = m.status.WithError(msg.err.Error())
			telemetry.Log("app", "load_error", map[string]string{"error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		m.status = deriveStatus(msg.status)
		telemetry.Log("app", "load_repo", map[string]string{
			"root":   msg.status.Root,
			"branch": msg.status.Branch,
			"head":   msg.status.Head,
		})
		return m, nil
	case tickMsg:
		return m, tea.Batch(scheduleRefresh(), refreshRepoState(m.repo, m.commitLimit))
	case refreshedMsg:
		if msg.err != nil {
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		if !m.branchOpen && (m.status.Mode == state.ModeBrowse || m.status.Mode == state.ModeBlocked || m.status.Mode == state.ModeEmpty || m.status.Mode == state.ModeError) {
			m.status = deriveStatus(msg.status)
		}
		return m, nil
	case fetchedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed.", msg.err.Error())
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		if m.status.Mode == state.ModeBrowse || m.status.Mode == state.ModeBlocked || m.status.Mode == state.ModeEmpty || m.status.Mode == state.ModeError {
			m.status = deriveStatus(msg.status)
		}
		telemetry.Log("app", "fetch_repo", map[string]string{
			"branch": msg.status.Branch,
			"head":   msg.status.Head,
		})
		return m, nil
	case preparedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed.", msg.err.Error())
			telemetry.Log("app", "prepare_failed", map[string]string{"action": string(msg.action), "error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		switch msg.action {
		case state.ActionMerge, state.ActionRebase, state.ActionReset:
			m.status = actionPickTargets(msg.status, msg.action)
		default:
			m.status = deriveStatus(msg.status)
		}
		telemetry.Log("app", "prepare_action", map[string]string{
			"action": string(msg.action),
			"branch": msg.status.Branch,
		})
		return m, nil
	case pullCheckedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed.", msg.err.Error())
			telemetry.Log("app", "pull_check_failed", map[string]string{"error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.repo
		syncBrowseState(&m, msg.repo)
		m.status = msg.status
		telemetry.Log("app", "pull_check", map[string]string{
			"upstream": msg.repo.Upstream,
			"blocked":  string(msg.status.Block),
		})
		return m, nil
	case previewMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Preview failed.", msg.err.Error())
			telemetry.Log("app", "preview_failed", map[string]string{"action": string(msg.action), "target": msg.target, "error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.repo
		syncBrowseState(&m, msg.repo)
		m.status = msg.status
		m.status.Selected = msg.target
		telemetry.Log("app", "preview_action", map[string]string{
			"action": string(msg.action),
			"target": msg.target,
			"mode":   string(msg.status.Mode),
		})
		return m, nil
	case pushFetchedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed before push.", msg.err.Error())
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		if msg.status.NoUpstream {
			branchName := msg.status.Branch
			titleMsg := "Push and Track Remote?"
			detailMsg := fmt.Sprintf("There is no upstream configured for the current branch. Do you want to push and set upstream tracking to origin/%s?", branchName)
			m.status = m.status.WithConfirm(state.ActionSetUpstream, titleMsg, detailMsg)
			m.status.Title = titleMsg
			return m, nil
		}
		m.status = state.New().WithLoading("Pushing commits...")
		return m, executePush(m.repo, msg.status.Branch, m.commitLimit)
	case pullFetchedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed before pull.", msg.err.Error())
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		track := m.repoStatus.Tracking[m.repoStatus.Branch]
		isFF := track.Behind > 0 && track.Ahead == 0
		m.status = state.New().WithLoading("Analyzing pull changes...")
		return m, loadPullPreviewCommits(m.repo, isFF)
	case pullPreviewReadyMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Analysis failed.", msg.err.Error())
			return m, nil
		}
		m.handshakeCommits = make(map[string]bool)
		if msg.isFF {
			if len(msg.commits) > 0 {
				m.handshakeCommits[msg.commits[0]] = true
			}
		} else {
			for _, hash := range msg.commits {
				m.handshakeCommits[hash] = true
			}
		}
		m.pullIsFastForward = msg.isFF
		var titleMsg, detailMsg string
		if msg.isFF {
			titleMsg = "Do you want to continue?"
			detailMsg = "The branch will fast-forward to the highlighted target commit."
			m.status = m.status.WithConfirm(state.ActionPull, titleMsg, detailMsg)
		} else {
			titleMsg = "Choose Pull Integration"
			detailMsg = "The branches have diverged. Choose integration strategy:\n\nm: merge pull (recreates merge commit)\nr: rebase pull (replays commits)\nesc: cancel integration"
			m.status = m.status.WithConfirm(state.ActionPull, titleMsg, detailMsg)
		}
		m.status.Title = titleMsg
		return m, nil
	case executedMsg:
		if msg.err != nil {
			isAuthError := strings.Contains(msg.err.Error(), "Permission denied") ||
				strings.Contains(msg.err.Error(), "Authentication failed") ||
				strings.Contains(msg.err.Error(), "Could not read from remote repository")

			if msg.action == state.ActionPush && !isAuthError && (strings.Contains(msg.err.Error(), "[rejected]") || strings.Contains(msg.err.Error(), "non-fast-forward")) {
				status := m.repoStatus
				if msg.status.Root != "" {
					status = msg.status
				}
				m.repoStatus = status
				m.handshakeCommits = make(map[string]bool)
				if status.Head != "" {
					m.handshakeCommits[status.Head] = true
				}
				remoteHash := findRemoteCommitHash(status, status.Upstream)
				if remoteHash != "" {
					m.handshakeCommits[remoteHash] = true
				}
				branchName := status.Branch
				titleMsg := fmt.Sprintf("Force push to origin/%s?", branchName)
				detailMsg := fmt.Sprintf("The remote branch has different history. Force pushing will overwrite origin/%s history with your local commits. Continue?", branchName)
				m.status = m.status.WithConfirm(state.ActionForcePush, titleMsg, detailMsg)
				m.status.Title = titleMsg
				return m, nil
			}
			if (msg.action == state.ActionPull || msg.action == state.ActionPullMerge || msg.action == state.ActionPullRebase) && (msg.status.MergeInProgress || msg.status.RebaseInProgress) {
				m.repoStatus = msg.status
				syncBrowseState(&m, msg.status)
				m.status = state.New().WithBrowse()
				m.status.Message = "Pull stopped with conflicts."
				m.status.Detail = "Press enter to abort the in-progress merge/rebase."
				telemetry.Log("app", "execute_conflicted", map[string]string{
					"action": string(msg.action),
					"head":   msg.status.Head,
				})
				return m, nil
			}
			if msg.action == state.ActionMerge && msg.status.MergeInProgress {
				m.repoStatus = msg.status
				syncBrowseState(&m, msg.status)
				m.status = state.New().WithBrowse()
				m.status.Message = "Merge stopped with conflicts."
				m.status.Detail = "Resolve conflicts, then press 'a' to abort or commit to complete."
				telemetry.Log("app", "execute_conflicted", map[string]string{
					"action": string(msg.action),
					"head":   msg.status.Head,
				})
				return m, nil
			}
			if msg.action == state.ActionRebase && msg.status.RebaseInProgress {
				m.repoStatus = msg.status
				syncBrowseState(&m, msg.status)
				m.status = state.New().WithBrowse()
				m.status.Message = "Rebase stopped with conflicts."
				m.status.Detail = "Resolve conflicts, then press 'a' to abort or continue rebase."
				telemetry.Log("app", "execute_conflicted", map[string]string{
					"action": string(msg.action),
					"head":   msg.status.Head,
				})
				return m, nil
			}
			reason := state.BlockUnknown
			message := "Execution failed."
			detail := msg.err.Error()
			if msg.action == state.ActionCheckout {
				message = "Checkout failed."
				if strings.Contains(detail, "local changes") || strings.Contains(detail, "overwritten by checkout") {
					reason = state.BlockDirtyTree
					message = "Checkout blocked by local changes."
					detail = "Your local changes would be overwritten by checkout. Commit or stash them before switching."
				}
			} else if isAuthError && (msg.action == state.ActionPush || msg.action == state.ActionForcePush || msg.action == state.ActionSetUpstream) {
				message = "Authentication or Permission error."
				detail = "Please check your remote credentials or network connection: " + msg.err.Error()
			} else if msg.action == state.ActionPush || msg.action == state.ActionForcePush || msg.action == state.ActionSetUpstream {
				message = "Push failed."
			}
			m.status = state.New().WithBlocked(reason, message, detail)
			telemetry.Log("app", "execute_failed", map[string]string{"action": string(msg.action), "target": msg.target, "error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.status
		if msg.action == state.ActionPush || msg.action == state.ActionForcePush || msg.action == state.ActionSetUpstream || msg.action == state.ActionPullMerge || msg.action == state.ActionPullRebase {
			m.handshakeCommits = make(map[string]bool)
			syncBrowseState(&m, msg.status)
			m.status = deriveStatus(msg.status)
			if msg.action == state.ActionPullMerge || msg.action == state.ActionPullRebase {
				m.status.Message = "Pull completed successfully."
			} else {
				m.status.Message = fmt.Sprintf("Push completed for %s.", msg.target)
			}
			telemetry.Log("app", "execute_action", map[string]string{
				"action": string(msg.action),
				"head":   msg.status.Head,
			})
			return m, nil
		}
		if msg.action == state.ActionCheckout {
			m.commitLimit = initialGraphCommitLimit
			rows := graph.Rows(msg.status)
			if len(rows) > 0 {
				m.sectionCursor[sectionGraph] = 0
				m.graphScroll = 0
				m.graphLaneCursor = graph.PointerLane(rows[0])
			}
			syncBrowseState(&m, msg.status)
			m.status = deriveStatus(msg.status)
			telemetry.Log("app", "execute_action", map[string]string{
				"action": string(msg.action),
				"target": msg.target,
				"head":   msg.status.Head,
			})
			return m, nil
		}
		if msg.action == state.ActionPull {
			syncBrowseState(&m, msg.status)
			m.status = deriveStatus(msg.status)
			telemetry.Log("app", "execute_action", map[string]string{
				"action": string(msg.action),
				"head":   msg.status.Head,
			})
			return m, nil
		}
		if msg.action == state.ActionAbort {
			m.handshakeCommits = make(map[string]bool)
			syncBrowseState(&m, msg.status)
			m.status = deriveStatus(msg.status)
			telemetry.Log("app", "execute_action", map[string]string{
				"action": string(msg.action),
				"head":   msg.status.Head,
			})
			return m, nil
		}
		if msg.action == state.ActionReset {
			rows := graph.Rows(msg.status)
			rowIdx := graph.FindRowByHash(rows, msg.status.Head)
			if rowIdx >= 0 {
				m.sectionCursor[sectionGraph] = rowIdx
				m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
			}
			syncBrowseState(&m, msg.status)
			m.status = deriveStatus(msg.status)
			m.status.Message = fmt.Sprintf("Hard reset completed to %s.", shorten(msg.target, 7))
			telemetry.Log("app", "execute_action", map[string]string{
				"action": string(msg.action),
				"target": msg.target,
				"head":   msg.status.Head,
			})
			return m, nil
		}
		if msg.action == state.ActionMerge || msg.action == state.ActionRebase {
			rows := graph.Rows(msg.status)
			rowIdx := graph.FindRowByHash(rows, msg.status.Head)
			if rowIdx >= 0 {
				m.sectionCursor[sectionGraph] = rowIdx
				m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
			}
		}
		syncBrowseState(&m, msg.status)
		m.status = state.New().WithOutcome(msg.action, "Completed.", executionDetail(msg.action, msg.target, msg.status), false)
		m.status.Selected = msg.target
		telemetry.Log("app", "execute_action", map[string]string{
			"action": string(msg.action),
			"target": msg.target,
			"head":   msg.status.Head,
		})
		return m, nil
	case createdBranchMsg:
		if msg.err != nil {
			m.branchOpen = false
			reason := state.BlockUnknown
			message := "Branch creation failed."
			detail := msg.err.Error()
			if strings.Contains(msg.err.Error(), "working tree is not clean") {
				reason = state.BlockDirtyTree
				message = "Working tree is not clean."
				detail = "Commit or stash local changes before creating and checking out a new branch."
			}
			m.status = state.New().WithBlocked(reason, message, detail)
			telemetry.Log("app", "branch_create_failed", map[string]string{"name": msg.name, "base": msg.base, "error": msg.err.Error()})
			return m, nil
		}
		m.branchOpen = false
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		m.status = deriveStatus(msg.status)
		telemetry.Log("app", "branch_create", map[string]string{"name": msg.name, "base": msg.base})
		return m, nil
	case tea.KeyMsg:
		if m.branchOpen {
			switch msg.String() {
			case "esc":
				m.branchOpen = false
				m.branchDraft = ""
				m.status = deriveStatus(m.repoStatus)
				return m, nil
			case "enter":
				name := strings.TrimSpace(m.branchDraft)
				base := m.branchBase
				m.branchOpen = false
				m.branchDraft = ""
				m.status = state.New().WithLoading("Creating branch...")
				return m, createBranch(m.repo, name, base, m.commitLimit)
			case "backspace":
				if len(m.branchDraft) > 0 {
					runes := []rune(m.branchDraft)
					m.branchDraft = string(runes[:len(runes)-1])
				}
				return m, nil
			default:
				if len(msg.Runes) > 0 {
					m.branchDraft += string(msg.Runes)
					return m, nil
				}
			}
		}
		if m.status.Mode == state.ModeConfirm {
			switch msg.String() {
			case "y", "enter":
				action := m.status.Action
				m.handshakeCommits = make(map[string]bool)
				if action == state.ActionPull {
					if m.pullIsFastForward {
						m.status = state.New().WithLoading("Running pull...")
						return m, executePull(m.repo, m.commitLimit)
					} else {
						m.status = state.New().WithLoading("Running merge pull...")
						return m, executePullMerge(m.repo, m.commitLimit)
					}
				} else if action == state.ActionSetUpstream {
					m.status = state.New().WithLoading("Pushing new branch and tracking upstream...")
					return m, executePushSetUpstream(m.repo, m.repoStatus.Branch, m.commitLimit)
				} else if action == state.ActionForcePush {
					m.status = state.New().WithLoading("Running force push...")
					return m, executeForcePush(m.repo, m.repoStatus.Branch, m.commitLimit)
				} else if action == state.ActionReset {
					target := m.status.Selected
					m.status = state.New().WithLoading("Running hard reset...")
					return m, executeAction(m.repo, action, target, m.commitLimit)
				} else if action == state.ActionMerge {
					target := m.status.Selected
					m.status = state.New().WithLoading("Running merge...")
					return m, executeAction(m.repo, action, target, m.commitLimit)
				} else if action == state.ActionRebase {
					target := m.status.Selected
					m.status = state.New().WithLoading("Running rebase...")
					return m, executeAction(m.repo, action, target, m.commitLimit)
				}
				m.status = deriveStatus(m.repoStatus)
				return m, nil
			case "m":
				action := m.status.Action
				if action == state.ActionPull && !m.pullIsFastForward {
					m.handshakeCommits = make(map[string]bool)
					m.status = state.New().WithLoading("Running merge pull...")
					return m, executePullMerge(m.repo, m.commitLimit)
				}
				return m, nil
			case "r":
				action := m.status.Action
				if action == state.ActionPull && !m.pullIsFastForward {
					m.handshakeCommits = make(map[string]bool)
					m.status = state.New().WithLoading("Running rebase pull...")
					return m, executePullRebase(m.repo, m.commitLimit)
				}
				return m, nil
			case "n", "esc":
				m.handshakeCommits = make(map[string]bool)
				m.status = deriveStatus(m.repoStatus)
				return m, nil
			default:
				return m, nil
			}
		}
		if m.awaitingGoTop && msg.String() != "g" {
			m.awaitingGoTop = false
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "1":
			if m.status.Mode == state.ModeBrowse {
				m = switchBrowseSection(m, sectionCurrent)
			}
		case "2":
			if m.status.Mode == state.ModeBrowse {
				m = switchBrowseSection(m, sectionRemote)
			}
		case "3":
			if m.status.Mode == state.ModeBrowse {
				m = switchBrowseSection(m, sectionTags)
			}
		case "4":
			if m.status.Mode == state.ModeBrowse {
				m = switchBrowseSection(m, sectionGraph)
			}
		case "f":
			if m.status.Mode == state.ModeBrowse {
				m.status.Message = "Fetching remotes..."
				m.status.Detail = "Refreshing remote refs and branch tracking in the background."
				return m, fetchRepoState(m.repo, m.commitLimit)
			}
		case "P":
			if m.status.Mode == state.ModeBrowse {
				if m.repoStatus.Root == "" || m.repoStatus.Detached || m.repoStatus.EmptyRepo {
					return m, nil
				}
				m.status = state.New().WithLoading("Fetching before push...")
				return m, executeFetchForPush(m.repo, m.commitLimit)
			}
		case "p":
			if pullReady(m.repoStatus) {
				m.status = state.New().WithLoading("Fetching upstream before pull...")
				return m, executeFetchForPull(m.repo, m.commitLimit)
			}
			m.status = actionPull(m.repoStatus)
		case "m":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				if !isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
					m.status = state.New().WithBlocked(state.BlockUnknown,
						"Merge not available.",
						"Move the lane cursor onto a local branch to enable merge.")
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
			}
		case "r":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				if !isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
					m.status = state.New().WithBlocked(state.BlockUnknown,
						"Rebase not available.",
						"Move the lane cursor onto a local branch to enable rebase.")
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
			}
		case "s":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
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
			}
		case "a":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionCurrent && (m.repoStatus.MergeInProgress || m.repoStatus.RebaseInProgress) {
				m.status = state.New().WithLoading("Aborting merge/rebase...")
				return m, executeAbort(m.repo, m.commitLimit)
			}
		case "esc":
			switch {
			case m.status.Mode == state.ModeOutcomePreview && m.status.Action != state.ActionPull && m.status.Action != state.ActionAbort:
				m.status = actionPickTargets(m.repoStatus, m.status.Action)
			case m.status.Mode == state.ModeOutcomePreview && (m.status.Action == state.ActionPull || m.status.Action == state.ActionAbort):
				m.status = deriveStatus(m.repoStatus)
			default:
				m.status = deriveStatus(m.repoStatus)
			}
		case "tab":
			if m.status.Mode == state.ModeBrowse {
				m.activeSection = nextGraphSection(m.activeSection)
			}
		case "shift+tab":
			if m.status.Mode == state.ModeBrowse {
				m.activeSection = prevGraphSection(m.activeSection)
			}
		case "up", "k":
			if m.status.Mode == state.ModeTargetPick {
				m.status = moveTarget(m.status, -1)
			} else if m.status.Mode == state.ModeBrowse {
				m = moveBrowseCursor(m, -1)
			}
		case "down", "j":
			if m.status.Mode == state.ModeTargetPick {
				m.status = moveTarget(m.status, 1)
			} else if m.status.Mode == state.ModeBrowse {
				m = moveBrowseCursor(m, 1)
				return maybeLoadMoreGraph(m)
			}
		case "left", "h":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				m = moveGraphLane(m, -1)
			}
		case "right", "l":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				m = moveGraphLane(m, 1)
			}
		case "g":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				if m.awaitingGoTop {
					m.sectionCursor[sectionGraph] = 0
					m.graphScroll = 0
					rows := graph.Rows(m.repoStatus)
					if len(rows) > 0 {
						m.graphLaneCursor = graph.PointerLane(rows[0])
					}
					m.awaitingGoTop = false
					return m, nil
				}
				m.awaitingGoTop = true
			}
		case "G":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				rows := graph.Rows(m.repoStatus)
				if len(rows) > 0 {
					last := len(rows) - 1
					m.sectionCursor[sectionGraph] = last
					m.graphScroll = clampScroll(last, len(rows), graphPageSize(&m))
					m.graphLaneCursor = graph.PointerLane(rows[last])
				}
				m.awaitingGoTop = false
				return maybeLoadMoreGraph(m)
			}
		case "H":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				rows := graph.Rows(m.repoStatus)
				rowIdx := graph.FindRowByHash(rows, m.repoStatus.Head)
				if rowIdx >= 0 {
					m.sectionCursor[sectionGraph] = rowIdx
					m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
					m.graphLaneCursor = graph.PointerLane(rows[rowIdx])
				}
				m.awaitingGoTop = false
			}
		case "ctrl+u":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				m = pageBrowseGraph(m, -1)
			}
		case "ctrl+d":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				m = pageBrowseGraph(m, 1)
				return maybeLoadMoreGraph(m)
			}
		case "space", " ":
			if m.status.Mode == state.ModeTargetPick {
				action := m.status.Action
				target := selectedTarget(m.status)
				if target == "" {
					m.status = state.New().WithBlocked(state.BlockTargetEmpty, "No target selected.", "Choose a branch, tag, or ref first.")
					return m, nil
				}
				m.status = state.New().WithLoading("Previewing result...")
				return m, previewSelection(m.repo, m.repoStatus, action, target)
			}
			if m.status.Mode == state.ModeBrowse {
				if m.activeSection == sectionCurrent || m.activeSection == sectionRemote {
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
			}
			if m.status.Mode == state.ModeOutcomePreview && m.status.CanExecute {
				action := m.status.Action
				target := m.status.Selected
				m.status = state.New().WithLoading("Running action...")
				switch action {
				case state.ActionPull:
					return m, executePull(m.repo, m.commitLimit)
				case state.ActionAbort:
					return m, executeAbort(m.repo, m.commitLimit)
				case state.ActionMerge, state.ActionRebase, state.ActionReset:
					return m, executeAction(m.repo, action, target, m.commitLimit)
				}
			}
		case "n":
			if m.status.Mode == state.ModeBrowse && (m.activeSection == sectionCurrent || m.activeSection == sectionGraph) {
				if !canCreateBranch(m.repoStatus) {
					m.status = state.New().WithBlocked(
						state.BlockDirtyTree,
						"Working tree is not clean.",
						"Commit or stash local changes before creating and checking out a new branch.",
					)
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
			}
		}
	}
	return m, nil
}
