package app

import (
	"testing"

	"hrllk/graphkeeper/internal/state"
)

func TestLoadingToastUsesSharedDetail(t *testing.T) {
	tests := []string{
		"Loading...",
		"Fetching for push...",
		"Preparing reset...",
		"Enter a branch name.",
		"Checking out target...",
		"Aborting...",
		"Fetching upstream...",
		"Previewing...",
		"Creating branch...",
		"Running...",
		"Pulling...",
		"Merging pull...",
		"Pushing and tracking...",
		"Force pushing...",
		"Merging...",
		"Rebasing...",
		"Pushing...",
		"Analyzing pull...",
	}

	for _, message := range tests {
		got := loadingToast(message)
		if got.Mode != state.ModeLoading {
			t.Fatalf("expected loading mode for %q, got %s", message, got.Mode)
		}
		if got.Title != "Loading" {
			t.Fatalf("expected loading title for %q, got %q", message, got.Title)
		}
		if got.Message != message {
			t.Fatalf("expected message %q, got %q", message, got.Message)
		}
		if got.Detail != "Please wait." {
			t.Fatalf("expected shared detail for %q, got %q", message, got.Detail)
		}
	}
}
