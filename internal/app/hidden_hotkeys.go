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

const hiddenHotkeyPopupFooter = "q: close"

const (
	hiddenHotkeyPopupMinWidth = 32
	hiddenHotkeyPopupMaxWidth = 50
)

func globalHotkeyItems() []hiddenHotkeyItem {
	return []hiddenHotkeyItem{
		{key: "tab", desc: "switch"},
		{key: "k/j", desc: "updown"},
		{key: "q", desc: "quit"},
		{key: "?", desc: "hotkeys"},
		{key: "ctrl + u/d", desc: "scroll"},
	}
}

func renderMainHotkeyFooter(width int) string {
	items := globalHotkeyItems()
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, renderHotkey(item.key)+": "+item.desc)
	}
	footer := strings.Join(parts, " · ")
	return fitVisibleWidth(muted.Render(footer), width)
}

func (m model) handleHiddenHotkeysKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?":
		m.hiddenHotkeysOpen = false
		return m, nil
	case "up", "k":
		return m.scrollHiddenHotkeys(-1), nil
	case "down", "j":
		return m.scrollHiddenHotkeys(1), nil
	case "ctrl+u":
		return m.scrollHiddenHotkeys(-m.hiddenHotkeyContentViewport()), nil
	case "ctrl+d":
		return m.scrollHiddenHotkeys(m.hiddenHotkeyContentViewport()), nil
	default:
		return m, nil
	}
}

func (m model) scrollHiddenHotkeys(delta int) model {
	viewport := m.hiddenHotkeyContentViewport()
	if viewport <= 0 {
		m.hiddenHotkeysScroll = 0
		return m
	}
	width, _ := hiddenHotkeyPopupBodySize(m)
	total := len(hiddenHotkeyContentLines(m, hiddenHotkeyPopupWidth(width)))
	m.hiddenHotkeysScroll = clampHiddenHotkeyScroll(m.hiddenHotkeysScroll+delta, total, viewport)
	return m
}

func (m model) hiddenHotkeyContentViewport() int {
	width, height := hiddenHotkeyPopupBodySize(m)
	_, viewport := hiddenHotkeyPopupLayout(m, width, height)
	return viewport
}

func hiddenHotkeyPopupBodySize(m model) (int, int) {
	if m.width <= 0 || m.height <= 0 {
		return m.width, 0
	}
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	return layoutShellContentSize(m, hMargin, topMargin, bottomMargin)
}

func clampHiddenHotkeyScroll(offset, totalLines, viewportHeight int) int {
	if totalLines <= 0 || viewportHeight <= 0 {
		return 0
	}
	maxOffset := max(0, totalLines-viewportHeight)
	if offset < 0 {
		return 0
	}
	if offset > maxOffset {
		return maxOffset
	}
	return offset
}

func hiddenHotkeyPopupStyle(width int) lipgloss.Style {
	return popupBorder.
		Padding(1, 2).
		Width(width).
		Align(lipgloss.Left)
}

func hiddenHotkeyPopupWidth(bodyWidth int) int {
	return popupWidthForBody(bodyWidth, hiddenHotkeyPopupMinWidth, hiddenHotkeyPopupMaxWidth)
}

func hiddenHotkeyContentLines(m model, width int) []string {
	lines := make([]string, 0)
	sections := visibleHiddenHotkeySections(m)
	for i, section := range sections {
		lines = append(lines, renderHiddenHotkeySectionTitle(section.title, section.active))
		for _, group := range section.groups {
			lines = append(lines, renderHiddenHotkeyGroupLines(group.title, group.items, width)...)
		}
		if i < len(sections)-1 {
			lines = append(lines, "")
		}
	}
	return lines
}

func visibleHiddenHotkeySections(m model) []hiddenHotkeySection {
	sections := hiddenHotkeySections(m)
	visible := make([]hiddenHotkeySection, 0, 2)
	for _, section := range sections {
		if section.active {
			visible = append(visible, section)
		}
	}
	return visible
}

func hiddenHotkeyPopupBody(m model, width int, content []string, showFocus bool, offset, viewport int) string {
	lines := []string{
		renderCenteredPopupLine(popupHeader.Render("Hidden hotkeys by section"), width),
	}
	if showFocus {
		lines = append(lines, renderCenteredPopupLine(popupHelp.Render("focus: "+sectionName(m.activeSection)), width), "")
	}
	if viewport > 0 && offset < len(content) {
		end := min(offset+viewport, len(content))
		lines = append(lines, content[offset:end]...)
	}
	lines = append(lines, "", renderCenteredPopupLine(popupHelp.Render(hiddenHotkeyPopupFooter), width))
	return strings.Join(lines, "\n")
}

func hiddenHotkeyPopupLayout(m model, bodyWidth, bodyHeight int) (string, int) {
	popupWidth := hiddenHotkeyPopupWidth(bodyWidth)
	popupBox := hiddenHotkeyPopupStyle(popupWidth)
	content := hiddenHotkeyContentLines(m, popupWidth)
	showFocusOptions := []bool{true, false}
	for _, showFocus := range showFocusOptions {
		for viewport := len(content); viewport >= 0; viewport-- {
			offset := clampHiddenHotkeyScroll(m.hiddenHotkeysScroll, len(content), viewport)
			body := hiddenHotkeyPopupBody(m, popupWidth, content, showFocus, offset, viewport)
			popup := renderFloatingTitlePopup(popupBox, "Hidden Hotkeys", body, popupWidth)
			if bodyHeight <= 0 || lipgloss.Height(popup) <= bodyHeight {
				return popup, viewport
			}
		}
	}
	return renderFloatingTitlePopup(popupBox, "Hidden Hotkeys", hiddenHotkeyPopupBody(m, popupWidth, content, false, 0, 0), popupWidth), 0
}

func renderHiddenHotkeysPopup(m model, bodyWidth, bodyHeight int) string {
	popup, _ := hiddenHotkeyPopupLayout(m, bodyWidth, bodyHeight)
	return popup
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
					items: globalHotkeyItems(),
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
