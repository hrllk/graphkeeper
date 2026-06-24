# graphkeeper Package Split Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Split the current MVP into small, testable packages so graph rules, Git access, UI rendering, and app wiring each live in the right place.

**Architecture:** Keep `cmd/graphkeeper` thin and move feature code into `internal/`. Treat `internal/graph` as a pure domain layer for lane/order/focus/row rules, not as UI code. Keep `internal/git` for raw Git commands and repo data, `internal/ui` for styles and render helpers, and `internal/app` for Bubble Tea orchestration. `internal/bootstrap` means a composition root only: open the repo, build the model, wire dependencies, and start the program. If the eventual `graph` split turns out to be too small to justify its own package, prefer merging tiny leftovers back into `app` rather than keeping a weak package just to follow the plan. Keep the current docs simple: `docs/structure.md` explains the current map, and `docs/roadmap.md` explains the next steps. Treat `lazygit` as a reference for role separation only; do not copy its large `pkg/` tree.

**Tech Stack:** Go 1.24+, Bubble Tea, Lip Gloss, standard library tests, `go test ./...`.

**Test layout preference:** Keep tests next to the code they protect. Use files like `model_test.go`, `update_test.go`, `view_test.go`, `repo_test.go`, and `graph_test.go` in the same package folder. Do not create a root-level `test` package for this split; it would hide package internals and make refactors harder.

---

## Current state

The repo already has a thin entrypoint at `cmd/graphkeeper/main.go`, but `internal/app/model.go` and `internal/git/repo.go` still hold many jobs at once:

- graph node and lane helpers live inside `internal/app`
- graph log parsing and ref selection live inside `internal/git`
- rendering helpers and styles live inside `internal/app/view.go`
- tests exist, but they mostly cover the current mixed layout

The goal of this plan is to split those responsibilities without changing user-facing behavior.

---

## Documentation strategy

Keep only these docs active:

- `docs/structure.md` — current code map
- `docs/roadmap.md` — next work order

Move any extra design notes or old implementation notes into `docs/archive/`.
If the split changes the package map, update `docs/structure.md` after the code lands so the doc stays honest.

---

## Target package map

```text
cmd/
  graphkeeper/
    main.go

internal/
  app/
    model.go
    update.go
    view.go
    commands.go
    navigation.go
    actions.go
  bootstrap/
    app.go
  git/
    repo.go
    status.go
    refs.go
    graph_log.go
  graph/
    lane.go
    layout.go
    sort.go
    focus.go
    rows.go
  ui/
    theme.go
    widgets.go
    render.go
  state/
    state.go
  telemetry/
    telemetry.go
```

Notes:
- `internal/bootstrap` is usually unnecessary unless the wiring starts to spread across many files.
- Do not create extra packages just to split files.
- Prefer fewer packages with clear responsibilities over many tiny packages.

---

## Safety baseline before refactor

### Task 0: Freeze current behavior with tests

**Objective:** Lock down the current graph and UI behavior before moving code.

**Files:**
- Modify: `internal/app/model_test.go`
- Modify: `internal/git/repo_test.go`
- Create if needed: `internal/graph/*_test.go`

**What to cover first:**
- graph row order for a small sample history
- lane order when branches diverge
- focus rules for the selected graph row
- selection/fallback behavior for empty sections
- Git log argument construction for the current graph view
- current build path: `go build ./cmd/graphkeeper`

**Expected result:**
The tests describe the current behavior clearly enough that later refactors can move code without changing output.

**Validation:**
- Run `go test ./...`
- Run `go build ./cmd/graphkeeper`

---

## Implementation plan

### Task 1: Extract pure graph rules into `internal/graph`

**Objective:** Move graph-domain logic out of app and Git code so it can be tested independently.

**Files:**
- Create: `internal/graph/lane.go`
- Create: `internal/graph/layout.go`
- Create: `internal/graph/sort.go`
- Create: `internal/graph/focus.go`
- Create: `internal/graph/rows.go`
- Create: `internal/graph/lane_test.go`
- Create: `internal/graph/layout_test.go`
- Create: `internal/graph/focus_test.go`
- Modify: `internal/app/model.go`
- Modify: `internal/app/view.go`
- Modify: `internal/git/repo.go` only if it still holds graph-only helpers

**What moves here:**
- lane order
- commit order
- focus rules
- graph row rules
- row sorting and row selection helpers

**Test strategy:**
Write table-driven tests for each pure rule:
- empty graph
- one-branch graph
- diverged branches
- remote-tracking refs
- branch with no upstream
- connector rows vs commit rows
- focused row selection around boundaries

**Validation:**
- `go test ./internal/graph/...`
- `go test ./...`

**Done when:**
- `internal/graph` has no UI imports
- graph ordering logic no longer lives in `internal/app`
- tests explain the rules clearly

---

### Task 2: Split Git access from graph interpretation

**Objective:** Keep raw Git calls and parsed repo data in `internal/git`, while graph rules stay in `internal/graph`.

**Files:**
- Modify: `internal/git/repo.go`
- Create: `internal/git/status.go`
- Create: `internal/git/refs.go`
- Create: `internal/git/graph_log.go`
- Create: `internal/git/status_test.go`
- Modify: `internal/git/repo_test.go`
- Modify: `internal/app/model.go`

**What to move here:**
- `git status` collection
- branch metadata collection
- Git command wrappers
- graph log command building
- raw parse helpers for `git log` output

**Important boundary:**
`internal/git` should return raw or lightly parsed repository data.
`internal/graph` should decide how that data becomes lanes, rows, and focus.

**Test strategy:**
Add tests for:
- graph log arguments
- ref selection rules
- parsing of branch/upstream lines
- parsing of graph commit lines
- fallback behavior when the repo is empty or has no upstream

**Validation:**
- `go test ./internal/git/...`
- `go test ./...`

---

### Task 3: Split Bubble Tea app orchestration inside `internal/app`

**Objective:** Make the app layer easier to read by separating update, commands, navigation, actions, and rendering.

**Files:**
- Modify: `internal/app/model.go`
- Create: `internal/app/update.go`
- Create: `internal/app/commands.go`
- Create: `internal/app/navigation.go`
- Create: `internal/app/actions.go`
- Modify: `internal/app/view.go`
- Modify: `internal/app/model_test.go`
- Create: `internal/app/update_test.go`
- Create: `internal/app/commands_test.go`
- Create: `internal/app/navigation_test.go`
- Create: `internal/app/actions_test.go`

**What belongs here:**
- Bubble Tea model
- message handling
- key handling
- command creation
- action dispatch
- navigation state updates

**Test strategy:**
Keep or add tests for:
- state transitions
- blocked/outcome modes
- section cycling
- cursor movement boundaries
- graph focus movement
- action preview / confirm behavior

**Validation:**
- `go test ./internal/app/...`
- `go test ./...`

**Done when:**
- `model.go` is mostly data + small init code
- `update.go` owns message handling
- `view.go` focuses on rendering only

---

### Task 4: Move visual helpers into `internal/ui`

**Objective:** Separate style and render helpers from app logic.

**Files:**
- Create: `internal/ui/theme.go`
- Create: `internal/ui/widgets.go`
- Create: `internal/ui/render.go`
- Create: `internal/ui/render_test.go`
- Modify: `internal/app/view.go`

**What belongs here:**
- Lip Gloss styles
- reusable widget helpers
- shared formatting helpers
- rendering utilities that do not need app state mutation

**Test strategy:**
Add tests that verify:
- important strings still render in the right order
- empty-state blocks still show the expected message
- highlight and fallback formatting stay stable

If full snapshot tests are too heavy, use string-level tests for stable helpers.

**Validation:**
- `go test ./internal/ui/...`
- `go test ./...`

---

### Task 5: Keep entrypoint thin and wire packages together

**Objective:** Make sure the executable stays a simple boot path.

**Files:**
- Modify: `cmd/graphkeeper/main.go`
- Create if needed: `internal/bootstrap/app.go`
- Modify: `internal/app/model.go`

**Expected shape:**
- open repo
- build app model
- run Bubble Tea program
- handle errors once in main

**Test strategy:**
The entrypoint itself may not need deep unit tests, but the package wiring must still compile and the full suite must pass.

**Validation:**
- `go build ./cmd/graphkeeper`
- `go test ./...`

---

### Task 6: Update docs after the code lands

**Objective:** Keep the docs honest after the split.

**Files:**
- Modify: `docs/structure.md`
- Modify: `docs/roadmap.md` only if the order changes
- Modify: `README.md` only if the build path or package names change

**What to update:**
- the current tree in `docs/structure.md`
- package responsibilities
- any old notes that are now wrong
- the new file names if `internal/graph` and `internal/ui` land

**Validation:**
- confirm the doc tree matches the real tree
- confirm the build command still points at `cmd/graphkeeper`

---

## Test policy for the split

Use strict, focused tests as each piece moves:

- Prefer table-driven tests for pure helpers.
- Add regression tests before moving code.
- Do not rely only on one big integration test.
- Keep tests close to the package they protect.
- For graph rules, test edge cases as well as normal flows.
- For UI helpers, test stable rendered strings instead of manual visual checks when possible.

The main goal is to preserve behavior while the packages get smaller.

---

## Suggested commit order

1. Baseline tests for current behavior
2. `internal/graph` extraction
3. `internal/git` cleanup
4. `internal/app` file split
5. `internal/ui` extraction
6. entrypoint wiring cleanup
7. docs sync
8. final `go test ./...` and `go build ./cmd/graphkeeper`

---

## Risks and tradeoffs

- Do not create packages too early just because a file feels big.
- Avoid moving code and changing behavior in the same step unless a test already covers it.
- Keep `cmd/graphkeeper` thin, but do not over-engineer bootstrap code.
- If tests become too brittle, prefer helper-level tests over full output snapshots.
- If a function is already stable and simple, leaving it in place is better than splitting for its own sake.

---

## Open questions for review

1. Should `internal/bootstrap` exist, or should `cmd/graphkeeper/main.go` stay the only wiring layer?
2. Should graph parsing live in `internal/git` or move fully into `internal/graph` after the first extraction step?
3. Do we want string-based UI tests only, or also a small set of snapshot-style rendering tests?
4. Which package should own graph row formatting after the split: `internal/graph` or `internal/ui`?

---

## Review checkpoint

Please review this plan before implementation starts.
Once approved, the split should happen in small commits, and each extracted package should get its own strict test coverage.
