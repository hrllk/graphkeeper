package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

func branchCreateBaseForActiveSection(m model) string {
	switch m.activeSection {
	case sectionGraph:
		focus := graph.CurrentFocus(m.repoStatus, m.sectionCursor[sectionGraph])
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
		m.status.Message = "Fetching..."
		m.status.Detail = "Refreshing refs and tracking."
		return true, m, fetchRepoState(m.repo, m.commitLimit)
	case "P":
		if m.repoStatus.Root == "" || m.repoStatus.Detached || m.repoStatus.EmptyRepo {
			return true, m, nil
		}
		m.status = loadingToast("Fetching for push...")
		return true, m, executeFetchForPush(m.repo, m.commitLimit)
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
			return true, m, nil
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
			return true, m, nil
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
			m.status = state.New().WithBlocked(state.BlockUnknown, "Merge unavailable.", "Select a local branch.")
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
			m.status = state.New().WithBlocked(state.BlockUnknown, "Rebase unavailable.", "Select a local branch.")
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
			m.status = state.New().WithBlocked(state.BlockUnknown, "No reset target.", "Move to a commit line.")
			return m, nil
		}
		m.status = loadingToast("Preparing reset...")
		return m, previewSelection(m.repo, m.repoStatus, state.ActionReset, focus.Hash)
	case "n":
		base := branchCreateBaseForActiveSection(m)
		m, _ = startBranchCreateInput(m, base)
		return m, nil
	default:
		return m, nil
	}
}

func (m model) handleBrowseSectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
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
		if m.activeSection == sectionGraph {
			return m, nil
		}
		m.status = state.New().WithBlocked(state.BlockUnknown, "Checkout unavailable here.", "Use the Local or Remote section.")
		return m, nil
	case "a":
		if m.activeSection == sectionCurrent && (m.repoStatus.MergeInProgress || m.repoStatus.RebaseInProgress) {
			m.status = loadingToast("Aborting...")
			return m, executeAbort(m.repo, m.commitLimit)
		}
		return m, nil
	case "p":
		if m.activeSection == sectionCurrent {
			if pullReady(m.repoStatus) {
				m.status = loadingToast("Fetching upstream...")
				return m, executeFetchForPull(m.repo, m.commitLimit)
			}
			m.status = actionPull(m.repoStatus)
			return m, nil
		}
		if m.activeSection == sectionGraph {
			if !isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
				return m, nil
			}
			if !pullReady(m.repoStatus) {
				m.status = actionPull(m.repoStatus)
				return m, nil
			}
			m.status = loadingToast("Fetching upstream...")
			return m, executeFetchForPull(m.repo, m.commitLimit)
		}
		return m, nil
	case "n":
		if m.activeSection == sectionCurrent {
			base := branchCreateBaseForActiveSection(m)
			m, _ = startBranchCreateInput(m, base)
			return m, nil
		}
		return m, nil
	default:
		return m, nil
	}
}
