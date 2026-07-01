package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
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

type branchState struct {
	local  bool
	remote bool
}

func compactDecorationInfo(decorations []string, localBranches []string) decorationInfo {
	if len(decorations) == 0 {
		return decorationInfo{Text: "-"}
	}
	localSet := make(map[string]struct{}, len(localBranches))
	for _, branch := range localBranches {
		localSet[branch] = struct{}{}
	}
	branches := make(map[string]*branchState)
	hasBranch := false
	hasLocalHead := false
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
				hasBranch = true
				hasLocalHead = true
				headBranch = strings.TrimSpace(name)
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
			if state := addBranch(name); state != nil {
				state.remote = true
				hasBranch = true
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
	name := pickCompactBranchName(branches, headBranch)
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
	overflowCount := len(branches) - 1
	if overflowCount > 0 {
		candidate := fmt.Sprintf("%s +%d", token, overflowCount)
		if len([]rune(candidate)) <= 10 {
			token = candidate
		}
	}
	return decorationInfo{Text: token, HasBranch: hasBranch, HasLocalHead: hasLocalHead}
}

func pickCompactBranchName(branches map[string]*branchState, headBranch string) string {
	if headBranch != "" {
		if state := branches[headBranch]; state != nil && (state.local || state.remote) {
			return headBranch
		}
	}
	localNames := make([]string, 0, len(branches))
	remoteNames := make([]string, 0, len(branches))
	for name, state := range branches {
		if state == nil {
			continue
		}
		if state.local {
			localNames = append(localNames, name)
			continue
		}
		if state.remote {
			remoteNames = append(remoteNames, name)
		}
	}
	if len(localNames) > 0 {
		sort.Strings(localNames)
		return localNames[0]
	}
	if len(remoteNames) > 0 {
		sort.Strings(remoteNames)
		return remoteNames[0]
	}
	return ""
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

func fitVisibleWidth(value string, width int) string {
	if width <= 0 || value == "" {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	runes := []rune(value)
	var b strings.Builder
	visible := 0
	sawANSI := false
	for i := 0; i < len(runes) && visible < width; {
		r := runes[i]
		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			sawANSI = true
			start := i
			i += 2
			for i < len(runes) {
				ch := runes[i]
				i++
				if ch >= '@' && ch <= '~' {
					break
				}
			}
			b.WriteString(string(runes[start:i]))
			continue
		}
		if r == '\x1b' {
			sawANSI = true
		}
		w := runewidth.RuneWidth(r)
		if w <= 0 {
			w = 1
		}
		if visible+w > width {
			break
		}
		b.WriteRune(r)
		visible += w
		i++
	}
	if sawANSI {
		b.WriteString("\x1b[0m")
	}
	return b.String()
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
