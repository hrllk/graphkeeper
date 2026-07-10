package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

func testKeyHandlingModel(repo *git.Repo, status git.Status) model {
	return model{
		repo:          repo,
		status:        state.New().WithBrowse(),
		repoStatus:    status,
		activeSection: sectionGraph,
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
		handshakeCommits: make(map[string]bool),
		stashByBase:      make(map[string][]git.StashEntry),
	}
}

func TestBranchOpenEscCancelsDraft(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{Root: fixture.root})
	m.branchOpen = true
	m.branchDraft = "feature"
	m.status = loadingToast("Enter a branch name.")

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no command on branch cancel, got %v", cmd)
	}
	if got.branchOpen {
		t.Fatal("expected branch modal to close")
	}
	if got.branchDraft != "" {
		t.Fatalf("expected branch draft to be cleared, got %q", got.branchDraft)
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode after cancel, got %s", got.status.Mode)
	}
}

func TestCreateBranchShortcutOpensInputInGraphSection(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main"},
		GraphCommits:  []git.GraphCommit{{Hash: fixture.initialHash}},
	})

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected create branch shortcut to stay synchronous, got %v", cmd)
	}
	if !got.branchOpen {
		t.Fatal("expected branch name input to open directly")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Enter a branch name." {
		t.Fatalf("expected branch prompt loading state, got %+v", got.status)
	}
	if got.branchBase != fixture.initialHash {
		t.Fatalf("expected branch base to be captured from graph focus, got %q", got.branchBase)
	}
}

func TestCreateBranchShortcutOpensInputInLocalSection(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main"},
	})
	m.activeSection = sectionCurrent
	m.sectionCursor[sectionCurrent] = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected create branch shortcut to stay synchronous, got %v", cmd)
	}
	if !got.branchOpen {
		t.Fatal("expected branch name input to open directly")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Enter a branch name." {
		t.Fatalf("expected branch prompt loading state, got %+v", got.status)
	}
	if got.branchBase != "main" {
		t.Fatalf("expected local branch base to be captured, got %q", got.branchBase)
	}
}

func TestCreateBranchShortcutBlockedWhenDirty(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		WorktreeDirty: true,
		Branch:        "main",
		Head:          fixture.initialHash,
	})
	m.activeSection = sectionGraph

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected dirty branch creation to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode, got %s", got.status.Mode)
	}
	if got.status.Block != state.BlockDirtyTree {
		t.Fatalf("expected dirty tree block, got %s", got.status.Block)
	}
}

func TestCreateBranchShortcutBlockedWhenMergeInProgress(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:            fixture.root,
		MergeInProgress: true,
		Branch:          "main",
		Head:            fixture.initialHash,
		LocalBranches:   []string{"main"},
	})
	m.activeSection = sectionCurrent

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected merge-in-progress branch creation to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode, got %s", got.status.Mode)
	}
	if got.status.Message != "Merge/rebase already in progress." {
		t.Fatalf("expected merge/rebase block message, got %q", got.status.Message)
	}
}

func TestCherryPickHotkeyOpensPopupAndReordersQueue(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root: fixture.root,
		Head: "head123",
		GraphCommits: []git.GraphCommit{
			{Hash: "pick123", Parents: []string{"root"}, Author: "Ada Lovelace", Subject: "pick me", RelativeAge: "2 days ago"},
			{Hash: "merge123", Parents: []string{"pick123", "other"}, Author: "Grace Hopper", Subject: "merge me", RelativeAge: "1 day ago"},
			{Hash: "other123", Parents: []string{"pick123"}, Author: "Edsger Dijkstra", Subject: "pick too", RelativeAge: "just now"},
			{Hash: "head123", Parents: []string{"other123"}, Author: "Test User", Subject: "head", RelativeAge: "now"},
		},
	})

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected cherry-pick hotkey to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeCherryPickPick {
		t.Fatalf("expected cherry-pick popup mode, got %s", got.status.Mode)
	}
	if len(got.status.Targets) != 2 {
		t.Fatalf("expected merge commits and HEAD to be excluded, got %#v", got.status.Targets)
	}
	originalGraphCursor := got.sectionCursor[sectionGraph]
	originalLaneCursor := got.graphLaneCursor

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeySpace})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected cherry-pick toggle to stay synchronous, got %v", cmd)
	}
	if len(got.status.SelectedQueue) != 1 || got.status.SelectedQueue[0] != "pick123" {
		t.Fatalf("expected first commit to be queued, got %#v", got.status.SelectedQueue)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected move to stay synchronous, got %v", cmd)
	}
	if got.sectionCursor[sectionGraph] != originalGraphCursor || got.graphLaneCursor != originalLaneCursor {
		t.Fatalf("expected graph focus to remain unchanged, got cursor=%d lane=%d", got.sectionCursor[sectionGraph], got.graphLaneCursor)
	}
	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeySpace})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected cherry-pick toggle to stay synchronous, got %v", cmd)
	}
	if len(got.status.SelectedQueue) != 2 || got.status.SelectedQueue[1] != "other123" {
		t.Fatalf("expected second commit to be appended, got %#v", got.status.SelectedQueue)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected move to stay synchronous, got %v", cmd)
	}
	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeySpace})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected toggle to stay synchronous, got %v", cmd)
	}
	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeySpace})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected recheck to stay synchronous, got %v", cmd)
	}
	if len(got.status.SelectedQueue) != 2 || got.status.SelectedQueue[0] != "other123" || got.status.SelectedQueue[1] != "pick123" {
		t.Fatalf("expected queue reorder after uncheck/recheck, got %#v", got.status.SelectedQueue)
	}
}

func TestBlockedAlertEnterDismissesToBrowse(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		WorktreeDirty: true,
	})
	m.status = state.New().WithBlocked(state.BlockUnknown, "Merge unavailable.", "Select a local branch.")

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected blocked alert dismiss to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode after dismiss, got %s", got.status.Mode)
	}
	if got.status.WorktreeState != state.WorktreeStateDirty {
		t.Fatalf("expected worktree state to be preserved, got %s", got.status.WorktreeState)
	}
}

func TestBlockedAlertEscDismissesToBrowse(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:   fixture.root,
		Branch: "main",
		Head:   fixture.initialHash,
	})
	m.status = state.New().WithBlocked(state.BlockUnknown, "Rebase unavailable.", "Select a local branch.")

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected blocked alert escape to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode after dismiss, got %s", got.status.Mode)
	}
	if got.status.WorktreeState != state.WorktreeStateClean {
		t.Fatalf("expected clean worktree state after dismiss, got %s", got.status.WorktreeState)
	}
}

func TestStashPopupOpensFromGlobalHotkey(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root: fixture.root,
	})
	m.stashEntries = []git.StashEntry{
		{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "abc1234", Subject: "latest change"},
	}

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected stash popup hotkey to stay synchronous, got %v", cmd)
	}
	if !got.stashPopupOpen {
		t.Fatal("expected stash popup to open from global hotkey")
	}
	if got.stashPopupCursor != 0 {
		t.Fatalf("expected stash popup cursor to start at first entry, got %d", got.stashPopupCursor)
	}
}

func TestStashPopupEnterJumpsToBaseCommit(t *testing.T) {
	fixture := newCommandRepo(t)
	status := git.Status{
		Root: fixture.root,
		GraphCommits: []git.GraphCommit{
			{Hash: "base1234", Subject: "base"},
			{Hash: "head1234", Parents: []string{"base1234"}, Subject: "head"},
		},
	}
	m := testKeyHandlingModel(fixture.repo, status)
	m.stashEntries = []git.StashEntry{
		{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "base1234", Subject: "latest change"},
	}
	m.stashPopupOpen = true

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected stash popup enter to stay synchronous, got %v", cmd)
	}
	if got.stashPopupOpen {
		t.Fatal("expected stash popup to close on enter")
	}
	if got.activeSection != sectionGraph {
		t.Fatalf("expected enter to jump to graph section, got %v", got.activeSection)
	}
	wantRow := findGraphRowByHash(graph.Rows(status), "base1234")
	if got.sectionCursor[sectionGraph] != wantRow {
		t.Fatalf("expected graph cursor to jump to stash base row %d, got %d", wantRow, got.sectionCursor[sectionGraph])
	}
}

func TestFetchTagsKeyTriggersTagRefresh(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root: fixture.root,
	})

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'F'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected F to trigger background tag refresh")
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected tag fetch to enter loading mode, got %s", got.status.Mode)
	}
	if got.status.Message != "Fetching tags..." {
		t.Fatalf("expected fetch tags message, got %q", got.status.Message)
	}
}

func TestStashPopupEscapeClosesAndKeepsCursor(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root: fixture.root,
	})
	m.stashEntries = []git.StashEntry{
		{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "abc1234", Subject: "latest change"},
		{Ref: "stash@{1}", Hash: "stashhash1", BaseHash: "abc1234", Subject: "older change"},
	}
	m.stashPopupOpen = true
	m.stashPopupCursor = 1

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected stash popup escape to stay synchronous, got %v", cmd)
	}
	if got.stashPopupOpen {
		t.Fatal("expected stash popup to close on escape")
	}
	if got.stashPopupCursor != 1 {
		t.Fatalf("expected stash popup cursor to stay on selected entry, got %d", got.stashPopupCursor)
	}
}

func TestStashPopupArrowKeysMoveSelection(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root: fixture.root,
	})
	m.stashEntries = []git.StashEntry{
		{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: "abc1234", Subject: "latest change"},
		{Ref: "stash@{1}", Hash: "stashhash1", BaseHash: "def5678", Subject: "older change"},
	}
	m.stashPopupOpen = true
	m.stashPopupCursor = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected stash popup move to stay synchronous, got %v", cmd)
	}
	if got.stashPopupCursor != 1 {
		t.Fatalf("expected stash popup cursor to move down, got %d", got.stashPopupCursor)
	}
	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected stash popup move to stay synchronous, got %v", cmd)
	}
	if got.stashPopupCursor != 0 {
		t.Fatalf("expected stash popup cursor to move back up, got %d", got.stashPopupCursor)
	}
}

func TestGraphStashPopHotkeyOpensConfirmForSingleEntry(t *testing.T) {
	fixture := newCommandRepo(t)
	headHash := fixture.initialHash
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root: fixture.root,
		Head: headHash,
		GraphCommits: []git.GraphCommit{
			{Hash: headHash, Decorations: []string{"HEAD -> main", "main"}},
		},
	})
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = 0
	m.stashByBase[headHash] = []git.StashEntry{
		{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: headHash, Subject: "latest change"},
	}

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected graph stash pop hotkey to stay synchronous, got %v", cmd)
	}
	if !got.graphStashPopOpen {
		t.Fatal("expected graph stash pop popup to open")
	}
	if got.graphStashPopMode != graphStashPopModeConfirm {
		t.Fatalf("expected single stash to open confirm, got %v", got.graphStashPopMode)
	}
	if len(got.graphStashPopEntries) != 1 {
		t.Fatalf("expected one stash entry, got %d", len(got.graphStashPopEntries))
	}
}

func TestGraphStashPopHotkeyUsesPickerForMultipleEntries(t *testing.T) {
	fixture := newCommandRepo(t)
	headHash := fixture.initialHash
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root: fixture.root,
		Head: headHash,
		GraphCommits: []git.GraphCommit{
			{Hash: headHash, Decorations: []string{"HEAD -> main", "main"}},
		},
	})
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = 0
	m.stashByBase[headHash] = []git.StashEntry{
		{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: headHash, Subject: "latest change"},
		{Ref: "stash@{1}", Hash: "stashhash1", BaseHash: headHash, Subject: "older change"},
	}

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected graph stash pop hotkey to stay synchronous, got %v", cmd)
	}
	if !got.graphStashPopOpen {
		t.Fatal("expected graph stash pop popup to open")
	}
	if got.graphStashPopMode != graphStashPopModePicker {
		t.Fatalf("expected multiple stashes to open picker, got %v", got.graphStashPopMode)
	}
	if got.graphStashPopCursor != 0 {
		t.Fatalf("expected picker cursor to start at first entry, got %d", got.graphStashPopCursor)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected picker navigation to stay synchronous, got %v", cmd)
	}
	if got.graphStashPopCursor != 1 {
		t.Fatalf("expected picker cursor to move down, got %d", got.graphStashPopCursor)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected picker enter to stay synchronous, got %v", cmd)
	}
	if got.graphStashPopMode != graphStashPopModeConfirm {
		t.Fatalf("expected picker enter to switch to confirm, got %v", got.graphStashPopMode)
	}
	if !got.graphStashPopOpen {
		t.Fatal("expected popup to remain open for confirm")
	}
}

func TestGraphStashPopConfirmExecutesOnEnter(t *testing.T) {
	fixture := newCommandRepo(t)
	headHash := fixture.initialHash
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root: fixture.root,
		Head: headHash,
		GraphCommits: []git.GraphCommit{
			{Hash: headHash, Decorations: []string{"HEAD -> main", "main"}},
		},
	})
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = 0
	m.stashByBase[headHash] = []git.StashEntry{
		{Ref: "stash@{0}", Hash: "stashhash0", BaseHash: headHash, Subject: "latest change"},
		{Ref: "stash@{1}", Hash: "stashhash1", BaseHash: headHash, Subject: "older change"},
	}
	m.graphStashPopOpen = true
	m.graphStashPopMode = graphStashPopModeConfirm
	m.graphStashPopCursor = 1
	m.graphStashPopEntries = m.stashesForCommit(headHash)

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected confirm enter to execute stash pop")
	}
	if got.graphStashPopOpen {
		t.Fatal("expected graph stash popup to close on execute")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Popping stash..." {
		t.Fatalf("expected pop loading state, got %+v", got.status)
	}
}

func TestStashShortcutOpensMessagePopupForDirtyLocalSection(t *testing.T) {
	fixture := newCommandRepo(t)
	writeRepoFile(t, fixture.root, "stash.txt", "stash\n")
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main"},
		WorktreeDirty: true,
	})
	m.activeSection = sectionCurrent

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected stash shortcut to stay synchronous, got %v", cmd)
	}
	if !got.stashMessageOpen {
		t.Fatal("expected stash message popup to open")
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode while message popup is open, got %s", got.status.Mode)
	}
	if got.stashMessageDraft != "" {
		t.Fatalf("expected empty draft, got %q", got.stashMessageDraft)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("stash message")})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected message typing to stay synchronous, got %v", cmd)
	}
	if got.stashMessageDraft != "stash message" {
		t.Fatalf("expected typed message to accumulate, got %q", got.stashMessageDraft)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected stash message submission to execute")
	}
	if got.stashMessageOpen {
		t.Fatal("expected stash message popup to close on submit")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Stashing changes..." {
		t.Fatalf("expected stash loading state, got %+v", got.status)
	}
	msg := cmd()
	executed, ok := msg.(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", msg)
	}
	if executed.action != state.ActionStash {
		t.Fatalf("expected stash executed action, got %s", executed.action)
	}
	if executed.err != nil {
		t.Fatalf("expected stash execution to succeed, got %v", executed.err)
	}
}

func TestCleanShortcutOpensConfirmForDirtyLocalSection(t *testing.T) {
	fixture := newCommandRepo(t)
	writeRepoFile(t, fixture.root, "dirty.txt", "dirty\n")
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main"},
		WorktreeDirty: true,
	})
	m.activeSection = sectionCurrent

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected clean shortcut to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionCleanWorkingTree {
		t.Fatalf("expected clean-working-tree action, got %s", got.status.Action)
	}
	if got.status.Title != "Clean working tree?" {
		t.Fatalf("expected clean confirm title, got %q", got.status.Title)
	}
}

func TestDeleteBranchShortcutOpensConfirmForLocalBranch(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main", "feature"},
		GraphCommits:  []git.GraphCommit{{Hash: fixture.initialHash}},
	})
	m.activeSection = sectionCurrent
	m.sectionCursor[sectionCurrent] = 1

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected delete shortcut to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionDeleteBranch {
		t.Fatalf("expected delete-branch action, got %s", got.status.Action)
	}
	if got.status.Title != "Delete branch?" {
		t.Fatalf("expected delete confirm title, got %q", got.status.Title)
	}
	if got.status.Selected != "feature" {
		t.Fatalf("expected local branch target to be stored, got %q", got.status.Selected)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected delete confirm acceptance to execute")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Deleting branch..." {
		t.Fatalf("expected delete loading state, got %+v", got.status)
	}
}

func TestStashShortcutRefreshesStashState(t *testing.T) {
	fixture := newCommandRepo(t)
	writeRepoFile(t, fixture.root, "dirty.txt", "dirty\n")
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main"},
		WorktreeDirty: true,
	})
	m.activeSection = sectionCurrent

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected stash shortcut to stay synchronous, got %v", cmd)
	}
	if !got.stashMessageOpen {
		t.Fatal("expected stash message popup to open")
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("local cleanup")})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected message typing to stay synchronous, got %v", cmd)
	}
	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected stash message submission to execute")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Stashing changes..." {
		t.Fatalf("expected stash loading state, got %+v", got.status)
	}

	msg := cmd()
	executed, ok := msg.(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", msg)
	}
	if executed.action != state.ActionStash {
		t.Fatalf("expected stash executed action, got %s", executed.action)
	}
	gotModel, cmd = got.Update(executed)
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected stash success to refresh stash state")
	}
	msg = cmd()
	loaded, ok := msg.(stashLoadedMsg)
	if !ok {
		t.Fatalf("expected stashLoadedMsg, got %T", msg)
	}
	if loaded.err != nil {
		t.Fatalf("expected stash refresh to succeed, got %v", loaded.err)
	}
	gotModel, cmd = got.Update(loaded)
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no follow-up command after stash refresh, got %v", cmd)
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode after stash, got %s", got.status.Mode)
	}
	if len(got.stashEntries) == 0 {
		t.Fatal("expected stash list to refresh after stash")
	}
}

func TestCleanShortcutRemovesTrackedAndUntrackedFiles(t *testing.T) {
	fixture := newCommandRepo(t)
	writeRepoFile(t, fixture.root, "file.txt", "changed\n")
	writeRepoFile(t, fixture.root, "untracked.txt", "temp\n")
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main"},
		WorktreeDirty: true,
	})
	m.activeSection = sectionCurrent

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected clean shortcut to stay synchronous, got %v", cmd)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected clean acceptance to execute")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Cleaning working tree..." {
		t.Fatalf("expected clean loading state, got %+v", got.status)
	}

	msg := cmd()
	executed, ok := msg.(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", msg)
	}
	if executed.action != state.ActionCleanWorkingTree {
		t.Fatalf("expected clean executed action, got %s", executed.action)
	}
	gotModel, cmd = got.Update(executed)
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no follow-up command after clean, got %v", cmd)
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode after clean, got %s", got.status.Mode)
	}
	if got.status.Message != "Working tree cleaned." {
		t.Fatalf("expected clean success message, got %q", got.status.Message)
	}
	if _, err := os.Stat(filepath.Join(fixture.root, "untracked.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected untracked file to be removed, stat err=%v", err)
	}
}

func TestDeleteBranchShortcutBlocksCurrentBranch(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main", "feature"},
		GraphCommits:  []git.GraphCommit{{Hash: fixture.initialHash}},
	})
	m.activeSection = sectionCurrent
	m.sectionCursor[sectionCurrent] = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected current-branch delete to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode, got %s", got.status.Mode)
	}
}

func TestBranchOpenRejectsDuplicateName(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main"},
	})
	m.branchOpen = true
	m.branchBase = fixture.initialHash
	m.branchDraft = "main"
	m.status = loadingToast("Enter a branch name.")

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected duplicate branch name to stay synchronous, got %v", cmd)
	}
	if !got.branchOpen {
		t.Fatal("expected branch modal to stay open on duplicate")
	}
	if got.branchError != "Branch name already exists." {
		t.Fatalf("expected branch error to be stored, got %q", got.branchError)
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Enter a branch name." {
		t.Fatalf("expected branch prompt to stay visible, got %+v", got.status)
	}
}

func TestBranchOpenSuccessShowsCreatedToast(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main"},
	})
	m.branchOpen = true
	m.branchBase = fixture.initialHash
	m.branchDraft = "feature/new-flow"
	m.status = loadingToast("Enter a branch name.")

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected branch creation command to be issued")
	}
	if got.branchOpen {
		t.Fatal("expected branch modal to close on success path")
	}
	msg := cmd()
	created, ok := msg.(createdBranchMsg)
	if !ok {
		t.Fatalf("expected createdBranchMsg, got %T", msg)
	}
	if created.err != nil {
		t.Fatalf("expected branch creation to succeed, got %v", created.err)
	}

	gotModel2, cmd2 := got.Update(created)
	got2 := gotModel2.(model)
	if cmd2 == nil {
		t.Fatal("expected branch success toast dismissal command")
	}
	if got2.activeSection != sectionGraph {
		t.Fatalf("expected branch create to focus graph section, got %v", got2.activeSection)
	}
	if got2.sectionCursor[sectionGraph] != 0 {
		t.Fatalf("expected branch create to focus head row, got %d", got2.sectionCursor[sectionGraph])
	}
	if got2.status.Mode != state.ModeLoading || got2.status.Message != "Branch created." {
		t.Fatalf("expected success toast, got %+v", got2.status)
	}
	done := cmd2()
	doneMsg, ok := done.(branchToastDoneMsg)
	if !ok {
		t.Fatalf("expected branchToastDoneMsg, got %T", done)
	}
	gotModel3, cmd3 := got2.Update(doneMsg)
	got3 := gotModel3.(model)
	if cmd3 != nil {
		t.Fatalf("expected no command after toast dismiss, got %v", cmd3)
	}
	if got3.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode after toast dismiss, got %s", got3.status.Mode)
	}
}

func TestDeleteBranchShortcutOpensConfirmForOriginBranch(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:           fixture.root,
		Branch:         "main",
		Head:           fixture.initialHash,
		LocalBranches:  []string{"main"},
		RemoteBranches: []string{"origin/feature"},
		Remote:         "origin",
		GraphCommits:   []git.GraphCommit{{Hash: fixture.initialHash}},
	})
	m.activeSection = sectionRemote
	m.sectionCursor[sectionRemote] = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected remote delete shortcut to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionDeleteBranch {
		t.Fatalf("expected delete-branch action, got %s", got.status.Action)
	}
	if got.status.Title != "Delete branch?" {
		t.Fatalf("expected origin delete confirm title, got %q", got.status.Title)
	}
	if got.status.Selected != "feature" {
		t.Fatalf("expected origin branch name to be stored, got %q", got.status.Selected)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected remote delete confirm acceptance to execute")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Deleting origin branch..." {
		t.Fatalf("expected remote delete loading state, got %+v", got.status)
	}
}

func TestDeleteTagShortcutOpensConfirm(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root: "repo",
		TagEntries: []git.TagEntry{
			{Name: "v1.0.0", CommitHash: fixture.initialHash, Subject: "initial", RelativeAge: "2 days ago"},
		},
		Tags: []string{"v1.0.0"},
		GraphCommits: []git.GraphCommit{
			{Hash: fixture.initialHash, Subject: "initial"},
		},
	})
	m.activeSection = sectionTags
	m.sectionCursor[sectionTags] = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected tag delete shortcut to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionDeleteTag {
		t.Fatalf("expected delete-tag action, got %s", got.status.Action)
	}
	if got.status.Title != "Delete tag?" {
		t.Fatalf("expected tag delete confirm title, got %q", got.status.Title)
	}
	if got.status.Selected != "v1.0.0" {
		t.Fatalf("expected tag target to be stored, got %q", got.status.Selected)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected tag delete confirm acceptance to execute")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Deleting tag..." {
		t.Fatalf("expected tag delete loading state, got %+v", got.status)
	}
}

func TestDeleteRemoteTagShortcutOpensConfirm(t *testing.T) {
	fixture := newCommandRepo(t)
	runGit(t, fixture.root, "tag", "v1.0.0", fixture.initialHash)
	runGit(t, fixture.root, "push", "origin", "v1.0.0")

	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:             fixture.root,
		TagEntries:       []git.TagEntry{{Name: "v1.0.0", CommitHash: fixture.initialHash, Subject: "initial", RelativeAge: "2 days ago", OriginKnown: true, OnOrigin: true}},
		TagEntriesLoaded: true,
		TagSyncSummary:   string(tagSyncSynced),
	})
	m.activeSection = sectionTags
	m.sectionCursor[sectionTags] = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected remote tag delete shortcut to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionDeleteRemoteTag {
		t.Fatalf("expected delete-remote-tag action, got %s", got.status.Action)
	}
	if got.status.Title != "Delete remote tag?" {
		t.Fatalf("expected remote tag delete confirm title, got %q", got.status.Title)
	}
	if got.status.Selected != "v1.0.0" {
		t.Fatalf("expected remote tag target to be stored, got %q", got.status.Selected)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected remote tag delete confirm acceptance to execute")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Deleting remote tag..." {
		t.Fatalf("expected remote tag delete loading state, got %+v", got.status)
	}
}

func TestDeleteBranchShortcutOpensConfirmFromGraph(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main", "feature"},
		GraphCommits: []git.GraphCommit{
			{Hash: fixture.initialHash, Graph: "*", Decorations: []string{"main"}},
			{Hash: "featurehash", Graph: "|", Decorations: []string{"feature"}},
		},
	})
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = 1
	m.graphLaneCursor = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected graph delete shortcut to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionDeleteBranch {
		t.Fatalf("expected delete-branch action, got %s", got.status.Action)
	}
	if got.status.Title != "Delete branch?" {
		t.Fatalf("expected delete confirm title, got %q", got.status.Title)
	}
	if got.status.Selected != "feature" {
		t.Fatalf("expected graph branch target to be stored, got %q", got.status.Selected)
	}
}

func TestGraphMergeShortcutChecksDivergenceBeforeConfirm(t *testing.T) {
	fixture := newCommandRepo(t)
	runGit(t, fixture.root, "checkout", "-b", "feature")
	featureHash := makeLocalCommit(t, fixture.root, "feature.txt", "feature\n", "feature commit")
	runGit(t, fixture.root, "checkout", "main")
	mainHash := makeLocalCommit(t, fixture.root, "main.txt", "main\n", "main commit")

	rs := git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          mainHash,
		LocalBranches: []string{"main", "feature"},
		GraphCommits: []git.GraphCommit{
			{Hash: mainHash, Parents: []string{fixture.initialHash}, Decorations: []string{"HEAD -> main", "main"}},
			{Hash: featureHash, Parents: []string{fixture.initialHash}, Decorations: []string{"feature"}},
			{Hash: fixture.initialHash, Parents: []string{}},
		},
	}
	rows := graphRows(rs)
	featureCursor := findGraphRowByHash(rows, featureHash)
	if featureCursor < 0 {
		t.Fatalf("expected feature hash %s in graph rows", featureHash)
	}

	m := testKeyHandlingModel(fixture.repo, rs)
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = featureCursor
	m.graphLaneCursor = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected merge shortcut to start graph target analysis")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Analyzing graph target..." {
		t.Fatalf("expected graph analysis loading state, got %+v", got.status)
	}

	msg := cmdResult(t, cmd)
	check, ok := msg.(graphActionCheckMsg)
	if !ok {
		t.Fatalf("expected graphActionCheckMsg, got %T", msg)
	}
	if check.base == "" {
		t.Fatalf("expected merge base to be populated")
	}
	if check.currentOnly == 0 || check.targetOnly == 0 {
		t.Fatalf("expected diverged graph target, got currentOnly=%d targetOnly=%d", check.currentOnly, check.targetOnly)
	}

	gotModel, cmd = got.Update(check)
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no follow-up command after graph check, got %v", cmd)
	}
	if got.status.Mode != state.ModeReview {
		t.Fatalf("expected review mode after diverged graph target, got %s", got.status.Mode)
	}
	if got.status.Message != "Branch has diverged" {
		t.Fatalf("expected diverged review title, got %q", got.status.Message)
	}
	if got.status.Action != state.ActionMerge {
		t.Fatalf("expected merge action, got %s", got.status.Action)
	}
	if got.status.Selected != featureHash {
		t.Fatalf("expected selected target %q, got %q", featureHash, got.status.Selected)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no command while opening final confirm, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected final confirm mode after review accept, got %s", got.status.Mode)
	}
}

func TestGraphMergeShortcutUsesFastForwardConfirm(t *testing.T) {
	fixture := newCommandRepo(t)
	runGit(t, fixture.root, "checkout", "-b", "feature")
	featureHash := makeLocalCommit(t, fixture.root, "feature.txt", "feature\n", "feature commit")
	runGit(t, fixture.root, "checkout", "main")

	rs := git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main", "feature"},
		GraphCommits: []git.GraphCommit{
			{Hash: featureHash, Parents: []string{fixture.initialHash}, Decorations: []string{"feature"}},
			{Hash: fixture.initialHash, Parents: []string{}, Decorations: []string{"HEAD -> main", "main"}},
		},
	}
	rows := graphRows(rs)
	featureCursor := findGraphRowByHash(rows, featureHash)
	if featureCursor < 0 {
		t.Fatalf("expected feature hash %s in graph rows", featureHash)
	}

	m := testKeyHandlingModel(fixture.repo, rs)
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = featureCursor
	m.graphLaneCursor = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected merge shortcut to start graph target analysis")
	}

	msg := cmdResult(t, cmd)
	check, ok := msg.(graphActionCheckMsg)
	if !ok {
		t.Fatalf("expected graphActionCheckMsg, got %T", msg)
	}
	if check.currentOnly != 0 || check.targetOnly == 0 {
		t.Fatalf("expected fast-forward graph target, got currentOnly=%d targetOnly=%d", check.currentOnly, check.targetOnly)
	}

	gotModel, cmd = got.Update(check)
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no follow-up command after graph check, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode for fast-forward graph target, got %s", got.status.Mode)
	}
	if got.status.Message != "Fast-forward available." {
		t.Fatalf("expected fast-forward title, got %q", got.status.Message)
	}
	if got.status.Detail != "HEAD can move to "+featureHash+"." {
		t.Fatalf("expected concise fast-forward detail, got %q", got.status.Detail)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected enter to execute the fast-forward action")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Merging..." {
		t.Fatalf("expected merge loading state after enter, got %+v", got.status)
	}
}

func TestGraphRebaseShortcutBlocksAncestorTarget(t *testing.T) {
	fixture := newCommandRepo(t)
	runGit(t, fixture.root, "checkout", "-b", "feature")
	featureHash := makeLocalCommit(t, fixture.root, "feature.txt", "feature\n", "feature commit")
	runGit(t, fixture.root, "checkout", "main")
	mainHash := makeLocalCommit(t, fixture.root, "main.txt", "main\n", "main commit")

	rs := git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          mainHash,
		LocalBranches: []string{"main", "feature"},
		GraphCommits: []git.GraphCommit{
			{Hash: mainHash, Parents: []string{fixture.initialHash}, Decorations: []string{"HEAD -> main", "main"}},
			{Hash: featureHash, Parents: []string{fixture.initialHash}, Decorations: []string{"feature"}},
			{Hash: fixture.initialHash, Parents: []string{}},
		},
	}
	rows := graphRows(rs)
	ancestorCursor := findGraphRowByHash(rows, fixture.initialHash)
	if ancestorCursor < 0 {
		t.Fatalf("expected initial hash %s in graph rows", fixture.initialHash)
	}

	m := testKeyHandlingModel(fixture.repo, rs)
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = ancestorCursor
	m.graphLaneCursor = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected rebase shortcut to start graph target analysis")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Analyzing graph target..." {
		t.Fatalf("expected graph analysis loading state, got %+v", got.status)
	}

	msg := cmdResult(t, cmd)
	check, ok := msg.(graphActionCheckMsg)
	if !ok {
		t.Fatalf("expected graphActionCheckMsg, got %T", msg)
	}
	if check.base == "" {
		t.Fatalf("expected merge base to be populated")
	}
	if check.targetOnly != 0 {
		t.Fatalf("expected ancestor target to have no target-only commits, got %d", check.targetOnly)
	}

	gotModel, cmd = got.Update(check)
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no follow-up command after graph check, got %v", cmd)
	}
	if got.status.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode for ancestor target, got %s", got.status.Mode)
	}
	if got.status.Message != "Target already included." {
		t.Fatalf("expected ancestor block message, got %q", got.status.Message)
	}
}

func TestTargetPickRejectsEmptySelection(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{Root: fixture.root})
	m.status = state.New().WithTargetPick(state.ActionMerge, nil)

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no command when target is empty, got %v", cmd)
	}
	if got.status.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode, got %s", got.status.Mode)
	}
	if got.status.Block != state.BlockTargetEmpty {
		t.Fatalf("expected target-empty block, got %s", got.status.Block)
	}
}

func TestTargetPickEnterStartsPreview(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{Root: fixture.root, Branch: "main", Head: fixture.initialHash, LocalBranches: []string{"main"}})
	m.status = state.New().WithTargetPick(state.ActionReset, []state.TargetItem{{Ref: fixture.initialHash}})

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected preview command to be issued")
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected loading mode while previewing, got %s", got.status.Mode)
	}
	if got.status.Message != "Previewing..." {
		t.Fatalf("expected preview message, got %q", got.status.Message)
	}
}

func TestTargetPickEnterConfirmsCheckout(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{Root: fixture.root, Branch: "main", Head: fixture.initialHash, LocalBranches: []string{"main", "feature"}})
	m.status = state.New().WithTargetPick(state.ActionCheckout, []state.TargetItem{
		{Kind: state.TargetKindLocal, Name: "main", Ref: "main"},
		{Kind: state.TargetKindLocal, Name: "feature", Ref: "feature"},
	})

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected checkout selection to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionCheckout {
		t.Fatalf("expected checkout action, got %s", got.status.Action)
	}
	if got.status.Selected != "main" {
		t.Fatalf("expected checkout selection to remain on first target, got %q", got.status.Selected)
	}
}

func TestResetModePickerKeyHandling(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{Root: fixture.root, Branch: "main", Head: fixture.initialHash})
	m.status = state.New().WithResetModePick("Choose a reset mode.", "")
	m.status.Selected = fixture.initialHash

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected hard reset key to trigger execution")
	}
	if got.status.ResetMode != state.ResetModeHard {
		t.Fatalf("expected hard reset selection, got %s", got.status.ResetMode)
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected loading mode while executing reset, got %s", got.status.Mode)
	}
	if got.status.Message != "Hard reset..." {
		t.Fatalf("expected hard reset toast, got %q", got.status.Message)
	}
}

func TestResetModePickerIgnoresEnter(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{Root: fixture.root, Branch: "main", Head: fixture.initialHash})
	m.status = state.New().WithResetModePick("Choose a reset mode.", "")
	m.status.Selected = fixture.initialHash
	m.status.ResetMode = state.ResetModeMixed

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected enter to be ignored, got %v", cmd)
	}
	if got.status.Mode != state.ModeResetModePick {
		t.Fatalf("expected reset mode picker to stay open, got %s", got.status.Mode)
	}
	if got.status.ResetMode != state.ResetModeMixed {
		t.Fatalf("expected reset mode to stay unchanged, got %s", got.status.ResetMode)
	}
}

func TestPullShortcutAvailableInCurrentSection(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		Upstream:      "origin/main",
		Remote:        "origin",
		LocalBranches: []string{"main"},
		Tracking: map[string]git.BranchTracking{
			"main": git.BranchTracking{Behind: 1},
		},
	})
	m.activeSection = sectionCurrent

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected pull command from current section")
	}
	if got.status.Mode != state.ModeLoading {
		t.Fatalf("expected loading mode for pull, got %s", got.status.Mode)
	}
}

func TestPullShortcutBlockedWhenDirty(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		Upstream:      "origin/main",
		Remote:        "origin",
		WorktreeDirty: true,
		LocalBranches: []string{"main"},
		Tracking: map[string]git.BranchTracking{
			"main": {Behind: 1},
		},
	})
	m.activeSection = sectionCurrent

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected dirty pull shortcut to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode, got %s", got.status.Mode)
	}
	if got.status.Block != state.BlockDirtyTree {
		t.Fatalf("expected dirty tree block, got %s", got.status.Block)
	}
}

func TestPullShortcutInGraphSectionRequiresLocalPointer(t *testing.T) {
	m := testKeyHandlingModel(nil, git.Status{
		Root:       "/repo",
		Branch:     "main",
		Head:       "c1",
		Upstream:   "origin/main",
		Remote:     "origin",
		HasCommits: true,
		GraphCommits: []git.GraphCommit{
			{Hash: "c1"},
		},
	})
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = 0
	m.graphLaneCursor = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no pull command when graph pointer is not clearly local, got %v", cmd)
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected browse mode unchanged, got %s", got.status.Mode)
	}
}

func TestCheckoutShortcutOpensConfirmWhenClean(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:           fixture.root,
		Branch:         "main",
		Head:           fixture.initialHash,
		RemoteBranches: []string{"origin/main"},
		LocalBranches:  []string{"main"},
		Remote:         "origin",
		HasCommits:     true,
	})
	m.activeSection = sectionRemote

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected checkout shortcut to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionCheckout {
		t.Fatalf("expected checkout action, got %s", got.status.Action)
	}
	if got.status.Title != "Checkout branch?" {
		t.Fatalf("expected checkout confirm title, got %q", got.status.Title)
	}
	if got.status.Selected != "origin/main" {
		t.Fatalf("expected selected checkout target to be stored, got %q", got.status.Selected)
	}
}

func TestCheckoutShortcutBlockedWhenDirty(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:           fixture.root,
		Branch:         "main",
		Head:           fixture.initialHash,
		RemoteBranches: []string{"origin/main"},
		LocalBranches:  []string{"main"},
		Remote:         "origin",
		WorktreeDirty:  true,
		HasCommits:     true,
	})
	m.activeSection = sectionRemote

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected dirty checkout to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode, got %s", got.status.Mode)
	}
	if got.status.Block != state.BlockDirtyTree {
		t.Fatalf("expected dirty tree block, got %s", got.status.Block)
	}
}

func TestGraphCheckoutShortcutOpensConfirmWhenClean(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main"},
		GraphCommits: []git.GraphCommit{{
			Graph:       "*",
			Hash:        fixture.initialHash,
			Decorations: []string{"HEAD -> main"},
		}},
	})
	m.sectionCursor[sectionGraph] = 0
	m.graphLaneCursor = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected graph checkout shortcut to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionCheckout {
		t.Fatalf("expected checkout action, got %s", got.status.Action)
	}
	if got.status.Selected != "main" {
		t.Fatalf("expected graph checkout target to be stored, got %q", got.status.Selected)
	}
}

func TestGraphCheckoutShortcutBlockedWhenDirty(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main"},
		WorktreeDirty: true,
		GraphCommits: []git.GraphCommit{{
			Graph:       "*",
			Hash:        fixture.initialHash,
			Decorations: []string{"HEAD -> main"},
		}},
	})
	m.sectionCursor[sectionGraph] = 0
	m.graphLaneCursor = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected dirty graph checkout to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeBlocked {
		t.Fatalf("expected blocked mode, got %s", got.status.Mode)
	}
	if got.status.Block != state.BlockDirtyTree {
		t.Fatalf("expected dirty tree block, got %s", got.status.Block)
	}
}

func TestGraphCheckoutShortcutOpensTargetPickWhenMultipleBranches(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main", "feature"},
		GraphCommits: []git.GraphCommit{{
			Graph:       "*",
			Hash:        fixture.initialHash,
			Decorations: []string{"HEAD -> main", "feature", "origin/main"},
		}},
	})
	m.sectionCursor[sectionGraph] = 0
	m.graphLaneCursor = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected graph checkout target pick to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeTargetPick {
		t.Fatalf("expected target pick mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionCheckout {
		t.Fatalf("expected checkout action, got %s", got.status.Action)
	}
	if len(got.status.Targets) != 2 {
		t.Fatalf("expected local-only branch targets, got %d", len(got.status.Targets))
	}
	for _, target := range got.status.Targets {
		if target.Kind != state.TargetKindLocal {
			t.Fatalf("expected local target only, got %+v", target)
		}
		if strings.HasPrefix(target.Ref, "origin/") {
			t.Fatalf("expected origin refs to be hidden, got %+v", target)
		}
	}
}

func TestTagSectionPushHotkeyPushesSelectedTag(t *testing.T) {
	fixture := newCommandRepo(t)
	runGit(t, fixture.root, "tag", "v1.0.0", fixture.initialHash)

	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:             fixture.root,
		TagEntries:       []git.TagEntry{{Name: "v1.0.0", CommitHash: fixture.initialHash, Subject: "initial", RelativeAge: "1 day ago"}},
		TagEntriesLoaded: true,
		Tags:             []string{"v1.0.0"},
	})
	m.activeSection = sectionTags
	m.sectionCursor[sectionTags] = 0

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected tag push command")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Pushing tag..." {
		t.Fatalf("expected tag push loading state, got %+v", got.status)
	}

	msg := cmd()
	executed, ok := msg.(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", msg)
	}
	if executed.action != state.ActionPushTag {
		t.Fatalf("expected push-tag action, got %s", executed.action)
	}
	if executed.target != "v1.0.0" {
		t.Fatalf("expected tag target v1.0.0, got %q", executed.target)
	}
	if executed.err != nil {
		t.Fatalf("expected push-tag to succeed, got %v", executed.err)
	}

	nextModel, nextCmd := got.Update(executed)
	next := nextModel.(model)
	if nextCmd != nil {
		t.Fatalf("expected no follow-up command, got %v", nextCmd)
	}
	if next.status.Message != "Tag pushed: v1.0.0." {
		t.Fatalf("expected tag push status message, got %+v", next.status)
	}
	if !next.tagSyncAttempted {
		t.Fatal("expected tag provenance sync to be marked after push")
	}
}

func TestConfirmPullShortcutVariants(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{Root: fixture.root, Branch: "main", Head: fixture.initialHash})
	m.status = state.New().WithConfirm(state.ActionPull, "Pull?", "Detail")
	m.pullIsFastForward = false

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	got := gotModel.(model)
	if cmd == nil {
		t.Fatal("expected merge-pull command for m shortcut")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Merging pull..." {
		t.Fatalf("expected merge-pull loading state, got %+v", got.status)
	}

	m = testKeyHandlingModel(fixture.repo, git.Status{Root: fixture.root, Branch: "main", Head: fixture.initialHash})
	m.status = state.New().WithConfirm(state.ActionPull, "Pull?", "Detail")
	m.pullIsFastForward = false
	gotModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected rebase-pull command for r shortcut")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Rebasing pull..." {
		t.Fatalf("expected rebase-pull loading state, got %+v", got.status)
	}
}

func TestOutcomePreviewEscapeRoutesByAction(t *testing.T) {
	fixture := newCommandRepo(t)
	baseStatus := git.Status{
		Root:          fixture.root,
		Branch:        "main",
		Head:          fixture.initialHash,
		LocalBranches: []string{"main", "feature"},
	}

	mergeModel := testKeyHandlingModel(fixture.repo, baseStatus)
	mergeModel.status = state.New().WithOutcome(state.ActionMerge, "Preview", "Detail", true)
	gotModel, cmd := mergeModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no command on outcome escape, got %v", cmd)
	}
	if got.status.Mode != state.ModeTargetPick {
		t.Fatalf("expected merge outcome escape to return to target pick, got %s", got.status.Mode)
	}

	pullModel := testKeyHandlingModel(fixture.repo, baseStatus)
	pullModel.status = state.New().WithOutcome(state.ActionPull, "Preview", "Detail", true)
	gotModel, cmd = pullModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no command on pull outcome escape, got %v", cmd)
	}
	if got.status.Mode != state.ModeBrowse {
		t.Fatalf("expected pull outcome escape to return to browse, got %s", got.status.Mode)
	}
}

func TestBrowseNavigationKeysDoNotSpawnLazyLoadCommands(t *testing.T) {
	m := testKeyHandlingModel(nil, git.Status{
		GraphCommits: []git.GraphCommit{
			{Hash: "c2", Parents: []string{"c1"}},
			{Hash: "c1"},
		},
	})

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected down key to stay synchronous, got %v", cmd)
	}
	if got.sectionCursor[sectionGraph] != 1 {
		t.Fatalf("expected down key to move graph cursor, got %d", got.sectionCursor[sectionGraph])
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected G key to stay synchronous, got %v", cmd)
	}
	if got.sectionCursor[sectionGraph] != 1 {
		t.Fatalf("expected G key to keep cursor on last row, got %d", got.sectionCursor[sectionGraph])
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected ctrl+d to stay synchronous, got %v", cmd)
	}
	if got.sectionCursor[sectionGraph] != 1 {
		t.Fatalf("expected ctrl+d to keep cursor on last row, got %d", got.sectionCursor[sectionGraph])
	}
}

func TestDeleteBranchShortcutTargetsSelectedCurrentSectionBranch(t *testing.T) {
	fixture := newCommandRepo(t)
	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "tmp2",
		Head:          fixture.initialHash,
		LocalBranches: []string{"tmp2", "tmp1"},
	})
	m.activeSection = sectionCurrent
	m.sectionCursor[sectionCurrent] = 1

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected delete shortcut to stay synchronous, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Selected != "tmp1" {
		t.Fatalf("expected delete target tmp1, got %q", got.status.Selected)
	}
}

func TestDeleteBranchShortcutDeletesSelectedCurrentSectionBranch(t *testing.T) {
	fixture := newCommandRepo(t)
	runGit(t, fixture.root, "checkout", "-b", "tmp2")
	runGit(t, fixture.root, "checkout", "-b", "tmp1")
	runGit(t, fixture.root, "checkout", "tmp2")

	m := testKeyHandlingModel(fixture.repo, git.Status{
		Root:          fixture.root,
		Branch:        "tmp2",
		Head:          fixture.initialHash,
		LocalBranches: []string{"tmp2", "tmp1"},
		GraphCommits:  []git.GraphCommit{{Hash: fixture.initialHash}},
	})
	m.activeSection = sectionCurrent
	m.sectionCursor[sectionCurrent] = 1

	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected delete shortcut to stay synchronous, got %v", cmd)
	}
	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected delete acceptance to execute")
	}
	msg := cmd()
	executed, ok := msg.(executedMsg)
	if !ok {
		t.Fatalf("expected executedMsg, got %T", msg)
	}
	if executed.err != nil {
		t.Fatalf("expected delete execution to succeed, got %v", executed.err)
	}
	if executed.target != "tmp1" {
		t.Fatalf("expected executed delete target tmp1, got %q", executed.target)
	}
}
