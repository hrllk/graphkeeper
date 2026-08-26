# QA Report — graphkeeper alpha.6 (live TUI)

- **Artifact**: `graphkeeper alpha.6`, built via `./scripts/build`
- **Source**: `main` @ `4d8fcbc`, working tree clean except untracked taskmaster json
- **sha256**: `0f2966bb5a8843d4d0ae6379e2111d539c2a28da31dfce70673105636392f31a`
- **Method**: tmux pty driving against a synthetic fixture repo; cross-checked on the real graphkeeper repo
- **Date**: 2026-08-26
- **Scope**: report-only, no code changed
- **Taskmaster**: task 6

## Static gates — all pass

| Gate | Result |
| --- | --- |
| `go build ./...` | clean |
| `go vet ./...` | no output |
| `go test ./...` | all packages `ok` |
| CLI vs `docs/cli-reference.md` | conforms (`-h`/`--help`/`--version` → 0, unknown option → 2, positional → 2, help beats version, startup error → 1) |

## Findings

### HIGH-1 — Stash entries never load

The `S` stash list reports `(no stash entries)` and the graph `state` column shows no `S`
marker, in a repo that has stashes.

- Fixture: 3 stashes (2 pathspec, 1 `--include-untracked`) on `78442a4` = main HEAD → `(no stash entries)`
- Real repo: `stash@{0}: On develop: theme plan` → `(no stash entries)`
- `git stash list --format=%gd%x1f%H%x1f%P%x1f%gs` (the exact command the app runs) returns valid rows, and `Repo.Stashes` parses them correctly.

Cause: `loadStashState` is only reachable from `loadedMsg` / `refreshedMsg`
(`internal/app/update_lifecycle.go:99`, `:139`). Production always wires
`RepositoryRead` (`cmd/graphkeeper/main.go:76`), so `Init` takes the snapshot path and
`refreshedMsg` early-returns at `internal/app/update_lifecycle.go:103-104`.
`m.stashEntries` is therefore always empty in the shipped binary.

Impact: a maintainer checking "is there local work to stash before switching focus?" —
a stated purpose of the tool — gets a confident wrong answer.

### HIGH-2 — Local tags never load at startup

Tags panel shows `No local tags found. / Press F to sync tag provenance.` and the graph
`state` column has no `T` markers until `F` is pressed manually.

- Fixture: 2 tags (`v0.1.0` annotated, `v0.2.0` lightweight) → "No local tags found."
- Real repo: 6 tags → "No local tags found."
- After `F`: both tags appear correctly, annotated tag dereferenced correctly, `T` markers appear in the graph.

Cause: same class as HIGH-1. `loadLocalTagStatus` is called only from the legacy
`loadRepoState` / `refreshRepoState` paths (`internal/app/commands.go:31`, `:111`).

The message is not merely stale — it asserts absence.

### HIGH-3 — Commit Inspector diff cannot be scrolled

An 800-line diff renders lines 1–39 and nothing else. 45 × `Ctrl+D` moved nothing.
Also tried `Ctrl+U`, `Down`, `PageDown`, `Space`, `Ctrl+F`, `G`, `j`, `k`, `Ctrl+E` — none scroll.
The hunk header reads `@@ -0,0 +1,800 @@`, so the data is loaded (`MaxLines` defaults to 2000);
761 lines are simply unreachable, with **no truncation indicator**.

Cause: `ctrl+u` / `ctrl+d` update `m.commitInspectorScroll`
(`internal/app/commit_inspector.go:98-105`), but the active screen renderer indexes
`diffLines[i]` from 0 and never reads the offset
(`internal/app/commit_inspector_screen.go:144-151`). The offset is applied only in the
dead legacy popup renderer (`internal/app/commit_inspector.go:218`).

### HIGH-4 — `?` in the Commit Inspector opens an invisible overlay that swallows input

Pressing `?` produces a **byte-identical** screen (verified by diffing captures), but the
next keypress is consumed: `Esc` #1 closes the invisible overlay, `Esc` #2 closes the
Inspector. `q`, `j`, `Enter` are likewise eaten.

Cause: `commitInspectorHelp` is toggled at `internal/app/commit_inspector.go:60-63` and
gates all key handling at `:75-77`, but `renderCommitInspectorScreen` never renders it.
The help text exists only in the dead popup renderer
(`internal/app/commit_inspector.go:153`).

User-visible effect: the documented `? help` key makes the Inspector look frozen.

### MED-5 — A pure rename renders as a whole-file addition

`R file1.txt → file1-renamed.txt` is a 100 % similarity rename; `git show --stat` reports
`file1.txt => file1-renamed.txt | 0`. The Inspector shows `@@ -0,0 +1,3 @@` with three `+`
lines — exactly `git diff --no-renames` output.

The changed-files list applies rename detection (status `R`, old→new path) while the diff
loader does not. This overstates the change, on a tool whose job is reading history accurately.

### MED-6 — Changed-files list does not scroll; the cursor marker vanishes

60-file commit at 200×50: the pane renders `many/f01.txt` … `many/f40.txt`. With the
selection on `many/f60.txt` (confirmed by the `path:` header and the diff content), **no
`>` marker appears anywhere on screen** — the user cannot see what is selected.

Same missing-offset defect as HIGH-3 (`fileRows[i]` from 0).

### MED-7 — `q` does not close the Commit Inspector

Spec Screen Contract requires the footer `q close  Esc back  ? help`, and the User Outcome
says "`q` 또는 `Esc`로 … 돌아갈 수 있어야 한다". README documents a `q/Esc/?` footer.
The rendered footer is `Esc back   ? help` and `q` is a no-op. Only `Esc` closes.

### MED-8 — `q` does not close the `?` hotkey overlay

README: "In an open overlay, `q` closes it and `esc` keeps its existing close/back behavior."
`q` is a no-op; only `esc` closes. (It does not quit the app, so nothing is lost — the
documented contract is simply unimplemented.)

### MED-9 — The `?` overlay omits documented, working keys

| Section | Missing from overlay |
| --- | --- |
| Graph | `enter` (open Commit Inspector — the headline feature), `S` (stash list), `F` (sync tag provenance) |
| Remote | `p` (pull), `d` (delete remote branch) |
| Tags | `P` (push tag), `F` (sync tag provenance) |

`F` matters most: tags do not load without it (HIGH-2), yet it is advertised only in the
Tags panel's empty-state text.

### LOW-10 — Binary files are not marked in the file list

`blob.bin` renders as `A blob.bin`. The renderer supports `[binary]`
(`internal/app/commit_inspector_screen.go:159-161`) but the adapter never sets
`Binary` / `StatusBinary`. The diff pane says `No textual changes`; the spec's wording is
`No textual diff`.

### LOW-11 — Inspector header row 1 breaks its own label convention

Renders `COMMIT <hash>`; the spec Screen Contract says `commit: <full hash>`, and rows 2–4
use the lowercase `message:` / `author:` / `path:` form.

### LOW-12 — `unsupported height` is raw developer text, shown in a mixed state

At 60×12 the diff pane shows `unsupported height` **and** a rendered diff below it.
The string is also developer-facing wording in a user surface.

### LOW-13 — Inspector width is inconsistent with the main screen

The main screen centres its body at 180 columns inside a 200-column terminal
(DESIGN.md: "main body centered inside the available viewport"). The Inspector uses the
full 200. Opening and closing it jumps the frame width.

### UNREPRODUCED-14 — Frame overflow to 214×59 in a 200×50 terminal

Observed once during the 60-file navigation run: the rendered frame was 14 columns wider
and 9 rows taller than the viewport, which in a real terminal would wrap lines and scroll
the top away. Four further attempts (keypress delays 0.05 / 0.1 / 0.2 / 0.35 s) did not
reproduce it. Capture kept at `shots/09-overflow.txt`. Reported as an observation, not a
confirmed defect.

## Verified working

- Layout centring and box geometry exact at 200×50 (180 wide, 20/20 margins)
- CJK + emoji column alignment exact — every row measured at exactly 200 display cells
- 80×24 rendering: author column drops, titles ellipsise, fits exactly
- Live resize 80×24 → 140×40 → 60×12 → 60×10, always within bounds
- Graph `Ctrl+U`/`Ctrl+D` paging, `H` jump-to-HEAD, `tab` cycling, `Esc` close
- Deleted-file diff, unicode diff, binary fallback text, annotated-tag dereference
- Local panel state: `(dirty)`, `⬇` behind-remote, `(no-up)` no-upstream — all correct
- Async load: no crash or hang across ~500 keypresses; stderr stayed empty throughout

## Fixture

Synthetic repo built in the session scratchpad (`qa/fixture/work`) with a bare origin:
16 commits, diverged feature branch, FF-able hotfix, main 1 behind origin, annotated +
lightweight tags, 3 stashes, dirty tree + untracked file, unicode/emoji commit, binary
blob, rename + delete, 800-line diff, 60-file commit, long subject lines.
