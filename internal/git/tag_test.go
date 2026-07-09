package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusIncludesTagEntries(t *testing.T) {
	base := t.TempDir()
	work := filepath.Join(base, "work")

	runGitGit(t, base, "init", "-b", "main", "work")
	configGitUser(t, work)
	writeGitFile(t, work, "file.txt", "one\n")
	runGitGit(t, work, "add", "file.txt")
	runGitGitEnv(t, work, map[string]string{
		"GIT_AUTHOR_DATE":    "2026-07-08T10:00:00+09:00",
		"GIT_COMMITTER_DATE": "2026-07-08T10:00:00+09:00",
	}, "commit", "-m", "first")
	first := runGitGit(t, work, "rev-parse", "HEAD")

	writeGitFile(t, work, "file.txt", "two\n")
	runGitGitEnv(t, work, map[string]string{
		"GIT_AUTHOR_DATE":    "2026-07-08T10:05:00+09:00",
		"GIT_COMMITTER_DATE": "2026-07-08T10:05:00+09:00",
	}, "commit", "-am", "second")
	second := runGitGit(t, work, "rev-parse", "HEAD")

	runGitGit(t, work, "tag", "-a", "v1.0.0", "-m", "annotated", first)
	runGitGit(t, work, "tag", "v1.1.0", second)

	repo, err := Open(work)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	status, err := repo.Status(context.Background(), 20)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if len(status.TagEntries) != 2 {
		t.Fatalf("expected 2 tag entries, got %+v", status.TagEntries)
	}
	if status.TagEntries[0].Name != "v1.1.0" || status.TagEntries[0].CommitHash != second {
		t.Fatalf("expected newest tag first, got %+v", status.TagEntries[0])
	}
	if status.TagEntries[1].Name != "v1.0.0" || status.TagEntries[1].CommitHash != first {
		t.Fatalf("expected annotated tag to peel to first commit, got %+v", status.TagEntries[1])
	}
	if len(status.Tags) != 2 || status.Tags[0] != "v1.1.0" || status.Tags[1] != "v1.0.0" {
		t.Fatalf("expected tag name list to mirror entries, got %+v", status.Tags)
	}
}

func TestCreateTagCreatesAndRejectsDuplicates(t *testing.T) {
	base := t.TempDir()
	work := filepath.Join(base, "work")

	runGitGit(t, base, "init", "-b", "main", "work")
	configGitUser(t, work)
	writeGitFile(t, work, "file.txt", "one\n")
	runGitGit(t, work, "add", "file.txt")
	runGitGitEnv(t, work, map[string]string{
		"GIT_AUTHOR_DATE":    "2026-07-08T11:00:00+09:00",
		"GIT_COMMITTER_DATE": "2026-07-08T11:00:00+09:00",
	}, "commit", "-m", "first")
	head := runGitGit(t, work, "rev-parse", "HEAD")

	repo, err := Open(work)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := repo.CreateTag(context.Background(), "v1.0.0", head); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}
	status, err := repo.Status(context.Background(), 20)
	if err != nil {
		t.Fatalf("Status failed after CreateTag: %v", err)
	}
	if len(status.TagEntries) != 1 || status.TagEntries[0].Name != "v1.0.0" || status.TagEntries[0].CommitHash != head {
		t.Fatalf("expected created tag in status, got %+v", status.TagEntries)
	}
	if err := repo.CreateTag(context.Background(), "v1.0.0", head); err == nil {
		t.Fatal("expected duplicate CreateTag to fail")
	}
}

func TestCreateTagRejectsEmptyInputs(t *testing.T) {
	base := t.TempDir()
	work := filepath.Join(base, "work")

	runGitGit(t, base, "init", "-b", "main", "work")
	repo, err := Open(work)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := repo.CreateTag(context.Background(), "", "abc123"); err == nil {
		t.Fatal("expected empty tag name to fail")
	}
	if err := repo.CreateTag(context.Background(), "v1.0.0", ""); err == nil {
		t.Fatal("expected empty tag target to fail")
	}
}

func runGitGitEnv(t *testing.T, dir string, env map[string]string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out.String())
	}
	return strings.TrimSpace(out.String())
}
