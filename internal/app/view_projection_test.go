package app

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
)

func TestRenderGraphProjectionUsesProjectionInput(t *testing.T) {
	projection := GraphProjection{
		Rows: []graph.Row{{
			Commit: graph.Node{Hash: "abcdef12", Author: "author", Subject: "projection subject"},
			Graph:  "*",
		}},
		PageSize: 1,
		Active:   true,
	}

	got := renderGraphProjection(projection, 120, 4)
	if !strings.Contains(got, "abcde") || !strings.Contains(got, "author") {
		t.Fatalf("projection values were not rendered: %q", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if lipgloss.Width(line) > 120 {
			t.Fatalf("line exceeds visible width: %d: %q", lipgloss.Width(line), line)
		}
	}
}

func TestRenderSectionProjectionDoesNotReadModelState(t *testing.T) {
	projection := SectionProjection{
		Items:  []state.TargetItem{{Name: "feature"}},
		Cursor: 0,
		Active: true,
	}

	got := renderSectionProjection(projection, sectionCurrent, true, 40, 3)
	if !strings.Contains(got, "feature") {
		t.Fatalf("projection item was not rendered: %q", got)
	}
}

func TestRenderContextProjectionPreservesViewportOffset(t *testing.T) {
	projection := ContextProjection{
		Title:       "Graph",
		InfoLines:   []string{"first", "second", "third", "fourth"},
		ActionLines: []string{"j/k move"},
		Scroll:      1,
	}

	got := renderContextViewport(projection.InfoLines, 2, projection.Scroll, 60)
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "second") || strings.Contains(joined, "first") {
		t.Fatalf("viewport offset was not applied: %q", joined)
	}
}

func TestRenderContextProjectionComposesDecisionAndRecommendationWithoutDuplicateCounts(t *testing.T) {
	decision := &BranchDecisionContext{
		CurrentRef:  "main",
		TargetRef:   "origin/main",
		CurrentOnly: 2,
		TargetOnly:  3,
		Relation:    RelationDiverged,
		ReasonLines: []string{"양쪽에 고유 commit이 있어 merge 또는 rebase로 통합합니다."},
	}
	recommendation := &DivergedRecommendation{Branch: "main", Upstream: "origin/main", LocalOnly: 2, UpstreamOnly: 3}
	got := renderContextProjection(ContextProjection{
		Title:          "Current",
		ActionLines:    []string{"• j/k: move"},
		Decision:       decision,
		Recommendation: recommendation,
	}, 120, 8)

	if !strings.Contains(got, "target-only 3") || !strings.Contains(got, "p: fetch, then m: merge / r: rebase") {
		t.Fatalf("decision context or action is missing: %q", got)
	}
	if strings.Contains(got, "upstream-only") || strings.Count(got, "origin/main") != 1 {
		t.Fatalf("expected one shared target/count block, got %q", got)
	}
	if strings.Index(got, "target-only 3") > strings.Index(got, "p: fetch") {
		t.Fatalf("expected decision context before action: %q", got)
	}
}

func TestContextProjectionKeepsDecisionOutOfGraphAndTags(t *testing.T) {
	for _, section := range []graphSection{sectionGraph, sectionTags} {
		t.Run(sectionName(section), func(t *testing.T) {
			m := model{
				navigationState: navigationState{
					activeSection: section,
				},
				repositoryState: repositoryState{
					repoStatus: git.Status{
						Branch: "main", Head: "abc", Upstream: "origin/main",
						TrackingKnown: true, TrackingFresh: true,
						Tracking: map[string]git.BranchTracking{"main": {Ahead: 0, Behind: 1}},
					},
				},
				pullState: pullState{},
				status:    state.Status{Mode: state.ModeBrowse}}
			if got := m.contextProjection(100); got.Decision != nil {
				t.Fatalf("section %v must not receive decision context: %+v", section, got.Decision)
			}
		})
	}
}

func TestContextProjectionOmitsUnavailableDecisionAndDoesNotMutateModel(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch: "main", Head: "abc", Upstream: "origin/main",
				TrackingKnown: false, TrackingFresh: true,
				Tracking: map[string]git.BranchTracking{"main": {Ahead: 2, Behind: 1}},
			},
		},
		pullState: pullState{
			activePullRequest: &pullRequest{ID: 7, Epoch: 3},
		},
		status: state.Status{Mode: state.ModeBrowse}}
	before := m
	got := m.contextProjection(80)
	if got.Decision != nil {
		t.Fatalf("unavailable snapshot must not render decision context: %+v", got.Decision)
	}
	if !reflect.DeepEqual(m, before) {
		t.Fatalf("projection mutated model: before=%+v after=%+v", before, m)
	}
}

func TestContextProjectionKeepsDecisionWhilePullIsActive(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionCurrent,
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch: "main", Head: "abc", Upstream: "origin/main",
				TrackingKnown: true, TrackingFresh: true,
				Tracking: map[string]git.BranchTracking{"main": {Ahead: 0, Behind: 2}},
			},
		},
		pullState: pullState{
			activePullRequest: &pullRequest{ID: 7, Epoch: 3},
		},
		status: state.Status{Mode: state.ModeBrowse}}
	got := m.contextProjection(80)
	if got.Decision == nil {
		t.Fatal("valid decision context should remain visible while pull workflow is active")
	}
	if got.Recommendation != nil {
		t.Fatal("legacy recommendation must remain suppressed while pull workflow is active")
	}
}

func TestRenderContextProjectionKeepsDecisionOutOfOtherSections(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionRemote,
		},
		repositoryState: repositoryState{
			repoStatus: git.Status{
				Branch: "main", Head: "abc", Upstream: "origin/main",
				TrackingKnown: true, TrackingFresh: true,
				Tracking: map[string]git.BranchTracking{"main": {Ahead: 0, Behind: 1}},
			},
		},
		pullState: pullState{}}
	if got := m.contextProjection(100); got.Decision != nil {
		t.Fatalf("non-current section must not receive decision context: %+v", got.Decision)
	}
}

func TestContextProjectionKeepsDecisionOutOfPreviewAndReview(t *testing.T) {
	for _, mode := range []state.Mode{state.ModeReview, state.ModeOutcomePreview} {
		t.Run(string(mode), func(t *testing.T) {
			m := model{
				navigationState: navigationState{
					activeSection: sectionCurrent,
				},
				repositoryState: repositoryState{
					repoStatus: git.Status{
						Branch: "main", Head: "abc", Upstream: "origin/main",
						TrackingKnown: true, TrackingFresh: true,
						Tracking: map[string]git.BranchTracking{"main": {Ahead: 0, Behind: 1}},
					},
				},
				pullState: pullState{},
				status:    state.Status{Mode: mode}}
			if got := m.contextProjection(100); got.Decision != nil {
				t.Fatalf("mode %s must not receive decision context: %+v", mode, got.Decision)
			}
		})
	}
}

func TestRenderContextProjectionPreservesLegacyRecommendationWithoutDecision(t *testing.T) {
	got := renderContextProjection(ContextProjection{
		Title:          "Current",
		ActionLines:    []string{"• j/k: move"},
		Recommendation: &DivergedRecommendation{Upstream: "origin/main", LocalOnly: 2, UpstreamOnly: 3},
	}, 120, 8)
	if !strings.Contains(got, "upstream-only: 3") || strings.Contains(got, "target-only 3") {
		t.Fatalf("legacy recommendation rendering changed unexpectedly: %q", got)
	}
}

func TestRenderContextProjectionFitsLongDecisionTextAtNarrowWidths(t *testing.T) {
	decision := &BranchDecisionContext{
		CurrentRef: "feature/with-a-very-long-name", TargetRef: "origin/release/with-a-very-long-name",
		CurrentOnly: 12, TargetOnly: 8, Relation: RelationDiverged,
		ReasonLines: []string{"양쪽에 고유 commit이 있어 merge 또는 rebase로 통합합니다."},
	}
	for _, width := range []int{22, 40, 120} {
		t.Run(fmt.Sprintf("width-%d", width), func(t *testing.T) {
			got := renderContextProjection(ContextProjection{Title: "Current", Decision: decision}, width, 8)
			for _, line := range strings.Split(got, "\n") {
				if lipgloss.Width(line) > width {
					t.Fatalf("line exceeds visible width %d: %d: %q", width, lipgloss.Width(line), line)
				}
			}
		})
	}
}
