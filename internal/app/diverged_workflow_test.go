package app

import (
	"reflect"
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestContextProjectionRecommendationOnlyCurrentAndSuppressedByPull(t *testing.T) {
	m := model{status: state.New().WithBrowse(), activeSection: sectionCurrent, repositoryEpoch: 7,
		repoStatus: git.Status{Branch: "main", Upstream: "origin/main", Head: "abc", Remote: "origin", Tracking: map[string]git.BranchTracking{"main": {Ahead: 2, Behind: 1}}, TrackingKnown: true, TrackingFresh: true}}
	if m.contextProjection(80).Recommendation == nil {
		t.Fatal("current section should project diverged recommendation")
	}
	m.activeSection = sectionGraph
	if m.contextProjection(80).Recommendation != nil {
		t.Fatal("graph section must not project recommendation")
	}
	m.activeSection = sectionCurrent
	m.activePullRequest = &pullRequest{ID: 1}
	if m.contextProjection(80).Recommendation != nil {
		t.Fatal("active pull must suppress recommendation")
	}
}

func TestStalePullFetchedMessageDoesNotReplaceState(t *testing.T) {
	original := git.Status{Root: "/repo", Branch: "main", Head: "old"}
	m := model{status: state.New().WithBrowse(), repoStatus: original,
		activePullRequest: &pullRequest{ID: 9, Epoch: 3}}
	next, cmd := m.Update(pullFetchedMsg{status: git.Status{Root: "/repo", Branch: "other", Head: "new"}, requestID: 8, requestEpoch: 3})
	got := next.(model)
	if cmd != nil || !reflect.DeepEqual(got.repoStatus, original) || !reflect.DeepEqual(got.status, m.status) {
		t.Fatalf("stale pull message mutated model: %+v", got)
	}
}

func TestPullFetchedMessageRequiresFreshKnownTrackingEntry(t *testing.T) {
	baseline := PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "abc", Upstream: "origin/main", TrackingKnown: true, TrackingFresh: true}
	m := model{status: state.New().WithBrowse(), repoStatus: git.Status{Root: "/repo", Branch: "main", Head: "abc"}, sectionCursor: map[graphSection]int{},
		activePullRequest: &pullRequest{ID: 9, Epoch: 3, Baseline: baseline}}

	next, cmd := m.Update(pullFetchedMsg{status: git.Status{Root: "/repo", Branch: "main", Head: "abc", TrackingKnown: true, TrackingFresh: true}, requestID: 9, requestEpoch: 3, baseline: baseline})
	got := next.(model)
	if got.activePullRequest != nil {
		t.Fatal("expected invalid tracking to clear the active pull request")
	}
	if got.status.Block != state.BlockStaleSnapshot || got.status.Message != "Repository changed." {
		t.Fatalf("expected stale snapshot guidance, got %+v", got.status)
	}
	if cmd == nil {
		t.Fatal("expected invalid tracking to schedule a refresh")
	}
}

func TestPullFetchedMessageRejectsBaselineMismatchBeforeInstallingState(t *testing.T) {
	original := git.Status{Root: "/repo", Branch: "main", Head: "old"}
	status := git.Status{Root: "/repo", Branch: "main", Head: "new", TrackingKnown: true, TrackingFresh: true}
	baseline := pullSnapshotIdentity(status, 3)
	m := model{status: state.New().WithBrowse(), repoStatus: original,
		activePullRequest: &pullRequest{ID: 9, Epoch: 3, Baseline: baseline}}

	next, cmd := m.Update(pullFetchedMsg{status: status, requestID: 9, requestEpoch: 3,
		baseline: PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "different"}})
	got := next.(model)
	if got.activePullRequest != nil {
		t.Fatal("expected baseline mismatch to clear the active pull request")
	}
	if !reflect.DeepEqual(got.repoStatus, original) {
		t.Fatalf("baseline mismatch installed repository state: %+v", got.repoStatus)
	}
	if got.status.Block != state.BlockStaleSnapshot || got.status.Message != "Repository changed." {
		t.Fatalf("expected stale snapshot guidance, got %+v", got.status)
	}
	if cmd == nil {
		t.Fatal("expected baseline mismatch to schedule a refresh")
	}
}

func TestPullValidationMessageRejectsBaselineMismatch(t *testing.T) {
	baseline := PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "old"}
	m := model{status: state.New().WithBrowse(), repoStatus: git.Status{Root: "/repo"},
		activePullRequest: &pullRequest{ID: 9, Epoch: 3, Baseline: baseline}}

	next, cmd := m.Update(pullValidationMsg{requestID: 9, requestEpoch: 3,
		baseline: PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "new"}, valid: true})
	got := next.(model)
	if got.activePullRequest != nil {
		t.Fatal("expected baseline mismatch to clear the active pull request")
	}
	if got.status.Block != state.BlockStaleSnapshot || got.status.Message != "Repository changed." {
		t.Fatalf("expected stale snapshot guidance, got %+v", got.status)
	}
	if cmd == nil {
		t.Fatal("expected baseline mismatch to schedule a refresh")
	}
}

func TestStalePullExecutionResultBlocksAndRefreshes(t *testing.T) {
	m := model{status: state.New().WithConfirm(state.ActionPull, "Pull", "Pull"), repoStatus: git.Status{Root: "/repo"},
		activePullRequest: &pullRequest{ID: 9, Epoch: 3}}

	next, cmd := m.Update(pullExecutionResultMsg{action: state.ActionPull, requestID: 9, requestEpoch: 3, stale: true})
	got := next.(model)
	if got.activePullRequest != nil {
		t.Fatal("expected stale execution to clear the active pull request")
	}
	if got.status.Block != state.BlockStaleSnapshot || got.status.Message != "Repository changed." {
		t.Fatalf("expected stale snapshot guidance, got %+v", got.status)
	}
	if cmd == nil {
		t.Fatal("expected stale execution to schedule a refresh")
	}
}

func TestPullExecutionResultRejectsBaselineMismatch(t *testing.T) {
	baseline := PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "old"}
	m := model{status: state.New().WithConfirm(state.ActionPull, "Pull", "Pull"), repoStatus: git.Status{Root: "/repo"},
		activePullRequest: &pullRequest{ID: 9, Epoch: 3, Baseline: baseline}}

	next, cmd := m.Update(pullExecutionResultMsg{action: state.ActionPull, requestID: 9, requestEpoch: 3,
		baseline: PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "new"}})
	got := next.(model)
	if got.activePullRequest != nil {
		t.Fatal("expected baseline mismatch to clear the active pull request")
	}
	if got.status.Block != state.BlockStaleSnapshot || got.status.Message != "Repository changed." {
		t.Fatalf("expected stale snapshot guidance, got %+v", got.status)
	}
	if cmd == nil {
		t.Fatal("expected baseline mismatch to schedule a refresh")
	}
}
