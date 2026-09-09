package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hrllk/graphkeeper/internal/state"
)

// resolveAppCmd runs a command and returns the message that is not the
// progress animation's own tick.
//
// Starting an operation now returns two commands batched together: the work,
// and the timer that animates the message while the work runs. Tests that care
// about the work go through here rather than each unwrapping the batch.
func resolveAppCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return msg
	}
	for _, inner := range batch {
		if inner == nil {
			continue
		}
		if got := resolveAppCmd(t, inner); got != nil {
			if _, isTick := got.(progressTickMsg); isTick {
				continue
			}
			return got
		}
	}
	return nil
}

func TestProgressAnimationCyclesTheEllipsisWithoutChangingWidth(t *testing.T) {
	const message = "Fetching upstream..."
	widths := map[int]bool{}
	seen := map[string]bool{}
	for frame := 0; frame < progressFrameCount; frame++ {
		got := progressAnimatedMessage(message, frame)
		widths[len(got)] = true
		seen[strings.TrimRight(got, " ")] = true
	}
	if len(widths) != 1 {
		t.Fatalf("the message must not change width while it animates, got widths %v", widths)
	}
	for _, want := range []string{"Fetching upstream.", "Fetching upstream..", "Fetching upstream..."} {
		if !seen[want] {
			t.Errorf("expected the cycle to pass through %q, got %v", want, seen)
		}
	}
}

// A period is punctuation, not progress. The loading mode carries prompts too.
func TestProgressAnimationLeavesNonProgressMessagesAlone(t *testing.T) {
	for _, message := range []string{"Enter a branch name.", "Choose a target.", "", "..."} {
		if progressMessageAnimates(message) {
			t.Errorf("%q is not a progress message", message)
		}
		if got := progressAnimatedMessage(message, 1); got != message {
			t.Errorf("expected %q untouched, got %q", message, got)
		}
	}
}

// The first paint has to match the message as written.
func TestProgressAnimationOpensOnTheWrittenForm(t *testing.T) {
	m := model{}
	m.status = operationLoadingStatusFor(progressFetch, "Fetching upstream...", state.ActionPull)
	armed, cmd := armProgressAnimation(m, state.ModeBrowse, nil)
	if cmd == nil {
		t.Fatal("entering the loading mode should arm the timer")
	}
	if got := armed.(model).statusMessageText(); got != "Fetching upstream..." {
		t.Fatalf("expected the first paint to match the written message, got %q", got)
	}
}

// A tick from a finished operation must not animate the next one, and must not
// keep the timer alive.
func TestProgressTickStopsWhenItNoLongerBelongs(t *testing.T) {
	m := model{}
	m.status = operationLoadingStatusFor(progressFetch, "Fetching upstream...", state.ActionPull)
	m.progressAnimating, m.progressEpoch = true, 7

	stale, cmd := handleProgressTick(m, progressTickMsg{epoch: 6})
	if cmd != nil {
		t.Error("a tick from an older operation must not re-arm the timer")
	}
	if stale.(model).progressFrame != m.progressFrame {
		t.Error("a tick from an older operation must not advance the frame")
	}

	done := m
	done.status = state.New().WithBrowse()
	finished, cmd := handleProgressTick(done, progressTickMsg{epoch: 7})
	if cmd != nil {
		t.Error("a tick that arrives after the work finished must not re-arm the timer")
	}
	if finished.(model).progressFrame != done.progressFrame {
		t.Error("a tick that arrives after the work finished must not advance the frame")
	}

	live, cmd := handleProgressTick(m, progressTickMsg{epoch: 7})
	if cmd == nil {
		t.Error("a tick that still belongs should keep the timer running")
	}
	if live.(model).progressFrame == m.progressFrame {
		t.Error("a tick that still belongs should advance the frame")
	}
}

// Leaving the loading mode retires the timer, whichever way the work ended.
func TestLeavingLoadingRetiresTheAnimation(t *testing.T) {
	for name, status := range map[string]state.Status{
		"success": state.New().WithBrowse(),
		"blocked": state.New().WithBlocked(state.BlockUnknown, "Blocked.", "Reason."),
		"result":  func() state.Status { s := state.New(); s.Mode = state.ModeOperationResult; return s }(),
	} {
		m := model{}
		m.status = status
		m.progressAnimating, m.progressFrame = true, 2
		next, cmd := armProgressAnimation(m, state.ModeLoading, nil)
		if cmd != nil {
			t.Errorf("%s: leaving the loading mode must not arm a timer", name)
		}
		got := next.(model)
		if got.progressAnimating || got.progressFrame != 0 {
			t.Errorf("%s: expected the animation retired, got animating=%v frame=%d", name, got.progressAnimating, got.progressFrame)
		}
	}
}

// The animation must not change what the frame looks like, at any width or
// under NO_COLOR, beyond the dots themselves.
func TestProgressAnimationHoldsTheFrameAtEveryWidth(t *testing.T) {
	for _, noColor := range []bool{false, true} {
		if noColor {
			t.Setenv("NO_COLOR", "1")
		}
		for _, width := range []int{40, 60, 80} {
			var widths []int
			for frame := 0; frame < progressFrameCount; frame++ {
				m := model{}
				m.width, m.height = width, 24
				m.status = operationLoadingStatusFor(progressFetch, "Fetching upstream...", state.ActionPull)
				m.progressAnimating, m.progressFrame = true, frame
				widths = append(widths, lipgloss.Width(renderLoadingPopup(m, width)))
			}
			for _, got := range widths {
				if got != widths[0] {
					t.Fatalf("no-color=%v width=%d: the popup changed width while animating: %v", noColor, width, widths)
				}
			}
		}
	}
}
