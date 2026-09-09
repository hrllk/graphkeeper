# Decisions

## 2026-09-10: The graph row budgets its columns, and the date column returns where it fits

Reverses two earlier decisions, in part. Recorded here because the code and the
decision log would otherwise disagree.

- 2026-08-01 said the graph title takes "the remaining width after the five-character
  hash, branches, state, narrower `graph` topology, and restored author metadata".
  The title still takes the remainder; the list it comes after now includes a date.
- 2026-07-10 said to keep state "as a compact marker, not a separate column". That
  still holds for stash and tag markers. The date is not a marker - there is no
  glyph that carries a date - so it takes a column or it is not shown.

What changed to allow it: the state and topology columns were budgeted for their
worst case and spent 14 columns on nothing in a linear graph with no tags. Sizing
them from what the graph actually contains moved the title budget at an 80-column
terminal from -2 to +14, which is what made a date column arithmetically possible
at all.

- Size the state and topology columns from the whole graph, not the visible window.
  A window-adaptive budget reclaims a few more columns but reflows the layout while
  the user scrolls.
- Drop the state column, and the `S stash · T tag` legend with it, when nothing in
  the graph carries a stash or a tag.
- Show the `yymmdd` date column only when the title keeps at least
  `graphTitleMinimumWidth` after it. That is from 100 columns up; below that the
  Details panel carries the date instead.
- Leave the branches column alone. Its widest decoration measures exactly its
  14-column budget, so shrinking it truncates content rather than reclaiming padding.

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
## 2026-08-02: Commit Inspector는 독립 read-only screen으로 확장한다

- Graph에서 `enter`하면 기존 Graph shell을 가리는 독립 Inspector screen state로
  전환한다. 기존 `overlayPopup`은 다른 popup 계약에 남긴다.
- Inspector는 Changed files tree와 `from`/`to` side-by-side diff를 구조화된
  hunk/line-pair contract에서 렌더링한다.
- diff와 Tree-sitter 입력은 untrusted data로 취급하며 byte/line limit, stale
  epoch 폐기, plain fallback, `NO_COLOR`에서도 보이는 A/M/D·+/- marker를 유지한다.
- 실제 Tree-sitter grammar/query와 merge parent 선택 UI는 후속 subtask로
  분리한다. MVP는 highlighter registry contract와 first-parent 표시까지만 둔다.

## 2026-08-03: Commit Inspector 구현 범위를 vertical slice로 고정한다

- 첫 구현 범위는 독립 bordered screen, Changed files tree, first-parent
  from/to pairing, app-owned reader contract, request identity/cancellation,
  bounded streaming/cache, visible tree projection으로 고정한다.
- Tree-sitter grammar/query bundle과 merge parent selector/combined diff는
  현재 subtask의 TODO로 남긴다. 이 둘을 MVP 완료 조건으로 암묵적으로 포함하지 않는다.
- Graph shell은 Inspector 진입 중 가려지고, 기존 popup overlay stack에는
  Inspector를 중복 등록하지 않는다.

## 2026-08-03: Commit Inspector 디자인 계약을 단순한 terminal 흐름으로 고정한다

- 저높이 화면은 commit identity·selected path·FROM/TO·최소 diff·복귀 키를
  우선하고, 전체 message는 `m`으로 여는 secondary viewport로 둔다.
- empty/binary/submodule은 오류와 구분되는 contextual empty state로 표시하며,
  footer는 현재 상태에 필요한 핵심 hotkey만 노출한다. 전체 legend는 `?` 도움말로
  이동한다.
- narrow mode는 별도 toggle key 없이 `Tab/Shift+Tab`으로 files/from-to/message를
  순환한다. 색상 없이도 marker·label·border·FROM/TO가 의미를 보존한다.

## 2026-08-06: Commit Inspector MVP는 unified diff와 최소 keymap으로 구현한다

- MVP 화면은 Graph를 완전히 대체하는 공통 bordered screen과 `commit/message/author`
  3행 고정 header를 사용한다. footer에는 `q`, `Esc`, `?`만 둔다.
- Changed files는 flat 경로 목록이 아닌 directory/file tree projection으로 제공하고,
  diff는 adapter가 만든 structured paired rows를 renderer가 직접 출력한다.
- metadata와 selected-file diff는 독립 state/request/cancel lifecycle을 가지며,
  parent는 한 번 resolve한 snapshot을 changed-files와 diff가 공유한다.
- metadata per-file Git N+1 호출은 금지하고 batch 조회한다. bounded output cap에
  도달하면 child process를 종료한 뒤 partial 결과를 반환한다.
- `Tab`, `m`, `r`, `j/k`, message footer와 full message viewport는 이 MVP에서 제거한다.
  Tree-sitter grammar/query, word-level diff, merge parent selector는 후속 TODO다.

## 2026-08-06: Commit Inspector는 파일 선택 중심으로 단순화한다

- Header는 `commit`, `message`, `author`, 선택 파일 `path`의 4행으로 고정한다.
- Changed files는 fold/unfold 없이 항상 표시하고, 파일명은 `../../filename` 형태로
  축약한다. 선택 파일의 전체 경로는 header에서 확인한다.
- 별도 pane focus 없이 files와 diff를 동시에 렌더링한다. `j/k`는 파일 선택,
  `Ctrl+U/D`는 선택 파일 diff scroll만 담당하며 `h/l`, Inspector 내부 `Enter`는
  사용하지 않는다.
- Diff의 context/일반 코드는 기본 foreground를 사용해 과도하게 어두워지지 않게
  하고, added/removed semantic color는 유지한다.

## 2026-09-10 — 축약 마커는 하나다 (task 6.6, D-007)

앱에는 축약 방식이 네 가지 있었다. `fitVisibleWidth`(마커 없이 하드컷),
`shorten`(마커 없이 **바이트** 슬라이스), `truncateInspector`(`…`), 그리고
네 곳에 흩어진 리터럴 `"..."`. 같은 화면 안에서 세 가지가 동시에 보였다.

**결정.** 마커는 `…` 하나다(`helpers.go`의 `ellipsis`). 넘칠 수 있는 텍스트는
전부 `truncateText`를 통과한다. `shorten`은 **커밋 해시 축약 전용**으로 좁힌다.
7자 해시가 더 긴 것의 앞부분이라는 건 독자가 이미 아는 관습이므로 마커가 필요
없지만, 브랜치 이름을 말없이 자르면 짧아진 이름이 아니라 *다른* 이름으로 읽힌다.

**같이 고친 것:**

- `shorten`이 바이트를 잘랐다. `shorten("한글제목입니다", 8)`는
  `"한글\xec\xa0"`을 돌려줬다 — 깨진 UTF-8. 이제 rune을 센다.
- `fitBranchField`가 `"..."` 몫으로 3칸을 예약했다. 마커가 1칸이 되면서 필드가
  2칸 짧게 렌더됐다. 예약량을 마커 폭에서 파생시킨다.
- `compactTagTitleText`가 필드는 10칸인데 7자에서 잘랐다. 마커가 1칸이므로
  이제 9자가 들어간다. 패딩도 `len`(바이트)에서 `padRight`(폭)로 바꿨다.
- 팝업의 세로 "더 있음" 표시도 같은 글리프를 쓴다.

DESIGN.md "Truncation uses a visible marker"의 구현이다.

## 2026-09-10 — `•` 는 줄을 시작하고 `·` 는 줄 안에서 잇는다 (task 6.6, D-008)

두 글리프가 네 가지 일을 하고 있었다. `•` 는 목록 불릿이면서 동시에 인라인
구분자였고(`"y: yes  •  n: no"`), `·` 는 인라인 구분자이면서 산문의 필드
구분자였다(`"branch: main • head: abc"`). 같은 footer 가 두 철자로 존재했다 —
`confirmation_projection.go` 는 불릿으로, `view_shell.go` 는 미들닷으로 같은
`n: close / esc: close` 를 만들었다. 이게 드리프트를 잡아낸 지점이다.

**결정.** 위치가 규칙이다.

| 글리프 | 자리 | 뜻 |
|---|---|---|
| `•` | 줄 머리 (들여쓰기 뒤 허용) | 목록 항목 하나 |
| `·` | 줄 안 | 같은 줄의 대등한 것들을 잇는다 |

독자가 **어디 있는지**로 둘을 구별한다. 사전을 외울 필요가 없다.
`  •  `(양쪽 두 칸) 인라인 형태와 공백으로 열 맞춘
`"• y: continue                    • n: cancel"` 형태는 사라졌다.

**계약 테스트.** `vocabulary_test.go` 가 `internal/app` 의 문자열 리터럴을
go/parser 로 걸어 규칙을 강제한다. 규칙이 조용히 되돌아갈 수 없다.

**테스트가 찾아낸 구분:** `"..."` 금지 규칙을 처음엔 너무 넓게 썼더니
`"Deleting branch..."` 40여 곳이 걸렸다. 그건 잘림 마커가 아니라 **진행 중**
표시이고 — DESIGN.md 가 허용하는 유일한 종류의 motion — 문장으로 읽힌다.
`"HEAD...@{upstream}"` 은 git range 문법이다. 규칙은 **잘라낸 자리**에만
적용된다. 테스트는 리터럴이 정확히 `"..."` 일 때만 잡는다.

## 2026-09-10 — 선은 두 어휘로 그리고, 서로의 일을 하지 않는다 (task 6.6, D-011)

**측정.** 박스 테두리는 `RoundedBorder`(`╭─│╰`)이고 내부 구분선도 `─`/`│`로
직접 그린다 — 여기까지는 일관된다. 그래프 레인은 `* | / \`, ASCII다.

**이 혼재는 의도다.** `*`와 `|`는 git 자신의 topology 표기이고 tig·lazygit
사용자가 그대로 읽는다. `╱`/`╲`로 바꾸면 예뻐지지만 터미널 폰트에서 고정폭이
보장되지 않는다. 프레임은 유니코드, 히스토리는 ASCII — 둘 다 남긴다.

**규칙은 "서로의 일을 하지 않는다"이고, 여기서 깨져 있었다.** 상태줄이
`"Browse | msg"`처럼 ASCII 파이프를 **필드 구분자**로 썼다. 진짜 `│` pane
split 과 진짜 그래프 레인 바로 옆에서, 같은 "한 줄 안에서 잇는다"를 세 번째
문자로 말한 것이다. `preview.go`는 `"  |  target: "`으로 네 번째 철자까지
갖고 있었다. 전부 D-008의 `·`로 접었다.

**`------` 날짜 placeholder 는 그대로 둔다.** 규칙선이 아니라 고정폭 데이터
컬럼의 "값 없음"이고, 한 칸짜리 `-`로 줄이면 컬럼이 시각적으로 사라진다.
`graphDateText`의 주석이 이미 그 근거를 갖고 있다.

**테스트가 규칙을 좁혀줬다.** `·`로 시작하는 리터럴을 전부 위반으로 잡았더니
`" · target: "` 같은 **연결 조각**이 걸렸다. 앞의 공백이 곧 "왼쪽에 붙는다"는
신호다 — 줄 머리 들여쓰기는 언제나 불릿을 달지 join 을 달지 않는다.

## 2026-09-10 — 팝업은 담긴 것만큼 넓고, 한 기준선을 쓴다 (task 6.7, D-009/D-010)

2026-08-28 감사 수치는 다시 쟀다. Hidden Hotkeys 의 "가장 긴 줄 24자"는 낡았고
(지금 36자), Create tag 는 56컬럼이 맞았다.

### D-009 — 편집 가능한 필드가 읽기 전용 컨텍스트와 같아 보였다

Create tag 는 `target: 4d8fcbcc`(읽기 전용)와 `name: v1.2.0`(편집 가능)를
**똑같이** 그렸다. 게다가 박스가 `Align(Center)`라 타이핑하면 값이 가운데에서
양쪽으로 자라며 **자기 라벨을 왼쪽으로 밀었다**.

세 가지가 다 필요하다: 라벨을 한 폭으로 패딩해 값이 같은 열에서 시작하고,
편집 필드는 전 구간에 밑줄을 그어 비어 있어도 보이며 값이 자라도 움직이지 않고,
캐럿은 reverse 블록으로 찍는다. 밑줄과 reverse 는 색이 아니라 attribute 이므로
NO_COLOR 에서 살아남는다 — `cursorSignal` 과 같은 근거. 그래프가 road 마크에
인라인으로 쓰던 `\x1b[4m` 을 `underlineSignal` 로 이름 붙여 한 정의로 모았다.

### D-010 — 팝업 폭이 콘텐츠가 아니라 터미널에서 나왔다

`popupWidthForBody` 는 body 폭만 보므로 3줄짜리 폼이 자리만 있으면 56컬럼을
차지했다. `popupWidthForContent` 를 추가했다. 터미널은 여전히 상한과 하한을
정하고, 콘텐츠는 그 사이 어디에 앉을지만 정한다. Hidden Hotkeys 는 52 → 44,
Create tag 는 58 → 38.

Hidden Hotkeys 는 **닭과 달걀**이 있다. 콘텐츠는 만들어야 재고, 만들 때 폭에
맞춰진다. 상한에서 한 번 만들고, 재고, 다시 만든다 — 좁히는 것은 줄을 길게
하지 않으므로 두 번째 패스에서 수렴한다. 측정은 스크롤 창이 아니라 **전체 줄**을
본다. 6.4 의 컬럼 예산과 같은 이유로, 스크롤 중에 박스가 리사이즈되면 안 된다.

### 정렬 규칙

혼재가 문제였지 정렬 자체가 아니다. **body 가 목록이면 팝업 전체가 왼쪽 정렬,
body 가 메시지 하나면 전체가 가운데 정렬.** 폼은 필드의 목록이므로 왼쪽이다.
확인·경고 팝업은 가운데로 남는다.

### 테스트가 찾아낸 것 둘

1. **가운데 정렬이 가운데가 아니었다.** `renderCenteredPopupLine` 이 팝업 폭을
   받았는데 박스는 그중 4칸을 패딩에 쓴다. 줄이 4칸 넓게 만들어져 뒤쪽 공백이
   잘렸고, `esc: close` 가 왼쪽에서 22칸 오른쪽에서 18칸에 앉았다. 왼쪽 기준선
   결정이 이 함수를 통째로 없앴다.
2. **팝업이 20컬럼에서 터미널을 2칸 넘쳤다.** `popupWidthForBody` 의 클램프가
   테두리 2칸을 빼먹었다 — 10.10 이 쉘 프레임에 대해 고친 것과 같은 결함이,
   팝업 경로에 남아 있었다. 약 34컬럼 아래 **모든** 팝업이 해당됐다.
   `formFieldMinWidth` 도 같은 종류였다 — 빈 필드를 보이게 하는 최소값이
   박스보다 우선해 lipgloss 가 상자를 한 칸 더 키웠다. 최소값은 가용폭을
   넘을 수 없다.

### 세 폼 팝업 전부에 적용

Create tag 하나만 고치면 이 저장소가 반복해온 패턴 — 한 결정, 두 표면, 한쪽만
수정 — 을 또 만든다. Create branch 와 Stash message 도 같은 결함을
갖고 있었고(`name:`/`base:` 가 똑같이 그려졌고, 빈 초안을 `" "` 로 패딩했다 —
보이지 않는 필드가 강요하는 우회다), 같은 처리를 받았다.
`TestEveryFormPopupMarksItsEditableField` 가 셋을 한꺼번에 붙든다.

## 2026-09-10 — Inspector 헤더: 해시는 통째로 아니면 짧게 (task 6.8, D-016/D-018)

**재측정이 감사보다 나빴다.** 감사는 40컬럼에서 해시가 잘린다고 적었는데,
**80컬럼에서도** `FROM 26d88da2e637d357253d8f67fec873a1b2…` 로 잘렸다. author
행 꼬리에 40자 해시를 얹으면 어떤 폭에서도 들어가지 않는다.

**두 렌더 경로가 자기 헤더에 대해 이미 서로 달랐다.** screen 은
`COMMIT <40자>` 에 이메일과 FROM 꼬리, popup 은 `commit: <hash>` 에 둘 다 없음.
`inspector_header.go` 가 카피 결정을 소유하고 두 경로가 각자 fitting 만 한다.

**결정 셋:**

- **해시는 통째로 아니면 짧게, 절대 잘라서 쓰지 않는다.**
  `4d8fcbcc16d09a9d81b2c3e4f50…` 는 identity 로 읽히지만 identity 가 아니다.
  확인에도 못 쓰고 어디 타이핑할 수도 없다. `screenStampText` 와 같은 계단 모양:
  전체 / 12자 / 7자 / 생략.
- **이메일은 확인용 부가정보다.** 이름이 저자를 지목하므로, 행이 둘 다 담지
  못하면 이메일이 빠진다. 잘린 도메인은 짧아진 주소가 아니라 **틀린 주소**다.
- **parent 는 자기 행을 갖는다.** 꼬리일 때는 짝이 되는 TO 가 없었고 가장 먼저
  잘렸다. root 커밋은 `parent: (root commit)` 로 말한다 — 행이 왜 없는지
  독자가 추측하지 않게.
- `COMMIT` 대문자 표제를 `commit:` 으로 통일. popup 경로가 이미 그랬다.

**우선순위 사다리.** 한 boolean(`withDates`) 대신 행마다 우선순위를 준다.
버리는 순서는 committed → date → parent 이고 거기서 **멈춘다**. author·path·
message·identity 를 버려 diff 행을 사는 것은, Inspector 가 어느 커밋을 보여주는지
말하지 못하게 만드는 대가다. 사다리는 diff 가 한 행을 얻는 즉시 멈추므로,
높이 11 에서는 date 만 빠지고 parent 는 남는다 — 이전의 all-or-none 은 프레임이
감당할 수 있는 행을 버리고 있었다.

**두 결함이 딸려 나왔다.**

1. **`renderCommitInspectorScreen` 이 크기를 두 곳에서 받았다.** 인자로도 받고
   `m.width` 에서도 읽었다(`inspectorBodyRowCount` 경로). 둘이 어긋나면 스크롤
   클램프와 렌더러가 행 수를 다르게 세고 **마지막 diff 줄에 닿을 수 없다** —
   기록된 T8 회귀와 같은 부류다. 헤더 행 수가 폭에 의존하게 되면서 드러났다.
   인자를 없앴다. 어긋남을 쓸 수 없게.
2. **`"M"` assertion 이 줄곧 무의미했다.** 수정 파일 마커를 검사한다고 적혀
   있었지만 프레임 안의 유일한 M 은 **"FROM" 의 M** 이었다. 꼬리가 사라지자
   드러났다. 이 픽스처의 파일 상태는 지금도 `?` 로 렌더된다 — 별도 과제.

## 2026-09-10 — Hidden Hotkeys 는 섹션당 평평한 목록이다 (task 6.8, D-014)

**라벨 넷이 전부 내부 어휘였다.** `Visible:`/`Conditional:`, 그리고 Global 의
`Common:`/`Moved out:`. "무엇에 대한 conditional 인가"에 화면이 답하지 않았고,
"moved out" 은 키가 푸터에서 옮겨졌다는 앱의 **연혁**이지 독자의 관심사가 아니다.

**앱이 이미 더 잘 답한다.** 조건이 안 맞는 키를 누르면 이유가 나온다
(`o` → "No stash available. Add a stash at HEAD first."). 그 순간의 답이
카테고리보다 쓸모 있다. 따라서 분류는 그룹당 한 행을 쓰는 장식이었고,
DESIGN.md 의 "Decoration level: Minimal" 에 어긋난다. 없앴다.

**불릿은 남긴다.** D-008 이 방금 세운 앱 공용 목록 마커이고, 여기서 아낀 행은
그룹 제목에서 나왔지 불릿에서 나온 게 아니다.

**조사가 진짜 결함을 찾았다.** Graph 목록이 `p: pull` 과 `a: abort` 를 광고하는데
**Graph 포커스에서 둘 다 아무 핸들러에도 닿지 않는다.** `handleBrowseKey` 는
Graph 를 `handleBrowseGraphKey` 로 보내고 거기엔 두 키의 case 가 없으며,
그 키들을 소유한 `handleBrowseSectionKey` 로의 fall-through 가 없다. 목록에서
뺐다. 나머지 섹션은 핸들러와 대조해 전부 일치했다.

팝업은 52 → 40컬럼이 됐다.

## 2026-09-10 — 진행 표시는 점을 돌린다, 그 이상은 하지 않는다 (task 1.27)

정적인 `Fetching upstream...` 는 멈춘 것과 구별되지 않는다. 뒤의 점을 돌리면
아직 일하는 중이라고 말할 수 있고, DESIGN.md 가 터미널에 허용하는 motion 은
정확히 거기까지다.

**타이머를 켜는 자리는 한 곳이다.** loading 상태를 대입하는 호출부가 20곳이
넘는다. 각자 타이머를 기억하게 하는 대신 `Update` 를 감싸서, 안쪽 update 가
끝난 뒤 한 번만 판단한다. 은퇴 지점도 같은 곳이라, 끝난 작업의 tick 이 다음
작업을 애니메이션하는 일이 생기지 않는다.

**loading "상태"가 아니라 loading "진입"에서 켠다.** 모드가 loading 이기만 하면
켜도록 하면, 작업 중에 도착하는 모든 메시지가 명령을 뱉는다 — pull 라이프사이클이
stale·mismatch 메시지에 대해 **정확한 no-op** 을 요구하는데 그게 깨진다.
테스트가 이걸 잡아줬다(28개 실패 중 9개).

**점 하나는 문장이고 셋은 진행이다.** 처음엔 뒤의 `.` 를 하나라도 잘라내
애니메이션 대상으로 봤더니 `"Enter a branch name."` 이 걸렸다. loading 모드는
작업이 아닌 프롬프트도 나른다. D-007 에서 그은 것과 같은 구분이다.

**폭은 변하지 않는다.** 점을 가장 넓은 프레임에 맞춰 공백으로 채운다. loading
팝업은 가운데 정렬이라, 늘었다 줄었다 하면 상자가 초당 세 번 흔들린다.
6.7 에서 입력 필드에 적용한 것과 같은 원칙.

**프레임 0 이 쓰인 그대로다.** 사이클은 `...` 에서 시작해 `.` `..` `...` 로 돈다.
Update 를 한 번도 거치지 않고 렌더러에 닿는 모델이 있고, 거친 모델의 첫 페인트도
코드가 쓴 문자열과 같아야 한다. 점 하나로 열면 꼬리가 잘린 메시지로 읽힌다.

## 2026-09-10 — 릴리스는 태그가 켜고, Linux 는 실제로 실행해 본다 (task 5.3)

이 저장소에는 CI 가 없었다. `.github/workflows/` 자체가 없었고, Linux 는
cross-build 만 됐지 **한 번도 실행된 적이 없다.** 컴파일된다는 것과 시작된다는
것은 다른 주장이다.

**`scripts/verify-linux-binary`.** 다섯 가지를 헤드리스로 확인한다 — `--version`
이 빌드에 박힌 버전을 말하는가, `--help` 가 깨끗이 끝나는가, 모르는 플래그를
crash 없이 거절하는가(exit 2), 저장소 밖에서 설명하고 끝나는가, 터미널 없이
저장소 안에서 깨끗이 실패하는가. 종료 코드만 보지 않고 출력에서 panic·goroutine
덤프도 찾는다 — crash 와 정중한 거절이 같은 종료 코드를 공유할 수 있다.

CI YAML 이 아니라 스크립트로 둔 것은 이 저장소의 관습이고(`scripts/check` 등),
로컬에서도 돌릴 수 있어야 하기 때문이다. `scripts/verify-linux-in-container` 가
macOS 에서 컨테이너로 같은 검증을 돌린다. **debian stable-slim amd64/arm64 양쪽에서
실제로 통과하는 것을 확인했다.**

**두 워크플로.** `ci.yml` 은 push/PR 에서 `scripts/check` 와 Linux 실행 검증을
돌린다 — CI 전용 복사본이 아니라 로컬과 같은 스크립트라 둘이 갈라지지 않는다.
`release.yml` 은 **`v*` 태그를 사람이 밀어야만** 돈다. 자동 릴리스는 없다.

**태그와 VERSION 이 어긋나면 멈춘다.** `scripts/build` 는 VERSION 을 바이너리에
박고 릴리스 이름은 태그가 정한다. 둘이 다르면 사용자가 받은 바이너리가 자기가
어느 릴리스인지 틀리게 말한다. 태그는 VERSION 으로 **끝나야** 한다.
지금까지 모든 태그가 `v0.1.0-alpha.6` / VERSION `alpha.6` 관계다 — CHANGELOG
제목과 DESIGN.md 는 짧은 형태를, 태그는 semver 접두사를 쓴다. 이 검사는 두 관습이
이미 갖고 있는 관계를 강제할 뿐, 어느 쪽으로 통일할지는 정하지 않는다.
**통일 여부는 사용자 결정으로 남긴다.**

**태그를 밀지 않았다.** 릴리스는 바깥으로 나가는 행위이므로 워크플로만 넣는다.

## 2026-09-10 — popup Inspector 렌더러를 지운다 (task 12.1)

`renderCommitInspectorPopup` 은 프로덕션 호출부가 없었다. `view_shell.go` 는
`renderCommitInspectorScreen` 만 부르고, popup 은 자기 테스트 두 개로만 살아
있었다. `renderRawGraphLine` 때와 같은 상황이고 같은 판단을 한다.

**지우기 전에 커버리지부터 옮겼다.** popup 테스트 둘 중
`TestCommitInspectorDiffDoesNotWrapLongCode` 는 **screen 경로에 대응물이 없었다.**
긴 diff 줄이 접히면 두 pane 의 정렬이 깨지고 독자가 보는 diff 행 수가 조용히
달라진다. 실제로 프로덕션에서 도는 렌더러에 대해 40/60/80 폭으로 다시 썼다.
나머지 하나(프레임 치수·헤더·트리·푸터)는 screen 쪽에 이미 대응물이 있다.

**곁다리 정정.** task 12.2 는 트리가 상태 글리프를 `?` 로 그리는 원인을
`inspectorStatus` 로 지목했는데, `inspectorStatus` 는 **삭제 전에도 이미 unused**
였다. 즉 popup 트리는 그 함수를 거치지 않았고 진단이 틀렸다. popup 이 사라졌으므로
항목 자체가 없어졌지만, 틀린 진단을 기록에 남겨두지 않는다.

삭제로 새로 죽은 것은 `commitInspectorSelectedPath` 하나뿐이라 같이 지웠다.
`inspectorStatus`·`screenPath` 등은 2026-08-28 lint 베이스라인이 이미 안고 있던
빚이고, 이 작업의 범위가 아니다.
