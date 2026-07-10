package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.graphStashPopOpen {
		return m.handleGraphStashPopKey(msg)
	}
	if m.stashMessageOpen {
		return m.handleStashMessageKey(msg)
	}
	if m.tagPopupOpen {
		return m.handleTagPopupKey(msg)
	}
	if m.stashPopupOpen {
		return m.handleStashPopupKey(msg)
	}
	if m.branchOpen {
		return m.handleBranchOpenKey(msg)
	}
	if m.hiddenHotkeysOpen {
		return m.handleHiddenHotkeysKey(msg)
	}
	if m.graphSearchOpen {
		return m.handleGraphSearchKey(msg)
	}
	switch m.status.Mode {
	case state.ModeTargetPick:
		return m.handleTargetPickKey(msg)
	case state.ModeCherryPickPick:
		return m.handleCherryPickPickKey(msg)
	case state.ModeConfirm:
		return m.handleConfirmKey(msg)
	case state.ModeReview:
		return m.handleReviewKey(msg)
	case state.ModeResetModePick:
		return m.handleResetModePickKey(msg)
	case state.ModeOutcomePreview:
		return m.handleOutcomePreviewKey(msg)
	case state.ModeBlocked:
		return m.handleBlockedKey(msg)
	case state.ModeBrowse:
		return m.handleBrowseKey(msg)
	default:
		return m, nil
	}
}
