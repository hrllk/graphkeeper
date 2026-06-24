package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/git-graph-tui/internal/state"
)

var (
	border        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	baseBox       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	activeBox     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205")).Padding(0, 1)
	title         = lipgloss.NewStyle().Bold(true)
	muted         = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	accent        = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	warn          = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	ok            = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	disabled      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	headMark      = lipgloss.NewStyle().Foreground(lipgloss.Color("118")).Bold(true)
	branchMark    = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	pointerMark   = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	localColor    = lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	remoteColor   = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	tagColor      = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	highlight     = lipgloss.NewStyle().Reverse(true).Bold(true)
	conflictColor = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	conflictMark  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
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
	graphBox := m.getBoxStyle(sectionGraph).Width(graphWidth).Height(graphHeight).Render("Graph (local branches)\n" + graphContent)

	// 5. 하단 Mode (Detail Pane) 박스 구성
	detailContent := m.renderDetailContent(detailWidth-4, detailHeight-3)
	detailBox := baseBox.Width(detailWidth).Height(detailHeight).Render(detailContent)

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, graphBox, detailBox)

	// 6. 세로 정렬 및 Place 정중앙 배치
	body := lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)

	centeredBody := lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center, body)

	if m.status.Mode == state.ModeConfirm {
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		popupBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2).
			Width(50).
			Align(lipgloss.Center)
		popupTitle := m.status.Title
		if popupTitle == "" || popupTitle == "Confirm" {
			popupTitle = "Do you want to continue?"
		}
		helpText := "y: yes  •  n: no"
		if m.status.Action == state.ActionPull && !m.pullIsFastForward {
			helpText = "m: merge  •  r: rebase  •  esc: cancel"
		}
		popupContent := popupBox.Render(
			titleStyle.Render(popupTitle) + "\n\n" +
				descStyle.Render(m.status.Detail) + "\n\n" +
				helpStyle.Render(helpText),
		)
		centeredBody = overlayPopup(centeredBody, popupContent)
	}

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

func overlayPopup(base string, popup string) string {
	baseLines := strings.Split(base, "\n")
	popupLines := strings.Split(popup, "\n")
	baseH := len(baseLines)
	popupH := len(popupLines)
	if baseH < popupH {
		return base
	}
	popupW := 0
	for _, l := range popupLines {
		w := lipgloss.Width(l)
		if w > popupW {
			popupW = w
		}
	}
	startY := (baseH - popupH) / 2
	for i, pl := range popupLines {
		y := startY + i
		if y >= len(baseLines) {
			break
		}
		bl := baseLines[y]
		blW := lipgloss.Width(bl)
		startX := (blW - popupW) / 2
		if startX < 0 {
			startX = 0
		}
		baseLines[y] = overlayLine(bl, pl, startX, popupW)
	}
	return strings.Join(baseLines, "\n")
}

func overlayLine(baseLine string, popupLine string, startX, popupW int) string {
	var left strings.Builder
	var right strings.Builder
	inAnsi := false
	visWidth := 0
	runes := []rune(baseLine)
	i := 0
	n := len(runes)
	for i < n && visWidth < startX {
		r := runes[i]
		if r == '\x1b' {
			inAnsi = true
		}
		left.WriteRune(r)
		if inAnsi {
			if r == 'm' {
				inAnsi = false
			}
		} else {
			visWidth += lipgloss.Width(string(r))
		}
		i++
	}
	targetVisWidth := visWidth + popupW
	for i < n && visWidth < targetVisWidth {
		r := runes[i]
		if r == '\x1b' {
			inAnsi = true
		}
		if inAnsi {
			if r == 'm' {
				inAnsi = false
			}
		} else {
			visWidth += lipgloss.Width(string(r))
		}
		i++
	}
	for i < n {
		right.WriteRune(runes[i])
		i++
	}
	return left.String() + popupLine + right.String()
}
