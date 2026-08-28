package app

import (
	"strings"

	"hrllk/graphkeeper/internal/git"
)

func isLocalGraphPointer(rs git.Status, cursor int, laneCursor int) bool {
	rows := graphRows(rs)
	if cursor < 0 || cursor >= len(rows) {
		return false
	}
	row := rows[cursor]
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		return false
	}

	localSet := make(map[string]struct{}, len(rs.LocalBranches))
	for _, b := range rs.LocalBranches {
		localSet[b] = struct{}{}
	}

	if row.Graph != "" {
		for _, dec := range row.Commit.Decorations {
			dec = strings.TrimSpace(dec)
			if strings.HasPrefix(dec, "HEAD -> ") {
				return true
			}
			if dec == "" || strings.HasPrefix(dec, "origin/") || strings.HasPrefix(dec, "tag: ") {
				continue
			}
			if _, ok := localSet[dec]; ok {
				return true
			}
			// Local branch names are not always loaded. Without the inventory the
			// only thing left to go on is the shape of the decoration, and
			// remote refs and tags were already skipped above.
			if !rs.LocalBranchesKnown {
				return true
			}
		}
		return false
	}

	if laneCursor >= 0 && laneCursor < len(row.Before) {
		return row.Before[laneCursor].Side == laneLocal
	}
	if laneCursor >= 0 && laneCursor < len(row.After) {
		return row.After[laneCursor].Side == laneLocal
	}
	return row.Lane == laneCursor
}
