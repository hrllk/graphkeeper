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
	headMark    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
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

	footer := muted.Render("global: tab/shift+tab section  •  up/down/j/k move  •  f fetch  •  q quit")

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
		b.WriteString(prefix + formatTargetItem(item) + "\n")
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
	for i := start; i < end; i++ {
		if len(lines) >= height {
			break
		}
		lines = append(lines, renderGraphLine(rows[i], graphActive && i == m.sectionCursor[sectionGraph], graphActive, m.graphLaneCursor, m.repoStatus.LocalBranches))
		if i+1 < len(rows) {
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
	lines = append(lines, fmt.Sprintf("upstr:  %-12s • remo: %s", shorten(emptyDash(m.repoStatus.Upstream), 10), shorten(emptyDash(m.repoStatus.Remote), 10)))

	focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
	if focus.Hash != "" {
		lines = append(lines, fmt.Sprintf("focus:  %s", shorten(focusLineSummary(focus), width-2)))
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
		b.WriteString(prefix + formatTargetItem(t) + "\n")
	}
	return b.String()
}

func formatTargetItem(t state.TargetItem) string {
	switch t.Kind {
	case state.TargetKindLocal:
		if t.Current {
			label := ok.Render("l->" + t.Name)
			if t.NeedsPull {
				label += " " + warn.Render("[pull]")
			}
			return label
		}
		label := "l->" + t.Name
		if t.NeedsPull {
			label += " " + warn.Render("[pull]")
		}
		return label
	case state.TargetKindRemote:
		name := t.Name
		if strings.HasPrefix(name, "origin/") {
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
			lines = append(lines, "• g: top          • G: bottom")
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

func renderGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string) string {
	hash := fmt.Sprintf("%-8s", shorten(row.Commit.Hash, 7))
	refInfo := compactDecorationInfo(row.Commit.Decorations, localBranches)
	refs := fmt.Sprintf("%-16s", refInfo.Text)
	isHead := hasHeadDecoration(row.Commit.Decorations)
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
		if isHead && i == lane {
			cell = headMark.Render(cell)
		} else if pointerFocused && i == lane {
			cell = pointerMark.Render(cell)
		}
		cells = append(cells, cell)
	}
	if isHead {
		refs = headMark.Render(refs)
	} else if pointerFocused && refInfo.HasBranch {
		refs = branchMark.Render(refs)
	}
	line := hash + " " + refs + " " + strings.Join(cells, " ")
	if selected {
		return "> " + line
	}
	return "  " + line
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
	Text      string
	HasBranch bool
}

func compactDecorationInfo(decorations []string, localBranches []string) decorationInfo {
	if len(decorations) == 0 {
		return decorationInfo{Text: "-"}
	}
	localSet := make(map[string]struct{}, len(localBranches))
	for _, branch := range localBranches {
		localSet[branch] = struct{}{}
	}
	parts := make([]string, 0, len(decorations))
	remoteParts := make([]string, 0, len(decorations))
	localParts := make([]string, 0, len(decorations))
	tagParts := make([]string, 0, len(decorations))
	seen := make(map[string]struct{})
	hasBranch := false
	for _, decoration := range decorations {
		token, kind := compactDecoration(decoration, localSet)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		switch kind {
		case "head":
			hasBranch = true
			parts = append(parts, token)
		case "remote":
			hasBranch = true
			remoteParts = append(remoteParts, token)
		case "local":
			hasBranch = true
			localParts = append(localParts, token)
		case "tag":
			tagParts = append(tagParts, token)
		}
	}
	parts = append(parts, remoteParts...)
	parts = append(parts, localParts...)
	parts = append(parts, tagParts...)
	if len(parts) == 0 {
		return decorationInfo{Text: "-", HasBranch: false}
	}
	joined := strings.Join(parts, ", ")
	runes := []rune(joined)
	if len(runes) <= 16 {
		return decorationInfo{Text: joined, HasBranch: hasBranch}
	}
	return decorationInfo{Text: string(runes[:16]), HasBranch: hasBranch}
}

func compactDecoration(decoration string, localSet map[string]struct{}) (string, string) {
	decoration = strings.TrimSpace(decoration)
	switch {
	case strings.HasPrefix(decoration, "HEAD -> "):
		return compactToken("l", strings.TrimPrefix(decoration, "HEAD -> ")), "head"
	case strings.HasPrefix(decoration, "tag: "):
		return compactToken("t", strings.TrimPrefix(decoration, "tag: ")), "tag"
	case strings.HasPrefix(decoration, "origin/"):
		name := strings.TrimPrefix(decoration, "origin/")
		if _, ok := localSet[name]; ok {
			return compactToken("o", name), "remote"
		}
		return "", ""
	case decoration != "":
		return compactToken("l", decoration), "local"
	default:
		return "", ""
	}
}

func compactToken(kind, name string) string {
	token := kind + "->" + name
	if len([]rune(token)) <= 10 {
		return token
	}
	runes := []rune(token)
	return string(runes[:9]) + "."
}

func focusLineSummary(node graphNode) string {
	if len(node.Decorations) == 0 {
		return node.Hash
	}
	return node.Hash + "  " + strings.Join(node.Decorations, ", ")
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
