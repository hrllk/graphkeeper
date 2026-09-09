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
		{name: "cherry-pick in progress", rs: git.Status{Root: "/repo", CherryPickInProgress: true}, want: state.ModeBrowse, wantMsg: "Cherry-pick in progress."},
		{name: "detached", rs: git.Status{Root: "/repo", Detached: true}, want: state.ModeBlocked, wantBlk: state.BlockDetached, wantMsg: "Detached HEAD."},
		{name: "empty repo", rs: git.Status{Root: "/repo", EmptyRepo: true}, want: state.ModeEmpty, wantMsg: "No commits yet."},
		// A missing remote is not a resting state any more. It used to return
		// ModeBlocked, and because key_handling.go only dispatches browse keys in
		// ModeBrowse, that left every browse key dead in a local-only repo -
		// merge and rebase included, neither of which needs a remote. The
		// requirement moved to the actions that cannot run without one.
		{name: "no remote no upstream browses", rs: git.Status{Root: "/repo", NoRemote: true, NoUpstream: true}, want: state.ModeBrowse},
		{name: "no remote alone browses", rs: git.Status{Root: "/repo", NoRemote: true}, want: state.ModeBrowse},
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
		{name: "cherry-pick in progress", rs: git.Status{Root: "/repo", CherryPickInProgress: true}, want: state.ModeBlocked, wantBlk: state.BlockUnknown},
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
		{name: "rebase in progress", rs: git.Status{Root: "/repo", RebaseInProgress: true}, want: false},
		{name: "cherry-pick in progress", rs: git.Status{Root: "/repo", CherryPickInProgress: true}, want: false},
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
	if !canCreateBranch(git.Status{Root: "/repo"}) {
		t.Fatal("expected clean repo to allow branch creation")
	}
	if canCreateBranch(git.Status{WorktreeDirty: true}) {
		t.Fatal("expected dirty worktree to block branch creation")
	}
	if canCreateBranch(git.Status{Root: "/repo", MergeInProgress: true}) {
		t.Fatal("expected merge in progress to block branch creation")
	}
	if canCreateBranch(git.Status{Root: "/repo", RebaseInProgress: true}) {
		t.Fatal("expected rebase in progress to block branch creation")
	}
	if canCreateBranch(git.Status{Root: "/repo", CherryPickInProgress: true}) {
		t.Fatal("expected cherry-pick in progress to block branch creation")
	}
}

func TestBranchNameExistsAndValidation(t *testing.T) {
	rs := git.Status{
		Root:          "/repo",
		Branch:        "main",
		Branches:      []string{"main", "feature"},
		LocalBranches: []string{"main", "feature"},
	}
	if !branchNameExists(rs, "main") {
		t.Fatal("expected main to be detected as existing")
	}
	if !branchNameExists(rs, "feature") {
		t.Fatal("expected feature to be detected as existing")
	}
	if branchNameExists(rs, "new-branch") {
		t.Fatal("expected new-branch to be available")
	}

	if err := branchCreateValidationError(rs, "", "main"); err == nil || err.Error() != "branch name is empty" {
		t.Fatalf("expected empty name error, got %v", err)
	}
	if err := branchCreateValidationError(rs, "new-branch", ""); err == nil || err.Error() != "branch base is empty" {
		t.Fatalf("expected empty base error, got %v", err)
	}
	if err := branchCreateValidationError(git.Status{Root: "/repo", WorktreeDirty: true}, "new-branch", "main"); err == nil || err.Error() != "working tree is not clean" {
		t.Fatalf("expected dirty worktree error, got %v", err)
	}
	if err := branchCreateValidationError(git.Status{Root: "/repo", MergeInProgress: true}, "new-branch", "main"); err == nil || err.Error() != "merge/rebase already in progress" {
		t.Fatalf("expected merge error, got %v", err)
	}
	if err := branchCreateValidationError(git.Status{Root: "/repo", RebaseInProgress: true}, "new-branch", "main"); err == nil || err.Error() != "merge/rebase already in progress" {
		t.Fatalf("expected rebase error, got %v", err)
	}
	if err := branchCreateValidationError(git.Status{Root: "/repo", CherryPickInProgress: true}, "new-branch", "main"); err == nil || err.Error() != "merge/rebase already in progress" {
		t.Fatalf("expected cherry-pick error, got %v", err)
	}
	if err := branchCreateValidationError(git.Status{}, "new-branch", "main"); err == nil || err.Error() != "not inside a git repository" {
		t.Fatalf("expected no repo error, got %v", err)
	}
	if err := branchCreateValidationError(rs, "feature", "main"); err == nil || err.Error() != "branch name already exists" {
		t.Fatalf("expected duplicate branch error, got %v", err)
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

func TestBuildCherryPickTargets(t *testing.T) {
	rs := git.Status{
		Head: "head123",
		GraphCommits: []git.GraphCommit{
			{Hash: "merge123", Parents: []string{"a", "b"}, Subject: "merge"},
			{Hash: "pick123", Parents: []string{"root"}, Author: "Ada Lovelace", Subject: "pick me", RelativeAge: "2 days ago"},
			{Hash: "head123", Parents: []string{"pick123"}, Subject: "head"},
		},
	}
	got := buildCherryPickTargets(rs)
	if len(got) != 1 {
		t.Fatalf("expected one cherry-pick target, got %#v", got)
	}
	if got[0].Kind != state.TargetKindCommit || got[0].Ref != "pick123" {
		t.Fatalf("unexpected cherry-pick target: %#v", got[0])
	}
	if got[0].Author != "Ada Lovelace" {
		t.Fatalf("expected author to be preserved, got %#v", got[0])
	}
}

func TestActionPickCherryTargets(t *testing.T) {
	t.Run("empty blocked", func(t *testing.T) {
		got := actionPickCherryTargets(git.Status{Root: "/repo"})
		if got.Mode != state.ModeBlocked || got.Block != state.BlockTargetEmpty {
			t.Fatalf("got = %#v", got)
		}
	})
	t.Run("target selection enabled", func(t *testing.T) {
		got := actionPickCherryTargets(git.Status{
			Root: "/repo",
			Head: "head123",
			GraphCommits: []git.GraphCommit{
				{Hash: "pick123", Parents: []string{"root"}, Subject: "pick me"},
				{Hash: "head123", Parents: []string{"pick123"}, Subject: "head"},
			},
		})
		if got.Mode != state.ModeCherryPickPick {
			t.Fatalf("got = %#v", got)
		}
		if got.Selected != "pick123" || got.TargetIdx != 0 {
			t.Fatalf("expected first cherry-pick target selected, got %#v", got)
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

// The remote requirement moved out of the resting state and onto the actions
// that genuinely need one. Each names itself so the message is actionable,
// unlike the old global "Set a remote target first."
func TestRemoteMissingStatusGatesOnlyRemoteActions(t *testing.T) {
	withRemote := git.Status{Root: "/repo"}
	if _, blocked := remoteMissingStatus(withRemote, "Fetch"); blocked {
		t.Fatal("a repo with a remote must not be gated")
	}

	noRemote := git.Status{Root: "/repo", NoRemote: true}
	for _, action := range []string{"Fetch", "Push", "Fetching tags", "Pushing a tag"} {
		got, blocked := remoteMissingStatus(noRemote, action)
		if !blocked {
			t.Fatalf("%s needs a remote and must be gated", action)
		}
		if got.Mode != state.ModeBlocked || got.Block != state.BlockNoRemote {
			t.Fatalf("%s: mode = %s block = %s, want blocked/no_remote", action, got.Mode, got.Block)
		}
		if want := action + " needs a remote."; got.Detail != want {
			t.Fatalf("%s: detail = %q, want %q", action, got.Detail, want)
		}
	}
}

// Pull keeps its own gates. They are the reason the global one was redundant:
// if the resting state were the pull guard, actionPull would not need these.
func TestActionPullStillRequiresARemote(t *testing.T) {
	got := actionPull(git.Status{Root: "/repo", NoRemote: true, NoUpstream: true})
	if got.Mode != state.ModeBlocked || got.Block != state.BlockNoRemote {
		t.Fatalf("pull without a remote: mode = %s block = %s, want blocked/no_remote", got.Mode, got.Block)
	}
	if got.Detail != "Pull needs a remote." {
		t.Fatalf("pull detail = %q", got.Detail)
	}
	upstreamOnly := actionPull(git.Status{Root: "/repo", NoUpstream: true})
	if upstreamOnly.Mode != state.ModeBlocked || upstreamOnly.Block != state.BlockNoUpstream {
		t.Fatalf("pull without an upstream: mode = %s block = %s", upstreamOnly.Mode, upstreamOnly.Block)
	}
}

// The whole point of 7.5: merge and rebase are local operations, so a repo with
// no remote must reach them. Before the fix the resting state was ModeBlocked
// and handleBrowseKey never ran at all.
func TestLocalOnlyRepoReachesBrowseKeys(t *testing.T) {
	rs := git.Status{Root: "/repo", NoRemote: true, NoUpstream: true, Branch: "main", Head: "abc"}
	if got := deriveStatus(rs); got.Mode != state.ModeBrowse {
		t.Fatalf("a local-only repo rests in %s, so no browse key can fire", got.Mode)
	}
	// The state stays visible; it just is not blocking.
	if hint := repositoryStateHint(rs, false, nil); hint != "No remote or upstream" {
		t.Fatalf("expected the missing remote to stay visible as a hint, got %q", hint)
	}
}
