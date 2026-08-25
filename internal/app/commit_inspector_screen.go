package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderCommitInspectorScreen(m model, width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	innerWidth := max(width-4, 1)
	contentHeight := max(height-2, 1)
	snapshot := m.commitInspectorSnapshot
	if snapshot.FullHash == "" {
		snapshot.FullHash = m.commitInspectorRequestedCommit
		snapshot.Parent = m.commitInspectorRequestedParent
	}
	selected := selectedScreenFile(snapshot, m.commitInspectorCursor)
	lines := make([]string, 0, contentHeight)
	lines = append(lines,
		fitScreenText("COMMIT "+snapshot.FullHash, innerWidth),
		fitScreenText("message: "+snapshot.Subject, innerWidth),
		fitScreenText(screenAuthorLine(snapshot), innerWidth),
		fitScreenText("path: "+screenSelectedPath(selected, max(innerWidth-6, 1)), innerWidth),
	)
	lines = append(lines, strings.Repeat("─", innerWidth))
	bodyHeight := max(height-6, 1)
	lines = append(lines, screenBody(m, snapshot, selected, innerWidth, bodyHeight, height < 12)...)
	footer := "Esc back   ? help"
	if m.commitInspectorDiffWindow.HasMore {
		footer += "   n next"
	}
	for len(lines) < contentHeight-1 {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight-1 {
		lines = lines[:contentHeight-1]
	}
	lines = append(lines, fitScreenText(footer, innerWidth))
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}
	style := popupBorder.Width(max(width-2, 0)).Height(max(height-2, 0)).Padding(0, 1)
	return style.Render(strings.Join(lines, "\n"))
}

func screenAuthorLine(snapshot CommitSnapshot) string {
	author := "author: " + snapshot.AuthorName
	if snapshot.AuthorEmail != "" {
		author += " <" + snapshot.AuthorEmail + ">"
	}
	if snapshot.IsRoot {
		return author + "  ROOT COMMIT"
	}
	if snapshot.Parent != "" {
		return author + "  FROM " + snapshot.Parent
	}
	return author
}

func selectedScreenFile(snapshot CommitSnapshot, cursor int) ChangedFile {
	if cursor >= 0 && cursor < len(snapshot.Files) {
		return snapshot.Files[cursor]
	}
	return ChangedFile{}
}

func screenSelectedPath(file ChangedFile, width int) string {
	if file.Path == "" {
		return "-"
	}
	if file.OldPath != "" && file.OldPath != file.Path {
		oldWidth := max((width-3)/2, 1)
		newWidth := max(width-3-oldWidth, 1)
		return screenPathForWidth(file.OldPath, oldWidth) + " → " + screenPathForWidth(file.Path, newWidth)
	}
	return screenPathForWidth(file.Path, width)
}

func screenPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if path == "" {
		return "-"
	}
	return path
}

func screenBody(m model, snapshot CommitSnapshot, selected ChangedFile, width, height int, unsupported bool) []string {
	if height < 1 {
		return nil
	}
	fileRatio := 30
	if width >= 60 && width < 80 {
		fileRatio = 28
	}
	fileWidth := max(width*fileRatio/100, 10)
	if fileWidth >= width-4 {
		fileWidth = max(width/2, 1)
	}
	diffWidth := max(width-fileWidth-3, 1)
	rows := []string{padInspectorCell(fitScreenText("Changed files", fileWidth), fileWidth) + " │ " + fitScreenText("Diff", diffWidth)}
	fileRows := make([]string, 0, len(snapshot.Files))
	for i, file := range snapshot.Files {
		prefix := "  "
		if i == m.commitInspectorCursor {
			prefix = "> "
		}
		fileRows = append(fileRows, prefix+screenFileLabel(file, max(fileWidth-2, 1)))
	}
	diffLines := renderInspectorDiffWindow(m.commitInspectorDiffWindow)
	if m.commitInspectorDiffWindow.HasMore {
		hint := "partial"
		if m.commitInspectorDiffWindow.PartialReason != "" {
			hint += " (" + string(m.commitInspectorDiffWindow.PartialReason) + ")"
		}
		diffLines = append([]string{hint + "; press n next"}, diffLines...)
	}
	if len(m.commitInspectorDiffWindow.Hunks) == 0 && !m.commitInspectorDiffWindow.HasMore {
		diffLines = []string{"No textual changes"}
	}
	if unsupported {
		diffLines = append([]string{"unsupported height"}, diffLines...)
	}
	if m.commitInspectorStale {
		diffLines = append([]string{"Repository changed; close and reopen to refresh."}, diffLines...)
	}
	if m.commitInspectorMetadataLoading || m.commitInspectorLoading {
		diffLines = []string{"Loading…"}
	}
	if m.commitInspectorError != "" {
		diffLines = []string{"Metadata error: " + m.commitInspectorError}
	}
	if m.commitInspectorDiffError != "" {
		diffLines = []string{"Diff error: " + m.commitInspectorDiffError}
	}
	for i := 0; i < height-1; i++ {
		left, right := "", ""
		if i < len(fileRows) {
			left = fileRows[i]
		}
		if i < len(diffLines) {
			right = diffLines[i]
		}
		rows = append(rows, padInspectorCell(fitScreenText(left, fileWidth), fileWidth)+" │ "+fitScreenText(right, diffWidth))
	}
	return rows
}

func screenFileLabel(file ChangedFile, width int) string {
	label := screenStatus(file.Status) + " "
	if file.Binary || file.Status == StatusBinary {
		label += "[binary] "
	}
	if file.Status == StatusModeOnly {
		label += "[mode-only] "
	}
	if file.Status == StatusSubmodule {
		label += "[submodule] "
	}
	if file.Status == StatusRenamed && file.OldPath != "" {
		label += screenPathForWidth(file.OldPath, max(width-lipgloss.Width(label)-4, 1)) + " → "
		label += screenPathForWidth(file.Path, max(width-lipgloss.Width(label), 1))
	} else {
		label += screenPathForWidth(file.Path, max(width-lipgloss.Width(label), 1))
	}
	return label
}

func screenStatus(status ChangedFileStatus) string {
	switch status {
	case StatusAdded:
		return "A"
	case StatusModified:
		return "M"
	case StatusDeleted:
		return "D"
	case StatusRenamed:
		return "R"
	case StatusCopied:
		return "C"
	case StatusBinary:
		return "B"
	case StatusModeOnly:
		return "Mode"
	case StatusSubmodule:
		return "Sub"
	default:
		return "?"
	}
}

func screenPathForWidth(path string, width int) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if lipgloss.Width(path) <= width {
		return path
	}
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	if lipgloss.Width(name) > width {
		return truncateInspector(name, width)
	}
	for suffixCount := len(parts) - 1; suffixCount >= 1; suffixCount-- {
		suffix := strings.Join(parts[len(parts)-suffixCount:], "/")
		candidate := parts[0] + "/…/" + suffix
		if lipgloss.Width(candidate) <= width {
			return candidate
		}
	}
	result := "…/" + name
	if lipgloss.Width(result) > width {
		return truncateInspector(name, width)
	}
	return result
}

func fitScreenText(text string, width int) string { return truncateInspector(text, width) }
