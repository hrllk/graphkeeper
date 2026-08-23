package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestClassifyPullPortPreviewOutcomeMatrix(t *testing.T) {
	status := git.Status{Branch: "main", Head: strings.Repeat("a", 40), Upstream: "origin/main", UpstreamOID: strings.Repeat("b", 40), Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 2}}, TrackingKnown: true, TrackingFresh: true}
	baseline := pullSnapshotIdentity(status, 3)
	request := &pullRequest{ID: 7, Epoch: 3, FetchBaseline: baseline}

	cases := map[string]struct {
		request   *pullRequest
		requestID uint64
		epoch     uint64
		stale     bool
		result    PullPreviewResult
		err       error
		local     PullSnapshotIdentity
		want      pullLifecycleOutcomeKind
	}{
		"identity mismatch is ignored":   {request: request, requestID: 8, epoch: 3, result: PullPreviewResult{RequestID: 8, RepositoryEpoch: 3}, local: baseline, want: pullLifecycleIdentityIgnored},
		"error is unavailable":           {request: request, requestID: 7, epoch: 3, result: PullPreviewResult{RequestID: 7, RepositoryEpoch: 3}, err: errors.New("preview failed"), local: baseline, want: pullLifecyclePreviewUnavailable},
		"ineligible is unavailable":      {request: request, requestID: 7, epoch: 3, result: PullPreviewResult{RequestID: 7, RepositoryEpoch: 3, Eligible: false}, local: baseline, want: pullLifecyclePreviewUnavailable},
		"baseline mismatch is stale":     {request: request, requestID: 7, epoch: 3, result: PullPreviewResult{RequestID: 7, RepositoryEpoch: 3, Eligible: true, Baseline: PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "changed"}, Commits: []PullPreviewCommit{{Hash: "commit"}}}, local: baseline, want: pullLifecyclePreviewStale},
		"no commits is no-op":            {request: request, requestID: 7, epoch: 3, result: PullPreviewResult{RequestID: 7, RepositoryEpoch: 3, Eligible: true, Baseline: baseline}, local: baseline, want: pullLifecycleNoOpCompleted},
		"commits are confirmation ready": {request: request, requestID: 7, epoch: 3, result: PullPreviewResult{RequestID: 7, RepositoryEpoch: 3, Eligible: true, Baseline: baseline, Commits: []PullPreviewCommit{{Hash: "commit"}}}, local: baseline, want: pullLifecycleConfirmationReady},
		"stale confirmation is ignored":  {request: request, requestID: 7, epoch: 3, stale: true, result: PullPreviewResult{RequestID: 7, RepositoryEpoch: 3, Eligible: true, Baseline: baseline, Commits: []PullPreviewCommit{{Hash: "commit"}}}, local: baseline, want: pullLifecycleIdentityIgnored},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := classifyPullPortPreviewOutcome(tc.request, tc.requestID, tc.epoch, tc.stale, tc.local, tc.result, tc.err)
			if got.Kind != tc.want {
				t.Fatalf("kind = %v, want %v", got.Kind, tc.want)
			}
		})
	}
}

func TestClassifyLegacyPreviewOutcomesPreservesGuardedDistinctions(t *testing.T) {
	status := git.Status{Branch: "main", Head: strings.Repeat("a", 40), Upstream: "origin/main", UpstreamOID: strings.Repeat("b", 40), Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 2}}, TrackingKnown: true, TrackingFresh: true}
	baseline := pullSnapshotIdentity(status, 3)
	request := &pullRequest{ID: 7, Epoch: 3, FetchBaseline: baseline, OperationBaseline: baseline}

	fetched := []struct {
		name string
		msg  pullFetchedMsg
		want pullLifecycleOutcomeKind
	}{
		{"identity mismatch", pullFetchedMsg{requestID: 8, requestEpoch: 3}, pullLifecycleIdentityIgnored},
		{"fetch baseline mismatch", pullFetchedMsg{requestID: 7, requestEpoch: 3, fetchBaseline: PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "changed"}, status: status, operationBaseline: baseline, operationBaselineSet: true}, pullLifecycleIdentityIgnored},
		{"missing operation baseline", pullFetchedMsg{requestID: 7, requestEpoch: 3, fetchBaseline: baseline, status: status}, pullLifecyclePreviewUnavailable},
		{"changed operation baseline", pullFetchedMsg{requestID: 7, requestEpoch: 3, fetchBaseline: baseline, status: status, operationBaseline: PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "changed"}, operationBaselineSet: true}, pullLifecyclePreviewStale},
		{"analysis ready", pullFetchedMsg{requestID: 7, requestEpoch: 3, fetchBaseline: baseline, status: status, operationBaseline: baseline, operationBaselineSet: true}, pullLifecycleFetchReady},
	}
	for _, tc := range fetched {
		t.Run("fetched/"+tc.name, func(t *testing.T) {
			got := classifyLegacyPullFetchedOutcome(request, false, tc.msg)
			if got.Kind != tc.want {
				t.Fatalf("kind = %v, want %v", got.Kind, tc.want)
			}
		})
	}

	ready := pullPreviewReadyMsg{requestID: 7, requestEpoch: 3, baseline: baseline, commits: []string{"commit"}}
	if got := classifyLegacyPullPreviewReadyOutcome(request, false, ready); got.Kind != pullLifecycleConfirmationReady {
		t.Fatalf("preview ready kind = %v, want confirmation ready", got.Kind)
	}
	ready.baseline.Head = "changed"
	if got := classifyLegacyPullPreviewReadyOutcome(request, false, ready); got.Kind != pullLifecycleIdentityIgnored {
		t.Fatalf("changed preview baseline kind = %v, want identity ignored", got.Kind)
	}
}

func TestTopLevelUpdateRoutesGuardedPreviewFamiliesWithoutChangingProjection(t *testing.T) {
	status := git.Status{Branch: "main", Head: strings.Repeat("a", 40), Upstream: "origin/main", UpstreamOID: strings.Repeat("b", 40), Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 2}}, TrackingKnown: true, TrackingFresh: true}
	baseline := pullSnapshotIdentity(status, 3)
	for name, msg := range map[string]any{
		"pull port": pullPortPreviewMsg{result: PullPreviewResult{RequestID: 7, RepositoryEpoch: 3, Baseline: baseline, Eligible: true, Commits: []PullPreviewCommit{{Hash: "commit"}}}},
		"legacy":    pullPreviewReadyMsg{requestID: 7, requestEpoch: 3, baseline: baseline, commits: []string{"commit"}, isFF: true},
	} {
		t.Run(name, func(t *testing.T) {
			m := model{
				repositoryState: repositoryState{repoStatus: status, repositoryEpoch: 3},
				pullState:       pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3, FetchBaseline: baseline, OperationBaseline: baseline}}, status: state.New().WithLoading("Previewing...")}
			got, cmd := m.Update(msg)
			modelGot := got.(model)
			if cmd != nil || modelGot.status.Mode != state.ModeConfirm || modelGot.status.Action != state.ActionPull {
				t.Fatalf("unexpected routed preview: status=%+v cmd=%v", modelGot.status, cmd)
			}
			if name == "pull port" {
				if modelGot.status.Title != "Pull into main?" || !strings.Contains(modelGot.status.Detail, "Merge: Histories will b") || !strings.Contains(modelGot.status.Detail, "Rebase: Local commits wi") || modelGot.mergeConfirmView == nil || modelGot.mergeConfirmView.MergeText != "Histories will be combined. A merge commit may be created." || modelGot.mergeConfirmView.RebaseText != "Local commits will be replayed onto origin/main. Commit identities may change." {
					t.Fatalf("unexpected PullPort confirmation projection: title=%q detail=%q merge=%#v", modelGot.status.Title, modelGot.status.Detail, modelGot.mergeConfirmView)
				}
			} else {
				if modelGot.status.Title != "Fast-forward pull?" || modelGot.status.Detail != "Fast-forward to the target commit." || !modelGot.pullIsFastForward || !modelGot.handshakeCommits["commit"] {
					t.Fatalf("unexpected legacy confirmation projection: status=%+v fastForward=%v handshake=%v", modelGot.status, modelGot.pullIsFastForward, modelGot.handshakeCommits)
				}
			}
		})
	}
}

func TestTopLevelUpdateRejectsLegacyFetchedFetchBaselineMismatch(t *testing.T) {
	status := git.Status{Branch: "main", Head: strings.Repeat("a", 40), Upstream: "origin/main", UpstreamOID: strings.Repeat("b", 40), Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 2}}, TrackingKnown: true, TrackingFresh: true}
	fetchBaseline := pullSnapshotIdentity(status, 3)
	before := model{
		repositoryState: repositoryState{
			repoStatus:      status,
			repositoryEpoch: 3,
		},
		pullState: pullState{
			activePullRequest: &pullRequest{ID: 7, Epoch: 3, FetchBaseline: fetchBaseline},
		},
		status: state.New().WithLoading("Fetching before pull...")}
	msg := pullFetchedMsg{
		requestID: 7, requestEpoch: 3,
		fetchBaseline:        PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "changed"},
		status:               status,
		operationBaseline:    pullSnapshotIdentity(status, 3),
		operationBaselineSet: true,
	}

	got, cmd := before.Update(msg)
	if cmd != nil || !reflect.DeepEqual(got.(model), before) {
		t.Fatalf("mismatched fetch baseline mutated model or emitted command: got=%#v cmd=%v", got, cmd)
	}
}

func TestTopLevelUpdateIgnoresStaleGuardedPreviewWithoutMutation(t *testing.T) {
	status := git.Status{Branch: "main", Head: strings.Repeat("a", 40), Upstream: "origin/main", UpstreamOID: strings.Repeat("b", 40), Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 2}}, TrackingKnown: true, TrackingFresh: true}
	before := model{
		repositoryState: repositoryState{repoStatus: status, repositoryEpoch: 3},
		pullState:       pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3}}, status: state.New().WithLoading("Previewing...")}
	for name, msg := range map[string]any{
		"pull port":      pullPortPreviewMsg{result: PullPreviewResult{RequestID: 8, RepositoryEpoch: 3, Eligible: true, Commits: []PullPreviewCommit{{Hash: "commit"}}}},
		"legacy fetched": pullFetchedMsg{requestID: 8, requestEpoch: 3},
		"legacy preview": pullPreviewReadyMsg{requestID: 8, requestEpoch: 3, commits: []string{"commit"}},
	} {
		t.Run(name, func(t *testing.T) {
			got, cmd := before.Update(msg)
			if cmd != nil || !reflect.DeepEqual(got.(model), before) {
				t.Fatalf("stale preview mutated model or emitted command: got=%#v cmd=%v", got, cmd)
			}
		})
	}
}

func TestTopLevelUpdateRoutesLegacyPullFetchedOutcomeMatrix(t *testing.T) {
	status := git.Status{Branch: "main", Head: strings.Repeat("a", 40), Upstream: "origin/main", UpstreamOID: strings.Repeat("b", 40), Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 0}}, TrackingKnown: true, TrackingFresh: true}
	baseline := pullSnapshotIdentity(status, 3)
	cases := map[string]struct {
		msg         pullFetchedMsg
		wantBlock   state.BlockReason
		wantTitle   string
		wantMessage string
		wantDetail  string
		wantMode    state.Mode
		wantCommand bool
	}{
		"fetch error":                {msg: pullFetchedMsg{requestID: 7, requestEpoch: 3, fetchBaseline: baseline, err: errors.New("network down")}, wantBlock: state.BlockFetchFailed, wantTitle: "Blocked", wantMessage: "Fetch before pull failed.", wantDetail: "network down"},
		"missing operation baseline": {msg: pullFetchedMsg{requestID: 7, requestEpoch: 3, fetchBaseline: baseline, status: status}, wantBlock: state.BlockUnknown, wantTitle: "Blocked", wantMessage: "Pull impact unavailable.", wantDetail: "Refresh before pulling again.", wantCommand: true},
		"tracking unavailable":       {msg: pullFetchedMsg{requestID: 7, requestEpoch: 3, fetchBaseline: baseline, status: git.Status{Branch: status.Branch}, operationBaseline: baseline, operationBaselineSet: true}, wantBlock: state.BlockStaleSnapshot, wantTitle: "Blocked", wantMessage: "Repository changed.", wantDetail: "Refresh before pulling again.", wantCommand: true},
		"fetched no-op":              {msg: pullFetchedMsg{requestID: 7, requestEpoch: 3, fetchBaseline: baseline, status: status, operationBaseline: baseline, operationBaselineSet: true}, wantMode: state.ModeOperationResult, wantTitle: "PULL COMPLETED", wantMessage: "PULL COMPLETED", wantDetail: "No action needed. Press q or Esc to return to the graph."},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			m := model{
				repositoryState: repositoryState{repoStatus: status, repositoryEpoch: 3},
				pullState:       pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3, FetchBaseline: baseline}}, status: state.New().WithLoading("Fetching...")}
			gotModel, cmd := m.Update(tc.msg)
			got := gotModel.(model)
			if got.activePullRequest != nil || (cmd != nil) != tc.wantCommand || got.status.Block != tc.wantBlock || got.status.Title != tc.wantTitle || got.status.Message != tc.wantMessage || got.status.Detail != tc.wantDetail || (tc.wantMode != "" && got.status.Mode != tc.wantMode) {
				t.Fatalf("unexpected fetched route: status=%+v active=%#v cmd=%v", got.status, got.activePullRequest, cmd)
			}
		})
	}
}

func TestTopLevelUpdateLegacyPullFetchedAnalysisReady(t *testing.T) {
	fixture := newCommandRepo(t)
	upstreamOID := advanceRemote(t, fixture.remote, "remote.txt", "remote\n", "remote advance")
	runGit(t, fixture.root, "fetch", "origin")

	status := git.Status{
		Root: fixture.root, Branch: "main", Head: fixture.initialHash,
		Upstream: "origin/main", UpstreamOID: upstreamOID,
		Tracking:      map[string]git.BranchTracking{"main": {Ahead: 0, Behind: 1}},
		TrackingKnown: true, TrackingFresh: true,
	}
	baseline := pullSnapshotIdentity(status, 3)
	request := &pullRequest{ID: 7, Epoch: 3, FetchBaseline: baseline, OperationBaseline: baseline, OperationBaselineSet: true}
	impactSnapshot := pullImpactSnapshot(baseline, *request)
	impact := pullImpactSet(impactSnapshot)
	before := model{
		repositoryState: repositoryState{repoStatus: status, repositoryEpoch: 3},
		pullState: pullState{
			activePullRequest: request,
		},
		repo:   fixture.repo,
		status: state.New().WithLoading("Fetching before pull...")}

	gotModel, cmd := before.Update(pullFetchedMsg{
		requestID: 7, requestEpoch: 3, fetchBaseline: baseline,
		status: status, operationBaseline: baseline, operationBaselineSet: true,
		snapshot: impactSnapshot,
	})
	if cmd == nil {
		t.Fatal("expected analysis command from legacy analysis-ready fetch")
	}
	readyMsg := cmd()
	ready, ok := readyMsg.(pullPreviewReadyMsg)
	if !ok {
		t.Fatalf("expected pullPreviewReadyMsg, got %T", readyMsg)
	}
	if ready.requestID != 7 || ready.requestEpoch != 3 || !samePullSnapshotIdentity(ready.baseline, baseline) {
		t.Fatalf("unexpected preview identity: request=%d/%d baseline=%+v", ready.requestID, ready.requestEpoch, ready.baseline)
	}
	if !reflect.DeepEqual(ready.snapshot, impactSnapshot) || !reflect.DeepEqual(ready.impact, impact) || ready.isFF != impactSnapshot.IsFastForward || ready.err != nil {
		t.Fatalf("unexpected preview payload: snapshot=%+v impact=%+v isFF=%v err=%v", ready.snapshot, ready.impact, ready.isFF, ready.err)
	}
	got := gotModel.(model)
	if got.activePullRequest == nil || got.activePullRequest.ID != 7 || got.activePullRequest.Epoch != 3 || !samePullSnapshotIdentity(got.activePullRequest.FetchBaseline, baseline) || !samePullSnapshotIdentity(got.activePullRequest.OperationBaseline, baseline) || !got.activePullRequest.OperationBaselineSet || got.status.Mode != state.ModeLoading || got.status.Action != state.ActionPull {
		t.Fatalf("legacy fetch changed active request/status unexpectedly: active=%#v status=%+v", got.activePullRequest, got.status)
	}
}

func TestTopLevelUpdateLegacyDivergentPreviewReadyConfirmsPull(t *testing.T) {
	status := git.Status{
		Branch: "main", Head: strings.Repeat("a", 40), Upstream: "origin/main", UpstreamOID: strings.Repeat("b", 40),
		Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 2}}, TrackingKnown: true, TrackingFresh: true,
	}
	baseline := pullSnapshotIdentity(status, 3)
	request := &pullRequest{ID: 7, Epoch: 3, OperationBaseline: baseline, OperationBaselineSet: true}
	snapshot := pullImpactSnapshot(baseline, *request)
	impact := pullImpactSet(snapshot)
	m := model{
		navigationState: navigationState{
			width: 120,
		},
		repositoryState: repositoryState{repoStatus: status, repositoryEpoch: 3},
		pullState:       pullState{activePullRequest: request}, status: state.New().WithLoading("Analyzing pull...")}

	gotModel, cmd := m.Update(pullPreviewReadyMsg{
		requestID: 7, requestEpoch: 3, baseline: baseline, isFF: false,
		snapshot: snapshot, impact: impact, commits: []string{"merge-commit", "local-commit"},
	})
	got := gotModel.(model)
	wantDetail := "Pull into main\nTarget: origin/main\nRelation: 1/2 commits\n\nMerge: Histories will be combined. A merge commit may be created.\nRebase: Local commits will be replayed onto origin/main. Commit identities may change.\nRisk: Conflicts may occur."
	if cmd != nil || got.status.Mode != state.ModeConfirm || got.status.Action != state.ActionPull || got.status.Title != "Pull into main?" || got.status.Detail != wantDetail || got.pullIsFastForward || !reflect.DeepEqual(got.handshakeCommits, map[string]bool{"merge-commit": true, "local-commit": true}) || got.mergeConfirmView == nil || got.mergeConfirmView.MergeText != "Histories will be combined. A merge commit may be created." || got.mergeConfirmView.RebaseText != "Local commits will be replayed onto origin/main. Commit identities may change." {
		t.Fatalf("unexpected divergent legacy confirmation: status=%+v fastForward=%v handshake=%v view=%#v cmd=%v", got.status, got.pullIsFastForward, got.handshakeCommits, got.mergeConfirmView, cmd)
	}
}

func TestTopLevelUpdateRoutesLegacyPreviewOutcomeMatrix(t *testing.T) {
	status := git.Status{Branch: "main", Head: strings.Repeat("a", 40), Upstream: "origin/main", UpstreamOID: strings.Repeat("b", 40), Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 2}}, TrackingKnown: true, TrackingFresh: true}
	baseline := pullSnapshotIdentity(status, 3)
	cases := map[string]struct {
		msg         pullPreviewReadyMsg
		wantBlock   state.BlockReason
		wantTitle   string
		wantMessage string
		wantDetail  string
		wantMode    state.Mode
	}{
		"analysis error": {msg: pullPreviewReadyMsg{requestID: 7, requestEpoch: 3, baseline: baseline, err: errors.New("analysis failed")}, wantBlock: state.BlockUnknown, wantTitle: "Blocked", wantMessage: "Analysis failed.", wantDetail: "analysis failed"},
		"analysis no-op": {msg: pullPreviewReadyMsg{requestID: 7, requestEpoch: 3, baseline: baseline}, wantMode: state.ModeOperationResult, wantTitle: "PULL COMPLETED", wantMessage: "PULL COMPLETED", wantDetail: "No action needed. Press q or Esc to return to the graph."},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			m := model{
				repositoryState: repositoryState{repoStatus: status, repositoryEpoch: 3},
				pullState:       pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3, OperationBaseline: baseline, OperationBaselineSet: true}}, status: state.New().WithLoading("Analyzing pull...")}
			gotModel, cmd := m.Update(tc.msg)
			got := gotModel.(model)
			if got.activePullRequest != nil || cmd != nil || got.status.Block != tc.wantBlock || got.status.Title != tc.wantTitle || got.status.Message != tc.wantMessage || got.status.Detail != tc.wantDetail || (tc.wantMode != "" && got.status.Mode != tc.wantMode) {
				t.Fatalf("unexpected preview route: status=%+v active=%#v cmd=%v", got.status, got.activePullRequest, cmd)
			}
		})
	}
}
