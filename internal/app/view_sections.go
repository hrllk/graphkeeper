package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/state"
)

const (
	tagHashColumnWidth = 7
	tagNameColumnWidth = 10
	tagAgeColumnWidth  = 3
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
	header := ""
	if section == sectionTags {
		header = renderTagSectionHeader(width)
		height--
	}
	if height <= 0 {
		if header != "" {
			return fitBlockLines([]string{header}, 1)
		}
		return ""
	}
	visibleHeight := height
	hasOverflow := len(items) > height
	if hasOverflow {
		visibleHeight = max(height-1, 0)
	}
	if m.activeSection == section && visibleHeight > 0 && len(items) > visibleHeight {
		start = cursor - visibleHeight + 1
		if start < 0 {
			start = 0
		}
		if start > len(items)-visibleHeight {
			start = len(items) - visibleHeight
		}
	}
	lines := make([]string, 0, height+1)
	if header != "" {
		lines = append(lines, header)
	}
	rendered := 0
	for i := start; i < len(items); i++ {
		if rendered >= visibleHeight {
			break
		}
		item := items[i]
		label := formatSectionTargetItem(item, width)
		if label == "" {
			continue
		}
		lines = append(lines, renderSelectableSectionItem(label, i == cursor && m.activeSection == section, width))
		rendered++
	}
	if hasOverflow {
		hidden := len(items) - rendered
		if hidden > 0 {
			lines = append(lines, fitVisibleWidth(muted.Render(fmt.Sprintf("… +%d", hidden)), width))
		}
	}
	return strings.Join(lines, "\n")
}

func renderStatusCompact(s state.Status) string {
	msg := shorten(s.Message, 30)
	switch s.Mode {
	case state.ModeBrowse:
		return ok.Render("Browse") + " | " + msg
	case state.ModeLoading:
		return accent.Render("Loading") + " | " + msg
	case state.ModeCherryPickPick:
		return ok.Render("Cherry-pick") + " | " + msg
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
		label := formatTargetItem(t)
		if label == "" {
			continue
		}
		b.WriteString(renderSelectableSectionText(label, i == s.TargetIdx))
		b.WriteString("\n")
	}
	return b.String()
}

func renderSelectableSectionItem(label string, selected bool, width int) string {
	return fitVisibleWidth(renderSelectableSectionText(label, selected), width)
}

func renderSelectableSectionText(label string, selected bool) string {
	if selected {
		return branchMark.Render(label)
	}
	return label
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
			source := renderTagProvenanceStateLabel(t.ProvenanceLoaded, t.OriginKnown, t.OnOrigin)
			parts := []string{
				padRight(shorten(t.CommitHash, 7), tagHashColumnWidth),
				padRight(shorten(t.Name, 8), tagNameColumnWidth),
				padRight(compactWhenText(t.RelativeAge), tagAgeColumnWidth),
				source,
			}
			return strings.Join(parts, "  ")
		}
		return "tag    " + t.Name
	default:
		return t.Name
	}
}

func renderTagSectionHeader(width int) string {
	parts := []string{
		padRight(renderContextKey("hash"), tagHashColumnWidth),
		padRight(renderContextKey("name"), tagNameColumnWidth),
		padRight(renderContextKey("age"), tagAgeColumnWidth),
		renderContextKey("status"),
	}
	return fitVisibleWidth(strings.Join(parts, "  "), width)
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
		parts = append(parts, renderTagProvenanceStateLabel(t.ProvenanceLoaded, t.OriginKnown, t.OnOrigin))
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
		renderHotkeyLine("m", "merge"),
		renderHotkeyLine("r", "rebase"),
		renderHotkeyLine("space", "checkout"),
		renderHotkeyLine("H", "jump to HEAD"),
	}
}

func renderLocalActionHelpLines(m model) []string {
	lines := make([]string, 0, 8)
	if m.repoStatus.WorktreeDirty {
		lines = append(lines, renderHotkeyLine("s", "stash changes"))
		lines = append(lines, renderHotkeyLine("c", "clean working tree"))
	} else {
		lines = append(lines, renderDisabledHotkeyLine("s", "stash changes")+" "+muted.Render("(dirty only)"))
		lines = append(lines, renderDisabledHotkeyLine("c", "clean working tree")+" "+muted.Render("(dirty only)"))
	}
	if m.repoStatus.WorktreeDirty {
		lines = append(lines, renderDisabledHotkeyLine("space", "checkout")+" "+muted.Render("(dirty)"))
	} else {
		lines = append(lines, renderHotkeyLine("space", "checkout"))
	}
	if m.repoStatus.MergeInProgress {
		lines = append(lines, renderHotkeyLine("a", "abort merge"))
	} else {
		lines = append(lines, renderDisabledHotkeyLine("a", "abort merge"))
	}
	deleteLabel := renderHotkeyLine("d", "delete branch")
	if item, ok := activeSectionTargetItem(m); ok && item.Current {
		lines = append(lines, renderDisabledHotkeyLine("d", "delete branch")+" "+muted.Render("(current branch)"))
	} else {
		lines = append(lines, deleteLabel)
	}
	if pullReady(m.repoStatus) {
		lines = append(lines, renderHotkeyLine("p", "pull"))
		lines = append(lines, renderHotkeyLine("P", "push"))
	} else {
		if m.repoStatus.Detached || m.repoStatus.EmptyRepo {
			lines = append(lines, renderDisabledHotkeyLine("p", "pull"))
			lines = append(lines, renderDisabledHotkeyLine("P", "push"))
		} else {
			lines = append(lines, renderDisabledHotkeyLine("p", "pull"))
			lines = append(lines, renderHotkeyLine("P", "push"))
		}
	}
	if canCreateBranch(m.repoStatus) {
		lines = append(lines, renderHotkeyLine("n", "new branch"))
	} else {
		lines = append(lines, renderDisabledHotkeyLine("n", "new branch")+" "+muted.Render("(dirty)"))
	}
	return lines
}

func renderRemoteActionHelpLines(m model) []string {
	return []string{
		renderHotkeyLine("space", "checkout"),
		renderHotkeyLine("f", "fetch"),
		renderHotkeyLine("p", "pull"),
		renderHotkeyLine("d", "delete branch"),
	}
}

func renderTagActionHelpLines(m model) []string {
	return []string{
		renderHotkeyLine("enter", "jump to graph"),
		renderHotkeyLine("d", "delete tag"),
		renderHotkeyLine("D", "delete remote tag"),
	}
}
