package app

import (
	"reflect"
	"testing"

	"hrllk/graphkeeper/internal/state"
)

func TestShellOverlayStackOrder(t *testing.T) {
	m := model{status: state.New().WithBrowse()}
	overlays := shellOverlayStack(m, 80, 24)
	got := make([]string, 0, len(overlays))
	for _, overlay := range overlays {
		got = append(got, overlay.name)
	}
	want := []string{
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
