package app

import (
	"errors"
	"strings"
	"testing"

	"hrllk/graphkeeper/internal/git"
)

// The stash popup used to print "(no stash entries)" for three different facts:
// the list was never loaded, loading failed, and loading succeeded with nothing
// in it. TODOS.md recorded the consequence: `git stash list` failing sent the
// error to the telemetry sink (update_stash.go) and left the user looking at a
// screen that says there is no stashed work.
//
// This matters more after the startup path loads stash state (T6), because then
// the failure is reachable on every launch rather than only after a mutation.
func TestStashPopupTellsTheThreeEmptyStatesApart(t *testing.T) {
	for _, tt := range []struct {
		name    string
		m       model
		wantNot string
		want    string
	}{
		{
			name: "never loaded",
			m:    model{},
			want: "not loaded",
		},
		{
			name: "load failed",
			m:    model{repositoryState: repositoryState{stashLoadAttempted: true, stashLoadError: "exit status 128"}},
			want: "unavailable",
		},
		{
			name: "loaded, none",
			m:    model{repositoryState: repositoryState{stashLoadAttempted: true}},
			want: "No stashed work",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := renderStashPopup(tt.m, 90, 24)
			if !strings.Contains(got, tt.want) {
				t.Fatalf("%s: want %q in popup, got:\n%s", tt.name, tt.want, got)
			}
		})
	}
}

// The failure has to name itself. A user who sees "unavailable" with no reason
// cannot tell a broken repository from a permissions problem.
func TestStashLoadFailureShowsTheReason(t *testing.T) {
	m := model{repositoryState: repositoryState{stashLoadAttempted: true, stashLoadError: "permission denied"}}
	got := renderStashPopup(m, 90, 24)
	if !strings.Contains(got, "permission denied") {
		t.Fatalf("popup dropped the reason:\n%s", got)
	}
}

// handleStashUpdate discarded the error entirely: it published to the event sink
// and returned without touching the model, so nothing downstream could tell that
// the load had failed.
func TestHandleStashUpdateKeepsTheError(t *testing.T) {
	got, _ := handleStashUpdate(model{}, stashLoadedMsg{err: errors.New("exit status 128")})
	m := got.(model)
	if !m.stashLoadAttempted {
		t.Fatal("a failed load is still an attempt; stashLoadAttempted must be set")
	}
	if m.stashLoadError == "" {
		t.Fatal("the error was dropped, so the renderer cannot distinguish failure from empty")
	}
}

// A later success has to clear the earlier failure, or the popup keeps reporting
// a problem that is gone. The 1s refresh tick makes this a real sequence.
func TestSuccessfulStashLoadClearsAPriorError(t *testing.T) {
	first, _ := handleStashUpdate(model{}, stashLoadedMsg{err: errors.New("boom")})
	second, _ := handleStashUpdate(first.(model), stashLoadedMsg{entries: []git.StashEntry{{Ref: "stash@{0}", Subject: "wip"}}})
	m := second.(model)
	if m.stashLoadError != "" {
		t.Fatalf("stale error survived a successful load: %q", m.stashLoadError)
	}
	if len(m.stashEntries) != 1 {
		t.Fatalf("entries not applied: %#v", m.stashEntries)
	}
}

// The Tags panel prints "No local tags found." whether or not the load ever ran.
// tagSyncAttempted only gated the second line (the "press F" hint), so the
// headline claimed a fact the app had not established. Startup does not load
// tags yet (T6), so today this is the state a user actually sees on launch.
func TestTagsPanelDoesNotClaimThereAreNoTagsBeforeLoading(t *testing.T) {
	notLoaded := renderSectionProjection(SectionProjection{}, sectionTags, false, 40, 6)
	if strings.Contains(notLoaded, "No local tags found") {
		t.Fatalf("unloaded Tags panel asserts there are no tags:\n%s", notLoaded)
	}
	if !strings.Contains(notLoaded, "not loaded") {
		t.Fatalf("unloaded Tags panel does not say so:\n%s", notLoaded)
	}

	loadedNone := renderSectionProjection(SectionProjection{}, sectionTags, true, 40, 6)
	if !strings.Contains(loadedNone, "No local tags") {
		t.Fatalf("loaded-and-empty Tags panel must state the fact:\n%s", loadedNone)
	}
}
