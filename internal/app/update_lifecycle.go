package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func handleWindowSize(m model, msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	return m, nil
}

func markPullStale(m model) (model, tea.Cmd) {
	if m.activePullRequest == nil || m.status.Action != state.ActionPull {
		return m, nil
	}
	if m.status.Mode == state.ModeLoading {
		if m.pullCancel != nil {
			m.pullCancel()
			m.pullCancel = nil
		}
		clearPullConfirmProjection(&m)
		m.activePullRequest = nil
		m.pullConfirmStale = false
		m.nextPullRequestID++
		m.status = state.New().WithBlocked(state.BlockStaleSnapshot, "Repository changed.", "Refresh before pulling again.")
		return m, m.refreshCmd()
	}
	if m.status.Mode == state.ModeReview || m.status.Mode == state.ModeConfirm {
		m.pullConfirmStale = true
	}
	return m, nil
}

func closeInspectorWithRecoveryAlert(m model, kind string) (model, tea.Cmd) {
	message := "Commit Inspector closed: selected file is no longer available."
	if kind == "commit_not_found" {
		message = "Commit Inspector closed: selected commit is no longer available."
	}
	m = m.closeCommitInspector()
	m.status = state.New().WithBlocked(state.BlockUnknown, message, "Press Enter to dismiss.")
	return m, nil
}

func handleLifecycleUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedSnapshotMsg:
		if msg.result.RepositoryEpoch != m.repositoryEpoch {
			return m, nil
		}
		if msg.result.ErrorKind != ReadErrorNone {
			m.err = startupReadError(msg.result)
			m.status = startupErrorStatus(msg.result)
			m.startupReadPending = false
			m.startupFailed = true
			return m, nil
		}
		m.graphReadSnapshot = msg.result.Snapshot.Graph
		m.repoStatus = applyRepositoryProjection(msg.result.Snapshot.Repository, msg.result.Snapshot.Graph)
		m.repoSnapshotLoaded = true
		m.err = nil
		m.startupReadPending = false
		m.startupFailed = false
		syncBrowseState(&m, m.repoStatus)
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	case refreshedSnapshotMsg:
		if msg.refreshGeneration != m.refreshGeneration || msg.result.RepositoryEpoch != m.repositoryEpoch {
			return markPullStale(m)
		}
		if msg.result.ErrorKind != ReadErrorNone {
			if m.status.Mode == state.ModeLoading {
				clearPullConfirmProjection(&m)
			}
			return m, nil
		}
		m.graphReadSnapshot = msg.result.Snapshot.Graph
		m.repoStatus = applyRepositoryProjection(msg.result.Snapshot.Repository, msg.result.Snapshot.Graph)
		m.repoSnapshotLoaded = true
		m.err = nil
		syncBrowseState(&m, m.repoStatus)
		m.status = deriveStatus(m.repoStatus)
		if m.commitInspectorOpen {
			return startCommitInspectorRevalidation(m)
		}
		return m, nil
	case loadedMsg:
		if m.repositoryRead != nil {
			return m, nil
		}
		if msg.epochSet && msg.epoch != m.repositoryEpoch {
			m.publish("app", "discard_stale_load", map[string]string{
				"expected_epoch": fmt.Sprintf("%d", m.repositoryEpoch),
				"received_epoch": fmt.Sprintf("%d", msg.epoch),
			})
			return m, nil
		}
		if msg.err != nil {
			m.err = msg.err
			m.status = m.status.WithError(msg.err.Error())
			m.publish("app", "load_error", map[string]string{"error": msg.err.Error()})
			return m, nil
		}
		m.err = nil
		m.repoSnapshotLoaded = true
		appliedStatus := applyRepositoryStatus(&m, msg.status)
		m.status = deriveStatus(appliedStatus)
		m.publish("app", "load_repo", map[string]string{
			"root":   appliedStatus.Root,
			"branch": appliedStatus.Branch,
			"head":   appliedStatus.Head,
		})
		return m, loadStashState(m.repo)
	case tickMsg:
		return m, tea.Batch(scheduleRefresh(), m.refreshCmd())
	case refreshedMsg:
		if m.repositoryRead != nil {
			return m, nil
		}
		generationMismatch := msg.generationSet && msg.refreshGeneration != m.refreshGeneration
		if (msg.epochSet && msg.epoch != m.repositoryEpoch) || generationMismatch {
			m, staleCmd := markPullStale(m)
			m.publish("app", "discard_stale_refresh", map[string]string{
				"expected_epoch": fmt.Sprintf("%d", m.repositoryEpoch),
				"received_epoch": fmt.Sprintf("%d", msg.epoch),
			})
			return m, staleCmd
		}
		if msg.epochSet && msg.epoch > m.commitInspectorEpoch {
			m.commitInspectorEpoch = msg.epoch
		}
		if msg.err != nil {
			if m.status.Mode == state.ModeLoading {
				clearPullConfirmProjection(&m)
			}
			if m.status.Mode == state.ModeOperationResult && m.operationResult != nil {
				m.operationResult.RefreshError = msg.err
				m.operationResult.Verification = VerificationUnknown
				m.operationResult.Headline = "PULL COMPLETED — STATE UNVERIFIED"
				m.operationResult.RefreshRetryable = true
				m.status.Message = m.operationResult.Headline
				m.status.Detail = "Press f to refresh repository state."
			}
			return m, nil
		}
		m.err = nil
		m.repoSnapshotLoaded = true
		appliedStatus := applyRepositoryStatus(&m, msg.status)
		if !m.branchOpen && (m.status.Mode == state.ModeBrowse || m.status.Mode == state.ModeEmpty || m.status.Mode == state.ModeError) {
			m.status = deriveStatus(appliedStatus)
		}
		stashCmd := loadStashState(m.repo)
		if m.commitInspectorOpen {
			var inspectorCmd tea.Cmd
			m, inspectorCmd = startCommitInspectorRevalidation(m)
			return m, tea.Batch(stashCmd, inspectorCmd)
		}
		return m, stashCmd
	default:
		return m, nil
	}
}
