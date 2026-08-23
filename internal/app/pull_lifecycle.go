package app

import tea "github.com/charmbracelet/bubbletea"

// pullLifecycleOutcomeKind is the neutral classification shared by guarded
// pull message families. IdentityIgnored is intentionally silent: callers
// must not mutate model state or schedule a command for a stale message.
type pullLifecycleOutcomeKind uint8

const (
	pullLifecycleIdentityIgnored pullLifecycleOutcomeKind = iota
	pullLifecycleActive
	pullLifecyclePreviewUnavailable
	pullLifecyclePreviewStale
	pullLifecycleNoOpCompleted
	pullLifecycleConfirmationReady
	pullLifecycleFetchReady
	pullLifecycleValidationRejected
	pullLifecycleExecutionFailed
	pullLifecycleRefreshFailed
	pullLifecycleRefreshIdentityIgnored
	pullLifecycleCompleted
)

type pullLifecycleOutcome struct {
	Kind                 pullLifecycleOutcomeKind
	RequestID            uint64
	RepositoryEpoch      uint64
	FetchBaseline        PullSnapshotIdentity
	OperationBaseline    PullSnapshotIdentity
	OperationBaselineSet bool
}

// reducePullLifecycleOutcome is deliberately only a guarded dispatch boundary.
// The callback owns all model mutation and concrete command construction for its
// source path; the neutral boundary never invents either.
func reducePullLifecycleOutcome(m model, outcome pullLifecycleOutcome, apply func(model) (tea.Model, tea.Cmd)) (tea.Model, tea.Cmd) {
	if outcome.Kind == pullLifecycleIdentityIgnored {
		return m, nil
	}
	return apply(m)
}

func classifyPullPortPreviewOutcome(req *pullRequest, requestID, repositoryEpoch uint64, confirmStale bool, local PullSnapshotIdentity, result PullPreviewResult, err error) pullLifecycleOutcome {
	identity := classifyPullLifecycleIdentity(req, requestID, repositoryEpoch, confirmStale)
	if identity.Kind == pullLifecycleIdentityIgnored {
		return identity
	}
	if err != nil || !result.Eligible {
		identity.Kind = pullLifecyclePreviewUnavailable
		return identity
	}
	if !samePullSnapshotIdentity(req.FetchBaseline, result.Baseline) || !samePullSnapshotIdentity(local, result.Baseline) {
		identity.Kind = pullLifecyclePreviewStale
		return identity
	}
	if len(result.Commits) == 0 {
		identity.Kind = pullLifecycleNoOpCompleted
		return identity
	}
	identity.Kind = pullLifecycleConfirmationReady
	return identity
}

func classifyLegacyPullFetchedOutcome(req *pullRequest, confirmStale bool, msg pullFetchedMsg) pullLifecycleOutcome {
	identity := classifyPullLifecycleIdentity(req, msg.requestID, msg.requestEpoch, confirmStale)
	if identity.Kind == pullLifecycleIdentityIgnored {
		return identity
	}
	if !samePullSnapshotIdentity(req.FetchBaseline, msg.fetchBaseline) {
		return pullLifecycleOutcome{Kind: pullLifecycleIdentityIgnored}
	}
	if msg.err != nil {
		identity.Kind = pullLifecyclePreviewUnavailable
		return identity
	}
	if msg.operationBaseline == (PullSnapshotIdentity{}) || !msg.operationBaselineSet {
		identity.Kind = pullLifecyclePreviewUnavailable
		return identity
	}
	if !samePullSnapshotIdentity(pullSnapshotIdentity(msg.status, msg.requestEpoch), msg.operationBaseline) {
		identity.Kind = pullLifecyclePreviewStale
		return identity
	}
	tracking, trackingKnown := msg.status.Tracking[msg.status.Branch]
	if !msg.status.TrackingKnown || !msg.status.TrackingFresh || !trackingKnown {
		identity.Kind = pullLifecyclePreviewStale
		return identity
	}
	if tracking.Behind == 0 {
		identity.Kind = pullLifecycleNoOpCompleted
		return identity
	}
	identity.Kind = pullLifecycleFetchReady
	return identity
}

func classifyLegacyPullPreviewReadyOutcome(req *pullRequest, confirmStale bool, msg pullPreviewReadyMsg) pullLifecycleOutcome {
	identity := classifyPullLifecycleIdentity(req, msg.requestID, msg.requestEpoch, confirmStale)
	if identity.Kind == pullLifecycleIdentityIgnored || req == nil || !samePullSnapshotIdentity(req.OperationBaseline, msg.baseline) {
		return pullLifecycleOutcome{Kind: pullLifecycleIdentityIgnored}
	}
	if msg.err != nil {
		identity.Kind = pullLifecyclePreviewUnavailable
		return identity
	}
	if len(msg.commits) == 0 {
		identity.Kind = pullLifecycleNoOpCompleted
		return identity
	}
	identity.Kind = pullLifecycleConfirmationReady
	return identity
}

// The guarded result classifiers deliberately return only neutral lifecycle
// outcomes. Concrete status text, state mutation, and command ownership remain
// in the existing handlers below the Update boundary.
func classifyPullValidationOutcome(req *pullRequest, confirmStale bool, msg pullValidationMsg) pullLifecycleOutcome {
	identity := classifyPullLifecycleIdentity(req, msg.requestID, msg.requestEpoch, confirmStale)
	if identity.Kind == pullLifecycleIdentityIgnored {
		return identity
	}
	if req == nil || !samePullSnapshotIdentity(req.OperationBaseline, msg.baseline) {
		identity.Kind = pullLifecyclePreviewStale
		return identity
	}
	if msg.err != nil || !msg.valid {
		identity.Kind = pullLifecycleValidationRejected
		return identity
	}
	identity.Kind = pullLifecycleConfirmationReady
	return identity
}

func classifyPullExecutionResultOutcome(req *pullRequest, msg pullExecutionResultMsg) pullLifecycleOutcome {
	identity := classifyPullLifecycleIdentity(req, msg.requestID, msg.requestEpoch, false)
	if identity.Kind == pullLifecycleIdentityIgnored {
		return identity
	}
	if req == nil || !samePullSnapshotIdentity(req.OperationBaseline, msg.baseline) || msg.stale {
		identity.Kind = pullLifecyclePreviewStale
		return identity
	}
	if msg.err != nil || msg.executionErr != nil || msg.refreshErr != nil {
		identity.Kind = pullLifecycleExecutionFailed
		return identity
	}
	identity.Kind = pullLifecycleCompleted
	return identity
}

func classifyPullWorkflowOutcome(req *pullRequest, msg pullWorkflowMsg) pullLifecycleOutcome {
	if req == nil || msg.result.OperationRequestID != req.ID || msg.result.OperationEpoch != req.Epoch {
		return pullLifecycleOutcome{Kind: pullLifecycleIdentityIgnored}
	}
	identity := pullLifecycleOutcome{Kind: pullLifecycleActive, RequestID: req.ID, RepositoryEpoch: req.Epoch,
		FetchBaseline: req.FetchBaseline, OperationBaseline: req.OperationBaseline, OperationBaselineSet: req.OperationBaselineSet}
	if msg.result.RefreshErrorKind != "" && msg.result.RefreshErrorKind != ReadErrorNone {
		identity.Kind = pullLifecycleRefreshFailed
		return identity
	}
	if msg.result.Execute.Reason != "" && msg.result.Execute.Reason != PullRejectNone {
		identity.Kind = pullLifecycleValidationRejected
		return identity
	}
	if msg.result.Execute.Mode == PullModeNoOp && msg.result.Execute.Succeeded {
		identity.Kind = pullLifecycleNoOpCompleted
		return identity
	}
	if !msg.result.Execute.Succeeded {
		identity.Kind = pullLifecycleExecutionFailed
		return identity
	}
	if msg.result.Refresh.RequestID != msg.result.RefreshRequestID || msg.result.Refresh.RepositoryEpoch != msg.result.RefreshEpoch {
		identity.Kind = pullLifecycleRefreshIdentityIgnored
		return identity
	}
	identity.Kind = pullLifecycleCompleted
	return identity
}

func classifyPullLifecycleIdentity(req *pullRequest, requestID, repositoryEpoch uint64, confirmStale bool) pullLifecycleOutcome {
	if req == nil || confirmStale || req.ID != requestID || req.Epoch != repositoryEpoch {
		return pullLifecycleOutcome{Kind: pullLifecycleIdentityIgnored}
	}
	return pullLifecycleOutcome{
		Kind:                 pullLifecycleActive,
		RequestID:            req.ID,
		RepositoryEpoch:      req.Epoch,
		FetchBaseline:        req.FetchBaseline,
		OperationBaseline:    req.OperationBaseline,
		OperationBaselineSet: req.OperationBaselineSet,
	}
}
