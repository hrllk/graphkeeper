# graphkeeper Architecture

## Goal

This document explains the code shape of `graphkeeper`.
It is a guide for structure, not a build plan.

## Simple rules

- Keep `cmd/` small.
- Put app code in `internal/`.
- Do not add `pkg/` yet.
- Keep Git work and UI work apart.
- Keep graph and lane logic pure when you can.
- Build graph rows from `git log` based data, not a DAG renderer.
- Keep the structure small and easy to read.

## Target folder map

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

## What each package does

### `cmd/graphkeeper`

- read flags and args
- load config
- call bootstrap
- start the Bubble Tea app

### `internal/bootstrap`

- connect all parts together
- make the repo, model, and UI work as one app
- leave one place for future setup work

### `internal/app`

- Bubble Tea model
- update loop
- key handling
- command dispatch
- state changes

### `internal/git`

- run Git commands
- read repo status
- collect raw branch, upstream, and commit data

### `internal/graph`

- lane order
- commit order
- focus rules
- graph row rules

### `internal/ui`

- colors and theme
- shared widgets
- render helpers

### `internal/state`

- state values
- action values
- status model

### `internal/telemetry`

- logs and diagnostics
- future observability hooks

## lazygit note

lazygit is a good reference.
It is bigger than graphkeeper, so we do not copy its large `pkg/` tree.

We do take these ideas:

- split responsibilities
- keep UI and command code apart
- keep tests and fixtures close to the code

We do not take these parts:

- a huge `pkg/` tree
- too many small packages too early
- heavy abstraction before it is needed

## Main architecture idea

- 1 binary
- 1 TUI app
- 1 thin main
- 1 bootstrap layer
- 1 Git adapter layer
- 1 graph domain layer
- 1 UI rendering layer
- no `pkg/`
- no subcommands for now

## Current state

The MVP works, but the code still mixes a few jobs in the same files.
This document sets the line so the structure stays clear as the code grows.
