package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenReturnsAbsoluteRoot(t *testing.T) {
	repo, err := Open(".")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if repo.MustRoot() == "" {
		t.Fatal("expected root to be set")
	}
	if !filepath.IsAbs(repo.MustRoot()) {
		t.Fatalf("expected absolute root, got %q", repo.MustRoot())
	}
}

func TestRunnerRejectsUnknownCommand(t *testing.T) {
	r := &Runner{}
	_, err := r.Run("not-a-real-git-command")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsNoCommits(t *testing.T) {
	err := os.ErrNotExist
	if isNoCommits(err) {
		t.Fatal("expected unrelated error to not be treated as no-commits")
	}
}

func TestFilterRemoteBranchesDropsSymbolicHead(t *testing.T) {
	got := filterRemoteBranches([]string{"origin/HEAD", "origin/main", "origin/tmp3"})
	if len(got) != 2 || got[0] != "origin/main" || got[1] != "origin/tmp3" {
		t.Fatalf("unexpected filtered remote branches: %v", got)
	}
}

func TestGraphLogArgsUsesLocalBranchesOnly(t *testing.T) {
	got := graphLogArgs([]string{"main", "origin/main"}, 40)
	wantContains := []string{
		"log",
		"--graph",
		"--decorate=short",
		"--decorate-refs=HEAD",
		"--decorate-refs=refs/heads/*",
		"--decorate-refs=refs/remotes/*",
		"--topo-order",
		"--format=%x00%H%x1f%P%x1f%ar%x1f%an%x1f%D%x1f%s",
		"--max-count=40",
		"main",
		"origin/main",
	}
	for _, want := range wantContains {
		found := false
		for _, arg := range got {
			if arg == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected graph log args to contain %q, got %v", want, got)
		}
	}
	for _, arg := range got {
		if arg == "--all" || arg == "--branches" {
			t.Fatalf("expected graph log args to exclude broad ref selectors, got %v", got)
		}
	}
}

func TestGraphRefsIncludesTrackedUpstreams(t *testing.T) {
	got := graphRefs([]string{"main", "develop", "tmp1"}, map[string]string{"main": "origin/main", "develop": "", "tmp1": "origin/tmp1"})
	want := []string{"main", "origin/main", "develop", "tmp1", "origin/tmp1", "HEAD"}
	if len(got) != len(want) {
		t.Fatalf("expected %d refs, got %v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected refs at %d: got %q want %q (all=%v)", i, got[i], want[i], got)
		}
	}
}

func TestGraphRefsFallsBackToHead(t *testing.T) {
	got := graphRefs(nil, nil)
	if len(got) != 1 || got[0] != "HEAD" {
		t.Fatalf("expected HEAD fallback ref, got %v", got)
	}
}

func TestParseBranchUpstreamLineDropsGoneUpstream(t *testing.T) {
	branch, upstream, ok := parseBranchUpstreamLine("tmp3|origin/tmp3|[gone]")
	if !ok {
		t.Fatal("expected branch upstream line to parse")
	}
	if branch != "tmp3" {
		t.Fatalf("unexpected branch name: %q", branch)
	}
	if upstream != "" {
		t.Fatalf("expected gone upstream to be dropped, got %q", upstream)
	}
}

func TestParseBranchUpstreamLineKeepsValidUpstream(t *testing.T) {
	branch, upstream, ok := parseBranchUpstreamLine("develop|origin/develop|")
	if !ok {
		t.Fatal("expected branch upstream line to parse")
	}
	if branch != "develop" || upstream != "origin/develop" {
		t.Fatalf("unexpected parsed upstream: branch=%q upstream=%q", branch, upstream)
	}
}

func TestParseBranchMetadataLineDropsGoneUpstream(t *testing.T) {
	branch, upstream, tracking, ok := parseBranchMetadataLine("tmp3|origin/tmp3|[gone]")
	if !ok {
		t.Fatal("expected branch metadata line to parse")
	}
	if branch != "tmp3" {
		t.Fatalf("unexpected branch name: %q", branch)
	}
	if upstream != "" {
		t.Fatalf("expected gone upstream to be dropped, got %q", upstream)
	}
	if tracking.Ahead != 0 || tracking.Behind != 0 {
		t.Fatalf("expected no tracking counts for gone upstream, got %+v", tracking)
	}
}

func TestStatusIgnoresGoneUpstream(t *testing.T) {
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	work := filepath.Join(base, "work")
	clone := filepath.Join(base, "clone")

	runGitGit(t, base, "init", "--bare", "remote.git")
	runGitGit(t, base, "init", "-b", "main", "work")
	configGitUser(t, work)
	writeGitFile(t, work, "file.txt", "base\n")
	runGitGit(t, work, "add", "file.txt")
	runGitGit(t, work, "commit", "-m", "initial")
	runGitGit(t, work, "remote", "add", "origin", remote)
	runGitGit(t, work, "push", "-u", "origin", "main")
	runGitGit(t, work, "checkout", "-b", "tmp3")
	writeGitFile(t, work, "tmp3.txt", "tmp3\n")
	runGitGit(t, work, "add", "tmp3.txt")
	runGitGit(t, work, "commit", "-m", "tmp3")
	runGitGit(t, work, "push", "-u", "origin", "tmp3")

	runGitGit(t, base, "clone", remote, "clone")
	configGitUser(t, clone)
	runGitGit(t, clone, "checkout", "-b", "tmp3", "origin/tmp3")
	runGitGit(t, clone, "push", "origin", "--delete", "tmp3")
	runGitGit(t, work, "fetch", "--prune", "origin")

	repo, err := Open(work)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	status, err := repo.Status(context.Background(), 0)
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.ErrorMessage != "" {
		t.Fatalf("expected no graph error, got %q", status.ErrorMessage)
	}
	if len(status.GraphCommits) == 0 {
		t.Fatalf("expected graph commits to remain available, got %+v", status)
	}
	if got := status.BranchUpstreams["tmp3"]; got != "" {
		t.Fatalf("expected gone upstream to be normalized away, got %q", got)
	}
}

func runGitGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out.String())
	}
	return strings.TrimSpace(out.String())
}

func configGitUser(t *testing.T, dir string) {
	t.Helper()
	runGitGit(t, dir, "config", "user.name", "Test User")
	runGitGit(t, dir, "config", "user.email", "test@example.com")
}

func writeGitFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s failed: %v", path, err)
	}
}

func TestSplitRawLinesPreservesGraphWhitespace(t *testing.T) {
	got := splitRawLines("  * \x00hash\x1fparents\x1fage\x1fauthor\x1f\x1fsubject\n |/  \n")
	want := []string{"  * \x00hash\x1fparents\x1fage\x1fauthor\x1f\x1fsubject", " |/  "}
	if len(got) != len(want) {
		t.Fatalf("expected %d lines, got %v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected raw line %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestParseGraphCommitLinesPreservesGraphPrefix(t *testing.T) {
	got := parseGraphCommitLines([]string{
		"  * \x00abc\x1fparent\x1f5 minutes ago\x1fdev\x1fHEAD -> main\x1fsubject",
		" |/",
	})
	if len(got) != 2 {
		t.Fatalf("expected graph commit and connector line, got %v", got)
	}
	if got[0].Graph != "  * " {
		t.Fatalf("expected leading graph spaces to be preserved, got %q", got[0].Graph)
	}
	if got[1].Graph != " |/" || got[1].Hash != "" {
		t.Fatalf("expected connector graph line to be preserved, got %+v", got[1])
	}
}

func TestParseTrackingInfo(t *testing.T) {
	tests := []struct {
		input  string
		ahead  int
		behind int
	}{
		{"[ahead 1, behind 2]", 1, 2},
		{"[ahead 5]", 5, 0},
		{"[behind 3]", 0, 3},
		{"[gone]", 0, 0},
		{"", 0, 0},
	}

	for _, tc := range tests {
		a, b := parseTrackingInfo(tc.input)
		if a != tc.ahead || b != tc.behind {
			t.Errorf("parseTrackingInfo(%q) = (%d, %d); want (%d, %d)", tc.input, a, b, tc.ahead, tc.behind)
		}
	}
}
