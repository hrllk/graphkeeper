package app

import (
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestExecutionDetail(t *testing.T) {
	tests := []struct {
		name   string
		action state.Action
		target string
		rs     git.Status
		want   string
	}{
		{name: "pull", action: state.ActionPull, want: "Upstream pointer is now reflected in the local branch."},
		{name: "merge", action: state.ActionMerge, target: "feature", rs: git.Status{Branch: "main"}, want: "Merge complete. HEAD now reflects main with target feature."},
		{name: "merge empty branch", action: state.ActionMerge, target: "feature", want: "Merge complete. HEAD now reflects - with target feature."},
		{name: "rebase", action: state.ActionRebase, target: "feature", want: "Rebase complete. The branch was replayed on top of feature."},
		{name: "reset", action: state.ActionReset, target: "feature", want: "Hard reset complete. HEAD now points at feature."},
		{name: "delete local", action: state.ActionDeleteBranch, target: "feature", want: "Branch deleted: feature."},
		{name: "delete origin", action: state.ActionDeleteBranch, target: "origin/feature", want: "Origin branch deleted: origin/feature."},
		{name: "delete tag", action: state.ActionDeleteTag, target: "v1.0.0", want: "Tag deleted: v1.0.0."},
		{name: "delete remote tag", action: state.ActionDeleteRemoteTag, target: "v1.0.0", want: "Remote tag deleted: v1.0.0."},
		{name: "default", action: state.ActionNone, want: "Action complete."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := executionDetail(tt.action, tt.target, tt.rs); got != tt.want {
				t.Fatalf("executionDetail() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRemoteCommitLookupHelpers(t *testing.T) {
	if got := normalizeRemoteUpstream("refs/remotes/origin/main"); got != "origin/main" {
		t.Fatalf("normalizeRemoteUpstream() = %q, want %q", got, "origin/main")
	}
	if got := normalizeRemoteUpstream("origin/main"); got != "origin/main" {
		t.Fatalf("normalizeRemoteUpstream() = %q, want %q", got, "origin/main")
	}

	commit := git.GraphCommit{Hash: "abc123", Decorations: []string{" main ", "origin/dev"}}
	if !commitHasRemoteDecoration(commit, "origin/main") {
		t.Fatal("expected direct decoration match")
	}
	if !commitHasRemoteDecoration(git.GraphCommit{Hash: "def456", Decorations: []string{"main"}}, "origin/main") {
		t.Fatal("expected origin/main to match bare main decoration")
	}
	if !commitHasRemoteDecoration(git.GraphCommit{Hash: "ghi789", Decorations: []string{"origin/main"}}, "main") {
		t.Fatal("expected bare upstream to match origin/main decoration")
	}
	if commitHasRemoteDecoration(git.GraphCommit{Hash: "zzz999", Decorations: []string{"feature"}}, "origin/main") {
		t.Fatal("did not expect remote decoration match")
	}
}

func TestFindRemoteCommitHash(t *testing.T) {
	rs := git.Status{
		GraphCommits: []git.GraphCommit{
			{Hash: "abc123", Decorations: []string{"main"}},
			{Hash: "def456", Decorations: []string{"origin/dev"}},
			{Hash: "ghi789", Decorations: []string{"feature"}},
		},
	}
	tests := []struct {
		name     string
		upstream string
		want     string
	}{
		{name: "empty", upstream: "", want: ""},
		{name: "refs remotes", upstream: "refs/remotes/origin/main", want: "abc123"},
		{name: "origin branch", upstream: "origin/dev", want: "def456"},
		{name: "bare branch", upstream: "main", want: "abc123"},
		{name: "missing", upstream: "origin/missing", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findRemoteCommitHash(rs, tt.upstream); got != tt.want {
				t.Fatalf("findRemoteCommitHash() = %q, want %q", got, tt.want)
			}
		})
	}
}
