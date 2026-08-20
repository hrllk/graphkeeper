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
	case commitInspectorLoadedMsg:
		if !m.commitInspectorOpen || msg.request != m.commitInspectorRequest || msg.epoch != m.commitInspectorEpoch {
			return m, nil
		}
		m.commitInspectorMetadataLoading = false
		if msg.err != nil {
			m.commitInspectorLoading = false
			m.commitInspectorError = msg.err.Error()
			return m, nil
		}
		m.commitInspector = msg.inspection
		m.commitInspectorCursor = 0
		m.commitInspectorScroll = 0
		m.commitInspectorError = ""
		if len(msg.inspection.Files) > 0 {
			return m.startInspectorDiff()
		}
		m.commitInspectorLoading = false
		return m, nil
	case commitInspectorDiffMsg:
		if !m.commitInspectorOpen || msg.request != m.commitInspectorRequest || msg.epoch != m.commitInspectorEpoch {
			return m, nil
		}
		m.commitInspectorLoading = false
		m.commitInspectorDiffLoading = false
		m.commitInspectorCancel = nil
		if msg.err != nil {
			m.commitInspectorDiffError = msg.err.Error()
			return m, nil
		}
		m.commitInspectorLines = msg.diff.Lines
		m.commitInspectorScroll = 0
		m.commitInspectorHasMore = msg.diff.HasMore
		m.commitInspectorDiffError = ""
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
