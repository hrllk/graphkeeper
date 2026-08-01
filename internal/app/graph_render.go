package app

import (
	"fmt"
	"strings"

	"hrllk/graphkeeper/internal/graph"
)

const (
	graphAuthorWidthTarget = 7
	graphTitleWidthTarget  = 20
	graphDateWidth         = 7
)

func renderGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string, graphColWidth int, rowWidth int, isHandshake bool, stashCount int) string {
	return renderGraphLineWithSearch(row, selected, graphActive, laneCursor, localBranches, graphColWidth, rowWidth, isHandshake, stashCount, "")
}

func renderGraphLineWithSearch(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string, graphColWidth int, rowWidth int, isHandshake bool, stashCount int, searchQuery string) string {
	if row.Graph != "" {
		return renderRawGraphLineWithSearch(row, selected, graphActive, laneCursor, localBranches, graphColWidth, rowWidth, isHandshake, stashCount, searchQuery)
	}
	var hash, refs string
	var refInfo decorationInfo
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		hash = "        "
		refs = "          "
	} else {
		refInfo = compactDecorationInfo(row.Commit.Decorations, localBranches)
		hash = renderSearchField(shorten(row.Commit.Hash, 7), searchQuery, 8, selected && graphActive)
		refs = renderSearchField(refInfo.Text, searchQuery, graphBranchFieldWidth, selected && graphActive)
		isHead := hasHeadDecoration(row.Commit.Decorations)
		pointerFocused := graphActive && selected
		if searchQuery == "" && isHead {
			refs = headMark.Render(refs)
		} else if searchQuery == "" && pointerFocused && refInfo.HasBranch {
			refs = branchMark.Render(refs)
		}
	}
	graphCell := graphLineCell(row, graphActive, selected, laneCursor, graphColWidth, stashCount)
	graphCell = padRight(graphCell, graphColWidth)
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		graphCell = strings.ReplaceAll(graphCell, "*", conflictMark.Render("*"))
		graphCell = strings.ReplaceAll(graphCell, "|", conflictColor.Render("|"))
		graphCell = strings.ReplaceAll(graphCell, "/", conflictColor.Render("/"))
		graphCell = strings.ReplaceAll(graphCell, "\\", conflictColor.Render("\\"))
	} else if isHandshake {
		graphCell = applyHandshakePoint(graphCell, stashCount, len(row.Commit.Tags))
	}
	status := strings.Repeat(" ", graphStatusWidth)
	if row.Commit.Hash != "VIRTUAL_CONFLICT_HASH" {
		status = renderGraphStatus(stashCount, len(row.Commit.Tags))
	}
	var when, title string
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		when = "       "
		title = conflictColor.Render(row.Commit.Subject)
	} else {
		when = fmt.Sprintf("%-7s", compactWhenText(row.Commit.RelativeAge))
		title = renderGraphTitleWithAuthor(row.Commit.Author, row.Commit.Subject, searchQuery, rowWidth, graphColWidth, selected && graphActive)
	}
	line := hash + " " + refs + " " + status + " " + graphCell + " " + when + " " + title
	return fitVisibleWidth(line, rowWidth)
}

func renderRawGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string, graphColWidth int, rowWidth int, isHandshake bool, stashCount int) string {
	return renderRawGraphLineWithSearch(row, selected, graphActive, laneCursor, localBranches, graphColWidth, rowWidth, isHandshake, stashCount, "")
}

func renderRawGraphLineWithSearch(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string, graphColWidth int, rowWidth int, isHandshake bool, stashCount int, searchQuery string) string {
	if row.Commit.Hash == "" && row.Commit.Subject == "" && len(row.Commit.Decorations) == 0 && len(row.Commit.Parents) == 0 {
		graphCell := padRight(row.Graph, graphColWidth)
		if isHandshake {
			graphCell = applyHandshakePoint(graphCell, stashCount, 0)
		}
		line := fmt.Sprintf("%-8s %-14s %-5s %-*s %-7s %s", "", "", "", graphColWidth, graphCell, "", "")
		return fitVisibleWidth(line, rowWidth)
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
		hash = renderSearchField(shorten(row.Commit.Hash, 7), searchQuery, 8, selected && graphActive)
		if searchQuery == "" && pointerFocused {
			hash = pointerMark.Render(hash)
		}
		refInfo = compactDecorationInfo(row.Commit.Decorations, localBranches)
		refs = renderSearchField(refInfo.Text, searchQuery, graphBranchFieldWidth, selected && graphActive)
		if searchQuery == "" && refInfo.HasLocalHead {
			refs = headMark.Render(refs)
		} else if searchQuery == "" && pointerFocused && refInfo.HasBranch {
			refs = branchMark.Render(refs)
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
		graphCell = highlightRawGraphPrefix(row.Graph, lane, pointerFocused, refInfo.HasLocalHead, stashCount, len(row.Commit.Tags))
	}
	graphCell = padRight(graphCell, graphColWidth)
	if row.Commit.Hash != "VIRTUAL_CONFLICT_HASH" && isHandshake {
		graphCell = applyHandshakePoint(graphCell, stashCount, len(row.Commit.Tags))
	}
	var when, title string
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		when = "       "
		title = conflictColor.Render(row.Commit.Subject)
	} else {
		when = fmt.Sprintf("%-7s", compactWhenText(row.Commit.RelativeAge))
		title = renderGraphTitleWithAuthor(row.Commit.Author, row.Commit.Subject, searchQuery, rowWidth, graphColWidth, selected && graphActive)
	}
	status := "   "
	if row.Commit.Hash != "VIRTUAL_CONFLICT_HASH" {
		status = renderGraphStatus(stashCount, len(row.Commit.Tags))
	}
	line := hash + " " + refs + " " + status + " " + graphCell + " " + when + " " + title
	return fitVisibleWidth(line, rowWidth)
}

func applyHandshakePoint(graphCell string, stashCount, tagCount int) string {
	point := "*"
	switch {
	case stashCount > 0 && tagCount > 0:
		point = tagOverlapColor.Render(point)
	case stashCount > 0:
		point = stashMark.Render(point)
	case tagCount > 0:
		point = tagColor.Render(point)
	}
	return strings.Replace(graphCell, point, handshakeMark.Render(point), 1)
}

func renderGraphTitleWithAuthor(author, subject, searchQuery string, rowWidth, graphColWidth int, focused bool) string {
	author = compactAuthorText(author)
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = "-"
	}
	available := rowWidth - graphRowFixedWidth(graphColWidth)
	if available <= 0 {
		return ""
	}
	if available < graphAuthorWidthTarget+graphTitlePreferredWidth {
		return fitGraphTitle(renderSearchField(subject, searchQuery, available, focused), available)
	}

	authorWidth := graphAuthorWidthTarget
	titleWidth := available - authorWidth
	renderedAuthor := fitGraphTitle(renderSearchField(author, searchQuery, authorWidth, focused), authorWidth)
	renderedTitle := fitGraphTitle(renderSearchField(subject, searchQuery, titleWidth, focused), titleWidth)
	return renderedAuthor + renderedTitle
}

func graphLineCell(row graphRow, graphActive bool, selected bool, laneCursor int, graphColWidth int, stashCount int) string {
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
		tagCount := len(row.Commit.Tags)
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
			if i == lane {
				switch {
				case stashCount > 0 && tagCount > 0:
					cell = tagOverlapColor.Render(cell)
				case stashCount > 0:
					cell = stashMark.Render(cell)
				case tagCount > 0:
					cell = tagColor.Render(cell)
				case hasHeadDecoration(row.Commit.Decorations):
					cell = headMark.Render(cell)
				case pointerFocused:
					cell = pointerMark.Render(cell)
				}
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
	tagCount := len(row.Commit.Tags)
	var b strings.Builder
	for i, r := range graphRunes {
		if i == lane {
			switch {
			case stashCount > 0 && tagCount > 0:
				b.WriteString(tagOverlapColor.Render(string(r)))
			case stashCount > 0:
				b.WriteString(stashMark.Render(string(r)))
			case tagCount > 0:
				b.WriteString(tagColor.Render(string(r)))
			case pointerFocused:
				b.WriteString(pointerMark.Render(string(r)))
			default:
				b.WriteRune(r)
			}
			continue
		}
		b.WriteRune(r)
	}
	return padRight(b.String(), graphColWidth)
}

func highlightRawGraphPrefix(graph string, lane int, focused bool, hasHead bool, stashCount int, tagCount int) string {
	if !focused {
		if !hasHead && stashCount == 0 && tagCount == 0 {
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
			switch {
			case stashCount > 0 && tagCount > 0:
				b.WriteString(tagOverlapColor.Render(string(r)))
			case stashCount > 0:
				b.WriteString(stashMark.Render(string(r)))
			case tagCount > 0:
				b.WriteString(tagColor.Render(string(r)))
			case hasHead:
				b.WriteString(headMark.Render(string(r)))
			case focused:
				b.WriteString(pointerMark.Render(string(r)))
			default:
				b.WriteRune(r)
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
