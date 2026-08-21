package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type testTagStore struct{ root string }

func newTestTagStore(root string) TagProvenanceStore { return testTagStore{root: root} }

func (s testTagStore) Load(context.Context) (TagProvenanceSnapshot, *ProvenanceError) {
	snapshot, err := loadTagSnapshot(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return TagProvenanceSnapshot{}, &ProvenanceError{Kind: ProvenanceNotFound}
		}
		return TagProvenanceSnapshot{}, &ProvenanceError{Kind: ProvenanceMalformed}
	}
	return snapshot, nil
}

func (s testTagStore) Save(_ context.Context, snapshot TagProvenanceSnapshot) *ProvenanceError {
	if err := writeTagSnapshot(s.root, snapshot); err != nil {
		return &ProvenanceError{Kind: ProvenanceUnavailable}
	}
	return nil
}

// These helpers are test fixtures for legacy app command tests. Production
// persistence is provided by TagProvenanceStore and its outbound adapter.
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
		snapshot.OriginSeen = map[string]bool{}
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
