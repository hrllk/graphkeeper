package app

import (
	"time"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

// ScreenProjection is the renderer input. It deliberately contains screen
// values and navigation metadata, not a repository or a mutation command.
type ScreenProjection struct {
	Width       int
	Height      int
	Graph       GraphProjection
	Sections    map[graphSection]SectionProjection
	Status      state.Status
	Active      graphSection
	TagSyncDone bool
}

type GraphProjection struct {
	Rows          []graphRow
	PageSize      int
	Scroll        int
	Cursor        int
	LaneCursor    int
	Active        bool
	LocalBranches []string
	Handshake     map[string]bool
	StashCounts   map[string]int
	SearchQuery   string
	StateHint     string
}

type SectionProjection struct {
	Items  []state.TargetItem
	Cursor int
	Active bool
}

// ContextProjection contains the already assembled lines consumed by the
// context renderer. Keeping viewport calculation in the renderer means the
// projection remains independent from terminal dimensions except for line
// wrapping performed by the existing detail helpers.
type ContextProjection struct {
	Title          string
	InfoLines      []string
	ActionLines    []string
	Scroll         int
	Recommendation *DivergedRecommendation
	Decision       *BranchDecisionContext
}

func (m model) screenProjection(width, height int) ScreenProjection {
	graph := graphRows(m.repoStatus)
	hint := repositoryStateHintForModel(&m)
	pageSize := graphPageSizeForRowsWithHint(&m, graph, m.graphScroll, max(height, 1), hint != "")
	stashCounts := make(map[string]int)
	for _, row := range graph {
		if row.Commit.Hash != "" {
			stashCounts[row.Commit.Hash] = len(m.stashesForCommit(row.Commit.Hash))
		}
	}
	sections := make(map[graphSection]SectionProjection, 3)
	for _, section := range []graphSection{sectionCurrent, sectionRemote, sectionTags} {
		sections[section] = SectionProjection{
			Items:  sectionTargets(m.repoStatus, section),
			Cursor: m.sectionCursor[section],
			Active: m.activeSection == section,
		}
	}
	return ScreenProjection{
		Width:       width,
		Height:      height,
		Graph:       GraphProjection{Rows: graph, PageSize: pageSize, Scroll: m.graphScroll, Cursor: m.sectionCursor[sectionGraph], LaneCursor: m.graphLaneCursor, Active: m.activeSection == sectionGraph, LocalBranches: append([]string(nil), m.repoStatus.LocalBranches...), Handshake: m.handshakeCommits, StashCounts: stashCounts, SearchQuery: m.graphSearchQuery, StateHint: hint},
		Sections:    sections,
		Status:      m.status,
		Active:      m.activeSection,
		TagSyncDone: m.tagSyncAttempted,
	}
}

func (m model) contextProjection(width int) ContextProjection {
	var recommendation *DivergedRecommendation
	var decision *BranchDecisionContext
	if m.activeSection == sectionCurrent {
		snapshot := divergedSnapshotFromStatus(m.repoStatus, m.repositoryEpoch)
		if m.status.Mode == state.ModeBrowse {
			if value, ok := explainBranchDecision(snapshot); ok {
				decision = &value
			}
		}
		if m.activePullRequest == nil {
			if value, ok := recommendDivergedPull(snapshot); ok {
				recommendation = &value
			}
		}
	}
	return ContextProjection{
		Title:       sectionName(m.activeSection),
		InfoLines:   m.renderContextInfoLines(width),
		ActionLines: renderActionHelpLines(m),
		Scroll:      m.contextScroll, Recommendation: recommendation, Decision: decision,
	}
}

// repositorySnapshot is the app-facing read contract used by the workflow
// seam. git.Status is converted here and does not cross into render helpers.
type repositorySnapshot struct {
	Epoch      uint64
	CapturedAt time.Time
	Status     git.Status
	Valid      snapshotValidity
}

type snapshotValidity struct {
	Graph    bool
	Branches bool
	Tags     bool
	Worktree bool
}
