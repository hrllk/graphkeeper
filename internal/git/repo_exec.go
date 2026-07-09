package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

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

func (r *Repo) PushTag(ctx context.Context, tag string) (string, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", fmt.Errorf("tag is empty")
	}
	return r.Run("push", "origin", tag)
}

func (r *Repo) DeleteBranch(ctx context.Context, branch string) (string, error) {
	return r.Run("branch", "-D", branch)
}

func (r *Repo) DeleteTag(ctx context.Context, name string) (string, error) {
	return r.Run("tag", "-d", name)
}

func (r *Repo) DeleteRemoteBranch(ctx context.Context, remote, branch string) (string, error) {
	return r.Run("push", remote, "--delete", branch)
}

func (r *Repo) CreateTag(ctx context.Context, name, target string) error {
	name = strings.TrimSpace(name)
	target = strings.TrimSpace(target)
	if name == "" {
		return fmt.Errorf("tag name is empty")
	}
	if target == "" {
		return fmt.Errorf("tag target is empty")
	}
	_, err := r.git(ctx, "tag", name, target)
	return err
}

func (r *Repo) FetchTags(ctx context.Context) error {
	_, err := r.git(ctx, "fetch", "origin", "--tags", "--prune")
	return err
}

func (r *Repo) StashAll(ctx context.Context, message string) error {
	_, err := r.git(ctx, "stash", "push", "--include-untracked", "-m", message)
	return err
}

func (r *Repo) StashPop(ctx context.Context, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("stash ref is empty")
	}
	_, err := r.git(ctx, "stash", "pop", ref)
	return err
}

func (r *Repo) CleanWorkingTree(ctx context.Context, includeIgnored bool) error {
	if _, err := r.git(ctx, "reset", "--hard"); err != nil {
		return err
	}
	args := []string{"clean", "-fd"}
	if includeIgnored {
		args = append(args, "-x")
	}
	_, err := r.git(ctx, args...)
	return err
}

func (r *Repo) worktreeDirty(ctx context.Context) (bool, error) {
	out, err := r.git(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (r *Repo) Stashes(ctx context.Context) ([]StashEntry, error) {
	lines, err := r.gitLines(ctx, "stash", "list", "--format=%gd%x1f%H%x1f%P%x1f%gs")
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, nil
	}
	entries := make([]StashEntry, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "\x1f", 4)
		if len(parts) < 4 {
			continue
		}
		parents := strings.Fields(parts[2])
		baseHash := ""
		if len(parents) > 0 {
			baseHash = parents[0]
		}
		entries = append(entries, StashEntry{
			Ref:      strings.TrimSpace(parts[0]),
			Hash:     strings.TrimSpace(parts[1]),
			BaseHash: baseHash,
			Subject:  strings.TrimSpace(parts[3]),
		})
	}
	return entries, nil
}

func (r *Repo) TagEntries(ctx context.Context) ([]TagEntry, error) {
	entries, err := r.LocalTagEntries(ctx)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	originTags, _ := r.OriginTagSet(ctx)
	for i := range entries {
		entries[i].OriginKnown = true
		entries[i].OnOrigin = originTags[entries[i].Name]
	}
	return entries, nil
}

func (r *Repo) LocalTagEntries(ctx context.Context) ([]TagEntry, error) {
	names, err := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/tags")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, nil
	}
	entries := make([]TagEntry, 0, len(names))
	for _, name := range names {
		target, err := r.git(ctx, "rev-parse", "--verify", name+"^{commit}")
		if err != nil {
			continue
		}
		subject, err := r.git(ctx, "show", "-s", "--format=%s", target)
		if err != nil {
			continue
		}
		relativeAge, err := r.git(ctx, "show", "-s", "--format=%cr", target)
		if err != nil {
			continue
		}
		commitUnixText, err := r.git(ctx, "show", "-s", "--format=%ct", target)
		if err != nil {
			continue
		}
		commitUnix, err := strconv.ParseInt(strings.TrimSpace(commitUnixText), 10, 64)
		if err != nil {
			continue
		}
		entries = append(entries, TagEntry{
			Name:        strings.TrimSpace(name),
			CommitHash:  strings.TrimSpace(target),
			Subject:     strings.TrimSpace(subject),
			RelativeAge: strings.TrimSpace(relativeAge),
			CommitUnix:  commitUnix,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].CommitUnix != entries[j].CommitUnix {
			return entries[i].CommitUnix > entries[j].CommitUnix
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func (r *Repo) OriginTagSet(ctx context.Context) (map[string]bool, error) {
	lines, err := r.gitLines(ctx, "ls-remote", "--tags", "origin")
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, nil
	}
	tags := make(map[string]bool, len(lines))
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ref := strings.TrimPrefix(parts[1], "refs/tags/")
		ref = strings.TrimSuffix(ref, "^{}")
		if ref == "" {
			continue
		}
		tags[ref] = true
	}
	return tags, nil
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

func (r *Repo) MergeBase(ctx context.Context, left, right string) (string, error) {
	if left == "" || right == "" {
		return "", fmt.Errorf("merge base requires two refs")
	}
	out, err := r.git(ctx, "merge-base", left, right)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
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
