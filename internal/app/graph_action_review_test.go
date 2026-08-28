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

func TestGraphActionVerbNamesTheAction(t *testing.T) {
	if got := graphActionVerb(state.ActionRebase); got != "rebase" {
		t.Fatalf("rebase verb = %q", got)
	}
	if got := graphActionVerb(state.ActionMerge); got != "merge" {
		t.Fatalf("merge verb = %q", got)
	}
}

// The graph gate's four states, from
// docs/20260701-0005-feature-graph-section-merge-rebase-gate-note.md. Two of them
// are not an action at all and now block with a typed reason. The fast-forward
// state stays executable: it is the only way to fast-forward a local branch onto
// another local branch's tip, and pull needs an upstream, so the note's advice to
// route it to pull does not hold.
func TestGraphActionCheckClassifiesEveryDivergenceState(t *testing.T) {
	for _, action := range []state.Action{state.ActionMerge, state.ActionRebase} {
		for _, tt := range []struct {
			name                    string
			currentOnly, targetOnly int
			wantMode                state.Mode
			wantBlock               state.BlockReason
		}{
			{"same commit", 0, 0, state.ModeBlocked, state.BlockNotDiverged},
			{"fast-forward", 0, 3, state.ModeConfirm, state.BlockNone},
			{"already contained", 3, 0, state.ModeBlocked, state.BlockNotDiverged},
			{"diverged", 3, 4, state.ModeReview, state.BlockNone},
		} {
			m := model{repositoryState: repositoryState{repoStatus: git.Status{Root: "/repo", Branch: "main", Head: "aaaaaaa"}}}
			updated, _ := m.Update(graphActionCheckMsg{
				action: action, target: "bbbbbbb", base: "ccccccc",
				repo:        git.Status{Root: "/repo", Branch: "main", Head: "aaaaaaa"},
				currentOnly: tt.currentOnly, targetOnly: tt.targetOnly,
			})
			got := updated.(model).status
			if got.Mode != tt.wantMode {
				t.Fatalf("%s %s: mode = %s, want %s (%q / %q)", action, tt.name, got.Mode, tt.wantMode, got.Message, got.Detail)
			}
			if got.Block != tt.wantBlock {
				t.Fatalf("%s %s: block = %q, want %q", action, tt.name, got.Block, tt.wantBlock)
			}
			if got.Message == "" || got.Detail == "" {
				t.Fatalf("%s %s: state without copy: %#v", action, tt.name, got)
			}
		}
	}
}
