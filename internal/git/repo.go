package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Repo struct {
	root   string
	runner Runner
}

type Status struct {
	Root           string
	Branch         string
	Head           string
	DefaultBranch  string
	Upstream       string
	Remote         string
	Detached       bool
	HasCommits     bool
	Graph          []string
	GraphCommits   []GraphCommit
	Branches       []string
	LocalBranches  []string
	RemoteBranches []string
	Tags           []string
	Remotes        []string
	EmptyRepo      bool
	NoUpstream     bool
	NoRemote       bool
	WorktreeDirty  bool
	ErrorMessage   string
	LoadingReason  string
}

type Runner struct {
	Timeout time.Duration
}

type GraphCommit struct {
	Hash        string
	Parents     []string
	Decorations []string
}

func Open(root string) (*Repo, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Repo{root: abs, runner: Runner{Timeout: 3 * time.Second}}, nil
}

func (r *Repo) Status(ctx context.Context) (Status, error) {
	branch, err := r.currentBranch(ctx)
	if err != nil {
		return Status{}, err
	}
	head, _ := r.git(ctx, "rev-parse", "HEAD")
	upstream, _ := r.git(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	remote, _ := r.git(ctx, "remote")
	branches, _ := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	localBranches, _ := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	remoteBranches, _ := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/remotes")
	filteredRemoteBranches := filterRemoteBranches(remoteBranches)
	defaultBranch := r.defaultRemoteBranch(ctx)
	tags, _ := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/tags")
	graphCommits, graphErr := r.graphCommits(ctx, localBranches, filteredRemoteBranches)
	if graphErr != nil && !isNoCommits(graphErr) {
		return Status{ErrorMessage: graphErr.Error()}, graphErr
	}
	worktreeDirty, _ := r.worktreeDirty(ctx)

	noUpstream := upstream == ""
	noRemote := remote == ""
	emptyRepo := isNoCommits(graphErr) || head == ""
	remotes, _ := r.gitLines(ctx, "remote")

	return Status{
		Root:           r.root,
		Branch:         branch,
		Head:           head,
		DefaultBranch:  defaultBranch,
		Upstream:       upstream,
		Remote:         strings.Join(remotes, ", "),
		Detached:       branch == "HEAD",
		HasCommits:     !emptyRepo,
		GraphCommits:   graphCommits,
		Branches:       branches,
		LocalBranches:  localBranches,
		RemoteBranches: filteredRemoteBranches,
		Tags:           tags,
		Remotes:        remotes,
		EmptyRepo:      emptyRepo,
		NoUpstream:     noUpstream,
		NoRemote:       noRemote,
		WorktreeDirty:  worktreeDirty,
	}, nil
}

func (r *Repo) graphCommits(ctx context.Context, localBranches, remoteBranches []string) ([]GraphCommit, error) {
	refs := graphRefs(localBranches, remoteBranches)
	if len(refs) == 0 {
		return nil, nil
	}
	args := append([]string{"log", "--format=%H%x1f%P%x1f%D", "--topo-order"}, refs...)
	lines, err := r.gitLines(ctx, args...)
	if err != nil {
		return nil, err
	}
	commits := make([]GraphCommit, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\x1f")
		if len(parts) < 3 {
			continue
		}
		entry := GraphCommit{Hash: strings.TrimSpace(parts[0])}
		if parents := strings.TrimSpace(parts[1]); parents != "" {
			entry.Parents = strings.Fields(parents)
		}
		if decorations := strings.TrimSpace(parts[2]); decorations != "" {
			entry.Decorations = splitDecorations(decorations)
		}
		if entry.Hash != "" {
			commits = append(commits, entry)
		}
	}
	return commits, nil
}

func graphRefs(localBranches, remoteBranches []string) []string {
	remoteSet := make(map[string]struct{}, len(remoteBranches))
	for _, branch := range remoteBranches {
		remoteSet[branch] = struct{}{}
	}
	refs := make([]string, 0, len(localBranches)*2)
	for _, branch := range localBranches {
		refs = append(refs, "refs/heads/"+branch)
		originBranch := "origin/" + branch
		if _, ok := remoteSet[originBranch]; ok {
			refs = append(refs, "refs/remotes/"+originBranch)
		}
	}
	return refs
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
