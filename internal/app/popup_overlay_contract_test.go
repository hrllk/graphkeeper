package app

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"hrllk/graphkeeper/internal/state"
)

func TestPopupTextInputsPreserveQAndEscCloses(t *testing.T) {
	tests := []struct {
		name string
		open func(model) model
		get  func(model) string
		call func(model, tea.KeyMsg) (model, tea.Cmd)
	}{
		{"branch", func(m model) model { m.overlayState.branchOpen = true; return m }, func(m model) string { return m.branchDraft }, func(m model, key tea.KeyMsg) (model, tea.Cmd) {
			next, cmd := m.handleBranchOpenKey(key)
			return next.(model), cmd
		}},
		{"tag", func(m model) model { m.overlayState.tagPopupOpen = true; return m }, func(m model) string { return m.tagPopupDraft }, func(m model, key tea.KeyMsg) (model, tea.Cmd) {
			next, cmd := m.handleTagPopupKey(key)
			return next.(model), cmd
		}},
		{"stash message", func(m model) model { m.overlayState.stashMessageOpen = true; return m }, func(m model) string { return m.stashMessageDraft }, func(m model, key tea.KeyMsg) (model, tea.Cmd) {
			next, cmd := m.handleStashMessageKey(key)
			return next.(model), cmd
		}},
		{"graph search", func(m model) model { m.overlayState.graphSearchOpen = true; return m }, func(m model) string { return m.graphSearchDraft }, func(m model, key tea.KeyMsg) (model, tea.Cmd) {
			next, cmd := m.handleGraphSearchKey(key)
			return next.(model), cmd
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.open(model{status: state.New().WithBrowse()})
			got, cmd := tt.call(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
			if cmd != nil || tt.get(got) != "q" {
				t.Fatalf("q was not preserved: draft=%q cmd=%v", tt.get(got), cmd != nil)
			}
			closed, cmd := tt.call(got, tea.KeyMsg{Type: tea.KeyEsc})
			if cmd != nil || closed.branchOpen || closed.tagPopupOpen || closed.stashMessageOpen || closed.graphSearchOpen {
				t.Fatalf("Esc did not close %s: %#v cmd=%v", tt.name, closed, cmd != nil)
			}
		})
	}
}

func TestNonInputPopupQIsNoopAndEscCloses(t *testing.T) {
	tests := []struct {
		name   string
		open   func(model) model
		isOpen func(model) bool
		call   func(model, tea.KeyMsg) (model, tea.Cmd)
	}{
		{"hidden", func(m model) model { m.overlayState.hiddenHotkeysOpen = true; return m }, func(m model) bool { return m.hiddenHotkeysOpen }, func(m model, key tea.KeyMsg) (model, tea.Cmd) {
			next, cmd := m.handleHiddenHotkeysKey(key)
			return next.(model), cmd
		}},
		{"stash", func(m model) model { m.overlayState.stashPopupOpen = true; return m }, func(m model) bool { return m.stashPopupOpen }, func(m model, key tea.KeyMsg) (model, tea.Cmd) {
			next, cmd := m.handleStashPopupKey(key)
			return next.(model), cmd
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.open(model{status: state.New().WithBrowse()})
			before := m
			got, cmd := tt.call(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
			if cmd != nil || !tt.isOpen(got) || !reflect.DeepEqual(got, before) {
				t.Fatalf("q mutated non-input popup: got=%#v before=%#v cmd=%v", got, before, cmd != nil)
			}
			got, cmd = tt.call(got, tea.KeyMsg{Type: tea.KeyEsc})
			if cmd != nil || tt.isOpen(got) {
				t.Fatalf("Esc did not close %s: %#v cmd=%v", tt.name, got, cmd != nil)
			}
		})
	}
}

func TestPopupRenderersDoNotAdvertiseQOrEscClose(t *testing.T) {
	m := model{status: state.New().WithBrowse(), overlayState: overlayState{branchOpen: true, branchDraft: "q", tagPopupOpen: true, tagPopupDraft: "q", tagPopupTarget: "head", stashMessageOpen: true, stashMessageDraft: "q", graphSearchOpen: true, graphSearchDraft: "q", hiddenHotkeysOpen: true}}
	renders := []string{
		renderBranchInputPopup(m, 80),
		renderTagPopup(m, 80, 24),
		renderStashMessagePopup(m, 80),
		renderGraphSearchPopup(m, 80),
		renderHiddenHotkeysPopup(m, 80, 24),
		renderAlertPopup(alertContent{Title: "Alert", Description: "blocked"}, 80),
	}
	for _, rendered := range renders {
		plain := strings.ToLower(ansi.Strip(rendered))
		for _, forbidden := range []string{"q: close", "q close", "q/esc", "q or esc", "esc: close", "esc: cancel"} {
			if strings.Contains(plain, forbidden) {
				t.Fatalf("popup advertised forbidden close affordance %q: %q", forbidden, plain)
			}
		}
	}
}

func TestShellOverlayStackRendererContract(t *testing.T) {
	cases := []struct {
		name  string
		setup func(model) model
	}{
		{"confirm", func(m model) model {
			m.status = state.New().WithConfirm(state.ActionStash, "Stash?", "detail")
			return m
		}},
		{"review", func(m model) model {
			m.status = state.New().WithReview(state.ActionMerge, "Review", "detail")
			return m
		}},
		{"reset-mode", func(m model) model {
			m.status = state.Status{Mode: state.ModeResetModePick, Title: "Reset mode", Message: "Choose a reset mode."}
			return m
		}},
		{"cherry-pick", func(m model) model {
			m.status = state.Status{Mode: state.ModeCherryPickPick, Title: "Cherry-pick", Message: "Choose a commit."}
			return m
		}},
		{"target-pick", func(m model) model {
			m.status = state.Status{Mode: state.ModeTargetPick, Title: "Target", Message: "Choose a target."}
			return m
		}},
		{"branch-input", func(m model) model { m.branchOpen = true; m.branchDraft = "q"; return m }},
		{"stash-message", func(m model) model { m.stashMessageOpen = true; m.stashMessageDraft = "q"; return m }},
		{"graph-stash-pop", func(m model) model { m.graphStashPopOpen = true; return m }},
		{"stash-popup", func(m model) model { m.stashPopupOpen = true; return m }},
		{"tag-popup", func(m model) model { m.tagPopupOpen = true; m.tagPopupDraft = "q"; m.tagPopupTarget = "head"; return m }},
		{"hidden-hotkeys", func(m model) model { m.hiddenHotkeysOpen = true; return m }},
		{"graph-search", func(m model) model { m.graphSearchOpen = true; m.graphSearchDraft = "q"; return m }},
		{"loading", func(m model) model { m.status = state.Status{Mode: state.ModeLoading, Message: "Working"}; return m }},
		{"blocked", func(m model) model {
			m.status = state.Status{Mode: state.ModeBlocked, Title: "Alert", Message: "blocked"}
			return m
		}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup(model{status: state.New().WithBrowse(), navigationState: navigationState{activeSection: sectionGraph}})
			var popup string
			for _, overlay := range shellOverlayStack(m, 80, 24) {
				if overlay.name == tt.name && overlay.active {
					popup = overlay.popup
					break
				}
			}
			if popup == "" {
				t.Fatalf("overlay %q was not active", tt.name)
			}
			plain := strings.ToLower(ansi.Strip(popup))
			for _, forbidden := range []string{"q: close", "q close", "q/esc", "q or esc", "esc: close", "esc: cancel"} {
				if strings.Contains(plain, forbidden) {
					t.Fatalf("overlay %q advertised forbidden affordance %q: %q", tt.name, forbidden, plain)
				}
			}
		})
	}
}

func TestPopupOverlayTopLevelRoutingContract(t *testing.T) {
	branch := model{status: state.New().WithBrowse(), overlayState: overlayState{branchOpen: true}}
	got, cmd := branch.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil || got.(model).branchDraft != "q" || !got.(model).branchOpen {
		t.Fatalf("top-level branch q routing failed: %#v cmd=%v", got, cmd != nil)
	}
	got, cmd = got.(model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || got.(model).branchOpen {
		t.Fatalf("top-level branch Esc routing failed: %#v cmd=%v", got, cmd != nil)
	}

	hidden := model{status: state.New().WithBrowse(), overlayState: overlayState{hiddenHotkeysOpen: true}}
	before := hidden
	got, cmd = hidden.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil || !reflect.DeepEqual(got.(model), before) {
		t.Fatalf("top-level hidden q was not a no-op: got=%#v before=%#v cmd=%v", got, before, cmd != nil)
	}

	confirm := model{status: state.New().WithConfirm(state.ActionStash, "Stash?", "detail")}
	before = confirm
	got, cmd = confirm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil || !reflect.DeepEqual(got.(model), before) {
		t.Fatalf("top-level confirmation q was not a no-op: got=%#v before=%#v cmd=%v", got, before, cmd != nil)
	}
}

func TestHiddenHotkeyLocalRemoteRegistryContract(t *testing.T) {
	local := hiddenHotkeySections(model{navigationState: navigationState{activeSection: sectionCurrent}})
	remote := hiddenHotkeySections(model{navigationState: navigationState{activeSection: sectionRemote}})
	find := func(sections []hiddenHotkeySection, title string) []hiddenHotkeyItem {
		for _, section := range sections {
			if section.title != title {
				continue
			}
			for _, group := range section.groups {
				if group.title == "Visible" {
					return group.items
				}
			}
		}
		return nil
	}
	localVisible := find(local, "Local")
	remoteVisible := find(remote, "Remote")
	localText, remoteText := renderHotkeyItems(localVisible), renderHotkeyItems(remoteVisible)
	if !strings.Contains(localText, "n: new branch") {
		t.Fatalf("local hidden help omitted new branch: %q", localText)
	}
	if !strings.Contains(remoteText, "space: checkout") {
		t.Fatalf("remote hidden help omitted checkout: %q", remoteText)
	}
	if !strings.Contains(remoteText, "d: delete branch") {
		t.Fatalf("remote hidden help omitted routed delete action: %q", remoteText)
	}
	for _, forbidden := range []string{"f: fetch", "p: pull"} {
		if strings.Contains(remoteText, forbidden) {
			t.Fatalf("remote hidden help advertised removed action %q: %q", forbidden, remoteText)
		}
	}
}

func renderHotkeyItems(items []hiddenHotkeyItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, item.key+": "+item.desc)
	}
	return strings.Join(parts, " · ")
}
func TestConfirmationQAndCheckoutEnterAreNoop(t *testing.T) {
	binary := confirmViewModel{Kind: confirmBinary, Action: state.ActionCheckout}
	for _, key := range []string{"q", "enter"} {
		if got := classifyConfirmKey(binary, key); got.Decision != decisionNoop {
			t.Fatalf("checkout key %q was not a no-op: %#v", key, got)
		}
	}
	if got := classifyConfirmKey(binary, "y"); got.Decision != decisionAccept {
		t.Fatalf("checkout y did not accept: %#v", got)
	}
}
