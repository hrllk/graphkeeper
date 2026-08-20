package app

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

const (
	defaultInspectorMaxLines = 2000
	defaultInspectorMaxBytes = 1 << 20
	maxInspectorMaxLines     = 10000
	maxInspectorMaxBytes     = 16 << 20
)

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
