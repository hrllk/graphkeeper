package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommitInspectorQAndEscClose(t *testing.T) {
	m := model{inspectorState: inspectorState{commitInspectorOpen: true}}
	next, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil || !next.(model).commitInspectorOpen {
		t.Fatal("q should remain a no-op in the Inspector")
	}

	m = model{inspectorState: inspectorState{commitInspectorOpen: true}}
	next, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(model)
	if got.commitInspectorOpen {
		t.Fatal("Esc should close the Inspector")
	}
}

func TestCommitInspectorKeymapKeepsNavigationInInspector(t *testing.T) {
	m := model{
		inspectorState: inspectorState{
			commitInspectorOpen: true,
		},
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
		inspectorState: inspectorState{
			commitInspectorOpen:   true,
			commitInspectorCursor: 0,
			commitInspector: CommitSnapshot{Files: []ChangedFile{
				{Status: "M", Path: "a.go"},
				{Status: "A", Path: "b.go"},
			}},
		},
	}
	next, cmd := m.handleCommitInspectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	got := next.(model)
	if got.commitInspectorCursor != 1 || cmd == nil {
		t.Fatalf("j should select the next file and request its diff: cursor=%d cmd=%v", got.commitInspectorCursor, cmd == nil)
	}
}
