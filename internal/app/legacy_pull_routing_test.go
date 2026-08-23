package app

import (
	"errors"
	"reflect"
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func legacyPullTestStatus(head string) git.Status {
	return git.Status{
		Root:       "/repo",
		Branch:     "main",
		Head:       head,
		Upstream:   "origin/main",
		HasCommits: true,
		GraphCommits: []git.GraphCommit{
			{Hash: head, Subject: "current"},
			{Hash: "base", Subject: "base"},
		},
	}
}

func legacyPullTestModel(sink *recordingEventSink) model {
	tags := []git.TagEntry{{Name: "v1.0.0", CommitHash: "base", Subject: "release"}}
	return model{
		status:          loadingToast("legacy operation"),
		repositoryState: repositoryState{repoStatus: legacyPullTestStatus("old-head"), tagEntries: tags},
		eventSink:       sink,
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
	}
}

func TestTopLevelUpdateRoutesPullCheckedThroughLegacyFetchHandler(t *testing.T) {
	t.Run("success preserves pull status and applies legacy repository side effects", func(t *testing.T) {
		sink := &recordingEventSink{}
		m := legacyPullTestModel(sink)
		incoming := legacyPullTestStatus("new-head")
		incoming.TagProvenanceLoaded = true
		incoming.TagEntries = []git.TagEntry{{Name: "v2.0.0", CommitHash: "new-head", Subject: "new release"}}
		incoming.TagEntriesLoaded = true
		pullStatus := state.New().WithBlocked(state.BlockNoUpstream, "No upstream.", "Set an upstream first.")

		gotModel, cmd := m.Update(pullCheckedMsg{repo: incoming, status: pullStatus})
		got := gotModel.(model)

		if cmd != nil {
			t.Fatalf("expected nil command, got %v", cmd)
		}
		if !reflect.DeepEqual(got.status, pullStatus) {
			t.Fatalf("status = %#v, want %#v", got.status, pullStatus)
		}
		if got.repoStatus.Head != "new-head" || !got.repoStatus.TagProvenanceLoaded {
			t.Fatalf("repoStatus did not preserve fetched status: %#v", got.repoStatus)
		}
		if !reflect.DeepEqual(got.tagEntries, incoming.TagEntries) {
			t.Fatalf("tag cache = %#v, want %#v", got.tagEntries, incoming.TagEntries)
		}
		if got.sectionCursor[sectionGraph] != 0 || got.sectionCursor[sectionCurrent] != 0 || got.sectionCursor[sectionRemote] != -1 || got.sectionCursor[sectionTags] != 0 {
			t.Fatalf("browse cursors were not synchronized: %#v", got.sectionCursor)
		}
		assertLegacyEvent(t, sink, "pull_check", map[string]string{"upstream": "origin/main", "blocked": string(pullStatus.Block)})
	})

	t.Run("error keeps repository state and reports legacy failure", func(t *testing.T) {
		sink := &recordingEventSink{}
		m := legacyPullTestModel(sink)
		oldRepo := m.repoStatus
		oldTags := append([]git.TagEntry(nil), m.tagEntries...)
		err := errors.New("remote unavailable")

		gotModel, cmd := m.Update(pullCheckedMsg{repo: legacyPullTestStatus("ignored"), status: state.New().WithBrowse(), err: err})
		got := gotModel.(model)

		if cmd != nil {
			t.Fatalf("expected nil command, got %v", cmd)
		}
		wantStatus := state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed.", err.Error())
		if !reflect.DeepEqual(got.status, wantStatus) {
			t.Fatalf("status = %#v, want %#v", got.status, wantStatus)
		}
		if !reflect.DeepEqual(got.repoStatus, oldRepo) || !reflect.DeepEqual(got.tagEntries, oldTags) {
			t.Fatalf("error path changed repository side effects: repo=%#v tags=%#v", got.repoStatus, got.tagEntries)
		}
		assertLegacyEvent(t, sink, "pull_check_failed", map[string]string{"error": err.Error()})
	})
}

func TestTopLevelUpdateRoutesLegacyExecutedPullActionsWithoutNeutralReduction(t *testing.T) {
	actions := []state.Action{state.ActionPull, state.ActionPullMerge, state.ActionPullRebase}
	for _, action := range actions {
		action := action
		t.Run(string(action)+" success", func(t *testing.T) {
			sink := &recordingEventSink{}
			m := legacyPullTestModel(sink)
			msg := executedMsg{action: action, target: "feature", status: legacyPullTestStatus("success-head")}

			gotModel, cmd := m.Update(msg)
			got := gotModel.(model)
			wantRepo := git.Status{
				Root:       "/repo",
				Branch:     "main",
				Head:       "success-head",
				Upstream:   "origin/main",
				HasCommits: true,
				GraphCommits: []git.GraphCommit{
					{Hash: "success-head", Subject: "current"},
					{Hash: "base", Subject: "base", Tags: []string{"v1.0.0"}},
				},
				Tags:                []string{"v1.0.0"},
				TagEntries:          []git.TagEntry{{Name: "v1.0.0", CommitHash: "base", Subject: "release"}},
				TagProvenanceLoaded: false,
				TagSyncSummary:      "",
			}
			wantMessage := "Choose an action."
			if action == state.ActionPullMerge || action == state.ActionPullRebase {
				wantMessage = "Pull complete."
			}
			wantStatus := state.Status{
				Mode:          state.ModeBrowse,
				Action:        state.ActionNone,
				Block:         state.BlockNone,
				WorktreeState: state.WorktreeStateClean,
				Title:         "Browse",
				Message:       wantMessage,
				TargetIdx:     -1,
				CanExecute:    false,
			}

			if cmd != nil {
				t.Fatalf("expected nil command, got %v", cmd)
			}
			if !reflect.DeepEqual(got.status, wantStatus) {
				t.Fatalf("status = %#v, want %#v", got.status, wantStatus)
			}
			if !reflect.DeepEqual(got.repoStatus, wantRepo) {
				t.Fatalf("repoStatus = %#v, want %#v", got.repoStatus, wantRepo)
			}
			if got.sectionCursor[sectionGraph] != 0 || got.sectionCursor[sectionCurrent] != 0 || got.sectionCursor[sectionRemote] != -1 || got.sectionCursor[sectionTags] != 0 {
				t.Fatalf("browse cursors were not synchronized: %#v", got.sectionCursor)
			}
			assertLegacyEvent(t, sink, "execute_action", map[string]string{"action": string(action), "head": "success-head"})
		})

		t.Run(string(action)+" ordinary failure", func(t *testing.T) {
			sink := &recordingEventSink{}
			m := legacyPullTestModel(sink)
			err := errors.New("operation failed")

			gotModel, cmd := m.Update(executedMsg{action: action, target: "feature", status: legacyPullTestStatus("ignored"), err: err})
			got := gotModel.(model)
			wantStatus := state.New().WithBlocked(state.BlockUnknown, "Action failed.", err.Error())
			if cmd != nil {
				t.Fatalf("expected nil command, got %v", cmd)
			}
			if !reflect.DeepEqual(got.status, wantStatus) {
				t.Fatalf("status = %#v, want %#v", got.status, wantStatus)
			}
			if !reflect.DeepEqual(got.repoStatus, m.repoStatus) || !reflect.DeepEqual(got.tagEntries, m.tagEntries) {
				t.Fatalf("ordinary failure changed repository side effects: repo=%#v tags=%#v", got.repoStatus, got.tagEntries)
			}
			assertLegacyEvent(t, sink, "execute_failed", map[string]string{"action": string(action), "target": "feature", "error": err.Error()})
		})

		t.Run(string(action)+" conflict", func(t *testing.T) {
			sink := &recordingEventSink{}
			m := legacyPullTestModel(sink)
			result := legacyPullTestStatus("conflict-head")
			result.MergeInProgress = action == state.ActionPull || action == state.ActionPullMerge
			result.RebaseInProgress = action == state.ActionPullRebase

			gotModel, cmd := m.Update(executedMsg{action: action, target: "feature", status: result, err: errors.New("conflict")})
			got := gotModel.(model)
			wantRepo := git.Status{
				Root:             "/repo",
				Branch:           "main",
				Head:             "conflict-head",
				Upstream:         "origin/main",
				HasCommits:       true,
				GraphCommits:     []git.GraphCommit{{Hash: "conflict-head", Subject: "current"}, {Hash: "base", Subject: "base", Tags: []string{"v1.0.0"}}},
				Tags:             []string{"v1.0.0"},
				TagEntries:       []git.TagEntry{{Name: "v1.0.0", CommitHash: "base", Subject: "release"}},
				MergeInProgress:  action == state.ActionPull || action == state.ActionPullMerge,
				RebaseInProgress: action == state.ActionPullRebase,
			}
			wantStatus := state.Status{
				Mode:       state.ModeBrowse,
				Action:     state.ActionNone,
				Block:      state.BlockNone,
				Title:      "Browse",
				Message:    "Pull conflicted.",
				Detail:     "Press Enter to abort.",
				TargetIdx:  -1,
				CanExecute: false,
			}
			if cmd != nil {
				t.Fatalf("expected nil command, got %v", cmd)
			}
			if !reflect.DeepEqual(got.status, wantStatus) {
				t.Fatalf("status = %#v, want %#v", got.status, wantStatus)
			}
			if !reflect.DeepEqual(got.repoStatus, wantRepo) {
				t.Fatalf("repoStatus = %#v, want %#v", got.repoStatus, wantRepo)
			}
			assertLegacyEvent(t, sink, "execute_conflicted", map[string]string{"action": string(action), "head": "conflict-head"})
		})
	}
}

func assertLegacyEvent(t *testing.T, sink *recordingEventSink, name string, fields map[string]string) {
	t.Helper()
	if len(sink.events) != 1 {
		t.Fatalf("telemetry events = %#v, want exactly one", sink.events)
	}
	event := sink.events[0]
	if event.Source != "app" || event.Name != name || !reflect.DeepEqual(event.Fields, fields) {
		t.Fatalf("event = %#v, want app/%s %#v", event, name, fields)
	}
}
