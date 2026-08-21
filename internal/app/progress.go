package app

import "hrllk/graphkeeper/internal/state"

type progressOperation string

const (
	progressFetch               progressOperation = "fetch"
	progressPull                progressOperation = "pull"
	progressPush                progressOperation = "push"
	progressStash               progressOperation = "stash"
	progressClean               progressOperation = "clean"
	progressCheckout            progressOperation = "checkout"
	progressReset               progressOperation = "reset"
	progressMerge               progressOperation = "merge"
	progressRebase              progressOperation = "rebase"
	progressBranch              progressOperation = "branch"
	progressTag                 progressOperation = "tag"
	progressDeleteBranch        progressOperation = "delete-branch"
	progressDeleteTag           progressOperation = "delete-tag"
	progressDeleteRemoteTag     progressOperation = "delete-remote-tag"
	progressStashPop            progressOperation = "stash-pop"
	progressPushTag             progressOperation = "push-tag"
	progressFetchSources        progressOperation = "fetch-sources"
	progressFetchTags           progressOperation = "fetch-tags"
	progressPrepareReset        progressOperation = "prepare-reset"
	progressGraphTargetAnalysis progressOperation = "graph-target-analysis"
	progressCherryPick          progressOperation = "cherry-pick"
	progressAbort               progressOperation = "abort"
	progressOutcomeExecution    progressOperation = "outcome-execution"
	progressTagCommit           progressOperation = "tag-commit"
	progressTargetPreview       progressOperation = "target-preview"
)

type progressSpec struct {
	Operation progressOperation
	Message   string
	Detail    string
	Action    state.Action
	LockInput bool
}

func operationLoadingSpec(op progressOperation, message string, action state.Action) (progressSpec, bool) {
	if !knownProgressOperation(op) {
		return progressSpec{}, false
	}
	return progressSpec{
		Operation: op,
		Message:   message,
		Detail:    "Please wait.",
		Action:    action,
		LockInput: true,
	}, true
}

func operationLoadingStatus(spec progressSpec) state.Status {
	status := state.New().WithLoading(spec.Message)
	status.Action = spec.Action
	status.Detail = spec.Detail
	return status
}

func knownProgressOperation(op progressOperation) bool {
	switch op {
	case progressFetch, progressPull, progressPush, progressStash, progressClean,
		progressCheckout, progressReset, progressMerge, progressRebase, progressBranch,
		progressTag, progressDeleteBranch, progressDeleteTag, progressDeleteRemoteTag,
		progressStashPop, progressPushTag, progressFetchSources, progressFetchTags,
		progressPrepareReset, progressGraphTargetAnalysis, progressCherryPick, progressAbort,
		progressOutcomeExecution, progressTagCommit, progressTargetPreview:
		return true
	default:
		return false
	}
}

func operationInputLocked(m model) bool {
	if m.status.Mode != state.ModeLoading {
		return false
	}
	return !m.branchOpen && !m.tagPopupOpen && !m.stashMessageOpen && !m.stashPopupOpen && !m.graphStashPopOpen && !m.commitInspectorOpen
}

func operationLoadingStatusFor(op progressOperation, message string, action state.Action) state.Status {
	spec, ok := operationLoadingSpec(op, message, action)
	if !ok {
		panic("unknown progress operation")
	}
	return operationLoadingStatus(spec)
}

func loadingToast(message string) state.Status {
	s := state.New().WithLoading(message)
	s.Detail = "Please wait."
	return s
}
