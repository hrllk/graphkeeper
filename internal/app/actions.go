package app

import (
	"fmt"
	"strings"
	"hrllk/git-graph-tui/internal/git"
	"hrllk/git-graph-tui/internal/state"
)

func deriveStatus(rs git.Status) state.Status {
	switch {
	case rs.Root == "":
		return state.New().WithBlocked(state.BlockNoRepo, "Not inside a Git repository.", "Run this tool from a repo root.")
	case rs.MergeInProgress || rs.RebaseInProgress:
		status := state.New().WithBrowse()
		status.Message = "Merge/Rebase in progress after conflict."
		status.Detail = "Press enter to abort the in-progress merge/rebase."
		return status
	case rs.Detached:
		return state.New().WithBlocked(state.BlockDetached, "Detached HEAD.", "Pick a branch before running pull, merge, or rebase.")
	case rs.EmptyRepo:
		return state.New().WithEmpty("Repository has no commits yet.")
	case rs.NoRemote && rs.NoUpstream:
		return state.New().WithBlocked(state.BlockNoRemote, "No remote or upstream configured.", "Pull, merge, and rebase need a branch with a remote target.")
	default:
		return state.New().WithBrowse()
	}
}

func actionPull(rs git.Status) state.Status {
	if rs.Root == "" {
		return state.New().WithBlocked(state.BlockNoRepo, "Not inside a Git repository.", "Run this tool from a repo root.")
	}
	if rs.Detached {
		return state.New().WithBlocked(state.BlockDetached, "Detached HEAD.", "Pull needs a branch with an upstream.")
	}
	if rs.MergeInProgress || rs.RebaseInProgress {
		return state.New().WithBlocked(state.BlockUnknown, "A merge/rebase is already in progress.", "Abort or resolve the existing merge/rebase before pulling again.")
	}
	if rs.NoRemote {
		return state.New().WithBlocked(state.BlockNoRemote, "No remote configured.", "Pull needs origin or another remote.")
	}
	if rs.NoUpstream {
		return state.New().WithBlocked(state.BlockNoUpstream, "No upstream configured.", "Set an upstream before pulling.")
	}
	return state.New().WithOutcome(state.ActionPull, "Pull is ready.", "Pull will fetch and merge upstream changes into the current branch.", true)
}

func pullReady(rs git.Status) bool {
	return rs.Root != "" && !rs.Detached && !rs.NoRemote && !rs.NoUpstream && !rs.MergeInProgress
}

func canCreateBranch(rs git.Status) bool {
	return !rs.WorktreeDirty
}

func actionPickTargets(rs git.Status, action state.Action) state.Status {
	if (action == state.ActionMerge || action == state.ActionRebase) && rs.Detached {
		return state.New().WithBlocked(state.BlockDetached, "Detached HEAD.", "Choose a branch before merging or rebasing.")
	}
	targets := make([]state.TargetItem, 0, len(rs.LocalBranches)+len(rs.RemoteBranches)+len(rs.Tags))
	for _, name := range rs.LocalBranches {
		upstream, known := branchUpstream(rs, name)
		targets = append(targets, state.TargetItem{
			Kind:       state.TargetKindLocal,
			Name:       name,
			Ref:        name,
			NoUpstream: known && upstream == "",
		})
	}
	for _, name := range rs.RemoteBranches {
		if strings.HasSuffix(name, "/HEAD") {
			continue
		}
		targets = append(targets, state.TargetItem{Kind: state.TargetKindRemote, Name: name, Ref: name})
	}
	for _, name := range rs.Tags {
		targets = append(targets, state.TargetItem{Kind: state.TargetKindTag, Name: name, Ref: name})
	}
	if len(targets) == 0 {
		for _, name := range rs.Branches {
			targets = append(targets, state.TargetItem{Kind: state.TargetKindLocal, Name: name, Ref: name})
		}
	}
	if len(targets) == 0 {
		return state.New().WithBlocked(state.BlockTargetEmpty, "No branch targets available.", "Create or fetch a branch before merging, rebasing, or resetting.")
	}
	status := state.New().WithTargetPick(action, targets)
	status.Message = "Use up/down to choose a target."
	status.Detail = "Enter previews the result. Esc returns to browse."
	return status
}

func checkoutTargetFromFocus(node graphNode) string {
	for _, decoration := range node.Decorations {
		decoration = strings.TrimSpace(decoration)
		if strings.HasPrefix(decoration, "HEAD -> ") {
			return strings.TrimPrefix(decoration, "HEAD -> ")
		}
		if strings.HasPrefix(decoration, "tag: ") {
			continue
		}
		if strings.Contains(decoration, "/") {
			return decoration
		}
		if decoration != "" {
			return decoration
		}
	}
	return ""
}

func selectedTarget(s state.Status) string {
	if s.Selected != "" {
		return s.Selected
	}
	if s.TargetIdx >= 0 && s.TargetIdx < len(s.Targets) {
		return s.Targets[s.TargetIdx].Ref
	}
	return ""
}

func buildActionPreview(action state.Action, target string, rs git.Status, currentOnly, targetOnly int) state.Status {
	head := shorten(rs.Head, 12)
	switch action {
	case state.ActionMerge:
		switch {
		case currentOnly == 0 && targetOnly == 0:
			return state.New().WithOutcome(state.ActionMerge, "Target already matches HEAD.", "Nothing moves. The branch already points at the same commit.", true)
		case currentOnly == 0:
			return state.New().WithOutcome(state.ActionMerge, "FF 가능. 포인터만 이동합니다.", "HEAD can move to "+target+". Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly), true)
		case targetOnly == 0:
			return state.New().WithOutcome(state.ActionMerge, "대상은 이미 포함되어 있습니다.", "Current branch already contains "+target+". Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly), true)
		default:
			return state.New().WithOutcome(state.ActionMerge, "FF 불가. merge commit이 필요합니다.", "HEAD "+head+" and target "+target+" have diverged. Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly), true)
		}
	case state.ActionRebase:
		switch {
		case currentOnly == 0 && targetOnly == 0:
			return state.New().WithOutcome(state.ActionRebase, "Target already matches HEAD.", "Nothing is rewritten because both refs point at the same commit.", true)
		case targetOnly == 0:
			return state.New().WithOutcome(state.ActionRebase, "Target is already in the base history.", "Current commits will replay onto "+target+". Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly), true)
		default:
			return state.New().WithOutcome(state.ActionRebase, "새 base 위로 커밋을 재배치합니다.", "Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly)+"  |  target: "+target, true)
		}
	case state.ActionReset:
		return state.New().WithOutcome(state.ActionReset, "현재 HEAD를 선택한 위치로 이동합니다.", "HEAD "+head+" -> "+target+"  |  Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly), true)
	default:
		return state.New().WithOutcome(action, "No action selected.", target, false)
	}
}

func executionDetail(action state.Action, target string, rs git.Status) string {
	switch action {
	case state.ActionPull:
		return "Upstream pointer is now reflected in the local branch."
	case state.ActionMerge:
		return "Merge complete. HEAD now reflects " + emptyDash(rs.Branch) + " with target " + target + "."
	case state.ActionRebase:
		return "Rebase complete. The branch was replayed on top of " + target + "."
	case state.ActionReset:
		return "Hard reset complete. HEAD now points at " + target + "."
	default:
		return "Action complete."
	}
}

func findRemoteCommitHash(rs git.Status, upstream string) string {
	if upstream == "" {
		return ""
	}
	target := upstream
	if strings.HasPrefix(target, "refs/remotes/") {
		target = strings.TrimPrefix(target, "refs/remotes/")
	}
	for _, commit := range rs.GraphCommits {
		for _, dec := range commit.Decorations {
			decTrim := strings.TrimSpace(dec)
			if decTrim == target || "origin/"+decTrim == target || decTrim == "origin/"+target {
				return commit.Hash
			}
		}
	}
	return ""
}
