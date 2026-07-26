package app

import "hrllk/graphkeeper/internal/state"

func openGraphSearch(m model) model {
	m.graphSearchOpen = true
	m.graphSearchDraft = m.graphSearchQuery
	m.graphSearchIndex = buildGraphSearchIndex(m.repoStatus)
	m.graphSearchError = ""
	return m
}

func checkoutConfirmStatus(target string) state.Status {
	titleMsg := "Checkout branch?"
	status := state.New().WithConfirm(state.ActionCheckout, titleMsg, "Switch to "+target+".")
	status.Title = titleMsg
	status.Selected = target
	return status
}

func deleteConfirmStatus(action state.Action, title, detail, target string) state.Status {
	status := state.New().WithConfirm(action, title, detail)
	status.Title = title
	status.Selected = target
	return status
}

func deleteBranchConfirmStatus(selection branchDeleteSelection) state.Status {
	status := deleteConfirmStatus(state.ActionDeleteBranch, selection.title, selection.detail, selection.target)
	status.DeleteRemote = selection.remote
	return status
}
