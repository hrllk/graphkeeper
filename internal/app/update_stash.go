package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
)

func handleStashUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	msg2, ok := msg.(stashLoadedMsg)
	if !ok {
		return m, nil
	}
	// A failed load is still an attempt. Publishing to the event sink and
	// returning left stashEntries nil, which the popup rendered the same way as a
	// repository with no stashed work, so a failing `git stash list` was invisible.
	m.stashLoadAttempted = true
	if msg2.err != nil {
		m.stashLoadError = msg2.err.Error()
		m.publish("app", "stash_load_failed", map[string]string{"error": msg2.err.Error()})
		return m, nil
	}
	m.stashLoadError = ""
	m.stashEntries = append([]git.StashEntry(nil), msg2.entries...)
	m.stashByBase = groupStashesByBase(msg2.entries)
	m.publish("app", "stash_load", map[string]string{"count": fmt.Sprint(len(msg2.entries))})
	return m, nil
}
