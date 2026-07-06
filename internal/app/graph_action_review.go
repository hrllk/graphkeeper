package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func buildGraphActionReviewStatus(action state.Action, rs git.Status, target, base string, currentOnly, targetOnly int) state.Status {
	status := state.New().WithReview(action, "Branch has diverged", buildGraphActionReviewDetail(action, rs, target, base, currentOnly, targetOnly))
	status.Selected = target
	return status
}

func buildGraphActionConfirmStatus(action state.Action, rs git.Status, target string) state.Status {
	titleMsg := "Merge into current branch?"
	if action == state.ActionRebase {
		titleMsg = "Rebase onto this commit?"
	}
	detailMsg := ""
	if action == state.ActionMerge {
		detailMsg = "This will merge commit " + shorten(target, 7) + " into " + rs.Branch + ".\nA merge commit will be created if histories have diverged.\n\nContinue? (y: yes  •  n: no)"
	} else {
		detailMsg = "This will rebase " + rs.Branch + " onto commit " + shorten(target, 7) + ".\nLocal commits will be replayed on top of the target.\n\n⚠️ Conflicts may occur during rebase.\n\nContinue? (y: yes  •  n: no)"
	}
	status := state.New().WithConfirm(action, titleMsg, detailMsg)
	status.Title = titleMsg
	status.Selected = target
	return status
}

func buildGraphActionReviewDetail(action state.Action, rs git.Status, target, base string, currentOnly, targetOnly int) string {
	subtitle := "Review before continuing."
	switch action {
	case state.ActionMerge:
		subtitle = "Review before merge."
	case state.ActionRebase:
		subtitle = "Review before rebase."
	}

	return strings.Join([]string{
		centerReviewLine(subtitle),
		"",
		buildGraphActionReviewDiagram(rs, target, base, currentOnly, targetOnly),
		"",
		centerReviewLine("y: continue  •  n: cancel"),
	}, "\n")
}

func centerReviewLine(text string) string {
	return reviewFooter.Render(text)
}

func reviewCurrentBranch(rs git.Status) string {
	branch := strings.TrimSpace(rs.Branch)
	if branch != "" {
		return branch
	}
	if rs.Detached && rs.Head != "" {
		return "HEAD"
	}
	return ""
}

func reviewBranchNameForHash(rs git.Status, hash string) string {
	if hash == "" {
		return ""
	}
	rows := graphRows(rs)
	idx := findGraphRowByHash(rows, hash)
	if idx < 0 {
		return ""
	}
	return reviewBranchNameFromDecorations(rows[idx].Commit.Decorations, rs.LocalBranches)
}

func reviewBranchNameFromDecorations(decorations []string, localBranches []string) string {
	if len(decorations) == 0 {
		return ""
	}
	localSet := make(map[string]struct{}, len(localBranches))
	for _, branch := range localBranches {
		localSet[strings.TrimSpace(branch)] = struct{}{}
	}
	branches := make(map[string]*branchState)
	headBranch := ""

	addBranch := func(name string) *branchState {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil
		}
		state, ok := branches[name]
		if !ok {
			state = &branchState{}
			branches[name] = state
		}
		return state
	}

	for _, decoration := range decorations {
		decoration = strings.TrimSpace(decoration)
		if decoration == "" {
			continue
		}
		switch {
		case strings.HasPrefix(decoration, "HEAD -> "):
			name := strings.TrimPrefix(decoration, "HEAD -> ")
			if state := addBranch(name); state != nil {
				state.local = true
				headBranch = strings.TrimSpace(name)
			}
		case strings.HasPrefix(decoration, "origin/HEAD -> origin/"):
			if state := addBranch(strings.TrimPrefix(decoration, "origin/HEAD -> origin/")); state != nil {
				state.remote = true
			}
		case decoration == "origin/HEAD":
			if state := addBranch("HEAD"); state != nil {
				state.remote = true
			}
		case strings.HasPrefix(decoration, "origin/"):
			name := strings.TrimPrefix(decoration, "origin/")
			if name == "HEAD" {
				continue
			}
			if state := addBranch(name); state != nil {
				state.remote = true
			}
		case strings.HasPrefix(decoration, "tag: "):
			continue
		default:
			if _, ok := localSet[decoration]; ok {
				if state := addBranch(decoration); state != nil {
					state.local = true
				}
			} else if !strings.Contains(decoration, "/") {
				if state := addBranch(decoration); state != nil {
					state.local = true
				}
			}
		}
	}

	name := pickCompactBranchName(branches, headBranch)
	if name == "" || name == "HEAD" {
		return ""
	}
	return name
}

func buildGraphActionReviewDiagram(rs git.Status, target, base string, currentOnly, targetOnly int) string {
	currentBranch := reviewCurrentBranch(rs)
	targetBranch := reviewBranchNameForHash(rs, target)
	if targetBranch == "" {
		targetBranch = "requested branch"
	}
	lines := []string{
		"*    " + reviewCurrent.Render(shorten(rs.Head, 7)) + " " + reviewCurrent.Render("CURRENT") + " " + reviewCount.Render(fmt.Sprintf("+%d", currentOnly)) + reviewBranchSuffix(currentBranch),
		"|",
		"| *  " + reviewTarget.Render(shorten(target, 7)) + " " + reviewTarget.Render("TARGET") + " " + reviewCount.Render(fmt.Sprintf("+%d", targetOnly)) + reviewBranchSuffix(targetBranch),
		"| |",
		"|/",
		"*    " + reviewBase.Render(shorten(base, 7)) + " " + reviewBase.Render("BASE"),
	}
	return strings.Join(padReviewDiagramLines(lines), "\n")
}

func reviewBranchSuffix(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return ""
	}
	return " (" + reviewBranch.Render(branch) + ")"
}

func padReviewDiagramLines(lines []string) []string {
	maxWidth := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > maxWidth {
			maxWidth = w
		}
	}
	if maxWidth == 0 {
		return lines
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = padRight(line, maxWidth)
	}
	return out
}
