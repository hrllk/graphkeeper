package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

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

// The road summary is the interaction in progress, so it survives a short rail.
//
// It used to be the third line of the Details panel, under focus and date, on
// the reasoning that a short rail folds away everything below the first few
// lines. At a 22-row terminal that panel gets three rows, so third was still
// folded away -- the commit that named the feature promised to stop hiding its
// state, and it was still hidden.
func TestRoadSummarySurvivesAShortRail(t *testing.T) {
	status := git.Status{
		Root: "/repo", Branch: "main", Head: "a39d548",
		LocalBranches: []string{"main"}, LocalBranchesKnown: true, LocalBranchesFresh: true,
		GraphCommits: []git.GraphCommit{
			{Hash: "a39d548", Subject: "newest", Decorations: []string{"HEAD -> main"}, Parents: []string{"b47e659"}, CommitDate: "2026-09-10T04:15:16+09:00"},
			{Hash: "b47e659", Subject: "middle", Parents: []string{"c58f76a"}, CommitDate: "2026-09-09T23:15:16+09:00"},
			{Hash: "c58f76a", Subject: "oldest", CommitDate: "2026-09-07T10:00:00+09:00"},
		},
	}
	for _, height := range []int{18, 22, 30, 40} {
		m := model{repositoryState: repositoryState{repoStatus: status}}
		m.width, m.height = 110, height
		m.graphRange = resolveGraphRange("a39d548", "c58f76a", 1, []string{"c58f76a", "b47e659"}, nil, nil)

		if !strings.Contains(ansi.Strip(renderAppView(m)), "road:") {
			t.Errorf("height %d folded the road summary out of sight", height)
		}
	}
}
