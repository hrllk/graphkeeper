package architecture

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestScannerRejectsForbiddenImportInExtractedPackage(t *testing.T) {
	violations, err := ScanSource("package sample\nimport \"hrllk/graphkeeper/internal/git\"\n", "internal/commitinspector", "fixture.go", DefaultForbiddenImports())
	if err != nil {
		t.Fatalf("scan fixture: %v", err)
	}
	if len(violations) != 1 || violations[0].ImportPath != "hrllk/graphkeeper/internal/git" {
		t.Fatalf("expected one Git import violation, got %#v", violations)
	}
}

func TestScannerAllowsCompositionRootGitWiring(t *testing.T) {
	violations, err := ScanSource("package main\nimport \"hrllk/graphkeeper/internal/git\"\n", "cmd/graphkeeper", "fixture.go", DefaultForbiddenImports())
	if err != nil {
		t.Fatalf("scan fixture: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("composition root should be allowed, got %#v", violations)
	}
}

func TestLedgerValidationFailsClosed(t *testing.T) {
	if err := ValidateLedger("# missing table\n"); err == nil {
		t.Fatal("expected malformed ledger to fail")
	}
	if err := ValidateLedger("| Package/file/symbol | Forbidden dependency | Owner | Removal condition |\n|---|---|---|---|\n| internal/app/model.go:model.repo | git.Repo | 2.7 | selected flow migrated |\n"); err != nil {
		t.Fatalf("valid ledger rejected: %v", err)
	}
	if err := ValidateLedger("| Package/file/symbol | Forbidden dependency | Owner | Removal condition |\n|---|---|---|---|\n| x | y | owner | selected flow migrated |\n"); err == nil {
		t.Fatal("expected invalid owner to fail")
	}
}

func TestLedgerExceptionIsUsedByScanner(t *testing.T) {
	ledger := "| Package/file/symbol | Forbidden dependency | Owner | Removal condition |\n|---|---|---|---|\n| `internal/app/model.go:model.repo` | `*git.Repo` | 2.7 | selected flow migrated |\n"
	entries, err := ParseLedger(ledger)
	if err != nil {
		t.Fatalf("parse ledger: %v", err)
	}
	violations, err := ScanSourceWithLedger("package app\nimport \"hrllk/graphkeeper/internal/git\"\n", "internal/app", "internal/app/model.go", DefaultForbiddenImports(), entries)
	if err != nil {
		t.Fatalf("scan legacy exception: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("owned legacy exception should pass, got %#v", violations)
	}
}

func TestRepositoryArchitectureGuard(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	ledger, err := os.ReadFile(filepath.Join(root, "internal", "architecture", "legacy-ledger.md"))
	if err != nil {
		t.Fatalf("read repository ledger: %v", err)
	}
	if err := ValidateLedger(string(ledger)); err != nil {
		t.Fatalf("validate repository ledger: %v", err)
	}
	entries, err := ParseLedger(string(ledger))
	if err != nil {
		t.Fatalf("parse repository ledger: %v", err)
	}
	for _, rel := range []string{"internal/architecture/testdata/extracted", "internal/commitinspector", "internal/events"} {
		dir := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if rel == "internal/architecture/testdata/extracted" {
				t.Fatalf("required architecture fixture root is missing: %s", rel)
			}
			continue
		}
		violations, err := ScanDirWithLedger(root, rel, forbiddenImportsForPackage(rel), entries)
		if err != nil {
			t.Fatalf("scan %s: %v", rel, err)
		}
		if len(violations) != 0 {
			t.Fatalf("forbidden imports in %s: %#v", rel, violations)
		}
	}

	// Existing legacy packages are scanned to keep the baseline observable, but
	// their known coupling is not treated as a new-boundary failure. The
	// repository ledger tracks removal ownership for this baseline separately.
	for _, rel := range []string{"internal/app", "internal/git", "internal/graph", "internal/adapter"} {
		if _, err := ScanDirWithLedger(root, rel, forbiddenImportsForPackage(rel), entries); err != nil {
			t.Fatalf("scan legacy baseline %s: %v", rel, err)
		}
	}
}
