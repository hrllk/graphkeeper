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

func handleLifecycleUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadedSnapshotMsg:
		if msg.result.RepositoryEpoch != m.repositoryEpoch {
			return m, nil
		}
		if msg.result.ErrorKind == ReadErrorInvalid || msg.result.ErrorKind == ReadErrorRepository || msg.result.ErrorKind == ReadErrorCanceled {
			return m, nil
		}
		m.graphReadSnapshot = msg.result.Snapshot.Graph
		return m, nil
	case refreshedSnapshotMsg:
		if msg.refreshGeneration != m.refreshGeneration || msg.result.RepositoryEpoch != m.repositoryEpoch {
			return m, nil
		}
		if msg.result.ErrorKind == ReadErrorInvalid || msg.result.ErrorKind == ReadErrorRepository || msg.result.ErrorKind == ReadErrorCanceled {
			return m, nil
		}
		m.graphReadSnapshot = msg.result.Snapshot.Graph
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
		msg.status = m.withCachedTagEntries(msg.status)
		m.err = nil
		m.repoSnapshotLoaded = true
		m.repoStatus = msg.status
		if msg.status.TagProvenanceLoaded {
			m.tagSyncAttempted = true
		}
		m.storeTagEntries(msg.status)
		syncBrowseState(&m, msg.status)
		m.status = deriveStatus(msg.status)
		m.publish("app", "load_repo", map[string]string{
			"root":   msg.status.Root,
			"branch": msg.status.Branch,
			"head":   msg.status.Head,
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
			if m.activePullRequest != nil && (m.status.Mode == state.ModeReview || m.status.Mode == state.ModeConfirm || m.status.Action == state.ActionPull) {
				m.pullConfirmStale = true
			}
			m.publish("app", "discard_stale_refresh", map[string]string{
				"expected_epoch": fmt.Sprintf("%d", m.repositoryEpoch),
				"received_epoch": fmt.Sprintf("%d", msg.epoch),
			})
			return m, nil
		}
		if msg.epochSet && msg.epoch > m.commitInspectorEpoch {
			m = invalidateCommitInspectorForEpoch(m)
			m.commitInspectorEpoch = msg.epoch
		}
		if msg.err != nil {
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
		msg.status = m.withCachedTagEntries(msg.status)
		m.err = nil
		m.repoSnapshotLoaded = true
		m.repoStatus = msg.status
		if msg.status.TagProvenanceLoaded {
			m.tagSyncAttempted = true
		}
		m.storeTagEntries(msg.status)
		syncBrowseState(&m, msg.status)
		if !m.branchOpen && (m.status.Mode == state.ModeBrowse || m.status.Mode == state.ModeEmpty || m.status.Mode == state.ModeError) {
			m.status = deriveStatus(msg.status)
		}
		return m, loadStashState(m.repo)
	default:
		return m, nil
	}
}
