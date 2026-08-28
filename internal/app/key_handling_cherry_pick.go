package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

func (m model) handleCherryPickPickKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.status = moveCherryPickTarget(m.status, -1)
		return m, nil
	case "down", "j":
		m.status = moveCherryPickTarget(m.status, 1)
		return m, nil
	case "space", " ":
		m.status = toggleCherryPickSelection(m.status)
		return m, nil
	case "enter":
		queue := selectedCherryPickTargets(m.status)
		if len(queue) == 0 {
			m.status = state.New().WithBlocked(state.BlockTargetEmpty, "No commits selected.", "Space toggles a commit into the cherry-pick queue.")
			return m, nil
		}
		m.status = operationLoadingStatusFor(progressCherryPick, "Cherry-picking...", state.ActionCherryPick)
		return m, executeCherryPick(m.repo, queue, m.commitLimit)
	case "esc":
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	default:
		return m, nil
	}
}
