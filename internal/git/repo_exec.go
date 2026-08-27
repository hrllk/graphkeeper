package git

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const remoteOperationTimeout = 30 * time.Second

func (r *Repo) runRemote(ctx context.Context, args ...string) (string, error) {
	runner := r.runner
	runner.Timeout = remoteOperationTimeout
	return runner.RunContext(ctx, args...)
}

func (r *Repo) Fetch(ctx context.Context) error {
	_, err := r.runRemote(ctx, "fetch", "--all", "--prune", "--tags")
	return err
}

// FetchContext is the context-aware fetch entry point used by outbound
// operation adapters. Fetch remains the compatibility name for callers that
// already pass a context.
func (r *Repo) FetchContext(ctx context.Context) error { return r.Fetch(ctx) }

func (r *Repo) Push(ctx context.Context, branch string, force bool, setUpstream bool) (string, error) {
	args := []string{"push"}
	if force {
		args = append(args, "--force")
	}
	if setUpstream {
		args = append(args, "-u", "origin", branch)
	}
	return r.runRemote(ctx, args...)
}

func (r *Repo) PushTag(ctx context.Context, tag string) (string, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", fmt.Errorf("tag is empty")
	}
	return r.runRemote(ctx, "push", "origin", tag)
}

func (r *Repo) DeleteBranch(ctx context.Context, branch string) (string, error) {
	return r.Run("branch", "-D", branch)
}

func (r *Repo) DeleteTag(ctx context.Context, name string) (string, error) {
	return r.Run("tag", "-d", name)
}

func (r *Repo) DeleteRemoteTag(ctx context.Context, remote, name string) (string, error) {
	return r.runRemote(ctx, "push", remote, "--delete", name)
}

func (r *Repo) DeleteRemoteBranch(ctx context.Context, remote, branch string) (string, error) {
	return r.runRemote(ctx, "push", remote, "--delete", branch)
}

func (r *Repo) CreateTag(ctx context.Context, name, target string) error {
	name = strings.TrimSpace(name)
	target = strings.TrimSpace(target)
	if name == "" {
		return fmt.Errorf("tag name is empty")
	}
	if target == "" {
		return fmt.Errorf("tag target is empty")
	}
	_, err := r.git(ctx, "tag", name, target)
	return err
}

func (r *Repo) FetchTags(ctx context.Context) error {
	_, err := r.runRemote(ctx, "fetch", "origin", "--tags")
	return err
}

func (r *Repo) StashAll(ctx context.Context, message string) error {
	_, err := r.git(ctx, "stash", "push", "--include-untracked", "-m", message)
	return err
}

func (r *Repo) StashPop(ctx context.Context, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("stash ref is empty")
	}
	_, err := r.git(ctx, "stash", "pop", ref)
	return err
}

func (r *Repo) CleanWorkingTree(ctx context.Context, includeIgnored bool) error {
	if _, err := r.git(ctx, "reset", "--hard"); err != nil {
		return err
	}
	args := []string{"clean", "-fd"}
	if includeIgnored {
		args = append(args, "-x")
	}
	_, err := r.git(ctx, args...)
	return err
}

func (r *Repo) worktreeDirty(ctx context.Context) (bool, error) {
	out, err := r.git(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (r *Repo) Stashes(ctx context.Context) ([]StashEntry, error) {
	lines, err := r.gitLines(ctx, "stash", "list", "--format=%gd%x1f%H%x1f%P%x1f%gs")
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, nil
	}
	entries := make([]StashEntry, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "\x1f", 4)
		if len(parts) < 4 {
			continue
		}
		parents := strings.Fields(parts[2])
		baseHash := ""
		if len(parents) > 0 {
			baseHash = parents[0]
		}
		entries = append(entries, StashEntry{
			Ref:      strings.TrimSpace(parts[0]),
			Hash:     strings.TrimSpace(parts[1]),
			BaseHash: baseHash,
			Subject:  strings.TrimSpace(parts[3]),
		})
	}
	return entries, nil
}

// InspectCommit returns read-only commit metadata and the changed file list.
// Paths are parsed from Git's NUL-delimited output so whitespace and newlines
// in filenames do not change the file identity.
func (r *Repo) InspectCommit(ctx context.Context, hash string) (CommitInspection, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return CommitInspection{}, fmt.Errorf("commit hash is empty")
	}
	meta, err := r.git(ctx, "show", "-s", "--format=%H%x00%P%x00%an%x00%ae%x00%s", hash)
	if err != nil {
		return CommitInspection{}, err
	}
	parts := strings.SplitN(meta, "\x00", 5)
	if len(parts) != 5 {
		return CommitInspection{}, fmt.Errorf("invalid commit metadata")
	}
	parents := strings.Fields(parts[1])
	parent := ""
	if len(parents) > 0 {
		parent = parents[0]
	}
	message, err := r.gitRaw(ctx, "show", "-s", "--format=%B", hash)
	if err != nil {
		return CommitInspection{}, err
	}
	commit := strings.TrimSpace(parts[0])
	// The listing must use the same explicit <parent> <commit> basis as the patch.
	// Passing the commit alone relies on Git's implicit merge diff semantics, which
	// emit nothing for a merge commit and leave the file list empty.
	raw, err := r.gitRaw(ctx, commitDiffTreeArgs(parent, commit, "--name-status", "-z", "-r", "-M", "-C")...)
	if err != nil {
		return CommitInspection{}, err
	}
	files := parseCommitDiffFiles(raw)
	author := strings.TrimSpace(parts[2])
	if email := strings.TrimSpace(parts[3]); email != "" {
		author += " <" + email + ">"
	}
	r.annotateCommitDiffFiles(ctx, files, parent, commit)
	return CommitInspection{
		Hash: commit, Subject: sanitizeTerminalText(parts[4]), Author: sanitizeTerminalText(author),
		Message: sanitizeTerminalText(message), Parent: parent, IsRoot: parent == "", Parents: parents, Files: files,
	}, nil
}

// commitDiffTreeArgs builds a diff-tree invocation for one commit.
//
// The parent selection is the load-bearing part and the reason this is shared.
// A root commit has no parent and relies on --root. Every other commit, merge
// commits included, must pass <parent> <commit> explicitly: with the commit
// alone Git falls back to its implicit merge diff semantics and emits nothing
// for a merge, which is how the changed-files pane came to render empty.
// Callers append their own "--" <path> arguments after this.
func commitDiffTreeArgs(parent, commit string, mode ...string) []string {
	args := append([]string{"diff-tree", "--root", "--no-commit-id"}, mode...)
	if parent == "" {
		return append(args, commit)
	}
	return append(args, parent, commit)
}

func (r *Repo) annotateCommitDiffFiles(ctx context.Context, files []CommitDiffFile, parent, commit string) {
	out, err := r.gitRaw(ctx, commitDiffTreeArgs(parent, commit, "--numstat", "-z", "-r")...)
	if err != nil {
		return
	}
	parts := strings.Split(out, "\x00")
	byPath := make(map[string][2]string, len(files))
	for i := 0; i+2 < len(parts); i += 3 {
		fields := strings.Fields(parts[i])
		if len(fields) < 2 {
			continue
		}
		byPath[parts[i+2]] = [2]string{fields[0], fields[1]}
	}
	for i := range files {
		stat, ok := byPath[files[i].Path]
		if !ok {
			// Rename/copy numstat records can retain the old path. The
			// status listing remains authoritative for identity; leave counts
			// at zero when Git does not emit a direct path record.
			continue
		}
		if stat[0] == "-" && stat[1] == "-" {
			files[i].Binary = true
			continue
		}
		files[i].Additions, _ = strconv.Atoi(stat[0])
		files[i].Deletions, _ = strconv.Atoi(stat[1])
	}
	summaryArgs := commitDiffTreeArgs(parent, commit, "--summary", "-r")
	if summary, summaryErr := r.gitRaw(ctx, summaryArgs...); summaryErr == nil {
		for i := range files {
			for _, line := range strings.Split(summary, "\n") {
				if !strings.Contains(line, files[i].Path) {
					continue
				}
				if strings.Contains(line, "Submodule") {
					files[i].Status = "S"
				} else if strings.Contains(line, "mode change") && files[i].Additions == 0 && files[i].Deletions == 0 {
					files[i].Status = "ModeOnly"
				}
				break
			}
		}
	}
}

func (r *Repo) CommitDiff(ctx context.Context, inspection CommitInspection, file CommitDiffFile, maxLines int, startLine int) (CommitDiff, error) {
	return r.CommitDiffWindow(ctx, inspection, file, maxLines, startLine, 1<<20)
}

func (r *Repo) CommitDiffWindow(ctx context.Context, inspection CommitInspection, file CommitDiffFile, maxLines int, startLine int, maxBytes int) (CommitDiff, error) {
	parent := ""
	if len(inspection.Parents) > 0 {
		parent = inspection.Parents[0]
	}
	args := commitDiffTreeArgs(parent, inspection.Hash, "--full-index", "--no-ext-diff", "--unified=80", "-p", "-M", "-C")
	args = append(args, "--", file.Path)
	raw, truncated, err := r.gitRawLogicalWindow(ctx, int64(maxBytes), startLine, args...)
	if err != nil {
		return CommitDiff{}, err
	}
	if strings.Contains(raw, incompletePairMarker) {
		return CommitDiff{}, fmt.Errorf("configuration: bounded window split an indivisible pair")
	}
	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	rawStart := 0
	end := len(lines)
	hasMore := truncated
	logical := 0
	usedBytes := 0
	for i := rawStart; i < len(lines); i++ {
		line := lines[i]
		if maxLines > 0 && isLogicalDiffLine(line) && logical >= maxLines {
			end, hasMore = i, true
			break
		}
		if maxBytes > 0 {
			next := usedBytes + len(line) + 1
			if next > maxBytes {
				end, hasMore = i, true
				break
			}
			usedBytes = next
		}
		if isLogicalDiffLine(line) {
			logical++
		}
	}
	if end < len(lines) && end > rawStart && isLogicalDiffLine(lines[end-1]) && (strings.HasPrefix(lines[end-1], "-") || strings.HasPrefix(lines[end-1], "+")) {
		groupStart := end - 1
		for groupStart > rawStart && (strings.HasPrefix(lines[groupStart-1], "-") || strings.HasPrefix(lines[groupStart-1], "+")) {
			groupStart--
		}
		groupEnd := end
		for groupEnd < len(lines) && (strings.HasPrefix(lines[groupEnd], "-") || strings.HasPrefix(lines[groupEnd], "+")) {
			groupEnd++
		}
		if groupEnd > end {
			groupLines := lines[groupStart:groupEnd]
			groupCount := logicalDiffLineCount(groupLines)
			beforeCount := logicalDiffLineCount(lines[rawStart:groupStart])
			beforeBytes := diffPayloadBytes(lines[rawStart:groupStart])
			if (maxLines > 0 && groupCount > maxLines) || (maxBytes > 0 && diffPayloadBytes(groupLines) > maxBytes) {
				return CommitDiff{}, fmt.Errorf("configuration: indivisible diff pair exceeds window budget")
			}
			fitsRemaining := (maxLines <= 0 || beforeCount+groupCount <= maxLines) && (maxBytes <= 0 || beforeBytes+diffPayloadBytes(groupLines) <= maxBytes)
			if fitsRemaining {
				end = groupEnd
			} else {
				end, hasMore = groupStart, true
			}
		}
	}
	parseLines := append([]string(nil), lines[rawStart:end]...)
	if header := activeHunkHeader(lines, rawStart); header != "" && (rawStart == len(lines) || !strings.HasPrefix(lines[rawStart], "@@")) {
		parseLines = append([]string{header}, parseLines...)
	}
	rows, err := parseCommitDiffRowsStrict(parseLines)
	if err != nil {
		return CommitDiff{}, err
	}
	windowLines := parseLines
	if hasHunkHeader(windowLines) && logicalDiffLineCount(windowLines) == 0 {
		return CommitDiff{}, fmt.Errorf("configuration: hunk header cannot fit a structural line")
	}
	if maxBytes > 0 {
		for _, line := range windowLines {
			if strings.HasPrefix(line, "@@") && maxBytes < len(line)+len("… [line truncated]\\n") {
				return CommitDiff{}, fmt.Errorf("configuration: hunk header and minimum placeholder exceed window budget")
			}
		}
	}
	if maxBytes > 0 && diffPayloadBytes(windowLines) > maxBytes {
		return CommitDiff{}, fmt.Errorf("configuration: hunk header and first structural line exceed window budget")
	}
	next := startLine + logicalDiffLineCount(lines[rawStart:end])
	reason := ""
	if hasMore {
		reason = "line_limit"
		lineTruncated := false
		for _, line := range windowLines {
			if strings.Contains(line, "[line truncated]") {
				lineTruncated = true
				break
			}
		}
		if lineTruncated {
			reason = "line_truncated"
		} else if truncated {
			reason = "byte_limit"
		} else if maxBytes > 0 && diffPayloadBytes(windowLines) >= maxBytes {
			reason = "byte_limit"
		}
	}
	return CommitDiff{FileID: file.ID, Lines: windowLines, Rows: rows, HasMore: hasMore, PartialReason: reason, NextStartLine: next}, nil
}

func hasHunkHeader(lines []string) bool {
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			return true
		}
	}
	return false
}
func diffPayloadBytes(lines []string) int {
	n := 0
	for _, line := range lines {
		n += len(line) + 1
	}
	return n
}
func logicalDiffLineCount(lines []string) int {
	n := 0
	for _, line := range lines {
		if isLogicalDiffLine(line) {
			n++
		}
	}
	return n
}
func activeHunkHeader(lines []string, at int) string {
	for i := at - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "@@") {
			return lines[i]
		}
		if strings.HasPrefix(lines[i], "diff ") {
			break
		}
	}
	return ""
}
func isLogicalDiffLine(line string) bool {
	return (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ")) && !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---")
}

func rawIndexForLogicalDiffLine(lines []string, logicalStart int) int {
	if logicalStart < 0 {
		logicalStart = 0
	}
	seen := 0
	for i, line := range lines {
		if isLogicalDiffLine(line) {
			if seen >= logicalStart {
				return i
			}
			seen++
		}
	}
	return len(lines)
}

func rowsForDiffLineWindow(lines []string, start, end int, rows []DiffRow) []DiffRow {
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		start = end
	}
	countMarkers := func(input []string) int {
		n := 0
		for _, line := range input {
			if (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ")) && !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") {
				n++
			}
		}
		return n
	}
	from, to := countMarkers(lines[:start]), countMarkers(lines[:end])
	if from > len(rows) {
		from = len(rows)
	}
	if to > len(rows) {
		to = len(rows)
	}
	if to < from {
		to = from
	}
	return rows[from:to]
}

// parseCommitDiffRows turns unified diff hunks into deterministic paired rows.
// It intentionally uses line structure only; syntax highlighting is a later layer.
var diffHunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

func parseCommitDiffRowsStrict(lines []string) ([]DiffRow, error) {
	type numberedLine struct {
		text string
		num  int
	}
	rows := make([]DiffRow, 0)
	oldLine, newLine := 0, 0
	removed, added := make([]numberedLine, 0), make([]numberedLine, 0)
	flush := func() {
		for i := 0; i < len(removed) || i < len(added); i++ {
			row := DiffRow{Kind: "modified"}
			if i < len(removed) {
				row.From, row.OldLine, row.FromPresent = removed[i].text, removed[i].num, true
			}
			if i < len(added) {
				row.To, row.NewLine, row.ToPresent = added[i].text, added[i].num, true
			}
			if !row.FromPresent {
				row.Kind = "added"
			}
			if !row.ToPresent {
				row.Kind = "removed"
			}
			rows = append(rows, row)
		}
		removed, added = nil, nil
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			flush()
			matches := diffHunkHeader.FindStringSubmatch(line)
			if matches == nil {
				return nil, fmt.Errorf("malformed diff hunk header")
			}
			oldLine, _ = strconv.Atoi(matches[1])
			newLine, _ = strconv.Atoi(matches[3])
			continue
		}
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "Binary files") {
			continue
		}
		if strings.HasPrefix(line, " ") {
			flush()
			text := line[1:]
			rows = append(rows, DiffRow{Kind: "context", OldLine: oldLine, NewLine: newLine, From: text, To: text, FromPresent: true, ToPresent: true})
			oldLine++
			newLine++
		} else if strings.HasPrefix(line, "-") {
			removed = append(removed, numberedLine{text: line[1:], num: oldLine})
			oldLine++
		} else if strings.HasPrefix(line, "+") {
			added = append(added, numberedLine{text: line[1:], num: newLine})
			newLine++
		} else if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "\\ No newline") {
			return nil, fmt.Errorf("malformed diff record")
		}
	}
	flush()
	return rows, nil
}

func parseCommitDiffRows(lines []string) []DiffRow {
	rows, _ := parseCommitDiffRowsStrict(lines)
	return rows
}

// ParseDiffRows is the read-only projection used by the terminal Inspector.
func ParseDiffRows(lines []string) []DiffRow { return parseCommitDiffRows(lines) }

func parseCommitDiffFiles(raw string) []CommitDiffFile {
	parts := strings.Split(raw, "\x00")
	files := make([]CommitDiffFile, 0)
	for i := 0; i < len(parts); i++ {
		status := parts[i]
		if status == "" {
			continue
		}
		fields := strings.Fields(status)
		if len(fields) == 0 {
			continue
		}
		code := fields[0]
		if code == "commit" || strings.HasPrefix(code, "tree") {
			continue
		}
		file := CommitDiffFile{Status: string(code[0])}
		switch code[0] {
		case 'R', 'C':
			if i+2 >= len(parts) {
				continue
			}
			rawOld, rawPath := parts[i+1], parts[i+2]
			file.OldPath, file.Path = sanitizeTerminalText(rawOld), sanitizeTerminalText(rawPath)
			file.ID = stableCommitFileID(file.Status, rawOld, rawPath)
			i += 2
		default:
			if i+1 >= len(parts) {
				continue
			}
			rawPath := parts[i+1]
			file.Path = sanitizeTerminalText(rawPath)
			file.ID = stableCommitFileID(file.Status, "", rawPath)
			i++
		}
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

func stableCommitFileID(status, oldPath, path string) string {
	h := sha256.Sum256([]byte(status + "\x00" + oldPath + "\x00" + path))
	return fmt.Sprintf("%x", h[:])
}

func sanitizeTerminalText(value string) string {
	if !utf8.ValidString(value) {
		value = strings.ToValidUTF8(value, "�")
	}
	var b strings.Builder
	escape := false
	for i := 0; i < len(value); i++ {
		c := value[i]
		if escape {
			if c == '[' {
				continue
			}
			if c >= '@' && c <= '~' {
				escape = false
			}
			continue
		}
		if c == 0x1b {
			escape = true
			continue
		}
		if c < 0x20 && c != '\n' && c != '\t' {
			continue
		}
		if c == 0x7f {
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func (r *Repo) TagEntries(ctx context.Context) ([]TagEntry, error) {
	entries, err := r.LocalTagEntries(ctx)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	originTags, _ := r.OriginTagSet(ctx)
	for i := range entries {
		entries[i].OriginKnown = true
		entries[i].OnOrigin = originTags[entries[i].Name]
	}
	return entries, nil
}

func (r *Repo) LocalTagEntries(ctx context.Context) ([]TagEntry, error) {
	names, err := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/tags")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, nil
	}
	entries := make([]TagEntry, 0, len(names))
	for _, name := range names {
		target, err := r.git(ctx, "rev-parse", "--verify", name+"^{commit}")
		if err != nil {
			continue
		}
		subject, err := r.git(ctx, "show", "-s", "--format=%s", target)
		if err != nil {
			continue
		}
		relativeAge, err := r.git(ctx, "show", "-s", "--format=%cr", target)
		if err != nil {
			continue
		}
		commitUnixText, err := r.git(ctx, "show", "-s", "--format=%ct", target)
		if err != nil {
			continue
		}
		commitUnix, err := strconv.ParseInt(strings.TrimSpace(commitUnixText), 10, 64)
		if err != nil {
			continue
		}

		tagType, taggerName, taggerDate, tagMessage, err := r.tagMetadata(ctx, name)
		if err != nil {
			continue
		}

		entry := TagEntry{
			Name:        strings.TrimSpace(name),
			CommitHash:  strings.TrimSpace(target),
			Subject:     strings.TrimSpace(subject),
			RelativeAge: strings.TrimSpace(relativeAge),
			CommitUnix:  commitUnix,
			Tagger:      strings.TrimSpace(taggerName),
			Message:     strings.TrimSpace(tagMessage),
		}
		if strings.TrimSpace(tagType) == "tag" {
			entry.Annotated = true
			if ts := strings.TrimSpace(taggerDate); ts != "" {
				if unix, err := strconv.ParseInt(ts, 10, 64); err == nil {
					entry.TaggedAt = time.Unix(unix, 0)
				}
			}
		} else {
			entry.Tagger = "lightweight"
			entry.TaggedAt = time.Unix(commitUnix, 0)
		}
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].CommitUnix != entries[j].CommitUnix {
			return entries[i].CommitUnix > entries[j].CommitUnix
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func (r *Repo) tagMetadata(ctx context.Context, name string) (tagType, taggerName, taggerDate, message string, err error) {
	if tagType, err = r.git(ctx, "for-each-ref", "--format=%(objecttype)", "refs/tags/"+name); err != nil {
		return "", "", "", "", err
	}
	if taggerName, err = r.git(ctx, "for-each-ref", "--format=%(taggername)", "refs/tags/"+name); err != nil {
		return "", "", "", "", err
	}
	if taggerDate, err = r.git(ctx, "for-each-ref", "--format=%(taggerdate:unix)", "refs/tags/"+name); err != nil {
		return "", "", "", "", err
	}
	if strings.TrimSpace(tagType) == "tag" {
		if message, err = r.git(ctx, "for-each-ref", "--format=%(contents:subject)", "refs/tags/"+name); err != nil {
			return "", "", "", "", err
		}
	}
	return strings.TrimSpace(tagType), strings.TrimSpace(taggerName), strings.TrimSpace(taggerDate), strings.TrimSpace(message), nil
}

func (r *Repo) OriginTagSet(ctx context.Context) (map[string]bool, error) {
	output, err := r.runRemote(ctx, "ls-remote", "--tags", "origin")
	if err != nil {
		return nil, err
	}
	lines := splitRawLines(output)
	if len(lines) == 0 {
		return nil, nil
	}
	tags := make(map[string]bool, len(lines))
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ref := strings.TrimPrefix(parts[1], "refs/tags/")
		ref = strings.TrimSuffix(ref, "^{}")
		if ref == "" {
			continue
		}
		tags[ref] = true
	}
	return tags, nil
}

func (r *Repo) Divergence(ctx context.Context, left, right string) (leftOnly int, rightOnly int, err error) {
	if left == "" || right == "" {
		return 0, 0, fmt.Errorf("divergence requires two refs")
	}
	out, err := r.git(ctx, "rev-list", "--left-right", "--count", left+"..."+right)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Fields(out)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected divergence output: %q", out)
	}
	_, scanErr := fmt.Sscanf(parts[0], "%d", &leftOnly)
	if scanErr != nil {
		return 0, 0, scanErr
	}
	_, scanErr = fmt.Sscanf(parts[1], "%d", &rightOnly)
	if scanErr != nil {
		return 0, 0, scanErr
	}
	return leftOnly, rightOnly, nil
}

func (r *Repo) MergeBase(ctx context.Context, left, right string) (string, error) {
	if left == "" || right == "" {
		return "", fmt.Errorf("merge base requires two refs")
	}
	out, err := r.git(ctx, "merge-base", left, right)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r *Repo) Run(args ...string) (string, error) {
	return r.runner.Run(args...)
}

// RunContext lets outbound adapters connect cancellation to the git process.
func (r *Repo) RunContext(ctx context.Context, args ...string) (string, error) {
	return r.runner.RunContext(ctx, args...)
}

func (r *Repo) currentBranch(ctx context.Context) (string, error) {
	out, err := r.git(ctx, "branch", "--show-current")
	if err == nil && out != "" {
		return out, nil
	}
	out, err = r.git(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return out, nil
}

func (r *Repo) git(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		msg := sanitizeTerminalText(strings.TrimSpace(stderr.String()))
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return out, nil
}

func (r *Repo) gitLines(ctx context.Context, args ...string) ([]string, error) {
	out, err := r.git(ctx, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	lines := strings.Split(out, "\n")
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		if s := strings.TrimSpace(line); s != "" {
			trimmed = append(trimmed, s)
		}
	}
	return trimmed, nil
}

func (r *Repo) gitRawLines(ctx context.Context, args ...string) ([]string, error) {
	out, err := r.gitRaw(ctx, args...)
	if err != nil {
		return nil, err
	}
	return splitRawLines(out), nil
}

func (r *Repo) gitRaw(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := stdout.String()
	if err != nil {
		msg := sanitizeTerminalText(strings.TrimSpace(stderr.String()))
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return out, nil
}

const incompletePairMarker = "__GRAPHKEEPER_INCOMPLETE_PAIR__"

func (r *Repo) gitRawLogicalWindow(ctx context.Context, maxBytes int64, startLine int, args ...string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.root
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", false, err
	}
	if err := cmd.Start(); err != nil {
		return "", false, err
	}
	stopKiller := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
			select {
			case <-time.After(3 * time.Second):
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			case <-stopKiller:
			}
		case <-stopKiller:
		}
	}()
	defer close(stopKiller)
	reader := bufio.NewReader(stdout)
	var out strings.Builder
	started := false
	truncated := false
	logical := 0
	activeHeader := ""
	var streamErr error
	lineBuf := make([]byte, 0, 1<<20)
	lineOverlong := false
	overflowPair := false
	processLine := func(raw []byte, overlong bool) {
		text := strings.TrimRight(string(raw), "\n")
		isPairLine := strings.HasPrefix(text, "-") || strings.HasPrefix(text, "+")
		if overflowPair && !isPairLine {
			out.WriteString(incompletePairMarker)
			out.WriteByte('\n')
			overflowPair = false
		}
		if strings.HasPrefix(text, "@@") {
			activeHeader = text
		}
		logicalLine := isLogicalDiffLine(text)
		if !started && logical >= max(startLine, 0) && logicalLine {
			started = true
			if activeHeader != "" {
				out.WriteString(activeHeader)
				out.WriteByte('\n')
			}
		}
		if started && (logicalLine || strings.HasPrefix(text, "\\") || strings.HasPrefix(text, "@@")) {
			payload := text + "\n"
			if logicalLine && (overlong || int64(len(raw)) > maxBytes) {
				payload = "… [line truncated]\n"
				truncated = true
			}
			if int64(out.Len()+len(payload)) <= maxBytes+4096 {
				out.WriteString(payload)
			} else {
				truncated = true
				if isPairLine {
					overflowPair = true
				}
			}
		}
		if logicalLine {
			logical++
		}
	}
	for {
		part, readErr := reader.ReadSlice('\n')
		if len(part) > 0 {
			if len(lineBuf) == 0 {
				lineOverlong = false
			}
			room := int(maxBytes+4096) - len(lineBuf)
			if room > 0 {
				take := part
				if len(take) > room {
					take = take[:room]
					lineOverlong = true
				}
				lineBuf = append(lineBuf, take...)
			} else {
				lineOverlong = true
			}
		}
		if readErr == bufio.ErrBufferFull {
			continue
		}
		if len(lineBuf) > 0 {
			processLine(lineBuf, lineOverlong)
		}
		lineBuf = lineBuf[:0]
		if readErr != nil {
			if readErr != io.EOF {
				streamErr = readErr
			}
			break
		}
	}
	if overflowPair {
		out.WriteString(incompletePairMarker)
		out.WriteByte('\n')
	}
	waitErr := cmd.Wait()
	if streamErr != nil {
		return out.String(), truncated, streamErr
	}
	if ctx.Err() != nil {
		return out.String(), truncated, ctx.Err()
	}
	if waitErr != nil {
		msg := sanitizeTerminalText(strings.TrimSpace(stderr.String()))
		if msg == "" {
			msg = waitErr.Error()
		}
		return out.String(), truncated, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), waitErr, msg)
	}
	return out.String(), truncated, nil
}

func (r *Repo) gitRawBounded(ctx context.Context, maxBytes int64, args ...string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.root
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", false, err
	}
	if err := cmd.Start(); err != nil {
		return "", false, err
	}
	readDone := make(chan struct{})
	var data []byte
	var readErr error
	go func() { data, readErr = io.ReadAll(io.LimitReader(stdout, maxBytes+1)); close(readDone) }()
	select {
	case <-readDone:
	case <-ctx.Done():
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		select {
		case <-readDone:
		case <-time.After(3 * time.Second):
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			<-readDone
		}
	}
	go func() { _, _ = io.Copy(io.Discard, stdout) }()
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	var waitErr error
	select {
	case waitErr = <-waitDone:
	case <-time.After(3 * time.Second):
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		waitErr = <-waitDone
	}
	truncated := int64(len(data)) > maxBytes
	if truncated {
		data = data[:maxBytes]
	}
	if readErr != nil {
		return string(data), truncated, readErr
	}
	if ctx.Err() != nil {
		return string(data), truncated, ctx.Err()
	}
	if waitErr != nil {
		msg := sanitizeTerminalText(strings.TrimSpace(stderr.String()))
		if msg == "" {
			msg = waitErr.Error()
		}
		return string(data), truncated, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), waitErr, msg)
	}
	return string(data), truncated, nil
}

func splitRawLines(out string) []string {
	out = strings.TrimRight(out, "\n")
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		filtered = append(filtered, line)
	}
	return filtered
}

func isNoCommits(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "does not have any commits yet") ||
		strings.Contains(err.Error(), "unknown revision or path not in the working tree")
}

func splitDecorations(v string) []string {
	parts := strings.Split(v, ", ")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (r *Runner) Run(args ...string) (string, error) {
	return r.RunContext(context.Background(), args...)
}

func (r *Runner) RunContext(ctx context.Context, args ...string) (string, error) {
	if r.Timeout <= 0 {
		r.Timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := sanitizeTerminalText(strings.TrimSpace(stderr.String()))
		if msg == "" {
			msg = err.Error()
		}
		return strings.TrimSpace(stdout.String()), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (r *Repo) MustRoot() string {
	return r.root
}
