package app

import (
	"strings"
	"time"

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
	header := screenHeaderLines(snapshot, selected, innerWidth)
	lines := make([]string, 0, contentHeight)
	lines = append(lines, header...)
	lines = append(lines, strings.Repeat("─", innerWidth))
	lines = append(lines, screenBody(m, snapshot, selected, innerWidth, inspectorBodyRowsFor(height, len(header)), height < 12)...)
	footer := "Esc back   ? help"
	if m.commitInspectorHelp {
		footer = "Esc back   ? close"
	} else if m.commitInspectorDiffWindow.HasMore {
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

// screenHeaderLines builds the Inspector header. Its length is the header's
// only definition: inspectorBodyRowsFor derives the chrome budget from it, so
// adding or removing a row here cannot silently steal rows from the diff pane.
//
// The row count is 5, or 6 when the author and committer dates disagree, which
// only happens once a commit has been rebased or cherry-picked.
func screenHeaderLines(snapshot CommitSnapshot, selected ChangedFile, innerWidth int) []string {
	lines := []string{
		fitScreenText("COMMIT "+snapshot.FullHash, innerWidth),
		fitScreenText("message: "+snapshot.Subject, innerWidth),
		fitScreenText(screenAuthorLine(snapshot), innerWidth),
	}
	if when := screenStampText(snapshot.AuthorDate, innerWidth-len("date: ")); when != "" {
		lines = append(lines, fitScreenText("date: "+when, innerWidth))
		// A separate committed: row only earns its line when it differs. On an
		// ordinary commit the two stamps are identical and a second copy would
		// cost a diff row for nothing.
		if snapshot.CommitDate != "" && snapshot.CommitDate != snapshot.AuthorDate {
			if committed := screenStampText(snapshot.CommitDate, innerWidth-len("committed: ")); committed != "" {
				lines = append(lines, fitScreenText("committed: "+committed, innerWidth))
			}
		}
	}
	return append(lines, fitScreenText("path: "+screenSelectedPath(selected, max(innerWidth-6, 1)), innerWidth))
}

// screenStampText renders a strict ISO 8601 stamp for the Inspector, which is
// the surface where a commit is confirmed rather than scanned, so it keeps the
// recorded offset when there is room for it.
//
//	budget >= 25  2026-09-08 23:15:16 +09:00
//	budget >= 16  2026-09-08 23:15
//	budget >= 10  2026-09-08
//	otherwise     row omitted
func screenStampText(value string, budget int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return ""
	}
	switch {
	// "2026-09-08 23:15:16 +09:00" is 26 columns, not 25. At exactly 25 the row
	// overflowed by one and fitScreenText clipped the offset to "+09:0", which is
	// the misleading render this whole step-down exists to avoid.
	case budget >= 26:
		return parsed.Format("2006-01-02 15:04:05 -07:00")
	case budget >= 16:
		return parsed.Format("2006-01-02 15:04")
	case budget >= 10:
		return parsed.Format("2006-01-02")
	default:
		return ""
	}
}

func screenAuthorLine(snapshot CommitSnapshot) string {
	author := "author: " + snapshot.AuthorName
	if snapshot.AuthorEmail != "" {
		author += " <" + snapshot.AuthorEmail + ">"
	}
	// The "FROM <hash>" tail stays. Task 6.8 owns replacing it with a parent: row
	// and owns the narrow-width copy priority that makes it the first thing cut.
	// Removing it here would not buy the date rows anything: the tail sits on the
	// author row while the dates are rows of their own, so they compete for
	// header height, which dropping the tail does not reclaim. Taking it out
	// would only lose parent visibility until 6.8 is scheduled, and
	// TestCommitInspectorScreenMatrixKeepsFrameAndPartialFooter pins it.
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

// inspectorBodyRows reports how many rows the Inspector body can show at the
// given outer height, and it is the only place that math lives: the renderer
// draws this many and the scroll keys clamp against it.
//
// The frame spends height-2 on content, keeps one row for the footer, and the
// four header rows, the separator and the pane header take six more. Generating
// more than that used to be silently truncated, which left the last rows of a
// scrolled diff unreachable.
// inspectorBodyRowsFor reports how many rows screenBody may produce.
//
// The chrome around the body is the header, one separator rule, one padding row
// and the footer. That used to be the constant 7, which encoded a four-row
// header; once the header became five or six rows the constant over-reported the
// budget and renderCommitInspectorScreen's lines[:contentHeight-1] trimmed the
// tail of the diff pane with no error and no marker. Deriving it from the header
// the renderer actually built keeps the two from drifting apart again.
func inspectorBodyRowsFor(height, headerRows int) int {
	// separator + the "Changed files │ Diff" pane header screenBody prepends +
	// the footer. screenBody then appends exactly bodyRows rows after its pane
	// header, so this is the count the frame keeps.
	const nonHeaderChrome = 3
	// Clamping to 1 was wrong: at height 11 with a six-row header the true
	// budget is 0, and reporting 1 made the clamp and the page size believe a
	// row existed while renderCommitInspectorScreen's lines[:contentHeight-1]
	// had already dropped it. Nothing visible has to report zero.
	return max(max(height-2, 1)-headerRows-nonHeaderChrome, 0)
}

// inspectorBodyRowCount is the single source of truth for how many diff rows the
// Inspector shows. The renderer draws that many and the scroll keys move by that
// many, and inspectorDiffLines states that invariant: "the renderer draws them
// and the scroll keys measure them, so both must agree on the count."
//
// It derives from the header the renderer actually builds because the header
// height varies with the snapshot (a rebased commit adds a committed: row) and
// with the width (a narrow frame drops the date rows entirely). A constant here
// is what produced the T8 regression the first time, where the clamp ran two
// rows ahead of what the frame kept and the last lines of a scrolled diff could
// not be reached. Counting the real header is slightly wasteful and the only
// version that cannot drift.
func (m model) inspectorBodyRowCount() int {
	header := screenHeaderLines(m.commitInspectorSnapshot, ChangedFile{}, max(m.width-4, 1))
	return inspectorBodyRowsFor(m.height, len(header))
}

// inspectorFileOffset scrolls the changed-files list only as far as needed to keep
// the selected row on screen. Without it the selection walks off the bottom and no
// "> " marker is rendered anywhere.
func inspectorFileOffset(cursor, total, visible int) int {
	if visible < 1 || total <= visible || cursor < visible {
		return 0
	}
	return min(cursor-visible+1, total-visible)
}

// inspectorDiffLines assembles the diff pane's lines for the current state. The
// renderer draws them and the scroll keys measure them, so both must agree on the
// count; that is why this is not inlined in screenBody.
func (m model) inspectorDiffLines(unsupported bool) []string {
	if m.commitInspectorDiffError != "" {
		return []string{"Diff error: " + m.commitInspectorDiffError}
	}
	if m.commitInspectorError != "" {
		return []string{"Metadata error: " + m.commitInspectorError}
	}
	if m.commitInspectorMetadataLoading || m.commitInspectorLoading {
		return []string{"Loading…"}
	}
	lines := renderInspectorDiffWindow(m.commitInspectorDiffWindow)
	if m.commitInspectorDiffWindow.HasMore {
		hint := "partial"
		if m.commitInspectorDiffWindow.PartialReason != "" {
			hint += " (" + string(m.commitInspectorDiffWindow.PartialReason) + ")"
		}
		lines = append([]string{hint + "; press n next"}, lines...)
	}
	if len(m.commitInspectorDiffWindow.Hunks) == 0 && !m.commitInspectorDiffWindow.HasMore {
		lines = []string{"No textual changes"}
	}
	if unsupported {
		lines = append([]string{"unsupported height"}, lines...)
	}
	if m.commitInspectorStale {
		lines = append([]string{"Repository changed; close and reopen to refresh."}, lines...)
	}
	return lines
}

// maxInspectorDiffScroll is the furthest the diff pane can scroll before the last
// line reaches the bottom of the viewport.
func (m model) maxInspectorDiffScroll() int {
	visible := m.inspectorBodyRowCount()
	if visible < 1 {
		// No diff row is drawn, so there is nowhere to scroll to. Without this
		// the clamp becomes len(diffLines) and Ctrl+D would walk the offset
		// through a pane that shows nothing.
		return 0
	}
	return max(len(m.inspectorDiffLines(m.height < 12))-visible, 0)
}

// inspectorHelpLines is the Inspector's key contract. It lists only keys that
// actually work: q is documented by the spec but not yet wired, and a help screen
// that names a dead key is worse than one that stays quiet about it.
func (m model) inspectorHelpLines() []string {
	lines := []string{
		"  j / k         previous or next changed file",
		"  Ctrl+U / D    scroll the diff by one screen",
	}
	if m.commitInspectorDiffWindow.HasMore {
		lines = append(lines, "  n             load the next part of this diff")
	}
	return append(lines,
		"  Esc           back to the graph",
		"  ?             close this help",
	)
}

// screenHelpBody draws the key contract across the whole body. The header and
// footer stay, and nothing about the selection or the scroll position is touched,
// so closing the help returns to exactly the view it covered.
func screenHelpBody(m model, width, bodyRows int) []string {
	rows := make([]string, 0, bodyRows+1)
	rows = append(rows, fitScreenText("Inspector keys", width))
	for _, line := range m.inspectorHelpLines() {
		rows = append(rows, fitScreenText(line, width))
	}
	for len(rows) < bodyRows+1 {
		rows = append(rows, "")
	}
	return rows[:bodyRows+1]
}

func screenBody(m model, snapshot CommitSnapshot, selected ChangedFile, width, bodyRows int, unsupported bool) []string {
	if bodyRows < 1 {
		return nil
	}
	if m.commitInspectorHelp {
		return screenHelpBody(m, width, bodyRows)
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
	diffLines := m.inspectorDiffLines(unsupported)
	diffOffset := min(max(m.commitInspectorScroll, 0), max(len(diffLines)-bodyRows, 0))
	fileOffset := inspectorFileOffset(m.commitInspectorCursor, len(fileRows), bodyRows)
	for i := 0; i < bodyRows; i++ {
		left, right := "", ""
		if fileIdx := fileOffset + i; fileIdx < len(fileRows) {
			left = fileRows[fileIdx]
		}
		if diffIdx := diffOffset + i; diffIdx < len(diffLines) {
			right = diffLines[diffIdx]
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
