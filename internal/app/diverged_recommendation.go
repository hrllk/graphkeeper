package app

import "hrllk/graphkeeper/internal/git"

type PullMode string

const (
	PullModeMerge  PullMode = "merge"
	PullModeRebase PullMode = "rebase"
)

type TrackingState string

const (
	TrackingFresh   TrackingState = "fresh"
	TrackingUnknown TrackingState = "unknown"
	TrackingStale   TrackingState = "stale"
)

type DivergedSnapshot struct {
	Branch, Upstream, Head                                                 string
	LocalOnly, UpstreamOnly                                                int
	TrackingKnown, TrackingFresh                                           bool
	WorktreeDirty, Detached, EmptyRepo, NoRemote, NoUpstream, UpstreamGone bool
	MergeInProgress, RebaseInProgress, CherryPickInProgress                bool
	Epoch                                                                  uint64
}

type DivergedRecommendation struct {
	Branch, Upstream, Head  string
	LocalOnly, UpstreamOnly int
	SnapshotEpoch           uint64
	Tracking                TrackingState
	PullModes               []PullMode
}

func recommendDivergedPull(s DivergedSnapshot) (DivergedRecommendation, bool) {
	if s.Branch == "" || s.Upstream == "" || s.Head == "" || s.LocalOnly <= 0 || s.UpstreamOnly <= 0 ||
		!s.TrackingKnown || !s.TrackingFresh || s.WorktreeDirty || s.Detached || s.EmptyRepo || s.NoRemote || s.NoUpstream || s.UpstreamGone ||
		s.MergeInProgress || s.RebaseInProgress || s.CherryPickInProgress {
		return DivergedRecommendation{}, false
	}
	return DivergedRecommendation{
		Branch: s.Branch, Upstream: s.Upstream, Head: s.Head,
		LocalOnly: s.LocalOnly, UpstreamOnly: s.UpstreamOnly, SnapshotEpoch: s.Epoch,
		Tracking: TrackingFresh, PullModes: []PullMode{PullModeMerge, PullModeRebase},
	}, true
}

func divergedSnapshotFromStatus(status git.Status, epoch uint64) DivergedSnapshot {
	tracking, known := status.Tracking[status.Branch]
	return DivergedSnapshot{
		Branch: status.Branch, Upstream: status.Upstream, Head: status.Head,
		LocalOnly: tracking.Ahead, UpstreamOnly: tracking.Behind,
		TrackingKnown: status.TrackingKnown && known,
		TrackingFresh: status.TrackingFresh,
		WorktreeDirty: status.WorktreeDirty, Detached: status.Detached, EmptyRepo: status.EmptyRepo,
		NoRemote: status.NoRemote, NoUpstream: status.NoUpstream, UpstreamGone: status.UpstreamGone,
		MergeInProgress: status.MergeInProgress, RebaseInProgress: status.RebaseInProgress,
		CherryPickInProgress: status.CherryPickInProgress, Epoch: epoch,
	}
}

type PullSnapshotIdentity struct {
	Epoch                                                                  uint64
	Branch, Head, Upstream, UpstreamOID                                    string
	Ahead, Behind                                                          int
	TrackingKnown, TrackingFresh                                           bool
	WorktreeDirty, Detached, EmptyRepo, NoRemote, NoUpstream, UpstreamGone bool
	MergeInProgress, RebaseInProgress, CherryPickInProgress                bool
}

func pullSnapshotIdentity(status git.Status, epoch uint64) PullSnapshotIdentity {
	tracking, known := status.Tracking[status.Branch]
	return PullSnapshotIdentity{Epoch: epoch, Branch: status.Branch, Head: status.Head, Upstream: status.Upstream, UpstreamOID: status.UpstreamOID,
		Ahead: tracking.Ahead, Behind: tracking.Behind, TrackingKnown: status.TrackingKnown && known,
		TrackingFresh: status.TrackingFresh, WorktreeDirty: status.WorktreeDirty, Detached: status.Detached,
		EmptyRepo: status.EmptyRepo, NoRemote: status.NoRemote, NoUpstream: status.NoUpstream, UpstreamGone: status.UpstreamGone,
		MergeInProgress: status.MergeInProgress, RebaseInProgress: status.RebaseInProgress,
		CherryPickInProgress: status.CherryPickInProgress}
}

func samePullSnapshotIdentity(a, b PullSnapshotIdentity) bool { return a == b }

type pullRequest struct {
	ID, Epoch            uint64
	FetchBaseline        PullSnapshotIdentity
	OperationBaseline    PullSnapshotIdentity
	OperationBaselineSet bool
}
