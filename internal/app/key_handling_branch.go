package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleBranchOpenKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		m.status = loadingToast("Creating branch...")
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
	return m, nil
}
