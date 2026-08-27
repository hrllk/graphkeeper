package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

type commitInspectorResultMsg struct {
	Result   InspectorResult[DiffWindow]
	Metadata *InspectorResult[CommitSnapshot]
}

func inspectCommitCommand(ctx context.Context, m model, req CommitRequest) tea.Cmd {
	reader := m.inspector()
	return func() tea.Msg {
		if ctx == nil {
			ctx = context.Background()
		}
		result := reader.InspectCommit(ctx, req)
		return commitInspectorResultMsg{Metadata: &result}
	}
}

func loadCommitInspectorDiffCommand(ctx context.Context, m model, req DiffRequest) tea.Cmd {
	reader := m.inspector()
	return func() tea.Msg {
		if ctx == nil {
			ctx = context.Background()
		}
		return commitInspectorResultMsg{Result: reader.LoadDiff(ctx, req)}
	}
}

func (m model) applyCommitInspectorResult(msg commitInspectorResultMsg) (model, tea.Cmd) {
	if msg.Metadata != nil {
		result := *msg.Metadata
		if !m.commitInspectorOpen || result.RequestID != m.commitInspectorRequest || result.RepositoryEpoch != m.commitInspectorEpoch || result.Commit != m.commitInspectorRequestedCommit {
			return m, nil
		}
		m.commitInspectorMetadataLoading = false
		if result.State == PaneCanceled {
			m.commitInspectorLoading = false
			return m, nil
		}
		if result.State == PaneError || result.Error != nil {
			m.commitInspectorLoading = false
			if result.Error != nil {
				m.commitInspectorError = result.Error.Message
			} else {
				m.commitInspectorError = "commit metadata could not be loaded"
			}
			return m, nil
		}
		m.commitInspectorSnapshot = result.Value
		m.commitInspector = result.Value
		m.commitInspectorCursor, m.commitInspectorScroll = 0, 0
		if len(result.Value.Files) == 0 {
			m.commitInspectorLoading = false
			return m, nil
		}
		return m.startInspectorDiffFromReader()
	}
	result := msg.Result
	// Request identity only. The payload is checked after the state is known:
	// every adapter error path returns a zero Value, so folding Value.FileID into
	// this guard discarded error results and left the pane loading forever.
	if !m.commitInspectorOpen || result.RequestID != m.commitInspectorRequest || result.RepositoryEpoch != m.commitInspectorEpoch || result.Commit != m.commitInspectorSnapshot.FullHash || result.Parent != m.commitInspectorSnapshot.Parent || result.FileID != m.currentInspectorFileID() || result.Window != m.commitInspectorWindowRequest {
		return m, nil
	}
	m.commitInspectorDiffLoading = false
	if result.State == PaneCanceled {
		m.commitInspectorLoading = false
		return m, nil
	}
	m.commitInspectorLoading = false
	if result.State == PaneError || result.Error != nil {
		if result.Error != nil {
			m.commitInspectorDiffError = result.Error.Message
		} else {
			m.commitInspectorDiffError = "diff could not be loaded"
		}
		return m, nil
	}
	if result.Value.FileID != result.FileID {
		m.commitInspectorDiffError = "diff response did not match the selected file"
		return m, nil
	}
	m.commitInspectorDiffWindow = result.Value
	m.commitInspectorHasMore = result.Value.HasMore
	return m, nil
}

func startInspectorContinuation(m model) (model, tea.Cmd) {
	if !m.commitInspectorDiffWindow.HasMore || m.commitInspectorSnapshot.FullHash == "" {
		return m, nil
	}
	m = m.cancelInspector()
	m.commitInspectorRequest++
	m.commitInspectorContext, m.commitInspectorCancel = context.WithCancel(context.Background())
	window := m.commitInspectorWindowRequest
	window.StartLine = m.commitInspectorDiffWindow.NextStartLine
	if window.MaxLines == 0 {
		window.MaxLines = 2000
	}
	if window.MaxBytes == 0 {
		window.MaxBytes = 1 << 20
	}
	m.commitInspectorWindowRequest = window
	m.commitInspectorDiffLoading = true
	m.commitInspectorLoading = true
	return m, loadCommitInspectorDiffCommand(m.commitInspectorContext, m, DiffRequest{Commit: m.commitInspectorSnapshot.FullHash, Parent: m.commitInspectorSnapshot.Parent, FileID: m.commitInspectorDiffWindow.FileID, RequestID: m.commitInspectorRequest, RepositoryEpoch: m.commitInspectorEpoch, Window: window})
}

func (m model) startInspectorDiffFromReader() (model, tea.Cmd) {
	if m.commitInspectorCursor < 0 || m.commitInspectorCursor >= len(m.commitInspectorSnapshot.Files) {
		return m, nil
	}
	file := m.commitInspectorSnapshot.Files[m.commitInspectorCursor]
	window := DiffWindowRequest{StartLine: 0, MaxLines: 2000, MaxBytes: 1 << 20}
	m.commitInspectorWindowRequest = window
	m.commitInspectorDiffLoading = true
	return m, loadCommitInspectorDiffCommand(m.commitInspectorContext, m, DiffRequest{Commit: m.commitInspectorSnapshot.FullHash, Parent: m.commitInspectorSnapshot.Parent, FileID: file.StableID, RequestID: m.commitInspectorRequest, RepositoryEpoch: m.commitInspectorEpoch, Window: window})
}

func (m model) currentInspectorFileID() string {
	if m.commitInspectorCursor < 0 || m.commitInspectorCursor >= len(m.commitInspectorSnapshot.Files) {
		return ""
	}
	return m.commitInspectorSnapshot.Files[m.commitInspectorCursor].StableID
}
