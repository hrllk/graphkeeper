package git

import (
	"context"
	"fmt"
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
	UpstreamOID           string
	Remote                string
	Detached              bool
	HasCommits            bool
	Graph                 []string
	GraphCommits          []GraphCommit
	Branches              []string
	LocalBranches         []string
	LocalBranchesKnown    bool
	LocalBranchesFresh    bool
	LocalBranchesError    string
	BranchUpstreams       map[string]string
	Tracking              map[string]BranchTracking
	TrackingKnown         bool
	TrackingFresh         bool
	TrackingError         string
	UpstreamGone          bool
	RemoteBranches        []string
	Tags                  []string
	TagEntries            []TagEntry
	TagEntriesLoaded      bool
	TagProvenanceLoaded   bool
	TagSyncSummary        string
	LastFetchAt           time.Time
	RemoteSyncSummary     string
	Remotes               []string
	EmptyRepo             bool
	NoUpstream            bool
	NoRemote              bool
	WorktreeDirty         bool
	MergeInProgress       bool
	RebaseInProgress      bool
	CherryPickInProgress  bool
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
	Tags        []string
}

type TagEntry struct {
	Name        string
	CommitHash  string
	Subject     string
	RelativeAge string
	CommitUnix  int64
	Tagger      string
	TaggedAt    time.Time
	Message     string
	Annotated   bool
	OriginKnown bool
	OnOrigin    bool
}

type StashEntry struct {
	Ref      string
	Hash     string
	BaseHash string
	Subject  string
}

type CommitInspection struct {
	Hash, Subject, Author, Message, Parent string
	IsRoot                                 bool
	Parents                                []string
	Files                                  []CommitDiffFile
}

type CommitDiffFile struct {
	ID, Path, OldPath    string
	Status               string
	Additions, Deletions int
	Binary               bool
}

type CommitDiff struct {
	FileID        string
	Lines         []string
	Rows          []DiffRow
	HasMore       bool
	PartialReason string
	NextStartLine int
}

type DiffRow struct {
	Kind                   string
	OldLine, NewLine       int
	From, To               string
	FromPresent, ToPresent bool
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
	head, headErr := r.git(ctx, "rev-parse", "HEAD")
	upstream, upstreamErr := r.git(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	upstreamOID, upstreamOIDErr := r.git(ctx, "rev-parse", "--verify", "@{upstream}")
	remote, remoteErr := r.git(ctx, "remote")
	branches, branchUpstreams, tracking, trackingKnown, trackingFresh, trackingError, upstreamGone := r.branchMetadata(ctx, branch)
	localBranchesKnown := trackingError == ""
	localBranchesFresh := localBranchesKnown
	if headErr != nil || remoteErr != nil {
		trackingFresh = false
		if trackingError == "" {
			if headErr != nil {
				trackingError = headErr.Error()
			} else {
				trackingError = remoteErr.Error()
			}
		}
	}
	if upstreamErr != nil && !isExpectedNoUpstreamError(upstreamErr) {
		trackingFresh = false
		if trackingError == "" {
			trackingError = upstreamErr.Error()
		}
	}
	if upstreamErr == nil && upstreamOIDErr != nil {
		trackingFresh = false
		if trackingError == "" {
			trackingError = upstreamOIDErr.Error()
		}
	}
	localBranches := branches
	remoteBranches, _ := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/remotes")
	defaultBranch := r.defaultRemoteBranch(ctx)
	graphCommits, graphErr := r.graphCommits(ctx, localBranches, branchUpstreams, limit)
	if graphErr != nil && !isNoCommits(graphErr) {
		return Status{ErrorMessage: graphErr.Error()}, graphErr
	}
	worktreeDirty, _ := r.worktreeDirty(ctx)
	lastFetchAt, _ := r.fetchHeadModTime(ctx)
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

	cherryPickHeadFile := filepath.Join(gitDirPath, "CHERRY_PICK_HEAD")
	cherryPickInProgress := false
	if data, err := os.ReadFile(cherryPickHeadFile); err == nil {
		cherryPickInProgress = true
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

	noUpstream := upstream == "" && (upstreamErr == nil || isExpectedNoUpstreamError(upstreamErr))
	noRemote := remote == ""
	emptyRepo := isNoCommits(graphErr) || head == ""
	remotes, remotesErr := r.gitLines(ctx, "remote")
	if remotesErr != nil {
		trackingFresh = false
		if trackingError == "" {
			trackingError = remotesErr.Error()
		}
	}

	conflictTargetSubject := ""
	if conflictTarget != "" {
		if subject, err := r.git(ctx, "show", "-s", "--format=%s", conflictTarget); err == nil {
			conflictTargetSubject = strings.TrimSpace(subject)
		}
	}

	return Status{
		Root:               r.root,
		Branch:             branch,
		Head:               head,
		DefaultBranch:      defaultBranch,
		Upstream:           upstream,
		UpstreamOID:        strings.TrimSpace(upstreamOID),
		Remote:             strings.Join(remotes, ", "),
		Detached:           branch == "HEAD",
		HasCommits:         !emptyRepo,
		GraphCommits:       graphCommits,
		Branches:           branches,
		LocalBranches:      localBranches,
		LocalBranchesKnown: localBranchesKnown,
		LocalBranchesFresh: localBranchesFresh,
		LocalBranchesError: func() string {
			if localBranchesKnown {
				return ""
			}
			return trackingError
		}(),
		BranchUpstreams:       branchUpstreams,
		Tracking:              tracking,
		TrackingKnown:         trackingKnown,
		TrackingFresh:         trackingFresh,
		TrackingError:         trackingError,
		UpstreamGone:          upstreamGone,
		RemoteBranches:        remoteBranches,
		Tags:                  nil,
		TagEntries:            nil,
		LastFetchAt:           lastFetchAt,
		RemoteSyncSummary:     remoteSyncSummary(branch, tracking, trackingKnown, remote, noRemote, noUpstream, branch == "HEAD"),
		Remotes:               remotes,
		EmptyRepo:             emptyRepo,
		NoUpstream:            noUpstream,
		NoRemote:              noRemote,
		WorktreeDirty:         worktreeDirty,
		MergeInProgress:       mergeInProgress,
		RebaseInProgress:      rebaseInProgress,
		CherryPickInProgress:  cherryPickInProgress,
		ConflictTarget:        conflictTarget,
		ConflictTargetSubject: conflictTargetSubject,
	}, nil
}

func isExpectedNoUpstreamError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no upstream configured") ||
		strings.Contains(message, "has no upstream branch")
}

func (r *Repo) fetchHeadModTime(ctx context.Context) (time.Time, bool) {
	path, err := r.git(ctx, "rev-parse", "--git-path", "FETCH_HEAD")
	if err != nil {
		return time.Time{}, false
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return time.Time{}, false
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(r.root, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

func remoteSyncSummary(branch string, tracking map[string]BranchTracking, trackingKnown bool, remote string, noRemote, noUpstream, detached bool) string {
	switch {
	case branch == "":
		return ""
	case noRemote:
		return "no remote"
	case noUpstream:
		return "no upstream"
	case detached:
		return "detached"
	case !trackingKnown:
		return "tracking unknown"
	}
	track := tracking[branch]
	switch {
	case track.Ahead == 0 && track.Behind == 0:
		return "synced"
	case track.Ahead > 0 && track.Behind == 0:
		return fmt.Sprintf("ahead %d", track.Ahead)
	case track.Ahead == 0 && track.Behind > 0:
		return fmt.Sprintf("behind %d", track.Behind)
	default:
		_ = remote
		return fmt.Sprintf("diverged (%d ahead, %d behind)", track.Ahead, track.Behind)
	}
}

func (r *Repo) branchMetadata(ctx context.Context, currentBranch string) ([]string, map[string]string, map[string]BranchTracking, bool, bool, string, bool) {
	branches := make([]string, 0)
	upstreams := make(map[string]string)
	tracking := make(map[string]BranchTracking)
	lines, err := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)|%(upstream:short)|%(upstream:track)", "refs/heads")
	if err != nil {
		telemetry.Log("git", "branch_metadata_error", map[string]string{"error": err.Error()})
		return branches, upstreams, tracking, false, false, err.Error(), false
	}
	trackingKnown, upstreamGone := false, false
	for _, line := range lines {
		// Git emits an empty track value for an upstream with no divergence.
		// Normalize that valid output before the strict metadata parser sees it.
		parts := strings.Split(line, "|")
		if len(parts) == 3 && strings.TrimSpace(parts[1]) != "" && strings.TrimSpace(parts[2]) == "" {
			line = strings.Join([]string{parts[0], parts[1], "[ahead 0, behind 0]"}, "|")
		}
		branchName, upstream, track, ok := parseBranchMetadataLine(line)
		if !ok {
			return branches, upstreams, tracking, false, false, "malformed branch metadata", false
		}
		branches = append(branches, branchName)
		upstreams[branchName] = upstream
		if branchName == currentBranch {
			fields := strings.Split(line, "|")
			upstreamGone = upstream == "" && len(fields) == 3 && strings.TrimSpace(fields[2]) == "[gone]"
			trackingKnown = upstream != ""
		}
		if upstream != "" {
			tracking[branchName] = track
		}
	}
	return branches, upstreams, tracking, trackingKnown && !upstreamGone, true, "", upstreamGone
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
