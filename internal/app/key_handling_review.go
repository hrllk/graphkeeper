package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleReviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m.handleReviewAccept()
	case "n", "esc":
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	default:
		return m, nil
	}
}

func (m model) handleReviewAccept() (tea.Model, tea.Cmd) {
	action := m.status.Action
	target := m.status.Selected
	if target == "" {
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	}
	m.status = buildGraphActionConfirmStatus(action, m.repoStatus, target)
	return m, nil
}
