package git

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// streamFixtureRepo builds a repo whose HEAD commit contains exactly the change
// the caller describes, and returns the opened repo plus the HEAD inspection.
func streamFixtureRepo(t *testing.T, build func(dir string)) (*Repo, CommitInspection) {
	t.Helper()
	dir := t.TempDir()
	runGitFixture(t, dir, "init", "-b", "main")
	runGitFixture(t, dir, "config", "user.name", "Fixture Developer")
	runGitFixture(t, dir, "config", "user.email", "fixture@example.test")
	writeFixtureFile(t, dir, "seed.txt", "seed\n")
	runGitFixture(t, dir, "add", "seed.txt")
	runGitFixture(t, dir, "commit", "-m", "seed")
	build(dir)
	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	hash := strings.TrimSpace(runGitFixture(t, dir, "rev-parse", "HEAD"))
	inspection, err := repo.InspectCommit(context.Background(), hash)
	if err != nil {
		t.Fatal(err)
	}
	return repo, inspection
}

func fileByPath(t *testing.T, inspection CommitInspection, path string) CommitDiffFile {
	t.Helper()
	for _, f := range inspection.Files {
		if f.Path == path {
			return f
		}
	}
	t.Fatalf("file %q not in %#v", path, inspection.Files)
	return CommitDiffFile{}
}

func numberedLines(prefix string, n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "%s%d\n", prefix, i)
	}
	return b.String()
}

// T24 regression. A file added whole is one contiguous run of additions with no
// counterpart to pair against, so the window may stop anywhere inside it. Before
// this fix the run was treated as indivisible and the whole request failed, which
// left the Inspector on "Loading…" forever.
func TestCommitDiffWindowSplitsOneSidedRunOverLineCap(t *testing.T) {
	repo, inspection := streamFixtureRepo(t, func(dir string) {
		writeFixtureFile(t, dir, "added.txt", numberedLines("", 300))
		runGitFixture(t, dir, "add", "added.txt")
		runGitFixture(t, dir, "commit", "-m", "add 300 lines")
	})
	file := fileByPath(t, inspection, "added.txt")

	diff, err := repo.CommitDiffWindow(context.Background(), inspection, file, 100, 0, 1<<20)
	if err != nil {
		t.Fatalf("one-sided run over the cap must not error: %v", err)
	}
	if !diff.HasMore {
		t.Fatal("expected HasMore on a run larger than the cap")
	}
	if diff.PartialReason != "line_limit" {
		t.Fatalf("PartialReason = %q, want line_limit", diff.PartialReason)
	}
	if len(diff.Rows) == 0 {
		t.Fatal("expected the window to carry rows")
	}
	if diff.NextStartLine <= 0 {
		t.Fatalf("NextStartLine = %d, want a resumable offset", diff.NextStartLine)
	}
}

// A deletion of a whole file is the mirror case and must behave the same way.
func TestCommitDiffWindowSplitsOneSidedDeletionOverLineCap(t *testing.T) {
	repo, inspection := streamFixtureRepo(t, func(dir string) {
		writeFixtureFile(t, dir, "doomed.txt", numberedLines("", 300))
		runGitFixture(t, dir, "add", "doomed.txt")
		runGitFixture(t, dir, "commit", "-m", "add doomed")
		runGitFixture(t, dir, "rm", "doomed.txt")
		runGitFixture(t, dir, "commit", "-m", "delete 300 lines")
	})
	file := fileByPath(t, inspection, "doomed.txt")

	diff, err := repo.CommitDiffWindow(context.Background(), inspection, file, 100, 0, 1<<20)
	if err != nil {
		t.Fatalf("one-sided deletion over the cap must not error: %v", err)
	}
	if !diff.HasMore || len(diff.Rows) == 0 {
		t.Fatalf("expected a partial window with rows, got HasMore=%v rows=%d", diff.HasMore, len(diff.Rows))
	}
}

// Continuation must reach the end of a one-sided run that exceeds the cap.
func TestCommitDiffWindowContinuationCoversWholeOneSidedRun(t *testing.T) {
	repo, inspection := streamFixtureRepo(t, func(dir string) {
		writeFixtureFile(t, dir, "added.txt", numberedLines("", 250))
		runGitFixture(t, dir, "add", "added.txt")
		runGitFixture(t, dir, "commit", "-m", "add 250 lines")
	})
	file := fileByPath(t, inspection, "added.txt")

	start, seen, windows := 0, 0, 0
	for {
		diff, err := repo.CommitDiffWindow(context.Background(), inspection, file, 100, start, 1<<20)
		if err != nil {
			t.Fatalf("window at %d: %v", start, err)
		}
		seen += len(diff.Rows)
		windows++
		if !diff.HasMore {
			break
		}
		if diff.NextStartLine <= start {
			t.Fatalf("NextStartLine did not advance: %d after start %d", diff.NextStartLine, start)
		}
		start = diff.NextStartLine
		if windows > 10 {
			t.Fatal("continuation did not terminate")
		}
	}
	if seen != 250 {
		t.Fatalf("continuation covered %d rows across %d windows, want 250", seen, windows)
	}
}

// A two-sided run is different: its removed lines pair with the added lines that
// follow, so cutting inside it would re-pair rows across windows. The window stops
// before the run and says why, instead of splitting it.
func TestCommitDiffWindowDefersTwoSidedRunOverLineCap(t *testing.T) {
	repo, inspection := streamFixtureRepo(t, func(dir string) {
		body := "keep-a\nkeep-b\nkeep-c\n" + numberedLines("orig ", 200)
		writeFixtureFile(t, dir, "rewritten.txt", body)
		runGitFixture(t, dir, "add", "rewritten.txt")
		runGitFixture(t, dir, "commit", "-m", "seed rewritten")
		rewritten := "keep-a\nkeep-b\nkeep-c\n" + numberedLines("NEW ", 200)
		writeFixtureFile(t, dir, "rewritten.txt", rewritten)
		runGitFixture(t, dir, "add", "rewritten.txt")
		runGitFixture(t, dir, "commit", "-m", "rewrite the tail")
	})
	file := fileByPath(t, inspection, "rewritten.txt")

	// The 400-line two-sided run cannot fit a 100-line window, and there is
	// context before it, so the window stops at the run boundary.
	diff, err := repo.CommitDiffWindow(context.Background(), inspection, file, 100, 0, 1<<20)
	if err != nil {
		t.Fatalf("two-sided run should defer, not error: %v", err)
	}
	if !diff.HasMore {
		t.Fatal("expected HasMore when a run is deferred")
	}
	if diff.PartialReason != "indivisible_pair" {
		t.Fatalf("PartialReason = %q, want indivisible_pair", diff.PartialReason)
	}
}
