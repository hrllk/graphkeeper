package app

import (
	"strings"
	"testing"
)

func inspectorDateFixture(authorDate, commitDate string) model {
	return model{inspectorState: inspectorState{
		commitInspectorSnapshot: CommitSnapshot{
			FullHash: "abc", Subject: "subject", AuthorName: "dev", Parent: "parent",
			AuthorDate: authorDate, CommitDate: commitDate,
			Files: []ChangedFile{{StableID: "f", Status: StatusModified, Path: "a.go"}},
		},
		commitInspectorDiffWindow: DiffWindow{FileID: "f", Hunks: []DiffHunk{{Header: "@@", Rows: []PairedRow{
			{Kind: "context", From: CodeLine{Number: 1, Text: "same"}, To: CodeLine{Number: 1, Text: "same"}, FromPresent: true, ToPresent: true},
		}}}},
	}}
}

// The header is six rows normally and seven once the two stamps disagree, which
// is what a rebase or a cherry-pick produces. A second identical row would cost
// a diff row for no information, so it only appears when it says something.
func TestScreenHeaderRowsGrowOnlyWhenTheStampsDiffer(t *testing.T) {
	const author = "2026-09-08T23:15:16+09:00"
	same := screenHeaderLines(inspectorDateFixture(author, author).commitInspectorSnapshot, ChangedFile{}, 60)
	if len(same) != 6 {
		t.Fatalf("expected 6 header rows when the stamps match, got %d: %q", len(same), same)
	}
	moved := screenHeaderLines(inspectorDateFixture(author, "2026-09-09T01:02:03+09:00").commitInspectorSnapshot, ChangedFile{}, 60)
	if len(moved) != 7 {
		t.Fatalf("expected 7 header rows when the stamps differ, got %d: %q", len(moved), moved)
	}
	if !strings.Contains(strings.Join(moved, "\n"), "committed:") {
		t.Fatalf("expected a committed: row when the stamps differ, got %q", moved)
	}
}

// This is the assertion the four-row constant used to make impossible. When the
// header grew, inspectorBodyRows kept reporting the old budget and
// renderCommitInspectorScreen trimmed the tail of the diff pane silently. The
// reported budget has to match what the header actually costs.
func TestInspectorBodyRowsTracksTheHeaderItIsGiven(t *testing.T) {
	for _, tt := range []struct{ height, headerRows, want int }{
		{30, 5, 20},
		{30, 6, 19},
		{20, 5, 10},
		{20, 6, 9},
		{12, 5, 2},
		{12, 6, 1},
		{12, 20, 0}, // a header taller than the frame keeps nothing
	} {
		if got := inspectorBodyRowsFor(tt.height, tt.headerRows); got != tt.want {
			t.Fatalf("inspectorBodyRowsFor(%d, %d) = %d, want %d", tt.height, tt.headerRows, got, tt.want)
		}
	}
}

// The stamp steps down with the width instead of being cut. A clipped timestamp
// reads as a different time, which is worse than showing less of it.
func TestScreenStampTextStepsDownWithBudget(t *testing.T) {
	const stamp = "2026-09-08T23:15:16+09:00"
	for _, tt := range []struct {
		budget int
		want   string
	}{
		// The full form is 26 columns. A 25-column guard clipped the offset to
		// "+09:0", which is exactly the wrong-looking value this step-down exists
		// to prevent, so 25 must fall through to the shorter form.
		{26, "2026-09-08 23:15:16 +09:00"},
		{25, "2026-09-08 23:15"},
		{16, "2026-09-08 23:15"},
		{15, "2026-09-08"},
		{10, "2026-09-08"},
		{9, ""},
	} {
		if got := screenStampText(stamp, tt.budget); got != tt.want {
			t.Fatalf("screenStampText(budget %d) = %q, want %q", tt.budget, got, tt.want)
		}
	}
	if got := screenStampText("", 25); got != "" {
		t.Fatalf("expected a missing stamp to drop the row, got %q", got)
	}
	if got := screenStampText("not a date", 25); got != "" {
		t.Fatalf("expected an unparseable stamp to drop the row, got %q", got)
	}
}

// The Inspector is where a commit is confirmed, so the whole date has to reach
// the screen at a width that can hold it.
func TestCommitInspectorScreenShowsTheDate(t *testing.T) {
	m := inspectorDateFixture("2026-09-08T23:15:16+09:00", "2026-09-08T23:15:16+09:00")
	m.width, m.height = 80, 30
	got := renderCommitInspectorScreen(m)
	if !strings.Contains(got, "date: 2026-09-08 23:15:16 +09:00") {
		t.Fatalf("expected the full stamp in the header, got %q", got)
	}
	if strings.Contains(got, "committed:") {
		t.Fatalf("expected no committed: row when the stamps match, got %q", got)
	}
}

// Nothing drawn has to report zero. Clamping to one made the scroll clamp and
// the page size believe a row existed at heights where
// renderCommitInspectorScreen had already dropped it.
func TestInspectorBodyRowsReportsZeroWhenTheFrameKeepsNothing(t *testing.T) {
	for _, tt := range []struct{ height, headerRows int }{
		// max(max(h-2,1)-H-3, 0). Only these actually reach zero:
		// (11,6)=0  (8,4)=-1  (1,4)=-6  (5,6)=-6
		// (11,5) and (10,4) legitimately keep one row, so they are not listed.
		{11, 6}, {8, 4}, {1, 4}, {5, 6},
	} {
		if got := inspectorBodyRowsFor(tt.height, tt.headerRows); got != 0 {
			t.Fatalf("inspectorBodyRowsFor(%d, %d) = %d, want 0", tt.height, tt.headerRows, got)
		}
	}
}

// The invariant has to hold below height 12 too. The earlier test started at 12
// and missed this band entirely, which is where the reported budget and the
// drawn rows came apart.
//
// Heights 1 through 9 keep nothing even after the header drops its date rows
// (the four-row header alone costs the whole frame), so the clamp must be zero
// there. From height 10 the frame keeps rows again and the clamp is meaningful;
// TestShortFramesDropTheDateRowsRatherThanTheDiff covers that band.
func TestInspectorBodyRowsMatchesTheFrameAtUnsupportedHeights(t *testing.T) {
	for _, height := range []int{1, 4, 8, 9} {
		m := scrollFixture(500, 3)
		m.commitInspectorSnapshot.AuthorDate = "2026-09-08T23:15:16+09:00"
		m.commitInspectorSnapshot.CommitDate = "2026-09-09T01:02:03+09:00"
		m.width, m.height = 120, height
		drawn := len(inspectorBodyLines(renderCommitInspectorScreen(m)))
		if want := m.inspectorBodyRowCount(); drawn != want {
			t.Fatalf("height %d drew %d body rows, inspectorBodyRowCount says %d", height, drawn, want)
		}
		if m.maxInspectorDiffScroll() != 0 {
			t.Fatalf("height %d has nothing to scroll but the clamp is %d", height, m.maxInspectorDiffScroll())
		}
		if m.inspectorScrollPage() != 0 {
			t.Fatalf("height %d shows nothing but Ctrl+U/D pages by %d", height, m.inspectorScrollPage())
		}
	}
}

// The date rows are the first thing dropped when the frame cannot afford them.
// decisions.md (2026-08-03) puts commit identity, the selected path and a
// minimum diff ahead of everything else on a short screen. Before this, a
// six-row header at height 11 left screenBody a budget of zero and it returned
// nil, so the Inspector showed no file list, no cursor and no diff at all -
// worse than the two body rows the same height used to give.
func TestShortFramesDropTheDateRowsRatherThanTheDiff(t *testing.T) {
	const (
		authored  = "2026-09-08T23:15:16+09:00"
		committed = "2026-09-09T01:02:03+09:00"
	)
	for _, height := range []int{11, 12, 13} {
		m := scrollFixture(500, 3)
		m.commitInspectorSnapshot.AuthorDate = authored
		m.commitInspectorSnapshot.CommitDate = committed
		m.width, m.height = 120, height

		if got := m.inspectorBodyRowCount(); got < 1 {
			t.Fatalf("height %d kept %d body rows; the header should have given way", height, got)
		}
		drawn := len(inspectorBodyLines(renderCommitInspectorScreen(m)))
		if want := m.inspectorBodyRowCount(); drawn != want {
			t.Fatalf("height %d drew %d body rows, inspectorBodyRowCount says %d", height, drawn, want)
		}
	}
}

// A tall frame keeps the dates; a short one does not. The same snapshot has to
// produce both, or the fallback is either never used or always used.
func TestHeaderKeepsDatesOnlyWhenTheFrameCanAffordThem(t *testing.T) {
	snapshot := CommitSnapshot{
		FullHash: "abc", Subject: "s", AuthorName: "dev", Parent: "p",
		AuthorDate: "2026-09-08T23:15:16+09:00",
		CommitDate: "2026-09-09T01:02:03+09:00",
	}
	tall := screenHeaderFor(snapshot, ChangedFile{}, 116, 40)
	if len(tall) != 7 {
		t.Fatalf("a tall frame should keep both date rows, got %d rows: %q", len(tall), tall)
	}
	// The ladder gives up one row at a time and stops as soon as the diff pane
	// has a line, so at this height the date rows go and parent stays. It used
	// to be all-dates-or-none, which threw away a row the frame could afford.
	// Below the floor it drops committed, date and parent and no further:
	// author, path, message and identity are not negotiable.
	short := screenHeaderFor(snapshot, ChangedFile{}, 116, 11)
	if len(short) != 5 {
		t.Fatalf("a short frame should drop the date rows and keep parent, got %d rows: %q", len(short), short)
	}
	if strings.Contains(strings.Join(short, "\n"), "date:") {
		t.Fatalf("expected the date rows dropped first, got %q", short)
	}
	floor := screenHeaderFor(snapshot, ChangedFile{}, 116, 8)
	if len(floor) != 4 {
		t.Fatalf("the floor is commit, message, author and path, got %d rows: %q", len(floor), floor)
	}
}
