package app

import (
	"strings"

	"hrllk/graphkeeper/internal/state"
)

type branchDeleteSelection struct {
	target  string
	remote  bool
	title   string
	detail  string
	blocked state.Status
	ok      bool
}

func graphBranchDeleteTargets(m model) []state.TargetItem {
	focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
	localBranches := make(map[string]struct{}, len(m.repoStatus.LocalBranches))
	for _, name := range m.repoStatus.LocalBranches {
		localBranches[name] = struct{}{}
	}
	remoteBranches := make(map[string]struct{}, len(m.repoStatus.RemoteBranches))
	for _, name := range m.repoStatus.RemoteBranches {
		remoteBranches[name] = struct{}{}
	}

	targets := make([]state.TargetItem, 0, len(focus.Decorations))
	seen := make(map[string]struct{}, len(focus.Decorations))
	for _, decoration := range focus.Decorations {
		decoration = strings.TrimSpace(decoration)
		if strings.HasPrefix(decoration, "HEAD -> ") {
			decoration = strings.TrimPrefix(decoration, "HEAD -> ")
		}
		if decoration == "" || strings.HasPrefix(decoration, "tag: ") {
			continue
		}

		kind := state.TargetKindLocal
		if strings.HasPrefix(decoration, "origin/") {
			if _, ok := remoteBranches[decoration]; !ok && len(remoteBranches) > 0 {
				continue
			}
			if isRemoteHeadRef(decoration) {
				continue
			}
			kind = state.TargetKindRemote
		} else {
			if _, ok := localBranches[decoration]; !ok && len(localBranches) > 0 {
				continue
			}
			if !m.repoStatus.Detached && decoration == m.repoStatus.Branch {
				continue
			}
		}

		if _, ok := seen[decoration]; ok {
			continue
		}
		seen[decoration] = struct{}{}
		targets = append(targets, state.TargetItem{
			Kind:    kind,
			Name:    decoration,
			Ref:     decoration,
			Current: kind == state.TargetKindLocal && decoration == m.repoStatus.Branch,
		})
	}
	return targets
}

func deleteBranchSelectionFromTarget(item state.TargetItem) branchDeleteSelection {
	if item.Kind == state.TargetKindRemote {
		name := strings.TrimPrefix(item.Ref, "origin/")
		if name == "" {
			return branchDeleteSelection{
				blocked: state.New().WithBlocked(state.BlockUnknown, "Delete unavailable.", "Choose an origin branch."),
			}
		}
		return branchDeleteSelection{
			target: name,
			remote: true,
			title:  "Delete branch?",
			detail: "Remote: origin/" + name,
			ok:     true,
		}
	}
	if item.Current {
		return branchDeleteSelection{
			blocked: state.New().WithBlocked(state.BlockUnknown, "Current branch cannot be deleted.", "Select a different local branch."),
		}
	}
	return branchDeleteSelection{
		target: item.Ref,
		title:  "Delete branch?",
		detail: "Local: " + item.Ref,
		ok:     item.Ref != "",
	}
}

func activeSectionTargetItem(m model) (state.TargetItem, bool) {
	items := sectionTargets(m.repoStatus, m.activeSection)
	cursor := m.sectionCursor[m.activeSection]
	if cursor < 0 || cursor >= len(items) {
		return state.TargetItem{}, false
	}
	return items[cursor], true
}

func deleteBranchSelection(m model) branchDeleteSelection {
	if m.activeSection == sectionGraph {
		targets := graphBranchDeleteTargets(m)
		if len(targets) == 0 {
			return branchDeleteSelection{
				blocked: state.New().WithBlocked(state.BlockUnknown, "Delete unavailable.", "Move to a branch line."),
			}
		}
		return deleteBranchSelectionFromTarget(targets[0])
	}

	item, ok := activeSectionTargetItem(m)
	if !ok {
		return branchDeleteSelection{
			blocked: state.New().WithBlocked(state.BlockTargetEmpty, "No branch selected.", "Choose a local or origin branch."),
		}
	}
	switch m.activeSection {
	case sectionCurrent:
		if item.Kind != state.TargetKindLocal {
			return branchDeleteSelection{
				blocked: state.New().WithBlocked(state.BlockUnknown, "Delete unavailable.", "Choose a local branch."),
			}
		}
		if item.Current {
			return branchDeleteSelection{
				blocked: state.New().WithBlocked(state.BlockUnknown, "Current branch cannot be deleted.", "Select a different local branch."),
			}
		}
		return branchDeleteSelection{
			target: item.Ref,
			title:  "Delete branch?",
			detail: "Local: " + item.Ref,
			ok:     true,
		}
	case sectionRemote:
		if item.Kind != state.TargetKindRemote || !strings.HasPrefix(item.Ref, "origin/") {
			return branchDeleteSelection{
				blocked: state.New().WithBlocked(state.BlockUnknown, "Delete unavailable.", "Choose an origin branch."),
			}
		}
		name := strings.TrimPrefix(item.Ref, "origin/")
		return branchDeleteSelection{
			target: name,
			remote: true,
			title:  "Delete branch?",
			detail: "Remote: origin/" + name,
			ok:     true,
		}
	default:
		return branchDeleteSelection{
			blocked: state.New().WithBlocked(state.BlockUnknown, "Delete unavailable here.", "Use the Context or Remote section."),
		}
	}
}

func deleteBranchLoadingMessage(remote bool) string {
	if remote {
		return "Deleting origin branch..."
	}
	return "Deleting branch..."
}

type tagDeleteSelection struct {
	target  string
	remote  bool
	title   string
	detail  string
	blocked state.Status
	ok      bool
}

func deleteTagSelection(m model) tagDeleteSelection {
	item, ok := activeSectionTargetItem(m)
	if !ok {
		return tagDeleteSelection{
			blocked: state.New().WithBlocked(state.BlockTargetEmpty, "No tag selected.", "Choose a tag row."),
		}
	}
	if item.Kind != state.TargetKindTag {
		return tagDeleteSelection{
			blocked: state.New().WithBlocked(state.BlockUnknown, "Delete unavailable.", "Choose a tag row."),
		}
	}
	if item.Ref == "" {
		return tagDeleteSelection{
			blocked: state.New().WithBlocked(state.BlockTargetEmpty, "Tag target is missing.", "Refresh the tag list and try again."),
		}
	}
	return tagDeleteSelection{
		target: item.Ref,
		title:  "Delete tag?",
		detail: "Tag: " + item.Ref,
		ok:     true,
	}
}

func deleteRemoteTagSelection(m model) tagDeleteSelection {
	item, ok := activeSectionTargetItem(m)
	if !ok {
		return tagDeleteSelection{
			blocked: state.New().WithBlocked(state.BlockTargetEmpty, "No tag selected.", "Choose a tag row."),
		}
	}
	if item.Kind != state.TargetKindTag {
		return tagDeleteSelection{
			blocked: state.New().WithBlocked(state.BlockUnknown, "Delete unavailable.", "Choose a tag row."),
		}
	}
	if !item.OriginKnown || !item.OnOrigin {
		return tagDeleteSelection{
			blocked: state.New().WithBlocked(state.BlockUnknown, "Remote tag not confirmed.", "Sync tag provenance first."),
		}
	}
	if item.Ref == "" {
		return tagDeleteSelection{
			blocked: state.New().WithBlocked(state.BlockTargetEmpty, "Tag target is missing.", "Refresh the tag list and try again."),
		}
	}
	return tagDeleteSelection{
		target: item.Ref,
		remote: true,
		title:  "Delete remote tag?",
		detail: "Remote: origin/" + item.Ref,
		ok:     true,
	}
}

func deleteTagLoadingMessage() string {
	return "Deleting tag..."
}

func deleteRemoteTagLoadingMessage() string {
	return "Deleting remote tag..."
}
