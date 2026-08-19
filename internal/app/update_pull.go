package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func handlePullExecutionResult(m model, msg pullExecutionResultMsg) (tea.Model, tea.Cmd) {
	return handleExecutedUpdate(m, executedMsg{action: msg.action, status: msg.status, err: msg.err})
}
