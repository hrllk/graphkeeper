package app

import (
	ci "hrllk/graphkeeper/internal/commitinspector"
	"hrllk/graphkeeper/internal/git"
)

// normalizeInspectorWindow is the contract's clamp. The app used to carry its
// own character-for-character copy.
func normalizeInspectorWindow(window DiffWindowRequest) (DiffWindowRequest, *InspectorError) {
	return ci.NormalizeDiffWindow(window)
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
