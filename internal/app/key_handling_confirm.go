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
		m.handshakeCommits = make(map[string]bool)
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
		if m.pullIsFastForward {
			m.status = loadingToast("Pulling...")
			return m, executePull(m.repo, m.commitLimit)
		}
		m.status = loadingToast("Merging pull...")
		return m, executePullMerge(m.repo, m.commitLimit)
	case state.ActionSetUpstream:
		m.status = loadingToast("Pushing and tracking...")
		return m, executePushSetUpstream(m.repo, m.repoStatus.Branch, m.commitLimit)
	case state.ActionForcePush:
		m.status = loadingToast("Force pushing...")
		return m, executeForcePush(m.repo, m.repoStatus.Branch, m.commitLimit)
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
		m.handshakeCommits = make(map[string]bool)
		m.status = loadingToast("Merging pull...")
		return m, executePullMerge(m.repo, m.commitLimit)
	}
	return m, nil
}

func (m model) handleConfirmPullRebase() (tea.Model, tea.Cmd) {
	if m.status.Action == state.ActionPull && !m.pullIsFastForward {
		m.handshakeCommits = make(map[string]bool)
		m.status = loadingToast("Rebasing pull...")
		return m, executePullRebase(m.repo, m.commitLimit)
	}
	return m, nil
}
