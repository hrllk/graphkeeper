package app

import "fmt"

func (m model) renderContextContent(width, height int) string {
	return renderContextProjection(m.contextProjection(width), width, height)
}

func renderContextProjection(p ContextProjection, width, height int) string {
	if height <= 0 {
		return ""
	}
	infoLines := renderContextViewport(p.InfoLines, max(height-1, 0), p.Scroll, width)
	leftLines := append([]string{renderSectionTitle(p.Title + " Details")}, infoLines...)
	actions := append([]string(nil), p.ActionLines...)
	if p.Recommendation != nil {
		r := p.Recommendation
		actions = append([]string{
			fmt.Sprintf("upstream: %s", r.Upstream),
			fmt.Sprintf("local-only: %d  upstream-only: %d", r.LocalOnly, r.UpstreamOnly),
			"p: fetch, then m: merge / r: rebase",
		}, actions...)
	}
	rightLines := append([]string{renderSectionTitle(p.Title + " Actions")}, actions...)
	rightLines = indentLines(rightLines, 1)
	return renderSplitColumns(leftLines, rightLines, width, height)
}

func renderContextViewport(lines []string, height, offset, width int) []string {
	if height <= 0 || len(lines) <= height {
		return lines
	}
	if height == 1 {
		return []string{fitVisibleWidth(muted.Render(fmt.Sprintf("… +%d hidden", len(lines)-1)), width)}
	}
	visible := height - 1
	maxOffset := len(lines) - visible
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	view := append([]string(nil), lines[offset:offset+visible]...)
	hidden := len(lines) - visible
	indicator := fitVisibleWidth(muted.Render(fmt.Sprintf("… +%d hidden", hidden)), width)
	if offset > 0 {
		if visible == 1 {
			return view
		}
		return append([]string{indicator}, view[len(view)-(visible-1):]...)
	}
	return append(view, indicator)
}
