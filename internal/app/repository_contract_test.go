package app

import (
	"context"
	"errors"
	"testing"

	"hrllk/graphkeeper/internal/state"
)

type fakeRepository struct {
	snapshot    repositorySnapshot
	validation  []targetRef
	operations  []operation
	validateErr error
	reloadErr   error
}

func (f *fakeRepository) Snapshot(context.Context, int) (repositorySnapshot, error) {
	return f.snapshot, nil
}

func (f *fakeRepository) ValidateTarget(_ context.Context, target targetRef) error {
	f.validation = append(f.validation, target)
	return f.validateErr
}

func (f *fakeRepository) Execute(_ context.Context, op operation) (commandResult, error) {
	if len(f.validation) == 0 {
		return commandResult{}, errors.New("execute called before target validation")
	}
	if f.validation[len(f.validation)-1] != op.Target {
		return commandResult{}, errors.New("execute target differs from validated target")
	}
	f.operations = append(f.operations, op)
	return commandResult{Args: op.Args}, nil
}

func (f *fakeRepository) Reload(context.Context, int) (repositorySnapshot, error) {
	return f.snapshot, f.reloadErr
}

func TestFakeRepositoryRequiresTargetValidationBeforeExecute(t *testing.T) {
	fake := &fakeRepository{}
	if _, err := fake.Execute(context.Background(), operation{Action: state.ActionReset, Args: []string{"reset", "--hard", "HEAD"}}); err == nil {
		t.Fatal("expected execute-before-validation to fail")
	}
	if err := fake.ValidateTarget(context.Background(), targetRef{Kind: state.TargetKindCommit, Hash: "abc123"}); err != nil {
		t.Fatalf("ValidateTarget failed: %v", err)
	}
	if _, err := fake.Execute(context.Background(), operation{Action: state.ActionReset, Target: targetRef{Kind: state.TargetKindCommit, Hash: "abc123"}, Args: []string{"reset", "--hard", "abc123"}}); err != nil {
		t.Fatalf("Execute after validation failed: %v", err)
	}
	if len(fake.operations) != 1 || fake.operations[0].Action != state.ActionReset {
		t.Fatalf("unexpected operations: %+v", fake.operations)
	}
}

func TestFakeRepositoryPreservesPartialReloadError(t *testing.T) {
	fake := &fakeRepository{reloadErr: errors.New("status timeout")}
	if err := fake.ValidateTarget(context.Background(), targetRef{Kind: state.TargetKindCommit, Hash: "abc123"}); err != nil {
		t.Fatalf("ValidateTarget failed: %v", err)
	}
	if _, err := fake.Execute(context.Background(), operation{Action: state.ActionMerge, Target: targetRef{Kind: state.TargetKindCommit, Hash: "abc123"}, Args: []string{"merge", "feature"}}); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if _, err := fake.Reload(context.Background(), 40); err == nil {
		t.Fatal("expected reload error to remain observable")
	}
}

func TestFakeRepositoryRejectsDifferentExecuteTarget(t *testing.T) {
	fake := &fakeRepository{}
	if err := fake.ValidateTarget(context.Background(), targetRef{Kind: state.TargetKindCommit, Hash: "abc123"}); err != nil {
		t.Fatalf("ValidateTarget failed: %v", err)
	}
	if _, err := fake.Execute(context.Background(), operation{
		Action: state.ActionMerge,
		Target: targetRef{Kind: state.TargetKindCommit, Hash: "different"},
		Args:   []string{"merge", "different"},
	}); err == nil {
		t.Fatal("expected execute with a different target to fail")
	}
}
