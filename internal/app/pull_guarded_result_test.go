package app

import (
	"errors"
	"reflect"
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestClassifyPullGuardedResultFamilies(t *testing.T) {
	baseline := PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "head"}
	request := &pullRequest{ID: 7, Epoch: 3, OperationBaseline: baseline, OperationBaselineSet: true}

	t.Run("validation identity and baseline guards", func(t *testing.T) {
		cases := []struct {
			name string
			msg  pullValidationMsg
			want pullLifecycleOutcomeKind
		}{
			{"request mismatch", pullValidationMsg{requestID: 8, requestEpoch: 3, baseline: baseline, valid: true}, pullLifecycleIdentityIgnored},
			{"epoch mismatch", pullValidationMsg{requestID: 7, requestEpoch: 4, baseline: baseline, valid: true}, pullLifecycleIdentityIgnored},
			{"fetch baseline supplied", pullValidationMsg{requestID: 7, requestEpoch: 3, baseline: PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "fetch"}, valid: true}, pullLifecyclePreviewStale},
			{"operation baseline supplied", pullValidationMsg{requestID: 7, requestEpoch: 3, baseline: baseline, valid: true}, pullLifecycleConfirmationReady},
			{"rejected", pullValidationMsg{requestID: 7, requestEpoch: 3, baseline: baseline, valid: false}, pullLifecycleValidationRejected},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if got := classifyPullValidationOutcome(request, false, tc.msg); got.Kind != tc.want {
					t.Fatalf("kind=%v want=%v", got.Kind, tc.want)
				}
			})
		}
	})

	t.Run("execution guards and outcomes", func(t *testing.T) {
		matching := pullExecutionResultMsg{requestID: 7, requestEpoch: 3, baseline: baseline}
		cases := []struct {
			name string
			msg  pullExecutionResultMsg
			want pullLifecycleOutcomeKind
		}{
			{"request mismatch", pullExecutionResultMsg{requestID: 8, requestEpoch: 3, baseline: baseline}, pullLifecycleIdentityIgnored},
			{"baseline mismatch", pullExecutionResultMsg{requestID: 7, requestEpoch: 3, baseline: PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "changed"}}, pullLifecyclePreviewStale},
			{"stale result", pullExecutionResultMsg{requestID: 7, requestEpoch: 3, baseline: baseline, stale: true}, pullLifecyclePreviewStale},
			{"matching result", matching, pullLifecycleCompleted},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if got := classifyPullExecutionResultOutcome(request, tc.msg); got.Kind != tc.want {
					t.Fatalf("kind=%v want=%v", got.Kind, tc.want)
				}
			})
		}
	})

	t.Run("workflow operation and refresh identity", func(t *testing.T) {
		cases := []struct {
			name string
			msg  pullWorkflowMsg
			want pullLifecycleOutcomeKind
		}{
			{"operation mismatch", pullWorkflowMsg{result: PullWorkflowResult{OperationRequestID: 8, OperationEpoch: 3}}, pullLifecycleIdentityIgnored},
			{"refresh failure", pullWorkflowMsg{result: PullWorkflowResult{OperationRequestID: 7, OperationEpoch: 3, RefreshErrorKind: ReadErrorRepository}}, pullLifecycleRefreshFailed},
			{"execution rejection", pullWorkflowMsg{result: PullWorkflowResult{OperationRequestID: 7, OperationEpoch: 3, RefreshErrorKind: ReadErrorNone, Execute: PullExecutionResult{Reason: PullRejectChangedBaseline}}}, pullLifecycleValidationRejected},
			{"no-op", pullWorkflowMsg{result: PullWorkflowResult{OperationRequestID: 7, OperationEpoch: 3, RefreshErrorKind: ReadErrorNone, Execute: PullExecutionResult{Mode: PullModeNoOp, Succeeded: true}}}, pullLifecycleNoOpCompleted},
			{"failed no-op", pullWorkflowMsg{result: PullWorkflowResult{OperationRequestID: 7, OperationEpoch: 3, RefreshErrorKind: ReadErrorNone, Execute: PullExecutionResult{Mode: PullModeNoOp, Succeeded: false}}}, pullLifecycleExecutionFailed},
			{"success", pullWorkflowMsg{result: PullWorkflowResult{OperationRequestID: 7, OperationEpoch: 3, RefreshRequestID: 8, RefreshEpoch: 4, Refresh: ReadSnapshotResult{RequestID: 8, RepositoryEpoch: 4}, RefreshErrorKind: ReadErrorNone, Execute: PullExecutionResult{Mode: PullModeMerge, Succeeded: true}}}, pullLifecycleCompleted},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if got := classifyPullWorkflowOutcome(request, tc.msg); got.Kind != tc.want {
					t.Fatalf("kind=%v want=%v", got.Kind, tc.want)
				}
			})
		}
	})
}

func TestTopLevelUpdateIgnoresGuardedValidationExecutionAndWorkflowMismatches(t *testing.T) {
	baseline := PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "head"}
	before := model{
		repositoryState: repositoryState{repoStatus: git.Status{Root: "/repo", Branch: "main"}},
		pullState:       pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3, OperationBaseline: baseline}}, status: state.New().WithLoading("Pulling...")}
	for name, msg := range map[string]any{
		"validation request": pullValidationMsg{requestID: 8, requestEpoch: 3, baseline: baseline, valid: true},
		"validation epoch":   pullValidationMsg{requestID: 7, requestEpoch: 4, baseline: baseline, valid: true},
		"execution request":  pullExecutionResultMsg{requestID: 8, requestEpoch: 3, baseline: baseline},
		"workflow request":   pullWorkflowMsg{result: PullWorkflowResult{OperationRequestID: 8, OperationEpoch: 3}},
		"workflow epoch":     pullWorkflowMsg{result: PullWorkflowResult{OperationRequestID: 7, OperationEpoch: 4}},
	} {
		t.Run(name, func(t *testing.T) {
			got, cmd := before.Update(msg)
			if cmd != nil || !reflect.DeepEqual(got.(model), before) {
				t.Fatalf("mismatch mutated model or emitted command: got=%#v cmd=%v", got, cmd)
			}
		})
	}
}

func TestTopLevelUpdateMatchingValidationRejectionPreservesLegacyGuidance(t *testing.T) {
	baseline := PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "head"}
	m := model{
		pullState: pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3, OperationBaseline: baseline}}, status: state.New().WithLoading("Pulling...")}
	gotModel, cmd := m.Update(pullValidationMsg{requestID: 7, requestEpoch: 3, baseline: baseline, valid: false, err: errors.New("rejected")})
	got := gotModel.(model)
	if cmd == nil || got.activePullRequest != nil || got.status.Block != state.BlockStaleSnapshot || got.status.Message != "Repository changed." || got.status.Detail != "Refresh before pulling again." {
		t.Fatalf("unexpected validation rejection: status=%+v active=%#v cmd=%v", got.status, got.activePullRequest, cmd)
	}
}

func TestTopLevelUpdateRoutesValidValidationToExecuteValidatedPull(t *testing.T) {
	baseline := PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "head"}
	m := model{
		pullState: pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3, OperationBaseline: baseline, OperationBaselineSet: true}}, status: state.New().WithLoading("Validating...")}
	next, cmd := m.Update(pullValidationMsg{requestID: 7, requestEpoch: 3, baseline: baseline, operationBaseline: baseline, operationBaselineSet: true, valid: true, mode: PullModeMerge})
	got := next.(model)
	if cmd == nil || got.status.Message != "Pulling..." || got.activePullRequest == nil {
		t.Fatalf("valid validation did not route to executeValidatedPull: status=%+v active=%#v cmd=%v", got.status, got.activePullRequest, cmd)
	}
}

func TestTopLevelUpdateReportsExecutionFailure(t *testing.T) {
	baseline := PullSnapshotIdentity{Epoch: 3, Branch: "main", Head: "head"}
	m := model{
		pullState: pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3, OperationBaseline: baseline, OperationBaselineSet: true}}}
	next, cmd := m.Update(pullExecutionResultMsg{requestID: 7, requestEpoch: 3, baseline: baseline, operationBaseline: baseline, operationBaselineSet: true, executionErr: errors.New("pull failed")})
	got := next.(model)
	if cmd != nil || got.status.Block != state.BlockUnknown || got.status.Message != "PULL FAILED" {
		t.Fatalf("unexpected execution failure: status=%+v cmd=%v", got.status, cmd)
	}
}

func TestTopLevelUpdateReportsRefreshFailure(t *testing.T) {
	m := model{
		pullState: pullState{activePullRequest: &pullRequest{ID: 7, Epoch: 3}}}
	next, cmd := m.Update(pullWorkflowMsg{result: PullWorkflowResult{OperationRequestID: 7, OperationEpoch: 3, RefreshErrorKind: ReadErrorRepository}})
	got := next.(model)
	if cmd != nil || got.status.Title != "PULL COMPLETED — STATE UNVERIFIED" || got.status.Detail != "Refresh failed. Press f to refresh repository state." {
		t.Fatalf("unexpected refresh failure: status=%+v cmd=%v", got.status, cmd)
	}
}
