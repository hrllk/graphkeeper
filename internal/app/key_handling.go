package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.branchOpen {
		return m.handleBranchOpenKey(msg)
	}
	switch m.status.Mode {
	case state.ModeTargetPick:
		return m.handleTargetPickKey(msg)
	case state.ModeConfirm:
		return m.handleConfirmKey(msg)
	case state.ModeOutcomePreview:
		return m.handleOutcomePreviewKey(msg)
	case state.ModeBrowse:
		return m.handleBrowseKey(msg)
	default:
		return m, nil
	}
}
