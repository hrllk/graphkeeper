package commitinspectoradapter

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"hrllk/graphkeeper/internal/commitinspector"
	"hrllk/graphkeeper/internal/git"
)

func TestInspectorErrorMappingMatrixAndPrecedence(t *testing.T) {
	typedNotFound := fmt.Errorf("lookup failed: %w", git.ErrCommitNotFound)
	for _, tc := range []struct {
		name string
		err  error
		kind string
	}{
		{name: "typed commit not found", err: typedNotFound, kind: "commit_not_found"},
		{name: "generic git exit", err: errors.New("git exited with status 1"), kind: "git_exit"},
		{name: "timeout", err: context.DeadlineExceeded, kind: "timeout"},
		{name: "cancellation", err: context.Canceled, kind: "canceled"},
		{name: "cancellation wins over typed not found", err: errors.Join(context.Canceled, git.ErrCommitNotFound), kind: "canceled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := inspectorError(tc.err)
			if got == nil || got.Kind != tc.kind {
				t.Fatalf("inspectorError(%v) = %#v, want kind %q", tc.err, got, tc.kind)
			}
		})
	}
}

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

func TestInspectorFileOccurrencesAreStableAcrossReordering(t *testing.T) {
	files := []git.CommitDiffFile{{Status: "M", Path: "b.go"}, {Status: "M", Path: "a.go"}, {Status: "M", Path: "b.go"}}
	got := inspectorFileOccurrences(files)
	if fmt.Sprint(got) != "[0 0 1]" {
		t.Fatalf("occurrences = %v, want [0 0 1]", got)
	}
	reordered := []git.CommitDiffFile{{Status: "M", Path: "b.go"}, {Status: "M", Path: "b.go"}, {Status: "M", Path: "a.go"}}
	other := inspectorFileOccurrences(reordered)
	if canonicalInspectorFileKey(mapInspectorStatus(files[0]), files[0].OldPath, files[0].Path, got[0]) != canonicalInspectorFileKey(mapInspectorStatus(reordered[0]), reordered[0].OldPath, reordered[0].Path, other[0]) || canonicalInspectorFileKey(mapInspectorStatus(files[2]), files[2].OldPath, files[2].Path, got[2]) != canonicalInspectorFileKey(mapInspectorStatus(reordered[1]), reordered[1].OldPath, reordered[1].Path, other[1]) {
		t.Fatal("reordering changed canonical identities")
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
