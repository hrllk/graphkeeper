package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
	"hrllk/graphkeeper/internal/telemetry"
)

func handleBranchUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	msg2, ok := msg.(createdBranchMsg)
	if !ok {
		return m, nil
	}
	if msg2.err != nil {
		m.branchOpen = false
		reason := state.BlockUnknown
		message := "Branch creation failed."
		detail := msg2.err.Error()
		if strings.Contains(msg2.err.Error(), "working tree is not clean") {
			reason = state.BlockDirtyTree
			message = "Working tree is dirty."
			detail = "Commit or stash changes first."
		}
		m.status = state.New().WithBlocked(reason, message, detail)
		telemetry.Log("app", "branch_create_failed", map[string]string{"name": msg2.name, "base": msg2.base, "error": msg2.err.Error()})
		return m, nil
	}
	m.branchOpen = false
	m.repoStatus = msg2.status
	syncBrowseState(&m, msg2.status)
	m.status = deriveStatus(msg2.status)
	telemetry.Log("app", "branch_create", map[string]string{"name": msg2.name, "base": msg2.base})
	return m, nil
}
