# Decisions

## 2026-06-24: Adopt golangci-lint for analysis, gofumpt/goimports for formatting

- Use `gofumpt` and `goimports` as formatter tools.
- Keep `golangci-lint` focused on analysis linters such as `errcheck`, `govet`, `ineffassign`, and `staticcheck`.
- Use the module path from `go.mod` (`hrllk/graphkeeper`) for `goimports.local-prefixes`.

## 2026-06-24: Offline fallback for lint bootstrap

- `scripts/bootstrap` tries to install `golangci-lint` into `.bin/` when network access is available.
- If installation is blocked, it writes a local shim that runs `gofmt -l` and `go vet ./...` so `scripts/check` still works offline.

## 2026-06-30: Centered shell frame with 10% margins and 3:7 header split

- Keep `layoutShellMargins` at 10% horizontal and vertical margins as the default shell frame.
- Keep `Global / Context` split at 3:7.
- Center the full shell in the terminal after composing the inner frame so the layout does not drift left.
- Keep the footer aligned to the full terminal width instead of reusing the inner body padding.

## 2026-06-30: Global hotkeys live in the top panel and graph paging uses full rail height

- Remove numeric section shortcuts from the browse shell; section switching should rely on tab navigation.
- Keep `tab/shift+tab`, `up/down/j/k`, `f fetch`, and `q quit` in the top global panel so the main navigation affordances stay visible.
- Remove the redundant `Mode` and `Context` labels from the browse shell so the panes read as direct content regions.
- Size graph paging from the actual graph rail height instead of an arbitrary 76% multiplier so the graph uses the full vertical space available beside the stacked side rail.
- Keep the graph content area aligned to the graph rail's inner height so the rendered graph fills the same vertical envelope as the stacked Local / Remote / Tags rail.

## 2026-06-30: Graph layout uses shared-height grid cells, not independent boxes

- Treat the Graph area and the right rail as cells in one shared-height grid row.
- Let the parent row determine the outer height, then let each cell consume that height with its own border and padding accounting.
- Keep `Local / Remote / Tags` as stacked child cells that split the right rail height and let the last cell absorb remainder height.
- Prevent width overflow inside any graph cell so wrapping cannot break the shared-height contract.

## 2026-06-30: Popup overlays must replace covered cells, not insert after them

- Confirm and loading popups are rendered as body overlays, but the overlay layer must replace the covered region instead of inserting itself before the remaining line content.
- Popup width should be derived from the available body width and clamped so the modal cannot expand the shell width or shove the side rails sideways.
- Keep the overlay logic display-width aware so ANSI styling and wide glyphs do not shift the popup position.

## 2026-06-30: Pull no-op path shows a transient loading toast

- When `pull` finds that the current branch has nothing new to receive from upstream, do not open the merge/rebase confirm flow.
- Show a transient loading toast instead, with a message that makes the no-op state explicit, and then return to browse state.
- Keep the no-op path separate from the normal analysis/confirm flow so future pull behavior changes do not accidentally reintroduce a confirm modal for the already-synced case.

## 2026-07-01: Graph local lane and divergence are separate gates

- Treat `Graph` local-lane detection as a display and navigation concern, not as the final merge/rebase execution rule.
- Consider commits on a local branch's passed path as local for Graph highlighting and shortcut availability.
- Require `HEAD...target` divergence analysis to decide whether merge/rebase is actually meaningful.
- Do not enable Graph merge/rebase for fast-forward-only or already-contained ancestor cases.

## 2026-07-01: Local branch delete uses force delete

- Use `git branch -D <branch>` for local branch deletion so unmerged local branches can be removed without blocking on merge state.
- Keep the current-branch guard in place so the active branch still cannot be deleted accidentally.
- Keep remote deletion unchanged and continue to require an explicit `origin` target for remote deletes.
