package app

import (
	"strings"

	"hrllk/graphkeeper/internal/state"
)

func (m model) renderSectionContent(section graphSection, width, height int) string {
	items := sectionTargets(m.repoStatus, section)
	if len(items) == 0 {
		return fitVisibleWidth(muted.Render("  (empty)"), width)
	}
	cursor := m.sectionCursor[section]
	var b strings.Builder
	for i, item := range items {
		if i >= height {
			break
		}
		prefix := "  "
		if i == cursor && m.activeSection == section {
			prefix = "> "
		}
		label := formatTargetItem(item)
		if label == "" {
			continue
		}
		b.WriteString(fitVisibleWidth(prefix+label, width))
		b.WriteString("\n")
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
		prefix := "  "
		if i == s.TargetIdx {
			prefix = "> "
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
		return "tag    " + t.Name
	default:
		return t.Name
	}
}

func renderActionHelpLines(m model) []string {
	switch m.status.Mode {
	case state.ModeBrowse:
		lines := make([]string, 0, 5)
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
			if pullReady(m.repoStatus) && isLocal {
				lines = append(lines, "• p: pull")
			} else {
				lines = append(lines, disabled.Render("• p: pull")+" "+muted.Render("(current branch lane)"))
			}
			lines = append(lines, "• s: reset         • ctrl+u/d: scroll")
			lines = append(lines, "• gg: top         • G: bottom")
			lines = append(lines, "• H: jump to HEAD")
			if canCreateBranch(m.repoStatus) {
				lines = append(lines, "• n: new branch")
			} else {
				lines = append(lines, disabled.Render("• n: new branch")+" "+muted.Render("(dirty)"))
			}
		case sectionCurrent, sectionRemote:
			if m.repoStatus.WorktreeDirty {
				lines = append(lines, disabled.Render("• space: checkout")+" "+muted.Render("(dirty)"))
			} else {
				lines = append(lines, "• space: checkout")
			}
			if m.activeSection == sectionCurrent {
				if pullReady(m.repoStatus) {
					lines = append(lines, "• p: pull           • P: push")
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
						lines = append(lines, disabled.Render(label)+"   "+disabled.Render(pushLabel))
					} else {
						lines = append(lines, disabled.Render(label)+"   "+pushLabel)
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
			lines = append(lines, "• no section actions")
		default:
			lines = append(lines, "• no section actions")
		}
		return lines
	case state.ModeTargetPick:
		return []string{"• up/down: choose target            • enter: preview", "• esc: back"}
	case state.ModeResetModePick:
		return []string{"• s: soft  •  m: mixed  •  h: hard", "• esc: back"}
	case state.ModeOutcomePreview:
		if m.status.CanExecute {
			return []string{"• enter: execute                    • esc: back"}
		}
		return []string{"• esc: back"}
	default:
		return []string{"• r: refresh"}
	}
}
