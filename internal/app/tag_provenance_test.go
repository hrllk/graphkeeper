package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRepoStateShowsLocalTagsWithoutSnapshot(t *testing.T) {
	fixture := newCommandRepo(t)
	if err := fixture.repo.CreateTag(context.Background(), "v1.0.0", fixture.initialHash); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	msg := cmdResult(t, loadRepoState(fixture.repo, 40))
	loaded, ok := msg.(loadedMsg)
	if !ok {
		t.Fatalf("expected loadedMsg, got %T", msg)
	}
	if loaded.err != nil {
		t.Fatalf("loadRepoState err = %v", loaded.err)
	}
	if !loaded.status.TagEntriesLoaded {
		t.Fatal("expected local tags to be loaded on startup")
	}
	if loaded.status.TagProvenanceLoaded {
		t.Fatal("expected provenance to remain unknown without snapshot")
	}
	if loaded.status.TagSyncSummary != string(tagSyncNeverSynced) {
		t.Fatalf("expected never synced summary, got %q", loaded.status.TagSyncSummary)
	}
	if len(loaded.status.TagEntries) != 1 {
		t.Fatalf("expected one local tag, got %+v", loaded.status.TagEntries)
	}
	if loaded.status.TagEntries[0].Name != "v1.0.0" || loaded.status.TagEntries[0].OriginKnown {
		t.Fatalf("expected local tag to render as unknown, got %+v", loaded.status.TagEntries[0])
	}
}

func TestFetchTagsWritesSnapshotAndMarksProvenance(t *testing.T) {
	fixture := newCommandRepo(t)
	if err := fixture.repo.CreateTag(context.Background(), "v1.0.0", fixture.initialHash); err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}
	runGit(t, fixture.root, "push", "origin", "v1.0.0")

	msg := cmdResult(t, fetchTagsRepoState(fixture.repo, 40, newTestTagStore(fixture.root)))
	fetched, ok := msg.(fetchedMsg)
	if !ok {
		t.Fatalf("expected fetchedMsg, got %T", msg)
	}
	if fetched.err != nil {
		t.Fatalf("fetchTagsRepoState err = %v", fetched.err)
	}
	if !fetched.status.TagProvenanceLoaded {
		t.Fatal("expected provenance to be loaded after fetch")
	}
	if fetched.status.TagSyncSummary != string(tagSyncSynced) {
		t.Fatalf("expected synced summary after fetch, got %q", fetched.status.TagSyncSummary)
	}
	if len(fetched.status.TagEntries) != 1 {
		t.Fatalf("expected one tag entry after fetch, got %+v", fetched.status.TagEntries)
	}
	if !fetched.status.TagEntries[0].OriginKnown || !fetched.status.TagEntries[0].OnOrigin {
		t.Fatalf("expected fetched tag to be marked as origin-visible, got %+v", fetched.status.TagEntries[0])
	}

	snapshotPath := tagSnapshotPath(fixture.root)
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("expected snapshot file at %s: %v", snapshotPath, err)
	}
	var snapshot tagSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("snapshot unmarshal failed: %v", err)
	}
	if snapshot.Summary != tagSyncSynced {
		t.Fatalf("expected synced snapshot summary, got %q", snapshot.Summary)
	}
	if !snapshot.OriginSeen["v1.0.0"] {
		t.Fatalf("expected snapshot to record remote tag presence, got %+v", snapshot.OriginSeen)
	}
}

func TestFetchTagsDoesNotPruneRemoteBranches(t *testing.T) {
	fixture := newCommandRepo(t)
	runGit(t, fixture.root, "checkout", "-b", "feature")
	makeLocalCommit(t, fixture.root, "feature.txt", "feature\n", "feature commit")
	runGit(t, fixture.root, "push", "-u", "origin", "feature")
	runGit(t, fixture.root, "checkout", "main")

	// Simulate a branch deleted on the remote while keeping the local
	// remote-tracking ref. Refreshing tags must not prune that ref.
	runGit(t, fixture.remote, "update-ref", "-d", "refs/heads/feature")

	msg := cmdResult(t, fetchTagsRepoState(fixture.repo, 40, newTestTagStore(fixture.root)))
	fetched, ok := msg.(fetchedMsg)
	if !ok {
		t.Fatalf("expected fetchedMsg, got %T", msg)
	}
	if fetched.err != nil {
		t.Fatalf("fetchTagsRepoState err = %v", fetched.err)
	}
	for _, branch := range fetched.status.RemoteBranches {
		if branch == "origin/feature" {
			return
		}
	}
	t.Fatalf("expected origin/feature to remain after tag refresh, got %v", fetched.status.RemoteBranches)
}

func TestTagSnapshotPathLivesUnderGitDir(t *testing.T) {
	got := tagSnapshotPath(filepath.Join("/tmp", "repo"))
	want := filepath.Join("/tmp", "repo", ".git", "graphkeeper", "tag-provenance.json")
	if got != want {
		t.Fatalf("expected snapshot path %q, got %q", want, got)
	}
}
