package git

import "testing"

func TestParseCommitDiffRowsPairsRemovedAndAddedLines(t *testing.T) {
	rows := parseCommitDiffRows([]string{"@@ -2,2 +2,2 @@", "-old", "+new", " context"})
	if len(rows) != 2 {
		t.Fatalf("expected two paired rows, got %#v", rows)
	}
	if rows[0].Kind != "modified" || rows[0].From != "old" || rows[0].To != "new" || rows[0].OldLine != 2 || rows[0].NewLine != 2 {
		t.Fatalf("unexpected modified row: %#v", rows[0])
	}
	if rows[1].Kind != "context" || rows[1].OldLine != 3 || rows[1].NewLine != 3 {
		t.Fatalf("unexpected context row: %#v", rows[1])
	}
}

func TestCommitInspectorSanitizesTerminalTextAndKeepsRawPathIdentity(t *testing.T) {
	if got := sanitizeTerminalText("subject\x1b[31mred\x1b[0m\x01"); got != "subjectred" {
		t.Fatalf("unexpected sanitized text %q", got)
	}
	plain := parseCommitDiffFiles("M\x00dir/file.go\x00")[0]
	colored := parseCommitDiffFiles("M\x00dir/fi\x1b[31mle.go\x00")[0]
	if plain.ID == colored.ID {
		t.Fatal("raw path variants must not collapse to one StableID")
	}
	if colored.Path != "dir/file.go" {
		t.Fatalf("expected display path sanitization, got %q", colored.Path)
	}
}
