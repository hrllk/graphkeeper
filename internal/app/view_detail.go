package app

import (
	"fmt"
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
		if m.activeSection == sectionTags {
			if m.repoStatus.TagProvenanceLoaded {
				lines = append(lines, fmt.Sprintf("sync: %s", tagSyncSummaryLabel(m.repoStatus.TagSyncSummary)))
				lines = append(lines, tagSyncSummaryHelp(m.repoStatus.TagSyncSummary))
			} else if !m.tagSyncAttempted {
				lines = append(lines, muted.Render("sync: Press F to sync tag provenance."))
			}
		}
		if len(items) == 0 {
			if m.activeSection == sectionTags && !m.tagSyncAttempted {
				lines = append(lines, muted.Render("  (provenance unknown)"))
			} else {
				lines = append(lines, muted.Render("  (empty)"))
			}
			return lines
		}
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
		if m.activeSection == sectionRemote {
			if m.repoStatus.Upstream != "" {
				lines = append(lines, fmt.Sprintf("upstream: %s", shorten(m.repoStatus.Upstream, max(width-10, 0))))
			} else {
				lines = append(lines, "upstream: -")
			}
			if m.repoStatus.DefaultBranch != "" {
				lines = append(lines, fmt.Sprintf("default: %s", shorten(m.repoStatus.DefaultBranch, max(width-9, 0))))
			}
			if !m.repoStatus.LastFetchAt.IsZero() {
				lines = append(lines, fmt.Sprintf("last fetch: %s", compactWhenTime(m.repoStatus.LastFetchAt)))
			} else {
				lines = append(lines, "last fetch: -")
			}
			if summary := remoteSyncSummaryForStatus(m.repoStatus); summary != "" {
				lines = append(lines, fmt.Sprintf("sync: %s", summary))
			}
			lines = append(lines, fmt.Sprintf("branches: %d", len(m.repoStatus.RemoteBranches)))
		}
		if m.activeSection == sectionTags {
			if entry, ok := selectedTagEntry(m); ok {
				lines = append(lines, fmt.Sprintf("selected: %s", entry.Name))
				lines = append(lines, fmt.Sprintf("status: %s", tagProvenanceStateLabel(m.repoStatus.TagProvenanceLoaded, entry.OriginKnown, entry.OnOrigin)))
				lines = append(lines, fmt.Sprintf("commit: %s", shorten(entry.CommitHash, 7)))
				if entry.Annotated {
					if entry.Tagger != "" {
						lines = append(lines, fmt.Sprintf("tagger: %s", shorten(entry.Tagger, max(width-9, 0))))
					} else {
						lines = append(lines, "tagger: -")
					}
					lines = append(lines, fmt.Sprintf("tagged: %s", compactWhenTime(entry.TaggedAt)))
					if entry.Message != "" {
						lines = append(lines, fmt.Sprintf("message: %s", shorten(entry.Message, max(width-9, 0))))
					}
				} else {
					lines = append(lines, "tagger: lightweight")
					lines = append(lines, fmt.Sprintf("tagged: %s", compactWhenText(entry.RelativeAge)))
					if entry.Subject != "" {
						lines = append(lines, fmt.Sprintf("message: %s", shorten(entry.Subject, max(width-9, 0))))
					}
				}
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
