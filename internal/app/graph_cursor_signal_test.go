package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"hrllk/graphkeeper/internal/graph"
)

// Tests run without a TTY, so lipgloss falls back to the Ascii profile and drops
// every style - graphCursorMark.Render("XX") returns "XX", and so does
// highlight.Render("XX"). That is why no test here has ever caught a styling
// defect. Setting a profile makes styling observable.
func withANSIProfile(t *testing.T) {
	t.Helper()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
}

func cursorProjection(cursor int) GraphProjection {
	return GraphProjection{
		Rows: []graphRow{
			{Commit: graph.Node{Hash: "aaaaaaa1", Author: "a", Subject: "first subject"}, Graph: "*"},
			{Commit: graph.Node{Hash: "bbbbbbb2", Author: "b", Subject: "second subject"}, Graph: "*"},
			{Commit: graph.Node{Hash: "ccccccc3", Author: "c", Subject: "third subject"}, Graph: "*"},
		},
		PageSize: 3,
		Cursor:   cursor,
		Active:   true,
	}
}

func subjectRows(rendered string) []string {
	var out []string
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "subject") {
			out = append(out, line)
		}
	}
	return out
}

// The defect: under NO_COLOR the cursor was invisible. lipgloss drops colour AND
// attributes on the Ascii profile it selects for NO_COLOR, so nothing at all is
// emitted - measured on the built binary, the graph rows are byte-identical with
// and without the cursor on them.
//
// The signal cannot be a "> " gutter: 094ca87 (2026-07-07, "Remove graph
// selection arrow") deleted that from all three render paths and locked it with
// a test. So the cursor cell emits reverse video directly when NO_COLOR is set,
// the way the Inspector already handles NO_COLOR itself
// (commit_inspector.go:352, :441, :466). no-color.org governs colour; reverse is
// an attribute, not a colour.
func TestGraphCursorIsVisibleUnderNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for cursor := 0; cursor < 3; cursor++ {
		rows := subjectRows(renderGraphProjection(cursorProjection(cursor), 120, 8))
		if len(rows) != 3 {
			t.Fatalf("cursor %d: expected 3 rows, got %d", cursor, len(rows))
		}
		marked := -1
		for i, line := range rows {
			if strings.Contains(line, "\x1b[7m") {
				if marked >= 0 {
					t.Fatalf("cursor %d: rows %d and %d both marked", cursor, marked, i)
				}
				marked = i
			}
		}
		if marked != cursor {
			t.Fatalf("cursor %d: reverse-video row is %d\n%q", cursor, marked, rows)
		}
	}
}

// NO_COLOR forbids colour, not attributes. Emitting an SGR colour here would be
// the very thing the user asked the app not to do.
func TestNoColorCursorEmitsNoColour(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, line := range subjectRows(renderGraphProjection(cursorProjection(1), 120, 8)) {
		for _, bad := range []string{"\x1b[3", "\x1b[4", "\x1b[9", "\x1b[1;3"} {
			if strings.Contains(line, bad) {
				t.Fatalf("NO_COLOR row carries a colour sequence %q: %q", bad, line)
			}
		}
	}
}

// With colour available the existing lipgloss path is untouched, so this only
// guards that the NO_COLOR branch does not leak into the normal rendering.
func TestColourPathDoesNotEmitRawReverse(t *testing.T) {
	withANSIProfile(t)
	for _, line := range subjectRows(renderGraphProjection(cursorProjection(1), 120, 8)) {
		if strings.Contains(line, "\x1b[7m") {
			t.Fatalf("colour path emitted the NO_COLOR fallback: %q", line)
		}
	}
}

// 094ca87 stands: no arrow, no left margin.
func TestGraphCursorSignalAddsNoArrowOrMargin(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, line := range subjectRows(renderGraphProjection(cursorProjection(1), 120, 8)) {
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, ">") {
			t.Fatalf("row gained a gutter: %q", line)
		}
	}
}
