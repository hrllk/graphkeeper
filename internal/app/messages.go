package app

import (
	"time"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

type loadedMsg struct {
	status   git.Status
	err      error
	epoch    uint64
	epochSet bool
}

type tickMsg time.Time

type stashLoadedMsg struct {
	entries []git.StashEntry
	err     error
}

type refreshedMsg struct {
	status            git.Status
	err               error
	epoch             uint64
	epochSet          bool
	refreshGeneration uint64
	generationSet     bool
	purpose           RefreshPurpose
	operationRequest  uint64
	operationEpoch    uint64
}

type fetchedMsg struct {
	status git.Status
	err    error
}

type preparedMsg struct {
	action state.Action
	status git.Status
	err    error
}

type pullCheckedMsg struct {
	repo   git.Status
	status state.Status
	err    error
}

type previewMsg struct {
	action state.Action
	target string
	repo   git.Status
	status state.Status
	err    error
}

type graphActionCheckMsg struct {
	action      state.Action
	target      string
	repo        git.Status
	base        string
	currentOnly int
	targetOnly  int
	err         error
}

type executedMsg struct {
	action    state.Action
	target    string
	resetMode state.ResetMode
	status    git.Status
	err       error
}

type createdBranchMsg struct {
	name   string
	base   string
	status git.Status
	err    error
}

type pullFetchedMsg struct {
	status                  git.Status
	err                     error
	requestID, requestEpoch uint64
	baseline                PullSnapshotIdentity
	fetchBaseline           PullSnapshotIdentity
	operationBaseline       PullSnapshotIdentity
	operationBaselineSet    bool
	snapshot                PullImpactSnapshot
}

type pushFetchedMsg struct {
	status git.Status
	err    error
}

type pullPreviewReadyMsg struct {
	commits                 []string
	isFF                    bool
	err                     error
	requestID, requestEpoch uint64
	baseline                PullSnapshotIdentity
	snapshot                PullImpactSnapshot
	impact                  PullImpactSet
}

type pullValidationMsg struct {
	requestID, requestEpoch uint64
	baseline                PullSnapshotIdentity
	operationBaseline       PullSnapshotIdentity
	operationBaselineSet    bool
	mode                    PullMode
	status                  git.Status
	valid                   bool
	err                     error
}

type pullExecutionResultMsg struct {
	action                  state.Action
	status                  git.Status
	err                     error
	requestID, requestEpoch uint64
	baseline                PullSnapshotIdentity
	operationBaseline       PullSnapshotIdentity
	operationBaselineSet    bool
	mode                    PullMode
	stale                   bool
	executionErr            error
	refreshErr              error
}

type pullToastDoneMsg struct {
	requestID, requestEpoch uint64
}

type branchToastDoneMsg struct{}

type commitInspectorLoadedMsg struct {
	inspection git.CommitInspection
	err        error
	request    uint64
	epoch      uint64
}

type commitInspectorDiffMsg struct {
	diff    git.CommitDiff
	err     error
	request uint64
	epoch   uint64
}
