package app

import "fmt"

type BranchRelation string

const (
	RelationFastForward    BranchRelation = "fast-forward"
	RelationAlreadyAligned BranchRelation = "already-aligned"
	RelationTargetIncluded BranchRelation = "target-included"
	RelationDiverged       BranchRelation = "diverged"
	RelationUnavailable    BranchRelation = "unavailable"
)

type BranchDecisionContext struct {
	CurrentRef  string
	TargetRef   string
	CurrentOnly int
	TargetOnly  int
	Relation    BranchRelation
	ReasonLines []string
	ActionLines []string
}

func explainBranchDecision(snapshot DivergedSnapshot) (BranchDecisionContext, bool) {
	if !validDecisionSnapshot(snapshot) {
		return BranchDecisionContext{}, false
	}

	context := BranchDecisionContext{
		CurrentRef:  snapshot.Branch,
		TargetRef:   snapshot.Upstream,
		CurrentOnly: snapshot.LocalOnly,
		TargetOnly:  snapshot.UpstreamOnly,
	}
	switch {
	case snapshot.LocalOnly == 0 && snapshot.UpstreamOnly > 0:
		context.Relation = RelationFastForward
		context.ReasonLines = []string{"upstream 변경을 fast-forward로 반영할 수 있습니다."}
		context.ActionLines = []string{"p: pull로 target 변경을 반영합니다."}
	case snapshot.LocalOnly == 0 && snapshot.UpstreamOnly == 0:
		context.Relation = RelationAlreadyAligned
		context.ReasonLines = []string{"현재 branch와 target이 이미 동일합니다."}
		context.ActionLines = []string{"추가 pull 불필요"}
	case snapshot.LocalOnly > 0 && snapshot.UpstreamOnly == 0:
		context.Relation = RelationTargetIncluded
		context.ReasonLines = []string{"현재 branch가 target 변경을 포함합니다. 추가 동기화가 필요하지 않습니다."}
		context.ActionLines = []string{"추가 pull 불필요"}
	default:
		context.Relation = RelationDiverged
		context.ReasonLines = []string{"양쪽에 고유 commit이 있어 merge 또는 rebase로 통합합니다."}
	}
	return context, true
}

func validDecisionSnapshot(snapshot DivergedSnapshot) bool {
	return snapshot.Branch != "" && snapshot.Upstream != "" && snapshot.Head != "" &&
		snapshot.LocalOnly >= 0 && snapshot.UpstreamOnly >= 0 &&
		snapshot.TrackingKnown && snapshot.TrackingFresh &&
		!snapshot.WorktreeDirty && !snapshot.Detached && !snapshot.EmptyRepo &&
		!snapshot.NoRemote && !snapshot.NoUpstream && !snapshot.UpstreamGone &&
		!snapshot.MergeInProgress && !snapshot.RebaseInProgress && !snapshot.CherryPickInProgress
}

func decisionSummaryLine(context BranchDecisionContext) string {
	return fmt.Sprintf("%s → %s · local-only %d · target-only %d", context.CurrentRef, context.TargetRef, context.CurrentOnly, context.TargetOnly)
}
