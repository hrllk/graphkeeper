package app

import (
	"fmt"

	"hrllk/graphkeeper/internal/git"
)

func stashSummaryLines(entries []git.StashEntry, width int) []string {
	if len(entries) == 0 {
		return nil
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		label := entry.Ref
		if entry.Subject != "" {
			label = fmt.Sprintf("%s - %s", entry.Ref, entry.Subject)
		}
		lines = append(lines, "  - "+shorten(label, max(width-4, 0)))
	}
	return lines
}
