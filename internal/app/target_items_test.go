package app

import (
	"testing"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestBranchUpstream(t *testing.T) {
	rs := git.Status{
		Branch:   "main",
		Upstream: "origin/main",
		BranchUpstreams: map[string]string{
			"feature": "origin/feature",
		},
	}
	tests := []struct {
		name      string
		ref       string
		wantUp    string
		wantKnown bool
	}{
		{name: "explicit mapping", ref: "feature", wantUp: "origin/feature", wantKnown: true},
		{name: "current branch fallback", ref: "main", wantUp: "origin/main", wantKnown: true},
		{name: "unknown", ref: "missing", wantUp: "", wantKnown: false},
		{name: "empty", ref: "", wantUp: "", wantKnown: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUp, gotKnown := branchUpstream(rs, tt.ref)
			if gotUp != tt.wantUp || gotKnown != tt.wantKnown {
				t.Fatalf("branchUpstream() = (%q, %v), want (%q, %v)", gotUp, gotKnown, tt.wantUp, tt.wantKnown)
			}
		})
	}
}

func TestBuildActionTargetItems(t *testing.T) {
	rs := git.Status{
		Branch:          "main",
		LocalBranches:   []string{"main", "feature"},
		BranchUpstreams: map[string]string{"main": "origin/main", "feature": ""},
		RemoteBranches:  []string{"origin/HEAD", "origin/main"},
		Tags:            []string{"v1.0.0"},
		Branches:        []string{"fallback"},
	}
	got := buildActionTargetItems(rs)
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}
	if got[0].Ref != "main" || got[0].NoUpstream {
		t.Fatalf("main target = %#v", got[0])
	}
	if got[1].Ref != "feature" || !got[1].NoUpstream {
		t.Fatalf("feature target = %#v", got[1])
	}
	if got[2].Kind != state.TargetKindRemote || got[2].Ref != "origin/main" {
		t.Fatalf("remote target = %#v", got[2])
	}
	if got[3].Kind != state.TargetKindTag || got[3].Ref != "v1.0.0" {
		t.Fatalf("tag target = %#v", got[3])
	}
}

func TestBuildActionTargetItemsFallsBackToBranches(t *testing.T) {
	got := buildActionTargetItems(git.Status{Branches: []string{"main", "dev"}})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Ref != "main" || got[1].Ref != "dev" {
		t.Fatalf("unexpected fallback targets = %#v", got)
	}
}

func TestBuildCurrentSectionTargets(t *testing.T) {
	got := buildCurrentSectionTargets(git.Status{
		Branch:          "main",
		LocalBranches:   []string{"main", "feature"},
		BranchUpstreams: map[string]string{"main": "origin/main", "feature": ""},
		Tracking:        map[string]git.BranchTracking{"main": {Behind: 2}, "feature": {Ahead: 1}},
		MergeInProgress: true,
	})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if !got[0].Current || !got[0].NeedsPull || got[0].NeedsPush || !got[0].MergeConflicted {
		t.Fatalf("main section target = %#v", got[0])
	}
	if got[0].WorktreeDirty {
		t.Fatalf("expected clean worktree by default, got %#v", got[0])
	}
	if !got[1].NeedsPush || !got[1].NoUpstream {
		t.Fatalf("feature section target = %#v", got[1])
	}
}

func TestBuildCurrentSectionTargetsMarksDirtyCurrentBranch(t *testing.T) {
	got := buildCurrentSectionTargets(git.Status{
		Branch:        "main",
		Head:          "abc123",
		WorktreeDirty: true,
	})
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if !got[0].Current || !got[0].WorktreeDirty {
		t.Fatalf("expected dirty current branch target, got %#v", got[0])
	}
}

func TestBuildCurrentSectionTargetsFallsBackToHead(t *testing.T) {
	got := buildCurrentSectionTargets(git.Status{Head: "abc123", MergeInProgress: true})
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Ref != "abc123" || !got[0].Current {
		t.Fatalf("head section target = %#v", got[0])
	}
}

func TestBuildRemoteSectionTargets(t *testing.T) {
	got := buildRemoteSectionTargets(git.Status{
		DefaultBranch:  "main",
		RemoteBranches: []string{"origin/HEAD", "origin/main", "invalid"},
	})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if !got[0].Default || !got[1].Default {
		t.Fatalf("expected default remote markers for HEAD and default branch")
	}
}

func TestBuildTagSectionTargets(t *testing.T) {
	got := buildTagSectionTargets(git.Status{Tags: []string{"v1", "v2"}})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Kind != state.TargetKindTag || got[1].Ref != "v2" {
		t.Fatalf("tags = %#v", got)
	}
}
