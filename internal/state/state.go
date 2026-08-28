package state

type Mode string

const (
	ModeBrowse          Mode = "browse"
	ModePullCheck       Mode = "pull_check"
	ModeTargetPick      Mode = "target_pick"
	ModeCherryPickPick  Mode = "cherry_pick_pick"
	ModeOutcomePreview  Mode = "outcome_preview"
	ModeReview          Mode = "review"
	ModeResetModePick   Mode = "reset_mode_pick"
	ModeBlocked         Mode = "blocked"
	ModeLoading         Mode = "loading"
	ModeEmpty           Mode = "empty"
	ModeError           Mode = "error"
	ModeConfirm         Mode = "confirm"
	ModeOperationResult Mode = "operation_result"
)

type Action string

const (
	ActionNone             Action = ""
	ActionPull             Action = "pull"
	ActionAbort            Action = "abort"
	ActionCheckout         Action = "checkout"
	ActionMerge            Action = "merge"
	ActionRebase           Action = "rebase"
	ActionCherryPick       Action = "cherry-pick"
	ActionReset            Action = "reset"
	ActionCreateBranch     Action = "create-branch"
	ActionDeleteBranch     Action = "delete-branch"
	ActionDeleteTag        Action = "delete-tag"
	ActionDeleteRemoteTag  Action = "delete-remote-tag"
	ActionStash            Action = "stash"
	ActionStashPop         Action = "stash-pop"
	ActionCleanWorkingTree Action = "clean-working-tree"
	ActionPush             Action = "push"
	ActionPushTag          Action = "push-tag"
	ActionForcePush        Action = "force-push"
	ActionSetUpstream      Action = "set-upstream"
	ActionPullMerge        Action = "pull-merge"
	ActionPullRebase       Action = "pull-rebase"
)

type BlockReason string

const (
	BlockNone          BlockReason = ""
	BlockNoRepo        BlockReason = "no_repo"
	BlockDetached      BlockReason = "detached_head"
	BlockNoUpstream    BlockReason = "no_upstream"
	BlockNoRemote      BlockReason = "no_remote"
	BlockDiverged      BlockReason = "diverged"
	BlockFetchFailed   BlockReason = "fetch_failed"
	BlockTargetEmpty   BlockReason = "target_empty"
	BlockDirtyTree     BlockReason = "dirty_worktree"
	BlockUnknown       BlockReason = "unknown"
	BlockStaleSnapshot BlockReason = "stale_snapshot"
	// BlockNotLocalPointer marks a graph row that no local branch points at, which
	// is what the merge and rebase gate actually tests.
	BlockNotLocalPointer BlockReason = "not_local_pointer"
	// BlockNotDiverged marks a target that shares all its history with HEAD, so a
	// merge is a no-op and a rebase rewrites commits the branch already has.
	BlockNotDiverged BlockReason = "not_diverged"
)

type ResetMode string

const (
	ResetModeSoft  ResetMode = "soft"
	ResetModeMixed ResetMode = "mixed"
	ResetModeHard  ResetMode = "hard"
)

type WorktreeState string

const (
	WorktreeStateClean WorktreeState = "clean"
	WorktreeStateDirty WorktreeState = "dirty"
)

type TargetKind string

const (
	TargetKindLocal  TargetKind = "local"
	TargetKindRemote TargetKind = "remote"
	TargetKindTag    TargetKind = "tag"
	TargetKindCommit TargetKind = "commit"
)

type TargetItem struct {
	Kind             TargetKind
	Name             string
	Ref              string
	CommitHash       string
	Author           string
	Subject          string
	RelativeAge      string
	ProvenanceLoaded bool
	OriginKnown      bool
	OnOrigin         bool
	Current          bool
	WorktreeDirty    bool
	Default          bool
	NeedsPull        bool
	NeedsPush        bool
	NoUpstream       bool
	MergeConflicted  bool
}

type Status struct {
	Mode          Mode
	Action        Action
	Block         BlockReason
	ResetMode     ResetMode
	WorktreeState WorktreeState
	Title         string
	Message       string
	Detail        string
	Targets       []TargetItem
	SelectedQueue []string
	TargetIdx     int
	Selected      string
	DeleteRemote  bool
	CanExecute    bool
}

func New() Status {
	return Status{Mode: ModeLoading, Action: ActionNone, TargetIdx: -1}
}

func (s Status) WithBrowse() Status {
	s.Mode = ModeBrowse
	s.Action = ActionNone
	s.Block = BlockNone
	s.Title = "Browse"
	s.Message = "Choose an action."
	s.Detail = ""
	s.TargetIdx = -1
	s.Selected = ""
	s.CanExecute = false
	return s
}

func (s Status) WithBlocked(reason BlockReason, message, detail string) Status {
	s.Mode = ModeBlocked
	s.Block = reason
	s.Title = "Blocked"
	s.Message = message
	s.Detail = detail
	s.CanExecute = false
	return s
}

func (s Status) WithTargetPick(action Action, targets []TargetItem) Status {
	s.Mode = ModeTargetPick
	s.Action = action
	s.Block = BlockNone
	s.Title = string(action)
	s.Message = "Choose a target."
	s.Detail = ""
	s.Targets = append([]TargetItem(nil), targets...)
	if len(targets) == 0 {
		s.TargetIdx = -1
		s.Selected = ""
	} else if s.TargetIdx < 0 || s.TargetIdx >= len(targets) {
		s.TargetIdx = 0
		s.Selected = targets[0].Ref
	} else {
		s.Selected = targets[s.TargetIdx].Ref
	}
	s.CanExecute = false
	return s
}

func (s Status) WithCherryPickPick(targets []TargetItem) Status {
	s.Mode = ModeCherryPickPick
	s.Action = ActionCherryPick
	s.Block = BlockNone
	s.Title = "Cherry-pick"
	s.Message = "Choose commits to cherry-pick."
	s.Detail = "Space toggles. Enter runs. Esc cancels."
	s.Targets = append([]TargetItem(nil), targets...)
	s.SelectedQueue = nil
	if len(targets) == 0 {
		s.TargetIdx = -1
		s.Selected = ""
	} else {
		if s.TargetIdx < 0 || s.TargetIdx >= len(targets) {
			s.TargetIdx = 0
		}
		s.Selected = targets[s.TargetIdx].Ref
	}
	s.CanExecute = false
	return s
}

func (s Status) WithOutcome(action Action, message, detail string, canExecute bool) Status {
	s.Mode = ModeOutcomePreview
	s.Action = action
	s.Block = BlockNone
	s.Title = string(action)
	s.Message = message
	s.Detail = detail
	s.CanExecute = canExecute
	return s
}

func (s Status) WithReview(action Action, message, detail string) Status {
	s.Mode = ModeReview
	s.Action = action
	s.Block = BlockNone
	s.Title = "Review"
	s.Message = message
	s.Detail = detail
	s.CanExecute = false
	return s
}

func (s Status) WithResetModePick(message, detail string) Status {
	s.Mode = ModeResetModePick
	s.Action = ActionReset
	s.Block = BlockNone
	s.Title = "Reset mode"
	s.Message = message
	s.Detail = detail
	if s.ResetMode == "" {
		s.ResetMode = ResetModeMixed
	}
	s.CanExecute = true
	return s
}

func (s Status) WithLoading(message string) Status {
	s.Mode = ModeLoading
	s.Action = ActionNone
	s.Block = BlockNone
	s.Title = "Loading"
	s.Message = message
	s.Detail = ""
	s.Selected = ""
	s.CanExecute = false
	return s
}

func (s Status) WithEmpty(message string) Status {
	s.Mode = ModeEmpty
	s.Action = ActionNone
	s.Block = BlockNone
	s.Title = "Empty"
	s.Message = message
	s.Detail = ""
	s.Selected = ""
	s.CanExecute = false
	return s
}

func (s Status) WithError(message string) Status {
	s.Mode = ModeError
	s.Action = ActionNone
	s.Block = BlockUnknown
	s.Title = "Error"
	s.Message = message
	s.Detail = ""
	s.Selected = ""
	s.CanExecute = false
	return s
}

func (s Status) WithConfirm(action Action, message, detail string) Status {
	s.Mode = ModeConfirm
	s.Action = action
	s.Block = BlockNone
	s.Title = "Confirm"
	s.Message = message
	s.Detail = detail
	s.CanExecute = true
	return s
}
