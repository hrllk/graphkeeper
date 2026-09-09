package app

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// DESIGN.md states the contract as ANSI 0-15 and terminal attributes only, and
// CLAUDE.md sends every visual decision to that file. This asserts the code
// keeps it, so the document cannot drift back into describing a palette the
// terminal never emits.
//
// TrueColor is forced on purpose: at the Ascii profile lipgloss strips
// everything and the test would pass without proving anything.
func TestRenderedShellEmitsNoAnsi256OrTrueColor(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	surfaces := map[string]string{
		"main shell":   widthFixture(120).View(),
		"alert popup":  renderAlertPopup(alertContent{Title: "Alert", Description: "body"}, 80),
		"reset popup":  renderResetModePopup(60),
		"graph header": renderGraphHeader(80, graphCols(4)),
	}
	for name, out := range surfaces {
		for _, seq := range []string{"38;2;", "48;2;", "38;5;", "48;5;"} {
			if strings.Contains(out, seq) {
				t.Fatalf("%s emitted %q, which the ANSI 0-15 contract forbids", name, seq)
			}
		}
	}
}

// theme.go is the only place allowed to build a colour. A style created next to
// its call site is how ANSI-256 got in: eleven of the thirteen offenders were in
// one popup file, invisible from theme.go.
func TestOnlyThemeBuildsColour(t *testing.T) {
	// The guard lives in the architecture package for imports; this is the
	// narrower rule that package cannot express, so it is asserted here against
	// the same source it governs.
	offenders := colourBuildersOutsideTheme(t)
	if len(offenders) > 0 {
		t.Fatalf("colour built outside theme.go: %v", offenders)
	}
}

// The empty tokens are aliases for the terminal default, not accidents. If one
// of them ever gains a colour this fails, which is the prompt to decide whether
// the others should move with it.
func TestDefaultForegroundTokensRenderIdentically(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	const sample = "text"
	want := sample
	for name, style := range map[string]lipgloss.Style{
		"popupBody":    popupBody,
		"popupHelp":    popupHelp,
		"muted":        muted,
		"disabled":     disabled,
		"reviewBranch": reviewBranch,
		"reviewFooter": reviewFooter,
	} {
		if got := style.Render(sample); got != want {
			t.Fatalf("%s is no longer the plain default (%q); decide whether the other aliases move with it", name, got)
		}
	}
}

// colourBuildersOutsideTheme scans the package source for lipgloss.Color, which
// takes an ANSI-256 or hex string. lipgloss.ANSIColor, the 0-15 constructor, is
// allowed and only inside theme.go.
func colourBuildersOutsideTheme(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	var offenders []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if name == "theme.go" {
			continue
		}
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for i, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if strings.Contains(line, "lipgloss.Color(") {
				offenders = append(offenders, fmt.Sprintf("%s:%d", name, i+1))
			}
		}
	}
	return offenders
}
