package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
	"hrllk/graphkeeper/internal/state"
	"hrllk/graphkeeper/internal/telemetry"
)

type tagCreatedMsg struct {
	Name   string
	Target string
	Status git.Status
	Err    error
}

type tagToastDoneMsg struct{}

func (m model) handleTagPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.tagPopupOpen = false
		m.tagPopupDraft = ""
		m.tagPopupError = ""
		m.tagPopupTarget = ""
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.tagPopupDraft)
		if name == "" {
			m.tagPopupError = "Tag name is empty."
			return m, nil
		}
		if m.tagPopupTarget == "" {
			m.tagPopupError = "Tag target is missing."
			return m, nil
		}
		m.tagPopupOpen = false
		m.status = loadingToast("Tagging commit...")
		return m, executeCreateTag(m.repo, name, m.tagPopupTarget, m.commitLimit)
	case "backspace":
		if len(m.tagPopupDraft) > 0 {
			runes := []rune(m.tagPopupDraft)
			m.tagPopupDraft = string(runes[:len(runes)-1])
			m.tagPopupError = ""
		}
		return m, nil
	case "delete":
		if len(m.tagPopupDraft) > 0 {
			runes := []rune(m.tagPopupDraft)
			m.tagPopupDraft = string(runes[:len(runes)-1])
			m.tagPopupError = ""
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.tagPopupDraft += string(msg.Runes)
			m.tagPopupError = ""
			return m, nil
		}
	}
	return m, nil
}

func executeCreateTag(repo *git.Repo, name, target string, limit int) tea.Cmd {
	return func() tea.Msg {
		if err := repo.CreateTag(context.Background(), name, target); err != nil {
			return tagCreatedMsg{Name: name, Target: target, Err: err}
		}
		status, err := repo.Status(context.Background(), limit)
		if err == nil {
			status, err = loadLocalTagStatus(repo, status)
		}
		return tagCreatedMsg{Name: name, Target: target, Status: status, Err: err}
	}
}

func (m model) handleTagCreatedMsg(msg tagCreatedMsg) (model, tea.Cmd) {
	if msg.Err != nil {
		reason := state.BlockUnknown
		message := "Tag creation failed."
		detail := msg.Err.Error()
		switch {
		case strings.Contains(msg.Err.Error(), "tag name is empty"):
			message = "Tag name is empty."
			detail = "Enter a tag name."
		case strings.Contains(msg.Err.Error(), "tag target is empty"):
			reason = state.BlockTargetEmpty
			message = "Tag target is missing."
			detail = "Move to a commit before tagging."
		case strings.Contains(msg.Err.Error(), "already exists"):
			message = "Tag already exists."
			detail = "Choose a different tag name."
		}
		m.status = state.New().WithBlocked(reason, message, detail)
		telemetry.Log("app", "tag_create_failed", map[string]string{
			"name":   msg.Name,
			"target": msg.Target,
			"error":  msg.Err.Error(),
		})
		return m, nil
	}

	msg.Status = m.withCachedTagEntries(msg.Status)
	m.repoStatus = msg.Status
	m.storeTagEntries(msg.Status)
	syncBrowseState(&m, msg.Status)
	rows := graph.Rows(msg.Status)
	row := graph.FindRowByHash(rows, msg.Target)
	if row < 0 {
		m.status = state.New().WithBlocked(state.BlockUnknown, "Tag target is missing.", "Refresh the repo and try again.")
		return m, nil
	}
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = row
	m.graphLaneCursor = graph.PointerLane(rows[row])
	hint := repositoryStateHintForModel(&m)
	m.graphScroll = clampScroll(row, len(rows), graphPageSizeForRowsWithHint(&m, rows, row, graphContentHeightForModel(&m), hint != ""))
	m.status = loadingToast("Tag created.")
	telemetry.Log("app", "tag_create", map[string]string{
		"name":   msg.Name,
		"target": msg.Target,
	})
	return m, tea.Tick(900*time.Millisecond, func(time.Time) tea.Msg {
		return tagToastDoneMsg{}
	})
}

func renderTagPopup(m model, bodyWidth, bodyHeight int) string {
	width := popupWidthForBody(bodyWidth, 36, 56)
	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(width).
		Align(lipgloss.Center)
	content := []string{
		fmt.Sprintf("target: %s", shorten(m.tagPopupTarget, 8)),
		fmt.Sprintf("name: %s", m.tagPopupDraft),
	}
	sections := []string{strings.Join(content, "\n")}
	if m.tagPopupError != "" {
		sections = append(sections, warn.Render(m.tagPopupError))
	}
	sections = append(sections, "enter: create", renderPopupFooter(width-4))
	return renderFloatingTitlePopup(popupBox, "Create tag", joinLayoutSections(sections...), width)
}

func handleTagUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tagCreatedMsg:
		return m.handleTagCreatedMsg(msg)
	case tagToastDoneMsg:
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	default:
		return m, nil
	}
}
