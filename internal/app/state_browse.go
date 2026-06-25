package app

import (
	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
)

func syncBrowseStateFromGraph(m *model, rs git.Status) {
	currentHash := ""
	if rows := graph.Rows(m.repoStatus); m.sectionCursor[sectionGraph] >= 0 && m.sectionCursor[sectionGraph] < len(rows) {
		currentHash = rows[m.sectionCursor[sectionGraph]].Commit.Hash
	}
	rowCount := len(graph.Rows(rs))
	m.graphScroll = clampScroll(m.graphScroll, rowCount, graphPageSize(m))
	if rowCount == 0 {
		return
	}
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

func syncBrowseStateSectionCursors(m *model, rs git.Status) {
	for _, section := range graphSectionOrder() {
		if section == sectionGraph {
			continue
		}
		limit := len(sectionTargets(rs, section))
		if limit == 0 {
			m.sectionCursor[section] = -1
			continue
		}
		m.sectionCursor[section] = clampCursor(m.sectionCursor[section], limit)
	}
}

func syncBrowseStateSelection(m *model, rs git.Status) {
	_ = rs
	if m.sectionCursor[sectionGraph] < 0 {
		m.graphLaneCursor = 0
	}
}
