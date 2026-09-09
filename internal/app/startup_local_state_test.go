package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// The neutral startup path -- the one production takes, because it injects a
// RepositoryRead port -- issued no commands at all, so the stash list and the
// Tags panel came up empty on it. The legacy branch that did load them opens
// with "if m.repositoryRead != nil { return m, nil }", so it only ever ran in a
// configuration without the port.
//
// See docs/20260910-0006-t6-startup-stash-tag-regression.md.
func TestNeutralStartupLoadsStashAndTagState(t *testing.T) {
	fixture := newCommandRepo(t)
	runGit(t, fixture.root, "tag", "v1.0.0")
	writeRepoFile(t, fixture.root, "file.txt", "dirty\n")
	runGit(t, fixture.root, "stash", "push", "-m", "wip")

	m := model{}
	m.repo = fixture.repo
	m.repositoryEpoch = 3

	cmd := loadLocalStateCmd(m)
	if cmd == nil {
		t.Fatal("the neutral path issued no command; stashes and tags never load")
	}

	var sawStash, sawTags bool
	for _, msg := range drainBatch(t, cmd) {
		switch loaded := msg.(type) {
		case stashLoadedMsg:
			sawStash = true
			if loaded.epoch != 3 {
				t.Errorf("stash load carries epoch %d, want 3", loaded.epoch)
			}
			if loaded.err != nil || len(loaded.entries) == 0 {
				t.Errorf("stash load found nothing: %+v", loaded)
			}
		case tagStateLoadedMsg:
			sawTags = true
			if loaded.epoch != 3 {
				t.Errorf("tag load carries epoch %d, want 3", loaded.epoch)
			}
			if loaded.err != nil || len(loaded.status.TagEntries) == 0 {
				t.Errorf("tag load found nothing: %+v", loaded)
			}
		}
	}
	if !sawStash {
		t.Error("no stash load was issued")
	}
	if !sawTags {
		t.Error("no tag load was issued")
	}
}

// A load that was in flight when the repository changed describes a repository
// that is no longer on screen. This mattered less when the load ran once at
// startup; it runs on every refresh now.
func TestStaleLocalStateLoadsAreDiscarded(t *testing.T) {
	m := model{}
	m.repositoryEpoch = 5

	stale, _ := handleStashUpdate(m, stashLoadedMsg{epoch: 4})
	if stale.(model).stashLoadAttempted {
		t.Error("a stash load from an older epoch was applied")
	}
	fresh, _ := handleStashUpdate(m, stashLoadedMsg{epoch: 5})
	if !fresh.(model).stashLoadAttempted {
		t.Error("a stash load for the current epoch was discarded")
	}
	// Epoch 0 means the caller did not stamp one, which the legacy call sites
	// still do not.
	unstamped, _ := handleStashUpdate(m, stashLoadedMsg{})
	if !unstamped.(model).stashLoadAttempted {
		t.Error("an unstamped stash load was discarded")
	}

	staleTags, _ := handleTagStateUpdate(m, tagStateLoadedMsg{epoch: 4})
	if len(staleTags.(model).tagEntries) != 0 {
		t.Error("a tag load from an older epoch was applied")
	}
}

func drainBatch(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	out := make([]tea.Msg, 0, len(batch))
	for _, inner := range batch {
		if inner == nil {
			continue
		}
		out = append(out, drainBatch(t, inner)...)
	}
	return out
}
