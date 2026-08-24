package app

import (
	"strings"
	"testing"

	"hrllk/graphkeeper/internal/state"
)

func TestClassifyConfirmKeyBinary(t *testing.T) {
	for _, tt := range []struct {
		key      string
		decision confirmDecision
	}{
		{"y", decisionAccept},
		{"enter", decisionAccept},
		{"n", decisionCancel},
		{"esc", decisionCancel},
		{"x", decisionNoop},
	} {
		got := classifyConfirmKey(confirmViewModel{Kind: confirmBinary}, tt.key)
		if got.Decision != tt.decision || got.ChoiceKey != "" {
			t.Fatalf("key %q: got %#v, want decision=%q with no choice", tt.key, got, tt.decision)
		}
	}
}

func TestClassifyConfirmKeyPullChoice(t *testing.T) {
	view := confirmViewModel{Kind: confirmChoiceKind, Action: state.ActionPull}
	for _, tt := range []struct {
		key      string
		decision confirmDecision
		choice   string
	}{
		{"m", decisionChoice, choiceMerge},
		{"r", decisionChoice, choiceRebase},
	} {
		got := classifyConfirmKey(view, tt.key)
		if got.Decision != tt.decision || got.ChoiceKey != tt.choice {
			t.Fatalf("key %q: got %#v, want decision=%q choice %q", tt.key, got, tt.decision, tt.choice)
		}
	}
}

func TestClassifyConfirmKeyStaleDisablesAffirmativeChoices(t *testing.T) {
	view := confirmViewModel{Kind: confirmChoiceKind, Action: state.ActionPull, Disabled: true}
	for _, key := range []string{"m", "r", "enter", "y"} {
		got := classifyConfirmKey(view, key)
		if got.Decision != decisionNoop {
			t.Fatalf("stale key %q was not disabled: %#v", key, got)
		}
	}
	for _, key := range []string{"n", "esc"} {
		if got := classifyConfirmKey(view, key); got.Decision != decisionCancel {
			t.Fatalf("stale close key %q: %#v", key, got)
		}
	}
}

func TestRenderStaleConfirmShowsDisabledProjection(t *testing.T) {
	m := model{
		navigationState: navigationState{
			width:  80,
			height: 30,
		},
		pullState: pullState{
			pullIsFastForward: false,
			pullConfirmStale:  true,
		},
		status: state.New().WithConfirm(state.ActionPull, "Pull preview", "Review pull impact.")}
	got := renderConfirmPopup(m, 80)
	if !strings.Contains(got, "Preview is stale") || strings.Contains(got, "esc") || strings.Contains(got, "n: close") {
		t.Fatalf("stale projection missing disabled copy: %q", got)
	}
	if strings.Contains(got, "m: merge") || strings.Contains(got, "r: rebase") {
		t.Fatalf("stale projection exposed affirmative choices: %q", got)
	}
}
