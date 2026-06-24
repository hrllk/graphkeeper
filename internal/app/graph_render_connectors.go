package app

import (
	"strings"

	"hrllk/git-graph-tui/internal/graph"
)

func renderGraphConnectorLines(current, next graphRow, isHandshake bool) []string {
	if shouldCollapseRowDisplay(next) {
		return collapseConnectorLines(current, isHandshake)
	}
	if lines := parentShiftConnectorLines(current, next, isHandshake); len(lines) > 0 {
		return lines
	}
	return nil
}

func collapseConnectorLines(current graphRow, isHandshake bool) []string {
	width := len(current.After)
	if width <= 1 {
		return nil
	}
	if width == 2 {
		return []string{renderGraphSpacer([]string{"|", "/"}, isHandshake)}
	}
	lines := make([]string, 0, width)
	full := make([]string, width)
	for i := range full {
		full[i] = "|"
	}
	lines = append(lines, renderGraphSpacer(full, isHandshake))
	for w := width; w >= 2; w-- {
		cells := make([]string, w)
		for i := range cells {
			cells[i] = "|"
		}
		cells[w-1] = "/"
		lines = append(lines, renderGraphSpacer(cells, isHandshake))
	}
	return lines
}

func parentShiftConnectorLines(current, next graphRow, isHandshake bool) []string {
	width := max(len(current.After), graph.RowWidth(next))
	if width <= 1 {
		return nil
	}
	targetLane := displayLane(next, width)
	for sourceLane := len(current.After) - 1; sourceLane >= 0; sourceLane-- {
		if current.After[sourceLane].Hash != next.Commit.Hash || sourceLane == targetLane {
			continue
		}
		cells := make([]string, width)
		for i := range cells {
			if i < len(current.After) {
				cells[i] = "|"
			} else {
				cells[i] = " "
			}
		}
		if sourceLane > targetLane {
			cells[sourceLane] = "/"
		} else {
			cells[sourceLane] = "\\"
		}
		full := make([]string, width)
		for i := range full {
			if i < len(current.After) {
				full[i] = "|"
			} else {
				full[i] = " "
			}
		}
		return []string{renderGraphSpacer(full, isHandshake), renderGraphSpacer(cells, isHandshake)}
	}
	return nil
}

func shouldHideConvergedDuplicateLane(row graphRow, idx, displayLane int) bool {
	if idx == displayLane || idx >= len(row.Before) {
		return false
	}
	if row.Before[idx].Hash == "" || row.Before[idx].Hash != row.Commit.Hash {
		return false
	}
	if idx < len(row.After) && row.After[idx].Hash != "" {
		return false
	}
	return true
}

func renderGraphSpacer(cells []string, isHandshake bool) string {
	prefix := strings.Repeat(" ", 8) + " " + strings.Repeat(" ", 16) + " "
	return "  " + prefix + strings.Join(cells, " ")
}

func shouldCollapseRowDisplay(row graphRow) bool {
	if len(row.Before) <= 1 {
		return false
	}
	for _, ref := range row.Before {
		if ref.Hash != row.Commit.Hash {
			return false
		}
	}
	return true
}
