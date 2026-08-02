package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/state"
)

type alertContent struct {
	Title       string
	Description string
}

func blockedAlertContent(s state.Status) alertContent {
	titleText := s.Title
	if titleText == "" || titleText == "Blocked" {
		titleText = "Alert"
	}

	descriptionLines := make([]string, 0, 2)
	if s.Message != "" {
		descriptionLines = append(descriptionLines, s.Message)
	}
	if s.Detail != "" {
		descriptionLines = append(descriptionLines, s.Detail)
	}

	return alertContent{
		Title:       titleText,
		Description: strings.Join(descriptionLines, "\n"),
	}
}

func renderAlertPopup(alert alertContent, bodyWidth int) string {
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 28, 50)).
		Align(lipgloss.Center)

	lines := []string{
		descStyle.Render(alert.Description),
		"",
		helpStyle.Render("enter: dismiss"),
		renderPopupFooter(popupWidthForBody(bodyWidth, 28, 50) - 4),
	}
	return renderFloatingTitlePopup(
		popupBox,
		alert.Title,
		strings.Join(lines, "\n"),
		popupWidthForBody(bodyWidth, 28, 50),
	)
}
