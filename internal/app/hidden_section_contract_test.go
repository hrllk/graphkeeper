package app

import (
	"strings"
	"testing"
)

func TestHiddenHotkeySectionContractForLocalAndRemote(t *testing.T) {
	local := hiddenHotkeyContentLines(model{navigationState: navigationState{activeSection: sectionCurrent}}, 80)
	localText := strings.Join(local, "\n")
	if !strings.Contains(localText, "n: new branch") {
		t.Fatalf("Local hidden help missing new-branch key: %q", localText)
	}

	remote := hiddenHotkeyContentLines(model{navigationState: navigationState{activeSection: sectionRemote}}, 80)
	remoteText := strings.Join(remote, "\n")
	if !strings.Contains(remoteText, "space: checkout") {
		t.Fatalf("Remote hidden help lost checkout key: %q", remoteText)
	}
	for _, forbidden := range []string{"f: fetch", "p: pull", "d: delete branch"} {
		if strings.Contains(remoteText, forbidden) {
			t.Fatalf("Remote hidden help contains forbidden key %q: %q", forbidden, remoteText)
		}
	}
}
