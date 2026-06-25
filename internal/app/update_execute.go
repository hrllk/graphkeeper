package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
	"hrllk/graphkeeper/internal/telemetry"
)

func handleExecutedUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	msg2, ok := msg.(executedMsg)
	if !ok {
		return m, nil
	}
	if msg2.err != nil {
		isAuthError := strings.Contains(msg2.err.Error(), "Permission denied") ||
			strings.Contains(msg2.err.Error(), "Authentication failed") ||
			strings.Contains(msg2.err.Error(), "Could not read from remote repository")

		if msg2.action == state.ActionPush && !isAuthError && (strings.Contains(msg2.err.Error(), "[rejected]") || strings.Contains(msg2.err.Error(), "non-fast-forward")) {
			status := m.repoStatus
			if msg2.status.Root != "" {
				status = msg2.status
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
		if (msg2.action == state.ActionPull || msg2.action == state.ActionPullMerge || msg2.action == state.ActionPullRebase) && (msg2.status.MergeInProgress || msg2.status.RebaseInProgress) {
			m.repoStatus = msg2.status
			syncBrowseState(&m, msg2.status)
			m.status = state.New().WithBrowse()
			m.status.Message = "Pull conflicted."
			m.status.Detail = "Press Enter to abort."
			telemetry.Log("app", "execute_conflicted", map[string]string{
				"action": string(msg2.action),
				"head":   msg2.status.Head,
			})
			return m, nil
		}
		if msg2.action == state.ActionMerge && msg2.status.MergeInProgress {
			m.repoStatus = msg2.status
			syncBrowseState(&m, msg2.status)
			m.status = state.New().WithBrowse()
			m.status.Message = "Merge conflicted."
			m.status.Detail = "Resolve conflicts, then abort or commit."
			telemetry.Log("app", "execute_conflicted", map[string]string{
				"action": string(msg2.action),
				"head":   msg2.status.Head,
			})
			return m, nil
		}
		if msg2.action == state.ActionRebase && msg2.status.RebaseInProgress {
			m.repoStatus = msg2.status
			syncBrowseState(&m, msg2.status)
			m.status = state.New().WithBrowse()
			m.status.Message = "Rebase conflicted."
			m.status.Detail = "Resolve conflicts, then abort or continue."
			telemetry.Log("app", "execute_conflicted", map[string]string{
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
		} else if isAuthError && (msg2.action == state.ActionPush || msg2.action == state.ActionForcePush || msg2.action == state.ActionSetUpstream) {
			message = "Auth or permission error."
			detail = "Check credentials or network: " + msg2.err.Error()
		} else if msg2.action == state.ActionPush || msg2.action == state.ActionForcePush || msg2.action == state.ActionSetUpstream {
			message = "Push failed."
		}
		m.status = state.New().WithBlocked(reason, message, detail)
		telemetry.Log("app", "execute_failed", map[string]string{"action": string(msg2.action), "target": msg2.target, "error": msg2.err.Error()})
		return m, nil
	}
	m.repoStatus = msg2.status
	if msg2.action == state.ActionPush || msg2.action == state.ActionForcePush || msg2.action == state.ActionSetUpstream || msg2.action == state.ActionPullMerge || msg2.action == state.ActionPullRebase {
		m.handshakeCommits = make(map[string]bool)
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		if msg2.action == state.ActionPullMerge || msg2.action == state.ActionPullRebase {
			m.status.Message = "Pull complete."
		} else {
			m.status.Message = fmt.Sprintf("Push complete: %s.", msg2.target)
		}
		telemetry.Log("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionCheckout {
		m.commitLimit = 0
		rows := graph.Rows(msg2.status)
		if len(rows) > 0 {
			m.sectionCursor[sectionGraph] = 0
			m.graphScroll = 0
			m.graphLaneCursor = graph.PointerLane(rows[0])
		}
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		telemetry.Log("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"target": msg2.target,
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionPull {
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		telemetry.Log("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionAbort {
		m.handshakeCommits = make(map[string]bool)
		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		telemetry.Log("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionReset {
		rows := graph.Rows(msg2.status)
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
		telemetry.Log("app", "execute_action", map[string]string{
			"action": string(msg2.action),
			"target": msg2.target,
			"head":   msg2.status.Head,
		})
		return m, nil
	}
	if msg2.action == state.ActionMerge || msg2.action == state.ActionRebase {
		rows := graph.Rows(msg2.status)
		rowIdx := graph.FindRowByHash(rows, msg2.status.Head)
		if rowIdx >= 0 {
			m.sectionCursor[sectionGraph] = rowIdx
			m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
		}
	}
	syncBrowseState(&m, msg2.status)
	m.status = state.New().WithOutcome(msg2.action, "Complete.", executionDetail(msg2.action, msg2.target, msg2.status), false)
	m.status.Selected = msg2.target
	telemetry.Log("app", "execute_action", map[string]string{
		"action": string(msg2.action),
		"target": msg2.target,
		"head":   msg2.status.Head,
	})
	return m, nil
}
