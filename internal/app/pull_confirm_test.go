package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestProjectMergeConfirmDeduplicatesRiskAndPreservesCounts(t *testing.T) {
	view, ok := projectMergeConfirm(PullImpactSnapshot{CurrentRef: "main", UpstreamRef: "origin/main", Ahead: 2, Behind: 3}, PullImpactSet{
		Valid: true, MergeSummary: "histories combine", RebaseSummary: "replay commits", MergeRisk: "Conflicts may occur.", RebaseRisk: "Conflicts may occur.",
	}, false)
	if !ok || view.CurrentBranch != "main" || view.TargetRef != "origin/main" || view.CurrentOnly != 2 || view.TargetOnly != 3 {
		t.Fatalf("unexpected projection: %#v, ok=%v", view, ok)
	}
	if view.RiskText != "Conflicts may occur." {
		t.Fatalf("risk was not deduplicated: %q", view.RiskText)
	}
	if view.Disabled {
		t.Fatal("fresh projection unexpectedly disabled")
	}
}

func TestProjectMergeConfirmRejectsUnknownImpact(t *testing.T) {
	if _, ok := projectMergeConfirm(PullImpactSnapshot{UpstreamRef: "origin/main", Ahead: 0, Behind: 2}, PullImpactSet{}, false); ok {
		t.Fatal("unknown impact was projected as a normal confirm")
	}
	if _, ok := projectMergeConfirm(PullImpactSnapshot{UpstreamRef: "origin/main", Ahead: -1, Behind: 2}, PullImpactSet{Valid: true}, false); ok {
		t.Fatal("negative counts were projected")
	}
}

func TestMergeConfirmBodyIncludesSemanticLines(t *testing.T) {
	view, _ := projectMergeConfirm(PullImpactSnapshot{CurrentRef: "main", UpstreamRef: "origin/main", Ahead: 1, Behind: 2}, PullImpactSet{
		Valid: true, MergeSummary: "histories combine", RebaseSummary: "replay commits", MergeRisk: "Conflicts may occur.",
	}, false)
	body := mergeConfirmBody(view, 80)
	for _, want := range []string{"Pull into main", "Target: origin/main", "Relation:", "Merge:", "Rebase:", "Risk:"} {
		if !contains(body, want) {
			t.Fatalf("body missing %q: %q", want, body)
		}
	}
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestPullPortPreviewDerivesDivergenceLocally(t *testing.T) {
	hash := strings.Repeat("a", 40)
	upstreamOID := strings.Repeat("b", 40)
	repoStatus := git.Status{Branch: "main", Head: hash, Upstream: "origin/main", UpstreamOID: upstreamOID, Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 2}}, TrackingKnown: true, TrackingFresh: true}
	baseline := pullSnapshotIdentity(repoStatus, 7)
	m := model{repoStatus: repoStatus, repositoryEpoch: 7, status: state.New().WithLoading("Previewing..."), activePullRequest: &pullRequest{ID: 4, Epoch: 7}}
	got, _ := handlePullPortPreview(m, pullPortPreviewMsg{result: PullPreviewResult{RequestID: 4, RepositoryEpoch: 7, Baseline: baseline, Eligible: true, Impact: PullImpact{IsFastForward: true}, Commits: []PullPreviewCommit{{Hash: hash}}}})
	modelGot := got.(model)
	if modelGot.status.Mode != state.ModeConfirm || modelGot.pullIsFastForward || modelGot.mergeConfirmView == nil {
		t.Fatalf("expected locally derived divergent confirm, got status=%+v fastForward=%v view=%#v", modelGot.status, modelGot.pullIsFastForward, modelGot.mergeConfirmView)
	}
}

func TestPullPortPreviewRejectsBaselineMismatch(t *testing.T) {
	hash := strings.Repeat("a", 40)
	repoStatus := git.Status{Branch: "main", Head: hash, Upstream: "origin/main", UpstreamOID: strings.Repeat("b", 40), Tracking: map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 2}}, TrackingKnown: true, TrackingFresh: true}
	m := model{repoStatus: repoStatus, repositoryEpoch: 7, status: state.New().WithLoading("Previewing..."), activePullRequest: &pullRequest{ID: 4, Epoch: 7}}
	bad := pullSnapshotIdentity(repoStatus, 7)
	bad.Head = strings.Repeat("c", 40)
	got, cmd := handlePullPortPreview(m, pullPortPreviewMsg{result: PullPreviewResult{RequestID: 4, RepositoryEpoch: 7, Baseline: bad, Eligible: true, Commits: []PullPreviewCommit{{Hash: hash}}}})
	if got.(model).status.Block != state.BlockStaleSnapshot || cmd == nil {
		t.Fatalf("expected stale baseline rejection, got model=%+v cmd=%v", got.(model), cmd)
	}
}

func TestMergeConfirmRendererKeepsFooterReachableAtNarrowWidths(t *testing.T) {
	view := mergeConfirmViewModel{CurrentBranch: "main", TargetRef: "origin/main", CurrentOnly: 2, TargetOnly: 3, ImpactKnown: true, MergeText: "histories combine", RebaseText: "replay commits"}
	for _, width := range []int{40, 60, 80} {
		got := renderMergeConfirmPopup(view, width)
		if !strings.Contains(got, "merge") || !strings.Contains(got, "rebase") {
			t.Fatalf("width %d lost choice footer: %q", width, got)
		}
	}
}

func TestMergeConfirmValueFitsCJKAndANSICells(t *testing.T) {
	value := "\x1b[31m非常に長いブランチ名\x1b[0m"
	got := fitMergeValue(value, 14, "Target: ")
	if lipgloss.Width("Target: "+got) > 14 {
		t.Fatalf("value exceeded terminal width: %d, %q", lipgloss.Width("Target: "+got), got)
	}
	if !strings.Contains(got, "…") {
		t.Fatalf("expected middle ellipsis for long value: %q", got)
	}
}
