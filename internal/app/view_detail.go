package app

import (
	"fmt"

	"hrllk/graphkeeper/internal/state"
)

func (m model) renderGlobalContent(width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := make([]string, 0, height)
	lines = append(lines, title.Render("Mode"))
	lines = append(lines, renderStatusCompact(m.status))
	lines = append(lines, "")
	lines = append(lines, title.Render("Hotkeys"))
	lines = append(lines, "tab / shift+tab   section")
	lines = append(lines, "j / k             move")
	lines = append(lines, "f                 fetch")
	lines = append(lines, "q                 quit")
	lines = append(lines, "")
	lines = append(lines, title.Render("Repo"))
	lines = append(lines, fmt.Sprintf("branch: %-12s • head: %s", shorten(m.repoStatus.Branch, 10), shorten(m.repoStatus.Head, 7)))
	lines = append(lines, fmt.Sprintf("upstream: %-10s • remote: %s", shorten(emptyDash(m.repoStatus.Upstream), 10), shorten(emptyDash(m.repoStatus.Remote), 10)))
	return fitBlockLines(lines, height)
}

func (m model) renderContextContent(width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := make([]string, 0, height)
	switch m.activeSection {
	case sectionGraph:
		focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash != "" {
			lines = append(lines, fmt.Sprintf("focus: %s", shorten(focus.Hash, max(width-7, 0))))
			lines = append(lines, focusParentLines(focus, width)...)
			if branchLines := focusBranchSummaryLines(focus, width); len(branchLines) > 0 {
				lines = append(lines, "branches:")
				lines = append(lines, branchLines...)
			}
			if stashLines := stashSummaryLines(m.stashesForCommit(focus.Hash), width); len(stashLines) > 0 {
				lines = append(lines, "stashes:")
				lines = append(lines, stashLines...)
			}
		}
	case sectionCurrent, sectionRemote, sectionTags:
		items := sectionTargets(m.repoStatus, m.activeSection)
		if len(items) == 0 {
			lines = append(lines, muted.Render("  (empty)"))
		} else {
			cursor := m.sectionCursor[m.activeSection]
			if cursor < 0 || cursor >= len(items) {
				cursor = 0
			}
			lines = append(lines, fmt.Sprintf("target: %s", formatTargetItem(items[cursor])))
			lines = append(lines, fmt.Sprintf("items: %d", len(items)))
			if m.activeSection == sectionCurrent {
				if m.status.WorktreeState != "" {
					worktree := string(m.status.WorktreeState)
					if m.status.WorktreeState == state.WorktreeStateDirty {
						worktree = dirtyMark.Render(worktree)
					}
					lines = append(lines, fmt.Sprintf("worktree: %s", worktree))
				}
				if current := items[cursor]; current.Current {
					if current.NeedsPull {
						lines = append(lines, "sync: pull available")
					}
					if current.NeedsPush {
						lines = append(lines, "sync: push required")
					}
					if current.NoUpstream {
						lines = append(lines, "sync: no upstream")
					}
				}
			}
		}
	}

	lines = append(lines, "")
	lines = append(lines, title.Render("Actions"))
	lines = append(lines, renderActionHelpLines(m)...)
	return fitBlockLines(lines, height)
}

func (m model) renderDetailContent(width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := make([]string, 0, height)
	lines = append(lines, renderStatusCompact(m.status))
	if m.status.WorktreeState != "" {
		worktree := string(m.status.WorktreeState)
		if m.status.WorktreeState == state.WorktreeStateDirty {
			worktree = dirtyMark.Render(worktree)
		}
		lines = append(lines, fmt.Sprintf("worktree: %s", worktree))
	}
	lines = append(lines, "")

	lines = append(lines, title.Render("Repo"))
	lines = append(lines, fmt.Sprintf("branch: %-12s • head: %s", shorten(m.repoStatus.Branch, 10), shorten(m.repoStatus.Head, 7)))
	lines = append(lines, fmt.Sprintf("upstream: %-10s • remote: %s", shorten(emptyDash(m.repoStatus.Upstream), 10), shorten(emptyDash(m.repoStatus.Remote), 10)))

	focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
	if focus.Hash != "" {
		lines = append(lines, fmt.Sprintf("focus: %s", shorten(focus.Hash, max(width-7, 0))))
		lines = append(lines, focusParentLines(focus, width)...)
		if branchLines := focusBranchSummaryLines(focus, width); len(branchLines) > 0 {
			lines = append(lines, "branches:")
			lines = append(lines, branchLines...)
		}
		if stashLines := stashSummaryLines(m.stashesForCommit(focus.Hash), width); len(stashLines) > 0 {
			lines = append(lines, "stashes:")
			lines = append(lines, stashLines...)
		}
	}
	lines = append(lines, fmt.Sprintf("active: %s", sectionName(m.activeSection)))
	if m.status.Selected != "" {
		lines = append(lines, fmt.Sprintf("select: %s", shorten(m.status.Selected, width-2)))
	}
	if m.branchOpen {
		lines = append(lines, fmt.Sprintf("new br: %s (base: %s)", m.branchDraft, shorten(m.branchBase, 7)))
	}
	lines = append(lines, "")
	lines = append(lines, title.Render("Actions"))
	lines = append(lines, renderActionHelpLines(m)...)
	return fitBlockLines(lines, height)
}
