package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

func startPullPreview(m *model, request pullRequest) tea.Cmd {
	if m.pull == nil {
		return nil
	}
	return func() tea.Msg {
		result, err := m.pull.Preview(context.Background(), PullPreviewRequest{RequestID: request.ID, RepositoryEpoch: request.Epoch, Baseline: request.FetchBaseline, Mode: PullModeMerge, CommitLimit: m.commitLimit})
		return pullPortPreviewMsg{result: result, err: err}
	}
}
