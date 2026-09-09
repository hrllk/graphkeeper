package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// 6.3 taught the graph to show its cursor under NO_COLOR and left the rail on a
// lipgloss style, so the two surfaces disagreed about how selection looks and
// the rail's selected row was indistinguishable from the rest. Both now go
// through cursorSignal.
func TestRailCursorSurvivesNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	selected := renderSelectableSectionText("main", true)
	plain := renderSelectableSectionText("main", false)
	if selected == plain {
		t.Fatalf("the rail's selected row must be distinguishable under NO_COLOR: %q", selected)
	}
	if !strings.Contains(selected, cursorSignalPrefix) {
		t.Fatalf("expected reverse video, got %q", selected)
	}
}

// With colour available the rail keeps its existing presentation, so this is a
// NO_COLOR repair rather than a restyle.
func TestRailCursorKeepsItsStyleWhenColourIsAvailable(t *testing.T) {
	withANSIProfile(t)
	selected := renderSelectableSectionText("main", true)
	if strings.Contains(selected, cursorSignalPrefix) {
		t.Fatalf("with colour the rail should use its style, not raw reverse: %q", selected)
	}
	if selected == "main" {
		t.Fatalf("the selected row must still be marked: %q", selected)
	}
}

// The graph and the rail must agree. Asserting them together is the point: this
// drifted once because each surface owned the decision separately.
func TestGraphAndRailAgreeOnTheCursorUnderNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	rail := renderSelectableSectionText("main", true)
	graph := renderGraphHashField("abc12", "", true, graphRowMarks{})
	for name, got := range map[string]string{"rail": rail, "graph": graph} {
		if !strings.Contains(got, cursorSignalPrefix) {
			t.Fatalf("%s lost its cursor signal under NO_COLOR: %q", name, got)
		}
	}
	// Both end their run, so neither leaks its attribute into the next cell.
	for name, got := range map[string]string{"rail": rail, "graph": graph} {
		if !strings.Contains(got, cursorSignalReset) {
			t.Fatalf("%s did not reset its cursor attribute: %q", name, got)
		}
	}
	_ = lipgloss.Width(rail)
}
