package app

import (
	"strings"
	"testing"

	"hrllk/graphkeeper/internal/git"
)

func TestCompactDecorationInfoHidesRemoteOnlyBranches(t *testing.T) {
	info := compactDecorationInfo([]string{"origin/feature", "origin/HEAD -> origin/main"}, LocalBranchInventory{Names: []string{"main"}, Known: true, Fresh: true})
	if info.Text != "-" || info.HasBranch {
		t.Fatalf("remote-only decorations must be hidden, got %+v", info)
	}
}

func TestCompactDecorationInfoShowsOnlyLocalBranchWithTrackingDecoration(t *testing.T) {
	info := compactDecorationInfo([]string{"HEAD -> main", "origin/main", "origin/feature"}, LocalBranchInventory{Names: []string{"main"}, Known: true, Fresh: true})
	if !strings.Contains(info.Text, "main") || strings.Contains(info.Text, "feature") || strings.Contains(info.Text, "origin/") {
		t.Fatalf("expected only local main label, got %q", info.Text)
	}
}

func TestGraphProjectionCarriesInventoryValidity(t *testing.T) {
	m := model{
		repositoryState: repositoryState{repoStatus: git.Status{LocalBranches: []string{}, LocalBranchesKnown: true, LocalBranchesFresh: true}, repositoryEpoch: 9},
		pullState:       pullState{}}
	projection := m.screenProjection(80, 10)
	if !projection.Graph.LocalBranchInventory.Known || projection.Graph.LocalBranchInventory.Epoch != 9 {
		t.Fatalf("expected known inventory projection, got %+v", projection.Graph.LocalBranchInventory)
	}
}
