package app

import (
	"fmt"
	"strings"

	"hrllk/git-graph-tui/internal/state"
)

func (m model) renderGraphContent(width, height int) string {
	if height <= 0 {
		return ""
	}
	rows := graphRows(m.repoStatus)
	if len(rows) == 0 {
		return fitBlockLines([]string{muted.Render("  (no graph to show yet)")}, height)
	}
	start := clampScroll(m.graphScroll, len(rows), graphPageSize(&m))
	end := start + graphPageSize(&m)
	if end > len(rows) {
		end = len(rows)
	}
	lines := make([]string, 0, height)
	lines = append(lines, "  "+muted.Render(fmt.Sprintf("graph page %d-%d/%d", start+1, end, len(rows))))
	graphActive := m.activeSection == sectionGraph
	graphColWidth := max(18, int(float64(width)*0.30))
	rawGraph := len(rows) > 0 && rows[0].Graph != ""
	if len(lines) < height {
		lines = append(lines, "  "+muted.Render(fmt.Sprintf("%-8s %-10s %-*s %-7s %-10s", "commit", "branches", graphColWidth, "graph", "when", "title")))
	}
	for i := start; i < end; i++ {
		if len(lines) >= height {
			break
		}
		isHandshake := rows[i].Commit.Hash != "" && m.handshakeCommits[rows[i].Commit.Hash]
		lineStr := renderGraphLine(rows[i], graphActive && i == m.sectionCursor[sectionGraph], graphActive, m.graphLaneCursor, m.repoStatus.LocalBranches, graphColWidth, isHandshake)
		lines = append(lines, lineStr)
		if !rawGraph && i+1 < len(rows) {
			isConnectorHandshake := rows[i].Commit.Hash != "" && m.handshakeCommits[rows[i].Commit.Hash] && rows[i+1].Commit.Hash != "" && m.handshakeCommits[rows[i+1].Commit.Hash]
			for _, line := range renderGraphConnectorLines(rows[i], rows[i+1], isConnectorHandshake) {
				if len(lines) >= height {
					break
				}
				if rows[i].Commit.Hash == "VIRTUAL_CONFLICT_HASH" || rows[i+1].Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
					line = strings.ReplaceAll(line, "|", conflictColor.Render("|"))
					line = strings.ReplaceAll(line, "/", conflictColor.Render("/"))
					line = strings.ReplaceAll(line, "\\", conflictColor.Render("\\"))
				}
				lines = append(lines, line)
			}
		}
	}
	return fitBlockLines(lines, height)
}

func renderStatusCompact(s state.Status) string {
	msg := shorten(s.Message, 30)
	switch s.Mode {
	case state.ModeBrowse:
		return ok.Render("Browse") + " | " + msg
	case state.ModeLoading:
		return accent.Render("Loading") + " | " + msg
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
		if strings.HasSuffix(name, "/HEAD") {
			name = name
		} else if strings.HasPrefix(name, "origin/") {
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
			lines = append(lines, "• s: reset         • ctrl+u/d: scroll")
			lines = append(lines, "• gg: top         • G: bottom")
			lines = append(lines, "• H: jump to HEAD")
			lines = append(lines, "• n: new branch")
		case sectionCurrent, sectionRemote:
			lines = append(lines, "• space: checkout")
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
				lines = append(lines, "• n: new branch")
			}
		case sectionTags:
			lines = append(lines, "• no section actions")
		default:
			lines = append(lines, "• no section actions")
		}
		return lines
	case state.ModeTargetPick:
		return []string{"• up/down: choose target            • enter: preview", "• esc: back"}
	case state.ModeOutcomePreview:
		if m.status.CanExecute {
			return []string{"• enter: execute                    • esc: back"}
		}
		return []string{"• esc: back"}
	default:
		return []string{"• r: refresh"}
	}
}
