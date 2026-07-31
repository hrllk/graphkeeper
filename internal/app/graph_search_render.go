package app

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func highlightSearchText(value, query string, focused bool) string {
	query = strings.TrimSpace(query)
	if query == "" || value == "" {
		return value
	}

	style := searchMatchMark
	if focused {
		style = searchFocusMark
	}

	re, err := regexp.Compile("(?i)" + query)
	if err != nil {
		return highlightLiteralText(value, query, style)
	}
	return re.ReplaceAllStringFunc(value, func(match string) string {
		return style.Render(match)
	})
}

func highlightLiteralText(value, query string, style lipgloss.Style) string {
	valueRunes := []rune(value)
	queryRunes := []rune(strings.ToLower(query))
	var b strings.Builder
	for i := 0; i < len(valueRunes); {
		if i+len(queryRunes) <= len(valueRunes) && strings.EqualFold(string(valueRunes[i:i+len(queryRunes)]), query) {
			b.WriteString(style.Render(string(valueRunes[i : i+len(queryRunes)])))
			i += len(queryRunes)
			continue
		}
		b.WriteRune(valueRunes[i])
		i++
	}
	return b.String()
}

func renderSearchField(value, query string, width int, focused bool) string {
	if value == "" {
		return padRight("", width)
	}
	value = highlightSearchText(value, query, focused)
	return padRight(value, width)
}
