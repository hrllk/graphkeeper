package app

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

type fakeRepositoryRead struct {
	result      ReadSnapshotResult
	returnedErr error
	request     ReadRequest
}

func (f *fakeRepositoryRead) ReadSnapshot(_ context.Context, request ReadRequest) (ReadSnapshotResult, error) {
	f.request = request
	return f.result, f.returnedErr
}

func TestLoadRepositorySnapshotUsesNeutralPortAndRequestIdentity(t *testing.T) {
	fake := &fakeRepositoryRead{result: ReadSnapshotResult{
		ErrorKind: ReadErrorNone, RepositoryEpoch: 4,
		Snapshot: ReadSnapshot{Root: "/repo", Graph: graph.Snapshot{Branch: "main", Head: "abc"}},
	}}
	msg := loadRepositorySnapshot(fake, 25, 4)()
	loaded, ok := msg.(loadedSnapshotMsg)
	if !ok || loaded.result.Snapshot.Graph.Head != "abc" {
		t.Fatalf("unexpected message: %#v", msg)
	}
	if fake.request.CommitLimit != 25 || fake.request.RepositoryEpoch != 4 {
		t.Fatalf("request identity not forwarded: %#v", fake.request)
	}
}

func TestLifecycleRejectsStaleNeutralSnapshotAndPreservesGraph(t *testing.T) {
	m := model{
		repositoryState: repositoryState{repositoryEpoch: 9, graphReadSnapshot: graph.Snapshot{Head: "current"}},
		pullState:       pullState{}}
	got, _ := handleLifecycleUpdate(m, loadedSnapshotMsg{result: ReadSnapshotResult{
		RepositoryEpoch: 8, ErrorKind: ReadErrorNone,
		Snapshot: ReadSnapshot{Graph: graph.Snapshot{Head: "stale"}},
	}})
	modelGot := got.(model)
	if modelGot.graphReadSnapshot.Head != "current" {
		t.Fatalf("stale snapshot replaced current graph: %#v", modelGot.graphReadSnapshot)
	}
}

func TestLifecycleAcceptsCurrentNeutralSnapshot(t *testing.T) {
	m := model{
		repositoryState: repositoryState{repositoryEpoch: 9},
		pullState:       pullState{}}
	got, _ := handleLifecycleUpdate(m, loadedSnapshotMsg{result: ReadSnapshotResult{
		RepositoryEpoch: 9, ErrorKind: ReadErrorNone,
		Snapshot: ReadSnapshot{Graph: graph.Snapshot{Head: "fresh"}},
	}})
	if got.(model).graphReadSnapshot.Head != "fresh" {
		t.Fatal("current neutral snapshot was not projected")
	}
}

func TestStartupProjectionCopiesRepositoryFactsAndGraphCommits(t *testing.T) {
	when := time.Unix(123, 0)
	projection := RepositoryProjection{
		Root: "/repo", Branch: "main", Head: "head", Upstream: "origin/main", UpstreamOID: "up", Remote: "origin", DefaultBranch: "main",
		Branches: []string{"main", "dev"}, LocalBranches: []string{"main"}, LocalBranchesKnown: true, LocalBranchesFresh: true, LocalBranchesError: "local-ok",
		RemoteBranches: []string{"origin/main"}, BranchUpstreams: map[string]string{"main": "origin/main"}, Tracking: map[string]BranchTracking{"main": {Ahead: 2, Behind: 3}}, TrackingKnown: true, TrackingFresh: true,
		WorktreeDirty: true, Detached: true, EmptyRepo: true, NoRemote: true, NoUpstream: true, UpstreamGone: true,
		MergeInProgress: true, RebaseInProgress: true, CherryPickInProgress: true, ConflictTarget: "target", ConflictTargetSubject: "subject", LastFetchAt: when, RemoteSyncSummary: "sync",
	}
	status := applyRepositoryProjection(projection, graph.Snapshot{Commits: []graph.Commit{{Hash: "head", Parents: []string{"base"}, Tags: []string{"v1"}}}})
	if status.Root != projection.Root || status.Branch != projection.Branch || status.Upstream != projection.Upstream || status.DefaultBranch != projection.DefaultBranch || status.LocalBranchesError != projection.LocalBranchesError || !status.TrackingKnown || status.Tracking["main"].Behind != 3 || !status.CherryPickInProgress || !status.LastFetchAt.Equal(when) || status.RemoteSyncSummary != "sync" {
		t.Fatalf("projection lost repository facts: %#v", status)
	}
	if len(status.GraphCommits) != 1 || status.GraphCommits[0].Hash != "head" || status.GraphCommits[0].Parents[0] != "base" {
		t.Fatalf("projection lost graph commits: %#v", status.GraphCommits)
	}
	if status.Tags != nil || status.TagEntries != nil || status.TagEntriesLoaded || status.TagProvenanceLoaded {
		t.Fatalf("neutral startup populated excluded tag state: %#v", status)
	}
}

func TestNormalizeReadSnapshotResultPrecedence(t *testing.T) {
	returned := errors.New("returned")
	stored := errors.New("stored")
	cases := []struct {
		name     string
		in       ReadSnapshotResult
		returned error
		kind     ReadErrorKind
		canceled bool
		err      error
	}{
		{"stored error wins", ReadSnapshotResult{Err: stored}, returned, ReadErrorRepository, false, stored},
		{"explicit kind wins", ReadSnapshotResult{ErrorKind: ReadErrorInvalid, Canceled: true}, returned, ReadErrorInvalid, false, returned},
		{"canceled bit", ReadSnapshotResult{Canceled: true}, nil, ReadErrorCanceled, true, nil},
		{"returned error", ReadSnapshotResult{}, returned, ReadErrorRepository, false, returned},
		{"success clears", ReadSnapshotResult{Canceled: false}, nil, ReadErrorNone, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeReadSnapshotResult(tc.in, tc.returned)
			if got.ErrorKind != tc.kind || got.Canceled != tc.canceled || !errors.Is(got.Err, tc.err) {
				t.Fatalf("got kind=%q canceled=%v err=%v", got.ErrorKind, got.Canceled, got.Err)
			}
		})
	}
}

func TestStartupFailureAndQuitAreReachable(t *testing.T) {
	m := model{
		repositoryState: repositoryState{repositoryEpoch: 1},
		pullState:       pullState{}, startupReadPending: true, status: state.New()}
	got, _ := handleLifecycleUpdate(m, loadedSnapshotMsg{result: ReadSnapshotResult{RepositoryEpoch: 1, ErrorKind: ReadErrorInvalid}})
	failed := got.(model)
	if failed.status.Mode != state.ModeBlocked || failed.status.Block != state.BlockNoRepo || failed.startupReadPending || !failed.startupFailed || failed.repoSnapshotLoaded {
		t.Fatalf("unexpected startup failure state: %#v", failed)
	}
	_, cmd := failed.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c did not quit from startup failure")
	}
	pending := model{startupReadPending: true, status: state.New()}
	_, cmd = pending.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q did not quit during startup loading")
	}
}
func TestRepositoryReadCommandPreservesErrors(t *testing.T) {
	for _, returnedErr := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(returnedErr.Error(), func(t *testing.T) {
			fake := &fakeRepositoryRead{result: ReadSnapshotResult{RequestID: 77, RepositoryEpoch: 12, ErrorKind: ReadErrorCanceled, Canceled: true}, returnedErr: returnedErr}
			msg, ok := loadRepositorySnapshot(fake, 25, 12)().(loadedSnapshotMsg)
			if !ok {
				t.Fatalf("expected loadedSnapshotMsg, got %T", loadRepositorySnapshot(fake, 25, 12)())
			}
			if fake.request.RequestID != 1 || fake.request.RepositoryEpoch != 12 || fake.request.CommitLimit != 25 {
				t.Fatalf("request identity not forwarded: %#v", fake.request)
			}
			if msg.result.ErrorKind != ReadErrorCanceled || !msg.result.Canceled || !errors.Is(msg.result.Err, returnedErr) || msg.result.RequestID != 77 || msg.result.RepositoryEpoch != 12 {
				t.Fatalf("returned error was not preserved: %#v", msg.result)
			}
		})
	}
}

func TestRefreshSnapshotCommandPreservesErrors(t *testing.T) {
	for _, returnedErr := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(returnedErr.Error(), func(t *testing.T) {
			fake := &fakeRepositoryRead{result: ReadSnapshotResult{RequestID: 9, RepositoryEpoch: 4, ErrorKind: ReadErrorCanceled, Canceled: true}, returnedErr: returnedErr}
			m := model{
				repositoryState: repositoryState{repositoryEpoch: 4},
				pullState:       pullState{}, repositoryRead: fake}
			msg, ok := m.refreshCmd()().(refreshedSnapshotMsg)
			if !ok {
				t.Fatalf("expected refreshedSnapshotMsg, got %T", m.refreshCmd()())
			}
			if fake.request.RequestID != 1 || fake.request.RepositoryEpoch != 4 || fake.request.CommitLimit != 0 {
				t.Fatalf("request identity not forwarded: %#v", fake.request)
			}
			if msg.refreshGeneration != 1 || msg.result.ErrorKind != ReadErrorCanceled || !msg.result.Canceled || !errors.Is(msg.result.Err, returnedErr) || msg.result.RequestID != 9 || msg.result.RepositoryEpoch != 4 {
				t.Fatalf("returned refresh error was not preserved: %#v", msg)
			}
		})
	}
}

func TestLifecycleStaleFailurePreservesState(t *testing.T) {
	priorErr := errors.New("prior")
	m := model{
		repositoryState: repositoryState{
			repositoryEpoch: 9, graphReadSnapshot: graph.Snapshot{Head: "current"},
			repoStatus: git.Status{Head: "current"}, err: priorErr,
		},
		status:             state.New().WithBrowse(),
		startupReadPending: true}
	got, cmd := handleLifecycleUpdate(m, loadedSnapshotMsg{result: ReadSnapshotResult{
		RepositoryEpoch: 8, ErrorKind: ReadErrorRepository, Err: errors.New("stale"),
	}})
	if cmd != nil {
		t.Fatalf("stale failure returned command: %v", cmd)
	}
	actual := got.(model)
	if actual.graphReadSnapshot.Head != "current" || actual.repoStatus.Head != "current" || !reflect.DeepEqual(actual.status, m.status) || !errors.Is(actual.err, priorErr) || !actual.startupReadPending || actual.startupFailed {
		t.Fatalf("stale failure changed existing state: %#v", actual)
	}
}

func TestNeutralStartupInitBatchIntegration(t *testing.T) {
	fake := &fakeRepositoryRead{result: ReadSnapshotResult{
		RequestID: 1, RepositoryEpoch: 0, ErrorKind: ReadErrorNone,
		Snapshot: ReadSnapshot{Repository: RepositoryProjection{Root: "/repo", Branch: "main", Head: "head", Branches: []string{"main"}, LocalBranches: []string{"main"}}, Graph: graph.Snapshot{
			Branch: "main", Head: "head", Commits: []graph.Commit{{Graph: "*", Hash: "head", Subject: "visible"}},
		}},
	}}
	initial, err := NewWithDependencies(Dependencies{RepositoryRead: fake})
	if err != nil {
		t.Fatal(err)
	}
	initialModel := initial.(model)
	batchValue := initialModel.Init()()
	batch, ok := batchValue.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init command returned %T, want tea.BatchMsg", batchValue)
	}
	var loaded tea.Msg
	for _, cmd := range batch {
		if cmd == nil {
			continue
		}
		candidate := cmd()
		if _, ok := candidate.(loadedSnapshotMsg); ok {
			loaded = candidate
			break
		}
	}
	if loaded == nil {
		t.Fatal("Init batch did not contain a loadedSnapshotMsg command")
	}
	updated, cmd := initialModel.Update(loaded)
	if cmd != nil {
		t.Fatalf("neutral startup scheduled an unexpected command: %v", cmd)
	}
	got := updated.(model)
	if got.status.Mode != state.ModeBrowse || !got.repoSnapshotLoaded || got.startupReadPending || got.startupFailed || got.err != nil {
		t.Fatalf("neutral startup did not reach browse: %#v", got)
	}
	if got.repoStatus.Head != "head" || len(got.screenProjection(80, 20).Graph.Rows) == 0 || got.screenProjection(80, 20).Graph.Rows[0].Commit.Hash != "head" {
		t.Fatalf("production graph projection is empty: %#v", got.screenProjection(80, 20).Graph)
	}
	if got.sectionCursor[sectionGraph] < 0 || got.sectionCursor[sectionCurrent] != 0 || got.sectionCursor[sectionRemote] != -1 || got.sectionCursor[sectionTags] != -1 || got.graphLaneCursor < 0 {
		t.Fatalf("syncBrowseState cursors were not initialized: %#v", got.sectionCursor)
	}
	if got.tagEntries != nil || got.tagSyncAttempted || got.stashEntries != nil {
		t.Fatalf("neutral startup loaded excluded tag/stash state: %#v", got)
	}
}

func TestLegacyStartupAndModalQuitPreserved(t *testing.T) {
	legacy := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{sectionGraph: 0},
			activeSection: sectionGraph,
		}, status: loadingToast("Loading...")}
	legacy.status = state.New().WithLoading("Loading...")
	updated, _ := legacy.Update(loadedMsg{status: git.Status{Root: "/repo", Branch: "main", Head: "head", GraphCommits: []git.GraphCommit{{Hash: "head"}}}})
	got := updated.(model)
	if got.status.Mode != state.ModeBrowse || got.startupReadPending || got.startupFailed {
		t.Fatalf("legacy startup flags/status changed unexpectedly: %#v", got)
	}
	_, quit := got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if quit == nil {
		t.Fatal("Browse q no longer quits")
	}

	overlays := []struct {
		name   string
		open   func(*model)
		closed func(model) bool
	}{
		{"graph stash", func(m *model) { m.graphStashPopOpen = true }, func(m model) bool { return !m.graphStashPopOpen }},
		{"stash message", func(m *model) { m.stashMessageOpen = true }, func(m model) bool { return !m.stashMessageOpen }},
		{"tag popup", func(m *model) { m.tagPopupOpen = true }, func(m model) bool { return !m.tagPopupOpen }},
		{"stash popup", func(m *model) { m.stashPopupOpen = true }, func(m model) bool { return !m.stashPopupOpen }},
		{"branch", func(m *model) { m.branchOpen = true }, func(m model) bool { return !m.branchOpen }},
		{"hotkeys", func(m *model) { m.hiddenHotkeysOpen = true }, func(m model) bool { return !m.hiddenHotkeysOpen }},
		{"search", func(m *model) { m.graphSearchOpen = true }, func(m model) bool { return !m.graphSearchOpen }},
	}
	for _, overlay := range overlays {
		t.Run(overlay.name, func(t *testing.T) {
			m := model{status: state.New().WithBrowse()}
			overlay.open(&m)
			next, quit := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
			if quit != nil || !overlay.closed(next.(model)) {
				t.Fatalf("q did not dismiss overlay: model=%#v cmd=%v", next, quit)
			}
		})
	}
	blocked := model{status: state.New().WithBlocked(state.BlockUnknown, "blocked", "detail")}
	next, quit := blocked.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if quit != nil || next.(model).status.Mode != state.ModeBlocked {
		t.Fatalf("non-startup blocked q changed behavior: model=%#v cmd=%v", next, quit)
	}
}

func TestNormalizeReadSnapshotResultCartesian(t *testing.T) {
	kinds := []ReadErrorKind{ReadErrorNone, ReadErrorInvalid, ReadErrorRepository, ReadErrorCanceled}
	for _, kind := range kinds {
		for _, canceled := range []bool{false, true} {
			for _, stored := range []bool{false, true} {
				for _, returned := range []bool{false, true} {
					t.Run(fmt.Sprintf("kind=%s/canceled=%t/stored=%t/returned=%t", kind, canceled, stored, returned), func(t *testing.T) {
						storedErr, returnedErr := errors.New("stored"), errors.New("returned")
						in := ReadSnapshotResult{RequestID: 17, RepositoryEpoch: 23, ErrorKind: kind, Canceled: canceled}
						if stored {
							in.Err = storedErr
						}
						if !returned {
							returnedErr = nil
						}
						got := normalizeReadSnapshotResult(in, returnedErr)
						wantErr := returnedErr
						if stored {
							wantErr = storedErr
						}
						wantKind, wantCanceled := ReadErrorNone, false
						if kind != ReadErrorNone {
							wantKind, wantCanceled = kind, kind == ReadErrorCanceled
						} else if canceled {
							wantKind, wantCanceled = ReadErrorCanceled, true
						} else if wantErr != nil {
							wantKind = ReadErrorRepository
						}
						if got.ErrorKind != wantKind || got.Canceled != wantCanceled || !errors.Is(got.Err, wantErr) || got.RequestID != 17 || got.RepositoryEpoch != 23 {
							t.Fatalf("got kind=%q canceled=%v err=%v identity=%d/%d; want %q/%v err=%v", got.ErrorKind, got.Canceled, got.Err, got.RequestID, got.RepositoryEpoch, wantKind, wantCanceled, wantErr)
						}
					})
				}
			}
		}
	}
}

func TestLifecycleStalePullDuringLoadingCancelsAndRefreshes(t *testing.T) {
	cancelled := 0
	read := &fakeRepositoryRead{}
	m := model{
		repositoryState: repositoryState{
			repositoryEpoch:   4,
			refreshGeneration: 9,
		},
		pullState: pullState{
			activePullRequest: &pullRequest{ID: 10, Epoch: 4},
			pullCancel:        func() { cancelled++ },
			nextPullRequestID: 10,
		},
		repositoryRead: read,
		status: func() state.Status {
			s := state.New().WithLoading("Analyzing pull...")
			s.Action = state.ActionPull
			return s
		}()}
	got, cmd := handleLifecycleUpdate(m, refreshedSnapshotMsg{
		refreshGeneration: 8,
		result:            ReadSnapshotResult{RepositoryEpoch: 3},
	})
	modelGot := got.(model)
	if cancelled != 1 || modelGot.pullCancel != nil || modelGot.activePullRequest != nil || modelGot.pullConfirmStale {
		t.Fatalf("stale loading pull was not invalidated: cancelled=%d model=%#v", cancelled, modelGot)
	}
	if modelGot.nextPullRequestID != 11 || modelGot.status.Block != state.BlockStaleSnapshot || cmd == nil {
		t.Fatalf("unexpected stale transition: next=%d status=%#v cmd=%v", modelGot.nextPullRequestID, modelGot.status, cmd)
	}
}
