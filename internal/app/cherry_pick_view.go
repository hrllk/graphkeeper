package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func graphBoxWidthForBodyWidth(bodyWidth int) int {
	graphBudget := bodyWidth - 4
	if graphBudget < 0 {
		graphBudget = 0
	}
	graphWidth := int(float64(graphBudget) * 0.72)
	if graphWidth < 56 {
		graphWidth = 56
	}
	if graphWidth > graphBudget-18 {
		graphWidth = graphBudget - 18
	}
	if graphWidth < 0 {
		graphWidth = 0
	}
	return graphWidth
}

func renderCherryPickPopup(m model, bodyWidth int) string {
	graphWidth := graphBoxWidthForBodyWidth(bodyWidth)
	graphHeight := graphBoxHeightForModel(&m)
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(0, 1).
		Width(graphWidth).
		Height(graphHeight)

	content := renderCherryPickList(m, max(graphWidth-2, 0), graphContentHeightForModel(&m))
	return renderFloatingTitleFrame(
		popupBox,
		"Cherry-pick Targets",
		content,
		graphWidth,
		graphHeight,
	)
}

func renderCherryPickList(m model, width, height int) string {
	if height <= 0 {
		return ""
	}
	targets := m.status.Targets
	if len(targets) == 0 {
		return fitBlockLines([]string{muted.Render("  (no cherry-pick targets)")}, height)
	}

	selectedSet := make(map[string]struct{}, len(m.status.SelectedQueue))
	for _, ref := range m.status.SelectedQueue {
		selectedSet[ref] = struct{}{}
	}

	start := 0
	if len(targets) > height && m.status.TargetIdx >= 0 {
		start = m.status.TargetIdx - height + 1
		if start < 0 {
			start = 0
		}
		if start > len(targets)-height {
			start = len(targets) - height
		}
	}

	lines := make([]string, 0, height)
	lines = append(lines, fitVisibleWidth(muted.Render(fmt.Sprintf("destination: current %s", cherryPickDestination(m.repoStatus))), width))
	lines = append(lines, fitVisibleWidth(muted.Render(fmt.Sprintf("selected: %d / %d", len(m.status.SelectedQueue), len(targets))), width))
	lines = append(lines, fitVisibleWidth(sectionTitle.Render(renderCherryPickHeader(width)), width))

	for i := start; i < len(targets); i++ {
		if len(lines) >= height {
			break
		}
		lines = append(lines, fitVisibleWidth(renderCherryPickTargetRow(targets[i], i == m.status.TargetIdx, selectedSet, width), width))
	}
	return fitBlockLines(lines, height)
}

func cherryPickDestination(rs git.Status) string {
	if rs.Branch != "" {
		return rs.Branch
	}
	if rs.Head != "" {
		return "HEAD"
	}
	return "-"
}

func renderCherryPickHeader(width int) string {
	parts := []string{
		padRight("target", 7),
		padRight("hash", 8),
		padRight("author", 7),
		padRight("age", 7),
		"title",
	}
	return fitVisibleWidth(strings.Join(parts, "  "), width)
}

func renderCherryPickTargetRow(target state.TargetItem, selected bool, selectedSet map[string]struct{}, width int) string {
	cursor := " "
	if selected {
		cursor = ">"
	}
	targetMark := "[ ]"
	if _, ok := selectedSet[target.Ref]; ok {
		targetMark = "[x]"
	}
	parts := []string{
		cursor,
		padRight(targetMark, 7),
		padRight(shorten(target.CommitHash, 7), 8),
		padRight(compactAuthorText(target.Author), 7),
		padRight(compactWhenText(target.RelativeAge), 7),
		shorten(target.Subject, max(width-40, 12)),
	}
	return strings.Join(parts, "  ")
}
