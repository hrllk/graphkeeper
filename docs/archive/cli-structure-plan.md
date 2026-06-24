# graphkeeper CLI Structure Plan

> Keep the structure small first. Add features later.

## Goal

This document explains how to shape `graphkeeper` into a clear Go CLI.
The main idea is simple:

- keep the entrypoint thin
- keep Git work and UI work apart
- keep graph logic pure when possible
- keep the package map easy to read
- build graph rows from `git log` based data, not a DAG renderer
- use lazygit as a reference, not as a copy

## Recommended package map

```text
cmd/
  graphkeeper/
    main.go

internal/
  bootstrap/
    app.go
  app/
    model.go
    update.go
    view.go
    commands.go
    navigation.go
    actions.go
  git/
    repo.go
    status.go
    graph.go
  graph/
    lane.go
    layout.go
    sort.go
  ui/
    theme.go
    widgets.go
  state/
    state.go
  telemetry/
    telemetry.go
```

## Main rules

### 1. `cmd/` only wires things together

- read flags and args
- load config
- call bootstrap
- start the app

### 2. `internal/app` is for Bubble Tea code

- model
- update loop
- command dispatch
- key handling
- state changes

### 3. `internal/git` is for raw Git access

- run Git commands
- read repo status
- collect branch, upstream, and commit data

### 4. `internal/graph` is for graph logic

- lane order
- commit order
- focus and selection rules
- graph row rules
- stable graph output

### 5. `internal/ui` is for visual code

- colors
- theme
- shared widgets
- render helpers

### 6. Do not add `pkg/` yet

This is still an MVP.
`internal/` is enough for now.

## lazygit as a reference

lazygit is a good model for a larger Go TUI.
We should borrow the good parts, but keep our structure smaller.

Borrow these ideas:

- split jobs by role
- keep UI code apart from command code
- keep tests and fixtures close to the code

Do not copy these parts:

- a very large `pkg/` tree
- too many small packages too early
- deep abstraction before it is needed

## Step plan

### Phase 0: Freeze the baseline

- run `go test ./...`
- run `go build ./cmd/graphkeeper`
- check the main flows

### Phase 1: Clean the entrypoint

- add `cmd/graphkeeper/main.go`
- add `internal/bootstrap/app.go`
- keep main as wiring only

### Phase 2: Split Git and graph code

- keep raw Git calls in `internal/git`
- move lane/order/layout code to `internal/graph`
- add tests for pure graph rules

### Phase 3: Split app files

- shrink `model.go`
- split `update.go`, `commands.go`, `navigation.go`, `actions.go`

### Phase 4: Clean the UI layer

- add `internal/ui/theme.go`
- add `internal/ui/widgets.go`
- simplify view code

### Phase 5: Keep docs in sync

- keep `docs/architecture.md`
- keep `docs/roadmap.md`
- update the README

## Document split note

This file stays as the main structure plan.
If the work grows later, we can split more detail into:

- `docs/architecture.md` for package roles
- `docs/roadmap.md` for step order and done rules
- `docs/archive/` for old UX notes

## Done when

- the binary name and docs match
- the code layout and docs match
- the core logic is testable
- UI and domain code are not mixed

## Next action

1. make the current file layout closer to this plan
2. shrink `internal/app/model.go`
3. move graph logic into `internal/graph`
4. clean the UI and bootstrap layers after that
