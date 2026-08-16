package app

import "context"

// The Inspector contract is intentionally owned by app. Git adapters may use a
// different internal representation, but the screen only depends on these
// request/result semantics and never reparses raw Git output.
type InspectorPaneState string

const (
	PaneIdle     InspectorPaneState = "idle"
	PaneLoading  InspectorPaneState = "loading"
	PaneReady    InspectorPaneState = "ready"
	PanePartial  InspectorPaneState = "partial"
	PaneError    InspectorPaneState = "error"
	PaneCanceled InspectorPaneState = "canceled"
)

type InspectorError struct {
	Kind      string
	Message   string
	Retryable bool
}

type InspectorResult[T any] struct {
	State           InspectorPaneState
	Value           T
	Error           *InspectorError
	Commit          string
	Parent          string
	FileID          string
	RequestID       string
	RepositoryEpoch uint64
}

type CommitRequest struct {
	Commit          string
	RequestID       string
	RepositoryEpoch uint64
}

type DiffWindowRequest struct {
	StartLine int
	MaxLines  int
	MaxBytes  int
}

type DiffRequest struct {
	Commit          string
	Parent          string
	FileID          string
	RequestID       string
	RepositoryEpoch uint64
	Window          DiffWindowRequest
}

// CommitInspectorReader is the boundary used by a future alternate reader;
// the production adapter is currently wired through git.Repo commands.
type CommitInspectorReader interface {
	InspectCommit(context.Context, CommitRequest) InspectorResult[CommitSnapshot]
	LoadDiff(context.Context, DiffRequest) InspectorResult[DiffWindow]
}

type CommitSnapshot struct {
	FullHash    string
	Subject     string
	MessageBody string
	MessageRaw  string
	AuthorEmail string
	Parent      string
	IsRoot      bool
	Files       []ChangedFile
}

type ChangedFile struct {
	StableID  string
	Status    string
	OldPath   string
	Path      string
	Additions int
	Deletions int
}

type DiffWindow struct {
	FileID        string
	Hunks         []DiffHunk
	HasMore       bool
	PartialReason string
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
