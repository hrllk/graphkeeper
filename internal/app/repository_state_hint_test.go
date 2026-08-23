package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func TestRepositoryStateHintPriority(t *testing.T) {
	tests := []struct {
		name    string
		status  git.Status
		blocked bool
		want    string
	}{
		{name: "load error", status: git.Status{Root: "/repo"}, want: "Repository unavailable"},
		{name: "not a repository", want: "Not a Git repository"},
		{name: "merge wins", status: git.Status{Root: "/repo", MergeInProgress: true, RebaseInProgress: true}, want: "Merge in progress"},
		{name: "cherry pick", status: git.Status{Root: "/repo", CherryPickInProgress: true}, want: "Cherry-pick in progress"},
		{name: "empty repository", status: git.Status{Root: "/repo", EmptyRepo: true, NoRemote: true, NoUpstream: true}, want: "No commits yet"},
		{name: "both missing", status: git.Status{Root: "/repo", NoRemote: true, NoUpstream: true}, want: "No remote or upstream"},
		{name: "diverged", status: git.Status{Root: "/repo", Branch: "기능", Upstream: "origin/기능", Tracking: map[string]git.BranchTracking{"기능": {Ahead: 1, Behind: 2}}}, want: "Diverged · 기능 ↔ origin/기능"},
		{name: "blocked dirty", status: git.Status{Root: "/repo", WorktreeDirty: true}, blocked: true, want: "Working tree is dirty"},
		{name: "normal", status: git.Status{Root: "/repo"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadErr := error(nil)
			if tt.name == "load error" {
				loadErr = errRepositoryUnavailable
			}
			if got := repositoryStateHint(tt.status, tt.blocked, loadErr); got != tt.want {
				t.Fatalf("repositoryStateHint() = %q, want %q", got, tt.want)
			}
		})
	}
}

var errRepositoryUnavailable = testError("repository unavailable")

type testError string

func (e testError) Error() string { return string(e) }

func TestRenderGraphProjectionIncludesHintWithoutExceedingWidth(t *testing.T) {
	p := GraphProjection{
		Rows:      []graphRow{{Commit: graphNode{Hash: "abc123", Subject: "subject"}, Graph: "*"}},
		PageSize:  1,
		StateHint: "Diverged · 매우-긴-branch ↔ origin/매우-긴-branch",
	}
	got := renderGraphProjection(p, 24, 4)
	lines := strings.Split(got, "\n")
	if !strings.Contains(got, "Diverged") {
		t.Fatalf("expected state hint in graph output: %q", got)
	}
	for _, line := range lines {
		if lipgloss.Width(line) > 24 {
			t.Fatalf("line exceeds visible width: %d: %q", lipgloss.Width(line), line)
		}
	}
}

func TestGraphPageSizeAccountsForStateHint(t *testing.T) {
	m := model{
		repositoryState: repositoryState{repoStatus: git.Status{Root: "/repo"}},
		pullState:       pullState{}, status: state.New().WithBrowse()}
	rows := []graphRow{{Commit: graphNode{Hash: "1"}, Graph: "*"}, {Commit: graphNode{Hash: "2"}, Graph: "*"}, {Commit: graphNode{Hash: "3"}, Graph: "*"}}
	withoutHint := graphPageSizeForRowsWithHint(&m, rows, 0, 5, false)
	withHint := graphPageSizeForRowsWithHint(&m, rows, 0, 5, true)
	if withHint >= withoutHint {
		t.Fatalf("hint should reduce topology page size: without=%d with=%d", withoutHint, withHint)
	}
}

func TestScreenProjectionDoesNotShowRepositoryHintBeforeInitialSnapshot(t *testing.T) {
	m := model{
		repositoryState: repositoryState{repoStatus: git.Status{}},
		pullState:       pullState{}, status: state.New().WithLoading("Loading...")}

	projection := m.screenProjection(80, 8)
	if projection.Graph.StateHint != "" {
		t.Fatalf("initial loading showed repository hint %q", projection.Graph.StateHint)
	}
}
