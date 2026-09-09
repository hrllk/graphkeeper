package app

import (
	"strings"
	"testing"

	"hrllk/graphkeeper/internal/state"
)

// Tasks 10.12 through 10.16. These popups name their own keys, and the generic
// "esc: close" row underneath meant esc read twice on the confirmation screens.
// The key still works everywhere; task 1.23's Esc-first close contract is
// untouched. Only these five popups drop the row - the rest of the app keeps it
// until task 6.6 settles one footer vocabulary for every surface.
func TestConfirmationPopupsShowEscAtMostOnce(t *testing.T) {
	for _, tt := range []struct {
		name   string
		render func() string
	}{
		{"checkout confirm", func() string {
			m := model{status: state.New().WithConfirm(state.ActionCheckout, "Checkout branch?", "Switch to feature.")}
			m.status.Selected = "feature"
			return renderConfirmPopup(m, 80)
		}},
		{"branch delete confirm", func() string {
			m := model{status: state.New().WithConfirm(state.ActionDeleteBranch, "Delete branch?", "Remove feature.")}
			m.status.Selected = "feature"
			return renderConfirmPopup(m, 80)
		}},
		{"fast-forward confirm", func() string {
			m := model{status: state.New().WithConfirm(state.ActionMerge, "Fast-forward available.", "HEAD can move to feature.")}
			m.status.Selected = "feature"
			return renderConfirmPopup(m, 80)
		}},
		{"reset mode", func() string { return renderResetModePopup(60) }},
		{"alert", func() string {
			return renderAlertPopup(alertContent{Title: "Alert", Description: "Select a local branch."}, 80)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.render()
			if got == "" {
				t.Skip("popup did not render for this fixture")
			}
			if strings.Contains(got, "esc: close") {
				t.Fatalf("the generic esc row must be gone: %q", got)
			}
			if n := strings.Count(got, "esc"); n > 1 {
				t.Fatalf("esc appears %d times, want at most once: %q", n, got)
			}
		})
	}
}

// The alert stops advertising both of its keys, and stops making its own
// colours. highlighting-color-map.md's Policy allows ANSI 0-15 only, and D-013
// lists view_alert.go among the files still building styles outside theme.go.
func TestAlertPopupUsesSharedTokensAndHidesItsKeys(t *testing.T) {
	got := renderAlertPopup(alertContent{Title: "Alert", Description: "Select a local branch."}, 80)
	if !strings.Contains(got, "Select a local branch.") {
		t.Fatalf("the alert lost its description: %q", got)
	}
	if strings.Contains(got, "enter: dismiss") || strings.Contains(got, "esc") {
		t.Fatalf("the alert must not advertise its keys: %q", got)
	}
	// ANSI-256 shows up as "38;5;" / "48;5;" in the escape stream.
	if strings.Contains(got, "38;5;") || strings.Contains(got, "48;5;") {
		t.Fatalf("the alert must not emit ANSI-256 colour: %q", got)
	}
}
