package app

import (
	"errors"
	"reflect"
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestTopLevelUpdateCheckoutSuccessAppliesRepositoryStatusOnce(t *testing.T) {
	cached := []git.TagEntry{{Name: "v1.0.0", CommitHash: "base", Subject: "release"}}
	status := git.Status{
		Root:       "/repo",
		Branch:     "feature",
		Head:       "head",
		HasCommits: true,
		GraphCommits: []git.GraphCommit{
			{Hash: "base", Subject: "base"},
			{Hash: "head", Parents: []string{"base"}, Subject: "current"},
		},
		TagEntriesLoaded:    true,
		TagProvenanceLoaded: true,
		TagSyncSummary:      "loaded",
		TagEntries:          cached,
	}
	wantRepo := git.Status{
		Root:       "/repo",
		Branch:     "feature",
		Head:       "head",
		HasCommits: true,
		GraphCommits: []git.GraphCommit{
			{Hash: "base", Subject: "base", Tags: []string{"v1.0.0"}},
			{Hash: "head", Parents: []string{"base"}, Subject: "current"},
		},
		Tags:                []string(nil),
		TagEntriesLoaded:    true,
		TagProvenanceLoaded: true,
		TagSyncSummary:      "loaded",
		TagEntries:          cached,
	}
	wantStatus := state.Status{
		Mode:          state.ModeBrowse,
		Action:        state.ActionNone,
		Block:         state.BlockNone,
		WorktreeState: state.WorktreeStateClean,
		Title:         "Browse",
		Message:       "Choose an action.",
		TargetIdx:     -1,
		CanExecute:    false,
	}

	sink := &recordingEventSink{}
	m := model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
			sectionCursor: map[graphSection]int{
				sectionGraph: 3, sectionCurrent: 0, sectionRemote: 0, sectionTags: 0,
			},
			graphScroll:     8,
			graphLaneCursor: 4,
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{Head: "old"},
			tagEntries: cached,
		},
		pullState: pullState{},
		status:    state.New().WithLoading("checkout"), commitLimit: 25, eventSink: sink}
	var synced []git.Status
	originalSync := applyRepositoryStatusSyncBrowseState
	applyRepositoryStatusSyncBrowseState = func(_ *model, got git.Status) { synced = append(synced, got) }
	defer func() { applyRepositoryStatusSyncBrowseState = originalSync }()

	gotModel, cmd := m.Update(executedMsg{action: state.ActionCheckout, target: "feature", status: status})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("command = %v, want nil", cmd)
	}
	if len(synced) != 1 {
		t.Fatalf("repository browse sync calls = %d, want 1", len(synced))
	}
	if !reflect.DeepEqual(got.repoStatus, wantRepo) {
		t.Fatalf("repoStatus = %#v, want %#v", got.repoStatus, wantRepo)
	}
	if !reflect.DeepEqual(got.tagEntries, cached) || !got.tagSyncAttempted {
		t.Fatalf("tag projection = %#v, attempted=%v", got.tagEntries, got.tagSyncAttempted)
	}
	if !reflect.DeepEqual(synced[0], wantRepo) {
		t.Fatalf("synced status = %#v, want %#v", synced[0], wantRepo)
	}
	if !reflect.DeepEqual(got.status, wantStatus) {
		t.Fatalf("status = %#v, want %#v", got.status, wantStatus)
	}
	if got.commitLimit != 0 || got.activeSection != sectionGraph {
		t.Fatalf("checkout graph projection = section %v, limit %d", got.activeSection, got.commitLimit)
	}
	if got.sectionCursor[sectionGraph] != 1 || got.graphScroll != 1 {
		t.Fatalf("checkout graph cursor = %d, scroll = %d", got.sectionCursor[sectionGraph], got.graphScroll)
	}
	assertLegacyEvent(t, sink, "execute_action", map[string]string{"action": "checkout", "target": "feature", "head": "head"})
}

func TestTopLevelUpdateCheckoutErrorDoesNotApplyRepositoryStatus(t *testing.T) {
	err := errors.New("checkout failed")
	oldRepo := git.Status{Root: "/repo", Branch: "main", Head: "old"}
	oldTags := []git.TagEntry{{Name: "old", CommitHash: "old"}}
	sink := &recordingEventSink{}
	m := model{
		navigationState: navigationState{
			activeSection:   sectionCurrent,
			sectionCursor:   map[graphSection]int{sectionGraph: 2, sectionCurrent: 0},
			graphScroll:     5,
			graphLaneCursor: 3,
		},
		repositoryState: repositoryState{
			repoStatus: oldRepo,
			tagEntries: oldTags,
		},
		pullState: pullState{},
		status:    state.New().WithLoading("checkout"), commitLimit: 25, eventSink: sink}
	calls := 0
	originalSync := applyRepositoryStatusSyncBrowseState
	applyRepositoryStatusSyncBrowseState = func(*model, git.Status) { calls++ }
	defer func() { applyRepositoryStatusSyncBrowseState = originalSync }()

	gotModel, cmd := m.Update(executedMsg{action: state.ActionCheckout, target: "feature", status: git.Status{Root: "/repo", Head: "ignored"}, err: err})
	got := gotModel.(model)
	wantStatus := state.New().WithBlocked(state.BlockUnknown, "Checkout failed.", err.Error())
	if cmd != nil || calls != 0 {
		t.Fatalf("error command/sync = %v/%d, want nil/0", cmd, calls)
	}
	if !reflect.DeepEqual(got.repoStatus, oldRepo) || !reflect.DeepEqual(got.tagEntries, oldTags) {
		t.Fatalf("error mutated repository state: repo=%#v tags=%#v", got.repoStatus, got.tagEntries)
	}
	if got.activeSection != sectionCurrent || got.sectionCursor[sectionGraph] != 2 || got.graphScroll != 5 || got.graphLaneCursor != 3 || got.commitLimit != 25 {
		t.Fatalf("error mutated graph state: section=%v cursors=%#v scroll=%d lane=%d limit=%d", got.activeSection, got.sectionCursor, got.graphScroll, got.graphLaneCursor, got.commitLimit)
	}
	if !reflect.DeepEqual(got.status, wantStatus) {
		t.Fatalf("status = %#v, want %#v", got.status, wantStatus)
	}
	assertLegacyEvent(t, sink, "execute_failed", map[string]string{"action": "checkout", "target": "feature", "error": err.Error()})
}
