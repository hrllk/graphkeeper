package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/git-graph-tui/internal/git"
	"hrllk/git-graph-tui/internal/state"
)

func loadRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		status, err := repo.Status(context.Background(), limit)
		return loadedMsg{status: status, err: err}
	}
}

func scheduleRefresh() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func refreshRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		status, err := repo.Status(context.Background(), limit)
		return refreshedMsg{status: status, err: err}
	}
}

func fetchRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Fetch(context.Background()); err != nil {
			return fetchedMsg{err: err}
		}
		status, err := repo.Status(context.Background(), limit)
		return fetchedMsg{status: status, err: err}
	}
}

func prepareAction(repo *git.Repo, action state.Action, limit int) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Fetch(context.Background()); err != nil {
			return preparedMsg{action: action, err: err}
		}
		status, err := repo.Status(context.Background(), limit)
		return preparedMsg{action: action, status: status, err: err}
	}
}

func pullCheck(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Fetch(context.Background()); err != nil {
			return pullCheckedMsg{err: err}
		}
		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return pullCheckedMsg{err: err}
		}
		behind, ahead, err := repo.Divergence(context.Background(), status.Upstream, "HEAD")
		if err != nil {
			return pullCheckedMsg{err: err}
		}
		if ahead > 0 {
			return pullCheckedMsg{
				repo: status,
				status: state.New().WithBlocked(
					state.BlockDiverged,
					"Fast-forward is not possible.",
					"The branch has diverged from its upstream.",
				),
			}
		}
		_ = behind
		return pullCheckedMsg{
			repo: status,
			status: state.New().WithOutcome(
				state.ActionPull,
				"Fast-forward is possible.",
				"The upstream can move to the current branch tip.",
				true,
			),
		}
	}
}

func executePull(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Run("pull", "--no-rebase", "--no-edit")
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionPull, err: statusErr}
		}
		return executedMsg{action: state.ActionPull, status: status, err: err}
	}
}

func executePullMerge(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Run("pull", "--no-rebase", "--no-edit")
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionPullMerge, err: statusErr}
		}
		return executedMsg{action: state.ActionPullMerge, status: status, err: err}
	}
}

func executePullRebase(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Run("pull", "--rebase")
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionPullRebase, err: statusErr}
		}
		return executedMsg{action: state.ActionPullRebase, status: status, err: err}
	}
}

func executeAbort(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		// 현재 리포지토리 상태를 1차 파악하여 merge abort 인지 rebase abort 인지 구분합니다.
		currentStatus, statusErr := repo.Status(context.Background(), limit)
		var err error
		if statusErr == nil && currentStatus.RebaseInProgress {
			_, err = repo.Run("rebase", "--abort")
		} else {
			_, err = repo.Run("merge", "--abort")
		}
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionAbort, err: statusErr}
		}
		return executedMsg{action: state.ActionAbort, status: status, err: err}
	}
}

func executePush(repo *git.Repo, branch string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Push(context.Background(), branch, false, false)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionPush, target: branch, err: statusErr}
		}
		return executedMsg{action: state.ActionPush, target: branch, status: status, err: err}
	}
}

func executeForcePush(repo *git.Repo, branch string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Push(context.Background(), branch, true, false)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionForcePush, target: branch, err: statusErr}
		}
		return executedMsg{action: state.ActionForcePush, target: branch, status: status, err: err}
	}
}

func executePushSetUpstream(repo *git.Repo, branch string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Push(context.Background(), branch, false, true)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionSetUpstream, target: branch, err: statusErr}
		}
		return executedMsg{action: state.ActionSetUpstream, target: branch, status: status, err: err}
	}
}

func previewSelection(repo *git.Repo, rs git.Status, action state.Action, target string) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return previewMsg{action: action, target: target, repo: rs, err: fmt.Errorf("target is empty")}
		}
		if (action == state.ActionMerge || action == state.ActionRebase) && rs.Detached {
			return previewMsg{
				action: action,
				target: target,
				repo:   rs,
				status: state.New().WithBlocked(state.BlockDetached, "Detached HEAD.", "Choose a branch before merging or rebasing."),
			}
		}
		currentOnly, targetOnly, err := repo.Divergence(context.Background(), "HEAD", target)
		if err != nil {
			return previewMsg{action: action, target: target, repo: rs, err: err}
		}
		return previewMsg{
			action: action,
			target: target,
			repo:   rs,
			status: buildActionPreview(action, target, rs, currentOnly, targetOnly),
		}
	}
}

func executeAction(repo *git.Repo, action state.Action, target string, limit int) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return executedMsg{action: action, err: fmt.Errorf("target is empty")}
		}
		var err error
		switch action {
		case state.ActionMerge:
			_, err = repo.Run("merge", "--no-edit", target)
		case state.ActionRebase:
			_, err = repo.Run("rebase", target)
		case state.ActionReset:
			_, err = repo.Run("reset", "--hard", target)
		default:
			err = fmt.Errorf("unsupported action %q", action)
		}
		if err != nil {
			return executedMsg{action: action, target: target, err: err}
		}
		status, statusErr := repo.Status(context.Background(), limit)
		return executedMsg{action: action, target: target, status: status, err: statusErr}
	}
}

func createBranch(repo *git.Repo, name, base string, limit int) tea.Cmd {
	return func() tea.Msg {
		if name == "" {
			return createdBranchMsg{err: fmt.Errorf("branch name is empty")}
		}
		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return createdBranchMsg{name: name, base: base, err: err}
		}
		if status.WorktreeDirty {
			return createdBranchMsg{
				name: name,
				base: base,
				err:  fmt.Errorf("working tree is not clean"),
			}
		}
		if base == "" {
			base = "HEAD"
		}
		if _, err := repo.Run("switch", "-c", name, base); err != nil {
			return createdBranchMsg{name: name, base: base, err: err}
		}
		status, err = repo.Status(context.Background(), limit)
		return createdBranchMsg{name: name, base: base, status: status, err: err}
	}
}

func executeCheckout(repo *git.Repo, target string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Run("switch", target)
		if err != nil && strings.Contains(target, "/") {
			localName := target[strings.Index(target, "/")+1:]
			_, err = repo.Run("switch", "--track", "-c", localName, target)
		}
		if err != nil {
			return executedMsg{action: state.ActionCheckout, target: target, err: err}
		}
		status, statusErr := repo.Status(context.Background(), limit)
		return executedMsg{action: state.ActionCheckout, target: target, status: status, err: statusErr}
	}
}

func executeFetchForPush(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		err := repo.Fetch(context.Background())
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return pushFetchedMsg{err: statusErr}
		}
		return pushFetchedMsg{status: status, err: err}
	}
}

func executeFetchForPull(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		err := repo.Fetch(context.Background())
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return pullFetchedMsg{err: statusErr}
		}
		return pullFetchedMsg{status: status, err: err}
	}
}

func loadPullPreviewCommits(repo *git.Repo, isFF bool) tea.Cmd {
	return func() tea.Msg {
		var arg string
		if isFF {
			arg = "HEAD..@{upstream}"
		} else {
			arg = "HEAD...@{upstream}"
		}
		out, err := repo.Run("rev-list", arg)
		if err != nil {
			return pullPreviewReadyMsg{err: err, isFF: isFF}
		}
		lines := strings.Split(out, "\n")
		commits := make([]string, 0, len(lines))
		for _, line := range lines {
			hash := strings.TrimSpace(line)
			if hash != "" {
				commits = append(commits, hash)
			}
		}
		if isFF {
			headOut, headErr := repo.Run("rev-parse", "HEAD")
			if headErr == nil && strings.TrimSpace(headOut) != "" {
				commits = append(commits, strings.TrimSpace(headOut))
			}
		}
		return pullPreviewReadyMsg{commits: commits, isFF: isFF}
	}
}
