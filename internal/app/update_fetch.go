package app

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func handleFetchUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed.", msg.err.Error())
			return m, nil
		}
		msg.status = m.withCachedTagEntries(msg.status)
		m.repoStatus = msg.status
		if msg.status.TagProvenanceLoaded {
			m.tagSyncAttempted = true
		}
		m.storeTagEntries(msg.status)
		syncBrowseState(&m, msg.status)
		if m.status.Mode == state.ModeBrowse || m.status.Mode == state.ModeEmpty || m.status.Mode == state.ModeError || m.status.Mode == state.ModeLoading {
			m.status = deriveStatus(msg.status)
		}
		m.publish("app", "fetch_repo", map[string]string{
			"branch": msg.status.Branch,
			"head":   msg.status.Head,
		})
		return m, nil
	case preparedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed.", msg.err.Error())
			m.publish("app", "prepare_failed", map[string]string{"action": string(msg.action), "error": msg.err.Error()})
			return m, nil
		}
		msg.status = m.withCachedTagEntries(msg.status)
		m.repoStatus = msg.status
		if msg.status.TagProvenanceLoaded {
			m.tagSyncAttempted = true
		}
		m.storeTagEntries(msg.status)
		syncBrowseState(&m, msg.status)
		switch msg.action {
		case state.ActionMerge, state.ActionRebase, state.ActionReset:
			m.status = actionPickTargets(msg.status, msg.action)
		default:
			m.status = deriveStatus(msg.status)
		}
		m.publish("app", "prepare_action", map[string]string{
			"action": string(msg.action),
			"branch": msg.status.Branch,
		})
		return m, nil
	case pullCheckedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch failed.", msg.err.Error())
			m.publish("app", "pull_check_failed", map[string]string{"error": msg.err.Error()})
			return m, nil
		}
		msg.repo = m.withCachedTagEntries(msg.repo)
		m.repoStatus = msg.repo
		if msg.repo.TagProvenanceLoaded {
			m.tagSyncAttempted = true
		}
		m.storeTagEntries(msg.repo)
		syncBrowseState(&m, msg.repo)
		m.status = msg.status
		m.publish("app", "pull_check", map[string]string{
			"upstream": msg.repo.Upstream,
			"blocked":  string(msg.status.Block),
		})
		return m, nil
	case previewMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Preview failed.", msg.err.Error())
			m.publish("app", "preview_failed", map[string]string{"action": string(msg.action), "target": msg.target, "error": msg.err.Error()})
			return m, nil
		}
		msg.repo = m.withCachedTagEntries(msg.repo)
		m.repoStatus = msg.repo
		if msg.repo.TagProvenanceLoaded {
			m.tagSyncAttempted = true
		}
		m.storeTagEntries(msg.repo)
		syncBrowseState(&m, msg.repo)
		m.status = msg.status
		m.status.Selected = msg.target
		m.publish("app", "preview_action", map[string]string{
			"action": string(msg.action),
			"target": msg.target,
			"mode":   string(msg.status.Mode),
		})
		return m, nil
	case graphActionCheckMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Graph action check failed.", msg.err.Error())
			m.publish("app", "graph_action_check_failed", map[string]string{"action": string(msg.action), "target": msg.target, "error": msg.err.Error()})
			return m, nil
		}
		msg.repo = m.withCachedTagEntries(msg.repo)
		m.repoStatus = msg.repo
		if msg.repo.TagProvenanceLoaded {
			m.tagSyncAttempted = true
		}
		m.storeTagEntries(msg.repo)
		syncBrowseState(&m, msg.repo)
		switch {
		case msg.currentOnly == 0 && msg.targetOnly == 0:
			m.status = state.New().WithBlocked(state.BlockUnknown, "Already aligned.", "Target already matches HEAD.")
		case msg.currentOnly == 0:
			m.status = buildGraphActionFastForwardStatus(msg.action, msg.target)
		case msg.targetOnly == 0:
			reason := "Target already included."
			detail := "Current branch already contains " + msg.target + ". Current: " + strconv.Itoa(msg.currentOnly) + "  Target: " + strconv.Itoa(msg.targetOnly)
			m.status = state.New().WithBlocked(state.BlockUnknown, reason, detail)
		default:
			m.status = buildGraphActionReviewStatus(msg.action, msg.repo, msg.target, msg.base, msg.currentOnly, msg.targetOnly)
			m.status.Selected = msg.target
		}
		m.publish("app", "graph_action_check", map[string]string{
			"action":      string(msg.action),
			"target":      msg.target,
			"base":        msg.base,
			"currentOnly": strconv.Itoa(msg.currentOnly),
			"targetOnly":  strconv.Itoa(msg.targetOnly),
		})
		return m, nil
	case pushFetchedMsg:
		if msg.err != nil {
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch before push failed.", msg.err.Error())
			return m, nil
		}
		msg.status = m.withCachedTagEntries(msg.status)
		m.repoStatus = msg.status
		if msg.status.TagProvenanceLoaded {
			m.tagSyncAttempted = true
		}
		m.storeTagEntries(msg.status)
		syncBrowseState(&m, msg.status)
		if msg.status.NoUpstream {
			branchName := msg.status.Branch
			titleMsg := "Push and track remote?"
			detailMsg := fmt.Sprintf("Set upstream to origin/%s?", branchName)
			m.status = m.status.WithConfirm(state.ActionSetUpstream, titleMsg, detailMsg)
			m.status.Title = titleMsg
			return m, nil
		}
		m.status = operationLoadingStatusFor(progressPush, "Pushing...", state.ActionPush)
		return m, executePush(m.repo, msg.status.Branch, m.commitLimit)
	case pullFetchedMsg:
		if !m.pullRequestMessageActive(msg.requestID, msg.requestEpoch) {
			return m, nil
		}
		if msg.err != nil {
			m.activePullRequest = nil
			m.status = state.New().WithBlocked(state.BlockFetchFailed, "Fetch before pull failed.", msg.err.Error())
			return m, nil
		}
		if msg.operationBaseline == (PullSnapshotIdentity{}) || !msg.operationBaselineSet {
			m.activePullRequest = nil
			m.status = state.New().WithBlocked(state.BlockUnknown, "Pull impact unavailable.", "Refresh before pulling again.")
			return m, m.refreshCmd()
		}
		if !samePullSnapshotIdentity(pullSnapshotIdentity(msg.status, msg.requestEpoch), msg.operationBaseline) {
			m.activePullRequest = nil
			m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
			return m, m.refreshCmd()
		}
		msg.status = m.withCachedTagEntries(msg.status)
		m.repoStatus = msg.status
		m.activePullRequest.FetchBaseline = msg.fetchBaseline
		m.activePullRequest.OperationBaseline = msg.operationBaseline
		m.activePullRequest.OperationBaselineSet = msg.operationBaselineSet
		impact := pullImpactSet(msg.snapshot)
		if msg.status.TagProvenanceLoaded {
			m.tagSyncAttempted = true
		}
		m.storeTagEntries(msg.status)
		syncBrowseState(&m, msg.status)
		track, trackingKnown := m.repoStatus.Tracking[m.repoStatus.Branch]
		if !m.repoStatus.TrackingKnown || !m.repoStatus.TrackingFresh || !trackingKnown {
			m.activePullRequest = nil
			m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
			return m, m.refreshCmd()
		}
		if track.Behind == 0 {
			request := *m.activePullRequest
			m.activePullRequest = nil
			return completePullNoOp(m, m.repoStatus, request, m.lastPullMode)
		}
		if !impact.Valid && track.Behind > 0 {
			m.activePullRequest = nil
			m.status = state.New().WithBlocked(state.BlockUnknown, "Pull impact unavailable.", "Refresh before pulling again.")
			return m, m.refreshCmd()
		}
		isFF := msg.snapshot.IsFastForward && msg.snapshot.FastForwardKnown
		m.status = operationLoadingStatusFor(progressPull, "Analyzing pull...", state.ActionPull)
		return m, loadPullPreviewCommits(m.repo, isFF, *m.activePullRequest)
	case pullPreviewReadyMsg:
		if !m.pullRequestMessageActive(msg.requestID, msg.requestEpoch) || !samePullSnapshotIdentity(m.activePullRequest.OperationBaseline, msg.baseline) {
			return m, nil
		}
		if msg.err != nil {
			m.activePullRequest = nil
			m.status = state.New().WithBlocked(state.BlockUnknown, "Analysis failed.", msg.err.Error())
			return m, nil
		}
		if len(msg.commits) == 0 {
			request := *m.activePullRequest
			m.activePullRequest = nil
			return completePullNoOp(m, m.repoStatus, request, m.lastPullMode)
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
			titleMsg = "Pull into " + msg.snapshot.CurrentRef + "?"
			if !applyMergeConfirmProjection(&m, msg.snapshot, msg.impact, m.pullConfirmStale) {
				m.activePullRequest = nil
				m.status = state.New().WithBlocked(state.BlockUnknown, "Pull impact unavailable.", "Refresh before pulling again.")
				return m, m.refreshCmd()
			}
		}
		m.status.Title = titleMsg
		return m, nil
	case pullToastDoneMsg:
		if !m.pullRequestMessageActive(msg.requestID, msg.requestEpoch) {
			return m, nil
		}
		m.activePullRequest = nil
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	case branchToastDoneMsg:
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	default:
		return m, nil
	}
}

func (m model) pullRequestMessageActive(id, epoch uint64) bool {
	return m.activePullRequest != nil && m.activePullRequest.ID == id && m.activePullRequest.Epoch == epoch && !m.pullConfirmStale
}
