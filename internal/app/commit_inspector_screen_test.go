package app

import (
	"fmt"
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

func TestCommitInspectorScreenHeightRowBudget(t *testing.T) {
	m := model{inspectorState: inspectorState{commitInspectorSnapshot: CommitSnapshot{FullHash: "abcdef", Files: []ChangedFile{{Path: "old/path.go", OldPath: "old/path.go"}}}}}
	for _, tc := range []struct {
		height int
		want   []string
		avoid  []string
	}{
		{3, []string{"COMMIT", "unsupported height"}, []string{"path:", "Esc back"}},
		{5, []string{"COMMIT", "unsupported height"}, []string{"path:", "Esc back"}},
		{6, []string{"COMMIT", "unsupported height", "path:"}, []string{"Esc back"}},
		{8, []string{"COMMIT", "unsupported height", "path:"}, []string{"Esc back"}},
		{9, []string{"COMMIT", "unsupported height", "path:", "Esc back"}, nil},
		{11, []string{"COMMIT", "unsupported height", "path:", "Esc back"}, nil},
	} {
		t.Run(fmt.Sprint(tc.height), func(t *testing.T) {
			got := renderCommitInspectorScreen(m, 40, tc.height)
			if lipgloss.Height(got) != tc.height {
				t.Fatalf("height=%d rendered=%d: %q", tc.height, lipgloss.Height(got), got)
			}
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("height=%d missing %q: %q", tc.height, want, got)
				}
			}
			for _, avoid := range tc.avoid {
				if strings.Contains(got, avoid) {
					t.Errorf("height=%d unexpectedly contains %q: %q", tc.height, avoid, got)
				}
			}
		})
	}
}

func TestCommitInspectorScreenStatusRowAppearsExactlyOnce(t *testing.T) {
	m := model{inspectorState: inspectorState{commitInspectorSnapshot: CommitSnapshot{FullHash: "abc"}, commitInspectorDiffWindow: DiffWindow{}}}
	got := renderCommitInspectorScreen(m, 80, 12)
	if strings.Count(got, "No textual changes") != 1 {
		t.Fatalf("ready status count = %d, output=%q", strings.Count(got, "No textual changes"), got)
	}
	m.commitInspectorDiffWindow.HasMore = true
	got = renderCommitInspectorScreen(m, 80, 12)
	if strings.Count(got, "partial") != 1 {
		t.Fatalf("partial status count = %d, output=%q", strings.Count(got, "partial"), got)
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

func TestCommitInspectorScreenTask43RendererMatrix(t *testing.T) {
	for _, noColor := range []bool{false, true} {
		t.Run(fmt.Sprintf("no-color-%v", noColor), func(t *testing.T) {
			if noColor {
				t.Setenv("NO_COLOR", "1")
			} else {
				t.Setenv("NO_COLOR", "")
			}
			m := model{inspectorState: inspectorState{commitInspectorSnapshot: CommitSnapshot{
				FullHash: "matrix-commit", Files: []ChangedFile{{Status: StatusRenamed, OldPath: "old.go", Path: "new.go"}},
			}}}
			for _, width := range []int{40, 60, 80} {
				for height := 1; height <= 13; height++ {
					got := renderCommitInspectorScreen(m, width, height)
					if height > 2 && lipgloss.Height(got) != height {
						t.Fatalf("%dx%d rendered height %d: %q", width, height, lipgloss.Height(got), got)
					}
					if height <= 2 {
						if !strings.Contains(got, "unsupported height") || strings.Contains(got, "matrix-commit") {
							t.Errorf("%dx%d violated single-row unsupported contract: %q", width, height, got)
						}
						continue
					}
					for _, want := range []string{"COMMIT matrix-commit"} {
						if !strings.Contains(got, want) {
							t.Errorf("%dx%d missing %q: %q", width, height, want, got)
						}
					}
					if height < 12 && !strings.Contains(got, "unsupported height") {
						t.Errorf("%dx%d missing %q: %q", width, height, "unsupported height", got)
					}
					if height >= 6 && (!strings.Contains(got, "old.go") || !strings.Contains(got, "new.go")) {
						t.Errorf("%dx%d omitted rename identity: %q", width, height, got)
					}
					if height < 6 && strings.Contains(got, "path:") {
						t.Errorf("%dx%d unexpectedly retained path: %q", width, height, got)
					}
					if height < 9 && strings.Contains(got, "Esc back") {
						t.Errorf("%dx%d unexpectedly retained close action: %q", width, height, got)
					}
					if height >= 9 && !strings.Contains(got, "Esc back") {
						t.Errorf("%dx%d omitted close action: %q", width, height, got)
					}
				}
			}
		})
	}
}
