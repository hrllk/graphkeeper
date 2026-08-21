package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m.handleConfirmAccept()
	case "m":
		return m.handleConfirmPullMerge()
	case "r":
		return m.handleConfirmPullRebase()
	case "n", "esc":
		if m.pullCancel != nil {
			m.pullCancel()
			m.pullCancel = nil
		}
		m.handshakeCommits = make(map[string]bool)
		m.activePullRequest = nil
		m.pullConfirmStale = false
		m.nextPullRequestID++
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	default:
		return m, nil
	}
}

func (m model) handleConfirmAccept() (tea.Model, tea.Cmd) {
	action := m.status.Action
	m.handshakeCommits = make(map[string]bool)
	switch action {
	case state.ActionPull:
		if m.activePullRequest == nil || m.pullConfirmStale {
			return m, nil
		}
		if m.pullIsFastForward {
			m.status = loadingToast("Pulling...")
			if m.pull != nil {
				return m, startPullWorkflow(&m, PullModeMerge)
			}
			return m, validateAndExecutePull(m.repo, m.commitLimit, *m.activePullRequest, PullModeMerge)
		}
		if m.pull != nil {
			return m, startPullWorkflow(&m, PullModeMerge)
		}
		return m, validateAndExecutePull(m.repo, m.commitLimit, *m.activePullRequest, PullModeMerge)
	case state.ActionSetUpstream:
		m.status = loadingToast("Pushing and tracking...")
		return m, executePushSetUpstream(m.repo, m.repoStatus.Branch, m.commitLimit)
	case state.ActionForcePush:
		m.status = loadingToast("Force pushing...")
		return m, executeForcePush(m.repo, m.repoStatus.Branch, m.commitLimit)
	case state.ActionDeleteBranch:
		target := m.status.Selected
		remote := m.status.DeleteRemote
		m.status = loadingToast(deleteBranchLoadingMessage(remote))
		return m, executeDeleteBranch(m.repo, target, remote, m.commitLimit)
	case state.ActionDeleteTag:
		target := m.status.Selected
		m.status = loadingToast(deleteTagLoadingMessage())
		return m, executeDeleteTag(m.repo, target, m.commitLimit, m.tagProvenance)
	case state.ActionDeleteRemoteTag:
		target := m.status.Selected
		m.status = loadingToast(deleteRemoteTagLoadingMessage())
		return m, executeDeleteRemoteTag(m.repo, target, m.commitLimit, m.tagProvenance)
	case state.ActionStash:
		m.status = loadingToast("Stashing changes...")
		return m, executeStashAll(m.repo, m.commitLimit, "graphkeeper: local cleanup")
	case state.ActionCleanWorkingTree:
		m.status = loadingToast("Cleaning working tree...")
		return m, executeCleanWorkingTree(m.repo, m.commitLimit, false)
	case state.ActionCheckout:
		target := m.status.Selected
		if target == "" {
			m.status = deriveStatus(m.repoStatus)
			return m, nil
		}
		m.status = loadingToast("Checking out...")
		return m, executeCheckout(m.repo, target, m.commitLimit)
	case state.ActionReset, state.ActionMerge, state.ActionRebase:
		target := m.status.Selected
		if action == state.ActionReset {
			mode := m.status.ResetMode
			if mode == "" {
				mode = state.ResetModeHard
			}
			m.status = loadingToast(strings.Title(string(mode)) + " reset...")
			return m, executeReset(m.repo, target, mode, m.commitLimit)
		} else if action == state.ActionMerge {
			m.status = loadingToast("Merging...")
		} else {
			m.status = loadingToast("Rebasing...")
		}
		return m, executeAction(m.repo, action, target, m.commitLimit)
	default:
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	}
}

func (m model) handleConfirmPullMerge() (tea.Model, tea.Cmd) {
	if m.status.Action == state.ActionPull && !m.pullIsFastForward {
		if m.activePullRequest == nil || m.pullConfirmStale {
			return m, nil
		}
		m.handshakeCommits = make(map[string]bool)
		m.status = loadingToast("Merging pull...")
		if m.pull != nil {
			return m, startPullWorkflow(&m, PullModeMerge)
		}
		return m, validateAndExecutePull(m.repo, m.commitLimit, *m.activePullRequest, PullModeMerge)
	}
	return m, nil
}

func (m model) handleConfirmPullRebase() (tea.Model, tea.Cmd) {
	if m.status.Action == state.ActionPull && !m.pullIsFastForward {
		if m.activePullRequest == nil || m.pullConfirmStale {
			return m, nil
		}
		m.handshakeCommits = make(map[string]bool)
		m.status = loadingToast("Rebasing pull...")
		if m.pull != nil {
			return m, startPullWorkflow(&m, PullModeRebase)
		}
		return m, validateAndExecutePull(m.repo, m.commitLimit, *m.activePullRequest, PullModeRebase)
	}
	return m, nil
}
