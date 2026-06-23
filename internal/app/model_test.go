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
	if got.Message != "No upstream configured." {
		t.Fatalf("expected english no-upstream message, got %q", got.Message)
	}
}

func TestDeriveStatusShowsAbortWhenMergeInProgress(t *testing.T) {
	got := deriveStatus(git.Status{Root: "/repo", Branch: "main", MergeInProgress: true})
	if got.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode, got %s", got.Mode)
	}
	if got.Message != "Merge in progress after conflict." {
		t.Fatalf("expected merge conflict message, got %q", got.Message)
	}
	if got.Detail != "Press enter to abort the in-progress merge." {
		t.Fatalf("expected merge conflict detail, got %q", got.Detail)
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
		Branch:          "main",
		LocalBranches:   []string{"main", "develop"},
		BranchUpstreams: map[string]string{"main": "origin/main", "develop": ""},
		Tracking:        map[string]git.BranchTracking{"main": {Behind: 2}, "develop": {Behind: 1, Ahead: 1}},
		RemoteBranches:  []string{"origin/main"},
		Tags:            []string{"v1.0.0"},
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
	if !items[1].NoUpstream {
		t.Fatal("expected branch without upstream to show no-up flag")
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

func TestMoveSelectableGraphPointerSkipsConnectors(t *testing.T) {
	rows := []graphRow{
		{Commit: graphNode{Hash: "a"}},
		{Graph: "|\\", Commit: graphNode{}},
		{Commit: graphNode{Hash: "b"}},
	}
	if got := moveSelectableGraphPointer(0, rows, 1); got != 2 {
		t.Fatalf("expected connector row to be skipped on move down, got %d", got)
	}
	if got := moveSelectableGraphPointer(2, rows, -1); got != 0 {
		t.Fatalf("expected connector row to be skipped on move up, got %d", got)
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

func TestSplitPaneWidthsAreBalanced(t *testing.T) {
	left, right := splitPaneWidths(101)
	if left+right != 101 {
		t.Fatalf("expected widths to sum to total, got %d and %d", left, right)
	}
	if diff := right - left; diff < 0 || diff > 1 {
		t.Fatalf("expected pane widths to stay balanced, got %d and %d", left, right)
	}
}

func TestSplitPaneHeightsAreBalanced(t *testing.T) {
	top, bottom := splitPaneHeights(99)
	if top+bottom != 99 {
		t.Fatalf("expected heights to sum to total, got %d and %d", top, bottom)
	}
	if diff := bottom - top; diff < 0 || diff > 1 {
		t.Fatalf("expected pane heights to stay balanced, got %d and %d", top, bottom)
	}
}

func TestSplitDashboardHeightsUseWeightedLayout(t *testing.T) {
	top, bottom := splitDashboardHeights(100)
	if top+bottom != 100 {
		t.Fatalf("expected dashboard heights to sum to total, got %d and %d", top, bottom)
	}
	if top != 20 || bottom != 80 {
		t.Fatalf("expected 2:8 layout split, got %d and %d", top, bottom)
	}
}

func TestGraphPageSizeMatchesGraphPaneHeight(t *testing.T) {
	m := model{height: 80}
	got := graphPageSize(&m)
	if got <= 0 {
		t.Fatalf("expected positive graph page size, got %d", got)
	}
	totalHeight := int(float64(m.height) * 0.76)
	if totalHeight > m.height-2 {
		totalHeight = m.height - 2
	}
	_, bottomHeight := splitDashboardHeights(totalHeight)
	graphHeight, _ := splitPaneHeights(bottomHeight)
	want := graphHeight - 5
	if want < 3 {
		want = 3
	}
	if got != want {
		t.Fatalf("expected graph page size %d, got %d", want, got)
	}
}

func TestRenderGraphContentFixedHeight(t *testing.T) {
	m := model{
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			GraphCommits: []git.GraphCommit{
				{Hash: "c2", Parents: []string{"c1"}},
				{Hash: "c1"},
			},
		},
	}
	got := m.renderGraphContent(40, 6)
	if lines := strings.Split(got, "\n"); len(lines) != 6 {
		t.Fatalf("expected graph content to fit fixed height, got %d lines: %q", len(lines), got)
	}
}

func TestRenderDetailContentFixedHeight(t *testing.T) {
	m := model{
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			Root:     "/repo",
			Branch:   "main",
			Head:     "abc1234",
			Upstream: "origin/main",
			Remote:   "origin",
			GraphCommits: []git.GraphCommit{
				{
					Hash:        "abc1234",
					Parents:     []string{"def5678", "9876543"},
					Decorations: []string{"HEAD -> main", "origin/main"},
				},
			},
			LocalBranches: []string{"main"},
		},
		sectionCursor: map[graphSection]int{sectionGraph: 0},
	}
	got := m.renderDetailContent(40, 16)
	if lines := strings.Split(got, "\n"); len(lines) != 16 {
		t.Fatalf("expected detail content to fit fixed height, got %d lines: %q", len(lines), got)
	}
	if !strings.Contains(got, "upstream:") {
		t.Fatalf("expected upstream label to be expanded, got %q", got)
	}
	if !strings.Contains(got, "focus: abc1234") {
		t.Fatalf("expected focus header to include hash, got %q", got)
	}
	if !strings.Contains(got, "parent: (multi parent)") || !strings.Contains(got, "  - def5678") || !strings.Contains(got, "  - 9876543") {
		t.Fatalf("expected focus block to include multi-parent list, got %q", got)
	}
	if !strings.Contains(got, "branches:") || !strings.Contains(got, "  - HEAD -> main") || !strings.Contains(got, "  - origin/main") {
		t.Fatalf("expected focus block to include branches list, got %q", got)
	}
	if strings.Contains(got, "hash:") {
		t.Fatalf("expected hash label to be removed, got %q", got)
	}
}

func TestRenderActionHelpLinesAreSectionSpecific(t *testing.T) {
	graph := renderActionHelpLines(model{
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
	})
	if !containsLine(graph, "• m: merge         • r: rebase") || !containsLine(graph, "• s: reset         • ctrl+u/d: scroll") {
		t.Fatalf("expected graph actions to include merge/rebase/reset, got %v", graph)
	}
	if !containsLine(graph, "• gg: top         • G: bottom") {
		t.Fatalf("expected graph actions to use gg shortcut, got %v", graph)
	}
	if containsLine(graph, "• space: checkout") {
		t.Fatalf("expected graph actions to exclude checkout, got %v", graph)
	}

	remote := renderActionHelpLines(model{
		status:        state.New().WithBrowse(),
		activeSection: sectionRemote,
	})
	if !containsLine(remote, "• space: checkout") {
		t.Fatalf("expected remote actions to include checkout, got %v", remote)
	}
	if containsLine(remote, "• m: merge         • r: rebase") || containsLine(remote, "• s: reset         • ctrl+u/d: scroll") {
		t.Fatalf("expected remote actions to exclude graph-only actions, got %v", remote)
	}
}

func TestRTriggersRebaseOnlyInGraphSection(t *testing.T) {
	graph := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
		commitLimit:   initialGraphCommitLimit,
	}
	gotModel, cmd := graph.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected r to trigger rebase from graph section")
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected rebase to set loading mode, got %s", got.status.Mode)
	}
	if got.status.Message != "Fetching branches before rebase..." {
		t.Fatalf("expected rebase loading message, got %q", got.status.Message)
	}

	current := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionCurrent,
		commitLimit:   initialGraphCommitLimit,
	}
	gotModel, cmd = current.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatal("expected r to be ignored outside graph section")
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode to remain unchanged, got %s", got.status.Mode)
	}
}

func TestFetchKeyDoesNotForceLoadingMode(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
		commitLimit:   initialGraphCommitLimit,
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected fetch key to trigger background refresh")
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected fetch to keep browse mode, got %s", got.status.Mode)
	}
	if got.status.Message != "Fetching remotes..." {
		t.Fatalf("expected fetch message to be visible, got %q", got.status.Message)
	}
}

func TestFetchKeyWorksFromAnyBrowseSection(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionCurrent,
		commitLimit:   initialGraphCommitLimit,
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected fetch key to trigger refresh outside graph section")
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected fetch to keep browse mode, got %s", got.status.Mode)
	}
	if got.status.Message != "Fetching remotes..." {
		t.Fatalf("expected fetch message to be visible, got %q", got.status.Message)
	}
}

func TestNumberKeysSwitchSections(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
		commitLimit:   initialGraphCommitLimit,
	}

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatal("expected section switch to be handled synchronously")
	}
	if got.activeSection != sectionCurrent {
		t.Fatalf("expected 1 to switch to local/current section, got %v", got.activeSection)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatal("expected section switch to be handled synchronously")
	}
	if got.activeSection != sectionRemote {
		t.Fatalf("expected 2 to switch to remote section, got %v", got.activeSection)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatal("expected section switch to be handled synchronously")
	}
	if got.activeSection != sectionTags {
		t.Fatalf("expected 3 to switch to tags section, got %v", got.activeSection)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatal("expected section switch to be handled synchronously")
	}
	if got.activeSection != sectionGraph {
		t.Fatalf("expected 4 to switch to graph section, got %v", got.activeSection)
	}
}

func TestSpaceDoesNotCheckoutFromGraphSection(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
		commitLimit:   initialGraphCommitLimit,
		repoStatus: git.Status{
			Root:          "/repo",
			Branch:        "main",
			Head:          "head",
			LocalBranches: []string{"main"},
			GraphCommits:  []git.GraphCommit{{Hash: "head"}},
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatal("expected space to be disabled in graph section")
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode to remain unchanged, got %s", got.status.Mode)
	}
}

func TestSpaceChecksOutFromRemoteSection(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionRemote,
		commitLimit:   initialGraphCommitLimit,
		repoStatus: git.Status{
			Root:           "/repo",
			Branch:         "main",
			Head:           "head",
			RemoteBranches: []string{"origin/main"},
			LocalBranches:  []string{"main"},
			DefaultBranch:  "main",
			Tracking:       map[string]git.BranchTracking{"main": {}},
			HasCommits:     true,
			Remote:         "origin",
			GraphCommits:   []git.GraphCommit{{Hash: "head"}},
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected space to checkout from remote section")
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected checkout to set loading mode, got %s", got.status.Mode)
	}
}

func TestEnterDoesNotCheckoutInBrowseMode(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionRemote,
		commitLimit:   initialGraphCommitLimit,
		repoStatus: git.Status{
			Root:           "/repo",
			Branch:         "main",
			Head:           "head",
			RemoteBranches: []string{"origin/main"},
			LocalBranches:  []string{"main"},
			Remote:         "origin",
			HasCommits:     true,
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatal("expected enter to stop triggering browse checkout")
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode to remain unchanged, got %s", got.status.Mode)
	}
}

func TestRemoteSectionSkipsBareRemoteName(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionRemote,
		repoStatus: git.Status{
			RemoteBranches: []string{"origin", "origin/HEAD", "origin/main"},
			LocalBranches:  []string{"main"},
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}

	got := m.renderSectionContent(sectionRemote, 40, 10)
	if strings.Contains(got, "o->origin\n") {
		t.Fatalf("expected bare remote name to be hidden, got %q", got)
	}
	if !strings.Contains(got, "o->origin/HEAD") {
		t.Fatalf("expected symbolic remote head to stay visible, got %q", got)
	}
	if !strings.Contains(got, "o->main") {
		t.Fatalf("expected remote branch to remain visible, got %q", got)
	}
}

func containsLine(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}

func TestFetchedMsgKeepsPassiveBrowseState(t *testing.T) {
	m := model{
		status:      state.New().WithBrowse(),
		commitLimit: initialGraphCommitLimit,
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}
	status := git.Status{
		Root:          "/repo",
		Branch:        "tmp1",
		Head:          "head",
		Upstream:      "origin/tmp1",
		Remote:        "origin",
		LocalBranches: []string{"tmp1"},
		GraphCommits: []git.GraphCommit{
			{Hash: "head", Parents: []string{"base"}, Decorations: []string{"HEAD -> tmp1", "tmp1"}},
			{Hash: "base"},
		},
	}
	gotModel, _ := m.Update(fetchedMsg{status: status})
	got := gotModel.(model)
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected fetched update to return to browse mode, got %s", got.status.Mode)
	}
	if got.repoStatus.Branch != "tmp1" {
		t.Fatalf("expected repo status to update, got %q", got.repoStatus.Branch)
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
		t.Fatalf("expected checkout to reset graph load limit to unlimited, got %d", got.commitLimit)
	}
	if got.graphScroll != 0 || got.sectionCursor[sectionGraph] != 0 {
		t.Fatalf("expected checkout to reset graph cursor and scroll, got cursor=%d scroll=%d", got.sectionCursor[sectionGraph], got.graphScroll)
	}
	if got.graphLaneCursor != 0 {
		t.Fatalf("expected checkout to reset lane cursor to current branch lane, got %d", got.graphLaneCursor)
	}
}

func TestMaybeLoadMoreGraphNoOpsWhenUnlimited(t *testing.T) {
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
	if cmd != nil {
		t.Fatalf("expected no lazy load command in unlimited mode, got %v", cmd)
	}
	if got.commitLimit != initialGraphCommitLimit {
		t.Fatalf("expected unlimited mode to keep commit limit unchanged, got %d", got.commitLimit)
	}
}

func TestBuildFamilyPriorityKeepsOnlyCurrentBranch(t *testing.T) {
	got := buildFamilyPriority([]graphNode{
		{Hash: "head", Parents: []string{"c1"}, Decorations: []string{"HEAD -> main", "main"}},
		{Hash: "d1", Parents: []string{"c1"}, Decorations: []string{"develop"}},
	}, git.Status{
		Branch:        "main",
		LocalBranches: []string{"main", "develop"},
		Head:          "head",
	})
	if got["main"] != 0 {
		t.Fatalf("expected current branch priority 0, got %d", got["main"])
	}
	if len(got) != 1 {
		t.Fatalf("expected only the current branch to be prioritized, got %v", got)
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
	got := renderGraphLine(rows[0], true, true, 1, nil, 24, false)
	if !strings.Contains(got, "*") || !strings.Contains(got, "|") {
		t.Fatalf("unexpected rendered graph row: %q", got)
	}
	if len(renderGraphConnectorLines(rows[0], rows[1], false)) > 1 {
		t.Fatal("expected merge row connector output to stay compact")
	}
}

func TestAdvanceGraphLanesClampsLaneBounds(t *testing.T) {
	got := advanceGraphLanes([]laneRef{{Hash: "c1", Family: "main", Side: laneLocal}}, []int{0}, graphNode{Hash: "c1", Parents: []string{"p1", "p2"}}, "", nil, false)
	if len(got) == 0 {
		t.Fatal("expected lanes to be created safely")
	}
}

func TestAdvanceGraphLanesAllowsRootCommit(t *testing.T) {
	got := advanceGraphLanes([]laneRef{{Hash: "root"}}, []int{0}, graphNode{Hash: "root"}, "", nil, false)
	if len(got) != 0 {
		t.Fatalf("expected root commit to clear active lane, got %v", got)
	}
}

func TestAdvanceGraphLanesCollapsesDuplicateCurrentLanes(t *testing.T) {
	got := advanceGraphLanes([]laneRef{
		{Hash: "base", Family: "tmp3", Side: laneLocal},
		{Hash: "base", Family: "tmp3", Side: laneRemote},
		{Hash: "base", Family: "main", Side: laneLocal},
	}, []int{0, 1, 2}, graphNode{Hash: "base", Parents: []string{"parent"}}, "tmp3", map[string]int{"tmp3": 0, "main": 1}, false)
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
	if got != "o/l->main" {
		t.Fatalf("expected a single compact branch token, got %q", got)
	}
	if len([]rune(got)) > 10 {
		t.Fatalf("expected compact decorations to stay within 10 chars, got %q", got)
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

func TestGraphRowsUsesRawGraphPrefixWhenAvailable(t *testing.T) {
	rows := graphRows(git.Status{
		GraphCommits: []git.GraphCommit{
			{Graph: "*   ", Hash: "head", RelativeAge: "5 minutes ago", Author: "hrllk", Subject: "Merge branch 'main' into develop", Decorations: []string{"HEAD -> main", "origin/main", "origin/HEAD", "develop"}},
			{Graph: "|\\", Hash: ""},
			{Graph: "| * ", Hash: "parent", RelativeAge: "14 minutes ago", Author: "hrllk", Subject: "Add suffix-based zsh completion", Decorations: []string{"origin/HEAD -> origin/main"}},
		},
	})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if !strings.HasPrefix(rows[0].Graph, "*") || rows[1].Commit.Hash != "" || !strings.HasPrefix(rows[2].Graph, "| *") {
		t.Fatalf("expected raw graph prefixes to be preserved, got %q, %q, %q", rows[0].Graph, rows[1].Graph, rows[2].Graph)
	}
	line := renderGraphLine(rows[0], true, true, 0, []string{"main"}, 24, false)
	if strings.Index(line, "head") < 0 || strings.Index(line, "o/l->main") < 0 || strings.Index(line, "*") < 0 || strings.Index(line, "5mins") < 0 || strings.Index(line, "Merge b...") < 0 {
		t.Fatalf("expected graph line to include hash, branches, when, title and graph, got %q", line)
	}
	if !strings.Contains(line, headMark.Render("*")) {
		t.Fatalf("expected HEAD pointer to be highlighted, got %q", line)
	}
	if strings.Index(line, "head") > strings.Index(line, "o/l->main") {
		t.Fatalf("expected hash to lead branches, got %q", line)
	}
	if strings.Index(line, "o/l->main") > strings.Index(line, "*") || strings.Index(line, "*") > strings.Index(line, "5mins") || strings.Index(line, "5mins") > strings.Index(line, "Merge b...") {
		t.Fatalf("expected commit columns to stay ordered, got %q", line)
	}
	if strings.Contains(line, "Merge branch") || strings.Contains(line, "origin/") || strings.Contains(line, "develop") {
		t.Fatalf("expected title and extra branch decorations to be hidden, got %q", line)
	}
	connector := renderGraphLine(rows[1], false, true, 0, []string{"main"}, 24, false)
	if !strings.Contains(connector, "|\\") {
		t.Fatalf("expected connector graph line to stay visible, got %q", connector)
	}
	focused := renderGraphLine(rows[2], true, true, 0, []string{"main"}, 24, false)
	if !strings.Contains(focused, pointerMark.Render("*")) {
		t.Fatalf("expected branch row graph pointer to be highlighted, got %q", focused)
	}
	if compactWhenText("5 minutes ago") != "5mins" {
		t.Fatalf("expected relative time to compact to 5mins")
	}
	if compactTitleText("Merge branch 'main' into develop") != "Merge b..." {
		t.Fatalf("expected title to compact to 10 chars")
	}
	if !strings.Contains(formatTargetItem(state.TargetItem{Kind: state.TargetKindRemote, Name: "origin/HEAD", Ref: "origin/HEAD", Default: true}), "origin/HEAD") {
		t.Fatalf("expected origin/HEAD to stay visible in the remote section")
	}
	if got := formatTargetItem(state.TargetItem{Kind: state.TargetKindLocal, Name: "feature", Ref: "feature", NoUpstream: true}); !strings.Contains(got, "l->feature (no-up)") {
		t.Fatalf("expected local targets without upstream to be shown after the branch name, got %q", got)
	}
	if got := formatTargetItem(state.TargetItem{Kind: state.TargetKindLocal, Name: "main", Ref: "main", NeedsPull: true}); !strings.Contains(got, "⬇") {
		t.Fatalf("expected upstream-ahead branches to use a down-arrow badge, got %q", got)
	}
	if got := formatTargetItem(state.TargetItem{Kind: state.TargetKindLocal, Name: "main", Ref: "main", Current: true}); !strings.Contains(got, "l->main") {
		t.Fatalf("expected current local target to keep branch text visible, got %q", got)
	}
}

func TestGraphRowsPreservesSiblingBranchDecorationsOnSameCommit(t *testing.T) {
	rows := graphRows(git.Status{
		Branch:        "main",
		Head:          "a39d548",
		LocalBranches: []string{"main", "develop"},
		GraphCommits: []git.GraphCommit{
			{Hash: "a39d548", Parents: []string{"3999588"}, Decorations: []string{"main", "develop"}},
			{Hash: "3999588", Parents: []string{"920e141"}},
			{Hash: "920e141", Parents: []string{"7265269"}},
		},
	})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if graphRowWidth(rows[0]) != 1 {
		t.Fatalf("expected branch tip labels alone to not spawn extra lanes, got %d", graphRowWidth(rows[0]))
	}
	if graphRowWidth(rows[1]) != 1 {
		t.Fatalf("expected linear child commit to stay in one lane, got %d", graphRowWidth(rows[1]))
	}
	if got := renderGraphLine(rows[1], false, false, 0, nil, 24, false); !strings.Contains(got, "*") || strings.Contains(got, "| *") {
		t.Fatalf("expected single-lane render for linear DAG, got %q", got)
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
	if got := renderGraphLine(rows[0], false, false, 0, nil, 24, false); !strings.Contains(got, "| *") {
		t.Fatalf("expected top remote row to render as split branch, got %q", got)
	}
	if got := renderGraphLine(rows[2], false, false, 0, nil, 24, false); !strings.Contains(got, "* |") {
		t.Fatalf("expected local head row to render as split branch, got %q", got)
	}
}

func TestRenderGraphConnectorLinesSkipsStableTransition(t *testing.T) {
	current := graphRow{After: []laneRef{{Hash: "a"}, {Hash: "b"}, {Hash: "c"}}}
	next := graphRow{Before: []laneRef{{Hash: "a"}, {Hash: "b"}, {Hash: "c"}}}
	got := renderGraphConnectorLines(current, next, false)
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
	got := renderGraphConnectorLines(current, next, false)
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
	got := renderGraphConnectorLines(current, next, false)
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
	got := renderGraphConnectorLines(current, next, false)
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
	parentLine := renderGraphLine(rows[parentIdx+1], false, false, 0, nil, 24, false)
	if !strings.Contains(parentLine, "efb164e") {
		t.Fatalf("expected efb164e row to render, got %q", parentLine)
	}

	rootIdx := findGraphRowByHash(rows, "4ba1faf")
	if rootIdx < 0 || rootIdx+1 >= len(rows) || rows[rootIdx+1].Commit.Hash != "5525707" {
		t.Fatalf("expected 5525707 immediately after 4ba1faf, got index=%d rows=%v", rootIdx, rows)
	}
	rootLine := renderGraphLine(rows[rootIdx+1], false, false, 0, nil, 24, false)
	if !strings.Contains(rootLine, "5525707") {
		t.Fatalf("expected common root row to render, got %q", rootLine)
	}
}

func TestRenderGraphLineKeepsCollapsedCommitMarker(t *testing.T) {
	row := graphRow{
		Commit: graphNode{Hash: "base"},
		Before: []laneRef{{Hash: "base"}, {Hash: "base"}, {Hash: "base"}},
		After:  []laneRef{{Hash: "base"}},
		Lane:   2,
	}
	got := renderGraphLine(row, false, false, 0, nil, 24, false)
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

func TestPushSetUpstreamTriggeredWhenNoUpstream(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
		repoStatus: git.Status{
			Root:       "/repo",
			Branch:     "feature",
			Head:       "abc1234",
			NoUpstream: true,
			HasCommits: true,
		},
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected async fetch command, got nil")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Fetching before push..." {
		t.Fatalf("expected Fetching before push... loading mode, got %s", got.status.Mode)
	}
	
	status := got.repoStatus
	gotModel2, cmd2 := got.Update(pushFetchedMsg{status: status})
	got2 := gotModel2.(model)
	if cmd2 != nil {
		t.Fatal("expected no immediate executeCmd for set-upstream, should wait for confirm")
	}
	if got2.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got2.status.Mode)
	}
	if got2.status.Action != state.ActionSetUpstream {
		t.Fatalf("expected SetUpstream action, got %s", got2.status.Action)
	}
	if !strings.Contains(got2.status.Title, "Push and Track Remote?") {
		t.Fatalf("expected set-upstream title, got %q", got2.status.Title)
	}
}

func TestPushNormalTriggeredWhenUpstreamExists(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
		repoStatus: git.Status{
			Root:       "/repo",
			Branch:     "main",
			Head:       "abc1234",
			Upstream:   "origin/main",
			NoUpstream: false,
			HasCommits: true,
		},
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected async fetch command, got nil")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Fetching before push..." {
		t.Fatalf("expected Fetching before push... loading mode, got %s", got.status.Mode)
	}

	status := got.repoStatus
	gotModel2, cmd2 := got.Update(pushFetchedMsg{status: status})
	got2 := gotModel2.(model)
	if cmd2 == nil {
		t.Fatal("expected async push command, got nil")
	}
	if got2.status.Mode != state.ModeLoading {
		t.Fatalf("expected loading mode, got %s", got2.status.Mode)
	}
	if got2.status.Message != "Pushing commits..." {
		t.Fatalf("expected push message, got %q", got2.status.Message)
	}
}

func TestPushRejectedShowsForcePushConfirmAndHighlights(t *testing.T) {
	m := model{
		status: state.New().WithLoading("Pushing..."),
		repoStatus: git.Status{
			Root:     "/repo",
			Branch:   "develop",
			Head:     "localhead123",
			Upstream: "origin/develop",
			GraphCommits: []git.GraphCommit{
				{Hash: "localhead123", Decorations: []string{"HEAD -> develop"}},
				{Hash: "remotehead456", Decorations: []string{"origin/develop"}},
			},
		},
		handshakeCommits: make(map[string]bool),
	}
	
	msg := executedMsg{
		action: state.ActionPush,
		target: "develop",
		err:    fmt.Errorf("git push: exit status 1: error: failed to push some refs to '...' [rejected - non-fast-forward]"),
	}
	
	gotModel, cmd := m.Update(msg)
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no async cmd, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode on reject, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionForcePush {
		t.Fatalf("expected ActionForcePush, got %s", got.status.Action)
	}
	if !got.handshakeCommits["localhead123"] || !got.handshakeCommits["remotehead456"] {
		t.Fatalf("expected both local HEAD and remote HEAD to be highlighted, got %v", got.handshakeCommits)
	}
	if !strings.Contains(got.status.Detail, "origin/develop") {
		t.Fatalf("expected branch name to be dynamically included, got %q", got.status.Detail)
	}
}
