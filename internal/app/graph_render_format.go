package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func hasHeadDecoration(decorations []string) bool {
	for _, decoration := range decorations {
		if strings.HasPrefix(strings.TrimSpace(decoration), "HEAD -> ") {
			return true
		}
	}
	return false
}

func formatCompactDecorations(decorations []string, localBranches []string) string {
	return compactDecorationInfo(decorations, localBranches).Text
}

type decorationInfo struct {
	Text         string
	HasBranch    bool
	HasLocalHead bool
}

func compactDecorationInfo(decorations []string, localBranches []string) decorationInfo {
	if len(decorations) == 0 {
		return decorationInfo{Text: "-"}
	}
	localSet := make(map[string]struct{}, len(localBranches))
	for _, branch := range localBranches {
		localSet[branch] = struct{}{}
	}
	type branchState struct {
		local  bool
		remote bool
	}
	branches := make(map[string]*branchState)
	order := make([]string, 0, len(decorations))
	hasBranch := false
	hasLocalHead := false

	addBranch := func(name string) *branchState {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil
		}
		state, ok := branches[name]
		if !ok {
			state = &branchState{}
			branches[name] = state
			order = append(order, name)
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
			if state := addBranch(strings.TrimPrefix(decoration, "HEAD -> ")); state != nil {
				state.local = true
				hasBranch = true
				hasLocalHead = true
			}
		case strings.HasPrefix(decoration, "origin/HEAD -> origin/"):
			if state := addBranch(strings.TrimPrefix(decoration, "origin/HEAD -> origin/")); state != nil {
				state.remote = true
				hasBranch = true
			}
		case decoration == "origin/HEAD":
			if state := addBranch("HEAD"); state != nil {
				state.remote = true
				hasBranch = true
			}
		case strings.HasPrefix(decoration, "origin/"):
			name := strings.TrimPrefix(decoration, "origin/")
			if name == "HEAD" {
				continue
			}
			if _, ok := localSet[name]; ok {
				if state := addBranch(name); state != nil {
					state.remote = true
					hasBranch = true
				}
			}
		case strings.HasPrefix(decoration, "tag: "):
			continue
		default:
			if _, ok := localSet[decoration]; ok {
				if state := addBranch(decoration); state != nil {
					state.local = true
					hasBranch = true
				}
			} else if !strings.Contains(decoration, "/") {
				if state := addBranch(decoration); state != nil {
					state.local = true
					hasBranch = true
				}
			}
		}
	}
	name := ""
	for _, candidate := range order {
		if state := branches[candidate]; state != nil && (state.local || state.remote) {
			name = candidate
			break
		}
	}
	if name == "" {
		return decorationInfo{Text: "-", HasBranch: false}
	}
	state := branches[name]
	token := "l->" + name
	if state.remote && state.local {
		token = "o/l->" + name
	} else if state.remote {
		token = "o->" + name
	}
	if len([]rune(token)) > 10 {
		runes := []rune(token)
		token = string(runes[:9]) + "."
	}
	return decorationInfo{Text: token, HasBranch: hasBranch, HasLocalHead: hasLocalHead}
}

func compactWhenText(relative string) string {
	relative = strings.TrimSpace(relative)
	if relative == "" {
		return "-"
	}
	if strings.HasSuffix(relative, " ago") {
		relative = strings.TrimSpace(strings.TrimSuffix(relative, " ago"))
	}
	parts := strings.Fields(relative)
	if len(parts) < 2 {
		return shorten(relative, 7)
	}
	n := parts[0]
	unit := parts[1]
	switch {
	case strings.HasPrefix(unit, "minute"):
		return n + "mins"
	case strings.HasPrefix(unit, "hour"):
		return n + "hours"
	case strings.HasPrefix(unit, "day"):
		return n + "days"
	case strings.HasPrefix(unit, "month"):
		return n + "mons"
	case strings.HasPrefix(unit, "year"):
		return n + "yrs"
	case strings.HasPrefix(unit, "week"):
		return n + "wks"
	default:
		return shorten(relative, 7)
	}
}

func compactTitleText(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "-"
	}
	runes := []rune(subject)
	if len(runes) <= 10 {
		return subject
	}
	return string(runes[:7]) + "..."
}

func padRight(value string, width int) string {
	if lipgloss.Width(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-lipgloss.Width(value))
}

func focusParentLines(node graphNode, width int) []string {
	if len(node.Parents) == 0 {
		return []string{"parent: -"}
	}
	if len(node.Parents) == 1 {
		return []string{fmt.Sprintf("parent: %s", node.Parents[0])}
	}
	lines := []string{"parent: (multi parent)"}
	parentWidth := max(width-4, 0)
	for _, parent := range node.Parents {
		lines = append(lines, fmt.Sprintf("  - %s", shorten(parent, parentWidth)))
	}
	return lines
}

func focusBranchSummaryLines(node graphNode, width int) []string {
	if len(node.Decorations) == 0 {
		return nil
	}
	lines := make([]string, 0, len(node.Decorations))
	for _, dec := range node.Decorations {
		lines = append(lines, fmt.Sprintf("  - %s", shorten(dec, max(width-6, 0))))
	}
	return lines
}
