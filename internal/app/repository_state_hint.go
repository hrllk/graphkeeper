package app

import (
	"fmt"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func repositoryStateHint(rs git.Status, dirtyBlocked bool, loadErr error) string {
	switch {
	case loadErr != nil:
		return "Repository unavailable"
	case rs.Root == "":
		return "Not a Git repository"
	case rs.MergeInProgress:
		return "Merge in progress"
	case rs.RebaseInProgress:
		return "Rebase in progress"
	case rs.CherryPickInProgress:
		return "Cherry-pick in progress"
	case rs.Detached:
		return "Detached HEAD · branch 선택 필요"
	case rs.EmptyRepo:
		return "No commits yet"
	case rs.NoRemote && rs.NoUpstream:
		return "No remote or upstream"
	case rs.NoRemote:
		return "No remote"
	case rs.NoUpstream:
		return "No upstream"
	case isRepositoryDiverged(rs):
		if rs.Branch != "" && rs.Upstream != "" {
			return fmt.Sprintf("Diverged · %s ↔ %s", rs.Branch, rs.Upstream)
		}
		return "Diverged"
	case rs.WorktreeDirty && dirtyBlocked:
		return "Working tree is dirty"
	default:
		return ""
	}
}

func repositoryStateHintForModel(m *model) string {
	if !m.repoSnapshotLoaded && m.err == nil {
		return ""
	}
	return repositoryStateHint(m.repoStatus, dirtyWorktreeBlocked(m.status), m.err)
}

func isRepositoryDiverged(rs git.Status) bool {
	tracking, ok := rs.Tracking[rs.Branch]
	return ok && tracking.Ahead > 0 && tracking.Behind > 0
}

func dirtyWorktreeBlocked(status state.Status) bool {
	return status.Mode == state.ModeBlocked && status.Block == state.BlockDirtyTree
}
