package app

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

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

func TestNavigationClampHelpers(t *testing.T) {
	if got := moveGraphScroll(3, 10, 4); got != 7 {
		t.Fatalf("expected graph scroll to advance within bounds, got %d", got)
	}
	if got := moveGraphScroll(9, 10, 5); got != 9 {
		t.Fatalf("expected graph scroll to clamp at max, got %d", got)
	}
	if got := clampScroll(12, 10, 4); got != 6 {
		t.Fatalf("expected page scroll to clamp to visible window, got %d", got)
	}
	if got := clampScroll(-2, 10, 4); got != 0 {
		t.Fatalf("expected scroll to clamp at top, got %d", got)
	}
	if got := clampCursor(-1, 3); got != 0 {
		t.Fatalf("expected cursor to clamp to first item, got %d", got)
	}
	if got := clampCursor(99, 3); got != 0 {
		t.Fatalf("expected cursor to clamp to first item when out of range, got %d", got)
	}
	row := graphRow{
		Commit: graphNode{Hash: "a"},
		After:  []laneRef{{Hash: "a"}, {Hash: "b"}},
	}
	if got := clampLaneCursor(7, row); got != 0 {
		t.Fatalf("expected lane cursor to clamp to pointer lane, got %d", got)
	}
}

func TestMoveSelectableGraphPointerSkipsConnectors(t *testing.T) {
	rows := []graphRow{
		{Commit: graphNode{Hash: "a"}},
		{Graph: "|\\", Commit: graphNode{}},
		{Commit: graphNode{Hash: "b"}},
	}
	if got := graph.MoveSelectableGraphPointer(0, rows, 1); got != 2 {
		t.Fatalf("expected connector row to be skipped on move down, got %d", got)
	}
	if got := graph.MoveSelectableGraphPointer(2, rows, -1); got != 0 {
		t.Fatalf("expected connector row to be skipped on move up, got %d", got)
	}
}

func TestWindowResizeDoesNotIncreaseInitialGraphLoadLimit(t *testing.T) {
	m := model{commitLimit: 0}
	gotModel, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 80})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatal("expected resize to keep graph load lazy")
	}
	if got.commitLimit != 0 {
		t.Fatalf("expected initial graph load limit to stay %d, got %d", 0, got.commitLimit)
	}
}

func TestSplitPaneWidthsUseThreeSevenRatio(t *testing.T) {
	left, right := splitPaneWidths(100)
	if left+right != 100 {
		t.Fatalf("expected widths to sum to total, got %d and %d", left, right)
	}
	if left != 30 || right != 70 {
		t.Fatalf("expected 3:7 split, got %d and %d", left, right)
	}
	left, right = splitPaneWidths(101)
	if left+right != 101 {
		t.Fatalf("expected widths to sum to total, got %d and %d", left, right)
	}
	if left < 30 || left > 31 {
		t.Fatalf("expected left pane to stay near 3/10, got %d and %d", left, right)
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
	if top != 12 || bottom != 88 {
		t.Fatalf("expected 1:7 layout split, got %d and %d", top, bottom)
	}
}

func TestSplitThreeHeightsUseStackedLayout(t *testing.T) {
	a, b, c := splitThreeHeights(100)
	if a+b+c != 100 {
		t.Fatalf("expected stacked heights to sum to total, got %d, %d, %d", a, b, c)
	}
	if a <= 0 || b <= 0 || c <= 0 {
		t.Fatalf("expected stacked heights to stay positive, got %d, %d, %d", a, b, c)
	}
}

func TestShellLayoutAllocatesSmallHeaderAndLargeGraphRail(t *testing.T) {
	m := model{width: 140, height: 60}
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	bodyWidth, bodyHeight := layoutShellBodySize(m, hMargin, topMargin, bottomMargin)
	headerHeight := layoutHeaderHeight(bodyHeight)
	graphRailHeight := layoutGraphRailHeight(bodyHeight)

	if bodyWidth != m.width-2*hMargin {
		t.Fatalf("expected body width to respect horizontal margin, got %d", bodyWidth)
	}
	if headerHeight <= 0 {
		t.Fatalf("expected positive header height, got %d", headerHeight)
	}
	if graphRailHeight <= headerHeight {
		t.Fatalf("expected graph rail to dominate header, got header=%d rail=%d", headerHeight, graphRailHeight)
	}
	if graphRailHeight < 12 {
		t.Fatalf("expected graph rail to keep minimum height, got %d", graphRailHeight)
	}
}

func TestGraphRailMatchesStackedSideRailHeight(t *testing.T) {
	m := model{width: 140, height: 60}
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	_, bodyHeight := layoutShellBodySize(m, hMargin, topMargin, bottomMargin)
	graphRailHeight := layoutGraphRailHeight(bodyHeight)
	localHeight, remoteHeight, tagsHeight := splitThreeHeights(graphRailHeight)
	if graphRailHeight != localHeight+remoteHeight+tagsHeight {
		t.Fatalf("expected graph rail height to match stacked side rail height, got %d vs %d", graphRailHeight, localHeight+remoteHeight+tagsHeight)
	}
}

func TestGraphContentMatchesStackedSideRailContent(t *testing.T) {
	m := model{width: 140, height: 60}
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	_, bodyHeight := layoutShellBodySize(m, hMargin, topMargin, bottomMargin)
	graphRailHeight := layoutGraphRailHeight(bodyHeight)
	want := graphRailHeight - 3
	if got := graphContentHeightForModel(&m); got != want {
		t.Fatalf("expected graph content height %d, got %d", want, got)
	}
}

func TestShellLayoutUsesTwelvePercentMargins(t *testing.T) {
	m := model{width: 140, height: 60}
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	if hMargin != 14 {
		t.Fatalf("expected horizontal margin to use 10%% of width, got %d", hMargin)
	}
	if topMargin != 7 {
		t.Fatalf("expected top margin to use 12%% of height, got %d", topMargin)
	}
	if bottomMargin != 7 {
		t.Fatalf("expected bottom margin to use 12%% of height, got %d", bottomMargin)
	}
}

func TestGraphPageSizeAccountsForConnectorBudget(t *testing.T) {
	m := model{height: 24}
	rows := []graphRow{
		{
			Commit:       graphNode{Hash: "c10"},
			Before:       []laneRef{{Hash: "x"}, {Hash: "c9"}},
			After:        []laneRef{{Hash: "x"}, {Hash: "c9"}},
			Lane:         0,
			DisplayWidth: 2,
		},
		{
			Commit:       graphNode{Hash: "c9"},
			Before:       []laneRef{{Hash: "x"}, {Hash: "c9"}},
			After:        []laneRef{{Hash: "c9"}},
			Lane:         0,
			DisplayWidth: 2,
		},
		{Commit: graphNode{Hash: "c8"}, Lane: 0, DisplayWidth: 1},
		{Commit: graphNode{Hash: "c7"}, Lane: 0, DisplayWidth: 1},
		{Commit: graphNode{Hash: "c6"}, Lane: 0, DisplayWidth: 1},
		{Commit: graphNode{Hash: "c5"}, Lane: 0, DisplayWidth: 1},
		{Commit: graphNode{Hash: "c4"}, Lane: 0, DisplayWidth: 1},
		{Commit: graphNode{Hash: "c3"}, Lane: 0, DisplayWidth: 1},
		{Commit: graphNode{Hash: "c2"}, Lane: 0, DisplayWidth: 1},
		{Commit: graphNode{Hash: "c1"}, Lane: 0, DisplayWidth: 1},
	}
	foundConnector := false
	for i := 0; i+1 < len(rows); i++ {
		if len(renderGraphConnectorLines(rows[i], rows[i+1], false)) > 0 {
			foundConnector = true
			break
		}
	}
	if !foundConnector {
		t.Fatal("expected at least one merge transition to produce connector lines")
	}
	got := graphPageSizeForRows(&m, rows, 0, graphContentHeightForModel(&m))
	if got <= 0 {
		t.Fatalf("expected positive graph page size, got %d", got)
	}
	if got != 5 {
		t.Fatalf("expected connector-aware page size to account for two connector lines, got %d", got)
	}
}

func TestRenderAppViewKeepsShellPlacementFullWidth(t *testing.T) {
	m := model{
		width:  140,
		height: 60,
		status: state.New().WithBrowse(),
	}
	got := renderAppView(m)
	for _, want := range []string{"Global", "Browse", "Actions", "tab: next section", "f: fetch"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected render to contain %q, got %q", want, got)
		}
	}
}

func TestMoveGraphBrowseCursorUpdatesCursorScrollAndLane(t *testing.T) {
	status := git.Status{
		GraphCommits: []git.GraphCommit{
			{Hash: "c3", Parents: []string{"b2", "a2"}},
			{Hash: "b2", Parents: []string{"a1"}},
			{Hash: "a2", Parents: []string{"a1"}},
			{Hash: "a1"},
		},
	}
	rows := graph.Rows(status)
	m := model{
		height:          80,
		repoStatus:      status,
		activeSection:   sectionGraph,
		sectionCursor:   map[graphSection]int{sectionGraph: 0},
		graphLaneCursor: 0,
		graphScroll:     0,
	}
	got := moveGraphBrowseCursor(m, 1)
	if got.sectionCursor[sectionGraph] != 1 {
		t.Fatalf("expected cursor to move to next selectable row, got %d", got.sectionCursor[sectionGraph])
	}
	if got.graphLaneCursor != graph.PointerLane(rows[1]) {
		t.Fatalf("expected lane cursor to follow selected row, got %d want %d", got.graphLaneCursor, graph.PointerLane(rows[1]))
	}
	if got.graphScroll != 0 {
		t.Fatalf("expected scroll to stay on first page, got %d", got.graphScroll)
	}
}

func TestMoveSectionBrowseCursorWraps(t *testing.T) {
	m := model{
		repoStatus: git.Status{
			Branch:         "main",
			LocalBranches:  []string{"main", "feature"},
			RemoteBranches: []string{"origin/main", "origin/dev"},
			Tags:           []string{"v1"},
		},
		activeSection: sectionCurrent,
		sectionCursor: map[graphSection]int{
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}
	got := moveSectionBrowseCursor(m, 1)
	if got.sectionCursor[sectionCurrent] != 1 {
		t.Fatalf("expected current section cursor to move forward, got %d", got.sectionCursor[sectionCurrent])
	}
	got = moveSectionBrowseCursor(got, 1)
	if got.sectionCursor[sectionCurrent] != 0 {
		t.Fatalf("expected current section cursor to wrap, got %d", got.sectionCursor[sectionCurrent])
	}
	got.activeSection = sectionTags
	got = moveSectionBrowseCursor(got, 1)
	if got.sectionCursor[sectionTags] != 0 {
		t.Fatalf("expected tags cursor to stay on only item, got %d", got.sectionCursor[sectionTags])
	}
}

func TestSyncBrowseStateRestoresGraphSelectionAndClampsSections(t *testing.T) {
	m := model{
		repoStatus: git.Status{
			GraphCommits: []git.GraphCommit{
				{Hash: "c3", Parents: []string{"b2"}},
				{Hash: "b2", Parents: []string{"a1"}},
				{Hash: "a1"},
			},
			Branch:          "main",
			LocalBranches:   []string{"main", "feature"},
			RemoteBranches:  []string{"origin/main"},
			Tags:            []string{"v1", "v2"},
			BranchUpstreams: map[string]string{"main": "origin/main", "feature": ""},
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   1,
			sectionCurrent: 1,
			sectionRemote:  0,
			sectionTags:    1,
		},
		graphScroll:     2,
		graphLaneCursor: 1,
	}
	rs := git.Status{
		GraphCommits: []git.GraphCommit{
			{Hash: "c3", Parents: []string{"b2"}},
			{Hash: "b2", Parents: []string{"a1"}},
			{Hash: "a1"},
		},
		Branch:          "main",
		LocalBranches:   []string{"main"},
		RemoteBranches:  []string{"origin/main"},
		Tags:            []string{"v1"},
		BranchUpstreams: map[string]string{"main": "origin/main"},
	}

	syncBrowseState(&m, rs)

	if m.sectionCursor[sectionGraph] != 1 {
		t.Fatalf("expected graph cursor to stay on matching hash, got %d", m.sectionCursor[sectionGraph])
	}
	if m.graphLaneCursor != graph.PointerLane(graph.Rows(rs)[1]) {
		t.Fatalf("expected graph lane cursor to be restored, got %d", m.graphLaneCursor)
	}
	if m.sectionCursor[sectionCurrent] != 0 {
		t.Fatalf("expected current section cursor to clamp to available target, got %d", m.sectionCursor[sectionCurrent])
	}
	if m.sectionCursor[sectionTags] != 0 {
		t.Fatalf("expected tags cursor to clamp to available target, got %d", m.sectionCursor[sectionTags])
	}
}

func TestSyncBrowseStatePreservesCurrentSectionBranchByRef(t *testing.T) {
	m := model{
		repoStatus: git.Status{
			Branch:          "tmp2",
			LocalBranches:   []string{"tmp2", "tmp1"},
			BranchUpstreams: map[string]string{"tmp2": "origin/tmp2", "tmp1": "origin/tmp1"},
		},
		sectionCursor: map[graphSection]int{
			sectionCurrent: 1,
		},
	}
	rs := git.Status{
		Branch:          "tmp2",
		LocalBranches:   []string{"tmp2", "tmp1"},
		BranchUpstreams: map[string]string{"tmp2": "origin/tmp2", "tmp1": "origin/tmp1"},
	}

	syncBrowseState(&m, rs)

	if m.sectionCursor[sectionCurrent] != 1 {
		t.Fatalf("expected current section cursor to stay on tmp1 by ref, got %d", m.sectionCursor[sectionCurrent])
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

func TestRenderContextContentShowsCurrentBranchState(t *testing.T) {
	m := model{
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			Branch:        "main",
			Head:          "abc1234",
			Upstream:      "origin/main",
			Remote:        "origin",
			WorktreeDirty: true,
			LocalBranches: []string{"main"},
			Tracking: map[string]git.BranchTracking{
				"main": {Behind: 1, Ahead: 2},
			},
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
		},
	}
	m.activeSection = sectionCurrent
	m.status.WorktreeState = state.WorktreeStateDirty

	got := m.renderContextContent(50, 16)
	if !strings.Contains(got, "target:") || !strings.Contains(got, "worktree:") {
		t.Fatalf("expected current branch context to show target and worktree, got %q", got)
	}
	if !strings.Contains(got, "worktree:") || !strings.Contains(got, "sync: push required") {
		t.Fatalf("expected worktree/sync details in context, got %q", got)
	}
}

func TestRenderContextContentClipsToWidth(t *testing.T) {
	m := model{
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			Branch:        "main",
			Head:          "abc1234",
			Remote:        "origin",
			WorktreeDirty: true,
			LocalBranches: []string{"main", "feature/this-is-an-extremely-long-branch-name"},
			Tags:          []string{"v1.0.0"},
		},
		activeSection: sectionCurrent,
		sectionCursor: map[graphSection]int{sectionCurrent: 0},
	}
	got := m.renderContextContent(28, 18)
	for i, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > 28 {
			t.Fatalf("expected context line %d to fit width, got width=%d line=%q", i, width, line)
		}
	}
}

func TestRenderDetailContentClipsToWidth(t *testing.T) {
	m := model{
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			Branch: "main",
			Head:   "abc1234",
			Remote: "origin",
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
	got := m.renderDetailContent(28, 18)
	for i, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > 28 {
			t.Fatalf("expected detail line %d to fit width, got width=%d line=%q", i, width, line)
		}
	}
}

func TestRenderContextContentSplitsInfoAndActions(t *testing.T) {
	m := model{
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			Branch:        "main",
			Head:          "abc1234",
			Upstream:      "origin/main",
			Remote:        "origin",
			WorktreeDirty: true,
			LocalBranches: []string{"main"},
			Tracking: map[string]git.BranchTracking{
				"main": {Behind: 1, Ahead: 2},
			},
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
		},
	}
	m.activeSection = sectionCurrent
	m.status.WorktreeState = state.WorktreeStateDirty

	got := m.renderContextContent(40, 12)
	lines := strings.Split(got, "\n")
	if len(lines) != 12 {
		t.Fatalf("expected split context to preserve fixed height, got %d lines: %q", len(lines), got)
	}
	if !strings.Contains(got, "│") {
		t.Fatalf("expected split context to include a center separator, got %q", got)
	}
	if !strings.Contains(got, "target:") || !strings.Contains(got, "worktree:") {
		t.Fatalf("expected left info column to include current branch details, got %q", got)
	}
	if !strings.Contains(got, "Context Details") || !strings.Contains(got, "Context Actions") {
		t.Fatalf("expected context header to prefix details/actions with the focused section name, got %q", got)
	}
	if !strings.Contains(got, "• space: checkout") || !strings.Contains(got, "• p: pull") {
		t.Fatalf("expected right actions column to include browse actions, got %q", got)
	}
}

func TestRenderContextContentKeepsSplitLayoutOnNarrowWidth(t *testing.T) {
	m := model{
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			Branch:        "main",
			Head:          "abc1234",
			Upstream:      "origin/main",
			Remote:        "origin",
			LocalBranches: []string{"main"},
			GraphCommits: []git.GraphCommit{
				{Hash: "abc1234", Subject: "Commit 1"},
			},
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
		},
	}
	m.activeSection = sectionGraph

	got := m.renderContextContent(22, 8)
	lines := strings.Split(got, "\n")
	if len(lines) != 8 {
		t.Fatalf("expected narrow split context to preserve height, got %d lines: %q", len(lines), got)
	}
	if !strings.Contains(got, "│") {
		t.Fatalf("expected narrow split context to keep separator, got %q", got)
	}
	if !strings.Contains(got, "focus:") || !strings.Contains(got, "m: merge") {
		t.Fatalf("expected narrow split context to keep info and actions visible, got %q", got)
	}
}

func TestRenderContextContentUsesSectionNameInHeaders(t *testing.T) {
	tests := []struct {
		name       string
		active     graphSection
		wantPrefix string
	}{
		{name: "current", active: sectionCurrent, wantPrefix: "Context"},
		{name: "graph", active: sectionGraph, wantPrefix: "Graph"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				status: state.New().WithBrowse(),
				repoStatus: git.Status{
					Branch:        "main",
					Head:          "abc1234",
					Upstream:      "origin/main",
					Remote:        "origin",
					LocalBranches: []string{"main"},
					GraphCommits: []git.GraphCommit{
						{Hash: "abc1234", Subject: "Commit 1"},
					},
				},
				sectionCursor: map[graphSection]int{
					sectionGraph:   0,
					sectionCurrent: 0,
				},
			}
			m.activeSection = tt.active

			got := m.renderContextContent(60, 12)
			if !strings.Contains(got, tt.wantPrefix+" Details") || !strings.Contains(got, tt.wantPrefix+" Actions") {
				t.Fatalf("expected headers to use %q prefix, got %q", tt.wantPrefix, got)
			}
		})
	}
}

func TestRenderContextContentSplitsAllSections(t *testing.T) {
	tests := []struct {
		name        string
		active      graphSection
		repoStatus  git.Status
		wantInfo    string
		wantActions string
	}{
		{
			name:   "graph",
			active: sectionGraph,
			repoStatus: git.Status{
				GraphCommits: []git.GraphCommit{{Hash: "abc1234", Parents: []string{"def5678"}}},
				LocalBranches: []string{
					"main",
				},
			},
			wantInfo:    "focus:",
			wantActions: "m: merge",
		},
		{
			name:   "current",
			active: sectionCurrent,
			repoStatus: git.Status{
				Branch:        "main",
				Head:          "abc1234",
				Upstream:      "origin/main",
				Remote:        "origin",
				LocalBranches: []string{"main"},
			},
			wantInfo:    "target:",
			wantActions: "space: checkout",
		},
		{
			name:   "remote",
			active: sectionRemote,
			repoStatus: git.Status{
				RemoteBranches: []string{"origin/main"},
				Remote:         "origin",
			},
			wantInfo:    "target:",
			wantActions: "space: checkout",
		},
		{
			name:   "tags",
			active: sectionTags,
			repoStatus: git.Status{
				Tags: []string{"v1.0.0"},
			},
			wantInfo:    "target:",
			wantActions: "no section actions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				status:     state.New().WithBrowse(),
				repoStatus: tt.repoStatus,
				sectionCursor: map[graphSection]int{
					sectionGraph:   0,
					sectionCurrent: 0,
					sectionRemote:  0,
					sectionTags:    0,
				},
			}
			m.activeSection = tt.active
			got := m.renderContextContent(48, 10)
			if !strings.Contains(got, "│") {
				t.Fatalf("expected split layout separator in %q", got)
			}
			if !strings.Contains(got, tt.wantInfo) {
				t.Fatalf("expected info column to include %q, got %q", tt.wantInfo, got)
			}
			if !strings.Contains(got, tt.wantActions) {
				t.Fatalf("expected actions column to include %q, got %q", tt.wantActions, got)
			}
		})
	}
}

func TestRenderAppViewUsesCenteredHeaderAndMainLayout(t *testing.T) {
	m := model{
		width:  140,
		height: 60,
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			Root:           "/repo",
			Branch:         "main",
			Head:           "abc1234",
			Upstream:       "origin/main",
			Remote:         "origin",
			LocalBranches:  []string{"main", "feature"},
			RemoteBranches: []string{"origin/main", "origin/dev"},
			Tags:           []string{"v1.0.0"},
			GraphCommits: []git.GraphCommit{
				{Hash: "abc1234", Parents: []string{"def5678"}, Decorations: []string{"HEAD -> main", "origin/main"}},
				{Hash: "def5678"},
			},
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}

	got := renderAppView(m)
	for _, want := range []string{"Global", "Context", "Graph", "Remote", "Tags"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected view to contain %q, got %q", want, got)
		}
	}

	localIdx := strings.Index(got, "Context")
	remoteIdx := strings.Index(got, "Remote")
	tagsIdx := strings.Index(got, "Tags")
	if localIdx < 0 || remoteIdx < 0 || tagsIdx < 0 {
		t.Fatalf("expected right rail sections to appear in output, got %q", got)
	}
	if !(localIdx < remoteIdx && remoteIdx < tagsIdx) {
		t.Fatalf("expected Context / Remote / Tags to stack in order, got %d / %d / %d", localIdx, remoteIdx, tagsIdx)
	}
}

func TestRenderGlobalContentUsesNewDigitMapping(t *testing.T) {
	m := model{
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			Branch: "main",
			Head:   "abc1234",
			Remote: "origin",
		},
	}
	got := m.renderGlobalContent(40, 14)
	for _, want := range []string{"Mode: Browse", "Actions", "tab: next section", "shift+tab: previous section", "j: up", "k: down", "f: fetch", "q: quit"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected global hotkeys to include %q, got %q", want, got)
		}
	}
	for _, want := range []string{"1 graph", "2 local", "3 remote", "4 tags"} {
		if strings.Contains(got, want) {
			t.Fatalf("expected numeric section hotkeys to be hidden, got %q", got)
		}
	}
}

func TestRenderLoadingShowsProgressToastOverlay(t *testing.T) {
	m := model{
		width:  120,
		height: 40,
		status: loadingToast("Fetching upstream..."),
		repoStatus: git.Status{
			Root:     "/repo",
			Branch:   "main",
			Head:     "abc1234",
			Upstream: "origin/main",
			Remote:   "origin",
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}

	got := renderAppView(m)
	if strings.Contains(got, "Mode: Loading") || strings.Contains(got, "Loading | Fetching upstream...") {
		t.Fatalf("expected loading state to stay out of the Global panel, got %q", got)
	}
	if !strings.Contains(got, "Working...") || !strings.Contains(got, "Fetching upstream...") {
		t.Fatalf("expected loading toast overlay, got %q", got)
	}
}

func TestRenderBranchOpenShowsCenteredPopupOverlay(t *testing.T) {
	m := model{
		width:       120,
		height:      40,
		branchOpen:  true,
		branchDraft: "feature/new-flow",
		branchBase:  "abc1234",
		branchError: "Branch name already exists.",
		status:      loadingToast("Enter a branch name."),
		repoStatus: git.Status{
			Root:   "/repo",
			Branch: "main",
			Head:   "abc1234",
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}

	got := renderAppView(m)
	if strings.Contains(got, "Mode: Loading") || strings.Contains(got, "Loading | Enter a branch name.") {
		t.Fatalf("expected branch input to stay out of the Global panel, got %q", got)
	}
	for _, want := range []string{"Create branch", "Enter a branch name.", "name: feature/new-flow", "base: abc1234", "Branch name already exists.", "esc: back"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected branch popup to contain %q, got %q", want, got)
		}
	}
}

func TestPullFetchWithoutIncomingCommitsShowsTransientToast(t *testing.T) {
	fixture := newCommandRepo(t)
	repoStatus := git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		Upstream:      "origin/main",
		Remote:        "origin",
		LocalBranches: []string{"main"},
		Tracking:      map[string]git.BranchTracking{"main": {}},
	}
	m := model{
		status:     state.New().WithBrowse(),
		repoStatus: repoStatus,
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}

	gotModel, cmd := m.Update(pullFetchedMsg{status: repoStatus})
	got := gotModel.(model)
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected transient loading toast, got %s", got.status.Mode)
	}
	if got.status.Message != "Already up to date." {
		t.Fatalf("expected no-op pull toast message, got %q", got.status.Message)
	}
	if got.status.Detail != "Nothing to pull from upstream." {
		t.Fatalf("expected no-op pull toast detail, got %q", got.status.Detail)
	}
	if cmd == nil {
		t.Fatal("expected transient toast dismissal command")
	}
	msg := cmd()
	done, ok := msg.(pullToastDoneMsg)
	if !ok {
		t.Fatalf("expected pullToastDoneMsg, got %T", msg)
	}
	gotModel2, cmd2 := got.Update(done)
	got2 := gotModel2.(model)
	if cmd2 != nil {
		t.Fatalf("expected no follow-up command after dismiss, got %v", cmd2)
	}
	if got2.status.Mode != state.ModeBrowse {
		t.Fatalf("expected no-op pull toast to return to browse, got %s", got2.status.Mode)
	}
}

func TestOverlayPopupKeepsBaseWidthStable(t *testing.T) {
	base := strings.Join([]string{
		"left-panel-content----right-panel",
		"left-panel-content----right-panel",
		"left-panel-content----right-panel",
	}, "\n")
	popup := strings.Join([]string{
		"popup",
		"line",
	}, "\n")

	got := overlayPopup(base, popup)
	baseLines := strings.Split(base, "\n")
	gotLines := strings.Split(got, "\n")
	if len(baseLines) != len(gotLines) {
		t.Fatalf("expected overlay to keep line count stable, got %d want %d", len(gotLines), len(baseLines))
	}
	for i, line := range gotLines {
		if width := lipgloss.Width(line); width != lipgloss.Width(baseLines[i]) {
			t.Fatalf("expected overlay to keep width stable on line %d, got %d want %d: %q", i, width, lipgloss.Width(baseLines[i]), line)
		}
	}
	if !strings.Contains(got, "popup") {
		t.Fatalf("expected popup content to remain visible, got %q", got)
	}
}

func TestSectionNameUsesContextLabel(t *testing.T) {
	if got := sectionName(sectionCurrent); got != "Context" {
		t.Fatalf("expected sectionCurrent to be labeled Context, got %q", got)
	}
}

func TestRenderAppViewUsesOuterMargins(t *testing.T) {
	m := model{
		width:  140,
		height: 60,
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			Root:     "/repo",
			Branch:   "main",
			Head:     "abc1234",
			Upstream: "origin/main",
			Remote:   "origin",
			GraphCommits: []git.GraphCommit{
				{Hash: "abc1234", Subject: "Commit 1"},
			},
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}

	got := renderAppView(m)
	lines := strings.Split(got, "\n")
	firstVisible := ""
	lastVisible := ""
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if firstVisible == "" {
			firstVisible = line
		}
		lastVisible = line
	}
	if firstVisible == "" || lastVisible == "" {
		t.Fatalf("expected visible content, got %q", got)
	}
	if !strings.HasPrefix(firstVisible, strings.Repeat(" ", 8)) {
		t.Fatalf("expected top margin of at least 8 spaces, got %q", firstVisible)
	}
	if !strings.HasPrefix(lastVisible, strings.Repeat(" ", 8)) {
		t.Fatalf("expected bottom content to keep horizontal padding, got %q", lastVisible)
	}
}

func TestRenderAppViewKeepsHeaderVisibleOnCompactScreens(t *testing.T) {
	m := model{
		width:  120,
		height: 24,
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			Root:     "/repo",
			Branch:   "main",
			Head:     "abc1234",
			Upstream: "origin/main",
			Remote:   "origin",
			GraphCommits: []git.GraphCommit{
				{Hash: "abc1234", Subject: "Commit 1"},
			},
		},
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
	}

	got := renderAppView(m)
	if !strings.Contains(got, "Global") || !strings.Contains(got, "Context") {
		t.Fatalf("expected compact render to keep the top header visible, got %q", got)
	}
}

func TestRenderActionHelpLinesAreSectionSpecific(t *testing.T) {
	graph := renderActionHelpLines(model{
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
	})
	// merge/rebase labels must be present (may be styled disabled when no local lane)
	graphJoined := strings.Join(graph, " ")
	if !strings.Contains(graphJoined, "m: merge") || !strings.Contains(graphJoined, "r: rebase") {
		t.Fatalf("expected graph actions to include merge/rebase labels, got %v", graph)
	}
	if !containsLine(graph, "• s: reset         • ctrl+u/d: scroll") {
		t.Fatalf("expected graph actions to include reset/scroll, got %v", graph)
	}
	if !containsLine(graph, "• gg: top         • G: bottom") {
		t.Fatalf("expected graph actions to use gg shortcut, got %v", graph)
	}
	if !strings.Contains(strings.Join(graph, " "), "d: delete branch") {
		t.Fatalf("expected graph actions to include delete branch, got %v", graph)
	}
	if containsLine(graph, "• space: checkout") {
		t.Fatalf("expected graph actions to exclude checkout, got %v", graph)
	}

	current := renderActionHelpLines(model{
		status:        state.New().WithBrowse(),
		activeSection: sectionCurrent,
		repoStatus: git.Status{
			Branch:        "main",
			LocalBranches: []string{"main", "feature"},
		},
		sectionCursor: map[graphSection]int{
			sectionCurrent: 0,
		},
	})
	currentJoined := strings.Join(current, " ")
	if !strings.Contains(currentJoined, "d: delete branch") {
		t.Fatalf("expected current actions to include delete branch, got %v", current)
	}
	if !strings.Contains(currentJoined, "(current branch)") {
		t.Fatalf("expected current branch delete to be disabled, got %v", current)
	}

	remote := renderActionHelpLines(model{
		status:        state.New().WithBrowse(),
		activeSection: sectionRemote,
	})
	if !containsLine(remote, "• space: checkout") {
		t.Fatalf("expected remote actions to include checkout, got %v", remote)
	}
	remoteJoined := strings.Join(remote, " ")
	if strings.Contains(remoteJoined, "m: merge") || strings.Contains(remoteJoined, "s: reset") {
		t.Fatalf("expected remote actions to exclude graph-only actions, got %v", remote)
	}
}

func TestRenderActionHelpLinesShowsCleanupActionsOnlyForDirtyCurrentSection(t *testing.T) {
	dirty := renderActionHelpLines(model{
		status:        state.New().WithBrowse(),
		activeSection: sectionCurrent,
		repoStatus: git.Status{
			Branch:        "main",
			LocalBranches: []string{"main"},
			WorktreeDirty: true,
		},
		sectionCursor: map[graphSection]int{
			sectionCurrent: 0,
		},
	})
	dirtyJoined := strings.Join(dirty, " ")
	if !strings.Contains(dirtyJoined, "s: stash changes") || !strings.Contains(dirtyJoined, "c: clean working tree") {
		t.Fatalf("expected dirty local actions to include cleanup shortcuts, got %v", dirty)
	}
	if strings.Contains(dirtyJoined, "dirty only") {
		t.Fatalf("expected dirty local actions to be enabled, got %v", dirty)
	}

	clean := renderActionHelpLines(model{
		status:        state.New().WithBrowse(),
		activeSection: sectionCurrent,
		repoStatus: git.Status{
			Branch:        "main",
			LocalBranches: []string{"main"},
		},
		sectionCursor: map[graphSection]int{
			sectionCurrent: 0,
		},
	})
	cleanJoined := strings.Join(clean, " ")
	if !strings.Contains(cleanJoined, "dirty only") {
		t.Fatalf("expected clean local actions to show dirty-only gating, got %v", clean)
	}
}

func TestRTriggersRebaseOnlyInGraphSection(t *testing.T) {
	// With no graph rows / no local lane -> 'r' should block with error message
	graph := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
		commitLimit:   0,
	}
	gotModel, cmd := graph.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatal("expected r to not trigger async cmd when not on local lane")
	}
	if got.status.Mode != state.ModeBlocked {
		t.Fatalf("expected rebase to block when not on local lane, got %s", got.status.Mode)
	}

	// Outside graph section 'r' should be ignored entirely
	current := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionCurrent,
		commitLimit:   0,
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
		commitLimit:   0,
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected fetch key to trigger background refresh")
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected fetch to keep browse mode, got %s", got.status.Mode)
	}
	if got.status.Message != "Fetching..." {
		t.Fatalf("expected fetch message to be visible, got %q", got.status.Message)
	}
}

func TestFetchKeyWorksFromAnyBrowseSection(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionCurrent,
		commitLimit:   0,
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected fetch key to trigger refresh outside graph section")
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected fetch to keep browse mode, got %s", got.status.Mode)
	}
	if got.status.Message != "Fetching..." {
		t.Fatalf("expected fetch message to be visible, got %q", got.status.Message)
	}
}

func TestNumberKeysSwitchSections(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
		commitLimit:   0,
	}

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatal("expected section switch to be handled synchronously")
	}
	if got.activeSection != sectionGraph {
		t.Fatalf("expected 1 to switch to graph section, got %v", got.activeSection)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatal("expected section switch to be handled synchronously")
	}
	if got.activeSection != sectionCurrent {
		t.Fatalf("expected 2 to switch to local/current section, got %v", got.activeSection)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatal("expected section switch to be handled synchronously")
	}
	if got.activeSection != sectionRemote {
		t.Fatalf("expected 3 to switch to remote section, got %v", got.activeSection)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatal("expected section switch to be handled synchronously")
	}
	if got.activeSection != sectionTags {
		t.Fatalf("expected 4 to switch to tags section, got %v", got.activeSection)
	}
}

func TestSpaceDoesNotCheckoutFromGraphSection(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
		commitLimit:   0,
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
		commitLimit:   0,
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
	if cmd != nil {
		t.Fatalf("expected checkout confirm to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected checkout confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionCheckout {
		t.Fatalf("expected checkout action, got %s", got.status.Action)
	}
}

func TestEnterDoesNotCheckoutInBrowseMode(t *testing.T) {
	m := model{
		status:        state.New().WithBrowse(),
		activeSection: sectionRemote,
		commitLimit:   0,
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
		commitLimit: 0,
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
		commitLimit:     0,
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
	if got.commitLimit != 0 {
		t.Fatalf("expected checkout to reset graph load limit to unlimited, got %d", got.commitLimit)
	}
	if got.graphScroll != 0 || got.sectionCursor[sectionGraph] != 0 {
		t.Fatalf("expected checkout to reset graph cursor and scroll, got cursor=%d scroll=%d", got.sectionCursor[sectionGraph], got.graphScroll)
	}
	if got.graphLaneCursor != 0 {
		t.Fatalf("expected checkout to reset lane cursor to current branch lane, got %d", got.graphLaneCursor)
	}
}

func TestGraphRowsExpandOnMerge(t *testing.T) {
	rows := graph.Rows(git.Status{
		GraphCommits: []git.GraphCommit{
			{Hash: "c3", Parents: []string{"b2", "a2"}},
			{Hash: "b2", Parents: []string{"a1"}},
			{Hash: "a2", Parents: []string{"a1"}},
		},
	})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if graph.RowWidth(rows[0]) < 2 {
		t.Fatalf("expected merge row to expand lanes, got %d", graph.RowWidth(rows[0]))
	}
	got := renderGraphLine(rows[0], true, true, 1, nil, 24, 80, false, 0)
	if !strings.Contains(got, "*") || !strings.Contains(got, "|") {
		t.Fatalf("unexpected rendered graph row: %q", got)
	}
	if len(renderGraphConnectorLines(rows[0], rows[1], false)) > 1 {
		t.Fatal("expected merge row connector output to stay compact")
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

func TestFormatCompactDecorationsUsesHeadThenAlphabeticalWithOverflow(t *testing.T) {
	got := formatCompactDecorations([]string{"HEAD -> main", "develop", "release"}, []string{"main", "develop", "release"})
	if got != "l->main +2" {
		t.Fatalf("expected HEAD branch to lead with overflow count, got %q", got)
	}

	got = formatCompactDecorations([]string{"release", "develop", "main"}, []string{"main", "develop", "release"})
	if got != "l->develop" {
		t.Fatalf("expected alphabetical local branch fallback, got %q", got)
	}

	got = formatCompactDecorations([]string{"origin/release", "origin/develop"}, nil)
	if got != "o->develop" {
		t.Fatalf("expected alphabetical remote branch fallback, got %q", got)
	}
}

func TestCanCreateBranchRequiresReadyRepo(t *testing.T) {
	if canCreateBranch(git.Status{Root: "/repo", WorktreeDirty: true}) {
		t.Fatal("expected dirty worktree to block branch creation")
	}
	if !canCreateBranch(git.Status{Root: "/repo"}) {
		t.Fatal("expected clean repo to allow branch creation")
	}
	if canCreateBranch(git.Status{Root: "/repo", MergeInProgress: true}) {
		t.Fatal("expected merge in progress to block branch creation")
	}
	if canCreateBranch(git.Status{Root: "/repo", RebaseInProgress: true}) {
		t.Fatal("expected rebase in progress to block branch creation")
	}
}

func TestFindGraphRowByHash(t *testing.T) {
	rows := []graphRow{{Commit: graphNode{Hash: "a1"}}, {Commit: graphNode{Hash: "b2"}}}
	if got := graph.FindRowByHash(rows, "b2"); got != 1 {
		t.Fatalf("expected to restore row by hash, got %d", got)
	}
}

func TestGraphRowsKeepsSiblingBranchesVisible(t *testing.T) {
	rows := graph.Rows(git.Status{
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
	if graph.RowWidth(rows[0]) < 1 || graph.RowWidth(rows[1]) < 2 || graph.RowWidth(rows[2]) < 2 {
		t.Fatalf("expected sibling rows to expand as new tips appear, got widths %d, %d, %d", graph.RowWidth(rows[0]), graph.RowWidth(rows[1]), graph.RowWidth(rows[2]))
	}
	if len(rows[3].Children) != 3 {
		t.Fatalf("expected branch point commit to know all children, got %d", len(rows[3].Children))
	}
}

func TestGraphRowsUsesRawGraphPrefixWhenAvailable(t *testing.T) {
	rows := graph.Rows(git.Status{
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
	line := renderGraphLine(rows[0], true, true, 0, []string{"main"}, 24, 80, false, 0)
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
	connector := renderGraphLine(rows[1], false, true, 0, []string{"main"}, 24, 80, false, 0)
	if !strings.Contains(connector, "|\\") {
		t.Fatalf("expected connector graph line to stay visible, got %q", connector)
	}
	focused := renderGraphLine(rows[2], true, true, 0, []string{"main"}, 24, 80, false, 0)
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
	if got := formatTargetItem(state.TargetItem{Kind: state.TargetKindLocal, Name: "main", Ref: "main", NeedsPush: true}); !strings.Contains(got, "⬆") {
		t.Fatalf("expected local-ahead branches to use an up-arrow badge, got %q", got)
	}
	if got := formatTargetItem(state.TargetItem{Kind: state.TargetKindLocal, Name: "main", Ref: "main", Current: true}); !strings.Contains(got, "l->main") {
		t.Fatalf("expected current local target to keep branch text visible, got %q", got)
	}
	if got := formatTargetItem(state.TargetItem{Kind: state.TargetKindLocal, Name: "main", Ref: "main", Current: true, WorktreeDirty: true}); !strings.Contains(got, "(dirty)") {
		t.Fatalf("expected current dirty local target to show dirty badge, got %q", got)
	}
	if !shouldHighlightStash(1, true) || shouldHighlightStash(1, false) || shouldHighlightStash(0, true) {
		t.Fatalf("expected stash highlight gating to depend on selection and count")
	}
}

func TestGraphRowsPreservesSiblingBranchDecorationsOnSameCommit(t *testing.T) {
	rows := graph.Rows(git.Status{
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
	if graph.RowWidth(rows[0]) != 1 {
		t.Fatalf("expected branch tip labels alone to not spawn extra lanes, got %d", graph.RowWidth(rows[0]))
	}
	if graph.RowWidth(rows[1]) != 1 {
		t.Fatalf("expected linear child commit to stay in one lane, got %d", graph.RowWidth(rows[1]))
	}
	if got := renderGraphLine(rows[1], false, false, 0, nil, 24, 80, false, 0); !strings.Contains(got, "*") || strings.Contains(got, "| *") {
		t.Fatalf("expected single-lane render for linear DAG, got %q", got)
	}
}

func TestGraphRowsKeepsLocalAndOriginDivergedFamiliesSeparate(t *testing.T) {
	rows := graph.Rows(git.Status{
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
	if graph.RowWidth(rows[0]) < 2 || graph.RowWidth(rows[1]) < 2 || graph.RowWidth(rows[2]) < 2 {
		t.Fatalf("expected diverged local/origin history to stay split before merge-base, got widths %d, %d, %d", graph.RowWidth(rows[0]), graph.RowWidth(rows[1]), graph.RowWidth(rows[2]))
	}
	if rows[0].Lane != 1 || rows[1].Lane != 1 {
		t.Fatalf("expected origin history to stay on the right lane before local head, got lanes %d and %d", rows[0].Lane, rows[1].Lane)
	}
	if graph.RowWidth(rows[3]) != 1 {
		t.Fatalf("expected merge-base to collapse to one lane, got %d", graph.RowWidth(rows[3]))
	}
	if rows[2].Lane != 0 {
		t.Fatalf("expected checkout branch family lane to stay leftmost, got lane %d", rows[2].Lane)
	}
	if got := renderGraphLine(rows[0], false, false, 0, nil, 24, 80, false, 0); !strings.Contains(got, "| *") {
		t.Fatalf("expected top remote row to render as split branch, got %q", got)
	}
	if got := renderGraphLine(rows[2], false, false, 0, nil, 24, 80, false, 0); !strings.Contains(got, "* |") {
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
	rows := graph.Rows(git.Status{
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

	parentIdx := graph.FindRowByHash(rows, "37f0954")
	if parentIdx < 0 || parentIdx+1 >= len(rows) || rows[parentIdx+1].Commit.Hash != "efb164e" {
		t.Fatalf("expected efb164e immediately after 37f0954, got index=%d rows=%v", parentIdx, rows)
	}
	parentLine := renderGraphLine(rows[parentIdx+1], false, false, 0, nil, 24, 80, false, 0)
	if !strings.Contains(parentLine, "efb164e") {
		t.Fatalf("expected efb164e row to render, got %q", parentLine)
	}

	rootIdx := graph.FindRowByHash(rows, "4ba1faf")
	if rootIdx < 0 || rootIdx+1 >= len(rows) || rows[rootIdx+1].Commit.Hash != "5525707" {
		t.Fatalf("expected 5525707 immediately after 4ba1faf, got index=%d rows=%v", rootIdx, rows)
	}
	rootLine := renderGraphLine(rows[rootIdx+1], false, false, 0, nil, 24, 80, false, 0)
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
	got := renderGraphLine(row, false, false, 0, nil, 24, 80, false, 0)
	if !strings.Contains(got, "*") {
		t.Fatalf("expected collapsed commit line to keep marker, got %q", got)
	}
}

func TestFitVisibleWidthTruncatesAnsiTextSafely(t *testing.T) {
	value := "\x1b[31mabcdef\x1b[0m"
	got := fitVisibleWidth(value, 2)
	if width := lipgloss.Width(got); width > 2 {
		t.Fatalf("expected ANSI text to fit width, got width=%d value=%q", width, got)
	}
	if !strings.Contains(got, "\x1b[31m") {
		t.Fatalf("expected ANSI prefix to be preserved, got %q", got)
	}
}

func TestRenderGraphLineNeverWraps(t *testing.T) {
	row := graphRow{
		Commit: graphNode{
			Hash:        "abcdef123456",
			RelativeAge: "1 minute ago",
			Subject:     "Merge branch 'feature/with-a-very-long-name' into main",
			Decorations: []string{
				"HEAD -> feature/with-a-very-long-name",
				"origin/feature/with-a-very-long-name",
			},
		},
		Graph: "*|||\\\\|||*",
	}
	got := renderGraphLine(row, true, true, 0, []string{"feature/with-a-very-long-name"}, 18, 40, false, 0)
	if width := lipgloss.Width(got); width > 40 {
		t.Fatalf("expected graph row to stay within width, got width=%d row=%q", width, got)
	}
	if !strings.Contains(got, "abcdef1") {
		t.Fatalf("expected hash to remain visible, got %q", got)
	}
	if !strings.Contains(got, "*") {
		t.Fatalf("expected graph marker to remain visible, got %q", got)
	}
}

func TestRenderGraphContentClipsHeadersBeforeRows(t *testing.T) {
	m := model{
		status: state.New().WithBrowse(),
		repoStatus: git.Status{
			GraphCommits: []git.GraphCommit{
				{
					Hash:        "abcdef123456",
					RelativeAge: "5 minutes ago",
					Subject:     "Merge branch 'main' into a-feature-branch-that-is-way-too-long",
					Decorations: []string{"HEAD -> feature-branch", "origin/feature-branch"},
				},
				{
					Graph: "*|||\\\\|||*",
					Hash:  "fedcba987654",
				},
			},
		},
		activeSection: sectionGraph,
		sectionCursor: map[graphSection]int{sectionGraph: 0},
	}
	got := m.renderGraphContent(40, 8)
	lines := strings.Split(got, "\n")
	if len(lines) != 8 {
		t.Fatalf("expected graph content to keep fixed height, got %d lines: %q", len(lines), got)
	}
	for i, line := range lines {
		if width := lipgloss.Width(line); width > 40 {
			t.Fatalf("expected line %d to fit width, got width=%d line=%q", i, width, line)
		}
	}
}

func TestRenderRightRailRendersStackedCards(t *testing.T) {
	m := model{
		repoStatus: git.Status{
			LocalBranches:  []string{"main"},
			RemoteBranches: []string{"origin/main"},
			Tags:           []string{"v1.0.0"},
		},
	}
	got := m.renderRightRail(40, 12)
	if got == "" {
		t.Fatal("expected right rail to render")
	}
	lines := strings.Split(got, "\n")
	if len(lines) != 12 {
		t.Fatalf("expected right rail to keep stacked card height, got %d lines: %q", len(lines), got)
	}
	for _, want := range []string{"Context", "Remote", "Tags"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected right rail to contain %q, got %q", want, got)
		}
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
	if got.status.Mode != state.ModeLoading || got.status.Message != "Fetching for push..." {
		t.Fatalf("expected Fetching for push... loading mode, got %s", got.status.Mode)
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
	if !strings.Contains(got2.status.Title, "Push and track remote?") {
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
	if got.status.Mode != state.ModeLoading || got.status.Message != "Fetching for push..." {
		t.Fatalf("expected Fetching for push... loading mode, got %s", got.status.Mode)
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
	if got2.status.Message != "Pushing..." {
		t.Fatalf("expected push message, got %q", got2.status.Message)
	}
}

func TestPushRejectedShowsForcePushConfirmAndHighlights(t *testing.T) {
	m := model{
		status: loadingToast("Pushing..."),
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

func TestResetTriggeredResetModePicker(t *testing.T) {
	fixture := newCommandRepo(t)
	m := model{
		repo:          fixture.repo,
		status:        state.New().WithBrowse(),
		activeSection: sectionGraph,
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
		repoStatus: git.Status{
			Root:       fixture.root,
			Branch:     "main",
			Head:       fixture.initialHash,
			HasCommits: true,
			GraphCommits: []git.GraphCommit{
				{Hash: fixture.initialHash, Subject: "Commit 1"},
			},
		},
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected async preview command for reset")
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected loading mode while preparing reset preview, got %s", got.status.Mode)
	}
	preview := cmd()
	previewMsg, ok := preview.(previewMsg)
	if !ok {
		t.Fatalf("expected previewMsg, got %T", preview)
	}
	gotModel, cmd = got.Update(previewMsg)
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no command after preview is applied, got %v", cmd)
	}
	if got.status.Mode != state.ModeResetModePick {
		t.Fatalf("expected reset mode picker, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionReset {
		t.Fatalf("expected ActionReset, got %s", got.status.Action)
	}
	if got.status.Selected != fixture.initialHash {
		t.Fatalf("expected target hash selected, got %q", got.status.Selected)
	}
	if got.status.ResetMode != state.ResetModeMixed {
		t.Fatalf("expected mixed reset to be the default, got %s", got.status.ResetMode)
	}
	if got.status.Message != "Choose a reset mode." {
		t.Fatalf("expected reset picker message, got %q", got.status.Message)
	}
	if got.status.Detail != "" {
		t.Fatalf("expected reset picker detail to be empty, got %q", got.status.Detail)
	}
}

func TestResetModePickerExecutesSelectedMode(t *testing.T) {
	fixture := newCommandRepo(t)
	m := model{
		repo:          fixture.repo,
		status:        state.New().WithResetModePick("Choose a reset mode.", ""),
		activeSection: sectionGraph,
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
		repoStatus: git.Status{
			Root:       fixture.root,
			Branch:     "main",
			Head:       fixture.initialHash,
			HasCommits: true,
		},
	}
	m.status.Selected = fixture.initialHash
	m.status.ResetMode = state.ResetModeSoft

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected async reset execution command, got nil")
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected loading mode on execute, got %s", got.status.Mode)
	}
	if got.status.Message != "Soft reset..." {
		t.Fatalf("expected soft reset running message, got %q", got.status.Message)
	}
}

func TestResetModePickerRendersCompactResetOnly(t *testing.T) {
	m := model{
		status: state.Status{
			Mode:      state.ModeResetModePick,
			Action:    state.ActionReset,
			Message:   "Choose a reset mode.",
			Detail:    "",
			ResetMode: state.ResetModeMixed,
		},
	}
	if got := renderStatusCompact(m.status); got != ok.Render("Reset") {
		t.Fatalf("expected compact reset status to hide extra text, got %q", got)
	}
}

func TestBlockedStatusRendersAsToast(t *testing.T) {
	got := renderStatusCompact(state.Status{Mode: state.ModeBlocked, Message: "Select a local branch."})
	if !strings.Contains(got, "Toast") {
		t.Fatalf("expected blocked status to render as toast, got %q", got)
	}
	if !strings.Contains(got, "Select a local branch.") {
		t.Fatalf("expected toast message to be preserved, got %q", got)
	}
}

func TestRenderResetModePopupUsesSingleModeList(t *testing.T) {
	got := renderResetModePopup(60)
	if strings.Contains(got, "enter: execute") {
		t.Fatalf("expected reset popup to hide enter trigger, got %q", got)
	}
	if strings.Count(got, "s: soft") != 1 || strings.Count(got, "m: mixed") != 1 || strings.Count(got, "h: hard") != 1 {
		t.Fatalf("expected single-line mode list, got %q", got)
	}
	if !strings.Contains(got, "Reset mode") || !strings.Contains(got, "Choose a reset mode.") || !strings.Contains(got, "esc: back") {
		t.Fatalf("expected reset popup to include title, body, and esc help, got %q", got)
	}
}

func TestResetExecutedSuccessfullyReturnsToBrowse(t *testing.T) {
	m := model{
		status:        loadingToast("Hard reset..."),
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
			Head:       "c1",
			HasCommits: true,
		},
	}

	msg := executedMsg{
		action:    state.ActionReset,
		target:    "c2",
		resetMode: state.ResetModeHard,
		status: git.Status{
			Root:       "/repo",
			Branch:     "main",
			Head:       "c2",
			HasCommits: true,
			GraphCommits: []git.GraphCommit{
				{Hash: "c2", Subject: "Commit 2"},
				{Hash: "c1", Subject: "Commit 1"},
			},
		},
	}

	gotModel, cmd := m.Update(msg)
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no async cmd on reset complete, got %v", cmd)
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected Browse mode, got %s", got.status.Mode)
	}
	if !strings.Contains(got.status.Message, "Hard reset complete: c2") {
		t.Fatalf("expected success message, got %q", got.status.Message)
	}
	if got.repoStatus.Head != "c2" {
		t.Fatalf("expected repoStatus.Head to be updated to c2, got %q", got.repoStatus.Head)
	}
}

func TestMergeExecutedSuccessfullyReturnsToBrowse(t *testing.T) {
	m := model{
		status: loadingToast("Merging..."),
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
		repoStatus: git.Status{
			Root:       "/repo",
			Branch:     "main",
			Head:       "c1",
			HasCommits: true,
		},
	}

	msg := executedMsg{
		action: state.ActionMerge,
		target: "feature",
		status: git.Status{
			Root:       "/repo",
			Branch:     "main",
			Head:       "c2",
			HasCommits: true,
			GraphCommits: []git.GraphCommit{
				{Hash: "c2", Subject: "Merge commit"},
				{Hash: "c1", Subject: "Commit 1"},
			},
		},
	}

	gotModel, cmd := m.Update(msg)
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no async cmd on merge complete, got %v", cmd)
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected Browse mode, got %s", got.status.Mode)
	}
	if !strings.Contains(got.status.Message, "Merge complete.") {
		t.Fatalf("expected merge success message, got %q", got.status.Message)
	}
	if got.repoStatus.Head != "c2" {
		t.Fatalf("expected repoStatus.Head to be updated to c2, got %q", got.repoStatus.Head)
	}
}

func TestRebaseExecutedSuccessfullyReturnsToBrowse(t *testing.T) {
	m := model{
		status: loadingToast("Rebasing..."),
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
		repoStatus: git.Status{
			Root:       "/repo",
			Branch:     "main",
			Head:       "c1",
			HasCommits: true,
		},
	}

	msg := executedMsg{
		action: state.ActionRebase,
		target: "feature",
		status: git.Status{
			Root:       "/repo",
			Branch:     "main",
			Head:       "c3",
			HasCommits: true,
			GraphCommits: []git.GraphCommit{
				{Hash: "c3", Subject: "Rebased head"},
				{Hash: "c2", Subject: "Replay"},
				{Hash: "c1", Subject: "Base"},
			},
		},
	}

	gotModel, cmd := m.Update(msg)
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no async cmd on rebase complete, got %v", cmd)
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected Browse mode, got %s", got.status.Mode)
	}
	if !strings.Contains(got.status.Message, "Rebase complete.") {
		t.Fatalf("expected rebase success message, got %q", got.status.Message)
	}
	if got.repoStatus.Head != "c3" {
		t.Fatalf("expected repoStatus.Head to be updated to c3, got %q", got.repoStatus.Head)
	}
}
