package app

import (
	"context"
	"reflect"
	"testing"

	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
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
	operationResult := OperationResultSummary{Operation: PullResultMetadata{Action: state.ActionPull}}
	m := model{
		repositoryState: repositoryState{
			graphReadSnapshot: graph.Snapshot{Branch: "main", Head: "head"},
		},
		pullState: pullState{
			activePullRequest: &pullRequest{ID: 7, Epoch: 3},
			operationResult:   &operationResult,
		}}
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
	if got.graphReadSnapshot.Branch != "main" || got.graphReadSnapshot.Head != "head" || got.operationResult != &operationResult {
		t.Fatalf("no-op cleared existing graph snapshot: %#v", got.graphReadSnapshot)
	}
	if got.status.Detail != "No action needed. Press q or Esc to return to the graph." {
		t.Fatalf("unexpected no-op detail: %q", got.status.Detail)
	}
}

func TestPullWorkflowRefreshIdentityMismatchDoesNotProjectGraph(t *testing.T) {
	prior := graph.Snapshot{Branch: "before", Head: "old"}
	m := model{
		repositoryState: repositoryState{graphReadSnapshot: prior, repoSnapshotLoaded: true},
		pullState:       pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3}}}
	next, cmd := m.Update(pullWorkflowMsg{result: PullWorkflowResult{
		OperationRequestID: 7, OperationEpoch: 3, RefreshRequestID: 8, RefreshEpoch: 4,
		Execute: PullExecutionResult{Mode: PullModeMerge, Succeeded: true},
		Refresh: ReadSnapshotResult{RequestID: 99, RepositoryEpoch: 4, Snapshot: ReadSnapshot{Graph: graph.Snapshot{Branch: "after", Head: "new"}}},
	}})
	if cmd != nil {
		t.Fatalf("identity mismatch scheduled command: %v", cmd)
	}
	got := next.(model)
	if !reflect.DeepEqual(got.graphReadSnapshot, prior) || !got.repoSnapshotLoaded {
		t.Fatalf("identity mismatch projected graph: %#v", got.graphReadSnapshot)
	}
}
