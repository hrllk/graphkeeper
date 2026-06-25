package app

import (
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
		{name: "merge same commit", action: state.ActionMerge, target: "feature", rs: git.Status{Head: "abc123"}, wantAction: state.ActionMerge, wantMsg: "Already aligned.", wantDetail: "Target already matches HEAD.", wantCanExec: true},
		{name: "merge fast forward", action: state.ActionMerge, target: "feature", rs: git.Status{Head: "abc123"}, currentOnly: 0, targetOnly: 3, wantAction: state.ActionMerge, wantMsg: "Fast-forward available.", wantDetail: "HEAD can move to feature. Current: 0  Target: 3", wantCanExec: true},
		{name: "merge contains target", action: state.ActionMerge, target: "feature", rs: git.Status{Head: "abc123"}, currentOnly: 3, targetOnly: 0, wantAction: state.ActionMerge, wantMsg: "Target already included.", wantDetail: "Current branch already contains feature. Current: 3  Target: 0", wantCanExec: true},
		{name: "merge diverged", action: state.ActionMerge, target: "feature", rs: git.Status{Head: "abc123"}, currentOnly: 2, targetOnly: 4, wantAction: state.ActionMerge, wantMsg: "Fast-forward unavailable.", wantDetail: "HEAD abc123 and target feature have diverged. Current: 2  Target: 4", wantCanExec: true},
		{name: "rebase same commit", action: state.ActionRebase, target: "feature", rs: git.Status{Head: "abc123"}, wantAction: state.ActionRebase, wantMsg: "Already aligned.", wantDetail: "Target already matches HEAD.", wantCanExec: true},
		{name: "rebase base history", action: state.ActionRebase, target: "feature", rs: git.Status{Head: "abc123"}, currentOnly: 4, targetOnly: 0, wantAction: state.ActionRebase, wantMsg: "Target already in history.", wantDetail: "Current commits will replay onto feature. Current: 4  Target: 0", wantCanExec: true},
		{name: "rebase normal", action: state.ActionRebase, target: "feature", rs: git.Status{Head: "abc123"}, currentOnly: 4, targetOnly: 2, wantAction: state.ActionRebase, wantMsg: "Rebase onto target.", wantDetail: "Current: 4  Target: 2  |  target: feature", wantCanExec: true},
		{name: "reset", action: state.ActionReset, target: "feature", rs: git.Status{Head: "abcdef123456", Root: "/repo"}, currentOnly: 1, targetOnly: 2, wantAction: state.ActionReset, wantMsg: "Choose reset mode.", wantDetail: "Preview: HEAD abcdef123456 -> feature  |  Current: 1  Target: 2\nWorktree: clean\n\ns: soft  •  m: mixed  •  h: hard  •  enter: execute  •  esc: back", wantCanExec: true},
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
