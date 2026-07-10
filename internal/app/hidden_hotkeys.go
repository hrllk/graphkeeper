package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type hiddenHotkeyItem struct {
	key  string
	desc string
}

type hiddenHotkeyGroup struct {
	title string
	items []hiddenHotkeyItem
}

type hiddenHotkeySection struct {
	title  string
	active bool
	groups []hiddenHotkeyGroup
}

func (m model) handleHiddenHotkeysKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?":
		m.hiddenHotkeysOpen = false
		return m, nil
	default:
		return m, nil
	}
}

func renderHiddenHotkeysPopup(m model, bodyWidth int) string {
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	popupWidth := popupWidthForBody(bodyWidth, 44, 72)
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidth).
		Align(lipgloss.Left)

	lines := []string{
		renderCenteredPopupLine(headerStyle.Render("Hidden hotkeys by section"), popupWidth),
		renderCenteredPopupLine(helpStyle.Render("focus: "+sectionName(m.activeSection)), popupWidth),
		"",
	}
	sections := hiddenHotkeySections(m)
	for i, section := range sections {
		lines = append(lines, renderHiddenHotkeySectionTitle(section.title, section.active))
		for _, group := range section.groups {
			lines = append(lines, renderHiddenHotkeyGroupLines(group.title, group.items, popupWidth)...)
		}
		if i < len(sections)-1 {
			lines = append(lines, "")
		}
	}
	lines = append(lines, "")
	lines = append(lines, renderCenteredPopupLine(helpStyle.Render("esc: close"), popupWidth))

	return renderFloatingTitlePopup(
		popupBox,
		"Hidden Hotkeys",
		strings.Join(lines, "\n"),
		popupWidth,
	)
}

func renderHiddenHotkeySectionTitle(title string, active bool) string {
	if active {
		return sectionTitle.Render("› " + title)
	}
	return muted.Render("  " + title)
}

func renderHiddenHotkeyGroupLines(title string, items []hiddenHotkeyItem, width int) []string {
	lines := []string{"  " + title + ":"}
	if len(items) == 0 {
		lines = append(lines, "    "+muted.Render("(none)"))
		return lines
	}
	for _, item := range items {
		lines = append(lines, fitVisibleWidth("    • "+renderHotkey(item.key)+": "+item.desc, width))
	}
	return lines
}

func hiddenHotkeySections(m model) []hiddenHotkeySection {
	return []hiddenHotkeySection{
		{
			title: "Global",
			groups: []hiddenHotkeyGroup{
				{
					title: "Common",
					items: []hiddenHotkeyItem{
						{key: "tab", desc: "next section"},
						{key: "shift+tab", desc: "previous section"},
						{key: "j/k", desc: "move"},
						{key: "f", desc: "fetch"},
						{key: "F", desc: "fetch tags"},
						{key: "S", desc: "stash list"},
						{key: "q", desc: "quit"},
						{key: "?", desc: "hidden hotkeys"},
					},
				},
				{
					title: "Moved out",
					items: []hiddenHotkeyItem{
						{key: "gg", desc: "top"},
						{key: "G", desc: "bottom"},
						{key: "ctrl+u/d", desc: "scroll"},
					},
				},
			},
		},
		{
			title:  "Graph",
			active: m.activeSection == sectionGraph,
			groups: []hiddenHotkeyGroup{
				{
					title: "Visible",
					items: []hiddenHotkeyItem{
						{key: "m", desc: "merge"},
						{key: "r", desc: "rebase"},
						{key: "space", desc: "checkout"},
						{key: "H", desc: "jump to HEAD"},
					},
				},
				{
					title: "Conditional",
					items: []hiddenHotkeyItem{
						{key: "s", desc: "reset"},
						{key: "d", desc: "delete branch"},
						{key: "p", desc: "pull"},
						{key: "P", desc: "push"},
						{key: "t", desc: "tag commit"},
						{key: "o", desc: "pop stash"},
						{key: "a", desc: "abort in-progress operation"},
						{key: "n", desc: "new branch or repeat search"},
						{key: "N", desc: "repeat search backward"},
					},
				},
			},
		},
		{
			title:  "Local",
			active: m.activeSection == sectionCurrent,
			groups: []hiddenHotkeyGroup{
				{
					title: "Visible",
					items: []hiddenHotkeyItem{
						{key: "s", desc: "stash changes"},
						{key: "c", desc: "clean working tree"},
						{key: "space", desc: "checkout"},
						{key: "d", desc: "delete branch"},
					},
				},
				{
					title: "Conditional",
					items: []hiddenHotkeyItem{
						{key: "a", desc: "abort merge"},
						{key: "P", desc: "push"},
					},
				},
			},
		},
		{
			title:  "Remote",
			active: m.activeSection == sectionRemote,
			groups: []hiddenHotkeyGroup{
				{
					title: "Visible",
					items: []hiddenHotkeyItem{
						{key: "space", desc: "checkout"},
						{key: "f", desc: "fetch"},
						{key: "p", desc: "pull"},
						{key: "d", desc: "delete branch"},
					},
				},
			},
		},
		{
			title:  "Tags",
			active: m.activeSection == sectionTags,
			groups: []hiddenHotkeyGroup{
				{
					title: "Visible",
					items: []hiddenHotkeyItem{
						{key: "enter", desc: "jump to graph"},
						{key: "d", desc: "delete tag"},
					},
				},
			},
		},
	}
}

func renderCenteredPopupLine(text string, width int) string {
	if width <= 0 {
		return text
	}
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(text)
}
