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

// A section is a flat list of keys. It used to be split into "Visible" and
// "Conditional" groups, and "Common" and "Moved out" for the global section --
// four labels describing how the code categorises keys, none of them something
// a reader can act on. "Conditional on what?" had no answer on screen.
//
// The app already answers it at the moment it matters: pressing a key whose
// condition is unmet produces the reason ("No stash available. Add a stash at
// HEAD first."), which is more use than a category. So the split was decoration
// costing a row per group, against DESIGN.md's minimal decoration.
//
// The bullet stays. It is the list marker the whole app uses (docs/decisions.md
// 2026-09-10, D-008), and the rows saved here come from the group titles.
type hiddenHotkeySection struct {
	title  string
	active bool
	items  []hiddenHotkeyItem
}

const hiddenHotkeyPopupFooter = "esc: close"

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
		lines = append(lines, renderHiddenHotkeyItemLines(section.items, width)...)
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

// This popup's body is a list, so everything in it shares the list's left
// baseline -- header, focus line and footer included. Centring belongs to
// popups whose body is a single message.
//
// The centred lines were also not centred. renderCenteredPopupLine was handed
// the popup width, but the box spends four of those columns on padding, so the
// line was built four columns too wide and its trailing space was clipped:
// "esc: close" sat 22 columns from the left edge and 18 from the right.
func hiddenHotkeyPopupBody(m model, width int, content []string, showFocus bool, offset, viewport int) string {
	lines := []string{
		popupHeader.Render("Hidden hotkeys by section"),
	}
	if showFocus {
		lines = append(lines, popupHelp.Render("focus: "+sectionName(m.activeSection)), "")
	}
	if viewport > 0 && offset < len(content) {
		end := min(offset+viewport, len(content))
		lines = append(lines, content[offset:end]...)
	}
	lines = append(lines, "", popupHelp.Render(hiddenHotkeyPopupFooter))
	return strings.Join(lines, "\n")
}

func hiddenHotkeyPopupLayout(m model, bodyWidth, bodyHeight int) (string, int) {
	// The content has to be built before it can be measured, and it is fitted
	// to the width it is built at. So: build once at the ceiling the terminal
	// allows, narrow to what the content actually needs, then rebuild. Narrowing
	// never lengthens a line, so the second pass settles.
	//
	// The measure covers every line, not the scrolled window, so the box does
	// not resize under the reader while they scroll. Same reasoning as the graph
	// column budget (docs/decisions.md 2026-09-10).
	popupWidth := hiddenHotkeyPopupWidth(bodyWidth)
	content := hiddenHotkeyContentLines(m, popupWidth)
	popupWidth = popupWidthForContent(content, bodyWidth, hiddenHotkeyPopupMinWidth, hiddenHotkeyPopupMaxWidth)
	content = hiddenHotkeyContentLines(m, popupWidth)
	popupBox := hiddenHotkeyPopupStyle(popupWidth)
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

func renderHiddenHotkeyItemLines(items []hiddenHotkeyItem, width int) []string {
	if len(items) == 0 {
		return []string{"  " + muted.Render("(none)")}
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fitVisibleWidth("  • "+renderHotkey(item.key)+": "+item.desc, width))
	}
	return lines
}

func hiddenHotkeySections(m model) []hiddenHotkeySection {
	return []hiddenHotkeySection{
		{
			title: "Global",
			items: append(globalHotkeyItems(),
				hiddenHotkeyItem{key: "gg", desc: "top"},
				hiddenHotkeyItem{key: "G", desc: "bottom"},
				hiddenHotkeyItem{key: "ctrl+u/d", desc: "scroll"},
			),
		},
		{
			title:  "Graph",
			active: m.activeSection == sectionGraph,
			// p and a used to be listed here. Neither reaches a handler under
			// Graph focus: handleBrowseKey routes Graph to handleBrowseGraphKey,
			// which has no case for either, and there is no fall-through to the
			// section handler that owns them. The popup was naming keys that do
			// nothing where it said they work.
			items: []hiddenHotkeyItem{
				{key: "enter", desc: "open commit inspector"},
				{key: "m", desc: "merge"},
				{key: "r", desc: "rebase"},
				{key: "space", desc: "checkout"},
				{key: "H", desc: "jump to HEAD"},
				{key: "w", desc: "road: pick from, then to"},
				{key: "s", desc: "reset"},
				{key: "d", desc: "delete branch"},
				{key: "P", desc: "push"},
				{key: "t", desc: "tag commit"},
				{key: "o", desc: "pop stash"},
				{key: "n", desc: "new branch or repeat search"},
				{key: "N", desc: "repeat search backward"},
			},
		},
		{
			title:  "Local",
			active: m.activeSection == sectionCurrent,
			items: []hiddenHotkeyItem{
				{key: "s", desc: "stash changes"},
				{key: "c", desc: "clean working tree"},
				{key: "space", desc: "checkout"},
				{key: "d", desc: "delete branch"},
				{key: "n", desc: "new branch"},
				{key: "a", desc: "abort merge"},
				{key: "p", desc: "pull"},
				{key: "P", desc: "push"},
			},
		},
		{
			title:  "Remote",
			active: m.activeSection == sectionRemote,
			items: []hiddenHotkeyItem{
				{key: "space", desc: "checkout"},
				{key: "d", desc: "delete remote branch"},
			},
		},
		{
			title:  "Tags",
			active: m.activeSection == sectionTags,
			items: []hiddenHotkeyItem{
				{key: "enter", desc: "jump to graph"},
				{key: "P", desc: "push tag"},
				{key: "d", desc: "delete tag"},
				{key: "D", desc: "delete tag on remote"},
			},
		},
	}
}
