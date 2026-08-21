package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func loadRepoState(repo *git.Repo, limit int, args ...interface{}) tea.Cmd {
	var epoch uint64
	var store TagProvenanceStore
	for _, arg := range args {
		switch value := arg.(type) {
		case uint64:
			epoch = value
		case TagProvenanceStore:
			store = value
		}
	}
	return func() tea.Msg {
		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return loadedMsg{status: status, err: err, epoch: epoch, epochSet: len(args) > 0}
		}
		status, err = loadLocalTagStatus(repo, status, store)
		if err != nil {
			return loadedMsg{status: status, err: err, epoch: epoch, epochSet: len(args) > 0}
		}
		return loadedMsg{status: status, err: err, epoch: epoch, epochSet: len(args) > 0}
	}
}

func loadRepositorySnapshot(port RepositoryReadPort, limit int, epoch uint64) tea.Cmd {
	return func() tea.Msg {
		if port == nil {
			return loadedSnapshotMsg{result: ReadSnapshotResult{RepositoryEpoch: epoch, ErrorKind: ReadErrorRepository}}
		}
		result, err := port.ReadSnapshot(context.Background(), ReadRequest{CommitLimit: int(limit), RequestID: 1, RepositoryEpoch: epoch})
		if err != nil && result.ErrorKind == ReadErrorNone {
			result.ErrorKind = ReadErrorRepository
		}
		return loadedSnapshotMsg{result: result}
	}
}

func (m *model) refreshCmd() tea.Cmd {
	m.refreshGeneration++
	if m.repositoryRead != nil {
		generation := m.refreshGeneration
		epoch := m.repositoryEpoch
		port := m.repositoryRead
		limit := m.commitLimit
		readCmd := func() tea.Msg {
			result, err := port.ReadSnapshot(context.Background(), ReadRequest{CommitLimit: limit, RequestID: generation, RepositoryEpoch: epoch})
			if err != nil && result.ErrorKind == ReadErrorNone {
				result.ErrorKind = ReadErrorRepository
			}
			return refreshedSnapshotMsg{result: result, refreshGeneration: generation}
		}
		return readCmd
	}
	return refreshRepoState(m.repo, m.commitLimit, m.repositoryEpoch, m.refreshGeneration, m.tagProvenance)
}

func scheduleRefresh() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func refreshRepoState(repo *git.Repo, limit int, args ...interface{}) tea.Cmd {
	var epoch, generation uint64
	var store TagProvenanceStore
	for _, arg := range args {
		switch value := arg.(type) {
		case uint64:
			if epoch == 0 {
				epoch = value
			} else {
				generation = value
			}
		case TagProvenanceStore:
			store = value
		}
	}
	return func() tea.Msg {
		status, err := repo.Status(context.Background(), limit)
		if err == nil {
			status, err = loadLocalTagStatus(repo, status, store)
		}
		return refreshedMsg{status: status, err: err, epoch: epoch, epochSet: len(args) > 0, refreshGeneration: generation, generationSet: len(args) > 1}
	}
}

func loadStashState(repo *git.Repo) tea.Cmd {
	return func() tea.Msg {
		entries, err := repo.Stashes(context.Background())
		return stashLoadedMsg{entries: entries, err: err}
	}
}

func executeStashAll(repo *git.Repo, limit int, message string) tea.Cmd {
	return func() tea.Msg {
		err := repo.StashAll(context.Background(), message)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionStash, err: statusErr}
		}
		return executedMsg{action: state.ActionStash, status: status, err: err}
	}
}

func executeCleanWorkingTree(repo *git.Repo, limit int, includeIgnored bool) tea.Cmd {
	return func() tea.Msg {
		err := repo.CleanWorkingTree(context.Background(), includeIgnored)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionCleanWorkingTree, err: statusErr}
		}
		return executedMsg{action: state.ActionCleanWorkingTree, status: status, err: err}
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

func firstTagStore(stores []TagProvenanceStore) TagProvenanceStore {
	if len(stores) == 0 {
		return nil
	}
	return stores[0]
}

func fetchTagsRepoState(repo *git.Repo, limit int, stores ...TagProvenanceStore) tea.Cmd {
	return func() tea.Msg {
		if err := repo.FetchTags(context.Background()); err != nil {
			return fetchedMsg{err: err}
		}
		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return fetchedMsg{status: status, err: err}
		}
		tags, tagErr := repo.LocalTagEntries(context.Background())
		if tagErr != nil {
			return fetchedMsg{status: status, err: tagErr}
		}
		remoteTags, remoteErr := repo.OriginTagSet(context.Background())
		if remoteErr != nil {
			return fetchedMsg{status: status, err: remoteErr}
		}
		status.TagEntries = tags
		status.TagEntriesLoaded = true
		status.Tags = make([]string, 0, len(tags))
		for _, entry := range tags {
			status.Tags = append(status.Tags, entry.Name)
		}
		status = attachGraphTagEntries(status)
		snapshot := buildTagSnapshot(tags, remoteTags, tagSyncSynced)
		if len(stores) > 0 && stores[0] != nil {
			if saveErr := stores[0].Save(context.Background(), snapshot); saveErr != nil {
				return fetchedMsg{status: status, err: saveErr}
			}
		}
		status = applyTagSnapshot(status, snapshot)
		status.TagProvenanceLoaded = true
		return fetchedMsg{status: status, err: err}
	}
}

func executePushTag(repo *git.Repo, tag string, limit int, stores ...TagProvenanceStore) tea.Cmd {
	return func() tea.Msg {
		if tag == "" {
			return executedMsg{action: state.ActionPushTag, err: fmt.Errorf("tag is empty")}
		}
		_, err := repo.PushTag(context.Background(), tag)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionPushTag, target: tag, err: statusErr}
		}
		status, statusErr = loadLocalTagStatus(repo, status, firstTagStore(stores))
		if statusErr != nil {
			return executedMsg{action: state.ActionPushTag, target: tag, status: status, err: statusErr}
		}
		if err != nil {
			return executedMsg{action: state.ActionPushTag, target: tag, status: status, err: err}
		}
		remoteTags, remoteErr := repo.OriginTagSet(context.Background())
		if remoteErr != nil {
			return executedMsg{action: state.ActionPushTag, target: tag, status: status, err: remoteErr}
		}
		snapshot := buildTagSnapshot(status.TagEntries, remoteTags, tagSyncSynced)
		if store := firstTagStore(stores); store != nil {
			if saveErr := store.Save(context.Background(), snapshot); saveErr != nil {
				return executedMsg{action: state.ActionPushTag, target: tag, status: status, err: saveErr}
			}
		}
		status = applyTagSnapshot(status, snapshot)
		return executedMsg{action: state.ActionPushTag, target: tag, status: status, err: err}
	}
}

func attachTagEntries(repo *git.Repo, status git.Status) git.Status {
	tagEntries, err := repo.TagEntries(context.Background())
	if err != nil {
		return status
	}
	status.TagEntries = tagEntries
	status.TagEntriesLoaded = true
	status.TagProvenanceLoaded = true
	status.TagSyncSummary = string(tagSyncSynced)
	status.Tags = make([]string, 0, len(tagEntries))
	for _, entry := range tagEntries {
		status.Tags = append(status.Tags, entry.Name)
	}
	status = attachGraphTagEntries(status)
	return status
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
		// Inspect the current repo state first to distinguish merge abort from rebase abort.
		currentStatus, statusErr := repo.Status(context.Background(), limit)
		var err error
		if statusErr == nil && currentStatus.CherryPickInProgress {
			_, err = repo.Run("cherry-pick", "--abort")
		} else if statusErr == nil && currentStatus.RebaseInProgress {
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

func executeCherryPick(repo *git.Repo, hashes []string, limit int) tea.Cmd {
	return func() tea.Msg {
		if len(hashes) == 0 {
			return executedMsg{action: state.ActionCherryPick, err: fmt.Errorf("no cherry-pick targets selected")}
		}
		args := append([]string{"cherry-pick"}, hashes...)
		_, err := repo.Run(args...)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionCherryPick, target: strings.Join(hashes, ","), err: statusErr}
		}
		return executedMsg{action: state.ActionCherryPick, target: strings.Join(hashes, ","), status: status, err: err}
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

func executeDeleteBranch(repo *git.Repo, target string, remote bool, limit int) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return executedMsg{action: state.ActionDeleteBranch, err: fmt.Errorf("target is empty")}
		}
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionDeleteBranch, target: target, err: statusErr}
		}
		if remote {
			branch := strings.TrimPrefix(target, "origin/")
			if branch == "" {
				return executedMsg{action: state.ActionDeleteBranch, target: target, err: fmt.Errorf("remote branch is empty")}
			}
			if _, err := repo.Run("ls-remote", "--exit-code", "--heads", "origin", "refs/heads/"+branch); err != nil {
				return executedMsg{action: state.ActionDeleteBranch, target: target, err: fmt.Errorf("remote branch %q no longer exists: %w", target, err)}
			}
			_, err := repo.DeleteRemoteBranch(context.Background(), "origin", branch)
			refreshed, refreshErr := repo.Status(context.Background(), limit)
			if refreshErr != nil {
				return executedMsg{action: state.ActionDeleteBranch, target: "origin/" + branch, err: refreshErr}
			}
			return executedMsg{action: state.ActionDeleteBranch, target: "origin/" + branch, status: refreshed, err: err}
		}
		if !status.Detached && status.Branch == target {
			return executedMsg{action: state.ActionDeleteBranch, target: target, err: fmt.Errorf("current branch cannot be deleted")}
		}
		if !containsString(status.LocalBranches, target) && !containsString(status.Branches, target) {
			return executedMsg{action: state.ActionDeleteBranch, target: target, err: fmt.Errorf("branch %q no longer exists", target)}
		}
		_, err := repo.DeleteBranch(context.Background(), target)
		refreshed, refreshErr := repo.Status(context.Background(), limit)
		if refreshErr != nil {
			return executedMsg{action: state.ActionDeleteBranch, target: target, err: refreshErr}
		}
		return executedMsg{action: state.ActionDeleteBranch, target: target, status: refreshed, err: err}
	}
}

func executeDeleteTag(repo *git.Repo, target string, limit int, stores ...TagProvenanceStore) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return executedMsg{action: state.ActionDeleteTag, err: fmt.Errorf("target is empty")}
		}
		if _, err := repo.Run("show-ref", "--verify", "refs/tags/"+target); err != nil {
			return executedMsg{action: state.ActionDeleteTag, target: target, err: fmt.Errorf("tag %q no longer exists: %w", target, err)}
		}
		_, err := repo.DeleteTag(context.Background(), target)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionDeleteTag, target: target, err: statusErr}
		}
		status, statusErr = loadLocalTagStatus(repo, status, firstTagStore(stores))
		if statusErr != nil {
			return executedMsg{action: state.ActionDeleteTag, target: target, status: status, err: statusErr}
		}
		return executedMsg{action: state.ActionDeleteTag, target: target, status: status, err: err}
	}
}

func executeDeleteRemoteTag(repo *git.Repo, target string, limit int, stores ...TagProvenanceStore) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return executedMsg{action: state.ActionDeleteRemoteTag, err: fmt.Errorf("target is empty")}
		}
		if _, err := repo.Run("ls-remote", "--exit-code", "--tags", "origin", "refs/tags/"+target); err != nil {
			return executedMsg{action: state.ActionDeleteRemoteTag, target: target, err: fmt.Errorf("remote tag %q no longer exists: %w", target, err)}
		}
		_, err := repo.DeleteRemoteTag(context.Background(), "origin", target)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionDeleteRemoteTag, target: target, err: statusErr}
		}
		status, statusErr = loadLocalTagStatus(repo, status, firstTagStore(stores))
		if statusErr != nil {
			return executedMsg{action: state.ActionDeleteRemoteTag, target: target, status: status, err: statusErr}
		}
		remoteTags, remoteErr := repo.OriginTagSet(context.Background())
		if remoteErr != nil {
			return executedMsg{action: state.ActionDeleteRemoteTag, target: target, status: status, err: remoteErr}
		}
		snapshot := buildTagSnapshot(status.TagEntries, remoteTags, tagSyncSynced)
		if store := firstTagStore(stores); store != nil {
			if saveErr := store.Save(context.Background(), snapshot); saveErr != nil {
				return executedMsg{action: state.ActionDeleteRemoteTag, target: target, status: status, err: saveErr}
			}
		}
		status = applyTagSnapshot(status, snapshot)
		return executedMsg{action: state.ActionDeleteRemoteTag, target: target, status: status, err: err}
	}
}

func executeStashPop(repo *git.Repo, limit int, entry git.StashEntry) tea.Cmd {
	return func() tea.Msg {
		if entry.Ref == "" {
			return executedMsg{action: state.ActionStashPop, err: fmt.Errorf("stash ref is empty")}
		}
		err := repo.StashPop(context.Background(), entry.Ref)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionStashPop, target: entry.Ref, err: statusErr}
		}
		return executedMsg{action: state.ActionStashPop, target: entry.Ref, status: status, err: err}
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

func checkGraphActionTarget(repo *git.Repo, action state.Action, target string, rs git.Status) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return graphActionCheckMsg{action: action, target: target, repo: rs, err: fmt.Errorf("repo is nil")}
		}
		if target == "" {
			return graphActionCheckMsg{action: action, target: target, repo: rs, err: fmt.Errorf("target is empty")}
		}
		base, err := repo.MergeBase(context.Background(), "HEAD", target)
		if err != nil {
			return graphActionCheckMsg{action: action, target: target, repo: rs, err: err}
		}
		currentOnly, targetOnly, err := repo.Divergence(context.Background(), "HEAD", target)
		if err != nil {
			return graphActionCheckMsg{action: action, target: target, repo: rs, err: err}
		}
		return graphActionCheckMsg{
			action:      action,
			target:      target,
			repo:        rs,
			base:        base,
			currentOnly: currentOnly,
			targetOnly:  targetOnly,
		}
	}
}

func executeAction(repo *git.Repo, action state.Action, target string, limit int) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return executedMsg{action: action, err: fmt.Errorf("target is empty")}
		}
		adapter := newGitRepositoryAdapter(repo)
		targetRef := targetRef{Kind: state.TargetKindCommit, Name: target}
		if err := adapter.ValidateTarget(context.Background(), targetRef); err != nil {
			return executedMsg{action: action, target: target, err: fmt.Errorf("target %q is no longer available: %w", target, err)}
		}
		var err error
		args := []string{}
		switch action {
		case state.ActionMerge:
			args = []string{"merge", "--no-edit", target}
		case state.ActionRebase:
			args = []string{"rebase", target}
		case state.ActionReset:
			args = []string{"reset", "--hard", target}
		default:
			err = fmt.Errorf("unsupported action %q", action)
		}
		if err == nil {
			_, err = adapter.Execute(context.Background(), operation{Action: action, Target: targetRef, Args: args})
		}
		if err != nil {
			return executedMsg{action: action, target: target, err: err}
		}
		status, statusErr := repo.Status(context.Background(), limit)
		return executedMsg{action: action, target: target, status: status, err: statusErr}
	}
}

func executeReset(repo *git.Repo, target string, mode state.ResetMode, limit int) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return executedMsg{action: state.ActionReset, err: fmt.Errorf("target is empty"), resetMode: mode}
		}
		adapter := newGitRepositoryAdapter(repo)
		targetRef := targetRef{Kind: state.TargetKindCommit, Name: target}
		if err := adapter.ValidateTarget(context.Background(), targetRef); err != nil {
			return executedMsg{action: state.ActionReset, target: target, err: fmt.Errorf("target %q is no longer available: %w", target, err), resetMode: mode}
		}
		if mode == "" {
			mode = state.ResetModeHard
		}
		args := []string{"reset", "--hard", target}
		switch mode {
		case state.ResetModeSoft:
			args = []string{"reset", "--soft", target}
		case state.ResetModeMixed:
			args = []string{"reset", "--mixed", target}
		case state.ResetModeHard:
			args = []string{"reset", "--hard", target}
		default:
			return executedMsg{action: state.ActionReset, target: target, err: fmt.Errorf("unsupported reset mode %q", mode), resetMode: mode}
		}
		_, err := adapter.Execute(context.Background(), operation{Action: state.ActionReset, Target: targetRef, Args: args})
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionReset, target: target, status: status, err: statusErr, resetMode: mode}
		}
		return executedMsg{action: state.ActionReset, target: target, status: status, err: err, resetMode: mode}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func createBranch(repo *git.Repo, name, base string, limit int) tea.Cmd {
	return func() tea.Msg {
		name = strings.TrimSpace(name)
		base = strings.TrimSpace(base)
		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return createdBranchMsg{name: name, base: base, err: err}
		}
		if err := branchCreateValidationError(status, name, base); err != nil {
			return createdBranchMsg{name: name, base: base, err: err}
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

func beginPullRequest(m *model) pullRequest {
	m.nextPullRequestID++
	baseline := pullSnapshotIdentity(m.repoStatus, m.repositoryEpoch+1)
	request := pullRequest{ID: m.nextPullRequestID, Epoch: m.repositoryEpoch + 1, FetchBaseline: baseline}
	m.activePullRequest = &request
	m.pullConfirmStale = false
	return request
}

func executeFetchForPull(repo *git.Repo, limit int, request pullRequest) tea.Cmd {
	return func() tea.Msg {
		err := repo.Fetch(context.Background())
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return pullFetchedMsg{err: statusErr, requestID: request.ID, requestEpoch: request.Epoch, baseline: request.FetchBaseline, fetchBaseline: request.FetchBaseline}
		}
		if err != nil {
			return pullFetchedMsg{status: status, err: err, requestID: request.ID, requestEpoch: request.Epoch, baseline: request.FetchBaseline, fetchBaseline: request.FetchBaseline}
		}
		operationBaseline := pullSnapshotIdentity(status, request.Epoch)
		fastForward, fastForwardKnown := resolvePullFastForward(repo)
		if !fastForwardKnown {
			return pullFetchedMsg{status: status, err: fmt.Errorf("fast-forward impact is unavailable"), requestID: request.ID, requestEpoch: request.Epoch, baseline: request.FetchBaseline, fetchBaseline: request.FetchBaseline}
		}
		statusAgain, statusAgainErr := repo.Status(context.Background(), limit)
		if statusAgainErr != nil || !samePullSnapshotIdentity(operationBaseline, pullSnapshotIdentity(statusAgain, request.Epoch)) {
			if statusAgainErr == nil {
				statusAgainErr = fmt.Errorf("repository changed while resolving pull impact")
			}
			return pullFetchedMsg{status: statusAgain, err: statusAgainErr, requestID: request.ID, requestEpoch: request.Epoch, baseline: request.FetchBaseline, fetchBaseline: request.FetchBaseline}
		}
		snapshot := pullImpactSnapshot(operationBaseline, request)
		snapshot.IsFastForward = fastForward
		snapshot.FastForwardKnown = fastForwardKnown
		snapshot.Validity.FastForwardKnown = fastForwardKnown
		return pullFetchedMsg{status: status, err: err, requestID: request.ID, requestEpoch: request.Epoch,
			baseline: request.FetchBaseline, fetchBaseline: request.FetchBaseline, operationBaseline: operationBaseline, operationBaselineSet: err == nil, snapshot: snapshot}
	}
}

func resolvePullFastForward(repo *git.Repo) (bool, bool) {
	_, err := repo.Run("merge-base", "--is-ancestor", "HEAD", "@{upstream}")
	if err == nil {
		return true, true
	}
	if strings.Contains(err.Error(), "exit status 1") {
		return false, true
	}
	return false, false
}

func loadPullPreviewCommits(repo *git.Repo, isFF bool, request pullRequest) tea.Cmd {
	return func() tea.Msg {
		var arg string
		if isFF {
			arg = "HEAD..@{upstream}"
		} else {
			arg = "HEAD...@{upstream}"
		}
		out, err := repo.Run("rev-list", arg)
		if err != nil {
			return pullPreviewReadyMsg{err: err, isFF: isFF, requestID: request.ID, requestEpoch: request.Epoch, baseline: request.OperationBaseline, snapshot: pullImpactSnapshot(request.OperationBaseline, request), impact: pullImpactSet(pullImpactSnapshot(request.OperationBaseline, request))}
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
		return pullPreviewReadyMsg{commits: commits, isFF: isFF, requestID: request.ID, requestEpoch: request.Epoch, baseline: request.OperationBaseline, snapshot: pullImpactSnapshot(request.OperationBaseline, request), impact: pullImpactSet(pullImpactSnapshot(request.OperationBaseline, request))}
	}
}

func validateAndExecutePull(repo *git.Repo, limit int, request pullRequest, mode PullMode) tea.Cmd {
	return func() tea.Msg {
		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return pullValidationMsg{requestID: request.ID, requestEpoch: request.Epoch, baseline: request.FetchBaseline, mode: mode, err: err}
		}
		if !request.OperationBaselineSet {
			return pullValidationMsg{requestID: request.ID, requestEpoch: request.Epoch, baseline: request.OperationBaseline, operationBaseline: request.OperationBaseline, operationBaselineSet: false, mode: mode, status: status, valid: false}
		}
		current := pullSnapshotIdentity(status, request.Epoch)
		if !samePullSnapshotIdentity(current, request.OperationBaseline) {
			return pullValidationMsg{requestID: request.ID, requestEpoch: request.Epoch, baseline: request.OperationBaseline, operationBaseline: request.OperationBaseline, operationBaselineSet: true, mode: mode, status: status, valid: false}
		}
		return pullValidationMsg{requestID: request.ID, requestEpoch: request.Epoch, baseline: request.OperationBaseline, operationBaseline: request.OperationBaseline, operationBaselineSet: true, mode: mode, status: status, valid: true}
	}
}

func executeValidatedPull(repo *git.Repo, limit int, request pullRequest, mode PullMode) tea.Cmd {
	return func() tea.Msg {
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return pullExecutionResultMsg{action: state.ActionPull, status: status, err: statusErr, requestID: request.ID, requestEpoch: request.Epoch, baseline: request.OperationBaseline, operationBaseline: request.OperationBaseline, operationBaselineSet: request.OperationBaselineSet, mode: mode, stale: true}
		}
		if !request.OperationBaselineSet {
			return pullExecutionResultMsg{action: state.ActionPull, status: status, requestID: request.ID, requestEpoch: request.Epoch, baseline: request.OperationBaseline, operationBaseline: request.OperationBaseline, operationBaselineSet: false, mode: mode, stale: true}
		}
		baseline := request.OperationBaseline
		if !samePullSnapshotIdentity(pullSnapshotIdentity(status, request.Epoch), baseline) {
			return pullExecutionResultMsg{action: state.ActionPull, status: status, requestID: request.ID, requestEpoch: request.Epoch, baseline: baseline, operationBaseline: baseline, operationBaselineSet: request.OperationBaselineSet, mode: mode, stale: true}
		}
		args := []string{"pull", "--no-rebase", "--no-edit"}
		if mode == PullModeRebase {
			args = []string{"pull", "--rebase"}
		}
		_, executionErr := repo.Run(args...)
		status, refreshErr := repo.Status(context.Background(), limit)
		var err error
		if executionErr != nil {
			err = executionErr
		} else if refreshErr != nil {
			err = refreshErr
		}
		return pullExecutionResultMsg{action: state.ActionPull, status: status, err: err, executionErr: executionErr, refreshErr: refreshErr, requestID: request.ID, requestEpoch: request.Epoch, baseline: baseline, operationBaseline: baseline, operationBaselineSet: request.OperationBaselineSet, mode: mode}
	}
}
