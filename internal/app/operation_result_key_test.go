package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"hrllk/graphkeeper/internal/state"
)

func TestOperationResultKeyDismissesWithoutQuitting(t *testing.T) {
	m := model{
		pullState: pullState{operationResult: &OperationResultSummary{Headline: "PULL COMPLETED"}}, status: state.Status{Mode: state.ModeOperationResult}}
	next, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		t.Fatalf("dismiss command = %v, want nil", cmd)
	}
	if next.(model).status.Mode != state.ModeBrowse {
		t.Fatalf("mode = %s, want browse", next.(model).status.Mode)
	}
}
