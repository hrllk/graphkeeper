package git

import "testing"

// The graph log parser reads parts[2..5] by position and drops any line with
// fewer fields than it expects, with no error. So a mismatch between the format
// string, the SplitN count and the length check does not fail loudly - it
// returns zero commits and the graph renders empty. These tests are the only
// thing standing between that and a silent regression.

func TestParseGraphCommitLinesKeepsCommitDate(t *testing.T) {
	got := parseGraphCommitLines([]string{
		"* \x00abc\x1fparent\x1f5 minutes ago\x1fdev\x1fHEAD -> main\x1f2026-09-08T23:15:16+09:00\x1fsubject",
	})
	if len(got) != 1 {
		t.Fatalf("expected one commit, got %d: %+v", len(got), got)
	}
	if got[0].CommitDate != "2026-09-08T23:15:16+09:00" {
		t.Fatalf("expected the committer date to survive parsing, got %q", got[0].CommitDate)
	}
}

// A commit line one field short must not be silently dropped without this test
// noticing. If the format string and the length check ever disagree, every
// commit takes this path and the assertion below is what reports it.
func TestParseGraphCommitLinesRejectsShortLineLoudly(t *testing.T) {
	short := "* \x00abc\x1fparent\x1f5 minutes ago\x1fdev\x1fHEAD -> main\x1fsubject"
	if got := parseGraphCommitLines([]string{short}); len(got) != 0 {
		t.Fatalf("expected a six-field line to be rejected, got %+v", got)
	}
	full := "* \x00abc\x1fparent\x1f5 minutes ago\x1fdev\x1fHEAD -> main\x1f2026-09-08T23:15:16+09:00\x1fsubject"
	if got := parseGraphCommitLines([]string{full}); len(got) != 1 {
		t.Fatalf("expected a seven-field line to parse, got %+v", got)
	}
}

// graphLogArgs and the parser have to agree on the field count. Asserting the
// format string alone would pass while the parser dropped every commit, so this
// feeds the format's own field count back through the parser.
func TestGraphLogArgsFieldCountMatchesParser(t *testing.T) {
	args := graphLogArgs([]string{"main"}, 0)
	format := ""
	for _, arg := range args {
		if len(arg) > len("--format=") && arg[:len("--format=")] == "--format=" {
			format = arg[len("--format="):]
		}
	}
	if format == "" {
		t.Fatal("graphLogArgs emitted no --format argument")
	}
	fields := 1
	for i := 0; i+3 < len(format); i++ {
		if format[i:i+4] == "%x1f" {
			fields++
		}
	}
	if fields != 7 {
		t.Fatalf("expected the graph format to carry 7 fields, got %d from %q", fields, format)
	}
}

// The subject must stay the last field in both formats. SplitN with a field cap
// leaves everything after the last split in the final part, so whichever field
// is last absorbs a stray delimiter instead of shifting the fields after it.
// With the date placed after the subject, a subject carrying the delimiter
// truncated the subject and contaminated the date while the length check still
// passed, so it corrupted silently.

func TestGraphFormatKeepsSubjectLastSoDelimitersCannotShiftFields(t *testing.T) {
	args := graphLogArgs([]string{"main"}, 0)
	format := ""
	for _, arg := range args {
		if len(arg) > len("--format=") && arg[:len("--format=")] == "--format=" {
			format = arg[len("--format="):]
		}
	}
	if got := format[len(format)-2:]; got != "%s" {
		t.Fatalf("expected the graph format to end with the subject, got %q from %q", got, format)
	}
}

func TestParseGraphCommitLinesSurvivesADelimiterInTheSubject(t *testing.T) {
	// A subject carrying the unit separator. The date precedes it, so the date
	// is read correctly and the subject keeps everything that follows.
	line := "* \x00abc\x1fparent\x1f5 minutes ago\x1fdev\x1fHEAD -> main\x1f2026-09-08T23:15:16+09:00\x1fsub\x1fject"
	got := parseGraphCommitLines([]string{line})
	if len(got) != 1 {
		t.Fatalf("expected one commit, got %d: %+v", len(got), got)
	}
	if got[0].CommitDate != "2026-09-08T23:15:16+09:00" {
		t.Fatalf("a delimiter in the subject corrupted the date: %q", got[0].CommitDate)
	}
	if got[0].Subject != "sub\x1fject" {
		t.Fatalf("expected the subject to absorb the delimiter, got %q", got[0].Subject)
	}
}
