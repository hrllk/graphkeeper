package app

import (
	"context"
	"fmt"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

// repository is the smallest contract a workflow needs. It prevents new
// workflow code from depending on every method of the concrete Git facade.
type repository interface {
	Snapshot(context.Context, int) (repositorySnapshot, error)
	ValidateTarget(context.Context, targetRef) error
	Execute(context.Context, operation) (commandResult, error)
	Reload(context.Context, int) (repositorySnapshot, error)
}

type targetRef struct {
	Kind state.TargetKind
	Name string
	Hash string
}

type operation struct {
	Action state.Action
	Target targetRef
	Args   []string
}

type commandResult struct {
	Output string
	Args   []string
}

type gitRepositoryAdapter struct{ repo *git.Repo }

func newGitRepositoryAdapter(repo *git.Repo) repository {
	return gitRepositoryAdapter{repo: repo}
}

func (a gitRepositoryAdapter) Snapshot(ctx context.Context, limit int) (repositorySnapshot, error) {
	status, err := a.repo.Status(ctx, limit)
	return repositorySnapshot{Status: status, Valid: snapshotValidity{
		Graph:    status.ErrorMessage == "" || len(status.GraphCommits) > 0,
		Branches: status.LocalBranches != nil,
		Tags:     status.TagEntriesLoaded,
		Worktree: status.Root != "",
	}}, err
}

func (a gitRepositoryAdapter) Reload(ctx context.Context, limit int) (repositorySnapshot, error) {
	return a.Snapshot(ctx, limit)
}

func (a gitRepositoryAdapter) ValidateTarget(ctx context.Context, target targetRef) error {
	if target.Name == "" && target.Hash == "" {
		return fmt.Errorf("target is empty")
	}
	value := target.Hash
	if value == "" {
		value = target.Name
	}
	_, err := a.repo.Run("rev-parse", "--verify", value+"^{commit}")
	return err
}

func (a gitRepositoryAdapter) Execute(_ context.Context, op operation) (commandResult, error) {
	if len(op.Args) == 0 {
		return commandResult{}, fmt.Errorf("operation %s has no Git arguments", op.Action)
	}
	out, err := a.repo.Run(op.Args...)
	return commandResult{Output: out, Args: append([]string(nil), op.Args...)}, err
}
