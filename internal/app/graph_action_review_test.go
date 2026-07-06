package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestBuildGraphActionReviewDetailUsesCurrentTargetBaseRows(t *testing.T) {
	rs := git.Status{
		Branch: "main",
		Head:   "abc1234",
		LocalBranches: []string{
			"main",
			"feature-x",
		},
		GraphCommits: []git.GraphCommit{
			{Hash: "abc1234", Parents: []string{"9ab0cde"}, Decorations: []string{"HEAD -> main", "main"}, Subject: "current commit"},
			{Hash: "def5678", Parents: []string{"9ab0cde"}, Decorations: []string{"feature-x"}, Subject: "target commit"},
			{Hash: "9ab0cde", Parents: []string{}, Subject: "merge base"},
		},
	}

	got := ansi.Strip(buildGraphActionReviewDetail(state.ActionMerge, rs, "def5678", "9ab0cde", 12, 5))

	for _, want := range []string{
		"Review before merge.",
		"*    abc1234 CURRENT +12 (main)",
		"| *  def5678 TARGET +5 (feature-x)",
		"*    9ab0cde BASE",
		"y: continue  •  n: cancel",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected review detail to contain %q, got %q", want, got)
		}
	}
	if idxCurrent, idxTarget, idxBase := strings.Index(got, "*    abc1234"), strings.Index(got, "| *  def5678"), strings.Index(got, "*    9ab0cde"); !(idxCurrent >= 0 && idxTarget > idxCurrent && idxBase > idxTarget) {
		t.Fatalf("expected graph order current -> target -> base, got %q", got)
	}
}

func TestBuildGraphActionReviewStatusUsesBranchHasDivergedTitle(t *testing.T) {
	rs := git.Status{
		Branch: "main",
		Head:   "abc1234",
		LocalBranches: []string{
			"main",
			"feature-x",
		},
		GraphCommits: []git.GraphCommit{
			{Hash: "abc1234", Parents: []string{"9ab0cde"}, Decorations: []string{"HEAD -> main", "main"}},
			{Hash: "def5678", Parents: []string{"9ab0cde"}, Decorations: []string{"feature-x"}},
			{Hash: "9ab0cde", Parents: []string{}},
		},
	}

	got := buildGraphActionReviewStatus(state.ActionMerge, rs, "def5678", "9ab0cde", 12, 5)
	if got.Message != "Branch has diverged" {
		t.Fatalf("expected diverged title, got %q", got.Message)
	}
	if got.Mode != state.ModeReview {
		t.Fatalf("expected review mode, got %s", got.Mode)
	}
	if got.Selected != "def5678" {
		t.Fatalf("expected target selection to be preserved, got %q", got.Selected)
	}
}

func TestBuildGraphActionFastForwardStatusUsesConciseCopy(t *testing.T) {
	got := buildGraphActionFastForwardStatus(state.ActionRebase, "def5678")
	if got.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.Mode)
	}
	if got.Message != "Fast-forward available." {
		t.Fatalf("expected fast-forward message, got %q", got.Message)
	}
	if got.Detail != "HEAD can move to def5678." {
		t.Fatalf("expected concise detail, got %q", got.Detail)
	}
	if got.Selected != "def5678" {
		t.Fatalf("expected selected target to be preserved, got %q", got.Selected)
	}
}
