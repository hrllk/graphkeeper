package app

import (
	"reflect"
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestApplyRepositoryStatusCharacterization(t *testing.T) {
	cached := []git.TagEntry{{Name: "cached", CommitHash: "cached-head"}}
	loaded := []git.TagEntry{{Name: "loaded", CommitHash: "loaded-head"}}

	tests := []struct {
		name             string
		initialTags      []git.TagEntry
		initialRepo      git.Status
		incoming         git.Status
		initialAttempted bool
		wantApplied      git.Status
		wantTags         []git.TagEntry
		wantAttempted    bool
	}{
		{
			name:        "cached tags attach when incoming status is not loaded",
			initialTags: cached,
			initialRepo: git.Status{TagProvenanceLoaded: true, TagSyncSummary: "cached provenance"},
			incoming:    git.Status{Head: "new", TagEntriesLoaded: false},
			wantApplied: git.Status{
				Head:                "new",
				Tags:                []string{"cached"},
				TagEntries:          []git.TagEntry{{Name: "cached", CommitHash: "cached-head"}},
				TagProvenanceLoaded: true,
				TagSyncSummary:      "cached provenance",
			},
			wantTags:      cached,
			wantAttempted: true,
		},
		{
			name:        "loaded tags take precedence over cache",
			initialTags: cached,
			incoming:    git.Status{Head: "new", TagEntriesLoaded: true, TagProvenanceLoaded: true, TagEntries: loaded},
			wantApplied: git.Status{
				Head:                "new",
				TagEntries:          []git.TagEntry{{Name: "loaded", CommitHash: "loaded-head"}},
				TagEntriesLoaded:    true,
				TagProvenanceLoaded: true,
			},
			wantTags:      loaded,
			wantAttempted: true,
		},
		{
			name:        "empty incoming tags preserve standard cache",
			initialTags: cached,
			incoming:    git.Status{Head: "new", TagEntriesLoaded: true},
			wantApplied: git.Status{
				Head:             "new",
				TagEntriesLoaded: true,
			},
			wantTags:         cached,
			initialAttempted: true,
			wantAttempted:    true,
		},
		{
			name:             "loaded provenance is monotonic",
			initialTags:      cached,
			initialAttempted: true,
			incoming:         git.Status{Head: "new", TagEntriesLoaded: true, TagProvenanceLoaded: false},
			wantApplied: git.Status{
				Head:             "new",
				TagEntriesLoaded: true,
			},
			wantTags:      cached,
			wantAttempted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var synced []git.Status
			original := applyRepositoryStatusSyncBrowseState
			applyRepositoryStatusSyncBrowseState = func(_ *model, status git.Status) {
				synced = append(synced, status)
			}
			defer func() { applyRepositoryStatusSyncBrowseState = original }()

			m := model{
				repositoryState: repositoryState{tagEntries: append([]git.TagEntry(nil), tt.initialTags...), repoStatus: tt.initialRepo, tagSyncAttempted: tt.initialAttempted},
				pullState:       pullState{}}
			got := applyRepositoryStatus(&m, tt.incoming)

			if len(synced) != 1 {
				t.Fatalf("sync calls = %d, want 1", len(synced))
			}
			if !reflect.DeepEqual(got, tt.wantApplied) {
				t.Fatalf("returned status = %#v, want independently expected status %#v", got, tt.wantApplied)
			}
			if !reflect.DeepEqual(m.repoStatus, tt.wantApplied) {
				t.Fatalf("repoStatus = %#v, want independently expected status %#v", m.repoStatus, tt.wantApplied)
			}
			if !reflect.DeepEqual(synced[0], tt.wantApplied) {
				t.Fatalf("synced status = %#v, want independently expected status %#v", synced[0], tt.wantApplied)
			}
			if !reflect.DeepEqual(m.tagEntries, tt.wantTags) {
				t.Fatalf("tag cache = %#v, want %#v", m.tagEntries, tt.wantTags)
			}

			if m.tagSyncAttempted != tt.wantAttempted {
				t.Fatalf("tagSyncAttempted = %v, want %v", m.tagSyncAttempted, tt.wantAttempted)
			}
		})
	}
}

func TestApplyRepositoryStatusDoesNotOwnUnrelatedState(t *testing.T) {
	original := applyRepositoryStatusSyncBrowseState
	applyRepositoryStatusSyncBrowseState = func(*model, git.Status) {}
	defer func() { applyRepositoryStatusSyncBrowseState = original }()

	sink := &recordingEventSink{}
	active := &pullRequest{ID: 7, Epoch: 3}
	handshake := map[string]bool{"commit": true}
	operation := &OperationResultSummary{NoOp: true, Headline: "existing result"}
	m := model{
		repositoryState: repositoryState{
			repoStatus: git.Status{Head: "old"},
			tagEntries: []git.TagEntry{{Name: "cached"}},
		},
		pullState: pullState{
			activePullRequest: active,
			handshakeCommits:  handshake,
			operationResult:   operation,
		},
		status:    state.New().WithLoading("working"),
		eventSink: sink}
	beforeStatus := m.status
	beforeActive := *m.activePullRequest
	beforeHandshake := make(map[string]bool, len(m.handshakeCommits))
	for key, value := range m.handshakeCommits {
		beforeHandshake[key] = value
	}
	beforeOperation := *m.operationResult
	beforeEventCount := 0

	applyRepositoryStatus(&m, git.Status{Head: "new", TagEntriesLoaded: true})

	if !reflect.DeepEqual(m.status, beforeStatus) {
		t.Fatalf("status changed: before=%#v after=%#v", beforeStatus, m.status)
	}
	if !reflect.DeepEqual(*m.activePullRequest, beforeActive) {
		t.Fatalf("active operation changed: before=%#v after=%#v", beforeActive, *m.activePullRequest)
	}
	if !reflect.DeepEqual(m.handshakeCommits, beforeHandshake) {
		t.Fatalf("handshake commits changed: before=%#v after=%#v", beforeHandshake, m.handshakeCommits)
	}
	if !reflect.DeepEqual(*m.operationResult, beforeOperation) {
		t.Fatalf("operation result changed: before=%#v after=%#v", beforeOperation, *m.operationResult)
	}
	if len(sink.events) != beforeEventCount {
		t.Fatalf("event count = %d, want %d", len(sink.events), beforeEventCount)
	}
}
