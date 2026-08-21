package tagprovenance

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hrllk/graphkeeper/internal/app"
)

func TestAdapterSaveLoadRoundTripUsesRepositoryScopedFile(t *testing.T) {
	root := t.TempDir()
	store := New(root)
	want := app.TagProvenanceSnapshot{
		LoadedAt:   time.Date(2026, 8, 21, 12, 0, 0, 123456789, time.UTC),
		Summary:    app.TagSyncSynced,
		OriginSeen: map[string]bool{"v1.0.0": true, "v2.0.0": false},
	}
	if err := store.Save(context.Background(), want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !got.LoadedAt.Equal(want.LoadedAt) || got.Summary != want.Summary || len(got.OriginSeen) != 2 || !got.OriginSeen["v1.0.0"] {
		t.Fatalf("round-trip mismatch: %#v", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "graphkeeper", fileName)); err != nil {
		t.Fatalf("repository-scoped file missing: %v", err)
	}
}

func TestAdapterClassifiesMissingAndMalformedDocuments(t *testing.T) {
	root := t.TempDir()
	store := New(root)
	if _, err := store.Load(context.Background()); err == nil || err.Kind != app.ProvenanceNotFound {
		t.Fatalf("missing document error = %#v", err)
	}
	path := filepath.Join(root, ".git", "graphkeeper", fileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(context.Background()); err == nil || err.Kind != app.ProvenanceMalformed {
		t.Fatalf("malformed document error = %#v", err)
	}
}

func TestAdapterRejectsInvalidSnapshotAndCancellation(t *testing.T) {
	store := New(t.TempDir())
	if err := store.Save(context.Background(), app.TagProvenanceSnapshot{Summary: "invalid"}); err == nil || err.Kind != app.ProvenanceMalformed {
		t.Fatalf("invalid snapshot error = %#v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.Save(ctx, app.TagProvenanceSnapshot{Summary: app.TagSyncSynced}); err == nil || err.Kind != app.ProvenanceUnavailable {
		t.Fatalf("canceled save error = %#v", err)
	}
}
