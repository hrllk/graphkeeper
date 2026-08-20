package app

import "testing"

func TestClassifyOperationResultNoOpIsVerifiedAndActionable(t *testing.T) {
	identity := PullSnapshotIdentity{Branch: "main", Head: "111", Upstream: "origin/main", TrackingKnown: true, TrackingFresh: true}
	result := classifyOperationResult(OperationResultInput{
		Operation: PullResultMetadata{Mode: PullModeMerge, Branch: "main", Upstream: "origin/main"},
		Before:    SnapshotObservation{Valid: true, Identity: identity},
		After:     SnapshotObservation{Valid: true, Identity: identity},
	})
	if result.Repository != RepositoryAligned || result.Verification != VerificationVerified {
		t.Fatalf("no-op result = %+v", result)
	}
	if !result.NoOp {
		t.Fatal("no-op result was not marked as no-op")
	}
	if result.Headline != "PULL COMPLETED" {
		t.Fatalf("headline = %q", result.Headline)
	}
}
