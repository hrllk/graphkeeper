package app

import (
	"regexp"
	"sort"
	"strings"

	"hrllk/graphkeeper/internal/git"
	"hrllk/graphkeeper/internal/graph"
)

type graphSearchEntry struct {
	Hash     string
	Title    string
	Branches []string
	Row      int
	Score    int
}

func buildGraphSearchIndex(rs git.Status) []graphSearchEntry {
	rows := graphRows(rs)
	index := make([]graphSearchEntry, 0, len(rows))
	for i, row := range rows {
		if row.Commit.Hash == "" || row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
			continue
		}

		entry := graphSearchEntry{
			Hash:  row.Commit.Hash,
			Title: row.Commit.Subject,
			Row:   i,
		}

		seen := make(map[string]struct{}, len(row.Commit.Decorations))
		for _, decoration := range row.Commit.Decorations {
			decoration = strings.TrimSpace(decoration)
			if decoration == "" || strings.HasPrefix(decoration, "tag: ") {
				continue
			}
			if strings.HasPrefix(decoration, "HEAD -> ") {
				decoration = strings.TrimPrefix(decoration, "HEAD -> ")
			}
			decoration = strings.TrimPrefix(decoration, "origin/")
			if _, ok := seen[decoration]; ok {
				continue
			}
			seen[decoration] = struct{}{}
			entry.Branches = append(entry.Branches, decoration)
		}

		index = append(index, entry)
	}
	return index
}

func graphSearchMatches(index []graphSearchEntry, q string) []graphSearchEntry {
	pattern, literal, regexOK := compileGraphSearchQuery(q)
	matches := make([]graphSearchEntry, 0, len(index))
	for _, entry := range index {
		if score, ok := scoreGraphSearchEntry(entry, q, pattern, literal, regexOK); ok {
			entry.Score = score
			matches = append(matches, entry)
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		if matches[i].Row != matches[j].Row {
			return matches[i].Row < matches[j].Row
		}
		return matches[i].Hash < matches[j].Hash
	})
	return matches
}

func compileGraphSearchQuery(q string) (*regexp.Regexp, string, bool) {
	raw := strings.TrimSpace(q)
	if raw == "" {
		return nil, "", false
	}
	if re, err := regexp.Compile("(?i)" + raw); err == nil {
		return re, "", true
	}
	return nil, strings.ToLower(raw), false
}

func scoreGraphSearchEntry(entry graphSearchEntry, q string, pattern *regexp.Regexp, literal string, regexOK bool) (int, bool) {
	if strings.TrimSpace(q) == "" {
		return 0, false
	}
	hash := strings.ToLower(entry.Hash)
	title := strings.ToLower(entry.Title)

	if regexOK && pattern != nil {
		if start, end, ok := matchBounds(pattern, hash); ok {
			return 4000 - start*10 - (end - start), true
		}
		if start, end, ok := bestMatchBounds(pattern, entry.Branches); ok {
			return 3000 - start*10 - (end - start), true
		}
		if start, end, ok := matchBounds(pattern, title); ok {
			return 2000 - start*10 - (end - start), true
		}
		return 0, false
	}

	switch {
	case strings.HasPrefix(hash, literal):
		return 3000 + len(hash) - len(literal), true
	case anyPrefix(entry.Branches, literal):
		return 2200, true
	case anyContains(entry.Branches, literal):
		return 2000, true
	case strings.Contains(title, literal):
		return 1000, true
	default:
		return 0, false
	}
}

func matchBounds(re *regexp.Regexp, value string) (int, int, bool) {
	if re == nil || value == "" {
		return 0, 0, false
	}
	loc := re.FindStringIndex(value)
	if len(loc) != 2 {
		return 0, 0, false
	}
	return loc[0], loc[1], true
}

func bestMatchBounds(re *regexp.Regexp, values []string) (int, int, bool) {
	bestStart := 0
	bestEnd := 0
	found := false
	for _, value := range values {
		start, end, ok := matchBounds(re, strings.ToLower(value))
		if !ok {
			continue
		}
		if !found || start < bestStart || (start == bestStart && end-start > bestEnd-bestStart) {
			bestStart = start
			bestEnd = end
			found = true
		}
	}
	return bestStart, bestEnd, found
}

func anyPrefix(items []string, q string) bool {
	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item), q) {
			return true
		}
	}
	return false
}

func anyContains(items []string, q string) bool {
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), q) {
			return true
		}
	}
	return false
}

func applyGraphSearchJump(m model, entry graphSearchEntry) model {
	rows := graphRows(m.repoStatus)
	if entry.Row < 0 || entry.Row >= len(rows) {
		return m
	}

	m.graphSearchOpen = false
	m.graphSearchDraft = ""
	m.graphSearchError = ""
	m.graphSearchIndex = buildGraphSearchIndex(m.repoStatus)
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = entry.Row

	hint := repositoryStateHintForModel(&m)
	page := graphPageSizeForRowsWithHint(&m, rows, entry.Row, graphContentHeightForModel(&m), hint != "")
	m.graphScroll = clampScroll(entry.Row, len(rows), page)
	m.graphLaneCursor = graph.PointerLane(rows[entry.Row])
	return m
}

func applyGraphSearchSelection(m model) model {
	query := strings.TrimSpace(m.graphSearchDraft)
	index := buildGraphSearchIndex(m.repoStatus)
	m.graphSearchIndex = index

	if query == "" {
		m.graphSearchOpen = false
		m.graphSearchDraft = ""
		m.graphSearchQuery = ""
		m.graphSearchCursor = 0
		m.graphSearchError = ""
		return m
	}

	matches := graphSearchMatches(index, query)
	if len(matches) == 0 {
		m.graphSearchError = "No graph match."
		m.graphSearchOpen = true
		return m
	}

	m.graphSearchQuery = query
	m.graphSearchCursor = 0
	return applyGraphSearchJump(m, matches[0])
}

func applyGraphSearchRepeat(m model, delta int) model {
	query := strings.TrimSpace(m.graphSearchQuery)
	if query == "" {
		return m
	}

	index := buildGraphSearchIndex(m.repoStatus)
	m.graphSearchIndex = index
	matches := graphSearchMatches(index, query)
	if len(matches) == 0 {
		return m
	}

	cursor := m.graphSearchCursor
	if cursor < 0 || cursor >= len(matches) {
		cursor = 0
	}
	cursor = (cursor + delta) % len(matches)
	if cursor < 0 {
		cursor += len(matches)
	}

	m.graphSearchCursor = cursor
	m.graphSearchError = ""
	return applyGraphSearchJump(m, matches[cursor])
}
