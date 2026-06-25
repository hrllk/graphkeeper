package app

import (
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
	totalWidth := int(float64(m.width) * 0.70)
	if totalWidth < 80 {
		totalWidth = 80
	}
	if totalWidth > m.width {
		totalWidth = m.width
	}

	totalHeight := int(float64(m.height) * 0.76)
	if totalHeight < 18 {
		totalHeight = 18
	}
	if totalHeight > m.height-2 {
		totalHeight = m.height - 2
	}

	leftColWidth := int(float64(totalWidth) * 0.70)
	rightColWidth := totalWidth - leftColWidth - 2

	topHeight, bottomHeight := splitDashboardHeights(totalHeight)
	graphHeight, detailHeight := splitPaneHeights(bottomHeight)
	graphWidth, detailWidth := splitPaneWidths(totalWidth)

	localWidth := leftColWidth / 2
	remoteWidth := leftColWidth - localWidth

	localContent := m.renderSectionContent(sectionCurrent, localWidth-4, topHeight-3)
	remoteContent := m.renderSectionContent(sectionRemote, remoteWidth-4, topHeight-3)

	localBox := m.getBoxStyle(sectionCurrent).Width(localWidth).Height(topHeight).Render("Local\n" + localContent)
	remoteBox := m.getBoxStyle(sectionRemote).Width(remoteWidth).Height(topHeight).Render("Remote\n" + remoteContent)

	branchesInner := lipgloss.JoinHorizontal(lipgloss.Top, localBox, remoteBox)

	tagsContent := m.renderSectionContent(sectionTags, rightColWidth-4, topHeight-3)
	tagsBox := m.getBoxStyle(sectionTags).Width(rightColWidth).Height(topHeight).Render("Tags\n" + tagsContent)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, branchesInner, tagsBox)

	graphContent := m.renderGraphContent(graphWidth-4, graphHeight-3)
	graphBox := m.getBoxStyle(sectionGraph).Width(graphWidth).Height(graphHeight).Render("Graph (local branches)\n" + graphContent)

	detailContent := m.renderDetailContent(detailWidth-4, detailHeight-3)
	detailBox := baseBox.Width(detailWidth).Height(detailHeight).Render(detailContent)

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, graphBox, detailBox)
	body := lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)

	centeredBody := lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center, body)

	if m.status.Mode == state.ModeConfirm {
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		popupBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2).
			Width(50).
			Align(lipgloss.Center)
		popupTitle := m.status.Title
		if popupTitle == "" || popupTitle == "Confirm" {
			popupTitle = "Do you want to continue?"
		}
		helpText := "y: yes  •  n: no"
		if m.status.Action == state.ActionPull && !m.pullIsFastForward {
			helpText = "m: merge  •  r: rebase  •  esc: cancel"
		}
		popupContent := popupBox.Render(
			titleStyle.Render(popupTitle) + "\n\n" +
				descStyle.Render(m.status.Detail) + "\n\n" +
				helpStyle.Render(helpText),
		)
		centeredBody = overlayPopup(centeredBody, popupContent)
	}

	footer := muted.Render("global: 1 local  •  2 remote  •  3 tags  •  4 graph  •  tab/shift+tab section  •  up/down/j/k move  •  f fetch  •  q quit")

	return centeredBody + "\n" + footer + "\n"
}
