package app

import "testing"

func TestHiddenHotkeySectionContractForLocalAndRemote(t *testing.T) {
	local := sectionHotkeyItems(t, sectionCurrent)
	if !hotkeyListed(local, "n", "new branch") {
		t.Fatalf("Local hidden help missing new-branch key: %#v", local)
	}

	remote := sectionHotkeyItems(t, sectionRemote)
	if !hotkeyListed(remote, "space", "checkout") {
		t.Fatalf("Remote hidden help lost checkout key: %#v", remote)
	}
	// d deletes the remote branch behind a confirmation and README advertises it,
	// so the help lists it rather than hiding a working destructive key.
	if !hotkeyListed(remote, "d", "delete remote branch") {
		t.Fatalf("Remote hidden help hides the remote delete: %#v", remote)
	}
	// p only branches on Current and Graph, so it does nothing here. f does work
	// in Remote, but it is global and belongs to the Global section, not this one.
	for _, forbidden := range []string{"p", "f", "F", "S", "P"} {
		if hotkeyKeyListed(remote, forbidden) {
			t.Fatalf("Remote section lists %q, which is not a Remote key: %#v", forbidden, remote)
		}
	}
}

func sectionHotkeyItems(t *testing.T, section graphSection) []hiddenHotkeyItem {
	t.Helper()
	for _, s := range hiddenHotkeySections(model{navigationState: navigationState{activeSection: section}}) {
		if s.active {
			items := make([]hiddenHotkeyItem, 0)
			for _, g := range s.groups {
				items = append(items, g.items...)
			}
			return items
		}
	}
	t.Fatalf("no active section for %v", section)
	return nil
}

func hotkeyListed(items []hiddenHotkeyItem, key, desc string) bool {
	for _, item := range items {
		if item.key == key && item.desc == desc {
			return true
		}
	}
	return false
}

func hotkeyKeyListed(items []hiddenHotkeyItem, key string) bool {
	for _, item := range items {
		if item.key == key {
			return true
		}
	}
	return false
}
