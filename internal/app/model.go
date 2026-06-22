package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/git-graph-tui/internal/git"
	"hrllk/git-graph-tui/internal/state"
	"hrllk/git-graph-tui/internal/telemetry"
)

type model struct {
	repo            *git.Repo
	status          state.Status
	repoStatus      git.Status
	activeSection   graphSection
	sectionCursor   map[graphSection]int
	graphLaneCursor int
	graphScroll     int
	awaitingGoTop   bool
	branchOpen      bool
	branchDraft     string
	branchBase      string
	width           int
	height          int
	err             error
}

type graphSection int

const (
	sectionGraph graphSection = iota
	sectionCurrent
	sectionLocal
	sectionRemote
	sectionTags
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
		graphLaneCursor: 0,
	}
	return m, nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadRepoState(m.repo), scheduleRefresh())
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

type graphNode struct {
	Hash        string
	Parents     []string
	Decorations []string
}

func loadRepoState(repo *git.Repo) tea.Cmd {
	return func() tea.Msg {
		status, err := repo.Status(context.Background())
		return loadedMsg{status: status, err: err}
	}
}

func scheduleRefresh() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func refreshRepoState(repo *git.Repo) tea.Cmd {
	return func() tea.Msg {
		status, err := repo.Status(context.Background())
		return refreshedMsg{status: status, err: err}
	}
}

func fetchRepoState(repo *git.Repo) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Fetch(context.Background()); err != nil {
			return fetchedMsg{err: err}
		}
		status, err := repo.Status(context.Background())
		return fetchedMsg{status: status, err: err}
	}
}

func prepareAction(repo *git.Repo, action state.Action) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Fetch(context.Background()); err != nil {
			return preparedMsg{action: action, err: err}
		}
		status, err := repo.Status(context.Background())
		return preparedMsg{action: action, status: status, err: err}
	}
}

func pullCheck(repo *git.Repo) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Fetch(context.Background()); err != nil {
			return pullCheckedMsg{err: err}
		}
		status, err := repo.Status(context.Background())
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

func executePull(repo *git.Repo) tea.Cmd {
	return func() tea.Msg {
		if _, err := repo.Run("pull", "--ff-only"); err != nil {
			return executedMsg{action: state.ActionPull, err: err}
		}
		status, err := repo.Status(context.Background())
		return executedMsg{action: state.ActionPull, status: status, err: err}
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

func executeAction(repo *git.Repo, action state.Action, target string) tea.Cmd {
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
			_, err = repo.Run("reset", "--soft", target)
		default:
			err = fmt.Errorf("unsupported action %q", action)
		}
		if err != nil {
			return executedMsg{action: action, target: target, err: err}
		}
		status, statusErr := repo.Status(context.Background())
		return executedMsg{action: action, target: target, status: status, err: statusErr}
	}
}

func createBranch(repo *git.Repo, name, base string) tea.Cmd {
	return func() tea.Msg {
		if name == "" {
			return createdBranchMsg{err: fmt.Errorf("branch name is empty")}
		}
		status, err := repo.Status(context.Background())
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
		status, err = repo.Status(context.Background())
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
		return m, tea.Batch(scheduleRefresh(), refreshRepoState(m.repo))
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
		m.status = state.New().WithOutcome(state.ActionPull, "Fetch completed.", "Remote pointers were updated from origin.", false)
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
	case executedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Execution failed.", msg.err.Error())
			telemetry.Log("app", "execute_failed", map[string]string{"action": string(msg.action), "target": msg.target, "error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.status
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
				return m, createBranch(m.repo, name, base)
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
		if m.awaitingGoTop && msg.String() != "g" {
			m.awaitingGoTop = false
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r":
			m.status = m.status.WithLoading("Refreshing repository state...")
			return m, loadRepoState(m.repo)
		case "f":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				m.status = state.New().WithLoading("Fetching remotes...")
				return m, fetchRepoState(m.repo)
			}
		case "p":
			if pullReady(m.repoStatus) {
				m.status = state.New().WithLoading("Fetching upstream before pull...")
				return m, pullCheck(m.repo)
			}
			m.status = actionPull(m.repoStatus)
		case "m":
			m.status = state.New().WithLoading("Fetching branches before merge...")
			return m, prepareAction(m.repo, state.ActionMerge)
		case "e":
			m.status = state.New().WithLoading("Fetching branches before rebase...")
			return m, prepareAction(m.repo, state.ActionRebase)
		case "s":
			if m.status.Mode == state.ModeBrowse {
				focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
				if focus.Hash == "" {
					m.status = state.New().WithBlocked(state.BlockUnknown, "No reset target.", "Move the pointer onto a commit line.")
					return m, nil
				}
				m.status = state.New().WithOutcome(state.ActionReset, "Reset preview from the current graph focus.", "Target: "+focus.Hash+"  |  Use enter to reset to this commit.", true)
				m.status.Selected = focus.Hash
				return m, nil
			}
			m.status = state.New().WithLoading("Fetching branches before reset...")
			return m, prepareAction(m.repo, state.ActionReset)
		case "esc":
			switch {
			case m.status.Mode == state.ModeOutcomePreview && m.status.Action != state.ActionPull:
				m.status = actionPickTargets(m.repoStatus, m.status.Action)
			case m.status.Mode == state.ModeOutcomePreview && m.status.Action == state.ActionPull:
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
					rows := graphRows(m.repoStatus)
					if len(rows) > 0 {
						m.graphLaneCursor = graphPointerLane(rows[0])
					}
					m.awaitingGoTop = false
					return m, nil
				}
				m.awaitingGoTop = true
			}
		case "G":
			if m.status.Mode == state.ModeBrowse && m.activeSection == sectionGraph {
				rows := graphRows(m.repoStatus)
				if len(rows) > 0 {
					last := len(rows) - 1
					m.sectionCursor[sectionGraph] = last
					m.graphScroll = clampScroll(last, len(rows), graphPageSize(&m))
					m.graphLaneCursor = graphPointerLane(rows[last])
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
			}
		case "enter":
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
				if m.activeSection == sectionGraph {
					focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
					if target := checkoutTargetFromFocus(focus); target != "" {
						m.status = state.New().WithLoading("Checking out " + target + "...")
						return m, executeCheckout(m.repo, target)
					}
					m.status = state.New().WithBlocked(state.BlockUnknown, "No checkout target.", "Move the pointer onto a branch decoration.")
					return m, nil
				}
				if target := activeSectionTarget(m); target != "" {
					m.status = state.New().WithLoading("Checking out " + target + "...")
					return m, executeCheckout(m.repo, target)
				}
				m.status = state.New().WithBlocked(state.BlockUnknown, "No checkout target.", "Move the pointer onto a local or remote branch.")
			}
			if m.status.Mode == state.ModeOutcomePreview && m.status.CanExecute {
				action := m.status.Action
				target := m.status.Selected
				m.status = state.New().WithLoading("Running action...")
				switch action {
				case state.ActionPull:
					return m, executePull(m.repo)
				case state.ActionMerge, state.ActionRebase, state.ActionReset:
					return m, executeAction(m.repo, action, target)
				}
			}
		case "c":
			if m.status.Mode == state.ModeBrowse {
				if m.activeSection == sectionGraph {
					focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
					if target := checkoutTargetFromFocus(focus); target != "" {
						m.status = state.New().WithLoading("Checking out " + target + "...")
						return m, executeCheckout(m.repo, target)
					}
					m.status = state.New().WithBlocked(state.BlockUnknown, "No checkout target.", "Move the pointer onto a branch decoration.")
					return m, nil
				}
				if target := activeSectionTarget(m); target != "" {
					m.status = state.New().WithLoading("Checking out " + target + "...")
					return m, executeCheckout(m.repo, target)
				}
				m.status = state.New().WithBlocked(state.BlockUnknown, "No checkout target.", "Move the pointer onto a local or remote branch.")
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
					focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
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
	if rs.NoRemote {
		return state.New().WithBlocked(state.BlockNoRemote, "No remote configured.", "Pull needs origin or another remote.")
	}
	if rs.NoUpstream {
		return state.New().WithBlocked(state.BlockNoUpstream, "No upstream configured.", "Set an upstream before pulling.")
	}
	return state.New().WithOutcome(state.ActionPull, "Fetch first, then check whether the upstream can fast-forward.", "If the branch diverged, show the remote pointer and stop.", false)
}

func pullReady(rs git.Status) bool {
	return rs.Root != "" && !rs.Detached && !rs.NoRemote && !rs.NoUpstream
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
		targets = append(targets, state.TargetItem{Kind: state.TargetKindLocal, Name: name, Ref: name})
	}
	for _, name := range rs.RemoteBranches {
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
	items := make([]graphNode, 0, len(rs.GraphCommits))
	for _, commit := range rs.GraphCommits {
		items = append(items, graphNode{
			Hash:        commit.Hash,
			Parents:     append([]string(nil), commit.Parents...),
			Decorations: append([]string(nil), commit.Decorations...),
		})
	}
	return items
}

type graphRow struct {
	Commit       graphNode
	Children     []string
	Before       []string
	After        []string
	Lane         int
	DisplayWidth int
	Collapse     bool
}

func graphRows(rs git.Status) []graphRow {
	commits := graphNodes(rs)
	rows := make([]graphRow, 0, len(commits))
	children := buildChildrenMap(commits)
	preferred := firstParentSet(commits, rs.Head)
	active := make([]string, 0, 8)
	for _, commit := range commits {
		lane := lastIndexOf(active, commit.Hash)
		if lane < 0 {
			switch {
			case len(active) == 0:
				active = append(active, commit.Hash)
				lane = 0
			case preferred[commit.Hash]:
				active = append([]string{commit.Hash}, active...)
				lane = 0
			default:
				active = append(active, commit.Hash)
				lane = len(active) - 1
			}
		}
		before := append([]string(nil), active...)
		after := advanceGraphLanes(before, lane, commit.Hash, commit.Parents, children)
		childRefs := append([]string(nil), children[commit.Hash]...)
		row := graphRow{
			Commit:       commit,
			Children:     childRefs,
			Before:       before,
			After:        after,
			Lane:         lane,
			DisplayWidth: max(max(max(len(before), len(after)), len(childRefs)), 1),
		}
		rows = append(rows, row)
		active = after
	}
	return rows
}

func buildChildrenMap(commits []graphNode) map[string][]string {
	children := make(map[string][]string)
	for _, commit := range commits {
		for _, parent := range commit.Parents {
			if parent == "" {
				continue
			}
			children[parent] = append(children[parent], commit.Hash)
		}
	}
	return children
}

func firstParentSet(commits []graphNode, head string) map[string]bool {
	if head == "" {
		return nil
	}
	byHash := make(map[string]graphNode, len(commits))
	for _, commit := range commits {
		byHash[commit.Hash] = commit
	}
	preferred := make(map[string]bool)
	current := head
	for current != "" {
		if preferred[current] {
			break
		}
		preferred[current] = true
		commit, ok := byHash[current]
		if !ok || len(commit.Parents) == 0 {
			break
		}
		current = commit.Parents[0]
	}
	return preferred
}

func graphRowWidth(row graphRow) int {
	if len(row.Before) > 1 {
		same := true
		for _, hash := range row.Before {
			if hash != row.Commit.Hash {
				same = false
				break
			}
		}
		if same {
			return 1
		}
	}
	if row.DisplayWidth > 0 {
		return row.DisplayWidth
	}
	width := len(row.Before)
	if len(row.After) > width {
		width = len(row.After)
	}
	if width == 0 {
		width = 1
	}
	return width
}

func indexOf(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

func lastIndexOf(values []string, target string) int {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] == target {
			return i
		}
	}
	return -1
}

func advanceGraphLanes(active []string, lane int, current string, parents []string, children map[string][]string) []string {
	if lane < 0 {
		lane = 0
	}
	if lane > len(active) {
		lane = len(active)
	}
	if allSameHash(active, current) {
		next := make([]string, 0, len(parents))
		for _, parent := range parents {
			if parent == "" {
				continue
			}
			next = append(next, parent)
		}
		return next
	}
	next := make([]string, 0, len(active)+len(parents))
	next = append(next, active[:lane]...)
	if len(parents) > 0 && parents[0] != "" {
		next = append(next, parents[0])
	}
	if len(parents) > 1 {
		for _, parent := range parents[1:] {
			if parent == "" {
				continue
			}
			next = append(next, parent)
		}
	}
	if lane+1 < len(active) {
		for _, hash := range active[lane+1:] {
			if hash == current {
				continue
			}
			next = append(next, hash)
		}
	}
	return next
}

func allSameHash(values []string, target string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if value != target {
			return false
		}
	}
	return true
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
	if rows := graphRows(m.repoStatus); m.sectionCursor[sectionGraph] >= 0 && m.sectionCursor[sectionGraph] < len(rows) {
		currentHash = rows[m.sectionCursor[sectionGraph]].Commit.Hash
	}
	rowCount := len(graphRows(rs))
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
		rows := graphRows(rs)
		row := findGraphRowByHash(rows, currentHash)
		if row < 0 {
			row = clampCursor(m.sectionCursor[sectionGraph], len(rows))
		}
		m.sectionCursor[sectionGraph] = row
		m.graphLaneCursor = graphPointerLane(rows[row])
	}
}

func findGraphRowByHash(rows []graphRow, hash string) int {
	if hash == "" {
		return -1
	}
	for i, row := range rows {
		if row.Commit.Hash == hash {
			return i
		}
	}
	return -1
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

func sectionTargets(rs git.Status, section graphSection) []state.TargetItem {
	switch section {
	case sectionCurrent:
		items := make([]state.TargetItem, 0, 1+len(rs.LocalBranches))
		if rs.Branch != "" {
			items = append(items, state.TargetItem{Kind: state.TargetKindLocal, Name: rs.Branch, Ref: rs.Branch, Current: true})
		} else if rs.Head != "" {
			items = append(items, state.TargetItem{Kind: state.TargetKindLocal, Name: "HEAD", Ref: rs.Head, Current: true})
		}
		for _, name := range rs.LocalBranches {
			if name == rs.Branch {
				continue
			}
			items = append(items, state.TargetItem{Kind: state.TargetKindLocal, Name: name, Ref: name})
		}
		return items
	case sectionRemote:
		items := make([]state.TargetItem, 0, len(rs.RemoteBranches))
		for _, name := range rs.RemoteBranches {
			branchName := name
			if strings.HasPrefix(branchName, "origin/") {
				branchName = strings.TrimPrefix(branchName, "origin/")
			}
			items = append(items, state.TargetItem{
				Kind:    state.TargetKindRemote,
				Name:    name,
				Ref:     name,
				Default: branchName != "" && branchName == rs.DefaultBranch,
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
		total := len(graphRows(m.repoStatus))
		cursor := moveGraphPointer(m.sectionCursor[sectionGraph], total, delta)
		m.sectionCursor[sectionGraph] = cursor
		page := graphPageSize(&m)
		if cursor < m.graphScroll {
			m.graphScroll = cursor
		} else if cursor >= m.graphScroll+page {
			m.graphScroll = cursor - page + 1
		}
		rows := graphRows(m.repoStatus)
		if cursor >= 0 && cursor < len(rows) {
			m.graphLaneCursor = graphPointerLane(rows[cursor])
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
	rows := graphRows(m.repoStatus)
	if len(rows) == 0 {
		return m
	}
	row := clampCursor(m.sectionCursor[sectionGraph], len(rows))
	m.graphLaneCursor = moveLanePointer(m.graphLaneCursor, rows[row], delta)
	return m
}

func pageBrowseGraph(m model, pages int) model {
	total := len(graphRows(m.repoStatus))
	if total == 0 {
		return m
	}
	page := graphPageSize(&m)
	delta := page * pages
	cursor := moveGraphPointer(m.sectionCursor[sectionGraph], total, delta)
	m.sectionCursor[sectionGraph] = cursor
	m.graphScroll = clampScroll(cursor, total, page)
	rows := graphRows(m.repoStatus)
	if cursor >= 0 && cursor < len(rows) {
		m.graphLaneCursor = graphPointerLane(rows[cursor])
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

func graphPageSize(m *model) int {
	if m.height <= 0 {
		return 12
	}
	size := m.height/2 - 8
	if size < 5 {
		size = 5
	}
	return size
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
	maxLane := graphRowWidth(row) - 1
	if maxLane < 0 {
		return 0
	}
	if current < 0 {
		current = graphPointerLane(row)
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
	maxLane := graphRowWidth(row) - 1
	if maxLane < 0 {
		return 0
	}
	if current < 0 || current > maxLane {
		return min(row.Lane, maxLane)
	}
	return current
}

func graphPointerLane(row graphRow) int {
	maxLane := graphRowWidth(row) - 1
	if maxLane < 0 {
		return 0
	}
	if len(row.Before) > 1 {
		same := true
		for _, hash := range row.Before {
			if hash != row.Commit.Hash {
				same = false
				break
			}
		}
		if same {
			return 0
		}
	}
	if row.Lane < 0 {
		return 0
	}
	if row.Lane > maxLane {
		return maxLane
	}
	return row.Lane
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

func currentGraphFocus(rs git.Status, cursor int) graphNode {
	items := graphRows(rs)
	if cursor < 0 || cursor >= len(items) {
		return graphNode{}
	}
	return items[cursor].Commit
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

func executeCheckout(repo *git.Repo, target string) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Run("switch", target)
		if err != nil && strings.Contains(target, "/") {
			localName := target[strings.Index(target, "/")+1:]
			_, err = repo.Run("switch", "--track", "-c", localName, target)
		}
		if err != nil {
			return executedMsg{action: state.ActionCheckout, target: target, err: err}
		}
		status, statusErr := repo.Status(context.Background())
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
		return "Reset complete. HEAD now points at " + target + "."
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
