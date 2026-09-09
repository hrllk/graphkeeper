package app

import (
	"errors"
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
		wantTo            string
		wantCount         int
	}{
		{
			name: "forward path", cursor: cursorHash,
			forward:  []string{cursorHash, "mid", "x"},
			wantKind: rangeForward, wantTo: cursorHash, wantCount: 3,
		},
		{
			name: "reverse pick is a path, not a fork", cursor: cursorHash,
			forward: nil, backward: []string{anchorHash, "mid"},
			wantKind: rangeBackward, wantTo: anchorHash, wantCount: 2,
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
