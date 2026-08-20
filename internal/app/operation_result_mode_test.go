package app

import "testing"

func TestClassifyOperationResultNoOpCarriesNoOpMode(t *testing.T) {
	identity := PullSnapshotIdentity{Branch: "main", Head: "111", Upstream: "origin/main", TrackingKnown: true, TrackingFresh: true}
	result := classifyOperationResult(OperationResultInput{
		Operation: PullResultMetadata{Mode: PullModeNoOp, Branch: "main", Upstream: "origin/main"},
		Before:    SnapshotObservation{Valid: true, Identity: identity},
		After:     SnapshotObservation{Valid: true, Identity: identity},
	})
	if result.Operation.Mode != PullModeNoOp {
		t.Fatalf("mode = %q, want no-op", result.Operation.Mode)
	}
}
