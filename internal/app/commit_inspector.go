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
)

const inspectorDiffPage = 18

func (m model) cancelInspector() model {
	if m.commitInspectorCancel != nil {
		m.commitInspectorCancel()
		m.commitInspectorCancel = nil
	}
	return m
}

func (m model) startInspectorDiff() (model, tea.Cmd) {
	if m.commitInspectorCursor < 0 || m.commitInspectorCursor >= len(m.commitInspectorSnapshot.Files) {
		if len(m.commitInspectorSnapshot.Files) == 0 && len(m.commitInspector.Files) > 0 {
			m.commitInspectorSnapshot = m.commitInspector
		} else {
			return m, nil
		}
	}
	m = m.cancelInspector()
	var cancel context.CancelFunc
	m.commitInspectorContext, cancel = context.WithCancel(context.Background())
	m.commitInspectorCancel = cancel
	m.commitInspectorDiffLoading = true
	m.commitInspectorLoading = true
	m.commitInspectorDiffError = ""
	m.commitInspectorRequest++
	file := m.commitInspectorSnapshot.Files[m.commitInspectorCursor]
	window := DiffWindowRequest{StartLine: 0, MaxLines: 2000, MaxBytes: 1 << 20}
	m.commitInspectorWindowRequest = window
	return m, loadCommitInspectorDiffCommand(m.commitInspectorContext, m, DiffRequest{Commit: m.commitInspectorSnapshot.FullHash, Parent: m.commitInspectorSnapshot.Parent, FileID: file.StableID, RequestID: m.commitInspectorRequest, RepositoryEpoch: m.commitInspectorEpoch, Window: window})
}

func (m model) closeCommitInspector() model {
	m = m.cancelInspector()
	m.commitInspectorOpen = false
	m.commitInspectorHelp = false
	m.commitInspectorMetadataLoading = false
	m.commitInspectorDiffLoading = false
	m.commitInspectorLoading = false
	m.commitInspectorContinuationPending = false
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
	filesCount := len(m.commitInspectorSnapshot.Files)
	if filesCount == 0 {
		filesCount = len(m.commitInspector.Files)
	}
	if m.commitInspectorHelp || m.commitInspectorMetadataLoading || m.commitInspectorDiffLoading {
		return m, nil
	}
	if key == "n" && m.commitInspectorDiffWindow.HasMore && !m.commitInspectorLoading && !m.commitInspectorMetadataLoading && !m.commitInspectorDiffLoading {
		return m, func() tea.Msg {
			return ContinuationRequested{Commit: m.commitInspectorSnapshot.FullHash, Parent: m.commitInspectorSnapshot.Parent, FileID: m.commitInspectorDiffWindow.FileID, RequestID: m.commitInspectorRequest, RepositoryEpoch: m.commitInspectorEpoch, Window: m.commitInspectorWindowRequest}
		}
	}

	switch key {
	case "j", "down":
		if filesCount == 0 {
			return m, nil
		}
		if m.commitInspectorCursor < filesCount-1 {
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
	if len(m.commitInspectorSnapshot.Files) == 0 && len(m.commitInspector.Files) == 0 {
		return m, nil
	}
	m.commitInspectorLines = nil
	m.commitInspectorDiffWindow = DiffWindow{}
	m.commitInspectorWindowRequest = DiffWindowRequest{}
	m.commitInspectorContinuationPending = false
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
		truncateInspector("commit: "+m.commitInspector.FullHash, innerWidth),
		truncateInspector("message: "+truncateInspector(m.commitInspector.Subject, min(100, max(innerWidth-lipgloss.Width("message: "), 1))), innerWidth),
		truncateInspector("author: "+m.commitInspector.AuthorName, innerWidth),
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
	if m.commitInspectorStale {
		lines = append(lines, truncateInspector("Repository changed; close and reopen to refresh.", innerWidth))
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

func inspectorFileLabel(file ChangedFile, width int) string {
	path := compactInspectorPath(file.Path)
	if file.OldPath != "" && file.OldPath != file.Path {
		path = compactInspectorPath(file.OldPath) + " → " + compactInspectorPath(file.Path)
	}
	status := screenStatus(file.Status)
	prefix := "  " + inspectorStatusStyle(status).Render(status) + " "
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

func renderInspectorDiffWindow(window DiffWindow) []string {
	lines := make([]string, 0)
	for _, hunk := range window.Hunks {
		if hunk.Header != "" {
			lines = append(lines, hunk.Header)
		}
		for _, row := range hunk.Rows {
			if row.Kind == "context" && row.FromPresent && row.ToPresent {
				lines = append(lines, formatInspectorDiffLine(" ", row.From.Number, row.To.Number, row.To.Text, "context"))
			} else if row.Kind == "modified" && row.FromPresent && row.ToPresent {
				lines = append(lines, formatInspectorDiffLine("-", row.From.Number, 0, row.From.Text, "removed"), formatInspectorDiffLine("+", 0, row.To.Number, row.To.Text, "added"))
			} else if row.FromPresent && !row.ToPresent {
				lines = append(lines, formatInspectorDiffLine("-", row.From.Number, 0, row.From.Text, "removed"))
			} else if row.ToPresent {
				lines = append(lines, formatInspectorDiffLine("+", 0, row.To.Number, row.To.Text, "added"))
			} else {
				lines = append(lines, formatInspectorDiffLine(" ", 0, 0, "", "context"))
			}
		}
	}
	if len(lines) == 0 {
		return []string{"No textual changes"}
	}
	return lines
}

func (m model) commitInspectorUnifiedLines() []string {
	if len(m.commitInspectorDiffWindow.Hunks) > 0 {
		return renderInspectorDiffWindow(m.commitInspectorDiffWindow)
	}
	if m.commitInspectorCursor < 0 || m.commitInspectorCursor >= len(m.commitInspector.Files) {
		return []string{"Select a file"}
	}
	file := m.commitInspector.Files[m.commitInspectorCursor]
	if file.Status == StatusBinary || file.Status == StatusSubmodule || file.Status == StatusModeOnly {
		return []string{"No textual diff"}
	}
	rows := parseInspectorDiffRows(m.commitInspectorLines)
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

func parseInspectorDiffRows(lines []string) []inspectorDiffRow {
	rows := make([]inspectorDiffRow, 0)
	oldLine, newLine := 0, 0
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			var oldStart, newStart int
			if _, err := fmt.Sscanf(line, "@@ -%d", &oldStart); err == nil {
				oldLine = oldStart
			}
			if marker := strings.Index(line, "+"); marker >= 0 {
				_, _ = fmt.Sscanf(line[marker:], "+%d", &newStart)
				newLine = newStart
			}
			continue
		}
		if len(line) < 1 {
			continue
		}
		switch line[0] {
		case ' ':
			rows = append(rows, inspectorDiffRow{Kind: "context", OldLine: oldLine, NewLine: newLine, From: line[1:], To: line[1:], FromPresent: true, ToPresent: true})
			oldLine++
			newLine++
		case '-':
			rows = append(rows, inspectorDiffRow{Kind: "removed", OldLine: oldLine, From: line[1:], FromPresent: true})
			oldLine++
		case '+':
			if strings.HasPrefix(line, "+++") {
				continue
			}
			rows = append(rows, inspectorDiffRow{Kind: "added", NewLine: newLine, To: line[1:], ToPresent: true})
			newLine++
		}
	}
	return rows
}

type inspectorDiffRow struct {
	Kind                   string
	OldLine, NewLine       int
	From, To               string
	FromPresent, ToPresent bool
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
