package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleTagStateUpdate folds a standalone tag load into the model.
//
// The legacy refresh command loaded tags inside itself and handed back a
// git.Status that already had them. The neutral path reads a snapshot that does
// not carry tags, so they arrive on their own, and this is where they are
// merged into the status already on screen.
func handleTagStateUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	loaded, ok := msg.(tagStateLoadedMsg)
	if !ok {
		return m, nil
	}
	if loaded.epoch != 0 && loaded.epoch != m.repositoryEpoch {
		return m, nil
	}
	if loaded.err != nil {
		m.publish("app", "tag_load_failed", map[string]string{"error": loaded.err.Error()})
		return m, nil
	}

	status := m.repoStatus
	status.TagEntries = loaded.status.TagEntries
	status.TagEntriesLoaded = loaded.status.TagEntriesLoaded
	status.Tags = loaded.status.Tags
	status.TagProvenanceLoaded = loaded.status.TagProvenanceLoaded
	status.TagSyncSummary = loaded.status.TagSyncSummary
	status = attachGraphTagEntries(status)

	m.repoStatus = status
	m.replaceTagEntries(status)
	if status.TagProvenanceLoaded {
		m.tagSyncAttempted = true
	}
	syncBrowseState(&m, status)
	return m, nil
}
