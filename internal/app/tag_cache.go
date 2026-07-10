package app

import "hrllk/graphkeeper/internal/git"

func (m *model) storeTagEntries(status git.Status) {
	if len(status.TagEntries) == 0 {
		return
	}
	m.tagEntries = append([]git.TagEntry(nil), status.TagEntries...)
}

func (m *model) replaceTagEntries(status git.Status) {
	m.tagEntries = append([]git.TagEntry(nil), status.TagEntries...)
}

func (m model) withCachedTagEntries(status git.Status) git.Status {
	if len(m.tagEntries) == 0 || status.TagEntriesLoaded {
		return attachGraphTagEntries(status)
	}
	status.TagEntries = append([]git.TagEntry(nil), m.tagEntries...)
	status.Tags = make([]string, 0, len(m.tagEntries))
	for _, entry := range m.tagEntries {
		status.Tags = append(status.Tags, entry.Name)
	}
	status.TagProvenanceLoaded = m.repoStatus.TagProvenanceLoaded
	status.TagSyncSummary = m.repoStatus.TagSyncSummary
	return attachGraphTagEntries(status)
}
