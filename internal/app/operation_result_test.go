package app

import (
	"errors"
	"testing"
)

func TestClassifyOperationResultSeparatesExecutionAndRefreshFailures(t *testing.T) {
	before := SnapshotObservation{Valid: true, Identity: PullSnapshotIdentity{Branch: "main", Head: "111"}}
	result := classifyOperationResult(OperationResultInput{
		Operation:      PullResultMetadata{Mode: PullModeMerge, Branch: "main", Upstream: "origin/main"},
		Before:         before,
		After:          SnapshotObservation{Valid: false},
		ExecutionError: errors.New("pull failed"),
		RefreshError:   errors.New("status failed"),
	})

	if result.Execution != ExecutionFailed {
		t.Fatalf("execution = %q, want failed", result.Execution)
	}
	if result.Verification != VerificationUnknown {
		t.Fatalf("verification = %q, want unknown", result.Verification)
	}
	if result.ExecutionError == nil || result.RefreshError == nil {
		t.Fatalf("errors were not preserved: %+v", result)
	}
	if result.Headline != "PULL FAILED" {
		t.Fatalf("headline = %q, want Git failure to remain primary", result.Headline)
	}
}

func TestClassifyOperationResultConflictOutranksAlignedSuccess(t *testing.T) {
	result := classifyOperationResult(OperationResultInput{
		Operation: PullResultMetadata{Mode: PullModeRebase, Branch: "main", Upstream: "origin/main"},
		Before:    SnapshotObservation{Valid: true, Identity: PullSnapshotIdentity{Branch: "main", Head: "111"}},
		After: SnapshotObservation{Valid: true, Identity: PullSnapshotIdentity{
			Branch: "main", Head: "222", TrackingKnown: true, TrackingFresh: true,
			MergeInProgress: true,
		}},
	})

	if result.Repository != RepositoryConflicted {
		t.Fatalf("repository = %q, want conflicted", result.Repository)
	}
	if result.Headline != "CONFLICTS REQUIRE ATTENTION" {
		t.Fatalf("headline = %q", result.Headline)
	}
}

func TestClassifyOperationResultDoesNotClaimLocalHistoryPreservedWithoutEvidence(t *testing.T) {
	result := classifyOperationResult(OperationResultInput{
		Operation: PullResultMetadata{Mode: PullModeRebase, Branch: "main", Upstream: "origin/main"},
		Before: SnapshotObservation{Valid: true, Identity: PullSnapshotIdentity{
			Branch: "main", Head: "111", Ahead: 2, Behind: 1, TrackingKnown: true, TrackingFresh: true,
		}},
		After: SnapshotObservation{Valid: true, Identity: PullSnapshotIdentity{
			Branch: "main", Head: "222", Ahead: 2, Behind: 0, TrackingKnown: true, TrackingFresh: true,
		}},
	})

	if result.Headline == "REMOTE CHANGES INCORPORATED" {
		t.Fatal("result overclaimed remote incorporation without reachability evidence")
	}
	if result.Headline != "PULL COMPLETED" {
		t.Fatalf("headline = %q, want observable-facts wording", result.Headline)
	}
}
