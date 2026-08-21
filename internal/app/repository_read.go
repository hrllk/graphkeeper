package app

import (
	"context"

	"hrllk/graphkeeper/internal/graph"
)

// RepositoryReadPort is the application boundary for the read-only snapshot
// used by the graph flow. Legacy repository operations intentionally remain on
// the broader repository contract until their own ports are introduced.
type RepositoryReadPort interface {
	ReadSnapshot(context.Context, ReadRequest) (ReadSnapshotResult, error)
}

type ReadRequest struct {
	CommitLimit     int
	RequestID       uint64
	RepositoryEpoch uint64
}

type ReadSnapshotResult struct {
	RequestID       uint64
	RepositoryEpoch uint64
	Snapshot        ReadSnapshot
	ErrorKind       ReadErrorKind
	Canceled        bool
}

type ReadSnapshot struct {
	Root      string
	Graph     graph.Snapshot
	Freshness PullSnapshotIdentity
}

type ReadErrorKind string

const (
	ReadErrorNone       ReadErrorKind = "none"
	ReadErrorRepository ReadErrorKind = "repository"
	ReadErrorInvalid    ReadErrorKind = "invalid"
	ReadErrorCanceled   ReadErrorKind = "canceled"
)

type ReadError struct{ Kind ReadErrorKind }

func (e *ReadError) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Kind)
}
