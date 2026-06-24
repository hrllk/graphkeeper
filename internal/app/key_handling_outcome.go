package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func (m model) handleOutcomePreviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "space", "enter":
		if !m.status.CanExecute {
			return m, nil
		}
		return m.handleOutcomePreviewExecute()
	case "esc":
		return m.handleOutcomePreviewEscape()
	default:
		return m, nil
	}
}

func (m model) handleOutcomePreviewExecute() (tea.Model, tea.Cmd) {
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
	default:
		return m, nil
	}
}

func (m model) handleOutcomePreviewEscape() (tea.Model, tea.Cmd) {
	switch {
	case m.status.Action == state.ActionPull || m.status.Action == state.ActionAbort:
		m.status = deriveStatus(m.repoStatus)
	default:
		m.status = actionPickTargets(m.repoStatus, m.status.Action)
	}
	return m, nil
}
