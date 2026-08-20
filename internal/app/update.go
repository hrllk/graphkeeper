package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"hrllk/graphkeeper/internal/state"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return handleWindowSize(m, msg)
	case loadedMsg, refreshedMsg, tickMsg:
		return handleLifecycleUpdate(m, msg)
	case stashLoadedMsg:
		return handleStashUpdate(m, msg)
	case tagCreatedMsg, tagToastDoneMsg:
		return handleTagUpdate(m, msg)
	case fetchedMsg, preparedMsg, pullCheckedMsg, previewMsg, graphActionCheckMsg, pushFetchedMsg, pullFetchedMsg, pullPreviewReadyMsg, pullToastDoneMsg, branchToastDoneMsg:
		return handleFetchUpdate(m, msg)
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
