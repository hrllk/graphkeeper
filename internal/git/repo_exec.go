package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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
