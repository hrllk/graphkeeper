package app

import (
	"strings"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

func buildCherryPickTargets(rs git.Status) []state.TargetItem {
	rows := graphRows(rs)
	targets := make([]state.TargetItem, 0, len(rows))
	for _, row := range rows {
		if row.Commit.Hash == "" || row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
			continue
		}
		if row.Commit.Hash == rs.Head {
			continue
		}
		if len(row.Commit.Parents) > 1 {
			continue
		}
		targets = append(targets, state.TargetItem{
			Kind:        state.TargetKindCommit,
			Name:        row.Commit.Hash,
			Ref:         row.Commit.Hash,
			CommitHash:  row.Commit.Hash,
			Author:      row.Commit.Author,
			Subject:     row.Commit.Subject,
			RelativeAge: row.Commit.RelativeAge,
		})
	}
	return targets
}

func actionPickCherryTargets(rs git.Status) state.Status {
	targets := buildCherryPickTargets(rs)
	if len(targets) == 0 {
		return state.New().WithBlocked(state.BlockTargetEmpty, "No cherry-pick targets available.", "Select a commit with a single parent.")
	}
	status := state.New().WithCherryPickPick(targets)
	status.Selected = targets[0].Ref
	status.TargetIdx = 0
	return applyRepoMetadata(status, rs)
}

func focusCherryPickTarget(m *model, idx int) {
	rows := graphRows(m.repoStatus)
	if len(rows) == 0 {
		m.status.TargetIdx = -1
		m.status.Selected = ""
		m.sectionCursor[sectionGraph] = 0
		m.graphScroll = 0
		m.graphLaneCursor = 0
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(rows) {
		idx = len(rows) - 1
	}
	m.status.TargetIdx = idx
	m.sectionCursor[sectionGraph] = idx
	hint := repositoryStateHintForModel(m)
	m.graphScroll = clampScroll(idx, len(rows), graphPageSizeForRowsWithHint(m, rows, idx, graphContentHeightForModel(m), hint != ""))
	m.graphLaneCursor = graph.PointerLane(rows[idx])
	if idx < len(m.status.Targets) {
		m.status.Selected = m.status.Targets[idx].Ref
	}
}

func moveCherryPickTarget(s state.Status, delta int) state.Status {
	if s.Mode != state.ModeCherryPickPick || len(s.Targets) == 0 {
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

func cherryPickSelectionIndex(queue []string, target string) int {
	target = strings.TrimSpace(target)
	for i, ref := range queue {
		if strings.TrimSpace(ref) == target {
			return i
		}
	}
	return -1
}

func toggleCherryPickSelection(s state.Status) state.Status {
	target := selectedTarget(s)
	if target == "" {
		return s
	}
	if idx := cherryPickSelectionIndex(s.SelectedQueue, target); idx >= 0 {
		s.SelectedQueue = append(append([]string(nil), s.SelectedQueue[:idx]...), s.SelectedQueue[idx+1:]...)
		return s
	}
	s.SelectedQueue = append(append([]string(nil), s.SelectedQueue...), target)
	return s
}

func selectedCherryPickTargets(s state.Status) []string {
	if len(s.SelectedQueue) == 0 {
		return nil
	}
	return append([]string(nil), s.SelectedQueue...)
}

func cherryPickQueuePosition(queue []string, target string) (int, bool) {
	if idx := cherryPickSelectionIndex(queue, target); idx >= 0 {
		return idx + 1, true
	}
	return 0, false
}
