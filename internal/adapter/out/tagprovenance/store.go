package tagprovenance

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"hrllk/graphkeeper/internal/app"
)

const fileName = "tag-provenance.json"

type Adapter struct{ root string }

// New constructs the repository-scoped provenance store. root must be the
// absolute repository root supplied by the composition root.
func New(root string) app.TagProvenanceStore { return &Adapter{root: root} }

type document struct {
	LoadedAt   string             `json:"loaded_at"`
	Summary    app.TagSyncSummary `json:"summary"`
	OriginSeen map[string]bool    `json:"origin_seen"`
}

func (a *Adapter) path() string { return filepath.Join(a.root, ".git", "graphkeeper", fileName) }

func (a *Adapter) Load(ctx context.Context) (app.TagProvenanceSnapshot, *app.ProvenanceError) {
	if err := ctx.Err(); err != nil {
		return emptySnapshot(), unavailable()
	}
	data, err := os.ReadFile(a.path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return emptySnapshot(), &app.ProvenanceError{Kind: app.ProvenanceNotFound}
		}
		return emptySnapshot(), unavailable()
	}
	var raw document
	if err := json.Unmarshal(data, &raw); err != nil {
		return emptySnapshot(), &app.ProvenanceError{Kind: app.ProvenanceMalformed}
	}
	loadedAt, err := parseTime(raw.LoadedAt)
	if err != nil || (raw.Summary != app.TagSyncNeverSynced && raw.Summary != app.TagSyncSynced) {
		return emptySnapshot(), &app.ProvenanceError{Kind: app.ProvenanceMalformed}
	}
	if raw.OriginSeen == nil {
		raw.OriginSeen = map[string]bool{}
	}
	return app.TagProvenanceSnapshot{LoadedAt: loadedAt, Summary: raw.Summary, OriginSeen: raw.OriginSeen}, nil
}

func (a *Adapter) Save(ctx context.Context, snapshot app.TagProvenanceSnapshot) *app.ProvenanceError {
	if err := ctx.Err(); err != nil {
		return unavailable()
	}
	if snapshot.OriginSeen == nil {
		snapshot.OriginSeen = map[string]bool{}
	}
	if snapshot.Summary != app.TagSyncNeverSynced && snapshot.Summary != app.TagSyncSynced {
		return &app.ProvenanceError{Kind: app.ProvenanceMalformed}
	}
	loadedAt := snapshot.LoadedAt.UTC()
	if loadedAt.IsZero() {
		loadedAt = loadedAt.UTC()
	}
	raw := document{LoadedAt: loadedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), Summary: snapshot.Summary, OriginSeen: snapshot.OriginSeen}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return unavailable()
	}
	dir := filepath.Dir(a.path())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return unavailable()
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(a.path()); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(dir, ".tag-provenance-*")
	if err != nil {
		return unavailable()
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = tmp.Close(); _ = os.Remove(tmpName) }
	if err := tmp.Chmod(mode); err != nil {
		cleanup()
		return unavailable()
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return unavailable()
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return unavailable()
	}
	if err := os.Rename(tmpName, a.path()); err != nil {
		_ = os.Remove(tmpName)
		return unavailable()
	}
	return nil
}

func emptySnapshot() app.TagProvenanceSnapshot {
	return app.TagProvenanceSnapshot{Summary: app.TagSyncNeverSynced, OriginSeen: map[string]bool{}}
}

func unavailable() *app.ProvenanceError { return &app.ProvenanceError{Kind: app.ProvenanceUnavailable} }

func parseTime(value string) (t time.Time, err error) {
	if value == "" {
		return time.Time{}, errors.New("missing loaded_at")
	}
	return time.Parse(time.RFC3339Nano, value)
}
