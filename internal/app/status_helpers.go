package app

import (
	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func worktreeState(rs git.Status) state.WorktreeState {
	if rs.WorktreeDirty {
		return state.WorktreeStateDirty
	}
	if rs.Root == "" {
		return ""
	}
	return state.WorktreeStateClean
}

func applyRepoMetadata(s state.Status, rs git.Status) state.Status {
	s.WorktreeState = worktreeState(rs)
	return s
}
