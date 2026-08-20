package app

import (
	"fmt"
	"strings"
	"unicode"
)

type operationImpactValidity struct {
	TargetKnown          bool
	SnapshotFresh        bool
	FastForwardKnown     bool
	HeadOIDValid         bool
	UpstreamOIDValid     bool
	WorktreeDirty        bool
	Detached             bool
	EmptyRepo            bool
	NoRemote             bool
	NoUpstream           bool
	UpstreamGone         bool
	MergeInProgress      bool
	RebaseInProgress     bool
	CherryPickInProgress bool
}

type PullImpactSnapshot struct {
	CurrentRef, HeadOID, UpstreamRef, UpstreamOID string
	Ahead, Behind                                 int
	IsFastForward, FastForwardKnown               bool
	Validity                                      operationImpactValidity
	FetchBaseline, OperationBaseline              PullSnapshotIdentity
	RequestID, RequestEpoch                       uint64
}

type PullImpactSet struct {
	MergeSummary, MergeRisk   string
	RebaseSummary, RebaseRisk string
	Valid                     bool
}

func validObjectID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !unicode.Is(unicode.ASCII_Hex_Digit, r) {
			return false
		}
	}
	return true
}

func operationImpact(input operationImpactInput) (summary, risk string, ok bool) {
	v := input.Validity
	if !v.TargetKnown || !v.SnapshotFresh || !v.HeadOIDValid || !v.UpstreamOIDValid || v.WorktreeDirty || v.Detached || v.EmptyRepo || v.NoRemote || v.NoUpstream || v.UpstreamGone || v.MergeInProgress || v.RebaseInProgress || v.CherryPickInProgress {
		return "", "", false
	}
	if input.Ahead < 0 || input.Behind < 0 {
		return "", "", false
	}
	if input.Behind == 0 {
		return "", "", false
	}
	if !v.FastForwardKnown {
		return "", "", false
	}
	fastForward := input.Ahead == 0 && input.Behind > 0
	if input.IsFastForward != fastForward {
		return "", "", false
	}
	target := input.UpstreamRef
	if target == "" {
		target = "upstream"
	}
	if fastForward {
		return fmt.Sprintf("HEAD can move to %s. No merge commit is needed.", target), "", true
	}
	if input.Mode == PullModeRebase {
		return fmt.Sprintf("Local commits will be replayed onto %s. Commit identities may change.", target), "Conflicts may occur.", true
	}
	return "Histories will be combined. A merge commit may be created.", "Conflicts may occur.", true
}

type operationImpactInput struct {
	Mode                            PullMode
	CurrentRef, UpstreamRef         string
	HeadOID, UpstreamOID            string
	Ahead, Behind                   int
	IsFastForward, FastForwardKnown bool
	Validity                        operationImpactValidity
}

func pullImpactSet(snapshot PullImpactSnapshot) PullImpactSet {
	if snapshot.Behind == 0 {
		return PullImpactSet{}
	}
	mergeSummary, mergeRisk, mergeOK := operationImpact(operationImpactInput{Mode: PullModeMerge, CurrentRef: snapshot.CurrentRef, UpstreamRef: snapshot.UpstreamRef, HeadOID: snapshot.HeadOID, UpstreamOID: snapshot.UpstreamOID, Ahead: snapshot.Ahead, Behind: snapshot.Behind, IsFastForward: snapshot.IsFastForward, FastForwardKnown: snapshot.FastForwardKnown, Validity: snapshot.Validity})
	rebaseSummary, rebaseRisk, rebaseOK := operationImpact(operationImpactInput{Mode: PullModeRebase, CurrentRef: snapshot.CurrentRef, UpstreamRef: snapshot.UpstreamRef, HeadOID: snapshot.HeadOID, UpstreamOID: snapshot.UpstreamOID, Ahead: snapshot.Ahead, Behind: snapshot.Behind, IsFastForward: snapshot.IsFastForward, FastForwardKnown: snapshot.FastForwardKnown, Validity: snapshot.Validity})
	if !mergeOK || !rebaseOK {
		return PullImpactSet{}
	}
	if snapshot.IsFastForward {
		rebaseSummary, rebaseRisk = "", ""
	}
	return PullImpactSet{MergeSummary: mergeSummary, MergeRisk: mergeRisk, RebaseSummary: rebaseSummary, RebaseRisk: rebaseRisk, Valid: true}
}

func pullImpactSnapshot(statusSnapshot PullSnapshotIdentity, request pullRequest) PullImpactSnapshot {
	fastForwardKnown := statusSnapshot.Ahead >= 0 && statusSnapshot.Behind >= 0 && statusSnapshot.Upstream != "" && statusSnapshot.UpstreamOID != "" && statusSnapshot.Head != "" && statusSnapshot.TrackingKnown && statusSnapshot.TrackingFresh
	isFastForward := statusSnapshot.Ahead == 0 && statusSnapshot.Behind > 0
	return PullImpactSnapshot{
		CurrentRef: statusSnapshot.Branch, HeadOID: statusSnapshot.Head, UpstreamRef: statusSnapshot.Upstream, UpstreamOID: statusSnapshot.UpstreamOID,
		Ahead: statusSnapshot.Ahead, Behind: statusSnapshot.Behind, IsFastForward: isFastForward, FastForwardKnown: fastForwardKnown,
		Validity:      operationImpactValidity{TargetKnown: statusSnapshot.Upstream != "" && validObjectID(statusSnapshot.Head) && validObjectID(statusSnapshot.UpstreamOID), SnapshotFresh: statusSnapshot.TrackingKnown && statusSnapshot.TrackingFresh, FastForwardKnown: fastForwardKnown, HeadOIDValid: validObjectID(statusSnapshot.Head), UpstreamOIDValid: validObjectID(statusSnapshot.UpstreamOID), WorktreeDirty: statusSnapshot.WorktreeDirty, Detached: statusSnapshot.Detached, EmptyRepo: statusSnapshot.EmptyRepo, NoRemote: statusSnapshot.NoRemote, NoUpstream: statusSnapshot.NoUpstream, MergeInProgress: statusSnapshot.MergeInProgress, RebaseInProgress: statusSnapshot.RebaseInProgress, CherryPickInProgress: statusSnapshot.CherryPickInProgress},
		FetchBaseline: request.FetchBaseline, OperationBaseline: statusSnapshot, RequestID: request.ID, RequestEpoch: request.Epoch,
	}
}

func formatPullImpact(set PullImpactSet) string {
	if !set.Valid {
		return ""
	}
	var b strings.Builder
	if set.MergeSummary != "" {
		b.WriteString("Merge:\n  ")
		b.WriteString(set.MergeSummary)
		if set.MergeRisk != "" {
			b.WriteString("\n  ")
			b.WriteString(set.MergeRisk)
		}
	}
	if set.RebaseSummary != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("Rebase:\n  ")
		b.WriteString(set.RebaseSummary)
		if set.RebaseRisk != "" {
			b.WriteString("\n  ")
			b.WriteString(set.RebaseRisk)
		}
	}
	return b.String()
}
