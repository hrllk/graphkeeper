package app

import "hrllk/graphkeeper/internal/state"

type shellOverlay struct {
	name   string
	active bool
	popup  string
}

func applyShellOverlays(m model, body string, bodyWidth, bodyHeight int) string {
	for _, overlay := range shellOverlayStack(m, bodyWidth, bodyHeight) {
		if !overlay.active {
			continue
		}
		body = overlayPopup(body, overlay.popup)
	}
	return body
}

func shellOverlayStack(m model, bodyWidth, bodyHeight int) []shellOverlay {
	return []shellOverlay{
		{
			name:   "confirm",
			active: m.status.Mode == state.ModeConfirm,
			popup:  renderConfirmPopup(m, bodyWidth),
		},
		{
			name:   "review",
			active: m.status.Mode == state.ModeReview,
			popup:  renderReviewPopup(m, bodyWidth),
		},
		{
			name:   "reset-mode",
			active: m.status.Mode == state.ModeResetModePick,
			popup:  renderResetModePopup(bodyWidth),
		},
		{
			name:   "cherry-pick",
			active: m.status.Mode == state.ModeCherryPickPick,
			popup:  renderCherryPickPopup(m, bodyWidth),
		},
		{
			name:   "target-pick",
			active: m.status.Mode == state.ModeTargetPick,
			popup:  renderTargetPickPopup(m, bodyWidth),
		},
		{
			name:   "branch-input",
			active: m.branchOpen,
			popup:  renderBranchInputPopup(m, bodyWidth),
		},
		{
			name:   "stash-message",
			active: m.stashMessageOpen,
			popup:  renderStashMessagePopup(m, bodyWidth),
		},
		{
			name:   "graph-stash-pop",
			active: m.graphStashPopOpen,
			popup:  renderGraphStashPopPopup(m, bodyWidth, bodyHeight),
		},
		{
			name:   "stash-popup",
			active: m.stashPopupOpen,
			popup:  renderStashPopup(m, bodyWidth, bodyHeight),
		},
		{
			name:   "tag-popup",
			active: m.tagPopupOpen,
			popup:  renderTagPopup(m, bodyWidth, bodyHeight),
		},
		{
			name:   "hidden-hotkeys",
			active: m.hiddenHotkeysOpen,
			popup:  renderHiddenHotkeysPopup(m, bodyWidth),
		},
		{
			name:   "graph-search",
			active: m.graphSearchOpen,
			popup:  renderGraphSearchPopup(m, bodyWidth),
		},
		{
			name:   "loading",
			active: m.status.Mode == state.ModeLoading && !m.branchOpen,
			popup:  renderLoadingPopup(m, bodyWidth),
		},
		{
			name:   "blocked",
			active: m.status.Mode == state.ModeBlocked && !m.branchOpen,
			popup:  renderAlertPopup(blockedAlertContent(m.status), bodyWidth),
		},
	}
}
