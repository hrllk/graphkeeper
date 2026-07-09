package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"hrllk/graphkeeper/internal/git"
)

type tagSyncSummary string

const (
	tagSyncNeverSynced tagSyncSummary = "never_synced"
	tagSyncStale       tagSyncSummary = "stale"
)

type tagSnapshot struct {
	LoadedAt   time.Time       `json:"loaded_at"`
	Summary    tagSyncSummary  `json:"summary"`
	OriginSeen map[string]bool `json:"origin_seen"`
}

func tagSnapshotPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".git", "graphkeeper", "tag-provenance.json")
}

func loadTagSnapshot(repoRoot string) (tagSnapshot, error) {
	data, err := os.ReadFile(tagSnapshotPath(repoRoot))
	if err != nil {
		return tagSnapshot{}, err
	}
	var snapshot tagSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return tagSnapshot{}, err
	}
	if snapshot.OriginSeen == nil {
		snapshot.OriginSeen = make(map[string]bool)
	}
	return snapshot, nil
}

func writeTagSnapshot(repoRoot string, snapshot tagSnapshot) error {
	path := tagSnapshotPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func buildTagSnapshot(entries []git.TagEntry, remoteTags map[string]bool, summary tagSyncSummary) tagSnapshot {
	originSeen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		originSeen[entry.Name] = remoteTags[entry.Name]
	}
	return tagSnapshot{
		LoadedAt:   time.Now(),
		Summary:    summary,
		OriginSeen: originSeen,
	}
}

func applyTagSnapshot(status git.Status, snapshot tagSnapshot) git.Status {
	status.TagProvenanceLoaded = true
	status.TagSyncSummary = string(snapshot.Summary)
	for i := range status.TagEntries {
		entry := &status.TagEntries[i]
		onOrigin, ok := snapshot.OriginSeen[entry.Name]
		entry.OriginKnown = ok
		entry.OnOrigin = ok && onOrigin
	}
	return status
}

func markTagOriginUnknown(status git.Status) git.Status {
	status.TagProvenanceLoaded = false
	status.TagSyncSummary = string(tagSyncNeverSynced)
	for i := range status.TagEntries {
		status.TagEntries[i].OriginKnown = false
		status.TagEntries[i].OnOrigin = false
	}
	return status
}

func tagSyncSummaryLabel(summary string) string {
	switch tagSyncSummary(summary) {
	case tagSyncStale:
		return "stale"
	case tagSyncNeverSynced:
		fallthrough
	default:
		return "never synced"
	}
}

func tagSyncSummaryHelp(summary string) string {
	switch tagSyncSummary(summary) {
	case tagSyncStale:
		return "Press F to refresh tag provenance."
	case tagSyncNeverSynced:
		fallthrough
	default:
		return "Press F to sync tag provenance."
	}
}

func tagSnapshotFromRepo(repo *git.Repo, tags []git.TagEntry) (tagSnapshot, error) {
	remoteTags, err := repo.OriginTagSet(context.Background())
	if err != nil {
		return tagSnapshot{}, err
	}
	return buildTagSnapshot(tags, remoteTags, tagSyncStale), nil
}
