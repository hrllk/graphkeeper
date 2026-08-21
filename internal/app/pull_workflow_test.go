package app

import (
	"context"
	"testing"

	"hrllk/graphkeeper/internal/graph"
)

type fakePull struct {
	validates, executes int
	validation          PullValidationResult
	execution           PullExecutionResult
}

func (f *fakePull) Preview(context.Context, PullPreviewRequest) (PullPreviewResult, error) {
	return PullPreviewResult{}, nil
}
func (f *fakePull) Validate(_ context.Context, req PullValidationRequest) (PullValidationResult, error) {
	f.validates++
	r := f.validation
	r.RequestID = req.RequestID
	r.RepositoryEpoch = req.RepositoryEpoch
	return r, nil
}
func (f *fakePull) Execute(_ context.Context, req PullExecutionRequest) (PullExecutionResult, error) {
	f.executes++
	r := f.execution
	r.RequestID = req.RequestID
	r.RepositoryEpoch = req.RepositoryEpoch
	return r, nil
}

type fakeRead struct {
	calls    int
	snapshot ReadSnapshot
}

func (f *fakeRead) ReadSnapshot(context.Context, ReadRequest) (ReadSnapshotResult, error) {
	f.calls++
	return ReadSnapshotResult{ErrorKind: ReadErrorNone, Snapshot: f.snapshot}, nil
}

func TestPullWorkflowValidatesThenExecutesThenRefreshesWithNewTuple(t *testing.T) {
	p := &fakePull{validation: PullValidationResult{Valid: true, Authorized: true, AuthorizedBaseline: PullSnapshotIdentity{Epoch: 4, Branch: "main", Head: "h", Upstream: "origin/main", Ahead: 1, Behind: 2, TrackingKnown: true, TrackingFresh: true}}, execution: PullExecutionResult{Succeeded: true}}
	r := &fakeRead{snapshot: ReadSnapshot{Freshness: PullSnapshotIdentity{Epoch: 4, Branch: "main", Head: "h", Upstream: "origin/main", Ahead: 1, Behind: 2, TrackingKnown: true, TrackingFresh: true}, Graph: graph.Snapshot{}}}
	got := runPullWorkflow(context.Background(), p, r, PullExecutionRequest{RequestID: 9, RepositoryEpoch: 4, Authorized: true, AuthorizedBaseline: p.validation.AuthorizedBaseline}, PullModeMerge, 20)
	if p.validates != 1 || p.executes != 1 || r.calls != 2 {
		t.Fatalf("calls validate=%d execute=%d read=%d", p.validates, p.executes, r.calls)
	}
	if got.RefreshRequestID != 10 || got.RefreshEpoch != 5 || !got.Execute.Succeeded {
		t.Fatalf("unexpected workflow: %#v", got)
	}
}

func TestPullWorkflowRejectsValidationWithoutExecute(t *testing.T) {
	p := &fakePull{validation: PullValidationResult{Reason: PullRejectChangedBaseline}}
	r := &fakeRead{snapshot: ReadSnapshot{Freshness: PullSnapshotIdentity{Epoch: 4, Branch: "main", Head: "new", Behind: 2, TrackingKnown: true, TrackingFresh: true}}}
	got := runPullWorkflow(context.Background(), p, r, PullExecutionRequest{RequestID: 2, RepositoryEpoch: 4, Authorized: true, AuthorizedBaseline: PullSnapshotIdentity{Epoch: 4, Branch: "main", Head: "old"}}, PullModeMerge, 20)
	if p.executes != 0 || got.Execute.Reason != PullRejectChangedBaseline {
		t.Fatalf("unexpected rejection: %#v", got)
	}
}

func TestPullWorkflowNoOpDoesNotValidateExecuteOrRefresh(t *testing.T) {
	p := &fakePull{}
	baseline := PullSnapshotIdentity{Epoch: 4, Branch: "main", Head: "h", Upstream: "origin/main", Ahead: 0, Behind: 0, TrackingKnown: true, TrackingFresh: true}
	r := &fakeRead{snapshot: ReadSnapshot{Freshness: baseline}}
	got := runPullWorkflow(context.Background(), p, r, PullExecutionRequest{RequestID: 2, RepositoryEpoch: 4, Authorized: true, AuthorizedBaseline: baseline}, PullModeMerge, 20)
	if p.validates != 0 || p.executes != 0 || r.calls != 1 || got.Execute.Mode != PullModeNoOp || !got.Execute.Succeeded {
		t.Fatalf("unexpected noop: %#v calls=%d/%d/%d", got, p.validates, p.executes, r.calls)
	}
}

func TestPullWorkflowNoOpDoesNotClearExistingGraphSnapshot(t *testing.T) {
	m := model{
		graphReadSnapshot: graph.Snapshot{Branch: "main", Head: "head"},
		activePullRequest: &pullRequest{ID: 7, Epoch: 3},
	}
	next, _ := m.Update(pullWorkflowMsg{result: PullWorkflowResult{
		OperationRequestID: 7,
		OperationEpoch:     3,
		RefreshRequestID:   8,
		Execute: PullExecutionResult{
			RequestID:       7,
			RepositoryEpoch: 3,
			Mode:            PullModeNoOp,
			Succeeded:       true,
		},
	}})
	got := next.(model)
	if got.graphReadSnapshot.Branch != "main" || got.graphReadSnapshot.Head != "head" {
		t.Fatalf("no-op cleared existing graph snapshot: %#v", got.graphReadSnapshot)
	}
}
