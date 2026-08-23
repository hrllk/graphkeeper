package app

import (
	"errors"
	"regexp"
	"strings"
)

var rejectedPushFetchFirst = regexp.MustCompile(`(?m)^\s*!?\s*\[rejected\]\s+\S+\s+->\s+\S+\s+\(fetch first\)\s*$`)

// classifyGitError maps Git diagnostics to the application-owned error
// vocabulary. It deliberately falls back to Unknown for nil, empty, and
// unrecognized diagnostics so callers never need to interpret Git text.
func classifyGitError(err error) GitErrorCategory {
	if err == nil {
		return Unknown
	}

	var diagnostics []string
	for current := err; current != nil; current = errors.Unwrap(current) {
		if text := strings.TrimSpace(current.Error()); text != "" {
			diagnostics = append(diagnostics, strings.ToLower(text))
		}
	}
	if len(diagnostics) == 0 {
		return Unknown
	}
	diagnostic := strings.Join(diagnostics, "\n")

	switch {
	case strings.Contains(diagnostic, "permission denied"),
		strings.Contains(diagnostic, "authentication failed"),
		strings.Contains(diagnostic, "could not read from remote repository"):
		return PermissionDenied
	case strings.Contains(diagnostic, "non-fast-forward"),
		strings.Contains(diagnostic, "non fast forward"),
		rejectedPushFetchFirst.MatchString(diagnostic):
		return NonFastForward
	case strings.Contains(diagnostic, "not found"),
		strings.Contains(diagnostic, "does not exist"),
		strings.Contains(diagnostic, "no such ref"),
		strings.Contains(diagnostic, "couldn't find remote ref"),
		strings.Contains(diagnostic, "cannot find remote ref"):
		return NotFound
	case strings.Contains(diagnostic, "local changes would be overwritten"),
		strings.Contains(diagnostic, "would be overwritten by checkout"),
		strings.Contains(diagnostic, "would be overwritten by merge"),
		strings.Contains(diagnostic, "would be overwritten by rebase"):
		return DirtyWorktree
	case strings.Contains(diagnostic, "conflict"),
		strings.Contains(diagnostic, "conflicts"):
		return Conflict
	default:
		return Unknown
	}
}

func executedDiagnostic(operationErr, statusErr error) (error, GitErrorCategory) {
	effectiveErr := operationErr
	category := GitErrorCategory("")
	if effectiveErr != nil {
		category = classifyGitError(effectiveErr)
	} else if statusErr != nil {
		effectiveErr = statusErr
		category = Unknown
	}
	return effectiveErr, category
}
