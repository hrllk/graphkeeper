package commitinspectoradapter

import (
	"context"
	"testing"

	"hrllk/graphkeeper/internal/commitinspector"
	"hrllk/graphkeeper/internal/git"
)

func TestMapInspectorStatusPreservesSpecialFileKinds(t *testing.T) {
	cases := map[string]commitinspector.ChangedFileStatus{
		"A":        commitinspector.StatusAdded,
		"M":        commitinspector.StatusModified,
		"D":        commitinspector.StatusDeleted,
		"R":        commitinspector.StatusRenamed,
		"C":        commitinspector.StatusCopied,
		"B":        commitinspector.StatusBinary,
		"S":        commitinspector.StatusSubmodule,
		"ModeOnly": commitinspector.StatusModeOnly,
	}
	for raw, want := range cases {
		if got := mapInspectorStatus(git.CommitDiffFile{Status: raw}); got != want {
			t.Fatalf("status %q = %q, want %q", raw, got, want)
		}
	}
}

func TestCanonicalInspectorFileIDUsesRawGitIdentity(t *testing.T) {
	file := git.CommitDiffFile{ID: "raw-id", Path: "sanitized/path.go"}
	if got := canonicalInspectorFileID("commit", "parent", file, 0); got != "raw-id" {
		t.Fatalf("stable ID = %q, want raw Git ID", got)
	}
}

func TestAdapterRejectsUnconfiguredReader(t *testing.T) {
	result := (&gitCommitInspectorReader{}).InspectCommit(context.Background(), commitinspector.CommitRequest{Commit: "abc"})
	if result.State != commitinspector.PaneError || result.Error == nil || result.Error.Kind != "configuration" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestNormalizeInspectorWindowBoundsAndDefaults(t *testing.T) {
	window, err := normalizeInspectorWindow(commitinspector.DiffWindowRequest{})
	if err != nil || window.MaxLines != defaultInspectorMaxLines || window.MaxBytes != defaultInspectorMaxBytes {
		t.Fatalf("defaults = %#v, err=%v", window, err)
	}
	if _, err := normalizeInspectorWindow(commitinspector.DiffWindowRequest{MaxBytes: 1}); err == nil || err.Kind != "configuration" {
		t.Fatal("expected structural-size configuration error")
	}
}

func TestDiffWindowHunksPreserveRowsAndContinuation(t *testing.T) {
	rows := []git.DiffRow{{Kind: "modified", OldLine: 1, NewLine: 1, From: "old", To: "new", FromPresent: true, ToPresent: true}}
	got := diffWindowHunks([]string{"@@ -1 +1 @@", "-old", "+new"}, rows)
	if len(got) != 1 || len(got[0].Rows) != 1 || got[0].Rows[0].Kind != "modified" {
		t.Fatalf("unexpected hunks: %#v", got)
	}
}
