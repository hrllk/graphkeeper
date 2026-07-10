package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

func testGraphSearchModel(rs git.Status) model {
	m := testKeyHandlingModel(nil, rs)
	m.status = state.New().WithBrowse()
	return m
}

func TestGraphSearchMatchesHashTitleAndBranch(t *testing.T) {
	rs := git.Status{
		GraphCommits: []git.GraphCommit{
			{Hash: "fea00001", Subject: "zeta"},
			{Hash: "abc00002", Subject: "zeta", Decorations: []string{"HEAD -> feature/login"}},
			{Hash: "abc00003", Subject: "deploy feature now"},
		},
	}

	matches := graphSearchMatches(buildGraphSearchIndex(rs), "fea")
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(matches))
	}
	if matches[0].Row != 0 {
		t.Fatalf("expected hash match first, got row %d", matches[0].Row)
	}
	if matches[1].Row != 1 {
		t.Fatalf("expected branch match second, got row %d", matches[1].Row)
	}
	if matches[2].Row != 2 {
		t.Fatalf("expected title match third, got row %d", matches[2].Row)
	}
}

func TestHighlightSearchTextMarksMatchedSegments(t *testing.T) {
	got := highlightSearchText("feature/login", "feat", false)
	want := searchMatchMark.Render("feat")
	if !strings.Contains(got, want) {
		t.Fatalf("expected non-focused highlight, got %q", ansi.Strip(got))
	}

	focused := highlightSearchText("feature/login", "feat", true)
	wantFocus := searchFocusMark.Render("feat")
	if !strings.Contains(focused, wantFocus) {
		t.Fatalf("expected focused highlight, got %q", ansi.Strip(focused))
	}
}

func TestGraphSearchSupportsUnanchoredRegexPattern(t *testing.T) {
	rs := git.Status{
		GraphCommits: []git.GraphCommit{
			{Hash: "a1", Subject: "zeta"},
			{Hash: "a2", Subject: "zeta", Decorations: []string{"HEAD -> feature/login"}},
			{Hash: "a3", Subject: "deploy feature now"},
		},
	}

	matches := graphSearchMatches(buildGraphSearchIndex(rs), "fea.*login")
	if len(matches) != 1 {
		t.Fatalf("expected 1 regex match, got %d", len(matches))
	}
	if matches[0].Row != 1 {
		t.Fatalf("expected regex match to hit branch decoration row, got %d", matches[0].Row)
	}
}

func TestRenderGraphLineWithSearchHighlightsVisibleFields(t *testing.T) {
	row := graphRow{
		Commit: graphNode{
			Hash:        "feat0001",
			Subject:     "feat commit",
			Decorations: []string{"HEAD -> feature/login"},
		},
	}

	got := renderGraphLineWithSearch(row, true, true, 0, []string{"feature/login"}, 20, 80, false, 0, "feat")
	if !strings.Contains(got, searchFocusMark.Render("feat")) {
		t.Fatalf("expected search highlight in rendered graph line, got %q", ansi.Strip(got))
	}
}

func TestGraphSearchSelectionMovesCursorScrollAndLane(t *testing.T) {
	rows := make([]git.GraphCommit, 0, 20)
	for i := 0; i < 20; i++ {
		subject := "row"
		if i == 18 {
			subject = "needle target"
		}
		rows = append(rows, git.GraphCommit{
			Hash:    "commit-" + string(rune('a'+(i%10))),
			Subject: subject,
		})
	}
	rs := git.Status{GraphCommits: rows}
	m := testGraphSearchModel(rs)
	m.width = 140
	m.height = 40
	m.graphSearchDraft = "needle"
	m.graphSearchOpen = true

	got := applyGraphSearchSelection(m)
	commits := graphRows(rs)
	wantPage := graphPageSizeForRows(&got, commits, 18, graphContentHeightForModel(&got))
	wantScroll := clampScroll(18, len(commits), wantPage)

	if got.graphSearchOpen {
		t.Fatal("expected search popup to close after selection")
	}
	if got.sectionCursor[sectionGraph] != 18 {
		t.Fatalf("expected graph cursor to move to row 18, got %d", got.sectionCursor[sectionGraph])
	}
	if got.graphScroll != wantScroll {
		t.Fatalf("expected graph scroll %d, got %d", wantScroll, got.graphScroll)
	}
	if got.graphLaneCursor != graph.PointerLane(commits[18]) {
		t.Fatalf("expected graph lane cursor to sync with selected row, got %d", got.graphLaneCursor)
	}
}

func TestGraphSearchNAndPrevCycleThroughMatches(t *testing.T) {
	rs := git.Status{
		GraphCommits: []git.GraphCommit{
			{Hash: "a1", Subject: "feat one"},
			{Hash: "a2", Subject: "feat two"},
			{Hash: "a3", Subject: "feat three"},
		},
	}
	m := testGraphSearchModel(rs)
	m.graphSearchQuery = "feat"
	m.graphSearchCursor = 2

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected repeat search to stay synchronous, got %v", cmd)
	}
	if got.sectionCursor[sectionGraph] != 0 {
		t.Fatalf("expected n to wrap to first match, got %d", got.sectionCursor[sectionGraph])
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected reverse repeat search to stay synchronous, got %v", cmd)
	}
	if got.sectionCursor[sectionGraph] != 2 {
		t.Fatalf("expected N to wrap to last match, got %d", got.sectionCursor[sectionGraph])
	}
}

func TestGraphSearchEmptyEnterClearsRepeatMode(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testGraphSearchModel(git.Status{
		Root:   fixture.root,
		Branch: "main",
		Head:   fixture.initialHash,
		GraphCommits: []git.GraphCommit{
			{Hash: fixture.initialHash, Subject: "base"},
		},
	})
	m.graphSearchOpen = true
	m.graphSearchDraft = ""
	m.graphSearchQuery = "base"
	m.graphSearchCursor = 1

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected empty enter to stay synchronous, got %v", cmd)
	}
	if got.graphSearchOpen {
		t.Fatal("expected popup to close after empty enter")
	}
	if got.graphSearchQuery != "" {
		t.Fatalf("expected repeat query to clear, got %q", got.graphSearchQuery)
	}
	if got.graphSearchCursor != 0 {
		t.Fatalf("expected repeat cursor to reset, got %d", got.graphSearchCursor)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected branch create to stay synchronous, got %v", cmd)
	}
	if !got.branchOpen {
		t.Fatal("expected branch create to work again after repeat clear")
	}
}

func TestGraphSearchEscClosesPopupWithoutClearingQuery(t *testing.T) {
	m := testGraphSearchModel(git.Status{
		GraphCommits: []git.GraphCommit{{Hash: "a1", Subject: "feat one"}},
	})
	m.graphSearchOpen = true
	m.graphSearchDraft = "feat"
	m.graphSearchQuery = "feat"
	m.graphSearchError = "No graph match."

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected escape to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode after cancel, got %s", got.status.Mode)
	}
	if got.graphSearchOpen {
		t.Fatal("expected popup to close after escape")
	}
	if got.graphSearchDraft != "" {
		t.Fatalf("expected draft to clear, got %q", got.graphSearchDraft)
	}
	if got.graphSearchError != "" {
		t.Fatalf("expected error to clear, got %q", got.graphSearchError)
	}
	if got.graphSearchQuery != "feat" {
		t.Fatalf("expected confirmed query to remain available, got %q", got.graphSearchQuery)
	}
	if got.graphSearchCursor != 0 {
		t.Fatalf("expected search cursor to stay unchanged, got %d", got.graphSearchCursor)
	}
}

func TestGraphSearchEscInBrowseClearsRepeatState(t *testing.T) {
	m := testGraphSearchModel(git.Status{
		GraphCommits: []git.GraphCommit{{Hash: "a1", Subject: "feat one"}},
	})
	m.graphSearchQuery = "feat"
	m.graphSearchCursor = 2

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected escape to stay synchronous, got %v", cmd)
	}
	if got.graphSearchQuery != "" {
		t.Fatalf("expected confirmed query to clear in browse mode, got %q", got.graphSearchQuery)
	}
	if got.graphSearchCursor != 0 {
		t.Fatalf("expected search cursor to reset in browse mode, got %d", got.graphSearchCursor)
	}
}

func TestGraphSearchIsGraphSectionOnly(t *testing.T) {
	m := testGraphSearchModel(git.Status{
		GraphCommits: []git.GraphCommit{{Hash: "a1", Subject: "feat one"}},
	})
	m.activeSection = sectionCurrent

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected slash in non-graph section to stay synchronous, got %v", cmd)
	}
	if got.graphSearchOpen {
		t.Fatal("expected slash to be ignored outside graph section")
	}
}

func TestGraphSearchQuestionMarkOpensHiddenHotkeys(t *testing.T) {
	m := testGraphSearchModel(git.Status{
		GraphCommits: []git.GraphCommit{{Hash: "a1", Subject: "feat one"}},
	})

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected question mark in graph section to stay synchronous, got %v", cmd)
	}
	if got.graphSearchOpen {
		t.Fatal("expected question mark to stop opening graph search popup")
	}
	if !got.hiddenHotkeysOpen {
		t.Fatal("expected question mark to open hidden hotkeys drawer")
	}
}

func TestGraphSearchNoMatchKeepsPopupOpen(t *testing.T) {
	m := testGraphSearchModel(git.Status{
		GraphCommits: []git.GraphCommit{{Hash: "a1", Subject: "feat one"}},
	})
	m.graphSearchOpen = true
	m.graphSearchDraft = "missing"

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected failed search to stay synchronous, got %v", cmd)
	}
	if !got.graphSearchOpen {
		t.Fatal("expected popup to stay open when no match exists")
	}
	if got.graphSearchError != "No graph match." {
		t.Fatalf("expected no-match error, got %q", got.graphSearchError)
	}
}

func TestGraphSearchBranchCreateStillWorksWhenRepeatCleared(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main"},
		GraphCommits:  []git.GraphCommit{{Hash: fixture.initialHash, Subject: "base"}},
	})
	m.activeSection = sectionGraph
	m.graphSearchQuery = ""

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected branch create shortcut to stay synchronous, got %v", cmd)
	}
	if !got.branchOpen {
		t.Fatal("expected branch create shortcut to remain available when repeat is cleared")
	}
}
