package graph

import "testing"

// Nodes and rowsFromGraph both rebuild Node field by field, so a new field that
// is added to the struct but not to those two literals reaches the renderer as
// a zero value with nothing failing. These assertions run the value through
// both paths.

func snapshotWithDate() Snapshot {
	return Snapshot{
		Commits: []Commit{{
			Hash:       "abc",
			Subject:    "subject",
			CommitDate: "2026-09-08T23:15:16+09:00",
		}},
	}
}

func TestNodesKeepsCommitDate(t *testing.T) {
	nodes := Nodes(snapshotWithDate())
	if len(nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(nodes))
	}
	if nodes[0].CommitDate != "2026-09-08T23:15:16+09:00" {
		t.Fatalf("Nodes dropped the committer date, got %q", nodes[0].CommitDate)
	}
}

func TestRowsKeepsCommitDateOnTheRawGraphPath(t *testing.T) {
	snapshot := snapshotWithDate()
	// A non-empty Graph prefix routes Rows through rowsFromGraph, which is the
	// path a real repository takes; rowsLegacy reuses Nodes and is covered above.
	snapshot.Commits[0].Graph = "* "
	rows := Rows(snapshot)
	found := false
	for _, row := range rows {
		if row.Commit.Hash != "abc" {
			continue
		}
		found = true
		if row.Commit.CommitDate != "2026-09-08T23:15:16+09:00" {
			t.Fatalf("rowsFromGraph dropped the committer date, got %q", row.Commit.CommitDate)
		}
	}
	if !found {
		t.Fatalf("expected a row for the commit, got %+v", rows)
	}
}
