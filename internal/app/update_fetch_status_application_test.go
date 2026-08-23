package app

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestFetchPrepareAndPullCheckSuccessApplyRepositoryStatusOnce(t *testing.T) {
	cached := []git.TagEntry{{Name: "cached", CommitHash: "cached-head"}}
	loaded := []git.TagEntry{{Name: "loaded", CommitHash: "loaded-head"}}
	pullStatus := state.New().WithBlocked(state.BlockNoUpstream, "No upstream.", "Set an upstream first.")

	tests := []struct {
		name       string
		msg        interface{}
		wantEvent  string
		wantFields map[string]string
		wantStatus state.Status
		wantRepo   git.Status
	}{
		{
			name:       "fetch uses cached tags and derives status",
			msg:        fetchedMsg{status: git.Status{Root: "/repo", Branch: "main", Head: "fetch-head"}},
			wantEvent:  "fetch_repo",
			wantFields: map[string]string{"branch": "main", "head": "fetch-head"},
			wantStatus: state.Status{Mode: state.ModeBrowse, Action: state.ActionNone, Block: state.BlockNone, WorktreeState: state.WorktreeStateClean, Title: "Browse", Message: "Choose an action.", TargetIdx: -1},
			wantRepo:   git.Status{Root: "/repo", Branch: "main", Head: "fetch-head", Tags: []string{"cached"}, TagEntries: cached, TagProvenanceLoaded: true, TagSyncSummary: "cached"},
		},
		{
			name:       "prepare preserves loaded tags and action status",
			msg:        preparedMsg{status: git.Status{Root: "/repo", Branch: "main", Head: "prepare-head", TagEntriesLoaded: true, TagProvenanceLoaded: true, TagSyncSummary: "loaded", TagEntries: loaded}, action: state.ActionNone},
			wantEvent:  "prepare_action",
			wantFields: map[string]string{"action": string(state.ActionNone), "branch": "main"},
			wantStatus: state.Status{Mode: state.ModeBrowse, Action: state.ActionNone, Block: state.BlockNone, WorktreeState: state.WorktreeStateClean, Title: "Browse", Message: "Choose an action.", TargetIdx: -1},
			wantRepo:   git.Status{Root: "/repo", Branch: "main", Head: "prepare-head", TagEntriesLoaded: true, TagProvenanceLoaded: true, TagSyncSummary: "loaded", TagEntries: loaded},
		},
		{
			name:       "pull check preserves pull status and loaded tags",
			msg:        pullCheckedMsg{repo: git.Status{Root: "/repo", Branch: "main", Head: "pull-head", Upstream: "origin/main", TagEntriesLoaded: true, TagProvenanceLoaded: true, TagEntries: loaded}, status: pullStatus},
			wantEvent:  "pull_check",
			wantFields: map[string]string{"upstream": "origin/main", "blocked": string(pullStatus.Block)},
			wantStatus: pullStatus,
			wantRepo:   git.Status{Root: "/repo", Branch: "main", Head: "pull-head", Upstream: "origin/main", TagEntriesLoaded: true, TagProvenanceLoaded: true, TagEntries: loaded},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := &recordingEventSink{}
			m := model{
				repositoryState: repositoryState{repoStatus: git.Status{TagProvenanceLoaded: true, TagSyncSummary: "cached"}, tagEntries: cached},
				pullState:       pullState{}, status: state.New().WithLoading("working"), eventSink: sink}
			var synced []git.Status
			original := applyRepositoryStatusSyncBrowseState
			applyRepositoryStatusSyncBrowseState = func(_ *model, status git.Status) { synced = append(synced, status) }
			defer func() { applyRepositoryStatusSyncBrowseState = original }()

			gotModel, cmd := m.Update(tt.msg)
			got := gotModel.(model)
			if cmd != nil {
				t.Fatalf("command = %v, want nil", cmd)
			}
			if len(synced) != 1 {
				t.Fatalf("repository browse sync calls = %d, want 1", len(synced))
			}
			if !reflect.DeepEqual(got.repoStatus, tt.wantRepo) {
				t.Fatalf("repoStatus = %#v, want %#v", got.repoStatus, tt.wantRepo)
			}
			if !reflect.DeepEqual(synced[0], tt.wantRepo) {
				t.Fatalf("synced status = %#v, want %#v", synced[0], tt.wantRepo)
			}
			if !reflect.DeepEqual(got.status, tt.wantStatus) {
				t.Fatalf("status = %#v, want %#v", got.status, tt.wantStatus)
			}
			if !reflect.DeepEqual(got.tagEntries, func() []git.TagEntry {
				if len(tt.wantRepo.TagEntries) > 0 {
					return tt.wantRepo.TagEntries
				}
				return cached
			}()) {
				t.Fatalf("tag cache = %#v", got.tagEntries)
			}
			if got.tagSyncAttempted != tt.wantRepo.TagProvenanceLoaded {
				t.Fatalf("tagSyncAttempted = %v, want %v", got.tagSyncAttempted, tt.wantRepo.TagProvenanceLoaded)
			}
			assertLegacyEvent(t, sink, tt.wantEvent, tt.wantFields)
		})
	}
}

func TestFetchPrepareAndPullCheckErrorsDoNotApplyRepositoryStatus(t *testing.T) {
	err := errors.New("remote unavailable")
	tests := []struct {
		name   string
		msg    interface{}
		event  string
		fields map[string]string
	}{
		{"fetch", fetchedMsg{status: git.Status{Head: "ignored"}, err: err}, "", nil},
		{"prepare", preparedMsg{status: git.Status{Head: "ignored"}, action: state.ActionMerge, err: err}, "prepare_failed", map[string]string{"action": string(state.ActionMerge), "error": err.Error()}},
		{"pull check", pullCheckedMsg{repo: git.Status{Head: "ignored"}, status: state.New().WithBrowse(), err: err}, "pull_check_failed", map[string]string{"error": err.Error()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := &recordingEventSink{}
			oldRepo := git.Status{Head: "old", TagProvenanceLoaded: true}
			oldTags := []git.TagEntry{{Name: "old"}}
			m := model{
				repositoryState: repositoryState{repoStatus: oldRepo, tagEntries: oldTags},
				pullState:       pullState{}, status: state.New().WithLoading("working"), eventSink: sink}
			calls := 0
			original := applyRepositoryStatusSyncBrowseState
			applyRepositoryStatusSyncBrowseState = func(*model, git.Status) { calls++ }
			defer func() { applyRepositoryStatusSyncBrowseState = original }()
			gotModel, cmd := m.Update(tt.msg)
			got := gotModel.(model)
			if cmd != nil {
				t.Fatalf("command = %v, want nil", cmd)
			}
			if calls != 0 {
				t.Fatalf("repository browse sync calls = %d, want 0", calls)
			}
			if !reflect.DeepEqual(got.repoStatus, oldRepo) || !reflect.DeepEqual(got.tagEntries, oldTags) {
				t.Fatalf("error mutated repository state: repo=%#v tags=%#v", got.repoStatus, got.tagEntries)
			}
			if tt.event == "" {
				if len(sink.events) != 0 {
					t.Fatalf("telemetry = %#v, want none", sink.events)
				}
			} else {
				assertLegacyEvent(t, sink, tt.event, tt.fields)
			}
		})
	}
}

func TestPreviewGraphActionCheckAndPushFetchedSuccessApplyRepositoryStatusOnce(t *testing.T) {
	cases := []struct {
		name       string
		msg        interface{}
		wantEvent  string
		wantFields map[string]string
		wantMode   state.Mode
		wantAction state.Action
		wantCmd    bool
	}{
		{"preview", previewMsg{action: state.ActionMerge, target: "target", repo: git.Status{Root: "/repo", Branch: "main", Head: "preview"}, status: state.New().WithConfirm(state.ActionMerge, "Merge?", "Review merge.")}, "preview_action", map[string]string{"action": "merge", "target": "target", "mode": "confirm"}, state.ModeConfirm, state.ActionMerge, false},
		{"graph check", graphActionCheckMsg{action: state.ActionRebase, target: "target", base: "base", repo: git.Status{Root: "/repo", Branch: "main", Head: "graph"}, currentOnly: 2, targetOnly: 1}, "graph_action_check", map[string]string{"action": "rebase", "target": "target", "base": "base", "currentOnly": "2", "targetOnly": "1"}, state.ModeReview, state.ActionRebase, false},
		{"push tracking", pushFetchedMsg{status: git.Status{Root: "/repo", Branch: "main", Head: "push", NoUpstream: true}}, "", nil, state.ModeConfirm, state.ActionSetUpstream, false},
		{"push", pushFetchedMsg{status: git.Status{Root: "/repo", Branch: "main", Head: "push", Upstream: "origin/main"}}, "", nil, state.ModeLoading, state.ActionPush, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			sink := &recordingEventSink{}
			m := model{
				repositoryState: repositoryState{tagEntries: []git.TagEntry{{Name: "cached"}}},
				pullState:       pullState{}, status: state.New().WithLoading("working"), eventSink: sink}
			calls := 0
			old := applyRepositoryStatusSyncBrowseState
			applyRepositoryStatusSyncBrowseState = func(*model, git.Status) { calls++ }
			defer func() { applyRepositoryStatusSyncBrowseState = old }()
			gotModel, cmd := m.Update(tt.msg)
			got := gotModel.(model)
			if calls != 1 || (cmd != nil) != tt.wantCmd || got.status.Mode != tt.wantMode || got.status.Action != tt.wantAction {
				t.Fatalf("calls=%d cmd=%v status=%#v", calls, cmd, got.status)
			}
			if tt.wantEvent != "" {
				assertLegacyEvent(t, sink, tt.wantEvent, tt.wantFields)
			}
		})
	}
}

func TestPreviewGraphActionCheckAndPushFetchedErrorsDoNotApplyRepositoryStatus(t *testing.T) {
	err := errors.New("failed")
	cases := []interface{}{previewMsg{repo: git.Status{Head: "ignored"}, err: err}, graphActionCheckMsg{repo: git.Status{Head: "ignored"}, err: err}, pushFetchedMsg{status: git.Status{Head: "ignored"}, err: err}}
	for i, msg := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			old := git.Status{Head: "old"}
			m := model{
				repositoryState: repositoryState{repoStatus: old},
				pullState:       pullState{}}
			calls := 0
			before := applyRepositoryStatusSyncBrowseState
			applyRepositoryStatusSyncBrowseState = func(*model, git.Status) { calls++ }
			defer func() { applyRepositoryStatusSyncBrowseState = before }()
			gotModel, cmd := m.Update(msg)
			got := gotModel.(model)
			if cmd != nil || calls != 0 || !reflect.DeepEqual(got.repoStatus, old) {
				t.Fatalf("error applied state: cmd=%v calls=%d repo=%#v", cmd, calls, got.repoStatus)
			}
		})
	}
}

func TestPreviewGraphActionCheckAndPushFetchedLoadedTagsApplyOnce(t *testing.T) {
	loadedTags := []git.TagEntry{{Name: "loaded", CommitHash: "loaded-head"}}
	cases := []struct {
		name string
		msg  interface{}
	}{
		{"preview", previewMsg{action: state.ActionMerge, target: "target", repo: git.Status{Root: "/repo", Branch: "main", Head: "preview", TagEntriesLoaded: true, TagProvenanceLoaded: true, TagSyncSummary: "loaded", TagEntries: loadedTags}, status: state.New().WithConfirm(state.ActionMerge, "Merge?", "Review merge.")}},
		{"graph action check", graphActionCheckMsg{action: state.ActionRebase, target: "target", base: "base", repo: git.Status{Root: "/repo", Branch: "main", Head: "graph", TagEntriesLoaded: true, TagProvenanceLoaded: true, TagSyncSummary: "loaded", TagEntries: loadedTags}, currentOnly: 1, targetOnly: 1}},
		{"push fetched", pushFetchedMsg{status: git.Status{Root: "/repo", Branch: "main", Head: "push", NoUpstream: true, TagEntriesLoaded: true, TagProvenanceLoaded: true, TagSyncSummary: "loaded", TagEntries: loadedTags}}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			m := model{status: state.New().WithLoading("working")}
			calls := 0
			before := applyRepositoryStatusSyncBrowseState
			applyRepositoryStatusSyncBrowseState = func(*model, git.Status) { calls++ }
			defer func() { applyRepositoryStatusSyncBrowseState = before }()
			gotModel, _ := m.Update(tt.msg)
			got := gotModel.(model)
			if calls != 1 || !reflect.DeepEqual(got.repoStatus.TagEntries, loadedTags) || !got.tagSyncAttempted {
				t.Fatalf("calls=%d repo=%#v tagSyncAttempted=%v", calls, got.repoStatus, got.tagSyncAttempted)
			}
		})
	}
}
