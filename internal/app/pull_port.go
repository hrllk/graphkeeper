package app

import "context"

// PullPort is the application boundary for pull preview, validation, and mutation.
type PullPort interface {
	Preview(context.Context, PullPreviewRequest) (PullPreviewResult, error)
	Validate(context.Context, PullValidationRequest) (PullValidationResult, error)
	Execute(context.Context, PullExecutionRequest) (PullExecutionResult, error)
}

type PullRejectReason string

const (
	PullRejectNone            PullRejectReason = "none"
	PullRejectStaleEpoch      PullRejectReason = "stale_epoch"
	PullRejectChangedTarget   PullRejectReason = "changed_target"
	PullRejectChangedBaseline PullRejectReason = "changed_baseline"
	PullRejectInvalidTarget   PullRejectReason = "invalid_target"
	PullRejectBlockedState    PullRejectReason = "blocked_state"
	PullRejectNotEligible     PullRejectReason = "not_eligible"
)

type PullErrorKind string

const (
	PullErrorNone     PullErrorKind = "none"
	PullErrorGit      PullErrorKind = "git_failure"
	PullErrorCanceled PullErrorKind = "canceled"
	PullErrorRefresh  PullErrorKind = "refresh_failure"
)

type PullImpact struct {
	Summary          string
	Risk             string
	FastForwardKnown bool
	IsFastForward    bool
}

type PullPreviewRequest struct {
	RequestID       uint64
	RepositoryEpoch uint64
	Baseline        PullSnapshotIdentity
	Mode            PullMode
	CommitLimit     int
}

type PullPreviewCommit struct{ Hash string }

type PullPreviewResult struct {
	RequestID       uint64
	RepositoryEpoch uint64
	Baseline        PullSnapshotIdentity
	Mode            PullMode
	Eligible        bool
	Reason          PullRejectReason
	Impact          PullImpact
	Commits         []PullPreviewCommit
}

type PullValidationRequest struct {
	RequestID       uint64
	RepositoryEpoch uint64
	Current         PullSnapshotIdentity
	Expected        PullSnapshotIdentity
	Mode            PullMode
}

type PullValidationResult struct {
	RequestID          uint64
	RepositoryEpoch    uint64
	Valid              bool
	Authorized         bool
	Reason             PullRejectReason
	AuthorizedBaseline PullSnapshotIdentity
}

type PullExecutionRequest struct {
	RequestID          uint64
	RepositoryEpoch    uint64
	Authorized         bool
	AuthorizedBaseline PullSnapshotIdentity
	Mode               PullMode
}

type PullExecutionResult struct {
	RequestID       uint64
	RepositoryEpoch uint64
	Mode            PullMode
	Succeeded       bool
	ErrorKind       PullErrorKind
	Reason          PullRejectReason
}

type PullWorkflowResult struct {
	OperationRequestID uint64
	OperationEpoch     uint64
	RefreshRequestID   uint64
	RefreshEpoch       uint64
	Execute            PullExecutionResult
	Refresh            ReadSnapshotResult
	RefreshError       ReadError
	RefreshErrorKind   ReadErrorKind
}
