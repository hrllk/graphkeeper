package app

import (
	"context"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

func TestTask214NewWithDependenciesPreservesGroupedDefaults(t *testing.T) {
	repo := &git.Repo{}
	read := &fakeRepositoryRead{}
	pull := &fakePull{}
	reader := &fakeInspectorReader{}
	deps := Dependencies{Repo: repo, RepositoryRead: read, Pull: pull, InspectorReader: reader}
	gotModel, err := NewWithDependencies(deps)
	if err != nil {
		t.Fatal(err)
	}
	got := gotModel.(model)
	if got.repo != repo || got.repositoryRead != read || got.pull != pull || got.inspectorReader != reader {
		t.Fatal("constructor did not preserve injected capability identity")
	}
	if got.status.Mode != state.ModeLoading || got.status.Message != "Loading..." || !got.startupReadPending {
		t.Fatalf("unexpected constructor status/defaults: %#v", got.status)
	}
	if len(got.sectionCursor) != 4 || got.sectionCursor[sectionGraph] != 0 || got.sectionCursor[sectionCurrent] != 0 || got.sectionCursor[sectionRemote] != 0 || got.sectionCursor[sectionTags] != 0 {
		t.Fatalf("unexpected section defaults: %#v", got.sectionCursor)
	}
	if got.handshakeCommits == nil || len(got.handshakeCommits) != 0 || got.stashByBase == nil || len(got.stashByBase) != 0 {
		t.Fatalf("constructor maps are not initialized empty: handshake=%#v stash=%#v", got.handshakeCommits, got.stashByBase)
	}
	if got.activeSection != sectionGraph || got.graphLaneCursor != 0 || got.commitLimit != 0 || got.graphStashPopMode != graphStashPopModePicker {
		t.Fatalf("unexpected scalar defaults: %#v", got)
	}
}

func TestTask214ModelValueCopyPreservesAliasesAndCapabilities(t *testing.T) {
	repo := &git.Repo{}
	read := &fakeRepositoryRead{}
	pull := &fakePull{}
	reader := &fakeInspectorReader{}
	original := model{
		navigationState: navigationState{
			sectionCursor: map[graphSection]int{sectionGraph: 1},
		},
		repositoryState: repositoryState{
			stashByBase: map[string][]git.StashEntry{"base": {{Hash: "stash"}}},
			tagEntries:  []git.TagEntry{{Name: "v1"}},
		},
		pullState: pullState{
			handshakeCommits: map[string]bool{"abc": true},
		},
		repo: repo, repositoryRead: read, pull: pull, inspectorReader: reader,
		inspectorState: inspectorState{
			commitInspectorLines: []string{"line"}},
	}
	copy := original
	copy.sectionCursor[sectionGraph] = 2
	copy.handshakeCommits["def"] = true
	copy.stashByBase["base"][0].Hash = "changed"
	copy.tagEntries[0].Name = "v2"
	copy.commitInspectorLines[0] = "changed"
	if original.sectionCursor[sectionGraph] != 2 || !original.handshakeCommits["def"] || original.stashByBase["base"][0].Hash != "changed" || original.tagEntries[0].Name != "v2" || original.commitInspectorLines[0] != "changed" {
		t.Fatal("model value copy stopped sharing existing map/slice backing storage")
	}
	if copy.repo != repo || copy.repositoryRead != read || copy.pull != pull || copy.inspectorReader != reader {
		t.Fatal("model value copy changed injected capability identity")
	}

}

type task214PullRecorder struct {
	mode PullMode
	req  PullExecutionRequest
}

func (p *task214PullRecorder) Preview(context.Context, PullPreviewRequest) (PullPreviewResult, error) {
	return PullPreviewResult{}, nil
}
func (p *task214PullRecorder) Validate(_ context.Context, req PullValidationRequest) (PullValidationResult, error) {
	return PullValidationResult{Valid: true, Authorized: true, AuthorizedBaseline: req.Expected}, nil
}
func (p *task214PullRecorder) Execute(_ context.Context, req PullExecutionRequest) (PullExecutionResult, error) {
	p.mode = req.Mode
	p.req = req
	return PullExecutionResult{Mode: req.Mode, Succeeded: true}, nil
}

func TestTask214PullOpenCancelAndStaleTransitions(t *testing.T) {
	baseline := PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "head", Upstream: "origin/main", TrackingKnown: true, TrackingFresh: true}
	for _, mode := range []PullMode{PullModeMerge, PullModeRebase} {
		t.Run("open-"+string(mode), func(t *testing.T) {
			p := &task214PullRecorder{}
			read := &fakeRepositoryRead{result: ReadSnapshotResult{ErrorKind: ReadErrorNone, Snapshot: ReadSnapshot{Freshness: PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "head", Upstream: "origin/main", Behind: 1, TrackingKnown: true, TrackingFresh: true}}}}
			m := model{
				pullState: pullState{nextPullRequestID: 4, activePullRequest: &pullRequest{Epoch: 3, OperationBaseline: baseline}}, pull: p, repositoryRead: read}
			cancelled := 0
			m.pullCancel = func() { cancelled++ }
			cmd := startPullWorkflow(&m, mode)
			if cmd == nil || cancelled != 1 || m.nextPullRequestID != 5 || m.activePullRequest.ID != 5 || m.pullCancel == nil {
				t.Fatalf("pull open transition changed: cmd=%v cancels=%d next=%d request=%#v", cmd != nil, cancelled, m.nextPullRequestID, m.activePullRequest)
			}
			msg := cmd()
			workflow, ok := msg.(pullWorkflowMsg)
			if !ok || p.mode != mode || p.req.Mode != mode || p.req.RequestID != 5 || p.req.RepositoryEpoch != 3 || !p.req.Authorized || p.req.AuthorizedBaseline != baseline {
				t.Fatalf("pull command did not execute exact mode/request: msg=%T mode=%q req=%#v", msg, p.mode, p.req)
			}
			if workflow.result.Execute.Mode != mode {
				t.Fatalf("pull workflow result mode=%q, want %q", workflow.result.Execute.Mode, mode)
			}
		})
	}
	for _, tc := range []struct {
		name      string
		mode      state.Mode
		wantCmd   bool
		wantStale bool
	}{
		{"loading", state.ModeLoading, true, false},
		{"review", state.ModeReview, false, true},
		{"confirm", state.ModeConfirm, false, true},
		{"browse", state.ModeBrowse, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cancelled := 0
			m := model{
				repositoryState: repositoryState{repositoryEpoch: 3},
				pullState:       pullState{activePullRequest: &pullRequest{ID: 8, Epoch: 3}, pullCancel: func() { cancelled++ }}, status: state.Status{Mode: tc.mode, Action: state.ActionPull}}
			gotModel, cmd := m.Update(refreshedMsg{epoch: 2, epochSet: true})
			got := gotModel.(model)
			wantCancelled := 0
			if tc.mode == state.ModeLoading {
				wantCancelled = 1
			}
			if (cmd != nil) != tc.wantCmd || (got.pullConfirmStale) != tc.wantStale || cancelled != wantCancelled {
				t.Fatalf("stale transition mismatch: cmd=%v stale=%v cancels=%d", cmd != nil, got.pullConfirmStale, cancelled)
			}
			if tc.mode == state.ModeLoading {
				if got.activePullRequest != nil || got.nextPullRequestID != 1 || got.status.Mode != state.ModeBlocked || got.status.Block != state.BlockStaleSnapshot {
					t.Fatalf("loading stale transition did not clear/block: %#v", got)
				}
			} else if got.activePullRequest == nil {
				t.Fatal("non-loading stale transition unexpectedly cleared active pull")
			}
		})
	}
}

func TestTask214InspectorLifecycleAndResultGuards(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := model{inspectorState: inspectorState{commitInspectorOpen: true, commitInspectorSnapshot: CommitSnapshot{FullHash: "commit", Parent: "parent", Files: []ChangedFile{{StableID: "file"}}}, commitInspectorCursor: 0, commitInspectorEpoch: 4, commitInspectorCancel: cancel, commitInspectorContext: ctx}}
	m, cmd := m.startInspectorDiff()
	if cmd == nil || m.commitInspectorRequest != 1 || !m.commitInspectorLoading || !m.commitInspectorDiffLoading || m.commitInspectorContext == ctx {
		t.Fatalf("inspector start transition mismatch: cmd=%v request=%d loading=%v diff=%v", cmd != nil, m.commitInspectorRequest, m.commitInspectorLoading, m.commitInspectorDiffLoading)
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("starting a new inspector request did not cancel the prior context")
	}
	m = m.cancelInspector()
	if m.commitInspectorCancel != nil || !m.commitInspectorOpen {
		t.Fatal("cancelInspector did not clear only the cancel function")
	}
	m.commitInspectorCancel = func() {}
	m.commitInspectorHelp, m.commitInspectorMetadataLoading, m.commitInspectorDiffLoading, m.commitInspectorLoading, m.commitInspectorContinuationPending = true, true, true, true, true
	m.commitInspectorError = "keep"
	m.commitInspectorRequest, m.commitInspectorEpoch = 7, 4
	m = m.closeCommitInspector()
	if m.commitInspectorOpen || m.commitInspectorHelp || m.commitInspectorMetadataLoading || m.commitInspectorDiffLoading || m.commitInspectorLoading || m.commitInspectorContinuationPending || m.commitInspectorError != "keep" || m.commitInspectorRequest != 8 || m.commitInspectorEpoch != 4 {
		t.Fatalf("closeCommitInspector reset/preserved wrong fields: %#v", m)
	}

	cancelled := 0
	m = model{
		repositoryState: repositoryState{repositoryEpoch: 6},
		pullState:       pullState{}, inspectorState: inspectorState{commitInspectorOpen: true, commitInspectorEpoch: 4, commitInspectorLoading: true, commitInspectorMetadataLoading: true, commitInspectorDiffLoading: true, commitInspectorStale: false, commitInspectorCancel: func() { cancelled++ }},
	}
	gotModel, cmd := m.Update(refreshedMsg{epoch: 6, epochSet: true})
	got := gotModel.(model)
	if cmd == nil || cancelled != 1 || got.commitInspectorStale || !got.commitInspectorLoading || !got.commitInspectorMetadataLoading || got.commitInspectorDiffLoading || !got.commitInspectorRevalidating || got.commitInspectorRequest != 1 || got.commitInspectorEpoch != 6 {
		t.Fatalf("epoch refresh revalidation mismatch: cmd=%v cancels=%d state=%#v", cmd != nil, cancelled, got)
	}

	guarded := model{inspectorState: inspectorState{commitInspectorOpen: true, commitInspectorRequest: 2, commitInspectorEpoch: 4, commitInspectorRequestedCommit: "commit", commitInspectorSnapshot: CommitSnapshot{FullHash: "commit", Parent: "parent", Files: []ChangedFile{{StableID: "file"}}}, commitInspectorWindowRequest: DiffWindowRequest{StartLine: 3, MaxLines: 2, MaxBytes: 64}, commitInspectorDiffWindow: DiffWindow{FileID: "prior"}, commitInspectorLines: []string{"prior"}}}
	matching := InspectorResult[DiffWindow]{State: PaneReady, Commit: "commit", Parent: "parent", FileID: "file", RequestID: 2, RepositoryEpoch: 4, Window: guarded.commitInspectorWindowRequest, Value: DiffWindow{FileID: "file", HasMore: true, NextStartLine: 5}}
	for _, tc := range []struct {
		name   string
		mutate func(*InspectorResult[DiffWindow])
	}{
		{"request-id", func(r *InspectorResult[DiffWindow]) { r.RequestID = 1 }},
		{"commit", func(r *InspectorResult[DiffWindow]) { r.Commit = "other" }},
		{"parent", func(r *InspectorResult[DiffWindow]) { r.Parent = "other-parent" }},
		{"file-id", func(r *InspectorResult[DiffWindow]) { r.FileID = "other-file" }},
		{"repository-epoch", func(r *InspectorResult[DiffWindow]) { r.RepositoryEpoch = 3 }},
		{"diff-window", func(r *InspectorResult[DiffWindow]) { r.Window.StartLine++ }},
	} {
		t.Run("reject-"+tc.name, func(t *testing.T) {
			result := matching
			tc.mutate(&result)
			before := guarded
			gotModel, cmd := guarded.Update(commitInspectorResultMsg{Result: result})
			if cmd != nil || !reflect.DeepEqual(gotModel.(model), before) {
				t.Fatalf("mismatched %s result mutated model: cmd=%v got=%#v before=%#v", tc.name, cmd != nil, gotModel.(model), before)
			}
		})
	}
}

func TestTask214OverlayPrecedenceAndQEsc(t *testing.T) {
	m := model{inspectorState: inspectorState{commitInspectorOpen: true},
		overlayState: overlayState{graphStashPopOpen: true, stashMessageOpen: true, tagPopupOpen: true, stashPopupOpen: true, branchOpen: true, hiddenHotkeysOpen: true, graphSearchOpen: true}}
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := gotModel.(model)
	if cmd != nil || !got.commitInspectorOpen || !got.graphStashPopOpen || !got.stashMessageOpen || !got.tagPopupOpen || !got.stashPopupOpen || !got.branchOpen || !got.hiddenHotkeysOpen || !got.graphSearchOpen {
		t.Fatalf("q mutated an overlay instead of remaining input/no-op: %#v cmd=%v", got, cmd != nil)
	}
	cases := []struct {
		name string
		m    model
		open func(model) bool
	}{
		{"commit inspector", model{inspectorState: inspectorState{commitInspectorOpen: true}}, func(m model) bool { return m.commitInspectorOpen }},
		{"graph stash pop", model{overlayState: overlayState{graphStashPopOpen: true}}, func(m model) bool { return m.graphStashPopOpen }},
		{"stash message", model{overlayState: overlayState{stashMessageOpen: true}}, func(m model) bool { return m.stashMessageOpen }},
		{"tag popup", model{overlayState: overlayState{tagPopupOpen: true}}, func(m model) bool { return m.tagPopupOpen }},
		{"stash popup", model{overlayState: overlayState{stashPopupOpen: true}}, func(m model) bool { return m.stashPopupOpen }},
		{"branch", model{overlayState: overlayState{branchOpen: true}}, func(m model) bool { return m.branchOpen }},
		{"hidden hotkeys", model{overlayState: overlayState{hiddenHotkeysOpen: true}}, func(m model) bool { return m.hiddenHotkeysOpen }},
		{"graph search", model{overlayState: overlayState{graphSearchOpen: true}}, func(m model) bool { return m.graphSearchOpen }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			qModel, qCmd := tc.m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
			if qCmd != nil || !tc.open(qModel.(model)) {
				t.Fatalf("q changed %s overlay: cmd=%v model=%#v", tc.name, qCmd != nil, qModel)
			}
			escModel, escCmd := qModel.(model).Update(tea.KeyMsg{Type: tea.KeyEsc})
			if escCmd != nil || tc.open(escModel.(model)) {
				t.Fatalf("Esc did not close %s overlay: cmd=%v model=%#v", tc.name, escCmd != nil, escModel)
			}
		})
	}
}

func TestTask214NavigationWrapBoundsAndReset(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
			sectionCursor: map[graphSection]int{sectionCurrent: 0, sectionRemote: 0, sectionTags: 0},
		},
		repositoryState: repositoryState{repoStatus: git.Status{Branch: "main", LocalBranches: []string{"main", "feature"}, RemoteBranches: []string{"origin/main", "upstream/main", "fork/main"}, Tags: []string{"v1", "v2", "v3"}}},
		pullState:       pullState{}}
	m = moveSectionBrowseCursor(m, -1)
	if m.sectionCursor[sectionCurrent] != 1 || activeSectionTarget(m) != "feature" {
		t.Fatalf("backward local navigation = %d/%q, want 1/feature", m.sectionCursor[sectionCurrent], activeSectionTarget(m))
	}
	m = moveSectionBrowseCursor(m, 1)
	if m.sectionCursor[sectionCurrent] != 0 || activeSectionTarget(m) != "main" {
		t.Fatalf("forward local navigation = %d/%q, want 0/main", m.sectionCursor[sectionCurrent], activeSectionTarget(m))
	}
	m.activeSection = sectionRemote
	m = moveSectionBrowseCursor(m, -1)
	if m.sectionCursor[sectionRemote] != 2 || activeSectionTarget(m) != "fork/main" {
		t.Fatalf("backward remote navigation = %d/%q, want 2/fork/main", m.sectionCursor[sectionRemote], activeSectionTarget(m))
	}
	m = moveSectionBrowseCursor(m, 1)
	if m.sectionCursor[sectionRemote] != 0 || activeSectionTarget(m) != "origin/main" {
		t.Fatalf("forward remote navigation = %d/%q, want 0/origin/main", m.sectionCursor[sectionRemote], activeSectionTarget(m))
	}
	m.activeSection = sectionTags
	m = moveSectionBrowseCursor(m, -1)
	if m.sectionCursor[sectionTags] != 2 || activeSectionTarget(m) != "v3" {
		t.Fatalf("backward tag navigation = %d/%q, want 2/v3", m.sectionCursor[sectionTags], activeSectionTarget(m))
	}
	m = moveSectionBrowseCursor(m, 1)
	if m.sectionCursor[sectionTags] != 0 || activeSectionTarget(m) != "v1" {
		t.Fatalf("forward tag navigation = %d/%q, want 0/v1", m.sectionCursor[sectionTags], activeSectionTarget(m))
	}
	m.repoStatus.RemoteBranches = nil
	m.activeSection = sectionRemote
	m = moveSectionBrowseCursor(m, 1)
	if m.sectionCursor[sectionRemote] != -1 || activeSectionTarget(m) != "" {
		t.Fatalf("empty remote section = %d/%q, want -1/empty", m.sectionCursor[sectionRemote], activeSectionTarget(m))
	}
	m.repoStatus.Tags = []string{"v1"}
	m.activeSection = sectionTags
	m = moveSectionBrowseCursor(m, -1)
	if m.sectionCursor[sectionTags] != 0 || activeSectionTarget(m) != "v1" {
		t.Fatalf("single-item tag section = %d/%q, want 0/v1", m.sectionCursor[sectionTags], activeSectionTarget(m))
	}
	status := git.Status{GraphCommits: []git.GraphCommit{{Hash: "head"}}, Branch: "main", LocalBranches: []string{"main"}, Tags: []string{"v1"}}
	m.graphScroll, m.graphLaneCursor, m.sectionCursor[sectionGraph] = 99, 99, 99
	m.repoStatus = status
	syncBrowseState(&m, status)
	if m.sectionCursor[sectionGraph] != 0 || m.graphLaneCursor != graph.PointerLane(graphRows(status)[0]) || m.graphScroll < 0 {
		t.Fatalf("navigation reset/bounds changed: cursor=%d lane=%d scroll=%d", m.sectionCursor[sectionGraph], m.graphLaneCursor, m.graphScroll)
	}
}

func task214OverlayPayload(m model) any {
	return struct {
		branchOpen, tagPopupOpen, stashMessageOpen, stashPopupOpen, graphStashPopOpen, hiddenHotkeysOpen, graphSearchOpen bool
		branchDraft, branchBase, branchError, tagPopupDraft, tagPopupError, tagPopupTarget                                string
		stashMessageDraft, stashMessageError                                                                              string
		stashPopupCursor, graphStashPopCursor, hiddenHotkeysScroll, graphSearchCursor                                     int
		graphStashPopMode                                                                                                 graphStashPopMode
		graphStashPopEntries                                                                                              []git.StashEntry
		graphSearchDraft, graphSearchQuery, graphSearchError                                                              string
		graphSearchIndex                                                                                                  []graphSearchEntry
	}{
		m.branchOpen, m.tagPopupOpen, m.stashMessageOpen, m.stashPopupOpen, m.graphStashPopOpen, m.hiddenHotkeysOpen, m.graphSearchOpen,
		m.branchDraft, m.branchBase, m.branchError, m.tagPopupDraft, m.tagPopupError, m.tagPopupTarget,
		m.stashMessageDraft, m.stashMessageError,
		m.stashPopupCursor, m.graphStashPopCursor, m.hiddenHotkeysScroll, m.graphSearchCursor,
		m.graphStashPopMode, m.graphStashPopEntries, m.graphSearchDraft, m.graphSearchQuery, m.graphSearchError,
		m.graphSearchIndex,
	}
}

func TestRepositoryRefreshWhileOverlayOpen(t *testing.T) {
	updated := git.Status{Root: "/repo", Branch: "fresh", Head: "new-head", GraphCommits: []git.GraphCommit{{Hash: "new-head"}}}
	cases := []struct {
		name  string
		setup func(*model)
		open  func(model) bool
		close func(model) (model, tea.Cmd)
	}{
		{"commitInspectorOpen", func(m *model) {
			m.commitInspectorOpen = true
			m.commitInspectorHelp = false
			m.commitInspectorRequestedCommit = "draft"
		}, func(m model) bool { return m.commitInspectorOpen }, func(m model) (model, tea.Cmd) {
			next, cmd := m.handleCommitInspectorKey(tea.KeyMsg{Type: tea.KeyEsc})
			return next.(model), cmd
		}},
		{"graphStashPopOpen", func(m *model) { m.graphStashPopOpen = true; m.graphStashPopEntries = []git.StashEntry{{Hash: "stash"}} }, func(m model) bool { return m.graphStashPopOpen }, func(m model) (model, tea.Cmd) {
			next, cmd := m.handleGraphStashPopKey(tea.KeyMsg{Type: tea.KeyEsc})
			return next.(model), cmd
		}},
		{"stashMessageOpen", func(m *model) {
			m.stashMessageOpen = true
			m.stashMessageDraft = "draft"
			m.stashMessageError = "error"
		}, func(m model) bool { return m.stashMessageOpen }, func(m model) (model, tea.Cmd) {
			next, cmd := m.handleStashMessageKey(tea.KeyMsg{Type: tea.KeyEsc})
			return next.(model), cmd
		}},
		{"tagPopupOpen", func(m *model) {
			m.tagPopupOpen = true
			m.tagPopupDraft = "v1"
			m.tagPopupError = "error"
			m.tagPopupTarget = "head"
		}, func(m model) bool { return m.tagPopupOpen }, func(m model) (model, tea.Cmd) {
			next, cmd := m.handleTagPopupKey(tea.KeyMsg{Type: tea.KeyEsc})
			return next.(model), cmd
		}},
		{"stashPopupOpen", func(m *model) { m.stashPopupOpen = true; m.stashPopupCursor = 2 }, func(m model) bool { return m.stashPopupOpen }, func(m model) (model, tea.Cmd) {
			next, cmd := m.handleStashPopupKey(tea.KeyMsg{Type: tea.KeyEsc})
			return next.(model), cmd
		}},
		{"branchOpen", func(m *model) {
			m.branchOpen = true
			m.branchDraft = "draft"
			m.branchError = "error"
			m.branchBase = "base"
		}, func(m model) bool { return m.branchOpen }, func(m model) (model, tea.Cmd) {
			next, cmd := m.handleBranchOpenKey(tea.KeyMsg{Type: tea.KeyEsc})
			return next.(model), cmd
		}},
		{"hiddenHotkeysOpen", func(m *model) { m.hiddenHotkeysOpen = true; m.hiddenHotkeysScroll = 3 }, func(m model) bool { return m.hiddenHotkeysOpen }, func(m model) (model, tea.Cmd) {
			next, cmd := m.handleHiddenHotkeysKey(tea.KeyMsg{Type: tea.KeyEsc})
			return next.(model), cmd
		}},
		{"graphSearchOpen", func(m *model) {
			m.graphSearchOpen = true
			m.graphSearchDraft = "draft"
			m.graphSearchQuery = "query"
			m.graphSearchError = "error"
		}, func(m model) bool { return m.graphSearchOpen }, func(m model) (model, tea.Cmd) {
			next, cmd := m.handleGraphSearchKey(tea.KeyMsg{Type: tea.KeyEsc})
			return next.(model), cmd
		}},
	}
	statusModes := []state.Mode{state.ModeConfirm, state.ModeReview, state.ModeBlocked, state.ModeOperationResult}
	for _, mode := range statusModes {
		mode := mode
		cases = append(cases, struct {
			name  string
			setup func(*model)
			open  func(model) bool
			close func(model) (model, tea.Cmd)
		}{"status-" + string(mode), func(m *model) {
			m.commitInspectorOpen = false
			m.status = state.Status{Mode: mode, Action: state.ActionPull, Block: state.BlockDirtyTree, Title: "payload title", Message: "payload message", Detail: "payload detail", TargetIdx: 2, Selected: "payload-target", CanExecute: mode == state.ModeConfirm}
			m.operationResult = &OperationResultSummary{Headline: "payload operation"}
		}, func(m model) bool { return m.status.Mode == mode }, func(m model) (model, tea.Cmd) {
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			return next.(model), cmd
		}})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inspectorContext, inspectorCancel := context.WithCancel(context.Background())
			m := model{
				repositoryState: repositoryState{repositoryEpoch: 1, repoStatus: git.Status{Head: "old-head"}},
				pullState:       pullState{},
				status:          state.New().WithBrowse(),
				overlayState:    overlayState{branchDraft: "branch draft", branchBase: "branch base", branchError: "branch error", tagPopupDraft: "tag draft", tagPopupError: "tag error", tagPopupTarget: "tag target", stashMessageDraft: "stash draft", stashMessageError: "stash error", stashPopupCursor: 4, graphStashPopCursor: 5, graphStashPopMode: graphStashPopModeConfirm, graphStashPopEntries: []git.StashEntry{{Hash: "stash"}}, hiddenHotkeysScroll: 6, graphSearchDraft: "search draft", graphSearchQuery: "search query", graphSearchError: "search error", graphSearchCursor: 7},

				inspectorState: inspectorState{
					commitInspectorOpen:          false,
					commitInspectorSnapshot:      CommitSnapshot{FullHash: "commit", Parent: "parent", Files: []ChangedFile{{StableID: "file", Path: "file.go"}}},
					commitInspectorDiffWindow:    DiffWindow{FileID: "file", HasMore: true, NextStartLine: 9},
					commitInspectorWindowRequest: DiffWindowRequest{StartLine: 2, MaxLines: 3, MaxBytes: 64},
					commitInspectorCursor:        1, commitInspectorScroll: 2, commitInspectorLines: []string{"line"},
					commitInspectorHasMore: true, commitInspectorLoading: true,
					commitInspectorError: "inspector error", commitInspectorRequest: 8, commitInspectorEpoch: 1,
					commitInspectorRequestedCommit: "commit", commitInspectorRequestedParent: "parent",
					commitInspectorHelp: true, commitInspectorMessage: true, commitInspectorMessageScroll: 3,
					commitInspectorMetadataLoading: true, commitInspectorDiffLoading: true,
					commitInspectorDiffError: "diff error", commitInspectorStale: true, commitInspectorContinuationPending: true,
					commitInspectorCancel: inspectorCancel, commitInspectorContext: inspectorContext,
					commitInspector: CommitSnapshot{FullHash: "commit", Subject: "subject", Parent: "parent", Files: []ChangedFile{{StableID: "file", Path: "file.go"}}},
				},
			}
			m.status.WorktreeState = state.WorktreeStateClean
			tc.setup(&m)
			if tc.name == "commitInspectorOpen" {
				m.commitInspectorOpen = true
			}
			before := m
			if !tc.open(m) {
				t.Fatal("fixture did not open overlay")
			}
			beforeOverlay := task214OverlayPayload(m)
			gotModel, cmd := m.Update(refreshedMsg{status: updated, epoch: 1, epochSet: true})
			got := gotModel.(model)
			if cmd == nil {
				t.Fatal("refresh did not retain load-stash command")
			}
			if got.repoStatus.Head != "new-head" {
				t.Fatalf("refresh did not apply repository head: %#v", got.repoStatus)
			}
			if !tc.open(got) {
				t.Fatalf("refresh closed overlay: %#v", got)
			}
			if !reflect.DeepEqual(got.status, before.status) {
				t.Fatalf("refresh changed status: got=%#v before=%#v", got.status, before.status)
			}
			if !reflect.DeepEqual(beforeOverlay, task214OverlayPayload(got)) {
				t.Fatalf("refresh changed overlay payload: got=%#v before=%#v", task214OverlayPayload(got), beforeOverlay)
			}
			closed, closeCmd := tc.close(got)
			if closeCmd != nil || tc.open(closed) {
				t.Fatalf("overlay close behavior changed: cmd=%v state=%#v", closeCmd != nil, closed)
			}
			if tc.name == "status-operation_result" {
				if closed.status.Mode != state.ModeBrowse || closed.status.Action != state.ActionNone || closed.status.Title != "Browse" || closed.status.Message != "Choose an action." || closed.status.Detail != "" || closed.status.TargetIdx != -1 || closed.status.Selected != "" || closed.status.CanExecute {
					t.Fatalf("operation-result Esc did not restore exact browse status: %#v", closed.status)
				}
			} else if len(tc.name) >= 7 && tc.name[:7] == "status-" {
				if closed.status.Mode != state.ModeBrowse || closed.status.Action != state.ActionNone || closed.status.Title != "Browse" || closed.status.Message != "Choose an action." || closed.status.Detail != "" || closed.status.TargetIdx != -1 || closed.status.Selected != "" || closed.status.CanExecute {
					t.Fatalf("status-mode Esc did not restore exact browse status: %#v", closed.status)
				}
				if closed.operationResult == nil || closed.operationResult.Headline != "payload operation" {
					t.Fatalf("Esc did not preserve exact operation-result payload: %#v", closed.operationResult)
				}
			}
		})
	}
}
