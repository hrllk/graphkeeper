package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

type model struct {
	repo       *git.Repo
	status     state.Status
	repoStatus git.Status

	// Repository-derived caches and snapshots.
	tagEntries        []git.TagEntry
	tagSyncAttempted  bool
	stashEntries      []git.StashEntry
	stashByBase       map[string][]git.StashEntry
	handshakeCommits  map[string]bool
	pullIsFastForward bool

	// Section navigation and graph focus.
	activeSection   graphSection
	sectionCursor   map[graphSection]int
	graphLaneCursor int
	graphScroll     int
	awaitingGoTop   bool

	// Graph search state.
	graphSearchOpen   bool
	graphSearchDraft  string
	graphSearchQuery  string
	graphSearchIndex  []graphSearchEntry
	graphSearchCursor int
	graphSearchError  string

	// Modal and popup state.
	branchOpen           bool
	branchDraft          string
	branchBase           string
	branchError          string
	tagPopupOpen         bool
	tagPopupDraft        string
	tagPopupError        string
	tagPopupTarget       string
	stashMessageOpen     bool
	stashMessageDraft    string
	stashMessageError    string
	stashPopupOpen       bool
	stashPopupCursor     int
	graphStashPopOpen    bool
	graphStashPopMode    graphStashPopMode
	graphStashPopCursor  int
	graphStashPopEntries []git.StashEntry
	hiddenHotkeysOpen    bool
	hiddenHotkeysScroll  int

	// Viewport and transient errors.
	width       int
	height      int
	commitLimit int
	err         error

	// repositoryEpoch invalidates repository reads that started before a user
	// operation. Bubble Tea commands run concurrently, so an older refresh
	// must not overwrite a mutation's result when it completes later.
	repositoryEpoch uint64
}

type graphSection int

type graphStashPopMode int

const (
	graphStashPopModePicker graphStashPopMode = iota
	graphStashPopModeConfirm
)

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
		graphLaneCursor:   0,
		commitLimit:       0,
		handshakeCommits:  make(map[string]bool),
		stashByBase:       make(map[string][]git.StashEntry),
		graphStashPopMode: graphStashPopModePicker,
	}
	return m, nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadRepoState(m.repo, m.commitLimit, m.repositoryEpoch), scheduleRefresh())
}
