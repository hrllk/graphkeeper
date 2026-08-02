package app

import (
	"reflect"
	"strings"
	"testing"

	"hrllk/graphkeeper/internal/state"
)

func TestOverlayLineRestoresBaseSGRForRightFragment(t *testing.T) {
	base := "\x1b[31mAAAA\x1b[0m"
	got := overlayLine(base, "\x1b[0mXX", 1, 2)
	if strings.Count(got, "\x1b[31m") != 2 {
		t.Fatalf("expected base color to be restored after opaque popup, got %q", got)
	}
	if !strings.Contains(got, "XX\x1b[31mA\x1b[0m") {
		t.Fatalf("expected popup to preserve the right fragment's base color, got %q", got)
	}
}

func TestOverlayLineDoesNotSplitWideRune(t *testing.T) {
	base := "a界b"
	got := overlayLine(base, "XX", 1, 1)
	if !strings.Contains(got, "界") {
		t.Fatalf("expected wide rune to remain intact, got %q", got)
	}
}

func TestShellOverlayStackOrder(t *testing.T) {
	m := model{status: state.New().WithBrowse()}
	overlays := shellOverlayStack(m, 80, 24)
	got := make([]string, 0, len(overlays))
	for _, overlay := range overlays {
		got = append(got, overlay.name)
	}
	want := []string{
		"commit-inspector",
		"confirm",
		"review",
		"reset-mode",
		"cherry-pick",
		"target-pick",
		"branch-input",
		"stash-message",
		"graph-stash-pop",
		"stash-popup",
		"tag-popup",
		"hidden-hotkeys",
		"graph-search",
		"loading",
		"blocked",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected overlay order\n got: %v\nwant: %v", got, want)
	}
}

func TestShellOverlayStackSuppressesLoadingAndBlockedDuringBranchInput(t *testing.T) {
	m := model{
		status:     state.New().WithLoading("Enter a branch name."),
		branchOpen: true,
	}
	overlays := shellOverlayStack(m, 80, 24)
	var loadingActive, blockedActive bool
	for _, overlay := range overlays {
		switch overlay.name {
		case "loading":
			loadingActive = overlay.active
		case "blocked":
			blockedActive = overlay.active
		}
	}
	if loadingActive {
		t.Fatal("expected loading overlay to stay suppressed while branch input is open")
	}
	if blockedActive {
		t.Fatal("expected blocked overlay to stay suppressed while branch input is open")
	}
}
