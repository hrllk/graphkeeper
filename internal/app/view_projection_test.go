package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

func TestRenderGraphProjectionUsesProjectionInput(t *testing.T) {
	projection := GraphProjection{
		Rows: []graph.Row{{
			Commit: graph.Node{Hash: "abcdef12", Author: "author", Subject: "projection subject"},
			Graph:  "*",
		}},
		PageSize: 1,
		Active:   true,
	}

	got := renderGraphProjection(projection, 120, 4)
	if !strings.Contains(got, "abcde") || !strings.Contains(got, "author") {
		t.Fatalf("projection values were not rendered: %q", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if lipgloss.Width(line) > 120 {
			t.Fatalf("line exceeds visible width: %d: %q", lipgloss.Width(line), line)
		}
	}
}

func TestRenderSectionProjectionDoesNotReadModelState(t *testing.T) {
	projection := SectionProjection{
		Items:  []state.TargetItem{{Name: "feature"}},
		Cursor: 0,
		Active: true,
	}

	got := renderSectionProjection(projection, sectionCurrent, true, 40, 3)
	if !strings.Contains(got, "feature") {
		t.Fatalf("projection item was not rendered: %q", got)
	}
}

func TestRenderContextProjectionPreservesViewportOffset(t *testing.T) {
	projection := ContextProjection{
		Title:       "Graph",
		InfoLines:   []string{"first", "second", "third", "fourth"},
		ActionLines: []string{"j/k move"},
		Scroll:      1,
	}

	got := renderContextViewport(projection.InfoLines, 2, projection.Scroll, 60)
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "second") || strings.Contains(joined, "first") {
		t.Fatalf("viewport offset was not applied: %q", joined)
	}
}
