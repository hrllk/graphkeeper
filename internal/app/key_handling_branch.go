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
		m.branchError = ""
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.branchDraft)
		base := m.branchBase
		if err := branchCreateValidationError(m.repoStatus, name, base); err != nil {
			if branchNameExists(m.repoStatus, name) {
				m.branchError = "Branch name already exists."
				m.status = loadingToast("Enter a branch name.")
				return m, nil
			}
			m.branchOpen = false
			m.branchDraft = ""
			m.branchError = ""
			m.status = branchCreateBlockedStatusFromError(err)
			return m, nil
		}
		m.branchOpen = false
		m.branchDraft = ""
		m.branchError = ""
		m.status = loadingToast("Creating branch...")
		return m, createBranch(m.repo, name, base, m.commitLimit)
	case "backspace":
		if len(m.branchDraft) > 0 {
			runes := []rune(m.branchDraft)
			m.branchDraft = string(runes[:len(runes)-1])
		}
		m.branchError = ""
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.branchDraft += string(msg.Runes)
			m.branchError = ""
			return m, nil
		}
	}
	return m, nil
}
