package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hrllk/graphkeeper/internal/telemetry"
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

type StashEntry struct {
	Ref      string
	Hash     string
	BaseHash string
	Subject  string
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

func (r *Repo) defaultRemoteBranch(ctx context.Context) string {
	out, err := r.git(ctx, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err != nil || out == "" {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(out), "origin/")
}
