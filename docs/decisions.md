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

## 2026-07-01: Graph compact branch labels use representative plus overflow

- When multiple branch decorations point at the same commit, choose the compact representative in this order: `HEAD` branch first, then alphabetical local branch name, then alphabetical remote branch name.
- Show additional branch presence as `+N` only when the compact token still fits the existing 10-character branch column budget.
- Keep the Graph row compact and let the detail panel carry the full branch list when the row cannot show every name.

## 2026-07-06: Graph search is popup-only and repeats with n/N

- Keep graph search scoped to the `Graph` section only.
- Use `/` to open a transient search popup, `enter` to jump to the first match, and `n` / `N` to repeat the last confirmed search.
- Keep branch creation on `n` when no graph search query is active.
- Do not add a result list; keep the popup focused on query entry, match count, and no-match feedback.

## 2026-07-06: Graph stash visualization should read as a pinned moment in the graph

- Treat stash as graph metadata, not a separate global concept.
- Use a strong stash highlight in the Graph row so stash-bearing commits read as recoverable points in history.
- Keep the focused commit's stash summary in the detail panel so the Graph surface stays visual-first.
- Keep Graph stash behavior separate from the session stash list UI, pop execution, and branch continuation docs.
- The detailed Graph stash flow lives in `docs/graph-stash-interaction.md` so the system design stays separate from the feature contract.

## 2026-07-08: Stash list opens from Global hotkey into an overlay popup

- Keep `Graph` limited to stash presence and focus summaries.
- Open the stash list from a Global hotkey so the entry point is always available.
- Render the stash list as an overlay popup in the same shell instead of a full screen swap.
- Keep `0003` as the overlay list UI contract and `0004` as the pop execution contract.

## 2026-07-09: Stash popup uses a flat list with 7-char hash tokens

- Render stash list rows as flat entries instead of grouping by `BaseHash`.
- Show only the first 7 characters of the stash base hash in the popup row text.
- Omit the `base:` prefix from the visible label so the hash, ref, and subject scan faster.
- Keep `BaseHash` internally for Graph jump behavior, but do not expose it in the popup label.

## 2026-07-09: Graph tag work splits read/list and create flows

- Keep tag inspection and tag creation in separate plans.
- Treat `Tags` as a read-first inspector for grouped tag data.
- Keep `Graph` tag creation as a focused CUD flow with popup input and repo refresh.
- Do not mix tag list rendering concerns into the create flow contract.

## 2026-07-09: Tag provenance is an app-managed snapshot over local refs

- Read local tags from local refs on startup so the Tag section can render immediately without waiting for remote provenance.
- Keep remote provenance in `.git/graphkeeper/tag-provenance.json` as app-owned metadata instead of trying to extend Git refs.
- Use `F` as the explicit refresh path for `git fetch --tags` plus `git ls-remote --tags origin`, then persist the resulting provenance snapshot.
- Treat `never synced` and `synced` as the only user-facing sync summary states.
- Do not use `(no-up)` when provenance has not been loaded yet; unknown provenance must remain visually distinct from missing remote provenance.
- Use `unknown` for new local tags until provenance is explicitly known.

## 2026-07-09: Tag push is explicit and tag fetch does not overwrite conflicts

- Use `t` for local tag creation only.
- Use `P` in the Tags section for explicit tag push.
- Keep `F` focused on provenance sync and remote tag refresh.
- Use `d` for local tag delete and `D` for remote tag delete.
- If `F` hits a tag-name or tag-content conflict, fail without overwriting the existing local tag ref.

## 2026-07-09: Graph stash pop is HEAD-gated and uses a two-step overlay

- Keep Graph stash pop available only when the focused `Graph` row is `HEAD` and that commit has at least one stash.
- Use `o` as the Graph pop hotkey because the existing Graph section already uses `p` and `P`.
- When multiple stashes exist, open a picker first and then a confirm overlay for the selected stash.
- When only one stash exists, skip the picker and open the confirm overlay directly.
- Keep the Graph pop overlay separate from the global stash list popup so read/browse and execution remain distinct.

## 2026-07-06: Diverged merge/rebase review uses a dedicated status inspection modal

- Keep the final confirm dialog unchanged for merge and rebase execution.
- Render the diverged-branch review as a wider left-aligned inspection modal so it reads as state awareness, not execution.
- Use the `CURRENT hash (branch • mark)` summary format for the current, target, and base rows so hash, branch, and role are scanned in one pass.
- Keep `CURRENT`, `TARGET`, and `BASE` as the primary visual anchors and treat the graph excerpt as supporting evidence.

## 2026-07-06: Graph fast-forward confirmations use enter to execute and omit count noise

- Keep graph merge/rebase fast-forward cases on the execution path instead of the blocked-alert path so `enter` performs the action and `esc` dismisses it.
- Keep the fast-forward modal concise: title plus a single `HEAD can move to ...` sentence, without `Current` or `Target` counts.
- Leave the diverged merge/rebase review modal unchanged so the richer comparison still applies when the histories have both sides.
