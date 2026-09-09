package app

import "github.com/charmbracelet/lipgloss"

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func emptyDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

// ellipsis is the one truncation marker. Before this the app used three: "..."
// in the graph, "…" in the Inspector, and nothing at all in the rail, where a
// hard cut dropped a closing bracket silently and read as broken rather than
// shortened.
const ellipsis = "…"

// shorten takes the first n characters of an identifier - a hash, a ref - and
// adds no marker, because a shortened hash is still a hash and an ellipsis on
// one reads as part of the value.
//
// shorten abbreviates a commit hash. That is its whole job: a reader already
// knows a 7-character hash is the head of a longer one, so the cut needs no
// marker. Everything else that can overflow -- names, subjects, messages,
// labels -- goes through truncateText, which marks the cut. A silently
// shortened branch name reads as a different branch, not a shortened one.
//
// It counts runes, not bytes. It used to slice bytes, so shorten("한글제목입니다", 8)
// returned "한글\xec\xa0" and put a broken byte sequence on screen. Hex hashes
// never showed it; subjects and tag names did.
//
// For prose, use truncateText: it marks what it removed.
func shorten(v string, n int) string {
	if v == "" || n <= 0 {
		return ""
	}
	runes := []rune(v)
	if len(runes) <= n {
		return v
	}
	return string(runes[:n])
}

// truncateText clamps prose to width and marks that it did. Width is measured in
// terminal cells, so a wide character costs two.
//
// This is the content counterpart to fitVisibleWidth, which clamps composed
// lines and frames and deliberately adds no marker: an ellipsis on a border is
// nonsense. Text goes through here; structure goes through there.
func truncateText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width <= lipgloss.Width(ellipsis) {
		return fitVisibleWidth(ellipsis, width)
	}
	return fitVisibleWidth(value, width-lipgloss.Width(ellipsis)) + ellipsis
}
