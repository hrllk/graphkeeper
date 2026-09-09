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
		case strings.HasPrefix(strings.TrimLeft(value, " \n"), "·"):
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
