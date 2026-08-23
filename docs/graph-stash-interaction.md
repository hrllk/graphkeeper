# Graph Stash Interaction

## Scope

This document covers the `Graph`-specific stash experience only.

It does not redefine the global design system. That stays in `DESIGN.md`.

## User Goal

The user should be able to read stash-bearing commits as recoverable points in history.

## Visual Model

- A stash belongs to the commit it was created from.
- In the graph row, stash presence should be shown as a stronger highlight on the commit pointer or branch reference, not as a separate global badge.
- The highlight should be visible by default in `Graph`, so stash-bearing commits read as recoverable points in history.
- When the row is focused, the stash summary should become visible and the state should feel actionable, not merely informational.

## Focus Behavior

When the focused graph row has one or more stashes:

- Show the stash summary in the detail panel.
- Keep the rest of the graph unchanged so the user still reads the repository topology first.
- Surface the entry point into the separate stash list popup, but keep the popup itself outside `Graph`.

## Global Entry

- The stash list is opened from a Global hotkey.
- The hotkey launches an overlay popup, not a full screen swap.
- The popup stays in the same shell so the user can inspect and act without leaving browse mode.

## Boundary

This document stops at the Graph visual layer.

The session-wide stash list UI lives in `~/.gstack/graphkeeper/20260706-0003-stash-session-list-ui-plan.md`.
The selected-stash pop flow lives in `~/.gstack/graphkeeper/20260706-0004-stash-pop-execution-plan.md`.
The `Graph`-specific stash pop entry flow lives in `~/.gstack/graphkeeper/20260709-0005-graph-stash-pop-plan.md`.
The `Graph` stash pop implementation plan lives in `~/.gstack/graphkeeper/20260709-0006-graph-stash-pop-implementation-plan.md`.
The branch continuation flow lives in `~/.gstack/graphkeeper/20260706-0005-stash-continue-from-branch-plan.md`.

## Out of Scope

- Global stash management outside `Graph`
- stash list UI
- stash apply / pop / drop / rename workflows
- stash editing
- changing how stash data is loaded from git
