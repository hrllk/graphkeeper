package app

import ci "hrllk/graphkeeper/internal/commitinspector"

type InspectorPaneState = ci.InspectorPaneState
type PartialReason = ci.PartialReason
type ChangedFileStatus = ci.ChangedFileStatus
type InspectorError = ci.InspectorError
type DiffWindowRequest = ci.DiffWindowRequest
type InspectorResult[T any] = ci.InspectorResult[T]
type CommitRequest = ci.CommitRequest
type DiffRequest = ci.DiffRequest
type CommitInspectorReader = ci.CommitInspectorReader
type CommitSnapshot = ci.CommitSnapshot
type ChangedFile = ci.ChangedFile
type DiffWindow = ci.DiffWindow
type DiffHunk = ci.DiffHunk
type PairedRow = ci.PairedRow
type CodeLine = ci.CodeLine

const (
	PaneIdle               = ci.PaneIdle
	PaneLoading            = ci.PaneLoading
	PaneReady              = ci.PaneReady
	PanePartial            = ci.PanePartial
	PaneError              = ci.PaneError
	PaneCanceled           = ci.PaneCanceled
	PartialByteLimit       = ci.PartialByteLimit
	PartialLineLimit       = ci.PartialLineLimit
	PartialLineTruncated   = ci.PartialLineTruncated
	PartialProcessLimit    = ci.PartialProcessLimit
	PartialIndivisiblePair = ci.PartialIndivisiblePair
	StatusAdded            = ci.StatusAdded
	StatusModified         = ci.StatusModified
	StatusDeleted          = ci.StatusDeleted
	StatusRenamed          = ci.StatusRenamed
	StatusCopied           = ci.StatusCopied
	StatusBinary           = ci.StatusBinary
	StatusModeOnly         = ci.StatusModeOnly
	StatusSubmodule        = ci.StatusSubmodule
)

const (
	defaultInspectorMaxLines = 2000
	defaultInspectorMaxBytes = 1 << 20
	maxInspectorMaxLines     = 10000
	maxInspectorMaxBytes     = 16 << 20
)
