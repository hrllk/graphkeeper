package app

import (
	"strings"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func executionDetail(action state.Action, target string, rs git.Status) string {
	switch action {
	case state.ActionPull:
		return "Upstream pointer is now reflected in the local branch."
	case state.ActionMerge:
		return "Merge complete. HEAD now reflects " + emptyDash(rs.Branch) + " with target " + target + "."
	case state.ActionRebase:
		return "Rebase complete. The branch was replayed on top of " + target + "."
	case state.ActionReset:
		return "Hard reset complete. HEAD now points at " + target + "."
	case state.ActionDeleteBranch:
		if strings.HasPrefix(target, "origin/") {
			return "Origin branch deleted: " + target + "."
		}
		return "Branch deleted: " + target + "."
	case state.ActionDeleteTag:
		return "Tag deleted: " + target + "."
	case state.ActionDeleteRemoteTag:
		return "Remote tag deleted: " + target + "."
	default:
		return "Action complete."
	}
}

func findRemoteCommitHash(rs git.Status, upstream string) string {
	if upstream == "" {
		return ""
	}
	target := normalizeRemoteUpstream(upstream)
	for _, commit := range rs.GraphCommits {
		if commitHasRemoteDecoration(commit, target) {
			return commit.Hash
		}
	}
	return ""
}

func normalizeRemoteUpstream(upstream string) string {
	return strings.TrimPrefix(upstream, "refs/remotes/")
}

func commitHasRemoteDecoration(commit git.GraphCommit, target string) bool {
	for _, dec := range commit.Decorations {
		decTrim := strings.TrimSpace(dec)
		if decTrim == target || "origin/"+decTrim == target || decTrim == "origin/"+target {
			return true
		}
	}
	return false
}
