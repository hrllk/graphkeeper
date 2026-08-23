package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/commitinspector"
	"hrllk/graphkeeper/internal/events"
	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

type compositionReadFake struct {
	requests []ReadRequest
	results  []ReadSnapshotResult
	errors   []error
}

func (f *compositionReadFake) ReadSnapshot(_ context.Context, request ReadRequest) (ReadSnapshotResult, error) {
	f.requests = append(f.requests, request)
	var result ReadSnapshotResult
	if len(f.results) > 0 {
		result, f.results = f.results[0], f.results[1:]
	}
	var err error
	if len(f.errors) > 0 {
		err, f.errors = f.errors[0], f.errors[1:]
	}
	return result, err
}

type compositionPullFake struct {
	previews              []PullPreviewRequest
	validates             []PullValidationRequest
	executes              []PullExecutionRequest
	preview               PullPreviewResult
	validation            PullValidationResult
	execution             PullExecutionResult
	rejectChangedBaseline bool
}

func (f *compositionPullFake) Preview(_ context.Context, request PullPreviewRequest) (PullPreviewResult, error) {
	f.previews = append(f.previews, request)
	return f.preview, nil
}
func (f *compositionPullFake) Validate(_ context.Context, request PullValidationRequest) (PullValidationResult, error) {
	f.validates = append(f.validates, request)
	if f.rejectChangedBaseline && request.Current != request.Expected {
		return PullValidationResult{Valid: false, Authorized: false, Reason: PullRejectChangedBaseline}, nil
	}
	return f.validation, nil
}
func (f *compositionPullFake) Execute(_ context.Context, request PullExecutionRequest) (PullExecutionResult, error) {
	f.executes = append(f.executes, request)
	return f.execution, nil
}

type compositionInspectorFake struct{}

func (compositionInspectorFake) InspectCommit(context.Context, commitinspector.CommitRequest) commitinspector.InspectorResult[commitinspector.CommitSnapshot] {
	return commitinspector.InspectorResult[commitinspector.CommitSnapshot]{}
}
func (compositionInspectorFake) LoadDiff(context.Context, commitinspector.DiffRequest) commitinspector.InspectorResult[commitinspector.DiffWindow] {
	return commitinspector.InspectorResult[commitinspector.DiffWindow]{}
}

type compositionTagFake struct{}

func (compositionTagFake) Load(context.Context) (TagProvenanceSnapshot, *ProvenanceError) {
	return TagProvenanceSnapshot{}, nil
}
func (compositionTagFake) Save(context.Context, TagProvenanceSnapshot) *ProvenanceError { return nil }

type compositionEventsFake struct{}

func (compositionEventsFake) Publish(events.Event) error { return nil }

func compositionSnapshot(epoch, request uint64, head string) ReadSnapshotResult {
	return ReadSnapshotResult{RequestID: request, RepositoryEpoch: epoch, ErrorKind: ReadErrorNone, Snapshot: ReadSnapshot{
		Root: "/sentinel/repository", Graph: graph.Snapshot{Branch: "main", Head: head, Commits: []graph.Commit{{Hash: head, Subject: "sentinel"}}},
		Freshness:  PullSnapshotIdentity{Epoch: epoch, Branch: "main", Head: head, Upstream: "origin/main", UpstreamOID: "2222222222222222222222222222222222222222", Ahead: 1, Behind: 2, TrackingKnown: true, TrackingFresh: true},
		Repository: RepositoryProjection{Root: "/sentinel/repository", Branch: "main", Head: head, Branches: []string{"main"}, LocalBranches: []string{"main"}, LocalBranchesKnown: true, LocalBranchesFresh: true},
	}}
}

func TestNewWithDependenciesRetainsInjectedPorts(t *testing.T) {
	read, pull := &compositionReadFake{}, &compositionPullFake{}
	inspector, tags, sink := compositionInspectorFake{}, compositionTagFake{}, compositionEventsFake{}
	got, err := NewWithDependencies(Dependencies{RepositoryRead: read, Pull: pull, InspectorReader: inspector, TagProvenance: tags, EventSink: sink})
	if err != nil {
		t.Fatal(err)
	}
	m := got.(model)
	if m.repositoryRead != read || m.pull != pull || m.inspectorReader != inspector || m.tagProvenance != tags || m.eventSink != sink {
		t.Fatalf("injected dependencies were not retained: %#v", m)
	}
	if !m.startupReadPending || m.startupFailed {
		t.Fatalf("unexpected injected startup flags: pending=%v failed=%v", m.startupReadPending, m.startupFailed)
	}
	legacy, err := NewWithDependencies(Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	legacyModel := legacy.(model)
	if legacyModel.startupReadPending || legacyModel.startupFailed {
		t.Fatalf("legacy startup flags set: %#v", legacyModel)
	}
}

func initReadMessage(t *testing.T, m model) loadedSnapshotMsg {
	t.Helper()
	value := m.Init()()
	batch, ok := value.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init returned %T, want tea.BatchMsg", value)
	}
	for _, cmd := range batch {
		if cmd == nil {
			continue
		}
		if msg, ok := cmd().(loadedSnapshotMsg); ok {
			return msg
		}
	}
	t.Fatal("Init batch had no loadedSnapshotMsg")
	return loadedSnapshotMsg{}
}

func TestCompositionInitUsesInjectedRepositoryReadPort(t *testing.T) {
	read := &compositionReadFake{results: []ReadSnapshotResult{compositionSnapshot(0, 1, "sentinel-head")}}
	initial, err := NewWithDependencies(Dependencies{RepositoryRead: read})
	if err != nil {
		t.Fatal(err)
	}
	m := initial.(model)
	loaded := initReadMessage(t, m)
	if len(read.requests) != 1 || read.requests[0] != (ReadRequest{RequestID: 1, RepositoryEpoch: 0, CommitLimit: 0}) {
		t.Fatalf("unexpected startup read request: %#v", read.requests)
	}
	updated, cmd := m.Update(loaded)
	if cmd != nil {
		t.Fatalf("startup update returned unexpected command: %v", cmd)
	}
	got := updated.(model)
	if got.repoStatus.Root != "/sentinel/repository" || got.repoStatus.Head != "sentinel-head" || !got.repoSnapshotLoaded || got.status.Mode != state.ModeBrowse || got.startupReadPending || got.startupFailed {
		t.Fatalf("injected snapshot was not projected: %#v", got)
	}
	if got.tagEntries != nil || got.stashEntries != nil || got.tagSyncAttempted {
		t.Fatalf("startup populated excluded state: %#v", got)
	}
	if got.sectionCursor[sectionGraph] < 0 {
		t.Fatalf("graph cursor not initialized: %#v", got.sectionCursor)
	}
}

func TestCompositionReadPortErrorTraversesUpdate(t *testing.T) {
	sentinel := errors.New("sentinel repository failure")
	read := &compositionReadFake{results: []ReadSnapshotResult{{RepositoryEpoch: 0, ErrorKind: ReadErrorRepository, Err: sentinel}}}
	initial, err := NewWithDependencies(Dependencies{RepositoryRead: read})
	if err != nil {
		t.Fatal(err)
	}
	m := initial.(model)
	updated, cmd := m.Update(initReadMessage(t, m))
	if cmd != nil {
		t.Fatalf("error startup returned command: %v", cmd)
	}
	failed := updated.(model)
	if !failed.startupFailed || failed.startupReadPending || failed.status.Mode != state.ModeBlocked || failed.err == nil {
		t.Fatalf("unexpected startup failure: %#v", failed)
	}
	if _, quit := failed.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}); quit == nil || quit() != tea.Quit() {
		t.Fatalf("q did not quit through top-level Update")
	}
}

func TestCompositionLegacyReadFallbackRemainsExplicit(t *testing.T) {
	fixture := newCommandRepo(t)
	read := &compositionReadFake{}
	initial, err := NewWithDependencies(Dependencies{Repo: fixture.repo, RepositoryRead: nil})
	if err != nil {
		t.Fatal(err)
	}
	m := initial.(model)
	value := m.Init()()
	batch, ok := value.(tea.BatchMsg)
	if !ok {
		t.Fatalf("legacy Init returned %T, want tea.BatchMsg", value)
	}
	var loaded tea.Msg
	for _, cmd := range batch {
		if cmd == nil {
			continue
		}
		candidate := cmd()
		if _, neutral := candidate.(loadedSnapshotMsg); neutral {
			t.Fatal("legacy Init selected neutral reader")
		}
		if _, ok := candidate.(loadedMsg); ok {
			loaded = candidate
			break
		}
	}
	if loaded == nil {
		t.Fatal("legacy Init had no loadedMsg")
	}
	updated, _ := m.Update(loaded)
	got := updated.(model)
	if got.repositoryRead != nil || got.startupReadPending || got.startupFailed || got.status.Mode != state.ModeBrowse || got.repoStatus.Root == "" {
		t.Fatalf("legacy fallback did not remain operational: %#v", got)
	}
	if len(read.requests) != 0 {
		t.Fatalf("unused reader was called: %#v", read.requests)
	}
	if _, quit := got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}); quit == nil {
		t.Fatal("legacy Browse q no longer quits")
	}
}

func compositionPullStatus(epoch uint64) (git.Status, PullSnapshotIdentity) {
	head := "1111111111111111111111111111111111111111"
	upstreamOID := "2222222222222222222222222222222222222222"
	status := git.Status{Root: "/sentinel/repository", Branch: "main", Head: head, Upstream: "origin/main", UpstreamOID: upstreamOID, TrackingKnown: true, TrackingFresh: true, Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 2}}}
	return status, pullSnapshotIdentity(status, epoch)
}

func TestCompositionPullPortPreviewUsesInjectedPort(t *testing.T) {
	pull := &compositionPullFake{}
	status, baseline := compositionPullStatus(7)
	pull.preview = PullPreviewResult{RequestID: 4, RepositoryEpoch: 7, Baseline: baseline, Eligible: true, Commits: []PullPreviewCommit{{Hash: "remote-commit"}}}
	m := model{
		repositoryState: repositoryState{repoStatus: status, repositoryEpoch: 7},
		pullState:       pullState{activePullRequest: &pullRequest{ID: 4, Epoch: 7, FetchBaseline: baseline}}, pull: pull, commitLimit: 37, status: state.New().WithBrowse()}
	cmd := startPullPreview(&m, *m.activePullRequest)
	if cmd == nil {
		t.Fatal("preview command was nil")
	}
	updated, updateCmd := m.Update(cmd())
	if updateCmd != nil {
		t.Fatalf("preview update returned command: %v", updateCmd)
	}
	got := updated.(model)
	if len(pull.previews) != 1 || pull.previews[0] != (PullPreviewRequest{RequestID: 4, RepositoryEpoch: 7, Baseline: baseline, Mode: PullModeMerge, CommitLimit: 37}) {
		t.Fatalf("unexpected preview request: %#v", pull.previews)
	}
	if got.activePullRequest == nil || !got.activePullRequest.OperationBaselineSet || got.activePullRequest.OperationBaseline != baseline || got.status.Mode != state.ModeConfirm {
		t.Fatalf("preview did not establish confirmation baseline: %#v", got)
	}
}

func TestCompositionPullPortPreviewRejectsMismatchedRequestFetchBaseline(t *testing.T) {
	pull := &compositionPullFake{}
	status, baseline := compositionPullStatus(7)
	requestFetchBaseline := baseline
	requestFetchBaseline.UpstreamOID = "3333333333333333333333333333333333333333"
	active := &pullRequest{ID: 4, Epoch: 7, FetchBaseline: requestFetchBaseline}
	m := model{
		repositoryState: repositoryState{
			repoStatus:      status,
			repositoryEpoch: 7,
		},
		pullState: pullState{
			activePullRequest: active,
		},
		pull:        pull,
		commitLimit: 37,
		status:      state.New().WithLoading("Previewing...")}

	updated, refreshCmd := m.Update(pullPortPreviewMsg{result: PullPreviewResult{
		RequestID: 4, RepositoryEpoch: 7, Baseline: baseline, Eligible: true,
		Commits: []PullPreviewCommit{{Hash: "remote-commit"}},
	}})
	got := updated.(model)
	if got.status.Mode == state.ModeConfirm || got.activePullRequest != nil || active.OperationBaselineSet || active.OperationBaseline != (PullSnapshotIdentity{}) {
		t.Fatalf("mismatched fetch baseline entered confirmation or mutated operation baseline: status=%#v active=%#v request=%#v", got.status, got.activePullRequest, active)
	}
	if got.status.Block != state.BlockStaleSnapshot || refreshCmd == nil {
		t.Fatalf("mismatched fetch baseline did not preserve stale-block/refresh behavior: status=%#v cmd=%v", got.status, refreshCmd)
	}
}

func TestCompositionPullPortExecutionUsesInjectedPorts(t *testing.T) {
	pull := &compositionPullFake{}
	status, baseline := compositionPullStatus(7)
	pull.validation = PullValidationResult{Valid: true, Authorized: true, AuthorizedBaseline: baseline}
	pull.execution = PullExecutionResult{Succeeded: true, RequestID: 99, RepositoryEpoch: 7, Mode: PullModeMerge, Reason: PullRejectNone}
	read := &compositionReadFake{results: []ReadSnapshotResult{compositionSnapshot(7, 4, status.Head), compositionSnapshot(8, 6, status.Head)}}
	m := model{
		repositoryState: repositoryState{repoStatus: status, repositoryEpoch: 7},
		pullState:       pullState{pullIsFastForward: false, activePullRequest: &pullRequest{ID: 4, Epoch: 7, FetchBaseline: baseline}}, pull: pull, repositoryRead: read, commitLimit: 37, status: state.New().WithLoading("Previewing...")}
	previewed, previewCmd := m.Update(pullPortPreviewMsg{result: PullPreviewResult{RequestID: 4, RepositoryEpoch: 7, Baseline: baseline, Eligible: true, Commits: []PullPreviewCommit{{Hash: "remote-commit"}}}})
	if previewCmd != nil {
		t.Fatalf("preview update returned unexpected command: %v", previewCmd)
	}
	confirmed := previewed.(model)
	if confirmed.status.Mode != state.ModeConfirm {
		t.Fatalf("preview did not route through confirmation mode: %#v", confirmed.status)
	}
	updated, cmd := confirmed.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd == nil {
		t.Fatal("merge confirmation did not start workflow")
	}
	workflowMsg, ok := cmd().(pullWorkflowMsg)
	if !ok {
		t.Fatalf("workflow command returned %T", cmd())
	}
	final, finalCmd := updated.(model).Update(workflowMsg)
	if finalCmd != nil {
		t.Fatalf("successful workflow returned unexpected command: %v", finalCmd)
	}
	got := final.(model)
	if len(read.requests) != 2 || read.requests[0] != (ReadRequest{RequestID: 1, RepositoryEpoch: 7, CommitLimit: 37}) || read.requests[1] != (ReadRequest{RequestID: 2, RepositoryEpoch: 8, CommitLimit: 37}) {
		t.Fatalf("unexpected refresh/read requests: %#v", read.requests)
	}
	if len(pull.validates) != 1 || pull.validates[0] != (PullValidationRequest{RequestID: 1, RepositoryEpoch: 7, Current: baseline, Expected: baseline, Mode: PullModeMerge}) {
		t.Fatalf("unexpected validation request: %#v", pull.validates)
	}
	if len(pull.executes) != 1 || pull.executes[0] != (PullExecutionRequest{RequestID: 1, RepositoryEpoch: 7, Authorized: true, AuthorizedBaseline: baseline, Mode: PullModeMerge}) {
		t.Fatalf("unexpected execution request: %#v", pull.executes)
	}
	if workflowMsg.result.OperationRequestID != 1 || workflowMsg.result.OperationEpoch != 7 || workflowMsg.result.RefreshRequestID != 2 || workflowMsg.result.RefreshEpoch != 8 || workflowMsg.result.Execute.RequestID != 99 || workflowMsg.result.Execute.RepositoryEpoch != 7 || !workflowMsg.result.Execute.Succeeded || got.status.Mode != state.ModeOperationResult {
		t.Fatalf("unexpected workflow/result transition: %#v", workflowMsg.result)
	}
}

func TestCompositionPullPortStaleMessagesThroughUpdate(t *testing.T) {
	baseline := PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "head"}
	m := model{
		pullState: pullState{activePullRequest: &pullRequest{ID: 4, Epoch: 3, OperationBaseline: baseline}}, status: state.New().WithConfirm(state.ActionPull, "Pull?", "confirm")}
	before := m
	for _, msg := range []tea.Msg{
		pullPortPreviewMsg{result: PullPreviewResult{RequestID: 5, RepositoryEpoch: 3}},
		pullValidationMsg{requestID: 5, requestEpoch: 3, baseline: baseline, valid: true},
		pullExecutionResultMsg{requestID: 5, requestEpoch: 3, baseline: baseline},
		pullWorkflowMsg{result: PullWorkflowResult{OperationRequestID: 5, OperationEpoch: 3}},
		pullPortPreviewMsg{result: PullPreviewResult{RequestID: 4, RepositoryEpoch: 4}},
		pullValidationMsg{requestID: 4, requestEpoch: 4, baseline: baseline, valid: true},
		pullExecutionResultMsg{requestID: 4, requestEpoch: 4, baseline: baseline},
		pullWorkflowMsg{result: PullWorkflowResult{OperationRequestID: 4, OperationEpoch: 4}},
	} {
		got, cmd := m.Update(msg)
		if cmd != nil || !reflect.DeepEqual(got, before) {
			t.Fatalf("stale message mutated composition state: msg=%T model=%#v cmd=%v", msg, got, cmd)
		}
	}
}

func TestCompositionPullPortChangedBaselineThroughWorkflowUpdate(t *testing.T) {
	pull := &compositionPullFake{rejectChangedBaseline: true}
	status, baseline := compositionPullStatus(3)
	current := compositionSnapshot(3, 1, "changed-head")
	read := &compositionReadFake{results: []ReadSnapshotResult{current}}
	m := model{
		repositoryState: repositoryState{
			repoStatus:      status,
			repositoryEpoch: 3,
		},
		pullState: pullState{
			activePullRequest: &pullRequest{ID: 4, Epoch: 3, OperationBaseline: baseline, OperationBaselineSet: true},
		},
		pull:           pull,
		repositoryRead: read,
		commitLimit:    37,
		status:         state.New().WithConfirm(state.ActionPull, "Pull?", "confirm")}

	cmd := startPullWorkflow(&m, PullModeMerge)
	if cmd == nil {
		t.Fatal("changed-baseline workflow command was nil")
	}
	workflowMsg, ok := cmd().(pullWorkflowMsg)
	if !ok {
		t.Fatalf("workflow command returned %T", cmd())
	}
	if len(read.requests) != 1 {
		t.Fatalf("changed-baseline workflow performed %d reads, want 1: %#v", len(read.requests), read.requests)
	}
	if len(pull.validates) != 1 {
		t.Fatalf("changed-baseline workflow performed %d validations, want 1", len(pull.validates))
	}
	if pull.validates[0].Current != current.Snapshot.Freshness || pull.validates[0].Expected != baseline {
		t.Fatalf("validation did not receive current/expected freshness: %#v", pull.validates[0])
	}
	if len(pull.executes) != 0 {
		t.Fatalf("changed-baseline workflow executed unauthorized pull: %#v", pull.executes)
	}
	if workflowMsg.result.RefreshErrorKind != ReadErrorNone || workflowMsg.result.Execute.Reason != PullRejectChangedBaseline {
		t.Fatalf("unexpected changed-baseline workflow result: %#v", workflowMsg.result)
	}

	updated, refreshCmd := m.Update(workflowMsg)
	got := updated.(model)
	if got.status.Block != state.BlockStaleSnapshot || refreshCmd == nil {
		t.Fatalf("changed-baseline workflow was not blocked and refreshed: status=%#v cmd=%v", got.status, refreshCmd)
	}
}

func TestCompositionBatchDoesNotAssumeCommandOrder(t *testing.T) {
	read := &compositionReadFake{results: []ReadSnapshotResult{compositionSnapshot(0, 1, "batch-head")}}
	initial, err := NewWithDependencies(Dependencies{RepositoryRead: read})
	if err != nil {
		t.Fatal(err)
	}
	m := initial.(model)
	value := m.Init()()
	batch := value.(tea.BatchMsg)
	var found bool
	for _, cmd := range batch {
		if cmd == nil {
			continue
		}
		switch cmd().(type) {
		case loadedSnapshotMsg:
			found = true
		case tickMsg:
			// Deliberately ignored: refresh timing is not a composition contract.
		}
	}
	if !found {
		t.Fatal("BatchMsg did not contain a typed startup read command")
	}
	if len(read.requests) != 1 {
		t.Fatalf("tick or command ordering caused extra reads: %#v", read.requests)
	}
}

var _ events.EventSink = compositionEventsFake{}
var _ commitinspector.CommitInspectorReader = compositionInspectorFake{}
var _ TagProvenanceStore = compositionTagFake{}
