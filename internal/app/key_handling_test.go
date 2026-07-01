package app

import (
	"os"
	"path/filepath"
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

func TestStashShortcutOpensConfirmForDirtyLocalSection(t *testing.T) {
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
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionStash {
		t.Fatalf("expected stash action, got %s", got.status.Action)
	}
	if got.status.Title != "Stash changes?" {
		t.Fatalf("expected stash confirm title, got %q", got.status.Title)
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

	gotModel, cmd = got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = gotModel.(model)
	if cmd == nil {
		t.Fatal("expected stash acceptance to execute")
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
	if check.currentOnly == 0 || check.targetOnly == 0 {
		t.Fatalf("expected diverged graph target, got currentOnly=%d targetOnly=%d", check.currentOnly, check.targetOnly)
	}

	gotModel, cmd = got.Update(check)
	got = gotModel.(model)
	if cmd != nil {
		t.Fatalf("expected no follow-up command after graph check, got %v", cmd)
	}
	if got.status.Mode != state.ModeConfirm {
		t.Fatalf("expected confirm mode after diverged graph target, got %s", got.status.Mode)
	}
	if got.status.Action != state.ActionMerge {
		t.Fatalf("expected merge action, got %s", got.status.Action)
	}
	if got.status.Selected != featureHash {
		t.Fatalf("expected selected target %q, got %q", featureHash, got.status.Selected)
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
