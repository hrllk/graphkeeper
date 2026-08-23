package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	projection, ok := buildConfirmationProjection(m)
	if !ok {
		return m, nil
	}
	result := classifyConfirmationKey(projection, msg.String())
	switch result.Decision {
	case decisionAccept:
		return m.handleConfirmAccept()
	case decisionChoice:
		switch result.ChoiceKey {
		case choiceMerge:
			return m.handleConfirmPullMerge()
		case choiceRebase:
			return m.handleConfirmPullRebase()
		}
	case decisionCancel:
		clearPullConfirmProjection(&m)
		if m.pullCancel != nil {
			m.pullCancel()
			m.pullCancel = nil
		}
		m.handshakeCommits = make(map[string]bool)
		m.activePullRequest = nil
		m.pullConfirmStale = false
		m.nextPullRequestID++
		m.status = deriveStatus(m.repoStatus)
	}
	return m, nil
}

func (m model) handleConfirmAccept() (tea.Model, tea.Cmd) {
	action := m.status.Action
	m.handshakeCommits = make(map[string]bool)
	switch action {
	case state.ActionPull:
		if m.activePullRequest == nil || m.pullConfirmStale {
			return m, nil
		}
		clearPullConfirmProjection(&m)
		if m.pullIsFastForward {
			m.status = operationLoadingStatusFor(progressPull, "Pulling...", state.ActionPull)
			if m.pull != nil {
				return m, startPullWorkflow(&m, PullModeMerge)
			}
			return m, validateAndExecutePull(m.repo, m.commitLimit, *m.activePullRequest, PullModeMerge)
		}
		m.status = operationLoadingStatusFor(progressPull, "Pulling...", state.ActionPull)
		if m.pull != nil {
			return m, startPullWorkflow(&m, PullModeMerge)
		}
		return m, validateAndExecutePull(m.repo, m.commitLimit, *m.activePullRequest, PullModeMerge)
	case state.ActionSetUpstream:
		m.status = operationLoadingStatusFor(progressPush, "Pushing and tracking...", state.ActionSetUpstream)
		return m, executePushSetUpstream(m.repo, m.repoStatus.Branch, m.commitLimit)
	case state.ActionForcePush:
		m.status = operationLoadingStatusFor(progressPush, "Force pushing...", state.ActionForcePush)
		return m, executeForcePush(m.repo, m.repoStatus.Branch, m.commitLimit)
	case state.ActionDeleteBranch:
		target := m.status.Selected
		remote := m.status.DeleteRemote
		m.status = operationLoadingStatusFor(progressDeleteBranch, deleteBranchLoadingMessage(remote), state.ActionDeleteBranch)
		return m, executeDeleteBranch(m.repo, target, remote, m.commitLimit)
	case state.ActionDeleteTag:
		target := m.status.Selected
		m.status = operationLoadingStatusFor(progressDeleteTag, deleteTagLoadingMessage(), state.ActionDeleteTag)
		return m, executeDeleteTag(m.repo, target, m.commitLimit, m.tagProvenance)
	case state.ActionDeleteRemoteTag:
		target := m.status.Selected
		m.status = operationLoadingStatusFor(progressDeleteRemoteTag, deleteRemoteTagLoadingMessage(), state.ActionDeleteRemoteTag)
		return m, executeDeleteRemoteTag(m.repo, target, m.commitLimit, m.tagProvenance)
	case state.ActionStash:
		m.status = operationLoadingStatusFor(progressStash, "Stashing changes...", state.ActionStash)
		return m, executeStashAll(m.repo, m.commitLimit, "graphkeeper: local cleanup")
	case state.ActionCleanWorkingTree:
		m.status = operationLoadingStatusFor(progressClean, "Cleaning working tree...", state.ActionCleanWorkingTree)
		return m, executeCleanWorkingTree(m.repo, m.commitLimit, false)
	case state.ActionCheckout:
		target := m.status.Selected
		if target == "" {
			m.status = deriveStatus(m.repoStatus)
			return m, nil
		}
		m.status = operationLoadingStatusFor(progressCheckout, "Checking out...", state.ActionCheckout)
		return m, executeCheckout(m.repo, target, m.commitLimit)
	case state.ActionReset, state.ActionMerge, state.ActionRebase:
		target := m.status.Selected
		if action == state.ActionReset {
			mode := m.status.ResetMode
			if mode == "" {
				mode = state.ResetModeHard
			}
			m.status = operationLoadingStatusFor(progressReset, strings.Title(string(mode))+" reset...", state.ActionReset)
			return m, executeReset(m.repo, target, mode, m.commitLimit)
		} else if action == state.ActionMerge {
			m.status = operationLoadingStatusFor(progressMerge, "Merging...", state.ActionMerge)
		} else {
			m.status = operationLoadingStatusFor(progressRebase, "Rebasing...", state.ActionRebase)
		}
		return m, executeAction(m.repo, action, target, m.commitLimit)
	default:
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	}
}

func (m model) handleConfirmPullMerge() (tea.Model, tea.Cmd) {
	if m.status.Action == state.ActionPull && !m.pullIsFastForward {
		if m.activePullRequest == nil || m.pullConfirmStale {
			return m, nil
		}
		clearPullConfirmProjection(&m)
		m.handshakeCommits = make(map[string]bool)
		m.status = operationLoadingStatusFor(progressMerge, "Merging pull...", state.ActionMerge)
		if m.pull != nil {
			return m, startPullWorkflow(&m, PullModeMerge)
		}
		return m, validateAndExecutePull(m.repo, m.commitLimit, *m.activePullRequest, PullModeMerge)
	}
	return m, nil
}

func (m model) handleConfirmPullRebase() (tea.Model, tea.Cmd) {
	if m.status.Action == state.ActionPull && !m.pullIsFastForward {
		if m.activePullRequest == nil || m.pullConfirmStale {
			return m, nil
		}
		clearPullConfirmProjection(&m)
		m.handshakeCommits = make(map[string]bool)
		m.status = operationLoadingStatusFor(progressRebase, "Rebasing pull...", state.ActionRebase)
		if m.pull != nil {
			return m, startPullWorkflow(&m, PullModeRebase)
		}
		return m, validateAndExecutePull(m.repo, m.commitLimit, *m.activePullRequest, PullModeRebase)
	}
	return m, nil
}
