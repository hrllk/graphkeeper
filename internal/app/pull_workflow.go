package app

import "context"

// runPullWorkflow is the application-owned confirmation lifecycle. It reads
// freshness immediately before validation, never executes an unauthorized
// request, and refreshes only after a successful mutation.
func runPullWorkflow(ctx context.Context, pull PullPort, read RepositoryReadPort, req PullExecutionRequest, mode PullMode, limit int) PullWorkflowResult {
	result := PullWorkflowResult{OperationRequestID: req.RequestID, OperationEpoch: req.RepositoryEpoch, RefreshRequestID: req.RequestID + 1, RefreshEpoch: req.RepositoryEpoch + 1, RefreshErrorKind: ReadErrorNone}
	if pull == nil || read == nil {
		result.Execute = PullExecutionResult{RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Mode: mode, Reason: PullRejectNotEligible}
		return result
	}
	if !req.Authorized {
		result.Execute = PullExecutionResult{RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Mode: mode, Reason: PullRejectNotEligible}
		return result
	}
	current, err := read.ReadSnapshot(ctx, ReadRequest{RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, CommitLimit: limit})
	if err != nil {
		result.RefreshErrorKind = ReadErrorRepository
		result.RefreshError = ReadError{Kind: ReadErrorRepository}
		return result
	}
	if current.ErrorKind != ReadErrorNone {
		result.RefreshErrorKind = current.ErrorKind
		result.RefreshError = ReadError{Kind: current.ErrorKind}
		return result
	}
	if mode == PullModeNoOp || (current.Snapshot.Freshness.TrackingKnown && current.Snapshot.Freshness.TrackingFresh && current.Snapshot.Freshness.Behind == 0 && samePullSnapshotIdentity(current.Snapshot.Freshness, req.AuthorizedBaseline)) {
		result.Execute = PullExecutionResult{RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Mode: PullModeNoOp, Succeeded: true}
		return result
	}
	validation, err := pull.Validate(ctx, PullValidationRequest{RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Current: current.Snapshot.Freshness, Expected: req.AuthorizedBaseline, Mode: mode})
	if err != nil {
		result.RefreshErrorKind = ReadErrorRepository
		result.RefreshError = ReadError{Kind: ReadErrorRepository}
		return result
	}
	if !validation.Valid || !validation.Authorized {
		result.Execute = PullExecutionResult{RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Mode: mode, Reason: validation.Reason}
		return result
	}
	result.Execute, err = pull.Execute(ctx, PullExecutionRequest{RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Authorized: true, AuthorizedBaseline: validation.AuthorizedBaseline, Mode: mode})
	if err != nil || !result.Execute.Succeeded {
		return result
	}
	refreshed, refreshErr := read.ReadSnapshot(ctx, ReadRequest{RequestID: result.RefreshRequestID, RepositoryEpoch: result.RefreshEpoch, CommitLimit: limit})
	result.Refresh = refreshed
	if refreshErr != nil {
		result.RefreshErrorKind = ReadErrorRepository
		result.RefreshError = ReadError{Kind: ReadErrorRepository}
		return result
	}
	if refreshed.ErrorKind != ReadErrorNone {
		result.RefreshErrorKind = refreshed.ErrorKind
		result.RefreshError = ReadError{Kind: refreshed.ErrorKind}
	}
	return result
}
