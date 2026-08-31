package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

// widthMatrix is the fixed width set every 6.2 assertion runs over. 20 is the
// narrowest width a person plausibly uses; 140 is wide enough that no clamp
// binds, so it guards against regressions at the top end.
var widthMatrix = []int{20, 30, 40, 60, 80, 140}

// degenerateWidths are the widths where the split arithmetic is at its limits.
// Found by /review: at terminal 9 the budget is 1, 0.72 truncates the graph to 0,
// and the rail came out wider than the graph - the inversion decisions.md:15
// forbids. widthMatrix starts at 20 and never reached it.
var degenerateWidths = []int{5, 6, 7, 8, 9, 10, 12, 15}

// widthFixture is the shared model for the width assertions. Two commits, one
// local branch, snapshot marked loaded so the graph renders rows rather than the
// empty state.
func widthFixture(w int) model {
	return model{
		navigationState: navigationState{
			width: w, height: 40,
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{sectionGraph: 0},
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Root: "/repo", Branch: "main", Head: "aaaaaaa1",
				LocalBranches: []string{"main"},
				GraphCommits: []git.GraphCommit{
					{Hash: "aaaaaaa1", Author: "a", Subject: "commit number one with a long subject", Decorations: []string{"HEAD -> main", "main"}},
					{Hash: "bbbbbbb2", Author: "b", Subject: "commit number two with a long subject", Parents: []string{"aaaaaaa1"}},
				},
			},
			repoSnapshotLoaded: true,
		},
		status: state.New().WithBrowse(),
	}
}

// The defect: layoutShellBodySize clamped the body up to 80 columns whatever the
// terminal was, so View() drew an 80-wide frame that lipgloss.Place then centred
// inside a narrower terminal. Measured before the fix: 64 columns off screen at
// terminal 20, 32 at terminal 60.
//
// decisions.md:91 (2026-06-30) already decided this: "Prevent width overflow
// inside any graph cell so wrapping cannot break the shared-height contract."
// This test is that decision, enforced.
func TestViewNeverExceedsTerminalWidth(t *testing.T) {
	for _, width := range widthMatrix {
		for i, line := range strings.Split(widthFixture(width).View(), "\n") {
			if got := lipgloss.Width(line); got > width {
				t.Fatalf("width %d: line %d is %d cells wide, %d over: %q",
					width, i, got, got-width, line)
			}
		}
	}
}

// The frame has to close, not just fit. An unclosed box is what the user
// actually sees when the right edge falls off the screen.
func TestNarrowFrameClosesAndKeepsTheRail(t *testing.T) {
	got := widthFixture(60).View()
	for _, want := range []string{"╮", "╯", "[2] Local"} {
		if !strings.Contains(got, want) {
			t.Fatalf("width 60: %q missing from the frame:\n%s", want, got)
		}
	}
}

// The footer used to be drawn at the 80-column body width and cut mid-word at
// "ctr". Truncation is fine; truncating inside a word is the symptom.
func TestNarrowFooterDoesNotCutMidWord(t *testing.T) {
	for _, width := range widthMatrix {
		for _, line := range strings.Split(widthFixture(width).View(), "\n") {
			if !strings.Contains(line, "tab: switch") {
				continue
			}
			if strings.HasSuffix(strings.TrimRight(line, " "), "ctr") {
				t.Fatalf("width %d: footer cut mid-word: %q", width, line)
			}
		}
	}
}

// decisions.md:15 makes Graph the primary full-height surface, so when the body
// shrinks the rail yields first. Before the fix the opposite happened: the rail
// held a hard 18 columns and the graph was squeezed to 2 at terminal 30.
func TestGraphNeverNarrowerThanRail(t *testing.T) {
	for _, width := range append(append([]int{}, widthMatrix...), degenerateWidths...) {
		m := widthFixture(width)
		hMargin, topMargin, bottomMargin := layoutShellMargins(m)
		bodyWidth, _ := layoutShellContentSize(m, hMargin, topMargin, bottomMargin)
		graphWidth, railWidth := graphAndRailWidths(bodyWidth)
		if graphWidth < railWidth {
			t.Fatalf("width %d: graph %d is narrower than rail %d", width, graphWidth, railWidth)
		}
	}
}

// The exact split per width, measured. Width 140 is unchanged by the fix; width
// 80 moves from 56/20 to 54/22 because the old 56 was the floor's own output and
// cannot survive the floor's removal. That 2-column shift was accepted
// deliberately rather than special-cased.
func TestGraphRailSplitPerWidth(t *testing.T) {
	for _, tt := range []struct{ width, graph, rail int }{
		{20, 8, 4},
		{30, 14, 6},
		{40, 20, 8},
		{60, 31, 13},
		{80, 54, 22},
		{140, 77, 31},
	} {
		m := widthFixture(tt.width)
		hMargin, topMargin, bottomMargin := layoutShellMargins(m)
		bodyWidth, _ := layoutShellContentSize(m, hMargin, topMargin, bottomMargin)
		graphWidth, railWidth := graphAndRailWidths(bodyWidth)
		if graphWidth != tt.graph || railWidth != tt.rail {
			t.Fatalf("width %d: got graph %d rail %d, want graph %d rail %d",
				tt.width, graphWidth, railWidth, tt.graph, tt.rail)
		}
	}
}

// Terminal 5 drives bodyWidth to 1. Both boxes end up zero-width; the
// requirement is that View() returns without panicking and still satisfies the
// width invariant, not that anything is readable.
func TestSmallestBodyWidthDoesNotPanic(t *testing.T) {
	got := widthFixture(5).View()
	for i, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > 5 {
			t.Fatalf("width 5: line %d is %d cells wide: %q", i, w, line)
		}
	}
}
