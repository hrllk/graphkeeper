package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const (
	graphCommitWidth          = 6 // five visible hash characters plus padding
	graphBranchTokenWidth     = 10
	graphBranchOverflowWidth  = 4
	graphBranchFieldWidth     = graphBranchTokenWidth + graphBranchOverflowWidth
	graphStatusWidth          = 5
	graphTitleMinimumWidth    = 12
	graphTitlePreferredWidth  = 24
	graphTopologyMinimumWidth = 12
)

// graphColumnWidths is the row layout, measured once per render from the whole
// graph rather than assumed. Every row and the header take the same value, so
// the columns line up; recomputing per row would both cost O(rows^2) and break
// the alignment it is meant to keep.
//
// Measured over the whole graph, not the visible window. A window-adaptive
// budget recovers a few more columns on a linear stretch, but the layout then
// reflows as you scroll, and a screen that rearranges itself under the cursor
// costs more than the columns it saves.
type graphColumnWidths struct {
	Topology int
	Status   int // 0 when nothing in the graph carries a stash or a tag
}

// graphCols builds a layout from a topology width alone, keeping the status
// column at its full budget. Callers that have measured the graph should use
// measureGraphColumns instead; this exists for the render tests, which pin a
// topology width and expect the state column present.
func graphCols(topology int) graphColumnWidths {
	return graphColumnWidths{Topology: topology, Status: graphStatusWidth}
}

func graphRowFixedWidth(cols graphColumnWidths) int {
	fixed := graphCommitWidth + 1 + graphBranchFieldWidth + 1 + cols.Topology + 1
	if cols.Status > 0 {
		fixed += cols.Status + 1
	}
	return fixed
}

// measureGraphColumns sizes the two columns that were previously fixed at their
// worst case. In this repository at 266 commits the topology cell never exceeds
// 4 of its 12 budgeted columns and no row carries a tag, so the two together
// were spending 14 columns on nothing while the title ran negative below 100.
//
// branches is deliberately left alone: its widest decoration measures exactly
// its 14-column budget, so shrinking it would truncate real content rather than
// reclaim padding.
func measureGraphColumns(rows []graphRow, width int, stashCounts map[string]int) graphColumnWidths {
	cols := graphColumnWidths{Topology: 1}
	for _, row := range rows {
		if cell := lipgloss.Width(row.Graph); cell > cols.Topology {
			cols.Topology = cell
		}
		if row.Commit.Hash == "" {
			continue
		}
		if len(row.Commit.Tags) > 0 || stashCounts[row.Commit.Hash] > 0 {
			cols.Status = graphStatusWidth
		}
	}
	if ceiling := graphTopologyCeiling(width); cols.Topology > ceiling {
		cols.Topology = ceiling
	}
	return cols
}

// graphTopologyCeiling keeps the old proportional formula as an upper bound, so
// a graph with many lanes cannot take the whole row.
func graphTopologyCeiling(width int) int {
	current := max(graphTopologyMinimumWidth+6, int(float64(width)*0.30))
	return max(graphTopologyMinimumWidth, int(float64(current)*0.70))
}

func graphStatusText(stashCount, tagCount int) string {
	switch {
	case stashCount > 0 && tagCount > 0:
		return "S·T"
	case stashCount > 0:
		return "S"
	case tagCount > 0:
		return "T"
	default:
		return ""
	}
}

func renderGraphStatus(stashCount, tagCount int) string {
	status := graphStatusText(stashCount, tagCount)
	if status == "" {
		return strings.Repeat(" ", graphStatusWidth)
	}
	return padRight(status, graphStatusWidth)
}

func hasHeadDecoration(decorations []string) bool {
	for _, decoration := range decorations {
		if strings.HasPrefix(strings.TrimSpace(decoration), "HEAD -> ") {
			return true
		}
	}
	return false
}

func formatCompactDecorations(decorations []string, inventory LocalBranchInventory) string {
	return compactDecorationInfo(decorations, inventory).Text
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

func compactDecorationInfo(decorations []string, inventory LocalBranchInventory) decorationInfo {
	if !inventory.Known || !inventory.Fresh {
		return decorationInfo{Text: "-"}
	}
	if len(decorations) == 0 {
		return decorationInfo{Text: "-"}
	}
	localSet := make(map[string]struct{}, len(inventory.Names))
	for _, branch := range inventory.Names {
		branch = strings.TrimSpace(branch)
		if branch != "" {
			localSet[branch] = struct{}{}
		}
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
			if _, ok := localSet[strings.TrimSpace(name)]; !ok {
				continue
			}
			if state := addBranch(name); state != nil {
				state.local = true
				hasBranch = true
				hasLocalHead = true
				headBranch = strings.TrimSpace(name)
			}
		case strings.HasPrefix(decoration, "origin/HEAD -> origin/"):
			name := strings.TrimPrefix(decoration, "origin/HEAD -> origin/")
			if _, ok := localSet[name]; !ok {
				continue
			}
			if state := addBranch(name); state != nil {
				state.remote = true
				hasBranch = true
			}
		case decoration == "origin/HEAD":
			continue
		case strings.HasPrefix(decoration, "origin/"):
			name := strings.TrimPrefix(decoration, "origin/")
			if name == "HEAD" {
				continue
			}
			if _, ok := localSet[name]; !ok {
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
			}
		}
	}
	name := pickCompactBranchName(branches, headBranch)
	if name == "" {
		return decorationInfo{Text: "-", HasBranch: false}
	}
	state := branches[name]
	token := compactBranchToken(name, state)
	overflowCount := len(branches) - 1
	return decorationInfo{
		Text:         token + compactBranchOverflowSuffix(overflowCount),
		HasBranch:    hasBranch,
		HasLocalHead: hasLocalHead,
	}
}

func compactBranchToken(name string, state *branchState) string {
	token := "l->" + name
	if state != nil && state.remote && state.local {
		token = "o/l->" + name
	} else if state != nil && state.remote {
		token = "o->" + name
	}
	return fitBranchField(token, graphBranchTokenWidth)
}

func compactBranchOverflowSuffix(overflowCount int) string {
	if overflowCount <= 0 {
		return strings.Repeat(" ", graphBranchOverflowWidth)
	}
	if overflowCount > 99 {
		overflowCount = 99
	}
	return fmt.Sprintf("%-4s", fmt.Sprintf("+%d", overflowCount))
}

func fitBranchField(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return padRight(value, width)
	}
	budget := width
	if width >= 3 {
		budget -= 3
	}
	var b strings.Builder
	visible := 0
	for _, r := range value {
		rw := runewidth.RuneWidth(r)
		if visible+rw > budget {
			break
		}
		b.WriteRune(r)
		visible += rw
	}
	if width >= 3 {
		return b.String() + "..."
	}
	return b.String()
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
	case strings.HasPrefix(unit, "second"):
		return n + "s"
	case strings.HasPrefix(unit, "minute"):
		return n + "m"
	case strings.HasPrefix(unit, "hour"):
		return n + "h"
	case strings.HasPrefix(unit, "day"):
		return n + "d"
	case strings.HasPrefix(unit, "month"):
		return n + "m"
	case strings.HasPrefix(unit, "year"):
		return n + "y"
	case strings.HasPrefix(unit, "week"):
		return n + "w"
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
	if len(runes) <= 20 {
		return subject
	}
	return string(runes[:17]) + "..."
}

func fitGraphTitle(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) > width {
		if width <= 3 {
			return fitVisibleWidth(value, width)
		}
		return fitVisibleWidth(value, width-3) + "..."
	}
	return padRight(value, width)
}

func compactAuthorText(author string) string {
	author = strings.TrimSpace(author)
	if author == "" {
		return "-"
	}
	runes := []rune(author)
	if len(runes) <= 7 {
		return author
	}
	if len(runes) <= 2 {
		return string(runes)
	}
	return string(runes[:5]) + ".."
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
		return []string{renderContextDetailListItem("parent", "-")}
	}
	if len(node.Parents) == 1 {
		return []string{renderContextDetailListItem("parent", shorten(node.Parents[0], 8))}
	}
	lines := []string{renderContextDetailListItem("parent", "(multi parent)")}
	for _, parent := range node.Parents {
		lines = append(lines, fmt.Sprintf("  - %s", shorten(parent, 8)))
	}
	return lines
}

func focusBranchSummaryLines(node graphNode, width int, inventory LocalBranchInventory) []string {
	if len(node.Decorations) == 0 {
		return nil
	}
	lines := make([]string, 0, len(node.Decorations))
	for _, dec := range node.Decorations {
		line, ok := formatBranchSummaryDecoration(dec, inventory)
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("  - %s", fitBranchField(line, max(width-6, 0))))
	}
	return lines
}

func formatBranchSummaryDecoration(decoration string, inventory LocalBranchInventory) (string, bool) {
	if !inventory.Known || !inventory.Fresh {
		return "", false
	}
	local := make(map[string]struct{}, len(inventory.Names))
	for _, name := range inventory.Names {
		if name = strings.TrimSpace(name); name != "" {
			local[name] = struct{}{}
		}
	}
	if !inventory.Known || !inventory.Fresh {
		return "", false
	}
	if decoration == "" || strings.HasPrefix(decoration, "tag: ") {
		return "", false
	}
	switch {
	case strings.HasPrefix(decoration, "HEAD -> "):
		name := strings.TrimSpace(strings.TrimPrefix(decoration, "HEAD -> "))
		if name == "" {
			return "", false
		}
		if _, ok := local[name]; !ok {
			return "", false
		}
		return "l->" + name + " (HEAD)", true
	case strings.HasPrefix(decoration, "origin/HEAD -> origin/"):
		return "", false
	case strings.HasPrefix(decoration, "origin/"):
		return "", false
	default:
		if _, ok := local[decoration]; !ok {
			return "", false
		}
		return "l->" + decoration, true
	}
}
