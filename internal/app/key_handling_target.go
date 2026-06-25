package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func moveTarget(s state.Status, delta int) state.Status {
	if s.Mode != state.ModeTargetPick || len(s.Targets) == 0 {
		return s
	}
	next := s.TargetIdx + delta
	if next < 0 {
		next = len(s.Targets) - 1
	}
	if next >= len(s.Targets) {
		next = 0
	}
	s.TargetIdx = next
	s.Selected = s.Targets[next].Ref
	return s
}

func (m model) handleTargetPickKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.status = moveTarget(m.status, -1)
	case "down", "j":
		m.status = moveTarget(m.status, 1)
	case "space", "enter":
		action := m.status.Action
		target := selectedTarget(m.status)
		if target == "" {
			m.status = state.New().WithBlocked(state.BlockTargetEmpty, "No target selected.", "Choose a branch, tag, or ref first.")
			return m, nil
		}
		m.status = state.New().WithLoading("Previewing result...")
		return m, previewSelection(m.repo, m.repoStatus, action, target)
	case "esc":
		m.handshakeCommits = make(map[string]bool)
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	default:
		return m, nil
	}
	return m, nil
}
