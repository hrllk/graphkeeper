package app

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/state"
)

// A static "Fetching upstream..." cannot be told from a hung one. Cycling its
// trailing dots says the app is still working, which is the whole of what
// DESIGN.md allows motion to do in a terminal: show that work is in flight,
// and stop.
//
// The suffix is padded to its widest frame so the message does not change
// width. A loading popup is centred, and a message that grows and shrinks
// would rock the box three times a second.
const (
	progressFrameCount    = 3
	progressFrameInterval = 300 * time.Millisecond
)

type progressTickMsg struct{ epoch uint64 }

func progressTickCmd(epoch uint64) tea.Cmd {
	return tea.Tick(progressFrameInterval, func(time.Time) tea.Msg {
		return progressTickMsg{epoch: epoch}
	})
}

// armProgressAnimation runs after every update, so the twenty-odd call sites
// that assign a loading status do not each have to remember to start a timer.
// It is also the only place the timer is retired, which is what keeps a tick
// belonging to a finished operation from animating the next one.
func armProgressAnimation(m model, before state.Mode, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if m.status.Mode != state.ModeLoading {
		m.progressAnimating = false
		m.progressFrame = 0
		return m, cmd
	}
	// Only a transition into loading arms the timer. Arming whenever the mode
	// merely *is* loading would make every message that arrives mid-operation
	// emit a command, including the stale and mismatched ones the pull
	// lifecycle requires to be exact no-ops.
	if before == state.ModeLoading || m.progressAnimating {
		return m, cmd
	}
	// Nothing to animate, nothing to time. The loading mode also carries
	// prompts that are not work in flight ("Enter a branch name."), and waking
	// the app three times a second to redraw a static string is worse than not
	// animating it.
	if !progressMessageAnimates(m.status.Message) {
		return m, cmd
	}
	m.progressAnimating = true
	m.progressEpoch++
	return m, tea.Batch(cmd, progressTickCmd(m.progressEpoch))
}

// handleProgressTick advances a frame, or does nothing at all. A tick arrives
// from a timer that was armed in the past, so it has to prove it still belongs:
// a stale epoch means a newer operation owns the animation, and a mode that is
// no longer loading means the work finished, failed, was cancelled or was
// blocked while this tick was in flight.
func handleProgressTick(m model, msg progressTickMsg) (tea.Model, tea.Cmd) {
	if msg.epoch != m.progressEpoch || m.status.Mode != state.ModeLoading {
		return m, nil
	}
	m.progressFrame = (m.progressFrame + 1) % progressFrameCount
	return m, progressTickCmd(m.progressEpoch)
}

// progressMessageAnimates reports whether a message ends in the ellipsis that
// marks work in flight.
//
// Three dots, not one. The loading mode also carries prompts that end in a
// full stop ("Enter a branch name."), and animating a sentence's period is
// both wrong and a reason to wake the app three times a second for a screen
// that is not going to change. Same distinction docs/decisions.md 2026-09-10
// draws for D-007: "..." at the end of a progress message is progress, a lone
// "." is punctuation.
func progressMessageAnimates(message string) bool {
	if !strings.HasSuffix(message, strings.Repeat(".", progressFrameCount)) {
		return false
	}
	return strings.TrimRight(message, ".") != ""
}

// progressAnimatedMessage cycles the dots a loading message already ends in. A
// message with no trailing dots is left exactly as written: this animates a
// suffix, it does not invent one.
func progressAnimatedMessage(message string, frame int) string {
	if !progressMessageAnimates(message) {
		return message
	}
	trimmed := strings.TrimRight(message, ".")
	// Frame 0 is the message as written. A model can reach the renderer without
	// ever having run an update, and the first paint of one that did should
	// match what the code wrote, so the cycle starts full and counts back up.
	dots := (frame+progressFrameCount-1)%progressFrameCount + 1
	return trimmed + strings.Repeat(".", dots) + strings.Repeat(" ", progressFrameCount-dots)
}

// statusMessageText is what the status line and the loading popup render. Both
// go through here so the animation cannot appear on one and not the other.
func (m model) statusMessageText() string {
	if m.status.Mode != state.ModeLoading {
		return m.status.Message
	}
	return progressAnimatedMessage(m.status.Message, m.progressFrame)
}
