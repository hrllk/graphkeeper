package app

import (
	"errors"
	"testing"

	"hrllk/graphkeeper/internal/state"
)

func TestOperationResultRefreshFailureMarksVerificationUnknown(t *testing.T) {
	m := model{
		pullState: pullState{
			operationResult: &OperationResultSummary{
				Execution:    ExecutionSucceeded,
				Verification: VerificationVerified,
				Headline:     "PULL COMPLETED",
			},
		},
		status: state.Status{Mode: state.ModeOperationResult, Action: state.ActionPull}}
	next, _ := handleLifecycleUpdate(m, refreshedMsg{err: errors.New("status unavailable")})
	got := next.(model)
	if got.operationResult.Verification != VerificationUnknown {
		t.Fatalf("verification = %q, want unknown", got.operationResult.Verification)
	}
	if got.status.Message != "PULL COMPLETED — STATE UNVERIFIED" {
		t.Fatalf("message = %q", got.status.Message)
	}
}
