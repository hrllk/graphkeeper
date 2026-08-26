package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

type commitInspectorResultMsg struct {
	Result   InspectorResult[DiffWindow]
	Metadata *InspectorResult[CommitSnapshot]
}

func startCommitInspectorRevalidation(m model) (model, tea.Cmd) {
	if !m.commitInspectorOpen {
		return m, nil
	}
	m = m.cancelInspector()
	m.commitInspectorRequest++
	m.commitInspectorContext, m.commitInspectorCancel = context.WithCancel(context.Background())
	m.commitInspectorRevalidating = true
	m.commitInspectorMetadataLoading = true
	m.commitInspectorDiffLoading = false
	m.commitInspectorLoading = true
	m.commitInspectorContinuationPending = false
	return m, inspectCommitCommand(m.commitInspectorContext, m, CommitRequest{Commit: m.commitInspectorRequestedCommit, RequestID: m.commitInspectorRequest, RepositoryEpoch: m.commitInspectorEpoch})
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
			m.commitInspectorRevalidating = false
			return m, nil
		}
		if result.State == PaneError || result.Error != nil {
			m.commitInspectorLoading = false
			m.commitInspectorRevalidating = false
			if result.Error != nil && (result.Error.Kind == "commit_not_found" || result.Error.Kind == "file_not_found") {
				return closeInspectorWithRecoveryAlert(m, result.Error.Kind)
			}
			if result.Error != nil {
				m.commitInspectorError = result.Error.Message
			}
			return m, nil
		}

		m.commitInspectorError = ""
		selectedID := m.commitInspectorSelectedFileID
		selectedKey := m.commitInspectorSelectedCanonicalKey
		selectedIndex := -1
		if m.commitInspectorRevalidating {
			stableMatches := 0
			for i, file := range result.Value.Files {
				if selectedID != "" && file.StableID == selectedID {
					selectedIndex, stableMatches = i, stableMatches+1
				}
			}
			if stableMatches > 1 {
				m.commitInspectorLoading = false
				m.commitInspectorRevalidating = false
				m.commitInspectorError = "duplicate selected file identity"
				return m, nil
			}
			if stableMatches != 1 && selectedKey != "" {
				selectedIndex = -1
				canonicalMatches := 0
				for i, file := range result.Value.Files {
					if file.CanonicalKey == selectedKey {
						selectedIndex, canonicalMatches = i, canonicalMatches+1
					}
				}
				if canonicalMatches > 1 {
					m.commitInspectorLoading = false
					m.commitInspectorRevalidating = false
					m.commitInspectorError = "ambiguous selected file identity"
					return m, nil
				}
			}
			if selectedIndex < 0 {
				m.commitInspectorLoading = false
				m.commitInspectorRevalidating = false
				return closeInspectorWithRecoveryAlert(m, "file_not_found")
			}
		} else if len(result.Value.Files) > 0 {
			selectedIndex = 0
		}

		m.commitInspectorSnapshot = result.Value
		m.commitInspector = result.Value
		m.commitInspectorCursor = max(selectedIndex, 0)
		m.commitInspectorScroll = 0
		if len(result.Value.Files) == 0 {
			m.commitInspectorLoading = false
			m.commitInspectorRevalidating = false
			return m, nil
		}
		file := result.Value.Files[m.commitInspectorCursor]
		m.commitInspectorSelectedFileID = file.StableID
		m.commitInspectorSelectedCanonicalKey = file.CanonicalKey
		m.commitInspectorRevalidating = false
		return m.startInspectorDiffFromReader()
	}
	result := msg.Result
	if !m.commitInspectorOpen || result.RequestID != m.commitInspectorRequest || result.RepositoryEpoch != m.commitInspectorEpoch || result.Commit != m.commitInspectorSnapshot.FullHash || result.Parent != m.commitInspectorSnapshot.Parent || result.FileID != m.currentInspectorFileID() || result.Value.FileID != result.FileID || result.Window != m.commitInspectorWindowRequest {
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
		}
		return m, nil
	}
	m.commitInspectorDiffError = ""
	m.commitInspectorDiffWindow = result.Value
	m.commitInspectorHasMore = result.Value.HasMore
	return m, nil
}

func startInspectorContinuation(m model) (model, tea.Cmd) {
	if m.commitInspectorStale || !m.commitInspectorOpen || !m.commitInspectorDiffWindow.HasMore || m.commitInspectorLoading || m.commitInspectorMetadataLoading || m.commitInspectorDiffLoading || m.commitInspectorSnapshot.FullHash == "" {
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
	return m, loadCommitInspectorDiffCommand(m.commitInspectorContext, m, DiffRequest{Commit: m.commitInspectorSnapshot.FullHash, Parent: m.commitInspectorSnapshot.Parent, FileID: m.commitInspectorDiffWindow.FileID, CanonicalKey: m.currentInspectorCanonicalKey(), RequestID: m.commitInspectorRequest, RepositoryEpoch: m.commitInspectorEpoch, Window: window})
}

func (m model) startInspectorDiffFromReader() (model, tea.Cmd) {
	if m.commitInspectorCursor < 0 || m.commitInspectorCursor >= len(m.commitInspectorSnapshot.Files) {
		return m, nil
	}
	m.commitInspectorRequest++
	file := m.commitInspectorSnapshot.Files[m.commitInspectorCursor]
	window := DiffWindowRequest{StartLine: 0, MaxLines: 2000, MaxBytes: 1 << 20}
	m.commitInspectorWindowRequest = window
	m.commitInspectorDiffLoading = true
	return m, loadCommitInspectorDiffCommand(m.commitInspectorContext, m, DiffRequest{Commit: m.commitInspectorSnapshot.FullHash, Parent: m.commitInspectorSnapshot.Parent, FileID: file.StableID, CanonicalKey: file.CanonicalKey, RequestID: m.commitInspectorRequest, RepositoryEpoch: m.commitInspectorEpoch, Window: window})
}

func (m model) currentInspectorCanonicalKey() string {
	if m.commitInspectorSelectedCanonicalKey != "" {
		return m.commitInspectorSelectedCanonicalKey
	}
	if m.commitInspectorCursor < 0 || m.commitInspectorCursor >= len(m.commitInspectorSnapshot.Files) {
		return ""
	}
	return m.commitInspectorSnapshot.Files[m.commitInspectorCursor].CanonicalKey
}

func (m model) currentInspectorFileID() string {
	if m.commitInspectorSelectedFileID != "" {
		return m.commitInspectorSelectedFileID
	}
	if m.commitInspectorCursor < 0 || m.commitInspectorCursor >= len(m.commitInspectorSnapshot.Files) {
		return ""
	}
	return m.commitInspectorSnapshot.Files[m.commitInspectorCursor].StableID
}
