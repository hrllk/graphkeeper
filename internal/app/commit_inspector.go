package app

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ansiutil "github.com/charmbracelet/x/ansi"

	"hrllk/graphkeeper/internal/git"
)

const inspectorDiffPage = 18

func inspectCommit(repo *git.Repo, hash string, ctx context.Context, request, epoch uint64) tea.Cmd {
	return func() tea.Msg {
		inspection, err := repo.InspectCommit(ctx, hash)
		return commitInspectorLoadedMsg{inspection: inspection, err: err, request: request, epoch: epoch}
	}
}

func loadCommitInspectorDiff(repo *git.Repo, inspection git.CommitInspection, index int, ctx context.Context, request, epoch uint64) tea.Cmd {
	return func() tea.Msg {
		if index < 0 || index >= len(inspection.Files) {
			return commitInspectorDiffMsg{err: fmt.Errorf("file selection is out of range"), request: request, epoch: epoch}
		}
		diff, err := repo.CommitDiff(ctx, inspection, inspection.Files[index], 2000, 0)
		return commitInspectorDiffMsg{diff: diff, err: err, request: request, epoch: epoch}
	}
}

func (m model) cancelInspector() model {
	if m.commitInspectorCancel != nil {
		m.commitInspectorCancel()
		m.commitInspectorCancel = nil
	}
	return m
}

func (m model) startInspectorDiff() (model, tea.Cmd) {
	if m.commitInspectorCursor < 0 || m.commitInspectorCursor >= len(m.commitInspector.Files) {
		return m, nil
	}
	m = m.cancelInspector()
	ctx, cancel := context.WithCancel(context.Background())
	m.commitInspectorCancel = cancel
	m.commitInspectorDiffLoading = true
	m.commitInspectorLoading = true
	m.commitInspectorDiffError = ""
	m.commitInspectorRequest++
	return m, loadCommitInspectorDiff(m.repo, m.commitInspector, m.commitInspectorCursor, ctx, m.commitInspectorRequest, m.commitInspectorEpoch)
}

func (m model) closeCommitInspector() model {
	m = m.cancelInspector()
	m.commitInspectorOpen = false
	m.commitInspectorHelp = false
	m.commitInspectorMetadataLoading = false
	m.commitInspectorDiffLoading = false
	m.commitInspectorLoading = false
	return m
}

func (m model) handleCommitInspectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "q" {
		return m.closeCommitInspector(), nil
	}
	if key == "?" {
		m.commitInspectorHelp = !m.commitInspectorHelp
		return m, nil
	}
	if key == "esc" {
		if m.commitInspectorHelp {
			m.commitInspectorHelp = false
			return m, nil
		}
		return m.closeCommitInspector(), nil
	}
	if m.commitInspectorHelp || m.commitInspectorMetadataLoading || m.commitInspectorDiffLoading {
		return m, nil
	}

	switch key {
	case "j", "down":
		if len(m.commitInspector.Files) == 0 {
			return m, nil
		}
		if m.commitInspectorCursor < len(m.commitInspector.Files)-1 {
			m.commitInspectorCursor++
			return m.selectInspectorFile()
		}
	case "k", "up":
		if m.commitInspectorCursor > 0 {
			m.commitInspectorCursor--
			return m.selectInspectorFile()
		}
	case "ctrl+u":
		m.commitInspectorScroll -= inspectorDiffPage
		if m.commitInspectorScroll < 0 {
			m.commitInspectorScroll = 0
		}
	case "ctrl+d":
		m.commitInspectorScroll += inspectorDiffPage
	}
	return m, nil
}

func (m model) selectInspectorFile() (tea.Model, tea.Cmd) {
	if len(m.commitInspector.Files) == 0 {
		return m, nil
	}
	m.commitInspectorLines = nil
	m.commitInspectorScroll = 0
	return m.startInspectorDiff()
}

func (m model) renderCommitInspectorPopup(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	innerWidth := max(width-4, 1)
	contentHeight := max(height-2, 1)

	lines := make([]string, 0, contentHeight)
	lines = append(lines,
		truncateInspector("commit: "+m.commitInspector.Hash, innerWidth),
		truncateInspector("message: "+truncateInspector(m.commitInspector.Subject, min(100, max(innerWidth-lipgloss.Width("message: "), 1))), innerWidth),
		truncateInspector("author: "+m.commitInspector.Author, innerWidth),
		truncateInspector("path: "+m.commitInspectorSelectedPath(), innerWidth),
	)
	for len(lines) < 3 {
		lines = append(lines, "")
	}
	if m.commitInspector.IsRoot {
		lines[2] = truncateInspector(lines[2]+"  FROM ROOT COMMIT", innerWidth)
	} else if m.commitInspector.Parent != "" {
		lines[2] = truncateInspector(lines[2]+"  FROM "+m.commitInspector.Parent, innerWidth)
	}
	lines = append(lines, strings.Repeat("─", innerWidth))

	bodyHeight := max(contentHeight-len(lines)-1, 1)
	if m.commitInspectorHelp {
		lines = append(lines, "Inspector help", "j/k changed files   Ctrl+U/D diff scroll", "q close   Esc close   ? help")
	} else if m.commitInspectorMetadataLoading {
		lines = append(lines, "Loading…")
	} else if m.commitInspectorError != "" {
		lines = append(lines, "Metadata error: "+truncateInspector(m.commitInspectorError, innerWidth))
	} else if len(m.commitInspector.Files) == 0 {
		lines = append(lines, "No changed files")
	} else {
		lines = append(lines, m.renderInspectorBody(innerWidth, bodyHeight)...)
	}
	for len(lines) < contentHeight-1 {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight-1 {
		lines = lines[:contentHeight-1]
	}
	lines = append(lines, truncateInspector("q close   Esc back   ? help", innerWidth))
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}

	style := popupBorder.Width(max(width-2, 0)).Height(max(height-2, 0)).Padding(0, 1)
	return style.Render(strings.Join(lines, "\n"))
}

func (m model) renderInspectorBody(width, height int) []string {
	if height < 1 {
		return nil
	}
	fileWidth := max(width*30/100, 18)
	if fileWidth > width-8 {
		fileWidth = max(width/2, 1)
	}
	diffWidth := max(width-fileWidth-3, 1)
	leftTitle := "Changed files"
	rightTitle := "Diff"
	rows := []string{padInspectorCell(truncateInspector(leftTitle, fileWidth), fileWidth) + " │ " + truncateInspector(rightTitle, diffWidth)}
	fileRows := m.inspectorTreeRows(fileWidth)
	fileStart := 0
	for i, row := range fileRows {
		if row.fileIndex == m.commitInspectorCursor {
			fileStart = max(i-height/2, 0)
			break
		}
	}
	diffLines := m.commitInspectorUnifiedLines()
	for i := 0; i < height-1; i++ {
		left := ""
		if idx := fileStart + i; idx >= 0 && idx < len(fileRows) {
			left = fileRows[idx].text
			if fileRows[idx].fileIndex == m.commitInspectorCursor {
				left = highlight.Render("> " + left)
			}
		}
		right := ""
		if m.commitInspectorDiffLoading && i == 0 {
			right = "Loading…"
		}
		if m.commitInspectorDiffError != "" && i == 0 {
			right = "Diff error: " + m.commitInspectorDiffError
		}
		if right == "" {
			if diffIndex := m.commitInspectorScroll + i; diffIndex >= 0 && diffIndex < len(diffLines) {
				right = diffLines[diffIndex]
			}
		}
		rows = append(rows, padInspectorCell(truncateInspector(left, fileWidth), fileWidth)+" │ "+truncateInspector(right, diffWidth))
	}
	return rows
}

// padInspectorCell pads by terminal cells, not bytes. File rows may contain
// ANSI status colors and the selected-row reverse style; fmt width verbs count
// those escape bytes and would move the pane divider to the right.
func padInspectorCell(s string, width int) string {
	missing := width - lipgloss.Width(s)
	if missing <= 0 {
		return s
	}
	return s + strings.Repeat(" ", missing)
}

type inspectorTreeRow struct {
	text      string
	fileIndex int
}

func (m model) inspectorTreeRows(width int) []inspectorTreeRow {
	type entry struct {
		path  string
		index int
	}
	entries := make([]entry, 0, len(m.commitInspector.Files))
	for i, file := range m.commitInspector.Files {
		entries = append(entries, entry{strings.ReplaceAll(file.Path, "\\", "/"), i})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	rows := make([]inspectorTreeRow, 0, len(entries))
	for _, item := range entries {
		rows = append(rows, inspectorTreeRow{text: inspectorFileLabel(m.commitInspector.Files[item.index], width), fileIndex: item.index})
	}
	return rows
}

func inspectorFileLabel(file git.CommitDiffFile, width int) string {
	path := compactInspectorPath(file.Path)
	if file.OldPath != "" && file.OldPath != file.Path {
		path = compactInspectorPath(file.OldPath) + " → " + compactInspectorPath(file.Path)
	}
	prefix := "  " + inspectorStatusStyle(file.Status).Render(inspectorStatus(file.Status)) + " "
	available := max(width-lipgloss.Width(prefix), 1)
	path = fitInspectorPath(path, available)
	return prefix + path
}

func (m model) commitInspectorSelectedPath() string {
	if m.commitInspectorCursor < 0 || m.commitInspectorCursor >= len(m.commitInspector.Files) {
		return "-"
	}
	file := m.commitInspector.Files[m.commitInspectorCursor]
	if file.OldPath != "" && file.OldPath != file.Path {
		return file.OldPath + " → " + file.Path
	}
	return file.Path
}

func compactInspectorPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	if len(parts) <= 1 {
		return "./" + path
	}
	return "../../" + parts[len(parts)-1]
}

func fitInspectorPath(path string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(path) <= width {
		return path
	}
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	if lipgloss.Width(name)+2 <= width {
		return truncateInspector("…/"+name, width)
	}
	return truncateInspector(name, width)
}

func (m model) commitInspectorUnifiedLines() []string {
	if m.commitInspectorCursor < 0 || m.commitInspectorCursor >= len(m.commitInspector.Files) {
		return []string{"Select a file"}
	}
	file := m.commitInspector.Files[m.commitInspectorCursor]
	if file.Status == "B" || file.Status == "S" || file.Status == "ModeOnly" {
		return []string{"No textual diff"}
	}
	rows := git.ParseDiffRows(m.commitInspectorLines)
	hunks := inspectorHunkHeaders(m.commitInspectorLines)
	if len(rows) == 0 && len(hunks) == 0 {
		return []string{"No textual changes"}
	}
	lines := make([]string, 0, len(rows)+len(hunks))
	for _, hunk := range hunks {
		if noColorEnabled() {
			lines = append(lines, hunk)
		} else {
			lines = append(lines, inspectorHunkStyle.Render(hunk))
		}
	}
	for _, row := range rows {
		if row.Kind == "modified" && row.FromPresent && row.ToPresent {
			lines = append(lines,
				formatInspectorDiffLine("-", row.OldLine, 0, row.From, "removed"),
				formatInspectorDiffLine("+", 0, row.NewLine, row.To, "added"),
			)
			continue
		}
		marker, oldNumber, newNumber, text := " ", row.OldLine, row.NewLine, row.To
		kind := row.Kind
		if row.Kind == "removed" {
			marker, oldNumber, newNumber, text = "-", row.OldLine, 0, row.From
		} else if row.Kind == "added" {
			marker, oldNumber, newNumber = "+", 0, row.NewLine
		}
		lines = append(lines, formatInspectorDiffLine(marker, oldNumber, newNumber, text, kind))
	}
	return lines
}

func inspectorHunkHeaders(lines []string) []string {
	result := make([]string, 0, 2)
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			result = append(result, line)
		}
	}
	return result
}

func formatInspectorDiffLine(marker string, oldNumber, newNumber int, text, kind string) string {
	oldText, newText := "—", "—"
	if oldNumber > 0 {
		oldText = fmt.Sprintf("%d", oldNumber)
	}
	if newNumber > 0 {
		newText = fmt.Sprintf("%d", newNumber)
	}
	line := fmt.Sprintf("%4s %4s %s %s", oldText, newText, marker, text)
	if noColorEnabled() {
		return line
	}
	switch kind {
	case "added":
		return inspectorAddedStyle.Render(line)
	case "removed":
		return inspectorRemovedStyle.Render(line)
	case "modified":
		return inspectorModifiedStyle.Render(line)
	default:
		return line
	}
}

func inspectorStatus(status string) string {
	switch status {
	case "A", "M", "D", "R", "C", "B", "S", "ModeOnly":
		return status
	default:
		return "?"
	}
}

func inspectorStatusStyle(status string) lipgloss.Style {
	if noColorEnabled() {
		return lipgloss.NewStyle()
	}
	switch status {
	case "A":
		return ansiBoldStyle(ansiGreen)
	case "M":
		return ansiBoldStyle(ansiBrightBlue)
	case "D":
		return ansiStyle(ansiBrightBlack)
	case "R":
		return ansiBoldStyle(ansiCyan)
	case "C":
		return ansiStyle(ansiBlue)
	case "B", "S", "ModeOnly":
		return ansiBoldStyle(ansiYellow)
	default:
		return lipgloss.NewStyle()
	}
}

func truncateInspector(s string, width int) string {
	if width <= 0 {
		return ""
	}
	// TruncateWc is ANSI-aware and never inserts a newline. This is important
	// for diff code: long lines must be clipped horizontally, never wrapped.
	return ansiutil.TruncateWc(s, width, "…")
}

var (
	inspectorAddedStyle    = ansiBoldStyle(ansiGreen)
	inspectorModifiedStyle = ansiBoldStyle(ansiBrightBlue)
	inspectorRemovedStyle  = ansiStyle(ansiBrightBlack)
	inspectorHunkStyle     = ansiBoldStyle(ansiCyan)
)

func noColorEnabled() bool { return os.Getenv("NO_COLOR") != "" }
