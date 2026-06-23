package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/git-graph-tui/internal/state"
)

var (
	border      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	baseBox     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	activeBox   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205")).Padding(0, 1)
	title       = lipgloss.NewStyle().Bold(true)
	muted       = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	accent      = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	warn        = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	ok          = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	headMark    = lipgloss.NewStyle().Foreground(lipgloss.Color("118")).Bold(true)
	branchMark  = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	pointerMark = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	localColor  = lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	remoteColor = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	tagColor    = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	highlight   = lipgloss.NewStyle().Reverse(true).Bold(true)
)

func (m model) getBoxStyle(section graphSection) lipgloss.Style {
	if m.activeSection == section {
		return activeBox
	}
	return baseBox
}

func (m model) View() string {
	// 1. 15% 마진이 적용된 가용 대시보드 너비/높이 계산 (클램핑 보호)
	totalWidth := int(float64(m.width) * 0.70)
	if totalWidth < 80 {
		totalWidth = 80
	}
	if totalWidth > m.width {
		totalWidth = m.width
	}

	totalHeight := int(float64(m.height) * 0.76)
	if totalHeight < 18 {
		totalHeight = 18
	}
	if totalHeight > m.height-2 {
		totalHeight = m.height - 2
	}

	leftColWidth := int(float64(totalWidth) * 0.70)
	rightColWidth := totalWidth - leftColWidth - 2

	topHeight, bottomHeight := splitDashboardHeights(totalHeight)
	graphHeight, detailHeight := splitPaneHeights(bottomHeight)
	graphWidth, detailWidth := splitPaneWidths(totalWidth)

	// 2. 상단 Local & Remote 박스 (Branches 대박스 없이 독립 배치)
	localWidth := leftColWidth / 2
	remoteWidth := leftColWidth - localWidth

	localContent := m.renderSectionContent(sectionCurrent, localWidth-4, topHeight-3)
	remoteContent := m.renderSectionContent(sectionRemote, remoteWidth-4, topHeight-3)

	localBox := m.getBoxStyle(sectionCurrent).Width(localWidth).Height(topHeight).Render("Local\n" + localContent)
	remoteBox := m.getBoxStyle(sectionRemote).Width(remoteWidth).Height(topHeight).Render("Remote\n" + remoteContent)

	branchesInner := lipgloss.JoinHorizontal(lipgloss.Top, localBox, remoteBox)

	// 3. 상단 Tags 박스 구성
	tagsContent := m.renderSectionContent(sectionTags, rightColWidth-4, topHeight-3)
	tagsBox := m.getBoxStyle(sectionTags).Width(rightColWidth).Height(topHeight).Render("Tags\n" + tagsContent)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, branchesInner, tagsBox)

	// 4. 하단 Graph 박스 구성
	graphContent := m.renderGraphContent(graphWidth-4, graphHeight-3)
	graphBox := m.getBoxStyle(sectionGraph).Width(graphWidth).Height(graphHeight).Render("Graph\n" + graphContent)

	// 5. 하단 Mode (Detail Pane) 박스 구성
	detailContent := m.renderDetailContent(detailWidth-4, detailHeight-3)
	detailBox := baseBox.Width(detailWidth).Height(detailHeight).Render(detailContent)

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, graphBox, detailBox)

	// 6. 세로 정렬 및 Place 정중앙 배치
	body := lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)

	centeredBody := lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center, body)

	footer := muted.Render("global: 1 local  •  2 remote  •  3 tags  •  4 graph  •  tab/shift+tab section  •  up/down/j/k move  •  f fetch  •  q quit")

	return centeredBody + "\n" + footer + "\n"
}

func (m model) renderSectionContent(section graphSection, width, height int) string {
	items := sectionTargets(m.repoStatus, section)
	if len(items) == 0 {
		return muted.Render("  (empty)")
	}
	cursor := m.sectionCursor[section]
	var b strings.Builder
	for i, item := range items {
		if i >= height {
			break
		}
		prefix := "  "
		if i == cursor && m.activeSection == section {
			prefix = "> "
		}
		label := formatTargetItem(item)
		if label == "" {
			continue
		}
		b.WriteString(prefix + label + "\n")
	}
	return b.String()
}

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
		lines = append(lines, renderGraphLine(rows[i], graphActive && i == m.sectionCursor[sectionGraph], graphActive, m.graphLaneCursor, m.repoStatus.LocalBranches, graphColWidth))
		if !rawGraph && i+1 < len(rows) {
			for _, line := range renderGraphConnectorLines(rows[i], rows[i+1]) {
				if len(lines) >= height {
					break
				}
				lines = append(lines, line)
			}
		}
	}
	return fitBlockLines(lines, height)
}

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

func renderStatusCompact(s state.Status) string {
	msg := shorten(s.Message, 30)
	switch s.Mode {
	case state.ModeBrowse:
		return ok.Render("Browse") + " | " + msg
	case state.ModeLoading:
		return accent.Render("Loading") + " | " + msg
	case state.ModeBlocked:
		return warn.Render("Blocked") + " | " + msg
	default:
		return msg
	}
}

func renderTargets(s state.Status) string {
	if len(s.Targets) == 0 {
		return muted.Render("(no targets)")
	}
	var b strings.Builder
	for i, t := range s.Targets {
		prefix := "  "
		if i == s.TargetIdx {
			prefix = "> "
		}
		label := formatTargetItem(t)
		if label == "" {
			continue
		}
		b.WriteString(prefix + label + "\n")
	}
	return b.String()
}

func formatTargetItem(t state.TargetItem) string {
	switch t.Kind {
	case state.TargetKindLocal:
		if t.Current {
			label := headMark.Render("l->" + t.Name)
			if t.NeedsPull {
				label += " " + warn.Render("⬇")
			}
			if t.NoUpstream {
				label += " " + warn.Render("(no-up)")
			}
			return label
		}
		label := "l->" + t.Name
		if t.NeedsPull {
			label += " " + warn.Render("⬇")
		}
		if t.NoUpstream {
			label += " " + warn.Render("(no-up)")
		}
		return label
	case state.TargetKindRemote:
		name := t.Name
		if !strings.Contains(name, "/") {
			return ""
		}
		if strings.HasSuffix(name, "/HEAD") {
			name = name
		} else if strings.HasPrefix(name, "origin/") {
			name = strings.TrimPrefix(name, "origin/")
		}
		label := "o->" + name
		if t.Default {
			label += " (default)"
		}
		return label
	case state.TargetKindTag:
		return "tag    " + t.Name
	default:
		return t.Name
	}
}

func renderActionHelpLines(m model) []string {
	switch m.status.Mode {
	case state.ModeBrowse:
		lines := make([]string, 0, 5)
		switch m.activeSection {
		case sectionGraph:
			lines = append(lines, "• m: merge         • r: rebase")
			lines = append(lines, "• s: reset         • ctrl+u/d: scroll")
			lines = append(lines, "• gg: top         • G: bottom")
			lines = append(lines, "• H: jump to HEAD")
			lines = append(lines, "• n: new branch")
		case sectionCurrent, sectionRemote:
			lines = append(lines, "• space: checkout")
			if m.activeSection == sectionCurrent && pullReady(m.repoStatus) {
				lines = append(lines, "• p: pull")
			}
			if m.activeSection == sectionCurrent {
				lines = append(lines, "• n: new branch")
			}
		case sectionTags:
			lines = append(lines, "• no section actions")
		default:
			lines = append(lines, "• no section actions")
		}
		return lines
	case state.ModeTargetPick:
		return []string{"• up/down: choose target            • enter: preview", "• esc: back"}
	case state.ModeOutcomePreview:
		if m.status.CanExecute {
			return []string{"• enter: execute                    • esc: back"}
		}
		return []string{"• esc: back"}
	default:
		return []string{"• r: refresh"}
	}
}

func renderGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string, graphColWidth int) string {
	if row.Graph != "" {
		return renderRawGraphLine(row, selected, graphActive, laneCursor, localBranches, graphColWidth)
	}
	hash := fmt.Sprintf("%-8s", shorten(row.Commit.Hash, 7))
	refInfo := compactDecorationInfo(row.Commit.Decorations, localBranches)
	refs := fmt.Sprintf("%-10s", refInfo.Text)
	graphCell := graphCells(row, graphActive, selected, laneCursor, graphColWidth)
	when := fmt.Sprintf("%-7s", compactWhenText(row.Commit.RelativeAge))
	title := fmt.Sprintf("%-10s", compactTitleText(row.Commit.Subject))
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
	line := hash + " " + refs + " " + padRight(graphCell, graphColWidth) + "  " + when + " " + title
	if selected {
		return "> " + line
	}
	return "  " + line
}

func renderRawGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string, graphColWidth int) string {
	if row.Commit.Hash == "" && row.Commit.Subject == "" && len(row.Commit.Decorations) == 0 && len(row.Commit.Parents) == 0 {
		line := fmt.Sprintf("%-8s %-10s %-*s  %-7s %-10s", "", "", graphColWidth, row.Graph, "", "")
		if selected {
			return "> " + line
		}
		return "  " + line
	}
	graphRunes := []rune(row.Graph)
	width := len(graphRunes)
	lane := graphPointerLane(row)
	cursorLane := laneCursor
	if width > 0 && cursorLane >= width {
		cursorLane = width - 1
	}
	pointerFocused := graphActive && selected && cursorLane == lane
	hash := fmt.Sprintf("%-8s", shorten(row.Commit.Hash, 7))
	if pointerFocused {
		hash = pointerMark.Render(hash)
	}
	refInfo := compactDecorationInfo(row.Commit.Decorations, localBranches)
	refs := fmt.Sprintf("%-10s", refInfo.Text)
	if refInfo.HasLocalHead {
		refs = headMark.Render(refs)
	} else if pointerFocused && refInfo.HasBranch {
		refs = branchMark.Render(refs)
	}
	graphCell := highlightRawGraphPrefix(row.Graph, lane, pointerFocused, refInfo.HasLocalHead)
	when := fmt.Sprintf("%-7s", compactWhenText(row.Commit.RelativeAge))
	title := fmt.Sprintf("%-10s", compactTitleText(row.Commit.Subject))
	line := hash + " " + refs + " " + padRight(graphCell, graphColWidth) + "  " + when + " " + title
	if selected {
		return "> " + line
	}
	return "  " + line
}

func graphCells(row graphRow, graphActive bool, selected bool, laneCursor int, graphColWidth int) string {
	return graphLineCell(row, graphActive, selected, laneCursor, graphColWidth)
}

func graphLineCell(row graphRow, graphActive bool, selected bool, laneCursor int, graphColWidth int) string {
	if row.Graph == "" && row.Commit.Hash == "" {
		return ""
	}
	if row.Graph == "" {
		width := graphRowWidth(row)
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
	lane := graphPointerLane(row)
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

func renderGraphConnectorLines(current, next graphRow) []string {
	if shouldCollapseRowDisplay(next) {
		return collapseConnectorLines(current)
	}
	if lines := parentShiftConnectorLines(current, next); len(lines) > 0 {
		return lines
	}
	return nil
}

func collapseConnectorLines(current graphRow) []string {
	width := len(current.After)
	if width <= 1 {
		return nil
	}
	if width == 2 {
		return []string{renderGraphSpacer([]string{"|", "/"})}
	}
	lines := make([]string, 0, width)
	full := make([]string, width)
	for i := range full {
		full[i] = "|"
	}
	lines = append(lines, renderGraphSpacer(full))
	for w := width; w >= 2; w-- {
		cells := make([]string, w)
		for i := range cells {
			cells[i] = "|"
		}
		cells[w-1] = "/"
		lines = append(lines, renderGraphSpacer(cells))
	}
	return lines
}

func parentShiftConnectorLines(current, next graphRow) []string {
	width := max(len(current.After), graphRowWidth(next))
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
		return []string{renderGraphSpacer(full), renderGraphSpacer(cells)}
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

func renderGraphSpacer(cells []string) string {
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

func isBlankGraphCells(cells []string) bool {
	for _, cell := range cells {
		if cell != " " {
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

func paneWidth(total int, ratio float64) int {
	if total <= 0 {
		return 0
	}
	return int(float64(total) * ratio)
}

func splitPaneWidths(total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	left := total / 2
	right := total - left
	return left, right
}

func splitDashboardHeights(total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	top := total / 5
	if top < 1 {
		top = 1
	}
	bottom := total - top
	if bottom < 1 {
		bottom = 1
		top = total - bottom
	}
	return top, bottom
}

func splitPaneHeights(total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	top := total / 2
	bottom := total - top
	return top, bottom
}

func fitBlockLines(lines []string, height int) string {
	if height <= 0 {
		return ""
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	if len(lines) < height {
		padding := make([]string, height-len(lines))
		lines = append(lines, padding...)
	}
	return strings.Join(lines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
