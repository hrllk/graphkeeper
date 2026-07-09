package app

import (
	"sort"
	"strings"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
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
	rows := graph.Rows(m.repoStatus)
	return graphPageSizeForRows(m, rows, m.graphScroll, graphContentHeightForModel(m))
}

func graphPageSizeForRows(m *model, rows []graphRow, start, contentHeight int) int {
	if len(rows) == 0 || contentHeight <= 0 {
		return 0
	}
	if start < 0 || start >= len(rows) {
		return 0
	}
	budget := contentHeight - 2
	if budget < 1 {
		budget = 1
	}
	used := 0
	count := 0
	rawGraph := rows[start].Graph != ""
	for i := start; i < len(rows); i++ {
		if used >= budget {
			break
		}
		used++
		count++
		if rawGraph || i+1 >= len(rows) {
			continue
		}
		isConnectorHandshake := rows[i].Commit.Hash != "" && m.handshakeCommits[rows[i].Commit.Hash] && rows[i+1].Commit.Hash != "" && m.handshakeCommits[rows[i+1].Commit.Hash]
		connectorLines := renderGraphConnectorLines(rows[i], rows[i+1], isConnectorHandshake)
		if len(connectorLines) == 0 {
			continue
		}
		if used+len(connectorLines) > budget {
			break
		}
		used += len(connectorLines)
	}
	if count < 1 {
		count = 1
	}
	return count
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

func graphCheckoutTarget(m model) (string, bool) {
	targets := graphCheckoutTargets(m)
	if len(targets) == 0 {
		return "", false
	}
	return targets[0].Ref, true
}

func graphCheckoutTargets(m model) []state.TargetItem {
	if !isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
		return nil
	}
	focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
	names := graphLocalBranchNames(focus.Decorations, m.repoStatus.LocalBranches)
	if len(names) == 0 {
		return nil
	}
	if len(names) > 1 {
		sort.SliceStable(names, func(i, j int) bool {
			if names[i] == m.repoStatus.Branch {
				return true
			}
			if names[j] == m.repoStatus.Branch {
				return false
			}
			return names[i] < names[j]
		})
	}
	targets := make([]state.TargetItem, 0, len(names))
	for _, name := range names {
		targets = append(targets, state.TargetItem{
			Kind:    state.TargetKindLocal,
			Name:    name,
			Ref:     name,
			Current: !m.repoStatus.Detached && name == m.repoStatus.Branch,
		})
	}
	return targets
}

func graphLocalBranchNames(decorations []string, localBranches []string) []string {
	localSet := make(map[string]struct{}, len(localBranches))
	for _, branch := range localBranches {
		branch = strings.TrimSpace(branch)
		if branch != "" {
			localSet[branch] = struct{}{}
		}
	}
	seen := make(map[string]struct{}, len(decorations))
	names := make([]string, 0, len(decorations))
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || name == "HEAD" {
			return
		}
		if strings.HasPrefix(name, "origin/") || strings.HasPrefix(name, "tag: ") {
			return
		}
		if len(localSet) > 0 {
			if _, ok := localSet[name]; !ok {
				return
			}
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for _, decoration := range decorations {
		decoration = strings.TrimSpace(decoration)
		if decoration == "" {
			continue
		}
		if strings.HasPrefix(decoration, "HEAD -> ") {
			add(strings.TrimPrefix(decoration, "HEAD -> "))
			continue
		}
		add(decoration)
	}
	return names
}

func focusGraphHead(m *model, rs git.Status) {
	_ = focusGraphCommit(m, rs, rs.Head)
}

func focusGraphCommit(m *model, rs git.Status, hash string) bool {
	m.activeSection = sectionGraph
	rows := graph.Rows(rs)
	if len(rows) == 0 {
		m.sectionCursor[sectionGraph] = 0
		m.graphScroll = 0
		m.graphLaneCursor = 0
		return hash == ""
	}

	row := graph.FindRowByHash(rows, hash)
	if row < 0 {
		return false
	}
	m.sectionCursor[sectionGraph] = row
	page := graphPageSizeForRows(m, rows, row, graphContentHeightForModel(m))
	m.graphScroll = clampScroll(row, len(rows), page)
	m.graphLaneCursor = graph.PointerLane(rows[row])
	return true
}

func moveGraphBrowseCursor(m model, delta int) model {
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
