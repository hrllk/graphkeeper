package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

func startPullWorkflow(m *model, mode PullMode) tea.Cmd {
	if m.pull == nil || m.repositoryRead == nil || m.activePullRequest == nil {
		return nil
	}
	if m.pullCancel != nil {
		m.pullCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.pullCancel = cancel
	m.nextPullRequestID++
	m.activePullRequest.ID = m.nextPullRequestID
	req := PullExecutionRequest{RequestID: m.activePullRequest.ID, RepositoryEpoch: m.activePullRequest.Epoch, Authorized: true, AuthorizedBaseline: m.activePullRequest.OperationBaseline}
	return func() tea.Msg {
		return pullWorkflowMsg{result: runPullWorkflow(ctx, m.pull, m.repositoryRead, req, mode, m.commitLimit)}
	}
}
