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
		focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		target := checkoutTargetFromFocus(focus)
		if target == "" {
			return branchDeleteSelection{
				blocked: state.New().WithBlocked(state.BlockUnknown, "Delete unavailable.", "Move to a branch line."),
			}
		}
		if !m.repoStatus.Detached && target == m.repoStatus.Branch {
			return branchDeleteSelection{
				blocked: state.New().WithBlocked(state.BlockUnknown, "Current branch cannot be deleted.", "Select a different local branch."),
			}
		}
		if strings.HasPrefix(target, "origin/") {
			name := strings.TrimPrefix(target, "origin/")
			return branchDeleteSelection{
				target: name,
				remote: true,
				title:  "Delete branch?",
				detail: "Remote: origin/" + name,
				ok:     true,
			}
		}
		return branchDeleteSelection{
			target: target,
			title:  "Delete branch?",
			detail: "Local: " + target,
			ok:     true,
		}
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

func deleteTagLoadingMessage() string {
	return "Deleting tag..."
}
