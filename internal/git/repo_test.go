package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenReturnsAbsoluteRoot(t *testing.T) {
	repo, err := Open(".")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if repo.MustRoot() == "" {
		t.Fatal("expected root to be set")
	}
	if !filepath.IsAbs(repo.MustRoot()) {
		t.Fatalf("expected absolute root, got %q", repo.MustRoot())
	}
}

func TestRunnerRejectsUnknownCommand(t *testing.T) {
	r := &Runner{}
	_, err := r.Run("not-a-real-git-command")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsNoCommits(t *testing.T) {
	err := os.ErrNotExist
	if isNoCommits(err) {
		t.Fatal("expected unrelated error to not be treated as no-commits")
	}
}
