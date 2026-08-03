package app

import (
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestHandleLifecycleUpdateDiscardsStaleRefresh(t *testing.T) {
	m := model{
		repositoryEpoch: 2,
		status:          state.New().WithBrowse(),
		repoStatus:      git.Status{Root: "/repo", Branch: "feature", Head: "new-head"},
	}

	got, cmd := handleLifecycleUpdate(m, refreshedMsg{
		status:   git.Status{Root: "/repo", Branch: "main", Head: "old-head"},
		epoch:    1,
		epochSet: true,
	})

	if cmd != nil {
		t.Fatal("stale refresh must not schedule follow-up work")
	}
	updated := got.(model)
	if updated.repoStatus.Head != "new-head" {
		t.Fatalf("stale refresh changed HEAD to %q", updated.repoStatus.Head)
	}
}

func TestHandleLifecycleUpdateDiscardsStaleInitialLoad(t *testing.T) {
	m := model{
		repositoryEpoch: 1,
		status:          state.New().WithBrowse(),
		repoStatus:      git.Status{Root: "/repo", Branch: "feature", Head: "new-head"},
	}

	got, cmd := handleLifecycleUpdate(m, loadedMsg{
		status:   git.Status{Root: "/repo", Branch: "main", Head: "old-head"},
		epoch:    0,
		epochSet: true,
	})

	if cmd != nil {
		t.Fatal("stale initial load must not schedule follow-up work")
	}
	updated := got.(model)
	if updated.repoStatus.Head != "new-head" {
		t.Fatalf("stale initial load changed HEAD to %q", updated.repoStatus.Head)
	}
}

func TestHandleLifecycleUpdateClearsErrorAfterSuccessfulLoad(t *testing.T) {
	m := model{
		err:           errRepositoryUnavailable,
		status:        state.New().WithError("repository unavailable"),
		sectionCursor: testSectionCursors(),
	}

	got, _ := handleLifecycleUpdate(m, loadedMsg{status: git.Status{Root: "/repo", Branch: "main"}})
	updated := got.(model)
	if updated.err != nil {
		t.Fatalf("successful load retained error: %v", updated.err)
	}
}

func TestHandleLifecycleUpdateClearsErrorAfterSuccessfulRefresh(t *testing.T) {
	m := model{
		err:             errRepositoryUnavailable,
		repositoryEpoch: 2,
		status:          state.New().WithBrowse(),
		repoStatus:      git.Status{Root: "/repo", Branch: "old"},
		sectionCursor:   testSectionCursors(),
	}

	got, _ := handleLifecycleUpdate(m, refreshedMsg{status: git.Status{Root: "/repo", Branch: "main"}, epoch: 2, epochSet: true})
	updated := got.(model)
	if updated.err != nil {
		t.Fatalf("successful refresh retained error: %v", updated.err)
	}
}

func testSectionCursors() map[graphSection]int {
	return map[graphSection]int{
		sectionGraph:   0,
		sectionCurrent: 0,
		sectionRemote:  0,
		sectionTags:    0,
	}
}
