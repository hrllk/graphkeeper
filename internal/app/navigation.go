package app

import "hrllk/graphkeeper/internal/git"

func syncBrowseState(m *model, rs git.Status) {
	syncBrowseStateFromGraph(m, rs)
	syncBrowseStateSectionCursors(m, rs)
	syncBrowseStateSelection(m, rs)
}

func moveBrowseCursor(m model, delta int) model {
	switch m.activeSection {
	case sectionGraph:
		return moveGraphBrowseCursor(m, delta)
	case sectionCurrent, sectionRemote, sectionTags:
		return moveSectionBrowseCursor(m, delta)
	default:
		return m
	}
}
