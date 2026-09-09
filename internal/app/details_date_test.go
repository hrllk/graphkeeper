package app

import "testing"

// The Details content width is cardWidth-4 (view_shell.go:378), which is 13 at
// an 80-column terminal. Both forms the plan assumed - "2026-09-08 23:15" and
// "2026-09-08" - overflow that, and a truncated date reads as a different date
// rather than as a shortened one. compactWhenISO steps down instead, and drops
// the row entirely rather than emit something misleading.
func TestCompactWhenISOStepsDownWithBudget(t *testing.T) {
	const stamp = "2026-09-08T23:15:16+09:00"
	for _, tt := range []struct {
		name   string
		budget int
		want   string
	}{
		{"wide keeps the time", 16, "2026-09-08 23:15"},
		{"medium drops the time", 10, "2026-09-08"},
		{"narrow keeps month and day", 5, "09-08"},
		{"too narrow drops the row", 4, ""},
		{"80-column rail budget", 13 - len("date: "), "09-08"},
	} {
		if got := compactWhenISO(stamp, tt.budget); got != tt.want {
			t.Fatalf("%s: compactWhenISO(budget %d) = %q, want %q", tt.name, tt.budget, got, tt.want)
		}
	}
}

// The offset in the stamp is the committer's own, not the reader's. Rendering
// must not shift it into local time, or "committed at 3am" becomes a different
// hour for whoever is reading.
func TestCompactWhenISOKeepsTheRecordedOffset(t *testing.T) {
	if got := compactWhenISO("2026-09-08T23:15:16+09:00", 16); got != "2026-09-08 23:15" {
		t.Fatalf("expected the +09:00 offset to be preserved, got %q", got)
	}
	if got := compactWhenISO("2026-09-08T23:15:16-07:00", 16); got != "2026-09-08 23:15" {
		t.Fatalf("expected the -07:00 offset to be preserved, got %q", got)
	}
}

// A value that is present but unparseable costs one cell, not a commit, so it
// renders the panel's existing placeholder rather than dropping the row.
func TestCompactWhenISODistinguishesMissingFromUnparseable(t *testing.T) {
	if got := compactWhenISO("", 16); got != "-" {
		t.Fatalf("expected an empty stamp to render %q, got %q", "-", got)
	}
	if got := compactWhenISO("not a date", 16); got != "-" {
		t.Fatalf("expected an unparseable stamp to render %q, got %q", "-", got)
	}
	if got := compactWhenISO("", 4); got != "-" {
		t.Fatalf("expected a missing stamp to stay %q even at a tiny budget, got %q", "-", got)
	}
}
