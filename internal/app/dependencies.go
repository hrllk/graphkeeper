package app

import (
	"hrllk/graphkeeper/internal/commitinspector"
	"hrllk/graphkeeper/internal/events"
	"hrllk/graphkeeper/internal/git"
)

type Dependencies struct {
	Repo            *git.Repo
	RepositoryRead  RepositoryReadPort
	Pull            PullPort
	InspectorReader commitinspector.CommitInspectorReader
	TagProvenance   TagProvenanceStore
	EventSink       events.EventSink
}
