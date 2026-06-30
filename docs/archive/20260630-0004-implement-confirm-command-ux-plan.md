# Confirm Command UX Implementation Plan

## 목적

`graphkeeper`의 실행형 command 를 공통 `confirm` UX 로 통일한다.

이번 계획의 핵심은 다음이다.

1. `pull`, `reset`, `checkout`, `merge`, `rebase`, `push`, `force push`, `set-upstream`, `create branch` 같은 실행형 행위를 같은 확인 흐름으로 묶는다.
2. 실행 직전에는 항상 `confirm` 을 거치게 한다.
3. 실행 중에는 기존에 구현한 공통 progress toast 를 사용한다.
4. `create branch` 는 `confirm -> branch name input -> execute toast -> create` 순서로 처리한다.
5. 읽기 전용 refresh 명령은 이번 범위에서 제외하고, 필요하면 별도 문서로 분리한다.

## 참조 문서

이 문서는 다음 최근 문서를 기준으로 작성한다.

- `docs/archive/20260630-0001-implement-progress-toast-plan-d.md`
- `docs/archive/20260625-0015-feature-pull-reset-ux-implementation-plan.md`
- `docs/archive/20260625-0016-feature-reset-stash-plan.md`
- `docs/archive/20260625-0018-refactor-alert-messages-english-concise-plan.md`
- `docs/structure.md`
- `docs/model-refactor-plan.md`

## 현재 관찰된 구현 상태

현재 코드는 이미 부분적으로 confirm / preview / loading 을 나눠 두었다.

- `pull` 은 fetch 후 analysis 를 거쳐 confirm 으로 들어간다.
- `merge` / `rebase` 는 target pick 또는 preview 이후 confirm 으로 들어간다.
- `reset` 은 Graph 섹션에서 target 선택 후 mode picker 로 이어진다.
- `create branch` 는 Graph 섹션에서 `n` 을 누르면 즉시 branch 이름 입력 모드로 들어간다.
- `progress toast` 는 구현되어 있으며, execution 중 짧은 실행 안내를 보여줄 수 있다.

문제는 이 흐름들이 command 별로 제각각이라서, 사용자 입장에서 같은 종류의 실행 행위가 서로 다른 UX 로 보인다는 점이다.

## 핵심 판단

### 1. confirm 은 실행형 command 의 공통 게이트다

이번 작업에서 말하는 `confirm` 은 단순한 yes/no 팝업이 아니라, 실제 Git command 를 실행하기 전에 반드시 거치는 공통 게이트다.

권장 원칙:

- `checkout` 과 `pull` 은 working tree 가 dirty 가 아닐 때만 confirm 을 연다.
- 그 외 실행형 command 는 가능한 한 모두 `confirm` 을 먼저 띄운다.
- confirm 에서는 행위와 결과를 짧게 보여준다.
- 실행 중에는 공통 progress toast 만 보여준다.
- 완료 후에는 기존 `Browse`, `Blocked`, `OutcomePreview`, `TargetPick` 상태로 복귀한다.

### 2. target picker 는 confirm 의 대체물이 아니다

`merge`, `rebase`, `reset` 같은 행위는 target 선택이 필요한 경우가 있다.

이때 target picker 나 preview 는 유지할 수 있지만, 최종 실행 전에 confirm 을 한 번 더 거치게 한다.

즉,

- target 선택 단계
- confirm 단계
- execute toast 단계

를 분리한다.

### 3. create branch 는 confirm 후 name input 을 받는다

브랜치 생성은 target 선택형 command 와 다르다.

권장 흐름:

1. Graph 에서 `n` 입력
2. `Create new branch?` confirm 표시
3. working tree 가 dirty 이면 즉시 막는다
4. 사용자가 승인하면 branch name input 열기
5. 이름 입력 후 실행
6. 실행 중에는 `Creating branch...` toast 표시

이 구조를 쓰면 사용자는 “지금 branch 를 만들 것인지”와 “무슨 이름으로 만들 것인지”를 분리해서 이해할 수 있다.

dirty 여부는 우선 `git.Status.WorktreeDirty` 하나로 판단한다. staged / unstaged 분리는 다음 단계에서 쪼갠다.

### 4. fetch 는 이번 범위에서 제외한다

`fetch` 는 상태 갱신/동기화 성격이 강하고, 사용자가 데이터 변경 결과를 직접 결정하는 command 가 아니다.

따라서 이번 문서에서는 `fetch` 를 confirm 게이트 대상에서 제외하고, 기존 progress toast 흐름만 유지한다.

필요하면 다음 단계에서 fetch 도 별도 정책으로 묶는다.

## 범위

### 포함

- `checkout`
- `pull`
- `reset`
- `merge`
- `rebase`
- `push`
- `force push`
- `set-upstream`
- `create branch`
- 위 command 들의 confirm 진입점 공통화
- branch name input 을 confirm 흐름 안으로 이동
- execution toast 와 confirm 문구의 일관성 정리
- `checkout` / `pull` 은 dirty 상태가 아닐 때만 confirm 을 연다

### 제외

- `fetch` 의 confirm 게이트 추가
- Git command 의미 변경
- preview 알고리즘 변경
- 충돌 해결 UX 재설계
- 새 notification framework 도입

## 상태 모델 방향

현재 `state.Status` 와 `model` 상태만으로도 구현 가능하다.

권장 방향은 다음과 같다.

### 1. confirm 은 기존 `ModeConfirm` 을 재사용한다

새로운 전역 modal mode 를 만들지 않는다.

confirm 화면은 이미 있는 `ModeConfirm` 으로 처리하고, action 별로 `Title` / `Message` / `Detail` 을 채운다.

### 2. branch 생성은 별도 action 으로 명시한다

현재 branch 생성은 input overlay 로만 시작된다.

이번 작업에서는 branch 생성 의도를 명확히 표현하기 위해 action 레벨에 branch create 를 추가하는 쪽이 맞다.

예시 방향:

```go
type Action string

const ActionCreateBranch Action = "create-branch"
```

### 3. branch name input 은 기존 overlay 를 재사용한다

브랜치 이름 입력은 confirm 이후의 두 번째 단계로 둔다.

즉, 새 UI component 를 만들기보다 현재 `branchOpen` / `branchDraft` 를 input 단계로 재사용한다.

### 4. 실행 중 표시는 progress toast 로 통일한다

이미 구현된 공통 toast helper 를 사용한다.

예시:

```go
m.status = loadingToast("Creating branch...")
return m, createBranch(m.repo, name, base, m.commitLimit)
```

## 구현 흐름

### checkout

- Graph 또는 branch 섹션에서 선택한 대상에 대해 confirm 을 띄운다.
- working tree 가 dirty 면 진입을 막는다.
- confirm 문구는 짧게 유지한다.
- 승인 후 checkout 을 실행한다.
- 실행 중에는 `Checking out...` toast 를 보여준다.

### pull

- 현재 브랜치의 upstream 상태를 기반으로 confirm 을 띄운다.
- working tree 가 dirty 면 진입을 막는다.
- diverged 이면 merge / rebase 선택은 기존 흐름을 유지하되, 최종 실행 전 confirm 을 거친다.
- 승인 후에는 `Pulling...`, `Merging pull...`, `Rebasing pull...` toast 를 보여준다.

### reset

- Graph 에서 target 을 고른 뒤 `target pick -> confirm -> mode picker -> execute` 순서로 진행한다.
- confirm 승인 후 reset mode picker 를 보여준다.
- mode 선택 후 실행한다.
- 실행 중에는 `Soft reset...`, `Mixed reset...`, `Hard reset...` toast 를 보여준다.

### merge / rebase

- `target pick -> preview -> confirm -> execute` 순서로 진행한다.
- preview 와 confirm 은 연속된 같은 흐름으로 보이게 유지한다.
- 승인 후 실행한다.
- 실행 중에는 `Merging...`, `Rebasing...` toast 를 보여준다.

### push / force push / set-upstream

- push 계열은 confirm 문구를 공통화한다.
- no-upstream, force push, set-upstream 은 action 이름만 다르고 UX 패턴은 같다.
- 승인 후 실행 중에는 `Pushing...`, `Force pushing...`, `Pushing and tracking...` toast 를 보여준다.

### create branch

- Graph 에서 `n` 을 누르면 곧바로 input 으로 가지 말고 confirm 을 먼저 띄운다.
- 사용자가 승인하면 branch name input 을 연다.
- 이름 입력 후 execute 한다.
- 실행 중에는 `Creating branch...` toast 를 보여준다.

## 공통 문구 규칙

문구는 기존 영문화/간결화 문서 규칙을 따른다.

- title 은 짧게
- message 는 현재 상태를 한 줄로
- detail 은 꼭 필요한 경우만 한 줄 더
- confirm 과 toast 문구는 서로 중복되지 않게 분리

예시:

- `Create branch?`
- `Move current branch pointer?`
- `Continue with merge?`
- `Please wait.`

## 권장 파일 경계

기존 구조를 크게 흔들지 않는 방향이 좋다.

권장 수정 후보:

- `internal/app/key_handling_browse.go`
- `internal/app/key_handling_confirm.go`
- `internal/app/key_handling_branch.go`
- `internal/app/key_handling_reset.go`
- `internal/app/key_handling_target.go`
- `internal/app/key_handling_outcome.go`
- `internal/app/update_execute.go`
- `internal/app/update_fetch.go`
- `internal/app/progress.go`
- `internal/app/view_shell.go`
- `internal/state/state.go`

`branch` 입력은 새 파일을 만들기보다 기존 `branchOpen` 상태를 유지하는 쪽이 안전하다.

## 구현 순서

1. 실행형 command 목록을 확정한다.
2. `ActionCreateBranch` 를 추가한다.
3. 각 command 의 confirm 진입점을 공통 helper 로 묶는다.
4. branch 생성에 `confirm -> name input -> execute` 흐름을 연결한다.
5. execution toast 를 모든 command 에서 동일한 경계로 보여준다.
6. confirm / toast 문구를 영어로 짧게 정리한다.
7. 관련 테스트를 추가한다.

## 테스트 전략

문자열과 상태 전이가 같이 바뀌므로, 테스트를 세 단계로 나눈다.

### 1. confirm 진입 테스트

- `checkout` 이 바로 실행되지 않고 confirm 으로 들어가는지
- `checkout` 이 dirty 상태에서는 진입이 막히는지
- `reset` 이 target 선택 후 confirm 을 거치는지
- `reset` 이 `target pick -> confirm -> mode picker -> execute` 순서를 따르는지
- `merge` / `rebase` 가 최종 실행 전에 confirm 을 거치는지
- `merge` / `rebase` 가 `target pick -> preview -> confirm -> execute` 순서를 따르는지
- `push` / `force push` / `set-upstream` 이 confirm 을 거치는지
- `create branch` 가 confirm 을 먼저 띄우는지
- `pull` 이 dirty 상태에서는 진입이 막히는지
- `pull` 이 clean 상태에서만 confirm 을 여는지

### 2. branch create 흐름 테스트

- `n` 입력 시 confirm 이 열리는지
- dirty 상태에서는 branch create 가 막히는지
- confirm 승인 후 branch name input 이 열리는지
- 이름 입력 후 loading toast 가 보이는지
- 실행 성공 시 browse 상태로 복귀하는지

### 3. toast / cancel 테스트

- 승인 직후 `ModeLoading` 이 되는지
- cancel 시 상태가 원래 browse / preview 로 돌아가는지
- confirm 에서 `esc` 로 취소하면 execute command 가 발행되지 않는지
- branch name input 에서 `esc` 로 취소하면 confirm 또는 browse 로 정상 복귀하는지
- preview 단계에서 `esc` 로 취소하면 target pick 으로 돌아가는지
- 실패 시 `Blocked` 또는 `Error` 로 수렴하는지

권장 테스트 파일:

- `internal/app/key_handling_test.go`
- `internal/app/model_test.go`
- `internal/app/commands_test.go`
- `internal/app/progress_test.go`

## 검증

```sh
go test ./internal/app
go test ./...
go build ./cmd/graphkeeper
```

## 비고

- 이번 계획은 command execution UX 를 통일하는 문서다.
- navigation 키와 read-only refresh 를 억지로 confirm 으로 감싸지 않는다.
- branch 생성만은 예외적으로 confirm 다음에 name input 이 들어가도록 분리한다.
- 이미 구현된 progress toast 와 충돌하지 않게, confirm 과 execution 을 층으로 나눠서 유지한다.
