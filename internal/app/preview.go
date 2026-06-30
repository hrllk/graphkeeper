package app

import (
	"fmt"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func buildActionPreview(action state.Action, target string, rs git.Status, currentOnly, targetOnly int) state.Status {
	switch action {
	case state.ActionMerge:
		return buildMergePreview(target, rs, currentOnly, targetOnly)
	case state.ActionRebase:
		return buildRebasePreview(target, currentOnly, targetOnly)
	case state.ActionReset:
		return buildResetPreview(target, rs, currentOnly, targetOnly)
	default:
		return state.New().WithOutcome(action, "No action selected.", target, false)
	}
}

func buildMergePreview(target string, rs git.Status, currentOnly, targetOnly int) state.Status {
	switch {
	case currentOnly == 0 && targetOnly == 0:
		return state.New().WithOutcome(state.ActionMerge, "Already aligned.", "Target already matches HEAD.", true)
	case currentOnly == 0:
		return state.New().WithOutcome(state.ActionMerge, "Fast-forward available.", "HEAD can move to "+target+". "+countDetail(currentOnly, targetOnly), true)
	case targetOnly == 0:
		return state.New().WithOutcome(state.ActionMerge, "Target already included.", "Current branch already contains "+target+". "+countDetail(currentOnly, targetOnly), true)
	default:
		return state.New().WithOutcome(state.ActionMerge, "Fast-forward unavailable.", "HEAD "+shorten(rs.Head, 12)+" and target "+target+" have diverged. "+countDetail(currentOnly, targetOnly), true)
	}
}

func buildRebasePreview(target string, currentOnly, targetOnly int) state.Status {
	switch {
	case currentOnly == 0 && targetOnly == 0:
		return state.New().WithOutcome(state.ActionRebase, "Already aligned.", "Target already matches HEAD.", true)
	case targetOnly == 0:
		return state.New().WithOutcome(state.ActionRebase, "Target already in history.", "Current commits will replay onto "+target+". "+countDetail(currentOnly, targetOnly), true)
	default:
		return state.New().WithOutcome(state.ActionRebase, "Rebase onto target.", countDetail(currentOnly, targetOnly)+"  |  target: "+target, true)
	}
}

func buildResetPreview(target string, rs git.Status, currentOnly, targetOnly int) state.Status {
	status := state.New().WithResetModePick(
		"Choose a reset mode.",
		"",
	)
	status.Selected = target
	status.ResetMode = state.ResetModeMixed
	return applyRepoMetadata(status, rs)
}

func countDetail(currentOnly, targetOnly int) string {
	return "Current: " + fmt.Sprint(currentOnly) + "  Target: " + fmt.Sprint(targetOnly)
}
