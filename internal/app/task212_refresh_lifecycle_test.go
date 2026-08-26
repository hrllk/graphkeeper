package app

import (
	"errors"
	"reflect"
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

func task212PullRequest() *pullRequest {
	return &pullRequest{ID: 7, Epoch: 3}
}

func task212Graph(head string) graph.Snapshot {
	return graph.Snapshot{Branch: "main", Head: head, Commits: []graph.Commit{{Hash: head, Subject: "commit " + head}}}
}

func task212WorkflowMessage(refreshRequestID, refreshEpoch uint64, refreshError ReadErrorKind, mode PullMode) pullWorkflowMsg {
	return pullWorkflowMsg{result: PullWorkflowResult{
		OperationRequestID: 7,
		OperationEpoch:     3,
		RefreshRequestID:   8,
		RefreshEpoch:       4,
		Refresh: ReadSnapshotResult{
			RequestID:       refreshRequestID,
			RepositoryEpoch: refreshEpoch,
			ErrorKind:       ReadErrorNone,
			Snapshot:        ReadSnapshot{Graph: task212Graph("refreshed")},
		},
		RefreshErrorKind: refreshError,
		Execute:          PullExecutionResult{RequestID: 7, RepositoryEpoch: 3, Mode: mode, Succeeded: true},
	}}
}

func TestTopLevelPullWorkflowRefreshOwnership(t *testing.T) {
	prior := task212Graph("prior")
	cases := []struct {
		name        string
		msg         pullWorkflowMsg
		wantGraph   graph.Snapshot
		wantTitle   string
		wantMessage string
		wantDetail  string
		wantActive  bool
	}{
		{
			name:        "matching post-success refresh projects graph",
			msg:         task212WorkflowMessage(8, 4, ReadErrorNone, PullModeMerge),
			wantGraph:   task212Graph("refreshed"),
			wantTitle:   "PULL COMPLETED",
			wantMessage: "PULL COMPLETED",
			wantDetail:  "Press esc to return to the graph.",
		},
		{
			name:        "refresh request mismatch ignores stale projection",
			msg:         task212WorkflowMessage(99, 4, ReadErrorNone, PullModeMerge),
			wantGraph:   prior,
			wantTitle:   "PULL COMPLETED",
			wantMessage: "PULL COMPLETED",
			wantDetail:  "Press esc to return to the graph.",
		},
		{
			name:        "refresh epoch mismatch ignores stale projection",
			msg:         task212WorkflowMessage(8, 99, ReadErrorNone, PullModeMerge),
			wantGraph:   prior,
			wantTitle:   "PULL COMPLETED",
			wantMessage: "PULL COMPLETED",
			wantDetail:  "Press esc to return to the graph.",
		},
		{
			name:        "refresh failure leaves graph unverified",
			msg:         task212WorkflowMessage(8, 4, ReadErrorRepository, PullModeMerge),
			wantGraph:   prior,
			wantTitle:   "PULL COMPLETED — STATE UNVERIFIED",
			wantMessage: "PULL COMPLETED — STATE UNVERIFIED",
			wantDetail:  "Refresh failed. Press f to refresh repository state.",
		},
		{
			name:        "successful operation without refresh leaves graph unchanged",
			msg:         task212WorkflowMessage(0, 0, ReadErrorNone, PullModeMerge),
			wantGraph:   prior,
			wantTitle:   "PULL COMPLETED",
			wantMessage: "PULL COMPLETED",
			wantDetail:  "Press esc to return to the graph.",
		},
		{
			name:        "successful no-op preserves graph and uses no-op copy",
			msg:         task212WorkflowMessage(0, 0, ReadErrorNone, PullModeNoOp),
			wantGraph:   prior,
			wantTitle:   "PULL COMPLETED",
			wantMessage: "PULL COMPLETED",
			wantDetail:  "No action needed. Press esc to return to the graph.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := model{
				repositoryState: repositoryState{graphReadSnapshot: prior, repoSnapshotLoaded: true},
				pullState:       pullState{activePullRequest: task212PullRequest()}}
			gotModel, cmd := m.Update(tc.msg)
			got := gotModel.(model)
			if cmd != nil {
				t.Fatalf("workflow emitted unexpected command: %v", cmd)
			}
			if !reflect.DeepEqual(got.graphReadSnapshot, tc.wantGraph) {
				t.Fatalf("graph projection = %#v, want %#v", got.graphReadSnapshot, tc.wantGraph)
			}
			if !got.repoSnapshotLoaded || got.activePullRequest != nil {
				t.Fatalf("workflow lifecycle state = loaded=%v active=%#v", got.repoSnapshotLoaded, got.activePullRequest)
			}
			if got.status.Title != tc.wantTitle || got.status.Message != tc.wantMessage || got.status.Detail != tc.wantDetail {
				t.Fatalf("status = title %q message %q detail %q", got.status.Title, got.status.Message, got.status.Detail)
			}
		})
	}
}

func TestTopLevelPullWorkflowFailedNoOpUsesFailureRouting(t *testing.T) {
	m := model{
		pullState: pullState{activePullRequest: task212PullRequest()}}
	msg := task212WorkflowMessage(0, 0, ReadErrorNone, PullModeNoOp)
	msg.result.Execute.Succeeded = false

	gotModel, cmd := m.Update(msg)
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("failed no-op emitted unexpected command: %v", cmd)
	}
	if got.status.Mode == state.ModeOperationResult || got.status.Title == "PULL COMPLETED" || got.status.Message == "PULL COMPLETED" {
		t.Fatalf("failed no-op was projected as completion: status=%+v", got.status)
	}
	if got.status.Block != state.BlockUnknown || got.status.Message != "PULL FAILED" || got.status.Detail != "Pull execution failed." {
		t.Fatalf("failed no-op status = %+v, want blocked PULL FAILED copy", got.status)
	}
	if got.activePullRequest != nil {
		t.Fatalf("failed no-op left active request: %#v", got.activePullRequest)
	}
}

func TestTopLevelLifecycleRefreshOwnershipAndRepositoryReadBoundary(t *testing.T) {
	prior := task212Graph("prior")
	updatedStatus := git.Status{Root: "/repo", Branch: "main", Head: "new-head"}

	t.Run("matching legacy refresh remains lifecycle-owned", func(t *testing.T) {
		m := model{
			repositoryState: repositoryState{repoStatus: git.Status{Head: "old-head"}},
			pullState:       pullState{}, status: state.New().WithBrowse()}
		gotModel, cmd := m.Update(refreshedMsg{status: updatedStatus})
		got := gotModel.(model)
		if cmd == nil {
			// The legacy refresh handler owns loading stash state; a non-nil command
			// is part of the existing behavior, not a pull-workflow command.
			t.Fatal("successful lifecycle refresh did not retain its existing command")
		}
		if got.repoStatus.Head != "new-head" || got.status.Mode != state.ModeBrowse {
			t.Fatalf("legacy refresh was not applied: status=%#v repo=%#v", got.status, got.repoStatus)
		}
	})

	t.Run("stale refresh reports telemetry and invalidates loading pull", func(t *testing.T) {
		sink := &recordingEventSink{}
		cancelled := 0
		m := model{
			repositoryState: repositoryState{
				repositoryEpoch: 3,
			},
			pullState: pullState{
				activePullRequest: task212PullRequest(),
				pullCancel:        func() { cancelled++ },
				nextPullRequestID: 7,
			},
			eventSink: sink,
			status:    state.New().WithLoading("Analyzing pull...")}
		m.status.Action = state.ActionPull
		gotModel, cmd := m.Update(refreshedMsg{status: updatedStatus, epoch: 2, epochSet: true})
		got := gotModel.(model)
		if cmd == nil || cancelled != 1 || got.activePullRequest != nil || got.status.Block != state.BlockStaleSnapshot {
			t.Fatalf("stale loading refresh transition = model=%#v cmd=%v cancelled=%d", got, cmd, cancelled)
		}
		if len(sink.events) != 1 || sink.events[0].Name != "discard_stale_refresh" {
			t.Fatalf("stale refresh telemetry = %#v", sink.events)
		}
	})

	t.Run("stale review and confirm refresh mark pull stale without command", func(t *testing.T) {
		for _, mode := range []state.Mode{state.ModeReview, state.ModeConfirm} {
			m := model{
				repositoryState: repositoryState{repositoryEpoch: 3},
				pullState:       pullState{activePullRequest: task212PullRequest()}, status: state.Status{Mode: mode, Action: state.ActionPull}}
			gotModel, cmd := m.Update(refreshedMsg{epoch: 2, epochSet: true})
			got := gotModel.(model)
			if cmd != nil || !got.pullConfirmStale || got.activePullRequest == nil {
				t.Fatalf("mode %v stale transition = model=%#v cmd=%v", mode, got, cmd)
			}
		}
	})

	t.Run("matching snapshot refresh projects graph", func(t *testing.T) {
		m := model{
			repositoryState: repositoryState{repositoryEpoch: 3, refreshGeneration: 5, graphReadSnapshot: prior},
			pullState:       pullState{}}
		gotModel, cmd := m.Update(refreshedSnapshotMsg{refreshGeneration: 5, result: ReadSnapshotResult{RepositoryEpoch: 3, ErrorKind: ReadErrorNone, Snapshot: ReadSnapshot{Graph: task212Graph("new")}}})
		got := gotModel.(model)
		if cmd != nil || got.graphReadSnapshot.Head != "new" {
			t.Fatalf("snapshot refresh = graph=%#v cmd=%v", got.graphReadSnapshot, cmd)
		}
	})

	t.Run("stale snapshot refresh uses existing pull lifecycle ownership", func(t *testing.T) {
		for _, mode := range []state.Mode{state.ModeLoading, state.ModeReview, state.ModeConfirm} {
			t.Run(map[state.Mode]string{state.ModeLoading: "loading", state.ModeReview: "review", state.ModeConfirm: "confirm"}[mode], func(t *testing.T) {
				cancelled := 0
				m := model{
					repositoryState: repositoryState{
						repositoryEpoch:   3,
						refreshGeneration: 5,
						graphReadSnapshot: prior,
					},
					pullState: pullState{
						activePullRequest: task212PullRequest(),
						pullCancel:        func() { cancelled++ },
					},
					status: state.Status{Mode: mode, Action: state.ActionPull}}
				gotModel, cmd := m.Update(refreshedSnapshotMsg{refreshGeneration: 4, result: ReadSnapshotResult{RepositoryEpoch: 3, Snapshot: ReadSnapshot{Graph: task212Graph("stale")}}})
				got := gotModel.(model)
				if !reflect.DeepEqual(got.graphReadSnapshot, prior) {
					t.Fatalf("stale snapshot projected: graph=%#v", got.graphReadSnapshot)
				}
				if mode == state.ModeLoading {
					if cmd == nil || cancelled != 1 || got.status.Block != state.BlockStaleSnapshot || got.activePullRequest != nil {
						t.Fatalf("loading stale snapshot transition = model=%#v cmd=%v cancelled=%d", got, cmd, cancelled)
					}
				} else if cmd != nil || !got.pullConfirmStale || got.activePullRequest == nil {
					t.Fatalf("review/confirm stale snapshot transition = model=%#v cmd=%v", got, cmd)
				}
			})
		}
	})

	t.Run("repository read does not own matching snapshot refresh messages", func(t *testing.T) {
		m := model{
			repositoryState: repositoryState{repositoryEpoch: 3, refreshGeneration: 5, graphReadSnapshot: prior, repoStatus: git.Status{Head: "prior"}},
			pullState:       pullState{}, repositoryRead: &fakeRepositoryRead{}}
		gotModel, cmd := m.Update(refreshedSnapshotMsg{refreshGeneration: 5, result: ReadSnapshotResult{RepositoryEpoch: 3, ErrorKind: ReadErrorNone, Snapshot: ReadSnapshot{Graph: task212Graph("stale")}}})
		got := gotModel.(model)
		if cmd != nil || !reflect.DeepEqual(got.graphReadSnapshot, task212Graph("stale")) || len(got.repoStatus.GraphCommits) != 1 || got.repoStatus.GraphCommits[0].Hash != "stale" {
			t.Fatalf("matching snapshot refresh did not project through injected reader: graph=%#v repo=%#v cmd=%v", got.graphReadSnapshot, got.repoStatus, cmd)
		}
		gotModel, cmd = m.Update(refreshedMsg{status: git.Status{Head: "stale"}, epoch: 2, epochSet: true, err: errors.New("stale")})
		got = gotModel.(model)
		if cmd != nil || got.repoStatus.Head != "prior" {
			t.Fatalf("legacy repository-read early return changed state: %#v cmd=%v", got, cmd)
		}
	})
}
