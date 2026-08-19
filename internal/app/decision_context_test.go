package app

import (
	"reflect"
	"testing"

	"hrllk/graphkeeper/internal/git"
)

func TestExplainBranchDecisionClassifiesRelations(t *testing.T) {
	base := DivergedSnapshot{Branch: "main", Upstream: "origin/main", Head: "abc", TrackingKnown: true, TrackingFresh: true}
	tests := []struct {
		name     string
		local    int
		upstream int
		relation BranchRelation
	}{
		{name: "fast-forward", local: 0, upstream: 3, relation: RelationFastForward},
		{name: "already aligned", local: 0, upstream: 0, relation: RelationAlreadyAligned},
		{name: "target included", local: 2, upstream: 0, relation: RelationTargetIncluded},
		{name: "diverged", local: 2, upstream: 3, relation: RelationDiverged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := base
			snapshot.LocalOnly = tt.local
			snapshot.UpstreamOnly = tt.upstream
			got, ok := explainBranchDecision(snapshot)
			if !ok || got.Relation != tt.relation {
				t.Fatalf("decision = %+v, ok=%v, want relation %s", got, ok, tt.relation)
			}
			if got.CurrentRef != "main" || got.TargetRef != "origin/main" || got.CurrentOnly != tt.local || got.TargetOnly != tt.upstream {
				t.Fatalf("decision refs/counts = %+v", got)
			}
			if len(got.ReasonLines) == 0 {
				t.Fatal("expected a relation reason")
			}
			if tt.relation != RelationDiverged && len(got.ActionLines) == 0 {
				t.Fatal("expected an action or no-op explanation")
			}
		})
	}
}

func TestExplainBranchDecisionRejectsUnavailableOrInvalidSnapshots(t *testing.T) {
	base := DivergedSnapshot{Branch: "main", Upstream: "origin/main", Head: "abc", LocalOnly: 1, UpstreamOnly: 1, TrackingKnown: true, TrackingFresh: true}
	tests := map[string]func(*DivergedSnapshot){
		"missing tracking":      func(s *DivergedSnapshot) { s.TrackingKnown = false },
		"stale tracking":        func(s *DivergedSnapshot) { s.TrackingFresh = false },
		"gone target":           func(s *DivergedSnapshot) { s.NoUpstream = true },
		"empty target":          func(s *DivergedSnapshot) { s.Upstream = "" },
		"no remote":             func(s *DivergedSnapshot) { s.NoRemote = true },
		"detached":              func(s *DivergedSnapshot) { s.Detached = true },
		"dirty":                 func(s *DivergedSnapshot) { s.WorktreeDirty = true },
		"in progress":           func(s *DivergedSnapshot) { s.RebaseInProgress = true },
		"negative local count":  func(s *DivergedSnapshot) { s.LocalOnly = -1 },
		"negative target count": func(s *DivergedSnapshot) { s.UpstreamOnly = -1 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			snapshot := base
			mutate(&snapshot)
			if got, ok := explainBranchDecision(snapshot); ok || !reflect.DeepEqual(got, BranchDecisionContext{}) {
				t.Fatalf("expected no decision, got %+v, ok=%v", got, ok)
			}
		})
	}
}

func TestExplainBranchDecisionAcceptsValidZeroTrackingRecord(t *testing.T) {
	snapshot := DivergedSnapshot{Branch: "main", Upstream: "origin/main", Head: "abc", TrackingKnown: true, TrackingFresh: true}
	got, ok := explainBranchDecision(snapshot)
	if !ok || got.Relation != RelationAlreadyAligned {
		t.Fatalf("valid zero tracking record = %+v, ok=%v", got, ok)
	}
}

func TestExplainBranchDecisionIsPureAndDoesNotMutateSnapshot(t *testing.T) {
	snapshot := DivergedSnapshot{Branch: "main", Upstream: "origin/main", Head: "abc", LocalOnly: 2, UpstreamOnly: 3, TrackingKnown: true, TrackingFresh: true}
	before := snapshot
	first, firstOK := explainBranchDecision(snapshot)
	second, secondOK := explainBranchDecision(snapshot)
	if !reflect.DeepEqual(snapshot, before) {
		t.Fatalf("policy mutated snapshot: before=%+v after=%+v", before, snapshot)
	}
	if firstOK != secondOK || !reflect.DeepEqual(first, second) {
		t.Fatalf("policy is not deterministic: first=%+v/%v second=%+v/%v", first, firstOK, second, secondOK)
	}
}

func TestExplainBranchDecisionRejectsUpstreamGoneFromStatusConversion(t *testing.T) {
	status := git.Status{
		Branch: "main", Upstream: "origin/main", Head: "abc",
		TrackingKnown: true, TrackingFresh: true, UpstreamGone: true,
		Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 1}},
	}
	snapshot := divergedSnapshotFromStatus(status, 1)
	if !snapshot.UpstreamGone {
		t.Fatal("expected upstream-gone state to survive snapshot conversion")
	}
	if _, ok := explainBranchDecision(snapshot); ok {
		t.Fatal("upstream-gone snapshot must not produce decision context")
	}
}

func TestDecisionSummaryUsesTargetOnlyLabel(t *testing.T) {
	got := decisionSummaryLine(BranchDecisionContext{CurrentRef: "main", TargetRef: "origin/main", CurrentOnly: 2, TargetOnly: 3})
	want := "main → origin/main · local-only 2 · target-only 3"
	if got != want {
		t.Fatalf("summary = %q, want %q", got, want)
	}
}
