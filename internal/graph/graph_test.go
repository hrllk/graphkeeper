package graph

import (
	"testing"

	"hrllk/git-graph-tui/internal/git"
)

func TestRowsUsesRawGraphPrefixWhenAvailable(t *testing.T) {
	rows := Rows(git.Status{
		GraphCommits: []git.GraphCommit{
			{Graph: "*", Hash: "a1", Subject: "first"},
			{Graph: "|", Hash: "b2", Subject: "second"},
		},
	})
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Commit.Hash != "a1" || rows[1].Commit.Hash != "b2" {
		t.Fatalf("unexpected row order: %#v", rows)
	}
	if rows[0].Graph != "*" || rows[1].Graph != "|" {
		t.Fatalf("expected raw graph prefix to be preserved, got %#v", rows)
	}
}

func TestRowsLegacyKeepsSiblingBranchesVisible(t *testing.T) {
	rows := Rows(git.Status{
		Branch: "main",
		Head:   "a",
		GraphCommits: []git.GraphCommit{
			{Hash: "a", Parents: []string{"b"}, Decorations: []string{"HEAD -> main", "main"}, Subject: "tip"},
			{Hash: "b", Parents: []string{"c"}, Decorations: []string{"origin/main"}, Subject: "next"},
			{Hash: "c", Subject: "root"},
		},
	})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if RowWidth(rows[0]) < 1 || RowWidth(rows[1]) < 1 || RowWidth(rows[2]) < 1 {
		t.Fatalf("expected all legacy rows to be visible, got widths %d %d %d", RowWidth(rows[0]), RowWidth(rows[1]), RowWidth(rows[2]))
	}
	if rows[0].Lane < 0 {
		t.Fatalf("expected a valid lane on the first row, got %d", rows[0].Lane)
	}
}

func TestRowWidthCollapsesMergeLineWhenAllLanesConverge(t *testing.T) {
	row := Row{
		Commit: Node{Hash: "base"},
		Before: []LaneRef{{Hash: "base"}, {Hash: "base"}, {Hash: "base"}},
		After:  []LaneRef{{Hash: "base"}},
	}
	if got := RowWidth(row); got != 1 {
		t.Fatalf("expected collapsed width 1, got %d", got)
	}
}

func TestMoveSelectableGraphPointerSkipsBlankRows(t *testing.T) {
	rows := []Row{
		{Commit: Node{Hash: "a"}},
		{Graph: "|\\"},
		{Commit: Node{Hash: "b"}},
	}
	if got := MoveSelectableGraphPointer(0, rows, 1); got != 2 {
		t.Fatalf("expected connector row to be skipped on move down, got %d", got)
	}
	if got := MoveSelectableGraphPointer(2, rows, -1); got != 0 {
		t.Fatalf("expected connector row to be skipped on move up, got %d", got)
	}
}

func TestCurrentFocusAndFindRowByHash(t *testing.T) {
	rows := []Row{{Commit: Node{Hash: "a1"}}, {Commit: Node{Hash: "b2"}}}
	if got := FindRowByHash(rows, "b2"); got != 1 {
		t.Fatalf("expected row hash lookup to return 1, got %d", got)
	}
	focus := CurrentFocus(git.Status{GraphCommits: []git.GraphCommit{{Hash: "a1"}, {Hash: "b2"}}}, 1)
	if focus.Hash != "b2" {
		t.Fatalf("expected focus to resolve to b2, got %#v", focus)
	}
}

func TestRowsInsertsVirtualConflictNodeDuringMerge(t *testing.T) {
	rows := Rows(git.Status{
		MergeInProgress: true,
		Head:            "abc123",
		ConflictTarget:  "def456",
		GraphCommits: []git.GraphCommit{{Hash: "abc123", Graph: "*", Subject: "tip"}},
	})
	if len(rows) != 2 {
		t.Fatalf("expected virtual conflict row plus original row, got %d", len(rows))
	}
	if rows[0].Commit.Hash != "VIRTUAL_CONFLICT_HASH" {
		t.Fatalf("expected virtual conflict row first, got %#v", rows[0])
	}
	if rows[0].Commit.Subject != "conflict" {
		t.Fatalf("expected conflict subject, got %#v", rows[0].Commit.Subject)
	}
	if rows[0].Commit.Parents[0] != "abc123" || rows[0].Commit.Parents[1] != "def456" {
		t.Fatalf("expected conflict parents to include head and target, got %#v", rows[0].Commit.Parents)
	}
}

func TestPageSizeClampsToMinimum(t *testing.T) {
	if got := PageSize(8); got != 3 {
		t.Fatalf("expected tiny layout to clamp to 3, got %d", got)
	}
}

