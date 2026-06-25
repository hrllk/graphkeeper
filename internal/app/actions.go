package app

import (
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
	return rs.Root != "" && !rs.Detached && !rs.NoRemote && !rs.NoUpstream && !rs.MergeInProgress
}

func canCreateBranch(rs git.Status) bool {
	return !rs.WorktreeDirty
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
