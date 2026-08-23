package app

import "hrllk/graphkeeper/internal/git"

var applyRepositoryStatusSyncBrowseState = syncBrowseState

func applyRepositoryStatus(m *model, status git.Status) git.Status {
	status = m.withCachedTagEntries(status)
	m.repoStatus = status
	if status.TagProvenanceLoaded {
		m.tagSyncAttempted = true
	}
	m.storeTagEntries(status)
	applyRepositoryStatusSyncBrowseState(m, status)
	return status
}
