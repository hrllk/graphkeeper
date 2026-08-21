package app

import (
	"context"
	"sort"
	"time"

	"hrllk/graphkeeper/internal/git"
)

// TagSyncSummary describes whether provenance has ever been synchronized.
type TagSyncSummary string

const (
	TagSyncNeverSynced TagSyncSummary = "never_synced"
	TagSyncSynced      TagSyncSummary = "synced"
)

// TagProvenanceSnapshot is the application-owned provenance value. Persistence
// details, including its on-disk location and encoding, belong to an adapter.
type TagProvenanceSnapshot struct {
	LoadedAt   time.Time
	Summary    TagSyncSummary
	OriginSeen map[string]bool
}

// TagProvenanceStore persists tag provenance for the current repository.
type TagProvenanceStore interface {
	Load(context.Context) (TagProvenanceSnapshot, *ProvenanceError)
	Save(context.Context, TagProvenanceSnapshot) *ProvenanceError
}

type ProvenanceErrorKind string

const (
	ProvenanceNotFound    ProvenanceErrorKind = "not_found"
	ProvenanceMalformed   ProvenanceErrorKind = "malformed"
	ProvenanceUnavailable ProvenanceErrorKind = "unavailable"
)

type ProvenanceError struct{ Kind ProvenanceErrorKind }

func (e *ProvenanceError) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Kind)
}

// Keep the existing private names local to the tag policy and its tests.
type tagSyncSummary = TagSyncSummary

const (
	tagSyncNeverSynced = TagSyncNeverSynced
	tagSyncSynced      = TagSyncSynced
)

type tagSnapshot = TagProvenanceSnapshot

func buildTagSnapshot(entries []git.TagEntry, remoteTags map[string]bool, summary TagSyncSummary) TagProvenanceSnapshot {
	originSeen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		originSeen[entry.Name] = remoteTags[entry.Name]
	}
	return TagProvenanceSnapshot{LoadedAt: time.Now().UTC(), Summary: summary, OriginSeen: originSeen}
}

func applyTagSnapshot(status git.Status, snapshot TagProvenanceSnapshot) git.Status {
	status.TagProvenanceLoaded = true
	status.TagSyncSummary = string(snapshot.Summary)
	for i := range status.TagEntries {
		entry := &status.TagEntries[i]
		onOrigin, ok := snapshot.OriginSeen[entry.Name]
		entry.OriginKnown = ok
		entry.OnOrigin = ok && onOrigin
	}
	return attachGraphTagEntries(status)
}

func markTagOriginUnknown(status git.Status) git.Status {
	status.TagProvenanceLoaded = false
	status.TagSyncSummary = string(TagSyncNeverSynced)
	for i := range status.TagEntries {
		status.TagEntries[i].OriginKnown = false
		status.TagEntries[i].OnOrigin = false
	}
	return attachGraphTagEntries(status)
}

func tagSyncSummaryLabel(summary string) string {
	switch TagSyncSummary(summary) {
	case TagSyncSynced:
		return "synced"
	default:
		return "never synced"
	}
}

func tagSyncSummaryHelp(summary string) string {
	switch TagSyncSummary(summary) {
	case TagSyncSynced:
		return "Press F to refresh tag provenance."
	default:
		return "Press F to sync tag provenance."
	}
}

func loadLocalTagStatus(repo *git.Repo, status git.Status, stores ...TagProvenanceStore) (git.Status, error) {
	tags, err := repo.LocalTagEntries(context.Background())
	if err != nil {
		return status, err
	}
	status.TagEntries = tags
	status.TagEntriesLoaded = true
	status.Tags = make([]string, 0, len(tags))
	for _, entry := range tags {
		status.Tags = append(status.Tags, entry.Name)
	}
	if len(stores) == 0 || stores[0] == nil {
		return markTagOriginUnknown(status), nil
	}
	snapshot, snapErr := stores[0].Load(context.Background())
	if snapErr == nil {
		return applyTagSnapshot(status, snapshot), nil
	}
	// Not-found, malformed, and unavailable are deliberately non-fatal to the
	// unrelated Git/tag read; all three preserve unknown provenance.
	return markTagOriginUnknown(status), nil
}

func tagSnapshotFromRepo(repo *git.Repo, tags []git.TagEntry) (TagProvenanceSnapshot, error) {
	remoteTags, err := repo.OriginTagSet(context.Background())
	if err != nil {
		return TagProvenanceSnapshot{}, err
	}
	return buildTagSnapshot(tags, remoteTags, TagSyncSynced), nil
}

func attachGraphTagEntries(status git.Status) git.Status {
	if len(status.GraphCommits) == 0 || len(status.TagEntries) == 0 {
		return status
	}
	tagsByHash := make(map[string][]string)
	for _, entry := range status.TagEntries {
		if entry.CommitHash == "" || entry.Name == "" {
			continue
		}
		tagsByHash[entry.CommitHash] = append(tagsByHash[entry.CommitHash], entry.Name)
	}
	if len(tagsByHash) == 0 {
		return status
	}
	for i := range status.GraphCommits {
		commit := &status.GraphCommits[i]
		names := tagsByHash[commit.Hash]
		if len(names) == 0 {
			continue
		}
		sort.Strings(names)
		commit.Tags = append([]string(nil), names...)
	}
	return status
}
