# graphkeeper

## Overview

`graphkeeper` is a graph-first Git TUI for people who want to read repo history fast.
It helps you see branches, upstreams, and commit shape without losing the big picture.

This is still an MVP.
The tool is small on purpose, and the structure is still being cleaned up.

### Demo
---
<img width="400" height="279" alt="Screen Recording 2026-06-24 at 13 36 32" src="https://github.com/user-attachments/assets/a84d1926-9bcc-46df-b0af-a3d760adce1e" />

### TOC
---

- [Overview](#overview)
- [What It Is](#what-it-is)
- [What It Is Not](#what-it-is-not)
- [Why It Exists](#why-it-exists)
- [Quickstart](#quickstart)
- [Neovim Entry Point](#neovim-entry-point)
- [Graph Mental Model](#graph-mental-model)
- [MVP Scope](#mvp-scope)
- [Keyboard](#keyboard)
- [Status](#status)

### What It Is
---

`graphkeeper` is made for people who need to read Git history as a graph.

- read the commit graph quickly
- see where local branches and remotes point
- understand ahead, behind, and diverged states
- choose graph-based actions from the graph itself
- keep the current branch context visible while working

### What It Is Not
---

`graphkeeper` is not a lazygit replacement.

`lazygit` is a broad Git cockpit.
It is great for staging, committing, diffs, stashing, file-level work, and daily Git tasks.

`graphkeeper` is narrower on purpose.
It focuses on graph awareness, branch topology, and maintainer-style work.
Think of it as a map for the repository, not the whole cockpit.

### Why It Exists
---

Git history is topology.
If you only read plain command output, it is easy to miss the shape of the repo and make a bad move.

`graphkeeper` exists to answer questions like:

- Where is this branch actually pointing?
- Which local branch should I operate on?
- What does the upstream know that my local branch does not?
- Is this commit safe to reset to?
- Should I merge, rebase, or leave this branch alone?

### Quickstart
---

Build the binary:

```bash
go build ./cmd/graphkeeper
```

This creates a `graphkeeper` binary in the current folder.

Run it:

```bash
./graphkeeper
```

### Neovim Entry Point
---

Neovim plugin support will be added later.

### Graph Mental Model
---

The graph is not just a list of commits.

- each node is a commit
- edges show ancestry
- branch labels show where refs point
- upstream context shows what is local and what still lives remotely

The current graph strategy is based on `git log` data.
`internal/graph` owns lane order, commit order, focus rules, and graph row rules.

Once you read the repository this way, operations become easier to reason about.
You stop guessing which ref is safe to move.

### MVP Scope
---

The current implementation is intentionally limited.

- the graph is local-branch focused
- remote-only branches do not add extra lanes to the main graph
- the UI favors branch topology over file-level Git work
- the feature set is small enough to understand quickly

That is the point.
This is a maintainer tool, not a full Git shell replacement.

### Keyboard
---

- `1` local branches
- `2` remotes
- `3` tags
- `4` graph
- `tab` / `shift+tab` switch sections
- `up` / `down` / `j` / `k` move
- `enter` inspect or execute the current action
- `f` fetch
- `q` quit

### Status
---

MVP.
This README is kept simple on purpose.
The implementation will keep changing as the graph workflow gets sharper.

### Docs

- `docs/structure.md` - current code map
- `docs/roadmap.md` - next work order
- `docs/archive/` - old plans and moved docs
