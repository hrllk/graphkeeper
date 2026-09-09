package app

import (
	"strings"
	"testing"
)

func roadRow(hash string) graphRow {
	return graphRow{Commit: graphNode{Hash: hash, Subject: "subject"}}
}

func rawRoadRow(hash string) graphRow {
	row := roadRow(hash)
	// A non-empty Graph prefix routes through renderRawGraphLineWithSearch,
	// which is the path a real repository takes.
	row.Graph = "* "
	return row
}

// Both renderers have to carry the marks. renderGraphLineWithSearch delegates to
// the raw path whenever Graph is non-empty, and that is the live path, so a test
// that only covers the compact path passes while the screen never changes.
func TestRoadMarksReachBothRenderPaths(t *testing.T) {
	member := graphRowMarks{RangeMember: true}
	anchor := graphRowMarks{RangeAnchor: true}

	for _, tt := range []struct {
		name string
		row  graphRow
	}{
		{"compact path", roadRow("abcdef1")},
		{"raw path", rawRoadRow("abcdef1")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plain := renderGraphLineWithSearch(tt.row, false, true, 0, nil, graphCols(16), 80, graphRowMarks{}, "")
			if strings.Contains(plain, "\x1b[4m") {
				t.Fatalf("an unmarked row must carry no underline: %q", plain)
			}
			withMember := renderGraphLineWithSearch(tt.row, false, true, 0, nil, graphCols(16), 80, member, "")
			if !strings.Contains(withMember, "\x1b[4m") {
				t.Fatalf("a path member must be underlined: %q", withMember)
			}
			withAnchor := renderGraphLineWithSearch(tt.row, false, true, 0, nil, graphCols(16), 80, anchor, "")
			if !strings.Contains(withAnchor, "\x1b[4m\x1b[1m") {
				t.Fatalf("an anchor must be underlined and bold: %q", withAnchor)
			}
		})
	}
}

// The marks are written as escapes rather than lipgloss styles because under
// NO_COLOR lipgloss emits nothing at all, attributes included. searchMatchMark
// is a lipgloss underline, so it would vanish exactly where the signal matters.
func TestRoadMarksSurviveNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	row := rawRoadRow("abcdef1")

	member := renderGraphLineWithSearch(row, false, true, 0, nil, graphCols(16), 80, graphRowMarks{RangeMember: true}, "")
	if !strings.Contains(member, "\x1b[4m") {
		t.Fatalf("path member lost its mark under NO_COLOR: %q", member)
	}
	anchor := renderGraphLineWithSearch(row, false, true, 0, nil, graphCols(16), 80, graphRowMarks{RangeAnchor: true}, "")
	if !strings.Contains(anchor, "\x1b[4m\x1b[1m") {
		t.Fatalf("anchor lost its mark under NO_COLOR: %q", anchor)
	}
}

// Attributes compose, so a row that is both the cursor and the anchor carries
// reverse and underline together rather than one winning outright.
func TestCursorAndAnchorCompose(t *testing.T) {
	row := rawRoadRow("abcdef1")
	got := renderGraphLineWithSearch(row, true, true, 0, nil, graphCols(16), 80, graphRowMarks{RangeAnchor: true}, "")
	if !strings.Contains(got, "\x1b[7m") {
		t.Fatalf("the cursor mark is missing: %q", got)
	}
	if !strings.Contains(got, "\x1b[4m") {
		t.Fatalf("the anchor mark is missing: %q", got)
	}
}

// An active search owns the hash cell, so the road marks stand down rather than
// fighting the search highlight for the same characters.
func TestSearchOwnsTheCellOverRoadMarks(t *testing.T) {
	row := rawRoadRow("abcdef1")
	got := renderGraphLineWithSearch(row, true, true, 0, nil, graphCols(16), 80, graphRowMarks{RangeAnchor: true}, "abc")
	if strings.Contains(got, "\x1b[4m\x1b[1m") {
		t.Fatalf("search focus must outrank the anchor mark: %q", got)
	}
}

// 094ca87 removed the graph selection arrow and locked it. The road marks must
// not reintroduce a gutter, which would also push content off the left edge.
func TestRoadMarksAddNoGutter(t *testing.T) {
	row := rawRoadRow("abcdef1")
	for _, marks := range []graphRowMarks{{RangeAnchor: true}, {RangeMember: true}} {
		got := renderGraphLineWithSearch(row, true, true, 0, nil, graphCols(16), 80, marks, "")
		stripped := stripAnsiForTest(got)
		if strings.HasPrefix(stripped, "> ") || strings.HasPrefix(stripped, "▏") {
			t.Fatalf("road marks must not add a gutter: %q", stripped)
		}
	}
}

func stripAnsiForTest(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
