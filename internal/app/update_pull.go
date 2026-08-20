package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func completePullNoOp(m model, status git.Status, request pullRequest, mode PullMode) (tea.Model, tea.Cmd) {
	if mode == "" {
		mode = PullModeNoOp
	}
	identity := pullSnapshotIdentity(status, request.Epoch)
	result := classifyOperationResult(OperationResultInput{
		Operation: PullResultMetadata{Action: state.ActionPull, Mode: mode, Branch: identity.Branch, Upstream: identity.Upstream},
		Before:    SnapshotObservation{Valid: request.OperationBaselineSet, Identity: request.OperationBaseline},
		After:     SnapshotObservation{Valid: true, Identity: identity},
	})
	m.operationResult = &result
	m.repoStatus = m.withCachedTagEntries(status)
	m.storeTagEntries(m.repoStatus)
	syncBrowseState(&m, m.repoStatus)
	m.status = state.New()
	m.status.Mode = state.ModeOperationResult
	m.status.Action = state.ActionPull
	m.status.Title = "PULL COMPLETED"
	m.status.Message = "PULL COMPLETED"
	m.status.Detail = "No action needed. Press q or Esc to return to the graph."
	return m, nil
}

func handlePullExecutionResult(m model, msg pullExecutionResultMsg) (tea.Model, tea.Cmd) {
	m.lastPullMode = msg.mode
	m.lastPullOperationBaseline = msg.operationBaseline
	input := OperationResultInput{
		Operation:      PullResultMetadata{Action: msg.action, Mode: msg.mode, Branch: msg.operationBaseline.Branch, Upstream: msg.operationBaseline.Upstream},
		Before:         SnapshotObservation{Valid: msg.operationBaselineSet, Identity: msg.operationBaseline},
		ExecutionError: msg.executionErr,
		RefreshError:   msg.refreshErr,
	}
	if msg.status.Root != "" {
		input.After = SnapshotObservation{Valid: msg.refreshErr == nil, Identity: pullSnapshotIdentity(msg.status, msg.requestEpoch)}
	}
	result := classifyOperationResult(input)
	m.operationResult = &result
	if msg.status.Root != "" {
		msg.status = m.withCachedTagEntries(msg.status)
		m.repoStatus = msg.status
		m.storeTagEntries(msg.status)
		syncBrowseState(&m, msg.status)
	}
	if result.Headline == "PULL FAILED" {
		m.status = state.New().WithBlocked(state.BlockUnknown, result.Headline, result.ExecutionError.Error())
		return m, nil
	}
	if result.Repository == RepositoryConflicted {
		m.status = state.New().WithBlocked(state.BlockUnknown, result.Headline, "Resolve conflicts outside Graphkeeper, then refresh state.")
		return m, nil
	}
	m.status = state.New()
	m.status.Mode = state.ModeOperationResult
	m.status.Action = state.ActionPull
	m.status.Title = result.Headline
	m.status.Message = result.Headline
	if result.Verification == VerificationUnknown {
		m.status.Detail = "Press f to refresh repository state."
	} else {
		m.status.Detail = "Press q or Esc to return to the graph."
	}
	return m, nil
}
