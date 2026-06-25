package app

import "hrllk/graphkeeper/internal/git"

func groupStashesByBase(entries []git.StashEntry) map[string][]git.StashEntry {
	grouped := make(map[string][]git.StashEntry)
	for _, entry := range entries {
		if entry.BaseHash == "" {
			continue
		}
		grouped[entry.BaseHash] = append(grouped[entry.BaseHash], entry)
	}
	return grouped
}

func (m model) stashesForCommit(hash string) []git.StashEntry {
	if hash == "" || len(m.stashByBase) == 0 {
		return nil
	}
	entries := m.stashByBase[hash]
	if len(entries) == 0 {
		return nil
	}
	out := make([]git.StashEntry, len(entries))
	copy(out, entries)
	return out
}
