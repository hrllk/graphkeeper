package app

import (
	"fmt"
	"strings"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/state"
)

func buildGraphActionReviewStatus(action state.Action, rs git.Status, target, base string, currentOnly, targetOnly int) state.Status {
	status := state.New().WithReview(action, "분기점 확인하기", buildGraphActionReviewDetail(rs, target, base, currentOnly, targetOnly))
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

func buildGraphActionReviewDetail(rs git.Status, target, base string, currentOnly, targetOnly int) string {
	lines := []string{
		fmt.Sprintf("target: %s", describeGraphActionRef(rs, target)),
		fmt.Sprintf("base:   %s", describeGraphActionRef(rs, base)),
		fmt.Sprintf("currentOnly: %d", currentOnly),
		fmt.Sprintf("targetOnly:  %d", targetOnly),
		"",
	}
	lines = append(lines, buildGraphActionReviewDiagram(rs, target, base, currentOnly, targetOnly)...)
	lines = append(lines, "")
	lines = append(lines, "y: continue  •  n: cancel")
	return strings.Join(lines, "\n")
}

func describeGraphActionRef(rs git.Status, hash string) string {
	if hash == "" {
		return "-"
	}
	if rows := graphRows(rs); len(rows) > 0 {
		if idx := findGraphRowByHash(rows, hash); idx >= 0 {
			subject := strings.TrimSpace(rows[idx].Commit.Subject)
			if subject != "" {
				return shorten(hash, 7) + "  " + subject
			}
		}
	}
	return shorten(hash, 7)
}

func buildGraphActionReviewDiagram(rs git.Status, target, base string, currentOnly, targetOnly int) []string {
	rows := graphRows(rs)
	if excerpt := compactGraphActionReviewExcerpt(rows, rs.Head, target, base); len(excerpt) > 0 {
		return excerpt
	}
	return []string{
		"* " + shorten(rs.Head, 7) + "  current branch",
		"|",
		"| + " + fmt.Sprintf("%d commits", currentOnly),
		"|",
		"| * " + shorten(base, 7) + "  merge base",
		"| | + " + fmt.Sprintf("%d commits", targetOnly),
		"|/",
		"* " + shorten(target, 7) + "  target branch",
	}
}

func compactGraphActionReviewExcerpt(rows []graphRow, head, target, base string) []string {
	if len(rows) == 0 {
		return nil
	}
	if !rowsHaveGraphPrefix(rows) {
		return nil
	}
	indices := []int{
		findGraphRowByHash(rows, head),
		findGraphRowByHash(rows, target),
		findGraphRowByHash(rows, base),
	}
	minIdx, maxIdx := -1, -1
	for _, idx := range indices {
		if idx < 0 {
			continue
		}
		if minIdx < 0 || idx < minIdx {
			minIdx = idx
		}
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	if minIdx < 0 || maxIdx < 0 {
		return nil
	}
	if maxIdx-minIdx > 5 {
		return nil
	}
	start := max(0, minIdx-1)
	end := min(len(rows), maxIdx+2)
	lines := make([]string, 0, end-start+1)
	for i := start; i < end; i++ {
		row := rows[i]
		if row.Commit.Hash == "" {
			continue
		}
		marker := ""
		switch row.Commit.Hash {
		case head:
			marker = "HEAD"
		case target:
			marker = "TARGET"
		case base:
			marker = "BASE"
		}
		graphCell := row.Graph
		if graphCell == "" {
			graphCell = "*"
		}
		line := fmt.Sprintf("  %-7s %-6s %-10s %s", shorten(row.Commit.Hash, 7), marker, graphCell, shorten(row.Commit.Subject, 32))
		lines = append(lines, strings.TrimRight(line, " "))
	}
	return lines
}

func rowsHaveGraphPrefix(rows []graphRow) bool {
	for _, row := range rows {
		if row.Graph != "" {
			return true
		}
	}
	return false
}
