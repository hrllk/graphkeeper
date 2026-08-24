package app

import "hrllk/graphkeeper/internal/state"

// confirmationProjection is the immutable, shared confirmation contract used by
// both the popup renderer and key classifier. Slices are owned by the projection.
type confirmationProjection struct {
	Kind         confirmKind
	FastForward  bool
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

	CurrentBranch string
	TargetRef     string
	CurrentOnly   int
	TargetOnly    int
	ImpactKnown   bool
	MergeText     string
	RebaseText    string
	RiskText      string
}

// clearPullConfirmProjection removes producer-owned pull confirmation data.
// Callers must invoke this only after their request/identity guards pass; stale
// or superseded messages must not erase a newer projection.
func clearPullConfirmProjection(m *model) {
	m.pullConfirmInput = nil
	m.mergeConfirmView = nil
}

// buildConfirmationProjection is the sole consumer-facing confirmation
// projection builder. mergeConfirmView is consulted only as a compatibility
// input while older producer fixtures migrate to pullConfirmInput.
func buildConfirmationProjection(m model) (confirmationProjection, bool) {
	if m.status.Mode != state.ModeConfirm {
		return confirmationProjection{}, false
	}
	p := confirmationProjection{
		Kind:         confirmBinary,
		FastForward:  false,
		Action:       m.status.Action,
		Title:        m.status.Message,
		Detail:       m.status.Detail,
		ApprovalKey:  "y",
		ApprovalText: "yes",
		FooterText:   "y: yes  •  n: no",
		ChoiceKeys:   nil,
		CancelKeys:   []string{"n", "esc"},
		Disabled:     false,
	}
	if p.Action == state.ActionPull && !m.pullIsFastForward {
		p.Kind = confirmChoiceKind
		p.ChoiceKeys = []confirmChoice{{Key: "m", Label: "merge"}, {Key: "r", Label: "rebase"}}
		p.FooterText = "m: merge · r: rebase"
		input := m.pullConfirmInput
		if input == nil && m.mergeConfirmView != nil {
			v := m.mergeConfirmView
			input = &pullConfirmInput{CurrentBranch: v.CurrentBranch, TargetRef: v.TargetRef, CurrentOnly: v.CurrentOnly, TargetOnly: v.TargetOnly, ImpactKnown: v.ImpactKnown, MergeText: v.MergeText, RebaseText: v.RebaseText, RiskText: v.RiskText}
		}
		if input == nil && !m.pullConfirmStale && m.activePullRequest == nil && m.status.Block == "" {
			return confirmationProjection{}, false
		}
		if input != nil {
			if !input.ImpactKnown && !m.pullConfirmStale {
				return confirmationProjection{}, false
			}
			p.CurrentBranch = input.CurrentBranch
			p.TargetRef = input.TargetRef
			p.CurrentOnly = input.CurrentOnly
			p.TargetOnly = input.TargetOnly
			p.ImpactKnown = input.ImpactKnown
			p.MergeText = input.MergeText
			p.RebaseText = input.RebaseText
			p.RiskText = input.RiskText
		}
	}
	if p.Action == state.ActionPull && m.pullIsFastForward {
		p.FastForward = true
		p.ApprovalKey = "f"
		p.ApprovalText = "fast-forward"
		p.FooterText = "f: fast-forward"
	} else if p.Action == state.ActionDeleteBranch || p.Action == state.ActionDeleteTag {
		p.FooterText = "y: delete  •  n: cancel"
	} else if p.Action == state.ActionStash {
		p.FooterText = "y: stash  •  n: cancel"
	} else if p.Action == state.ActionCleanWorkingTree {
		p.FooterText = "y: clean  •  n: cancel"
	}
	if m.pullConfirmStale && p.Action == state.ActionPull {
		p.Disabled = true
		p.DisabledText = "Preview is stale. Refresh before continuing."
		p.FooterText = ""
	}
	p.ChoiceKeys = append([]confirmChoice(nil), p.ChoiceKeys...)
	p.CancelKeys = append([]string(nil), p.CancelKeys...)
	return p, true
}

func confirmationAsView(p confirmationProjection) confirmViewModel {
	return confirmViewModel{Kind: p.Kind, FastForward: p.FastForward, Action: p.Action, Title: p.Title, Detail: p.Detail, ApprovalKey: p.ApprovalKey, ApprovalText: p.ApprovalText, FooterText: p.FooterText, ChoiceKeys: append([]confirmChoice(nil), p.ChoiceKeys...), CancelKeys: append([]string(nil), p.CancelKeys...), Disabled: p.Disabled, DisabledText: p.DisabledText}
}

func (p confirmationProjection) mergeView() mergeConfirmViewModel {
	return mergeConfirmViewModel{CurrentBranch: p.CurrentBranch, TargetRef: p.TargetRef, CurrentOnly: p.CurrentOnly, TargetOnly: p.TargetOnly, ImpactKnown: p.ImpactKnown, MergeText: p.MergeText, RebaseText: p.RebaseText, RiskText: p.RiskText, Disabled: p.Disabled, DisabledText: p.DisabledText}
}
