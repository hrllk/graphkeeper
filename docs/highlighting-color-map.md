# Highlighting Color Map

This is the current inventory of color-bearing UI elements. It is organized by color so the first hardcoded recolor pass can work top-down without hunting through the codebase.

## `#00b7eb` cyan
- `sectionTitle` and `contextValue` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) color section headers and context labels.

## `#ff5ca8` pink
- `hotkey` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) colors the visible keyboard shortcuts.

## `240` border gray
- `baseBox` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) sets the inactive shell border.

## `205` magenta / alert
- `activeBox` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) sets the active shell border.
- `accent` and `warn` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) color loading, warning, and alert text.
- `reviewTarget` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) colors the target row in the merge/rebase review diagram.
- `searchFocusMark` in [internal/app/graph_search_render.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_search_render.go) uses a magenta background for the focused search hit.
- Popup borders and error/warn text in [internal/app/view_alert.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_alert.go), [internal/app/stash_popup.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/stash_popup.go), [internal/app/tagging.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/tagging.go), [internal/app/hidden_hotkeys.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/hidden_hotkeys.go), and [internal/app/cherry_pick_view.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/cherry_pick_view.go) use the same border color for modal framing.

## `226` yellow
- `branchMark` and `pointerMark` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) are the shared selection and pointer emphasis styles.
- `branchMark` is used for selected section rows in [internal/app/view_sections.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_sections.go) and for the selected row in the cherry-pick target list.
- `branchMark` is also used in [internal/app/graph_render.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_render.go) for focused branch references in the graph.
- `searchMatchMark` in [internal/app/graph_search_render.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_search_render.go) uses a yellow background for non-focused search matches.

## `118` green
- `headMark` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) marks the current HEAD/current branch.
- `headMark` is reused in [internal/app/view_sections.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_sections.go), [internal/app/view_detail.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_detail.go), and [internal/app/graph_render.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_render.go) for current-branch emphasis.
- `reviewCurrent` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) inherits the same style for the review modal.

## `42` green / success
- `ok` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) colors success and ready-state chips such as `Browse`, `Cherry-pick`, and `Reset`.
- `ok` is rendered through `renderStatusCompact` in [internal/app/view_sections.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_sections.go).

## `9` red
- `dirtyMark` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) marks dirty worktree state.
- `dirtyMark` is reused in [internal/app/view_sections.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_sections.go) and [internal/app/view_detail.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_detail.go).
- `conflictColor` and `conflictMark` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) color conflict paths, conflict markers, and conflict labels.
- `conflictColor` and `conflictMark` are used in [internal/app/graph_render.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_render.go), [internal/app/view_sections.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_sections.go), and [internal/app/view_detail.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_detail.go).

## `208` orange
- `stashMark` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) marks stash-bearing commits and stash popup rows.
- `stashMark` is used in [internal/app/graph_render.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_render.go) and [internal/app/stash_popup.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/stash_popup.go).

## `81` blue
- `remoteColor` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) marks remote provenance.
- `remoteColor` is used in [internal/app/view_sections.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_sections.go) and [internal/app/view_detail.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_detail.go).

## `#9D00FF` purple
- `tagColor` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) marks local tag provenance and tag labels.
- `tagColor` is used in [internal/app/view_sections.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_sections.go), [internal/app/view_detail.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_detail.go), and [internal/app/graph_render.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_render.go).

## `#A14743` overlap brown
- `tagOverlapColor` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) marks the stash/tag overlap case.
- `tagOverlapColor` is used in [internal/app/graph_render.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_render.go).

## `214` amber
- `reviewTarget` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) colors the target row in the review diagram.
- `reviewTarget` is rendered in [internal/app/graph_action_review.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_action_review.go).

## `245` muted gray
- `disabled` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) dims disabled hotkey descriptions.
- `reviewBase` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) colors the base row in the review diagram.

## `250` cool gray
- `reviewBranch` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) colors branch labels inside the review diagram.
- `reviewBranch` is rendered in [internal/app/graph_action_review.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_action_review.go).

## `252` bright gray
- `reviewHash` and `reviewCount` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) color the hash and count values in the review diagram.
- `descStyle` in [internal/app/view_alert.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_alert.go), [internal/app/stash_popup.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/stash_popup.go), and [internal/app/cherry_pick_view.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/cherry_pick_view.go) uses the same bright text color for modal body copy.

## `243` dim gray
- `reviewMark` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) is reserved for secondary review labels.

## `241` muted text gray
- `reviewFooter` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) colors the review modal footer.
- `muted` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) is the shared low-emphasis text color.
- `muted` appears throughout [internal/app/view_sections.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_sections.go), [internal/app/view_detail.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_detail.go), [internal/app/graph_render.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_render.go), [internal/app/stash_popup.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/stash_popup.go), [internal/app/cherry_pick_view.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/cherry_pick_view.go), [internal/app/hidden_hotkeys.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/hidden_hotkeys.go), and [internal/app/view_graph.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_graph.go).

## `240` / `245` / `241` / `243` / `250` gray family in the shell
- These are the main low-emphasis supporting tones for borders, muted copy, review labels, and secondary detail text in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go).
- If the first hardcoded recolor pass wants to simplify the palette, these are the easiest family to consolidate without changing the main information hierarchy.

## `162` / `255` handshake accent
- The graph handshake marker in [internal/app/graph_render.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_render.go) uses a magenta background with white foreground for the temporary handshake star.

## `232` search text foreground
- `searchMatchMark` and `searchFocusMark` in [internal/app/graph_search_render.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/graph_search_render.go) use dark text on top of the colored search background so the match stays readable.

## `highlight` reverse emphasis
- `highlight` in [internal/app/view_shell.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go) is the generic reverse/bold emphasis style.
- It is used for selected stash rows in [internal/app/stash_popup.go](/Users/hrk/task/sources/opensources/graphkeeper/internal/app/stash_popup.go).
