package app

import "hrllk/graphkeeper/internal/state"

type confirmKind string

const (
	confirmBinary     confirmKind = "binary"
	confirmChoiceKind confirmKind = "choice"
)

type confirmDecision string

const (
	decisionAccept confirmDecision = "accept"
	decisionCancel confirmDecision = "cancel"
	decisionChoice confirmDecision = "choice"
	decisionNoop   confirmDecision = "noop"
)

const (
	choiceMerge  = "merge"
	choiceRebase = "rebase"
)

type confirmResult struct {
	Decision  confirmDecision
	ChoiceKey string
}

type confirmChoice struct {
	Key   string
	Label string
}

type confirmViewModel struct {
	Kind         confirmKind
	Action       state.Action
	Title        string
	Detail       string
	ApprovalKey  string
	ApprovalText string
	FooterText   string
	ChoiceKeys   []confirmChoice
	CancelKeys   []string
	Disabled     bool
	DisabledText string
}

func classifyConfirmKey(view confirmViewModel, key string) confirmResult {
	if view.Disabled {
		switch key {
		case "n", "esc":
			return confirmResult{Decision: decisionCancel}
		default:
			return confirmResult{Decision: decisionNoop}
		}
	}
	if view.Kind == confirmChoiceKind {
		switch key {
		case "m":
			return confirmResult{Decision: decisionChoice, ChoiceKey: choiceMerge}
		case "enter":
			return confirmResult{Decision: decisionAccept, ChoiceKey: choiceMerge}
		case "r":
			return confirmResult{Decision: decisionChoice, ChoiceKey: choiceRebase}
		case "n", "esc":
			return confirmResult{Decision: decisionCancel}
		default:
			return confirmResult{Decision: decisionNoop}
		}
	}
	switch key {
	case "y", "enter":
		return confirmResult{Decision: decisionAccept}
	case "n", "esc":
		return confirmResult{Decision: decisionCancel}
	default:
		return confirmResult{Decision: decisionNoop}
	}
}

func confirmView(m model) confirmViewModel {
	view := confirmViewModel{
		Kind:         confirmBinary,
		Action:       m.status.Action,
		Title:        m.status.Title,
		Detail:       m.status.Detail,
		ApprovalKey:  "y",
		ApprovalText: "yes",
		FooterText:   "y: yes  •  n: no",
		CancelKeys:   []string{"n", "esc"},
	}
	if m.status.Action == state.ActionPull && !m.pullIsFastForward {
		view.Kind = confirmChoiceKind
		view.ChoiceKeys = []confirmChoice{{Key: "m", Label: "merge"}, {Key: "r", Label: "rebase"}}
		view.FooterText = "m: merge  •  enter: merge  •  r: rebase  •  n: cancel"
	} else if m.status.Message == "Fast-forward available." {
		view.FooterText = "enter: fast-forward"
	} else if m.status.Action == state.ActionDeleteBranch || m.status.Action == state.ActionDeleteTag {
		view.FooterText = "y: delete  •  n: cancel"
	} else if m.status.Action == state.ActionStash {
		view.FooterText = "y: stash  •  n: cancel"
	} else if m.status.Action == state.ActionCleanWorkingTree {
		view.FooterText = "y: clean  •  n: cancel"
	}
	if m.pullConfirmStale && m.status.Action == state.ActionPull {
		view.Disabled = true
		view.DisabledText = "Preview is stale. Refresh before continuing."
		view.FooterText = "n: close  •  esc: close"
	}
	return view
}
