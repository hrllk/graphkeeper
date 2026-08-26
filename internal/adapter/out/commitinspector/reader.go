package commitinspectoradapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"hrllk/graphkeeper/internal/commitinspector"
	"hrllk/graphkeeper/internal/git"
)

const (
	defaultInspectorMaxLines = 2000
	defaultInspectorMaxBytes = 1 << 20
	maxInspectorMaxLines     = 10000
	maxInspectorMaxBytes     = 16 << 20
)

type gitCommitInspectorReader struct{ repo *git.Repo }

func New(repo *git.Repo) commitinspector.CommitInspectorReader {
	return &gitCommitInspectorReader{repo: repo}
}

func inspectorError(err error) *commitinspector.InspectorError {
	if err == nil {
		return nil
	}
	kind := "reader"
	if errors.Is(err, git.ErrCommitNotFound) {
		kind = "commit_not_found"
	}
	if errors.Is(err, context.Canceled) {
		kind = "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		kind = "timeout"
	}
	if kind == "reader" && (strings.Contains(err.Error(), "malformed diff") || strings.Contains(err.Error(), "parse:")) {
		kind = "parse"
	}
	if kind == "reader" && strings.Contains(err.Error(), "configuration") {
		kind = "configuration"
	}
	if kind == "reader" && strings.Contains(err.Error(), "git ") {
		kind = "git_exit"
	}
	return &commitinspector.InspectorError{Kind: kind, Message: sanitizeInspectorError(err.Error()), Retryable: kind == "timeout" || kind == "reader"}
}

func sanitizeInspectorError(message string) string {
	message = strings.ReplaceAll(message, "\n", " ")
	if len(message) > 256 {
		return message[:256]
	}
	return message
}

func (r *gitCommitInspectorReader) InspectCommit(ctx context.Context, req commitinspector.CommitRequest) commitinspector.InspectorResult[commitinspector.CommitSnapshot] {
	if r == nil || r.repo == nil {
		return commitinspector.InspectorResult[commitinspector.CommitSnapshot]{State: commitinspector.PaneError, Error: &commitinspector.InspectorError{Kind: "configuration", Message: "commit inspector reader is not configured"}, Commit: req.Commit, RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch}
	}
	inspection, err := r.repo.InspectCommit(ctx, req.Commit)
	result := commitinspector.InspectorResult[commitinspector.CommitSnapshot]{Commit: req.Commit, RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			result.State = commitinspector.PaneCanceled
		} else {
			result.State = commitinspector.PaneError
			result.Error = inspectorError(err)
		}
		return result
	}
	files := make([]commitinspector.ChangedFile, 0, len(inspection.Files))
	seen := make(map[string]struct{}, len(inspection.Files))
	seenCanonical := make(map[string]struct{}, len(inspection.Files))
	occurrences := inspectorFileOccurrences(inspection.Files)
	for index, file := range inspection.Files {
		occurrence := occurrences[index]
		stableID := canonicalInspectorFileID(req.Commit, inspection.Parent, file, occurrence)
		canonicalKey := canonicalInspectorFileKey(mapInspectorStatus(file), file.OldPath, file.Path, occurrence)
		if _, exists := seen[stableID]; exists {
			result.State = commitinspector.PaneError
			result.Error = &commitinspector.InspectorError{Kind: "identity", Message: "duplicate file identity"}
			return result
		}
		if _, exists := seenCanonical[canonicalKey]; exists {
			result.State = commitinspector.PaneError
			result.Error = &commitinspector.InspectorError{Kind: "identity", Message: "duplicate canonical file identity"}
			return result
		}
		seen[stableID] = struct{}{}
		seenCanonical[canonicalKey] = struct{}{}
		files = append(files, commitinspector.ChangedFile{StableID: stableID, CanonicalKey: canonicalKey, Status: mapInspectorStatus(file), OldPath: file.OldPath, Path: file.Path, Additions: file.Additions, Deletions: file.Deletions, Binary: file.Binary})
	}
	authorName, authorEmail := splitInspectorAuthor(inspection.Author)
	result.State = commitinspector.PaneReady
	result.Value = commitinspector.CommitSnapshot{FullHash: inspection.Hash, Subject: inspection.Subject, AuthorName: authorName, AuthorEmail: authorEmail, MessageBody: inspection.Message, Parent: inspection.Parent, IsRoot: inspection.IsRoot, Files: files}
	result.Parent = inspection.Parent
	return result
}

func inspectorLogicalLineCount(lines []string) int {
	count := 0
	for _, line := range lines {
		if (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ")) && !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") {
			count++
		}
	}
	return count
}

func normalizeInspectorWindow(window commitinspector.DiffWindowRequest) (commitinspector.DiffWindowRequest, *commitinspector.InspectorError) {
	if window.StartLine < 0 || window.MaxLines < 0 || window.MaxBytes < 0 || window.MaxLines > maxInspectorMaxLines || window.MaxBytes > maxInspectorMaxBytes {
		return window, &commitinspector.InspectorError{Kind: "configuration", Message: "invalid inspector diff window"}
	}
	if window.MaxLines == 0 {
		window.MaxLines = defaultInspectorMaxLines
	}
	if window.MaxBytes == 0 {
		window.MaxBytes = defaultInspectorMaxBytes
	}
	if window.MaxLines < 1 || window.MaxBytes < 16 {
		return window, &commitinspector.InspectorError{Kind: "configuration", Message: "inspector diff window cannot fit a structural record"}
	}
	return window, nil
}

func (r *gitCommitInspectorReader) LoadDiff(ctx context.Context, req commitinspector.DiffRequest) commitinspector.InspectorResult[commitinspector.DiffWindow] {
	normalized, configErr := normalizeInspectorWindow(req.Window)
	if configErr != nil {
		return commitinspector.InspectorResult[commitinspector.DiffWindow]{State: commitinspector.PaneError, Error: configErr, Commit: req.Commit, Parent: req.Parent, FileID: req.FileID, RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Window: req.Window}
	}
	req.Window = normalized
	result := commitinspector.InspectorResult[commitinspector.DiffWindow]{Commit: req.Commit, Parent: req.Parent, FileID: req.FileID, RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Window: req.Window}
	if r == nil || r.repo == nil {
		result.State, result.Error = commitinspector.PaneError, &commitinspector.InspectorError{Kind: "configuration", Message: "commit inspector reader is not configured"}
		return result
	}
	inspection, err := r.repo.InspectCommit(ctx, req.Commit)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			result.State = commitinspector.PaneCanceled
		} else {
			result.State = commitinspector.PaneError
			result.Error = inspectorError(err)
		}
		return result
	}
	var file git.CommitDiffFile
	resolvedID := ""
	found := false
	stableMatches := 0
	canonicalMatches := 0
	occurrences := inspectorFileOccurrences(inspection.Files)
	for index, candidate := range inspection.Files {
		occurrence := occurrences[index]
		stableID := canonicalInspectorFileID(req.Commit, inspection.Parent, candidate, occurrence)
		if stableID == req.FileID {
			file, resolvedID, stableMatches = candidate, stableID, stableMatches+1
		}
	}
	if stableMatches > 1 {
		result.State, result.Error = commitinspector.PaneError, &commitinspector.InspectorError{Kind: "identity", Message: "duplicate stable file identity"}
		return result
	}
	if stableMatches == 1 {
		found = true
	}
	if !found && req.CanonicalKey != "" {
		for index, candidate := range inspection.Files {
			key := canonicalInspectorFileKey(mapInspectorStatus(candidate), candidate.OldPath, candidate.Path, occurrences[index])
			if key == req.CanonicalKey {
				file, resolvedID, canonicalMatches = candidate, canonicalInspectorFileID(req.Commit, inspection.Parent, candidate, occurrences[index]), canonicalMatches+1
			}
		}
		if canonicalMatches > 1 {
			result.State, result.Error = commitinspector.PaneError, &commitinspector.InspectorError{Kind: "identity", Message: "ambiguous canonical file identity"}
			return result
		}
		found = canonicalMatches == 1
	}
	if !found {
		result.State, result.Error = commitinspector.PaneError, &commitinspector.InspectorError{Kind: "file_not_found", Message: fmt.Sprintf("file %q was not found", req.FileID)}
		return result
	}
	result.FileID = resolvedID
	diff, err := r.repo.CommitDiffWindow(ctx, inspection, file, req.Window.MaxLines, req.Window.StartLine, req.Window.MaxBytes)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			result.State = commitinspector.PaneCanceled
		} else {
			result.State = commitinspector.PaneError
			result.Error = inspectorError(err)
		}
		return result
	}
	lines := diff.Lines
	boundedRows := diff.Rows
	partialReason := commitinspector.PartialReason(diff.PartialReason)
	window := commitinspector.DiffWindow{FileID: resolvedID, HasMore: diff.HasMore, PartialReason: partialReason, Hunks: diffWindowHunks(lines, boundedRows), NextStartLine: diff.NextStartLine}
	result.Value, result.State = window, commitinspector.PaneReady
	if diff.HasMore {
		result.State = commitinspector.PanePartial
	}
	return result
}

func splitInspectorAuthor(author string) (string, string) {
	end := strings.LastIndex(author, ">")
	start := strings.LastIndex(author[:max(end, 0)], "<")
	if start >= 0 && end > start {
		return strings.TrimSpace(author[:start]), strings.TrimSpace(author[start+1 : end])
	}
	return strings.TrimSpace(author), ""
}

func inspectorFileOccurrences(files []git.CommitDiffFile) []int {
	occurrences := make([]int, len(files))
	seen := make(map[string]int, len(files))
	for i, file := range files {
		key := strings.Join([]string{file.Status, file.OldPath, file.Path}, "\x00")
		occurrences[i] = seen[key]
		seen[key]++
	}
	return occurrences
}

func canonicalInspectorFileKey(status commitinspector.ChangedFileStatus, oldPath, path string, occurrence int) string {
	parts := []string{string(status), oldPath, path, strconv.Itoa(occurrence)}
	var b strings.Builder
	for _, part := range parts {
		b.WriteString(strconv.Itoa(len(part)))
		b.WriteByte(':')
		b.WriteString(part)
	}
	return b.String()
}

func canonicalInspectorFileID(commit, parent string, file git.CommitDiffFile, occurrence int) string {
	if file.ID != "" {
		return file.ID
	}
	canonical := strings.Join([]string{commit, parent, file.Status, file.OldPath, file.Path, strconv.Itoa(occurrence)}, "\x00")
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

func mapInspectorStatus(file git.CommitDiffFile) commitinspector.ChangedFileStatus {
	switch file.Status {
	case "A":
		return commitinspector.StatusAdded
	case "M":
		return commitinspector.StatusModified
	case "D":
		return commitinspector.StatusDeleted
	case "R":
		return commitinspector.StatusRenamed
	case "C":
		return commitinspector.StatusCopied
	case "B":
		return commitinspector.StatusBinary
	case "S":
		return commitinspector.StatusSubmodule
	case "ModeOnly":
		return commitinspector.StatusModeOnly
	default:
		return commitinspector.StatusModified
	}
}

func diffWindowHunks(lines []string, rows []git.DiffRow) []commitinspector.DiffHunk {
	hunks := []commitinspector.DiffHunk{}
	current := -1
	rowIndex := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			hunks = append(hunks, commitinspector.DiffHunk{Header: line})
			current++
			continue
		}
		if current >= 0 && rowIndex < len(rows) && (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ")) && !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") {
			row := rows[rowIndex]
			hunks[current].Rows = append(hunks[current].Rows, commitinspector.PairedRow{Kind: row.Kind, From: commitinspector.CodeLine{Number: row.OldLine, Text: row.From}, To: commitinspector.CodeLine{Number: row.NewLine, Text: row.To}, FromPresent: row.FromPresent, ToPresent: row.ToPresent})
			rowIndex++
		}
	}
	if len(hunks) == 0 {
		if len(rows) == 0 {
			return []commitinspector.DiffHunk{}
		}
		continued := commitinspector.DiffHunk{Header: "@@ (continued)"}
		for _, row := range rows {
			continued.Rows = append(continued.Rows, commitinspector.PairedRow{Kind: row.Kind, From: commitinspector.CodeLine{Number: row.OldLine, Text: row.From}, To: commitinspector.CodeLine{Number: row.NewLine, Text: row.To}, FromPresent: row.FromPresent, ToPresent: row.ToPresent})
		}
		return []commitinspector.DiffHunk{continued}
	}
	return hunks
}
