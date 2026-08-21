package app

import (
	"testing"

	"hrllk/graphkeeper/internal/state"
)

func TestOperationLoadingFactoryPreservesActionAndLocksInput(t *testing.T) {
	operations := []progressOperation{
		progressFetch, progressPull, progressPush, progressStash, progressClean,
		progressCheckout, progressReset, progressMerge, progressRebase, progressBranch,
		progressTag, progressDeleteBranch, progressDeleteTag, progressDeleteRemoteTag,
		progressStashPop, progressPushTag, progressFetchSources, progressFetchTags,
		progressPrepareReset, progressGraphTargetAnalysis, progressCherryPick, progressAbort,
		progressOutcomeExecution, progressTagCommit, progressTargetPreview,
	}
	for _, operation := range operations {
		spec, ok := operationLoadingSpec(operation, "working", state.ActionPull)
		if !ok || spec.Operation != operation || !spec.LockInput {
			t.Fatalf("operation %q was not accepted: %#v, %v", operation, spec, ok)
		}
		status := operationLoadingStatus(spec)
		if status.Mode != state.ModeLoading || status.Action != state.ActionPull || status.Message != "working" || status.Detail != "Please wait." {
			t.Fatalf("operation %q produced wrong status: %#v", operation, status)
		}
	}
}

func TestOperationLoadingFactoryRejectsUnknownOperation(t *testing.T) {
	if _, ok := operationLoadingSpec(progressOperation("unknown"), "working", state.ActionPull); ok {
		t.Fatal("unknown operation was accepted")
	}
}

func TestOperationInputLockedOnlyForUnownedLoading(t *testing.T) {
	m := model{status: loadingToast("working")}
	if !operationInputLocked(m) {
		t.Fatal("plain loading state should lock input")
	}
	m.branchOpen = true
	if operationInputLocked(m) {
		t.Fatal("branch prompt should own input")
	}
}

func TestLoadingToastUsesSharedDetail(t *testing.T) {
	for _, message := range []string{
		"Loading...", "Fetching for push...", "Preparing reset...", "Enter a branch name.",
		"Checking out...", "Aborting...", "Fetching upstream...", "Previewing...",
		"Creating branch...", "Running...", "Pulling...", "Merging pull...",
		"Pushing and tracking...", "Force pushing...", "Merging...", "Rebasing...",
		"Pushing...", "Analyzing pull...",
	} {
		got := loadingToast(message)
		if got.Mode != state.ModeLoading || got.Title != "Loading" || got.Message != message || got.Detail != "Please wait." {
			t.Fatalf("unexpected loading status for %q: %#v", message, got)
		}
	}
}
