package app

import (
	"errors"
	"testing"

	"hrllk/graphkeeper/internal/state"
)

func TestOperationResultRefreshFailureMarksVerificationUnknown(t *testing.T) {
	m := model{
		status: state.Status{Mode: state.ModeOperationResult, Action: state.ActionPull},
		operationResult: &OperationResultSummary{
			Execution:    ExecutionSucceeded,
			Verification: VerificationVerified,
			Headline:     "PULL COMPLETED",
		},
	}
	next, _ := handleLifecycleUpdate(m, refreshedMsg{err: errors.New("status unavailable")})
	got := next.(model)
	if got.operationResult.Verification != VerificationUnknown {
		t.Fatalf("verification = %q, want unknown", got.operationResult.Verification)
	}
	if got.status.Message != "PULL COMPLETED — STATE UNVERIFIED" {
		t.Fatalf("message = %q", got.status.Message)
	}
}
