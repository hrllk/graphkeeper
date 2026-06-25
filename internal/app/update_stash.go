package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/telemetry"
)

func handleStashUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	msg2, ok := msg.(stashLoadedMsg)
	if !ok {
		return m, nil
	}
	if msg2.err != nil {
		telemetry.Log("app", "stash_load_failed", map[string]string{"error": msg2.err.Error()})
		return m, nil
	}
	m.stashEntries = append([]git.StashEntry(nil), msg2.entries...)
	m.stashByBase = groupStashesByBase(msg2.entries)
	telemetry.Log("app", "stash_load", map[string]string{"count": fmt.Sprint(len(msg2.entries))})
	return m, nil
}
