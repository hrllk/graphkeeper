package app

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestBuildTagSectionTargetsUsesTagEntries(t *testing.T) {
	got := buildTagSectionTargets(git.Status{
		TagEntries: []git.TagEntry{
			{Name: "v1.0.0", CommitHash: "abc1234", Subject: "initial release", RelativeAge: "2 days ago"},
			{Name: "v1.1.0", CommitHash: "def5678", Subject: "second release", RelativeAge: "1 day ago"},
		},
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 tag targets, got %d", len(got))
	}
	if got[0].Name != "v1.0.0" || got[0].Ref != "v1.0.0" || got[0].CommitHash != "abc1234" {
		t.Fatalf("unexpected first tag target: %#v", got[0])
	}
	if got[1].Subject != "second release" || got[1].RelativeAge != "1 day ago" {
		t.Fatalf("unexpected second tag target: %#v", got[1])
	}
}

func TestTagSectionEnterJumpsToGraphCommit(t *testing.T) {
	status := git.Status{
		Root: "repo",
		GraphCommits: []git.GraphCommit{
			{Hash: "aaa1111", Subject: "first"},
			{Hash: "bbb2222", Subject: "second"},
		},
		TagEntries: []git.TagEntry{
			{Name: "v2.0.0", CommitHash: "bbb2222", Subject: "second", RelativeAge: "2 days ago"},
			{Name: "v1.0.0", CommitHash: "aaa1111", Subject: "first", RelativeAge: "1 day ago"},
		},
		Tags: []string{"v2.0.0", "v1.0.0"},
	}
	m := testKeyHandlingModel(nil, status)
	m.activeSection = sectionTags
	m.sectionCursor[sectionTags] = 1

	gotModel, cmd := m.handleBrowseSectionKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected tag jump to be synchronous, got %v", cmd)
	}
	if got.activeSection != sectionGraph {
		t.Fatalf("expected graph section after enter, got %v", got.activeSection)
	}
	rows := graphRows(status)
	wantRow := findGraphRowByHash(rows, "aaa1111")
	if got.sectionCursor[sectionGraph] != wantRow {
		t.Fatalf("expected graph cursor %d, got %d", wantRow, got.sectionCursor[sectionGraph])
	}
}

func TestTagShortcutOpensPopupFromGraph(t *testing.T) {
	status := git.Status{
		Root: "repo",
		GraphCommits: []git.GraphCommit{
			{Hash: "aaa1111", Subject: "first"},
		},
	}
	m := testKeyHandlingModel(nil, status)
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = 0

	gotModel, cmd := m.handleBrowseGraphKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected tag popup shortcut to be synchronous, got %v", cmd)
	}
	if !got.tagPopupOpen {
		t.Fatal("expected tag popup to open")
	}
	if got.tagPopupTarget != "aaa1111" {
		t.Fatalf("expected tag target to use graph focus hash, got %q", got.tagPopupTarget)
	}
}

func TestTagPopupValidationAndEditing(t *testing.T) {
	m := model{status: state.New().WithBrowse(), tagPopupOpen: true, tagPopupTarget: "aaa1111"}

	gotModel, cmd := m.handleTagPopupKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected empty tag validation to stay synchronous, got %v", cmd)
	}
	if got.tagPopupError == "" {
		t.Fatal("expected validation error for empty tag name")
	}
	if !got.tagPopupOpen {
		t.Fatal("expected popup to remain open for local validation errors")
	}

	got.tagPopupDraft = "v1"
	gotModel, cmd = got.handleTagPopupKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected typing to stay synchronous, got %v", cmd)
	}
	if got.tagPopupDraft != "v12" {
		t.Fatalf("expected draft to append rune, got %q", got.tagPopupDraft)
	}
	gotModel, cmd = got.handleTagPopupKey(tea.KeyMsg{Type: tea.KeyBackspace})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected backspace to stay synchronous, got %v", cmd)
	}
	if got.tagPopupDraft != "v1" {
		t.Fatalf("expected backspace to remove last rune, got %q", got.tagPopupDraft)
	}
	gotModel, cmd = got.handleTagPopupKey(tea.KeyMsg{Type: tea.KeyDelete})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected delete to stay synchronous, got %v", cmd)
	}
	if got.tagPopupDraft != "v" {
		t.Fatalf("expected delete to remove last rune, got %q", got.tagPopupDraft)
	}
}

func TestCreateTagFlowRefreshesAndFocusesSameCommit(t *testing.T) {
	fixture := newCommandRepo(t)
	status, err := fixture.repo.Status(context.Background(), 20)
	if err != nil {
		t.Fatalf("initial status failed: %v", err)
	}

	m := testKeyHandlingModel(fixture.repo, status)
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = 0
	m.tagPopupOpen = true
	m.tagPopupTarget = fixture.initialHash
	m.tagPopupDraft = "v1.0.0"

	gotModel, cmd := m.handleTagPopupKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected tag creation command")
	}
	if got.tagPopupOpen {
		t.Fatal("expected popup to close before execution")
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected loading state during tag creation, got %s", got.status.Mode)
	}

	msg := cmd()
	created, ok := msg.(tagCreatedMsg)
	if !ok {
		t.Fatalf("expected tagCreatedMsg, got %T", msg)
	}
	if created.Err != nil {
		t.Fatalf("tag creation command failed: %v", created.Err)
	}
	if len(created.Status.TagEntries) != 1 {
		t.Fatalf("expected one created tag entry, got %+v", created.Status.TagEntries)
	}
	if created.Status.TagEntries[0].OriginKnown {
		t.Fatalf("expected newly created tag to stay local-only, got %+v", created.Status.TagEntries[0])
	}
	if created.Status.TagProvenanceLoaded {
		t.Fatalf("expected provenance to remain unknown before sync, got %+v", created.Status)
	}

	nextModel, nextCmd := got.Update(created)
	next := nextModel.(model)
	if nextCmd == nil {
		t.Fatal("expected completion toast command")
	}
	if next.status.Mode != state.ModeLoading {
		t.Fatalf("expected loading toast after tag creation, got %s", next.status.Mode)
	}

	done := nextCmd()
	if _, ok := done.(tagToastDoneMsg); !ok {
		t.Fatalf("expected tagToastDoneMsg, got %T", done)
	}
	finalModel, finalCmd := next.Update(done)
	final := finalModel.(model)
	if finalCmd != nil {
		t.Fatalf("expected no further command, got %v", finalCmd)
	}
	if final.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode after toast, got %s", final.status.Mode)
	}
	if final.activeSection != sectionGraph {
		t.Fatalf("expected graph section to stay focused, got %v", final.activeSection)
	}
	if final.sectionCursor[sectionGraph] != 0 {
		t.Fatalf("expected graph cursor to stay on created tag target, got %d", final.sectionCursor[sectionGraph])
	}

	refreshed, err := fixture.repo.Status(context.Background(), 20)
	if err != nil {
		t.Fatalf("refreshed status failed: %v", err)
	}
	refreshed = attachTagEntries(fixture.repo, refreshed)
	if len(refreshed.TagEntries) != 1 {
		t.Fatalf("expected one created tag after manual load, got %+v", refreshed.TagEntries)
	}
	if refreshed.TagEntries[0].Name != "v1.0.0" || refreshed.TagEntries[0].CommitHash != fixture.initialHash {
		t.Fatalf("unexpected refreshed tag entry: %+v", refreshed.TagEntries[0])
	}
}

func TestCreateTagDuplicateFails(t *testing.T) {
	fixture := newCommandRepo(t)
	status, err := fixture.repo.Status(context.Background(), 20)
	if err != nil {
		t.Fatalf("initial status failed: %v", err)
	}
	if err := fixture.repo.CreateTag(context.Background(), "v1.0.0", fixture.initialHash); err != nil {
		t.Fatalf("initial CreateTag failed: %v", err)
	}

	m := testKeyHandlingModel(fixture.repo, status)
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = 0
	m.tagPopupOpen = true
	m.tagPopupTarget = fixture.initialHash
	m.tagPopupDraft = "v1.0.0"

	gotModel, cmd := m.handleTagPopupKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected duplicate tag command")
	}
	msg := cmd()
	created := msg.(tagCreatedMsg)
	if created.Err == nil {
		t.Fatal("expected duplicate tag creation to fail")
	}
	nextModel, nextCmd := got.Update(created)
	next := nextModel.(model)
	if nextCmd != nil {
		t.Fatalf("expected no follow-up command on duplicate error, got %v", nextCmd)
	}
	if next.status.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode on duplicate tag, got %s", next.status.Mode)
	}
	if next.status.Message != "Tag already exists." {
		t.Fatalf("expected duplicate message, got %q", next.status.Message)
	}
}
