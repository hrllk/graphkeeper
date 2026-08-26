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
	case fetchedMsg, preparedMsg, pullCheckedMsg, previewMsg, graphActionCheckMsg, pushFetchedMsg, pullToastDoneMsg, branchToastDoneMsg:
		return handleFetchUpdate(m, msg)
	case pullFetchedMsg:
		outcome := classifyLegacyPullFetchedOutcome(m.activePullRequest, m.pullConfirmStale, msg)
		return reducePullLifecycleOutcome(m, outcome, func(m model) (tea.Model, tea.Cmd) {
			return handleFetchUpdate(m, msg)
		})
	case pullPreviewReadyMsg:
		outcome := classifyLegacyPullPreviewReadyOutcome(m.activePullRequest, m.pullConfirmStale, msg)
		return reducePullLifecycleOutcome(m, outcome, func(m model) (tea.Model, tea.Cmd) {
			return handleFetchUpdate(m, msg)
		})
	case pullPortPreviewMsg:
		return handlePullPortPreview(m, msg)
	case pullValidationMsg:
		outcome := classifyPullValidationOutcome(m.activePullRequest, m.pullConfirmStale, msg)
		return reducePullLifecycleOutcome(m, outcome, func(m model) (tea.Model, tea.Cmd) {
			if outcome.Kind != pullLifecycleConfirmationReady {
				clearPullConfirmProjection(&m)
				m.activePullRequest = nil
				m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
				return m, m.refreshCmd()
			}
			m.status = operationLoadingStatusFor(progressPull, "Pulling...", state.ActionPull)
			return m, executeValidatedPull(m.repo, m.commitLimit, *m.activePullRequest, msg.mode)
		})
	case pullWorkflowMsg:
		outcome := classifyPullWorkflowOutcome(m.activePullRequest, msg)
		return reducePullLifecycleOutcome(m, outcome, func(m model) (tea.Model, tea.Cmd) {
			m.pullCancel = nil
			if outcome.Kind == pullLifecycleRefreshFailed {
				clearPullConfirmProjection(&m)
				m.activePullRequest = nil
				m.status = state.New()
				m.status.Mode = state.ModeOperationResult
				m.status.Action = state.ActionPull
				m.status.Title = "PULL COMPLETED — STATE UNVERIFIED"
				m.status.Message = m.status.Title
				m.status.Detail = "Refresh failed. Press f to refresh repository state."
				return m, nil
			}
			if outcome.Kind == pullLifecycleRefreshIdentityIgnored || outcome.Kind == pullLifecycleCompleted {
				clearPullConfirmProjection(&m)
			}
			m.activePullRequest = nil
			if outcome.Kind == pullLifecycleExecutionFailed {
				clearPullConfirmProjection(&m)
				m.status = state.New().WithBlocked(state.BlockUnknown, "PULL FAILED", "Pull execution failed.")
				return m, nil
			}
			if outcome.Kind == pullLifecycleNoOpCompleted {
				clearPullConfirmProjection(&m)
				m.status = state.New()
				m.status.Mode = state.ModeOperationResult
				m.status.Action = state.ActionPull
				m.status.Title = "PULL COMPLETED"
				m.status.Message = "PULL COMPLETED"
				m.status.Detail = "No action needed. Press esc to return to the graph."
				return m, nil
			}
			if outcome.Kind == pullLifecyclePreviewStale || outcome.Kind == pullLifecycleValidationRejected {
				clearPullConfirmProjection(&m)
				m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
				return m, m.refreshCmd()
			}
			if outcome.Kind != pullLifecycleRefreshIdentityIgnored && msg.result.Refresh.RequestID == msg.result.RefreshRequestID && msg.result.Refresh.RepositoryEpoch == msg.result.RefreshEpoch && (msg.result.Refresh.Snapshot.Graph.Commits != nil || msg.result.Refresh.Snapshot.Graph.Branch != "" || msg.result.Refresh.Snapshot.Graph.Head != "") {
				m.graphReadSnapshot = msg.result.Refresh.Snapshot.Graph
				m.repoSnapshotLoaded = true
			}
			m.status = state.New()
			m.status.Mode = state.ModeOperationResult
			m.status.Action = state.ActionPull
			m.status.Title = "PULL COMPLETED"
			m.status.Message = "PULL COMPLETED"
			m.status.Detail = "Press esc to return to the graph."
			return m, nil
		})
	case pullExecutionResultMsg:
		outcome := classifyPullExecutionResultOutcome(m.activePullRequest, msg)
		return reducePullLifecycleOutcome(m, outcome, func(m model) (tea.Model, tea.Cmd) {
			m.activePullRequest = nil
			if outcome.Kind == pullLifecyclePreviewStale {
				clearPullConfirmProjection(&m)
				m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
				return m, m.refreshCmd()
			}
			return handlePullExecutionResult(m, msg)
		})
	case executedMsg:
		return handleExecutedUpdate(m, msg)
	case commitInspectorResultMsg:
		return m.applyCommitInspectorResult(msg)
	case ContinuationRequested:
		if m.commitInspectorStale || !m.commitInspectorOpen || msg.Commit != m.commitInspectorSnapshot.FullHash || msg.Parent != m.commitInspectorSnapshot.Parent || msg.FileID != m.commitInspectorDiffWindow.FileID || msg.RepositoryEpoch != m.commitInspectorEpoch || msg.RequestID != m.commitInspectorRequest || msg.Window != m.commitInspectorWindowRequest || m.commitInspectorLoading || m.commitInspectorMetadataLoading || m.commitInspectorDiffLoading {
			return m, nil
		}
		return startInspectorContinuation(m)
	case inspectorContinuationKeyMsg:
		if m.commitInspectorOpen && !m.commitInspectorStale && m.commitInspectorDiffWindow.HasMore && !m.commitInspectorLoading && !m.commitInspectorMetadataLoading && !m.commitInspectorDiffLoading {
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
