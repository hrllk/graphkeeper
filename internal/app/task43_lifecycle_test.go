package app

import (
	"context"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"hrllk/graphkeeper/internal/commitinspector"
	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

type task43InspectorReader struct {
	metadata InspectorResult[CommitSnapshot]
	diff     InspectorResult[DiffWindow]
	commits  []CommitRequest
	diffs    []DiffRequest
}

func (r *task43InspectorReader) InspectCommit(_ context.Context, req CommitRequest) InspectorResult[CommitSnapshot] {
	r.commits = append(r.commits, req)
	r.metadata.Commit, r.metadata.RequestID, r.metadata.RepositoryEpoch = req.Commit, req.RequestID, req.RepositoryEpoch
	return r.metadata
}

func (r *task43InspectorReader) LoadDiff(_ context.Context, req DiffRequest) InspectorResult[DiffWindow] {
	r.diffs = append(r.diffs, req)
	r.diff.Commit, r.diff.Parent, r.diff.FileID = req.Commit, req.Parent, req.FileID
	r.diff.RequestID, r.diff.RepositoryEpoch, r.diff.Window = req.RequestID, req.RepositoryEpoch, req.Window
	r.diff.Value.FileID = req.FileID
	return r.diff
}

func TestTask43GraphEnterRunsTopLevelInspectorFlow(t *testing.T) {
	reader := &task43InspectorReader{
		metadata: InspectorResult[CommitSnapshot]{State: PaneReady, Value: CommitSnapshot{
			FullHash: "graph-commit", Parent: "graph-parent", Subject: "graph subject",
			Files: []ChangedFile{{StableID: "file-1", CanonicalKey: "file-key", Path: "src/graph.go"}},
		}},
		diff: InspectorResult[DiffWindow]{State: PaneReady, Value: DiffWindow{Hunks: []DiffHunk{{Header: "@@", Rows: []PairedRow{{Kind: "context", From: CodeLine{Number: 1, Text: "same"}, To: CodeLine{Number: 1, Text: "same"}, FromPresent: true, ToPresent: true}}}}}},
	}
	m := model{
		repositoryState: repositoryState{repoStatus: git.Status{GraphCommits: []git.GraphCommit{{Hash: "graph-commit", Subject: "graph subject"}}}},
		navigationState: navigationState{activeSection: sectionGraph, sectionCursor: map[graphSection]int{sectionGraph: 0}, graphScroll: 7},
		status:          state.New().WithBrowse(), inspectorReader: reader,
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("top-level Graph Enter did not start metadata request")
	}
	opened := updated.(model)
	if !opened.commitInspectorOpen || len(reader.commits) != 0 {
		t.Fatalf("Enter did not open Inspector through the expected command boundary: %#v", opened.inspectorState)
	}
	updated, cmd = opened.Update(cmd())
	metadataApplied := updated.(model)
	if cmd == nil || len(reader.commits) != 1 || reader.commits[0].Commit != "graph-commit" || metadataApplied.commitInspectorSelectedFileID != "file-1" {
		t.Fatalf("metadata flow = cmd=%v requests=%#v state=%#v", cmd != nil, reader.commits, metadataApplied.inspectorState)
	}
	updated, cmd = metadataApplied.Update(cmd())
	ready := updated.(model)
	if cmd != nil || len(reader.diffs) != 1 || reader.diffs[0].FileID != "file-1" || reader.diffs[0].Window != (DiffWindowRequest{StartLine: 0, MaxLines: 2000, MaxBytes: 1 << 20}) {
		t.Fatalf("first diff flow = cmd=%v requests=%#v state=%#v", cmd != nil, reader.diffs, ready.inspectorState)
	}
	rendered := ansi.Strip(renderCommitInspectorScreen(ready, 80, 12))
	if cmd != nil || !strings.Contains(rendered, "graph-commit") || !strings.Contains(rendered, "src/graph.go") {
		t.Fatalf("diff/render flow = cmd=%v rendered=%q", cmd != nil, rendered)
	}
	updated, cmd = ready.Update(tea.KeyMsg{Type: tea.KeyEsc})
	closed := updated.(model)
	if cmd != nil || closed.commitInspectorOpen || closed.sectionCursor[sectionGraph] != 0 || closed.graphScroll != 7 {
		t.Fatalf("Esc did not return to preserved Graph state: cmd=%v state=%#v", cmd != nil, closed.navigationState)
	}
}

func TestTask43TopLevelRefreshRevalidatesOpenInspector(t *testing.T) {
	reader := &task43InspectorReader{
		metadata: InspectorResult[CommitSnapshot]{State: PaneReady, Commit: "abc", RequestID: 8, RepositoryEpoch: 4, Value: CommitSnapshot{FullHash: "abc", Parent: "parent", Files: []ChangedFile{{StableID: "new-file", CanonicalKey: "key", Path: "new.go"}}}},
		diff:     InspectorResult[DiffWindow]{State: PaneReady, Commit: "abc", Parent: "parent", FileID: "new-file", RequestID: 9, RepositoryEpoch: 4, Window: DiffWindowRequest{StartLine: 0, MaxLines: 2000, MaxBytes: 1 << 20}, Value: DiffWindow{FileID: "new-file"}},
	}
	m := model{
		repositoryState: repositoryState{repositoryEpoch: 4, refreshGeneration: 3},
		inspectorState:  inspectorState{commitInspectorOpen: true, commitInspectorRequest: 7, commitInspectorEpoch: 4, commitInspectorRequestedCommit: "abc", commitInspectorSelectedFileID: "old-file", commitInspectorSelectedCanonicalKey: "key", commitInspectorError: "old metadata error", commitInspectorDiffError: "old diff error", commitInspectorSnapshot: CommitSnapshot{FullHash: "abc", Parent: "parent", Files: []ChangedFile{{StableID: "old-file", CanonicalKey: "key", Path: "old.go"}}}},
		inspectorReader: reader,
	}
	gotModel, cmd := m.Update(refreshedSnapshotMsg{refreshGeneration: 3, result: ReadSnapshotResult{RepositoryEpoch: 4, ErrorKind: ReadErrorNone, Snapshot: ReadSnapshot{Graph: task212Graph("fresh")}}})
	got := gotModel.(model)
	if cmd == nil || got.graphReadSnapshot.Head != "fresh" || got.repoStatus.GraphCommits[0].Hash != "fresh" || got.commitInspectorRequest != 8 || !got.commitInspectorRevalidating {
		t.Fatalf("accepted refresh lifecycle = cmd=%v graph=%#v repo=%#v inspector=%#v", cmd != nil, got.graphReadSnapshot, got.repoStatus, got.inspectorState)
	}
	gotModel, cmd = got.Update(cmd())
	got = gotModel.(model)
	if cmd == nil || len(reader.commits) != 1 || reader.commits[0].RequestID != 8 || got.commitInspectorRequest != 9 || got.commitInspectorSelectedFileID != "new-file" || got.commitInspectorError != "" {
		t.Fatalf("metadata revalidation = cmd=%v requests=%#v inspector=%#v", cmd != nil, reader.commits, got.inspectorState)
	}
	gotModel, cmd = got.Update(cmd())
	got = gotModel.(model)
	if cmd != nil || len(reader.diffs) != 1 || reader.diffs[0].RequestID != 9 || reader.diffs[0].FileID != "new-file" || got.commitInspectorDiffWindow.FileID != "new-file" || got.commitInspectorDiffError != "" {
		t.Fatalf("diff revalidation = cmd=%v requests=%#v inspector=%#v", cmd != nil, reader.diffs, got.inspectorState)
	}
}

func TestTask43RefreshFailurePreservesOpenInspector(t *testing.T) {
	for _, kind := range []ReadErrorKind{ReadErrorInvalid, ReadErrorRepository, ReadErrorCanceled} {
		t.Run(string(kind), func(t *testing.T) {
			canceled := 0
			m := model{repositoryState: repositoryState{repositoryEpoch: 4, refreshGeneration: 3}, inspectorState: inspectorState{commitInspectorOpen: true, commitInspectorRequest: 7, commitInspectorSnapshot: CommitSnapshot{FullHash: "abc"}, commitInspectorSelectedFileID: "file", commitInspectorLoading: true, commitInspectorCancel: func() { canceled++ }}}
			before := m.inspectorState
			gotModel, cmd := m.Update(refreshedSnapshotMsg{refreshGeneration: 3, result: ReadSnapshotResult{RepositoryEpoch: 4, ErrorKind: kind}})
			got := gotModel.(model)
			before.commitInspectorCancel, before.commitInspectorContext = nil, nil
			got.commitInspectorCancel, got.commitInspectorContext = nil, nil
			if cmd != nil || canceled != 0 || !reflect.DeepEqual(got.inspectorState, before) {
				t.Fatalf("refresh failure changed Inspector: cmd=%v canceled=%d got=%#v before=%#v", cmd != nil, canceled, got.inspectorState, before)
			}
		})
	}
}

func TestTask43CloseCommitInspectorInvalidatesLateResults(t *testing.T) {
	canceled := 0
	m := model{inspectorState: inspectorState{
		commitInspectorOpen: true, commitInspectorRequest: 7,
		commitInspectorCancel: func() { canceled++ },
	}}
	got := m.closeCommitInspector()
	if got.commitInspectorRequest != 8 || canceled != 1 || got.commitInspectorOpen {
		t.Fatalf("close lifecycle = request %d canceled %d open %v", got.commitInspectorRequest, canceled, got.commitInspectorOpen)
	}
}

func TestTask43RendererUnsupportedHeightWins(t *testing.T) {
	m := model{inspectorState: inspectorState{
		commitInspectorSnapshot:        CommitSnapshot{FullHash: "abc", Files: []ChangedFile{{StableID: "f", Path: "x.go"}}},
		commitInspectorMetadataLoading: true, commitInspectorStale: true,
	}}
	rendered := ansi.Strip(renderCommitInspectorScreen(m, 60, 11))
	if !strings.Contains(rendered, "unsupported height") || strings.Contains(rendered, "Loading…") || strings.Contains(rendered, "Repository changed") {
		t.Fatalf("unsupported renderer precedence: %q", rendered)
	}
}

func TestTask43CanonicalKeySurvivesStableIDChange(t *testing.T) {
	key := canonicalFileKey(commitinspector.StatusRenamed, "old.go", "new.go", 0)
	if key == "" || key == canonicalFileKey(commitinspector.StatusRenamed, "old.go", "new.go", 1) {
		t.Fatalf("canonical key is not occurrence-sensitive: %q", key)
	}
}

func TestTask43StaleContinuationIsNoOpAtEveryBoundary(t *testing.T) {
	m := model{inspectorState: inspectorState{
		commitInspectorOpen: true, commitInspectorStale: true,
		commitInspectorSnapshot:      CommitSnapshot{FullHash: "abc", Parent: "p"},
		commitInspectorDiffWindow:    DiffWindow{FileID: "f", HasMore: true, NextStartLine: 10},
		commitInspectorWindowRequest: DiffWindowRequest{StartLine: 0, MaxLines: 20, MaxBytes: 128},
	}}
	if next, cmd := m.handleCommitInspectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}); cmd != nil || next.(model).commitInspectorRequest != m.commitInspectorRequest {
		t.Fatalf("stale n mutated inspector: cmd=%v model=%#v", cmd != nil, next)
	}
	msg := inspectorContinuationKeyMsg{}
	if next, cmd := m.Update(msg); cmd != nil || next.(model).commitInspectorRequest != m.commitInspectorRequest {
		t.Fatalf("stale key message mutated inspector: cmd=%v model=%#v", cmd != nil, next)
	}
	if next, cmd := m.Update(ContinuationRequested{Commit: "abc", Parent: "p", FileID: "f", Window: m.commitInspectorWindowRequest}); cmd != nil || next.(model).commitInspectorRequest != m.commitInspectorRequest {
		t.Fatalf("stale continuation mutated inspector: cmd=%v model=%#v", cmd != nil, next)
	}
}

func TestTask43DuplicateStableIDDoesNotCanonicalFallback(t *testing.T) {
	m := model{inspectorState: inspectorState{
		commitInspectorOpen: true, commitInspectorRevalidating: true, commitInspectorRequest: 4,
		commitInspectorEpoch: 2, commitInspectorRequestedCommit: "abc",
		commitInspectorSelectedFileID: "same", commitInspectorSelectedCanonicalKey: "key-a",
	}}
	msg := commitInspectorResultMsg{Metadata: &InspectorResult[CommitSnapshot]{State: PaneReady, Commit: "abc", RequestID: 4, RepositoryEpoch: 2, Value: CommitSnapshot{FullHash: "abc", Files: []ChangedFile{{StableID: "same", CanonicalKey: "key-a"}, {StableID: "same", CanonicalKey: "key-b"}}}}}
	got, cmd := m.applyCommitInspectorResult(msg)
	if cmd != nil || got.commitInspectorOpen != true || got.commitInspectorError == "" {
		t.Fatalf("duplicate stable identity was accepted: cmd=%v state=%#v", cmd != nil, got.inspectorState)
	}
}

func TestTask43UnsupportedHeightAlwaysVisible(t *testing.T) {
	m := model{}
	for height := 1; height <= 2; height++ {
		got := ansi.Strip(renderCommitInspectorScreen(m, 40, height))
		if !strings.Contains(got, "unsupported height") {
			t.Fatalf("height %d omitted unsupported marker: %q", height, got)
		}
	}
}
