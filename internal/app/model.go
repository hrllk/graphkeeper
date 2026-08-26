package app

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"

	commitinspector "hrllk/graphkeeper/internal/commitinspector"
	"hrllk/graphkeeper/internal/events"
	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

type repositoryState struct {
	repoStatus         git.Status
	tagEntries         []git.TagEntry
	tagSyncAttempted   bool
	stashEntries       []git.StashEntry
	stashByBase        map[string][]git.StashEntry
	repoSnapshotLoaded bool
	graphReadSnapshot  graph.Snapshot
	repositoryEpoch    uint64
	refreshGeneration  uint64
	err                error
}

type pullState struct {
	handshakeCommits          map[string]bool
	pullIsFastForward         bool
	nextPullRequestID         uint64
	activePullRequest         *pullRequest
	pullCancel                context.CancelFunc
	pullConfirmStale          bool
	lastPullMode              PullMode
	lastPullOperationBaseline PullSnapshotIdentity
	operationResult           *OperationResultSummary
}

type pullConfirmInput struct {
	CurrentBranch string
	TargetRef     string
	CurrentOnly   int
	TargetOnly    int
	ImpactKnown   bool
	MergeText     string
	RebaseText    string
	RiskText      string
}

type inspectorState struct {
	commitInspectorOpen                 bool
	commitInspector                     CommitSnapshot
	commitInspectorSnapshot             CommitSnapshot
	commitInspectorDiffWindow           DiffWindow
	commitInspectorWindowRequest        DiffWindowRequest
	commitInspectorCursor               int
	commitInspectorScroll               int
	commitInspectorLines                []string
	commitInspectorHasMore              bool
	commitInspectorLoading              bool
	commitInspectorError                string
	commitInspectorRequest              uint64
	commitInspectorEpoch                uint64
	commitInspectorRequestedCommit      string
	commitInspectorRequestedParent      string
	commitInspectorCancel               context.CancelFunc
	commitInspectorContext              context.Context
	commitInspectorHelp                 bool
	commitInspectorMessage              bool
	commitInspectorMessageScroll        int
	commitInspectorMetadataLoading      bool
	commitInspectorDiffLoading          bool
	commitInspectorDiffError            string
	commitInspectorStale                bool
	commitInspectorRevalidating         bool
	commitInspectorSelectedFileID       string
	commitInspectorSelectedCanonicalKey string
	commitInspectorContinuationPending  bool
}

type overlayState struct {
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
	graphSearchOpen      bool
	graphSearchDraft     string
	graphSearchQuery     string
	graphSearchIndex     []graphSearchEntry
	graphSearchCursor    int
	graphSearchError     string
	mergeConfirmView     *mergeConfirmViewModel
}

type navigationState struct {
	activeSection   graphSection
	sectionCursor   map[graphSection]int
	graphLaneCursor int
	graphScroll     int
	contextScroll   int
	awaitingGoTop   bool
	width           int
	height          int
}

type model struct {
	repositoryState
	pullState
	inspectorState
	overlayState
	navigationState

	repo             *git.Repo
	repositoryRead   RepositoryReadPort
	pull             PullPort
	inspectorReader  commitinspector.CommitInspectorReader
	tagProvenance    TagProvenanceStore
	eventSink        events.EventSink
	status           state.Status
	pullConfirmInput *pullConfirmInput

	// Repository-derived caches and snapshots.

	// Viewport and transient errors.
	commitLimit        int
	startupReadPending bool
	startupFailed      bool
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

func NewWithDependencies(deps Dependencies) (tea.Model, error) {
	m := model{
		repositoryState: repositoryState{
			stashByBase: make(map[string][]git.StashEntry),
		},
		pullState: pullState{
			handshakeCommits: make(map[string]bool),
		},
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{
				sectionGraph:   0,
				sectionCurrent: 0,
				sectionRemote:  0,
				sectionTags:    0,
			},
			graphLaneCursor: 0,
		},
		repo:               deps.Repo,
		repositoryRead:     deps.RepositoryRead,
		pull:               deps.Pull,
		inspectorReader:    deps.InspectorReader,
		tagProvenance:      deps.TagProvenance,
		eventSink:          deps.EventSink,
		status:             loadingToast("Loading..."),
		commitLimit:        0,
		overlayState:       overlayState{graphStashPopMode: graphStashPopModePicker},
		startupReadPending: deps.RepositoryRead != nil,
		startupFailed:      false}
	return m, nil
}

func (m model) publish(source, name string, fields map[string]string) {
	if m.eventSink != nil {
		_ = m.eventSink.Publish(events.Event{Source: source, Name: name, Fields: fields})
	}
}

func (m model) Init() tea.Cmd {
	if m.repositoryRead != nil {
		return tea.Batch(loadRepositorySnapshot(m.repositoryRead, m.commitLimit, m.repositoryEpoch), scheduleRefresh())
	}
	return tea.Batch(loadRepoState(m.repo, m.commitLimit, m.repositoryEpoch, m.tagProvenance), scheduleRefresh())
}
