package app

import (
	"fmt"
	"strings"
	"time"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func (m model) renderGlobalContent(width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := make([]string, 0, height)
	if m.branchOpen {
		lines = append(lines, "")
	} else if m.status.Mode == state.ModeLoading || m.status.Mode == state.ModeBlocked {
		lines = append(lines, "")
	} else {
		lines = append(lines, "Mode: "+renderStatusCompact(m.status))
	}
	lines = append(lines, "")
	lines = append(lines, renderSectionTitle("Actions"))
	lines = append(lines, renderHotkeyLine("tab", "next section"))
	lines = append(lines, renderHotkeyLine("shift+tab", "previous section"))
	lines = append(lines, renderHotkeyLine("j/k", "move"))
	lines = append(lines, renderHotkeyLine("f", "fetch"))
	lines = append(lines, renderHotkeyLine("F", "fetch tags"))
	lines = append(lines, renderHotkeyLine("S", "stash list"))
	lines = append(lines, renderHotkeyLine("q", "quit"))
	lines = append(lines, renderHotkeyLine("?", "show hidden hotkeys"))
	lines = fitBlockWidth(lines, width)
	return fitBlockLines(lines, height)
}

func (m model) renderContextContent(width, height int) string {
	if height <= 0 {
		return ""
	}
	sectionTitle := sectionName(m.activeSection)
	leftLines := append([]string{renderSectionTitle(sectionTitle + " Details")}, m.renderContextInfoLines(width)...)
	rightLines := append([]string{renderSectionTitle(sectionTitle + " Actions")}, renderActionHelpLines(m)...)
	rightLines = indentLines(rightLines, 1)
	return renderSplitColumns(leftLines, rightLines, width, height)
}

func (m model) renderContextInfoLines(width int) []string {
	lines := make([]string, 0, 8)
	switch m.activeSection {
	case sectionGraph:
		focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("focus"), shorten(focus.Hash, 8)))
			lines = append(lines, focusParentLines(focus, width)...)
			if branchLines := focusBranchSummaryLines(focus, width); len(branchLines) > 0 {
				lines = append(lines, fmt.Sprintf("%s:", renderContextKey("branches")))
				lines = append(lines, branchLines...)
			}
			if stashLines := stashSummaryLines(m.stashesForCommit(focus.Hash), width); len(stashLines) > 0 {
				lines = append(lines, fmt.Sprintf("%s:", renderContextKey("stashes")))
				lines = append(lines, stashLines...)
			}
			if len(focus.Tags) > 0 {
				lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("tags"), renderTagList(focus.Tags)))
			}
		}
	case sectionCurrent, sectionRemote, sectionTags:
		items := sectionTargets(m.repoStatus, m.activeSection)
		if len(items) == 0 {
			lines = append(lines, muted.Render("  (empty)"))
			return lines
		}
		cursor := m.sectionCursor[m.activeSection]
		if cursor < 0 || cursor >= len(items) {
			cursor = 0
		}
		switch m.activeSection {
		case sectionCurrent:
			lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("target"), renderLocalDetailTarget(items[cursor])))
			if upstream := strings.TrimSpace(m.repoStatus.Upstream); upstream != "" {
				lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("upstream"), upstream))
			} else {
				lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("upstream"), warn.Render("(none)")))
			}
			if m.status.WorktreeState != "" {
				worktree := string(m.status.WorktreeState)
				if m.status.WorktreeState == state.WorktreeStateDirty {
					worktree = dirtyMark.Render(worktree)
				}
				lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("worktree"), worktree))
			}
		case sectionRemote:
			lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("target"), renderRemoteDetailTarget(items[cursor])))
			if !m.repoStatus.LastFetchAt.IsZero() {
				lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("last fetch"), compactWhenTime(m.repoStatus.LastFetchAt)))
			} else {
				lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("last fetch"), "-"))
			}
			lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("sync status"), remoteSyncSummaryForStatus(m.repoStatus)))
		case sectionTags:
			if entry, ok := selectedTagEntry(m); ok {
				lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("title"), tagDetailTitle(entry)))
				lines = append(lines, fmt.Sprintf("%s: %s", renderContextKey("target"), shorten(entry.CommitHash, 8)))
			}
		}
	}
	if len(lines) == 0 {
		lines = append(lines, muted.Render("  (empty)"))
	}
	return lines
}

func selectedTagEntry(m model) (git.TagEntry, bool) {
	if len(m.repoStatus.TagEntries) == 0 {
		return git.TagEntry{}, false
	}
	cursor := m.sectionCursor[sectionTags]
	if cursor < 0 || cursor >= len(m.repoStatus.TagEntries) {
		return git.TagEntry{}, false
	}
	entry := m.repoStatus.TagEntries[cursor]
	if entry.Name == "" {
		return git.TagEntry{}, false
	}
	return entry, true
}

func renderContextDetailListItem(label, value string) string {
	return fmt.Sprintf("%s: %s", renderContextKey(label), value)
}

func renderLocalDetailTarget(item state.TargetItem) string {
	target := item.Name
	if target == "" {
		target = "-"
	}
	target = "l->" + target
	if item.WorktreeDirty {
		target += " " + dirtyMark.Render("(dirty)")
	}
	if item.NeedsPull {
		target += " " + warn.Render("⬇")
	}
	if item.NeedsPush {
		target += " " + warn.Render("⬆")
	}
	if item.MergeConflicted {
		target += " " + conflictMark.Render("(conflict)")
	}
	return target
}

func renderRemoteDetailTarget(item state.TargetItem) string {
	target := item.Name
	if target == "" {
		target = "-"
	}
	if strings.HasPrefix(target, "origin/") {
		target = strings.TrimPrefix(target, "origin/")
	}
	return "o->" + target
}

func renderTagList(tags []string) string {
	if len(tags) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		parts = append(parts, tagColor.Render(tag))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " ")
}

func tagDetailTitle(entry git.TagEntry) string {
	if strings.TrimSpace(entry.Message) != "" {
		return entry.Message
	}
	if strings.TrimSpace(entry.Subject) != "" {
		return entry.Subject
	}
	return "-"
}

func compactWhenTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04")
}

func remoteSyncSummaryForStatus(rs git.Status) string {
	if rs.RemoteSyncSummary != "" {
		return rs.RemoteSyncSummary
	}
	switch {
	case rs.Root == "":
		return ""
	case rs.NoRemote:
		return "no remote"
	case rs.NoUpstream:
		return "no upstream"
	case rs.Detached:
		return "detached"
	}
	track := rs.Tracking[rs.Branch]
	switch {
	case track.Ahead == 0 && track.Behind == 0:
		return "synced"
	case track.Ahead > 0 && track.Behind == 0:
		return fmt.Sprintf("ahead %d", track.Ahead)
	case track.Ahead == 0 && track.Behind > 0:
		return fmt.Sprintf("behind %d", track.Behind)
	default:
		return fmt.Sprintf("diverged (%d ahead, %d behind)", track.Ahead, track.Behind)
	}
}

func (m model) renderDetailContent(width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := make([]string, 0, height)
	if m.branchOpen {
		lines = append(lines, "")
	} else if m.status.Mode == state.ModeLoading || m.status.Mode == state.ModeBlocked {
		lines = append(lines, "")
	} else {
		lines = append(lines, renderStatusCompact(m.status))
	}
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
	lines = append(lines, renderSectionTitle("Actions"))
	lines = append(lines, indentLines(renderActionHelpLines(m), 1)...)
	lines = fitBlockWidth(lines, width)
	return fitBlockLines(lines, height)
}
