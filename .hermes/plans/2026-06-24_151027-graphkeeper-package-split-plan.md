# graphkeeper Package Split Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

### TOC
- [Goal](#goal)
- [Current context](#current-context)
- [Decision summary](#decision-summary)
- [Documentation strategy](#documentation-strategy)
- [Target package map](#target-package-map)
- [Baseline tests before refactor](#baseline-tests-before-refactor)
- [Implementation order](#implementation-order)
- [Test layout rule](#test-layout-rule)
- [Risks and tradeoffs](#risks-and-tradeoffs)
- [Review questions](#review-questions)

### Goal
Split the MVP into small, testable pieces so graph rules, Git access, app orchestration, and rendering each have a clear home.

### Current context
The repo already has a thin entrypoint at `cmd/graphkeeper/main.go`, but the current app and Git packages still mix several responsibilities:
- graph node helpers and row logic still live inside `internal/app`
- graph-log parsing and ref selection still live inside `internal/git`
- rendering helpers and styles still live inside `internal/app/view.go`
- tests exist, but they mostly protect the current mixed layout

The refactor should keep behavior stable while making each responsibility easier to test.

### Decision summary
- Keep `internal/graph` only if it still has a real domain job after the gitlog change.
- Do not create `internal/bootstrap` yet. Keep `cmd/graphkeeper/main.go` as the only wiring layer unless the entrypoint grows again.
- Use package-local `_test.go` files next to the code they protect. Do not introduce a root-level `test` package.

### Documentation strategy
Keep only these docs active:
- `docs/structure.md` — current code map
- `docs/roadmap.md` — next work order

Move extra notes and old implementation docs into `docs/archive/`.
If the package map changes, update `docs/structure.md` after the code lands so the doc matches reality.

### Target package map
```text
cmd/
  graphkeeper/
    main.go

internal/
  app/
    model.go
    update.go
    commands.go
    navigation.go
    actions.go
    view.go
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
- `internal/bootstrap` is intentionally absent.
- If `internal/graph` becomes too small to justify itself, merge the leftover rules back into `internal/app` or `internal/git` instead of keeping a weak package.
- Prefer fewer packages with clear responsibilities over many tiny packages.

### Baseline tests before refactor
Write regression tests before moving code.

Cover first:
- graph row order for a small sample history
- lane order when branches diverge
- focus rules for the selected graph row
- selection and fallback behavior for empty sections
- Git log argument construction for the current graph view
- current build path: `go build ./cmd/graphkeeper`

Validation:
- `go test ./...`
- `go build ./cmd/graphkeeper`

### Implementation order
#### Task 1: Freeze current behavior
Add or tighten tests in:
- `internal/app/model_test.go`
- `internal/git/repo_test.go`
- `internal/graph/*_test.go` if the package exists during the split

Goal: lock down the current behavior before any moves.

#### Task 2: Decide whether `internal/graph` stays separate
If graph rules still have real domain logic after the gitlog shift, move lane/order/focus/row helpers into `internal/graph`.
If not, keep the logic in `internal/app` and `internal/git` and remove the package from the plan.

Goal: avoid a weak package that exists only for symmetry.

#### Task 3: Split Git access from graph interpretation
Keep raw Git calls and parsed repo data in `internal/git`.
Move raw Git command wrappers, ref parsing, and graph-log parsing into focused files such as:
- `internal/git/status.go`
- `internal/git/refs.go`
- `internal/git/graph_log.go`

Goal: make `internal/git` the source of repo data, not UI rules.

#### Task 4: Split Bubble Tea orchestration inside `internal/app`
Split the app layer into smaller files:
- `model.go`
- `update.go`
- `commands.go`
- `navigation.go`
- `actions.go`
- `view.go`

Add matching tests next to each file as needed.

Goal: keep each file small and easy to scan.

#### Task 5: Move render helpers into `internal/ui`
Move reusable styles and render helpers into `internal/ui` when they are not tied to state changes.

Goal: separate presentation from app state.

#### Task 6: Sync docs after code lands
Update:
- `docs/structure.md`
- `docs/roadmap.md` only if the order changes
- `README.md` only if build or package names change

Goal: keep docs honest and simple.

### Test layout rule
Use package-local tests next to the code they protect.

Recommended filenames:
- `internal/app/model_test.go`
- `internal/app/update_test.go`
- `internal/app/commands_test.go`
- `internal/app/navigation_test.go`
- `internal/app/actions_test.go`
- `internal/app/view_test.go`
- `internal/git/repo_test.go`
- `internal/git/status_test.go`
- `internal/git/refs_test.go`
- `internal/git/graph_log_test.go`
- `internal/graph/lane_test.go`
- `internal/graph/layout_test.go`
- `internal/graph/focus_test.go`
- `internal/graph/rows_test.go`

Why this layout:
- code and tests stay close together
- package-private helpers are easy to test
- refactors are easier to follow
- no root-level `test` package is needed

### Risks and tradeoffs
- If `internal/graph` becomes too small, it is better to remove it than to keep it as a symbolic package.
- Do not split code and change behavior in the same step unless a regression test already covers the behavior.
- Do not add `internal/bootstrap` unless the wiring truly becomes messy.
- Prefer string-level tests for stable rendering helpers before reaching for heavy snapshot tests.

### Review questions
1. Should `internal/graph` stay as a separate package after the gitlog refactor, or should its rules move back into `internal/app`?
2. Should `internal/bootstrap` stay out of the tree for now, with `cmd/graphkeeper/main.go` remaining the only wiring layer?
3. Are package-local `_test.go` files enough, or do we want a small shared `internal/testutil` later if helpers repeat?
4. Which graph rules should be tested first: lane order, focus rules, or row rendering?

### Review note
Please review this plan before implementation starts.
Once approved, the work should happen in small commits, and every extracted package should get strict tests.
