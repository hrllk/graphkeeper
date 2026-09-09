package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// The Create tag popup drew read-only context and an editable field
// identically, and centred both, so nothing said which line accepted typing
// and the label slid left as the value grew out from the middle.
func TestTagPopupSeparatesTheEditableFieldFromContext(t *testing.T) {
	forceTrueColorProfile(t)
	raw := renderTagPopup(model{
		overlayState: overlayState{tagPopupTarget: "abc1234", tagPopupDraft: "v1.2.3"},
	}, 72, 24)

	if !strings.Contains(raw, underlineSignalPrefix) {
		t.Error("expected the editable field to be underlined so it reads as a field")
	}
	if !strings.Contains(raw, cursorSignalPrefix) {
		t.Error("expected a caret in the editable field")
	}
	// The read-only value carries neither, which is what tells them apart.
	target := lineContaining(t, ansi.Strip(raw), "target:")
	if strings.Contains(target, underlineSignalPrefix) {
		t.Error("expected the read-only target to carry no field marking")
	}
}

// Typing must not move the field. It used to: both lines were centred, so the
// value grew outward from the middle and pushed its own label leftward.
func TestTagPopupFieldDoesNotMoveWhileTyping(t *testing.T) {
	forceTrueColorProfile(t)
	var widths, indents []int
	for _, draft := range []string{"", "v", "v1", "v1.2.0-rc.1"} {
		out := ansi.Strip(renderTagPopup(model{
			overlayState: overlayState{tagPopupTarget: "abc1234", tagPopupDraft: draft},
		}, 72, 24))
		widths = append(widths, lipgloss.Width(out))
		line := lineContaining(t, out, "name:")
		body := strings.TrimPrefix(line, "│")
		indents = append(indents, len(body)-len(strings.TrimLeft(body, " ")))
	}
	for i := range widths {
		if widths[i] != widths[0] {
			t.Fatalf("popup width moved while typing: %v", widths)
		}
		if indents[i] != indents[0] {
			t.Fatalf("field label moved while typing: %v", indents)
		}
	}
}

// A popup is sized by what it holds. It used to be sized by the terminal, so a
// three-line form took 56 columns wherever there was room for 56.
func TestPopupsAreSizedByContentNotByTheTerminal(t *testing.T) {
	forceTrueColorProfile(t)
	narrow := lipgloss.Width(ansi.Strip(renderTagPopup(model{
		overlayState: overlayState{tagPopupTarget: "abc1234", tagPopupDraft: "v1"},
	}, 60, 24)))
	wide := lipgloss.Width(ansi.Strip(renderTagPopup(model{
		overlayState: overlayState{tagPopupTarget: "abc1234", tagPopupDraft: "v1"},
	}, 200, 24)))
	if narrow != wide {
		t.Fatalf("expected the same content to take the same width, got %d at 60 and %d at 200", narrow, wide)
	}
}

// Whatever the width, the box stays rectangular and inside the terminal.
func TestPopupsStayRectangularAcrossWidths(t *testing.T) {
	forceTrueColorProfile(t)
	for _, bodyWidth := range []int{20, 40, 60, 80, 120, 180} {
		m := model{overlayState: overlayState{tagPopupTarget: "abc1234", tagPopupDraft: "v1.2.0"}}
		for name, out := range map[string]string{
			"tag":     ansi.Strip(renderTagPopup(m, bodyWidth, 30)),
			"hotkeys": ansi.Strip(renderHiddenHotkeysPopup(m, bodyWidth, 30)),
		} {
			if out == "" {
				continue
			}
			lines := strings.Split(out, "\n")
			want := lipgloss.Width(lines[0])
			for i, line := range lines {
				if got := lipgloss.Width(line); got != want {
					t.Errorf("%s popup at body %d: line %d is %d wide, box is %d: %q", name, bodyWidth, i, got, want, line)
					break
				}
			}
			if want > bodyWidth {
				t.Errorf("%s popup at body %d overflows the terminal at %d wide", name, bodyWidth, want)
			}
		}
	}
}

// The app has three form popups. Fixing the affordance on one and not the
// others is a pattern this repo has hit repeatedly (the Inspector's two render
// paths, the graph and the rail cursor, the renderer and the scroll callers),
// so all three are held to the contract here.
func TestEveryFormPopupMarksItsEditableField(t *testing.T) {
	forceTrueColorProfile(t)
	m := model{overlayState: overlayState{
		tagPopupTarget:    "abc1234",
		tagPopupDraft:     "v1.2.3",
		branchDraft:       "feature/x",
		branchBase:        "abc1234",
		stashMessageDraft: "wip",
	}}
	for name, out := range map[string]string{
		"create tag":    renderTagPopup(m, 72, 24),
		"create branch": renderBranchInputPopup(m, 72),
		"stash message": renderStashMessagePopup(m, 72),
	} {
		if !strings.Contains(out, underlineSignalPrefix) {
			t.Errorf("%s: the editable field is not marked, so it reads as read-only context", name)
		}
		if !strings.Contains(out, cursorSignalPrefix) {
			t.Errorf("%s: no caret in the editable field", name)
		}
	}
}

// An empty field still has to be visible. Padding the draft to " " -- what each
// of these popups used to do -- is the workaround an invisible field forces.
func TestFormFieldIsVisibleWhenEmpty(t *testing.T) {
	empty := formInput("", 12)
	if lipgloss.Width(ansi.Strip(empty)) != 12 {
		t.Fatalf("expected an empty field to hold its full extent, got %d", lipgloss.Width(ansi.Strip(empty)))
	}
	if !strings.Contains(empty, underlineSignalPrefix) {
		t.Fatal("expected an empty field to still be underlined")
	}
}

// The minimum keeps an empty field visible; it must not outrank the box.
func TestFormFieldMinimumYieldsToTheBox(t *testing.T) {
	if got := formFieldWidth(10, 6); got > 10-formLabelColumn(6) && got > 1 {
		t.Fatalf("expected the field to fit the room left after the label, got %d in 10", got)
	}
	if got := formFieldWidth(2, 6); got < 1 {
		t.Fatalf("expected at least one column, got %d", got)
	}
}

func lineContaining(t *testing.T, block, want string) string {
	t.Helper()
	for _, line := range strings.Split(block, "\n") {
		if strings.Contains(line, want) {
			return line
		}
	}
	t.Fatalf("expected a line containing %q in:\n%s", want, block)
	return ""
}
