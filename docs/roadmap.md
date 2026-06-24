### TOC
- [Goal](#goal)
- [Current status](#current-status)
- [Next work order](#next-work-order)
- [Test and verification](#test-and-verification)
- [Archive rule](#archive-rule)

### Goal
This file shows the next work order for `graphkeeper`.
It keeps the next steps small and easy to follow.

### Current status
The repo already has these important pieces in place:
- a thin `cmd/graphkeeper/main.go`
- a separate `internal/graph` package for graph rules
- `internal/app` keeps getting smaller, with `model.go`, `update.go`, `commands.go`, `navigation.go`, `actions.go`, `view.go`, and `graph_render.go`
- package-local tests next to the code they cover

### Next work order
1. Keep the current baseline stable.
   - do not change behavior while the tree is being cleaned up
   - keep the current split easy to test

2. Finish any remaining `internal/app` cleanup only when it is clearly useful.
   - keep `model.go` for model state and shared types
   - move more code out only when a file becomes too dense
   - do not add new packages just for symmetry

3. Keep `internal/graph` as the home for graph rules.
   - lane order
   - row width
   - focus and pointer rules
   - row lookup helpers

4. Keep render helpers close to the app unless they become clearly reusable.
   - `internal/ui` is optional, not required
   - only create it if multiple packages need the same render pieces

5. Sync docs after code changes.
   - update `docs/structure.md` when the tree changes
   - update `README.md` only when the build or package names change

### Test and verification
Run these after each meaningful refactor:
- `go test ./...`
- `go build ./cmd/graphkeeper`

For UI or rendering changes, also check the relevant package tests in `internal/app`.

### Archive rule
Move old one-off plans and outdated implementation notes into `docs/archive/`.
Keep this file focused on the next practical steps, not long-term brainstorming.
