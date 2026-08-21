# Architecture Legacy Coupling Ledger

This repository-local ledger is the sole input for ARCH-G0 guard tests. External copies under `~/.gstack` are planning references only and are not read by CI.

| Package/file/symbol | Forbidden dependency | Owner | Removal condition |
|---|---|---|---|
| `internal/app/model.go:model.repo` | `*git.Repo` | 2.7/2.8 | selected read/pull flows no longer use it |
| `internal/app/model.go:model.repoStatus` | `git.Status` | 2.6/2.7/2.8 | selected graph/read/pull flows use neutral snapshots |
| Inspector path in `internal/app/commit_inspector*.go` and `model.commitInspectorInspection` | `internal/git`, `git.CommitInspection` | 2.5 | all Inspector symbols consume neutral DTOs |
| `internal/app/repository_contract.go:gitRepositoryAdapter` | Git adapter implementation | 2.7/2.8 | selected adapters move to outbound packages |
| `internal/graph/graph.go` | `internal/git` | 2.6 | graph consumes graph-owned snapshot types |
| `internal/app/tag_provenance.go` persistence symbols | filesystem, JSON, `.git` path | 2.9 | persistence moves to tagprovenance adapter |
| `internal/app/commands.go:executeFetchForPull` | pull prefetch Git calls | 2.8 | PullPort.Preview outbound step |
| `internal/app/commands.go:resolvePullFastForward` | Git-based pull impact | 2.8 | PullPort.Preview |
| `internal/app/commands.go:loadPullPreviewCommits` | Git preview commit loading | 2.8 | PullPort.Preview |
| `internal/app/commands.go:executeValidatedPull` | final pull dispatch | 2.8 | PullPort.Execute |
| `internal/app/commands.go:executePull`, `executePullMerge`, `executePullRebase` | Git pull command construction | 2.8 | outbound pull adapter |
| `internal/app/commands.go:validateAndExecutePull` | pull orchestration coupled to Git | 2.8 | neutral workflow over PullPort/P3 read port |
| `internal/app/key_handling_outcome.go:ActionPull` | direct pull workflow invocation | 2.8 | neutral workflow |
| `internal/app/key_handling_confirm.go` pull cases | direct pull execution | 2.8 | neutral workflow |
| `internal/app/key_handling_browse.go` pull callers | direct preview workflow | 2.8 | neutral PullPort.Preview |
| `internal/app/update_fetch.go` pull caller | direct pull prefetch | 2.8 | neutral workflow |
| `internal/app/update.go` pull caller | direct pull update | 2.8 | neutral workflow |
| `internal/app/model.go:activePullRequest` | legacy pull identity/port coupling | 2.8 | neutral pull workflow identity |
| `internal/app/*.go` and `internal/git/repo.go` telemetry calls | concrete global telemetry | 2.10 | all 46 calls use injected EventSink |

Malformed or missing entries are a failing guard, not an empty baseline. Changes to this ledger require a corresponding architecture review entry and focused guard test update.
