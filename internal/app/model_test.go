package app

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

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

func TestGraphStatusTextUsesFixedSemanticValues(t *testing.T) {
	tests := []struct {
		name       string
		stashCount int
		tagCount   int
		want       string
	}{
		{name: "none", want: ""},
		{name: "stash", stashCount: 1, want: "S"},
		{name: "tag", tagCount: 1, want: "T"},
		{name: "both", stashCount: 2, tagCount: 3, want: "S·T"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := graphStatusText(tt.stashCount, tt.tagCount); got != tt.want {
				t.Fatalf("expected status %q, got %q", tt.want, got)
			}
			if got := lipgloss.Width(renderGraphStatus(tt.stashCount, tt.tagCount)); got != graphStatusWidth {
				t.Fatalf("expected fixed status width %d, got %d", graphStatusWidth, got)
			}
		})
	}
}

func TestGraphStatusColumnKeepsTopologyPointer(t *testing.T) {
	rows := graphRows(git.Status{
		GraphCommits: []git.GraphCommit{{Hash: "abc1234", Subject: "Release", Tags: []string{"v1.0.0"}}},
	})
	if len(rows) != 1 {
		t.Fatalf("expected one graph row, got %d", len(rows))
	}
	line := ansi.Strip(renderGraphLine(rows[0], false, false, 0, nil, 18, 80, false, 1))
	if !strings.Contains(line, "*") || !strings.Contains(line, "S·T") {
		t.Fatalf("expected topology pointer and combined status, got %q", line)
	}
	if strings.Index(line, "S·T") > strings.Index(line, "*") {
		t.Fatalf("expected state column to precede topology, got %q", line)
	}
}

func TestGraphStatusColumnRemainsVisibleAcrossColorProfiles(t *testing.T) {
	for _, profileCase := range allColorProfiles() {
		t.Run(profileCase.name, func(t *testing.T) {
			withColorProfile(t, profileCase.profile)
			got := strings.TrimSpace(ansi.Strip(renderGraphStatus(1, 1)))
			if got != "S·T" {
				t.Fatalf("expected visible combined marker in %s profile, got %q", profileCase.name, got)
			}
		})
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

func withColorProfile(t *testing.T, profile termenv.Profile) {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(profile)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(previous)
	})
}

func forceTrueColorProfile(t *testing.T) {
	withColorProfile(t, termenv.TrueColor)
}

func allColorProfiles() []struct {
	name    string
	profile termenv.Profile
} {
	return []struct {
		name    string
		profile termenv.Profile
	}{
		{"ascii", termenv.Ascii},
		{"ansi", termenv.ANSI},
		{"ansi256", termenv.ANSI256},
		{"truecolor", termenv.TrueColor},
	}
}

func TestSemanticWarningUsesANSIColorAcrossProfiles(t *testing.T) {
	for _, profileCase := range allColorProfiles() {
		t.Run(profileCase.name, func(t *testing.T) {
			withColorProfile(t, profileCase.profile)
			got := warn.Render("warning")
			if ansi.Strip(got) != "warning" {
				t.Fatalf("visible text changed: %q", got)
			}
			if profileCase.profile == termenv.Ascii {
				if got != "warning" {
					t.Fatalf("ascii profile emitted style: %q", got)
				}
				return
			}
			if strings.Contains(got, "38;2;") || strings.Contains(got, "48;2;") {
				t.Fatalf("semantic warning must not become RGB: %q", got)
			}
			if !strings.Contains(got, "33") {
				t.Fatalf("warning must use ANSI yellow: %q", got)
			}
		})
	}
}

func TestHotkeyUsesANSIColorWithoutUnderline(t *testing.T) {
	withColorProfile(t, termenv.TrueColor)
	got := renderHotkey("k")
	if ansi.Strip(got) != "k" {
		t.Fatalf("visible hotkey changed: %q", got)
	}
	if !strings.Contains(got, "35") {
		t.Fatalf("hotkey must use ANSI magenta: %q", got)
	}
	if strings.Contains(got, "\x1b[4") {
		t.Fatalf("hotkey must not use underline: %q", got)
	}
}

func TestHandshakePreservesStashAndTagPointColors(t *testing.T) {
	forceTrueColorProfile(t)
	row := graphRow{Commit: graphNode{Hash: "abc1234", Tags: []string{"v1.0.0"}}, Graph: "*"}
	got := renderGraphLine(row, false, false, 0, nil, 12, 80, true, 1)
	if !strings.Contains(got, handshakeMark.Render(tagOverlapColor.Render("*"))) {
		t.Fatalf("expected handshake marker to preserve stash/tag point color, got %q", got)
	}
	raw := graphRow{Commit: graphNode{Hash: "abc1234", Tags: []string{"v1.0.0"}}, Graph: "*   "}
	got = renderGraphLine(raw, false, false, 0, nil, 12, 80, true, 1)
	if !strings.Contains(got, handshakeMark.Render(tagOverlapColor.Render("*"))) {
		t.Fatalf("expected raw handshake marker to preserve stash/tag point color, got %q", got)
	}
}

func TestTagSelectionUsesForegroundWithoutReverse(t *testing.T) {
	withColorProfile(t, termenv.TrueColor)
	label := formatSectionTargetItem(state.TargetItem{
		Kind:       state.TargetKindTag,
		Name:       "v1.0.0",
		CommitHash: "abc1234",
	}, 80)
	got := renderSelectableSectionItem(label, true, 80)
	if ansi.Strip(got) == "" {
		t.Fatalf("expected visible selected tag row: %q", got)
	}
	if strings.Contains(got, "\x1b[7") {
		t.Fatalf("tag hover must not use reverse/background: %q", got)
	}
	if !strings.Contains(got, "33") || !strings.Contains(got, "35") {
		t.Fatalf("tag hover must preserve yellow selection and magenta tag foreground: %q", got)
	}
}

func TestSearchHighlightUsesTerminalAttributes(t *testing.T) {
	for _, profileCase := range allColorProfiles() {
		t.Run(profileCase.name, func(t *testing.T) {
			withColorProfile(t, profileCase.profile)
			match := highlightSearchText("Add branch", "branch", false)
			focus := highlightSearchText("Add branch", "branch", true)
			for name, got := range map[string]string{"match": match, "focus": focus} {
				if ansi.Strip(got) != "Add branch" {
					t.Fatalf("%s changed visible text: %q", name, got)
				}
				if strings.Contains(got, "38;2;") || strings.Contains(got, "48;2;") {
					t.Fatalf("%s uses RGB: %q", name, got)
				}
			}
			if profileCase.profile == termenv.Ascii {
				if match != "Add branch" || focus != "Add branch" {
					t.Fatalf("ascii profile emitted style: %q / %q", match, focus)
				}
				return
			}
			if match == "Add branch" || focus == "Add branch" || match == focus {
				t.Fatalf("search attributes are not distinguishable: %q / %q", match, focus)
			}
		})
	}
}

func TestHandshakeUsesTerminalAttributes(t *testing.T) {
	for _, profileCase := range allColorProfiles() {
		t.Run(profileCase.name, func(t *testing.T) {
			withColorProfile(t, profileCase.profile)
			got := handshakeMark.Render("*")
			if ansi.Strip(got) != "*" {
				t.Fatalf("visible marker changed: %q", got)
			}
			if strings.Contains(got, "38;2;") || strings.Contains(got, "48;2;") {
				t.Fatalf("handshake uses RGB: %q", got)
			}
			if profileCase.profile != termenv.Ascii && got == "*" {
				t.Fatalf("handshake lost terminal attributes: %q", got)
			}
		})
	}
}

func TestSecondaryStylesUseTerminalForeground(t *testing.T) {
	styles := []struct {
		name  string
		style lipgloss.Style
	}{
		{"muted", muted},
		{"popup-help", popupHelp},
		{"disabled", disabled},
		{"review-footer", reviewFooter},
	}

	for _, profileCase := range allColorProfiles() {
		t.Run(profileCase.name, func(t *testing.T) {
			withColorProfile(t, profileCase.profile)
			for _, styleCase := range styles {
				t.Run(styleCase.name, func(t *testing.T) {
					got := styleCase.style.Render("help text")
					if ansi.Strip(got) != "help text" {
						t.Fatalf("visible text changed: %q", got)
					}
					if strings.Contains(got, "38;") || strings.Contains(got, "48;") {
						t.Fatalf("secondary style must use terminal foreground: %q", got)
					}
				})
			}
		})
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

func TestModelInitLoadsInitialState(t *testing.T) {
	m := model{status: state.New().WithBrowse()}
	if cmd := m.Init(); cmd == nil {
		t.Fatal("expected init to load the initial repo state")
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

func TestShellLayoutUsesFullHeightForGraphAndRightRail(t *testing.T) {
	m := model{
		navigationState: navigationState{
			width:  140,
			height: 60,
		}}
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	bodyWidth, bodyHeight := layoutShellContentSize(m, hMargin, topMargin, bottomMargin)
	graphHeight := graphBoxHeightForModel(&m)

	if bodyWidth != m.width-2*hMargin {
		t.Fatalf("expected body width to respect horizontal margin, got %d", bodyWidth)
	}
	if graphHeight != bodyHeight {
		t.Fatalf("expected Graph to use full body height, got %d vs %d", graphHeight, bodyHeight)
	}
}

func TestGraphRailMatchesFourStackedPanelHeight(t *testing.T) {
	m := model{
		navigationState: navigationState{
			width:  140,
			height: 60,
		}}
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	_, bodyHeight := layoutShellContentSize(m, hMargin, topMargin, bottomMargin)
	detailsHeight, localHeight, remoteHeight, tagsHeight := splitFourHeights(bodyHeight)
	if bodyHeight != detailsHeight+localHeight+remoteHeight+tagsHeight {
		t.Fatalf("expected four right rail panels to consume body height, got %d vs %d", bodyHeight, detailsHeight+localHeight+remoteHeight+tagsHeight)
	}
}

func TestSplitFourHeightsDistributesSmallRemainders(t *testing.T) {
	for total, want := range map[int][4]int{
		0: {0, 0, 0, 0},
		1: {1, 0, 0, 0},
		2: {1, 1, 0, 0},
		3: {1, 1, 1, 0},
		5: {2, 1, 1, 1},
	} {
		got := [4]int{}
		got[0], got[1], got[2], got[3] = splitFourHeights(total)
		if got != want {
			t.Fatalf("splitFourHeights(%d) = %v, want %v", total, got, want)
		}
	}
}

func TestGraphContentMatchesStackedSideRailContent(t *testing.T) {
	m := model{
		navigationState: navigationState{
			width:  140,
			height: 60,
		}}
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	_, bodyHeight := layoutShellContentSize(m, hMargin, topMargin, bottomMargin)
	want := bodyHeight - 2
	if got := graphContentHeightForModel(&m); got != want {
		t.Fatalf("expected graph content height %d, got %d", want, got)
	}
}

func TestRenderTitleStripFitsVisibleWidth(t *testing.T) {
	got := renderTitleStrip(baseBox, "Global", 20)
	if width := lipgloss.Width(got); width != 22 {
		t.Fatalf("expected title strip to use adjusted width 22, got width=%d strip=%q", width, got)
	}
	if !strings.Contains(got, "Global") {
		t.Fatalf("expected title strip to contain title, got %q", got)
	}
	if !strings.HasPrefix(ansi.Strip(got), "╭") {
		t.Fatalf("expected title strip to start at left edge, got %q", got)
	}
}

func TestJoinLayoutSectionsUsesSharedSingleLineGap(t *testing.T) {
	if got := joinLayoutSections("title", "body", "footer"); got != "title\n\nbody\n\nfooter" {
		t.Fatalf("expected shared section gap, got %q", got)
	}
	if got := joinLayoutSections("title", "", "footer"); got != "title\n\nfooter" {
		t.Fatalf("expected empty sections to be ignored, got %q", got)
	}
}

func TestRenderFloatingTitleFrameUsesOuterHeight(t *testing.T) {
	got := renderFloatingTitleFrame(baseBox, "Test", "body\nmore body", 24, 6)
	if lines := strings.Split(got, "\n"); len(lines) != 6 {
		t.Fatalf("expected frame outer height 6, got %d: %q", len(lines), got)
	}
	last := ansi.Strip(strings.Split(got, "\n")[5])
	if !strings.Contains(last, "╰") && !strings.Contains(last, "└") {
		t.Fatalf("expected frame bottom border to remain on the final row, got %q", last)
	}
}

func TestRenderFloatingTitlePopupPlacesTitleOnBorder(t *testing.T) {
	popupBox := popupBorder.
		Padding(1, 2).
		Width(40).
		Align(lipgloss.Center)

	got := renderFloatingTitlePopup(popupBox, "Confirm", "body", 40)
	if strings.Contains(got, "\nConfirm\n") {
		t.Fatalf("expected popup title to move out of body, got %q", got)
	}
	if !strings.Contains(got, "Confirm") || !strings.Contains(got, "body") {
		t.Fatalf("expected popup helper to keep title and body visible, got %q", got)
	}
}

func TestShellLayoutUsesTwelvePercentMargins(t *testing.T) {
	m := model{
		navigationState: navigationState{
			width:  140,
			height: 60,
		}}
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
	m := model{
		navigationState: navigationState{
			height: 24,
		}}
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
	if got != 10 {
		t.Fatalf("expected full-height graph page size to account for connector lines, got %d", got)
	}
}

func TestRenderAppViewKeepsShellPlacementFullWidth(t *testing.T) {
	m := model{
		navigationState: navigationState{
			width:  140,
			height: 60,
		},
		status: state.New().WithBrowse(),
	}
	got := ansi.Strip(renderAppView(m))
	for _, want := range []string{"Graph Details", "tab: switch", "k/j: updown", "q: quit", "?: hotkeys", "ctrl + u/d: scroll"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected render to contain %q, got %q", want, got)
		}
	}
	for _, want := range []string{"[1] Graph", "[2] Local", "[3] Remote", "[4] Tags"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected render to contain %q, got %q", want, got)
		}
	}
	for _, bad := range []string{"\nGlobal\n", "\nContext\n", "\nContext Actions\n"} {
		if strings.Contains(got, bad) {
			t.Fatalf("expected title %q to move out of body, got %q", bad, got)
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
	rows := graphRows(status)
	m := model{
		navigationState: navigationState{
			height:          80,
			activeSection:   sectionGraph,
			sectionCursor:   map[graphSection]int{sectionGraph: 0},
			graphLaneCursor: 0,
			graphScroll:     0,
		},
		repositoryState: repositoryState{
			repoStatus: status,
		},
		pullState: pullState{}}
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
		navigationState: navigationState{
			activeSection: sectionCurrent,
			sectionCursor: map[graphSection]int{
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch:         "main",
				LocalBranches:  []string{"main", "feature"},
				RemoteBranches: []string{"origin/main", "origin/dev"},
				Tags:           []string{"v1"},
			},
		},
		pullState: pullState{}}
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
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionGraph:   1,
				sectionCurrent: 1,
				sectionRemote:  0,
				sectionTags:    1,
			},
			graphScroll:     2,
			graphLaneCursor: 1,
		},
		repositoryState: repositoryState{
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
		}}
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
	if m.graphLaneCursor != graph.PointerLane(graphRows(rs)[1]) {
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
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionCurrent: 1,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch:          "tmp2",
				LocalBranches:   []string{"tmp2", "tmp1"},
				BranchUpstreams: map[string]string{"tmp2": "origin/tmp2", "tmp1": "origin/tmp1"},
			},
		}}
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
		repositoryState: repositoryState{
			repoStatus: git.Status{
				GraphCommits: []git.GraphCommit{
					{Hash: "c2", Parents: []string{"c1"}},
					{Hash: "c1"},
				},
			},
		},
		status: state.New().WithBrowse()}
	got := m.renderGraphContent(40, 6)
	if lines := strings.Split(got, "\n"); len(lines) != 6 {
		t.Fatalf("expected graph content to fit fixed height, got %d lines: %q", len(lines), got)
	}
}

func TestRenderGraphContentShowsStatusLegendWhenItFits(t *testing.T) {
	m := model{
		repositoryState: repositoryState{
			repoStatus: git.Status{
				GraphCommits: []git.GraphCommit{{Hash: "c1", Subject: "Initial"}},
			},
		},
		status: state.New().WithBrowse()}
	got := ansi.Strip(m.renderGraphContent(80, 4))
	if !strings.Contains(got, "S stash · T tag") {
		t.Fatalf("expected graph page line to include status legend, got %q", got)
	}
	if strings.Contains(got, "?: hotkeys") {
		t.Fatalf("expected graph page line not to repeat the hotkey overlay entrypoint, got %q", got)
	}
	pageLine := strings.Split(got, "\n")[0]
	if !strings.HasSuffix(pageLine, "S stash · T tag") {
		t.Fatalf("expected status legend to be right-aligned, got %q", pageLine)
	}
}

func TestRenderGraphContentHidesLegendBeforeGraphDataOnNarrowWidth(t *testing.T) {
	m := model{
		repositoryState: repositoryState{
			repoStatus: git.Status{
				GraphCommits: []git.GraphCommit{{Hash: "c1", Subject: "Initial"}},
			},
		},
		status: state.New().WithBrowse()}
	got := ansi.Strip(m.renderGraphContent(28, 4))
	if strings.Contains(got, "S stash") {
		t.Fatalf("expected narrow graph page to preserve data before legend, got %q", got)
	}
	if !strings.Contains(got, "graph page 1-1/1") {
		t.Fatalf("expected page indicator to remain visible, got %q", got)
	}
}

func TestRenderGraphProjectionDoesNotRepeatHotkeyHint(t *testing.T) {
	projection := GraphProjection{}
	wide := ansi.Strip(renderGraphProjection(projection, 40, 2))
	if strings.Contains(wide, "?: hotkeys") {
		t.Fatalf("expected empty graph not to show hotkey hint, got %q", wide)
	}
	narrow := ansi.Strip(renderGraphProjection(projection, 24, 2))
	if strings.Contains(narrow, "?: hotkeys") {
		t.Fatalf("expected narrow empty graph not to show hotkey hint, got %q", narrow)
	}
}

func TestRenderGraphContentUsesDateAndLongTitle(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				GraphCommits: []git.GraphCommit{
					{
						Hash:        "c2",
						Subject:     "Merge branch 'main' into develop with a longer title",
						Author:      "alexander",
						Parents:     []string{"c1"},
						Decorations: []string{"HEAD -> main"},
						RelativeAge: "5 minutes ago",
					},
				},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
	got := m.renderGraphContent(88, 6)
	if strings.Contains(got, "date") {
		t.Fatalf("expected graph header to omit date label, got %q", got)
	}
	header := strings.Split(got, "\n")[1]
	if strings.Index(header, "state") > strings.Index(header, "graph") {
		t.Fatalf("expected state column between branches and graph, got %q", header)
	}
	if !strings.Contains(got, "author") {
		t.Fatalf("expected medium graph width to restore author, got %q", got)
	}
	if !strings.Contains(got, "Merge branch") {
		t.Fatalf("expected graph row to prioritize the subject, got %q", got)
	}
	if !strings.Contains(got, "Merge branch") || !strings.Contains(got, "...") {
		t.Fatalf("expected graph title to use the available width with ellipsis, got %q", got)
	}
}

func TestRenderGraphContentHidesAuthorHeaderWhenSpaceIsTight(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				GraphCommits: []git.GraphCommit{
					{
						Hash:        "c2",
						Subject:     "Merge branch 'main' into develop",
						Author:      "hrllk",
						Parents:     []string{"c1"},
						Decorations: []string{"HEAD -> main"},
						RelativeAge: "5 minutes ago",
					},
				},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
	got := m.renderGraphContent(70, 6)
	if strings.Contains(got, "author") {
		t.Fatalf("expected author header to disappear on narrow rows, got %q", got)
	}
	if !strings.Contains(got, "title") {
		t.Fatalf("expected title header to remain visible on narrow rows, got %q", got)
	}
}

func TestRenderGraphContentStartsAtLeftEdge(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				GraphCommits: []git.GraphCommit{
					{Hash: "c2", Parents: []string{"c1"}},
					{Hash: "c1"},
				},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
	got := ansi.Strip(m.renderGraphContent(40, 6))
	lines := strings.Split(got, "\n")
	if len(lines) == 0 {
		t.Fatal("expected graph content to render at least one line")
	}
	if strings.HasPrefix(lines[0], " ") {
		t.Fatalf("expected graph content to start without extra left margin, got %q", lines[0])
	}
}

func TestRenderGraphContentOmitsSelectionArrow(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection:   sectionGraph,
			sectionCursor:   map[graphSection]int{sectionGraph: 0},
			graphLaneCursor: 0,
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				GraphCommits: []git.GraphCommit{
					{Hash: "c2", Parents: []string{"c1"}},
					{Hash: "c1"},
				},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
	got := ansi.Strip(m.renderGraphContent(40, 6))
	if strings.Contains(got, ">") {
		t.Fatalf("expected graph content to omit selection arrow, got %q", got)
	}
}

func TestRenderDetailContentFixedHeight(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:     "/repo",
				Branch:   "main",
				Head:     "abc1234",
				Upstream: "origin/main",
				Remote:   "origin",
				GraphCommits: []git.GraphCommit{
					{
						Hash:        "abc1234",
						Parents:     []string{"def56789abcdef", "9876543210fedcba"},
						Decorations: []string{"HEAD -> main", "origin/main"},
					},
				},
				LocalBranches:      []string{"main"},
				LocalBranchesKnown: true,
				LocalBranchesFresh: true,
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
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
	if !strings.Contains(got, "parent: (multi parent)") || !strings.Contains(got, "  - def56789") || !strings.Contains(got, "  - 98765432") {
		t.Fatalf("expected focus block to include multi-parent list, got %q", got)
	}
	if !strings.Contains(got, "branches:") || !strings.Contains(got, "  - l->main (HEAD)") || strings.Contains(got, "  - o->main") {
		t.Fatalf("expected focus block to include only local branches, got %q", got)
	}
	if strings.Contains(got, "hash:") {
		t.Fatalf("expected hash label to be removed, got %q", got)
	}
}

func TestRenderDetailContentShowsGraphOptionalMetadata(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				GraphCommits: []git.GraphCommit{
					{
						Hash:        "abc1234",
						Parents:     []string{"def5678"},
						Decorations: []string{"HEAD -> main", "feature/topic"},
						Tags:        []string{"v1.0.0", "v1.1.0"},
						Subject:     "commit subject",
					},
				},
				LocalBranches:      []string{"main", "feature/topic"},
				LocalBranchesKnown: true,
				LocalBranchesFresh: true,
			},
			stashByBase: map[string][]git.StashEntry{
				"abc1234": []git.StashEntry{{Ref: "stash@{0}", Subject: "wip", BaseHash: "abc1234"}},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}

	got := ansi.Strip(m.renderDetailContent(60, 14))
	for _, want := range []string{"branches:", "stashes:", "tags:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected graph details to include %q, got %q", want, got)
		}
	}
	if !strings.Contains(got, "v1.0.0") || !strings.Contains(got, "v1.1.0") {
		t.Fatalf("expected graph details to include tag list, got %q", got)
	}
}

func TestCompactDecorationInfoUsesFourteenCharBranchField(t *testing.T) {
	info := compactDecorationInfo([]string{
		"HEAD -> main",
		"origin/main",
		"featureone",
		"featuretwo",
	}, LocalBranchInventory{Names: []string{"main"}, Known: true, Fresh: true})
	got := ansi.Strip(info.Text)
	if width := lipgloss.Width(got); width != graphBranchFieldWidth {
		t.Fatalf("expected branch field width %d, got %d for %q", graphBranchFieldWidth, width, got)
	}
	if !strings.Contains(got, "l->main") {
		t.Fatalf("expected compact branch token to keep the branch name visible, got %q", got)
	}
	if strings.Contains(got, "+2") || strings.Contains(got, " + 2") {
		t.Fatalf("remote-only decorations must not contribute branch overflow, got %q", got)
	}
}

func TestFormatBranchSummaryDecorationUsesLocalAndRemotePrefixes(t *testing.T) {
	tests := []struct {
		name       string
		decoration string
		want       string
		wantOK     bool
	}{
		{name: "head", decoration: "HEAD -> tmp2", want: "l->tmp2 (HEAD)", wantOK: true},
		{name: "local", decoration: "tmp2-3", want: "l->tmp2-3", wantOK: true},
		{name: "remote", decoration: "origin/tmp2", wantOK: false},
		{name: "remote head", decoration: "origin/HEAD -> origin/tmp2", wantOK: false},
		{name: "tag", decoration: "tag: v1.0.0", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := formatBranchSummaryDecoration(tt.decoration, LocalBranchInventory{Names: []string{"tmp2", "tmp2-3"}, Known: true, Fresh: true})
			if ok != tt.wantOK {
				t.Fatalf("expected ok=%t, got %t with %q", tt.wantOK, ok, got)
			}
			if tt.wantOK && got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestRenderContextContentShowsCurrentBranchState(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
			},
		},
		repositoryState: repositoryState{
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
				TrackingKnown: true,
				TrackingFresh: true,
			},
		},
		status: state.New().WithBrowse()}
	m.activeSection = sectionCurrent
	m.status.WorktreeState = state.WorktreeStateDirty

	got := ansi.Strip(m.renderContextContent(80, 16))
	if !strings.Contains(got, "target:") || !strings.Contains(got, "worktree:") {
		t.Fatalf("expected current branch context to show target and worktree, got %q", got)
	}
	if !strings.Contains(got, "upstream:") || !strings.Contains(got, "origin/main") {
		t.Fatalf("expected current branch context to show upstream target, got %q", got)
	}
	for _, want := range []string{"ahead: 2", "behind: 1", "divergence: diverged"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected current branch context to show %q, got %q", want, got)
		}
	}
}

func TestRenderContextContentShowsMissingUpstreamAsNone(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
			sectionCursor: map[graphSection]int{
				sectionCurrent: 0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch:        "main",
				Head:          "abc1234",
				LocalBranches: []string{"main"},
				BranchUpstreams: map[string]string{
					"main": "",
				},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
	got := ansi.Strip(m.renderContextContent(80, 12))
	if !strings.Contains(got, "upstream: (none)") {
		t.Fatalf("expected missing upstream to render as none, got %q", got)
	}
}

func TestRenderContextContentDoesNotTreatMissingTrackingAsEqual(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
			sectionCursor: map[graphSection]int{
				sectionCurrent: 0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch:        "main",
				Head:          "abc1234",
				Upstream:      "origin/main",
				Remote:        "origin",
				LocalBranches: []string{"main"},
				BranchUpstreams: map[string]string{
					"main": "origin/main",
				},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
	got := ansi.Strip(m.renderContextContent(80, 12))
	for _, unwanted := range []string{"ahead: 0", "behind: 0", "divergence: equal"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("missing tracking must not render %q, got %q", unwanted, got)
		}
	}
}

func TestRenderContextContentClipsToWidth(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
			sectionCursor: map[graphSection]int{sectionCurrent: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch:        "main",
				Head:          "abc1234",
				Remote:        "origin",
				WorktreeDirty: true,
				LocalBranches: []string{"main", "feature/this-is-an-extremely-long-branch-name"},
				Tags:          []string{"v1.0.0"},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
	got := m.renderContextContent(28, 18)
	for i, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > 28 {
			t.Fatalf("expected context line %d to fit width, got width=%d line=%q", i, width, line)
		}
	}
}

func TestRenderContextViewportShowsOverflowIndicatorAndScrolls(t *testing.T) {
	lines := []string{"one", "two", "three", "four", "five"}
	top := ansi.Strip(strings.Join(renderContextViewport(lines, 3, 0, 20), "\n"))
	if !strings.Contains(top, "… +3 hidden") {
		t.Fatalf("expected top context viewport to expose hidden count, got %q", top)
	}
	bottom := ansi.Strip(strings.Join(renderContextViewport(lines, 3, 99, 20), "\n"))
	if !strings.Contains(bottom, "… +3 hidden") || !strings.Contains(bottom, "five") {
		t.Fatalf("expected bottom context viewport to expose hidden count and final item, got %q", bottom)
	}
}

func TestRenderContextContentShowsTagTitleAndTarget(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionTags,
			sectionCursor: map[graphSection]int{sectionTags: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				TagEntries: []git.TagEntry{
					{Name: "v1.0.0", CommitHash: "abc1234", RelativeAge: "2 weeks ago", Message: "release title", OriginKnown: true, OnOrigin: true},
				},
				TagProvenanceLoaded: true,
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}

	got := ansi.Strip(m.renderContextContent(80, 18))
	for _, want := range []string{"title: v1.0.0", "target: abc1234", "age: 2w", "message: release title", "provenance: (origin)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected tag detail to show %q, got %q", want, got)
		}
	}
}

func TestRenderDetailContentClipsToWidth(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
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
				LocalBranches:      []string{"main"},
				LocalBranchesKnown: true,
				LocalBranchesFresh: true,
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
	got := m.renderDetailContent(28, 18)
	for i, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > 28 {
			t.Fatalf("expected detail line %d to fit width, got width=%d line=%q", i, width, line)
		}
	}
}

func TestRenderContextContentSplitsInfoAndActions(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
			},
		},
		repositoryState: repositoryState{
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
		},
		status: state.New().WithBrowse()}
	m.activeSection = sectionCurrent
	m.status.WorktreeState = state.WorktreeStateDirty

	got := ansi.Strip(m.renderContextContent(40, 12))
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
	if !strings.Contains(got, "Local Details") || !strings.Contains(got, "Local Actions") {
		t.Fatalf("expected local header to prefix details/actions with the focused section name, got %q", got)
	}
	if !strings.Contains(got, "• space: checkout") || !strings.Contains(got, "• p: pull") {
		t.Fatalf("expected right actions column to include browse actions, got %q", got)
	}
}

func TestRenderContextContentAddsGraphActionsLeftMargin(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				GraphCommits: []git.GraphCommit{
					{Hash: "abc1234", Subject: "Commit 1"},
				},
			},
		},
		status: state.New().WithBrowse()}
	m.activeSection = sectionGraph

	got := ansi.Strip(m.renderContextContent(50, 12))
	if !strings.Contains(got, "│ Graph Actions") {
		t.Fatalf("expected graph actions column to keep a left margin, got %q", got)
	}
}

func TestRenderContextContentKeepsSplitLayoutOnNarrowWidth(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
			},
		},
		repositoryState: repositoryState{
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
		},
		status: state.New().WithBrowse()}
	m.activeSection = sectionGraph

	got := ansi.Strip(m.renderContextContent(22, 8))
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
		{name: "current", active: sectionCurrent, wantPrefix: "Local"},
		{name: "graph", active: sectionGraph, wantPrefix: "Graph"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				navigationState: navigationState{
					sectionCursor: map[graphSection]int{
						sectionGraph:   0,
						sectionCurrent: 0,
					},
				},
				repositoryState: repositoryState{
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
				},
				status: state.New().WithBrowse()}
			m.activeSection = tt.active

			got := ansi.Strip(m.renderContextContent(60, 12))
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
				TagEntries: []git.TagEntry{{Name: "v1.0.0", CommitHash: "abc1234", Message: "release"}},
				Tags:       []string{"v1.0.0"},
			},
			wantInfo:    "title:",
			wantActions: "enter: jump to graph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				navigationState: navigationState{
					sectionCursor: map[graphSection]int{
						sectionGraph:   0,
						sectionCurrent: 0,
						sectionRemote:  0,
						sectionTags:    0,
					},
				},
				repositoryState: repositoryState{
					repoStatus: tt.repoStatus,
				},
				status: state.New().WithBrowse()}
			m.activeSection = tt.active
			got := ansi.Strip(m.renderContextContent(48, 10))
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
		navigationState: navigationState{
			width:  140,
			height: 60,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
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
		},
		status: state.New().WithBrowse()}

	got := renderAppView(m)
	for _, want := range []string{"Graph Details", "Graph", "Local", "Remote", "Tags"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected view to contain %q, got %q", want, got)
		}
	}

	localIdx := strings.Index(got, "Local")
	remoteIdx := strings.Index(got, "Remote")
	tagsIdx := strings.Index(got, "Tags")
	if localIdx < 0 || remoteIdx < 0 || tagsIdx < 0 {
		t.Fatalf("expected right rail sections to appear in output, got %q", got)
	}
	if !(localIdx < remoteIdx && remoteIdx < tagsIdx) {
		t.Fatalf("expected Local / Remote / Tags to stack in order, got %d / %d / %d", localIdx, remoteIdx, tagsIdx)
	}
}

func TestRenderGlobalContentUsesNewDigitMapping(t *testing.T) {
	m := model{
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch: "main",
				Head:   "abc1234",
				Remote: "origin",
			},
		},
		status: state.New().WithBrowse()}
	got := ansi.Strip(m.renderGlobalContent(40, 14))
	for _, want := range []string{"Mode: Browse", "Actions", "tab: next section", "shift+tab: previous section", "j/k: move", "f: fetch", "F: fetch tags", "S: stash list", "q: quit", "?: show hidden hotkeys"} {
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
		navigationState: navigationState{
			width:  120,
			height: 40,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:     "/repo",
				Branch:   "main",
				Head:     "abc1234",
				Upstream: "origin/main",
				Remote:   "origin",
			},
		},
		status: loadingToast("Fetching upstream...")}

	got := renderAppView(m)
	if strings.Contains(got, "Mode: Loading") || strings.Contains(got, "Loading | Fetching upstream...") {
		t.Fatalf("expected loading state to stay out of the Global panel, got %q", got)
	}
	if !strings.Contains(got, "Working...") || !strings.Contains(got, "Fetching upstream...") {
		t.Fatalf("expected loading toast overlay, got %q", got)
	}
}

func TestRenderBlockedShowsAlertOverlay(t *testing.T) {
	m := model{
		navigationState: navigationState{
			width:  120,
			height: 40,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:   "/repo",
				Branch: "main",
				Head:   "abc1234",
			},
		},
		status: state.New().WithBlocked(state.BlockUnknown, "Select a local branch.", "Move to a branch line.")}

	got := renderAppView(m)
	if strings.Contains(got, "Mode: Blocked") || strings.Contains(got, "Blocked | Select a local branch.") {
		t.Fatalf("expected blocked state to stay out of the Global panel, got %q", got)
	}
	if !strings.Contains(got, "Alert") || !strings.Contains(got, "Select a local branch.") || !strings.Contains(got, "Move to a branch line.") {
		t.Fatalf("expected blocked alert overlay, got %q", got)
	}
	if !strings.Contains(got, "enter: dismiss") || !strings.Contains(got, "q: close") {
		t.Fatalf("expected blocked alert dismiss help, got %q", got)
	}
}

func TestRenderFastForwardConfirmShowsConciseHelp(t *testing.T) {
	m := model{
		navigationState: navigationState{
			width:  120,
			height: 40,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:   "/repo",
				Branch: "main",
				Head:   "abc1234",
			},
		},
		status: state.New().WithConfirm(state.ActionMerge, "Fast-forward available.", "HEAD can move to feature.")}
	m.status.Selected = "feature"

	got := renderAppView(m)
	if strings.Contains(got, "Current:") || strings.Contains(got, "Target:") {
		t.Fatalf("expected fast-forward popup to omit count detail, got %q", got)
	}
	if !strings.Contains(got, "enter: fast-forward") || !strings.Contains(got, "q: close") {
		t.Fatalf("expected fast-forward confirm help, got %q", got)
	}
}

func TestRefreshedMsgDoesNotClearBlockedAlert(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:   "/repo",
				Branch: "main",
				Head:   "abc1234",
			},
		},
		pullState: pullState{},
		status:    state.New().WithBlocked(state.BlockUnknown, "Select a local branch.", "Move to a branch line.")}

	gotModel, _ := m.Update(refreshedMsg{status: git.Status{Root: "/repo", Branch: "main", Head: "def5678"}})
	got := gotModel.(model)
	if got.status.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode to persist across refresh, got %s", got.status.Mode)
	}
	if got.status.Message != "Select a local branch." {
		t.Fatalf("expected blocked message to persist, got %q", got.status.Message)
	}
}

func TestFetchedMsgDoesNotClearBlockedAlert(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:   "/repo",
				Branch: "main",
				Head:   "abc1234",
			},
		},
		pullState: pullState{},
		status:    state.New().WithBlocked(state.BlockUnknown, "Select a local branch.", "Move to a branch line.")}

	gotModel, cmd := m.Update(fetchedMsg{status: git.Status{Root: "/repo", Branch: "main", Head: "def5678"}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected fetched update to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode to persist across fetch, got %s", got.status.Mode)
	}
	if got.status.Message != "Select a local branch." {
		t.Fatalf("expected blocked message to persist, got %q", got.status.Message)
	}
}

func TestRenderBranchOpenShowsCenteredPopupOverlay(t *testing.T) {
	m := model{
		navigationState: navigationState{
			width:  120,
			height: 40,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:   "/repo",
				Branch: "main",
				Head:   "abc1234",
			},
		},
		pullState:    pullState{},
		overlayState: overlayState{branchOpen: true, branchDraft: "feature/new-flow", branchBase: "abc1234", branchError: "Branch name already exists."},

		status: loadingToast("Enter a branch name.")}

	got := renderAppView(m)
	if strings.Contains(got, "Mode: Loading") || strings.Contains(got, "Loading | Enter a branch name.") {
		t.Fatalf("expected branch input to stay out of the Global panel, got %q", got)
	}
	for _, want := range []string{"Create branch", "Enter a branch name.", "name: feature/new-flow", "base: abc1234", "Branch name already exists.", "q: close"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected branch popup to contain %q, got %q", want, got)
		}
	}
}

func TestPullFetchWithoutIncomingCommitsShowsVerifiedResultSummary(t *testing.T) {
	fixture := newCommandRepo(t)
	repoStatus := git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		Upstream:      "origin/main",
		Remote:        "origin",
		LocalBranches: []string{"main"},
		Tracking:      map[string]git.BranchTracking{"main": {}},
		TrackingKnown: true,
		TrackingFresh: true,
	}
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus:      repoStatus,
			repositoryEpoch: 3,
		},
		pullState: pullState{
			activePullRequest: &pullRequest{
				ID:                9,
				Epoch:             3,
				OperationBaseline: pullSnapshotIdentity(repoStatus, 3),
			},
		},
		status: state.New().WithBrowse()}

	gotModel, cmd := m.Update(pullFetchedMsg{
		status:               repoStatus,
		requestID:            9,
		requestEpoch:         3,
		baseline:             pullSnapshotIdentity(repoStatus, 3),
		operationBaseline:    pullSnapshotIdentity(repoStatus, 3),
		operationBaselineSet: true,
	})
	got := gotModel.(model)
	if got.status.Mode != state.ModeOperationResult {
		t.Fatalf("expected operation result mode, got %s", got.status.Mode)
	}
	if got.status.Message != "PULL COMPLETED" {
		t.Fatalf("expected result summary message, got %q", got.status.Message)
	}
	if got.operationResult == nil || !got.operationResult.NoOp {
		t.Fatalf("expected verified no-op result, got %+v", got.operationResult)
	}
	if cmd != nil {
		t.Fatalf("expected no dismissal command for result summary")
	}
}

func TestZeroIdentityPullFetchedMessageIsIgnored(t *testing.T) {
	original := git.Status{Root: "/repo", Branch: "main", Head: "old"}
	m := model{
		repositoryState: repositoryState{repoStatus: original},
		pullState:       pullState{}, status: state.New().WithBrowse()}

	next, cmd := m.Update(pullFetchedMsg{status: git.Status{Root: "/repo", Branch: "other", Head: "new"}})
	got := next.(model)
	if cmd != nil {
		t.Fatalf("expected zero-identity pull message to be ignored, got command %v", cmd)
	}
	if !reflect.DeepEqual(got.repoStatus, original) {
		t.Fatalf("zero-identity pull message mutated repo status: %+v", got.repoStatus)
	}
	if got.activePullRequest != nil {
		t.Fatalf("zero-identity pull message created an active request: %+v", got.activePullRequest)
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
	if got := sectionName(sectionCurrent); got != "Local" {
		t.Fatalf("expected sectionCurrent to be labeled Local, got %q", got)
	}
}

func TestRenderAppViewUsesOuterMargins(t *testing.T) {
	m := model{
		navigationState: navigationState{
			width:  140,
			height: 60,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
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
		},
		status: state.New().WithBrowse()}

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
		navigationState: navigationState{
			width:  120,
			height: 24,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
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
		},
		status: state.New().WithBrowse()}

	got := renderAppView(m)
	if !strings.Contains(got, "Graph") || !strings.Contains(got, "Graph Details") {
		t.Fatalf("expected compact render to keep Graph and Details visible, got %q", got)
	}
}

func TestRenderActionHelpLinesAreSectionSpecific(t *testing.T) {
	graph := renderActionHelpLines(model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph: 0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				GraphCommits: []git.GraphCommit{{
					Graph:       "*",
					Hash:        "abc1234",
					Parents:     []string{"def5678"},
					Decorations: []string{"HEAD -> main"},
				}},
				LocalBranches: []string{"main"},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()})
	graphJoined := ansi.Strip(strings.Join(graph, " "))
	if len(graph) != 4 {
		t.Fatalf("expected graph actions to stay at four visible lines, got %v", graph)
	}
	for _, want := range []string{"m: merge", "r: rebase", "space: checkout", "H: jump to HEAD"} {
		if !strings.Contains(graphJoined, want) {
			t.Fatalf("expected graph actions to include %q, got %v", want, graph)
		}
	}
	for _, hidden := range []string{"s: reset", "d: delete branch", "o: pop stash", "gg: top", "G: bottom", "ctrl+u/d: scroll"} {
		if strings.Contains(graphJoined, hidden) {
			t.Fatalf("expected graph actions to hide %q, got %v", hidden, graph)
		}
	}
	current := renderActionHelpLines(model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
			sectionCursor: map[graphSection]int{
				sectionCurrent: 0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch:        "main",
				LocalBranches: []string{"main", "feature"},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()})
	currentJoined := ansi.Strip(strings.Join(current, " "))
	if !strings.Contains(currentJoined, "d: delete branch") {
		t.Fatalf("expected current actions to include delete branch, got %v", current)
	}
	if !strings.Contains(currentJoined, "(current branch)") {
		t.Fatalf("expected current branch delete to be disabled, got %v", current)
	}

	remote := renderActionHelpLines(model{
		navigationState: navigationState{
			activeSection: sectionRemote,
		},
		status: state.New().WithBrowse(),
	})
	if !containsLine(strings.Split(ansi.Strip(strings.Join(remote, "\n")), "\n"), "• space: checkout") {
		t.Fatalf("expected remote actions to include checkout, got %v", remote)
	}
	remoteJoined := ansi.Strip(strings.Join(remote, " "))
	if strings.Contains(remoteJoined, "m: merge") || strings.Contains(remoteJoined, "s: reset") {
		t.Fatalf("expected remote actions to exclude graph-only actions, got %v", remote)
	}
}

func TestHiddenHotkeysPopupShowsMovedAndConditionalActions(t *testing.T) {
	got := ansi.Strip(renderHiddenHotkeysPopup(model{
		navigationState: navigationState{
			activeSection: sectionGraph,
		},
		status: state.New().WithBrowse(),
	}, 90, 0))
	for _, want := range []string{
		"Hidden hotkeys by section",
		"focus: Graph",
		"Graph",
		"Visible:",
		"m: merge",
		"Conditional:",
		"s: reset",
		hiddenHotkeyPopupFooter,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected hidden hotkeys popup to contain %q, got %q", want, got)
		}
	}
}

func TestHiddenHotkeysPopupCentersHeaderAndFooter(t *testing.T) {
	got := ansi.Strip(renderHiddenHotkeysPopup(model{
		navigationState: navigationState{
			activeSection: sectionGraph,
		},
		status: state.New().WithBrowse(),
	}, 90, 0))
	lines := strings.Split(got, "\n")
	findLine := func(want string) string {
		for _, line := range lines {
			if strings.Contains(line, want) {
				return line
			}
		}
		return ""
	}

	headerLine := findLine("Hidden hotkeys by section")
	if headerLine == "" {
		t.Fatal("expected centered header line to be present")
	}
	headerBody := strings.TrimPrefix(headerLine, "│")
	headerBody = strings.TrimSuffix(headerBody, "│")
	if leading := len(headerBody) - len(strings.TrimLeft(headerBody, " ")); leading <= 2 {
		t.Fatalf("expected header to be centered, got %q", headerLine)
	}

	footerLine := findLine("q: close")
	if footerLine == "" {
		t.Fatal("expected centered footer line to be present")
	}
	footerBody := strings.TrimPrefix(footerLine, "│")
	footerBody = strings.TrimSuffix(footerBody, "│")
	if leading := len(footerBody) - len(strings.TrimLeft(footerBody, " ")); leading <= 2 {
		t.Fatalf("expected footer to be centered, got %q", footerLine)
	}
}

func TestRenderHiddenHotkeySectionTitleHighlightsActiveSection(t *testing.T) {
	active := renderHiddenHotkeySectionTitle("Graph", true)
	if want := sectionTitle.Render("› Graph"); active != want {
		t.Fatalf("expected active hidden hotkey section title to use a subtle marker, got %q", active)
	}

	inactive := renderHiddenHotkeySectionTitle("Local", false)
	if want := muted.Render("  Local"); inactive != want {
		t.Fatalf("expected inactive hidden hotkey section title to be muted, got %q", inactive)
	}
}

func TestRenderCherryPickPopupShowsQueueOrderAndHelp(t *testing.T) {
	got := ansi.Strip(renderCherryPickPopup(model{
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch: "main",
				Head:   "other123",
				GraphCommits: []git.GraphCommit{
					{Hash: "pick123", Parents: []string{"root"}, Author: "Ada Lovelace", Subject: "first change", RelativeAge: "2 days ago"},
					{Hash: "other123", Parents: []string{"pick123"}, Author: "Grace Hopper", Subject: "second change", RelativeAge: "1 day ago"},
				},
			},
		},
		status: state.Status{
			Mode: state.ModeCherryPickPick,
			Targets: []state.TargetItem{
				{Ref: "pick123", CommitHash: "pick123", Author: "Ada Lovelace", Subject: "first change", RelativeAge: "2 days ago"},
				{Ref: "other123", CommitHash: "other123", Author: "Grace Hopper", Subject: "second change", RelativeAge: "1 day ago"},
			},
			TargetIdx:     1,
			SelectedQueue: []string{"pick123"},
		}}, 90))
	for _, want := range []string{
		"Cherry-pick Targets",
		"destination: current main",
		"selected: 1 / 2",
		"target   hash      author   age      title",
		"[x] ",
		"[ ] ",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected cherry-pick popup to contain %q, got %q", want, got)
		}
	}
}

func TestRenderActionHelpLinesShowsCleanupActionsOnlyForDirtyCurrentSection(t *testing.T) {
	dirty := renderActionHelpLines(model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
			sectionCursor: map[graphSection]int{
				sectionCurrent: 0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch:        "main",
				LocalBranches: []string{"main"},
				WorktreeDirty: true,
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()})
	dirtyJoined := ansi.Strip(strings.Join(dirty, " "))
	if !strings.Contains(dirtyJoined, "s: stash changes") || !strings.Contains(dirtyJoined, "c: clean working tree") {
		t.Fatalf("expected dirty local actions to include cleanup shortcuts, got %v", dirty)
	}
	if strings.Contains(dirtyJoined, "dirty only") {
		t.Fatalf("expected dirty local actions to be enabled, got %v", dirty)
	}

	clean := renderActionHelpLines(model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
			sectionCursor: map[graphSection]int{
				sectionCurrent: 0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch:        "main",
				LocalBranches: []string{"main"},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()})
	cleanJoined := ansi.Strip(strings.Join(clean, " "))
	if !strings.Contains(cleanJoined, "dirty only") {
		t.Fatalf("expected clean local actions to show dirty-only gating, got %v", clean)
	}
}

func TestRenderTagActionHelpLinesIncludesRemoteDelete(t *testing.T) {
	got := ansi.Strip(strings.Join(renderTagActionHelpLines(model{
		status: state.New().WithBrowse(),
	}), " "))
	if !strings.Contains(got, "enter: jump to graph") {
		t.Fatalf("expected tag actions to keep graph jump shortcut, got %q", got)
	}
	if !strings.Contains(got, "d: delete tag") {
		t.Fatalf("expected tag actions to keep local delete shortcut, got %q", got)
	}
	if !strings.Contains(got, "D: delete remote tag") {
		t.Fatalf("expected tag actions to show remote delete shortcut, got %q", got)
	}
}

func TestRenderGraphStashPopPopupShowsPickerAndConfirmStates(t *testing.T) {
	forceTrueColorProfile(t)
	picker := model{
		overlayState: overlayState{graphStashPopOpen: true, graphStashPopMode: graphStashPopModePicker, graphStashPopEntries: []git.StashEntry{
			{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "abc1234", Subject: "latest change"},
			{Ref: "stash@{1}", Hash: "stashhash1", BaseHash: "abc1234", Subject: "older change"},
		}},
	}
	gotPicker := ansi.Strip(renderGraphStashPopPopup(picker, 90, 24))
	if !strings.Contains(gotPicker, "Pop stash") {
		t.Fatalf("expected pop stash title, got %q", gotPicker)
	}
	if !strings.Contains(gotPicker, "Choose a stash to pop.") {
		t.Fatalf("expected picker description, got %q", gotPicker)
	}
	if !strings.Contains(gotPicker, "stash@{0}") || !strings.Contains(gotPicker, "stash@{1}") {
		t.Fatalf("expected picker rows, got %q", gotPicker)
	}
	if !strings.Contains(gotPicker, "enter: choose") {
		t.Fatalf("expected picker footer, got %q", gotPicker)
	}

	confirm := model{
		overlayState: overlayState{graphStashPopOpen: true, graphStashPopMode: graphStashPopModeConfirm, graphStashPopEntries: []git.StashEntry{
			{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "abc1234", Subject: "latest change"},
		}},
	}
	gotConfirm := ansi.Strip(renderGraphStashPopPopup(confirm, 90, 24))
	if !strings.Contains(gotConfirm, "Confirm stash pop.") {
		t.Fatalf("expected confirm description, got %q", gotConfirm)
	}
	if !strings.Contains(gotConfirm, "enter: pop") {
		t.Fatalf("expected confirm footer, got %q", gotConfirm)
	}
	if !strings.Contains(gotConfirm, "This will remove the stash") {
		t.Fatalf("expected confirm warning, got %q", gotConfirm)
	}
}

func TestRTriggersRebaseOnlyInGraphSection(t *testing.T) {
	// With no graph rows / no local lane -> 'r' should block with error message
	graph := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
		},
		status:      state.New().WithBrowse(),
		commitLimit: 0,
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
		navigationState: navigationState{
			activeSection: sectionCurrent,
		},
		status:      state.New().WithBrowse(),
		commitLimit: 0,
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
		navigationState: navigationState{
			activeSection: sectionGraph,
		},
		status:      state.New().WithBrowse(),
		commitLimit: 0,
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected fetch key to trigger background refresh")
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected fetch to enter loading mode, got %s", got.status.Mode)
	}
	if got.status.Message != "Fetching sources..." {
		t.Fatalf("expected fetch message to be visible, got %q", got.status.Message)
	}
}

func TestFetchKeyWorksFromAnyBrowseSection(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
		},
		status:      state.New().WithBrowse(),
		commitLimit: 0,
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected fetch key to trigger refresh outside graph section")
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected fetch to enter loading mode, got %s", got.status.Mode)
	}
	if got.status.Message != "Fetching sources..." {
		t.Fatalf("expected fetch message to be visible, got %q", got.status.Message)
	}
}

func TestFetchTagsKeyEntersLoadingMode(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
		},
		status:      state.New().WithBrowse(),
		commitLimit: 0,
	}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'F'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected fetch tags key to trigger background refresh")
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected fetch tags to enter loading mode, got %s", got.status.Mode)
	}
	if got.status.Message != "Fetching tags..." {
		t.Fatalf("expected fetch tags message to be visible, got %q", got.status.Message)
	}
}

func TestNumberKeysSwitchSections(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
		},
		status:      state.New().WithBrowse(),
		commitLimit: 0,
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

func TestSpaceChecksOutFromGraphSection(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
			graphLaneCursor: 0,
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:          "/repo",
				Branch:        "main",
				Head:          "head",
				LocalBranches: []string{"main"},
				GraphCommits: []git.GraphCommit{{
					Graph:       "*",
					Hash:        "head",
					Decorations: []string{"HEAD -> main"},
				}},
			},
		},
		pullState:   pullState{},
		status:      state.New().WithBrowse(),
		commitLimit: 0}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected space checkout to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionCheckout {
		t.Fatalf("expected checkout action, got %s", got.status.Action)
	}
	if got.status.Selected != "main" {
		t.Fatalf("expected graph checkout target to be stored, got %q", got.status.Selected)
	}
}

func TestSpaceChecksOutFromRemoteSection(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionRemote,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
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
		},
		pullState:   pullState{},
		status:      state.New().WithBrowse(),
		commitLimit: 0}
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
		navigationState: navigationState{
			activeSection: sectionRemote,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:           "/repo",
				Branch:         "main",
				Head:           "head",
				RemoteBranches: []string{"origin/main"},
				LocalBranches:  []string{"main"},
				Remote:         "origin",
				HasCommits:     true,
			},
		},
		pullState:   pullState{},
		status:      state.New().WithBrowse(),
		commitLimit: 0}
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
		navigationState: navigationState{
			activeSection: sectionRemote,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				RemoteBranches: []string{"origin", "origin/HEAD", "origin/main"},
				LocalBranches:  []string{"main"},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}

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

func TestRenderSectionContentStartsAtLeftEdge(t *testing.T) {
	tests := []struct {
		name    string
		section graphSection
		repo    git.Status
	}{
		{
			name:    "current",
			section: sectionCurrent,
			repo:    git.Status{LocalBranches: []string{"main"}},
		},
		{
			name:    "remote",
			section: sectionRemote,
			repo:    git.Status{RemoteBranches: []string{"origin/main"}},
		},
		{
			name:    "tags",
			section: sectionTags,
			repo:    git.Status{Tags: []string{"v1.0.0"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				navigationState: navigationState{
					sectionCursor: map[graphSection]int{
						sectionGraph:   0,
						sectionCurrent: 0,
						sectionRemote:  0,
						sectionTags:    0,
					},
					activeSection: sectionGraph,
				},
				repositoryState: repositoryState{
					repoStatus: tt.repo,
				},
				status: state.New().WithBrowse()}
			got := ansi.Strip(m.renderSectionContent(tt.section, 40, 4))
			lines := strings.Split(got, "\n")
			if len(lines) == 0 {
				t.Fatal("expected section content to render at least one line")
			}
			if strings.HasPrefix(lines[0], " ") {
				t.Fatalf("expected section content to start without extra left margin, got %q", lines[0])
			}
		})
	}
}

func TestRenderSectionContentKeepsActiveCursorVisible(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 5,
				sectionRemote:  0,
				sectionTags:    0,
			},
			activeSection: sectionCurrent,
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				LocalBranches: []string{"main", "feature", "tmp1", "tmp2", "tmp2-2", "tmp2-3"},
			},
		},
		status: state.New().WithBrowse()}
	got := ansi.Strip(m.renderSectionContent(sectionCurrent, 30, 4))
	if !strings.Contains(got, "tmp2-3") {
		t.Fatalf("expected active cursor item to remain visible, got %q", got)
	}
	if strings.Contains(got, ">l->") {
		t.Fatalf("expected section content to omit selection arrows, got %q", got)
	}
	rendered := m.renderSectionContent(sectionCurrent, 30, 4)
	if !strings.Contains(rendered, branchMark.Render("l->tmp2-3")) {
		t.Fatalf("expected active cursor item to use branch highlight, got %q", rendered)
	}
	if strings.Contains(got, "main\n") && strings.Contains(got, "tmp2-3") {
		t.Fatalf("expected section view to window toward the cursor, got %q", got)
	}
}

func TestRenderSectionContentShowsInactiveOverflowCount(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				LocalBranches: []string{"main", "feature", "release", "hotfix"},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
	got := ansi.Strip(m.renderSectionContent(sectionCurrent, 30, 3))
	if !strings.Contains(got, "… +2") {
		t.Fatalf("expected inactive section to show hidden item count, got %q", got)
	}
}

func TestRenderTargetPickPopupHighlightsSelectionWithoutArrow(t *testing.T) {
	status := state.New().WithTargetPick(state.ActionCheckout, []state.TargetItem{
		{Kind: state.TargetKindLocal, Name: "main", Ref: "main"},
		{Kind: state.TargetKindLocal, Name: "feature", Ref: "feature"},
	})
	status.TargetIdx = 1
	status.Selected = "feature"
	m := model{
		status: status,
	}
	got := ansi.Strip(renderTargetPickPopup(m, 80))
	if strings.Contains(got, ">l->") {
		t.Fatalf("expected target picker to omit selection arrows, got %q", got)
	}
	if !strings.Contains(renderTargetPickPopup(m, 80), branchMark.Render("l->feature")) {
		t.Fatalf("expected target picker to highlight the selected item, got %q", got)
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
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		status:      state.New().WithBrowse(),
		commitLimit: 0,
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

func TestRefreshedMsgPreservesCachedTagEntries(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			tagEntries: []git.TagEntry{
				{Name: "v1.0.0", CommitHash: "abc1234", Subject: "initial", RelativeAge: "2 days ago", OnOrigin: true},
			},
		},
		status: state.New().WithBrowse()}

	gotModel, _ := m.Update(refreshedMsg{status: git.Status{Root: "/repo", Branch: "main", Head: "def5678"}})
	got := gotModel.(model)
	if len(got.repoStatus.TagEntries) != 1 {
		t.Fatalf("expected cached tag entries to survive refresh, got %+v", got.repoStatus.TagEntries)
	}
	if got.repoStatus.TagEntries[0].Name != "v1.0.0" || !got.repoStatus.TagEntries[0].OnOrigin {
		t.Fatalf("unexpected cached tag entry after refresh: %+v", got.repoStatus.TagEntries[0])
	}
}

func TestCheckoutFocusesGraphHeadRow(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection:   sectionCurrent,
			graphScroll:     12,
			graphLaneCursor: 3,
			sectionCursor: map[graphSection]int{
				sectionGraph:   15,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		commitLimit: 0,
	}
	status := git.Status{
		Branch:        "tmp1",
		Head:          "head",
		LocalBranches: []string{"tmp1"},
		GraphCommits: []git.GraphCommit{
			{Hash: "base"},
			{Hash: "head", Parents: []string{"base"}, Decorations: []string{"HEAD -> tmp1", "tmp1"}},
		},
	}
	gotModel, _ := m.Update(executedMsg{action: state.ActionCheckout, target: "tmp1", status: status})
	got := gotModel.(model)
	if got.commitLimit != 0 {
		t.Fatalf("expected checkout to reset graph load limit to unlimited, got %d", got.commitLimit)
	}
	if got.activeSection != sectionGraph {
		t.Fatalf("expected checkout to focus graph section, got %v", got.activeSection)
	}
	rows := graphRows(status)
	headRow := findGraphRowByHash(rows, status.Head)
	if headRow < 0 {
		t.Fatalf("expected head row for %q", status.Head)
	}
	if got.sectionCursor[sectionGraph] != headRow {
		t.Fatalf("expected checkout to focus head row %d, got %d", headRow, got.sectionCursor[sectionGraph])
	}
	wantPage := graphPageSizeForRows(&m, rows, headRow, graphContentHeightForModel(&m))
	if got.graphScroll != clampScroll(headRow, len(rows), wantPage) {
		t.Fatalf("expected checkout to clamp scroll to head row, got %d", got.graphScroll)
	}
	if got.graphLaneCursor != graph.PointerLane(rows[headRow]) {
		t.Fatalf("expected checkout to align lane cursor to head row, got %d want %d", got.graphLaneCursor, graph.PointerLane(rows[headRow]))
	}
}

func TestBranchCreateFocusesGraphHeadRow(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection:   sectionCurrent,
			graphScroll:     9,
			graphLaneCursor: 2,
			sectionCursor: map[graphSection]int{
				sectionGraph:   6,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		status: loadingToast("Branch created."),
	}
	status := git.Status{
		Branch:        "feature/new-flow",
		Head:          "head",
		LocalBranches: []string{"feature/new-flow"},
		GraphCommits: []git.GraphCommit{
			{Hash: "base"},
			{Hash: "head", Parents: []string{"base"}, Decorations: []string{"HEAD -> feature/new-flow", "feature/new-flow"}},
		},
	}
	gotModel, _ := m.Update(createdBranchMsg{name: "feature/new-flow", base: "base", status: status})
	got := gotModel.(model)
	if got.activeSection != sectionGraph {
		t.Fatalf("expected branch create to focus graph section, got %v", got.activeSection)
	}
	rows := graphRows(status)
	headRow := findGraphRowByHash(rows, status.Head)
	if headRow < 0 {
		t.Fatalf("expected head row for %q", status.Head)
	}
	if got.sectionCursor[sectionGraph] != headRow {
		t.Fatalf("expected branch create to focus head row %d, got %d", headRow, got.sectionCursor[sectionGraph])
	}
	if got.graphLaneCursor != graph.PointerLane(rows[headRow]) {
		t.Fatalf("expected branch create to align lane cursor to head row, got %d want %d", got.graphLaneCursor, graph.PointerLane(rows[headRow]))
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
	got := formatCompactDecorations([]string{"HEAD -> main", "develop", "origin/main", "tag: v1.0.0"}, LocalBranchInventory{Names: []string{"main", "develop"}, Known: true, Fresh: true})
	if !strings.HasPrefix(got, "o/l->") || !strings.Contains(got, "+1") || strings.Contains(got, " + 1") {
		t.Fatalf("expected a single compact branch token with compact overflow count, got %q", got)
	}
	if len([]rune(got)) != graphBranchFieldWidth {
		t.Fatalf("expected compact decorations to use %d chars, got %q", graphBranchFieldWidth, got)
	}
}

func TestFormatCompactDecorationsUsesHeadThenAlphabeticalWithOverflow(t *testing.T) {
	got := formatCompactDecorations([]string{"HEAD -> main", "develop", "release"}, LocalBranchInventory{Names: []string{"main", "develop", "release"}, Known: true, Fresh: true})
	if !strings.HasPrefix(got, "l->main") || !strings.Contains(got, "+2") || strings.Contains(got, " + 2") {
		t.Fatalf("expected HEAD branch to lead with compact overflow count, got %q", got)
	}

	got = formatCompactDecorations([]string{"release", "develop", "main"}, LocalBranchInventory{Names: []string{"main", "develop", "release"}, Known: true, Fresh: true})
	if !strings.HasPrefix(got, "l->") {
		t.Fatalf("expected alphabetical local branch fallback, got %q", got)
	}

	got = formatCompactDecorations([]string{"origin/release", "origin/develop"}, LocalBranchInventory{Known: false, Fresh: false})
	if got != "-" {
		t.Fatalf("expected remote-only decorations to be hidden, got %q", got)
	}
}

func TestFormatCompactDecorationsKeepsOverflowCountForLongBranches(t *testing.T) {
	got := formatCompactDecorations(
		[]string{"HEAD -> feature/very-long-branch-name", "develop", "release"},
		LocalBranchInventory{Names: []string{"feature/very-long-branch-name", "develop", "release"}, Known: true, Fresh: true},
	)
	if !strings.Contains(got, "+2") || strings.Contains(got, " + 2") {
		t.Fatalf("expected compact overflow count to survive long branch names, got %q", got)
	}
	if len([]rune(got)) != graphBranchFieldWidth {
		t.Fatalf("expected compact decorations to stay within %d chars, got %q", graphBranchFieldWidth, got)
	}
}

func TestCompactWhenTextUsesShortUnitLabels(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "1 second ago", want: "1s"},
		{in: "2 minutes ago", want: "2m"},
		{in: "3 hours ago", want: "3h"},
		{in: "4 days ago", want: "4d"},
		{in: "5 weeks ago", want: "5w"},
		{in: "6 months ago", want: "6m"},
		{in: "7 years ago", want: "7y"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := compactWhenText(tt.in); got != tt.want {
				t.Fatalf("compactWhenText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCompactAuthorTextUsesSevenCharBudget(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: "-"},
		{in: "hrllk", want: "hrllk"},
		{in: "alexander", want: "alexa.."},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := compactAuthorText(tt.in); got != tt.want {
				t.Fatalf("compactAuthorText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
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
	if graph.RowWidth(rows[0]) < 1 || graph.RowWidth(rows[1]) < 2 || graph.RowWidth(rows[2]) < 2 {
		t.Fatalf("expected sibling rows to expand as new tips appear, got widths %d, %d, %d", graph.RowWidth(rows[0]), graph.RowWidth(rows[1]), graph.RowWidth(rows[2]))
	}
	if len(rows[3].Children) != 3 {
		t.Fatalf("expected branch point commit to know all children, got %d", len(rows[3].Children))
	}
}

func TestGraphRowsUsesRawGraphPrefixWhenAvailable(t *testing.T) {
	rows := graphRows(git.Status{
		GraphCommits: []git.GraphCommit{
			{Graph: "*   ", Hash: "head", RelativeAge: "5 minutes ago", Author: "alexander", Subject: "Merge branch 'main' into develop", Decorations: []string{"HEAD -> main", "origin/main", "origin/HEAD", "develop"}},
			{Graph: "|\\", Hash: ""},
			{Graph: "| * ", Hash: "parent", RelativeAge: "14 minutes ago", Author: "alexander", Subject: "Add suffix-based zsh completion", Decorations: []string{"origin/HEAD -> origin/main"}},
		},
	})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if !strings.HasPrefix(rows[0].Graph, "*") || rows[1].Commit.Hash != "" || !strings.HasPrefix(rows[2].Graph, "| *") {
		t.Fatalf("expected raw graph prefixes to be preserved, got %q, %q, %q", rows[0].Graph, rows[1].Graph, rows[2].Graph)
	}
	line := renderGraphLine(rows[0], true, true, 0, LocalBranchInventory{Names: []string{"main"}, Known: true, Fresh: true}, 24, 88, false, 0)
	if strings.Index(line, "head") < 0 || strings.Index(line, "o/l->") < 0 || strings.Index(line, "*") < 0 || strings.Index(line, "Merge branch") < 0 {
		t.Fatalf("expected graph line to include hash, local branch, title and graph, got %q", line)
	}
	if !strings.Contains(line, headMark.Render("*")) {
		t.Fatalf("expected HEAD pointer to be highlighted, got %q", line)
	}
	if strings.Index(line, "head") > strings.Index(line, "o/l->") {
		t.Fatalf("expected hash to lead branches, got %q", line)
	}
	if strings.Index(line, "o/l->") > strings.Index(line, "*") || strings.Index(line, "*") > strings.Index(line, "Merge branch") {
		t.Fatalf("expected commit columns to stay ordered, got %q", line)
	}
	if strings.Contains(line, "Merge branch 'main' into develop") || strings.Contains(line, "origin/") {
		t.Fatalf("expected title and extra branch decorations to be hidden, got %q", line)
	}
	narrow := renderGraphLine(rows[0], true, true, 0, LocalBranchInventory{Names: []string{"main"}, Known: true, Fresh: true}, 24, 70, false, 0)
	if strings.Contains(narrow, "alexa..") {
		t.Fatalf("expected author to shrink away before title on narrow rows, got %q", narrow)
	}
	if !strings.Contains(narrow, "Merge branch '") {
		t.Fatalf("expected title to stay visible with a narrow-width ellipsis, got %q", narrow)
	}
	connector := renderGraphLine(rows[1], false, true, 0, LocalBranchInventory{Names: []string{"main"}, Known: true, Fresh: true}, 24, 80, false, 0)
	if !strings.Contains(connector, "|\\") {
		t.Fatalf("expected connector graph line to stay visible, got %q", connector)
	}
	focused := renderGraphLine(rows[2], true, true, 0, LocalBranchInventory{Names: []string{"main"}, Known: true, Fresh: true}, 24, 80, false, 0)
	if !strings.Contains(focused, pointerMark.Render("*")) {
		t.Fatalf("expected branch row graph pointer to be highlighted, got %q", focused)
	}
	if compactWhenText("5 minutes ago") != "5m" {
		t.Fatalf("expected relative time to compact to 5m")
	}
	if compactWhenText("1 second ago") != "1s" {
		t.Fatalf("expected second unit to compact to 1s")
	}
	if compactTitleText("Merge branch 'main' into develop") != "Merge branch 'mai..." {
		t.Fatalf("expected title to compact to 20 chars")
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
	if got := formatSectionTargetItem(state.TargetItem{Kind: state.TargetKindTag, Name: "v1.0.0", Ref: "v1.0.0", CommitHash: "abc1234", Subject: "initial release", RelativeAge: "2 days ago"}, 80); !strings.Contains(got, "abc1234") || !strings.Contains(got, "v1.0.0") || !strings.Contains(got, "2d") || !strings.Contains(got, "(unknown)") {
		t.Fatalf("expected tag rows to use hash, name, age and unknown state, got %q", got)
	}
	if got := formatSectionTargetItem(state.TargetItem{Kind: state.TargetKindTag, Name: "v1.1.0", Ref: "v1.1.0", CommitHash: "def5678", Subject: "second release", RelativeAge: "1 day ago", ProvenanceLoaded: true, OriginKnown: true, OnOrigin: true}, 80); !strings.Contains(got, "def5678") || !strings.Contains(got, "v1.1.0") || !strings.Contains(got, "(origin)") {
		t.Fatalf("expected origin-synced tag rows to show hash, name and origin marker, got %q", got)
	}
	if got := formatSectionTargetItem(state.TargetItem{Kind: state.TargetKindTag, Name: "v1.2.0", Ref: "v1.2.0", CommitHash: "fedcba9", Subject: "third release", RelativeAge: "just now", ProvenanceLoaded: true, OriginKnown: true}, 80); !strings.Contains(got, "fedcba9") || !strings.Contains(got, "v1.2.0") || !strings.Contains(got, "(local)") {
		t.Fatalf("expected known but missing remote tags to show name and local state, got %q", got)
	}
}

func TestGraphFocusedRowStashHighlightChangesRendering(t *testing.T) {
	forceTrueColorProfile(t)
	rows := graphRows(git.Status{
		LocalBranches: []string{"main"},
		GraphCommits: []git.GraphCommit{
			{Hash: "abc1234", RelativeAge: "5 minutes ago", Subject: "Marker commit", Decorations: []string{"main"}},
		},
	})
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	withoutStash := renderGraphLine(rows[0], true, true, 0, LocalBranchInventory{Names: []string{"main"}, Known: true, Fresh: true}, 24, 80, false, 0)
	withStash := renderGraphLine(rows[0], true, true, 0, LocalBranchInventory{Names: []string{"main"}, Known: true, Fresh: true}, 24, 80, false, 1)
	if withStash == withoutStash {
		t.Fatalf("expected stash highlight to change focused graph row rendering\nwithout: %q\nwith:    %q", withoutStash, withStash)
	}
	if !strings.Contains(withStash, stashMark.Render("*")) {
		t.Fatalf("expected focused graph row to use stash marker style on the graph pointer, got %q", withStash)
	}
}

func TestGraphDetailShowsStashSummaryForFocusedCommit(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				LocalBranches: []string{"main"},
				GraphCommits: []git.GraphCommit{
					{Hash: "abc1234", RelativeAge: "5 minutes ago", Subject: "Add stash marker", Decorations: []string{"main"}},
				},
			},
			stashByBase: map[string][]git.StashEntry{
				"abc1234": {
					{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "abc1234", Subject: "latest change"},
					{Ref: "stash@{1}", Hash: "stashhash1", BaseHash: "abc1234", Subject: "older change"},
				},
			},
		},
		pullState: pullState{}}

	got := m.renderDetailContent(80, 24)
	if !strings.Contains(got, "stashes:") {
		t.Fatalf("expected detail panel to include stash section, got %q", got)
	}
	if !strings.Contains(got, "stash@{0} - latest change") {
		t.Fatalf("expected newest stash to be listed first, got %q", got)
	}
	if !strings.Contains(got, "stash@{1} - older change") {
		t.Fatalf("expected older stash to remain visible in summary, got %q", got)
	}
	if strings.Index(got, "stash@{0} - latest change") > strings.Index(got, "stash@{1} - older change") {
		t.Fatalf("expected stash summary to keep newest-first order, got %q", got)
	}
}

func TestRenderGraphContentShowsStashBadgeForFocusedCommit(t *testing.T) {
	forceTrueColorProfile(t)
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				LocalBranches: []string{"main"},
				GraphCommits: []git.GraphCommit{
					{Hash: "abc1234", RelativeAge: "5 minutes ago", Subject: "Marker commit", Decorations: []string{"main"}},
				},
			},
			stashByBase: map[string][]git.StashEntry{
				"abc1234": {
					{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "abc1234", Subject: "latest change"},
				},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}

	raw := m.renderGraphContent(80, 8)
	if !strings.Contains(raw, stashMark.Render("*")) {
		t.Fatalf("expected graph content to color the stash pointer, got %q", raw)
	}
}

func TestRenderGraphContentShowsTagPointerForTaggedCommit(t *testing.T) {
	forceTrueColorProfile(t)
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				LocalBranches: []string{"main"},
				GraphCommits: []git.GraphCommit{
					{Graph: "*", Hash: "abc1234", RelativeAge: "5 minutes ago", Subject: "Marker commit", Decorations: []string{"main"}, Tags: []string{"v1.0.0"}},
				},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}

	raw := m.renderGraphContent(80, 8)
	if !strings.Contains(raw, tagColor.Render("*")) {
		t.Fatalf("expected graph content to color the tag pointer, got %q", raw)
	}
}

func TestFormatSectionTargetItemUsesTagHashColor(t *testing.T) {
	forceTrueColorProfile(t)
	got := formatSectionTargetItem(state.TargetItem{Kind: state.TargetKindTag, Name: "v1.0.0", Ref: "v1.0.0", CommitHash: "abc1234", Subject: "initial release", RelativeAge: "2 days ago"}, 80)
	if !strings.Contains(got, "35") {
		t.Fatalf("expected tag hash to use ANSI magenta foreground, got %q", got)
	}
}

func TestTagProvenanceStateLabel(t *testing.T) {
	forceTrueColorProfile(t)
	if got := ansi.Strip(tagProvenanceStateLabel(false, false, false)); got != "(unknown)" {
		t.Fatalf("expected unknown provenance label, got %q", got)
	}
	if got := tagProvenanceStateLabel(true, true, false); !strings.Contains(got, "35") || !strings.Contains(ansi.Strip(got), "(local)") {
		t.Fatalf("expected local provenance label to use ANSI magenta, got %q", got)
	}
	if got := tagProvenanceStateLabel(true, true, true); !strings.Contains(got, "34") || !strings.Contains(ansi.Strip(got), "(origin)") {
		t.Fatalf("expected origin provenance label to use ANSI blue, got %q", got)
	}
}

func TestCompactTagTitleText(t *testing.T) {
	if got := compactTagTitleText("tag"); got != "tag       " {
		t.Fatalf("expected short tag names to pad to 10 chars, got %q", got)
	}
	if got := compactTagTitleText("abcdefg"); got != "abcdefg   " {
		t.Fatalf("expected 7-char tag names to pad to 10 chars, got %q", got)
	}
	if got := compactTagTitleText("abcdefgh"); got != "abcdefg..." {
		t.Fatalf("expected long tag names to truncate to 7 chars plus ellipsis, got %q", got)
	}
}

func TestAttachGraphTagEntriesCopiesCommitTags(t *testing.T) {
	status := git.Status{
		GraphCommits: []git.GraphCommit{
			{Hash: "abc1234", Subject: "marker"},
			{Hash: "def5678", Subject: "other"},
		},
		TagEntries: []git.TagEntry{
			{Name: "v1.1.0", CommitHash: "abc1234"},
			{Name: "v1.0.0", CommitHash: "abc1234"},
			{Name: "v2.0.0", CommitHash: "def5678"},
		},
	}

	got := attachGraphTagEntries(status)
	if !reflect.DeepEqual(got.GraphCommits[0].Tags, []string{"v1.0.0", "v1.1.0"}) {
		t.Fatalf("expected first commit tags to be copied and sorted, got %#v", got.GraphCommits[0].Tags)
	}
	if !reflect.DeepEqual(got.GraphCommits[1].Tags, []string{"v2.0.0"}) {
		t.Fatalf("expected second commit tags to be copied, got %#v", got.GraphCommits[1].Tags)
	}
}

func TestRenderGraphContentShowsTagPointerForMultipleTags(t *testing.T) {
	forceTrueColorProfile(t)
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				LocalBranches: []string{"main"},
				GraphCommits: []git.GraphCommit{
					{Graph: "*", Hash: "abc1234", RelativeAge: "5 minutes ago", Subject: "Marker commit", Decorations: []string{"main"}, Tags: []string{"v1.0.0", "v1.0.1"}},
				},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}

	raw := m.renderGraphContent(80, 8)
	if !strings.Contains(raw, tagColor.Render("*")) {
		t.Fatalf("expected graph content to color the tag pointer with multiple tags, got %q", raw)
	}
}

func TestRenderGraphContentShowsOverlapPointerForStashAndTag(t *testing.T) {
	forceTrueColorProfile(t)
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				LocalBranches: []string{"main"},
				GraphCommits: []git.GraphCommit{
					{Graph: "*", Hash: "abc1234", RelativeAge: "5 minutes ago", Subject: "Marker commit", Decorations: []string{"main"}, Tags: []string{"v1.0.0"}},
				},
			},
			stashByBase: map[string][]git.StashEntry{
				"abc1234": {
					{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "abc1234", Subject: "latest change"},
				},
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}

	raw := m.renderGraphContent(80, 8)
	if !strings.Contains(raw, tagOverlapColor.Render("*")) {
		t.Fatalf("expected overlap pointer to use #A14743, got %q", raw)
	}
}

func TestRenderStashPopupListsEntriesFlatAndKeepsOrder(t *testing.T) {
	forceTrueColorProfile(t)
	m := model{
		repositoryState: repositoryState{
			stashEntries: []git.StashEntry{
				{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "abc1234", Subject: "latest change"},
				{Ref: "stash@{1}", Hash: "stashhash1", BaseHash: "abc1234", Subject: "older change"},
				{Ref: "stash@{2}", Hash: "stashhash2", BaseHash: "def5678", Subject: "feature WIP"},
			},
		},
		pullState:    pullState{},
		overlayState: overlayState{stashPopupCursor: 1}}

	got := renderStashPopup(m, 90, 24)
	plain := ansi.Strip(got)
	if !strings.Contains(got, "Stash list") {
		t.Fatalf("expected stash popup title, got %q", got)
	}
	first := strings.Index(plain, "abc1234  stash@{0}  latest change")
	second := strings.Index(plain, "abc1234  stash@{1}  older change")
	third := strings.Index(plain, "def5678  stash@{2}  feature WIP")
	if first < 0 || second < 0 || third < 0 {
		t.Fatalf("expected stash popup to include stash entries, got %q", plain)
	}
	if first > second {
		t.Fatalf("expected newest stash to appear before older stash, got %q", plain)
	}
	if !strings.Contains(got, "enter: jump") || !strings.Contains(got, "q: close") {
		t.Fatalf("expected stash popup help text, got %q", got)
	}
	if strings.Contains(got, "up/down: move") {
		t.Fatalf("expected stash popup to omit up/down help, got %q", got)
	}
}

func TestRenderStashPopupTruncatesLongSubject(t *testing.T) {
	forceTrueColorProfile(t)
	m := model{
		repositoryState: repositoryState{
			stashEntries: []git.StashEntry{
				{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "abc1234", Subject: "this subject is definitely longer than twenty chars"},
			},
		},
	}

	got := ansi.Strip(renderStashPopup(m, 90, 24))
	if !strings.Contains(got, "abc1234  stash@{0}  this subject is de") {
		t.Fatalf("expected long subject to truncate near 20 chars, got %q", got)
	}
	if strings.Contains(got, "definitely longer than twenty chars") {
		t.Fatalf("expected long subject to be truncated, got %q", got)
	}
}

func TestRenderStashPopupCentersDescriptionLine(t *testing.T) {
	forceTrueColorProfile(t)
	m := model{
		repositoryState: repositoryState{
			stashEntries: []git.StashEntry{
				{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "abc1234", Subject: "latest change"},
			},
		},
	}

	lines := strings.Split(ansi.Strip(renderStashPopup(m, 90, 24)), "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "Browse stash entries.") {
			found = true
			if idx := strings.Index(line, "Browse stash entries."); idx <= 1 {
				t.Fatalf("expected description line to have centered padding, got %q", line)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected description line to be present")
	}
}

func TestRenderConfirmPopupCentersHotkeys(t *testing.T) {
	m := model{
		status: state.New().WithConfirm(state.ActionStash, "Stash changes?", "Continue?"),
	}
	got := renderConfirmPopup(m, 60)
	if !strings.Contains(got, "y: stash  •  n: cancel") {
		t.Fatalf("expected stash confirm help, got %q", got)
	}
	if strings.Contains(got, "\ny: stash  •  n: cancel") {
		t.Fatalf("expected hotkeys line to be centered, got %q", got)
	}
}

func TestRenderConfirmPopupCentersPushYesNoFooter(t *testing.T) {
	m := model{
		status: state.New().WithConfirm(state.ActionSetUpstream, "Push and track remote?", "Set upstream to origin/tmp2?"),
	}
	got := ansi.Strip(renderConfirmPopup(m, 60))
	lines := strings.Split(got, "\n")
	const want = "y: yes  •  n: no"
	found := false
	for _, line := range lines {
		if !strings.Contains(line, want) {
			continue
		}
		found = true
		if idx := strings.Index(line, want); idx <= 1 {
			t.Fatalf("expected push footer to be centered, got %q", line)
		}
		if trailing := len(line) - strings.Index(line, want) - len(want); trailing <= 1 {
			t.Fatalf("expected push footer to leave trailing padding as well, got %q", line)
		}
	}
	if !found {
		t.Fatalf("expected push footer line, got %q", got)
	}
}

func TestRenderStashMessagePopupShowsInputAndHelp(t *testing.T) {
	m := model{
		overlayState: overlayState{stashMessageDraft: "wip: local cleanup"},
	}
	got := renderStashMessagePopup(m, 72)
	if !strings.Contains(got, "Enter a message for this stash.") {
		t.Fatalf("expected stash message prompt, got %q", got)
	}
	if !strings.Contains(got, "message: wip: local cleanup") {
		t.Fatalf("expected stash message draft, got %q", got)
	}
	if !strings.Contains(got, "enter: stash") || !strings.Contains(got, "q: close") {
		t.Fatalf("expected stash message help, got %q", got)
	}
}

func TestRenderTagPopupUsesSingleTitleStrip(t *testing.T) {
	forceTrueColorProfile(t)
	got := ansi.Strip(renderTagPopup(model{
		overlayState: overlayState{tagPopupTarget: "abc1234", tagPopupDraft: "v1.2.3"},
	}, 72, 24))
	if !strings.Contains(got, "Create tag") {
		t.Fatalf("expected tag popup title, got %q", got)
	}
	if strings.Contains(got, "Tag commit") {
		t.Fatalf("expected tag popup to avoid nested body title, got %q", got)
	}
	if !strings.Contains(got, "target: abc1234") || !strings.Contains(got, "name: v1.2.3") {
		t.Fatalf("expected tag popup fields, got %q", got)
	}
	if !strings.Contains(got, "enter: create") || !strings.Contains(got, "q: close") {
		t.Fatalf("expected tag popup help, got %q", got)
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

	parentIdx := graph.FindRowByHash(rows, "37f0954")
	if parentIdx < 0 || parentIdx+1 >= len(rows) || rows[parentIdx+1].Commit.Hash != "efb164e" {
		t.Fatalf("expected efb164e immediately after 37f0954, got index=%d rows=%v", parentIdx, rows)
	}
	parentLine := renderGraphLine(rows[parentIdx+1], false, false, 0, nil, 24, 80, false, 0)
	if !strings.Contains(parentLine, "efb16") {
		t.Fatalf("expected efb164e row to render, got %q", parentLine)
	}

	rootIdx := graph.FindRowByHash(rows, "4ba1faf")
	if rootIdx < 0 || rootIdx+1 >= len(rows) || rows[rootIdx+1].Commit.Hash != "5525707" {
		t.Fatalf("expected 5525707 immediately after 4ba1faf, got index=%d rows=%v", rootIdx, rows)
	}
	rootLine := renderGraphLine(rows[rootIdx+1], false, false, 0, nil, 24, 80, false, 0)
	if !strings.Contains(rootLine, "55257") {
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
	if !strings.Contains(got, "abcde") {
		t.Fatalf("expected hash to remain visible, got %q", got)
	}
	if !strings.Contains(got, "*") {
		t.Fatalf("expected graph marker to remain visible, got %q", got)
	}
}

func TestRenderGraphContentClipsHeadersBeforeRows(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
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
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
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
		repositoryState: repositoryState{
			repoStatus: git.Status{
				LocalBranches:       []string{"feature/super-long-local-branch-name"},
				RemoteBranches:      []string{"origin/super-long-remote-branch-name"},
				TagProvenanceLoaded: true,
				TagEntries: []git.TagEntry{
					{Name: "v1.0.0", CommitHash: "abc1234", Subject: "release with an extremely long title", RelativeAge: "2 days ago", OriginKnown: true, OnOrigin: true},
				},
			},
		},
	}
	got := m.renderRightRail(40, 18)
	if got == "" {
		t.Fatal("expected right rail to render")
	}
	lines := strings.Split(got, "\n")
	if len(lines) != 18 {
		t.Fatalf("expected right rail outer height to be exact, got %d lines: %q", len(lines), got)
	}
	for _, want := range []string{"Local", "Remote", "Tags"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected right rail to contain %q, got %q", want, got)
		}
	}
	if detailsIdx := strings.Index(got, "Details"); detailsIdx < 0 || detailsIdx > strings.Index(got, "Local") {
		t.Fatalf("expected Details panel to precede Local in the right rail, got %q", got)
	}
	if strings.Contains(got, "Actions") {
		t.Fatalf("expected right rail to omit Context Actions, got %q", got)
	}
	for i, line := range lines {
		if width := lipgloss.Width(ansi.Strip(line)); width > 42 {
			t.Fatalf("expected right rail line %d to fit helper width, got width=%d line=%q", i, width, line)
		}
	}
	if !strings.Contains(ansi.Strip(got), "feature/super-long") {
		t.Fatalf("expected local section to remain readable, got %q", got)
	}
	if !strings.Contains(ansi.Strip(got), "super-long-remote") {
		t.Fatalf("expected remote section to remain readable, got %q", got)
	}
	if !strings.Contains(ansi.Strip(got), "abc1234") {
		t.Fatalf("expected tag section to show the commit hash, got %q", got)
	}
}

func TestRenderTagsEmptyStateStopsPromptAfterProvenanceLoad(t *testing.T) {
	m := model{
		repositoryState: repositoryState{
			repoStatus: git.Status{
				TagProvenanceLoaded: true,
				TagSyncSummary:      string(tagSyncSynced),
			},
			tagSyncAttempted: true,
		},
	}
	got := m.renderSectionContent(sectionTags, 40, 4)
	if strings.Contains(got, "Press F to sync tag provenance.") {
		t.Fatalf("expected provenanced empty tags state to stop prompting for F, got %q", got)
	}
	if !strings.Contains(got, "No local tags found.") {
		t.Fatalf("expected empty tags state to explain no tags were found, got %q", got)
	}
}

func TestRenderTagsSectionAddsColumnHeader(t *testing.T) {
	m := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{sectionTags: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				TagProvenanceLoaded: true,
				TagEntries: []git.TagEntry{
					{Name: "v1.0.0", CommitHash: "abc1234", RelativeAge: "2 days ago", OriginKnown: true, OnOrigin: true},
				},
			},
		},
		pullState: pullState{}}
	got := m.renderSectionContent(sectionTags, 80, 4)
	if !strings.Contains(ansi.Strip(got), "hash") || !strings.Contains(ansi.Strip(got), "name") || !strings.Contains(ansi.Strip(got), "age") || !strings.Contains(ansi.Strip(got), "status") {
		t.Fatalf("expected tags section to render column headers, got %q", got)
	}
	if !strings.Contains(got, renderContextKey("hash")) || !strings.Contains(got, renderContextKey("status")) {
		t.Fatalf("expected tags column headers to use blue styling, got %q", got)
	}
}

func TestRenderAppViewFitsViewportWidth(t *testing.T) {
	m := model{
		navigationState: navigationState{
			width:  140,
			height: 60,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch:              "feature/super-long-local-branch-name",
				Head:                "abcdef1234567890",
				Remote:              "origin",
				LocalBranches:       []string{"feature/super-long-local-branch-name"},
				RemoteBranches:      []string{"origin/super-long-remote-branch-name"},
				TagProvenanceLoaded: true,
				TagEntries: []git.TagEntry{
					{Name: "v1.0.0", CommitHash: "abc1234", Subject: "release with an extremely long title", RelativeAge: "2 days ago", OriginKnown: true, OnOrigin: true},
				},
			},
		},
		status: state.New().WithBrowse()}
	got := renderAppView(m)
	lines := strings.Split(got, "\n")
	for i, line := range lines {
		if width := lipgloss.Width(ansi.Strip(line)); width > 140 {
			t.Fatalf("expected renderAppView line %d to fit viewport width, got width=%d line=%q", i, width, line)
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
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:       "/repo",
				Branch:     "feature",
				Head:       "abc1234",
				NoUpstream: true,
				HasCommits: true,
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
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
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:       "/repo",
				Branch:     "main",
				Head:       "abc1234",
				Upstream:   "origin/main",
				NoUpstream: false,
				HasCommits: true,
			},
		},
		pullState: pullState{},
		status:    state.New().WithBrowse()}
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
		repositoryState: repositoryState{
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
		},
		pullState: pullState{
			handshakeCommits: make(map[string]bool),
		},
		status: loadingToast("Pushing...")}

	err := fmt.Errorf("git push: exit status 1: error: failed to push some refs to '...' [rejected - non-fast-forward]")
	msg := executedMsg{
		action:        state.ActionPush,
		target:        "develop",
		err:           err,
		operationErr:  err,
		errorCategory: NonFastForward,
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
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:       fixture.root,
				Branch:     "main",
				Head:       fixture.initialHash,
				HasCommits: true,
				GraphCommits: []git.GraphCommit{
					{Hash: fixture.initialHash, Subject: "Commit 1"},
				},
			},
		},
		pullState: pullState{},
		repo:      fixture.repo,
		status:    state.New().WithBrowse()}
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
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:       fixture.root,
				Branch:     "main",
				Head:       fixture.initialHash,
				HasCommits: true,
			},
		},
		pullState: pullState{},
		repo:      fixture.repo,
		status:    state.New().WithResetModePick("Choose a reset mode.", "")}
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

func TestRenderResetModePopupUsesSingleModeList(t *testing.T) {
	got := renderResetModePopup(60)
	if strings.Contains(got, "enter: execute") {
		t.Fatalf("expected reset popup to hide enter trigger, got %q", got)
	}
	if strings.Count(got, "s: soft") != 1 || strings.Count(got, "m: mixed") != 1 || strings.Count(got, "h: hard") != 1 {
		t.Fatalf("expected single-line mode list, got %q", got)
	}
	if !strings.Contains(got, "Reset mode") || !strings.Contains(got, "Choose a reset mode.") || !strings.Contains(got, "q: close") {
		t.Fatalf("expected reset popup to include title, body, and esc help, got %q", got)
	}
	if strings.Contains(got, "\nReset mode\n") {
		t.Fatalf("expected reset popup title to sit on the border, got %q", got)
	}
}

func TestResetExecutedSuccessfullyReturnsToBrowse(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:       "/repo",
				Branch:     "main",
				Head:       "c1",
				HasCommits: true,
			},
		},
		pullState: pullState{},
		status:    loadingToast("Hard reset...")}

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
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:       "/repo",
				Branch:     "main",
				Head:       "c1",
				HasCommits: true,
			},
		},
		status: loadingToast("Merging...")}

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
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root:       "/repo",
				Branch:     "main",
				Head:       "c1",
				HasCommits: true,
			},
		},
		status: loadingToast("Rebasing...")}

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
