# graphkeeper

`graphkeeper` is a graph-first Git TUI for maintainers.

It helps you read repository topology, understand how local branches relate to upstreams, and perform graph-driven operations without losing sight of the shape of the repo.

This is an MVP.
The product is intentionally small, opinionated, and not yet fully structured.
The goal right now is clarity, not breadth.

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

`graphkeeper` is built for people who need to monitor and operate on Git history as a graph.

- read the commit graph quickly
- see where local branches and remotes point
- understand ahead, behind, and diverged states
- choose graph-based actions from the graph itself
- keep the current branch context visible while working


### What It Is Not
---

`graphkeeper` is not a lazygit replacement.

`lazygit` is a broad Git cockpit.
It is great for staging, committing, diffs, stashing, file-level work, and general daily Git operations.

`graphkeeper` is narrower on purpose.
It focuses on graph awareness, branch topology, and maintainer-style operations.
Think of it as a map for the repository, not the whole cockpit.



### Why It Exists
---

Git history is topology.
If you only look at command output, you can miss the shape of the repository and make bad moves.

`graphkeeper` exists to answer questions like:

- Where is this branch actually pointing?
- Which local branch is the one I should operate on?
- What does the upstream still know that my local branch does not?
- Is this commit safe to reset to?
- Should I merge, rebase, or leave this branch alone?



### Quickstart
---

Build the binary:

```bash
go build -o graphkeeper ./cmd/git-graph-tui
```

Run it:

```bash
./graphkeeper
```



### Neovim Entry Point
---

<!-- Neovim plugin support will be provided. -->
Neovim plugin support will be provided.



### Graph Mental Model
---

The graph is not just a list of commits.

- each node is a commit
- edges show ancestry
- branch labels show where refs point
- upstream context shows what is local and what still lives remotely

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
The README is intentionally conceptual, not a full product spec.
The implementation will keep changing as the graph workflow gets sharper.
