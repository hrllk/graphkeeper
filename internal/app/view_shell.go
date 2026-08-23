package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/state"
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
	if m.commitInspectorOpen {
		return renderCommitInspectorScreen(m, m.width, m.height)
	}
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	bodyWidth, bodyHeight := layoutShellContentSize(m, hMargin, topMargin, bottomMargin)

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
		m.getBoxStyle(sectionGraph).Width(graphWidth).Height(bodyHeight),
		"[1] Graph",
		m.renderGraphContent(max(graphWidth-4, 0), graphContentHeight),
		graphWidth,
		bodyHeight,
	)
	rightRail := m.renderRightRail(rightWidth, bodyHeight)
	graphRow := lipgloss.JoinHorizontal(lipgloss.Top, graphBox, rightRail)

	body := graphRow
	body = applyShellOverlays(m, body, bodyWidth, bodyHeight)
	shellBody := body + "\n" + renderMainHotkeyFooter(bodyWidth)
	centeredBody := applyOuterMargins(shellBody, bodyWidth, bodyHeight+layoutShellFooterHeight, hMargin, topMargin, max(bottomMargin-1, 0))

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

func renderPopupFooter(width int) string {
	return centerReviewLineInWidth(popupHelp.Render("q: close"), width)
}

func renderMergeConfirmPopup(view mergeConfirmViewModel, bodyWidth int) string {
	width := popupWidthForBody(bodyWidth, 36, 42)
	return renderDivergentConfirmPopup(confirmationProjection{
		Kind:          confirmChoiceKind,
		Title:         "Pull into " + view.CurrentBranch + "?",
		Detail:        mergeConfirmBody(view, width-4),
		FooterText:    "m/enter: merge · r: rebase · n/esc: cancel",
		CurrentBranch: view.CurrentBranch,
		TargetRef:     view.TargetRef,
		CurrentOnly:   view.CurrentOnly,
		TargetOnly:    view.TargetOnly,
		ImpactKnown:   view.ImpactKnown,
		MergeText:     view.MergeText,
		RebaseText:    view.RebaseText,
		RiskText:      view.RiskText,
		Disabled:      view.Disabled,
		DisabledText:  view.DisabledText,
	}, bodyWidth)
}

func renderDivergentConfirmPopup(projection confirmationProjection, bodyWidth int) string {
	width := popupWidthForBody(bodyWidth, 36, 42)
	popupBox := popupBorder.Padding(1, 2).Width(width).Align(lipgloss.Center)
	body := mergeConfirmBodyFromProjection(projection, width-4)
	if projection.Detail != "" && projection.Detail != body {
		body = joinLayoutSections(projection.Detail, body)
	}
	footer := projection.FooterText
	if projection.Disabled {
		footer = "n: close · esc: close"
	} else if width < 54 && footer == "m/enter: merge · r: rebase · n/esc: cancel" {
		footer = "m/enter: merge\nr: rebase · n/esc: cancel"
	}
	body = joinLayoutSections(body, popupHelp.Render(footer), renderPopupFooter(width-4))
	return renderFloatingTitlePopup(popupBox, projection.Title, centerReviewFooterLine(body, width-4), width)
}

func mergeConfirmBodyFromProjection(projection confirmationProjection, width int) string {
	if projection.Disabled {
		return fitMergeLines(projection.DisabledText, width)
	}
	lines := []string{
		"Pull into " + fitMergeValue(projection.CurrentBranch, width, "Pull into "),
		"Target: " + fitMergeValue(projection.TargetRef, width, "Target: "),
		"Relation: " + fitMergeCounts(projection.CurrentOnly, projection.TargetOnly, width),
		"",
		"Merge: " + fitMergeValue(projection.MergeText, width, "Merge: "),
		"Rebase: " + fitMergeValue(projection.RebaseText, width, "Rebase: "),
	}
	if projection.RiskText != "" {
		lines = append(lines, "Risk: "+fitMergeValue(projection.RiskText, width, "Risk: "))
	}
	return strings.Join(lines, "\n")
}

func renderConfirmPopup(m model, bodyWidth int) string {
	projection, ok := buildConfirmationProjection(m)
	if !ok {
		return ""
	}
	if projection.Kind == confirmChoiceKind {
		return renderDivergentConfirmPopup(projection, bodyWidth)
	}
	width := popupWidthForBody(bodyWidth, 28, 42)
	popupBox := popupBorder.
		Padding(1, 2).
		Width(width).
		Align(lipgloss.Center)
	popupTitle := projection.Title
	if popupTitle == "" || popupTitle == "Confirm" {
		popupTitle = "Continue?"
	}
	detail := projection.Detail
	if projection.Disabled {
		detail = joinLayoutSections(detail, projection.DisabledText)
	}
	return renderFloatingTitlePopup(
		popupBox,
		popupTitle,
		centerReviewFooterLine(joinLayoutSections(
			popupBody.Render(detail),
			popupHelp.Render(projection.FooterText),
			renderPopupFooter(width-4),
		), width-4),
		width,
	)
}

func renderReviewPopup(m model, bodyWidth int) string {
	width := popupWidthForBody(bodyWidth, 26, 40)
	popupBox := popupBorder.
		Padding(1, 2).
		Width(width).
		Align(lipgloss.Center)
	popupTitle := m.status.Message
	if popupTitle == "" {
		popupTitle = "Branch has diverged"
	}
	body := joinLayoutSections(centerReviewFooterLine(m.status.Detail, width-4), renderPopupFooter(width-4))
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
	for i := range lines {
		lines[i] = fitVisibleWidth(lines[i], width)
	}
	lines[last] = centerReviewLineInWidth(lines[last], width)
	return strings.Join(lines, "\n")
}

func centerReviewLineInWidth(line string, width int) string {
	line = fitVisibleWidth(line, width)
	visible := lipgloss.Width(line)
	if visible >= width {
		return line
	}
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(line)
}

func renderResetModePopup(bodyWidth int) string {
	bodyStyle := popupBody
	popupBox := popupBorder.
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 32, 50)).
		Align(lipgloss.Center)
	return renderFloatingTitlePopup(
		popupBox,
		"Reset mode",
		strings.Join([]string{
			bodyStyle.Render("Choose a reset mode."),
			bodyStyle.Render("s: soft  •  m: mixed  •  h: hard"),
			renderPopupFooter(popupWidthForBody(bodyWidth, 32, 50) - 4),
		}, "\n\n"),
		popupWidthForBody(bodyWidth, 32, 50),
	)
}

func renderLoadingPopup(m model, bodyWidth int) string {
	descStyle := popupBody
	popupBox := popupBorder.
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
	descStyle := popupBody
	helpStyle := popupHelp
	popupBox := popupBorder.
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 28, 40)).
		Align(lipgloss.Center)
	helpText := "enter: preview"
	if m.status.Action == state.ActionCheckout {
		helpText = "enter: checkout"
	} else if m.status.Action == state.ActionDeleteBranch {
		helpText = "enter: delete"
	}
	lines := []string{
		descStyle.Render(m.status.Message),
		"",
		renderTargets(m.status),
		"",
		helpStyle.Render(helpText),
		renderPopupFooter(popupWidthForBody(bodyWidth, 28, 40) - 4),
	}
	return renderFloatingTitlePopup(
		popupBox,
		m.status.Title,
		strings.Join(lines, "\n"),
		popupWidthForBody(bodyWidth, 28, 40),
	)
}

func renderBranchInputPopup(m model, bodyWidth int) string {
	descStyle := popupBody
	popupBox := popupBorder.
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
		lines = append(lines, errorStyle.Render(m.branchError))
	}
	lines = append(lines, "")
	lines = append(lines, renderPopupFooter(popupWidthForBody(bodyWidth, 36, 56)-4))
	return renderFloatingTitlePopup(
		popupBox,
		"Create branch",
		strings.Join(lines, "\n"),
		popupWidthForBody(bodyWidth, 36, 56),
	)
}

func renderGraphSearchPopup(m model, bodyWidth int) string {
	descStyle := popupBody
	helpStyle := popupHelp
	errStyle := errorStyle
	popupBox := popupBorder.
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
		helpStyle.Render("enter: jump  •  n/N: next/prev"),
	}
	if m.graphSearchError != "" {
		lines = append(lines, "")
		lines = append(lines, errStyle.Render(m.graphSearchError))
	} else {
		matches := graphSearchMatches(buildGraphSearchIndex(m.repoStatus), m.graphSearchDraft)
		lines = append(lines, "")
		lines = append(lines, descStyle.Render(fmt.Sprintf("%d matches", len(matches))))
	}
	lines = append(lines, "", renderPopupFooter(popupWidthForBody(bodyWidth, 38, 58)-4))
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
	detailsHeight, localHeight, remoteHeight, tagsHeight := splitFourHeights(height)
	detailsBox := renderFloatingTitleFrame(
		baseBox.Width(cardWidth).Height(detailsHeight),
		sectionName(m.activeSection)+" Details",
		m.renderDetailsContent(max(cardWidth-4, 0), max(detailsHeight-2, 0)),
		cardWidth,
		detailsHeight,
	)
	localBox := renderFloatingTitleFrame(
		m.getBoxStyle(sectionCurrent).Width(cardWidth).Height(localHeight),
		"[2] Local",
		m.renderSectionContent(sectionCurrent, max(cardWidth-4, 0), max(localHeight-2, 0)),
		cardWidth,
		localHeight,
	)
	remoteBox := renderFloatingTitleFrame(
		m.getBoxStyle(sectionRemote).Width(cardWidth).Height(remoteHeight),
		"[3] Remote",
		m.renderSectionContent(sectionRemote, max(cardWidth-4, 0), max(remoteHeight-2, 0)),
		cardWidth,
		remoteHeight,
	)
	tagsBox := renderFloatingTitleFrame(
		m.getBoxStyle(sectionTags).Width(cardWidth).Height(tagsHeight),
		"[4] Tags",
		m.renderSectionContent(sectionTags, max(cardWidth-4, 0), max(tagsHeight-2, 0)),
		cardWidth,
		tagsHeight,
	)
	return lipgloss.JoinVertical(lipgloss.Left, detailsBox, localBox, remoteBox, tagsBox)
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
