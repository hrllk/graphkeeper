package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderGraphContent(width, height int) string {
	return renderGraphProjection(m.screenProjection(width, height).Graph, width, height)
}

func renderGraphProjection(p GraphProjection, width, height int) string {
	if height <= 0 {
		return ""
	}
	rows := p.Rows
	if len(rows) == 0 {
		emptyLine := muted.Render("  (no graph to show yet)")
		return fitBlockLines([]string{fitVisibleWidth(emptyLine, width)}, height)
	}
	page := p.PageSize
	if page <= 0 {
		page = max(height-2, 1)
	}
	start := clampScroll(p.Scroll, len(rows), page)
	end := start + page
	if end > len(rows) {
		end = len(rows)
	}
	lines := make([]string, 0, height)
	pageLabel := fmt.Sprintf("graph page %d-%d/%d", start+1, end, len(rows))
	legend := "S stash · T tag"
	pageLine := pageLabel
	if available := width - lipgloss.Width(pageLabel) - lipgloss.Width(legend); available >= 2 {
		pageLine += strings.Repeat(" ", available) + legend
	}
	lines = append(lines, fitVisibleWidth(muted.Render(pageLine), width))
	graphActive := p.Active
	graphColWidth := graphTopologyWidth(width)
	rawGraph := len(rows) > 0 && rows[0].Graph != ""
	if len(lines) < height {
		lines = append(lines, fitVisibleWidth(sectionTitle.Render(renderGraphHeader(width, graphColWidth)), width))
	}
	for i := start; i < end; i++ {
		if len(lines) >= height {
			break
		}
		isHandshake := rows[i].Commit.Hash != "" && p.Handshake[rows[i].Commit.Hash]
		stashCount := p.StashCounts[rows[i].Commit.Hash]
		lineStr := renderGraphLineWithSearch(rows[i], graphActive && i == p.Cursor, graphActive, p.LaneCursor, p.LocalBranches, graphColWidth, width, isHandshake, stashCount, p.SearchQuery)
		lines = append(lines, lineStr)
		if !rawGraph && i+1 < len(rows) {
			isConnectorHandshake := rows[i].Commit.Hash != "" && p.Handshake[rows[i].Commit.Hash] && rows[i+1].Commit.Hash != "" && p.Handshake[rows[i+1].Commit.Hash]
			for _, line := range renderGraphConnectorLinesWithWidth(rows[i], rows[i+1], isConnectorHandshake, graphColWidth) {
				if len(lines) >= height {
					break
				}
				if rows[i].Commit.Hash == "VIRTUAL_CONFLICT_HASH" || rows[i+1].Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
					line = strings.ReplaceAll(line, "|", conflictColor.Render("|"))
					line = strings.ReplaceAll(line, "/", conflictColor.Render("/"))
					line = strings.ReplaceAll(line, "\\", conflictColor.Render("\\"))
				}
				lines = append(lines, fitVisibleWidth(line, width))
			}
		}
	}
	return fitBlockLines(lines, height)
}

func renderGraphHeader(width, graphColWidth int) string {
	available := width - graphRowFixedWidth(graphColWidth)
	if available <= 0 {
		return ""
	}
	prefix := fmt.Sprintf("%-*s %-14s %-*s %-*s ", graphCommitWidth, "commit", "branches", graphStatusWidth, "state", graphColWidth, "graph")
	if available < graphAuthorWidthTarget+graphTitlePreferredWidth {
		return prefix + fmt.Sprintf("%-*s", available, "title")
	}
	titleWidth := available - graphAuthorWidthTarget
	return prefix + fmt.Sprintf("%-*s%-*s", graphAuthorWidthTarget, "author", titleWidth, "title")
}
