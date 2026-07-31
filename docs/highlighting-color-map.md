# Highlighting Color Map

Graphkeeper terminal UI는 semantic token에 ANSI 기본색(0–15)과 terminal attribute를 사용한다. ANSI 색상 번호는 고정 RGB 값이 아니라 사용자의 terminal palette 슬롯이다.

## Semantic tokens

| Token | Terminal expression | Meaning | Non-color signal |
|---|---|---|---|
| `sectionTitle` | ANSI blue + bold | section/navigation heading | section position |
| `contextValue` | ANSI blue | context value | key/value label |
| `hotkey` | ANSI magenta + bold | keyboard shortcut | visible key text |
| `muted`, `disabled`, `popupHelp`, `reviewFooter` | terminal default foreground | secondary/help text | copy and placement |
| `accent`, `warn`, `stashMark` | ANSI yellow + bold | loading, warning, stash | warning copy, stash marker |
| `ok`, `headMark` | ANSI green + bold | success/current | action label, HEAD marker |
| `dirtyMark`, `conflictColor`, `conflictMark`, `errorStyle` | ANSI red + bold | dirty/error/conflict | `(dirty)`, `(conflict)`, `Blocked` |
| `remoteColor`, `reviewTarget` | ANSI blue | remote/target context | provenance or `TARGET` label |
| `tagColor` | ANSI magenta | tag/provenance | tag name, `(local)` |
| `tagOverlapColor` | ANSI magenta + bold | stash/tag overlap | overlap marker |
| `branchMark`, `pointerMark` | ANSI yellow + bold | section/graph hover | selected row or pointer |
| `highlight`, `searchFocusMark` | reverse + bold | popup/search focus | selected stash row or query text |
| `searchMatchMark` | underline + bold | ordinary search match | matched query text |
| `reviewCurrent` | `headMark` | current review row | `CURRENT` label |
| `reviewBase` | ANSI bright black + bold | base review row | `BASE` label |
| `reviewHash`, `reviewBranch`, `reviewCount` | bold/default foreground | review metadata | hash, branch, count text |
| `reviewMark` | default foreground | secondary review marker | marker text |

## Policy

- Semantic styles use `lipgloss.ANSIColor(0..15)` only; RGB and ANSI 256 colors are not used in the in-scope shell, graph, or search renderers.
- Warning uses the ANSI yellow slot. ANSI has no fixed orange slot, so the terminal palette may make yellow appear orange without the application forcing an orange RGB value.
- Error and conflict use the ANSI red slot. Warning and error remain distinguishable through visible copy and markers even when a terminal palette makes their colors similar.
- Section/graph hover uses ANSI yellow + bold foreground. Search focus and popup selection retain reverse/bold because they are focus states, not hover styling.
- Secondary/help styles use the terminal default foreground instead of ANSI bright black, which can be low contrast on white or beige backgrounds.
- Ascii, ANSI, ANSI256, TrueColor, and `NO_COLOR=1` smoke conditions must preserve visible labels and markers.

## Scope

Task 4.1 updates `internal/app/theme.go`, `view_shell.go`, `graph_render.go`, `graph_search_render.go`, and their tests. Direct color creation in `view_alert.go`, `stash_popup.go`, `tagging.go`, `cherry_pick_view.go`, and `hidden_hotkeys.go` remains a follow-up scope.
