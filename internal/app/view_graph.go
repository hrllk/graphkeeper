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
	lines := make([]string, 0, height)
	if p.StateHint != "" {
		lines = append(lines, fitRepositoryStateHint(p.StateHint, width))
	}
	rows := p.Rows
	if len(rows) == 0 {
		emptyLine := muted.Render("  (no graph to show yet)")
		if len(lines) < height {
			lines = append(lines, fitVisibleWidth(emptyLine, width))
		}
		return fitBlockLines(lines, height)
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
	// Measured once, over the whole graph, and handed to the legend, the header
	// and every row so all three agree. See measureGraphColumns for why the whole
	// graph rather than the visible window.
	cols := measureGraphColumns(rows, width, p.StashCounts)
	pageLabel := fmt.Sprintf("graph page %d-%d/%d", start+1, end, len(rows))
	// The legend explains the state column, so it goes when the column does.
	// Advertising S and T while the column that carries them has zero width
	// tells the user to look for something that is not on screen.
	legend := ""
	if cols.Status > 0 {
		legend = "S stash · T tag"
	}
	pageLine := pageLabel
	if available := width - lipgloss.Width(pageLabel) - lipgloss.Width(legend); available >= 2 {
		pageLine += strings.Repeat(" ", available) + legend
	}
	lines = append(lines, fitVisibleWidth(muted.Render(pageLine), width))
	graphActive := p.Active
	rawGraph := len(rows) > 0 && rows[0].Graph != ""
	if len(lines) < height {
		lines = append(lines, fitVisibleWidth(sectionTitle.Render(renderGraphHeader(width, cols)), width))
	}
	for i := start; i < end; i++ {
		if len(lines) >= height {
			break
		}
		hash := rows[i].Commit.Hash
		marks := graphRowMarks{
			Handshake:   hash != "" && p.Handshake[hash],
			StashCount:  p.StashCounts[hash],
			RangeMember: hash != "" && p.RangeMembers[hash],
			RangeAnchor: hash != "" && hash == p.RangeAnchor,
		}
		lineStr := renderGraphLineWithSearch(rows[i], graphActive && i == p.Cursor, graphActive, p.LaneCursor, p.LocalBranchInventory, cols, width, marks, p.SearchQuery)
		lines = append(lines, lineStr)
		if !rawGraph && i+1 < len(rows) {
			isConnectorHandshake := rows[i].Commit.Hash != "" && p.Handshake[rows[i].Commit.Hash] && rows[i+1].Commit.Hash != "" && p.Handshake[rows[i+1].Commit.Hash]
			for _, line := range renderGraphConnectorLinesWithWidth(rows[i], rows[i+1], isConnectorHandshake, cols.Topology) {
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

func fitRepositoryStateHint(hint string, width int) string {
	return fitVisibleWidth(warn.Render(hint), width)
}

func renderGraphHeader(width int, cols graphColumnWidths) string {
	available := width - graphRowFixedWidth(cols)
	if available <= 0 {
		return ""
	}
	prefix := fmt.Sprintf("%-*s %-14s ", graphCommitWidth, "commit", "branches")
	if cols.Status > 0 {
		prefix += fmt.Sprintf("%-*s ", cols.Status, "state")
	}
	prefix += fmt.Sprintf("%-*s ", cols.Topology, "graph")
	if cols.Date > 0 {
		prefix += fmt.Sprintf("%-*s ", cols.Date, "date")
	}
	if available < graphAuthorWidthTarget+graphTitlePreferredWidth {
		return prefix + fmt.Sprintf("%-*s", available, "title")
	}
	titleWidth := available - graphAuthorWidthTarget
	return prefix + fmt.Sprintf("%-*s%-*s", graphAuthorWidthTarget, "author", titleWidth, "title")
}
