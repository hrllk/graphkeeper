package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func linearRows(n int) []graphRow {
	rows := make([]graphRow, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, graphRow{Graph: "* ", Commit: graphNode{Hash: "h", Subject: "s"}})
	}
	return rows
}

// 6.4's completion criteria. The fixed cells used to cost 41 columns at every
// width because the state and topology columns were budgeted for their worst
// case, which left the title negative below 100 columns.
func TestColumnBudgetLeavesTheTitlePositiveAtEightyColumns(t *testing.T) {
	// 39 is the graph content width at an 80-column terminal, measured through
	// graphAndRailWidths.
	const contentAt80 = 39
	cols := measureGraphColumns(linearRows(20), contentAt80, nil)
	title := contentAt80 - graphRowFixedWidth(cols)
	if title <= 0 {
		t.Fatalf("title budget at 80 columns is %d, want positive (topo=%d status=%d)",
			title, cols.Topology, cols.Status)
	}
}

// The original criterion, kept: more room at 100 than at 80.
func TestColumnBudgetGrowsWithWidth(t *testing.T) {
	rows := linearRows(20)
	at80 := 39 - graphRowFixedWidth(measureGraphColumns(rows, 39, nil))
	at100 := 50 - graphRowFixedWidth(measureGraphColumns(rows, 50, nil))
	if at100 <= at80 {
		t.Fatalf("title budget did not grow: 80 -> %d, 100 -> %d", at80, at100)
	}
}

// A graph with nothing to report in the state column does not reserve one.
func TestStateColumnCollapsesWithoutStashOrTag(t *testing.T) {
	rows := linearRows(5)
	if cols := measureGraphColumns(rows, 60, nil); cols.Status != 0 {
		t.Fatalf("no stash and no tag, but the state column is %d wide", cols.Status)
	}
	tagged := linearRows(5)
	tagged[2].Commit.Tags = []string{"v1"}
	if cols := measureGraphColumns(tagged, 60, nil); cols.Status == 0 {
		t.Fatalf("a tagged row needs the state column")
	}
	stashed := linearRows(5)
	stashed[1].Commit.Hash = "withstash"
	if cols := measureGraphColumns(stashed, 60, map[string]int{"withstash": 1}); cols.Status == 0 {
		t.Fatalf("a stashed row needs the state column")
	}
}

// The topology column takes the graph's actual widest cell, never more.
func TestTopologyColumnMatchesTheWidestLane(t *testing.T) {
	rows := linearRows(3)
	rows[1].Graph = "| * "
	cols := measureGraphColumns(rows, 200, nil)
	if want := lipgloss.Width("| * "); cols.Topology != want {
		t.Fatalf("topology = %d, want %d (the widest cell)", cols.Topology, want)
	}
	// The proportional formula stays as a ceiling so a wide graph cannot take
	// the whole row.
	wide := linearRows(2)
	wide[0].Graph = strings.Repeat("| ", 40)
	if cols := measureGraphColumns(wide, 60, nil); cols.Topology > graphTopologyCeiling(60) {
		t.Fatalf("topology %d exceeded its ceiling %d", cols.Topology, graphTopologyCeiling(60))
	}
}

// Every row and the header take the same widths, or the columns do not line up.
// The header is easy to forget: it builds its own prefix rather than reusing the
// row renderer.
func TestHeaderAndRowsShareTheSameColumnWidths(t *testing.T) {
	m := model{}
	m.repoStatus.GraphCommits = nil
	rows := linearRows(3)
	cols := measureGraphColumns(rows, 80, nil)
	header := renderGraphHeader(80, cols)
	if cols.Status == 0 && strings.Contains(header, "state") {
		t.Fatalf("header kept a state column the rows dropped: %q", header)
	}
	rendered := renderGraphLineWithSearch(rows[0], false, true, 0, nil, cols, 80, graphRowMarks{}, "")
	if lipgloss.Width(rendered) > 80 {
		t.Fatalf("row is %d wide at 80: %q", lipgloss.Width(rendered), rendered)
	}
}

// 11.9. The date column returns, on the terms 6.4 set: a column appears only
// where the row can spare it. Measured against graphTitleMinimumWidth, the
// repository's own floor, rather than a threshold invented for this feature.
func TestDateColumnAppearsOnlyWhereTheTitleCanSpareIt(t *testing.T) {
	rows := linearRows(10)
	for _, tt := range []struct {
		name    string
		content int
		want    bool
	}{
		{"60-column terminal", 27, false},
		{"80-column terminal", 39, false},
		{"100-column terminal", 50, true},
		{"120-column terminal", 62, true},
		{"180-column terminal", 96, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cols := measureGraphColumns(rows, tt.content, nil)
			if got := cols.Date > 0; got != tt.want {
				t.Fatalf("date column present = %v, want %v (content %d, title %d)",
					got, tt.want, tt.content, tt.content-graphRowFixedWidth(cols))
			}
			if cols.Date > 0 {
				if left := tt.content - graphRowFixedWidth(cols); left < graphTitleMinimumWidth {
					t.Fatalf("date column left the title %d, below the %d floor", left, graphTitleMinimumWidth)
				}
			}
		})
	}
}

// yymmdd, six columns, fixed. A date that changed width row to row would break
// the alignment the columns exist for, so an unreadable stamp becomes dashes
// rather than blanks or a shorter string.
func TestGraphDateTextIsAlwaysSixColumns(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"2026-09-08T23:15:16+09:00", "260908"},
		{"2026-01-02T00:00:00Z", "260102"},
		{"1999-12-31T23:59:59-08:00", "991231"},
		{"", "------"},
		{"not a date", "------"},
		{"20xx-09-08T00:00:00Z", "------"},
		{"2026-09", "------"},
	} {
		if got := graphDateText(tt.in); got != tt.want {
			t.Fatalf("graphDateText(%q) = %q, want %q", tt.in, got, tt.want)
		}
		if len(graphDateText(tt.in)) != graphDateWidth {
			t.Fatalf("graphDateText(%q) is %d columns, want %d", tt.in, len(graphDateText(tt.in)), graphDateWidth)
		}
	}
}

// Both render paths and the header carry the date, or the columns disagree.
// Only checking the compact path is how the raw path, which is what a real
// repository takes, gets left behind.
func TestDateColumnReachesBothRenderPathsAndTheHeader(t *testing.T) {
	const content = 100
	rows := linearRows(2)
	rows[0].Commit.CommitDate = "2026-09-08T23:15:16+09:00"
	raw := rows[0]
	raw.Graph = "* "
	compact := rows[0]
	compact.Graph = ""
	cols := measureGraphColumns(rows, content, nil)
	if cols.Date == 0 {
		t.Fatal("fixture width should hold a date column")
	}
	if header := renderGraphHeader(content, cols); !strings.Contains(header, "date") {
		t.Fatalf("header lost the date column: %q", header)
	}
	for name, row := range map[string]graphRow{"raw path": raw, "compact path": compact} {
		got := renderGraphLineWithSearch(row, false, true, 0, nil, cols, content, graphRowMarks{}, "")
		if !strings.Contains(got, "260908") {
			t.Fatalf("%s did not render the date: %q", name, got)
		}
	}
}
