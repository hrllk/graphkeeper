package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/git"
)

func TestCommitInspectorQAndEscClose(t *testing.T) {
	m := model{commitInspectorOpen: true}
	next, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if next.(model).commitInspectorOpen {
		t.Fatal("q should close the Inspector before global overlay rewriting")
	}

	m = model{commitInspectorOpen: true}
	next, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(model)
	if got.commitInspectorOpen {
		t.Fatal("Esc should close the Inspector")
	}
}

func TestCommitInspectorKeymapKeepsNavigationInInspector(t *testing.T) {
	m := model{
		commitInspectorOpen: true,
	}
	for _, key := range []string{"tab", "m", "r", "h", "l", "enter"} {
		next, _ := m.handleCommitInspectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		if !next.(model).commitInspectorOpen {
			t.Fatalf("%q should not close Inspector", key)
		}
	}
}

func TestCommitInspectorJKMovesChangedFileSelection(t *testing.T) {
	m := model{
		commitInspectorOpen: true,
		commitInspector: git.CommitInspection{Files: []git.CommitDiffFile{
			{Status: "M", Path: "a.go"},
			{Status: "A", Path: "b.go"},
		}},
		commitInspectorCursor: 0,
	}
	next, cmd := m.handleCommitInspectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	got := next.(model)
	if got.commitInspectorCursor != 1 || cmd == nil {
		t.Fatalf("j should select the next file and request its diff: cursor=%d cmd=%v", got.commitInspectorCursor, cmd == nil)
	}
}

func TestCommitInspectorRendersBorderedTreeAndUnifiedRows(t *testing.T) {
	m := model{
		width: 80, height: 20,
		commitInspectorOpen: true,
		commitInspector: git.CommitInspection{
			Hash: "abc123", Subject: "change", Author: "dev", Parent: "parent",
			Files: []git.CommitDiffFile{{Status: "M", Path: "internal/app/main.go"}},
		},
		commitInspectorLines:  []string{"@@ -1 +1 @@", "-old", "+new"},
		commitInspectorCursor: 0,
	}
	got := m.renderCommitInspectorPopup(80, 20)
	if lipgloss.Width(got) != 80 || lipgloss.Height(got) != 20 {
		t.Fatalf("expected exact frame dimensions, got %dx%d", lipgloss.Width(got), lipgloss.Height(got))
	}
	for _, want := range []string{"commit: abc123", "message: change", "author: dev", "path: internal/app/main.go", "Changed files", "Diff", "@@", "old", "new", "M"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in Inspector frame: %q", want, got)
		}
	}
}

func TestCommitInspectorDiffDoesNotWrapLongCode(t *testing.T) {
	m := model{
		commitInspector: git.CommitInspection{
			Hash: "abc123", Subject: "change", Author: "dev",
			Files: []git.CommitDiffFile{{Status: "M", Path: "main.go"}},
		},
		commitInspectorLines:  []string{"@@ -1 +1 @@", "-old", "+a very long line that must stay on one terminal row"},
		commitInspectorCursor: 0,
	}
	got := m.renderCommitInspectorPopup(60, 20)
	if lipgloss.Width(got) != 60 || lipgloss.Height(got) != 20 {
		t.Fatalf("expected fixed no-wrap frame, got %dx%d", lipgloss.Width(got), lipgloss.Height(got))
	}
	if strings.Contains(got, "\n+a very") {
		t.Fatal("long diff code should be horizontally truncated, not wrapped")
	}
}

func TestCommitInspectorKeepsDividerAlignedWithANSISelectedRow(t *testing.T) {
	m := model{
		commitInspector: git.CommitInspection{Files: []git.CommitDiffFile{
			{Status: "M", Path: "first.go"},
			{Status: "A", Path: "second.go"},
		}},
		commitInspectorCursor: 0,
		commitInspectorLines:  []string{"@@ -1 +1 @@", "-old", "+new"},
	}
	rows := m.renderInspectorBody(70, 8)
	firstSeparator := strings.Index(rows[0], "│")
	if firstSeparator < 0 {
		t.Fatal("header is missing the pane divider")
	}
	wantWidth := lipgloss.Width(rows[0][:firstSeparator])
	for i, row := range rows {
		separator := strings.Index(row, "│")
		if separator < 0 || lipgloss.Width(row[:separator]) != wantWidth {
			t.Fatalf("row %d divider is not aligned by visible width: %q", i, row)
		}
	}
}
