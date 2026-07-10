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

func TestStatusDoesNotAutoLoadTagEntries(t *testing.T) {
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
	if len(status.TagEntries) != 0 {
		t.Fatalf("expected status to skip tag auto-load, got %+v", status.TagEntries)
	}
	if len(status.Tags) != 0 {
		t.Fatalf("expected status tag names to stay empty without manual fetch, got %+v", status.Tags)
	}
	entries, err := repo.TagEntries(context.Background())
	if err != nil {
		t.Fatalf("TagEntries failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 tag entries, got %+v", entries)
	}
	if entries[0].Name != "v1.1.0" || entries[0].CommitHash != second {
		t.Fatalf("expected newest tag first, got %+v", entries[0])
	}
	if entries[0].Annotated {
		t.Fatalf("expected lightweight tag to stay unannotated, got %+v", entries[0])
	}
	if entries[0].Tagger != "lightweight" {
		t.Fatalf("expected lightweight tag marker, got %+v", entries[0])
	}
	if entries[1].Name != "v1.0.0" || entries[1].CommitHash != first {
		t.Fatalf("expected annotated tag to peel to first commit, got %+v", entries[1])
	}
	if !entries[1].Annotated {
		t.Fatalf("expected annotated tag metadata to be detected, got %+v", entries[1])
	}
	if entries[1].Tagger == "" {
		t.Fatalf("expected annotated tag to include tagger metadata, got %+v", entries[1])
	}
	if entries[1].TaggedAt.IsZero() {
		t.Fatalf("expected annotated tag to include tagged time, got %+v", entries[1])
	}
}

func TestLocalTagEntriesReadsLocalRefsWithoutOriginFlags(t *testing.T) {
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
	head := runGitGit(t, work, "rev-parse", "HEAD")
	runGitGit(t, work, "tag", "v1.0.0", head)

	repo, err := Open(work)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	entries, err := repo.LocalTagEntries(context.Background())
	if err != nil {
		t.Fatalf("LocalTagEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one local tag entry, got %+v", entries)
	}
	if entries[0].OriginKnown {
		t.Fatalf("expected local tag read to keep origin unknown, got %+v", entries[0])
	}
}

func TestOriginTagSetReadsRemoteTags(t *testing.T) {
	base := t.TempDir()
	origin := filepath.Join(base, "origin.git")
	work := filepath.Join(base, "work")

	runGitGit(t, base, "init", "--bare", origin)
	runGitGit(t, base, "init", "-b", "main", "work")
	configGitUser(t, work)
	writeGitFile(t, work, "file.txt", "one\n")
	runGitGit(t, work, "add", "file.txt")
	runGitGitEnv(t, work, map[string]string{
		"GIT_AUTHOR_DATE":    "2026-07-08T12:00:00+09:00",
		"GIT_COMMITTER_DATE": "2026-07-08T12:00:00+09:00",
	}, "commit", "-m", "first")
	head := runGitGit(t, work, "rev-parse", "HEAD")
	runGitGit(t, work, "remote", "add", "origin", origin)
	runGitGit(t, work, "push", "-u", "origin", "main")
	runGitGit(t, work, "tag", "v1.0.0", head)
	runGitGit(t, work, "push", "origin", "v1.0.0")

	repo, err := Open(work)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	tags, err := repo.OriginTagSet(context.Background())
	if err != nil {
		t.Fatalf("OriginTagSet failed: %v", err)
	}
	if !tags["v1.0.0"] {
		t.Fatalf("expected remote tag set to include v1.0.0, got %+v", tags)
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
	if len(status.TagEntries) != 0 {
		t.Fatalf("expected status to skip tag auto-load after create, got %+v", status.TagEntries)
	}
	entries, err := repo.TagEntries(context.Background())
	if err != nil {
		t.Fatalf("TagEntries failed after create: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "v1.0.0" || entries[0].CommitHash != head {
		t.Fatalf("expected created tag via manual load, got %+v", entries)
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

func TestDeleteTagRemovesLocalRef(t *testing.T) {
	base := t.TempDir()
	work := filepath.Join(base, "work")

	runGitGit(t, base, "init", "-b", "main", "work")
	configGitUser(t, work)
	writeGitFile(t, work, "file.txt", "one\n")
	runGitGit(t, work, "add", "file.txt")
	runGitGitEnv(t, work, map[string]string{
		"GIT_AUTHOR_DATE":    "2026-07-08T12:00:00+09:00",
		"GIT_COMMITTER_DATE": "2026-07-08T12:00:00+09:00",
	}, "commit", "-m", "first")
	head := runGitGit(t, work, "rev-parse", "HEAD")
	runGitGit(t, work, "tag", "v1.0.0", head)

	repo, err := Open(work)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if _, err := repo.DeleteTag(context.Background(), "v1.0.0"); err != nil {
		t.Fatalf("DeleteTag failed: %v", err)
	}
	status, err := repo.Status(context.Background(), 20)
	if err != nil {
		t.Fatalf("Status failed after DeleteTag: %v", err)
	}
	if len(status.TagEntries) != 0 {
		t.Fatalf("expected deleted tag to stay out of cached status, got %+v", status.TagEntries)
	}
	entries, err := repo.TagEntries(context.Background())
	if err != nil {
		t.Fatalf("TagEntries failed after DeleteTag: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected tag list to be empty after delete, got %+v", entries)
	}
}

func TestStatusMarksOriginTags(t *testing.T) {
	base := t.TempDir()
	origin := filepath.Join(base, "origin.git")
	work := filepath.Join(base, "work")

	runGitGit(t, base, "init", "--bare", origin)
	runGitGit(t, base, "init", "-b", "main", "work")
	configGitUser(t, work)
	writeGitFile(t, work, "file.txt", "one\n")
	runGitGit(t, work, "add", "file.txt")
	runGitGitEnv(t, work, map[string]string{
		"GIT_AUTHOR_DATE":    "2026-07-08T12:00:00+09:00",
		"GIT_COMMITTER_DATE": "2026-07-08T12:00:00+09:00",
	}, "commit", "-m", "first")
	head := runGitGit(t, work, "rev-parse", "HEAD")
	runGitGit(t, work, "remote", "add", "origin", origin)
	runGitGit(t, work, "push", "-u", "origin", "main")
	runGitGit(t, work, "tag", "v1.0.0", head)
	runGitGit(t, work, "push", "origin", "v1.0.0")

	repo, err := Open(work)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	status, err := repo.Status(context.Background(), 20)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if len(status.TagEntries) != 0 {
		t.Fatalf("expected status to skip tag auto-load, got %+v", status.TagEntries)
	}
	entries, err := repo.TagEntries(context.Background())
	if err != nil {
		t.Fatalf("TagEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one tag entry, got %+v", entries)
	}
	if !entries[0].OnOrigin {
		t.Fatalf("expected tag to be marked as present on origin, got %+v", entries[0])
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
