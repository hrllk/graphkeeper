package app

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/git"
)

// graphRange is the "road" selection: two commits and what git says lies
// between them. A nil *graphRange is the only representation of "no anchor",
// which is why Anchor is never empty and Kind is never a zero-value "off".
// Storing the anchor and an "off" state separately made anchor == "" with an
// active kind representable, and that combination would call AncestryPath with
// a blank ref.
type graphRange struct {
	Anchor  string          // never "". Never a synthetic hash.
	Kind    graphRangeKind  //
	To      string          // the endpoint the path runs toward, for the summary
	Members map[string]bool // nil unless Kind is rangeForward or rangeBackward
	Count   int
	Epoch   uint64 // repositoryEpoch when the anchor was set
	Err     string // "" unless Kind == rangeUnavailable
}

type graphRangeKind int

const (
	rangeAnchorOnly  graphRangeKind = iota // anchor set, second commit not chosen
	rangeSame                              // anchor == cursor
	rangeForward                           // path runs anchor -> cursor
	rangeBackward                          // path runs cursor -> anchor
	rangeDiverged                          // both directions empty
	rangeUnavailable                       // the query failed
)

// resolveGraphRange decides which of the six states two commits are in, given
// what AncestryPath returned for each direction.
//
// The direction test is the whole point. `--ancestry-path A..B` comes back empty
// in three different situations and only one of them is divergence:
//
//	A and B genuinely diverged      -> empty
//	B is an ancestor of A           -> empty, but a path exists the other way
//	A == B                          -> empty, and neither is true
//
// Reporting all three as "diverged" would tell a user who picked the newer
// commit first that their history had forked. The screen-row-range approach was
// rejected for teaching exactly that kind of falsehood, so this cannot repeat it.
//
// forward is AncestryPath(anchor, cursor); backward is AncestryPath(cursor, anchor).
// backward is only consulted when forward came back empty, so the caller may
// pass nil for it in that case.
func resolveGraphRange(anchor, cursor string, epoch uint64, forward, backward []string, err error) *graphRange {
	if anchor == "" {
		return nil
	}
	base := &graphRange{Anchor: anchor, Epoch: epoch}
	if err != nil {
		base.Kind = rangeUnavailable
		base.Err = err.Error()
		return base
	}
	if cursor == "" {
		base.Kind = rangeAnchorOnly
		return base
	}
	if cursor == anchor {
		base.Kind = rangeSame
		return base
	}
	if len(forward) > 0 {
		base.Kind = rangeForward
		base.To = cursor
		base.Members = hashSet(forward)
		base.Count = len(forward)
		return base
	}
	if len(backward) > 0 {
		base.Kind = rangeBackward
		base.To = anchor
		base.Members = hashSet(backward)
		base.Count = len(backward)
		return base
	}
	base.Kind = rangeDiverged
	return base
}

func hashSet(hashes []string) map[string]bool {
	set := make(map[string]bool, len(hashes))
	for _, hash := range hashes {
		if hash != "" {
			set[hash] = true
		}
	}
	return set
}

// rangeMembers and rangeAnchor read through a nil *graphRange so the projection
// and the renderers never have to nil-check it themselves.
func (r *graphRange) rangeMembers() map[string]bool {
	if r == nil {
		return nil
	}
	return r.Members
}

func (r *graphRange) rangeAnchor() string {
	if r == nil {
		return ""
	}
	return r.Anchor
}

// staleFor reports whether this selection was made against a different
// repository epoch. Graph rows are rebuilt when the epoch moves, so a selection
// from before then points at rows whose meaning has changed. The Inspector
// already handles its own staleness this way (commitInspectorEpoch).
func (r *graphRange) staleFor(epoch uint64) bool {
	return r != nil && r.Epoch != epoch
}

// graphRangeMsg carries an ancestry-path query result back to Update. It uses
// the message-err idiom that checkGraphActionTarget (commands.go) already
// follows for one-shot async queries, not the Known/Fresh/Error projection
// idiom (repository_read.go), which is for state that repository refresh
// regenerates.
type graphRangeMsg struct {
	anchor, cursor    string
	epoch             uint64
	forward, backward []string
	err               error
}

// queryGraphRange asks git for the path in the forward direction and, only when
// that comes back empty, in the reverse. Worst case is two rev-list calls; a
// forward hit costs one.
func queryGraphRange(repo *git.Repo, anchor, cursor string, epoch uint64) tea.Cmd {
	return func() tea.Msg {
		msg := graphRangeMsg{anchor: anchor, cursor: cursor, epoch: epoch}
		if repo == nil {
			msg.err = fmt.Errorf("repo is nil")
			return msg
		}
		forward, err := repo.AncestryPath(context.Background(), anchor, cursor)
		if err != nil {
			msg.err = err
			return msg
		}
		if len(forward) > 0 {
			msg.forward = forward
			return msg
		}
		backward, err := repo.AncestryPath(context.Background(), cursor, anchor)
		if err != nil {
			msg.err = err
			return msg
		}
		msg.backward = backward
		return msg
	}
}

// applyGraphRangeMsg installs a query result, discarding one that raced a
// repository refresh. update_lifecycle.go drops stale results the same way; an
// epoch mismatch here would otherwise highlight rows the graph has rebuilt.
func applyGraphRangeMsg(m model, msg graphRangeMsg) model {
	if msg.epoch != m.repositoryEpoch {
		return m
	}
	if m.graphRange == nil || m.graphRange.Anchor != msg.anchor {
		// The anchor was cleared or moved while the query was in flight.
		return m
	}
	m.graphRange = resolveGraphRange(msg.anchor, msg.cursor, msg.epoch, msg.forward, msg.backward, msg.err)
	return m
}
