package app

import (
	"time"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

type loadedMsg struct {
	status git.Status
	err    error
}

type tickMsg time.Time

type stashLoadedMsg struct {
	entries []git.StashEntry
	err     error
}

type refreshedMsg struct {
	status git.Status
	err    error
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
	status git.Status
	err    error
}

type pushFetchedMsg struct {
	status git.Status
	err    error
}

type pullPreviewReadyMsg struct {
	commits []string
	isFF    bool
	err     error
}

type pullToastDoneMsg struct{}

type branchToastDoneMsg struct{}
