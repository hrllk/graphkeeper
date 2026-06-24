package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/git-graph-tui/internal/git"
	"hrllk/git-graph-tui/internal/graph"
	"hrllk/git-graph-tui/internal/state"
	"hrllk/git-graph-tui/internal/telemetry"
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

func loadRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		status, err := repo.Status(context.Background(), limit)
		return loadedMsg{status: status, err: err}
	}
}

func scheduleRefresh() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func refreshRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		status, err := repo.Status(context.Background(), limit)
		return refreshedMsg{status: status, err: err}
	}
}

func fetchRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Fetch(context.Background()); err != nil {
			return fetchedMsg{err: err}
		}
		status, err := repo.Status(context.Background(), limit)
		return fetchedMsg{status: status, err: err}
	}
}

func prepareAction(repo *git.Repo, action state.Action, limit int) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Fetch(context.Background()); err != nil {
			return preparedMsg{action: action, err: err}
		}
		status, err := repo.Status(context.Background(), limit)
		return preparedMsg{action: action, status: status, err: err}
	}
}

func pullCheck(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Fetch(context.Background()); err != nil {
			return pullCheckedMsg{err: err}
		}
		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return pullCheckedMsg{err: err}
		}
		behind, ahead, err := repo.Divergence(context.Background(), status.Upstream, "HEAD")
		if err != nil {
			return pullCheckedMsg{err: err}
		}
		if ahead > 0 {
			return pullCheckedMsg{
				repo: status,
				status: state.New().WithBlocked(
					state.BlockDiverged,
					"Fast-forward is not possible.",
					"The branch has diverged from its upstream.",
				),
			}
		}
		_ = behind
		return pullCheckedMsg{
			repo: status,
			status: state.New().WithOutcome(
				state.ActionPull,
				"Fast-forward is possible.",
				"The upstream can move to the current branch tip.",
				true,
			),
		}
	}
}

func executePull(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Run("pull", "--no-rebase", "--no-edit")
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionPull, err: statusErr}
		}
		return executedMsg{action: state.ActionPull, status: status, err: err}
	}
}

func executePullMerge(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Run("pull", "--no-rebase", "--no-edit")
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionPullMerge, err: statusErr}
		}
		return executedMsg{action: state.ActionPullMerge, status: status, err: err}
	}
}

func executePullRebase(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Run("pull", "--rebase")
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionPullRebase, err: statusErr}
		}
		return executedMsg{action: state.ActionPullRebase, status: status, err: err}
	}
}

func executeAbort(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		// 현재 리포지토리 상태를 1차 파악하여 merge abort 인지 rebase abort 인지 구분합니다.
		currentStatus, statusErr := repo.Status(context.Background(), limit)
		var err error
		if statusErr == nil && currentStatus.RebaseInProgress {
			_, err = repo.Run("rebase", "--abort")
		} else {
			_, err = repo.Run("merge", "--abort")
		}
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionAbort, err: statusErr}
		}
		return executedMsg{action: state.ActionAbort, status: status, err: err}
	}
}

func executePush(repo *git.Repo, branch string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Push(context.Background(), branch, false, false)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionPush, target: branch, err: statusErr}
		}
		return executedMsg{action: state.ActionPush, target: branch, status: status, err: err}
	}
}

func executeForcePush(repo *git.Repo, branch string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Push(context.Background(), branch, true, false)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionForcePush, target: branch, err: statusErr}
		}
		return executedMsg{action: state.ActionForcePush, target: branch, status: status, err: err}
	}
}

func executePushSetUpstream(repo *git.Repo, branch string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Push(context.Background(), branch, false, true)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionSetUpstream, target: branch, err: statusErr}
		}
		return executedMsg{action: state.ActionSetUpstream, target: branch, status: status, err: err}
	}
}

func previewSelection(repo *git.Repo, rs git.Status, action state.Action, target string) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return previewMsg{action: action, target: target, repo: rs, err: fmt.Errorf("target is empty")}
		}
		if (action == state.ActionMerge || action == state.ActionRebase) && rs.Detached {
			return previewMsg{
				action: action,
				target: target,
				repo:   rs,
				status: state.New().WithBlocked(state.BlockDetached, "Detached HEAD.", "Choose a branch before merging or rebasing."),
			}
		}
		currentOnly, targetOnly, err := repo.Divergence(context.Background(), "HEAD", target)
		if err != nil {
			return previewMsg{action: action, target: target, repo: rs, err: err}
		}
		return previewMsg{
			action: action,
			target: target,
			repo:   rs,
			status: buildActionPreview(action, target, rs, currentOnly, targetOnly),
		}
	}
}

func executeAction(repo *git.Repo, action state.Action, target string, limit int) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return executedMsg{action: action, err: fmt.Errorf("target is empty")}
		}
		var err error
		switch action {
		case state.ActionMerge:
			_, err = repo.Run("merge", "--no-edit", target)
		case state.ActionRebase:
			_, err = repo.Run("rebase", target)
		case state.ActionReset:
			_, err = repo.Run("reset", "--hard", target)
		default:
			err = fmt.Errorf("unsupported action %q", action)
		}
		if err != nil {
			return executedMsg{action: action, target: target, err: err}
		}
		status, statusErr := repo.Status(context.Background(), limit)
		return executedMsg{action: action, target: target, status: status, err: statusErr}
	}
}

func createBranch(repo *git.Repo, name, base string, limit int) tea.Cmd {
	return func() tea.Msg {
		if name == "" {
			return createdBranchMsg{err: fmt.Errorf("branch name is empty")}
		}
		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return createdBranchMsg{name: name, base: base, err: err}
		}
		if status.WorktreeDirty {
			return createdBranchMsg{
				name: name,
				base: base,
				err:  fmt.Errorf("working tree is not clean"),
			}
		}
		if base == "" {
			base = "HEAD"
		}
		if _, err := repo.Run("switch", "-c", name, base); err != nil {
			return createdBranchMsg{name: name, base: base, err: err}
		}
		status, err = repo.Status(context.Background(), limit)
		return createdBranchMsg{name: name, base: base, status: status, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case loadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = m.status.WithError(msg.err.Error())
			telemetry.Log("app", "load_error", map[string]string{"error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		m.status = deriveStatus(msg.status)
		telemetry.Log("app", "load_repo", map[string]string{
			"root":   msg.status.Root,
			"branch": msg.status.Branch,
			"head":   msg.status.Head,
		})
		return m, nil
	case tickMsg:
		return m, tea.Batch(scheduleRefresh(), refreshRepoState(m.repo, m.commitLimit))
	case refreshedMsg:
		if msg.err != nil {
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		if !m.branchOpen && (m.status.Mode == state.ModeBrowse || m.status.Mode == state.ModeBlocked || m.status.Mode == state.ModeEmpty || m.status.Mode == state.ModeError) {
			m.status = deriveStatus(msg.status)
		}
		return m, nil
	case fetchedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed.", msg.err.Error())
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		if m.status.Mode == state.ModeBrowse || m.status.Mode == state.ModeBlocked || m.status.Mode == state.ModeEmpty || m.status.Mode == state.ModeError {
			m.status = deriveStatus(msg.status)
		}
		telemetry.Log("app", "fetch_repo", map[string]string{
			"branch": msg.status.Branch,
			"head":   msg.status.Head,
		})
		return m, nil
	case preparedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed.", msg.err.Error())
			telemetry.Log("app", "prepare_failed", map[string]string{"action": string(msg.action), "error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		switch msg.action {
		case state.ActionMerge, state.ActionRebase, state.ActionReset:
			m.status = actionPickTargets(msg.status, msg.action)
		default:
			m.status = deriveStatus(msg.status)
		}
		telemetry.Log("app", "prepare_action", map[string]string{
			"action": string(msg.action),
			"branch": msg.status.Branch,
		})
		return m, nil
	case pullCheckedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed.", msg.err.Error())
			telemetry.Log("app", "pull_check_failed", map[string]string{"error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.repo
		syncBrowseState(&m, msg.repo)
		m.status = msg.status
		telemetry.Log("app", "pull_check", map[string]string{
			"upstream": msg.repo.Upstream,
			"blocked":  string(msg.status.Block),
		})
		return m, nil
	case previewMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Preview failed.", msg.err.Error())
			telemetry.Log("app", "preview_failed", map[string]string{"action": string(msg.action), "target": msg.target, "error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.repo
		syncBrowseState(&m, msg.repo)
		m.status = msg.status
		m.status.Selected = msg.target
		telemetry.Log("app", "preview_action", map[string]string{
			"action": string(msg.action),
			"target": msg.target,
			"mode":   string(msg.status.Mode),
		})
		return m, nil
	case pushFetchedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed before push.", msg.err.Error())
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		if msg.status.NoUpstream {
			branchName := msg.status.Branch
			titleMsg := "Push and Track Remote?"
			detailMsg := fmt.Sprintf("There is no upstream configured for the current branch. Do you want to push and set upstream tracking to origin/%s?", branchName)
			m.status = m.status.WithConfirm(state.ActionSetUpstream, titleMsg, detailMsg)
			m.status.Title = titleMsg
			return m, nil
		}
		m.status = state.New().WithLoading("Pushing commits...")
		return m, executePush(m.repo, msg.status.Branch, m.commitLimit)
	case pullFetchedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed before pull.", msg.err.Error())
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		track := m.repoStatus.Tracking[m.repoStatus.Branch]
		isFF := track.Behind > 0 && track.Ahead == 0
		m.status = state.New().WithLoading("Analyzing pull changes...")
		return m, loadPullPreviewCommits(m.repo, isFF)
	case pullPreviewReadyMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Analysis failed.", msg.err.Error())
			return m, nil
		}
		m.handshakeCommits = make(map[string]bool)
		if msg.isFF {
			if len(msg.commits) > 0 {
				m.handshakeCommits[msg.commits[0]] = true
			}
		} else {
			for _, hash := range msg.commits {
				m.handshakeCommits[hash] = true
			}
		}
		m.pullIsFastForward = msg.isFF
		var titleMsg, detailMsg string
		if msg.isFF {
			titleMsg = "Do you want to continue?"
			detailMsg = "The branch will fast-forward to the highlighted target commit."
			m.status = m.status.WithConfirm(state.ActionPull, titleMsg, detailMsg)
		} else {
			titleMsg = "Choose Pull Integration"
			detailMsg = "The branches have diverged. Choose integration strategy:\n\nm: merge pull (recreates merge commit)\nr: rebase pull (replays commits)\nesc: cancel integration"
			m.status = m.status.WithConfirm(state.ActionPull, titleMsg, detailMsg)
		}
		m.status.Title = titleMsg
		return m, nil
	case executedMsg:
		if msg.err != nil {
			isAuthError := strings.Contains(msg.err.Error(), "Permission denied") ||
				strings.Contains(msg.err.Error(), "Authentication failed") ||
				strings.Contains(msg.err.Error(), "Could not read from remote repository")

			if msg.action == state.ActionPush && !isAuthError && (strings.Contains(msg.err.Error(), "[rejected]") || strings.Contains(msg.err.Error(), "non-fast-forward")) {
				status := m.repoStatus
				if msg.status.Root != "" {
					status = msg.status
				}
				m.repoStatus = status
				m.handshakeCommits = make(map[string]bool)
				if status.Head != "" {
					m.handshakeCommits[status.Head] = true
				}
				remoteHash := findRemoteCommitHash(status, status.Upstream)
				if remoteHash != "" {
					m.handshakeCommits[remoteHash] = true
				}
				branchName := status.Branch
				titleMsg := fmt.Sprintf("Force push to origin/%s?", branchName)
				detailMsg := fmt.Sprintf("The remote branch has different history. Force pushing will overwrite origin/%s history with your local commits. Continue?", branchName)
				m.status = m.status.WithConfirm(state.ActionForcePush, titleMsg, detailMsg)
				m.status.Title = titleMsg
				return m, nil
			}
			if (msg.action == state.ActionPull || msg.action == state.ActionPullMerge || msg.action == state.ActionPullRebase) && (msg.status.MergeInProgress || msg.status.RebaseInProgress) {
				m.repoStatus = msg.status
				syncBrowseState(&m, msg.status)
				m.status = state.New().WithBrowse()
				m.status.Message = "Pull stopped with conflicts."
				m.status.Detail = "Press enter to abort the in-progress merge/rebase."
				telemetry.Log("app", "execute_conflicted", map[string]string{
					"action": string(msg.action),
					"head":   msg.status.Head,
				})
				return m, nil
			}
			if msg.action == state.ActionMerge && msg.status.MergeInProgress {
				m.repoStatus = msg.status
				syncBrowseState(&m, msg.status)
				m.status = state.New().WithBrowse()
				m.status.Message = "Merge stopped with conflicts."
				m.status.Detail = "Resolve conflicts, then press 'a' to abort or commit to complete."
				telemetry.Log("app", "execute_conflicted", map[string]string{
					"action": string(msg.action),
					"head":   msg.status.Head,
				})
				return m, nil
			}
			if msg.action == state.ActionRebase && msg.status.RebaseInProgress {
				m.repoStatus = msg.status
				syncBrowseState(&m, msg.status)
				m.status = state.New().WithBrowse()
				m.status.Message = "Rebase stopped with conflicts."
				m.status.Detail = "Resolve conflicts, then press 'a' to abort or continue rebase."
				telemetry.Log("app", "execute_conflicted", map[string]string{
					"action": string(msg.action),
					"head":   msg.status.Head,
				})
				return m, nil
			}
			reason := state.BlockUnknown
			message := "Execution failed."
			detail := msg.err.Error()
			if msg.action == state.ActionCheckout {
				message = "Checkout failed."
				if strings.Contains(detail, "local changes") || strings.Contains(detail, "overwritten by checkout") {
					reason = state.BlockDirtyTree
					message = "Checkout blocked by local changes."
					detail = "Your local changes would be overwritten by checkout. Commit or stash them before switching."
				}
			} else if isAuthError && (msg.action == state.ActionPush || msg.action == state.ActionForcePush || msg.action == state.ActionSetUpstream) {
				message = "Authentication or Permission error."
				detail = "Please check your remote credentials or network connection: " + msg.err.Error()
			} else if msg.action == state.ActionPush || msg.action == state.ActionForcePush || msg.action == state.ActionSetUpstream {
				message = "Push failed."
			}
			m.status = state.New().WithBlocked(reason, message, detail)
			telemetry.Log("app", "execute_failed", map[string]string{"action": string(msg.action), "target": msg.target, "error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.status
		if msg.action == state.ActionPush || msg.action == state.ActionForcePush || msg.action == state.ActionSetUpstream || msg.action == state.ActionPullMerge || msg.action == state.ActionPullRebase {
			m.handshakeCommits = make(map[string]bool)
			syncBrowseState(&m, msg.status)
			m.status = deriveStatus(msg.status)
			if msg.action == state.ActionPullMerge || msg.action == state.ActionPullRebase {
				m.status.Message = "Pull completed successfully."
			} else {
				m.status.Message = fmt.Sprintf("Push completed for %s.", msg.target)
			}
			telemetry.Log("app", "execute_action", map[string]string{
				"action": string(msg.action),
				"head":   msg.status.Head,
			})
			return m, nil
		}
		if msg.action == state.ActionCheckout {
			m.commitLimit = initialGraphCommitLimit
			rows := graph.Rows(msg.status)
			if len(rows) > 0 {
				m.sectionCursor[sectionGraph] = 0
				m.graphScroll = 0
				m.graphLaneCursor = graph.PointerLane(rows[0])
			}
			syncBrowseState(&m, msg.status)
			m.status = deriveStatus(msg.status)
			telemetry.Log("app", "execute_action", map[string]string{
				"action": string(msg.action),
				"target": msg.target,
				"head":   msg.status.Head,
			})
			return m, nil
		}
		if msg.action == state.ActionPull {
			syncBrowseState(&m, msg.status)
			m.status = deriveStatus(msg.status)
			telemetry.Log("app", "execute_action", map[string]string{
				"action": string(msg.action),
				"head":   msg.status.Head,
			})
			return m, nil
		}
		if msg.action == state.ActionAbort {
			m.handshakeCommits = make(map[string]bool)
			syncBrowseState(&m, msg.status)
			m.status = deriveStatus(msg.status)
			telemetry.Log("app", "execute_action", map[string]string{
				"action": string(msg.action),
				"head":   msg.status.Head,
			})
			return m, nil
		}
		if msg.action == state.ActionReset {
			rows := graph.Rows(msg.status)
			rowIdx := graph.FindRowByHash(rows, msg.status.Head)
			if rowIdx >= 0 {
				m.sectionCursor[sectionGraph] = rowIdx
				m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
			}
			syncBrowseState(&m, msg.status)
			m.status = deriveStatus(msg.status)
			m.status.Message = fmt.Sprintf("Hard reset completed to %s.", shorten(msg.target, 7))
			telemetry.Log("app", "execute_action", map[string]string{
				"action": string(msg.action),
				"target": msg.target,
				"head":   msg.status.Head,
			})
			return m, nil
		}
		if msg.action == state.ActionMerge || msg.action == state.ActionRebase {
			rows := graph.Rows(msg.status)
			rowIdx := graph.FindRowByHash(rows, msg.status.Head)
			if rowIdx >= 0 {
				m.sectionCursor[sectionGraph] = rowIdx
				m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
			}
		}
		syncBrowseState(&m, msg.status)
		m.status = state.New().WithOutcome(msg.action, "Completed.", executionDetail(msg.action, msg.target, msg.status), false)
		m.status.Selected = msg.target
		telemetry.Log("app", "execute_action", map[string]string{
			"action": string(msg.action),
			"target": msg.target,
			"head":   msg.status.Head,
		})
		return m, nil
	case createdBranchMsg:
		if msg.err != nil {
			m.branchOpen = false
			reason := state.BlockUnknown
			message := "Branch creation failed."
			detail := msg.err.Error()
			if strings.Contains(msg.err.Error(), "working tree is not clean") {
				reason = state.BlockDirtyTree
				message = "Working tree is not clean."
				detail = "Commit or stash local changes before creating and checking out a new branch."
			}
			m.status = state.New().WithBlocked(reason, message, detail)
			telemetry.Log("app", "branch_create_failed", map[string]string{"name": msg.name, "base": msg.base, "error": msg.err.Error()})
			return m, nil
		}
		m.branchOpen = false
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		m.status = deriveStatus(msg.status)
		telemetry.Log("app", "branch_create", map[string]string{"name": msg.name, "base": msg.base})
		return m, nil
	case tea.KeyMsg:
		if m.branchOpen {
			switch msg.String() {
			case "esc":
				m.branchOpen = false
				m.branchDraft = ""
				m.status = deriveStatus(m.repoStatus)
				return m, nil
			case "enter":
				name := strings.TrimSpace(m.branchDraft)
				base := m.branchBase
				m.branchOpen = false
				m.branchDraft = ""
				m.status = state.New().WithLoading("Creating branch...")
				return m, createBranch(m.repo, name, base, m.commitLimit)
			case "backspace":
				if len(m.branchDraft) > 0 {
					runes := []rune(m.branchDraft)
					m.branchDraft = string(runes[:len(runes)-1])
				}
				return m, nil
			default:
				if len(msg.Runes) > 0 {
					m.branchDraft += string(msg.Runes)
					return m, nil
				}
			}
		}
		if m.status.Mode == state.ModeConfirm {
			switch msg.String() {
			case "y", "enter":
				action := m.status.Action
				m.handshakeCommits = make(map[string]bool)
				if action == state.ActionPull {
					if m.pullIsFastForward {
						m.status = state.New().WithLoading("Running pull...")
						return m, executePull(m.repo, m.commitLimit)
					} else {
						m.status = state.New().WithLoading("Running merge pull...")
						return m, executePullMerge(m.repo, m.commitLimit)
					}
				} else if action == state.ActionSetUpstream {
					m.status = state.New().WithLoading("Pushing new branch and tracking upstream...")
					return m, executePushSetUpstream(m.repo, m.repoStatus.Branch, m.commitLimit)
				} else if action == state.ActionForcePush {
					m.status = state.New().WithLoading("Running force push...")
					return m, executeForcePush(m.repo, m.repoStatus.Branch, m.commitLimit)
				} else if action == state.ActionReset {
					target := m.status.Selected
					m.status = state.New().WithLoading("Running hard reset...")
					return m, executeAction(m.repo, action, target, m.commitLimit)
				} else if action == state.ActionMerge {
					target := m.status.Selected
					m.status = state.New().WithLoading("Running merge...")
					return m, executeAction(m.repo, action, target, m.commitLimit)
				} else if action == state.ActionRebase {
					target := m.status.Selected
					m.status = state.New().WithLoading("Running rebase...")
					return m, executeAction(m.repo, action, target, m.commitLimit)
				}
				m.status = deriveStatus(m.repoStatus)
				return m, nil
			case "m":
				action := m.status.Action
				if action == state.ActionPull && !m.pullIsFastForward {
					m.handshakeCommits = make(map[string]bool)
					m.status = state.New().WithLoading("Running merge pull...")
					return m, executePullMerge(m.repo, m.commitLimit)
				}
				return m, nil
			case "r":
				action := m.status.Action
				if action == state.ActionPull && !m.pullIsFastForward {
					m.handshakeCommits = make(map[string]bool)
					m.status = state.New().WithLoading("Running rebase pull...")
					return m, executePullRebase(m.repo, m.commitLimit)
				}
				return m, nil
			case "n", "esc":
				m.handshakeCommits = make(map[string]bool)
				m.status = deriveStatus(m.repoStatus)
				return m, nil
			default:
				return m, nil
			}
		}
		if m.awaitingGoTop && msg.String() != "g" {
			m.awaitingGoTop = false
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "1":
			if m.status.Mode == state.ModeBrowse {
				m = switchBrowseSection(m, sectionCurrent)
			}
		case "2":
			if m.status.Mode == state.ModeBrowse {
				m = switchBrowseSection(m, sectionRemote)
			}
		case "3":
			if m.status.Mode == state.ModeBrowse {
				m = switchBrowseSection(m, sectionTags)
			}
		case "4":
			if m.status.Mode == state.ModeBrowse {
				m = switchBrowseSection(m, sectionGraph)
			}
		case "f":
			if m.status.Mode == state.ModeBrowse {
				m.status.Message = "Fetching remotes..."
				m.status.Detail = "Refreshing remote refs and branch tracking in the background."
				return m, fetchRepoState(m.repo, m.commitLimit)
			}
		case "P":
			if m.status.Mode == state.ModeBrowse {
				if m.repoStatus.Root == "" || m.repoStatus.Detached || m.repoStatus.EmptyRepo {
					return m, nil
				}
				m.status = state.New().WithLoading("Fetching before push...")
				return m, executeFetchForPush(m.repo, m.commitLimit)
			}
		case "p":
			if pullReady(m.repoStatus) {
				m.status = state.New().WithLoading("Fetching upstream before pull...")
				return m, executeFetchForPull(m.repo, m.commitLimit)
			}
			m.status = actionPull(m.repoStatus)
		case "m":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				if !isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
					m.status = state.New().WithBlocked(state.BlockUnknown,
						"Merge not available.",
						"Move the lane cursor onto a local branch to enable merge.")
					return m, nil
				}
				focus := graph.CurrentFocus(m.repoStatus, m.sectionCursor[sectionGraph])
				if focus.Hash == "" || focus.Hash == "VIRTUAL_CONFLICT_HASH" {
					return m, nil
				}
				titleMsg := "Merge into current branch?"
				detailMsg := fmt.Sprintf("This will merge commit %s into %s.\nA merge commit will be created if histories have diverged.\n\nContinue? (y: yes  •  n: no)",
					shorten(focus.Hash, 7), m.repoStatus.Branch)
				m.status = m.status.WithConfirm(state.ActionMerge, titleMsg, detailMsg)
				m.status.Title = titleMsg
				m.status.Selected = focus.Hash
				return m, nil
			}
		case "r":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				if !isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
					m.status = state.New().WithBlocked(state.BlockUnknown,
						"Rebase not available.",
						"Move the lane cursor onto a local branch to enable rebase.")
					return m, nil
				}
				focus := graph.CurrentFocus(m.repoStatus, m.sectionCursor[sectionGraph])
				if focus.Hash == "" || focus.Hash == "VIRTUAL_CONFLICT_HASH" {
					return m, nil
				}
				titleMsg := "Rebase onto this commit?"
				detailMsg := fmt.Sprintf("This will rebase %s onto commit %s.\nLocal commits will be replayed on top of the target.\n\n⚠️ Conflicts may occur during rebase.\n\nContinue? (y: yes  •  n: no)",
					m.repoStatus.Branch, shorten(focus.Hash, 7))
				m.status = m.status.WithConfirm(state.ActionRebase, titleMsg, detailMsg)
				m.status.Title = titleMsg
				m.status.Selected = focus.Hash
				return m, nil
			}
		case "s":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				focus := graph.CurrentFocus(m.repoStatus, m.sectionCursor[sectionGraph])
				if focus.Hash == "" {
					m.status = state.New().WithBlocked(state.BlockUnknown, "No reset target.", "Move the pointer onto a commit line.")
					return m, nil
				}
				titleMsg := "Hard reset to commit?"
				detailMsg := fmt.Sprintf("This will reset your HEAD, index, and working tree. Any uncommitted changes will be lost. Target commit: %s. Continue?", focus.Hash)
				if m.repoStatus.WorktreeDirty {
					detailMsg = fmt.Sprintf("⚠️ WARNING: You have uncommitted changes in your working tree! Hard reset will permanently OVERWRITE and LOSE all uncommitted changes. Target commit: %s. Continue?", focus.Hash)
				}
				m.status = m.status.WithConfirm(state.ActionReset, titleMsg, detailMsg)
				m.status.Title = titleMsg
				m.status.Selected = focus.Hash
				return m, nil
			}
		case "a":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionCurrent && (m.repoStatus.MergeInProgress || m.repoStatus.RebaseInProgress) {
				m.status = state.New().WithLoading("Aborting merge/rebase...")
				return m, executeAbort(m.repo, m.commitLimit)
			}
		case "esc":
			switch {
			case m.status.Mode == state.ModeOutcomePreview && m.status.Action != state.ActionPull && m.status.Action != state.ActionAbort:
				m.status = actionPickTargets(m.repoStatus, m.status.Action)
			case m.status.Mode == state.ModeOutcomePreview && (m.status.Action == state.ActionPull || m.status.Action == state.ActionAbort):
				m.status = deriveStatus(m.repoStatus)
			default:
				m.status = deriveStatus(m.repoStatus)
			}
		case "tab":
			if m.status.Mode == state.ModeBrowse {
				m.activeSection = nextGraphSection(m.activeSection)
			}
		case "shift+tab":
			if m.status.Mode == state.ModeBrowse {
				m.activeSection = prevGraphSection(m.activeSection)
			}
		case "up", "k":
			if m.status.Mode == state.ModeTargetPick {
				m.status = moveTarget(m.status, -1)
			} else if m.status.Mode == state.ModeBrowse {
				m = moveBrowseCursor(m, -1)
			}
		case "down", "j":
			if m.status.Mode == state.ModeTargetPick {
				m.status = moveTarget(m.status, 1)
			} else if m.status.Mode == state.ModeBrowse {
				m = moveBrowseCursor(m, 1)
				return maybeLoadMoreGraph(m)
			}
		case "left", "h":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				m = moveGraphLane(m, -1)
			}
		case "right", "l":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				m = moveGraphLane(m, 1)
			}
		case "g":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				if m.awaitingGoTop {
					m.sectionCursor[sectionGraph] = 0
					m.graphScroll = 0
					rows := graph.Rows(m.repoStatus)
					if len(rows) > 0 {
						m.graphLaneCursor = graph.PointerLane(rows[0])
					}
					m.awaitingGoTop = false
					return m, nil
				}
				m.awaitingGoTop = true
			}
		case "G":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				rows := graph.Rows(m.repoStatus)
				if len(rows) > 0 {
					last := len(rows) - 1
					m.sectionCursor[sectionGraph] = last
					m.graphScroll = clampScroll(last, len(rows), graphPageSize(&m))
					m.graphLaneCursor = graph.PointerLane(rows[last])
				}
				m.awaitingGoTop = false
				return maybeLoadMoreGraph(m)
			}
		case "H":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				rows := graph.Rows(m.repoStatus)
				rowIdx := graph.FindRowByHash(rows, m.repoStatus.Head)
				if rowIdx >= 0 {
					m.sectionCursor[sectionGraph] = rowIdx
					m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
					m.graphLaneCursor = graph.PointerLane(rows[rowIdx])
				}
				m.awaitingGoTop = false
			}
		case "ctrl+u":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				m = pageBrowseGraph(m, -1)
			}
		case "ctrl+d":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				m = pageBrowseGraph(m, 1)
				return maybeLoadMoreGraph(m)
			}
		case "space", " ":
			if m.status.Mode == state.ModeTargetPick {
				action := m.status.Action
				target := selectedTarget(m.status)
				if target == "" {
					m.status = state.New().WithBlocked(state.BlockTargetEmpty, "No target selected.", "Choose a branch, tag, or ref first.")
					return m, nil
				}
				m.status = state.New().WithLoading("Previewing result...")
				return m, previewSelection(m.repo, m.repoStatus, action, target)
			}
			if m.status.Mode == state.ModeBrowse {
				if m.activeSection == sectionCurrent || m.activeSection == sectionRemote {
					if target := activeSectionTarget(m); target != "" {
						m.status = state.New().WithLoading("Checking out " + target + "...")
						return m, executeCheckout(m.repo, target, initialGraphCommitLimit)
					}
					m.status = state.New().WithBlocked(state.BlockUnknown, "No checkout target.", "Move the pointer onto a local or remote branch.")
					return m, nil
				}
				if m.activeSection == sectionGraph {
					return m, nil
				}
				m.status = state.New().WithBlocked(state.BlockUnknown, "Checkout unavailable in this section.", "Use the Local or Remote sections to switch branches.")
			}
			if m.status.Mode == state.ModeOutcomePreview && m.status.CanExecute {
				action := m.status.Action
				target := m.status.Selected
				m.status = state.New().WithLoading("Running action...")
				switch action {
				case state.ActionPull:
					return m, executePull(m.repo, m.commitLimit)
				case state.ActionAbort:
					return m, executeAbort(m.repo, m.commitLimit)
				case state.ActionMerge, state.ActionRebase, state.ActionReset:
					return m, executeAction(m.repo, action, target, m.commitLimit)
				}
			}
		case "n":
			if m.status.Mode == state.ModeBrowse && (m.activeSection == sectionCurrent || m.activeSection == sectionGraph) {
				if !canCreateBranch(m.repoStatus) {
					m.status = state.New().WithBlocked(
						state.BlockDirtyTree,
						"Working tree is not clean.",
						"Commit or stash local changes before creating and checking out a new branch.",
					)
					return m, nil
				}
				base := activeSectionTarget(m)
				if base == "" {
					focus := graph.CurrentFocus(m.repoStatus, m.sectionCursor[sectionGraph])
					base = focus.Hash
				}
				m.branchBase = base
				m.branchOpen = true
				m.branchDraft = ""
				m.status = state.New().WithLoading("Type a new branch name and press enter.")
			}
		}
	}
	return m, nil
}

func deriveStatus(rs git.Status) state.Status {
	switch {
	case rs.Root == "":
		return state.New().WithBlocked(state.BlockNoRepo, "Not inside a Git repository.", "Run this tool from a repo root.")
	case rs.MergeInProgress || rs.RebaseInProgress:
		status := state.New().WithBrowse()
		status.Message = "Merge/Rebase in progress after conflict."
		status.Detail = "Press enter to abort the in-progress merge/rebase."
		return status
	case rs.Detached:
		return state.New().WithBlocked(state.BlockDetached, "Detached HEAD.", "Pick a branch before running pull, merge, or rebase.")
	case rs.EmptyRepo:
		return state.New().WithEmpty("Repository has no commits yet.")
	case rs.NoRemote && rs.NoUpstream:
		return state.New().WithBlocked(state.BlockNoRemote, "No remote or upstream configured.", "Pull, merge, and rebase need a branch with a remote target.")
	default:
		return state.New().WithBrowse()
	}
}

func actionPull(rs git.Status) state.Status {
	if rs.Root == "" {
		return state.New().WithBlocked(state.BlockNoRepo, "Not inside a Git repository.", "Run this tool from a repo root.")
	}
	if rs.Detached {
		return state.New().WithBlocked(state.BlockDetached, "Detached HEAD.", "Pull needs a branch with an upstream.")
	}
	if rs.MergeInProgress || rs.RebaseInProgress {
		return state.New().WithBlocked(state.BlockUnknown, "A merge/rebase is already in progress.", "Abort or resolve the existing merge/rebase before pulling again.")
	}
	if rs.NoRemote {
		return state.New().WithBlocked(state.BlockNoRemote, "No remote configured.", "Pull needs origin or another remote.")
	}
	if rs.NoUpstream {
		return state.New().WithBlocked(state.BlockNoUpstream, "No upstream configured.", "Set an upstream before pulling.")
	}
	return state.New().WithOutcome(state.ActionPull, "Pull is ready.", "Pull will fetch and merge upstream changes into the current branch.", true)
}

func pullReady(rs git.Status) bool {
	return rs.Root != "" && !rs.Detached && !rs.NoRemote && !rs.NoUpstream && !rs.MergeInProgress
}

func canCreateBranch(rs git.Status) bool {
	return !rs.WorktreeDirty
}

func actionPickTargets(rs git.Status, action state.Action) state.Status {
	if (action == state.ActionMerge || action == state.ActionRebase) && rs.Detached {
		return state.New().WithBlocked(state.BlockDetached, "Detached HEAD.", "Choose a branch before merging or rebasing.")
	}
	targets := make([]state.TargetItem, 0, len(rs.LocalBranches)+len(rs.RemoteBranches)+len(rs.Tags))
	for _, name := range rs.LocalBranches {
		upstream, known := branchUpstream(rs, name)
		targets = append(targets, state.TargetItem{
			Kind:       state.TargetKindLocal,
			Name:       name,
			Ref:        name,
			NoUpstream: known && upstream == "",
		})
	}
	for _, name := range rs.RemoteBranches {
		if strings.HasSuffix(name, "/HEAD") {
			continue
		}
		targets = append(targets, state.TargetItem{Kind: state.TargetKindRemote, Name: name, Ref: name})
	}
	for _, name := range rs.Tags {
		targets = append(targets, state.TargetItem{Kind: state.TargetKindTag, Name: name, Ref: name})
	}
	if len(targets) == 0 {
		for _, name := range rs.Branches {
			targets = append(targets, state.TargetItem{Kind: state.TargetKindLocal, Name: name, Ref: name})
		}
	}
	if len(targets) == 0 {
		return state.New().WithBlocked(state.BlockTargetEmpty, "No branch targets available.", "Create or fetch a branch before merging, rebasing, or resetting.")
	}
	status := state.New().WithTargetPick(action, targets)
	status.Message = "Use up/down to choose a target."
	status.Detail = "Enter previews the result. Esc returns to browse."
	return status
}

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
		items := make([]state.TargetItem, 0, 1+len(rs.LocalBranches))
		if rs.Branch != "" {
			track := rs.Tracking[rs.Branch]
			upstream, known := branchUpstream(rs, rs.Branch)
			items = append(items, state.TargetItem{
				Kind:            state.TargetKindLocal,
				Name:            rs.Branch,
				Ref:             rs.Branch,
				Current:         true,
				NeedsPull:       track.Behind > 0 && track.Ahead == 0,
				NeedsPush:       track.Ahead > 0,
				NoUpstream:      known && upstream == "",
				MergeConflicted: rs.MergeInProgress,
			})
		} else if rs.Head != "" {
			items = append(items, state.TargetItem{Kind: state.TargetKindLocal, Name: "HEAD", Ref: rs.Head, Current: true, MergeConflicted: rs.MergeInProgress})
		}
		for _, name := range rs.LocalBranches {
			if name == rs.Branch {
				continue
			}
			track := rs.Tracking[name]
			upstream, known := branchUpstream(rs, name)
			items = append(items, state.TargetItem{
				Kind:       state.TargetKindLocal,
				Name:       name,
				Ref:        name,
				NeedsPull:  track.Behind > 0 && track.Ahead == 0,
				NeedsPush:  track.Ahead > 0,
				NoUpstream: known && upstream == "",
			})
		}
		return items
	case sectionRemote:
		items := make([]state.TargetItem, 0, len(rs.RemoteBranches))
		for _, name := range rs.RemoteBranches {
			if !strings.Contains(name, "/") {
				continue
			}
			items = append(items, state.TargetItem{
				Kind:    state.TargetKindRemote,
				Name:    name,
				Ref:     name,
				Default: strings.HasSuffix(name, "/HEAD") || name == "origin/"+rs.DefaultBranch,
			})
		}
		return items
	case sectionTags:
		items := make([]state.TargetItem, 0, len(rs.Tags))
		for _, name := range rs.Tags {
			items = append(items, state.TargetItem{Kind: state.TargetKindTag, Name: name, Ref: name})
		}
		return items
	default:
		return nil
	}
}

func branchUpstream(rs git.Status, name string) (string, bool) {
	if name == "" {
		return "", false
	}
	if rs.BranchUpstreams != nil {
		if upstream, ok := rs.BranchUpstreams[name]; ok {
			return upstream, true
		}
	}
	if name == rs.Branch && rs.Branch != "HEAD" {
		return rs.Upstream, true
	}
	return "", false
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

const (
	// graphViewHeightOffset은 그래프 렌더링 시 레이아웃 테두리(2줄), 페이지 정보 표시(1줄),
	// 컬럼 헤더(1줄), 기본 패딩 등을 고려하여 제외해야 하는 세로 높이 총합입니다.
	graphViewHeightOffset = 5
)

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

// isLocalGraphPointer returns true when the current graph cursor is pointing
// at a local-branch lane. Merge and Rebase from Graph are only allowed in this state.
func isLocalGraphPointer(rs git.Status, cursor int, laneCursor int) bool {
	rows := graph.Rows(rs)
	if cursor < 0 || cursor >= len(rows) {
		return false
	}
	row := rows[cursor]
	if row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
		return false
	}

	// Build local branch set for fast lookup
	localSet := make(map[string]struct{}, len(rs.LocalBranches))
	for _, b := range rs.LocalBranches {
		localSet[b] = struct{}{}
	}

	// raw-graph mode: check decorations to see if this commit has any local branch
	// This matches the same logic used in compactDecorationInfo to render "l->name"
	if row.Graph != "" {
		for _, dec := range row.Commit.Decorations {
			dec = strings.TrimSpace(dec)
			// HEAD -> <branch> is always local
			if strings.HasPrefix(dec, "HEAD -> ") {
				return true
			}
			// A decoration that is in localBranches (not origin/*, not tag:*)
			if dec == "" || strings.HasPrefix(dec, "origin/") || strings.HasPrefix(dec, "tag: ") {
				continue
			}
			if !strings.Contains(dec, "/") {
				// bare name — if it's in local branches, it's a local lane
				if _, ok := localSet[dec]; ok {
					return true
				}
				// even if not in the set, an unqualified name is treated local (new branch edge case)
				return true
			}
		}
		return false
	}

	// legacy lane mode: check laneRef at the lane cursor position
	if laneCursor >= 0 && laneCursor < len(row.Before) {
		return row.Before[laneCursor].Side == laneLocal
	}
	if laneCursor >= 0 && laneCursor < len(row.After) {
		return row.After[laneCursor].Side == laneLocal
	}
	// fallback: treat single-lane commits on the pointer lane as local
	return row.Lane == laneCursor
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

func checkoutTargetFromFocus(node graphNode) string {
	for _, decoration := range node.Decorations {
		decoration = strings.TrimSpace(decoration)
		if strings.HasPrefix(decoration, "HEAD -> ") {
			return strings.TrimPrefix(decoration, "HEAD -> ")
		}
		if strings.HasPrefix(decoration, "tag: ") {
			continue
		}
		if strings.Contains(decoration, "/") {
			return decoration
		}
		if decoration != "" {
			return decoration
		}
	}
	return ""
}

func executeCheckout(repo *git.Repo, target string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Run("switch", target)
		if err != nil && strings.Contains(target, "/") {
			localName := target[strings.Index(target, "/")+1:]
			_, err = repo.Run("switch", "--track", "-c", localName, target)
		}
		if err != nil {
			return executedMsg{action: state.ActionCheckout, target: target, err: err}
		}
		status, statusErr := repo.Status(context.Background(), limit)
		return executedMsg{action: state.ActionCheckout, target: target, status: status, err: statusErr}
	}
}

func selectedTarget(s state.Status) string {
	if s.Selected != "" {
		return s.Selected
	}
	if s.TargetIdx >= 0 && s.TargetIdx < len(s.Targets) {
		return s.Targets[s.TargetIdx].Ref
	}
	return ""
}

func buildActionPreview(action state.Action, target string, rs git.Status, currentOnly, targetOnly int) state.Status {
	head := shorten(rs.Head, 12)
	switch action {
	case state.ActionMerge:
		switch {
		case currentOnly == 0 && targetOnly == 0:
			return state.New().WithOutcome(state.ActionMerge, "Target already matches HEAD.", "Nothing moves. The branch already points at the same commit.", true)
		case currentOnly == 0:
			return state.New().WithOutcome(state.ActionMerge, "FF 가능. 포인터만 이동합니다.", "HEAD can move to "+target+". Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly), true)
		case targetOnly == 0:
			return state.New().WithOutcome(state.ActionMerge, "대상은 이미 포함되어 있습니다.", "Current branch already contains "+target+". Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly), true)
		default:
			return state.New().WithOutcome(state.ActionMerge, "FF 불가. merge commit이 필요합니다.", "HEAD "+head+" and target "+target+" have diverged. Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly), true)
		}
	case state.ActionRebase:
		switch {
		case currentOnly == 0 && targetOnly == 0:
			return state.New().WithOutcome(state.ActionRebase, "Target already matches HEAD.", "Nothing is rewritten because both refs point at the same commit.", true)
		case targetOnly == 0:
			return state.New().WithOutcome(state.ActionRebase, "Target is already in the base history.", "Current commits will replay onto "+target+". Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly), true)
		default:
			return state.New().WithOutcome(state.ActionRebase, "새 base 위로 커밋을 재배치합니다.", "Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly)+"  |  target: "+target, true)
		}
	case state.ActionReset:
		return state.New().WithOutcome(state.ActionReset, "현재 HEAD를 선택한 위치로 이동합니다.", "HEAD "+head+" -> "+target+"  |  Current-only: "+fmt.Sprint(currentOnly)+"  Target-only: "+fmt.Sprint(targetOnly), true)
	default:
		return state.New().WithOutcome(action, "No action selected.", target, false)
	}
}

func executionDetail(action state.Action, target string, rs git.Status) string {
	switch action {
	case state.ActionPull:
		return "Upstream pointer is now reflected in the local branch."
	case state.ActionMerge:
		return "Merge complete. HEAD now reflects " + emptyDash(rs.Branch) + " with target " + target + "."
	case state.ActionRebase:
		return "Rebase complete. The branch was replayed on top of " + target + "."
	case state.ActionReset:
		return "Hard reset complete. HEAD now points at " + target + "."
	default:
		return "Action complete."
	}
}

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

func executeFetchForPush(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		err := repo.Fetch(context.Background())
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return pushFetchedMsg{err: statusErr}
		}
		return pushFetchedMsg{status: status, err: err}
	}
}

type pullPreviewReadyMsg struct {
	commits []string
	isFF    bool
	err     error
}

func executeFetchForPull(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		err := repo.Fetch(context.Background())
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return pullFetchedMsg{err: statusErr}
		}
		return pullFetchedMsg{status: status, err: err}
	}
}

func loadPullPreviewCommits(repo *git.Repo, isFF bool) tea.Cmd {
	return func() tea.Msg {
		var arg string
		if isFF {
			arg = "HEAD..@{upstream}"
		} else {
			arg = "HEAD...@{upstream}"
		}
		out, err := repo.Run("rev-list", arg)
		if err != nil {
			return pullPreviewReadyMsg{err: err, isFF: isFF}
		}
		lines := strings.Split(out, "\n")
		commits := make([]string, 0, len(lines))
		for _, line := range lines {
			hash := strings.TrimSpace(line)
			if hash != "" {
				commits = append(commits, hash)
			}
		}
		if isFF {
			headOut, headErr := repo.Run("rev-parse", "HEAD")
			if headErr == nil && strings.TrimSpace(headOut) != "" {
				commits = append(commits, strings.TrimSpace(headOut))
			}
		}
		return pullPreviewReadyMsg{commits: commits, isFF: isFF}
	}
}

func findRemoteCommitHash(rs git.Status, upstream string) string {
	if upstream == "" {
		return ""
	}
	target := upstream
	if strings.HasPrefix(target, "refs/remotes/") {
		target = strings.TrimPrefix(target, "refs/remotes/")
	}
	for _, commit := range rs.GraphCommits {
		for _, dec := range commit.Decorations {
			decTrim := strings.TrimSpace(dec)
			if decTrim == target || "origin/"+decTrim == target || decTrim == "origin/"+target {
				return commit.Hash
			}
		}
	}
	return ""
}
