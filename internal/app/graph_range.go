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
	Anchor string         // never "". Never a synthetic hash.
	Kind   graphRangeKind //
	// From and To always describe the path as it actually runs, so the summary
	// never has to know which direction was picked. On a forward pick From is
	// the anchor; on a reversed one it is the cursor, and From != Anchor is what
	// makes the reversal detectable.
	From    string
	To      string
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
		base.From, base.To = anchor, cursor
		base.Members = hashSet(forward)
		base.Count = len(forward)
		return base
	}
	if len(backward) > 0 {
		base.Kind = rangeBackward
		base.From, base.To = cursor, anchor
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

// summaryText renders the selection for the Details panel. The direction is
// part of the answer, not decoration: showing only a count leaves the user
// unable to tell which commit is the ancestor, and telling them that is the
// point of the feature.
//
// A reversed pick says so rather than quietly swapping the endpoints. Once the
// interaction asks for FROM and then TO, the user has declared a direction, and
// silently showing the other one is still showing something they did not ask
// for - better than the "diverged" lie, but not the whole way there.
//
// Returns "" when there is no selection, so the caller drops the row.
func (r *graphRange) summaryText(width int) string {
	if r == nil {
		return ""
	}
	short := func(hash string) string { return shorten(hash, 8) }
	switch r.Kind {
	case rangeAnchorOnly:
		return fitRangeSummary(fmt.Sprintf("from %s · pick a second commit", short(r.Anchor)),
			fmt.Sprintf("from %s", short(r.Anchor)), width)
	case rangeSame:
		return "same commit"
	case rangeForward:
		return fitRangeSummary(fmt.Sprintf("%d commits (%s → %s)", r.Count, short(r.From), short(r.To)),
			fmt.Sprintf("%d commits", r.Count), width)
	case rangeBackward:
		return fitRangeSummary(fmt.Sprintf("%d commits (reversed: %s → %s)", r.Count, short(r.From), short(r.To)),
			fmt.Sprintf("%d commits reversed", r.Count), width)
	case rangeDiverged:
		return fitRangeSummary("diverged (no ancestry path)", "diverged", width)
	case rangeUnavailable:
		return "unavailable"
	}
	return ""
}

// fitRangeSummary prefers the full phrasing and falls back to the short one
// rather than letting the Details viewport clip a sentence mid-word. The rail is
// 13 columns wide at an 80-column terminal, so the long form rarely fits.
func fitRangeSummary(full, short string, width int) string {
	if width >= len(full) {
		return full
	}
	if width >= len(short) {
		return short
	}
	return ""
}
