package app

import "testing"

func validImpactInput() operationImpactInput {
	return operationImpactInput{Mode: PullModeMerge, CurrentRef: "main", UpstreamRef: "origin/main", HeadOID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", UpstreamOID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Ahead: 2, Behind: 3, FastForwardKnown: true, IsFastForward: false, Validity: operationImpactValidity{TargetKnown: true, SnapshotFresh: true, FastForwardKnown: true, HeadOIDValid: true, UpstreamOIDValid: true}}
}

func TestOperationImpactDivergedMergeAndRebase(t *testing.T) {
	input := validImpactInput()
	input.Mode = PullModeMerge
	summary, risk, ok := operationImpact(input)
	if !ok || summary != "Histories will be combined. A merge commit may be created." || risk != "Conflicts may occur." {
		t.Fatalf("unexpected merge impact: %q / %q / %t", summary, risk, ok)
	}
	input.Mode = PullModeRebase
	summary, risk, ok = operationImpact(input)
	if !ok || summary != "Local commits will be replayed onto origin/main. Commit identities may change." || risk != "Conflicts may occur." {
		t.Fatalf("unexpected rebase impact: %q / %q / %t", summary, risk, ok)
	}
}

func TestOperationImpactFastForwardUsesAncestorResult(t *testing.T) {
	input := validImpactInput()
	input.Ahead, input.Behind, input.IsFastForward = 0, 3, true
	summary, risk, ok := operationImpact(input)
	if !ok || risk != "" || summary != "HEAD can move to origin/main. No merge commit is needed." {
		t.Fatalf("unexpected fast-forward impact: %q / %q / %t", summary, risk, ok)
	}
}

func TestOperationImpactRejectsUnknownAndInconsistentSnapshots(t *testing.T) {
	input := validImpactInput()
	input.FastForwardKnown = false
	input.Validity.FastForwardKnown = false
	if _, _, ok := operationImpact(input); ok {
		t.Fatal("unknown fast-forward state must be unavailable")
	}
	input = validImpactInput()
	input.Ahead, input.Behind, input.IsFastForward = 0, 2, false
	if _, _, ok := operationImpact(input); ok {
		t.Fatal("inconsistent fast-forward state must be unavailable")
	}
}

func TestPullImpactSetNoOpBypassesFastForwardKnowledge(t *testing.T) {
	snapshot := PullImpactSnapshot{Ahead: 0, Behind: 0, FastForwardKnown: false}
	if got := pullImpactSet(snapshot); got.Valid {
		t.Fatalf("no-op must not render impact: %+v", got)
	}
}
