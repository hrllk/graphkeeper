package app

import (
	"fmt"
	"strings"
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
