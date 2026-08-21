package gitpull

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"hrllk/graphkeeper/internal/app"
	"hrllk/graphkeeper/internal/git"
)

type Adapter struct{ repo *git.Repo }

func New(repo *git.Repo) app.PullPort { return &Adapter{repo: repo} }

func (a *Adapter) Preview(ctx context.Context, req app.PullPreviewRequest) (app.PullPreviewResult, error) {
	result := app.PullPreviewResult{RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Mode: req.Mode, Reason: app.PullRejectNone}
	if err := a.check(); err != nil {
		return result, err
	}
	if err := a.repo.FetchContext(ctx); err != nil {
		return result, pullError(err)
	}
	status, err := a.repo.Status(ctx, req.CommitLimit)
	if err != nil {
		return result, pullError(err)
	}
	result.Baseline = identity(status, req.RepositoryEpoch)
	if reason := eligibility(result.Baseline, req.Mode); reason != app.PullRejectNone {
		result.Reason = reason
		return result, nil
	}
	ff, known, err := fastForward(ctx, a.repo)
	if err != nil {
		return result, pullError(err)
	}
	result.Impact = app.PullImpact{FastForwardKnown: known, IsFastForward: ff}
	if !known {
		result.Reason = app.PullRejectNotEligible
		return result, nil
	}
	result.Eligible = true
	result.Impact.Summary = "Pull upstream changes."
	if ff {
		result.Impact.Summary = "Fast-forward to upstream."
		result.Impact.Risk = "low"
	} else {
		result.Impact.Risk = "high"
	}
	ref := "HEAD...@{upstream}"
	if ff {
		ref = "HEAD..@{upstream}"
	}
	out, err := a.repo.RunContext(ctx, "rev-list", ref)
	if err != nil {
		return result, pullError(err)
	}
	for _, line := range strings.Split(out, "\n") {
		if h := strings.TrimSpace(line); h != "" {
			result.Commits = append(result.Commits, app.PullPreviewCommit{Hash: h})
		}
	}
	if ff {
		if head, err := a.repo.RunContext(ctx, "rev-parse", "HEAD"); err == nil && strings.TrimSpace(head) != "" {
			result.Commits = append(result.Commits, app.PullPreviewCommit{Hash: strings.TrimSpace(head)})
		}
	}
	return result, nil
}

func (a *Adapter) Validate(_ context.Context, req app.PullValidationRequest) (app.PullValidationResult, error) {
	result := app.PullValidationResult{RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch}
	if req.RepositoryEpoch != req.Expected.Epoch || req.Current.Epoch != req.Expected.Epoch {
		result.Reason = app.PullRejectStaleEpoch
		return result, nil
	}
	if !same(req.Current, req.Expected) {
		result.Reason = app.PullRejectChangedBaseline
		return result, nil
	}
	if reason := eligibility(req.Current, req.Mode); reason != app.PullRejectNone {
		result.Reason = reason
		return result, nil
	}
	result.Valid, result.Authorized, result.AuthorizedBaseline = true, true, req.Expected
	return result, nil
}

func (a *Adapter) Execute(ctx context.Context, req app.PullExecutionRequest) (app.PullExecutionResult, error) {
	result := app.PullExecutionResult{RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Mode: req.Mode, ErrorKind: app.PullErrorNone}
	if !req.Authorized {
		result.Reason = app.PullRejectNotEligible
		return result, nil
	}
	if req.Mode != app.PullModeMerge && req.Mode != app.PullModeRebase {
		result.Reason = app.PullRejectNotEligible
		return result, nil
	}
	if err := a.check(); err != nil {
		return result, err
	}
	status, err := a.repo.Status(ctx, 0)
	if err != nil {
		return result, pullError(err)
	}
	current := identity(status, req.RepositoryEpoch)
	if !same(current, req.AuthorizedBaseline) {
		result.Reason = app.PullRejectChangedBaseline
		return result, nil
	}
	if reason := eligibility(current, req.Mode); reason != app.PullRejectNone {
		result.Reason = reason
		return result, nil
	}
	args := []string{"pull", "--no-rebase", "--no-edit"}
	if req.Mode == app.PullModeRebase {
		args = []string{"pull", "--rebase"}
	}
	if _, err := a.repo.RunContext(ctx, args...); err != nil {
		if errors.Is(err, context.Canceled) {
			result.ErrorKind = app.PullErrorCanceled
			return result, nil
		}
		result.ErrorKind = app.PullErrorGit
		return result, nil
	}
	result.Succeeded = true
	return result, nil
}

func (a *Adapter) check() error {
	if a == nil || a.repo == nil {
		return fmt.Errorf("pull adapter is not configured")
	}
	return nil
}
func pullError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	return err
}
func same(a, b app.PullSnapshotIdentity) bool { return a == b }

func eligibility(s app.PullSnapshotIdentity, mode app.PullMode) app.PullRejectReason {
	if mode != app.PullModeMerge && mode != app.PullModeRebase && mode != app.PullModeNoOp {
		return app.PullRejectInvalidTarget
	}
	if s.Branch == "" || s.Head == "" || s.Upstream == "" || !s.TrackingKnown || !s.TrackingFresh || s.NoRemote || s.NoUpstream || s.UpstreamGone || s.EmptyRepo || s.Detached {
		return app.PullRejectNotEligible
	}
	if s.WorktreeDirty || s.MergeInProgress || s.RebaseInProgress || s.CherryPickInProgress {
		return app.PullRejectBlockedState
	}
	return app.PullRejectNone
}

func fastForward(ctx context.Context, repo *git.Repo) (bool, bool, error) {
	_, err := repo.RunContext(ctx, "merge-base", "--is-ancestor", "HEAD", "@{upstream}")
	if err == nil {
		return true, true, nil
	}
	if strings.Contains(err.Error(), "exit status 1") {
		return false, true, nil
	}
	return false, false, err
}

func identity(status git.Status, epoch uint64) app.PullSnapshotIdentity {
	tracking, known := status.Tracking[status.Branch]
	return app.PullSnapshotIdentity{Epoch: epoch, Branch: status.Branch, Head: status.Head, Upstream: status.Upstream, UpstreamOID: status.UpstreamOID, Ahead: tracking.Ahead, Behind: tracking.Behind, TrackingKnown: status.TrackingKnown && known, TrackingFresh: status.TrackingFresh, WorktreeDirty: status.WorktreeDirty, Detached: status.Detached, EmptyRepo: status.EmptyRepo, NoRemote: status.NoRemote, NoUpstream: status.NoUpstream, UpstreamGone: status.UpstreamGone, MergeInProgress: status.MergeInProgress, RebaseInProgress: status.RebaseInProgress, CherryPickInProgress: status.CherryPickInProgress}
}
