package app

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

type commandRepoFixture struct {
	root        string
	remote      string
	repo        *git.Repo
	initialHash string
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out.String())
	}
	return strings.TrimSpace(out.String())
}

func runGitExpectError(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected git %v to fail", args)
	}
	return strings.TrimSpace(out.String())
}

func writeRepoFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s failed: %v", path, err)
	}
}

func configUser(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "user.email", "test@example.com")
}

func checkoutMainTrackingOrigin(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "checkout", "-B", "main", "origin/main")
}

func newCommandRepo(t *testing.T) commandRepoFixture {
	t.Helper()
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	work := filepath.Join(base, "work")

	runGit(t, base, "init", "--bare", "remote.git")
	runGit(t, base, "init", "-b", "main", "work")
	configUser(t, work)
	writeRepoFile(t, work, "file.txt", "base\n")
	runGit(t, work, "add", "file.txt")
	runGit(t, work, "commit", "-m", "initial")
	runGit(t, work, "remote", "add", "origin", remote)
	runGit(t, work, "push", "-u", "origin", "main")

	initialHash := runGit(t, work, "rev-parse", "HEAD")
	repo, err := git.Open(work)
	if err != nil {
		t.Fatalf("git.Open failed: %v", err)
	}
	return commandRepoFixture{root: work, remote: remote, repo: repo, initialHash: initialHash}
}

func cloneRepoAtHash(t *testing.T, remote, hash string) commandRepoFixture {
	t.Helper()
	base := t.TempDir()
	clone := filepath.Join(base, "clone")
	runGit(t, base, "clone", remote, "clone")
	checkoutMainTrackingOrigin(t, clone)
	configUser(t, clone)
	runGit(t, clone, "reset", "--hard", hash)
	repo, err := git.Open(clone)
	if err != nil {
		t.Fatalf("git.Open failed: %v", err)
	}
	return commandRepoFixture{root: clone, remote: remote, repo: repo, initialHash: hash}
}

func advanceRemote(t *testing.T, remote, fileName, content, commitMessage string) string {
	t.Helper()
	base := t.TempDir()
	clone := filepath.Join(base, "clone")
	runGit(t, base, "clone", remote, "clone")
	checkoutMainTrackingOrigin(t, clone)
	configUser(t, clone)
	writeRepoFile(t, clone, fileName, content)
	runGit(t, clone, "add", fileName)
	runGit(t, clone, "commit", "-m", commitMessage)
	runGit(t, clone, "push", "origin", "main")
	return runGit(t, clone, "rev-parse", "HEAD")
}

func advanceRemoteBranch(t *testing.T, remote, branch, fileName, content, commitMessage string) string {
	t.Helper()
	base := t.TempDir()
	clone := filepath.Join(base, "clone")
	runGit(t, base, "clone", remote, "clone")
	configUser(t, clone)
	runGit(t, clone, "checkout", "-b", branch)
	writeRepoFile(t, clone, fileName, content)
	runGit(t, clone, "add", fileName)
	runGit(t, clone, "commit", "-m", commitMessage)
	runGit(t, clone, "push", "-u", "origin", branch)
	return runGit(t, clone, "rev-parse", "HEAD")
}

func makeLocalCommit(t *testing.T, dir, fileName, content, commitMessage string) string {
	t.Helper()
	writeRepoFile(t, dir, fileName, content)
	runGit(t, dir, "add", fileName)
	runGit(t, dir, "commit", "-m", commitMessage)
	return runGit(t, dir, "rev-parse", "HEAD")
}

func cmdResult(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected command")
	}
	return cmd()
}

func TestLoadAndRefreshRepoState(t *testing.T) {
	fixture := newCommandRepo(t)

	loaded, ok := cmdResult(t, loadRepoState(fixture.repo, 40)).(loadedMsg)
	if !ok {
		t.Fatalf("expected loadedMsg, got %T", cmdResult(t, loadRepoState(fixture.repo, 40)))
	}
	if loaded.err != nil {
		t.Fatalf("loadRepoState err = %v", loaded.err)
	}
	if loaded.status.Branch != "main" || loaded.status.Root == "" {
		t.Fatalf("unexpected loaded status: %+v", loaded.status)
	}

	refreshed, ok := cmdResult(t, refreshRepoState(fixture.repo, 40)).(refreshedMsg)
	if !ok {
		t.Fatalf("expected refreshedMsg, got %T", cmdResult(t, refreshRepoState(fixture.repo, 40)))
	}
	if refreshed.err != nil {
		t.Fatalf("refreshRepoState err = %v", refreshed.err)
	}
	if refreshed.status.Branch != "main" {
		t.Fatalf("unexpected refreshed status: %+v", refreshed.status)
	}
}

func TestLoadStashState(t *testing.T) {
	fixture := newCommandRepo(t)
	writeRepoFile(t, fixture.root, "stash.txt", "stash\n")
	runGit(t, fixture.root, "add", "stash.txt")
	runGit(t, fixture.root, "stash", "push", "-m", "wip stash")

	got, ok := cmdResult(t, loadStashState(fixture.repo)).(stashLoadedMsg)
	if !ok {
		t.Fatalf("expected stashLoadedMsg, got %T", cmdResult(t, loadStashState(fixture.repo)))
	}
	if got.err != nil {
		t.Fatalf("loadStashState err = %v", got.err)
	}
	if len(got.entries) != 1 {
		t.Fatalf("expected one stash entry, got %+v", got.entries)
	}
	if got.entries[0].BaseHash == "" || got.entries[0].Ref == "" {
		t.Fatalf("expected stash entry to include base hash and ref, got %+v", got.entries[0])
	}
}

func TestExecuteStashPopRefreshesStashList(t *testing.T) {
	fixture := newCommandRepo(t)
	writeRepoFile(t, fixture.root, "stash.txt", "stash\n")
	runGit(t, fixture.root, "add", "stash.txt")
	runGit(t, fixture.root, "stash", "push", "-m", "wip stash")

	stashes, err := fixture.repo.Stashes(context.Background())
	if err != nil {
		t.Fatalf("repo.Stashes err = %v", err)
	}
	if len(stashes) != 1 {
		t.Fatalf("expected one stash entry, got %+v", stashes)
	}

	got, ok := cmdResult(t, executeStashPop(fixture.repo, 40, stashes[0])).(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeStashPop(fixture.repo, 40, stashes[0])))
	}
	if got.action != state.ActionStashPop {
		t.Fatalf("expected stash pop action, got %s", got.action)
	}
	if got.err != nil {
		t.Fatalf("executeStashPop err = %v", got.err)
	}
	if got.status.Root == "" {
		t.Fatalf("expected refreshed status after stash pop, got %+v", got.status)
	}

	remaining, err := fixture.repo.Stashes(context.Background())
	if err != nil {
		t.Fatalf("repo.Stashes after pop err = %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected stash list to be empty after pop, got %+v", remaining)
	}
}

func TestExecuteStashAllIncludesUntracked(t *testing.T) {
	fixture := newCommandRepo(t)
	writeRepoFile(t, fixture.root, "tracked.txt", "tracked\n")
	runGit(t, fixture.root, "add", "tracked.txt")
	writeRepoFile(t, fixture.root, "untracked.txt", "untracked\n")

	got, ok := cmdResult(t, executeStashAll(fixture.repo, 40, "graphkeeper: local cleanup")).(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeStashAll(fixture.repo, 40, "graphkeeper: local cleanup")))
	}
	if got.action != state.ActionStash {
		t.Fatalf("expected stash action, got %s", got.action)
	}
	if got.err != nil {
		t.Fatalf("executeStashAll err = %v", got.err)
	}
	if got.status.WorktreeDirty {
		t.Fatalf("expected stash to leave worktree clean, got %+v", got.status)
	}
	if len(got.status.LocalBranches) == 0 {
		t.Fatalf("expected status refresh after stash, got %+v", got.status)
	}
	stashes, err := fixture.repo.Stashes(context.Background())
	if err != nil {
		t.Fatalf("repo.Stashes err = %v", err)
	}
	if len(stashes) != 1 {
		t.Fatalf("expected one stash entry, got %+v", stashes)
	}
}

func TestExecuteCleanWorkingTreeRemovesTrackedAndUntracked(t *testing.T) {
	fixture := newCommandRepo(t)
	writeRepoFile(t, fixture.root, "file.txt", "changed\n")
	writeRepoFile(t, fixture.root, "untracked.txt", "temp\n")

	got, ok := cmdResult(t, executeCleanWorkingTree(fixture.repo, 40, false)).(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeCleanWorkingTree(fixture.repo, 40, false)))
	}
	if got.action != state.ActionCleanWorkingTree {
		t.Fatalf("expected clean action, got %s", got.action)
	}
	if got.err != nil {
		t.Fatalf("executeCleanWorkingTree err = %v", got.err)
	}
	if got.status.WorktreeDirty {
		t.Fatalf("expected clean to leave worktree clean, got %+v", got.status)
	}
	if _, err := os.Stat(filepath.Join(fixture.root, "untracked.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected untracked file to be removed, stat err=%v", err)
	}
}

func TestCleanWorkingTreeCanRemoveIgnoredFiles(t *testing.T) {
	fixture := newCommandRepo(t)
	writeRepoFile(t, fixture.root, ".gitignore", "ignored.txt\n")
	writeRepoFile(t, fixture.root, "ignored.txt", "ignored\n")

	if err := fixture.repo.CleanWorkingTree(context.Background(), true); err != nil {
		t.Fatalf("CleanWorkingTree(includeIgnored=true) err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(fixture.root, "ignored.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected ignored file to be removed, stat err=%v", err)
	}
}

func TestFetchAndPrepareState(t *testing.T) {
	fixture := newCommandRepo(t)
	advanceRemote(t, fixture.remote, "remote.txt", "remote\n", "remote advance")

	fetched, ok := cmdResult(t, fetchRepoState(fixture.repo, 40)).(fetchedMsg)
	if !ok {
		t.Fatalf("expected fetchedMsg, got %T", cmdResult(t, fetchRepoState(fixture.repo, 40)))
	}
	if fetched.err != nil {
		t.Fatalf("fetchRepoState err = %v", fetched.err)
	}
	if len(fetched.status.RemoteBranches) == 0 {
		t.Fatalf("expected remote branches after fetch, got %+v", fetched.status)
	}

	prepared, ok := cmdResult(t, prepareAction(fixture.repo, state.ActionMerge, 40)).(preparedMsg)
	if !ok {
		t.Fatalf("expected preparedMsg, got %T", cmdResult(t, prepareAction(fixture.repo, state.ActionMerge, 40)))
	}
	if prepared.err != nil {
		t.Fatalf("prepareAction err = %v", prepared.err)
	}
	if prepared.action != state.ActionMerge {
		t.Fatalf("unexpected prepared action: %s", prepared.action)
	}
}

func TestPullCheck(t *testing.T) {
	t.Run("fast-forward possible", func(t *testing.T) {
		fixture := newCommandRepo(t)
		advanceRemote(t, fixture.remote, "remote.txt", "remote\n", "remote advance")
		got, ok := cmdResult(t, pullCheck(fixture.repo, 40)).(pullCheckedMsg)
		if !ok {
			t.Fatalf("expected pullCheckedMsg, got %T", cmdResult(t, pullCheck(fixture.repo, 40)))
		}
		if got.err != nil {
			t.Fatalf("pullCheck err = %v", got.err)
		}
		if got.status.Mode != state.ModeOutcomePreview || got.status.Action != state.ActionPull {
			t.Fatalf("unexpected pullCheck status: %+v", got.status)
		}
	})

	t.Run("diverged blocks fast-forward", func(t *testing.T) {
		fixture := newCommandRepo(t)
		makeLocalCommit(t, fixture.root, "local.txt", "local\n", "local ahead")
		got, ok := cmdResult(t, pullCheck(fixture.repo, 40)).(pullCheckedMsg)
		if !ok {
			t.Fatalf("expected pullCheckedMsg, got %T", cmdResult(t, pullCheck(fixture.repo, 40)))
		}
		if got.err != nil {
			t.Fatalf("pullCheck err = %v", got.err)
		}
		if got.status.Mode != state.ModeBlocked || got.status.Block != state.BlockDiverged {
			t.Fatalf("expected divergence block, got %+v", got.status)
		}
	})
}

func TestExecutePullVariants(t *testing.T) {
	fixture := newCommandRepo(t)
	remoteHead := advanceRemote(t, fixture.remote, "remote.txt", "remote\n", "remote advance")

	tests := []struct {
		name   string
		action state.Action
	}{
		{name: "pull", action: state.ActionPull},
		{name: "pull merge", action: state.ActionPullMerge},
		{name: "pull rebase", action: state.ActionPullRebase},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clone := cloneRepoAtHash(t, fixture.remote, fixture.initialHash)
			var cmd tea.Cmd
			switch tt.action {
			case state.ActionPull:
				cmd = executePull(clone.repo, 40)
			case state.ActionPullMerge:
				cmd = executePullMerge(clone.repo, 40)
			case state.ActionPullRebase:
				cmd = executePullRebase(clone.repo, 40)
			}
			got, ok := cmdResult(t, cmd).(executedMsg)
			if !ok {
				t.Fatalf("expected executedMsg, got %T", cmdResult(t, cmd))
			}
			if got.action != tt.action {
				t.Fatalf("action = %s, want %s", got.action, tt.action)
			}
			if got.err != nil {
				t.Fatalf("pull variant err = %v", got.err)
			}
			if got.status.Head != remoteHead {
				t.Fatalf("expected HEAD %q, got %q", remoteHead, got.status.Head)
			}
		})
	}
}

func TestExecuteAbortKeepsMergeAndRebaseSplit(t *testing.T) {
	t.Run("merge abort", func(t *testing.T) {
		fixture := newCommandRepo(t)
		runGit(t, fixture.root, "checkout", "-b", "feature")
		makeLocalCommit(t, fixture.root, "file.txt", "feature\n", "feature change")
		runGit(t, fixture.root, "checkout", "main")
		makeLocalCommit(t, fixture.root, "file.txt", "main\n", "main change")
		runGitExpectError(t, fixture.root, "merge", "feature")

		got, ok := cmdResult(t, executeAbort(fixture.repo, 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeAbort(fixture.repo, 40)))
		}
		if got.action != state.ActionAbort {
			t.Fatalf("action = %s, want abort", got.action)
		}
		if got.err != nil {
			t.Fatalf("merge abort err = %v", got.err)
		}
		if got.status.MergeInProgress || got.status.RebaseInProgress {
			t.Fatalf("expected abort to clear in-progress state, got %+v", got.status)
		}
	})

	t.Run("rebase abort", func(t *testing.T) {
		fixture := newCommandRepo(t)
		runGit(t, fixture.root, "checkout", "-b", "feature")
		makeLocalCommit(t, fixture.root, "file.txt", "feature\n", "feature change")
		runGit(t, fixture.root, "checkout", "main")
		makeLocalCommit(t, fixture.root, "file.txt", "main\n", "main change")
		runGit(t, fixture.root, "checkout", "feature")
		runGitExpectError(t, fixture.root, "rebase", "main")

		got, ok := cmdResult(t, executeAbort(fixture.repo, 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeAbort(fixture.repo, 40)))
		}
		if got.action != state.ActionAbort {
			t.Fatalf("action = %s, want abort", got.action)
		}
		if got.err != nil {
			t.Fatalf("rebase abort err = %v", got.err)
		}
		if got.status.MergeInProgress || got.status.RebaseInProgress {
			t.Fatalf("expected abort to clear in-progress state, got %+v", got.status)
		}
	})
}

func TestExecutePushVariants(t *testing.T) {
	t.Run("push", func(t *testing.T) {
		fixture := newCommandRepo(t)
		newHead := makeLocalCommit(t, fixture.root, "file.txt", "push\n", "local push")
		got, ok := cmdResult(t, executePush(fixture.repo, "main", 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executePush(fixture.repo, "main", 40)))
		}
		if got.action != state.ActionPush {
			t.Fatalf("action = %s, want push", got.action)
		}
		if got.err != nil {
			t.Fatalf("push err = %v", got.err)
		}
		if got.status.Head != newHead {
			t.Fatalf("expected pushed HEAD %q, got %q", newHead, got.status.Head)
		}
	})

	t.Run("force push overwrites remote", func(t *testing.T) {
		fixture := newCommandRepo(t)
		advanceRemote(t, fixture.remote, "remote.txt", "remote\n", "remote advance")
		localHead := makeLocalCommit(t, fixture.root, "file.txt", "rewrite\n", "local rewrite")
		got, ok := cmdResult(t, executeForcePush(fixture.repo, "main", 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeForcePush(fixture.repo, "main", 40)))
		}
		if got.action != state.ActionForcePush {
			t.Fatalf("action = %s, want force-push", got.action)
		}
		if got.err != nil {
			t.Fatalf("force push err = %v", got.err)
		}
		remoteHead := runGit(t, fixture.remote, "rev-parse", "main")
		if remoteHead != localHead {
			t.Fatalf("expected remote head %q after force push, got %q", localHead, remoteHead)
		}
	})

	t.Run("set upstream", func(t *testing.T) {
		fixture := newCommandRepo(t)
		runGit(t, fixture.root, "checkout", "-b", "feature")
		newHead := makeLocalCommit(t, fixture.root, "feature.txt", "feature\n", "feature commit")
		got, ok := cmdResult(t, executePushSetUpstream(fixture.repo, "feature", 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executePushSetUpstream(fixture.repo, "feature", 40)))
		}
		if got.action != state.ActionSetUpstream {
			t.Fatalf("action = %s, want set-upstream", got.action)
		}
		if got.err != nil {
			t.Fatalf("set-upstream err = %v", got.err)
		}
		if got.status.Upstream != "origin/feature" || got.status.Head != newHead {
			t.Fatalf("expected upstream origin/feature and head %q, got %+v", newHead, got.status)
		}
	})
}

func TestExecutePushTagUpdatesRemoteSnapshot(t *testing.T) {
	fixture := newCommandRepo(t)
	runGit(t, fixture.root, "tag", "v1.0.0", fixture.initialHash)

	got, ok := cmdResult(t, executePushTag(fixture.repo, "v1.0.0", 40)).(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", cmdResult(t, executePushTag(fixture.repo, "v1.0.0", 40)))
	}
	if got.action != state.ActionPushTag {
		t.Fatalf("action = %s, want push-tag", got.action)
	}
	if got.err != nil {
		t.Fatalf("push tag err = %v", got.err)
	}
	if !got.status.TagProvenanceLoaded {
		t.Fatalf("expected push-tag result to record provenance, got %+v", got.status)
	}
	snapshot, err := loadTagSnapshot(fixture.root)
	if err != nil {
		t.Fatalf("loadTagSnapshot failed: %v", err)
	}
	if snapshot.Summary != tagSyncSynced {
		t.Fatalf("expected synced snapshot summary, got %q", snapshot.Summary)
	}
	if !snapshot.OriginSeen["v1.0.0"] {
		t.Fatalf("expected pushed tag to be recorded in snapshot, got %+v", snapshot.OriginSeen)
	}
}

func TestExecuteDeleteBranchVariants(t *testing.T) {
	t.Run("local branch delete", func(t *testing.T) {
		fixture := newCommandRepo(t)
		runGit(t, fixture.root, "checkout", "-b", "feature")
		runGit(t, fixture.root, "checkout", "main")

		got, ok := cmdResult(t, executeDeleteBranch(fixture.repo, "feature", false, 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeDeleteBranch(fixture.repo, "feature", false, 40)))
		}
		if got.action != state.ActionDeleteBranch {
			t.Fatalf("action = %s, want delete-branch", got.action)
		}
		if got.err != nil {
			t.Fatalf("local delete err = %v", got.err)
		}
		if got.target != "feature" {
			t.Fatalf("expected local delete target feature, got %q", got.target)
		}
		branchList := runGit(t, fixture.root, "branch", "--list", "feature")
		if branchList != "" {
			t.Fatalf("expected feature branch to be deleted, got %q", branchList)
		}
	})

	t.Run("unmerged local branch delete", func(t *testing.T) {
		fixture := newCommandRepo(t)
		runGit(t, fixture.root, "checkout", "-b", "feature")
		makeLocalCommit(t, fixture.root, "feature.txt", "feature\n", "feature commit")
		runGit(t, fixture.root, "checkout", "main")

		got, ok := cmdResult(t, executeDeleteBranch(fixture.repo, "feature", false, 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeDeleteBranch(fixture.repo, "feature", false, 40)))
		}
		if got.action != state.ActionDeleteBranch {
			t.Fatalf("action = %s, want delete-branch", got.action)
		}
		if got.err != nil {
			t.Fatalf("unmerged local delete err = %v", got.err)
		}
		if got.target != "feature" {
			t.Fatalf("expected local delete target feature, got %q", got.target)
		}
		branchList := runGit(t, fixture.root, "branch", "--list", "feature")
		if branchList != "" {
			t.Fatalf("expected feature branch to be deleted, got %q", branchList)
		}
	})

	t.Run("current branch delete blocked", func(t *testing.T) {
		fixture := newCommandRepo(t)
		got, ok := cmdResult(t, executeDeleteBranch(fixture.repo, "main", false, 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeDeleteBranch(fixture.repo, "main", false, 40)))
		}
		if got.err == nil {
			t.Fatal("expected current branch delete to fail")
		}
		if !strings.Contains(got.err.Error(), "current branch cannot be deleted") {
			t.Fatalf("expected current branch error, got %v", got.err)
		}
	})

	t.Run("origin branch delete", func(t *testing.T) {
		fixture := newCommandRepo(t)
		runGit(t, fixture.root, "checkout", "-b", "feature")
		makeLocalCommit(t, fixture.root, "feature.txt", "feature\n", "feature commit")
		runGit(t, fixture.root, "push", "-u", "origin", "feature")
		runGit(t, fixture.root, "checkout", "main")

		got, ok := cmdResult(t, executeDeleteBranch(fixture.repo, "feature", true, 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeDeleteBranch(fixture.repo, "feature", true, 40)))
		}
		if got.action != state.ActionDeleteBranch {
			t.Fatalf("action = %s, want delete-branch", got.action)
		}
		if got.err != nil {
			t.Fatalf("origin delete err = %v", got.err)
		}
		if got.target != "origin/feature" {
			t.Fatalf("expected origin delete target origin/feature, got %q", got.target)
		}
		runGitExpectError(t, fixture.remote, "show-ref", "--verify", "refs/heads/feature")
	})
}

func TestExecuteDeleteTagVariants(t *testing.T) {
	t.Run("tag delete", func(t *testing.T) {
		fixture := newCommandRepo(t)
		if err := fixture.repo.CreateTag(context.Background(), "v1.0.0", fixture.initialHash); err != nil {
			t.Fatalf("CreateTag failed: %v", err)
		}

		got, ok := cmdResult(t, executeDeleteTag(fixture.repo, "v1.0.0", 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeDeleteTag(fixture.repo, "v1.0.0", 40)))
		}
		if got.action != state.ActionDeleteTag {
			t.Fatalf("action = %s, want delete-tag", got.action)
		}
		if got.err != nil {
			t.Fatalf("tag delete err = %v", got.err)
		}
		if got.target != "v1.0.0" {
			t.Fatalf("expected delete target v1.0.0, got %q", got.target)
		}
		entries := got.status.TagEntries
		if len(entries) != 0 {
			t.Fatalf("expected deleted tag to disappear from refreshed status, got %+v", entries)
		}
	})

	t.Run("tag delete missing", func(t *testing.T) {
		fixture := newCommandRepo(t)
		got, ok := cmdResult(t, executeDeleteTag(fixture.repo, "v9.9.9", 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeDeleteTag(fixture.repo, "v9.9.9", 40)))
		}
		if got.action != state.ActionDeleteTag {
			t.Fatalf("action = %s, want delete-tag", got.action)
		}
		if got.err == nil {
			t.Fatal("expected missing tag delete to fail")
		}
	})
}

func TestExecuteCheckoutKeepsRemoteFallback(t *testing.T) {
	fixture := newCommandRepo(t)
	advanceRemoteBranch(t, fixture.remote, "feature", "feature.txt", "feature\n", "feature branch")
	runGit(t, fixture.root, "fetch", "origin")

	got, ok := cmdResult(t, executeCheckout(fixture.repo, "origin/feature", 40)).(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeCheckout(fixture.repo, "origin/feature", 40)))
	}
	if got.action != state.ActionCheckout {
		t.Fatalf("action = %s, want checkout", got.action)
	}
	if got.err != nil {
		t.Fatalf("checkout err = %v", got.err)
	}
	if got.status.Branch != "feature" {
		t.Fatalf("expected local tracking branch feature, got %+v", got.status)
	}
}

func TestExecuteAction(t *testing.T) {
	t.Run("reset", func(t *testing.T) {
		fixture := newCommandRepo(t)
		newHead := makeLocalCommit(t, fixture.root, "file.txt", "change\n", "local change")
		got, ok := cmdResult(t, executeAction(fixture.repo, state.ActionReset, fixture.initialHash, 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeAction(fixture.repo, state.ActionReset, fixture.initialHash, 40)))
		}
		if got.action != state.ActionReset {
			t.Fatalf("action = %s, want reset", got.action)
		}
		if got.err != nil {
			t.Fatalf("reset err = %v", got.err)
		}
		if got.status.Head != fixture.initialHash {
			t.Fatalf("expected reset to %q, got %+v", fixture.initialHash, got.status)
		}
		if newHead == got.status.Head {
			t.Fatal("expected reset to move HEAD")
		}
	})
}

func TestExecuteCherryPickAppliesQueuedCommitsInOrder(t *testing.T) {
	fixture := newCommandRepo(t)
	runGit(t, fixture.root, "checkout", "-b", "feature")
	first := makeLocalCommit(t, fixture.root, "file.txt", "first\n", "first change")
	second := makeLocalCommit(t, fixture.root, "file.txt", "second\n", "second change")
	runGit(t, fixture.root, "checkout", "main")
	runGit(t, fixture.root, "reset", "--hard", fixture.initialHash)

	got, ok := cmdResult(t, executeCherryPick(fixture.repo, []string{first, second}, 40)).(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeCherryPick(fixture.repo, []string{first, second}, 40)))
	}
	if got.action != state.ActionCherryPick {
		t.Fatalf("action = %s, want cherry-pick", got.action)
	}
	if got.err != nil {
		t.Fatalf("executeCherryPick err = %v", got.err)
	}
	if got.status.Head == fixture.initialHash {
		t.Fatal("expected cherry-pick to move HEAD")
	}
	content, err := os.ReadFile(filepath.Join(fixture.root, "file.txt"))
	if err != nil {
		t.Fatalf("read cherry-picked file failed: %v", err)
	}
	if strings.TrimSpace(string(content)) != "second" {
		t.Fatalf("expected final file content to reflect queued order, got %q", string(content))
	}
}

func TestExecuteCherryPickConflictCanAbort(t *testing.T) {
	fixture := newCommandRepo(t)
	runGit(t, fixture.root, "checkout", "-b", "feature")
	featureHash := makeLocalCommit(t, fixture.root, "file.txt", "feature\n", "feature change")
	runGit(t, fixture.root, "checkout", "main")
	_ = makeLocalCommit(t, fixture.root, "file.txt", "main\n", "main change")

	got, ok := cmdResult(t, executeCherryPick(fixture.repo, []string{featureHash}, 40)).(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeCherryPick(fixture.repo, []string{featureHash}, 40)))
	}
	if got.action != state.ActionCherryPick {
		t.Fatalf("action = %s, want cherry-pick", got.action)
	}
	if got.err == nil {
		t.Fatal("expected conflicting cherry-pick to fail")
	}
	if !got.status.CherryPickInProgress {
		t.Fatalf("expected cherry-pick to remain in progress, got %+v", got.status)
	}

	aborted, ok := cmdResult(t, executeAbort(fixture.repo, 40)).(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeAbort(fixture.repo, 40)))
	}
	if aborted.action != state.ActionAbort {
		t.Fatalf("action = %s, want abort", aborted.action)
	}
	if aborted.err != nil {
		t.Fatalf("executeAbort err = %v", aborted.err)
	}
	if aborted.status.CherryPickInProgress {
		t.Fatalf("expected cherry-pick abort to clear sequencer state, got %+v", aborted.status)
	}
}

func TestExecuteResetModes(t *testing.T) {
	for _, tt := range []struct {
		name string
		mode state.ResetMode
	}{
		{name: "soft", mode: state.ResetModeSoft},
		{name: "mixed", mode: state.ResetModeMixed},
		{name: "hard", mode: state.ResetModeHard},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newCommandRepo(t)
			writeRepoFile(t, fixture.root, "file.txt", "change\n")
			runGit(t, fixture.root, "add", "file.txt")
			runGit(t, fixture.root, "commit", "-m", "local change")

			got, ok := cmdResult(t, executeReset(fixture.repo, fixture.initialHash, tt.mode, 40)).(executedMsg)
			if !ok {
				t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeReset(fixture.repo, fixture.initialHash, tt.mode, 40)))
			}
			if got.action != state.ActionReset {
				t.Fatalf("action = %s, want reset", got.action)
			}
			if got.resetMode != tt.mode {
				t.Fatalf("resetMode = %s, want %s", got.resetMode, tt.mode)
			}
			if got.err != nil {
				t.Fatalf("executeReset err = %v", got.err)
			}
			if got.status.Head != fixture.initialHash {
				t.Fatalf("expected reset to %q, got %+v", fixture.initialHash, got.status)
			}
			switch tt.mode {
			case state.ResetModeSoft:
				if !got.status.WorktreeDirty {
					t.Fatalf("expected soft reset to keep worktree dirty, got %+v", got.status)
				}
			case state.ResetModeMixed:
				if !got.status.WorktreeDirty {
					t.Fatalf("expected mixed reset to keep worktree dirty, got %+v", got.status)
				}
			case state.ResetModeHard:
				if got.status.WorktreeDirty {
					t.Fatalf("expected hard reset to clean worktree, got %+v", got.status)
				}
			}
		})
	}
}

func TestExecuteDeleteRemoteTagVariants(t *testing.T) {
	t.Run("remote tag delete", func(t *testing.T) {
		fixture := newCommandRepo(t)
		if err := fixture.repo.CreateTag(context.Background(), "v1.0.0", fixture.initialHash); err != nil {
			t.Fatalf("CreateTag failed: %v", err)
		}
		runGit(t, fixture.root, "push", "origin", "v1.0.0")

		got, ok := cmdResult(t, executeDeleteRemoteTag(fixture.repo, "v1.0.0", 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeDeleteRemoteTag(fixture.repo, "v1.0.0", 40)))
		}
		if got.action != state.ActionDeleteRemoteTag {
			t.Fatalf("action = %s, want delete-remote-tag", got.action)
		}
		if got.err != nil {
			t.Fatalf("remote tag delete err = %v", got.err)
		}
		if got.target != "v1.0.0" {
			t.Fatalf("expected delete target v1.0.0, got %q", got.target)
		}
		entries := got.status.TagEntries
		if len(entries) != 1 || entries[0].Name != "v1.0.0" || entries[0].OnOrigin {
			t.Fatalf("expected local tag to remain and lose origin provenance, got %+v", entries)
		}
		runGitExpectError(t, fixture.remote, "show-ref", "--verify", "refs/tags/v1.0.0")
	})

	t.Run("remote tag delete missing", func(t *testing.T) {
		fixture := newCommandRepo(t)
		got, ok := cmdResult(t, executeDeleteRemoteTag(fixture.repo, "v9.9.9", 40)).(executedMsg)
		if !ok {
			t.Fatalf("expected executedMsg, got %T", cmdResult(t, executeDeleteRemoteTag(fixture.repo, "v9.9.9", 40)))
		}
		if got.action != state.ActionDeleteRemoteTag {
			t.Fatalf("action = %s, want delete-remote-tag", got.action)
		}
		if got.err == nil {
			t.Fatal("expected missing remote tag delete to fail")
		}
	})
}

func TestCreateBranch(t *testing.T) {
	t.Run("empty name blocks", func(t *testing.T) {
		fixture := newCommandRepo(t)
		got, ok := cmdResult(t, createBranch(fixture.repo, "", "main", 40)).(createdBranchMsg)
		if !ok {
			t.Fatalf("expected createdBranchMsg, got %T", cmdResult(t, createBranch(fixture.repo, "", "main", 40)))
		}
		if got.err == nil {
			t.Fatal("expected empty branch name to block branch creation")
		}
	})

	t.Run("empty base blocks", func(t *testing.T) {
		fixture := newCommandRepo(t)
		got, ok := cmdResult(t, createBranch(fixture.repo, "feature", "", 40)).(createdBranchMsg)
		if !ok {
			t.Fatalf("expected createdBranchMsg, got %T", cmdResult(t, createBranch(fixture.repo, "feature", "", 40)))
		}
		if got.err == nil {
			t.Fatal("expected empty base to block branch creation")
		}
	})

	t.Run("clean worktree", func(t *testing.T) {
		fixture := newCommandRepo(t)
		got, ok := cmdResult(t, createBranch(fixture.repo, "feature", "main", 40)).(createdBranchMsg)
		if !ok {
			t.Fatalf("expected createdBranchMsg, got %T", cmdResult(t, createBranch(fixture.repo, "feature", "main", 40)))
		}
		if got.err != nil {
			t.Fatalf("createBranch err = %v", got.err)
		}
		if got.status.Branch != "feature" {
			t.Fatalf("expected branch feature, got %+v", got.status)
		}
	})

	t.Run("duplicate branch blocks", func(t *testing.T) {
		fixture := newCommandRepo(t)
		runGit(t, fixture.root, "checkout", "-b", "feature")
		runGit(t, fixture.root, "checkout", "main")
		got, ok := cmdResult(t, createBranch(fixture.repo, "feature", "main", 40)).(createdBranchMsg)
		if !ok {
			t.Fatalf("expected createdBranchMsg, got %T", cmdResult(t, createBranch(fixture.repo, "feature", "main", 40)))
		}
		if got.err == nil {
			t.Fatal("expected duplicate branch name to block branch creation")
		}
	})

	t.Run("dirty worktree blocks", func(t *testing.T) {
		fixture := newCommandRepo(t)
		writeRepoFile(t, fixture.root, "dirty.txt", "dirty\n")
		got, ok := cmdResult(t, createBranch(fixture.repo, "dirty-feature", "main", 40)).(createdBranchMsg)
		if !ok {
			t.Fatalf("expected createdBranchMsg, got %T", cmdResult(t, createBranch(fixture.repo, "dirty-feature", "main", 40)))
		}
		if got.err == nil {
			t.Fatal("expected dirty worktree to block branch creation")
		}
	})
}

func TestExecuteFetchForPushAndPull(t *testing.T) {
	fixture := newCommandRepo(t)
	advanceRemote(t, fixture.remote, "remote.txt", "remote\n", "remote advance")

	pushMsg, ok := cmdResult(t, executeFetchForPush(fixture.repo, 40)).(pushFetchedMsg)
	if !ok {
		t.Fatalf("expected pushFetchedMsg, got %T", cmdResult(t, executeFetchForPush(fixture.repo, 40)))
	}
	if pushMsg.err != nil {
		t.Fatalf("executeFetchForPush err = %v", pushMsg.err)
	}
	if len(pushMsg.status.RemoteBranches) == 0 {
		t.Fatalf("expected remote branches after fetch-for-push, got %+v", pushMsg.status)
	}

	pullRequest := pullRequest{ID: 1, Epoch: 7}
	pullMsg, ok := cmdResult(t, executeFetchForPull(fixture.repo, 40, pullRequest)).(pullFetchedMsg)
	if !ok {
		t.Fatalf("expected pullFetchedMsg, got %T", cmdResult(t, executeFetchForPull(fixture.repo, 40, pullRequest)))
	}
	if pullMsg.err != nil {
		t.Fatalf("executeFetchForPull err = %v", pullMsg.err)
	}
	if pullMsg.status.Root == "" {
		t.Fatalf("expected repo status after fetch-for-pull, got %+v", pullMsg.status)
	}
	if pullMsg.requestID != pullRequest.ID || pullMsg.requestEpoch != pullRequest.Epoch {
		t.Fatalf("expected pull request identity %d/%d, got %d/%d", pullRequest.ID, pullRequest.Epoch, pullMsg.requestID, pullMsg.requestEpoch)
	}
}

func TestExecuteValidatedPullRechecksSnapshotBeforeMutation(t *testing.T) {
	fixture := newCommandRepo(t)
	status, err := fixture.repo.Status(context.Background(), 40)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	baseline := pullSnapshotIdentity(status, 7)
	baseline.Head = "different-head"
	request := pullRequest{ID: 1, Epoch: 7, Baseline: baseline}

	result, ok := cmdResult(t, executeValidatedPull(fixture.repo, 40, request, PullModeMerge)).(pullExecutionResultMsg)
	if !ok {
		t.Fatalf("expected pullExecutionResultMsg, got %T", cmdResult(t, executeValidatedPull(fixture.repo, 40, request, PullModeMerge)))
	}
	if !result.stale {
		t.Fatalf("expected stale execution result, got %+v", result)
	}
}

func TestLoadPullPreviewCommitsUsesCorrectRange(t *testing.T) {
	fixture := newCommandRepo(t)
	remoteHead := advanceRemote(t, fixture.remote, "remote.txt", "remote\n", "remote advance")
	behind := cloneRepoAtHash(t, fixture.remote, fixture.initialHash)
	request := pullRequest{ID: 1, Epoch: 7}

	ffReady, ok := cmdResult(t, loadPullPreviewCommits(behind.repo, true, request)).(pullPreviewReadyMsg)
	if !ok {
		t.Fatalf("expected pullPreviewReadyMsg, got %T", cmdResult(t, loadPullPreviewCommits(behind.repo, true, request)))
	}
	nonFFReady, ok := cmdResult(t, loadPullPreviewCommits(behind.repo, false, request)).(pullPreviewReadyMsg)
	if !ok {
		t.Fatalf("expected pullPreviewReadyMsg, got %T", cmdResult(t, loadPullPreviewCommits(behind.repo, false, request)))
	}
	if ffReady.requestID != request.ID || ffReady.requestEpoch != request.Epoch || nonFFReady.requestID != request.ID || nonFFReady.requestEpoch != request.Epoch {
		t.Fatalf("expected preview messages to carry request identity %d/%d, got ff=%d/%d nonff=%d/%d", request.ID, request.Epoch, ffReady.requestID, ffReady.requestEpoch, nonFFReady.requestID, nonFFReady.requestEpoch)
	}
	if ffReady.err != nil || nonFFReady.err != nil {
		t.Fatalf("unexpected preview errors: ff=%v nonff=%v", ffReady.err, nonFFReady.err)
	}
	if len(ffReady.commits) != len(nonFFReady.commits)+1 {
		t.Fatalf("expected ff preview to include HEAD, ff=%v nonff=%v", ffReady.commits, nonFFReady.commits)
	}
	if ffReady.commits[len(ffReady.commits)-1] != fixture.initialHash {
		t.Fatalf("expected ff preview to append HEAD %q, got %v", fixture.initialHash, ffReady.commits)
	}
	if len(nonFFReady.commits) == 0 || nonFFReady.commits[0] != remoteHead {
		t.Fatalf("expected non-ff preview to include remote head %q, got %v", remoteHead, nonFFReady.commits)
	}
}
