package app

import (
	"testing"

	"hrllk/graphkeeper/internal/git"
)

// pointerFixture puts one commit on the graph carrying the given decorations.
func pointerFixture(decorations []string, localBranches []string, known bool) git.Status {
	return git.Status{
		Root:               "/repo",
		Branch:             "main",
		Head:               "aaaaaaa",
		LocalBranches:      localBranches,
		LocalBranchesKnown: known,
		GraphCommits: []git.GraphCommit{
			{Hash: "bbbbbbb", Graph: "* ", Subject: "target", Decorations: decorations},
		},
	}
}

// The merge and rebase gate decides whether either action is offered at all, and
// it had no tests. A local branch whose name contains a slash was rejected, so on
// feature/x, hotfix/y or fix/z merge and rebase were unreachable: the decoration
// was skipped for containing "/", a check meant for remote refs that origin/ had
// already handled.
func TestIsLocalGraphPointerAcceptsSlashedBranchNames(t *testing.T) {
	for _, name := range []string{"feature/inspector-qa", "hotfix/ff-able", "fix/a/b", "main"} {
		rs := pointerFixture([]string{name}, []string{name}, true)
		if !isLocalGraphPointer(rs, 0, 0) {
			t.Fatalf("local branch %q was not treated as a local pointer", name)
		}
	}
}

func TestIsLocalGraphPointerRejectsRemoteAndTagDecorations(t *testing.T) {
	for _, dec := range []string{"origin/main", "origin/feature/x", "tag: v0.1.0"} {
		rs := pointerFixture([]string{dec}, []string{"main"}, true)
		if isLocalGraphPointer(rs, 0, 0) {
			t.Fatalf("%q should not be a local pointer", dec)
		}
	}
}

func TestIsLocalGraphPointerFollowsHeadArrow(t *testing.T) {
	rs := pointerFixture([]string{"HEAD -> feature/inspector-qa"}, nil, true)
	if !isLocalGraphPointer(rs, 0, 0) {
		t.Fatal("HEAD -> branch must be a local pointer even without the inventory")
	}
}

// With the inventory loaded, a decoration that is not one of the repository's
// local branches is not a local pointer.
func TestIsLocalGraphPointerUsesTheLocalBranchInventory(t *testing.T) {
	rs := pointerFixture([]string{"feature/gone"}, []string{"main", "feature/kept"}, true)
	if isLocalGraphPointer(rs, 0, 0) {
		t.Fatal("a decoration outside the local branch inventory was accepted")
	}
}

// Without the inventory the only thing left is the shape of the decoration, so
// the gate stays permissive rather than blocking every commit.
func TestIsLocalGraphPointerFallsBackWhenInventoryUnknown(t *testing.T) {
	rs := pointerFixture([]string{"feature/inspector-qa"}, nil, false)
	if !isLocalGraphPointer(rs, 0, 0) {
		t.Fatal("gate blocked everything while the local branch inventory was unknown")
	}
}
