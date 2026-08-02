package app

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
)

const (
	inspectorFilesPane = iota
	inspectorDiffPane
	inspectorDiffPage = 18
)

func inspectCommit(repo *git.Repo, hash string) tea.Cmd {
	return func() tea.Msg {
		inspection, err := repo.InspectCommit(context.Background(), hash)
		return commitInspectorLoadedMsg{inspection: inspection, err: err}
	}
}

func loadCommitInspectorDiff(repo *git.Repo, inspection git.CommitInspection, index int) tea.Cmd {
	return func() tea.Msg {
		if index < 0 || index >= len(inspection.Files) {
			return commitInspectorDiffMsg{err: fmt.Errorf("file selection is out of range")}
		}
		diff, err := repo.CommitDiff(context.Background(), inspection, inspection.Files[index], 2000, 0)
		return commitInspectorDiffMsg{diff: diff, err: err}
	}
}

func (m model) handleCommitInspectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		if m.commitInspectorPane == inspectorDiffPane {
			m.commitInspectorPane = inspectorFilesPane
			return m, nil
		}
		m.commitInspectorOpen = false
		m.commitInspectorLines = nil
		return m, nil
	}
	if m.commitInspectorLoading {
		return m, nil
	}
	switch msg.String() {
	case "tab":
		if m.commitInspectorPane == inspectorFilesPane {
			m.commitInspectorPane = inspectorDiffPane
		}
		return m, nil
	case "j", "down":
		if m.commitInspectorPane == inspectorFilesPane && len(m.commitInspector.Files) > 0 {
			if m.commitInspectorCursor < len(m.commitInspector.Files)-1 {
				m.commitInspectorCursor++
				m.commitInspectorScroll = clampScroll(m.commitInspectorCursor, len(m.commitInspector.Files), inspectorDiffPage)
				m.commitInspectorLoading = true
				return m, loadCommitInspectorDiff(m.repo, m.commitInspector, m.commitInspectorCursor)
			}
		} else {
			m.commitInspectorScroll++
		}
	case "k", "up":
		if m.commitInspectorPane == inspectorFilesPane && m.commitInspectorCursor > 0 {
			m.commitInspectorCursor--
			m.commitInspectorScroll = clampScroll(m.commitInspectorCursor, len(m.commitInspector.Files), inspectorDiffPage)
			m.commitInspectorLoading = true
			return m, loadCommitInspectorDiff(m.repo, m.commitInspector, m.commitInspectorCursor)
		}
	case "ctrl+u":
		m.commitInspectorScroll -= inspectorDiffPage
		if m.commitInspectorScroll < 0 {
			m.commitInspectorScroll = 0
		}
	case "ctrl+d":
		m.commitInspectorScroll += inspectorDiffPage
	case "h":
		// Files pane has no horizontal action; diff uses a plain text viewport.
		return m, nil
	case "l":
		return m, nil
	case "enter":
		if m.commitInspectorPane == inspectorFilesPane && len(m.commitInspector.Files) > 0 {
			m.commitInspectorPane = inspectorDiffPane
		}
	}
	return m, nil
}

func (m model) renderCommitInspectorPopup(width, height int) string {
	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}
	innerWidth := width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}
	lines := []string{
		truncateInspector("Commit Inspector  "+m.commitInspector.Hash+"  "+m.commitInspector.Subject, innerWidth),
		"Files (j/k)                                      Diff (Tab)",
	}
	if m.commitInspectorLoading {
		lines = append(lines, "Loading commit changes...")
	} else if m.commitInspectorError != "" {
		lines = append(lines, "Error: "+m.commitInspectorError, "Esc: close")
	} else if len(m.commitInspector.Files) == 0 {
		lines = append(lines, "No changed files.", "Esc: close")
	} else {
		for i, file := range m.commitInspector.Files {
			if i < m.commitInspectorScroll || i >= m.commitInspectorScroll+height-6 {
				continue
			}
			marker := " "
			if i == m.commitInspectorCursor {
				marker = ">"
			}
			lines = append(lines, fmt.Sprintf("%s %s %s", marker, inspectorStatus(file.Status), inspectorTreePath(file.Path)))
		}
		if m.commitInspectorPane == inspectorDiffPane {
			lines = append(lines, "", "--- diff ---")
			start := m.commitInspectorScroll
			if start > len(m.commitInspectorLines) {
				start = len(m.commitInspectorLines)
			}
			end := start + height - len(lines) - 3
			if end > len(m.commitInspectorLines) {
				end = len(m.commitInspectorLines)
			}
			for _, line := range m.commitInspectorLines[start:end] {
				lines = append(lines, truncateInspector(line, innerWidth))
			}
			if m.commitInspectorHasMore {
				lines = append(lines, "... more diff lines; Ctrl+D to continue")
			}
		}
	}
	lines = append(lines, "", "Enter select  Tab pane  j/k move  Ctrl+U/D page  Esc back")
	return strings.Join(lines, "\n")
}

func inspectorTreePath(path string) string {
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	if len(parts) <= 1 {
		return path
	}
	return strings.Repeat("  ", len(parts)-1) + "└ " + parts[len(parts)-1]
}

func inspectorStatus(status string) string {
	switch status {
	case "A":
		return "A"
	case "D":
		return "D"
	case "R":
		return "R"
	case "C":
		return "C"
	default:
		return "M"
	}
}

func truncateInspector(s string, width int) string {
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width < 4 {
		return string(r[:width])
	}
	return string(r[:width-3]) + "..."
}
