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

func classifyConfirmationKey(view confirmationProjection, key string) confirmResult {
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

func classifyConfirmKey(view confirmViewModel, key string) confirmResult {
	return classifyConfirmationKey(confirmationProjection{
		Kind: view.Kind, Action: view.Action, Title: view.Title, Detail: view.Detail,
		ApprovalKey: view.ApprovalKey, ApprovalText: view.ApprovalText, FooterText: view.FooterText,
		ChoiceKeys: append([]confirmChoice(nil), view.ChoiceKeys...), CancelKeys: append([]string(nil), view.CancelKeys...),
		Disabled: view.Disabled, DisabledText: view.DisabledText,
	}, key)
}

func confirmView(m model) confirmViewModel {
	projection, ok := buildConfirmationProjection(m)
	if !ok {
		return confirmViewModel{}
	}
	return confirmationAsView(projection)
}
