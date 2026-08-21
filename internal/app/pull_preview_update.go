package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"hrllk/graphkeeper/internal/state"
)

func handlePullPortPreview(m model, msg pullPortPreviewMsg) (tea.Model, tea.Cmd) {
	if !m.pullRequestMessageActive(msg.result.RequestID, msg.result.RepositoryEpoch) {
		return m, nil
	}
	if msg.err != nil || !msg.result.Eligible {
		m.activePullRequest = nil
		m.status = state.New().WithBlocked(state.BlockUnknown, "Pull unavailable.", "Refresh before pulling again.")
		return m, nil
	}
	m.activePullRequest.OperationBaseline = msg.result.Baseline
	m.activePullRequest.OperationBaselineSet = true
	m.pullIsFastForward = msg.result.Impact.IsFastForward
	if len(msg.result.Commits) == 0 {
		request := *m.activePullRequest
		m.activePullRequest = nil
		return completePullNoOp(m, m.repoStatus, request, PullModeNoOp)
	}
	m.status = state.New().WithConfirm(state.ActionPull, "Fast-forward pull?", msg.result.Impact.Summary)
	m.status.Title = "Fast-forward pull?"
	if !msg.result.Impact.IsFastForward {
		m.status = m.status.WithConfirm(state.ActionPull, "Choose pull mode", msg.result.Impact.Summary+"\n\n m: merge\n r: rebase\n esc: cancel")
		m.status.Title = "Choose pull mode"
	}
	return m, nil
}
