package gitread

import (
	"context"
	"errors"

	"hrllk/graphkeeper/internal/app"
	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
)

type Adapter struct{ repo *git.Repo }

func New(repo *git.Repo) app.RepositoryReadPort { return &Adapter{repo: repo} }

func (a *Adapter) ReadSnapshot(ctx context.Context, req app.ReadRequest) (app.ReadSnapshotResult, error) {
	result := app.ReadSnapshotResult{RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, ErrorKind: app.ReadErrorNone}
	if a == nil || a.repo == nil {
		result.ErrorKind = app.ReadErrorRepository
		err := &app.ReadError{Kind: app.ReadErrorRepository}
		result.Err = err
		return result, err
	}
	status, err := a.repo.Status(ctx, req.CommitLimit)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			result.ErrorKind = app.ReadErrorCanceled
			result.Canceled = true
			result.Err = err
			return result, err
		}
		result.ErrorKind = app.ReadErrorRepository
		result.Err = err
		return result, err
	}
	if status.Root == "" {
		result.ErrorKind = app.ReadErrorInvalid
		return result, nil
	}
	result.Snapshot = MapStatusToSnapshot(status, req.RepositoryEpoch)
	return result, nil
}

// MapStatusToSnapshot is the sole Git-to-graph projection for repository reads.
func MapStatusToSnapshot(status git.Status, epoch uint64) app.ReadSnapshot {
	commits := make([]graph.Commit, 0, len(status.GraphCommits))
	for _, commit := range status.GraphCommits {
		commits = append(commits, graph.Commit{
			Graph: commit.Graph, Hash: commit.Hash, Parents: append([]string(nil), commit.Parents...),
			RelativeAge: commit.RelativeAge, Author: commit.Author,
			Decorations: append([]string(nil), commit.Decorations...), Subject: commit.Subject,
			Tags: append([]string(nil), commit.Tags...),
		})
	}
	return app.ReadSnapshot{
		Root: status.Root,
		Repository: app.RepositoryProjection{
			Root: status.Root, Branch: status.Branch, Head: status.Head, Upstream: status.Upstream,
			UpstreamOID: status.UpstreamOID, Remote: status.Remote, DefaultBranch: status.DefaultBranch,
			Branches: append([]string(nil), status.Branches...), LocalBranches: append([]string(nil), status.LocalBranches...),
			LocalBranchesKnown: status.LocalBranchesKnown, LocalBranchesFresh: status.LocalBranchesFresh, LocalBranchesError: status.LocalBranchesError,
			RemoteBranches: append([]string(nil), status.RemoteBranches...), BranchUpstreams: copyStringMap(status.BranchUpstreams),
			Tracking: mapTracking(status.Tracking), TrackingKnown: status.TrackingKnown, TrackingFresh: status.TrackingFresh,
			WorktreeDirty: status.WorktreeDirty, Detached: status.Detached, EmptyRepo: status.EmptyRepo,
			NoRemote: status.NoRemote, NoUpstream: status.NoUpstream, UpstreamGone: status.UpstreamGone,
			MergeInProgress: status.MergeInProgress, RebaseInProgress: status.RebaseInProgress, CherryPickInProgress: status.CherryPickInProgress,
			ConflictTarget: status.ConflictTarget, ConflictTargetSubject: status.ConflictTargetSubject,
			LastFetchAt: status.LastFetchAt, RemoteSyncSummary: status.RemoteSyncSummary,
		},
		Graph: graph.Snapshot{
			Commits: commits, Branch: status.Branch, Head: status.Head,
			LocalBranches: append([]string(nil), status.LocalBranches...),
			Conflict: graph.ConflictState{
				Active:          status.MergeInProgress || status.RebaseInProgress,
				MergeInProgress: status.MergeInProgress, RebaseInProgress: status.RebaseInProgress,
				Head: status.Head, Target: status.ConflictTarget,
			},
		},
		Freshness: snapshotIdentity(status, epoch),
	}
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mapTracking(in map[string]git.BranchTracking) map[string]app.BranchTracking {
	if in == nil {
		return nil
	}
	out := make(map[string]app.BranchTracking, len(in))
	for k, v := range in {
		out[k] = app.BranchTracking{Ahead: v.Ahead, Behind: v.Behind}
	}
	return out
}

func snapshotIdentity(status git.Status, epoch uint64) app.PullSnapshotIdentity {
	tracking, known := status.Tracking[status.Branch]
	return app.PullSnapshotIdentity{
		Epoch: epoch, Branch: status.Branch, Head: status.Head, Upstream: status.Upstream, UpstreamOID: status.UpstreamOID,
		Ahead: tracking.Ahead, Behind: tracking.Behind,
		TrackingKnown: status.TrackingKnown && known, TrackingFresh: status.TrackingFresh,
		WorktreeDirty: status.WorktreeDirty, Detached: status.Detached, EmptyRepo: status.EmptyRepo,
		NoRemote: status.NoRemote, NoUpstream: status.NoUpstream, UpstreamGone: status.UpstreamGone,
		MergeInProgress: status.MergeInProgress, RebaseInProgress: status.RebaseInProgress,
		CherryPickInProgress: status.CherryPickInProgress,
	}
}
