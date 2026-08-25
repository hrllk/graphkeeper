package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func TestTextInputQIsPreservedThroughTopLevelRouting(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*model)
		read  func(model) string
		want  string
	}{
		{"branch", func(m *model) { m.branchOpen = true; m.branchDraft = "feature/" }, func(m model) string { return m.branchDraft }, "feature/q"},
		{"tag", func(m *model) { m.tagPopupOpen = true; m.tagPopupDraft = "release/" }, func(m model) string { return m.tagPopupDraft }, "release/q"},
		{"stash message", func(m *model) { m.stashMessageOpen = true; m.stashMessageDraft = "wip/" }, func(m model) string { return m.stashMessageDraft }, "wip/q"},
		{"graph search", func(m *model) { m.graphSearchOpen = true; m.graphSearchDraft = "fix/" }, func(m model) string { return m.graphSearchDraft }, "fix/q"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				navigationState: navigationState{activeSection: sectionGraph},
				overlayState:    overlayState{},
				status:          state.New().WithBrowse(),
			}
			tt.setup(&m)
			gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
			got := gotModel.(model)
			if cmd != nil {
				t.Fatalf("q input returned command %v", cmd)
			}
			if !got.overlayOpen() {
				t.Fatal("q input closed the text-input modal")
			}
			if value := tt.read(got); value != tt.want {
				t.Fatalf("draft = %q, want %q", value, tt.want)
			}
		})
	}
}

func TestTextInputEscClosesThroughTopLevelRouting(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*model)
		closed func(model) bool
	}{
		{"branch", func(m *model) { m.branchOpen = true; m.branchDraft = "feature/q" }, func(m model) bool { return !m.branchOpen && m.branchDraft == "" }},
		{"tag", func(m *model) { m.tagPopupOpen = true; m.tagPopupDraft = "release/q" }, func(m model) bool { return !m.tagPopupOpen && m.tagPopupDraft == "" }},
		{"stash message", func(m *model) { m.stashMessageOpen = true; m.stashMessageDraft = "wip/q" }, func(m model) bool { return !m.stashMessageOpen && m.stashMessageDraft == "" }},
		{"graph search", func(m *model) { m.graphSearchOpen = true; m.graphSearchDraft = "fix/q" }, func(m model) bool { return !m.graphSearchOpen && m.graphSearchDraft == "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				navigationState: navigationState{activeSection: sectionGraph},
				status:          state.New().WithBrowse(),
			}
			tt.setup(&m)
			gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			got := gotModel.(model)
			if cmd != nil {
				t.Fatalf("Esc close returned command %v", cmd)
			}
			if !tt.closed(got) {
				t.Fatalf("Esc did not clear text-input state: %+v", got)
			}
		})
	}
}

// Keep this contract test focused on top-level routing.
