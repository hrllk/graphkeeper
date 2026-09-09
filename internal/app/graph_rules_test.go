package app

import (
	"testing"

	"hrllk/graphkeeper/internal/git"
)

// isLocalGraphPointer answers differently for the same commit depending on
// whether git supplied a raw graph prefix for its row. On a raw-prefix row it
// reads that row's decorations and never looks at the lane; without one it
// reads the lane, which is seeded at a tip and carries down to the commits the
// branch passed through.
//
// The gate note used to describe only the second half as the whole rule, and
// the note is what later changes are argued from. This pins what the code
// actually does so the two cannot drift apart again while the split stands.
//
// See docs/20260701-0005-feature-graph-section-merge-rebase-gate-note.md
// (2026-09-10 correction, task 7.4) and task 12.4 for unifying the two.
func TestLocalPointerJudgementSplitsOnTheRawGraphPrefix(t *testing.T) {
	base := func(rawPrefix string) git.Status {
		return git.Status{
			Branch: "main", Head: "aaa1111",
			LocalBranches: []string{"main"}, LocalBranchesKnown: true, LocalBranchesFresh: true,
			GraphCommits: []git.GraphCommit{
				{Hash: "aaa1111", Subject: "tip", Parents: []string{"bbb2222"}, Decorations: []string{"HEAD -> main"}},
				{Hash: "bbb2222", Subject: "on main's path, undecorated", Parents: []string{"ccc3333"}, Graph: rawPrefix},
				{Hash: "ccc3333", Subject: "on main's path, undecorated", Graph: rawPrefix},
			},
		}
	}

	// Without a raw prefix the lane decides, and the lane runs down the path.
	drawn := base("")
	for row := 0; row < 3; row++ {
		if !isLocalGraphPointer(drawn, row, graphRows(drawn)[row].Lane) {
			t.Errorf("row %d: an undecorated commit on the local lane should be local when the app draws the row", row)
		}
	}

	// With one, only the row's own decorations count.
	raw := base("* ")
	if !isLocalGraphPointer(raw, 0, graphRows(raw)[0].Lane) {
		t.Error("the decorated tip should be local either way")
	}
	for row := 1; row < 3; row++ {
		if isLocalGraphPointer(raw, row, graphRows(raw)[row].Lane) {
			t.Errorf("row %d: a raw-prefix row is judged on its decorations alone, and it has none", row)
		}
	}
}
