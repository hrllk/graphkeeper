package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/state"
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
	dirtyMark     = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	stashMark     = lipgloss.NewStyle().Foreground(lipgloss.Color("110"))
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
	return renderAppView(m)
}

func renderAppView(m model) string {
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	bodyWidth, bodyHeight := layoutShellBodySize(m, hMargin, topMargin, bottomMargin)
	headerHeight := layoutHeaderHeight(bodyHeight)
	graphRailHeight := layoutGraphRailHeight(bodyHeight)

	globalWidth, contextWidth := splitPaneWidths(bodyWidth)
	globalBox := baseBox.Width(globalWidth).Height(headerHeight).Render(
		"Global\n" + m.renderGlobalContent(max(globalWidth-4, 0), max(headerHeight-3, 0)),
	)
	contextBox := baseBox.Width(contextWidth).Height(headerHeight).Render(
		"Local\n" + m.renderContextContent(max(contextWidth-4, 0), max(headerHeight-3, 0)),
	)
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, globalBox, contextBox)

	graphWidth := int(float64(bodyWidth) * 0.72)
	if graphWidth < 56 {
		graphWidth = 56
	}
	if graphWidth > bodyWidth-18 {
		graphWidth = bodyWidth - 18
	}
	if graphWidth < 0 {
		graphWidth = 0
	}
	rightWidth := bodyWidth - graphWidth
	graphContentHeight := graphContentHeightForModel(&m)
	graphBox := m.getBoxStyle(sectionGraph).Width(graphWidth).Height(graphRailHeight).Render(
		"Graph\n" + m.renderGraphContent(max(graphWidth-4, 0), graphContentHeight),
	)
	rightRail := m.renderRightRail(rightWidth, graphRailHeight)
	graphRow := lipgloss.JoinHorizontal(lipgloss.Top, graphBox, rightRail)

	body := lipgloss.JoinVertical(lipgloss.Left, headerRow, graphRow)
	centeredBody := applyOuterMargins(body, bodyWidth, bodyHeight, hMargin, topMargin, max(bottomMargin-1, 0))

	if m.status.Mode == state.ModeConfirm || m.status.Mode == state.ModeResetModePick {
		if m.status.Mode == state.ModeResetModePick {
			centeredBody = overlayPopup(centeredBody, renderResetModePopup(bodyWidth))
		} else {
			centeredBody = overlayPopup(centeredBody, renderConfirmPopup(m, bodyWidth))
		}
	}
	if m.status.Mode == state.ModeLoading && !m.branchOpen {
		centeredBody = overlayPopup(centeredBody, renderLoadingPopup(m, bodyWidth))
	}

	shell := centeredBody + "\n"
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, shell)
}

func popupWidthForBody(bodyWidth, minWidth, maxWidth int) int {
	if bodyWidth <= 0 {
		return minWidth
	}
	width := bodyWidth - 12
	if width > maxWidth {
		width = maxWidth
	}
	if width < minWidth {
		width = minWidth
	}
	if width > bodyWidth {
		width = bodyWidth
	}
	return width
}

func renderConfirmPopup(m model, bodyWidth int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 32, 54)).
		Align(lipgloss.Center)
	popupTitle := m.status.Title
	if popupTitle == "" || popupTitle == "Confirm" {
		popupTitle = "Continue?"
	}
	helpText := "y: yes  •  n: no"
	if m.status.Action == state.ActionPull && !m.pullIsFastForward {
		helpText = "m: merge  •  r: rebase  •  esc: cancel"
	}
	return popupBox.Render(
		titleStyle.Render(popupTitle) + "\n\n" +
			descStyle.Render(m.status.Detail) + "\n\n" +
			helpStyle.Render(helpText),
	)
}

func renderResetModePopup(bodyWidth int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 32, 50)).
		Align(lipgloss.Center)
	return popupBox.Render(
		titleStyle.Render("Reset mode") + "\n" +
			bodyStyle.Render("Choose a reset mode.") + "\n\n" +
			bodyStyle.Render("s: soft  •  m: mixed  •  h: hard") + "\n\n" +
			helpStyle.Render("esc: back"),
	)
}

func renderLoadingPopup(m model, bodyWidth int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 28, 44)).
		Align(lipgloss.Center)
	return popupBox.Render(
		titleStyle.Render("Working...") + "\n\n" +
			descStyle.Render(m.status.Message) + "\n" +
			descStyle.Render(m.status.Detail),
	)
}

func (m model) renderRightRail(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	localHeight, remoteHeight, tagsHeight := splitThreeHeights(height)
	localBox := m.getBoxStyle(sectionCurrent).Width(width).Height(localHeight).Render("Local\n" + m.renderSectionContent(sectionCurrent, max(width-4, 0), max(localHeight-3, 0)))
	remoteBox := m.getBoxStyle(sectionRemote).Width(width).Height(remoteHeight).Render("Remote\n" + m.renderSectionContent(sectionRemote, max(width-4, 0), max(remoteHeight-3, 0)))
	tagsBox := m.getBoxStyle(sectionTags).Width(width).Height(tagsHeight).Render("Tags\n" + m.renderSectionContent(sectionTags, max(width-4, 0), max(tagsHeight-3, 0)))
	return lipgloss.JoinVertical(lipgloss.Left, localBox, remoteBox, tagsBox)
}

func applyOuterMargins(content string, totalWidth, totalHeight, hMargin, topMargin, bottomMargin int) string {
	lines := strings.Split(content, "\n")
	leftPad := strings.Repeat(" ", hMargin)
	rightPad := strings.Repeat(" ", hMargin)
	blank := strings.Repeat(" ", totalWidth)
	out := make([]string, 0, totalHeight+topMargin+bottomMargin)
	for i := 0; i < topMargin; i++ {
		out = append(out, blank)
	}
	for _, line := range lines {
		out = append(out, leftPad+line+rightPad)
	}
	for i := 0; i < bottomMargin; i++ {
		out = append(out, blank)
	}
	return strings.Join(out, "\n")
}
