package app

import (
	"reflect"
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestHandleLifecycleUpdateDiscardsStaleRefresh(t *testing.T) {
	m := model{
		repositoryState: repositoryState{
			repositoryEpoch: 2,
			repoStatus:      git.Status{Root: "/repo", Branch: "feature", Head: "new-head"},
		},
		status: state.New().WithBrowse()}

	got, cmd := handleLifecycleUpdate(m, refreshedMsg{
		status:   git.Status{Root: "/repo", Branch: "main", Head: "old-head"},
		epoch:    1,
		epochSet: true,
	})

	if cmd != nil {
		t.Fatal("stale refresh must not schedule follow-up work")
	}
	updated := got.(model)
	if updated.repoStatus.Head != "new-head" {
		t.Fatalf("stale refresh changed HEAD to %q", updated.repoStatus.Head)
	}
}

func TestHandleLifecycleUpdateDiscardsStaleInitialLoad(t *testing.T) {
	m := model{
		repositoryState: repositoryState{
			repositoryEpoch: 1,
			repoStatus:      git.Status{Root: "/repo", Branch: "feature", Head: "new-head"},
		},
		status: state.New().WithBrowse()}

	got, cmd := handleLifecycleUpdate(m, loadedMsg{
		status:   git.Status{Root: "/repo", Branch: "main", Head: "old-head"},
		epoch:    0,
		epochSet: true,
	})

	if cmd != nil {
		t.Fatal("stale initial load must not schedule follow-up work")
	}
	updated := got.(model)
	if updated.repoStatus.Head != "new-head" {
		t.Fatalf("stale initial load changed HEAD to %q", updated.repoStatus.Head)
	}
}

func TestHandleLifecycleUpdateClearsErrorAfterSuccessfulLoad(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: testSectionCursors(),
		},
		repositoryState: repositoryState{
			err: errRepositoryUnavailable,
		},
		pullState: pullState{},
		status:    state.New().WithError("repository unavailable")}

	got, _ := handleLifecycleUpdate(m, loadedMsg{status: git.Status{Root: "/repo", Branch: "main"}})
	updated := got.(model)
	if updated.err != nil {
		t.Fatalf("successful load retained error: %v", updated.err)
	}
}

func TestHandleLifecycleUpdateClearsErrorAfterSuccessfulRefresh(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: testSectionCursors(),
		},
		repositoryState: repositoryState{
			err:             errRepositoryUnavailable,
			repositoryEpoch: 2,
			repoStatus:      git.Status{Root: "/repo", Branch: "old"},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}

	got, _ := handleLifecycleUpdate(m, refreshedMsg{status: git.Status{Root: "/repo", Branch: "main"}, epoch: 2, epochSet: true})
	updated := got.(model)
	if updated.err != nil {
		t.Fatalf("successful refresh retained error: %v", updated.err)
	}
}

func TestHandleLifecycleUpdateLegacyLoadUsesAppliedStatusOnce(t *testing.T) {
	originalSync := applyRepositoryStatusSyncBrowseState
	var synced []git.Status
	applyRepositoryStatusSyncBrowseState = func(_ *model, status git.Status) {
		synced = append(synced, status)
	}
	defer func() { applyRepositoryStatusSyncBrowseState = originalSync }()

	sink := &recordingEventSink{}
	m := model{
		navigationState: navigationState{
			sectionCursor: testSectionCursors(),
		},
		repositoryState: repositoryState{
			tagEntries: []git.TagEntry{{Name: "cached", CommitHash: "cached-head"}},
		},
		pullState: pullState{},
		repo:      &git.Repo{},
		status:    state.New().WithLoading("Loading..."),
		eventSink: sink}
	incoming := git.Status{Root: "/repo", Branch: "main", Head: "fresh", GraphCommits: []git.GraphCommit{{Hash: "fresh"}}}

	got, cmd := handleLifecycleUpdate(m, loadedMsg{status: incoming})
	updated := got.(model)

	if cmd == nil {
		t.Fatal("successful legacy load must retain loadStashState command ownership")
	}
	want := incoming
	want.Tags = []string{"cached"}
	want.TagEntries = []git.TagEntry{{Name: "cached", CommitHash: "cached-head"}}
	want.TagProvenanceLoaded = false
	if !reflect.DeepEqual(updated.repoStatus, want) {
		t.Fatalf("repoStatus = %#v, want %#v", updated.repoStatus, want)
	}
	if updated.status.Mode != state.ModeBrowse || updated.status.Message == "Loading..." {
		t.Fatalf("successful load did not derive browse status: %#v", updated.status)
	}
	if len(synced) != 1 || !reflect.DeepEqual(synced[0], want) {
		t.Fatalf("sync calls/status = %#v, want exactly one call with %#v", synced, want)
	}
	if len(sink.events) != 1 || sink.events[0].Name != "load_repo" || sink.events[0].Fields["root"] != "/repo" || sink.events[0].Fields["branch"] != "main" || sink.events[0].Fields["head"] != "fresh" {
		t.Fatalf("unexpected load telemetry: %#v", sink.events)
	}
}

func TestHandleLifecycleUpdateLegacyRefreshUsesAppliedStatusAndPreservesBranchInput(t *testing.T) {
	originalSync := applyRepositoryStatusSyncBrowseState
	var syncCalls int
	applyRepositoryStatusSyncBrowseState = func(_ *model, _ git.Status) { syncCalls++ }
	defer func() { applyRepositoryStatusSyncBrowseState = originalSync }()

	m := model{
		navigationState: navigationState{
			sectionCursor: testSectionCursors(),
		},
		repositoryState: repositoryState{
			tagEntries: []git.TagEntry{{Name: "cached"}},
		},
		pullState:    pullState{},
		repo:         &git.Repo{},
		status:       state.New().WithBrowse(),
		overlayState: overlayState{branchOpen: true}}
	incoming := git.Status{Root: "/repo", Branch: "feature", Head: "new-head"}

	got, cmd := handleLifecycleUpdate(m, refreshedMsg{status: incoming})
	updated := got.(model)

	if cmd == nil {
		t.Fatal("successful legacy refresh must retain loadStashState command ownership")
	}
	if updated.repoStatus.Root != incoming.Root || updated.repoStatus.Branch != incoming.Branch || updated.repoStatus.Head != incoming.Head {
		t.Fatalf("refresh status not applied: %#v", updated.repoStatus)
	}
	if !reflect.DeepEqual(updated.status, m.status) {
		t.Fatalf("branch input status changed: got %#v want %#v", updated.status, m.status)
	}
	if syncCalls != 1 {
		t.Fatalf("sync calls = %d, want 1", syncCalls)
	}
}

func testSectionCursors() map[graphSection]int {
	return map[graphSection]int{
		sectionGraph:   0,
		sectionCurrent: 0,
		sectionRemote:  0,
		sectionTags:    0,
	}
}
