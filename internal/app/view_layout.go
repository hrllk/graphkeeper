package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
			visWidth++
		}
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
