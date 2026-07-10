package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/state"
)

func (m model) renderSectionContent(section graphSection, width, height int) string {
	items := sectionTargets(m.repoStatus, section)
	if len(items) == 0 {
		if section == sectionTags {
			lines := []string{fitVisibleWidth(muted.Render("No local tags found."), width)}
			if !m.tagSyncAttempted {
				lines = append(lines, fitVisibleWidth(muted.Render("Press F to sync tag provenance."), width))
			}
			return fitBlockLines(lines, height)
		}
		return fitVisibleWidth(muted.Render("(empty)"), width)
	}
	cursor := m.sectionCursor[section]
	start := 0
	if m.activeSection == section && height > 0 && len(items) > height {
		start = cursor - height + 1
		if start < 0 {
			start = 0
		}
		if start > len(items)-height {
			start = len(items) - height
		}
	}
	var b strings.Builder
	rendered := 0
	for i := start; i < len(items); i++ {
		if rendered >= height {
			break
		}
		item := items[i]
		prefix := ""
		if i == cursor && m.activeSection == section {
			prefix = ">"
		}
		label := formatSectionTargetItem(item, width)
		if label == "" {
			continue
		}
		b.WriteString(fitVisibleWidth(prefix+label, width))
		b.WriteString("\n")
		rendered++
	}
	return b.String()
}

func renderStatusCompact(s state.Status) string {
	msg := shorten(s.Message, 30)
	switch s.Mode {
	case state.ModeBrowse:
		return ok.Render("Browse") + " | " + msg
	case state.ModeLoading:
		return accent.Render("Loading") + " | " + msg
	case state.ModeResetModePick:
		return ok.Render("Reset")
	case state.ModeBlocked:
		return warn.Render("Blocked") + " | " + msg
	default:
		return msg
	}
}

func renderTargets(s state.Status) string {
	if len(s.Targets) == 0 {
		return muted.Render("(no targets)")
	}
	var b strings.Builder
	for i, t := range s.Targets {
		prefix := ""
		if i == s.TargetIdx {
			prefix = ">"
		}
		label := formatTargetItem(t)
		if label == "" {
			continue
		}
		b.WriteString(prefix + label + "\n")
	}
	return b.String()
}

func formatTargetItem(t state.TargetItem) string {
	switch t.Kind {
	case state.TargetKindLocal:
		if t.Current {
			label := headMark.Render("l->" + t.Name)
			if t.WorktreeDirty {
				label += " " + dirtyMark.Render("(dirty)")
			}
			if t.NeedsPull {
				label += " " + warn.Render("⬇")
			}
			if t.NeedsPush {
				label += " " + warn.Render("⬆")
			}
			if t.NoUpstream {
				label += " " + warn.Render("(no-up)")
			}
			if t.MergeConflicted {
				label += " " + conflictMark.Render("(conflict)")
			}
			return label
		}
		label := "l->" + t.Name
		if t.WorktreeDirty {
			label += " " + dirtyMark.Render("(dirty)")
		}
		if t.NeedsPull {
			label += " " + warn.Render("⬇")
		}
		if t.NeedsPush {
			label += " " + warn.Render("⬆")
		}
		if t.NoUpstream {
			label += " " + warn.Render("(no-up)")
		}
		return label
	case state.TargetKindRemote:
		name := t.Name
		if !strings.Contains(name, "/") {
			return ""
		}
		if !strings.HasSuffix(name, "/HEAD") && strings.HasPrefix(name, "origin/") {
			name = strings.TrimPrefix(name, "origin/")
		}
		label := "o->" + name
		if t.Default {
			label += " (default)"
		}
		return label
	case state.TargetKindTag:
		if t.CommitHash != "" {
			source := tagOriginStateLabel(t.OriginKnown, t.OnOrigin)
			return fmt.Sprintf("%-24s  %-8s  %-10s  %s",
				shorten(t.CommitHash, 7),
				shorten(t.Name, 8),
				compactWhenText(t.RelativeAge),
				source,
			)
		}
		return "tag    " + t.Name
	default:
		return t.Name
	}
}

func tagOriginStateLabel(originKnown, onOrigin bool) string {
	if originKnown && onOrigin {
		return remoteColor.Render("(origin)")
	}
	return muted.Render("(unknown)")
}

func formatSectionTargetItem(t state.TargetItem, width int) string {
	if width <= 0 {
		return ""
	}
	switch t.Kind {
	case state.TargetKindLocal:
		return formatSectionBranchTarget("l->", t.Name, width, t.Current, t.WorktreeDirty, t.NeedsPull, t.NeedsPush, t.NoUpstream, t.MergeConflicted)
	case state.TargetKindRemote:
		name := t.Name
		if !strings.Contains(name, "/") {
			return ""
		}
		if !strings.HasSuffix(name, "/HEAD") && strings.HasPrefix(name, "origin/") {
			name = strings.TrimPrefix(name, "origin/")
		}
		label := formatSectionBranchTarget("o->", name, width, false, false, false, false, false, false)
		if t.Default {
			label += " " + muted.Render("(default)")
		}
		return fitVisibleWidth(label, width)
	case state.TargetKindTag:
		if t.CommitHash == "" {
			return fitVisibleWidth(tagColor.Render(t.Name), width)
		}
		parts := []string{
			tagColor.Render(shorten(t.CommitHash, 7)),
			compactTagTitleText(t.Name),
		}
		if t.RelativeAge != "" {
			parts = append(parts, compactWhenText(t.RelativeAge))
		}
		parts = append(parts, tagOriginStateLabel(t.OriginKnown, t.OnOrigin))
		return fitVisibleWidth(strings.Join(parts, "  "), width)
	default:
		return fitVisibleWidth(t.Name, width)
	}
}

func compactTagTitleText(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return strings.Repeat(" ", 10)
	}
	if len(name) > 7 {
		return shorten(name, 7) + "..."
	}
	if len(name) < 10 {
		return name + strings.Repeat(" ", 10-len(name))
	}
	return name
}

func formatSectionBranchTarget(prefix, name string, width int, current, dirty, needsPull, needsPush, noUpstream, conflicted bool) string {
	base := prefix + shorten(name, max(width-lipgloss.Width(prefix), 4))
	var label string
	if current {
		label = headMark.Render(base)
	} else {
		label = base
	}
	parts := []string{label}
	if dirty {
		parts = append(parts, dirtyMark.Render("(dirty)"))
	}
	if needsPull {
		parts = append(parts, warn.Render("⬇"))
	}
	if needsPush {
		parts = append(parts, warn.Render("⬆"))
	}
	if noUpstream {
		parts = append(parts, warn.Render("(no-up)"))
	}
	if conflicted {
		parts = append(parts, conflictMark.Render("(conflict)"))
	}
	return fitVisibleWidth(strings.Join(parts, " "), width)
}

func renderActionHelpLines(m model) []string {
	switch m.status.Mode {
	case state.ModeBrowse:
		switch m.activeSection {
		case sectionGraph:
			return renderGraphActionHelpLines(m)
		case sectionCurrent:
			return renderLocalActionHelpLines(m)
		case sectionRemote:
			return renderRemoteActionHelpLines(m)
		case sectionTags:
			return renderTagActionHelpLines(m)
		default:
			return []string{"• no section actions"}
		}
	case state.ModeTargetPick:
		if m.status.Action == state.ActionCheckout {
			return []string{"• enter: checkout", "• esc: back"}
		}
		return []string{"• enter: preview", "• esc: back"}
	case state.ModeResetModePick:
		return []string{"• s: soft  •  m: mixed  •  h: hard", "• esc: back"}
	case state.ModeReview:
		return []string{"• y: continue                    • n: cancel"}
	case state.ModeOutcomePreview:
		if m.status.CanExecute {
			return []string{"• enter: execute                    • esc: back"}
		}
		return []string{"• esc: back"}
	default:
		return []string{"• r: refresh"}
	}
}

func renderGraphActionHelpLines(m model) []string {
	return []string{
		"• m: merge",
		"• r: rebase",
		"• space: checkout",
		"• H: jump to HEAD",
	}
}

func renderLocalActionHelpLines(m model) []string {
	lines := make([]string, 0, 8)
	if m.repoStatus.WorktreeDirty {
		lines = append(lines, "• s: stash changes")
		lines = append(lines, "• c: clean working tree")
	} else {
		lines = append(lines, disabled.Render("• s: stash changes")+" "+muted.Render("(dirty only)"))
		lines = append(lines, disabled.Render("• c: clean working tree")+" "+muted.Render("(dirty only)"))
	}
	if m.repoStatus.WorktreeDirty {
		lines = append(lines, disabled.Render("• space: checkout")+" "+muted.Render("(dirty)"))
	} else {
		lines = append(lines, "• space: checkout")
	}
	lines = append(lines, "• d: delete branch")
	if m.repoStatus.MergeInProgress {
		lines = append(lines, "• a: abort merge")
	} else {
		lines = append(lines, disabled.Render("• a: abort merge"))
	}
	deleteLabel := "• d: delete branch"
	if item, ok := activeSectionTargetItem(m); ok && item.Current {
		lines = append(lines, disabled.Render(deleteLabel)+" "+muted.Render("(current branch)"))
	} else {
		lines = append(lines, deleteLabel)
	}
	if pullReady(m.repoStatus) {
		lines = append(lines, "• p: pull")
		lines = append(lines, "• P: push")
	} else {
		label := "• p: pull"
		switch {
		case m.repoStatus.NoUpstream:
			label += " (no upstream)"
		case m.repoStatus.NoRemote:
			label += " (no remote)"
		case m.repoStatus.Detached:
			label += " (detached)"
		}
		pushLabel := "• P: push"
		if m.repoStatus.Detached || m.repoStatus.EmptyRepo {
			lines = append(lines, disabled.Render(label))
			lines = append(lines, disabled.Render(pushLabel))
		} else {
			lines = append(lines, disabled.Render(label))
			lines = append(lines, pushLabel)
		}
	}
	if canCreateBranch(m.repoStatus) {
		lines = append(lines, "• n: new branch")
	} else {
		lines = append(lines, disabled.Render("• n: new branch")+" "+muted.Render("(dirty)"))
	}
	return lines
}

func renderRemoteActionHelpLines(m model) []string {
	return []string{
		"• space: checkout",
		"• f: fetch",
		"• p: pull",
		"• d: delete branch",
	}
}

func renderTagActionHelpLines(m model) []string {
	return []string{
		"• enter: jump to graph",
		"• d: delete tag",
	}
}
