# graphkeeper

`graphkeeper` is a graph-based Git TUI for people who manage repositories.
It keeps branch topology, remote state, tags, and stash state visible at the same time so you can make maintenance decisions from the graph instead of guessing from command output.

This README reflects the current version in [`VERSION`](VERSION).

## Demo



https://github.com/user-attachments/assets/dbbb3cf1-e336-4046-9d3d-eb02066930bd



## TOC

- [Demo](#demo)
- [Overview](#overview)
- [Why It Exists](#why-it-exists)
- [Who It Is For](#who-it-is-for)
- [What It Helps You Do](#what-it-helps-you-do)
- [What It Is Not](#what-it-is-not)
- [Quick Start](#quick-start)
- [Working Model](#working-model)
- [Keyboard](#keyboard)
- [AI-Assisted Development](#ai-assisted-development)
- [Local Diagnostic Logs](#local-diagnostic-logs)
- [Alpha Note](#alpha-note)
- [Docs](#docs)

## Overview

`graphkeeper` is built for the maintainer view of Git.
It is for the person who needs to answer questions like:

- Where does this branch actually point?
- Is FF possible, or do I need a merge or rebase?
- Which commit should be tagged as a release?
- Is there local work to stash before switching focus?
- What is the safest next operation on this repository?

The UI is graph-first on purpose.
The commit graph is the main surface, while Current, Remote, and Tags provide supporting context.

## Why It Exists

Inspired by Owen's `HIPHOP`, `graphkeeper` asks the same kind of questions from a
maintainer's point of view: who owns the current state, where the repository is,
and what needs to happen next.

```text
who am I (develop)
when am I (Tue Jul 28)
where am I (a34bb6eb)
what am I (have to merge)

how am I (rebase or merge)
why am I (for apply)
```

The answer is the graph. It keeps the repository's shape visible while you make
the next Git decision.

## Who It Is For

This tool is for people who manage repositories rather than just contribute to them.

- release managers
- maintainers
- engineering leads who review branch state
- people who teach others how to read Git topology

If you need to explain Git history to someone else, this tool is especially useful because it keeps the graph visible while you talk through the decision.

## What It Helps You Do

- inspect the commit graph quickly
- see current branch, upstream, remotes, and tags in one place
- understand ahead, behind, and diverged states
- decide whether fast-forward is possible
- create or switch to a branch from a graph point
- tag a release point
- stash or clean local work when needed
- keep the selected graph context visible while navigating

## What It Is Not

`graphkeeper` is not a full Git cockpit.

It is not trying to replace tools like `lazygit` for file staging, diff browsing, or everyday commit authoring.
It is narrower on purpose: graph awareness, repository shape, and maintainer-style decisions.

It also does not handle conflict resolution inside the TUI.
When an operation conflicts, you finish the resolution in another Git-capable tool.

## Quick Start

Set the application version in `VERSION`, then build the binary:

```bash
./scripts/build
```

Run it:

```bash
./graphkeeper
```

Show the configured build version:

```bash
./graphkeeper --version
```

The build script reads `VERSION` and injects it into the binary. You can pass a
different output path as its first argument:

```bash
./scripts/build /tmp/graphkeeper
```

Or run it directly:

```bash
go run ./cmd/graphkeeper
```

## Working Model

The graph is the primary mental model.

- each row is a commit
- edges show ancestry
- branch labels show where refs point
- remote labels show what still lives on origin
- tag labels show release points
- stash markers show local work that has been parked

The usual maintainer flow looks like this:

1. Inspect the graph and find the current point of truth.
2. Check whether the branch is clean, ahead, behind, or diverged.
3. Decide whether to merge, rebase, reset, tag, or switch branches.
4. Keep the graph visible while you verify the choice.

## Keyboard

### Global

- `1` graph
- `2` current
- `3` remote
- `4` tags
- `tab` / `shift+tab` switch sections
- `j` / `k` or `up` / `down` move
- `enter` inspect or execute the current action
- `f` fetch repository state
- `F` fetch tags
- `?` show hidden hotkeys
- `q` quit

### Graph

- `space` checkout the selected commit or ref
- `m` merge
- `r` rebase
- `s` reset
- `d` delete branch
- `p` pull
- `P` push
- `t` tag the selected commit
- `o` pop stash at HEAD
- `H` jump to HEAD

### Current

- `space` checkout the selected branch
- `s` stash changes
- `c` clean working tree
- `d` delete branch

### Remote

- `space` checkout the selected remote branch
- `p` pull
- `d` delete remote branch

### Tags

- `enter` jump to the selected tag in the graph
- `P` push the selected tag
- `d` delete tag

## AI-Assisted Development

Product direction, requirements, prioritization, and final decisions are handled by the project maintainer. Architecture, implementation, tests, and documentation are developed with AI assistance. The maintainer reviews the changes, verifies the behavior, and is responsible for the final code.

## Local Diagnostic Logs

Graphkeeper does not send telemetry or diagnostic logs to the project author or a third-party service. When an event occurs, it may write local JSON Lines events to `graphkeeper-events.jsonl` in Go's temporary directory: usually `/tmp` on Linux and `$TMPDIR` on macOS.

Events cover repository loading, fetch and pull checks, branch and tag operations, stash loading, action previews and executions, conflicts, and errors. Each event contains a timestamp, source, event name, and optional fields such as repository path, branch, commit, action, target, count, or error text. The repository path and Git error messages may contain local path or repository details.

Graphkeeper may still contact configured Git remotes when you explicitly run Git operations such as fetch, pull, or push. This is separate from local diagnostic logging.

To remove the log on Unix-like systems:

```bash
rm -f "${TMPDIR:-/tmp}/graphkeeper-events.jsonl"
```

## Alpha Note

Read the version in `VERSION` as a concept and workflow preview, not a finished product.

This README describes the current shape of the product, not a promise of final polish.

The core graph workflow is in place, but the product is still evolving.
Expect the UI and shortcut map to keep tightening as the maintainer flow gets sharper.

What works now:

- graph navigation
- current / remote / tag inspection
- branch, merge, rebase, reset, push, pull, stash, and tag flows
- maintainer-style graph reading with ref context visible

What is intentionally out of scope:

- conflict resolution inside the app
- a full file-level Git workflow

## Docs

- `docs/structure.md` - current code map
- `docs/roadmap.md` - next work order
- `docs/highlighting-color-map.md` - UI color map
- `docs/archive/` - older plans and moved docs
