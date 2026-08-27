package git

import (
	"context"
	"fmt"
	"strconv"
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

// N5 regression. A continuation window used to carry the original hunk header
// verbatim, so the row parser reseeded its cursors from the hunk origin and every
// window after the first renumbered from 1: a row whose content was source line
// 1334 rendered as line 1.
func TestResumeHunkHeaderAdvancesStartsAndCounts(t *testing.T) {
	for _, tt := range []struct {
		name                     string
		header                   string
		oldConsumed, newConsumed int
		want                     string
	}{
		{"two sided", "@@ -1,1999 +1,5000 @@", 1333, 1333, "@@ -1334,666 +1334,3667 @@"},
		{"new file", "@@ -0,0 +1,2001 @@", 0, 2000, "@@ -0,0 +2001,1 @@"},
		{"deleted file", "@@ -1,300 +0,0 @@", 100, 0, "@@ -101,200 +0,0 @@"},
		{"omitted counts mean one line", "@@ -5 +5 @@", 0, 0, "@@ -5,1 +5,1 @@"},
		{"section heading preserved", "@@ -1,5 +1,5 @@ func foo()", 2, 2, "@@ -3,3 +3,3 @@ func foo()"},
		{"consumed is clamped to the hunk", "@@ -1,5 +1,5 @@", 99, 99, "@@ -6,0 +6,0 @@"},
		{"malformed header is left alone", "@@ not a header", 3, 3, "@@ not a header"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := resumeHunkHeader(tt.header, tt.oldConsumed, tt.newConsumed); got != tt.want {
				t.Fatalf("resumeHunkHeader(%q, %d, %d) = %q, want %q", tt.header, tt.oldConsumed, tt.newConsumed, got, tt.want)
			}
		})
	}
}

func TestConsumedHunkLinesCountsPerSide(t *testing.T) {
	lines := []string{" ctx", "-gone", "+new", " ctx", "--- a/file", "+++ b/file", "\\ No newline at end of file"}
	oldAdvance, newAdvance := consumedHunkLines(lines)
	if oldAdvance != 3 || newAdvance != 3 {
		t.Fatalf("consumedHunkLines = (%d, %d), want (3, 3)", oldAdvance, newAdvance)
	}
}

// The continuation window's rows must carry the line numbers they actually
// resume from, not restart at the hunk origin.
func TestCommitDiffWindowContinuationKeepsLineNumbers(t *testing.T) {
	repo, inspection := streamFixtureRepo(t, func(dir string) {
		writeFixtureFile(t, dir, "added.txt", numberedLines("", 250))
		runGitFixture(t, dir, "add", "added.txt")
		runGitFixture(t, dir, "commit", "-m", "add 250 lines")
	})
	file := fileByPath(t, inspection, "added.txt")

	first, err := repo.CommitDiffWindow(context.Background(), inspection, file, 100, 0, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if first.Rows[0].NewLine != 1 {
		t.Fatalf("first window starts at new line %d, want 1", first.Rows[0].NewLine)
	}
	if !first.HasMore {
		t.Fatal("expected a continuation")
	}

	second, err := repo.CommitDiffWindow(context.Background(), inspection, file, 100, first.NextStartLine, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	want := first.NextStartLine + 1
	if second.Rows[0].NewLine != want {
		t.Fatalf("continuation starts at new line %d, want %d", second.Rows[0].NewLine, want)
	}
	if second.Rows[0].OldLine != 0 || second.Rows[0].FromPresent {
		t.Fatalf("an added row must have no old side: %#v", second.Rows[0])
	}
}

// Same requirement when both cursors advance: context and removed lines move the
// old cursor, context and added lines move the new one.
func TestCommitDiffWindowContinuationKeepsBothCursors(t *testing.T) {
	repo, inspection := streamFixtureRepo(t, func(dir string) {
		var before, after strings.Builder
		for i := 1; i <= 200; i++ {
			fmt.Fprintf(&before, "line %d\n", i)
			if i%2 == 0 {
				fmt.Fprintf(&after, "CHANGED %d\n", i)
			} else {
				fmt.Fprintf(&after, "line %d\n", i)
			}
		}
		writeFixtureFile(t, dir, "mixed.txt", before.String())
		runGitFixture(t, dir, "add", "mixed.txt")
		runGitFixture(t, dir, "commit", "-m", "seed mixed")
		writeFixtureFile(t, dir, "mixed.txt", after.String())
		runGitFixture(t, dir, "add", "mixed.txt")
		runGitFixture(t, dir, "commit", "-m", "change every other line")
	})
	file := fileByPath(t, inspection, "mixed.txt")

	first, err := repo.CommitDiffWindow(context.Background(), inspection, file, 60, 0, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !first.HasMore {
		t.Skip("fixture did not produce a continuation; nothing to assert")
	}
	second, err := repo.CommitDiffWindow(context.Background(), inspection, file, 60, first.NextStartLine, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	row := second.Rows[0]
	if row.OldLine <= 1 && row.NewLine <= 1 {
		t.Fatalf("continuation restarted numbering: %#v", row)
	}
	// The content carries its own source line number, so the row must agree.
	for _, text := range []string{row.From, row.To} {
		if text == "" {
			continue
		}
		fields := strings.Fields(text)
		n, convErr := strconv.Atoi(fields[len(fields)-1])
		if convErr != nil {
			continue
		}
		if n != row.OldLine && n != row.NewLine {
			t.Fatalf("row %#v does not match its own content line %d", row, n)
		}
	}
}
