package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/state"
)

var (
	border          = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	baseBox         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	activeBox       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205")).Padding(0, 1)
	title           = lipgloss.NewStyle().Bold(true)
	sectionTitle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00b7eb")).Bold(true)
	contextValue    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00b7eb"))
	hotkey          = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5ca8")).Bold(true)
	muted           = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	accent          = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	warn            = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	ok              = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	disabled        = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	headMark        = lipgloss.NewStyle().Foreground(lipgloss.Color("118")).Bold(true)
	branchMark      = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	pointerMark     = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	dirtyMark       = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	stashMark       = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	remoteColor     = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	tagColor        = lipgloss.NewStyle().Foreground(lipgloss.Color("#9D00FF"))
	tagOverlapColor = lipgloss.NewStyle().Foreground(lipgloss.Color("#A14743")).Bold(true)
	highlight       = lipgloss.NewStyle().Reverse(true).Bold(true)
	conflictColor   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	conflictMark    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	reviewCurrent   = headMark
	reviewTarget    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	reviewBase      = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
	reviewHash      = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	reviewBranch    = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	reviewMark      = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	reviewCount     = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	reviewFooter    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func renderHotkey(key string) string {
	return hotkey.Render(key)
}

func renderHotkeyLine(key, desc string) string {
	return "• " + renderHotkey(key) + ": " + desc
}

func renderDisabledHotkeyLine(key, desc string) string {
	return "• " + renderHotkey(key) + ": " + disabled.Render(desc)
}

func renderSectionTitle(text string) string {
	return sectionTitle.Render(text)
}

func renderContextKey(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		text = "-"
	}
	return contextValue.Render(text)
}

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

	headerPaneWidth := bodyWidth - 4
	if headerPaneWidth < 0 {
		headerPaneWidth = 0
	}
	globalWidth, contextWidth := splitPaneWidths(headerPaneWidth)
	globalBox := renderFloatingTitleFrame(
		baseBox.Width(globalWidth).Height(headerHeight),
		"Global",
		m.renderGlobalContent(max(globalWidth-4, 0), max(headerHeight-2, 0)),
		globalWidth,
		headerHeight,
	)
	contextBox := renderFloatingTitleFrame(
		baseBox.Width(contextWidth).Height(headerHeight),
		"Context",
		m.renderContextContent(max(contextWidth-4, 0), max(headerHeight-2, 0)),
		contextWidth,
		headerHeight,
	)
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, globalBox, contextBox)

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
	rightWidth := graphBudget - graphWidth
	graphContentHeight := graphContentHeightForModel(&m)
	graphBox := renderFloatingTitleFrame(
		m.getBoxStyle(sectionGraph).Width(graphWidth).Height(graphRailHeight),
		"[1] Graph",
		m.renderGraphContent(max(graphWidth-4, 0), graphContentHeight),
		graphWidth,
		graphRailHeight,
	)
	rightRail := m.renderRightRail(rightWidth, graphRailHeight)
	graphRow := lipgloss.JoinHorizontal(lipgloss.Top, graphBox, rightRail)

	body := lipgloss.JoinVertical(lipgloss.Left, headerRow, graphRow)
	centeredBody := applyOuterMargins(body, bodyWidth, bodyHeight, hMargin, topMargin, max(bottomMargin-1, 0))
	centeredBody = applyShellOverlays(m, centeredBody, bodyWidth, bodyHeight)

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
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	width := popupWidthForBody(bodyWidth, 28, 42)
	align := lipgloss.Center
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(width).
		Align(align)
	popupTitle := m.status.Title
	if popupTitle == "" || popupTitle == "Confirm" {
		popupTitle = "Continue?"
	}
	helpText := "y: yes  •  n: no"
	if m.status.Action == state.ActionPull && !m.pullIsFastForward {
		helpText = "m: merge  •  r: rebase  •  esc: cancel"
	} else if m.status.Message == "Fast-forward available." {
		helpText = "enter: fast-forward  •  esc: dismiss"
	} else if m.status.Action == state.ActionDeleteBranch {
		helpText = "y: delete  •  n: cancel"
	} else if m.status.Action == state.ActionDeleteTag {
		helpText = "y: delete  •  n: cancel"
	} else if m.status.Action == state.ActionStash {
		helpText = "y: stash  •  n: cancel"
	} else if m.status.Action == state.ActionCleanWorkingTree {
		helpText = "y: clean  •  n: cancel"
	}
	return renderFloatingTitlePopup(
		popupBox,
		popupTitle,
		centerReviewFooterLine(strings.Join([]string{
			descStyle.Render(m.status.Detail),
			helpStyle.Render(helpText),
		}, "\n\n"), width-4),
		width,
	)
}

func renderReviewPopup(m model, bodyWidth int) string {
	width := popupWidthForBody(bodyWidth, 26, 40)
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(width).
		Align(lipgloss.Center)
	popupTitle := m.status.Message
	if popupTitle == "" {
		popupTitle = "Branch has diverged"
	}
	body := centerReviewFooterLine(m.status.Detail, width-4)
	return renderFloatingTitlePopup(
		popupBox,
		popupTitle,
		body,
		width,
	)
}

func centerReviewFooterLine(body string, width int) string {
	if width <= 0 || body == "" {
		return body
	}
	lines := strings.Split(body, "\n")
	last := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			last = i
			break
		}
	}
	if last < 0 {
		return body
	}
	lines[last] = centerReviewLineInWidth(lines[last], width)
	return strings.Join(lines, "\n")
}

func centerReviewLineInWidth(line string, width int) string {
	visible := lipgloss.Width(line)
	if visible >= width {
		return line
	}
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(line)
}

func renderResetModePopup(bodyWidth int) string {
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 32, 50)).
		Align(lipgloss.Center)
	return renderFloatingTitlePopup(
		popupBox,
		"Reset mode",
		strings.Join([]string{
			bodyStyle.Render("Choose a reset mode."),
			bodyStyle.Render("s: soft  •  m: mixed  •  h: hard"),
			helpStyle.Render("esc: back"),
		}, "\n\n"),
		popupWidthForBody(bodyWidth, 32, 50),
	)
}

func renderLoadingPopup(m model, bodyWidth int) string {
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 28, 44)).
		Align(lipgloss.Center)
	return renderFloatingTitlePopup(
		popupBox,
		"Working...",
		strings.Join([]string{
			descStyle.Render(m.status.Message),
			descStyle.Render(m.status.Detail),
		}, "\n"),
		popupWidthForBody(bodyWidth, 28, 44),
	)
}

func renderTargetPickPopup(m model, bodyWidth int) string {
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 28, 40)).
		Align(lipgloss.Center)
	helpText := "enter: preview  •  esc: back"
	if m.status.Action == state.ActionCheckout {
		helpText = "enter: checkout  •  esc: back"
	} else if m.status.Action == state.ActionDeleteBranch {
		helpText = "enter: delete  •  esc: back"
	}
	lines := []string{
		descStyle.Render(m.status.Message),
		"",
		renderTargets(m.status),
		"",
		helpStyle.Render(helpText),
	}
	return renderFloatingTitlePopup(
		popupBox,
		m.status.Title,
		strings.Join(lines, "\n"),
		popupWidthForBody(bodyWidth, 28, 40),
	)
}

func renderBranchInputPopup(m model, bodyWidth int) string {
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 36, 56)).
		Align(lipgloss.Center)
	draft := m.branchDraft
	if draft == "" {
		draft = " "
	}
	base := m.branchBase
	if base == "" {
		base = "-"
	}
	lines := []string{
		descStyle.Render("Enter a branch name."),
		"",
		descStyle.Render("name: " + draft),
		descStyle.Render("base: " + shorten(base, 24)),
	}
	if m.branchError != "" {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(m.branchError))
	}
	lines = append(lines, "")
	lines = append(lines, helpStyle.Render("esc: back"))
	return renderFloatingTitlePopup(
		popupBox,
		"Create branch",
		strings.Join(lines, "\n"),
		popupWidthForBody(bodyWidth, 36, 56),
	)
}

func renderGraphSearchPopup(m model, bodyWidth int) string {
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 38, 58)).
		Align(lipgloss.Center)
	query := m.graphSearchDraft
	if query == "" {
		query = " "
	}
	lines := []string{
		descStyle.Render("query: " + query),
		"",
		helpStyle.Render("enter: jump  •  n/N: next/prev  •  esc: cancel"),
	}
	if m.graphSearchError != "" {
		lines = append(lines, "")
		lines = append(lines, errStyle.Render(m.graphSearchError))
	} else {
		matches := graphSearchMatches(buildGraphSearchIndex(m.repoStatus), m.graphSearchDraft)
		lines = append(lines, "")
		lines = append(lines, descStyle.Render(fmt.Sprintf("%d matches", len(matches))))
	}
	return renderFloatingTitlePopup(
		popupBox,
		"Search Graph",
		strings.Join(lines, "\n"),
		popupWidthForBody(bodyWidth, 38, 58),
	)
}

func (m model) renderRightRail(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	cardWidth := width
	if cardWidth < 1 {
		cardWidth = 1
	}
	sectionHeight := height - 3
	if sectionHeight < 1 {
		sectionHeight = 1
	}
	localHeight, remoteHeight, tagsHeight := splitThreeHeights(sectionHeight)
	localBox := renderFloatingTitleFrame(
		m.getBoxStyle(sectionCurrent).Width(cardWidth).Height(localHeight),
		"[2] Local",
		m.renderSectionContent(sectionCurrent, max(cardWidth-4, 0), localHeight),
		cardWidth,
		localHeight,
	)
	remoteBox := renderFloatingTitleFrame(
		m.getBoxStyle(sectionRemote).Width(cardWidth).Height(remoteHeight),
		"[3] Remote",
		m.renderSectionContent(sectionRemote, max(cardWidth-4, 0), remoteHeight),
		cardWidth,
		remoteHeight,
	)
	tagsBox := renderFloatingTitleFrame(
		m.getBoxStyle(sectionTags).Width(cardWidth).Height(tagsHeight),
		"[4] Tags",
		m.renderSectionContent(sectionTags, max(cardWidth-4, 0), tagsHeight),
		cardWidth,
		tagsHeight,
	)
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
