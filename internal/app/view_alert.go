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

// renderAlertPopup uses the shared popup tokens rather than building its own
// styles. It used to reach for lipgloss.Color("252"), ("241") and ("205"), which
// are ANSI-256 values; highlighting-color-map.md's Policy allows only ANSI 0-15,
// and D-013 lists this file as one of the places still making colour outside
// theme.go.
//
// Neither "enter: dismiss" nor "esc: close" is drawn any more. Both keys still
// work; the alert just stops spending two of its rows saying so.
func renderAlertPopup(alert alertContent, bodyWidth int) string {
	popupBox := popupBorder.
		Padding(1, 2).
		Width(popupWidthForBody(bodyWidth, 28, 50)).
		Align(lipgloss.Center)

	lines := []string{
		popupBody.Render(alert.Description),
	}
	return renderFloatingTitlePopup(
		popupBox,
		alert.Title,
		strings.Join(lines, "\n"),
		popupWidthForBody(bodyWidth, 28, 50),
	)
}
