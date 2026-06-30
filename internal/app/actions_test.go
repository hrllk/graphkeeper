package app

import (
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestDeriveStatusCases(t *testing.T) {
	tests := []struct {
		name    string
		rs      git.Status
		want    state.Mode
		wantBlk state.BlockReason
		wantMsg string
	}{
		{name: "no repo", rs: git.Status{}, want: state.ModeBlocked, wantBlk: state.BlockNoRepo, wantMsg: "Not inside a Git repository."},
		{name: "merge in progress", rs: git.Status{Root: "/repo", MergeInProgress: true}, want: state.ModeBrowse, wantMsg: "Merge/rebase in progress."},
		{name: "rebase in progress", rs: git.Status{Root: "/repo", RebaseInProgress: true}, want: state.ModeBrowse, wantMsg: "Merge/rebase in progress."},
		{name: "detached", rs: git.Status{Root: "/repo", Detached: true}, want: state.ModeBlocked, wantBlk: state.BlockDetached, wantMsg: "Detached HEAD."},
		{name: "empty repo", rs: git.Status{Root: "/repo", EmptyRepo: true}, want: state.ModeEmpty, wantMsg: "No commits yet."},
		{name: "no remote no upstream", rs: git.Status{Root: "/repo", NoRemote: true, NoUpstream: true}, want: state.ModeBlocked, wantBlk: state.BlockNoRemote, wantMsg: "No remote or upstream."},
		{name: "browse", rs: git.Status{Root: "/repo"}, want: state.ModeBrowse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveStatus(tt.rs)
			if got.Mode != tt.want {
				t.Fatalf("mode = %s, want %s", got.Mode, tt.want)
			}
			if got.Block != tt.wantBlk {
				t.Fatalf("block = %s, want %s", got.Block, tt.wantBlk)
			}
			if tt.wantMsg != "" && got.Message != tt.wantMsg {
				t.Fatalf("message = %q, want %q", got.Message, tt.wantMsg)
			}
		})
	}
}

func TestActionPullCases(t *testing.T) {
	tests := []struct {
		name    string
		rs      git.Status
		want    state.Mode
		wantBlk state.BlockReason
		wantMsg string
	}{
		{name: "no repo", rs: git.Status{}, want: state.ModeBlocked, wantBlk: state.BlockNoRepo},
		{name: "dirty worktree", rs: git.Status{Root: "/repo", WorktreeDirty: true}, want: state.ModeBlocked, wantBlk: state.BlockDirtyTree, wantMsg: "Working tree is dirty."},
		{name: "detached", rs: git.Status{Root: "/repo", Detached: true}, want: state.ModeBlocked, wantBlk: state.BlockDetached},
		{name: "merge in progress", rs: git.Status{Root: "/repo", MergeInProgress: true}, want: state.ModeBlocked, wantBlk: state.BlockUnknown},
		{name: "rebase in progress", rs: git.Status{Root: "/repo", RebaseInProgress: true}, want: state.ModeBlocked, wantBlk: state.BlockUnknown},
		{name: "no remote", rs: git.Status{Root: "/repo", NoRemote: true}, want: state.ModeBlocked, wantBlk: state.BlockNoRemote},
		{name: "no upstream", rs: git.Status{Root: "/repo", NoUpstream: true}, want: state.ModeBlocked, wantBlk: state.BlockNoUpstream},
		{name: "ready", rs: git.Status{Root: "/repo"}, want: state.ModeOutcomePreview, wantMsg: "Pull ready."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := actionPull(tt.rs)
			if got.Mode != tt.want {
				t.Fatalf("mode = %s, want %s", got.Mode, tt.want)
			}
			if got.Block != tt.wantBlk {
				t.Fatalf("block = %s, want %s", got.Block, tt.wantBlk)
			}
			if tt.wantMsg != "" && got.Message != tt.wantMsg {
				t.Fatalf("message = %q, want %q", got.Message, tt.wantMsg)
			}
		})
	}
}

func TestPullReadyCases(t *testing.T) {
	tests := []struct {
		name string
		rs   git.Status
		want bool
	}{
		{name: "ready", rs: git.Status{Root: "/repo"}, want: true},
		{name: "no repo", rs: git.Status{}, want: false},
		{name: "dirty worktree", rs: git.Status{Root: "/repo", WorktreeDirty: true}, want: false},
		{name: "detached", rs: git.Status{Root: "/repo", Detached: true}, want: false},
		{name: "no remote", rs: git.Status{Root: "/repo", NoRemote: true}, want: false},
		{name: "no upstream", rs: git.Status{Root: "/repo", NoUpstream: true}, want: false},
		{name: "merge in progress", rs: git.Status{Root: "/repo", MergeInProgress: true}, want: false},
		{name: "rebase in progress currently allowed", rs: git.Status{Root: "/repo", RebaseInProgress: true}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pullReady(tt.rs); got != tt.want {
				t.Fatalf("pullReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanCreateBranch(t *testing.T) {
	if !canCreateBranch(git.Status{}) {
		t.Fatal("expected clean worktree to allow branch creation")
	}
	if canCreateBranch(git.Status{WorktreeDirty: true}) {
		t.Fatal("expected dirty worktree to block branch creation")
	}
}

func TestActionPickTargets(t *testing.T) {
	t.Run("merge blocked when detached", func(t *testing.T) {
		got := actionPickTargets(git.Status{Detached: true}, state.ActionMerge)
		if got.Mode != state.ModeBlocked || got.Block != state.BlockDetached {
			t.Fatalf("got = %#v", got)
		}
	})
	t.Run("rebase blocked when detached", func(t *testing.T) {
		got := actionPickTargets(git.Status{Detached: true}, state.ActionRebase)
		if got.Mode != state.ModeBlocked || got.Block != state.BlockDetached {
			t.Fatalf("got = %#v", got)
		}
	})
	t.Run("reset allowed while detached", func(t *testing.T) {
		got := actionPickTargets(git.Status{Detached: true, Branches: []string{"main"}}, state.ActionReset)
		if got.Mode != state.ModeTargetPick {
			t.Fatalf("got = %#v", got)
		}
	})
	t.Run("reset excludes remote and tags", func(t *testing.T) {
		got := actionPickTargets(git.Status{
			Root:           "/repo",
			LocalBranches:  []string{"main"},
			RemoteBranches: []string{"origin/main"},
			Tags:           []string{"v1.0.0"},
		}, state.ActionReset)
		if got.Mode != state.ModeTargetPick {
			t.Fatalf("got = %#v", got)
		}
		if len(got.Targets) != 1 || got.Targets[0].Ref != "main" {
			t.Fatalf("expected reset targets to stay local only, got %#v", got.Targets)
		}
	})
	t.Run("empty targets blocked", func(t *testing.T) {
		got := actionPickTargets(git.Status{Root: "/repo"}, state.ActionMerge)
		if got.Mode != state.ModeBlocked || got.Block != state.BlockTargetEmpty {
			t.Fatalf("got = %#v", got)
		}
	})
	t.Run("selected target defaults to first", func(t *testing.T) {
		got := actionPickTargets(git.Status{
			Root:           "/repo",
			LocalBranches:  []string{"main"},
			RemoteBranches: []string{"origin/main"},
		}, state.ActionMerge)
		if got.Mode != state.ModeTargetPick {
			t.Fatalf("got = %#v", got)
		}
		if got.Selected != "main" {
			t.Fatalf("selected = %q, want main", got.Selected)
		}
	})
}

func TestCheckoutTargetFromFocus(t *testing.T) {
	tests := []struct {
		name string
		node graphNode
		want string
	}{
		{name: "head reference", node: graphNode{Decorations: []string{"HEAD -> main"}}, want: "main"},
		{name: "skip tag then remote", node: graphNode{Decorations: []string{"tag: v1.0.0", "origin/main"}}, want: "origin/main"},
		{name: "bare branch", node: graphNode{Decorations: []string{" feature "}}, want: "feature"},
		{name: "empty", node: graphNode{}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkoutTargetFromFocus(tt.node); got != tt.want {
				t.Fatalf("checkoutTargetFromFocus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSelectedTarget(t *testing.T) {
	tests := []struct {
		name string
		s    state.Status
		want string
	}{
		{name: "selected wins", s: state.Status{Selected: "feature", TargetIdx: 0, Targets: []state.TargetItem{{Ref: "main"}}}, want: "feature"},
		{name: "index fallback", s: state.Status{TargetIdx: 1, Targets: []state.TargetItem{{Ref: "main"}, {Ref: "feature"}}}, want: "feature"},
		{name: "out of range", s: state.Status{TargetIdx: 3, Targets: []state.TargetItem{{Ref: "main"}}}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectedTarget(tt.s); got != tt.want {
				t.Fatalf("selectedTarget() = %q, want %q", got, tt.want)
			}
		})
	}
}
