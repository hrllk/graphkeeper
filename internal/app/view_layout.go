package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	contentHeight := railHeight - 3
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
