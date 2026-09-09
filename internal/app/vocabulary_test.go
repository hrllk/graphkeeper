package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"hrllk/graphkeeper/internal/git"
)

// The app used two glyphs for four jobs. "•" was both a list bullet and an
// inline separator; "·" was both an inline separator and a field separator in
// prose. The same footer existed in both spellings -- one built by
// confirmation_projection.go with a bullet, one by view_shell.go with a middot
// -- which is how the drift was found.
//
// One positional rule replaces all of it: "•" starts a line, "·" joins within
// one. A reader tells them apart by where they sit, and this test tells them
// apart the same way, so the rule cannot quietly rot back.
//
// See docs/decisions.md 2026-09-10 (task 6.6, D-008).
func TestSeparatorGlyphsKeepTheirPositions(t *testing.T) {
	for _, lit := range appStringLiterals(t) {
		value := lit.value
		switch {
		case strings.Contains(value, "•"):
			body := strings.TrimLeft(value, " \n")
			if !strings.HasPrefix(body, "•") {
				t.Errorf("%s: \"•\" must start the line, not join within it -- use \"·\": %q", lit.where, value)
			}
			if strings.Count(value, "•") > 1 {
				t.Errorf("%s: one bullet per line; the rest of the joins are \"·\": %q", lit.where, value)
			}
		case strings.TrimSpace(value) == "·":
			// A literal that is only the joiner is a strings.Join separator. By
			// construction it never starts a line.
		case strings.HasPrefix(value, " "):
			// A leading space means the literal is a fragment concatenated onto
			// what sits to its left, so it joins by construction. Line-leading
			// indentation carries a bullet, never a join.
		case strings.HasPrefix(value, "·"):
			t.Errorf("%s: \"·\" joins within a line and cannot start one -- use \"•\": %q", lit.where, value)
		}
	}
}

// The whole point of D-007 is that there is one truncation marker. A bare
// "..." literal is that marker wearing three cells, and it is always appended
// to something the code just cut.
//
// The rule governs cuts, not prose. "Deleting branch..." is progress -- work in
// flight, DESIGN.md's one permitted kind of motion -- and reads as a sentence,
// not as a field that ran out of room. "HEAD...@{upstream}" is git range
// syntax. Neither is a marker, so neither is flagged.
func TestTruncationMarkerIsNotSpelledOut(t *testing.T) {
	for _, lit := range appStringLiterals(t) {
		if lit.value == "..." {
			t.Errorf("%s: append the shared ellipsis constant, not a literal \"...\"", lit.where)
		}
	}
}

// The app draws with two line vocabularies on purpose. Box-drawing characters
// frame the app -- rounded borders, the "│" pane split, the "─" rule -- and
// ASCII draws repository history, because "*", "|", "/" and "\\" are git's own
// topology notation and the diagonal box-drawing characters are not reliably
// monospaced anyway.
//
// The rule is that neither vocabulary takes the other's job. An ASCII "|" used
// as a field separator is the case that broke it: the status line spelled the
// same "join these on one line" idea a third way, next to a real "│" pane split
// and a real graph lane. Graph lanes arrive from git at runtime, never as a
// literal here, so this check cannot reach them.
//
// See docs/decisions.md 2026-09-10 (task 6.6, D-011).
func TestAsciiPipeIsNotAFieldSeparator(t *testing.T) {
	for _, lit := range appStringLiterals(t) {
		if strings.Contains(lit.value, " | ") {
			t.Errorf("%s: separators within a line are \"·\"; box drawing frames, ASCII draws history: %q", lit.where, lit.value)
		}
	}
}

type sourceLiteral struct {
	where string
	value string
}

func appStringLiterals(t *testing.T) []sourceLiteral {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	fset := token.NewFileSet()
	var literals []sourceLiteral
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			basic, ok := node.(*ast.BasicLit)
			if !ok || basic.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(basic.Value)
			if err != nil {
				return true
			}
			literals = append(literals, sourceLiteral{
				where: fset.Position(basic.Pos()).String(),
				value: value,
			})
			return true
		})
	}
	if len(literals) == 0 {
		t.Fatal("found no string literals to check; the walk is broken, not the vocabulary")
	}
	return literals
}

// Prose that outgrows its box has to say it was cut. This was found by looking
// at a rendered frame rather than by a test: at 80 columns the footer read
// "ctrl + u/d: s" and the Tags panel "Press F to sy", which read as broken
// rather than shortened -- the exact thing DESIGN.md's truncation rule and
// D-007 exist to prevent.
//
// fitVisibleWidth stays the primitive for text already built to fit, like the
// graph header and the "… +N hidden" indicators, where a marker would be noise.
func TestNarrowFramesMarkTheirProseCuts(t *testing.T) {
	status := git.Status{
		Root: "/repo", Branch: "main", Head: "a39d548",
		GraphCommits: []git.GraphCommit{{Hash: "a39d548", Subject: "a commit"}},
	}
	for _, width := range []int{60, 70, 80, 90} {
		m := model{repositoryState: repositoryState{repoStatus: status}}
		m.width, m.height = width, 28
		rendered := ansi.Strip(renderAppView(m))

		for _, sentence := range []struct{ full, prefix string }{
			{"ctrl + u/d: scroll", "ctrl + u/d: "},
			{"Tags not loaded yet.", "Tags not loa"},
			{"Press F to sync tag provenance.", "Press F to s"},
		} {
			for _, line := range strings.Split(rendered, "\n") {
				trimmed := strings.TrimRight(strings.Trim(line, "│ "), " ")
				if !strings.Contains(trimmed, sentence.prefix) {
					continue
				}
				if strings.HasSuffix(trimmed, sentence.full) {
					continue // it fit whole
				}
				if !strings.HasSuffix(trimmed, ellipsis) {
					t.Errorf("width %d: %q was cut without a marker", width, trimmed)
				}
			}
		}
	}
}

// A panel title is a name, and the "+N hidden" indicator is a sentence. Both
// were hard-cut at 60 columns -- "Graph Detai" and "… +2 hidd" -- which reads
// as a different panel and as a broken indicator.
func TestPanelTitlesAndIndicatorsMarkTheirCuts(t *testing.T) {
	status := git.Status{
		Root: "/repo", Branch: "main", Head: "a39d548",
		GraphCommits: []git.GraphCommit{{Hash: "a39d548", Subject: "a commit"}},
	}
	for _, width := range []int{50, 60, 70} {
		m := model{repositoryState: repositoryState{repoStatus: status}}
		m.width, m.height = width, 20
		for _, line := range strings.Split(ansi.Strip(renderAppView(m)), "\n") {
			for _, name := range []string{"Graph Detai", "+2 hidd", "[2] Loca", "[4] Tag"} {
				if !strings.Contains(line, name) {
					continue
				}
				// Either the whole thing fits, or the cut is marked.
				body := strings.TrimRight(strings.Trim(line, "│╭╮╰╯─ "), " ")
				if strings.Contains(body, name) && !strings.Contains(body, ellipsis) &&
					!strings.Contains(body, "Graph Details") && !strings.Contains(body, "hidden") &&
					!strings.Contains(body, "[2] Local") && !strings.Contains(body, "[4] Tags") {
					t.Errorf("width %d: %q was cut without a marker", width, body)
				}
			}
		}
	}
}
