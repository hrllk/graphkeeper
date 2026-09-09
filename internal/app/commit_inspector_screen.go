package app

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// The frame's size comes from the model and nowhere else. It used to be passed
// in as well, and the two could disagree: inspectorBodyRowCount derives the
// diff pane's budget from m.width while the renderer drew at the width it was
// handed, so a caller that set one and not the other made the scroll clamp and
// the render disagree about how many rows exist -- and the last diff lines
// became unreachable. Taking the parameters away makes that unwritable.
func renderCommitInspectorScreen(m model) string {
	width := m.width
	height := m.height
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	innerWidth := inspectorInnerWidth(width)
	contentHeight := max(height-2, 1)
	snapshot := m.commitInspectorSnapshot
	if snapshot.FullHash == "" {
		snapshot.FullHash = m.commitInspectorRequestedCommit
		snapshot.Parent = m.commitInspectorRequestedParent
	}
	selected := selectedScreenFile(snapshot, m.commitInspectorCursor)
	header := screenHeaderFor(snapshot, selected, innerWidth, height)
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
// Header rows in the order they are given up. decisions.md (2026-08-03) puts
// commit identity, the selected path and a minimum diff ahead of everything
// else on a short screen, and a header that eats the whole frame leaves
// screenBody with nothing to draw at all -- so rows are dropped from the bottom
// of this list until the diff pane has a row.
const (
	headerRowCommitted = iota // the rarest row: only present when the dates disagree
	headerRowDate
	headerRowParent
	headerRowAuthor
	headerRowPath
	headerRowMessage
	headerRowCommit
)

func screenHeaderFor(snapshot CommitSnapshot, selected ChangedFile, innerWidth, height int) []string {
	rows := screenHeaderRows(snapshot, selected, innerWidth)
	// The first pass drops nothing: the full header is what the frame gets when
	// the frame can afford it.
	for drop := headerRowCommitted - 1; ; drop++ {
		lines := screenHeaderTexts(rows, drop)
		// The ladder has a floor. Author, path, message and the identity row are
		// not negotiable -- a header that drops them buys diff rows by making
		// the Inspector unable to say which commit it is showing, and the
		// heights below the floor are the ones the frame already treats as
		// unsupported.
		if inspectorBodyRowsFor(height, len(lines)) >= 1 || drop >= headerRowParent {
			return lines
		}
	}
}

func screenHeaderLines(snapshot CommitSnapshot, selected ChangedFile, innerWidth int) []string {
	return screenHeaderTexts(screenHeaderRows(snapshot, selected, innerWidth), -1)
}

type screenHeaderRow struct {
	kind int
	text string
}

// screenHeaderTexts renders the rows whose kind outranks dropBelow.
func screenHeaderTexts(rows []screenHeaderRow, dropBelow int) []string {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.kind <= dropBelow || row.text == "" {
			continue
		}
		lines = append(lines, row.text)
	}
	return lines
}

func screenHeaderRows(snapshot CommitSnapshot, selected ChangedFile, innerWidth int) []screenHeaderRow {
	rows := []screenHeaderRow{
		{headerRowCommit, fitScreenText(inspectorCommitText(snapshot.FullHash, innerWidth), innerWidth)},
		{headerRowMessage, fitScreenText(inspectorMessageText(snapshot.Subject, innerWidth), innerWidth)},
		{headerRowAuthor, fitScreenText(inspectorAuthorText(snapshot.AuthorName, snapshot.AuthorEmail, innerWidth), innerWidth)},
		{headerRowParent, fitScreenText(inspectorParentText(snapshot.Parent, snapshot.IsRoot, innerWidth), innerWidth)},
	}
	if when := screenStampText(snapshot.AuthorDate, innerWidth-len("date: ")); when != "" {
		rows = append(rows, screenHeaderRow{headerRowDate, fitScreenText("date: "+when, innerWidth)})
		// A separate committed: row only earns its line when it differs. On an
		// ordinary commit the two stamps are identical and a second copy would
		// cost a diff row for nothing.
		if snapshot.CommitDate != "" && snapshot.CommitDate != snapshot.AuthorDate {
			if committed := screenStampText(snapshot.CommitDate, innerWidth-len("committed: ")); committed != "" {
				rows = append(rows, screenHeaderRow{headerRowCommitted, fitScreenText("committed: "+committed, innerWidth)})
			}
		}
	}
	return append(rows, screenHeaderRow{headerRowPath, fitScreenText("path: "+screenSelectedPath(selected, max(innerWidth-6, 1)), innerWidth)})
}

// screenStampText renders a strict ISO 8601 stamp for the Inspector, which is
// the surface where a commit is confirmed rather than scanned, so it keeps the
// recorded offset when there is room for it.
//
//	budget >= 26  2026-09-08 23:15:16 +09:00
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
// inspectorInnerWidth is what the frame leaves for content: the border and the
// horizontal padding on both sides.
//
// The renderer and the scroll budget both need it, and they used to spell the
// arithmetic out separately. That is the shape of the defect that made the last
// diff lines unreachable -- two places deriving the same number, free to
// disagree -- so it is derived once.
func inspectorInnerWidth(width int) int {
	return max(width-4, 1)
}

func (m model) inspectorBodyRowCount() int {
	innerWidth := inspectorInnerWidth(m.width)
	header := screenHeaderFor(m.commitInspectorSnapshot, ChangedFile{}, innerWidth, m.height)
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
		// At the end, where the content actually stops. This used to be a
		// header line, so it scrolled away before the reader reached the cut --
		// the one place the truncation is visible had nothing to say about it.
		// The footer already carries "n next" from the first frame, so the
		// header line was also telling the reader what the footer had told
		// them.
		lines = append(lines, inspectorTruncationNote(m.commitInspectorDiffWindow.PartialReason))
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

// inspectorTruncationNote says the diff stops here and what to do about it.
//
// The reason used to be printed raw, so the reader saw "partial (line_limit)"
// -- an internal enum spelled in snake_case, the same defect 6.8 took out of
// the hotkey overlay. Four of the five reasons mean one thing to a reader:
// there is more, press n. The fifth does not, and that is the distinction
// worth spending words on: when a single line was too long, pressing n will
// not bring back what is missing from inside it.
func inspectorTruncationNote(reason PartialReason) string {
	if reason == PartialLineTruncated {
		return ellipsis + " a line here was too long to show in full"
	}
	return ellipsis + " more of this diff follows; press n"
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
