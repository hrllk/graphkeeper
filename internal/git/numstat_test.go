package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The two record shapes `git diff-tree --numstat -z` emits, verbatim from git.
// The parser used to understand only the second, so on an ordinary commit it
// mapped the empty string to the first file's counts and every file came out
// with zero additions and zero deletions.
func TestParseNumstatZReadsBothRecordShapes(t *testing.T) {
	t.Run("ordinary records", func(t *testing.T) {
		got := parseNumstatZ("1\t1\trenamed.txt\x001\t0\tsecond.txt\x00")
		want := map[string][2]string{
			"renamed.txt": {"1", "1"},
			"second.txt":  {"1", "0"},
		}
		assertCounts(t, got, want)
	})

	t.Run("rename record", func(t *testing.T) {
		// The path slot is empty and the old and new paths follow as their own
		// fields. The new path is the file's identity.
		got := parseNumstatZ("1\t1\t\x00original.txt\x00renamed.txt\x00")
		assertCounts(t, got, map[string][2]string{"renamed.txt": {"1", "1"}})
	})

	t.Run("a rename beside an ordinary file", func(t *testing.T) {
		got := parseNumstatZ("1\t1\t\x00old.txt\x00new.txt\x004\t2\tplain.txt\x00")
		assertCounts(t, got, map[string][2]string{
			"new.txt":   {"1", "1"},
			"plain.txt": {"4", "2"},
		})
	})

	t.Run("binary", func(t *testing.T) {
		// "-\t-" is how git says binary, and it is what sets ChangedFile.Binary.
		// The old parser never reached it, so the Inspector's binary marker
		// could not fire.
		got := parseNumstatZ("-\t-\timage.png\x00")
		assertCounts(t, got, map[string][2]string{"image.png": {"-", "-"}})
	})

	t.Run("empty and truncated input", func(t *testing.T) {
		for name, in := range map[string]string{
			"empty":             "",
			"trailing NUL only": "\x00",
			"no tabs":           "garbage\x00",
			"truncated rename":  "1\t1\t\x00old.txt\x00",
		} {
			if got := parseNumstatZ(in); len(got) != 0 {
				t.Errorf("%s: expected nothing parsed, got %v", name, got)
			}
		}
	})
}

func assertCounts(t *testing.T, got, want map[string][2]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d records, want %d: %v", len(got), len(want), got)
	}
	for path, counts := range want {
		if got[path] != counts {
			t.Errorf("%s = %v, want %v", path, got[path], counts)
		}
	}
}

// End to end, against git. The parser bug meant every file arrived with zero
// additions and zero deletions and no binary file was ever recognised, so the
// Inspector's binary marker could not fire and the --summary pass marked
// anything with a mode change as ModeOnly.
func TestInspectCommitCarriesCountsAndBinary(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@e.com",
			"GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@e.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name string, content []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	run("init", "-q")
	write("a.txt", []byte("one\ntwo\n"))
	write("img.bin", []byte{0, 1, 2, 0, 3})
	run("add", "-A")
	run("commit", "-qm", "base")
	write("a.txt", []byte("one\ntwo\nthree\n"))
	write("img.bin", []byte{9, 9, 9, 0, 7})
	run("add", "-A")
	run("commit", "-qm", "change")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	head, err := repo.git(context.Background(), "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := repo.InspectCommit(context.Background(), head)
	if err != nil {
		t.Fatal(err)
	}

	byPath := map[string]CommitDiffFile{}
	for _, file := range inspection.Files {
		byPath[file.Path] = file
	}
	if text := byPath["a.txt"]; text.Additions != 1 || text.Deletions != 0 || text.Binary {
		t.Errorf("a.txt = +%d -%d binary=%v, want +1 -0 and not binary", text.Additions, text.Deletions, text.Binary)
	}
	if binary := byPath["img.bin"]; !binary.Binary {
		t.Errorf("img.bin was not recognised as binary: %+v", binary)
	}
}

// A rename must reach the file list with both paths and its own counts, rather
// than as the delete-plus-add git reports without -M.
func TestInspectCommitDetectsRenames(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@e.com",
			"GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@e.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-q")
	body := ""
	for i := 0; i < 60; i++ {
		body += "line\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "original.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-qm", "base")
	run("mv", "original.txt", "renamed.txt")
	if err := os.WriteFile(filepath.Join(dir, "renamed.txt"), []byte("changed\n"+body), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-qm", "rename")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	head, err := repo.git(context.Background(), "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := repo.InspectCommit(context.Background(), head)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.Files) != 1 {
		t.Fatalf("a rename is one file, got %d: %+v", len(inspection.Files), inspection.Files)
	}
	file := inspection.Files[0]
	if file.Status != "R" || file.Path != "renamed.txt" || file.OldPath != "original.txt" {
		t.Fatalf("expected a rename from original.txt to renamed.txt, got %+v", file)
	}
	// numstat now agrees with the status listing, so the counts land on the
	// renamed file instead of being split across two paths that do not exist.
	if file.Additions != 1 || file.Deletions != 0 {
		t.Errorf("rename counts = +%d -%d, want +1 -0", file.Additions, file.Deletions)
	}
}
