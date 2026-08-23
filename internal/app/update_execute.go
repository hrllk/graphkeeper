package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

func handleExecutedUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	msg2, ok := msg.(executedMsg)
	if !ok {
		return m, nil
	}
	if msg2.err != nil {
		isPushRoutingAction := msg2.action == state.ActionPush || msg2.action == state.ActionForcePush || msg2.action == state.ActionSetUpstream
		isLegacyPushTagAction := msg2.action == state.ActionPushTag || msg2.action == state.ActionDeleteRemoteTag
		isAuthError := (isPushRoutingAction && msg2.errorCategory == PermissionDenied) ||
			(isLegacyPushTagAction && (strings.Contains(msg2.err.Error(), "Permission denied") ||
				strings.Contains(msg2.err.Error(), "Authentication failed") ||
				strings.Contains(msg2.err.Error(), "Could not read from remote repository")))

		if msg2.action == state.ActionStash || msg2.action == state.ActionCleanWorkingTree {
			if msg2.status.Root != "" {
				msg2.status = m.withCachedTagEntries(msg2.status)
				m.repoStatus = msg2.status
				m.storeTagEntries(msg2.status)
				syncBrowseState(&m, msg2.status)
			}
			reason := state.BlockUnknown
			message := "Action failed."
			if msg2.action == state.ActionStash {
				message = "Stash failed."
			} else {
				message = "Working tree cleanup failed."
			}
			m.status = state.New().WithBlocked(reason, message, msg2.err.Error())
			m.publish("app", "execute_failed", map[string]string{"action": string(msg2.action), "target": msg2.target, "error": msg2.err.Error()})
			return m, nil
		}

		if msg2.action == state.ActionPush && msg2.errorCategory == NonFastForward && msg2.operationErr != nil && msg2.statusErr == nil {
			status := m.repoStatus
			if msg2.status.Root != "" {
				status = m.withCachedTagEntries(msg2.status)
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
			m.publish("app", "push_force_confirmation", map[string]string{
				"action": "push",
				"target": msg2.target,
				"error":  msg2.operationErr.Error(),
			})
			return m, nil
		}
		if msg2.action == state.ActionDeleteBranch && strings.Contains(msg2.err.Error(), "current branch cannot be deleted") {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Current branch cannot be deleted.", "Select a different local branch.")
			m.publish("app", "execute_failed", map[string]string{"action": string(msg2.action), "target": msg2.target, "error": msg2.err.Error()})
			return m, nil
		}
		if msg2.action == state.ActionDeleteTag && strings.Contains(strings.ToLower(msg2.err.Error()), "not found") {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Tag not found.", "Refresh the tag list and try again.")
			m.publish("app", "execute_failed", map[string]string{"action": string(msg2.action), "target": msg2.target, "error": msg2.err.Error()})
			return m, nil
		}
		if msg2.action == state.ActionDeleteRemoteTag && strings.Contains(strings.ToLower(msg2.err.Error()), "not found") {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Remote tag not found.", "Refresh tag provenance and try again.")
			m.publish("app", "execute_failed", map[string]string{"action": string(msg2.action), "target": msg2.target, "error": msg2.err.Error()})
			return m, nil
		}
		if msg2.action == state.ActionStashPop {
			if msg2.status.Root != "" {
				msg2.status = m.withCachedTagEntries(msg2.status)
				m.repoStatus = msg2.status
				m.storeTagEntries(msg2.status)
				syncBrowseState(&m, msg2.status)
			}
			m.status = state.New().WithBlocked(state.BlockUnknown, "Stash pop failed.", msg2.err.Error())
			m.publish("app", "execute_failed", map[string]string{
				"action": string(msg2.action),
				"target": msg2.target,
				"error":  msg2.err.Error(),
			})
			return m, loadStashState(m.repo)
		}
		if (msg2.action == state.ActionPull || msg2.action == state.ActionPullMerge || msg2.action == state.ActionPullRebase) && (msg2.status.MergeInProgress || msg2.status.RebaseInProgress || msg2.status.CherryPickInProgress) {
			msg2.status = m.withCachedTagEntries(msg2.status)
			m.repoStatus = msg2.status
			m.storeTagEntries(msg2.status)
			syncBrowseState(&m, msg2.status)
			m.status = state.New().WithBrowse()
			m.status.Message = "Pull conflicted."
			m.status.Detail = "Press Enter to abort."
			m.publish("app", "execute_conflicted", map[string]string{
				"action": string(msg2.action),
				"head":   msg2.status.Head,
			})
			return m, nil
		}
		if msg2.action == state.ActionMerge && msg2.status.MergeInProgress {
			msg2.status = m.withCachedTagEntries(msg2.status)
			m.repoStatus = msg2.status
			m.storeTagEntries(msg2.status)
			syncBrowseState(&m, msg2.status)
			m.status = state.New().WithBrowse()
			m.status.Message = "Merge conflicted."
			m.status.Detail = "Resolve conflicts, then abort or commit."
			m.publish("app", "execute_conflicted", map[string]string{
				"action": string(msg2.action),
				"head":   msg2.status.Head,
			})
			return m, nil
		}
		if msg2.action == state.ActionRebase && msg2.status.RebaseInProgress {
			msg2.status = m.withCachedTagEntries(msg2.status)
			m.repoStatus = msg2.status
			m.storeTagEntries(msg2.status)
			syncBrowseState(&m, msg2.status)
			m.status = state.New().WithBrowse()
			m.status.Message = "Rebase conflicted."
			m.status.Detail = "Resolve conflicts, then abort."
			m.publish("app", "execute_conflicted", map[string]string{
				"action": string(msg2.action),
				"head":   msg2.status.Head,
			})
			return m, nil
		}
		if msg2.action == state.ActionCherryPick && msg2.status.CherryPickInProgress {
			msg2.status = m.withCachedTagEntries(msg2.status)
			m.repoStatus = msg2.status
			m.storeTagEntries(msg2.status)
			syncBrowseState(&m, msg2.status)
			m.status = state.New().WithBrowse()
			m.status.Message = "Cherry-pick conflicted."
			m.status.Detail = "Resolve conflicts, then abort."
			m.publish("app", "execute_conflicted", map[string]string{
				"action": string(msg2.action),
				"head":   msg2.status.Head,
			})
			return m, nil
		}
		reason := state.BlockUnknown
		message := "Action failed."
		detail := msg2.err.Error()
		if msg2.action == state.ActionCheckout {
			message = "Checkout failed."
			if strings.Contains(detail, "local changes") || strings.Contains(detail, "overwritten by checkout") {
				reason = state.BlockDirtyTree
				message = "Checkout blocked by local changes."
				detail = "Commit or stash changes first."
			}
		} else if isAuthError && (msg2.action == state.ActionPush || msg2.action == state.ActionPushTag || msg2.action == state.ActionDeleteRemoteTag || msg2.action == state.ActionForcePush || msg2.action == state.ActionSetUpstream) {
			message = "Auth or permission error."
			detail = "Check credentials or network: " + msg2.err.Error()
		} else if msg2.action == state.ActionPush || msg2.action == state.ActionPushTag || msg2.action == state.ActionDeleteRemoteTag || msg2.action == state.ActionForcePush || msg2.action == state.ActionSetUpstream {
			message = "Push failed."
		} else if msg2.action == state.ActionCherryPick {
			message = "Cherry-pick failed."
			if strings.Contains(detail, "empty commit") {
				message = "Cherry-pick skipped an empty commit."
				detail = "Adjust the queue and retry."
			}
		} else if msg2.action == state.ActionDeleteBranch {
			message = "Branch delete failed."
			if strings.Contains(detail, "branch not found") {
				message = "Branch not found."
				detail = "Refresh the branch list and try again."
			}
		}
		m.status = state.New().WithBlocked(reason, message, detail)
		m.publish("app", "execute_failed", map[string]string{"action": string(msg2.action), "target": msg2.target, "error": msg2.err.Error()})
		return m, nil
	}
	if msg2.action == state.ActionCheckout {
		msg2.status = applyRepositoryStatus(&m, msg2.status)
	} else {
		if msg2.action != state.ActionDeleteTag {
			msg2.status = m.withCachedTagEntries(msg2.status)
		}
		m.repoStatus = msg2.status
		if msg2.status.TagProvenanceLoaded {
			m.tagSyncAttempted = true
		}
		m.storeTagEntries(msg2.status)
	}
	if msg2.action == state.ActionStash {
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		m.status.Message = "Changes stashed."
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"head":   msg2.status.Head,
		})
		return m, loadStashState(m.repo)
	}
	if msg2.action == state.ActionCleanWorkingTree {
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		m.status.Message = "Working tree cleaned."
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionCherryPick {
		rows := graphRows(msg2.status)
		rowIdx := graph.FindRowByHash(rows, msg2.status.Head)
		if rowIdx >= 0 {
			m.sectionCursor[sectionGraph] = rowIdx
			m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
		}
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		m.status.Message = "Cherry-pick complete."
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"target": msg2.target,
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionPush || msg2.action == state.ActionPushTag || msg2.action == state.ActionForcePush || msg2.action == state.ActionSetUpstream || msg2.action == state.ActionPullMerge || msg2.action == state.ActionPullRebase {
		m.handshakeCommits = make(map[string]bool)
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		if msg2.action == state.ActionPullMerge || msg2.action == state.ActionPullRebase {
			m.status.Message = "Pull complete."
		} else if msg2.action == state.ActionPushTag {
			m.status.Message = fmt.Sprintf("Tag pushed: %s.", msg2.target)
		} else {
			m.status.Message = fmt.Sprintf("Push complete: %s.", msg2.target)
		}
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionDeleteBranch {
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		if strings.HasPrefix(msg2.target, "origin/") {
			m.status.Message = "Origin branch deleted."
		} else {
			m.status.Message = "Branch deleted."
		}
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"target": msg2.target,
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionDeleteTag {
		syncBrowseState(&m, msg2.status)
		m.replaceTagEntries(msg2.status)
		m.status = deriveStatus(msg2.status)
		m.status.Message = "Tag deleted."
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"target": msg2.target,
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionDeleteRemoteTag {
		syncBrowseState(&m, msg2.status)
		m.replaceTagEntries(msg2.status)
		m.status = deriveStatus(msg2.status)
		m.status.Message = "Remote tag deleted."
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"target": msg2.target,
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionStashPop {
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		m.status.Message = "Stash popped."
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"target": msg2.target,
			"head":   msg2.status.Head,
		})
		return m, loadStashState(m.repo)
	}
	if msg2.action == state.ActionCheckout {
		m.commitLimit = 0
		focusGraphHead(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"target": msg2.target,
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionPull {
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionAbort {
		m.handshakeCommits = make(map[string]bool)
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionReset {
		rows := graphRows(msg2.status)
		rowIdx := graph.FindRowByHash(rows, msg2.status.Head)
		if rowIdx >= 0 {
			m.sectionCursor[sectionGraph] = rowIdx
			m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
		}
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		mode := msg2.resetMode
		if mode == "" {
			mode = state.ResetModeHard
		}
		m.status.Message = fmt.Sprintf("%s reset complete: %s.", strings.Title(string(mode)), shorten(msg2.target, 7))
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"target": msg2.target,
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionMerge || msg2.action == state.ActionRebase {
		rows := graphRows(msg2.status)
		rowIdx := graph.FindRowByHash(rows, msg2.status.Head)
		if rowIdx >= 0 {
			m.sectionCursor[sectionGraph] = rowIdx
			m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
		}
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		m.status.Message = strings.TrimSuffix(executionDetail(msg2.action, msg2.target, msg2.status), ".")
		m.publish("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"target": msg2.target,
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	syncBrowseState(&m, msg2.status)
	m.status = state.New().WithOutcome(msg2.action, "Complete.", executionDetail(msg2.action, msg2.target, msg2.status), false)
	m.status.Selected = msg2.target
	m.publish("app", "execute_action", map[string]string{
		"action": string(msg2.action),
		"target": msg2.target,
		"head":   msg2.status.Head,
	})
	return m, nil
}
