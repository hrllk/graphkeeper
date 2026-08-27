package app

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestCommitInspectorScreenMatrixKeepsFrameAndPartialFooter(t *testing.T) {
	m := model{
		inspectorState: inspectorState{
			commitInspectorOpen: true,
			commitInspectorSnapshot: CommitSnapshot{
				FullHash: "abcdef0123456789", Subject: "change", AuthorName: "dev", Parent: "parent",
				Files: []ChangedFile{{StableID: "f", Status: StatusModified, Path: "internal/app/model.go"}},
			},
			commitInspectorDiffWindow: DiffWindow{FileID: "f", HasMore: true, PartialReason: PartialLineLimit, NextStartLine: 4,
				Hunks: []DiffHunk{{Header: "@@ -1 +1 @@", Rows: []PairedRow{{Kind: "context", From: CodeLine{Number: 1, Text: "same"}, To: CodeLine{Number: 1, Text: "same"}, FromPresent: true, ToPresent: true}}}},
			},
		},
	}
	for _, size := range [][2]int{{40, 12}, {60, 20}, {80, 30}} {
		got := renderCommitInspectorScreen(m, size[0], size[1])
		if lipgloss.Width(got) != size[0] || lipgloss.Height(got) != size[1] {
			t.Fatalf("size %dx%d rendered as %dx%d", size[0], size[1], lipgloss.Width(got), lipgloss.Height(got))
		}
		if !strings.Contains(got, "n next") {
			t.Fatalf("partial screen %dx%d omitted n next: %q", size[0], size[1], got)
		}
		if !strings.Contains(got, "FROM parent") {
			t.Fatalf("screen %dx%d omitted parent direction", size[0], size[1])
		}
	}
}

func TestCommitInspectorScreenNoColorPreservesContextAndPathIdentity(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := model{inspectorState: inspectorState{commitInspectorSnapshot: CommitSnapshot{FullHash: "abc", Subject: "subject", AuthorName: "dev", IsRoot: true, Files: []ChangedFile{{StableID: "f", Status: StatusAdded, Path: "src/한글/very_long_file.go"}}}, commitInspectorDiffWindow: DiffWindow{FileID: "f", Hunks: []DiffHunk{{Header: "@@", Rows: []PairedRow{{Kind: "context", From: CodeLine{Number: 1, Text: "same"}, To: CodeLine{Number: 1, Text: "same"}, FromPresent: true, ToPresent: true}}}}}}}
	got := renderCommitInspectorScreen(m, 40, 12)
	if !strings.Contains(got, "ROOT COMMIT") || !strings.Contains(got, "very_long_file.go") || !strings.Contains(got, "same") {
		t.Fatalf("no-color screen lost identity/context: %q", got)
	}
}

// scrollFixture builds an Inspector model with the given number of diff rows and
// changed files, so the scroll tests can measure what actually renders.
func scrollFixture(diffRows, files int) model {
	rows := make([]PairedRow, 0, diffRows)
	for i := 1; i <= diffRows; i++ {
		rows = append(rows, PairedRow{Kind: "added", To: CodeLine{Number: i, Text: fmt.Sprintf("line %d", i)}, ToPresent: true})
	}
	changed := make([]ChangedFile, 0, files)
	for i := 1; i <= files; i++ {
		changed = append(changed, ChangedFile{StableID: fmt.Sprintf("f%02d", i), Status: StatusAdded, Path: fmt.Sprintf("many/f%02d.txt", i)})
	}
	return model{inspectorState: inspectorState{
		commitInspectorOpen:     true,
		commitInspectorSnapshot: CommitSnapshot{FullHash: "abc", Subject: "s", AuthorName: "dev", Parent: "p", Files: changed},
		commitInspectorDiffWindow: DiffWindow{FileID: "f01",
			Hunks: []DiffHunk{{Header: "@@ -0,0 +1," + strconv.Itoa(diffRows) + " @@", Rows: rows}}},
	}}
}

func inspectorBodyLines(rendered string) []string {
	lines := strings.Split(rendered, "\n")
	body := make([]string, 0, len(lines))
	seenPaneHeader := false
	for _, line := range lines {
		if !seenPaneHeader {
			if strings.Contains(line, "Changed files") && strings.Contains(line, "Diff") {
				seenPaneHeader = true
			}
			continue
		}
		if strings.Contains(line, "Esc back") {
			break
		}
		body = append(body, line)
	}
	return body
}

// T8 regression, structural half. The renderer used to generate more body rows
// than the frame keeps and rely on truncation to trim them, so the scroll clamp
// was two rows too generous and the last lines of a scrolled diff could never be
// reached. Generation and the clamp must agree.
func TestInspectorBodyRowsMatchesWhatTheFrameKeeps(t *testing.T) {
	m := scrollFixture(500, 3)
	for _, height := range []int{12, 14, 20, 30, 40, 41, 50} {
		body := inspectorBodyLines(renderCommitInspectorScreen(m, 120, height))
		if want := inspectorBodyRows(height); len(body) != want {
			t.Fatalf("height %d rendered %d body rows, inspectorBodyRows says %d", height, len(body), want)
		}
	}
}

// T8 regression. Ctrl+U/Ctrl+D moved commitInspectorScroll but the screen renderer
// indexed from zero, so an 800-line diff showed its first screen and nothing else.
func TestInspectorDiffPaneScrollsWithTheOffset(t *testing.T) {
	m := scrollFixture(500, 1)
	first := renderCommitInspectorScreen(m, 120, 40)
	if !strings.Contains(first, "line 1") {
		t.Fatalf("unscrolled pane missing the first row: %q", first)
	}

	m.commitInspectorScroll = 100
	scrolled := renderCommitInspectorScreen(m, 120, 40)
	if strings.Contains(inspectorBodyLines(scrolled)[0], "line 1 ") {
		t.Fatal("scrolled pane still starts at the first row")
	}
	if !strings.Contains(scrolled, "line 101") {
		t.Fatalf("scroll offset 100 did not reach line 101: %q", scrolled)
	}
}

// The last diff line must be reachable. It is the line the reader is usually
// looking for, and it was the one the off-by-two hid.
func TestInspectorDiffPaneReachesTheLastLine(t *testing.T) {
	m := scrollFixture(500, 1)
	m.height = 40
	m.commitInspectorScroll = m.maxInspectorDiffScroll()
	got := renderCommitInspectorScreen(m, 120, 40)
	if !strings.Contains(got, "line 500") {
		t.Fatalf("max scroll did not reveal the last line: %q", got)
	}
}

// Ctrl+D must stop at the end instead of growing without bound, and Ctrl+U must
// move on the very next press rather than working off an inflated offset.
func TestInspectorScrollKeysClampAndReverseImmediately(t *testing.T) {
	m := scrollFixture(500, 1)
	m.height = 40
	maxScroll := m.maxInspectorDiffScroll()
	if maxScroll <= 0 {
		t.Fatalf("fixture produced no scrollable content: max=%d", maxScroll)
	}
	for i := 0; i < 200; i++ {
		next, _ := m.handleCommitInspectorKey(tea.KeyMsg{Type: tea.KeyCtrlD})
		m = next.(model)
	}
	if m.commitInspectorScroll != maxScroll {
		t.Fatalf("ctrl+d settled at %d, want the clamp %d", m.commitInspectorScroll, maxScroll)
	}
	next, _ := m.handleCommitInspectorKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	if got := next.(model).commitInspectorScroll; got >= maxScroll {
		t.Fatalf("ctrl+u after the clamp did not move: %d", got)
	}
}

// T8 regression, file pane. With more files than rows the selection used to walk
// off the bottom and no "> " marker was rendered anywhere on screen.
func TestInspectorFilePaneKeepsTheCursorVisible(t *testing.T) {
	m := scrollFixture(10, 60)
	for _, cursor := range []int{0, 45, 59} {
		m.commitInspectorCursor = cursor
		got := renderCommitInspectorScreen(m, 120, 40)
		want := fmt.Sprintf("> A many/f%02d.txt", cursor+1)
		if !strings.Contains(got, want) {
			t.Fatalf("cursor %d: %q not rendered", cursor, want)
		}
	}
}

func TestInspectorFileOffsetScrollsOnlyAsNeeded(t *testing.T) {
	for _, tt := range []struct {
		cursor, total, visible, want int
	}{
		{0, 60, 30, 0},
		{29, 60, 30, 0},
		{30, 60, 30, 1},
		{59, 60, 30, 30},
		{5, 10, 30, 0},
		{0, 0, 30, 0},
	} {
		if got := inspectorFileOffset(tt.cursor, tt.total, tt.visible); got != tt.want {
			t.Fatalf("inspectorFileOffset(%d, %d, %d) = %d, want %d", tt.cursor, tt.total, tt.visible, got, tt.want)
		}
	}
}
