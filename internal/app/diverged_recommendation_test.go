package app

import (
	"testing"

	"hrllk/graphkeeper/internal/git"
)

func TestRecommendDivergedPullRequiresFreshKnownTracking(t *testing.T) {
	tests := []struct {
		name     string
		snapshot DivergedSnapshot
		want     bool
	}{
		{"valid", DivergedSnapshot{Branch: "main", Upstream: "origin/main", Head: "abc", LocalOnly: 2, UpstreamOnly: 1, TrackingKnown: true, TrackingFresh: true}, true},
		{"equal", DivergedSnapshot{Branch: "main", Upstream: "origin/main", Head: "abc", TrackingKnown: true, TrackingFresh: true}, false},
		{"missing tracking", DivergedSnapshot{Branch: "main", Upstream: "origin/main", Head: "abc", LocalOnly: 2, UpstreamOnly: 1, TrackingFresh: true}, false},
		{"stale tracking", DivergedSnapshot{Branch: "main", Upstream: "origin/main", Head: "abc", LocalOnly: 2, UpstreamOnly: 1, TrackingKnown: true}, false},
		{"dirty", DivergedSnapshot{Branch: "main", Upstream: "origin/main", Head: "abc", LocalOnly: 2, UpstreamOnly: 1, TrackingKnown: true, TrackingFresh: true, WorktreeDirty: true}, false},
		{"in progress", DivergedSnapshot{Branch: "main", Upstream: "origin/main", Head: "abc", LocalOnly: 2, UpstreamOnly: 1, TrackingKnown: true, TrackingFresh: true, RebaseInProgress: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := recommendDivergedPull(tt.snapshot)
			if ok != tt.want {
				t.Fatalf("recommendation presence = %v, want %v (%+v)", ok, tt.want, got)
			}
			if ok && (len(got.PullModes) != 2 || got.PullModes[0] != PullModeMerge || got.PullModes[1] != PullModeRebase) {
				t.Fatalf("pull modes = %#v", got.PullModes)
			}
		})
	}
}

func TestPullSnapshotIdentityExactComparison(t *testing.T) {
	base := PullSnapshotIdentity{Epoch: 4, Branch: "main", Head: "abc", Upstream: "origin/main", Ahead: 2, Behind: 1, TrackingKnown: true, TrackingFresh: true}
	if !samePullSnapshotIdentity(base, base) {
		t.Fatal("identical pull snapshots should match")
	}
	for name, changed := range map[string]PullSnapshotIdentity{
		"epoch":             {Epoch: 5, Branch: "main", Head: "abc", Upstream: "origin/main", Ahead: 2, Behind: 1, TrackingKnown: true, TrackingFresh: true},
		"head":              {Epoch: 4, Branch: "main", Head: "def", Upstream: "origin/main", Ahead: 2, Behind: 1, TrackingKnown: true, TrackingFresh: true},
		"tracking validity": {Epoch: 4, Branch: "main", Head: "abc", Upstream: "origin/main", Ahead: 2, Behind: 1, TrackingFresh: true},
	} {
		if samePullSnapshotIdentity(base, changed) {
			t.Fatalf("changed %s identity should be stale", name)
		}
	}
}

func TestMissingCurrentBranchTrackingCannotValidateOrRecommendPull(t *testing.T) {
	status := git.Status{
		Branch:        "main",
		Upstream:      "origin/main",
		Head:          "abc",
		TrackingKnown: true,
		TrackingFresh: true,
	}

	identity := pullSnapshotIdentity(status, 4)
	if identity.TrackingKnown {
		t.Fatal("tracking should be unknown when the current branch has no tracking entry")
	}
	knownBaseline := identity
	knownBaseline.TrackingKnown = true
	if samePullSnapshotIdentity(identity, knownBaseline) {
		t.Fatal("snapshot with missing current branch tracking should not validate as known")
	}

	snapshot := divergedSnapshotFromStatus(status, 4)
	if _, ok := recommendDivergedPull(snapshot); ok {
		t.Fatal("snapshot with missing current branch tracking should not produce a pull recommendation")
	}
}
