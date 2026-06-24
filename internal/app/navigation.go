package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"hrllk/git-graph-tui/internal/git"
	"hrllk/git-graph-tui/internal/graph"
	"hrllk/git-graph-tui/internal/state"
)

func graphNodes(rs git.Status) []graphNode {
	return graph.Nodes(rs)
}

func graphRows(rs git.Status) []graphRow {
	return graph.Rows(rs)
}

func graphRowWidth(row graphRow) int {
	return graph.RowWidth(row)
}

func findGraphRowByHash(rows []graphRow, hash string) int {
	return graph.FindRowByHash(rows, hash)
}

func graphPageSize(m *model) int {
	return graph.PageSize(m.height)
}

func moveSelectableGraphPointer(current int, rows []graphRow, delta int) int {
	return graph.MoveSelectableGraphPointer(current, rows, delta)
}

func nearestSelectableGraphRow(rows []graphRow, start, step int) int {
	return graph.NearestSelectableGraphRow(rows, start, step)
}

func graphPointerLane(row graphRow) int {
	return graph.PointerLane(row)
}

func currentGraphFocus(rs git.Status, cursor int) graphNode {
	return graph.CurrentFocus(rs, cursor)
}

func indexOf(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

func lastIndexOf(values []laneRef, target string) int {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i].Hash == target {
			return i
		}
	}
	return -1
}

func pendingChildren(children []string, current string) []string {
	for i, child := range children {
		if child == current {
			return append([]string(nil), children[i+1:]...)
		}
	}
	return nil
}

func syncBrowseState(m *model, rs git.Status) {
	currentHash := ""
	if rows := graph.Rows(m.repoStatus); m.sectionCursor[sectionGraph] >= 0 && m.sectionCursor[sectionGraph] < len(rows) {
		currentHash = rows[m.sectionCursor[sectionGraph]].Commit.Hash
	}
	rowCount := len(graph.Rows(rs))
	m.graphScroll = clampScroll(m.graphScroll, rowCount, graphPageSize(m))
	for _, section := range graphSectionOrder() {
		limit := len(sectionTargets(rs, section))
		if limit == 0 {
			m.sectionCursor[section] = -1
			continue
		}
		m.sectionCursor[section] = clampCursor(m.sectionCursor[section], limit)
	}
	if rowCount > 0 {
		rows := graph.Rows(rs)
		row := graph.FindRowByHash(rows, currentHash)
		if row < 0 {
			row = clampCursor(m.sectionCursor[sectionGraph], len(rows))
			if row >= 0 {
				row = graph.NearestSelectableGraphRow(rows, row, 1)
			}
		}
		m.sectionCursor[sectionGraph] = row
		m.graphLaneCursor = graph.PointerLane(rows[row])
	}
}

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

func moveBrowseCursor(m model, delta int) model {
	switch m.activeSection {
	case sectionGraph:
		rows := graph.Rows(m.repoStatus)
		cursor := graph.MoveSelectableGraphPointer(m.sectionCursor[sectionGraph], rows, delta)
		m.sectionCursor[sectionGraph] = cursor
		page := graphPageSize(&m)
		if cursor < m.graphScroll {
			m.graphScroll = cursor
		} else if cursor >= m.graphScroll+page {
			m.graphScroll = cursor - page + 1
		}
		if cursor >= 0 && cursor < len(rows) {
			m.graphLaneCursor = graph.PointerLane(rows[cursor])
		}
	case sectionCurrent, sectionLocal, sectionRemote, sectionTags:
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
	}
	return m
}

func moveGraphLane(m model, delta int) model {
	rows := graph.Rows(m.repoStatus)
	if len(rows) == 0 {
		return m
	}
	row := clampCursor(m.sectionCursor[sectionGraph], len(rows))
	m.graphLaneCursor = moveLanePointer(m.graphLaneCursor, rows[row], delta)
	return m
}

func pageBrowseGraph(m model, pages int) model {
	total := len(graph.Rows(m.repoStatus))
	if total == 0 {
		return m
	}
	page := graphPageSize(&m)
	delta := page * pages
	rows := graph.Rows(m.repoStatus)
	cursor := graph.MoveSelectableGraphPointer(m.sectionCursor[sectionGraph], rows, delta)
	m.sectionCursor[sectionGraph] = cursor
	m.graphScroll = clampScroll(cursor, total, page)
	if cursor >= 0 && cursor < len(rows) {
		m.graphLaneCursor = graph.PointerLane(rows[cursor])
	}
	return m
}

func maybeLoadMoreGraph(m model) (model, tea.Cmd) {
	if m.commitLimit <= 0 {
		return m, nil
	}
	if m.activeSection != sectionGraph {
		return m, nil
	}
	rows := graph.Rows(m.repoStatus)
	if len(rows) != m.commitLimit {
		return m, nil
	}
	if m.sectionCursor[sectionGraph] < m.commitLimit-graphLoadThreshold {
		return m, nil
	}
	m.commitLimit += graphLoadIncrement
	return m, refreshRepoState(m.repo, m.commitLimit)
}

func moveGraphScroll(current, total, delta int) int {
	if total <= 0 {
		return 0
	}
	next := current + delta
	if next < 0 {
		next = 0
	}
	maxScroll := max(0, total-1)
	if next > maxScroll {
		next = maxScroll
	}
	return next
}

func clampScroll(current, total, page int) int {
	if total <= 0 {
		return 0
	}
	maxScroll := max(0, total-page)
	if current < 0 {
		return 0
	}
	if current > maxScroll {
		return maxScroll
	}
	return current
}

func moveTarget(s state.Status, delta int) state.Status {
	if s.Mode != state.ModeTargetPick || len(s.Targets) == 0 {
		return s
	}
	next := s.TargetIdx + delta
	if next < 0 {
		next = len(s.Targets) - 1
	}
	if next >= len(s.Targets) {
		next = 0
	}
	s.TargetIdx = next
	s.Selected = s.Targets[next].Ref
	return s
}

func moveGraphPointer(current, total, delta int) int {
	if total <= 0 {
		return -1
	}
	if current < 0 {
		current = 0
	}
	next := current + delta
	if next < 0 {
		return 0
	}
	if next >= total {
		return total - 1
	}
	return next
}

func moveLanePointer(current int, row graphRow, delta int) int {
	maxLane := graph.RowWidth(row) - 1
	if maxLane < 0 {
		return 0
	}
	if current < 0 {
		current = graph.PointerLane(row)
	}
	next := current + delta
	if next < 0 {
		next = 0
	}
	if next > maxLane {
		next = maxLane
	}
	return next
}

func clampLaneCursor(current int, row graphRow) int {
	maxLane := graph.RowWidth(row) - 1
	if maxLane < 0 {
		return 0
	}
	if current < 0 || current > maxLane {
		return min(graph.PointerLane(row), maxLane)
	}
	return current
}

func clampCursor(current, total int) int {
	if total <= 0 {
		return -1
	}
	if current < 0 || current >= total {
		return 0
	}
	return current
}
