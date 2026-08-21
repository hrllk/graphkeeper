package app

import (
	"context"
	"testing"

	"hrllk/graphkeeper/internal/graph"
)

type fakeRepositoryRead struct {
	result  ReadSnapshotResult
	request ReadRequest
}

func (f *fakeRepositoryRead) ReadSnapshot(_ context.Context, request ReadRequest) (ReadSnapshotResult, error) {
	f.request = request
	return f.result, nil
}

func TestLoadRepositorySnapshotUsesNeutralPortAndRequestIdentity(t *testing.T) {
	fake := &fakeRepositoryRead{result: ReadSnapshotResult{
		ErrorKind: ReadErrorNone, RepositoryEpoch: 4,
		Snapshot: ReadSnapshot{Root: "/repo", Graph: graph.Snapshot{Branch: "main", Head: "abc"}},
	}}
	msg := loadRepositorySnapshot(fake, 25, 4)()
	loaded, ok := msg.(loadedSnapshotMsg)
	if !ok || loaded.result.Snapshot.Graph.Head != "abc" {
		t.Fatalf("unexpected message: %#v", msg)
	}
	if fake.request.CommitLimit != 25 || fake.request.RepositoryEpoch != 4 {
		t.Fatalf("request identity not forwarded: %#v", fake.request)
	}
}

func TestLifecycleRejectsStaleNeutralSnapshotAndPreservesGraph(t *testing.T) {
	m := model{repositoryEpoch: 9, graphReadSnapshot: graph.Snapshot{Head: "current"}}
	got, _ := handleLifecycleUpdate(m, loadedSnapshotMsg{result: ReadSnapshotResult{
		RepositoryEpoch: 8, ErrorKind: ReadErrorNone,
		Snapshot: ReadSnapshot{Graph: graph.Snapshot{Head: "stale"}},
	}})
	modelGot := got.(model)
	if modelGot.graphReadSnapshot.Head != "current" {
		t.Fatalf("stale snapshot replaced current graph: %#v", modelGot.graphReadSnapshot)
	}
}

func TestLifecycleAcceptsCurrentNeutralSnapshot(t *testing.T) {
	m := model{repositoryEpoch: 9}
	got, _ := handleLifecycleUpdate(m, loadedSnapshotMsg{result: ReadSnapshotResult{
		RepositoryEpoch: 9, ErrorKind: ReadErrorNone,
		Snapshot: ReadSnapshot{Graph: graph.Snapshot{Head: "fresh"}},
	}})
	if got.(model).graphReadSnapshot.Head != "fresh" {
		t.Fatal("current neutral snapshot was not projected")
	}
}
