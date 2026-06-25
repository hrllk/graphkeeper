package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/graph"
)

func renderGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string, graphColWidth int, isHandshake bool, stashCount int) string {
	if row.Graph != "" {
		return renderRawGraphLine(row, selected, graphActive, laneCursor, localBranches, graphColWidth, isHandshake, stashCount)
	}
	var hash, refs string
	var refInfo decorationInfo
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		hash = "        "
		refs = "          "
	} else {
		hash = fmt.Sprintf("%-8s", shorten(row.Commit.Hash, 7))
		refInfo = compactDecorationInfo(row.Commit.Decorations, localBranches)
		refs = fmt.Sprintf("%-10s", refInfo.Text)
		isHead := hasHeadDecoration(row.Commit.Decorations)
		pointerFocused := graphActive && selected
		if isHead {
			refs = headMark.Render(refs)
		} else if pointerFocused && refInfo.HasBranch {
			refs = branchMark.Render(refs)
		}
		if shouldHighlightStash(stashCount, selected) {
			refs = stashMark.Render(refs)
		}
		if graphActive && selected {
			hash = pointerMark.Render(hash)
		}
	}
	graphCell := graphLineCell(row, graphActive, selected, laneCursor, graphColWidth)
	graphCell = padRight(graphCell, graphColWidth)
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		graphCell = strings.ReplaceAll(graphCell, "*", conflictMark.Render("*"))
		graphCell = strings.ReplaceAll(graphCell, "|", conflictColor.Render("|"))
		graphCell = strings.ReplaceAll(graphCell, "/", conflictColor.Render("/"))
		graphCell = strings.ReplaceAll(graphCell, "\\", conflictColor.Render("\\"))
	} else if isHandshake {
		pinkBg := lipgloss.NewStyle().Background(lipgloss.Color("162")).Foreground(lipgloss.Color("255")).Bold(true)
		graphCell = strings.ReplaceAll(graphCell, "*", pinkBg.Render("*"))
	}
	var when, title string
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		when = "       "
		title = conflictColor.Render(row.Commit.Subject)
	} else {
		when = fmt.Sprintf("%-7s", compactWhenText(row.Commit.RelativeAge))
		title = fmt.Sprintf("%-10s", compactTitleText(row.Commit.Subject))
	}
	line := hash + " " + refs + " " + graphCell + "  " + when + " " + title
	if selected {
		return "> " + line
	}
	return "  " + line
}

func renderRawGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string, graphColWidth int, isHandshake bool, stashCount int) string {
	if row.Commit.Hash == "" && row.Commit.Subject == "" && len(row.Commit.Decorations) == 0 && len(row.Commit.Parents) == 0 {
		graphCell := padRight(row.Graph, graphColWidth)
		if isHandshake {
			pinkBg := lipgloss.NewStyle().Background(lipgloss.Color("162")).Foreground(lipgloss.Color("255")).Bold(true)
			graphCell = strings.ReplaceAll(graphCell, "*", pinkBg.Render("*"))
		}
		line := fmt.Sprintf("%-8s %-10s %s  %-7s %-10s", "", "", graphCell, "", "")
		if selected {
			return "> " + line
		}
		return "  " + line
	}
	var hash, refs string
	var refInfo decorationInfo
	pointerFocused := false
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		hash = "        "
		refs = "          "
	} else {
		graphRunes := []rune(row.Graph)
		width := len(graphRunes)
		lane := graph.PointerLane(row)
		cursorLane := laneCursor
		if width > 0 && cursorLane >= width {
			cursorLane = width - 1
		}
		pointerFocused = graphActive && selected && cursorLane == lane
		hash = fmt.Sprintf("%-8s", shorten(row.Commit.Hash, 7))
		if pointerFocused {
			hash = pointerMark.Render(hash)
		}
		refInfo = compactDecorationInfo(row.Commit.Decorations, localBranches)
		refs = fmt.Sprintf("%-10s", refInfo.Text)
		if refInfo.HasLocalHead {
			refs = headMark.Render(refs)
		} else if pointerFocused && refInfo.HasBranch {
			refs = branchMark.Render(refs)
		}
		if shouldHighlightStash(stashCount, selected) {
			refs = stashMark.Render(refs)
		}
	}
	var graphCell string
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		var b strings.Builder
		for _, r := range row.Graph {
			charStr := string(r)
			if charStr == "*" {
				b.WriteString(conflictMark.Render(charStr))
			} else if charStr == "|" || charStr == "/" || charStr == "\\" {
				b.WriteString(conflictColor.Render(charStr))
			} else {
				b.WriteString(charStr)
			}
		}
		graphCell = b.String()
	} else {
		lane := graph.PointerLane(row)
		graphCell = highlightRawGraphPrefix(row.Graph, lane, pointerFocused, refInfo.HasLocalHead)
	}
	graphCell = padRight(graphCell, graphColWidth)
	if row.Commit.Hash != "VIRTUAL_CONFLICT_HASH" && isHandshake {
		pinkBg := lipgloss.NewStyle().Background(lipgloss.Color("162")).Foreground(lipgloss.Color("255")).Bold(true)
		graphCell = strings.ReplaceAll(graphCell, "*", pinkBg.Render("*"))
	}
	var when, title string
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		when = "       "
		title = conflictColor.Render(row.Commit.Subject)
	} else {
		when = fmt.Sprintf("%-7s", compactWhenText(row.Commit.RelativeAge))
		title = fmt.Sprintf("%-10s", compactTitleText(row.Commit.Subject))
	}
	line := hash + " " + refs + " " + graphCell + "  " + when + " " + title
	if selected {
		return "> " + line
	}
	return "  " + line
}

func shouldHighlightStash(stashCount int, selected bool) bool {
	return stashCount > 0 && selected
}

func graphLineCell(row graphRow, graphActive bool, selected bool, laneCursor int, graphColWidth int) string {
	if row.Graph == "" && row.Commit.Hash == "" {
		return ""
	}
	if row.Graph == "" {
		width := graph.RowWidth(row)
		lane := displayLane(row, width)
		cursorLane := laneCursor
		if width > 0 && cursorLane >= width {
			cursorLane = width - 1
		}
		pointerFocused := graphActive && selected && cursorLane == lane
		cells := make([]string, 0, width)
		for i := 0; i < width; i++ {
			cell := " "
			beforeActive := i < len(row.Before)
			afterActive := i < len(row.After)
			switch {
			case i == lane:
				cell = "*"
			case shouldHideConvergedDuplicateLane(row, i, lane):
				cell = " "
			case beforeActive || afterActive:
				cell = "|"
			}
			if hasHeadDecoration(row.Commit.Decorations) && i == lane {
				cell = headMark.Render(cell)
			} else if pointerFocused && i == lane {
				cell = pointerMark.Render(cell)
			}
			cells = append(cells, cell)
		}
		return padRight(strings.Join(cells, " "), graphColWidth)
	}
	graphRunes := []rune(row.Graph)
	width := len(graphRunes)
	lane := graph.PointerLane(row)
	cursorLane := laneCursor
	if width > 0 && cursorLane >= width {
		cursorLane = width - 1
	}
	pointerFocused := graphActive && selected && cursorLane == lane
	var b strings.Builder
	for i, r := range graphRunes {
		if pointerFocused && i == lane {
			b.WriteString(pointerMark.Render(string(r)))
			continue
		}
		b.WriteRune(r)
	}
	return padRight(b.String(), graphColWidth)
}

func highlightRawGraphPrefix(graph string, lane int, focused bool, hasHead bool) string {
	if !focused {
		if !hasHead {
			return graph
		}
	}
	runes := []rune(graph)
	if lane < 0 || lane >= len(runes) {
		return graph
	}
	var b strings.Builder
	for i, r := range runes {
		if i == lane {
			if hasHead {
				b.WriteString(headMark.Render(string(r)))
			} else {
				b.WriteString(pointerMark.Render(string(r)))
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func displayLane(row graphRow, width int) int {
	if width <= 1 && shouldCollapseRowDisplay(row) {
		return 0
	}
	if row.Lane < 0 {
		return 0
	}
	if width > 0 && row.Lane >= width {
		return width - 1
	}
	return row.Lane
}
