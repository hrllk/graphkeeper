package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

type model struct {
	repo              *git.Repo
	status            state.Status
	repoStatus        git.Status
	stashEntries      []git.StashEntry
	stashByBase       map[string][]git.StashEntry
	activeSection     graphSection
	sectionCursor     map[graphSection]int
	graphLaneCursor   int
	graphScroll       int
	awaitingGoTop     bool
	branchOpen        bool
	branchDraft       string
	branchBase        string
	width             int
	height            int
	commitLimit       int
	err               error
	handshakeCommits  map[string]bool
	pullIsFastForward bool
}

type graphSection int

const (
	sectionGraph graphSection = iota
	sectionCurrent
	sectionRemote
	sectionTags
)

func New(repo *git.Repo) (tea.Model, error) {
	m := model{
		repo:          repo,
		status:        loadingToast("Loading..."),
		activeSection: sectionGraph,
		sectionCursor: map[graphSection]int{
			sectionGraph:   0,
			sectionCurrent: 0,
			sectionRemote:  0,
			sectionTags:    0,
		},
		graphLaneCursor:  0,
		commitLimit:      0,
		handshakeCommits: make(map[string]bool),
		stashByBase:      make(map[string][]git.StashEntry),
	}
	return m, nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadRepoState(m.repo, m.commitLimit), scheduleRefresh())
}
