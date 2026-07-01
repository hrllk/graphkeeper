package app

import (
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
	"hrllk/graphkeeper/internal/telemetry"
)

func handleFetchUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
	case graphActionCheckMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Graph action check failed.", msg.err.Error())
			telemetry.Log("app", "graph_action_check_failed", map[string]string{"action": string(msg.action), "target": msg.target, "error": msg.err.Error()})
			return m, nil
		}
		m.repoStatus = msg.repo
		syncBrowseState(&m, msg.repo)
		switch {
		case msg.currentOnly == 0 && msg.targetOnly == 0:
			m.status = state.New().WithBlocked(state.BlockUnknown, "Already aligned.", "Target already matches HEAD.")
		case msg.currentOnly == 0:
			m.status = state.New().WithBlocked(state.BlockUnknown, "Fast-forward available.", "HEAD can move to "+msg.target+". Current: "+strconv.Itoa(msg.currentOnly)+"  Target: "+strconv.Itoa(msg.targetOnly))
		case msg.targetOnly == 0:
			reason := "Target already included."
			detail := "Current branch already contains " + msg.target + ". Current: " + strconv.Itoa(msg.currentOnly) + "  Target: " + strconv.Itoa(msg.targetOnly)
			m.status = state.New().WithBlocked(state.BlockUnknown, reason, detail)
		default:
			titleMsg := "Merge into current branch?"
			if msg.action == state.ActionRebase {
				titleMsg = "Rebase onto this commit?"
			}
			detailMsg := ""
			if msg.action == state.ActionMerge {
				detailMsg = "This will merge commit " + shorten(msg.target, 7) + " into " + msg.repo.Branch + ".\nA merge commit will be created if histories have diverged.\n\nContinue? (y: yes  •  n: no)"
			} else {
				detailMsg = "This will rebase " + msg.repo.Branch + " onto commit " + shorten(msg.target, 7) + ".\nLocal commits will be replayed on top of the target.\n\n⚠️ Conflicts may occur during rebase.\n\nContinue? (y: yes  •  n: no)"
			}
			m.status = m.status.WithConfirm(msg.action, titleMsg, detailMsg)
			m.status.Title = titleMsg
			m.status.Selected = msg.target
		}
		telemetry.Log("app", "graph_action_check", map[string]string{
			"action":      string(msg.action),
			"target":      msg.target,
			"currentOnly": strconv.Itoa(msg.currentOnly),
			"targetOnly":  strconv.Itoa(msg.targetOnly),
		})
		return m, nil
	case pushFetchedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch before push failed.", msg.err.Error())
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		if msg.status.NoUpstream {
			branchName := msg.status.Branch
			titleMsg := "Push and track remote?"
			detailMsg := fmt.Sprintf("Set upstream to origin/%s?", branchName)
			m.status = m.status.WithConfirm(state.ActionSetUpstream, titleMsg, detailMsg)
			m.status.Title = titleMsg
			return m, nil
		}
		m.status = loadingToast("Pushing...")
		return m, executePush(m.repo, msg.status.Branch, m.commitLimit)
	case pullFetchedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch before pull failed.", msg.err.Error())
			return m, nil
		}
		m.repoStatus = msg.status
		syncBrowseState(&m, msg.status)
		track := m.repoStatus.Tracking[m.repoStatus.Branch]
		if track.Behind == 0 {
			m.status = loadingToast("Already up to date.")
			m.status.Detail = "Nothing to pull from upstream."
			return m, tea.Tick(900*time.Millisecond, func(time.Time) tea.Msg {
				return pullToastDoneMsg{}
			})
		}
		isFF := track.Behind > 0 && track.Ahead == 0
		m.status = loadingToast("Analyzing pull...")
		return m, loadPullPreviewCommits(m.repo, isFF)
	case pullPreviewReadyMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Analysis failed.", msg.err.Error())
			return m, nil
		}
		if len(msg.commits) == 0 {
			m.status = loadingToast("Already up to date.")
			m.status.Detail = "Nothing to pull from upstream."
			return m, tea.Tick(900*time.Millisecond, func(time.Time) tea.Msg {
				return pullToastDoneMsg{}
			})
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
			titleMsg = "Fast-forward pull?"
			detailMsg = "Fast-forward to the target commit."
			m.status = m.status.WithConfirm(state.ActionPull, titleMsg, detailMsg)
		} else {
			titleMsg = "Choose pull mode"
			detailMsg = "Branches diverged.\n\nm: merge\nr: rebase\nesc: cancel"
			m.status = m.status.WithConfirm(state.ActionPull, titleMsg, detailMsg)
		}
		m.status.Title = titleMsg
		return m, nil
	case pullToastDoneMsg:
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	case branchToastDoneMsg:
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	default:
		return m, nil
	}
}
