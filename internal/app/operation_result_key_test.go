package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"hrllk/graphkeeper/internal/state"
)

func TestOperationResultQIsNoOpAndEscDismissesWithoutQuitting(t *testing.T) {
	m := model{
		pullState: pullState{operationResult: &OperationResultSummary{Headline: "PULL COMPLETED"}}, status: state.Status{Mode: state.ModeOperationResult}}
	next, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		t.Fatalf("q command = %v, want nil", cmd)
	}
	if next.(model).status.Mode != state.ModeOperationResult {
		t.Fatalf("q mode = %s, want operation result", next.(model).status.Mode)
	}

	next, cmd = next.(model).handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("Esc command = %v, want nil", cmd)
	}
	if next.(model).status.Mode != state.ModeBrowse {
		t.Fatalf("Esc mode = %s, want browse", next.(model).status.Mode)
	}
}
