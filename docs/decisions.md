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
