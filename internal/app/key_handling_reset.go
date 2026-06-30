package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func (m model) handleResetModePickKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s":
		return m.executeResetModePick(state.ResetModeSoft)
	case "m":
		return m.executeResetModePick(state.ResetModeMixed)
	case "h":
		return m.executeResetModePick(state.ResetModeHard)
	case "esc":
		m.handshakeCommits = make(map[string]bool)
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	default:
		return m, nil
	}
}

func (m model) executeResetModePick(mode state.ResetMode) (tea.Model, tea.Cmd) {
	target := m.status.Selected
	if target == "" {
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	}
	if mode == "" {
		mode = state.ResetModeMixed
	}
	m.status.ResetMode = mode
	switch mode {
	case state.ResetModeSoft:
		m.status = loadingToast("Soft reset...")
	case state.ResetModeMixed:
		m.status = loadingToast("Mixed reset...")
	default:
		m.status = loadingToast("Hard reset...")
	}
	m.status.ResetMode = mode
	return m, executeReset(m.repo, target, mode, m.commitLimit)
}
