# graphkeeper Structure

## Goal

This document gives a quick view of the current code structure.
It shows where the main work lives and what each part does.

## Current tree

```text
cmd/
  graphkeeper/
    main.go

internal/
  app/
    model.go
    view.go
    model_test.go
    zz_temp_graph_bench_test.go
  git/
    repo.go
    repo_test.go
    zz_temp_status_timing_test.go
  state/
    state.go
  telemetry/
    telemetry.go
```

## What each part does

### `cmd/graphkeeper`

- starts the app
- opens the Git repo
- creates the Bubble Tea model
- runs the TUI program

### `internal/app`

- owns the Bubble Tea model
- handles update flow
- renders the screen
- keeps app state and navigation together for now

### `internal/git`

- reads repo data
- runs Git commands
- collects branch and status info
- provides data to the app layer

### `internal/state`

- stores shared state types
- keeps app status values in one place

### `internal/telemetry`

- writes diagnostics
- keeps logging helpers together

## Notes

- The current code is still small and simple.
- Some graph rules still live inside `internal/app` and `internal/git`.
- Later, those graph rules can move into a separate package if the code grows.

## What this document is for

Use this file when you want to quickly understand where things live.
It is not a design plan.
It is a map of the current code.
