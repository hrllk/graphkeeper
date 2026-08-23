package gitread

import (
	"testing"
	"time"

	"hrllk/graphkeeper/internal/app"
	"hrllk/graphkeeper/internal/git"
)

func TestMapStatusToSnapshotCopiesGraphFieldsAndFreshness(t *testing.T) {
	status := git.Status{
		Root: "/repo", Branch: "main", Head: "head", DefaultBranch: "trunk", Upstream: "origin/main", UpstreamOID: "up", Remote: "origin",
		Branches: []string{"main", "dev"}, LocalBranches: []string{"main"}, LocalBranchesKnown: true, LocalBranchesFresh: true, LocalBranchesError: "known",
		RemoteBranches: []string{"origin/main"}, BranchUpstreams: map[string]string{"main": "origin/main"}, Tracking: map[string]git.BranchTracking{"main": {Ahead: 2, Behind: 1}}, TrackingKnown: true, TrackingFresh: true,
		GraphCommits:    []git.GraphCommit{{Graph: "* ", Hash: "head", Parents: []string{"base"}, Decorations: []string{"HEAD -> main"}, Subject: "message"}},
		MergeInProgress: true, ConflictTarget: "target", ConflictTargetSubject: "subject", WorktreeDirty: true, Detached: true, EmptyRepo: true, NoRemote: true, NoUpstream: true, UpstreamGone: true,
		LastFetchAt: time.Unix(123, 0), RemoteSyncSummary: "sync",
	}
	got := MapStatusToSnapshot(status, 7)
	if got.Root != "/repo" || got.Graph.Branch != "main" || len(got.Graph.Commits) != 1 || got.Graph.Conflict.Target != "target" {
		t.Fatalf("unexpected snapshot: %#v", got)
	}
	if got.Graph.Commits[0].Parents[0] != "base" || got.Freshness.Ahead != 2 || !got.Freshness.TrackingKnown || got.Freshness.Epoch != 7 {
		t.Fatalf("mapping lost fields: %#v", got)
	}
	p := got.Repository
	if p.DefaultBranch != "trunk" || p.UpstreamOID != "up" || p.Remote != "origin" || p.LocalBranchesError != "known" || p.BranchUpstreams["main"] != "origin/main" || p.Tracking["main"].Behind != 1 || !p.Detached || !p.EmptyRepo || !p.NoRemote || !p.UpstreamGone || p.ConflictTargetSubject != "subject" || !p.LastFetchAt.Equal(time.Unix(123, 0)) || p.RemoteSyncSummary != "sync" {
		t.Fatalf("repository projection lost fields: %#v", p)
	}
	status.GraphCommits[0].Parents[0] = "changed"
	if got.Graph.Commits[0].Parents[0] != "base" {
		t.Fatal("snapshot must own copied slices")
	}
}

func TestUnconfiguredAdapterClassifiesRepositoryError(t *testing.T) {
	result, err := (&Adapter{}).ReadSnapshot(t.Context(), app.ReadRequest{})
	if err == nil || result.ErrorKind != app.ReadErrorRepository || result.Canceled {
		t.Fatalf("unexpected result: %#v err=%v", result, err)
	}
}

func TestAdapterRejectsDirectoryWithoutRepository(t *testing.T) {
	repo, err := git.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	result, readErr := New(repo).ReadSnapshot(t.Context(), app.ReadRequest{RepositoryEpoch: 3})
	if readErr == nil || result.ErrorKind != app.ReadErrorRepository || result.RepositoryEpoch != 3 {
		t.Fatalf("unexpected invalid repository result: %#v err=%v", result, readErr)
	}
}
