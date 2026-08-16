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
