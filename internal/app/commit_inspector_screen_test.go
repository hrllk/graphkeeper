package app

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
		got := renderCommitInspectorScreen(withFrame(m, size[0], size[1]))
		if lipgloss.Width(got) != size[0] || lipgloss.Height(got) != size[1] {
			t.Fatalf("size %dx%d rendered as %dx%d", size[0], size[1], lipgloss.Width(got), lipgloss.Height(got))
		}
		if !strings.Contains(got, "n next") {
			t.Fatalf("partial screen %dx%d omitted n next: %q", size[0], size[1], got)
		}
		// The parent has its own key/value row now. As a "FROM <hash>" tail on
		// the author row it had no matching TO and was the first thing cut --
		// at 80 columns, not only at 40.
		if !strings.Contains(got, "parent: parent") {
			t.Fatalf("screen %dx%d omitted the parent row", size[0], size[1])
		}
	}
}

func TestCommitInspectorScreenNoColorPreservesContextAndPathIdentity(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := model{inspectorState: inspectorState{commitInspectorSnapshot: CommitSnapshot{FullHash: "abc", Subject: "subject", AuthorName: "dev", IsRoot: true, Files: []ChangedFile{{StableID: "f", Status: StatusAdded, Path: "src/한글/very_long_file.go"}}}, commitInspectorDiffWindow: DiffWindow{FileID: "f", Hunks: []DiffHunk{{Header: "@@", Rows: []PairedRow{{Kind: "context", From: CodeLine{Number: 1, Text: "same"}, To: CodeLine{Number: 1, Text: "same"}, FromPresent: true, ToPresent: true}}}}}}}
	got := renderCommitInspectorScreen(withFrame(m, 40, 12))
	if !strings.Contains(got, "(root commit)") || !strings.Contains(got, "very_long_file.go") || !strings.Contains(got, "same") {
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

// inspectorBodyLines picks the diff pane's rows out of a rendered frame.
//
// It finds them by position, not by text. Looking for the words "Changed
// files" and "Diff" only works while the frame is wide enough to print them:
// at 40 columns the pane header truncates and the search silently reported
// zero body rows, which reads as the renderer drawing nothing.
//
// The anchor is the rule line under the header -- a row whose content is
// nothing but "─" -- which is present at every width. The pane header is the
// row after it, and the footer ends the body.
func inspectorBodyLines(rendered string) []string {
	lines := strings.Split(rendered, "\n")
	rule := -1
	for i, line := range lines {
		content := strings.Trim(ansi.Strip(line), "│ ")
		if content != "" && strings.Trim(content, "─") == "" && !strings.ContainsAny(line, "╭╰") {
			rule = i
			break
		}
	}
	if rule < 0 || rule+2 > len(lines) {
		return nil
	}
	body := make([]string, 0, len(lines))
	for _, line := range lines[rule+2:] {
		if strings.Contains(line, "Esc back") || strings.Contains(line, "Esc close") {
			break
		}
		if strings.ContainsAny(line, "╰") {
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
//
// The date rows made this sharper. The header is 5 rows with a date and 6 once
// the author and committer stamps diverge, so a clamp derived from a constant
// four-row header runs ahead of the frame again. The subtests below cover all
// three header shapes; the undated one alone would pass against a constant and
// hide the regression, which is how it got through the first time.
//
// m.width and m.height are set to the values handed to the renderer because
// production does exactly that (view_shell.go:49 passes m.width, m.height), and
// inspectorBodyRowCount reads them off the model.
func TestInspectorBodyRowsMatchesWhatTheFrameKeeps(t *testing.T) {
	const (
		authored  = "2026-09-08T23:15:16+09:00"
		committed = "2026-09-09T01:02:03+09:00"
	)
	for _, tt := range []struct {
		name                   string
		authorDate, commitDate string
	}{
		{"no dates, 4-row header", "", ""},
		{"one date, 5-row header", authored, authored},
		{"diverged dates, 6-row header", authored, committed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, height := range []int{12, 14, 20, 30, 40, 41, 50} {
				m := scrollFixture(500, 3)
				m.commitInspectorSnapshot.AuthorDate = tt.authorDate
				m.commitInspectorSnapshot.CommitDate = tt.commitDate
				m.width, m.height = 120, height
				body := inspectorBodyLines(renderCommitInspectorScreen(m))
				if want := m.inspectorBodyRowCount(); len(body) != want {
					t.Fatalf("height %d rendered %d body rows, inspectorBodyRowCount says %d", height, len(body), want)
				}
			}
		})
	}
}

// The clamp and the page size have to come from the same count the frame keeps,
// or Ctrl+D pages past what was drawn and silently skips diff lines. This is the
// behavioural half of the same invariant.
func TestInspectorScrollPageMatchesTheVisibleRows(t *testing.T) {
	const authored = "2026-09-08T23:15:16+09:00"
	m := scrollFixture(500, 3)
	m.commitInspectorSnapshot.AuthorDate = authored
	m.commitInspectorSnapshot.CommitDate = "2026-09-09T01:02:03+09:00"
	m.width, m.height = 120, 30

	visible := len(inspectorBodyLines(renderCommitInspectorScreen(m)))
	if page := m.inspectorScrollPage(); page != visible {
		t.Fatalf("Ctrl+U/D pages by %d rows but only %d are drawn", page, visible)
	}
	total := len(m.inspectorDiffLines(m.height < 12))
	if want := max(total-visible, 0); m.maxInspectorDiffScroll() != want {
		t.Fatalf("scroll clamp is %d, want %d for %d diff lines over %d visible rows",
			m.maxInspectorDiffScroll(), want, total, visible)
	}
}

// T8 regression. Ctrl+U/Ctrl+D moved commitInspectorScroll but the screen renderer
// indexed from zero, so an 800-line diff showed its first screen and nothing else.
func TestInspectorDiffPaneScrollsWithTheOffset(t *testing.T) {
	m := scrollFixture(500, 1)
	first := renderCommitInspectorScreen(withFrame(m, 120, 40))
	if !strings.Contains(first, "line 1") {
		t.Fatalf("unscrolled pane missing the first row: %q", first)
	}

	m.commitInspectorScroll = 100
	scrolled := renderCommitInspectorScreen(withFrame(m, 120, 40))
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
	// The frame has to be set before the clamp is read: both come from the
	// model, which is the point of taking the size parameters away.
	m := withFrame(scrollFixture(500, 1), 120, 40)
	m.commitInspectorScroll = m.maxInspectorDiffScroll()
	got := renderCommitInspectorScreen(m)
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
		got := renderCommitInspectorScreen(withFrame(m, 120, 40))
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

// T9 regression, the exact shape of the original bug. Pressing ? toggled
// commitInspectorHelp but the screen renderer never read it, so the frame came
// back byte-identical while the modal gate silently ate the next keypress. The
// Inspector looked frozen. Whatever the help says, the screen must change.
func TestInspectorHelpChangesTheScreen(t *testing.T) {
	m := scrollFixture(500, 3)
	closed := renderCommitInspectorScreen(withFrame(m, 120, 30))
	m.commitInspectorHelp = true
	open := renderCommitInspectorScreen(withFrame(m, 120, 30))
	if closed == open {
		t.Fatal("? produced a byte-identical screen while swallowing input")
	}
	if !strings.Contains(open, "Inspector keys") {
		t.Fatalf("help body missing: %q", open)
	}
	if !strings.Contains(open, "? close") {
		t.Fatalf("footer should offer the way out: %q", open)
	}
}

func TestInspectorHelpListsOnlyWorkingKeys(t *testing.T) {
	m := scrollFixture(500, 3)
	m.commitInspectorHelp = true
	got := renderCommitInspectorScreen(withFrame(m, 120, 30))
	for _, want := range []string{"j / k", "Ctrl+U / D", "Esc", "?"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help omitted %q: %q", want, got)
		}
	}
	// q is in the spec but not wired yet (T17). A help screen that names a dead
	// key is a fresh lie, so it must stay out until the binding lands.
	for _, line := range m.inspectorHelpLines() {
		if strings.Contains(line, "q ") || strings.HasPrefix(strings.TrimSpace(line), "q") {
			t.Fatalf("help claims q works: %q", line)
		}
	}
}

func TestInspectorHelpMentionsContinuationOnlyWhenThereIsMore(t *testing.T) {
	m := scrollFixture(500, 3)
	if strings.Contains(strings.Join(m.inspectorHelpLines(), "\n"), "next part") {
		t.Fatal("help offered n with nothing more to load")
	}
	m.commitInspectorDiffWindow.HasMore = true
	if !strings.Contains(strings.Join(m.inspectorHelpLines(), "\n"), "next part") {
		t.Fatal("help omitted n while the diff was partial")
	}
}

// Closing the help must land back on the same view: the spec requires the
// selected file and the scroll position to survive.
func TestInspectorHelpTogglePreservesSelectionAndScroll(t *testing.T) {
	m := scrollFixture(500, 60)
	m.height = 30
	m.commitInspectorCursor = 40
	m.commitInspectorScroll = 120

	opened, _ := m.handleCommitInspectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	withHelp := opened.(model)
	if !withHelp.commitInspectorHelp {
		t.Fatal("? did not open the help")
	}
	closed, _ := withHelp.handleCommitInspectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	back := closed.(model)
	if back.commitInspectorHelp {
		t.Fatal("? did not close the help")
	}
	if back.commitInspectorCursor != 40 || back.commitInspectorScroll != 120 {
		t.Fatalf("toggle lost position: cursor=%d scroll=%d", back.commitInspectorCursor, back.commitInspectorScroll)
	}
}

// Esc with the help open closes the help, not the Inspector, and the keys the
// modal swallows must not change anything behind it.
func TestInspectorHelpEscClosesHelpAndSwallowsQuietly(t *testing.T) {
	m := scrollFixture(500, 60)
	m.height = 30
	m.commitInspectorCursor = 10
	m.commitInspectorScroll = 50
	m.commitInspectorHelp = true

	for _, key := range []tea.KeyMsg{{Type: tea.KeyCtrlD}, {Type: tea.KeyRunes, Runes: []rune("j")}} {
		next, _ := m.handleCommitInspectorKey(key)
		m = next.(model)
	}
	if !m.commitInspectorHelp || m.commitInspectorCursor != 10 || m.commitInspectorScroll != 50 {
		t.Fatalf("keys leaked through the help: help=%v cursor=%d scroll=%d", m.commitInspectorHelp, m.commitInspectorCursor, m.commitInspectorScroll)
	}

	afterEsc, _ := m.handleCommitInspectorKey(tea.KeyMsg{Type: tea.KeyEsc})
	got := afterEsc.(model)
	if got.commitInspectorHelp {
		t.Fatal("esc did not close the help")
	}
	if !got.commitInspectorOpen {
		t.Fatal("esc closed the Inspector instead of the help")
	}
}

// withFrame sets the frame size on the model, which is the only place the
// Inspector reads it from. Tests used to pass a size alongside a model that
// carried a different one; the renderer honoured the argument and the scroll
// clamp honoured the model, and the last diff lines went out of reach.
func withFrame(m model, width, height int) model {
	m.width, m.height = width, height
	return m
}

// A long diff line must be cut at the pane edge, not folded onto the next row.
// Folding breaks the alignment between the two panes and silently changes how
// many diff rows the reader is looking at.
//
// The popup renderer had this test and the screen -- the renderer actually in
// production -- did not. See task 12.
func TestInspectorScreenDoesNotWrapLongDiffCode(t *testing.T) {
	const long = "+a very long line of code that must stay on one terminal row no matter how narrow the pane gets"
	for _, size := range [][2]int{{40, 16}, {60, 20}, {80, 30}} {
		m := model{}
		m.width, m.height = size[0], size[1]
		m.commitInspectorSnapshot = CommitSnapshot{
			FullHash: "abc123", Subject: "change", AuthorName: "dev",
			Files: []ChangedFile{{StableID: "f", Status: StatusModified, Path: "main.go"}},
		}
		m.commitInspectorLines = []string{"@@ -1 +1 @@", "-old", long}

		got := renderCommitInspectorScreen(m)
		if lipgloss.Width(got) != size[0] || lipgloss.Height(got) != size[1] {
			t.Fatalf("%dx%d rendered as %dx%d", size[0], size[1], lipgloss.Width(got), lipgloss.Height(got))
		}
		// A wrapped line puts the tail at the start of a row of its own.
		for _, line := range strings.Split(ansi.Strip(got), "\n") {
			trimmed := strings.TrimLeft(strings.Trim(line, "│ "), " ")
			if strings.HasPrefix(trimmed, "no matter how narrow") {
				t.Fatalf("%dx%d wrapped the long diff line: %q", size[0], size[1], line)
			}
		}
	}
}

// A truncated diff has to say so where it stops. The note used to be a header
// line, so it scrolled out of view long before the reader reached the cut.
func TestPartialDiffSaysSoAtTheEndNotTheTop(t *testing.T) {
	m := scrollFixture(120, 1)
	m.width, m.height = 100, 30
	m.commitInspectorDiffWindow.HasMore = true
	m.commitInspectorDiffWindow.PartialReason = PartialLineLimit

	top := ansi.Strip(renderCommitInspectorScreen(m))
	if strings.Contains(top, "more of this diff follows") {
		t.Error("the note belongs at the cut, not on the first screen of a long diff")
	}

	m.commitInspectorScroll = m.maxInspectorDiffScroll()
	end := ansi.Strip(renderCommitInspectorScreen(m))
	if !strings.Contains(end, "more of this diff follows; press n") {
		t.Errorf("expected the note where the diff stops, got:\n%s", end)
	}

	// The reader is told about "n" from the first frame regardless: that is the
	// footer's job, which is why the header line was a duplicate.
	if !strings.Contains(top, "n next") {
		t.Error("expected the footer to advertise n while more remains")
	}
}

// An internal enum is not user copy, and one of the five reasons means
// something different to the reader than the other four.
func TestTruncationNoteDoesNotLeakTheInternalReason(t *testing.T) {
	for _, reason := range []PartialReason{
		PartialByteLimit, PartialLineLimit, PartialProcessLimit, PartialIndivisiblePair, PartialLineTruncated, "",
	} {
		note := inspectorTruncationNote(reason)
		if reason != "" && strings.Contains(note, string(reason)) {
			t.Errorf("%q leaked into the note: %q", reason, note)
		}
		if !strings.HasPrefix(note, ellipsis) {
			t.Errorf("%q: expected the shared overflow marker, got %q", reason, note)
		}
	}
	// Pressing n cannot recover what is missing from inside a line, so that
	// case must not promise it can.
	if strings.Contains(inspectorTruncationNote(PartialLineTruncated), "press n") {
		t.Error("a truncated line is not recoverable with n; the note must not say it is")
	}
}
