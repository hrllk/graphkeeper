package app

import (
	"strings"
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestBuildActionPreview(t *testing.T) {
	tests := []struct {
		name        string
		action      state.Action
		target      string
		rs          git.Status
		currentOnly int
		targetOnly  int
		wantAction  state.Action
		wantMsg     string
		wantDetail  string
		wantCanExec bool
	}{
		{name: "merge same commit", action: state.ActionMerge, target: "feature", rs: git.Status{Head: "abc123"}, wantAction: state.ActionMerge, wantMsg: "Nothing to merge.", wantDetail: "Target is the commit HEAD already points at.", wantCanExec: false},
		{name: "merge fast forward", action: state.ActionMerge, target: "feature", rs: git.Status{Head: "abc123"}, currentOnly: 0, targetOnly: 3, wantAction: state.ActionMerge, wantMsg: "Fast-forward available.", wantDetail: "HEAD can move straight to feature; no merge commit is created. Current: 0  Target: 3", wantCanExec: true},
		{name: "merge contains target", action: state.ActionMerge, target: "feature", rs: git.Status{Head: "abc123"}, currentOnly: 3, targetOnly: 0, wantAction: state.ActionMerge, wantMsg: "Nothing to merge.", wantDetail: "This branch already contains feature. Current: 3  Target: 0", wantCanExec: false},
		{name: "merge diverged", action: state.ActionMerge, target: "feature", rs: git.Status{Head: "abc123"}, currentOnly: 2, targetOnly: 4, wantAction: state.ActionMerge, wantMsg: "Merge target into HEAD.", wantDetail: "HEAD abc123 and target feature have diverged, so a merge commit is created. Current: 2  Target: 4", wantCanExec: true},
		{name: "rebase same commit", action: state.ActionRebase, target: "feature", rs: git.Status{Head: "abc123"}, wantAction: state.ActionRebase, wantMsg: "Nothing to rebase.", wantDetail: "Target is the commit HEAD already points at.", wantCanExec: false},
		{name: "rebase base history", action: state.ActionRebase, target: "feature", rs: git.Status{Head: "abc123"}, currentOnly: 4, targetOnly: 0, wantAction: state.ActionRebase, wantMsg: "Target is already in this history.", wantDetail: "Replaying onto feature would rewrite commits this branch already contains. Current: 4  Target: 0", wantCanExec: false},
		{name: "rebase fast forward", action: state.ActionRebase, target: "feature", rs: git.Status{Head: "abc123"}, currentOnly: 0, targetOnly: 3, wantAction: state.ActionRebase, wantMsg: "Fast-forward available.", wantDetail: "This branch has no commits of its own to replay, so HEAD moves straight to feature. Current: 0  Target: 3", wantCanExec: true},
		{name: "rebase normal", action: state.ActionRebase, target: "feature", rs: git.Status{Head: "abc123"}, currentOnly: 4, targetOnly: 2, wantAction: state.ActionRebase, wantMsg: "Rebase onto target.", wantDetail: "Current: 4  Target: 2  |  target: feature", wantCanExec: true},
		{name: "reset", action: state.ActionReset, target: "feature", rs: git.Status{Head: "abcdef123456", Root: "/repo"}, currentOnly: 1, targetOnly: 2, wantAction: state.ActionReset, wantMsg: "Choose a reset mode.", wantDetail: "", wantCanExec: true},
		{name: "unknown", action: state.ActionNone, target: "feature", rs: git.Status{Head: "abc123"}, wantAction: state.ActionNone, wantMsg: "No action selected.", wantDetail: "feature", wantCanExec: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildActionPreview(tt.action, tt.target, tt.rs, tt.currentOnly, tt.targetOnly)
			if got.Action != tt.wantAction {
				t.Fatalf("action = %s, want %s", got.Action, tt.wantAction)
			}
			if got.Message != tt.wantMsg {
				t.Fatalf("message = %q, want %q", got.Message, tt.wantMsg)
			}
			if got.Detail != tt.wantDetail {
				t.Fatalf("detail = %q, want %q", got.Detail, tt.wantDetail)
			}
			if got.CanExecute != tt.wantCanExec {
				t.Fatalf("canExecute = %v, want %v", got.CanExecute, tt.wantCanExec)
			}
		})
	}
}

// The diverged case is the one merge exists for, per
// docs/20260701-0005-feature-graph-section-merge-rebase-gate-note.md, so its
// headline must name the action. It used to read "Fast-forward unavailable.",
// which describes the thing that will not happen and left the most valid case
// sounding like a refusal, while the rebase branch already said "Rebase onto
// target."
func TestMergePreviewNamesTheActionWhenHistoriesDiverged(t *testing.T) {
	got := buildActionPreview(state.ActionMerge, "feature", git.Status{Head: "abc123"}, 2, 4)
	if strings.Contains(got.Message, "unavailable") || strings.Contains(got.Message, "cannot") {
		t.Fatalf("diverged merge headline describes a refusal: %q", got.Message)
	}
	if !strings.Contains(got.Message, "Merge") {
		t.Fatalf("diverged merge headline does not name the action: %q", got.Message)
	}
	if !got.CanExecute {
		t.Fatalf("diverged merge should be executable: %#v", got)
	}
}

// The gate tests whether a local branch points at the selected row, not whether
// the user is on a local branch. Standing on main and selecting one of main's own
// earlier commits hits it, so the copy has to say which one it means.
func TestMergeRebaseGateCopyDescribesWhatItTests(t *testing.T) {
	for _, action := range []string{"Merge", "Rebase"} {
		blocked := state.New().WithBlocked(state.BlockNotLocalPointer, action+" unavailable.", "Select a commit a local branch points at.")
		if blocked.Block != state.BlockNotLocalPointer {
			t.Fatalf("%s gate lost its typed reason: %q", action, blocked.Block)
		}
		if strings.Contains(blocked.Detail, "Select a local branch.") {
			t.Fatalf("%s gate still tells the user to select a local branch: %q", action, blocked.Detail)
		}
	}
}

// The gate note's four states, minus one departure from its conclusion. Two of
// them are not an action at all: the target is the commit HEAD is on, or the
// branch already contains it, so offering an execute key invites a no-op or a
// rewrite of history the branch already has.
//
// The note also wanted the fast-forward state off, routed to pull. It stays on:
// pull needs an upstream, so this is the only way to fast-forward a local branch
// onto another local branch's tip. Removing it would remove the capability rather
// than relocate it.
//
// update_fetch.go's graph shortcut classifies the same four states. If these two
// ever disagree, m from the graph and m from the target picker do different
// things on the same target.
func TestPreviewOffersActionOnlyWhereThereIsOne(t *testing.T) {
	for _, action := range []state.Action{state.ActionMerge, state.ActionRebase} {
		for _, tt := range []struct {
			name                    string
			currentOnly, targetOnly int
			wantExec                bool
		}{
			{"same commit", 0, 0, false},
			{"fast-forward", 0, 3, true},
			{"already contained", 3, 0, false},
			{"diverged", 3, 4, true},
		} {
			got := buildActionPreview(action, "feature", git.Status{Head: "abc123"}, tt.currentOnly, tt.targetOnly)
			if got.CanExecute != tt.wantExec {
				t.Fatalf("%s %s: CanExecute = %v, want %v (%q / %q)", action, tt.name, got.CanExecute, tt.wantExec, got.Message, got.Detail)
			}
		}
	}
}

// A preview the user cannot act on still has to say what the repository is doing,
// or the screen is a dead end.
func TestNonExecutablePreviewsExplainThemselves(t *testing.T) {
	for _, action := range []state.Action{state.ActionMerge, state.ActionRebase} {
		for _, counts := range [][2]int{{0, 0}, {3, 0}} {
			got := buildActionPreview(action, "feature", git.Status{Head: "abc123"}, counts[0], counts[1])
			if got.Message == "" || got.Detail == "" {
				t.Fatalf("%s %v: preview without copy: %#v", action, counts, got)
			}
			if strings.Contains(got.Message, "unavailable") {
				t.Fatalf("%s %v: headline reads as a refusal: %q", action, counts, got.Message)
			}
		}
	}
}
