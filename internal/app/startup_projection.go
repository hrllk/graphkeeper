package app

import (
	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

// applyRepositoryProjection is the sole neutral-startup owner of repoStatus.
func applyRepositoryProjection(p RepositoryProjection, snapshot graph.Snapshot) git.Status {
	status := git.Status{
		Root: p.Root, Branch: p.Branch, Head: p.Head, Upstream: p.Upstream, UpstreamOID: p.UpstreamOID,
		Remote: p.Remote, DefaultBranch: p.DefaultBranch,
		Branches: append([]string(nil), p.Branches...), LocalBranches: append([]string(nil), p.LocalBranches...),
		LocalBranchesKnown: p.LocalBranchesKnown, LocalBranchesFresh: p.LocalBranchesFresh, LocalBranchesError: p.LocalBranchesError,
		RemoteBranches: append([]string(nil), p.RemoteBranches...), BranchUpstreams: cloneStrings(p.BranchUpstreams),
		Tracking: cloneTracking(p.Tracking), TrackingKnown: p.TrackingKnown, TrackingFresh: p.TrackingFresh,
		WorktreeDirty: p.WorktreeDirty, Detached: p.Detached, EmptyRepo: p.EmptyRepo,
		NoRemote: p.NoRemote, NoUpstream: p.NoUpstream, UpstreamGone: p.UpstreamGone,
		MergeInProgress: p.MergeInProgress, RebaseInProgress: p.RebaseInProgress, CherryPickInProgress: p.CherryPickInProgress,
		ConflictTarget: p.ConflictTarget, ConflictTargetSubject: p.ConflictTargetSubject,
		LastFetchAt: p.LastFetchAt, RemoteSyncSummary: p.RemoteSyncSummary,
		GraphCommits: make([]git.GraphCommit, 0, len(snapshot.Commits)),
	}
	for _, commit := range snapshot.Commits {
		status.GraphCommits = append(status.GraphCommits, git.GraphCommit{
			Graph: commit.Graph, Hash: commit.Hash, Parents: append([]string(nil), commit.Parents...),
			RelativeAge: commit.RelativeAge, Author: commit.Author, Decorations: append([]string(nil), commit.Decorations...),
			Subject: commit.Subject, Tags: append([]string(nil), commit.Tags...),
		})
	}
	return status
}

func cloneStrings(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneTracking(in map[string]BranchTracking) map[string]git.BranchTracking {
	if in == nil {
		return nil
	}
	out := make(map[string]git.BranchTracking, len(in))
	for k, v := range in {
		out[k] = git.BranchTracking{Ahead: v.Ahead, Behind: v.Behind}
	}
	return out
}

func startupReadError(result ReadSnapshotResult) error {
	if result.Err != nil {
		return result.Err
	}
	return &ReadError{Kind: result.ErrorKind}
}

func startupErrorStatus(result ReadSnapshotResult) state.Status {
	switch result.ErrorKind {
	case ReadErrorInvalid:
		return state.New().WithBlocked(state.BlockNoRepo, "Not inside a Git repository.", "Run this tool from a repo root.")
	case ReadErrorCanceled:
		return state.New().WithBlocked(state.BlockUnknown, "Repository load canceled.", "Press q to quit or retry from the application entry point.")
	default:
		detail := "Repository status could not be read."
		if result.Err != nil && result.Err.Error() != "" {
			detail = result.Err.Error()
		}
		return state.New().WithBlocked(state.BlockUnknown, "Repository read failed.", detail)
	}
}
