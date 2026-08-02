# Decisions

## 2026-08-01: Main and overlay q semantics are context-sensitive

- Keep `q: quit` in the main footer and let `q` quit only from the main Browse view.
- In an open overlay, use `q: close` and route it through the same close/cancel/back
  behavior as `esc`; do not insert `q` into text inputs or execute confirmations.
- Keep only Global core navigation keys in the main footer. Active-section actions
  remain discoverable through the `?` overlay, which omits the duplicated Global group.
- Reduce the Graph section's internal `graph`/topology column by 30% of its current
  calculated width. Keep the Graph pane and right rail proportions unchanged.

## 2026-08-01: Graph is the primary full-height surface and Details heads the right rail

- Render the main body as a two-column layout: full-height `Graph` on the left and
  `Details`, `Local`, `Remote`, `Tags` stacked on the right.
- Remove the always-visible `Global` and `Context Actions` panels from the main
  body. Keep their useful commands discoverable through the section-aware `?`
  hidden-hotkey overlay.
- Keep the `Graph` topology `*` unchanged and render stash/tag state in the
  fixed state column. Use the `S·T` overlap state when both apply, including
  handshake and raw-graph rendering paths.
- Give the graph title/subject the remaining width after the five-character hash,
  branches, state, narrower `graph` topology, and restored author metadata.
- Treat Graph and the four right-rail panels as one shared outer-height grid so
  the bottom borders align after resizing. The Details panel renders the same
  projection as Context Details without rendering action commands a second
  time.

## 2026-07-11: Shell overlay precedence and shared status labels are centralized

- Keep the shell overlay order in one helper so confirm/review/reset, popups, hidden hotkeys, search, loading, and blocked states stay in a single readable stack.
- Use one shared tag provenance label renderer for both section rows and detail rows so the local/origin/unknown mapping cannot drift.
- Keep browse key handling thin by routing repeated popup-open and confirm-status setup through small helpers instead of duplicating the same state writes in multiple cases.

## 2026-07-11: Overlay families share centered titles and tag popup keeps a single title strip

- Keep overlay popups grouped as `toast`, `confirm`, and `inspect` so new surfaces fit an existing family instead of inventing a one-off frame.
- Keep popup titles centered, footer shortcut hints centered, and the body block centered on the popup axis.
- Keep dense list bodies readable, but do not let them drift off the popup center line.
- Render tag creation with the popup title strip only; do not repeat a nested title inside the body.

## 2026-07-10: Section lists and target pickers reuse Graph-style selection highlight

- Render active Local/Remote/Tags rows and target-pick popup rows through a shared helper instead of a leading `>` arrow.
- Use the existing Graph yellow emphasis style for the selected row so the focus cue reads the same across sections.

## 2026-07-10: Graph branch overflow suffix stays compact without leading padding

- Render overflow branch presence as `+N` with no leading space so the branch column reads as one compact token.
- Keep the branch field width contract unchanged so the graph layout does not shift.

## 2026-07-10: Context and Global rebalance uses a hidden drawer, not +N more

- Keep `?` as the hidden hotkey drawer entrypoint across sections.
- Keep `Graph` search on `/` and do not reuse the search popup for `?`.
- Do not introduce `+N more`; use the drawer as the overflow discovery path.
- Treat `Remote` `last fetch` / `sync status` and richer `Tags` metadata as follow-up work, not part of the first-pass layout rebalance.

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

## 2026-07-10: Tag rows use explicit provenance states and colors

- Treat Tag provenance as three visible states: `unknown`, `local`, and `origin`.
- Use muted gray for `unknown`, `#9D00FF` for `local`, and the remote accent for `origin`.
- Reuse the same provenance label and color mapping in the Tag section rows and the selected-tag detail panel.

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

## 2026-07-10: Global hotkeys own shared navigation and `?` reveals hidden actions

- Keep `scroll`, `top`, and `bottom` hotkeys in `Global` instead of repeating them in `Graph Actions`.
- Reassign `?` away from search and use it as the hidden hotkey drawer entrypoint.
- Keep `Graph` / `Local` / `Remote` / `Tags` action panels compact by default and let `?` expose the overflow.
- Do not let section help text imply that `?` is a search shortcut after the rebalance.

## 2026-07-10: Graph tag markers should stay compact and use a single overlap color on stash collision

Superseded on 2026-08-01 by the topology/status-column decision above.

- Keep tag presence in the Graph row as a compact marker, not a separate column.
- Keep commit hash styling neutral so the identifier never competes with status coloring.
- When stash and tag land on the same commit point, render a single `#A14743` overlap badge instead of split or dual markers.
- Do not use the preview SVG as the contract; keep the collision rule text-only and color-specific.

## 2026-07-06: Diverged merge/rebase review uses a dedicated status inspection modal

- Keep the final confirm dialog unchanged for merge and rebase execution.
- Render the diverged-branch review as a wider left-aligned inspection modal so it reads as state awareness, not execution.
- Use the `CURRENT hash (branch • mark)` summary format for the current, target, and base rows so hash, branch, and role are scanned in one pass.
- Keep `CURRENT`, `TARGET`, and `BASE` as the primary visual anchors and treat the graph excerpt as supporting evidence.

## 2026-07-06: Graph fast-forward confirmations use enter to execute and omit count noise

- Keep graph merge/rebase fast-forward cases on the execution path instead of the blocked-alert path so `enter` performs the action and `esc` dismisses it.
- Keep the fast-forward modal concise: title plus a single `HEAD can move to ...` sentence, without `Current` or `Target` counts.
- Leave the diverged merge/rebase review modal unchanged so the richer comparison still applies when the histories have both sides.

## 2026-07-31: Core terminal UI uses ANSI-first semantic colors

- Use `lipgloss.ANSIColor(0..15)` for the in-scope shell, graph, and search semantic styles.
- Delegate the final color to the user's terminal palette instead of forcing RGB or ANSI 256-color values.
- Use ANSI yellow for warning/loading/stash accents and ANSI red for error/conflict states. ANSI has no fixed orange slot.
- Use ANSI yellow + bold foreground for section/graph hover. Keep reverse/bold only for search focus and popup selection; preserve visible labels and markers for dirty, conflict, provenance, stash, and current/target states.
- Use the terminal default foreground for muted/help/disabled/footer text because bright black can be low contrast on white or beige backgrounds.
- Validate Ascii, ANSI, ANSI256, TrueColor, and `NO_COLOR=1` smoke conditions without changing layout, key bindings, or state transitions.
- Limit this task to `theme.go`, shell, graph, graph search, related tests, and color policy docs. Defer direct color migration in individual popup files.
## 2026-08-01: Graph topology와 stash/tag 상태 컬럼을 분리한다

- Graph topology의 `*`는 stash/tag 여부와 관계없이 항상 유지한다.
- stash/tag 상태는 branches와 topology 사이의 고정 `state` 컬럼에 `S`, `T`, `S·T`로 표시한다.
- Graph page 정보 줄 오른쪽에 `S stash · T tag` 축약 범례를 표시한다. `S·T`는
  두 상태의 조합으로 읽을 수 있으므로 범례에서 반복하지 않는다.
- 과거의 색상 기반 단일 overlap marker는 보조 표현으로만 남기고, 상태 식별은
  visible text marker를 기준으로 한다.

## 2026-08-01: Hidden Hotkeys 오버레이 색상과 Global 항목

- Hidden Hotkeys 오버레이는 개별 256색을 직접 지정하지 않고 공통 popup
  semantic style을 사용한다. ANSI profile과 `NO_COLOR` 정책이 다른 팝업과
  동일하게 적용되어야 한다.
- Global 목록에는 섹션별로 반복되는 `f/F/S` 항목을 넣지 않는다. 섹션 이동,
  이동·스크롤, 종료, 오버레이 호출처럼 공통으로 필요한 항목과 별도 이동 그룹만
  유지한다.
