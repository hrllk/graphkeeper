# Local Stash / Cleanup Plan

## 목적

`Local` 섹션에서 dirty 상태의 변경사항을 한 번에 정리할 수 있는 기능을 추가한다.

이번 계획의 핵심은 다음이다.

1. dirty 상태에서만 정리 동작을 노출한다.
2. dirty 에 포함된 staged / unstaged / untracked 를 하나의 묶음으로 다룬다.
3. 사용자는 변경사항을 `stash` 하거나 `working tree` 를 완전히 정리할 수 있어야 한다.
4. 파괴적인 동작은 반드시 confirm 을 거치게 한다.

## 참조 문서

이 문서는 다음 최근 문서를 기준으로 작성한다.

- `docs/archive/20260625-0016-feature-reset-stash-plan.md`
- `docs/archive/20260630-0004-implement-confirm-command-ux-plan.md`
- `docs/archive/20260701-0002-local-pane-split-layout-plan.md`
- `docs/roadmap.md`
- `docs/structure.md`

## 현재 관찰된 구현 상태

현재 코드에는 dirty / stash 관련 기반은 이미 존재한다.

- `WorktreeDirty` 와 `WorktreeStateDirty` 가 dirty 상태를 표현한다.
- `stashSummaryLines()` 와 `stashesForCommit()` 로 stash 요약을 보여준다.
- `update_stash.go` 는 stash 로드 상태를 받아 UI 에 반영한다.
- `actions.go` 와 `key_handling_browse.go` 에서 dirty worktree 는 이미 공통 차단 사유로 쓰이고 있다.
- branch 생성, pull, checkout 같은 실행형 action 은 dirty 상태에서 차단되는 경우가 있다.

즉, 이번 작업은 dirty 개념을 새로 만드는 것이 아니라, 이미 있는 dirty 판단 위에 `stash all` 과 `clean working tree` 정리 동작을 얹는 작업이다.

## 핵심 판단

### 1. dirty 는 staged / unstaged / untracked 를 모두 포함한다

이번 기능에서 dirty 는 단순한 변경 여부가 아니라, 현재 작업을 정리해야 하는 상태 전체를 뜻한다.

포함 대상:

- staged changes
- unstaged changes
- untracked files

즉, 사용자가 "정리해야 할 것"은 한 가지 dirty 상태로 묶어서 판단한다.

### 2. stash 는 dirty 상태를 보존하는 안전한 정리 수단이다

stash 는 현재 작업을 잃지 않고 다음 작업으로 넘어가기 위한 경로다.

이번 계획에서 stash 는 다음을 보장해야 한다.

- 현재 working tree 의 변경사항을 모두 담는다.
- untracked 파일도 함께 포함한다.
- stash 후에는 working tree 가 clean 상태가 된다.

### 3. clean working tree 는 변경사항을 버리는 파괴적 정리 수단이다

`clean working tree` 는 stash 와 다르다.

- stash: 변경사항을 보존한다.
- clean working tree: 변경사항을 버린다.

따라서 clean 경로는 confirm 을 반드시 거쳐야 한다.

권장 문구는 다음과 같다.

- `Commit or stash changes first.`
- `Clean working tree?`
- `This will discard local changes and untracked files.`

## 범위

### 포함

- `Local` 섹션에서 dirty 상태 정리 액션 노출
- `stash` 동작에 untracked 포함
- `clean working tree` 동작 추가
- dirty 상태에서만 액션 활성화
- confirm / blocked 문구 정리
- 관련 테스트 추가

### 제외

- staged / unstaged / untracked 세부 상태를 별도 UI 로 분리
- conflict 해결 UX 재설계
- stash 목록 편집 기능
- Graph 섹션 레이아웃 변경

## 기능 정의

### 1. stash all

`stash` 는 현재 작업 중인 변경사항을 모두 저장한다.

권장 구현 방향:

- `git stash push --include-untracked -m <msg>`
- dirty 상태가 아닐 때는 비활성화
- 실행 후에는 stash 목록과 dirty 상태를 다시 갱신

stash 메시지 예시:

- `graphkeeper: local cleanup`
- `graphkeeper: local changes`

권장 구현은 `git add` 전처리를 두지 않는 것이다.
`git stash push --include-untracked` 자체가 staged / unstaged / untracked 를 한 번에 보존하므로, 인덱스를 먼저 바꾸지 않는 편이 더 안전하다.

구현 스니펫:

```go
func (r *Repo) StashAll(ctx context.Context, message string) error {
	_, err := r.Run("stash", "push", "--include-untracked", "-m", message)
	return err
}
```

UI 실행 스니펫:

```go
func executeStashAll(repo *git.Repo, limit int, message string) tea.Cmd {
	return func() tea.Msg {
		if err := repo.StashAll(context.Background(), message); err != nil {
			return executedMsg{action: state.ActionStash, err: err}
		}
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionStash, err: statusErr}
		}
		return executedMsg{action: state.ActionStash, status: status}
	}
}
```

### 2. clean working tree

`clean working tree` 는 현재 작업 디렉터리를 완전히 정리하는 동작이다.

권장 구현 방향은 두 단계다.

1. tracked 변경사항은 `git reset --hard` 로 되돌린다.
2. untracked 파일은 별도 제거 단계를 거친다.

`git reset --hard` 는 tracked 파일만 HEAD 상태로 되돌린다. `untracked` 는 남기 때문에, "완전 정리"를 만들려면 `git clean -fd` 같은 별도 단계가 꼭 필요하다.

가능한 구현 후보:

- `git clean -fd`
- `git clean -fdx` 는 너무 공격적일 수 있으므로 기본값에서 제외

이 기능은 실수 비용이 크므로, `stash` 보다 더 강한 confirm 이 필요하다.

기본 권장안은 `reset --hard` 후 `clean -fd` 다.

```go
func (r *Repo) CleanWorkingTree(ctx context.Context, includeIgnored bool) error {
	if _, err := r.Run("reset", "--hard"); err != nil {
		return err
	}
	args := []string{"clean", "-fd"}
	if includeIgnored {
		args = append(args, "x")
	}
	_, err := r.Run(args...)
	return err
}
```

ignored 파일까지 지울지 여부는 별도 플래그로 둔다.

```go
func (r *Repo) CleanWorkingTreeDefault(ctx context.Context) error {
	return r.CleanWorkingTree(ctx, false)
}
```

### 3. dirty 상태에서만 활성화

정리 관련 액션은 dirty 상태에서만 보이거나 활성화한다.

권장 노출 규칙:

- clean 상태: `stash`, `clean working tree` 둘 다 비활성
- dirty 상태: 둘 다 활성
- untracked 만 있는 경우도 dirty 로 본다

상태 판정은 기존 helper 를 재사용한다.

```go
func worktreeState(rs git.Status) state.WorktreeState {
	if rs.WorktreeDirty {
		return state.WorktreeStateDirty
	}
	if rs.Root == "" {
		return ""
	}
	return state.WorktreeStateClean
}
```

## UX 배치 원칙

### Local 섹션

정리 액션은 `Local` 섹션에 둔다.

이유:

- 현재 브랜치 기준의 작업물 정리가 목적이다.
- Graph 포인터와 무관하게 현재 작업 디렉터리 자체를 정리하는 행위다.
- `Local` 패널은 현재 브랜치 상태와 작업 상태를 함께 보여주기 적합하다.

현재 코드에서는 `renderContextContent()` 가 상단 우측 `Local` 패널을 담당한다.
따라서 stash / clean 액션도 이 경로의 `Actions` 목록에 추가하는 편이 맞다.

예시 렌더링 스니펫:

```go
func renderActionHelpLines(m model) []string {
	switch m.status.Mode {
	case state.ModeBrowse:
		switch m.activeSection {
		case sectionCurrent:
			lines := []string{}
			if m.repoStatus.WorktreeDirty {
				lines = append(lines,
					"• s: stash changes",
					"• c: clean working tree",
				)
			} else {
				lines = append(lines,
					disabled.Render("• s: stash changes")+" "+muted.Render("(dirty only)"),
					disabled.Render("• c: clean working tree")+" "+muted.Render("(dirty only)"),
				)
			}
			lines = append(lines,
				"• p: pull           • P: push",
				"• a: abort merge",
			)
			return lines
		}
	}
	return []string{"• r: refresh"}
}
```

### confirm

파괴적인 동작은 confirm 을 반드시 거친다.

권장 흐름:

1. `Local` 섹션에서 `stash` 또는 `clean working tree` 선택
2. dirty 상태인지 확인
3. confirm 표시
4. 실행
5. 작업 결과를 다시 refresh

예시 confirm 스니펫:

```go
func confirmStashAll() state.Status {
	return state.New().WithConfirm(
		state.ActionStash,
		"Stash local changes?",
		"This will save staged, unstaged, and untracked files.",
	)
}

func confirmCleanWorkingTree() state.Status {
	return state.New().WithConfirm(
		state.ActionCleanWorkingTree,
		"Clean working tree?",
		"This will discard local changes and untracked files.",
	)
}
```

### 상태 문구

정리 동작은 짧고 명확하게 보여준다.

권장 예시:

- `stash changes`
- `clean working tree`
- `discard untracked files`

## 상태 모델 방향

이번 기능은 기존 `WorktreeState` 를 유지하되, 액션 레벨에서 더 세밀한 확인이 필요하다.

권장 방향:

### 1. dirty 판정은 기존 상태를 재사용한다

새로운 상위 상태를 만들기보다, 기존 `dirty` 판단을 확장해서 쓰는 편이 안전하다.

### 2. untracked 포함 여부는 별도 플래그로 유지할 수 있다

untracked 만 따로 보존해야 하는지, stash / clean 에 함께 포함할지는 내부 구현에서 결정한다.

하지만 UI 에서는 다음처럼만 보인다.

- `dirty`
- `clean`

### 3. stash / clean 액션은 기존 action 흐름에 붙인다

새로운 전역 모달을 만들기보다 기존 confirm / blocked / loading 흐름을 재사용한다.

권장 key handler 흐름:

```go
case "s":
	if !m.repoStatus.WorktreeDirty {
		m.status = state.New().WithBlocked(state.BlockDirtyTree, "Working tree is clean.", "Nothing to stash.")
		return m, nil
	}
	m.status = confirmStashAll()
	return m, nil

case "c":
	if !m.repoStatus.WorktreeDirty {
		m.status = state.New().WithBlocked(state.BlockDirtyTree, "Working tree is clean.", "Nothing to clean.")
		return m, nil
	}
	m.status = confirmCleanWorkingTree()
	return m, nil
```

권장 confirm 실행 흐름:

```go
func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		switch m.status.Action {
		case state.ActionStash:
			m.status = loadingToast("Stashing changes...")
			return m, executeStashAll(m.repo, m.commitLimit, "graphkeeper: local cleanup")
		case state.ActionCleanWorkingTree:
			m.status = loadingToast("Cleaning working tree...")
			return m, executeCleanWorkingTree(m.repo, m.commitLimit, false)
		}
	}
	return m, nil
}
```

## 권장 파일 경계

이 기능은 다음 파일만 건드리면 구현 가능해야 한다.

- `internal/state/state.go`
- `internal/git/repo_exec.go`
- `internal/app/commands.go`
- `internal/app/key_handling_browse.go`
- `internal/app/view_sections.go`
- `internal/app/model_test.go`
- `internal/app/commands_test.go`
- `internal/app/key_handling_test.go`

## 구현 초안

### `internal/state/state.go`

```go
const (
	ActionNone            Action = ""
	ActionPull            Action = "pull"
	ActionAbort           Action = "abort"
	ActionCheckout        Action = "checkout"
	ActionMerge           Action = "merge"
	ActionRebase          Action = "rebase"
	ActionReset           Action = "reset"
	ActionCreateBranch    Action = "create-branch"
	ActionDeleteBranch    Action = "delete-branch"
	ActionPush            Action = "push"
	ActionForcePush       Action = "force-push"
	ActionSetUpstream     Action = "set-upstream"
	ActionPullMerge       Action = "pull-merge"
	ActionPullRebase      Action = "pull-rebase"
	ActionStash           Action = "stash"
	ActionCleanWorkingTree Action = "clean-working-tree"
)
```

### `internal/git/repo_exec.go`

```go
func (r *Repo) StashAll(ctx context.Context, message string) error
func (r *Repo) CleanWorkingTree(ctx context.Context, includeIgnored bool) error
```

### `internal/app/commands.go`

```go
func executeStashAll(repo *git.Repo, limit int, message string) tea.Cmd
func executeCleanWorkingTree(repo *git.Repo, limit int, includeIgnored bool) tea.Cmd
```

## 구현 순서

1. dirty 판정이 untracked 를 포함하는지 기준을 확정한다.
2. `stash` 를 untracked 포함 동작으로 정리한다.
3. `clean working tree` 의 파괴 범위를 확정한다.
4. `Local` 섹션에 stash / clean 액션을 노출한다.
5. confirm 문구를 정리한다.
6. dirty 상태에서만 활성화되도록 gating 을 붙인다.
7. stash / cleanup 후 refresh 흐름을 연결한다.
8. 관련 테스트를 추가한다.

## 테스트 방향

아래 항목을 테스트로 고정한다.

1. dirty 상태에서만 `stash` 와 `clean working tree` 가 활성화되는지 확인한다.
2. untracked 파일만 있어도 dirty 로 인식되는지 확인한다.
3. `stash` 가 untracked 를 포함하는지 확인한다.
4. `stash` 후 working tree 가 clean 으로 전환되는지 확인한다.
5. `clean working tree` 가 tracked 변경사항과 untracked 파일을 모두 정리하는지 확인한다.
6. clean 상태에서는 관련 액션이 비활성인지 확인한다.
7. confirm 문구가 파괴적 동작을 명확히 경고하는지 확인한다.

테스트 초안:

```go
func TestStashAllIncludesUntrackedAndTracked(t *testing.T)
func TestCleanWorkingTreeRemovesTrackedAndUntracked(t *testing.T)
func TestLocalSectionShowsCleanupActionsOnlyWhenDirty(t *testing.T)
func TestCleanWorkingTreeConfirmExplainsDiscard(t *testing.T)
```

## 제약

1. staged / unstaged / untracked 를 각각 별도 UI 로 나누지 않는다.
2. `dirty` 와 `clean` 만 기본 노출 상태로 유지한다.
3. `clean working tree` 는 confirm 없이 바로 실행하지 않는다.
4. stash 는 변경사항 보존용이고, cleanup 은 변경사항 폐기용으로 역할을 분리한다.
5. Graph 섹션은 이번 작업 범위에서 건드리지 않는다.
6. `git add` 를 전처리로 사용하지 않는다. stash 는 `git stash push --include-untracked` 를 직접 호출한다.

## 완료 기준

- `Local` 섹션에서 dirty 상태 정리 액션을 사용할 수 있다.
- stash 는 dirty 변경사항과 untracked 를 함께 보존한다.
- clean working tree 는 tracked / untracked 를 모두 정리한다.
- 파괴적 동작은 confirm 을 거친다.
- dirty 상태가 아닌 경우에는 관련 액션이 비활성으로 보인다.
- 문서만 보고도 `internal/git/repo_exec.go`, `internal/app/commands.go`, `internal/app/key_handling_browse.go`, `internal/app/view_sections.go` 를 수정할 수 있다.

## 메모

이 기능은 단순히 stash 버튼을 추가하는 것이 아니다.

핵심은 `dirty` 를 "정리해야 할 현재 작업 상태"로 보고, 이를 `보존(stash)` 과 `폐기(clean)` 두 경로로 명확히 나누는 것이다.

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | — | — |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | — | — |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 2 | issues_open | 0 issues, 0 critical gaps |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | — |
| DX Review | `/plan-devex-review` | Developer experience gaps | 0 | — | — |

**UNRESOLVED:** none.
**VERDICT:** ENG REVIEW CLEARED — ready to implement.
