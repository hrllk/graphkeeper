package state

type Mode string

const (
	ModeBrowse         Mode = "browse"
	ModePullCheck      Mode = "pull_check"
	ModeTargetPick     Mode = "target_pick"
	ModeOutcomePreview Mode = "outcome_preview"
	ModeBlocked        Mode = "blocked"
	ModeLoading        Mode = "loading"
	ModeEmpty          Mode = "empty"
	ModeError          Mode = "error"
	ModeConfirm        Mode = "confirm"
)

type Action string

const (
	ActionNone     Action = ""
	ActionPull     Action = "pull"
	ActionAbort    Action = "abort"
	ActionCheckout Action = "checkout"
	ActionMerge    Action = "merge"
	ActionRebase   Action = "rebase"
	ActionReset    Action = "reset"
	ActionPush     Action = "push"
	ActionForcePush Action = "force-push"
	ActionSetUpstream Action = "set-upstream"
)

type BlockReason string

const (
	BlockNone        BlockReason = ""
	BlockNoRepo      BlockReason = "no_repo"
	BlockDetached    BlockReason = "detached_head"
	BlockNoUpstream  BlockReason = "no_upstream"
	BlockNoRemote    BlockReason = "no_remote"
	BlockDiverged    BlockReason = "diverged"
	BlockFetchFailed BlockReason = "fetch_failed"
	BlockTargetEmpty BlockReason = "target_empty"
	BlockDirtyTree   BlockReason = "dirty_worktree"
	BlockUnknown     BlockReason = "unknown"
)

type TargetKind string

const (
	TargetKindLocal  TargetKind = "local"
	TargetKindRemote TargetKind = "remote"
	TargetKindTag    TargetKind = "tag"
)

type TargetItem struct {
	Kind            TargetKind
	Name            string
	Ref             string
	Current         bool
	Default         bool
	NeedsPull       bool
	NoUpstream      bool
	MergeConflicted bool
}

type Status struct {
	Mode       Mode
	Action     Action
	Block      BlockReason
	Title      string
	Message    string
	Detail     string
	Targets    []TargetItem
	TargetIdx  int
	Selected   string
	CanExecute bool
}

func New() Status {
	return Status{Mode: ModeLoading, Action: ActionNone, TargetIdx: -1}
}

func (s Status) WithBrowse() Status {
	s.Mode = ModeBrowse
	s.Action = ActionNone
	s.Block = BlockNone
	s.Title = "Browse"
	s.Message = "Inspect the graph and choose an action."
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
