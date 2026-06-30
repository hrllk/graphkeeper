package app

import (
	"fmt"
	"strings"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func deriveStatus(rs git.Status) state.Status {
	switch {
	case rs.Root == "":
		return applyRepoMetadata(state.New().WithBlocked(state.BlockNoRepo, "Not inside a Git repository.", "Run this tool from a repo root."), rs)
	case rs.MergeInProgress || rs.RebaseInProgress:
		status := state.New().WithBrowse()
		status.Message = "Merge/rebase in progress."
		status.Detail = "Press Enter to abort."
		return applyRepoMetadata(status, rs)
	case rs.Detached:
		return applyRepoMetadata(state.New().WithBlocked(state.BlockDetached, "Detached HEAD.", "Pick a branch before running pull, merge, or rebase."), rs)
	case rs.EmptyRepo:
		return applyRepoMetadata(state.New().WithEmpty("No commits yet."), rs)
	case rs.NoRemote && rs.NoUpstream:
		return applyRepoMetadata(state.New().WithBlocked(state.BlockNoRemote, "No remote or upstream.", "Set a remote target first."), rs)
	default:
		return applyRepoMetadata(state.New().WithBrowse(), rs)
	}
}

func actionPull(rs git.Status) state.Status {
	if rs.Root == "" {
		return applyRepoMetadata(state.New().WithBlocked(state.BlockNoRepo, "Not inside a Git repository.", "Run this tool from a repo root."), rs)
	}
	if rs.WorktreeDirty {
		return applyRepoMetadata(state.New().WithBlocked(state.BlockDirtyTree, "Working tree is dirty.", "Commit or stash changes first."), rs)
	}
	if rs.Detached {
		return applyRepoMetadata(state.New().WithBlocked(state.BlockDetached, "Detached HEAD.", "Pull needs a branch with an upstream."), rs)
	}
	if rs.MergeInProgress || rs.RebaseInProgress {
		return applyRepoMetadata(state.New().WithBlocked(state.BlockUnknown, "Merge/rebase already in progress.", "Abort or resolve it before pulling again."), rs)
	}
	if rs.NoRemote {
		return applyRepoMetadata(state.New().WithBlocked(state.BlockNoRemote, "No remote.", "Pull needs a remote."), rs)
	}
	if rs.NoUpstream {
		return applyRepoMetadata(state.New().WithBlocked(state.BlockNoUpstream, "No upstream.", "Set an upstream first."), rs)
	}
	return applyRepoMetadata(state.New().WithOutcome(state.ActionPull, "Pull ready.", "Will fetch and merge upstream changes.", true), rs)
}

func pullReady(rs git.Status) bool {
	return rs.Root != "" && !rs.WorktreeDirty && !rs.Detached && !rs.NoRemote && !rs.NoUpstream && !rs.MergeInProgress
}

func canCreateBranch(rs git.Status) bool {
	return rs.Root != "" && !rs.WorktreeDirty && !rs.MergeInProgress && !rs.RebaseInProgress
}

func branchNameExists(rs git.Status, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if rs.Branch == name && rs.Branch != "" && rs.Branch != "HEAD" {
		return true
	}
	for _, existing := range rs.Branches {
		if existing == name {
			return true
		}
	}
	return false
}

func branchCreateValidationError(rs git.Status, name, base string) error {
	name = strings.TrimSpace(name)
	base = strings.TrimSpace(base)
	switch {
	case rs.Root == "":
		return fmt.Errorf("not inside a git repository")
	case rs.WorktreeDirty:
		return fmt.Errorf("working tree is not clean")
	case rs.MergeInProgress || rs.RebaseInProgress:
		return fmt.Errorf("merge/rebase already in progress")
	case base == "":
		return fmt.Errorf("branch base is empty")
	case name == "":
		return fmt.Errorf("branch name is empty")
	case branchNameExists(rs, name):
		return fmt.Errorf("branch name already exists")
	default:
		return nil
	}
}

func branchCreateBaseValidationError(rs git.Status, base string) error {
	base = strings.TrimSpace(base)
	switch {
	case rs.Root == "":
		return fmt.Errorf("not inside a git repository")
	case rs.WorktreeDirty:
		return fmt.Errorf("working tree is not clean")
	case rs.MergeInProgress || rs.RebaseInProgress:
		return fmt.Errorf("merge/rebase already in progress")
	case base == "":
		return fmt.Errorf("branch base is empty")
	default:
		return nil
	}
}

func branchCreateBlockedStatusFromError(err error) state.Status {
	if err == nil {
		return state.New().WithBrowse()
	}
	switch {
	case strings.Contains(err.Error(), "not inside a git repository"):
		return state.New().WithBlocked(state.BlockNoRepo, "Not inside a Git repository.", "Run this tool from a repo root.")
	case strings.Contains(err.Error(), "working tree is not clean"):
		return state.New().WithBlocked(state.BlockDirtyTree, "Working tree is dirty.", "Commit or stash changes first.")
	case strings.Contains(err.Error(), "merge/rebase already in progress"):
		return state.New().WithBlocked(state.BlockUnknown, "Merge/rebase already in progress.", "Abort or resolve it before creating a branch.")
	case strings.Contains(err.Error(), "branch base is empty"):
		return state.New().WithBlocked(state.BlockTargetEmpty, "No branch base.", "Select a commit or branch first.")
	case strings.Contains(err.Error(), "branch name is empty"):
		return state.New().WithBlocked(state.BlockUnknown, "Branch name is empty.", "Enter a branch name.")
	case strings.Contains(err.Error(), "branch name already exists"):
		return state.New().WithBlocked(state.BlockUnknown, "Branch name already exists.", "Choose a different branch name.")
	default:
		return state.New().WithBlocked(state.BlockUnknown, "Branch creation failed.", err.Error())
	}
}

func actionPickTargets(rs git.Status, action state.Action) state.Status {
	if (action == state.ActionMerge || action == state.ActionRebase) && rs.Detached {
		return state.New().WithBlocked(state.BlockDetached, "Detached HEAD.", "Choose a branch before merging or rebasing.")
	}
	targets := buildActionTargetItems(rs)
	if action == state.ActionReset {
		targets = buildResetTargetItems(rs)
	}
	if len(targets) == 0 {
		return state.New().WithBlocked(state.BlockTargetEmpty, "No targets available.", "Create or fetch a branch first.")
	}
	status := state.New().WithTargetPick(action, targets)
	status.Message = "Choose a target."
	status.Detail = "Enter previews. Esc returns."
	return applyRepoMetadata(status, rs)
}

func checkoutTargetFromFocus(node graphNode) string {
	for _, decoration := range node.Decorations {
		decoration = strings.TrimSpace(decoration)
		if strings.HasPrefix(decoration, "HEAD -> ") {
			return strings.TrimPrefix(decoration, "HEAD -> ")
		}
		if strings.HasPrefix(decoration, "tag: ") {
			continue
		}
		if strings.Contains(decoration, "/") {
			return decoration
		}
		if decoration != "" {
			return decoration
		}
	}
	return ""
}

func selectedTarget(s state.Status) string {
	if s.Selected != "" {
		return s.Selected
	}
	if s.TargetIdx >= 0 && s.TargetIdx < len(s.Targets) {
		return s.Targets[s.TargetIdx].Ref
	}
	return ""
}
