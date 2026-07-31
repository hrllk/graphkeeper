package app

import "github.com/charmbracelet/lipgloss"

const (
	ansiBlack         uint = 0
	ansiRed           uint = 1
	ansiGreen         uint = 2
	ansiYellow        uint = 3
	ansiBlue          uint = 4
	ansiMagenta       uint = 5
	ansiCyan          uint = 6
	ansiWhite         uint = 7
	ansiBrightBlack   uint = 8
	ansiBrightRed     uint = 9
	ansiBrightGreen   uint = 10
	ansiBrightYellow  uint = 11
	ansiBrightBlue    uint = 12
	ansiBrightMagenta uint = 13
	ansiBrightCyan    uint = 14
	ansiBrightWhite   uint = 15
)

func ansiStyle(color uint) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(color))
}

func ansiBoldStyle(color uint) lipgloss.Style {
	return ansiStyle(color).Bold(true)
}

var (
	border          = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	baseBox         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.ANSIColor(ansiBrightBlack)).Padding(0, 1)
	activeBox       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.ANSIColor(ansiMagenta)).Padding(0, 1)
	title           = lipgloss.NewStyle().Bold(true)
	sectionTitle    = ansiBoldStyle(ansiBlue)
	contextValue    = ansiStyle(ansiBlue)
	hotkey          = ansiBoldStyle(ansiMagenta)
	muted           = lipgloss.NewStyle()
	accent          = ansiBoldStyle(ansiYellow)
	warn            = ansiBoldStyle(ansiYellow)
	ok              = ansiBoldStyle(ansiGreen)
	disabled        = lipgloss.NewStyle()
	headMark        = ansiBoldStyle(ansiGreen)
	branchMark      = ansiBoldStyle(ansiYellow)
	pointerMark     = ansiBoldStyle(ansiYellow)
	dirtyMark       = ansiBoldStyle(ansiRed)
	stashMark       = ansiStyle(ansiYellow)
	remoteColor     = ansiStyle(ansiBlue)
	tagColor        = ansiStyle(ansiMagenta)
	tagOverlapColor = ansiBoldStyle(ansiMagenta)
	highlight       = lipgloss.NewStyle().Reverse(true).Bold(true)
	conflictColor   = ansiBoldStyle(ansiRed)
	conflictMark    = ansiBoldStyle(ansiRed)
	reviewCurrent   = headMark
	reviewTarget    = ansiBoldStyle(ansiBlue)
	reviewBase      = ansiBoldStyle(ansiBrightBlack)
	reviewHash      = lipgloss.NewStyle().Bold(true)
	reviewBranch    = lipgloss.NewStyle()
	reviewMark      = lipgloss.NewStyle()
	reviewCount     = lipgloss.NewStyle().Bold(true)
	reviewFooter    = lipgloss.NewStyle()

	popupBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.ANSIColor(ansiMagenta))
	popupBody     = lipgloss.NewStyle()
	popupHelp     = lipgloss.NewStyle()
	popupHeader   = lipgloss.NewStyle().Bold(true)
	errorStyle    = ansiBoldStyle(ansiRed)
	handshakeMark = lipgloss.NewStyle().Reverse(true).Bold(true)

	searchMatchMark = lipgloss.NewStyle().Underline(true).Bold(true)
	searchFocusMark = lipgloss.NewStyle().Reverse(true).Bold(true)
)
