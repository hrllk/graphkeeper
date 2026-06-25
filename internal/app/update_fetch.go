package app

import (
	"fmt"

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
	default:
		return m, nil
	}
}
