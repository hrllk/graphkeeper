# graphkeeper Roadmap

## Goal

This file shows the next work order for `graphkeeper`.
It is a simple path for the next refactor steps.

## 1. Freeze the current state

Goal:
- keep the current behavior safe
- set a test baseline before bigger changes

Work:
- run `go test ./...`
- run `go build ./cmd/graphkeeper`
- check the main user flows
- keep the temporary test files in view

Done when:
- the current behavior still works
- we have a clear test baseline

## 2. Split Git and graph logic

Goal:
- separate raw Git work from graph rules

Work:
- keep raw Git calls in `internal/git`
- move lane/order/selection logic to `internal/graph`
- add pure function tests

Done when:
- graph logic is no longer tied to UI code
- Git calls and Git interpretation are separate

## 3. Split the Bubble Tea app files

Goal:
- make `model.go` smaller and easier to read

Work:
- move update logic to `update.go`
- move command code to `commands.go`
- move cursor work to `navigation.go`
- move action flow to `actions.go`
- keep `view.go` focused on rendering

Done when:
- state changes, command creation, and rendering are split

## 4. Clean up the UI layer

Goal:
- make styles and widgets easy to reuse

Work:
- add `internal/ui/theme.go`
- add `internal/ui/widgets.go`
- move shared render helpers out of view code

Done when:
- styles are not spread across many files
- render code is easier to follow

## 5. Keep docs in sync

Goal:
- make the current layout easy to read

Work:
- keep `docs/structure.md`
- keep `docs/roadmap.md`
- update README when the code layout changes

Done when:
- docs match the code layout
- the binary name and folder names make sense together

## Archive rule

Old UX plans and one-off implementation notes move to `docs/archive/`.
New docs should focus on the current structure and the next steps.

## Recommended order

1. freeze the baseline with tests
2. split git/graph
3. split app files
4. clean up UI/theme
5. update docs
6. run full build and tests
