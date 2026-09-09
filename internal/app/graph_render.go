package app

import (
	"fmt"
	"strings"

	"hrllk/graphkeeper/internal/graph"
)

const (
	graphAuthorWidthTarget = 7
	graphTitleWidthTarget  = 20
)

func renderInventory(value any) LocalBranchInventory {
	switch v := value.(type) {
	case LocalBranchInventory:
		return v
	case []string:
		return LocalBranchInventory{Names: v, Known: false, Fresh: false}
	default:
		return LocalBranchInventory{Known: false, Fresh: false}
	}
}

// graphRowMarks carries the per-row signals the row renderers need.
//
// It exists so the render call does not grow a third and fourth same-typed
// positional argument that a caller can transpose silently. view_graph.go's call
// already passed ten positionals with graphColWidth/rowWidth adjacent ints and
// isHandshake/stashCount an adjacent bool/int; adding RangeMember and
// RangeAnchor as bare bools would have made three consecutive same-typed pairs,
// and anchor and member render as similar marks, so a swap would compile and
// look plausible on screen.
type graphRowMarks struct {
	Handshake   bool
	StashCount  int
	RangeMember bool
	RangeAnchor bool
}

func renderGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, inventory any, cols graphColumnWidths, rowWidth int, marks graphRowMarks) string {
	return renderGraphLineWithSearch(row, selected, graphActive, laneCursor, inventory, cols, rowWidth, marks, "")
}

// renderGraphHashField owns the commit-hash cell for both the compact and the
// raw render path, so the cursor signal cannot drift between them.
//
// Before this existed, `selected` reached renderSearchField, which forwards it to
// highlightSearchText, which returns the value untouched when the search query is
// empty (graph_search_render.go:12-14). Outside a search the flag threaded through
// eight call sites and did nothing, and the cursor was conveyed by colour alone.
//
// Under NO_COLOR lipgloss selects the Ascii profile and emits nothing at all -
// attributes included - so a lipgloss style cannot carry the signal there. The
// cell writes reverse video directly instead, matching how the Inspector already
// handles NO_COLOR itself (commit_inspector.go:352, :441, :466). no-color.org
// governs colour; reverse is an attribute. A "> " gutter is not an option:
// 094ca87 (2026-07-07) removed exactly that and locked it with a test.
// Marker priority, highest wins. A row can carry several of these at once, so
// the order is fixed here rather than left to whichever branch runs last:
//
//	1  search focus   reverse+bold, owned by renderSearchField
//	2  cursor         reverse            (\x1b[7m)
//	3  road anchor    underline+bold     (\x1b[4m\x1b[1m)
//	4  road member    underline          (\x1b[4m)
//	5  HEAD           headMark, on the branches cell
//	6  stash/tag      S/T characters, in the state column
//
// Attributes compose, so an anchor that is also the cursor gets reverse and
// underline and bold together, and one reset clears them.
//
// These are written as escapes rather than lipgloss styles on purpose. Under
// NO_COLOR lipgloss selects the Ascii profile and emits nothing at all,
// attributes included, so searchMatchMark (theme.go:74, a lipgloss underline)
// would vanish exactly where the signal matters most. The Inspector already
// writes escapes directly for the same reason (commit_inspector.go:352, :441).
// no-color.org governs colour; reverse and underline are attributes.
//
// A gutter glyph is not an option either: 094ca87 removed the graph selection
// arrow and locked it with TestRenderGraphContentOmitsSelectionArrow and
// TestRenderGraphContentStartsAtLeftEdge.
func renderGraphHashField(hash, searchQuery string, focused bool, marks graphRowMarks) string {
	field := renderSearchField(hash, searchQuery, graphCommitWidth, focused)
	// Priority 1: an active search owns the cell, so nothing else marks it.
	if strings.TrimSpace(searchQuery) != "" {
		return field
	}
	attrs := ""
	if focused {
		attrs += cursorSignalPrefix
	}
	switch {
	case marks.RangeAnchor:
		attrs += underlineSignalPrefix + "\x1b[1m"
	case marks.RangeMember:
		attrs += underlineSignalPrefix
	}
	if attrs == "" {
		return field
	}
	// The cursor alone was previously drawn only under NO_COLOR, because with
	// colour available the surrounding styles already conveyed it. The road
	// marks have no such fallback, so they are always drawn.
	if focused && !marks.RangeAnchor && !marks.RangeMember && !noColorEnabled() {
		return field
	}
	return attrs + field + cursorSignalReset
}

func renderGraphLineWithSearch(row graphRow, selected bool, graphActive bool, laneCursor int, inventory any, cols graphColumnWidths, rowWidth int, marks graphRowMarks, searchQuery string) string {
	if row.Graph != "" {
		return renderRawGraphLineWithSearch(row, selected, graphActive, laneCursor, inventory, cols, rowWidth, marks, searchQuery)
	}
	var hash, refs string
	var refInfo decorationInfo
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		hash = strings.Repeat(" ", graphCommitWidth)
		refs = "          "
	} else {
		refInfo = compactDecorationInfo(row.Commit.Decorations, renderInventory(inventory))
		hash = renderGraphHashField(shorten(row.Commit.Hash, 5), searchQuery, selected && graphActive, marks)
		refs = renderSearchField(refInfo.Text, searchQuery, graphBranchFieldWidth, selected && graphActive)
		isHead := hasHeadDecoration(row.Commit.Decorations)
		pointerFocused := graphActive && selected
		if searchQuery == "" && isHead {
			refs = headMark.Render(refs)
		} else if searchQuery == "" && pointerFocused && refInfo.HasBranch {
			refs = branchMark.Render(refs)
		}
	}
	graphCell := graphLineCell(row, graphActive, selected, laneCursor, cols.Topology, marks.StashCount)
	graphCell = padRight(graphCell, cols.Topology)
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		graphCell = strings.ReplaceAll(graphCell, "*", conflictMark.Render("*"))
		graphCell = strings.ReplaceAll(graphCell, "|", conflictColor.Render("|"))
		graphCell = strings.ReplaceAll(graphCell, "/", conflictColor.Render("/"))
		graphCell = strings.ReplaceAll(graphCell, "\\", conflictColor.Render("\\"))
	} else if marks.Handshake {
		graphCell = applyHandshakePoint(graphCell, marks.StashCount, len(row.Commit.Tags))
	}
	status := ""
	if cols.Status > 0 {
		status = strings.Repeat(" ", cols.Status)
		if row.Commit.Hash != "VIRTUAL_CONFLICT_HASH" {
			status = renderGraphStatus(marks.StashCount, len(row.Commit.Tags))
		}
		status += " "
	}
	var title string
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		title = conflictColor.Render(row.Commit.Subject)
	} else {
		title = renderGraphTitleWithAuthor(row.Commit.Author, row.Commit.Subject, searchQuery, rowWidth, cols, selected && graphActive)
	}
	when := ""
	if cols.Date > 0 {
		when = padRight(graphDateText(row.Commit.CommitDate), cols.Date) + " "
	}
	line := hash + " " + refs + " " + status + graphCell + " " + when + title
	return fitVisibleWidth(line, rowWidth)
}

func renderRawGraphLineWithSearch(row graphRow, selected bool, graphActive bool, laneCursor int, inventory any, cols graphColumnWidths, rowWidth int, marks graphRowMarks, searchQuery string) string {
	if row.Commit.Hash == "" && row.Commit.Subject == "" && len(row.Commit.Decorations) == 0 && len(row.Commit.Parents) == 0 {
		graphCell := padRight(row.Graph, cols.Topology)
		if marks.Handshake {
			graphCell = applyHandshakePoint(graphCell, marks.StashCount, 0)
		}
		line := fmt.Sprintf("%-*s %-*s %-*s %-*s %s", graphCommitWidth, "", graphBranchFieldWidth, "", graphStatusWidth, "", cols.Topology, graphCell, "")
		return fitVisibleWidth(line, rowWidth)
	}
	var hash, refs string
	var refInfo decorationInfo
	pointerFocused := false
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		hash = strings.Repeat(" ", graphCommitWidth)
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
		hash = renderGraphHashField(shorten(row.Commit.Hash, 5), searchQuery, selected && graphActive, marks)
		if searchQuery == "" && pointerFocused {
			hash = pointerMark.Render(hash)
		}
		refInfo = compactDecorationInfo(row.Commit.Decorations, renderInventory(inventory))
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
		graphCell = highlightRawGraphPrefix(row.Graph, lane, pointerFocused, refInfo.HasLocalHead, marks.StashCount, len(row.Commit.Tags))
	}
	graphCell = padRight(graphCell, cols.Topology)
	if row.Commit.Hash != "VIRTUAL_CONFLICT_HASH" && marks.Handshake {
		graphCell = applyHandshakePoint(graphCell, marks.StashCount, len(row.Commit.Tags))
	}
	var title string
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		title = conflictColor.Render(row.Commit.Subject)
	} else {
		title = renderGraphTitleWithAuthor(row.Commit.Author, row.Commit.Subject, searchQuery, rowWidth, cols, selected && graphActive)
	}
	status := ""
	if cols.Status > 0 {
		status = strings.Repeat(" ", cols.Status)
		if row.Commit.Hash != "VIRTUAL_CONFLICT_HASH" {
			status = renderGraphStatus(marks.StashCount, len(row.Commit.Tags))
		}
		status += " "
	}
	when := ""
	if cols.Date > 0 {
		when = padRight(graphDateText(row.Commit.CommitDate), cols.Date) + " "
	}
	line := hash + " " + refs + " " + status + graphCell + " " + when + title
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

func renderGraphTitleWithAuthor(author, subject, searchQuery string, rowWidth int, cols graphColumnWidths, focused bool) string {
	author = compactAuthorText(author)
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = "-"
	}
	available := rowWidth - graphRowFixedWidth(cols)
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
