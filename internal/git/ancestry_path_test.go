package git

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// Drives real git rather than asserting on a mock, because the three empty
// cases are a property of rev-list, not of this wrapper.
func TestAncestryPathAgainstRealGit(t *testing.T) {
	base := t.TempDir()
	work := filepath.Join(base, "work")
	runGitGit(t, base, "init", "-b", "main", "work")
	configGitUser(t, work)

	commit := func(name string) string {
		writeGitFile(t, work, name+".txt", name+"\n")
		runGitGit(t, work, "add", name+".txt")
		runGitGit(t, work, "commit", "-m", name)
		return strings.TrimSpace(runGitGit(t, work, "rev-parse", "HEAD"))
	}

	first := commit("first")
	second := commit("second")
	third := commit("third")

	// A side branch off first, so it and third have no ancestry either way.
	runGitGit(t, work, "checkout", "-b", "side", first)
	side := commit("side")

	repo, err := Open(work)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ctx := context.Background()

	// from is excluded, to is included, newest first.
	got, err := repo.AncestryPath(ctx, first, third)
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected two commits between first and third, got %d: %v", len(got), got)
	}
	if got[0] != third {
		t.Fatalf("expected newest first, got %q want %q", got[0], third)
	}
	if got[1] != second {
		t.Fatalf("expected second in the middle, got %q", got[1])
	}
	for _, hash := range got {
		if hash == first {
			t.Fatal("from must be excluded from the result")
		}
	}

	// Empty case 1: same commit.
	if got, err := repo.AncestryPath(ctx, third, third); err != nil || len(got) != 0 {
		t.Fatalf("same commit: got %v err %v, want empty and nil", got, err)
	}
	// Empty case 2: reversed direction. A path exists, just not this way.
	if got, err := repo.AncestryPath(ctx, third, first); err != nil || len(got) != 0 {
		t.Fatalf("reversed: got %v err %v, want empty and nil", got, err)
	}
	// Empty case 3: genuinely diverged, empty in both directions.
	if got, err := repo.AncestryPath(ctx, side, third); err != nil || len(got) != 0 {
		t.Fatalf("diverged forward: got %v err %v", got, err)
	}
	if got, err := repo.AncestryPath(ctx, third, side); err != nil || len(got) != 0 {
		t.Fatalf("diverged backward: got %v err %v", got, err)
	}
}

// git reads `..to` as `HEAD..to`, so a blank ref answers a different question
// and reports no error. The guard is the only thing that makes that loud.
func TestAncestryPathRejectsBlankRefs(t *testing.T) {
	base := t.TempDir()
	work := filepath.Join(base, "work")
	runGitGit(t, base, "init", "-b", "main", "work")
	configGitUser(t, work)
	writeGitFile(t, work, "a.txt", "a\n")
	runGitGit(t, work, "add", "a.txt")
	runGitGit(t, work, "commit", "-m", "a")

	repo, err := Open(work)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	head := strings.TrimSpace(runGitGit(t, work, "rev-parse", "HEAD"))
	for _, tt := range []struct{ from, to string }{{"", head}, {head, ""}, {"", ""}} {
		if _, err := repo.AncestryPath(context.Background(), tt.from, tt.to); err == nil {
			t.Fatalf("AncestryPath(%q, %q) must reject a blank ref", tt.from, tt.to)
		}
	}
}
