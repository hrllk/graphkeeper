package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestCommitInspectorScreenMatrixKeepsFrameAndPartialFooter(t *testing.T) {
	m := model{
		inspectorState: inspectorState{
			commitInspectorOpen: true,
			commitInspectorSnapshot: CommitSnapshot{
				FullHash: "abcdef0123456789", Subject: "change", AuthorName: "dev", Parent: "parent",
				Files: []ChangedFile{{StableID: "f", Status: StatusModified, Path: "internal/app/model.go"}},
			},
			commitInspectorDiffWindow: DiffWindow{FileID: "f", HasMore: true, PartialReason: PartialLineLimit, NextStartLine: 4,
				Hunks: []DiffHunk{{Header: "@@ -1 +1 @@", Rows: []PairedRow{{Kind: "context", From: CodeLine{Number: 1, Text: "same"}, To: CodeLine{Number: 1, Text: "same"}, FromPresent: true, ToPresent: true}}}},
			},
		},
	}
	for _, size := range [][2]int{{40, 12}, {60, 20}, {80, 30}} {
		got := renderCommitInspectorScreen(m, size[0], size[1])
		if lipgloss.Width(got) != size[0] || lipgloss.Height(got) != size[1] {
			t.Fatalf("size %dx%d rendered as %dx%d", size[0], size[1], lipgloss.Width(got), lipgloss.Height(got))
		}
		if !strings.Contains(got, "n next") {
			t.Fatalf("partial screen %dx%d omitted n next: %q", size[0], size[1], got)
		}
		if !strings.Contains(got, "FROM parent") {
			t.Fatalf("screen %dx%d omitted parent direction", size[0], size[1])
		}
	}
}

func TestCommitInspectorScreenNoColorPreservesContextAndPathIdentity(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := model{inspectorState: inspectorState{commitInspectorSnapshot: CommitSnapshot{FullHash: "abc", Subject: "subject", AuthorName: "dev", IsRoot: true, Files: []ChangedFile{{StableID: "f", Status: StatusAdded, Path: "src/한글/very_long_file.go"}}}, commitInspectorDiffWindow: DiffWindow{FileID: "f", Hunks: []DiffHunk{{Header: "@@", Rows: []PairedRow{{Kind: "context", From: CodeLine{Number: 1, Text: "same"}, To: CodeLine{Number: 1, Text: "same"}, FromPresent: true, ToPresent: true}}}}}}}
	got := renderCommitInspectorScreen(m, 40, 12)
	if !strings.Contains(got, "ROOT COMMIT") || !strings.Contains(got, "very_long_file.go") || !strings.Contains(got, "same") {
		t.Fatalf("no-color screen lost identity/context: %q", got)
	}
}
