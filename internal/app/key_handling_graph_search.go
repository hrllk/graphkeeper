package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleGraphSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.graphSearchOpen = false
		m.graphSearchDraft = ""
		m.graphSearchError = ""
		return m, nil
	case "enter":
		return applyGraphSearchSelection(m), nil
	case "backspace":
		if len(m.graphSearchDraft) > 0 {
			runes := []rune(m.graphSearchDraft)
			m.graphSearchDraft = string(runes[:len(runes)-1])
			m.graphSearchError = ""
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.graphSearchDraft += string(msg.Runes)
			m.graphSearchError = ""
			return m, nil
		}
	}
	return m, nil
}
