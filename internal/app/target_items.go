package app

import (
	"strings"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func branchUpstream(rs git.Status, name string) (string, bool) {
	if name == "" {
		return "", false
	}
	if rs.BranchUpstreams != nil {
		if upstream, ok := rs.BranchUpstreams[name]; ok {
			return upstream, true
		}
	}
	if name == rs.Branch && rs.Branch != "HEAD" {
		return rs.Upstream, true
	}
	return "", false
}

func buildActionTargetItems(rs git.Status) []state.TargetItem {
	targets := make([]state.TargetItem, 0, len(rs.LocalBranches)+len(rs.RemoteBranches)+len(rs.Tags))
	targets = appendLocalTargets(targets, rs)
	targets = appendRemoteTargets(targets, rs.RemoteBranches)
	targets = appendTagTargets(targets, rs.Tags)
	if len(targets) == 0 {
		targets = appendFallbackBranchTargets(targets, rs.Branches)
	}
	return targets
}

func buildResetTargetItems(rs git.Status) []state.TargetItem {
	targets := make([]state.TargetItem, 0, len(rs.LocalBranches)+len(rs.Branches))
	targets = appendLocalTargets(targets, rs)
	if len(targets) == 0 {
		targets = appendFallbackBranchTargets(targets, rs.Branches)
	}
	return targets
}

func buildCurrentSectionTargets(rs git.Status) []state.TargetItem {
	items := make([]state.TargetItem, 0, 1+len(rs.LocalBranches))
	if rs.Branch != "" {
		track := rs.Tracking[rs.Branch]
		upstream, known := branchUpstream(rs, rs.Branch)
		items = append(items, state.TargetItem{
			Kind:            state.TargetKindLocal,
			Name:            rs.Branch,
			Ref:             rs.Branch,
			Current:         true,
			WorktreeDirty:   rs.WorktreeDirty,
			NeedsPull:       track.Behind > 0 && track.Ahead == 0,
			NeedsPush:       track.Ahead > 0,
			NoUpstream:      known && upstream == "",
			MergeConflicted: rs.MergeInProgress,
		})
	} else if rs.Head != "" {
		items = append(items, state.TargetItem{
			Kind:            state.TargetKindLocal,
			Name:            "HEAD",
			Ref:             rs.Head,
			Current:         true,
			WorktreeDirty:   rs.WorktreeDirty,
			MergeConflicted: rs.MergeInProgress,
		})
	}
	for _, name := range rs.LocalBranches {
		if name == rs.Branch {
			continue
		}
		track := rs.Tracking[name]
		upstream, known := branchUpstream(rs, name)
		items = append(items, state.TargetItem{
			Kind:       state.TargetKindLocal,
			Name:       name,
			Ref:        name,
			NeedsPull:  track.Behind > 0 && track.Ahead == 0,
			NeedsPush:  track.Ahead > 0,
			NoUpstream: known && upstream == "",
		})
	}
	return items
}

func buildRemoteSectionTargets(rs git.Status) []state.TargetItem {
	items := make([]state.TargetItem, 0, len(rs.RemoteBranches))
	for _, name := range rs.RemoteBranches {
		if !strings.Contains(name, "/") {
			continue
		}
		items = append(items, state.TargetItem{
			Kind:    state.TargetKindRemote,
			Name:    name,
			Ref:     name,
			Default: strings.HasSuffix(name, "/HEAD") || name == "origin/"+rs.DefaultBranch,
		})
	}
	return items
}

func buildTagSectionTargets(rs git.Status) []state.TargetItem {
	items := make([]state.TargetItem, 0, len(rs.Tags))
	for _, name := range rs.Tags {
		items = append(items, state.TargetItem{Kind: state.TargetKindTag, Name: name, Ref: name})
	}
	return items
}

func appendLocalTargets(targets []state.TargetItem, rs git.Status) []state.TargetItem {
	for _, name := range rs.LocalBranches {
		upstream, known := branchUpstream(rs, name)
		targets = append(targets, state.TargetItem{
			Kind:       state.TargetKindLocal,
			Name:       name,
			Ref:        name,
			NoUpstream: known && upstream == "",
		})
	}
	return targets
}

func appendRemoteTargets(targets []state.TargetItem, remoteBranches []string) []state.TargetItem {
	for _, name := range remoteBranches {
		if isRemoteHeadRef(name) {
			continue
		}
		targets = append(targets, state.TargetItem{
			Kind: state.TargetKindRemote,
			Name: name,
			Ref:  name,
		})
	}
	return targets
}

func appendTagTargets(targets []state.TargetItem, tags []string) []state.TargetItem {
	for _, name := range tags {
		targets = append(targets, state.TargetItem{
			Kind: state.TargetKindTag,
			Name: name,
			Ref:  name,
		})
	}
	return targets
}

func appendFallbackBranchTargets(targets []state.TargetItem, branches []string) []state.TargetItem {
	for _, name := range branches {
		targets = append(targets, state.TargetItem{
			Kind: state.TargetKindLocal,
			Name: name,
			Ref:  name,
		})
	}
	return targets
}

func isRemoteHeadRef(name string) bool {
	return strings.HasSuffix(name, "/HEAD")
}
