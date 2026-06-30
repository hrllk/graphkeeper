package app

import (
	"bytes"
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

	pullMsg, ok := cmdResult(t, executeFetchForPull(fixture.repo, 40)).(pullFetchedMsg)
	if !ok {
		t.Fatalf("expected pullFetchedMsg, got %T", cmdResult(t, executeFetchForPull(fixture.repo, 40)))
	}
	if pullMsg.err != nil {
		t.Fatalf("executeFetchForPull err = %v", pullMsg.err)
	}
	if pullMsg.status.Root == "" {
		t.Fatalf("expected repo status after fetch-for-pull, got %+v", pullMsg.status)
	}
}

func TestLoadPullPreviewCommitsUsesCorrectRange(t *testing.T) {
	fixture := newCommandRepo(t)
	remoteHead := advanceRemote(t, fixture.remote, "remote.txt", "remote\n", "remote advance")
	behind := cloneRepoAtHash(t, fixture.remote, fixture.initialHash)

	ffReady, ok := cmdResult(t, loadPullPreviewCommits(behind.repo, true)).(pullPreviewReadyMsg)
	if !ok {
		t.Fatalf("expected pullPreviewReadyMsg, got %T", cmdResult(t, loadPullPreviewCommits(behind.repo, true)))
	}
	nonFFReady, ok := cmdResult(t, loadPullPreviewCommits(behind.repo, false)).(pullPreviewReadyMsg)
	if !ok {
		t.Fatalf("expected pullPreviewReadyMsg, got %T", cmdResult(t, loadPullPreviewCommits(behind.repo, false)))
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
