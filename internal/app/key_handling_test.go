package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
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

func TestCreateBranchShortcutOpensConfirmBeforeInput(t *testing.T) {
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
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionCreateBranch {
		t.Fatalf("expected create-branch action, got %s", got.status.Action)
	}
	if got.status.Title != "Create new branch?" {
		t.Fatalf("expected confirm title, got %q", got.status.Title)
	}
	if got.status.Selected != fixture.initialHash {
		t.Fatalf("expected branch base to be stored in confirm state, got %q", got.status.Selected)
	}

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected confirm acceptance to stay synchronous, got %v", cmd)
	}
	if !got.branchOpen {
		t.Fatal("expected branch name input to open after confirm")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Enter a branch name." {
		t.Fatalf("expected branch prompt loading state, got %+v", got.status)
	}
	if got.branchBase != fixture.initialHash {
		t.Fatalf("expected branch base to be copied into input state, got %q", got.branchBase)
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
