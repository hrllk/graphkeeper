package app

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"hrllk/graphkeeper/internal/state"
)

type mergeConfirmViewModel struct {
	CurrentBranch string
	TargetRef     string
	CurrentOnly   int
	TargetOnly    int
	ImpactKnown   bool
	MergeText     string
	RebaseText    string
	RiskText      string
	Disabled      bool
	DisabledText  string
}

func projectMergeConfirm(snapshot PullImpactSnapshot, impacts PullImpactSet, stale bool) (mergeConfirmViewModel, bool) {
	if !impacts.Valid || strings.TrimSpace(snapshot.UpstreamRef) == "" || snapshot.Ahead < 0 || snapshot.Behind < 0 {
		return mergeConfirmViewModel{}, false
	}
	view := mergeConfirmViewModel{
		CurrentBranch: snapshot.CurrentRef,
		TargetRef:     snapshot.UpstreamRef,
		CurrentOnly:   snapshot.Ahead,
		TargetOnly:    snapshot.Behind,
		ImpactKnown:   impacts.Valid,
		MergeText:     impacts.MergeSummary,
		RebaseText:    impacts.RebaseSummary,
		RiskText:      joinUniqueNonEmpty(impacts.MergeRisk, impacts.RebaseRisk),
		Disabled:      stale,
	}
	if stale {
		view.DisabledText = "Pull preview is stale.\nRefresh before continuing."
	}
	return view, true
}

func joinUniqueNonEmpty(values ...string) string {
	seen := make(map[string]struct{}, len(values))
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		parts = append(parts, value)
	}
	return strings.Join(parts, " ")
}

func applyMergeConfirmProjection(m *model, snapshot PullImpactSnapshot, impacts PullImpactSet, stale bool) bool {
	view, ok := projectMergeConfirm(snapshot, impacts, stale)
	if !ok {
		clearPullConfirmProjection(m)
		return false
	}
	m.pullConfirmInput = &pullConfirmInput{
		CurrentBranch: view.CurrentBranch,
		TargetRef:     view.TargetRef,
		CurrentOnly:   view.CurrentOnly,
		TargetOnly:    view.TargetOnly,
		ImpactKnown:   view.ImpactKnown,
		MergeText:     view.MergeText,
		RebaseText:    view.RebaseText,
		RiskText:      view.RiskText,
	}
	m.mergeConfirmView = &view
	m.status = m.status.WithConfirm(state.ActionPull, "Pull into "+view.CurrentBranch+"?", mergeConfirmBody(view, mergeMax(40, m.width-8)))
	m.status.Title = "Pull into " + view.CurrentBranch + "?"
	return true
}

func mergeMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func mergeConfirmBody(view mergeConfirmViewModel, width int) string {
	if view.Disabled {
		return fitMergeLines(view.DisabledText, width)
	}
	lines := []string{
		"Pull into " + fitMergeValue(view.CurrentBranch, width, "Pull into "),
		"Target: " + fitMergeValue(view.TargetRef, width, "Target: "),
		"Relation: " + fitMergeCounts(view.CurrentOnly, view.TargetOnly, width),
		"",
		"Merge: " + fitMergeValue(view.MergeText, width, "Merge: "),
		"Rebase: " + fitMergeValue(view.RebaseText, width, "Rebase: "),
	}
	if view.RiskText != "" {
		lines = append(lines, "Risk: "+fitMergeValue(view.RiskText, width, "Risk: "))
	}
	return strings.Join(lines, "\n")
}

func fitMergeLines(value string, width int) string {
	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = fitVisibleWidth(lines[i], width)
	}
	return strings.Join(lines, "\n")
}

func fitMergeValue(value string, width int, label string) string {
	available := width - lipgloss.Width(label)
	if available < 1 {
		return "…"
	}
	plain := ansi.Strip(value)
	if lipgloss.Width(plain) <= available {
		return plain
	}
	if available == 1 {
		return "…"
	}
	left := (available - 1 + 1) / 2
	right := available - 1 - left
	return takeMergeCellsLeft(plain, left) + "…" + takeMergeCellsRight(plain, right)
}

func takeMergeCellsLeft(value string, cells int) string {
	result := ""
	for _, r := range []rune(value) {
		part := string(r)
		if lipgloss.Width(result+part) > cells {
			break
		}
		result += part
	}
	return result
}

func takeMergeCellsRight(value string, cells int) string {
	result := ""
	runes := []rune(value)
	for i := len(runes) - 1; i >= 0; i-- {
		part := string(runes[i])
		if lipgloss.Width(part+result) > cells {
			break
		}
		result = part + result
	}
	return result
}

func fitMergeCounts(current, target, width int) string {
	value := formatMergeCounts(current, target)
	return fitMergeValue(value, width, "Relation: ")
}

func formatMergeCounts(current, target int) string {
	return strconv.Itoa(current) + "/" + strconv.Itoa(target) + " commits"
}
