package app

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
		Tracking:       map[string]git.BranchTracking{"main": {Behind: 2}, "develop": {Behind: 1, Ahead: 1}},
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
	if !items[0].NeedsPull {
		t.Fatal("expected current branch to show pull flag when origin is ahead")
	}
	if items[1].NeedsPull {
		t.Fatal("expected diverged branch to avoid simple pull flag")
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

func TestInitialGraphLanesSeedsCurrentBranchWithoutRemoteTip(t *testing.T) {
	got := initialGraphLanes([]graphNode{
		{Hash: "head", Parents: []string{"base"}, Decorations: []string{"HEAD -> tmp1", "tmp1"}},
		{Hash: "base"},
	}, git.Status{
		Branch: "tmp1",
		Head:   "head",
	})
	if len(got) != 1 {
		t.Fatalf("expected current branch lane without remote tip, got %v", got)
	}
	if got[0] != (laneRef{Hash: "head", Family: "tmp1", Side: laneLocal}) {
		t.Fatalf("unexpected current branch lane: %v", got[0])
	}
}

func TestWindowResizeDoesNotIncreaseInitialGraphLoadLimit(t *testing.T) {
	m := model{commitLimit: initialGraphCommitLimit}
	gotModel, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 80})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatal("expected resize to keep graph load lazy")
	}
	if got.commitLimit != initialGraphCommitLimit {
		t.Fatalf("expected initial graph load limit to stay %d, got %d", initialGraphCommitLimit, got.commitLimit)
	}
}

func TestCheckoutResetsGraphLoadState(t *testing.T) {
	m := model{
		commitLimit:     initialGraphCommitLimit + graphLoadIncrement,
		activeSection:   sectionGraph,
		graphScroll:     12,
		graphLaneCursor: 3,
		sectionCursor: map[graphSection]int{
			sectionGraph:   15,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}
	status := git.Status{
		Branch:        "tmp1",
		Head:          "head",
		LocalBranches: []string{"tmp1"},
		GraphCommits: []git.GraphCommit{
			{Hash: "head", Parents: []string{"base"}, Decorations: []string{"HEAD -> tmp1", "tmp1"}},
			{Hash: "base"},
		},
	}
	gotModel, _ := m.Update(executedMsg{action: state.ActionCheckout, target: "tmp1", status: status})
	got := gotModel.(model)
	if got.commitLimit != initialGraphCommitLimit {
		t.Fatalf("expected checkout to reset graph load limit, got %d", got.commitLimit)
	}
	if got.graphScroll != 0 || got.sectionCursor[sectionGraph] != 0 {
		t.Fatalf("expected checkout to reset graph cursor and scroll, got cursor=%d scroll=%d", got.sectionCursor[sectionGraph], got.graphScroll)
	}
	if got.graphLaneCursor != 0 {
		t.Fatalf("expected checkout to reset lane cursor to current branch lane, got %d", got.graphLaneCursor)
	}
}

func TestMaybeLoadMoreGraphIncrementsNearLoadedBoundary(t *testing.T) {
	commits := make([]git.GraphCommit, initialGraphCommitLimit)
	for i := range commits {
		hash := fmt.Sprintf("c%02d", i)
		commits[i] = git.GraphCommit{Hash: hash}
		if i+1 < len(commits) {
			commits[i].Parents = []string{fmt.Sprintf("c%02d", i+1)}
		}
	}
	m := model{
		activeSection: sectionGraph,
		commitLimit:   initialGraphCommitLimit,
		repoStatus:    git.Status{GraphCommits: commits},
		sectionCursor: map[graphSection]int{sectionGraph: initialGraphCommitLimit - graphLoadThreshold},
	}
	got, cmd := maybeLoadMoreGraph(m)
	if cmd == nil {
		t.Fatal("expected lazy graph load command")
	}
	if got.commitLimit != initialGraphCommitLimit+graphLoadIncrement {
		t.Fatalf("expected graph load limit to increment by %d, got %d", graphLoadIncrement, got.commitLimit)
	}
}

func TestBuildFamilyPriorityUsesCurrentThenTopoDecorations(t *testing.T) {
	got := buildFamilyPriority([]graphNode{
		{Hash: "t3", Decorations: []string{"tmp3"}},
		{Hash: "t2", Decorations: []string{"tmp2"}},
		{Hash: "origin-main", Decorations: []string{"origin/main"}},
		{Hash: "head", Decorations: []string{"HEAD -> tmp1", "tmp1"}},
	}, git.Status{
		Branch:         "tmp1",
		LocalBranches:  []string{"tmp1", "tmp2", "tmp3", "main"},
		RemoteBranches: []string{"origin/main"},
	})
	if got["tmp1"] != 0 {
		t.Fatalf("expected current branch priority 0, got %d", got["tmp1"])
	}
	if !(got["tmp3"] < got["tmp2"] && got["tmp2"] < got["main"]) {
		t.Fatalf("expected non-current families to follow topo decoration order, got %v", got)
	}
}

func TestPrioritizeLaneRefsUsesFamilyPriority(t *testing.T) {
	got := prioritizeLaneRefs([]laneRef{
		{Hash: "h2", Family: "tmp2", Side: laneLocal},
		{Hash: "r1", Family: "tmp1", Side: laneRemote},
		{Hash: "h3", Family: "tmp3", Side: laneLocal},
		{Hash: "h1", Family: "tmp1", Side: laneLocal},
	}, "tmp1", map[string]int{"tmp1": 0, "tmp3": 1, "tmp2": 2})
	want := []laneRef{
		{Hash: "h1", Family: "tmp1", Side: laneLocal},
		{Hash: "r1", Family: "tmp1", Side: laneRemote},
		{Hash: "h3", Family: "tmp3", Side: laneLocal},
		{Hash: "h2", Family: "tmp2", Side: laneLocal},
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d lanes, got %v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected lane order at %d: got %v want %v (all got %v)", i, got[i], want[i], got)
		}
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
	got := advanceGraphLanes([]laneRef{{Hash: "c1", Family: "main", Side: laneLocal}}, []int{0}, graphNode{Hash: "c1", Parents: []string{"p1", "p2"}}, "", nil)
	if len(got) == 0 {
		t.Fatal("expected lanes to be created safely")
	}
}

func TestAdvanceGraphLanesAllowsRootCommit(t *testing.T) {
	got := advanceGraphLanes([]laneRef{{Hash: "root"}}, []int{0}, graphNode{Hash: "root"}, "", nil)
	if len(got) != 0 {
		t.Fatalf("expected root commit to clear active lane, got %v", got)
	}
}

func TestAdvanceGraphLanesCollapsesDuplicateCurrentLanes(t *testing.T) {
	got := advanceGraphLanes([]laneRef{
		{Hash: "base", Family: "tmp3", Side: laneLocal},
		{Hash: "base", Family: "tmp3", Side: laneRemote},
		{Hash: "base", Family: "main", Side: laneLocal},
	}, []int{0, 1, 2}, graphNode{Hash: "base", Parents: []string{"parent"}}, "tmp3", map[string]int{"tmp3": 0, "main": 1})
	if len(got) != 1 || got[0].Hash != "parent" {
		t.Fatalf("expected collapsed lanes to continue as single parent, got %v", got)
	}
}

func TestCompactLaneRefsOnlyRemovesExactDuplicates(t *testing.T) {
	got := compactLaneRefs([]laneRef{
		{Hash: "base", Family: "tmp3", Side: laneLocal},
		{Hash: "base", Family: "tmp3", Side: laneLocal},
		{Hash: "base", Family: "tmp2", Side: laneLocal},
		{Hash: "base", Family: "tmp3", Side: laneRemote},
	})
	if len(got) != 3 {
		t.Fatalf("expected only exact duplicate lanes to compact, got %v", got)
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

func TestGraphRowsKeepsLocalAndOriginDivergedFamiliesSeparate(t *testing.T) {
	rows := graphRows(git.Status{
		Branch:         "tmp3",
		Head:           "dee56f4",
		LocalBranches:  []string{"tmp3"},
		RemoteBranches: []string{"origin/tmp3"},
		GraphCommits: []git.GraphCommit{
			{Hash: "7d23746", Parents: []string{"37f0954"}, Decorations: []string{"origin/tmp3"}},
			{Hash: "37f0954", Parents: []string{"efb164e"}},
			{Hash: "dee56f4", Parents: []string{"efb164e"}, Decorations: []string{"HEAD -> tmp3", "tmp3"}},
			{Hash: "efb164e", Parents: []string{"base"}},
			{Hash: "base"},
		},
	})
	if len(rows) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(rows))
	}
	if graphRowWidth(rows[0]) < 2 || graphRowWidth(rows[1]) < 2 || graphRowWidth(rows[2]) < 2 {
		t.Fatalf("expected diverged local/origin history to stay split before merge-base, got widths %d, %d, %d", graphRowWidth(rows[0]), graphRowWidth(rows[1]), graphRowWidth(rows[2]))
	}
	if rows[0].Lane != 1 || rows[1].Lane != 1 {
		t.Fatalf("expected origin history to stay on the right lane before local head, got lanes %d and %d", rows[0].Lane, rows[1].Lane)
	}
	if graphRowWidth(rows[3]) != 1 {
		t.Fatalf("expected merge-base to collapse to one lane, got %d", graphRowWidth(rows[3]))
	}
	if rows[2].Lane != 0 {
		t.Fatalf("expected checkout branch family lane to stay leftmost, got lane %d", rows[2].Lane)
	}
	if got := renderGraphLine(rows[0], false, false, 0, nil); !strings.Contains(got, "| *") {
		t.Fatalf("expected top remote row to render as split branch, got %q", got)
	}
	if got := renderGraphLine(rows[2], false, false, 0, nil); !strings.Contains(got, "* |") {
		t.Fatalf("expected local head row to render as split branch, got %q", got)
	}
}

func TestRenderGraphConnectorLinesSkipsStableTransition(t *testing.T) {
	current := graphRow{After: []laneRef{{Hash: "a"}, {Hash: "b"}, {Hash: "c"}}}
	next := graphRow{Before: []laneRef{{Hash: "a"}, {Hash: "b"}, {Hash: "c"}}}
	got := renderGraphConnectorLines(current, next)
	if len(got) != 0 {
		t.Fatalf("expected no connector lines for stable transition, got %v", got)
	}
}

func TestRenderGraphConnectorLinesUsesSingleLineForTwoLaneCollapse(t *testing.T) {
	current := graphRow{After: []laneRef{{Hash: "base", Side: laneLocal}, {Hash: "base", Side: laneRemote}}}
	next := graphRow{
		Commit: graphNode{Hash: "base"},
		Before: []laneRef{{Hash: "base", Side: laneLocal}, {Hash: "base", Side: laneRemote}},
		After:  []laneRef{{Hash: "parent", Side: laneLocal}},
		Lane:   0,
	}
	got := renderGraphConnectorLines(current, next)
	if len(got) != 1 {
		t.Fatalf("expected single connector line for two-lane collapse, got %v", got)
	}
	if !strings.Contains(got[0], "| /") {
		t.Fatalf("expected compact connector line, got %q", got[0])
	}
}

func TestRenderGraphConnectorLinesShowsProgressiveMultiLaneCollapse(t *testing.T) {
	current := graphRow{After: []laneRef{{Hash: "base"}, {Hash: "base"}, {Hash: "base"}, {Hash: "base"}}}
	next := graphRow{
		Commit: graphNode{Hash: "base"},
		Before: []laneRef{
			{Hash: "base"},
			{Hash: "base"},
			{Hash: "base"},
			{Hash: "base"},
		},
		After: []laneRef{{Hash: "parent"}},
	}
	got := renderGraphConnectorLines(current, next)
	if len(got) != 4 {
		t.Fatalf("expected multi-lane collapse connector to show progressive convergence, got %v", got)
	}
	if !strings.Contains(got[0], "| | | |") || !strings.Contains(got[len(got)-1], "| /") {
		t.Fatalf("expected collapse connector to converge to the left lane, got %v", got)
	}
}

func TestRenderGraphConnectorLinesShowsParentShiftWithoutFullCollapse(t *testing.T) {
	current := graphRow{
		After: []laneRef{
			{Hash: "tmp1-head", Family: "tmp1", Side: laneLocal},
			{Hash: "efb164e", Family: "tmp3", Side: laneLocal},
			{Hash: "efb164e", Family: "tmp3", Side: laneRemote},
		},
	}
	next := graphRow{
		Commit: graphNode{Hash: "efb164e"},
		Before: []laneRef{
			{Hash: "tmp1-head", Family: "tmp1", Side: laneLocal},
			{Hash: "efb164e", Family: "tmp3", Side: laneLocal},
			{Hash: "efb164e", Family: "tmp3", Side: laneRemote},
		},
		After: []laneRef{
			{Hash: "tmp1-head", Family: "tmp1", Side: laneLocal},
			{Hash: "a458b4b", Family: "tmp3", Side: laneLocal},
		},
		Lane:         1,
		DisplayWidth: 3,
	}
	got := renderGraphConnectorLines(current, next)
	if len(got) != 2 {
		t.Fatalf("expected parent shift connector to keep vertical context before diagonal, got %v", got)
	}
	if !strings.Contains(got[0], "| | |") || !strings.Contains(got[1], "| | /") {
		t.Fatalf("expected shifted parent lane connector, got %v", got)
	}
}

func TestGraphRowsRenderTmp1CheckoutParentAndRootConvergence(t *testing.T) {
	rows := graphRows(git.Status{
		Branch:         "tmp1",
		Head:           "5df093e",
		LocalBranches:  []string{"tmp1", "tmp2", "tmp3", "main", "develop"},
		RemoteBranches: []string{"origin/tmp3", "origin/main"},
		GraphCommits: []git.GraphCommit{
			{Hash: "1507a22", Parents: []string{"dee56f4"}, Decorations: []string{"tmp3"}},
			{Hash: "dee56f4", Parents: []string{"efb164e"}},
			{Hash: "7d23746", Parents: []string{"37f0954"}, Decorations: []string{"origin/tmp3"}},
			{Hash: "37f0954", Parents: []string{"efb164e"}},
			{Hash: "efb164e", Parents: []string{"a458b4b"}},
			{Hash: "a458b4b", Parents: []string{"5525707"}},
			{Hash: "b219ab5", Parents: []string{"5525707"}, Decorations: []string{"tmp2"}},
			{Hash: "5df093e", Parents: []string{"5525707"}, Decorations: []string{"HEAD -> tmp1", "tmp1"}},
			{Hash: "a39d548", Parents: []string{"3999588"}, Decorations: []string{"main", "develop"}},
			{Hash: "3999588", Parents: []string{"920e141"}, Decorations: []string{"origin/main"}},
			{Hash: "920e141", Parents: []string{"7265269"}},
			{Hash: "7265269", Parents: []string{"633942e"}},
			{Hash: "633942e", Parents: []string{"93985b9"}},
			{Hash: "93985b9", Parents: []string{"460aefd"}},
			{Hash: "460aefd", Parents: []string{"4ba1faf"}},
			{Hash: "4ba1faf", Parents: []string{"5525707"}},
			{Hash: "5525707"},
		},
	})

	parentIdx := findGraphRowByHash(rows, "37f0954")
	if parentIdx < 0 || parentIdx+1 >= len(rows) || rows[parentIdx+1].Commit.Hash != "efb164e" {
		t.Fatalf("expected efb164e immediately after 37f0954, got index=%d rows=%v", parentIdx, rows)
	}
	parentConnector := renderGraphConnectorLines(rows[parentIdx], rows[parentIdx+1])
	if len(parentConnector) != 2 || !strings.Contains(parentConnector[0], "| | |") || !strings.Contains(parentConnector[1], "| | /") {
		t.Fatalf("expected 37f0954 parent edge to efb164e to render a diagonal connector, got %v", parentConnector)
	}
	parentLine := renderGraphLine(rows[parentIdx+1], false, false, 0, nil)
	if strings.Contains(parentLine, "| * |") {
		t.Fatalf("expected efb164e row to hide the converged duplicate lane, got %q", parentLine)
	}

	rootIdx := findGraphRowByHash(rows, "4ba1faf")
	if rootIdx < 0 || rootIdx+1 >= len(rows) || rows[rootIdx+1].Commit.Hash != "5525707" {
		t.Fatalf("expected 5525707 immediately after 4ba1faf, got index=%d rows=%v", rootIdx, rows)
	}
	rootConnector := renderGraphConnectorLines(rows[rootIdx], rows[rootIdx+1])
	if len(rootConnector) < 2 {
		t.Fatalf("expected common root convergence to render progressive connector lines, got %v", rootConnector)
	}
	if !strings.Contains(rootConnector[len(rootConnector)-1], "| /") {
		t.Fatalf("expected common root convergence to finish on left lane, got %v", rootConnector)
	}
}

func TestRenderGraphLineKeepsCollapsedCommitMarker(t *testing.T) {
	row := graphRow{
		Commit: graphNode{Hash: "base"},
		Before: []laneRef{{Hash: "base"}, {Hash: "base"}, {Hash: "base"}},
		After:  []laneRef{{Hash: "base"}},
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
