package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

type stashPopupRow struct {
	Entry     git.StashEntry
	ItemIndex int
}

func (m model) handleStashMessageKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.stashMessageOpen = false
		m.stashMessageDraft = ""
		m.stashMessageError = ""
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	case "enter":
		message := strings.TrimSpace(m.stashMessageDraft)
		if message == "" {
			m.stashMessageError = "Stash message is empty."
			return m, nil
		}
		m.stashMessageOpen = false
		m.stashMessageDraft = ""
		m.stashMessageError = ""
		m.status = operationLoadingStatusFor(progressStash, "Stashing changes...", state.ActionStash)
		return m, executeStashAll(m.repo, m.commitLimit, message)
	case "backspace":
		if len(m.stashMessageDraft) > 0 {
			runes := []rune(m.stashMessageDraft)
			m.stashMessageDraft = string(runes[:len(runes)-1])
			m.stashMessageError = ""
		}
		return m, nil
	case "delete":
		if len(m.stashMessageDraft) > 0 {
			runes := []rune(m.stashMessageDraft)
			m.stashMessageDraft = string(runes[:len(runes)-1])
			m.stashMessageError = ""
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.stashMessageDraft += string(msg.Runes)
			m.stashMessageError = ""
			return m, nil
		}
	}
	return m, nil
}

func (m model) handleGraphStashPopKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.graphStashPopOpen = false
		m.graphStashPopEntries = nil
		m.graphStashPopCursor = 0
		m.graphStashPopMode = graphStashPopModePicker
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	case "up", "k":
		if m.graphStashPopMode == graphStashPopModePicker && len(m.graphStashPopEntries) > 0 {
			m.graphStashPopCursor = moveCircularCursor(m.graphStashPopCursor, -1, len(m.graphStashPopEntries))
		}
		return m, nil
	case "down", "j":
		if m.graphStashPopMode == graphStashPopModePicker && len(m.graphStashPopEntries) > 0 {
			m.graphStashPopCursor = moveCircularCursor(m.graphStashPopCursor, 1, len(m.graphStashPopEntries))
		}
		return m, nil
	case "enter":
		if len(m.graphStashPopEntries) == 0 {
			m.graphStashPopOpen = false
			m.graphStashPopMode = graphStashPopModePicker
			m.status = deriveStatus(m.repoStatus)
			return m, nil
		}
		if m.graphStashPopMode == graphStashPopModePicker {
			m.graphStashPopMode = graphStashPopModeConfirm
			return m, nil
		}
		entry, ok := graphStashPopSelected(m)
		if !ok {
			m.graphStashPopOpen = false
			m.graphStashPopMode = graphStashPopModePicker
			m.status = deriveStatus(m.repoStatus)
			return m, nil
		}
		m.graphStashPopOpen = false
		m.graphStashPopMode = graphStashPopModePicker
		m.status = operationLoadingStatusFor(progressStashPop, "Popping stash...", state.ActionStashPop)
		return m, executeStashPop(m.repo, m.commitLimit, entry)
	default:
		return m, nil
	}
}

func (m model) handleStashPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.stashPopupOpen = false
		return m, nil
	case "up", "k":
		m = moveStashPopupCursor(m, -1)
		return m, nil
	case "down", "j":
		m = moveStashPopupCursor(m, 1)
		return m, nil
	case "enter":
		if len(m.stashEntries) == 0 {
			m.stashPopupOpen = false
			return m, nil
		}
		cursor := m.stashPopupCursor
		if cursor < 0 || cursor >= len(m.stashEntries) {
			cursor = 0
		}
		entry := m.stashEntries[cursor]
		if entry.BaseHash != "" && focusGraphCommit(&m, m.repoStatus, entry.BaseHash) {
			m.stashPopupOpen = false
		}
		return m, nil
	default:
		return m, nil
	}
}

func moveCircularCursor(cursor, delta, count int) int {
	if count <= 0 {
		return 0
	}
	if cursor < 0 || cursor >= count {
		cursor = 0
	}
	cursor = (cursor + delta) % count
	if cursor < 0 {
		cursor += count
	}
	return cursor
}

// stashEmptyStateLines names which of the three empty states the popup is in.
// Collapsing them into one string is what made a failing `git stash list` read
// as "you have no stashed work".
func stashEmptyStateLines(m model) []string {
	switch {
	case !m.stashLoadAttempted:
		return []string{"Stash list not loaded yet."}
	case m.stashLoadError != "":
		return []string{"Stash list unavailable.", m.stashLoadError}
	default:
		return []string{"No stashed work."}
	}
}

func renderStashPopup(m model, bodyWidth, bodyHeight int) string {
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	popupWidth := popupWidthForBody(bodyWidth, 44, 76)
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidth).
		Align(lipgloss.Left)

	lines := []string{
		centerReviewLineInWidth(descStyle.Render("Browse stash entries."), popupWidth-4),
		"",
	}
	rows := buildStashPopupRows(m.stashEntries)
	if len(rows) == 0 {
		// Three different facts used to share one string. Say which one it is:
		// the list was never loaded, loading failed, or there is nothing stashed.
		for _, line := range stashEmptyStateLines(m) {
			lines = append(lines, descStyle.Render(line))
		}
		lines = append(lines, "")
		lines = append(lines, renderPopupFooter(popupWidth-4))
		body := centerReviewFooterLine(strings.Join(lines, "\n"), popupWidth-4)
		titleLine := renderTitleStrip(popupBox, "Stash list", popupWidth)
		bodyBlock := popupBox.BorderTop(false).Align(lipgloss.Left).Width(popupWidth).Render(body)
		return titleLine + "\n" + bodyBlock
	}

	visibleListHeight := bodyHeight - 8
	if visibleListHeight < 1 {
		visibleListHeight = 1
	}

	selectedRow := 0
	for i, row := range rows {
		if row.ItemIndex == m.stashPopupCursor {
			selectedRow = i
			break
		}
	}
	start := selectedRow - visibleListHeight/2
	if start < 0 {
		start = 0
	}
	end := start + visibleListHeight
	if end > len(rows) {
		end = len(rows)
		start = end - visibleListHeight
		if start < 0 {
			start = 0
		}
	}

	if start > 0 {
		lines = append(lines, muted.Render("..."))
	}

	for _, row := range rows[start:end] {
		lines = append(lines, renderStashPopupEntry(row.Entry, row.ItemIndex == m.stashPopupCursor, popupWidth-4))
	}

	if end < len(rows) {
		lines = append(lines, muted.Render("..."))
	}
	lines = append(lines, "")
	lines = append(lines, helpStyle.Render("enter: jump"), renderPopupFooter(popupWidth-4))
	body := centerReviewFooterLine(strings.Join(lines, "\n"), popupWidth-4)
	titleLine := renderTitleStrip(popupBox, "Stash list", popupWidth)
	bodyBlock := popupBox.BorderTop(false).Align(lipgloss.Left).Width(popupWidth).Render(body)
	return titleLine + "\n" + bodyBlock
}

func renderStashMessagePopup(m model, bodyWidth int) string {
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	popupWidth := popupWidthForBody(bodyWidth, 40, 60)
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidth).
		Align(lipgloss.Center)
	draft := m.stashMessageDraft
	if draft == "" {
		draft = " "
	}
	sections := []string{strings.Join([]string{
		descStyle.Render("Enter a message for this stash."),
		descStyle.Render("message: " + draft),
	}, "\n")}
	if m.stashMessageError != "" {
		sections = append(sections, errStyle.Render(m.stashMessageError))
	}
	sections = append(sections, helpStyle.Render("enter: stash"), renderPopupFooter(popupWidth-4))
	return renderFloatingTitlePopup(popupBox, "Stash changes", joinLayoutSections(sections...), popupWidth)
}

func renderGraphStashPopPopup(m model, bodyWidth, bodyHeight int) string {
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	popupWidth := popupWidthForBody(bodyWidth, 44, 76)
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidth).
		Align(lipgloss.Left)

	lines := []string{
		centerReviewLineInWidth(descStyle.Render("Pop stash from HEAD."), popupWidth-4),
		"",
	}

	if len(m.graphStashPopEntries) == 0 {
		lines = append(lines, descStyle.Render("(no stash entries)"))
		lines = append(lines, "")
		lines = append(lines, renderPopupFooter(popupWidth-4))
		body := centerReviewFooterLine(strings.Join(lines, "\n"), popupWidth-4)
		titleLine := renderTitleStrip(popupBox, "Pop stash", popupWidth)
		bodyBlock := popupBox.BorderTop(false).Align(lipgloss.Left).Width(popupWidth).Render(body)
		return titleLine + "\n" + bodyBlock
	}

	if m.graphStashPopMode == graphStashPopModePicker {
		lines = append(lines, descStyle.Render("Choose a stash to pop."))
		lines = append(lines, "")
		visibleListHeight := bodyHeight - 10
		if visibleListHeight < 1 {
			visibleListHeight = 1
		}
		start := m.graphStashPopCursor - visibleListHeight/2
		if start < 0 {
			start = 0
		}
		end := start + visibleListHeight
		if end > len(m.graphStashPopEntries) {
			end = len(m.graphStashPopEntries)
			start = end - visibleListHeight
			if start < 0 {
				start = 0
			}
		}
		if start > 0 {
			lines = append(lines, muted.Render("..."))
		}
		for i := start; i < end; i++ {
			lines = append(lines, renderStashPopupEntry(m.graphStashPopEntries[i], i == m.graphStashPopCursor, popupWidth-4))
		}
		if end < len(m.graphStashPopEntries) {
			lines = append(lines, muted.Render("..."))
		}
		lines = append(lines, "")
		lines = append(lines, helpStyle.Render("enter: choose"))
	} else {
		entry, ok := graphStashPopSelected(m)
		if !ok {
			lines = append(lines, descStyle.Render("(no selection)"))
			lines = append(lines, "")
		} else {
			lines = append(lines, descStyle.Render("Confirm stash pop."))
			lines = append(lines, "")
			lines = append(lines, renderStashPopupEntry(entry, true, popupWidth-4))
			lines = append(lines, "")
			lines = append(lines, warnStyle.Render("This will remove the stash if the pop succeeds."))
			lines = append(lines, "")
			lines = append(lines, helpStyle.Render("enter: pop"))
		}
	}

	lines = append(lines, renderPopupFooter(popupWidth-4))
	body := centerReviewFooterLine(strings.Join(lines, "\n"), popupWidth-4)
	titleLine := renderTitleStrip(popupBox, "Pop stash", popupWidth)
	bodyBlock := popupBox.BorderTop(false).Align(lipgloss.Left).Width(popupWidth).Render(body)
	return titleLine + "\n" + bodyBlock
}

func renderStashPopupEntry(entry git.StashEntry, selected bool, width int) string {
	if width <= 0 {
		return ""
	}
	base := entry.BaseHash
	if base == "" {
		base = "-"
	}
	parts := []string{
		stashMark.Render(shorten(base, 7)),
		muted.Render(entry.Ref),
	}
	if entry.Subject != "" {
		parts = append(parts, shorten(entry.Subject, 20))
	}
	label := strings.Join(parts, "  ")
	label = fitVisibleWidth(label, width-2)
	line := "  " + label
	if selected {
		return highlight.Render(fitVisibleWidth(line, width))
	}
	return fitVisibleWidth(line, width)
}

func moveStashPopupCursor(m model, delta int) model {
	count := len(m.stashEntries)
	if count == 0 {
		m.stashPopupCursor = 0
		return m
	}
	cursor := m.stashPopupCursor
	if cursor < 0 || cursor >= count {
		cursor = 0
	}
	cursor = (cursor + delta) % count
	if cursor < 0 {
		cursor += count
	}
	m.stashPopupCursor = cursor
	return m
}

func buildStashPopupRows(entries []git.StashEntry) []stashPopupRow {
	if len(entries) == 0 {
		return nil
	}
	rows := make([]stashPopupRow, 0, len(entries))
	for i, entry := range entries {
		rows = append(rows, stashPopupRow{
			Entry:     entry,
			ItemIndex: i,
		})
	}
	return rows
}
