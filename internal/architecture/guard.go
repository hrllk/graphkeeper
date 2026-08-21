package architecture

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ImportViolation struct {
	File       string
	Package    string
	ImportPath string
}

func (v ImportViolation) Error() string {
	return fmt.Sprintf("%s (%s) imports forbidden %q", v.File, v.Package, v.ImportPath)
}

type LedgerEntry struct {
	Symbol              string
	ForbiddenDependency string
	Owner               string
	RemovalCondition    string
}

func DefaultForbiddenImports() []string {
	return []string{
		"github.com/charmbracelet/bubbletea",
		"github.com/charmbracelet/lipgloss",
		"hrllk/graphkeeper/internal/git",
		"hrllk/graphkeeper/internal/telemetry",
		"hrllk/graphkeeper/internal/app",
		"hrllk/graphkeeper/internal/graph",
		"hrllk/graphkeeper/internal/state",
		"os/exec",
		"os",
		"path/filepath",
		"encoding/json",
	}
}

func forbiddenImportsForPackage(packagePath string) []string {
	return DefaultForbiddenImports()
}

func ScanSource(source, packagePath, fileName string, forbidden []string) ([]ImportViolation, error) {
	return scanSourceWithLedger(source, packagePath, fileName, forbidden, nil)
}

func ScanSourceWithLedger(source, packagePath, fileName string, forbidden []string, ledger []LedgerEntry) ([]ImportViolation, error) {
	return scanSourceWithLedger(source, packagePath, fileName, forbidden, ledger)
}

func scanSourceWithLedger(source, packagePath, fileName string, forbidden []string, ledger []LedgerEntry) ([]ImportViolation, error) {
	file, err := parser.ParseFile(token.NewFileSet(), fileName, source, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", fileName, err)
	}
	if packagePath == "cmd/graphkeeper" {
		return nil, nil
	}
	violations := make([]ImportViolation, 0)
	for _, spec := range file.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		for _, forbiddenPath := range forbidden {
			if path != forbiddenPath && !strings.HasPrefix(path, forbiddenPath+"/") {
				continue
			}
			if ledgerAllows(fileName, path, ledger) {
				continue
			}
			violations = append(violations, ImportViolation{File: fileName, Package: packagePath, ImportPath: path})
			break
		}
	}
	return violations, nil
}

func ScanDir(root, packagePath string, forbidden []string) ([]ImportViolation, error) {
	return ScanDirWithLedger(root, packagePath, forbidden, nil)
}

func ScanDirWithLedger(root, packagePath string, forbidden []string, ledger []LedgerEntry) ([]ImportViolation, error) {
	dir := filepath.Join(root, filepath.FromSlash(packagePath))
	violations := make([]ImportViolation, 0)
	scanned := 0
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		scanned++
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		found, err := scanSourceWithLedger(string(source), packagePath, path, forbidden, ledger)
		if err != nil {
			return err
		}
		violations = append(violations, found...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if scanned == 0 {
		return nil, fmt.Errorf("architecture root %q contains no production Go files", packagePath)
	}
	return violations, nil
}

func ParseLedger(content string) ([]LedgerEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNo := 0
	state := 0
	entries := make([]LedgerEntry, 0)
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitLedgerRow(line)
		if state == 0 {
			if len(cells) == 4 && cells[0] == "Package/file/symbol" && cells[1] == "Forbidden dependency" && cells[2] == "Owner" && cells[3] == "Removal condition" {
				state = 1
				continue
			}
			return nil, fmt.Errorf("ledger header missing or invalid at line %d", lineNo)
		}
		if state == 1 {
			if len(cells) != 4 || !isSeparatorRow(cells) {
				return nil, fmt.Errorf("ledger separator missing or invalid at line %d", lineNo)
			}
			state = 2
			continue
		}
		if len(cells) != 4 {
			return nil, fmt.Errorf("ledger row at line %d has %d columns; want 4", lineNo, len(cells))
		}
		for i, cell := range cells {
			if cell == "" {
				return nil, fmt.Errorf("ledger row at line %d column %d is empty", lineNo, i+1)
			}
		}
		if !validOwner(cells[2]) {
			return nil, fmt.Errorf("ledger owner at line %d is invalid: %q", lineNo, cells[2])
		}
		if !validRemovalCondition(cells[3]) {
			return nil, fmt.Errorf("ledger removal condition at line %d is invalid: %q", lineNo, cells[3])
		}
		entries = append(entries, LedgerEntry{Symbol: cells[0], ForbiddenDependency: cells[1], Owner: cells[2], RemovalCondition: cells[3]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if state != 2 {
		return nil, fmt.Errorf("ledger table is incomplete")
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("ledger contains no entries")
	}
	return entries, nil
}

func ValidateLedger(content string) error {
	_, err := ParseLedger(content)
	return err
}

func ledgerAllows(fileName, importPath string, ledger []LedgerEntry) bool {
	fileName = filepath.ToSlash(fileName)
	for _, entry := range ledger {
		symbol := strings.SplitN(entry.Symbol, ":", 2)[0]
		symbol = filepath.ToSlash(strings.Trim(strings.TrimSpace(symbol), "`"))
		dependency := normalizeDependency(entry.ForbiddenDependency)
		if (fileName == symbol || strings.HasSuffix(fileName, "/"+symbol)) && dependencyMatches(dependency, importPath) {
			return true
		}
	}
	return false
}

func normalizeDependency(dependency string) string {
	dependency = strings.TrimSpace(strings.Trim(dependency, "`"))
	switch dependency {
	case "*git.Repo", "git.Status", "git.CommitInspection":
		return "hrllk/graphkeeper/internal/git"
	default:
		return dependency
	}
}

func dependencyMatches(dependency, importPath string) bool {
	return dependency == importPath || dependency == "internal/git" && strings.HasSuffix(importPath, "/internal/git")
}

var ownerPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:/[0-9]+\.[0-9]+)*$`)

func validOwner(owner string) bool {
	return ownerPattern.MatchString(owner)
}

func validRemovalCondition(condition string) bool {
	condition = strings.TrimSpace(condition)
	if len(condition) < 8 {
		return false
	}
	switch strings.ToLower(condition) {
	case "migrate", "remove", "pending", "tbd":
		return false
	default:
		return true
	}
}

func isSeparatorRow(cells []string) bool {
	for _, cell := range cells {
		cell = strings.Trim(cell, "-")
		cell = strings.Trim(cell, ":")
		if cell != "" {
			return false
		}
	}
	return true
}

func splitLedgerRow(line string) []string {
	line = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|"))
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
