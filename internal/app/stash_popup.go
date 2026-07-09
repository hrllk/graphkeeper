package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/git"
)

type stashPopupRow struct {
	Entry       git.StashEntry
	ItemIndex   int
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
		lines = append(lines, descStyle.Render("(no stash entries)"))
		lines = append(lines, "")
		lines = append(lines, helpStyle.Render("esc: dismiss"))
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
	lines = append(lines, helpStyle.Render("enter: jump  •  esc: dismiss"))
	body := centerReviewFooterLine(strings.Join(lines, "\n"), popupWidth-4)
	titleLine := renderTitleStrip(popupBox, "Stash list", popupWidth)
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
