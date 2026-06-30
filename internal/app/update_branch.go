package app

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
	"hrllk/graphkeeper/internal/telemetry"
)

func handleBranchUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	msg2, ok := msg.(createdBranchMsg)
	if ok {
		if msg2.err != nil {
			m.branchOpen = false
			m.branchError = ""
			reason := state.BlockUnknown
			message := "Branch creation failed."
			detail := msg2.err.Error()
			if strings.Contains(msg2.err.Error(), "working tree is not clean") {
				reason = state.BlockDirtyTree
				message = "Working tree is dirty."
				detail = "Commit or stash changes first."
			} else if strings.Contains(msg2.err.Error(), "merge/rebase already in progress") {
				message = "Merge/rebase already in progress."
				detail = "Abort or resolve it before creating a branch."
			} else if strings.Contains(msg2.err.Error(), "branch base is empty") {
				reason = state.BlockTargetEmpty
				message = "No branch base."
				detail = "Select a commit or branch first."
			} else if strings.Contains(msg2.err.Error(), "branch name is empty") {
				message = "Branch name is empty."
				detail = "Enter a branch name."
			} else if strings.Contains(msg2.err.Error(), "branch name already exists") {
				message = "Branch name already exists."
				detail = "Choose a different branch name."
			}
			m.status = state.New().WithBlocked(reason, message, detail)
			telemetry.Log("app", "branch_create_failed", map[string]string{"name": msg2.name, "base": msg2.base, "error": msg2.err.Error()})
			return m, nil
		}
		m.branchOpen = false
		m.branchError = ""
		m.repoStatus = msg2.status
		syncBrowseState(&m, msg2.status)
		m.status = loadingToast("Branch created.")
		telemetry.Log("app", "branch_create", map[string]string{"name": msg2.name, "base": msg2.base})
		return m, tea.Tick(900*time.Millisecond, func(time.Time) tea.Msg {
			return branchToastDoneMsg{}
		})
	}
	return m, nil
}
