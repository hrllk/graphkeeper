### TOC
- [Goal](#goal)
- [Current tree](#current-tree)
- [Responsibility map](#responsibility-map)
- [Notes](#notes)

### Goal
This file gives a quick map of the current `graphkeeper` code layout.
It is a simple reference for where the main work lives right now.

### Current tree
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
    graph_render.go
    model_test.go
  graph/
    graph.go
    graph_test.go
  git/
    repo.go
    repo_test.go
  state/
    state.go
  telemetry/
    telemetry.go
```

### Responsibility map

#### `cmd/graphkeeper`
- thin entrypoint only
- opens the repo
- creates the app model
- starts Bubble Tea

#### `internal/app`
- owns the Bubble Tea model and update flow
- keeps command creation and user actions in small files
- keeps navigation, graph focus, and graph rendering close to the app layer
- `graph_render.go` holds the render helpers that still depend on app state

#### `internal/graph`
- owns graph rules, lane order, row width, and focus helpers
- keeps graph-specific logic pure and testable

#### `internal/git`
- owns raw Git access and parsed repo state
- wraps Git commands and repo metadata

#### `internal/state`
- holds shared UI state types and statuses

#### `internal/telemetry`
- holds logging and diagnostic helpers

### Notes
- `internal/ui` does not exist yet.
- The current split keeps render helpers in `internal/app` because they still depend on app state.
- Tests live next to the code they protect, usually as `_test.go` files in the same package. This document should stay close to the actual tree, not the aspirational one.
