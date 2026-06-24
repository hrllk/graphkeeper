package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/git-graph-tui/internal/graph"
)

func renderGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string, graphColWidth int, isHandshake bool) string {
	if row.Graph != "" {
		return renderRawGraphLine(row, selected, graphActive, laneCursor, localBranches, graphColWidth, isHandshake)
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

func renderRawGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string, graphColWidth int, isHandshake bool) string {
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

func padRight(value string, width int) string {
	if lipgloss.Width(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-lipgloss.Width(value))
}

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

func hasHeadDecoration(decorations []string) bool {
	for _, decoration := range decorations {
		if strings.HasPrefix(strings.TrimSpace(decoration), "HEAD -> ") {
			return true
		}
	}
	return false
}

func formatCompactDecorations(decorations []string, localBranches []string) string {
	return compactDecorationInfo(decorations, localBranches).Text
}

type decorationInfo struct {
	Text         string
	HasBranch    bool
	HasLocalHead bool
}

func compactDecorationInfo(decorations []string, localBranches []string) decorationInfo {
	if len(decorations) == 0 {
		return decorationInfo{Text: "-"}
	}
	localSet := make(map[string]struct{}, len(localBranches))
	for _, branch := range localBranches {
		localSet[branch] = struct{}{}
	}
	type branchState struct {
		local  bool
		remote bool
	}
	branches := make(map[string]*branchState)
	order := make([]string, 0, len(decorations))
	hasBranch := false
	hasLocalHead := false

	addBranch := func(name string) *branchState {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil
		}
		state, ok := branches[name]
		if !ok {
			state = &branchState{}
			branches[name] = state
			order = append(order, name)
		}
		return state
	}

	for _, decoration := range decorations {
		decoration = strings.TrimSpace(decoration)
		if decoration == "" {
			continue
		}
		switch {
		case strings.HasPrefix(decoration, "HEAD -> "):
			if state := addBranch(strings.TrimPrefix(decoration, "HEAD -> ")); state != nil {
				state.local = true
				hasBranch = true
				hasLocalHead = true
			}
		case strings.HasPrefix(decoration, "origin/HEAD -> origin/"):
			if state := addBranch(strings.TrimPrefix(decoration, "origin/HEAD -> origin/")); state != nil {
				state.remote = true
				hasBranch = true
			}
		case decoration == "origin/HEAD":
			if state := addBranch("HEAD"); state != nil {
				state.remote = true
				hasBranch = true
			}
		case strings.HasPrefix(decoration, "origin/"):
			name := strings.TrimPrefix(decoration, "origin/")
			if name == "HEAD" {
				continue
			}
			if _, ok := localSet[name]; ok {
				if state := addBranch(name); state != nil {
					state.remote = true
					hasBranch = true
				}
			}
		case strings.HasPrefix(decoration, "tag: "):
			continue
		default:
			if _, ok := localSet[decoration]; ok {
				if state := addBranch(decoration); state != nil {
					state.local = true
					hasBranch = true
				}
			} else if !strings.Contains(decoration, "/") {
				if state := addBranch(decoration); state != nil {
					state.local = true
					hasBranch = true
				}
			}
		}
	}
	name := ""
	for _, candidate := range order {
		if state := branches[candidate]; state != nil && (state.local || state.remote) {
			name = candidate
			break
		}
	}
	if name == "" {
		return decorationInfo{Text: "-", HasBranch: false}
	}
	state := branches[name]
	token := "l->" + name
	if state.remote && state.local {
		token = "o/l->" + name
	} else if state.remote {
		token = "o->" + name
	}
	if len([]rune(token)) > 10 {
		runes := []rune(token)
		token = string(runes[:9]) + "."
	}
	return decorationInfo{Text: token, HasBranch: hasBranch, HasLocalHead: hasLocalHead}
}

func compactWhenText(relative string) string {
	relative = strings.TrimSpace(relative)
	if relative == "" {
		return "-"
	}
	if strings.HasSuffix(relative, " ago") {
		relative = strings.TrimSpace(strings.TrimSuffix(relative, " ago"))
	}
	parts := strings.Fields(relative)
	if len(parts) < 2 {
		return shorten(relative, 7)
	}
	n := parts[0]
	unit := parts[1]
	switch {
	case strings.HasPrefix(unit, "minute"):
		return n + "mins"
	case strings.HasPrefix(unit, "hour"):
		return n + "hours"
	case strings.HasPrefix(unit, "day"):
		return n + "days"
	case strings.HasPrefix(unit, "month"):
		return n + "mons"
	case strings.HasPrefix(unit, "year"):
		return n + "yrs"
	case strings.HasPrefix(unit, "week"):
		return n + "wks"
	default:
		return shorten(relative, 7)
	}
}

func compactTitleText(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "-"
	}
	runes := []rune(subject)
	if len(runes) <= 10 {
		return subject
	}
	return string(runes[:7]) + "..."
}

func focusParentLines(node graphNode, width int) []string {
	if len(node.Parents) == 0 {
		return []string{"parent: -"}
	}
	if len(node.Parents) == 1 {
		return []string{fmt.Sprintf("parent: %s", node.Parents[0])}
	}
	lines := []string{"parent: (multi parent)"}
	parentWidth := max(width-4, 0)
	for _, parent := range node.Parents {
		lines = append(lines, fmt.Sprintf("  - %s", shorten(parent, parentWidth)))
	}
	return lines
}

func focusBranchSummaryLines(node graphNode, width int) []string {
	if len(node.Decorations) == 0 {
		return nil
	}
	lines := make([]string, 0, len(node.Decorations))
	indentWidth := max(width-4, 0)
	for _, decoration := range node.Decorations {
		decoration = strings.TrimSpace(decoration)
		if decoration == "" || strings.HasPrefix(decoration, "tag: ") {
			continue
		}
		lines = append(lines, fmt.Sprintf("  - %s", shorten(decoration, indentWidth)))
	}
	return lines
}
