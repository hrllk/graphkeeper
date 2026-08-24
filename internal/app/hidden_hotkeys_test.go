package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"hrllk/graphkeeper/internal/state"
)

func TestVisibleHiddenHotkeySectionsReturnsActiveSectionOnly(t *testing.T) {
	tests := []struct {
		name    string
		section graphSection
		want    string
	}{
		{name: "graph", section: sectionGraph, want: "Graph"},
		{name: "local", section: sectionCurrent, want: "Local"},
		{name: "remote", section: sectionRemote, want: "Remote"},
		{name: "tags", section: sectionTags, want: "Tags"},
		{name: "invalid", section: graphSection(99), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sections := visibleHiddenHotkeySections(model{
				navigationState: navigationState{
					activeSection: tt.section,
				}})
			wantLen := 1
			if tt.want == "" {
				wantLen = 0
			}
			if len(sections) != wantLen {
				t.Fatalf("expected %d visible sections, got %d", wantLen, len(sections))
			}
			if tt.want != "" && sections[0].title != tt.want {
				t.Fatalf("expected active section %q, got %q", tt.want, sections[0].title)
			}
		})
	}
}

func TestHiddenHotkeysPopupShowsActiveSectionOnly(t *testing.T) {
	tests := []struct {
		name   string
		active graphSection
		want   []string
		hide   []string
	}{
		{name: "graph", active: sectionGraph, want: []string{"Graph", "m: merge"}, hide: []string{"Global", "Moved out:", "Local", "Remote", "Tags", "s: stash changes", "enter: jump to graph"}},
		{name: "local", active: sectionCurrent, want: []string{"Local", "s: stash changes"}, hide: []string{"Global", "Moved out:", "Graph", "Remote", "Tags", "m: merge"}},
		{name: "remote", active: sectionRemote, want: []string{"Remote", "space: checkout"}, hide: []string{"Global", "Moved out:", "Graph", "Local", "Tags", "s: stash changes"}},
		{name: "tags", active: sectionTags, want: []string{"Tags", "enter: jump to graph"}, hide: []string{"Global", "Moved out:", "Graph", "Local", "Remote", "s: stash changes"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Strip(renderHiddenHotkeysPopup(model{
				navigationState: navigationState{
					activeSection: tt.active,
				},
				status: state.New().WithBrowse(),
			}, 90, 0))
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Fatalf("expected popup to contain %q, got %q", want, got)
				}
			}
			for _, hidden := range tt.hide {
				if strings.Contains(got, hidden) {
					t.Fatalf("expected popup to omit %q, got %q", hidden, got)
				}
			}
			if strings.Contains(got, "q: close") || strings.Contains(got, "esc: close") {
				t.Fatalf("popup must not advertise a close key, got %q", got)
			}
		})
	}
}

func TestGlobalHotkeyItemsAreMainFooterSource(t *testing.T) {
	got := ansi.Strip(renderMainHotkeyFooter(120))
	for _, want := range []string{"tab: switch", "k/j: updown", "q: quit", "?: hotkeys", "ctrl + u/d: scroll"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected global hotkeys to contain %q, got %q", want, got)
		}
	}
	if len(globalHotkeyItems()) != 5 {
		t.Fatalf("expected five global footer items, got %d", len(globalHotkeyItems()))
	}
}

func TestHiddenHotkeysPopupUsesANSISemanticColors(t *testing.T) {
	withColorProfile(t, termenv.TrueColor)
	got := renderHiddenHotkeysPopup(model{
		navigationState: navigationState{
			activeSection: sectionGraph,
		},
		status: state.New().WithBrowse(),
	}, 90, 0)
	if strings.Contains(got, "38;5;205") {
		t.Fatalf("hidden hotkeys popup must not use pink 256-color border: %q", got)
	}
	if !strings.Contains(got, "35") {
		t.Fatalf("hidden hotkeys popup must use ANSI semantic popup color: %q", got)
	}
}

func TestClampHiddenHotkeyScroll(t *testing.T) {
	tests := []struct {
		name                string
		offset, total, view int
		want                int
	}{
		{name: "empty", offset: 4, total: 0, view: 5, want: 0},
		{name: "zero viewport", offset: 4, total: 10, view: 0, want: 0},
		{name: "top clamp", offset: -1, total: 10, view: 4, want: 0},
		{name: "bottom clamp", offset: 99, total: 10, view: 4, want: 6},
		{name: "within bounds", offset: 3, total: 10, view: 4, want: 3},
		{name: "viewport larger than content", offset: 3, total: 4, view: 8, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampHiddenHotkeyScroll(tt.offset, tt.total, tt.view); got != tt.want {
				t.Fatalf("expected offset %d, got %d", tt.want, got)
			}
		})
	}
}

func TestHiddenHotkeyPopupUsesHotkeyWidthPolicy(t *testing.T) {
	for _, tt := range []struct {
		bodyWidth int
		want      int
	}{
		{bodyWidth: 90, want: hiddenHotkeyPopupMaxWidth},
		{bodyWidth: 50, want: 38},
		{bodyWidth: 20, want: 20},
	} {
		if got := hiddenHotkeyPopupWidth(tt.bodyWidth); got != tt.want {
			t.Fatalf("hiddenHotkeyPopupWidth(%d) = %d, want %d", tt.bodyWidth, got, tt.want)
		}
	}
}

func TestHiddenHotkeyPopupInputIsConsumedAndReopensAtTop(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			sectionCursor: map[graphSection]int{sectionGraph: 0},
			width:         120,
			height:        24,
		},
		status:       state.New().WithBrowse(),
		overlayState: overlayState{hiddenHotkeysOpen: true, graphSearchOpen: false, hiddenHotkeysScroll: 0},
	}
	initialSection := m.activeSection
	initialCursor := m.sectionCursor[sectionGraph]

	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'j'}},
		{Type: tea.KeyDown},
		{Type: tea.KeyCtrlD},
		{Type: tea.KeyCtrlU},
		{Type: tea.KeyRunes, Runes: []rune{'k'}},
		{Type: tea.KeyUp},
		{Type: tea.KeyTab},
	} {
		gotModel, cmd := m.Update(msg)
		if cmd != nil {
			t.Fatalf("expected popup key to be synchronous, got %v", cmd)
		}
		m = gotModel.(model)
		if !m.hiddenHotkeysOpen {
			t.Fatal("expected popup to remain open while scrolling")
		}
		if m.activeSection != initialSection || m.sectionCursor[sectionGraph] != initialCursor {
			t.Fatal("expected popup input not to change Browse state")
		}
	}

	m.hiddenHotkeysScroll = 99
	gotModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("expected escape to be synchronous, got %v", cmd)
	}
	m = gotModel.(model)
	if m.hiddenHotkeysOpen {
		t.Fatal("expected escape to close popup")
	}

	gotModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if cmd != nil {
		t.Fatalf("expected question mark to be synchronous, got %v", cmd)
	}
	m = gotModel.(model)
	if !m.hiddenHotkeysOpen || m.hiddenHotkeysScroll != 0 {
		t.Fatalf("expected popup reopen to reset scroll, open=%v scroll=%d", m.hiddenHotkeysOpen, m.hiddenHotkeysScroll)
	}
	gotModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if cmd != nil {
		t.Fatalf("expected question mark close to be synchronous, got %v", cmd)
	}
	if gotModel.(model).hiddenHotkeysOpen {
		t.Fatal("expected question mark to close an open popup")
	}
}

func TestHiddenHotkeyPopupFitsSmallHeightAndOverlay(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			width:         120,
			height:        24,
		},
		status:       state.New().WithBrowse(),
		overlayState: overlayState{hiddenHotkeysOpen: true, hiddenHotkeysScroll: 99},
	}
	bodyWidth, bodyHeight := 80, 12
	popup := renderHiddenHotkeysPopup(m, bodyWidth, bodyHeight)
	if got := lipgloss.Height(popup); got > bodyHeight {
		t.Fatalf("expected popup height <= %d, got %d", bodyHeight, got)
	}
	if !strings.Contains(ansi.Strip(popup), "Hidden Hotkeys") || strings.Contains(ansi.Strip(popup), "q: close") || strings.Contains(ansi.Strip(popup), "esc: close") {
		t.Fatalf("expected popup title without close footer, got %q", popup)
	}

	base := strings.Repeat(strings.Repeat("x", 100)+"\n", bodyHeight-1) + strings.Repeat("x", 100)
	composed := overlayPopup(base, popup)
	if composed == base {
		t.Fatal("expected small-height popup to be composed over base view")
	}
}

func TestHiddenHotkeyPopupResizesWithoutBlankContent(t *testing.T) {
	m := model{
		navigationState: navigationState{
			activeSection: sectionGraph,
			width:         120,
			height:        40,
		},
		status:       state.New().WithBrowse(),
		overlayState: overlayState{hiddenHotkeysOpen: true, hiddenHotkeysScroll: 99},
	}
	gotModel, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	if cmd != nil {
		t.Fatalf("expected resize to be synchronous, got %v", cmd)
	}
	m = gotModel.(model)
	width, height := hiddenHotkeyPopupBodySize(m)
	popup := renderHiddenHotkeysPopup(m, width, height)
	plain := ansi.Strip(popup)
	if strings.Contains(plain, "Hidden hotkeys by section\n\n\n") {
		t.Fatalf("expected resize to avoid blank popup content, got %q", plain)
	}
	if strings.Contains(plain, "q: close") || strings.Contains(plain, "esc: close") {
		t.Fatalf("expected no close footer after resize, got %q", plain)
	}
}
