package gitpull

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"hrllk/graphkeeper/internal/app"
	"hrllk/graphkeeper/internal/git"
)

func TestPreviewMapsFastForwardAndExactCommits(t *testing.T) {
	repo := testRepo(t)
	got, err := New(repo).Preview(context.Background(), app.PullPreviewRequest{RequestID: 7, RepositoryEpoch: 3, Mode: app.PullModeMerge, CommitLimit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if got.RequestID != 7 || got.RepositoryEpoch != 3 || !got.Eligible || !got.Impact.FastForwardKnown || !got.Impact.IsFastForward {
		t.Fatalf("unexpected preview: %#v", got)
	}
	if len(got.Commits) == 0 {
		t.Fatal("expected preview commits")
	}
}

func TestValidateRejectsChangedBaselineWithoutGitAccess(t *testing.T) {
	adapter := &Adapter{}
	expected := app.PullSnapshotIdentity{Epoch: 4, Branch: "main", Head: "old", TrackingKnown: true, TrackingFresh: true}
	got, err := adapter.Validate(context.Background(), app.PullValidationRequest{RequestID: 2, RepositoryEpoch: 4, Current: app.PullSnapshotIdentity{Epoch: 4, Branch: "main", Head: "new"}, Expected: expected, Mode: app.PullModeMerge})
	if err != nil {
		t.Fatal(err)
	}
	if got.Valid || got.Authorized || got.Reason != app.PullRejectChangedBaseline {
		t.Fatalf("unexpected validation: %#v", got)
	}
}

func TestExecuteRejectsUnauthorizedWithoutMutation(t *testing.T) {
	repo := testRepo(t)
	got, err := New(repo).Execute(context.Background(), app.PullExecutionRequest{RequestID: 1, RepositoryEpoch: 1, Mode: app.PullModeMerge})
	if err != nil {
		t.Fatal(err)
	}
	if got.Succeeded || got.Reason != app.PullRejectNotEligible {
		t.Fatalf("unexpected execution: %#v", got)
	}
}

func testRepo(t *testing.T) *git.Repo {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	if err := writeFile(dir+"/file", "one\n"); err != nil {
		t.Fatal(err)
	}
	run("add", "file")
	run("commit", "-m", "initial")
	remote := t.TempDir()
	c := exec.Command("git", "init", "--bare", remote)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("remote: %v: %s", err, out)
	}
	run("remote", "add", "origin", remote)
	run("push", "-u", "origin", "main")
	other := t.TempDir()
	c = exec.Command("git", "clone", remote, other)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("clone: %v: %s", err, out)
	}
	c = exec.Command("git", "-C", other, "config", "user.email", "test@example.com")
	_ = c.Run()
	c = exec.Command("git", "-C", other, "config", "user.name", "Test")
	_ = c.Run()
	if err := writeFile(other+"/file", "two\n"); err != nil {
		t.Fatal(err)
	}
	c = exec.Command("git", "-C", other, "add", "file")
	_ = c.Run()
	c = exec.Command("git", "-C", other, "commit", "-m", "remote")
	_ = c.Run()
	c = exec.Command("git", "-C", other, "push")
	_ = c.Run()
	repo, err := git.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func writeFile(path, content string) error { return os.WriteFile(path, []byte(content), 0o644) }
