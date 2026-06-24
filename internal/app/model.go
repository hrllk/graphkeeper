package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/git-graph-tui/internal/git"
	"hrllk/git-graph-tui/internal/graph"
	"hrllk/git-graph-tui/internal/state"
)

type model struct {
	repo               *git.Repo
	status             state.Status
	repoStatus         git.Status
	activeSection      graphSection
	sectionCursor      map[graphSection]int
	graphLaneCursor    int
	graphScroll        int
	awaitingGoTop      bool
	branchOpen         bool
	branchDraft        string
	branchBase         string
	width              int
	height             int
	commitLimit        int
	err                error
	handshakeCommits   map[string]bool
	pullIsFastForward  bool
}

type graphSection int

const (
	sectionGraph graphSection = iota
	sectionCurrent
	sectionLocal
	sectionRemote
	sectionTags
)

const (
	initialGraphCommitLimit = 0
	graphLoadIncrement      = 0
	graphLoadThreshold      = 0
)

func New(repo *git.Repo) (tea.Model, error) {
	m := model{
		repo:          repo,
		status:        state.New().WithLoading("Loading repository state..."),
		activeSection: sectionGraph,
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionLocal:   0,
			sectionRemote:  0,
			sectionTags:    0,
		},
		graphLaneCursor:  0,
		commitLimit:      initialGraphCommitLimit,
		handshakeCommits: make(map[string]bool),
	}
	return m, nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadRepoState(m.repo, m.commitLimit), scheduleRefresh())
}

type loadedMsg struct {
	status git.Status
	err    error
}

type tickMsg time.Time

type refreshedMsg struct {
	status git.Status
	err    error
}

type fetchedMsg struct {
	status git.Status
	err    error
}

type preparedMsg struct {
	action state.Action
	status git.Status
	err    error
}

type pullCheckedMsg struct {
	repo   git.Status
	status state.Status
	err    error
}

type previewMsg struct {
	action state.Action
	target string
	repo   git.Status
	status state.Status
	err    error
}

type executedMsg struct {
	action state.Action
	target string
	status git.Status
	err    error
}

type createdBranchMsg struct {
	name   string
	base   string
	status git.Status
	err    error
}

type graphNode = graph.Node

type laneSide = graph.LaneSide

const (
	laneLocal  = graph.LaneLocal
	laneRemote = graph.LaneRemote
	laneOther  = graph.LaneOther
)

type laneRef = graph.LaneRef

type graphRow = graph.Row

const (
	// graphViewHeightOffset은 그래프 렌더링 시 레이아웃 테두리(2줄), 페이지 정보 표시(1줄),
	// 컬럼 헤더(1줄), 기본 패딩 등을 고려하여 제외해야 하는 세로 높이 총합입니다.
	graphViewHeightOffset = 5
)

// isLocalGraphPointer returns true when the current graph cursor is pointing
// at a local-branch lane. Merge and Rebase from Graph are only allowed in this state.

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

func shorten(v string, n int) string {
	if v == "" || len(v) <= n {
		return v
	}
	return v[:n]
}

type pullFetchedMsg struct {
	status git.Status
	err    error
}

type pushFetchedMsg struct {
	status git.Status
	err    error
}

type pullPreviewReadyMsg struct {
	commits []string
	isFF    bool
	err     error
}

