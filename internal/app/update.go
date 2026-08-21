package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"hrllk/graphkeeper/internal/state"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return handleWindowSize(m, msg)
	case loadedMsg, refreshedMsg, loadedSnapshotMsg, refreshedSnapshotMsg, tickMsg:
		return handleLifecycleUpdate(m, msg)
	case stashLoadedMsg:
		return handleStashUpdate(m, msg)
	case tagCreatedMsg, tagToastDoneMsg:
		return handleTagUpdate(m, msg)
	case fetchedMsg, preparedMsg, pullCheckedMsg, previewMsg, graphActionCheckMsg, pushFetchedMsg, pullFetchedMsg, pullPreviewReadyMsg, pullToastDoneMsg, branchToastDoneMsg:
		return handleFetchUpdate(m, msg)
	case pullPortPreviewMsg:
		return handlePullPortPreview(m, msg)
	case pullValidationMsg:
		if !m.pullRequestMessageActive(msg.requestID, msg.requestEpoch) {
			return m, nil
		}
		if !samePullSnapshotIdentity(m.activePullRequest.OperationBaseline, msg.baseline) {
			m.activePullRequest = nil
			m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
			return m, m.refreshCmd()
		}
		if msg.err != nil || !msg.valid {
			m.activePullRequest = nil
			m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
			return m, m.refreshCmd()
		}
		m.status = loadingToast("Pulling...")
		return m, executeValidatedPull(m.repo, m.commitLimit, *m.activePullRequest, msg.mode)
	case pullWorkflowMsg:
		if m.activePullRequest == nil || msg.result.OperationRequestID != m.activePullRequest.ID || msg.result.OperationEpoch != m.activePullRequest.Epoch {
			return m, nil
		}
		m.pullCancel = nil
		if msg.result.RefreshErrorKind != ReadErrorNone {
			m.activePullRequest = nil
			m.status = state.New()
			m.status.Mode = state.ModeOperationResult
			m.status.Action = state.ActionPull
			m.status.Title = "PULL COMPLETED — STATE UNVERIFIED"
			m.status.Message = m.status.Title
			m.status.Detail = "Refresh failed. Press f to refresh repository state."
			return m, nil
		}
		m.activePullRequest = nil
		if msg.result.Execute.Reason != PullRejectNone || !msg.result.Execute.Succeeded {
			m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
			return m, m.refreshCmd()
		}
		if msg.result.Refresh.RequestID == msg.result.RefreshRequestID && msg.result.Refresh.RepositoryEpoch == msg.result.RefreshEpoch && (msg.result.Refresh.Snapshot.Graph.Commits != nil || msg.result.Refresh.Snapshot.Graph.Branch != "" || msg.result.Refresh.Snapshot.Graph.Head != "") {
			m.graphReadSnapshot = msg.result.Refresh.Snapshot.Graph
			m.repoSnapshotLoaded = true
		}
		m.status = state.New()
		m.status.Mode = state.ModeOperationResult
		m.status.Action = state.ActionPull
		m.status.Title = "PULL COMPLETED"
		m.status.Message = "PULL COMPLETED"
		m.status.Detail = "Press q or Esc to return to the graph."
		return m, nil
	case pullExecutionResultMsg:
		if !m.pullRequestMessageActive(msg.requestID, msg.requestEpoch) {
			return m, nil
		}
		if !samePullSnapshotIdentity(m.activePullRequest.OperationBaseline, msg.baseline) {
			m.activePullRequest = nil
			m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
			return m, m.refreshCmd()
		}
		m.activePullRequest = nil
		if msg.stale {
			m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
			return m, m.refreshCmd()
		}
		return handlePullExecutionResult(m, msg)
	case executedMsg:
		return handleExecutedUpdate(m, msg)
	case commitInspectorResultMsg:
		return m.applyCommitInspectorResult(msg)
	case ContinuationRequested:
		if msg.Commit != m.commitInspectorSnapshot.FullHash || msg.Parent != m.commitInspectorSnapshot.Parent || msg.FileID != m.commitInspectorDiffWindow.FileID || msg.RepositoryEpoch != m.commitInspectorEpoch || msg.RequestID != m.commitInspectorRequest || msg.Window != m.commitInspectorWindowRequest || m.commitInspectorLoading || m.commitInspectorMetadataLoading || m.commitInspectorDiffLoading {
			return m, nil
		}
		return startInspectorContinuation(m)
	case inspectorContinuationKeyMsg:
		if m.commitInspectorOpen && m.commitInspectorDiffWindow.HasMore && !m.commitInspectorLoading && !m.commitInspectorMetadataLoading && !m.commitInspectorDiffLoading {
			m.commitInspectorContinuationPending = false
			return startInspectorContinuation(m)
		}
		return m, nil
	case createdBranchMsg:
		return handleBranchUpdate(m, msg)
	case tea.KeyMsg:
		next, cmd := m.handleKeyMsg(msg)
		if cmd == nil {
			return next, nil
		}
		nextModel, ok := next.(model)
		if !ok {
			return next, cmd
		}
		// Inspector requests own their request/epoch identity. They must not be
		// advanced by the generic repository-operation epoch below.
		if nextModel.commitInspectorOpen {
			return nextModel, cmd
		}
		// A user operation starts a new repository epoch. Scheduled refreshes
		// use this value to reject reads started before the operation.
		nextModel.repositoryEpoch++
		return nextModel, cmd
	default:
		return m, nil
	}
}
