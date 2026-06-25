package app

import (
	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func graphSectionOrder() []graphSection {
	return []graphSection{sectionGraph, sectionCurrent, sectionRemote, sectionTags}
}

func sectionName(section graphSection) string {
	switch section {
	case sectionGraph:
		return "Graph"
	case sectionCurrent:
		return "Branches"
	case sectionRemote:
		return "Remote"
	case sectionTags:
		return "Tags"
	default:
		return "Unknown"
	}
}

func nextGraphSection(current graphSection) graphSection {
	order := graphSectionOrder()
	for i, section := range order {
		if section == current {
			return order[(i+1)%len(order)]
		}
	}
	return sectionGraph
}

func prevGraphSection(current graphSection) graphSection {
	order := graphSectionOrder()
	for i, section := range order {
		if section == current {
			return order[(i-1+len(order))%len(order)]
		}
	}
	return sectionGraph
}

func switchBrowseSection(m model, section graphSection) model {
	m.activeSection = section
	m.awaitingGoTop = false
	return m
}

func sectionTargets(rs git.Status, section graphSection) []state.TargetItem {
	switch section {
	case sectionCurrent:
		return buildCurrentSectionTargets(rs)
	case sectionRemote:
		return buildRemoteSectionTargets(rs)
	case sectionTags:
		return buildTagSectionTargets(rs)
	default:
		return nil
	}
}

func activeSectionTarget(m model) string {
	items := sectionTargets(m.repoStatus, m.activeSection)
	cursor := m.sectionCursor[m.activeSection]
	if cursor < 0 || cursor >= len(items) {
		return ""
	}
	return items[cursor].Ref
}

func moveSectionBrowseCursor(m model, delta int) model {
	items := sectionTargets(m.repoStatus, m.activeSection)
	if len(items) == 0 {
		m.sectionCursor[m.activeSection] = -1
		return m
	}
	cur := m.sectionCursor[m.activeSection]
	if cur < 0 || cur >= len(items) {
		cur = 0
	}
	next := cur + delta
	if next < 0 {
		next = len(items) - 1
	}
	if next >= len(items) {
		next = 0
	}
	m.sectionCursor[m.activeSection] = next
	return m
}
