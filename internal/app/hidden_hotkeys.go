package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/state"
)

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
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	popupWidth := popupWidthForBody(bodyWidth, 44, 72)
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidth).
		Align(lipgloss.Left)

	lines := []string{
		descStyle.Render(sectionName(m.activeSection) + " hidden hotkeys"),
		"",
	}
	lines = append(lines, renderHiddenHotkeyGroup("Visible", hiddenVisibleHotkeys(m))...)
	lines = append(lines, "")
	lines = append(lines, renderHiddenHotkeyGroup("Conditional", hiddenConditionalHotkeys(m))...)
	lines = append(lines, "")
	lines = append(lines, renderHiddenHotkeyGroup("Hidden / moved out", hiddenMovedOutHotkeys(m))...)
	lines = append(lines, "")
	lines = append(lines, helpStyle.Render("esc: close"))

	return renderFloatingTitlePopup(
		popupBox,
		"Hidden Hotkeys",
		strings.Join(lines, "\n"),
		popupWidth,
	)
}

func renderHiddenHotkeyGroup(title string, items []string) []string {
	lines := []string{title + ":"}
	if len(items) == 0 {
		lines = append(lines, "  (none)")
		return lines
	}
	for _, item := range items {
		lines = append(lines, "  • "+item)
	}
	return lines
}

func hiddenVisibleHotkeys(m model) []string {
	switch m.activeSection {
	case sectionGraph:
		return []string{
			"m: merge",
			"r: rebase",
			"space: checkout",
			"H: jump to HEAD",
		}
	case sectionCurrent:
		return []string{
			"s: stash changes",
			"c: clean working tree",
			"space: checkout",
			"d: delete branch",
		}
	case sectionRemote:
		return []string{
			"space: checkout",
			"f: fetch",
			"p: pull",
			"d: delete branch",
		}
	case sectionTags:
		return []string{
			"enter: jump to graph",
			"d: delete tag",
		}
	default:
		return nil
	}
}

func hiddenConditionalHotkeys(m model) []string {
	switch m.activeSection {
	case sectionGraph:
		return []string{
			"s: reset",
			"d: delete branch",
			"p: pull",
			"P: push",
			"t: tag commit",
			"o: pop stash",
			"n: new branch or repeat search",
			"N: repeat search backward",
		}
	case sectionCurrent:
		return []string{
			"a: abort merge",
			"P: push",
		}
	case sectionRemote:
		return []string{
			"push is read elsewhere",
		}
	case sectionTags:
		return []string{
			"tag metadata is shown in Details",
		}
	default:
		return nil
	}
}

func hiddenMovedOutHotkeys(m model) []string {
	lines := []string{
		"tab: next section",
		"shift+tab: previous section",
		"j/k: move",
		"f: fetch",
		"F: fetch tags",
		"S: stash list",
		"q: quit",
	}
	if m.status.Mode == state.ModeBrowse {
		lines = append(lines, "?: hidden hotkeys")
	}
	lines = append(lines,
		"gg: top",
		"G: bottom",
		"ctrl+u/d: scroll",
	)
	return lines
}
