package app

import (
	"fmt"
)

func (m model) renderDetailContent(width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := make([]string, 0, height)
	lines = append(lines, title.Render("Mode"))
	lines = append(lines, renderStatusCompact(m.status))
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
