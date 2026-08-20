package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"hrllk/graphkeeper/internal/git"
)

type gitCommitInspectorReader struct{ repo *git.Repo }

func newGitCommitInspectorReader(repo *git.Repo) CommitInspectorReader {
	return &gitCommitInspectorReader{repo: repo}
}

func inspectorError(err error) *InspectorError {
	if err == nil {
		return nil
	}
	kind := "reader"
	if errors.Is(err, context.DeadlineExceeded) {
		kind = "timeout"
	}
	if strings.Contains(err.Error(), "malformed diff") || strings.Contains(err.Error(), "parse:") {
		kind = "parse"
	}
	if strings.Contains(err.Error(), "configuration") {
		kind = "configuration"
	}
	if strings.Contains(err.Error(), "git ") && kind != "timeout" && kind != "configuration" && kind != "parse" {
		kind = "git_exit"
	}
	return &InspectorError{Kind: kind, Message: sanitizeInspectorError(err.Error()), Retryable: kind == "timeout" || kind == "reader"}
}

func sanitizeInspectorError(message string) string {
	message = strings.ReplaceAll(message, "\n", " ")
	if len(message) > 256 {
		return message[:256]
	}
	return message
}

func (r *gitCommitInspectorReader) InspectCommit(ctx context.Context, req CommitRequest) InspectorResult[CommitSnapshot] {
	if r == nil || r.repo == nil {
		return InspectorResult[CommitSnapshot]{State: PaneError, Error: &InspectorError{Kind: "configuration", Message: "commit inspector reader is not configured"}, Commit: req.Commit, RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch}
	}
	inspection, err := r.repo.InspectCommit(ctx, req.Commit)
	result := InspectorResult[CommitSnapshot]{Commit: req.Commit, RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			result.State = PaneCanceled
		} else {
			result.State = PaneError
			result.Error = inspectorError(err)
		}
		return result
	}
	files := make([]ChangedFile, 0, len(inspection.Files))
	seen := make(map[string]struct{}, len(inspection.Files))
	for index, file := range inspection.Files {
		stableID := canonicalInspectorFileID(req.Commit, inspection.Parent, file, index)
		if _, exists := seen[stableID]; exists {
			result.State = PaneError
			result.Error = &InspectorError{Kind: "identity", Message: "duplicate file identity"}
			return result
		}
		seen[stableID] = struct{}{}
		files = append(files, ChangedFile{StableID: stableID, Status: mapInspectorStatus(file), OldPath: file.OldPath, Path: file.Path, Additions: file.Additions, Deletions: file.Deletions, Binary: file.Binary})
	}
	authorName, authorEmail := splitInspectorAuthor(inspection.Author)
	result.State = PaneReady
	result.Value = CommitSnapshot{FullHash: inspection.Hash, Subject: inspection.Subject, AuthorName: authorName, AuthorEmail: authorEmail, MessageBody: inspection.Message, Parent: inspection.Parent, IsRoot: inspection.IsRoot, Files: files}
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

func normalizeInspectorWindow(window DiffWindowRequest) (DiffWindowRequest, *InspectorError) {
	if window.StartLine < 0 || window.MaxLines < 0 || window.MaxBytes < 0 || window.MaxLines > maxInspectorMaxLines || window.MaxBytes > maxInspectorMaxBytes {
		return window, &InspectorError{Kind: "configuration", Message: "invalid inspector diff window"}
	}
	if window.MaxLines == 0 {
		window.MaxLines = defaultInspectorMaxLines
	}
	if window.MaxBytes == 0 {
		window.MaxBytes = defaultInspectorMaxBytes
	}
	if window.MaxLines < 1 || window.MaxBytes < 16 {
		return window, &InspectorError{Kind: "configuration", Message: "inspector diff window cannot fit a structural record"}
	}
	return window, nil
}

func (r *gitCommitInspectorReader) LoadDiff(ctx context.Context, req DiffRequest) InspectorResult[DiffWindow] {
	normalized, configErr := normalizeInspectorWindow(req.Window)
	if configErr != nil {
		return InspectorResult[DiffWindow]{State: PaneError, Error: configErr, Commit: req.Commit, Parent: req.Parent, FileID: req.FileID, RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Window: req.Window}
	}
	req.Window = normalized
	result := InspectorResult[DiffWindow]{Commit: req.Commit, Parent: req.Parent, FileID: req.FileID, RequestID: req.RequestID, RepositoryEpoch: req.RepositoryEpoch, Window: req.Window}
	if r == nil || r.repo == nil {
		result.State, result.Error = PaneError, &InspectorError{Kind: "configuration", Message: "commit inspector reader is not configured"}
		return result
	}
	inspection, err := r.repo.InspectCommit(ctx, req.Commit)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			result.State = PaneCanceled
		} else {
			result.State = PaneError
			result.Error = inspectorError(err)
		}
		return result
	}
	var file git.CommitDiffFile
	found := false
	for index, candidate := range inspection.Files {
		stableID := canonicalInspectorFileID(req.Commit, inspection.Parent, candidate, index)
		if stableID == req.FileID {
			file, found = candidate, true
			break
		}
	}
	if !found {
		result.State, result.Error = PaneError, &InspectorError{Kind: "file_not_found", Message: fmt.Sprintf("file %q was not found", req.FileID)}
		return result
	}
	result.FileID = req.FileID
	diff, err := r.repo.CommitDiffWindow(ctx, inspection, file, req.Window.MaxLines, req.Window.StartLine, req.Window.MaxBytes)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			result.State = PaneCanceled
		} else {
			result.State = PaneError
			result.Error = inspectorError(err)
		}
		return result
	}
	lines := diff.Lines
	boundedRows := diff.Rows
	partialReason := PartialReason(diff.PartialReason)
	window := DiffWindow{FileID: req.FileID, HasMore: diff.HasMore, PartialReason: partialReason, Hunks: diffWindowHunks(lines, boundedRows), NextStartLine: diff.NextStartLine}
	result.Value, result.State = window, PaneReady
	if diff.HasMore {
		result.State = PanePartial
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

func canonicalInspectorFileID(commit, parent string, file git.CommitDiffFile, occurrence int) string {
	canonical := strings.Join([]string{commit, parent, file.Status, file.OldPath, file.Path, strconv.Itoa(occurrence)}, "\x00")
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

func mapInspectorStatus(file git.CommitDiffFile) ChangedFileStatus {
	switch file.Status {
	case "A":
		return StatusAdded
	case "M":
		return StatusModified
	case "D":
		return StatusDeleted
	case "R":
		return StatusRenamed
	case "C":
		return StatusCopied
	case "S":
		return StatusSubmodule
	case "ModeOnly":
		return StatusModeOnly
	default:
		return StatusModified
	}
}

func diffWindowHunks(lines []string, rows []git.DiffRow) []DiffHunk {
	hunks := []DiffHunk{}
	current := -1
	rowIndex := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			hunks = append(hunks, DiffHunk{Header: line})
			current++
			continue
		}
		if current >= 0 && rowIndex < len(rows) && (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ")) && !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") {
			row := rows[rowIndex]
			hunks[current].Rows = append(hunks[current].Rows, PairedRow{Kind: row.Kind, From: CodeLine{Number: row.OldLine, Text: row.From}, To: CodeLine{Number: row.NewLine, Text: row.To}, FromPresent: row.FromPresent, ToPresent: row.ToPresent})
			rowIndex++
		}
	}
	if len(hunks) == 0 {
		if len(rows) == 0 {
			return []DiffHunk{}
		}
		continued := DiffHunk{Header: "@@ (continued)"}
		for _, row := range rows {
			continued.Rows = append(continued.Rows, PairedRow{Kind: row.Kind, From: CodeLine{Number: row.OldLine, Text: row.From}, To: CodeLine{Number: row.NewLine, Text: row.To}, FromPresent: row.FromPresent, ToPresent: row.ToPresent})
		}
		return []DiffHunk{continued}
	}
	return hunks
}

func newModelWithInspectorReader(repo *git.Repo, reader CommitInspectorReader) model {
	return model{repo: repo, inspectorReader: reader}
}

func (m model) inspector() CommitInspectorReader {
	return m.inspectorReader
}

func invalidateCommitInspectorForEpoch(m model) model {
	if m.commitInspectorCancel != nil {
		m.commitInspectorCancel()
		m.commitInspectorCancel = nil
	}
	m.commitInspectorMetadataLoading = false
	m.commitInspectorDiffLoading = false
	m.commitInspectorLoading = false
	m.commitInspectorStale = true
	return m
}
