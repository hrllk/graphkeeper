package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func handlePullPortPreview(m model, msg pullPortPreviewMsg) (tea.Model, tea.Cmd) {
	outcome := classifyPullPortPreviewOutcome(m.activePullRequest, msg.result.RequestID, msg.result.RepositoryEpoch, m.pullConfirmStale, pullSnapshotIdentity(m.repoStatus, msg.result.RepositoryEpoch), msg.result, msg.err)
	if outcome.Kind == pullLifecyclePreviewStale {
		clearPullConfirmProjection(&m)
		m.activePullRequest = nil
		m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
		return m, m.refreshCmd()
	}
	return reducePullLifecycleOutcome(m, outcome, func(m model) (tea.Model, tea.Cmd) {
		return handlePullPortPreviewActive(m, msg)
	})
}

func handlePullPortPreviewActive(m model, msg pullPortPreviewMsg) (tea.Model, tea.Cmd) {
	if !m.pullRequestMessageActive(msg.result.RequestID, msg.result.RepositoryEpoch) {
		return m, nil
	}
	if msg.err != nil || !msg.result.Eligible {
		clearPullConfirmProjection(&m)
		m.activePullRequest = nil
		m.status = state.New().WithBlocked(state.BlockUnknown, "Pull unavailable.", "Refresh before pulling again.")
		return m, nil
	}
	m.activePullRequest.OperationBaseline = msg.result.Baseline
	m.activePullRequest.OperationBaselineSet = true
	local := pullSnapshotIdentity(m.repoStatus, m.activePullRequest.Epoch)
	if !samePullSnapshotIdentity(local, msg.result.Baseline) {
		clearPullConfirmProjection(&m)
		m.activePullRequest = nil
		m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
		return m, m.refreshCmd()
	}
	localSnapshot := pullImpactSnapshot(local, *m.activePullRequest)
	localImpact := pullImpactSet(localSnapshot)
	m.pullIsFastForward = local.Ahead == 0 && local.Behind > 0 && local.TrackingFresh && local.TrackingKnown
	if len(msg.result.Commits) == 0 {
		request := *m.activePullRequest
		m.activePullRequest = nil
		return completePullNoOp(m, m.repoStatus, request, PullModeNoOp)
	}
	if !localImpact.Valid {
		clearPullConfirmProjection(&m)
		m.activePullRequest = nil
		m.status = state.New().WithBlocked(state.BlockUnknown, "Pull impact unavailable.", "Refresh before pulling again.")
		return m, m.refreshCmd()
	}
	m.status = state.New().WithConfirm(state.ActionPull, "Pull?", localImpact.MergeSummary)
	m.status.Title = "Pull into " + local.Branch + "?"
	if !m.pullIsFastForward && !applyMergeConfirmProjection(&m, localSnapshot, localImpact, m.pullConfirmStale) {
		m.activePullRequest = nil
		m.status = state.New().WithBlocked(state.BlockUnknown, "Pull impact unavailable.", "Refresh before pulling again.")
		return m, m.refreshCmd()
	}
	return m, nil
}
