# Branch Create Name Input Immediate Checkout Plan

## 목적

이 문서는 브랜치 생성 UX 를 1안 기준으로 고정하기 위한 구현 계획서다.

이번 변경의 핵심은 다음 두 가지다.

1. `n` 을 눌렀을 때 기존 `yes / no` confirm 을 거치지 않고, 바로 브랜치 이름 입력 팝업을 연다.
2. 브랜치를 만들면 끝나는 것이 아니라, 생성 직후 곧바로 해당 브랜치로 checkout 까지 진행한다.

즉, 이번 작업은 `branch create` 를 단순 생성 기능이 아니라 `create + checkout` 단일 흐름으로 고정하는 작업이다.

## 참조 문서

이 문서는 다음 최근 문서를 기준으로 작성한다.

- `docs/archive/20260630-0004-implement-confirm-command-ux-plan.md`
- `docs/archive/20260630-0001-implement-progress-toast-plan-d.md`
- `docs/archive/20260630-0002-centered-layout-relayout-followup.md`
- `docs/archive/20260630-0006-reset-confirm-concise-immediate-execution-plan.md`
- `docs/archive/20260625-0016-feature-reset-stash-plan.md`
- `docs/archive/20260625-0018-refactor-alert-messages-english-concise-plan.md`
- `docs/structure.md`

## 현재 관찰된 구현 상태

현재 `create branch` 흐름은 아래처럼 나뉘어 있다.

- Graph 섹션과 Local 섹션에서 `n` 을 누르면 branch 생성 의도를 연다.
- 다만 현재는 `Create new branch?` confirm 을 먼저 띄운다.
- confirm 에서 `enter` 를 눌러야 branch name 입력 overlay 가 열린다.
- branch name 입력 overlay 에서 `enter` 를 눌러야 실제 `git switch -c` 실행이 시작된다.
- 실행 중에는 `Creating branch...` toast 가 표시된다.

즉, 현재 UX 는 `n -> confirm -> input -> execute` 4단계다.

이번 계획은 이 흐름을 `n -> input -> execute` 3단계로 줄인다.

## 핵심 판단

### 1. 브랜치 생성은 confirm 대상이 아니라 입력 대상이다

브랜치 생성은 yes/no 를 묻는 행위가 아니다.

사용자가 실제로 결정해야 하는 것은 "만들지 말지"가 아니라 "어떤 이름으로 만들지"다.

따라서 `Create new branch?` confirm 은 제거하고, `n` 을 누르면 바로 이름 입력 팝업이 떠야 한다.

### 2. create 와 checkout 은 하나의 동작으로 본다

이번 문서에서 branch 생성은 `git switch -c <name> <base>` 로 해석한다.

즉, 성공 조건은 다음과 같다.

- 브랜치가 생성된다
- 동시에 해당 브랜치로 checkout 된다
- 이후 상태 패널과 그래프가 새 브랜치 기준으로 갱신된다

따라서 "브랜치만 만들고 현재 브랜치는 유지"하는 흐름은 이번 범위에 넣지 않는다.

### 3. Graph 섹션과 Local 섹션 모두에서 같은 기능을 제공한다

브랜치 생성은 Graph 섹션 전용 기능이 아니라, Graph 섹션과 Local 섹션 모두에서 지원해야 한다.

두 섹션의 차이는 아래처럼만 둔다.

- Graph 섹션: 현재 가리킨 commit 또는 branch pointer 를 base 로 사용한다
- Local 섹션: 현재 선택된 로컬 branch 를 base 로 사용한다

즉, `n` 의 의미는 두 섹션에서 동일하지만, base 해석만 현재 포커스에 따라 달라진다.

### 4. checkout 이 포함되므로 dirty gate 가 필요하다

브랜치 생성 후 즉시 checkout 까지 진행하면, 현재 worktree 상태가 checkout 가능한지 미리 확인해야 한다.

따라서 `n` 입력 시점에 `worktree dirty` 여부를 검사하고, dirty 면 branch name input 을 열지 말고 즉시 막는다.

또한 merge 또는 rebase 가 진행 중이면 checkout 이 안전하지 않으므로 같은 진입 단계에서 막는다.

이때 사용자는 이미 다른 checkout 류 액션과 같은 이유로 차단되어야 한다.

권장 차단 조건은 기존과 동일하다.

- `BlockDirtyTree`
- `Working tree is dirty.`
- `Commit or stash changes first.`

추가로 다음 상태도 branch create + checkout 진입을 막는 것으로 고정한다.

- `MergeInProgress`
- `RebaseInProgress`
- `Merge/rebase already in progress.`

만약 실행 직전에 상태가 바뀌는 레이스가 있더라도, 실제 command 레이어에서도 한 번 더 dirty 검사를 유지한다.

## 목표 UX

### branch create 흐름

1. Graph 섹션 또는 Local 섹션에서 대상 commit / branch 를 가리킨다.
2. `n` 을 누른다.
3. 현재 worktree 가 clean 이면 branch name 입력 팝업이 바로 열린다.
4. 이름을 입력하고 `enter` 를 누르면 create + checkout 이 실행된다.
5. 실행 중에는 `Creating branch...` toast 가 표시된다.
6. 실행 완료 후 새 브랜치 기준으로 상태가 갱신된다.

### 화면상 기대 모습

- confirm 단계는 보이지 않아야 한다
- branch name 입력 팝업은 바로 열려야 한다
- Graph 섹션과 Local 섹션에서 동작이 일관돼야 한다
- dirty 상태에서는 입력 팝업 자체가 열리지 않아야 한다

## 범위

### 포함

- `internal/app/key_handling_browse.go`
- `internal/app/key_handling_confirm.go`
- `internal/app/key_handling_branch.go`
- `internal/app/update_branch.go`
- `internal/app/commands.go`
- `internal/app/view_shell.go`
- `internal/app/view_detail.go`
- `internal/app/view_sections.go`
- `internal/app/key_handling_test.go`
- `internal/app/commands_test.go`
- `internal/app/model_test.go`

### 제외

- `checkout` 의 별도 UX 재설계
- branch name 추천 기능
- branch rename / delete
- remote branch 에 대한 create branch UX
- confirm 모달 공통 구조 변경

## 상태 모델 방향

이번 작업은 새로운 전역 상태 타입을 추가할 필요가 없다.

현재 있는 `branchOpen`, `branchDraft`, `branchBase` 를 재사용하면 된다.

### `branchOpen`

`branchOpen` 은 branch name 입력 overlay 가 열려 있는지를 나타낸다.

이번 변경 이후의 의미는 아래와 같다.

- `branchOpen = true` 면 사용자는 branch name 을 입력 중이다
- confirm 대기 상태는 더 이상 없다
- enter 는 입력 완료 후 즉시 실행을 시작한다

### `branchBase`

`branchBase` 는 새 브랜치를 어떤 기준점에서 만들지 저장하는 값이다.

Graph 섹션에서는 commit hash 또는 branch pointer 가 들어갈 수 있고, Local 섹션에서는 현재 선택된 branch ref 가 들어갈 수 있다.

이 값은 input overlay 가 열린 시점에 확정되어야 한다.

### `branchDraft`

`branchDraft` 는 사용자가 입력한 branch name 을 누적 저장한다.

이번 변경으로 입력 팝업의 역할이 더 중요해지므로, draft 를 비우는 시점은 다음 두 개로 고정한다.

- `esc` 로 취소할 때
- `enter` 로 실행을 시작할 때

## 구현 계약

### 1. 진입 계약

`n` 키의 진입 계약을 다음처럼 바꾼다.

- Graph 섹션에서 `n` 을 누르면 branch name input 을 바로 연다
- Local 섹션에서 `n` 을 누르면 branch name input 을 바로 연다
- confirm 단계는 거치지 않는다
- `Remote` / `Tags` 섹션은 이번 범위에서 제외한다

### 2. base 선택 계약

branch 생성 시 base 는 현재 포커스에서 직접 계산한다.

권장 해석은 다음과 같다.

- Graph 섹션: 현재 가리키는 commit hash 를 base 로 쓴다
- Local 섹션: 현재 선택된 local branch ref 를 base 로 쓴다

base 가 비어 있으면 branch 생성은 진행하지 않는다.

### 3. dirty gate 계약

`n` 을 눌렀을 때 먼저 dirty 를 검사한다.

dirty 면 다음과 같이 처리한다.

- branch name input 을 열지 않는다
- `BlockDirtyTree` 를 보여준다
- 현재 상태를 browse 로 유지한다

이 체크는 UI 진입 단계와 command 실행 단계 둘 다에서 유지한다.

### 4. duplicate name 계약

브랜치 이름이 이미 존재하면 생성 자체를 막아야 한다.

권장 계약은 다음과 같다.

- `enter` 시점에 입력값을 trim 한다
- 빈 문자열은 기존과 같이 block 한다
- 현재 repo 상태에 같은 이름의 로컬 branch 가 보이면 즉시 block 한다
- command 레이어에서도 `git switch -c` 실패를 받아 동일한 실패 메시지로 정리한다

사용자 관점에서는 `이미 존재하는 브랜치 이름입니다.` 같은 명확한 문구가 보여야 한다.
실패 판정의 기준은 `LocalBranches` 와 현재 checkout 중인 branch name 이다.
즉, 새로 만들려는 이름이 현재 로컬 브랜치 목록에 이미 있으면 막는다.
원격 branch, tag, 기타 ref 와의 더 넓은 충돌은 command 레이어의 Git 실패를 최종 안전망으로 둔다.

### 5. branch input 계약

branch name input overlay 는 기존 `branchOpen` UI 를 그대로 쓴다.

입력 팝업의 기본 동작은 아래처럼 고정한다.

- `esc`: 취소하고 browse 로 복귀
- `backspace`: 마지막 글자 삭제
- 일반 문자 입력: draft 에 추가
- `enter`: branch 생성 + checkout 실행

입력 overlay 에서는 더 이상 `yes / no` confirm 문구를 보여주지 않는다.

### 6. 실행 계약

사용자가 branch name 을 확정하면 아래 순서로 진행한다.

1. 입력 popup 을 닫는다
2. `Creating branch...` progress toast 를 띄운다
3. `git switch -c <name> <base>` 를 실행한다
4. 결과가 오면 repo status 를 갱신한다
5. 새 브랜치로 checkout 된 상태를 반영한다

실행 문구는 짧게 유지하고, success 이후에는 별도 confirm 없이 browse 상태로 돌아간다.

### 7. 중복 문구 정리 계약

branch input overlay 가 열린 동안에는 상단 / detail / footer 에 같은 문구가 중복으로 쌓이지 않게 한다.

권장 원칙은 다음과 같다.

- modal title 은 `Create branch` 또는 동등한 짧은 제목만 보여준다
- body 는 `Enter a branch name.` 같은 한 줄 설명만 보여준다
- footer 는 `esc: back` 정도만 남긴다
- detail 패널은 draft 와 base 를 보여주되, 같은 문장을 다른 위치에 반복하지 않는다

## UI 배치 원칙

### branch modal

- branch modal 은 confirm modal 이 아니라 input modal 이다
- confirm modal 의 `y / n` 도움말은 재사용하지 않는다
- graph/local 두 섹션 모두 같은 branch modal 을 연다
- modal 이 열려도 섹션 위계는 바뀌지 않아야 한다

### help text

Graph 섹션과 Local 섹션의 action help 에서 `n` 은 계속 보여야 한다.

다만 dirty 상태에서는 disabled 표현으로 바꾸고, 이유를 함께 보여준다.

## 테스트 계획

이 변경은 UX 경로가 바뀌므로 테스트를 함께 갱신해야 한다.

### key handling tests

다음 시나리오를 추가하거나 갱신한다.

- Graph 섹션에서 `n` 을 누르면 confirm 없이 branch input 이 열린다
- Local 섹션에서 `n` 을 눌러도 같은 branch input 이 열린다
- dirty 상태에서는 branch input 이 열리지 않고 blocked 상태가 나온다
- 이미 존재하는 branch name 이면 입력 완료 시 막힌다
- `enter` 는 confirm 단계가 아니라 branch input 단계에서만 실행을 시작한다
- `esc` 는 branch input 을 닫고 draft 를 비운다

### command tests

다음 시나리오를 확인한다.

- `createBranch()` 가 `git switch -c` 기반으로 동작하는지
- base 인자가 비어 있을 때 fallback 이 안전한지
- dirty worktree 에서는 branch creation 이 거부되는지
- 중복 branch/ref 이름 충돌이 적절히 실패하는지
- 생성 후 HEAD 가 새 branch 로 이동하는지

### view tests

다음 시나리오를 확인한다.

- Graph 섹션과 Local 섹션 모두 `n: new branch` 가 노출되는지
- dirty 상태에서는 disabled 문구가 보이는지
- branch input overlay 에서는 confirm 문구가 보이지 않는지

## 구현 우선순위

1. `n` 입력 경로에서 confirm 제거
2. Graph / Local 공통 branch input 진입으로 정리
3. dirty gate 를 진입 시점에 추가
4. branch input 의 enter 동작을 create + checkout 에 맞게 고정
5. help text 와 detail 문구를 정리
6. 테스트를 업데이트하고 전체 검증을 통과시킨다

## 최종 구현 기준

이 문서를 보고 바로 구현할 수 있으려면 다음 조건이 충족되어야 한다.

- Graph 섹션과 Local 섹션 모두에서 `n` 으로 branch 생성이 가능해야 한다
- confirm 단계를 거치지 않고 바로 name input 이 열려야 한다
- 생성된 branch 는 즉시 checkout 되어야 한다
- dirty worktree 에서는 진입 전에 막아야 한다
- 테스트는 Graph / Local / dirty / cancel / success 경로를 모두 커버해야 한다
