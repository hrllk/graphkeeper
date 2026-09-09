package app

import (
	"errors"
	"strings"
	"testing"
)

const (
	anchorHash = "aaaaaaa"
	cursorHash = "bbbbbbb"
)

// The direction test is the reason this function exists. `--ancestry-path A..B`
// comes back empty in three situations and only one of them is divergence, so
// reporting all three the same way would tell a user who picked the newer commit
// first that their history had forked.
func TestResolveGraphRangeTellsTheThreeEmptyCasesApart(t *testing.T) {
	for _, tt := range []struct {
		name              string
		cursor            string
		forward, backward []string
		wantKind          graphRangeKind
		wantFrom, wantTo  string
		wantCount         int
	}{
		{
			name: "forward path", cursor: cursorHash,
			forward:  []string{cursorHash, "mid", "x"},
			wantKind: rangeForward, wantFrom: anchorHash, wantTo: cursorHash, wantCount: 3,
		},
		{
			name: "reverse pick is a path, not a fork", cursor: cursorHash,
			forward: nil, backward: []string{anchorHash, "mid"},
			wantKind: rangeBackward, wantFrom: cursorHash, wantTo: anchorHash, wantCount: 2,
		},
		{
			name: "same commit is neither", cursor: anchorHash,
			wantKind: rangeSame,
		},
		{
			name: "both directions empty is the only divergence", cursor: cursorHash,
			forward: nil, backward: nil,
			wantKind: rangeDiverged,
		},
		{
			name: "no cursor yet", cursor: "",
			wantKind: rangeAnchorOnly,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveGraphRange(anchorHash, tt.cursor, 7, tt.forward, tt.backward, nil)
			if got == nil {
				t.Fatal("an anchored selection must not resolve to nil")
			}
			if got.Kind != tt.wantKind {
				t.Fatalf("kind = %d, want %d", got.Kind, tt.wantKind)
			}
			if got.From != tt.wantFrom {
				t.Fatalf("from = %q, want %q", got.From, tt.wantFrom)
			}
			if got.To != tt.wantTo {
				t.Fatalf("to = %q, want %q", got.To, tt.wantTo)
			}
			if got.Count != tt.wantCount {
				t.Fatalf("count = %d, want %d", got.Count, tt.wantCount)
			}
			if got.Epoch != 7 {
				t.Fatalf("epoch = %d, want 7", got.Epoch)
			}
			hasMembers := tt.wantKind == rangeForward || tt.wantKind == rangeBackward
			if hasMembers && len(got.Members) != tt.wantCount {
				t.Fatalf("members = %d, want %d", len(got.Members), tt.wantCount)
			}
			if !hasMembers && got.Members != nil {
				t.Fatalf("only a real path carries members, got %v", got.Members)
			}
		})
	}
}

// A failed query is not divergence either. Collapsing them would report a fork
// whenever git was unavailable.
func TestResolveGraphRangeKeepsFailureSeparateFromDivergence(t *testing.T) {
	got := resolveGraphRange(anchorHash, cursorHash, 1, nil, nil, errors.New("boom"))
	if got.Kind != rangeUnavailable {
		t.Fatalf("kind = %d, want rangeUnavailable", got.Kind)
	}
	if got.Err != "boom" {
		t.Fatalf("err = %q, want %q", got.Err, "boom")
	}
	if got.Members != nil {
		t.Fatalf("a failed query has no members, got %v", got.Members)
	}
}

// The nil pointer is the only "off". An empty anchor with an active kind was
// representable before and would have called AncestryPath with a blank ref,
// which git reads as HEAD.
func TestResolveGraphRangeHasNoAnchorlessState(t *testing.T) {
	if got := resolveGraphRange("", cursorHash, 1, []string{"x"}, nil, nil); got != nil {
		t.Fatalf("an empty anchor must resolve to nil, got %+v", got)
	}
	var off *graphRange
	if off.rangeAnchor() != "" || off.rangeMembers() != nil {
		t.Fatal("a nil selection must read through as empty")
	}
	if off.staleFor(3) {
		t.Fatal("a nil selection is never stale")
	}
}

// Graph rows are rebuilt when the epoch moves, so a selection from before then
// points at rows whose meaning has changed.
func TestGraphRangeStaleAcrossEpochs(t *testing.T) {
	sel := resolveGraphRange(anchorHash, cursorHash, 4, []string{cursorHash}, nil, nil)
	if sel.staleFor(4) {
		t.Fatal("same epoch is not stale")
	}
	if !sel.staleFor(5) {
		t.Fatal("a moved epoch must be stale")
	}
}

// A result that raced a refresh, or that answers for an anchor the user already
// moved, must not install itself.
func TestApplyGraphRangeMsgDiscardsRacedResults(t *testing.T) {
	base := model{}
	base.repositoryEpoch = 9
	base.graphRange = &graphRange{Anchor: anchorHash, Kind: rangeAnchorOnly, Epoch: 9}

	stale := applyGraphRangeMsg(base, graphRangeMsg{anchor: anchorHash, cursor: cursorHash, epoch: 8, forward: []string{cursorHash}})
	if stale.graphRange.Kind != rangeAnchorOnly {
		t.Fatal("a result from a previous epoch must be discarded")
	}

	moved := applyGraphRangeMsg(base, graphRangeMsg{anchor: "other", cursor: cursorHash, epoch: 9, forward: []string{cursorHash}})
	if moved.graphRange.Anchor != anchorHash || moved.graphRange.Kind != rangeAnchorOnly {
		t.Fatal("a result for a different anchor must be discarded")
	}

	cleared := base
	cleared.graphRange = nil
	if got := applyGraphRangeMsg(cleared, graphRangeMsg{anchor: anchorHash, cursor: cursorHash, epoch: 9, forward: []string{cursorHash}}); got.graphRange != nil {
		t.Fatal("a result for a cleared selection must be discarded")
	}

	live := applyGraphRangeMsg(base, graphRangeMsg{anchor: anchorHash, cursor: cursorHash, epoch: 9, forward: []string{cursorHash, "mid"}})
	if live.graphRange.Kind != rangeForward || live.graphRange.Count != 2 {
		t.Fatalf("a live result must install, got kind=%d count=%d", live.graphRange.Kind, live.graphRange.Count)
	}
}

// The road key is section-local, so it belongs in the graph group of the ?
// overlay and NOT in the main footer. decisions.md:8-9 reserves the footer for
// Global core navigation keys, and the footer is sourced only from
// globalHotkeyItems. Task 8 is already carrying ten working keys that appear
// nowhere, so this must not become the eleventh.
func TestRoadKeyIsDocumentedInTheOverlayAndNotTheFooter(t *testing.T) {
	for _, item := range globalHotkeyItems() {
		if item.key == "w" {
			t.Fatal("w is a graph-section action and must stay out of the main footer")
		}
	}
	m := model{}
	m.activeSection = sectionGraph
	found := false
	for _, section := range hiddenHotkeySections(m) {
		if section.title != "Graph" {
			continue
		}
		for _, group := range section.groups {
			for _, item := range group.items {
				if item.key == "w" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("w must appear in the Graph group of the ? overlay")
	}
}

// The direction is part of the answer. A count alone leaves the user unable to
// tell which commit is the ancestor, and telling them that is the point.
func TestRangeSummaryAlwaysCarriesTheDirection(t *testing.T) {
	forward := resolveGraphRange("aaaaaaaa1", "bbbbbbbb2", 1, []string{"bbbbbbbb2", "mid"}, nil, nil)
	got := forward.summaryText(60)
	if !strings.Contains(got, "aaaaaaaa") || !strings.Contains(got, "bbbbbbbb") {
		t.Fatalf("forward summary must name both ends: %q", got)
	}
	if !strings.Contains(got, "→") {
		t.Fatalf("forward summary must show the direction: %q", got)
	}
	if !strings.Contains(got, "2 commits") {
		t.Fatalf("forward summary must carry the count: %q", got)
	}
}

// A reversed pick says so rather than quietly swapping the endpoints. Once the
// interaction asks for FROM then TO the user has declared a direction, and
// showing the other one without saying so is still not what they asked for.
func TestRangeSummaryNamesAReversedPick(t *testing.T) {
	backward := resolveGraphRange("newer111", "older222", 1, nil, []string{"newer111", "mid"}, nil)
	got := backward.summaryText(60)
	if !strings.Contains(got, "reversed") {
		t.Fatalf("a reversed pick must say so: %q", got)
	}
	// The arrow points the way the path actually runs, older -> newer.
	if !strings.Contains(got, "older222 → newer111") {
		t.Fatalf("expected the real path direction, got %q", got)
	}
}

// The three empty cases stay distinguishable in the copy, not just in the state.
func TestRangeSummaryKeepsTheEmptyCasesApart(t *testing.T) {
	same := resolveGraphRange("aaa", "aaa", 1, nil, nil, nil).summaryText(60)
	diverged := resolveGraphRange("aaa", "bbb", 1, nil, nil, nil).summaryText(60)
	failed := resolveGraphRange("aaa", "bbb", 1, nil, nil, errors.New("boom")).summaryText(60)
	if same == diverged || same == failed || diverged == failed {
		t.Fatalf("the empty cases must read differently: same=%q diverged=%q failed=%q", same, diverged, failed)
	}
	if !strings.Contains(diverged, "diverged") {
		t.Fatalf("diverged summary = %q", diverged)
	}
	if !strings.Contains(failed, "unavailable") {
		t.Fatalf("failed summary = %q", failed)
	}
}

// The rail is 13 columns at an 80-column terminal, so the long phrasing rarely
// fits. It steps down to a shorter form rather than letting the viewport clip a
// sentence mid-word, and drops the row when even that will not fit.
func TestRangeSummaryStepsDown(t *testing.T) {
	sel := resolveGraphRange("aaaaaaaa1", "bbbbbbbb2", 1, []string{"bbbbbbbb2", "mid"}, nil, nil)
	wide := sel.summaryText(60)
	narrow := sel.summaryText(12)
	if wide == narrow {
		t.Fatalf("a narrow rail must shorten the summary: %q", wide)
	}
	if !strings.Contains(narrow, "2 commits") {
		t.Fatalf("the short form must keep the count: %q", narrow)
	}
	if got := sel.summaryText(3); got != "" {
		t.Fatalf("too narrow must drop the row, got %q", got)
	}
}

// No selection means no row at all.
func TestRangeSummaryIsEmptyWithoutASelection(t *testing.T) {
	var off *graphRange
	if got := off.summaryText(60); got != "" {
		t.Fatalf("a nil selection must render nothing, got %q", got)
	}
}
