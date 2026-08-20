package app

import "hrllk/graphkeeper/internal/state"

type ExecutionOutcome string

type RepositoryOutcome string

type VerificationConfidence string

type RefreshPurpose string

const (
	RefreshPeriodic      RefreshPurpose = "periodic"
	RefreshPostOperation RefreshPurpose = "post-operation"
	RefreshRetry         RefreshPurpose = "retry"
)

const (
	ExecutionSucceeded ExecutionOutcome = "succeeded"
	ExecutionFailed    ExecutionOutcome = "failed"
	ExecutionUnknown   ExecutionOutcome = "unknown"

	RepositoryAligned    RepositoryOutcome = "aligned"
	RepositoryDiverged   RepositoryOutcome = "diverged"
	RepositoryConflicted RepositoryOutcome = "conflicted"
	RepositoryUnknown    RepositoryOutcome = "unknown"

	VerificationVerified VerificationConfidence = "verified"
	VerificationUnknown  VerificationConfidence = "unknown"
	VerificationStale    VerificationConfidence = "stale"
)

type PullResultMetadata struct {
	Action   state.Action
	Mode     PullMode
	Branch   string
	Upstream string
}

type SnapshotObservation struct {
	Identity PullSnapshotIdentity
	Valid    bool
	Reason   string
}

type OperationResultInput struct {
	Operation      PullResultMetadata
	Before         SnapshotObservation
	After          SnapshotObservation
	ExecutionError error
	RefreshError   error
}

type OperationResultSummary struct {
	Operation        PullResultMetadata
	Execution        ExecutionOutcome
	Repository       RepositoryOutcome
	Verification     VerificationConfidence
	Before           SnapshotObservation
	After            SnapshotObservation
	ExecutionError   error
	RefreshError     error
	Headline         string
	RefreshRetryable bool
	NoOp             bool
}

func classifyOperationResult(input OperationResultInput) OperationResultSummary {
	result := OperationResultSummary{
		Operation:      input.Operation,
		Before:         input.Before,
		After:          input.After,
		ExecutionError: input.ExecutionError,
		RefreshError:   input.RefreshError,
		Execution:      ExecutionSucceeded,
		Repository:     RepositoryUnknown,
		Verification:   VerificationUnknown,
	}
	if input.ExecutionError != nil {
		result.Execution = ExecutionFailed
	}
	if !input.After.Valid {
		result.Verification = VerificationUnknown
		if input.ExecutionError != nil {
			result.Headline = "PULL FAILED"
		} else {
			result.Headline = "PULL COMPLETED — STATE UNVERIFIED"
		}
		result.RefreshRetryable = true
		return result
	}

	result.Verification = VerificationVerified
	s := input.After.Identity
	result.NoOp = input.Before.Valid && input.Before.Identity == s
	switch {
	case s.MergeInProgress || s.RebaseInProgress || s.CherryPickInProgress:
		result.Repository = RepositoryConflicted
		result.Headline = "CONFLICTS REQUIRE ATTENTION"
	case !s.TrackingKnown || !s.TrackingFresh || s.UpstreamGone || s.NoUpstream || s.Detached || s.EmptyRepo:
		result.Repository = RepositoryUnknown
		result.Headline = "PULL COMPLETED"
	case s.Ahead == 0 && s.Behind == 0:
		result.Repository = RepositoryAligned
		result.Headline = "PULL COMPLETED"
	default:
		result.Repository = RepositoryDiverged
		result.Headline = "PULL COMPLETED"
	}
	return result
}
