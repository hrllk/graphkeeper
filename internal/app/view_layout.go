package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

func layoutShellMargins(m model) (hMargin, topMargin, bottomMargin int) {
	hMargin = int(float64(m.width) * 0.10)
	topMargin = int(float64(m.height) * 0.12)
	bottomMargin = int(float64(m.height) * 0.12)
	if hMargin < 2 {
		hMargin = 2
	}
	if topMargin < 2 {
		topMargin = 2
	}
	if bottomMargin < 2 {
		bottomMargin = 2
	}
	if maxMargin := (m.width - 80) / 2; maxMargin >= 0 && hMargin > maxMargin {
		hMargin = maxMargin
	}
	if maxTop := m.height - 20; maxTop >= 0 && topMargin > maxTop {
		topMargin = maxTop
	}
	if maxBottom := m.height - topMargin - 19; maxBottom >= 0 && bottomMargin > maxBottom {
		bottomMargin = maxBottom
	}
	return hMargin, topMargin, bottomMargin
}

func layoutShellBodySize(m model, hMargin, topMargin, bottomMargin int) (width, height int) {
	width = m.width - hMargin*2
	if width < 80 {
		width = 80
	}
	height = m.height - topMargin - bottomMargin
	if height < 12 {
		height = 12
	}
	return width, height
}

func layoutHeaderHeight(bodyHeight int) int {
	if bodyHeight <= 0 {
		return 0
	}
	height := 12
	if bodyHeight < 24 {
		height = 11
	}
	if height > bodyHeight-12 {
		height = bodyHeight - 12
	}
	if height < 9 {
		height = 9
	}
	if height >= bodyHeight {
		height = bodyHeight - 1
	}
	if height < 1 {
		height = 1
	}
	return height
}

func layoutGraphRailHeight(bodyHeight int) int {
	railHeight := bodyHeight - layoutHeaderHeight(bodyHeight)
	if railHeight < 12 {
		railHeight = 12
	}
	return railHeight
}

func graphBoxHeightForModel(m *model) int {
	hMargin, topMargin, bottomMargin := layoutShellMargins(*m)
	_, bodyHeight := layoutShellBodySize(*m, hMargin, topMargin, bottomMargin)
	return layoutGraphRailHeight(bodyHeight)
}

func graphContentHeightForModel(m *model) int {
	railHeight := graphBoxHeightForModel(m)
	contentHeight := railHeight - 2
	if contentHeight < 1 {
		return 1
	}
	return contentHeight
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
	left := total * 3 / 10
	if left < 1 {
		left = 1
	}
	if left > total-1 {
		left = total - 1
	}
	right := total - left
	return left, right
}

func splitLocalPaneWidths(total int) (int, int, int) {
	if total <= 0 {
		return 0, 0, 0
	}
	if total == 1 {
		return 1, 0, 0
	}
	if total == 2 {
		return 1, 0, 1
	}
	separator := 1
	inner := total - separator
	left := inner / 2
	if left < 1 {
		left = 1
	}
	right := inner - left
	if right < 1 {
		right = 1
		left = inner - right
	}
	return left, separator, right
}

func splitDashboardHeights(total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	top := total / 8
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

func splitThreeHeights(total int) (int, int, int) {
	if total <= 0 {
		return 0, 0, 0
	}
	first := total / 3
	second := total / 3
	third := total - first - second
	if first == 0 {
		first = 1
	}
	if second == 0 && total > 1 {
		second = 1
	}
	if third == 0 && total > 2 {
		third = 1
	}
	for first+second+third > total {
		switch {
		case third > 1:
			third--
		case second > 1:
			second--
		case first > 1:
			first--
		default:
			return total, 0, 0
		}
	}
	if rem := total - (first + second + third); rem > 0 {
		third += rem
	}
	return first, second, third
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

func fitBlockWidth(lines []string, width int) []string {
	if width <= 0 || len(lines) == 0 {
		return lines
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, fitVisibleWidth(line, width))
	}
	return out
}

func fitLineWidth(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) > width {
		return ansi.Truncate(value, width, "")
	}
	return padRight(value, width)
}

func renderTitleStrip(style lipgloss.Style, title string, width int) string {
	border, hasTop, _, _, _ := style.GetBorder()
	if !hasTop || width <= 0 {
		return fitVisibleWidth(title, width)
	}

	stripWidth := width + 2
	if stripWidth < 1 {
		stripWidth = 1
	}

	title = strings.TrimSpace(title)
	if title == "" {
		title = " "
	}

	leftWidth := lipgloss.Width(border.TopLeft)
	rightWidth := lipgloss.Width(border.TopRight)
	innerWidth := stripWidth - leftWidth - rightWidth
	if innerWidth <= 0 {
		return fitVisibleWidth(title, stripWidth)
	}
	if innerWidth < 3 {
		return fitVisibleWidth(title, stripWidth)
	}

	maxTitleWidth := innerWidth - 2
	if maxTitleWidth < 1 {
		maxTitleWidth = 1
	}
	title = fitVisibleWidth(title, maxTitleWidth)
	titleWidth := lipgloss.Width(title)
	if titleWidth+2 > innerWidth {
		return fitVisibleWidth(title, stripWidth)
	}

	titleText := "\x1b[1m" + title + "\x1b[22m"
	fillWidth := innerWidth - titleWidth - 2
	leftFill := 2
	if leftFill > fillWidth {
		leftFill = fillWidth
	}
	if leftFill < 1 {
		leftFill = 1
	}
	rightFill := fillWidth - leftFill
	if rightFill < 0 {
		rightFill = 0
		leftFill = fillWidth
	}
	line := border.TopLeft +
		strings.Repeat(border.Top, leftFill) +
		" " + titleText + " " +
		strings.Repeat(border.Top, rightFill) +
		border.TopRight

	return renderBorderLine(line, style)
}

func renderBorderLine(line string, style lipgloss.Style) string {
	borderStyle := lipgloss.NewStyle()
	if c := style.GetBorderTopForeground(); c != nil {
		borderStyle = borderStyle.Foreground(c)
	}
	if c := style.GetBorderTopBackground(); c != nil {
		borderStyle = borderStyle.Background(c)
	}
	return borderStyle.Render(line)
}

func renderFloatingTitleFrame(style lipgloss.Style, title, body string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	titleLine := renderTitleStrip(style, title, width)
	bodyStyle := style.BorderTop(false)
	bodyHeight := height
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	bodyBlock := bodyStyle.Width(width).Height(bodyHeight).Render(body)
	return titleLine + "\n" + bodyBlock
}

func renderFloatingTitlePopup(style lipgloss.Style, title, body string, width int) string {
	if width <= 0 {
		return ""
	}

	titleLine := renderTitleStrip(style, title, width)
	bodyStyle := style.BorderTop(false)
	bodyBlock := bodyStyle.Width(width).Render(body)
	return titleLine + "\n" + bodyBlock
}

func renderSplitColumns(leftLines, rightLines []string, width, height int) string {
	if height <= 0 || width <= 0 {
		return ""
	}
	leftWidth, separatorWidth, rightWidth := splitLocalPaneWidths(width)
	if separatorWidth == 0 {
		return fitBlockLines(leftLines, height)
	}
	leftLines = normalizeColumnLines(leftLines, leftWidth, height)
	rightLines = normalizeColumnLines(rightLines, rightWidth, height)
	separator := muted.Render("│")
	if separatorWidth > 1 {
		separator = padRight(separator, separatorWidth)
	}
	lines := make([]string, 0, height)
	for i := 0; i < height; i++ {
		lines = append(lines,
			fitLineWidth(leftLines[i], leftWidth)+separator+fitLineWidth(rightLines[i], rightWidth),
		)
	}
	return strings.Join(lines, "\n")
}

func indentLines(lines []string, indent int) []string {
	if indent <= 0 || len(lines) == 0 {
		return lines
	}
	prefix := strings.Repeat(" ", indent)
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = prefix + line
	}
	return out
}

func normalizeColumnLines(lines []string, width, height int) []string {
	if height <= 0 {
		return nil
	}
	lines = fitColumnHeight(lines, height)
	for i, line := range lines {
		lines[i] = fitLineWidth(line, width)
	}
	return lines
}

func fitColumnHeight(lines []string, height int) []string {
	if height <= 0 {
		return nil
	}
	out := make([]string, 0, height)
	if len(lines) > height {
		lines = lines[:height]
	}
	out = append(out, lines...)
	if len(out) < height {
		padding := make([]string, height-len(out))
		out = append(out, padding...)
	}
	return out
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
	baseW := 0
	for _, l := range baseLines {
		if w := lipgloss.Width(l); w > baseW {
			baseW = w
		}
	}
	if popupW > baseW {
		popupW = baseW
	}
	startY := (baseH - popupH) / 2
	startX := 0
	if baseW > popupW {
		startX = (baseW - popupW) / 2
	}
	for i, pl := range popupLines {
		y := startY + i
		if y >= len(baseLines) {
			break
		}
		baseLines[y] = overlayLine(baseLines[y], pl, startX, popupW)
	}
	return strings.Join(baseLines, "\n")
}

func overlayLine(baseLine string, popupLine string, startX, popupW int) string {
	var left strings.Builder
	var right strings.Builder
	visWidth := 0
	runes := []rune(baseLine)
	i := 0
	n := len(runes)
	for i < n && visWidth < startX {
		r := runes[i]
		left.WriteRune(r)
		if r == '\x1b' {
			i++
			for i < n {
				left.WriteRune(runes[i])
				if runes[i] == 'm' {
					i++
					break
				}
				i++
			}
			continue
		}
		visWidth += runewidth.RuneWidth(r)
		i++
	}
	covered := 0
	for i < n && covered < popupW {
		r := runes[i]
		if r == '\x1b' {
			i++
			for i < n {
				if runes[i] == 'm' {
					i++
					break
				}
				i++
			}
			continue
		}
		covered += runewidth.RuneWidth(r)
		i++
	}
	if i < n {
		right.WriteString(string(runes[i:]))
	}
	paddedPopup := popupLine
	if lipgloss.Width(paddedPopup) < popupW {
		paddedPopup += strings.Repeat(" ", popupW-lipgloss.Width(paddedPopup))
	}
	return left.String() + paddedPopup + right.String()
}
