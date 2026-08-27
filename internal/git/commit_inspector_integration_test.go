package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectCommitUsesChangedFilesAndAuthorEmail(t *testing.T) {
	dir := t.TempDir()
	runGitFixture(t, dir, "init")
	runGitFixture(t, dir, "config", "user.name", "Fixture Developer")
	runGitFixture(t, dir, "config", "user.email", "fixture@example.test")
	if err := os.WriteFile(filepath.Join(dir, "changed.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitFixture(t, dir, "add", "changed.txt")
	runGitFixture(t, dir, "commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(dir, "changed.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "unrelated.txt"), []byte("unrelated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitFixture(t, dir, "add", "changed.txt", "unrelated.txt")
	runGitFixture(t, dir, "commit", "-m", "update")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	hash := strings.TrimSpace(runGitFixture(t, dir, "rev-parse", "HEAD"))
	inspection, err := repo.InspectCommit(context.Background(), hash)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Author != "Fixture Developer <fixture@example.test>" {
		t.Fatalf("unexpected author: %q", inspection.Author)
	}
	if len(inspection.Files) != 2 {
		t.Fatalf("expected only changed files, got %#v", inspection.Files)
	}
	if inspection.Parent == "" || inspection.IsRoot {
		t.Fatalf("expected normal commit parent, got %#v", inspection)
	}

	var changed CommitDiffFile
	for _, file := range inspection.Files {
		if file.Path == "changed.txt" {
			changed = file
		}
	}
	diff, err := repo.CommitDiff(context.Background(), inspection, changed, 2000, 0)
	if err != nil {
		t.Fatal(err)
	}
	rows := ParseDiffRows(diff.Lines)
	if len(rows) != 1 || rows[0].From != "old" || rows[0].To != "new" {
		t.Fatalf("unexpected paired diff: %#v", rows)
	}
}

func runGitFixture(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// T26 regression. The changed-files listing used to pass the commit alone, which
// relies on Git's implicit merge diff semantics: for a merge commit `diff-tree`
// emits nothing, so the Inspector showed an empty Changed files pane while the
// patch command, which does pass <parent> <commit>, had content to show. The
// spec's Git Semantics table requires both to use the explicit first parent.
func TestInspectCommitListsMergeFilesAgainstFirstParent(t *testing.T) {
	dir := t.TempDir()
	runGitFixture(t, dir, "init", "-b", "main")
	runGitFixture(t, dir, "config", "user.name", "Fixture Developer")
	runGitFixture(t, dir, "config", "user.email", "fixture@example.test")
	writeFixtureFile(t, dir, "base.txt", "base\n")
	runGitFixture(t, dir, "add", "base.txt")
	runGitFixture(t, dir, "commit", "-m", "base")
	mainTip := strings.TrimSpace(runGitFixture(t, dir, "rev-parse", "HEAD"))

	runGitFixture(t, dir, "checkout", "-b", "side")
	writeFixtureFile(t, dir, "side.txt", "side\n")
	runGitFixture(t, dir, "add", "side.txt")
	runGitFixture(t, dir, "commit", "-m", "side change")

	runGitFixture(t, dir, "checkout", "main")
	runGitFixture(t, dir, "merge", "--no-ff", "side", "-m", "merge side into main")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	hash := strings.TrimSpace(runGitFixture(t, dir, "rev-parse", "HEAD"))
	inspection, err := repo.InspectCommit(context.Background(), hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.Parents) != 2 {
		t.Fatalf("expected a merge commit with two parents, got %#v", inspection.Parents)
	}
	if inspection.Parent != mainTip {
		t.Fatalf("merge parent = %q, want first parent %q", inspection.Parent, mainTip)
	}
	if len(inspection.Files) != 1 || inspection.Files[0].Path != "side.txt" {
		t.Fatalf("merge commit listed %#v, want side.txt against the first parent", inspection.Files)
	}
	if inspection.Files[0].Status != "A" {
		t.Fatalf("merge file status = %q, want A", inspection.Files[0].Status)
	}
}

// A root commit has no parent and must keep relying on --root.
func TestInspectCommitListsRootCommitFiles(t *testing.T) {
	dir := t.TempDir()
	runGitFixture(t, dir, "init", "-b", "main")
	runGitFixture(t, dir, "config", "user.name", "Fixture Developer")
	runGitFixture(t, dir, "config", "user.email", "fixture@example.test")
	writeFixtureFile(t, dir, "first.txt", "first\n")
	runGitFixture(t, dir, "add", "first.txt")
	runGitFixture(t, dir, "commit", "-m", "root")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	hash := strings.TrimSpace(runGitFixture(t, dir, "rev-parse", "HEAD"))
	inspection, err := repo.InspectCommit(context.Background(), hash)
	if err != nil {
		t.Fatal(err)
	}
	if !inspection.IsRoot || inspection.Parent != "" {
		t.Fatalf("expected a root commit, got %#v", inspection)
	}
	if len(inspection.Files) != 1 || inspection.Files[0].Path != "first.txt" {
		t.Fatalf("root commit listed %#v, want first.txt", inspection.Files)
	}
}

func writeFixtureFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
