package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hrllk/git-graph-tui/internal/state"
)

var (
	border      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	title       = lipgloss.NewStyle().Bold(true)
	muted       = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	accent      = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	warn        = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	ok          = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	headMark    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	branchMark  = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	pointerMark = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	localColor  = lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	remoteColor = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	tagColor    = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	highlight   = lipgloss.NewStyle().Reverse(true).Bold(true)
)

func (m model) View() string {
	left := m.renderGraphPane()
	right := m.renderDetailPane()
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	footer := muted.Render("tab/shift+tab: section  •  up/down: move  •  ctrl+u/d: page  •  g/G: top/bottom  •  f: fetch  •  enter/c: checkout  •  n: new branch  •  q: quit")
	return body + "\n" + footer + "\n"
}

func (m model) renderGraphPane() string {
	var content strings.Builder
	content.WriteString(renderSectionTitle("Graph", m.activeSection == sectionGraph))
	content.WriteString(renderGraphViewport(m))
	content.WriteString(renderSectionBlock("Branches", m, sectionCurrent))
	content.WriteString(renderSectionBlock("Remote", m, sectionRemote))
	content.WriteString(renderSectionBlock("Tags", m, sectionTags))
	return border.Width(max(56, paneWidth(m.width, 0.74))).Render(content.String())
}

func (m model) renderDetailPane() string {
	var content strings.Builder
	content.WriteString(title.Render("Mode") + "\n\n")
	content.WriteString(renderStatus(m.status) + "\n\n")
	content.WriteString(title.Render("Repo") + "\n")
	content.WriteString(fmt.Sprintf("branch: %s\n", emptyDash(m.repoStatus.Branch)))
	content.WriteString(fmt.Sprintf("head:   %s\n", shorten(m.repoStatus.Head, 12)))
	content.WriteString(fmt.Sprintf("upstream: %s\n", emptyDash(m.repoStatus.Upstream)))
	content.WriteString(fmt.Sprintf("remote: %s\n", emptyDash(m.repoStatus.Remote)))
	focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
	if focus.Hash != "" {
		content.WriteString(fmt.Sprintf("focus: %s\n", focusLineSummary(focus)))
	}
	content.WriteString(fmt.Sprintf("section: %s\n", sectionName(m.activeSection)))
	if m.status.Selected != "" {
		content.WriteString(fmt.Sprintf("selected: %s\n", m.status.Selected))
	}
	if m.branchOpen {
		content.WriteString("\n")
		content.WriteString(title.Render("New Branch") + "\n")
		content.WriteString(fmt.Sprintf("base: %s\n", emptyDash(m.branchBase)))
		content.WriteString(muted.Render("> ") + m.branchDraft + "\n")
	}
	content.WriteString("\n")
	content.WriteString(title.Render("Actions") + "\n")
	content.WriteString(renderActionHelp(m.status))
	return border.Width(max(34, paneWidth(m.width, 0.38))).Render(content.String())
}

func renderStatus(s state.Status) string {
	switch s.Mode {
	case state.ModeBrowse:
		return ok.Render("Browse") + "\n" + s.Message + "\n" + muted.Render("Use tab to change section.")
	case state.ModeLoading:
		return accent.Render("Loading") + "\n" + s.Message
	case state.ModeEmpty:
		return warn.Render("Empty") + "\n" + s.Message
	case state.ModeError:
		return warn.Render("Error") + "\n" + s.Message
	case state.ModeBlocked:
		return warn.Render("Blocked") + "\n" + s.Message + "\n" + muted.Render(s.Detail)
	case state.ModeTargetPick:
		return accent.Render(strings.ToUpper(string(s.Action))) + "\n" + s.Message + "\n" + renderTargets(s)
	case state.ModeOutcomePreview:
		return accent.Render(strings.ToUpper(string(s.Action))) + "\n" + s.Message + "\n" + muted.Render(s.Detail)
	default:
		return s.Message
	}
}

func renderTargets(s state.Status) string {
	if len(s.Targets) == 0 {
		return muted.Render("(no targets)")
	}
	var b strings.Builder
	for i, t := range s.Targets {
		prefix := "  "
		if i == s.TargetIdx {
			prefix = "> "
		}
		b.WriteString(prefix + formatTargetItem(t) + "\n")
	}
	return b.String()
}

func formatTargetItem(t state.TargetItem) string {
	switch t.Kind {
	case state.TargetKindLocal:
		if t.Current {
			return ok.Render("l->" + t.Name)
		}
		return "l->" + t.Name
	case state.TargetKindRemote:
		name := t.Name
		if strings.HasPrefix(name, "origin/") {
			name = strings.TrimPrefix(name, "origin/")
		}
		label := "o->" + name
		if t.Default {
			label += " (default)"
		}
		return label
	case state.TargetKindTag:
		return "tag    " + t.Name
	default:
		return t.Name
	}
}

func renderActionHelp(s state.Status) string {
	switch s.Mode {
	case state.ModeBrowse:
		return "up/down: move graph pointer\nf: fetch remotes\nenter/c: checkout\nn: new branch\np: pull\nm: merge\ne: rebase\ns: reset\n"
	case state.ModeTargetPick:
		return "up/down: choose target\nenter: preview\nesc: back\n"
	case state.ModeOutcomePreview:
		if s.CanExecute {
			return "enter: execute\nesc: back\n"
		}
		return "esc: back\n"
	case state.ModeBlocked:
		return "esc: back\nr: refresh\n"
	default:
		return "r: refresh\n"
	}
}

func renderGraphViewport(m model) string {
	rows := graphRows(m.repoStatus)
	if len(rows) == 0 {
		return muted.Render("(no graph to show yet)") + "\n\n"
	}
	start := clampScroll(m.graphScroll, len(rows), graphPageSize(&m))
	end := start + graphPageSize(&m)
	if end > len(rows) {
		end = len(rows)
	}
	var b strings.Builder
	b.WriteString(muted.Render(fmt.Sprintf("graph page %d-%d/%d", start+1, end, len(rows))) + "\n")
	graphActive := m.activeSection == sectionGraph
	for i := start; i < end; i++ {
		b.WriteString(renderGraphLine(rows[i], graphActive && i == m.sectionCursor[sectionGraph], graphActive, m.graphLaneCursor, m.repoStatus.LocalBranches) + "\n")
		if i+1 < len(rows) {
			for _, line := range renderGraphConnectorLines(rows[i], rows[i+1]) {
				b.WriteString(line + "\n")
			}
		}
	}
	b.WriteString("\n")
	return b.String()
}

func renderSectionBlock(label string, m model, section graphSection) string {
	items := sectionTargets(m.repoStatus, section)
	var b strings.Builder
	b.WriteString(renderSectionTitle(label, m.activeSection == section))
	if len(items) == 0 {
		b.WriteString(muted.Render("  (empty)") + "\n\n")
		return b.String()
	}
	cursor := m.sectionCursor[section]
	for i, item := range items {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		b.WriteString(prefix + formatTargetItem(item) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

func renderSectionTitle(label string, active bool) string {
	if active {
		return warn.Render(label) + "\n"
	}
	return title.Render(label) + "\n"
}

func renderRefSection(label string, refs []string, kind state.TargetKind, cursor, offset int) string {
	if len(refs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(muted.Render(label+":") + "\n")
	for i, ref := range refs {
		prefix := "  "
		if cursor == offset+i {
			prefix = "> "
		}
		b.WriteString(prefix + formatTargetItem(state.TargetItem{Kind: kind, Name: ref, Ref: ref}) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

func renderGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string) string {
	hash := fmt.Sprintf("%-8s", shorten(row.Commit.Hash, 7))
	refInfo := compactDecorationInfo(row.Commit.Decorations, localBranches)
	refs := fmt.Sprintf("%-16s", refInfo.Text)
	isHead := hasHeadDecoration(row.Commit.Decorations)
	width := graphRowWidth(row)
	lane := displayLane(row, width)
	cursorLane := laneCursor
	if width > 0 && cursorLane >= width {
		cursorLane = width - 1
	}
	pointerFocused := graphActive && selected && cursorLane == lane
	cells := make([]string, 0, width)
	for i := 0; i < width; i++ {
		cell := " "
		beforeActive := i < len(row.Before)
		afterActive := i < len(row.After)
		switch {
		case i == lane:
			cell = "*"
		case beforeActive || afterActive:
			cell = "|"
		}
		if isHead && i == lane {
			cell = headMark.Render(cell)
		} else if pointerFocused && i == lane {
			cell = pointerMark.Render(cell)
		}
		cells = append(cells, cell)
	}
	if isHead {
		refs = headMark.Render(refs)
	} else if pointerFocused && refInfo.HasBranch {
		refs = branchMark.Render(refs)
	}
	line := hash + " " + refs + " " + strings.Join(cells, " ")
	if selected {
		return "> " + line
	}
	return "  " + line
}

func renderGraphConnectorLines(current, next graphRow) []string {
	if shouldCollapseRowDisplay(next) {
		return collapseConnectorLines(current)
	}
	return nil
}

func collapseConnectorLines(current graphRow) []string {
	width := len(current.After)
	if width <= 1 {
		return nil
	}
	lines := make([]string, 0, width)
	full := make([]string, width)
	for i := range full {
		full[i] = "|"
	}
	lines = append(lines, renderGraphSpacer(full))
	for w := width; w >= 2; w-- {
		cells := make([]string, w)
		for i := range cells {
			cells[i] = "|"
		}
		cells[w-1] = "/"
		lines = append(lines, renderGraphSpacer(cells))
	}
	return lines
}

func renderGraphSpacer(cells []string) string {
	prefix := strings.Repeat(" ", 8) + " " + strings.Repeat(" ", 16) + " "
	return "  " + prefix + strings.Join(cells, " ")
}

func shouldCollapseRowDisplay(row graphRow) bool {
	if len(row.Before) <= 1 {
		return false
	}
	for _, hash := range row.Before {
		if hash != row.Commit.Hash {
			return false
		}
	}
	return true
}

func isBlankGraphCells(cells []string) bool {
	for _, cell := range cells {
		if cell != " " {
			return false
		}
	}
	return true
}

func displayLane(row graphRow, width int) int {
	if width <= 1 && shouldCollapseRowDisplay(row) {
		return 0
	}
	if row.Lane < 0 {
		return 0
	}
	if width > 0 && row.Lane >= width {
		return width - 1
	}
	return row.Lane
}

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
	Text      string
	HasBranch bool
}

func compactDecorationInfo(decorations []string, localBranches []string) decorationInfo {
	if len(decorations) == 0 {
		return decorationInfo{Text: "-"}
	}
	localSet := make(map[string]struct{}, len(localBranches))
	for _, branch := range localBranches {
		localSet[branch] = struct{}{}
	}
	parts := make([]string, 0, len(decorations))
	remoteParts := make([]string, 0, len(decorations))
	localParts := make([]string, 0, len(decorations))
	tagParts := make([]string, 0, len(decorations))
	seen := make(map[string]struct{})
	hasBranch := false
	for _, decoration := range decorations {
		token, kind := compactDecoration(decoration, localSet)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		switch kind {
		case "head":
			hasBranch = true
			parts = append(parts, token)
		case "remote":
			hasBranch = true
			remoteParts = append(remoteParts, token)
		case "local":
			hasBranch = true
			localParts = append(localParts, token)
		case "tag":
			tagParts = append(tagParts, token)
		}
	}
	parts = append(parts, remoteParts...)
	parts = append(parts, localParts...)
	parts = append(parts, tagParts...)
	if len(parts) == 0 {
		return decorationInfo{Text: "-", HasBranch: false}
	}
	joined := strings.Join(parts, ", ")
	runes := []rune(joined)
	if len(runes) <= 16 {
		return decorationInfo{Text: joined, HasBranch: hasBranch}
	}
	return decorationInfo{Text: string(runes[:16]), HasBranch: hasBranch}
}

func compactDecoration(decoration string, localSet map[string]struct{}) (string, string) {
	decoration = strings.TrimSpace(decoration)
	switch {
	case strings.HasPrefix(decoration, "HEAD -> "):
		return compactToken("l", strings.TrimPrefix(decoration, "HEAD -> ")), "head"
	case strings.HasPrefix(decoration, "tag: "):
		return compactToken("t", strings.TrimPrefix(decoration, "tag: ")), "tag"
	case strings.HasPrefix(decoration, "origin/"):
		name := strings.TrimPrefix(decoration, "origin/")
		if _, ok := localSet[name]; ok {
			return compactToken("o", name), "remote"
		}
		return "", ""
	case decoration != "":
		return compactToken("l", decoration), "local"
	default:
		return "", ""
	}
}

func compactToken(kind, name string) string {
	token := kind + "->" + name
	if len([]rune(token)) <= 10 {
		return token
	}
	runes := []rune(token)
	return string(runes[:9]) + "."
}

func focusLineSummary(node graphNode) string {
	if len(node.Decorations) == 0 {
		return node.Hash
	}
	return node.Hash + "  " + strings.Join(node.Decorations, ", ")
}

func paneWidth(total int, ratio float64) int {
	if total <= 0 {
		return 0
	}
	return int(float64(total) * ratio)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
