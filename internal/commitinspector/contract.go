package commitinspector

import "context"

type InspectorPaneState string

const (
	PaneIdle     InspectorPaneState = "idle"
	PaneLoading  InspectorPaneState = "loading"
	PaneReady    InspectorPaneState = "ready"
	PanePartial  InspectorPaneState = "partial"
	PaneError    InspectorPaneState = "error"
	PaneCanceled InspectorPaneState = "canceled"
)

type PartialReason string

const (
	PartialByteLimit     PartialReason = "byte_limit"
	PartialLineLimit     PartialReason = "line_limit"
	PartialLineTruncated PartialReason = "line_truncated"
	PartialProcessLimit  PartialReason = "process_limit"
	// PartialIndivisiblePair marks a window that stopped before a paired
	// removed/added run because the run alone exceeds the budget. Splitting it
	// would re-pair its rows across windows, so the run is deferred whole.
	PartialIndivisiblePair PartialReason = "indivisible_pair"
)

type ChangedFileStatus string

const (
	StatusAdded     ChangedFileStatus = "added"
	StatusModified  ChangedFileStatus = "modified"
	StatusDeleted   ChangedFileStatus = "deleted"
	StatusRenamed   ChangedFileStatus = "renamed"
	StatusCopied    ChangedFileStatus = "copied"
	StatusBinary    ChangedFileStatus = "binary"
	StatusModeOnly  ChangedFileStatus = "mode_only"
	StatusSubmodule ChangedFileStatus = "submodule"
)

type InspectorError struct {
	Kind      string
	Message   string
	Retryable bool
}

// The diff window's limits. They were declared here and then declared again,
// identically, in internal/app and in the adapter -- and bypassed by literals
// in three more places, so the contract package's copy was unused while three
// other copies decided the behaviour.
//
// A limit that is written down four times is four limits.
const (
	DefaultDiffWindowLines = 2000
	DefaultDiffWindowBytes = 1 << 20
	MaxDiffWindowLines     = 10000
	MaxDiffWindowBytes     = 16 << 20

	// minDiffWindowBytes is the smallest window that can still hold one
	// structural record, below which the request is a configuration error
	// rather than a small window.
	minDiffWindowBytes = 16
)

// DefaultDiffWindow is the window a caller gets by asking for nothing.
func DefaultDiffWindow() DiffWindowRequest {
	return DiffWindowRequest{StartLine: 0, MaxLines: DefaultDiffWindowLines, MaxBytes: DefaultDiffWindowBytes}
}

// NormalizeDiffWindow fills in the defaults and rejects a window that cannot
// work. The app and the adapter each had their own copy of this, character for
// character; one copy is what makes them agree by construction rather than by
// coincidence.
func NormalizeDiffWindow(window DiffWindowRequest) (DiffWindowRequest, *InspectorError) {
	if window.StartLine < 0 || window.MaxLines < 0 || window.MaxBytes < 0 ||
		window.MaxLines > MaxDiffWindowLines || window.MaxBytes > MaxDiffWindowBytes {
		return window, &InspectorError{Kind: "configuration", Message: "invalid inspector diff window"}
	}
	if window.MaxLines == 0 {
		window.MaxLines = DefaultDiffWindowLines
	}
	if window.MaxBytes == 0 {
		window.MaxBytes = DefaultDiffWindowBytes
	}
	if window.MaxLines < 1 || window.MaxBytes < minDiffWindowBytes {
		return window, &InspectorError{Kind: "configuration", Message: "inspector diff window cannot fit a structural record"}
	}
	return window, nil
}

type DiffWindowRequest struct {
	StartLine int
	MaxLines  int
	MaxBytes  int
}

type InspectorResult[T any] struct {
	State           InspectorPaneState
	Value           T
	Error           *InspectorError
	Commit          string
	Parent          string
	FileID          string
	RequestID       uint64
	RepositoryEpoch uint64
	Window          DiffWindowRequest
}

type CommitRequest struct {
	Commit          string
	RequestID       uint64
	RepositoryEpoch uint64
}

type DiffRequest struct {
	Commit          string
	Parent          string
	FileID          string
	RequestID       uint64
	RepositoryEpoch uint64
	Window          DiffWindowRequest
}

type CommitInspectorReader interface {
	InspectCommit(context.Context, CommitRequest) InspectorResult[CommitSnapshot]
	LoadDiff(context.Context, DiffRequest) InspectorResult[DiffWindow]
}

type CommitSnapshot struct {
	FullHash    string
	Subject     string
	AuthorName  string
	AuthorEmail string
	// AuthorDate and CommitDate are strict ISO 8601 with the commit's own UTC
	// offset. They are strings because this package holds no time arithmetic and
	// may not import anything that would let it grow some.
	AuthorDate  string
	CommitDate  string
	MessageBody string
	Parent      string
	IsRoot      bool
	Files       []ChangedFile
}

type ChangedFile struct {
	StableID  string
	Status    ChangedFileStatus
	OldPath   string
	Path      string
	Additions int
	Deletions int
	Binary    bool
}

type DiffWindow struct {
	FileID        string
	Hunks         []DiffHunk
	HasMore       bool
	PartialReason PartialReason
	NextStartLine int
}

type DiffHunk struct {
	Header string
	Rows   []PairedRow
}

type PairedRow struct {
	ID                     string
	Kind                   string
	From, To               CodeLine
	FromPresent, ToPresent bool
}

type CodeLine struct {
	Number int
	Text   string
}
