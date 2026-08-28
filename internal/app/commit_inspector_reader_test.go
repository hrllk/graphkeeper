package app

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
)

type fakeInspectorReader struct {
	metadata InspectorResult[CommitSnapshot]
	diff     InspectorResult[DiffWindow]
	seenMeta []CommitRequest
	seenDiff []DiffRequest
}

func (f *fakeInspectorReader) InspectCommit(_ context.Context, req CommitRequest) InspectorResult[CommitSnapshot] {
	f.seenMeta = append(f.seenMeta, req)
	result := f.metadata
	result.Commit = req.Commit
	result.RequestID = req.RequestID
	result.RepositoryEpoch = req.RepositoryEpoch
	return result
}

func (f *fakeInspectorReader) LoadDiff(_ context.Context, req DiffRequest) InspectorResult[DiffWindow] {
	f.seenDiff = append(f.seenDiff, req)
	result := f.diff
	result.Commit = req.Commit
	result.Parent = req.Parent
	result.FileID = req.FileID
	result.RequestID = req.RequestID
	result.RepositoryEpoch = req.RepositoryEpoch
	result.Window = req.Window
	return result
}

func TestNormalizeInspectorWindowDefaultsAndBounds(t *testing.T) {
	got, err := normalizeInspectorWindow(DiffWindowRequest{})
	if err != nil || got.MaxLines != defaultInspectorMaxLines || got.MaxBytes != defaultInspectorMaxBytes {
		t.Fatalf("defaults = %#v, err=%v", got, err)
	}
	for _, window := range []DiffWindowRequest{{MaxLines: -1}, {MaxBytes: -1}, {MaxLines: maxInspectorMaxLines + 1}, {MaxBytes: maxInspectorMaxBytes + 1}, {MaxBytes: 3}} {
		if _, err := normalizeInspectorWindow(window); err == nil || err.Kind != "configuration" {
			t.Fatalf("window %#v error = %#v", window, err)
		}
	}
}

func TestInspectorLogicalLineCountIgnoresHeaders(t *testing.T) {
	lines := []string{"diff --git a/a b/a", "@@ -1 +1 @@", "-old", "+new", "\\\\ No newline at end of file"}
	if got := inspectorLogicalLineCount(lines); got != 2 {
		t.Fatalf("logical count = %d", got)
	}
}

func TestCommitInspectorUsesInjectedReaderForMetadataAndDiff(t *testing.T) {
	fake := &fakeInspectorReader{
		metadata: InspectorResult[CommitSnapshot]{
			State: PaneReady,
			Value: CommitSnapshot{FullHash: "abc", Files: []ChangedFile{{StableID: "file-1", Status: StatusModified, Path: "main.go"}}},
		},
		diff: InspectorResult[DiffWindow]{
			State: PaneReady,
			Value: DiffWindow{FileID: "file-1"},
		},
	}
	m := newModelWithInspectorReader(&git.Repo{}, fake)
	m.commitInspectorOpen = true
	m.commitInspectorEpoch = 7
	m.commitInspectorRequestedCommit = "abc"
	m.commitInspectorRequest = 11
	m.commitInspectorMetadataLoading = true

	cmd := inspectCommitCommand(context.Background(), m, CommitRequest{Commit: "abc", RequestID: 11, RepositoryEpoch: 7})
	msg := cmd().(commitInspectorResultMsg)
	updated, next := m.applyCommitInspectorResult(msg)
	if next == nil {
		t.Fatal("metadata ready should start the first diff command")
	}
	resultMsg := next().(commitInspectorResultMsg)
	updated, _ = updated.applyCommitInspectorResult(resultMsg)
	if updated.commitInspectorDiffWindow.FileID != "file-1" {
		t.Fatalf("accepted diff result was not stored: %#v", updated.commitInspectorDiffWindow)
	}
	if updated.commitInspectorSnapshot.Files[0].StableID != "file-1" {
		t.Fatalf("unexpected metadata projection: %#v", updated.commitInspectorSnapshot)
	}
	if len(fake.seenMeta) != 1 || fake.seenMeta[0].RequestID != 11 {
		t.Fatalf("metadata request was not sent through injected reader: %#v", fake.seenMeta)
	}
}

func TestCommitInspectorRejectsStaleDiffWindowIdentity(t *testing.T) {
	m := model{inspectorState: inspectorState{commitInspectorOpen: true, commitInspectorRequest: 4, commitInspectorEpoch: 2}}
	m.commitInspectorSnapshot = CommitSnapshot{FullHash: "abc", Parent: "p", Files: []ChangedFile{{StableID: "current", Path: "a.go"}}}
	msg := commitInspectorResultMsg{Result: InspectorResult[DiffWindow]{
		State: PaneReady, Commit: "abc", Parent: "p", FileID: "stale", RequestID: 4, RepositoryEpoch: 2,
		Window: DiffWindowRequest{StartLine: 0, MaxLines: 2000, MaxBytes: 1 << 20},
		Value:  DiffWindow{FileID: "stale"},
	}}
	updated, _ := m.applyCommitInspectorResult(msg)
	if updated.commitInspectorDiffWindow.FileID != "" {
		t.Fatalf("stale FileID changed diff state: %#v", updated.commitInspectorDiffWindow)
	}
}

func TestCommitInspectorKeyToUpdateUsesFakeReader(t *testing.T) {
	fake := &fakeInspectorReader{metadata: InspectorResult[CommitSnapshot]{State: PaneReady, Value: CommitSnapshot{FullHash: "abc", Files: []ChangedFile{{StableID: "f", Path: "a.go"}}}}}
	m := newModelWithInspectorReader(&git.Repo{}, fake)
	m.commitInspectorOpen = true
	m.commitInspectorSnapshot = CommitSnapshot{FullHash: "abc", Files: []ChangedFile{{StableID: "f", Path: "a.go"}, {StableID: "g", Path: "b.go"}}}
	next, cmd := m.handleCommitInspectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	got := next.(model)
	if got.inspectorReader != fake || cmd == nil {
		t.Fatalf("Inspector key path did not retain injected reader or command: reader=%T same=%v cmdnil=%v cursor=%d files=%d snapshot=%#v", got.inspectorReader, got.inspectorReader == fake, cmd == nil, got.commitInspectorCursor, len(got.commitInspectorSnapshot.Files), got.commitInspectorSnapshot)
	}
}

// T25 regression. Every error path in the commit inspector adapter returns early
// without setting result.Value, so Value.FileID is empty. The identity guard used
// to fold Value.FileID into its request-identity check, which discarded the whole
// result and left commitInspectorDiffLoading true forever: the pane sat on
// "Loading…" with no error and no way out. Reproduced in the real binary by
// opening a commit that adds a file of more than MaxLines contiguous lines.
func TestCommitInspectorSurfacesDiffErrorWithZeroValue(t *testing.T) {
	window := DiffWindowRequest{StartLine: 0, MaxLines: 2000, MaxBytes: 1 << 20}
	m := model{inspectorState: inspectorState{
		commitInspectorOpen:          true,
		commitInspectorRequest:       4,
		commitInspectorEpoch:         2,
		commitInspectorLoading:       true,
		commitInspectorDiffLoading:   true,
		commitInspectorWindowRequest: window,
	}}
	m.commitInspectorSnapshot = CommitSnapshot{FullHash: "abc", Parent: "p", Files: []ChangedFile{{StableID: "f", Path: "a.go"}}}
	msg := commitInspectorResultMsg{Result: InspectorResult[DiffWindow]{
		State: PaneError, Commit: "abc", Parent: "p", FileID: "f", RequestID: 4, RepositoryEpoch: 2,
		Window: window,
		Error:  &InspectorError{Kind: "configuration", Message: "indivisible diff pair exceeds window budget"},
	}}

	updated, _ := m.applyCommitInspectorResult(msg)

	if updated.commitInspectorDiffLoading || updated.commitInspectorLoading {
		t.Fatalf("diff error left the pane loading: diff=%v meta=%v", updated.commitInspectorDiffLoading, updated.commitInspectorLoading)
	}
	if updated.commitInspectorDiffError != "indivisible diff pair exceeds window budget" {
		t.Fatalf("diff error was not surfaced: %q", updated.commitInspectorDiffError)
	}
}

// A PaneError carrying no Error value must still leave the pane, with copy the
// user can read rather than an indefinite spinner.
func TestCommitInspectorSurfacesDiffErrorWithoutMessage(t *testing.T) {
	window := DiffWindowRequest{StartLine: 0, MaxLines: 2000, MaxBytes: 1 << 20}
	m := model{inspectorState: inspectorState{
		commitInspectorOpen:          true,
		commitInspectorRequest:       1,
		commitInspectorEpoch:         1,
		commitInspectorDiffLoading:   true,
		commitInspectorWindowRequest: window,
	}}
	m.commitInspectorSnapshot = CommitSnapshot{FullHash: "abc", Parent: "p", Files: []ChangedFile{{StableID: "f", Path: "a.go"}}}
	msg := commitInspectorResultMsg{Result: InspectorResult[DiffWindow]{
		State: PaneError, Commit: "abc", Parent: "p", FileID: "f", RequestID: 1, RepositoryEpoch: 1, Window: window,
	}}

	updated, _ := m.applyCommitInspectorResult(msg)

	if updated.commitInspectorDiffLoading {
		t.Fatal("message-less diff error left the pane loading")
	}
	if updated.commitInspectorDiffError == "" {
		t.Fatal("message-less diff error produced no user-visible copy")
	}
}

// The payload identity check still has to run for a successful result, so a
// window for the wrong file cannot overwrite the current selection.
func TestCommitInspectorRejectsReadyResultWithMismatchedPayload(t *testing.T) {
	window := DiffWindowRequest{StartLine: 0, MaxLines: 2000, MaxBytes: 1 << 20}
	m := model{inspectorState: inspectorState{
		commitInspectorOpen:          true,
		commitInspectorRequest:       1,
		commitInspectorEpoch:         1,
		commitInspectorDiffLoading:   true,
		commitInspectorWindowRequest: window,
	}}
	m.commitInspectorSnapshot = CommitSnapshot{FullHash: "abc", Parent: "p", Files: []ChangedFile{{StableID: "f", Path: "a.go"}}}
	msg := commitInspectorResultMsg{Result: InspectorResult[DiffWindow]{
		State: PaneReady, Commit: "abc", Parent: "p", FileID: "f", RequestID: 1, RepositoryEpoch: 1, Window: window,
		Value: DiffWindow{FileID: "other"},
	}}

	updated, _ := m.applyCommitInspectorResult(msg)

	if updated.commitInspectorDiffWindow.FileID != "" {
		t.Fatalf("mismatched payload overwrote the diff window: %#v", updated.commitInspectorDiffWindow)
	}
	if updated.commitInspectorDiffError == "" {
		t.Fatal("mismatched payload was dropped silently")
	}
}

// The metadata path has the same shape as the diff path: a PaneError carrying no
// Error value must still produce copy the user can read, or the failure is silent
// (commit_inspector_screen.go only renders when commitInspectorError is non-empty).
func TestCommitInspectorSurfacesMetadataErrorWithoutMessage(t *testing.T) {
	m := model{inspectorState: inspectorState{
		commitInspectorOpen:            true,
		commitInspectorRequest:         3,
		commitInspectorEpoch:           1,
		commitInspectorMetadataLoading: true,
		commitInspectorLoading:         true,
		commitInspectorRequestedCommit: "abc",
	}}
	metadata := InspectorResult[CommitSnapshot]{State: PaneError, Commit: "abc", RequestID: 3, RepositoryEpoch: 1}
	msg := commitInspectorResultMsg{Metadata: &metadata}

	updated, _ := m.applyCommitInspectorResult(msg)

	if updated.commitInspectorMetadataLoading || updated.commitInspectorLoading {
		t.Fatalf("metadata error left the pane loading: meta=%v load=%v", updated.commitInspectorMetadataLoading, updated.commitInspectorLoading)
	}
	if updated.commitInspectorError == "" {
		t.Fatal("message-less metadata error produced no user-visible copy")
	}
}
