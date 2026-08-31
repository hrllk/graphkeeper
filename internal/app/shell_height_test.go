package app

import (
	"strings"
	"testing"
)

// heightMatrix mirrors widthMatrix on the vertical axis. It starts at 5 because
// that is the shell's structural floor: two border rows, one content row, the
// section gap and the footer. Nothing below that can fit whatever the margins do.
var heightMatrix = []int{5, 6, 8, 10, 12, 14, 16, 24, 40}

func heightFixture(h int) model {
	m := widthFixture(100)
	m.height = h
	return m
}

func renderedRows(m model) int {
	return len(strings.Split(strings.TrimRight(m.View(), "\n"), "\n"))
}

// The defect, found by /review on the graph-width fix: layoutShellBodySize had a
// floor of 12 on the body height, the exact twin of the 80-column width floor that
// fix removed. Every terminal 15 rows or shorter rendered a 15-row frame, and
// lipgloss.Place centred it, so the Graph title and the commit rows were the part
// that scrolled off the top. Before the fix this loop reports 15 rows at every
// height up to 15.
func TestViewNeverExceedsTerminalHeight(t *testing.T) {
	for _, height := range heightMatrix {
		rows := renderedRows(heightFixture(height))
		if rows > height {
			t.Errorf("terminal height %d: rendered %d rows, %d off screen", height, rows, rows-height)
		}
	}
}

// Fitting inside the terminal is not enough on its own - a frame that collapsed to
// five rows in a 40-row terminal would also pass. The frame has to actually claim
// the height it is given.
func TestShellHeightTracksTerminalHeight(t *testing.T) {
	for _, height := range heightMatrix {
		rows := renderedRows(heightFixture(height))
		if rows < height-1 {
			t.Errorf("terminal height %d: rendered only %d rows", height, rows)
		}
	}
}

// The vertical margins are floored at 2 each, which an eight-row terminal cannot
// pay for. They have to yield to the content rather than push it off screen.
func TestVerticalMarginsYieldOnShortTerminals(t *testing.T) {
	for _, height := range []int{5, 6, 7, 8, 9} {
		m := heightFixture(height)
		_, topMargin, bottomMargin := layoutShellMargins(m)
		if got := topMargin + bottomMargin; got > height-shellRowsReservedForContent {
			t.Errorf("terminal height %d: margins total %d, leaving fewer than %d rows for content",
				height, got, shellRowsReservedForContent)
		}
	}
}

// The horizontal margin used to be capped at (m.width - 80) / 2, the matched pair
// of the removed body-width floor. That cap froze the body at 80 columns for every
// terminal between 80 and 100 wide. Before its removal body(100) == body(80) == 80
// and this fails.
func TestBodyWidthKeepsGrowingPast80Columns(t *testing.T) {
	at := func(width int) int {
		m := widthFixture(width)
		hMargin, topMargin, bottomMargin := layoutShellMargins(m)
		bodyWidth, _ := layoutShellBodySize(m, hMargin, topMargin, bottomMargin)
		return bodyWidth
	}
	if narrow, wide := at(80), at(100); wide <= narrow {
		t.Errorf("body width froze: terminal 80 gives %d, terminal 100 gives %d", narrow, wide)
	}
	previous := 0
	for _, width := range []int{60, 70, 80, 90, 100, 110, 120, 140, 200} {
		bodyWidth := at(width)
		if bodyWidth < previous {
			t.Errorf("terminal %d: body width shrank to %d from %d at the previous width",
				width, bodyWidth, previous)
		}
		previous = bodyWidth
	}
}
