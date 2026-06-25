package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func (m model) handleResetModePickKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s":
		m.status.ResetMode = state.ResetModeSoft
		m.status.Message = "Soft reset."
		return m, nil
	case "m":
		m.status.ResetMode = state.ResetModeMixed
		m.status.Message = "Mixed reset."
		return m, nil
	case "h":
		m.status.ResetMode = state.ResetModeHard
		m.status.Message = "Hard reset."
		return m, nil
	case "enter", "space":
		target := m.status.Selected
		mode := m.status.ResetMode
		if mode == "" {
			mode = state.ResetModeMixed
		}
		m.status = state.New().WithLoading(strings.Title(string(mode)) + " reset...")
		return m, executeReset(m.repo, target, mode, m.commitLimit)
	case "esc":
		m.handshakeCommits = make(map[string]bool)
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	default:
		return m, nil
	}
}
