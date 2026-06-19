package app

import (
	"strings"
	"testing"

	"hrllk/git-graph-tui/internal/git"
	"hrllk/git-graph-tui/internal/state"
)

func TestDeriveStatusBlockedWhenDetached(t *testing.T) {
	got := deriveStatus(git.Status{Root: "/repo", Branch: "HEAD", Detached: true})
	if got.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode, got %s", got.Mode)
	}
	if got.Block != state.BlockDetached {
		t.Fatalf("expected detached block, got %s", got.Block)
	}
}

func TestActionPullRequiresUpstream(t *testing.T) {
	got := actionPull(git.Status{Root: "/repo", Branch: "main", NoRemote: false, NoUpstream: true})
	if got.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode, got %s", got.Mode)
	}
	if got.Block != state.BlockNoUpstream {
		t.Fatalf("expected no upstream block, got %s", got.Block)
	}
}

func TestActionPickTargetsBlocksWhenEmpty(t *testing.T) {
	got := actionPickTargets(git.Status{Root: "/repo", Branch: "main"}, state.ActionMerge)
	if got.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode, got %s", got.Mode)
	}
	if got.Block != state.BlockTargetEmpty {
		t.Fatalf("expected empty target block, got %s", got.Block)
	}
}

func TestActionPickTargetsUsesRefs(t *testing.T) {
	got := actionPickTargets(git.Status{
		Root:           "/repo",
		Branch:         "main",
		LocalBranches:  []string{"main"},
		RemoteBranches: []string{"origin/main"},
	}, state.ActionMerge)
	if got.Mode != state.ModeTargetPick {
		t.Fatalf("expected target pick, got %s", got.Mode)
	}
	if got.Selected != "main" {
		t.Fatalf("expected first ref selected, got %q", got.Selected)
	}
}

func TestBuildActionPreviewMergeFastForward(t *testing.T) {
	got := buildActionPreview(state.ActionMerge, "feature", git.Status{Root: "/repo", Branch: "main", Head: "abc123"}, 0, 3)
	if got.Mode != state.ModeOutcomePreview {
		t.Fatalf("expected outcome preview, got %s", got.Mode)
	}
	if !got.CanExecute {
		t.Fatalf("expected preview to be executable")
	}
	if got.Action != state.ActionMerge {
		t.Fatalf("expected merge action, got %s", got.Action)
	}
}

func TestSelectedTargetFallsBackToIndex(t *testing.T) {
	start := state.Status{
		Mode:      state.ModeTargetPick,
		Targets:   []state.TargetItem{{Name: "main", Ref: "main"}, {Name: "feature", Ref: "feature"}},
		TargetIdx: 1,
	}
	if got := selectedTarget(start); got != "feature" {
		t.Fatalf("expected selected target to fall back to index, got %q", got)
	}
}

func TestParseGraphLineExtractsHashAndDecorations(t *testing.T) {
	node := graphNode{Hash: "1a2b3c4", Parents: []string{"0a0a0a0"}}
	if node.Hash != "1a2b3c4" {
		t.Fatalf("expected hash, got %q", node.Hash)
	}
	if len(node.Parents) != 1 {
		t.Fatalf("expected parent, got %d", len(node.Parents))
	}
}

func TestCheckoutTargetFromFocusPrefersBranch(t *testing.T) {
	node := graphNode{Decorations: []string{"HEAD -> main", "origin/main"}}
	if got := checkoutTargetFromFocus(node); got != "main" {
		t.Fatalf("expected main checkout target, got %q", got)
	}
}

func TestSectionTargetsIncludesCurrentBranch(t *testing.T) {
	items := sectionTargets(git.Status{
		Branch:         "main",
		LocalBranches:  []string{"main", "develop"},
		RemoteBranches: []string{"origin/main"},
		Tags:           []string{"v1.0.0"},
	}, sectionCurrent)
	if len(items) < 2 {
		t.Fatalf("expected current section to include current plus locals, got %d", len(items))
	}
	if items[0].Ref != "main" {
		t.Fatalf("expected current branch ref, got %q", items[0].Ref)
	}
	if !items[0].Current {
		t.Fatal("expected current branch to be flagged")
	}
}

func TestGraphSectionCycle(t *testing.T) {
	if got := nextGraphSection(sectionTags); got != sectionGraph {
		t.Fatalf("expected cycle to graph, got %v", got)
	}
	if got := prevGraphSection(sectionGraph); got != sectionTags {
		t.Fatalf("expected reverse cycle to tags, got %v", got)
	}
}

func TestMoveGraphPointerClamps(t *testing.T) {
	if got := moveGraphPointer(0, 10, -1); got != 0 {
		t.Fatalf("expected top clamp, got %d", got)
	}
	if got := moveGraphPointer(9, 10, 1); got != 9 {
		t.Fatalf("expected bottom clamp, got %d", got)
	}
}

func TestGraphRowsExpandOnMerge(t *testing.T) {
	rows := graphRows(git.Status{
		GraphCommits: []git.GraphCommit{
			{Hash: "c3", Parents: []string{"b2", "a2"}},
			{Hash: "b2", Parents: []string{"a1"}},
			{Hash: "a2", Parents: []string{"a1"}},
		},
	})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if graphRowWidth(rows[0]) < 2 {
		t.Fatalf("expected merge row to expand lanes, got %d", graphRowWidth(rows[0]))
	}
	got := renderGraphLine(rows[0], true, true, 1, nil)
	if !strings.Contains(got, "*") || !strings.Contains(got, "|") {
		t.Fatalf("unexpected rendered graph row: %q", got)
	}
	if len(renderGraphConnectorLines(rows[0], rows[1])) > 1 {
		t.Fatal("expected merge row connector output to stay compact")
	}
}

func TestAdvanceGraphLanesClampsLaneBounds(t *testing.T) {
	got := advanceGraphLanes(nil, 3, "c1", []string{"p1", "p2"}, map[string][]string{})
	if len(got) == 0 {
		t.Fatal("expected lanes to be created safely")
	}
}

func TestAdvanceGraphLanesAllowsRootCommit(t *testing.T) {
	got := advanceGraphLanes([]string{"root"}, 0, "root", nil, map[string][]string{})
	if len(got) != 0 {
		t.Fatalf("expected root commit to clear active lane, got %v", got)
	}
}

func TestAdvanceGraphLanesCollapsesDuplicateCurrentLanes(t *testing.T) {
	got := advanceGraphLanes([]string{"base", "base", "base", "base"}, 3, "base", []string{"parent"}, map[string][]string{})
	if len(got) != 1 || got[0] != "parent" {
		t.Fatalf("expected collapsed lanes to continue as single parent, got %v", got)
	}
}

func TestFormatCompactDecorations(t *testing.T) {
	got := formatCompactDecorations([]string{"HEAD -> main", "develop", "origin/main", "tag: v1.0.0"}, []string{"main", "develop"})
	if !strings.Contains(got, "o->main") || !strings.Contains(got, "l->main") {
		t.Fatalf("unexpected compact decorations: %q", got)
	}
	if strings.Contains(got, "r->") {
		t.Fatalf("remote token should be omitted: %q", got)
	}
	if len([]rune(got)) > 16 {
		t.Fatalf("expected compact decorations to be clipped to 16 chars: %q", got)
	}
}

func TestCanCreateBranchRequiresCleanWorktree(t *testing.T) {
	if canCreateBranch(git.Status{WorktreeDirty: true}) {
		t.Fatal("expected dirty worktree to block branch creation")
	}
	if !canCreateBranch(git.Status{WorktreeDirty: false}) {
		t.Fatal("expected clean worktree to allow branch creation")
	}
}

func TestFindGraphRowByHash(t *testing.T) {
	rows := []graphRow{{Commit: graphNode{Hash: "a1"}}, {Commit: graphNode{Hash: "b2"}}}
	if got := findGraphRowByHash(rows, "b2"); got != 1 {
		t.Fatalf("expected to restore row by hash, got %d", got)
	}
}

func TestGraphRowsKeepsSiblingBranchesVisible(t *testing.T) {
	rows := graphRows(git.Status{
		GraphCommits: []git.GraphCommit{
			{Hash: "t3", Parents: []string{"base"}},
			{Hash: "t2", Parents: []string{"base"}},
			{Hash: "t1", Parents: []string{"base"}},
			{Hash: "base"},
		},
	})
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	if graphRowWidth(rows[0]) < 1 || graphRowWidth(rows[1]) < 2 || graphRowWidth(rows[2]) < 2 {
		t.Fatalf("expected sibling rows to expand as new tips appear, got widths %d, %d, %d", graphRowWidth(rows[0]), graphRowWidth(rows[1]), graphRowWidth(rows[2]))
	}
	if len(rows[3].Children) != 3 {
		t.Fatalf("expected branch point commit to know all children, got %d", len(rows[3].Children))
	}
}

func TestRenderGraphConnectorLinesSkipsStableTransition(t *testing.T) {
	current := graphRow{After: []string{"a", "b", "c"}}
	next := graphRow{Before: []string{"a", "b", "c"}}
	got := renderGraphConnectorLines(current, next)
	if len(got) != 0 {
		t.Fatalf("expected no connector lines for stable transition, got %v", got)
	}
}

func TestRenderGraphLineKeepsCollapsedCommitMarker(t *testing.T) {
	row := graphRow{
		Commit: graphNode{Hash: "base"},
		Before: []string{"base", "base", "base"},
		After:  []string{"base"},
		Lane:   2,
	}
	got := renderGraphLine(row, false, false, 0, nil)
	if !strings.Contains(got, "*") {
		t.Fatalf("expected collapsed commit line to keep marker, got %q", got)
	}
}

func TestHasHeadDecoration(t *testing.T) {
	if !hasHeadDecoration([]string{"HEAD -> main", "main"}) {
		t.Fatal("expected HEAD decoration to be detected")
	}
	if hasHeadDecoration([]string{"main", "origin/main"}) {
		t.Fatal("expected non-HEAD decorations to stay false")
	}
}
