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

// buildMergePreview follows the four-state table in
// docs/20260701-0005-feature-graph-section-merge-rebase-gate-note.md. Two of the
// four states are not an action at all and say so instead of offering one. A
// fast-forward stays executable against the note's advice: it is the only way to
// fast-forward a local branch onto another local branch's tip, and the pull the
// note routes it to needs an upstream. The graph shortcut in update_fetch.go
// classifies the same four states and must agree with this function.
func buildMergePreview(target string, rs git.Status, currentOnly, targetOnly int) state.Status {
	switch {
	case currentOnly == 0 && targetOnly == 0:
		return state.New().WithOutcome(state.ActionMerge, "Nothing to merge.", "Target is the commit HEAD already points at.", false)
	case currentOnly == 0:
		return state.New().WithOutcome(state.ActionMerge, "Fast-forward available.", "HEAD can move straight to "+target+"; no merge commit is created. "+countDetail(currentOnly, targetOnly), true)
	case targetOnly == 0:
		return state.New().WithOutcome(state.ActionMerge, "Nothing to merge.", "This branch already contains "+target+". "+countDetail(currentOnly, targetOnly), false)
	default:
		return state.New().WithOutcome(state.ActionMerge, "Merge target into HEAD.", "HEAD "+shorten(rs.Head, 12)+" and target "+target+" have diverged, so a merge commit is created. "+countDetail(currentOnly, targetOnly), true)
	}
}

// buildRebasePreview follows the same four-state table as buildMergePreview. The
// fast-forward state used to fall through to the "target already in history"
// branch, which described replaying commits that do not exist.
func buildRebasePreview(target string, currentOnly, targetOnly int) state.Status {
	switch {
	case currentOnly == 0 && targetOnly == 0:
		return state.New().WithOutcome(state.ActionRebase, "Nothing to rebase.", "Target is the commit HEAD already points at.", false)
	case currentOnly == 0:
		return state.New().WithOutcome(state.ActionRebase, "Fast-forward available.", "This branch has no commits of its own to replay, so HEAD moves straight to "+target+". "+countDetail(currentOnly, targetOnly), true)
	case targetOnly == 0:
		return state.New().WithOutcome(state.ActionRebase, "Target is already in this history.", "Replaying onto "+target+" would rewrite commits this branch already contains. "+countDetail(currentOnly, targetOnly), false)
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
