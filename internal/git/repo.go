package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"hrllk/git-graph-tui/internal/telemetry"
)

type Repo struct {
	root   string
	runner Runner
}

type Status struct {
	Root                  string
	Branch                string
	Head                  string
	DefaultBranch         string
	Upstream              string
	Remote                string
	Detached              bool
	HasCommits            bool
	Graph                 []string
	GraphCommits          []GraphCommit
	Branches              []string
	LocalBranches         []string
	BranchUpstreams       map[string]string
	Tracking              map[string]BranchTracking
	RemoteBranches        []string
	Tags                  []string
	Remotes               []string
	EmptyRepo             bool
	NoUpstream            bool
	NoRemote              bool
	WorktreeDirty         bool
	MergeInProgress       bool
	RebaseInProgress      bool
	ConflictTarget        string
	ConflictTargetSubject string
	ErrorMessage          string
	LoadingReason         string
}

type Runner struct {
	Timeout time.Duration
	Dir     string
}

type GraphCommit struct {
	Graph       string
	Hash        string
	Parents     []string
	RelativeAge string
	Author      string
	Decorations []string
	Subject     string
}

type BranchTracking struct {
	Ahead  int
	Behind int
}

func Open(root string) (*Repo, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Repo{root: abs, runner: Runner{Timeout: 3 * time.Second, Dir: abs}}, nil
}

func (r *Repo) Status(ctx context.Context, limit int) (Status, error) {
	branch, err := r.currentBranch(ctx)
	if err != nil {
		return Status{}, err
	}
	head, _ := r.git(ctx, "rev-parse", "HEAD")
	upstream, _ := r.git(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	remote, _ := r.git(ctx, "remote")
	branches, branchUpstreams, tracking := r.branchMetadata(ctx)
	localBranches := branches
	remoteBranches, _ := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/remotes")
	defaultBranch := r.defaultRemoteBranch(ctx)
	tags, _ := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/tags")
	graphCommits, graphErr := r.graphCommits(ctx, localBranches, branchUpstreams, limit)
	if graphErr != nil && !isNoCommits(graphErr) {
		return Status{ErrorMessage: graphErr.Error()}, graphErr
	}
	worktreeDirty, _ := r.worktreeDirty(ctx)
	mergeInProgress := false
	rebaseInProgress := false
	conflictTarget := ""

	gitDirPath := filepath.Join(r.root, ".git")
	if gitDir, err := r.git(ctx, "rev-parse", "--git-dir"); err == nil {
		gDir := strings.TrimSpace(gitDir)
		if !filepath.IsAbs(gDir) {
			gDir = filepath.Join(r.root, gDir)
		}
		gitDirPath = gDir
	}

	// 1. 머지 상태 및 충돌 대상 검사
	mergeHeadFile := filepath.Join(gitDirPath, "MERGE_HEAD")
	if data, err := os.ReadFile(mergeHeadFile); err == nil {
		mergeInProgress = true
		conflictTarget = strings.TrimSpace(string(data))
	}

	// 2. 리베이스 상태 및 충돌 대상 검사
	rebaseMergeDir := filepath.Join(gitDirPath, "rebase-merge")
	if stat, err := os.Stat(rebaseMergeDir); err == nil && stat.IsDir() {
		rebaseInProgress = true
		if data, err := os.ReadFile(filepath.Join(rebaseMergeDir, "stopped-sha")); err == nil {
			conflictTarget = strings.TrimSpace(string(data))
		} else if data, err := os.ReadFile(filepath.Join(rebaseMergeDir, "onto")); err == nil {
			conflictTarget = strings.TrimSpace(string(data))
		}
	}
	rebaseApplyDir := filepath.Join(gitDirPath, "rebase-apply")
	if stat, err := os.Stat(rebaseApplyDir); err == nil && stat.IsDir() {
		rebaseInProgress = true
		if data, err := os.ReadFile(filepath.Join(rebaseApplyDir, "onto")); err == nil {
			conflictTarget = strings.TrimSpace(string(data))
		}
	}

	noUpstream := upstream == ""
	noRemote := remote == ""
	emptyRepo := isNoCommits(graphErr) || head == ""
	remotes, _ := r.gitLines(ctx, "remote")

	conflictTargetSubject := ""
	if conflictTarget != "" {
		if subject, err := r.git(ctx, "show", "-s", "--format=%s", conflictTarget); err == nil {
			conflictTargetSubject = strings.TrimSpace(subject)
		}
	}

	return Status{
		Root:                  r.root,
		Branch:                branch,
		Head:                  head,
		DefaultBranch:         defaultBranch,
		Upstream:              upstream,
		Remote:                strings.Join(remotes, ", "),
		Detached:              branch == "HEAD",
		HasCommits:            !emptyRepo,
		GraphCommits:          graphCommits,
		Branches:              branches,
		LocalBranches:         localBranches,
		BranchUpstreams:       branchUpstreams,
		Tracking:              tracking,
		RemoteBranches:        remoteBranches,
		Tags:                  tags,
		Remotes:               remotes,
		EmptyRepo:             emptyRepo,
		NoUpstream:            noUpstream,
		NoRemote:              noRemote,
		WorktreeDirty:         worktreeDirty,
		MergeInProgress:       mergeInProgress,
		RebaseInProgress:      rebaseInProgress,
		ConflictTarget:        conflictTarget,
		ConflictTargetSubject: conflictTargetSubject,
	}, nil
}

func parseBranchMetadataLine(line string) (branchName string, upstream string, tracking BranchTracking, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(line), "|", 3)
	if len(parts) == 0 {
		return "", "", BranchTracking{}, false
	}
	branchName = strings.TrimSpace(parts[0])
	if branchName == "" {
		return "", "", BranchTracking{}, false
	}
	if len(parts) > 1 {
		upstream = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		tracking.Ahead, tracking.Behind = parseTrackingInfo(parts[2])
	}
	return branchName, upstream, tracking, true
}

func (r *Repo) branchMetadata(ctx context.Context) ([]string, map[string]string, map[string]BranchTracking) {
	branches := make([]string, 0)
	upstreams := make(map[string]string)
	tracking := make(map[string]BranchTracking)
	lines, err := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)|%(upstream:short)|%(upstream:track)", "refs/heads")
	if err != nil {
		telemetry.Log("git", "branch_metadata_error", map[string]string{"error": err.Error()})
		return branches, upstreams, tracking
	}
	for _, line := range lines {
		branchName, upstream, track, ok := parseBranchMetadataLine(line)
		if !ok {
			continue
		}
		branches = append(branches, branchName)
		upstreams[branchName] = upstream
		if track.Ahead > 0 || track.Behind > 0 {
			tracking[branchName] = track
		}
	}
	return branches, upstreams, tracking
}

func (r *Repo) branchTracking(ctx context.Context, localBranches, remoteBranches []string) map[string]BranchTracking {
	tracking := make(map[string]BranchTracking, len(localBranches))
	lines, err := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short) %(upstream:track)", "refs/heads")
	if err != nil {
		telemetry.Log("git", "branch_tracking_error", map[string]string{"error": err.Error()})
		return tracking
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		branchName := parts[0]
		if len(parts) < 2 {
			continue
		}
		trackInfo := parts[1]
		ahead, behind := parseTrackingInfo(trackInfo)
		if ahead > 0 || behind > 0 {
			tracking[branchName] = BranchTracking{
				Ahead:  ahead,
				Behind: behind,
			}
		}
	}
	return tracking
}

func (r *Repo) branchUpstreams(ctx context.Context) map[string]string {
	upstreams := make(map[string]string)
	lines, err := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)|%(upstream:short)|%(upstream:track)", "refs/heads")
	if err != nil {
		telemetry.Log("git", "branch_upstreams_error", map[string]string{"error": err.Error()})
		return upstreams
	}
	for _, line := range lines {
		branchName, upstream, ok := parseBranchUpstreamLine(line)
		if !ok {
			continue
		}
		upstreams[branchName] = upstream
	}
	return upstreams
}

func parseBranchUpstreamLine(line string) (branchName string, upstream string, ok bool) {
	parts := strings.SplitN(line, "|", 3)
	if len(parts) == 0 {
		return "", "", false
	}
	branchName = strings.TrimSpace(parts[0])
	if branchName == "" {
		return "", "", false
	}
	if len(parts) > 1 {
		upstream = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 && strings.Contains(parts[2], "gone") {
		upstream = ""
	}
	return branchName, upstream, true
}

func parseTrackingInfo(track string) (ahead, behind int) {
	track = strings.Trim(track, "[]")
	parts := strings.Split(track, ", ")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "ahead ") {
			fmt.Sscanf(part, "ahead %d", &ahead)
		} else if strings.HasPrefix(part, "behind ") {
			fmt.Sscanf(part, "behind %d", &behind)
		}
	}
	return ahead, behind
}

func (r *Repo) graphCommits(ctx context.Context, localBranches []string, branchUpstreams map[string]string, limit int) ([]GraphCommit, error) {
	graphRefs := graphRefs(localBranches, branchUpstreams)
	if len(graphRefs) == 0 {
		return nil, nil
	}
	args := graphLogArgs(graphRefs, limit)
	lines, err := r.gitRawLines(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseGraphCommitLines(lines), nil
}

func parseGraphCommitLines(lines []string) []GraphCommit {
	commits := make([]GraphCommit, 0, len(lines))
	for _, line := range lines {
		nul := strings.IndexRune(line, '\x00')
		if nul < 0 {
			graph := strings.TrimRight(line, "\r\n")
			if strings.TrimSpace(graph) != "" {
				commits = append(commits, GraphCommit{Graph: graph})
			}
			continue
		}
		graph := line[:nul]
		parts := strings.SplitN(line[nul+1:], "\x1f", 6)
		if len(parts) < 6 {
			continue
		}
		entry := GraphCommit{Graph: graph, Hash: strings.TrimSpace(parts[0])}
		if parents := strings.TrimSpace(parts[1]); parents != "" {
			entry.Parents = strings.Fields(parents)
		}
		if relativeAge := strings.TrimSpace(parts[2]); relativeAge != "" {
			entry.RelativeAge = relativeAge
		}
		if author := strings.TrimSpace(parts[3]); author != "" {
			entry.Author = author
		}
		if decorations := strings.TrimSpace(parts[4]); decorations != "" {
			entry.Decorations = splitDecorations(decorations)
		}
		entry.Subject = strings.TrimSpace(parts[5])
		if entry.Hash != "" {
			commits = append(commits, entry)
		}
	}
	return commits
}

func graphRefs(localBranches []string, branchUpstreams map[string]string) []string {
	refs := make([]string, 0, len(localBranches)+len(branchUpstreams)+1)
	seen := make(map[string]struct{}, len(localBranches)+len(branchUpstreams)+1)
	add := func(ref string) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		if _, ok := seen[ref]; ok {
			return
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}
	for _, branch := range localBranches {
		add(branch)
		if upstream := branchUpstreams[branch]; upstream != "" {
			add(upstream)
		}
	}
	add("HEAD")
	return refs
}

func graphLogArgs(refs []string, limit int) []string {
	args := []string{
		"log",
		"--graph",
		"--decorate=short",
		"--decorate-refs=HEAD",
		"--decorate-refs=refs/heads/*",
		"--decorate-refs=refs/remotes/*",
		"--topo-order",
		"--format=%x00%H%x1f%P%x1f%ar%x1f%an%x1f%D%x1f%s",
	}
	if limit > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", limit))
	}
	args = append(args, refs...)
	return args
}

func filterRemoteBranches(remoteBranches []string) []string {
	filtered := make([]string, 0, len(remoteBranches))
	for _, branch := range remoteBranches {
		if strings.HasSuffix(branch, "/HEAD") {
			continue
		}
		filtered = append(filtered, branch)
	}
	return filtered
}

func (r *Repo) defaultRemoteBranch(ctx context.Context) string {
	out, err := r.git(ctx, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err != nil || out == "" {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(out), "origin/")
}

func (r *Repo) Fetch(ctx context.Context) error {
	_, err := r.runner.Run("fetch", "--all", "--prune", "--tags")
	return err
}

func (r *Repo) Push(ctx context.Context, branch string, force bool, setUpstream bool) (string, error) {
	args := []string{"push"}
	if force {
		args = append(args, "--force")
	}
	if setUpstream {
		args = append(args, "-u", "origin", branch)
	}
	return r.Run(args...)
}

func (r *Repo) worktreeDirty(ctx context.Context) (bool, error) {
	out, err := r.git(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (r *Repo) Divergence(ctx context.Context, left, right string) (leftOnly int, rightOnly int, err error) {
	if left == "" || right == "" {
		return 0, 0, fmt.Errorf("divergence requires two refs")
	}
	out, err := r.git(ctx, "rev-list", "--left-right", "--count", left+"..."+right)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Fields(out)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected divergence output: %q", out)
	}
	_, scanErr := fmt.Sscanf(parts[0], "%d", &leftOnly)
	if scanErr != nil {
		return 0, 0, scanErr
	}
	_, scanErr = fmt.Sscanf(parts[1], "%d", &rightOnly)
	if scanErr != nil {
		return 0, 0, scanErr
	}
	return leftOnly, rightOnly, nil
}

func (r *Repo) Run(args ...string) (string, error) {
	return r.runner.Run(args...)
}

func (r *Repo) currentBranch(ctx context.Context) (string, error) {
	out, err := r.git(ctx, "branch", "--show-current")
	if err == nil && out != "" {
		return out, nil
	}
	out, err = r.git(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return out, nil
}

func (r *Repo) git(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return out, nil
}

func (r *Repo) gitLines(ctx context.Context, args ...string) ([]string, error) {
	out, err := r.git(ctx, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	lines := strings.Split(out, "\n")
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		if s := strings.TrimSpace(line); s != "" {
			trimmed = append(trimmed, s)
		}
	}
	return trimmed, nil
}

func (r *Repo) gitRawLines(ctx context.Context, args ...string) ([]string, error) {
	out, err := r.gitRaw(ctx, args...)
	if err != nil {
		return nil, err
	}
	return splitRawLines(out), nil
}

func (r *Repo) gitRaw(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := stdout.String()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return out, nil
}

func splitRawLines(out string) []string {
	out = strings.TrimRight(out, "\n")
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		filtered = append(filtered, line)
	}
	return filtered
}

func isNoCommits(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "does not have any commits yet") ||
		strings.Contains(err.Error(), "unknown revision or path not in the working tree")
}

func splitDecorations(v string) []string {
	parts := strings.Split(v, ", ")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (r *Runner) Run(args ...string) (string, error) {
	if r.Timeout <= 0 {
		r.Timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return strings.TrimSpace(stdout.String()), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (r *Repo) MustRoot() string {
	return r.root
}
