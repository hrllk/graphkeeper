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
			source := muted.Render("(unknown)")
			if t.OriginKnown && t.OnOrigin {
				source = remoteColor.Render("(origin)")
			} else if t.OriginKnown {
				source = warn.Render("(no-up)")
			}
			return fmt.Sprintf("%-24s  %-28s  %-10s  %s",
				t.Name,
				compactTitleText(t.Subject),
				compactWhenText(t.RelativeAge),
				source,
			)
		}
		return "tag    " + t.Name
	default:
		return t.Name
	}
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
			tagColor.Render(shorten(t.Name, max(width-18, 4))),
		}
		if t.Subject != "" {
			parts = append(parts, compactTitleText(t.Subject))
		}
		if t.RelativeAge != "" {
			parts = append(parts, compactWhenText(t.RelativeAge))
		}
		source := muted.Render("(unknown)")
		if t.OriginKnown && t.OnOrigin {
			source = remoteColor.Render("(origin)")
		} else if t.OriginKnown {
			source = warn.Render("(no-up)")
		}
		parts = append(parts, source)
		return fitVisibleWidth(strings.Join(parts, "  "), width)
	default:
		return fitVisibleWidth(t.Name, width)
	}
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
		lines := make([]string, 0, 8)
		switch m.activeSection {
		case sectionGraph:
			isLocal := isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor)
			mergeLabel := "• m: merge"
			rebaseLabel := "• r: rebase"
			if isLocal {
				lines = append(lines, mergeLabel+"         "+rebaseLabel)
			} else {
				lines = append(lines, disabled.Render(mergeLabel)+"         "+disabled.Render(rebaseLabel)+" "+muted.Render("(local lane only)"))
			}
			if _, ok := graphCheckoutTarget(m); ok {
				if m.repoStatus.WorktreeDirty {
					lines = append(lines, disabled.Render("• space: checkout")+" "+muted.Render("(dirty)"))
				} else {
					lines = append(lines, "• space: checkout")
				}
			} else {
				lines = append(lines, disabled.Render("• space: checkout")+" "+muted.Render("(local lane only)"))
			}
			if pullReady(m.repoStatus) && isLocal {
				lines = append(lines, "• p: pull")
			} else {
				lines = append(lines, disabled.Render("• p: pull")+" "+muted.Render("(current branch lane)"))
			}
			lines = append(lines, "• s: reset         • ctrl+u/d: scroll")
			lines = append(lines, "• gg: top         • G: bottom")
			lines = append(lines, "• H: jump to HEAD")
			deleteLabel := "• d: delete branch"
			if !isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
				lines = append(lines, disabled.Render(deleteLabel)+" "+muted.Render("(local lane only)"))
			} else {
				target, _ := graphCheckoutTarget(m)
				if target != "" && !m.repoStatus.Detached && target == m.repoStatus.Branch {
					lines = append(lines, disabled.Render(deleteLabel)+" "+muted.Render("(current branch)"))
				} else {
					lines = append(lines, deleteLabel)
				}
			}
			if strings.TrimSpace(m.graphSearchQuery) != "" {
				lines = append(lines, "• /: search          • n/N: repeat search")
			} else if canCreateBranch(m.repoStatus) {
				lines = append(lines, "• / ?: search        • n: new branch")
			} else {
				lines = append(lines, "• / ?: search        "+disabled.Render("• n: new branch")+" "+muted.Render("(dirty)"))
			}
			popEntries := graphStashPopEntriesForFocus(m)
			if len(popEntries) > 0 {
				lines = append(lines, "• t: tag commit   • o: pop stash")
			} else if graphFocusIsHead(m) {
				line := "• t: tag commit   " + disabled.Render("• o: pop stash") + " " + muted.Render("(no stash)")
				lines = append(lines, line)
			} else {
				line := "• t: tag commit   " + disabled.Render("• o: pop stash") + " " + muted.Render("(HEAD only)")
				lines = append(lines, line)
			}
			return lines
		case sectionCurrent, sectionRemote:
			if m.activeSection == sectionCurrent {
				if m.repoStatus.WorktreeDirty {
					lines = append(lines, "• s: stash changes")
					lines = append(lines, "• c: clean working tree")
				} else {
					lines = append(lines, disabled.Render("• s: stash changes")+" "+muted.Render("(dirty only)"))
					lines = append(lines, disabled.Render("• c: clean working tree")+" "+muted.Render("(dirty only)"))
				}
			}
			if m.repoStatus.WorktreeDirty {
				lines = append(lines, disabled.Render("• space: checkout")+" "+muted.Render("(dirty)"))
			} else {
				lines = append(lines, "• space: checkout")
			}
			deleteLabel := "• d: delete branch"
			if item, ok := activeSectionTargetItem(m); ok && m.activeSection == sectionCurrent && item.Current {
				lines = append(lines, disabled.Render(deleteLabel)+" "+muted.Render("(current branch)"))
			} else {
				lines = append(lines, deleteLabel)
			}
			if m.activeSection == sectionCurrent {
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
				if m.repoStatus.MergeInProgress {
					lines = append(lines, "• a: abort merge")
				} else {
					lines = append(lines, disabled.Render("• a: abort merge"))
				}
			}
			if m.activeSection == sectionCurrent {
				if canCreateBranch(m.repoStatus) {
					lines = append(lines, "• n: new branch")
				} else {
					lines = append(lines, disabled.Render("• n: new branch")+" "+muted.Render("(dirty)"))
				}
			}
		case sectionTags:
			lines = append(lines, "• enter: jump to graph")
			lines = append(lines, "• d: delete tag")
		default:
			lines = append(lines, "• no section actions")
		}
		return lines
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
