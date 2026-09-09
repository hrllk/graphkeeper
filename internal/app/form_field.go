package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// A form popup renders read-only context and editable fields on the same
// baseline, so the two have to look different or the reader cannot tell which
// line accepts typing. The Create tag popup drew "target: <hash>" and
// "name: <draft>" identically, and centred both, so typing pushed the label
// leftward as the value grew out from the middle.
//
// The fix has three parts, and all three are needed:
//
//   - Labels pad to one width, so every value starts at the same column.
//   - The editable field is underlined for its full extent, so it is visible
//     even when empty and it does not move as the value grows.
//   - A reverse-video block marks the caret.
//
// Underline and reverse are attributes, not colour, so they survive NO_COLOR
// where lipgloss drops both. Same reasoning as cursorSignal, and the same
// escapes the graph already uses for road marks.
const formFieldMinWidth = 16

// formLabelColumn is where values start: the longest label, its colon, and one
// space of gap.
func formLabelColumn(labelWidth int) int { return labelWidth + 2 }

// formLabel pads a label so values line up in one column.
func formLabel(text string, labelWidth int) string {
	return padRight(text+":", formLabelColumn(labelWidth))
}

// formValue renders a read-only value. It carries no field marking, which is
// the whole point: the absence is what tells a reader it cannot be typed into.
func formValue(value string) string {
	return muted.Render(value)
}

// formInput renders an editable field of a fixed extent with the caret at the
// end of the value. The extent does not depend on the value, so the field
// neither grows nor shifts while the user types.
func formInput(value string, width int) string {
	if width < 1 {
		width = 1
	}
	// One cell of the extent belongs to the caret.
	body := value
	if lipgloss.Width(body) > width-1 {
		body = truncateText(body, width-1)
	}
	trailing := width - lipgloss.Width(body) - 1
	if trailing < 0 {
		trailing = 0
	}
	field := underlineSignal(body) + cursorSignal(" ")
	if trailing > 0 {
		field += underlineSignal(strings.Repeat(" ", trailing))
	}
	return field
}

// formFieldWidth is what is left for the field once the label column and the
// popup's own padding are taken out.
// The minimum keeps an empty field visible, but it cannot outrank the box it
// sits in: as a hard floor it pushed the field past the border and lipgloss
// grew the popup a column wider than the terminal allowed.
func formFieldWidth(innerWidth, labelWidth int) int {
	available := innerWidth - formLabelColumn(labelWidth)
	width := min(formFieldMinWidth, available)
	if available > formFieldMinWidth {
		width = available
	}
	if width < 1 {
		width = 1
	}
	return width
}
