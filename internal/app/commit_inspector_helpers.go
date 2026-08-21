package app

import (
	"hrllk/graphkeeper/internal/git"
)

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

func inspectorLogicalLineCount(lines []string) int {
	count := 0
	for _, line := range lines {
		if (len(line) > 0 && (line[0] == '+' || line[0] == '-' || line[0] == ' ')) && len(line) >= 3 && line[:3] != "+++" && line[:3] != "---" {
			count++
		}
	}
	return count
}

func newModelWithInspectorReader(repo *git.Repo, reader CommitInspectorReader) model {
	return model{repo: repo, inspectorReader: reader}
}
func (m model) inspector() CommitInspectorReader { return m.inspectorReader }

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
