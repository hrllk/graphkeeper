package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.commitInspectorOpen {
		return m.handleCommitInspectorKey(msg)
	}
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
	if (m.startupReadPending || m.startupFailed) && (msg.String() == "q" || msg.String() == "ctrl+c") {
		return m, tea.Quit
	}
	if operationInputLocked(m) {
		return m, nil
	}
	switch m.status.Mode {
	case state.ModeOperationResult:
		return m.handleOperationResultKey(msg)
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

func (m model) handleOperationResultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.status = m.status.WithBrowse()
		return m, nil
	case "f":
		m.status.Message = "Refreshing repository state..."
		return m, m.refreshCmd()
	default:
		return m, nil
	}
}

func (m model) overlayOpen() bool {
	if m.commitInspectorOpen || m.graphStashPopOpen || m.stashMessageOpen || m.tagPopupOpen || m.stashPopupOpen ||
		m.branchOpen || m.hiddenHotkeysOpen || m.graphSearchOpen {
		return true
	}
	switch m.status.Mode {
	case state.ModeTargetPick, state.ModeCherryPickPick, state.ModeConfirm, state.ModeReview,
		state.ModeResetModePick, state.ModeOutcomePreview, state.ModeBlocked, state.ModeOperationResult:
		return true
	default:
		return false
	}
}
