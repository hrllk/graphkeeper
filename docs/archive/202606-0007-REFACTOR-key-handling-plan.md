# `internal/app/key_handling.go` 리팩토링 계획서

## 목표

`key_handling.go`는 모든 모드와 섹션의 key event를 중앙에서 디스패치하고, 상태 전이와 실행 정책은 각 책임 helper로 나눈다.

핵심은 `handleKeyMsg`가 모든 key event를 받되, 너무 많은 상태와 액션을 직접 다루지 않도록 만드는 것이다.
`branchOpen`, `ModeTargetPick`, `ModeConfirm`, `ModeOutcomePreview`, `ModeBrowse`를 기준으로 흐름을 먼저 분리하고, 그 안에서 액션별 helper를 호출하는 구조로 바꾼다.

## 범위

- 대상 파일
  - `internal/app/key_handling.go`
- 관련 파일
  - `internal/app/actions.go`
  - `internal/app/navigation.go`
  - `internal/app/target_items.go`
  - `internal/app/graph_rules.go`
- 관련 테스트
  - `internal/app/model_test.go`
  - `internal/app/actions_test.go`
  - `internal/app/commands_test.go`
- 새 package 생성: 하지 않음

## 참조 문서

이 계획은 다음 문서의 분리 원칙을 이어받는다.

- `docs/archive/cli-structure-plan.md`
- `docs/archive/architecture.md`
- `docs/archive/202606-0001-REFACTOR-actions-refactor-test-plan.md`
- `docs/archive/202606-0005-REFACTOR-navigation-graph-rules-plan.md`
- `docs/archive/pull-reset-ux-implementation-plan.md`

## 현재 문제

현재 `handleKeyMsg`는 다음을 한 함수에서 모두 처리한다.

- branch 입력 모달
- target 선택 모드
- confirm 모달
- outcome preview 모드
- browse state 키 입력
- pull / push / merge / rebase / reset / abort 실행 분기
- graph section의 target gating
- section 이동
- graph navigation
- checkout / fetch / branch creation

이 구조는 동작상 문제를 바로 만들지는 않지만, 다음 문제가 있다.

- 상태별 책임이 섞여 읽기 어렵다
- action policy와 key dispatch가 한 함수에 붙어 있다
- pull/reset UX 같은 정책이 키 처리 안에 그대로 묻힌다
- 나중에 키를 바꿀 때 상태 전이가 같이 흔들리기 쉽다

## 분리 원칙

### `key_handling.go`에 남길 것

- 모든 모드와 섹션의 키 입력 수신
- 키 입력의 1차 분기
- 상태별 라우팅
- command 반환
- 입력 모달의 초벌 처리

### helper로 뺄 것

- target 선택 모드 처리
- confirm 상태별 실행 분기
- outcome preview 상태별 실행 / 취소 분기
- browse 상태별 action gating
- branch 생성 모달 처리
- action별 실행 의도 결정
- key와 무관한 policy 해석

## 권장 구조

```go
func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd)

func (m model) handleBranchOpenKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
func (m model) handleTargetPickKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
func (m model) handleOutcomePreviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
func (m model) handleBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)

func (m model) handleBrowseGraphKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
func (m model) handleBrowseSectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)

func (m model) handleConfirmPullKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
func (m model) handleConfirmActionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd)
```

추가로 필요하면 아래 helper를 둘 수 있다.

```go
func browseKeyActionForSection(m model, key string) (tea.Model, tea.Cmd)
func canStartGraphAction(rs git.Status, section graphSection) bool
func confirmActionCommand(m model) (tea.Model, tea.Cmd)
```

중요한 점은 helper가 단순 분기만 담당하고, 실제 Git side effect 는 `commands.go`가 유지하는 것이다.

## 구체 분해안

### 1. branch 모달 분리

현재 `branchOpen` 처리 로직은 `handleKeyMsg` 초반에 붙어 있다.
이 부분은 별도 helper로 분리한다.

책임:

- `esc` 로 branch 모달 닫기
- `enter` 로 branch 생성 실행
- `backspace` 처리
- 문자 입력 누적

### 2. confirm 모달 분리

현재 confirm 상태는 pull / merge / rebase / reset / push / abort를 모두 한 switch 에서 처리한다.
이 부분은 액션별 helper로 쪼갠다.

권장 흐름:

- `ActionPull` 은 fast-forward 여부에 따라 `executePull` 또는 `executePullMerge` / `executePullRebase` 를 호출
- `ActionSetUpstream` 은 push upstream 실행
- `ActionForcePush` 는 force push 실행
- `ActionReset` / `ActionMerge` / `ActionRebase` 는 `executeAction` 실행
- `ActionAbort` 는 abort 실행

### 3. target 선택 모드 분리

`ModeTargetPick` 는 up/down 으로 대상을 고르고, enter 또는 space 로 preview 로 넘어간다.

책임:

- `up/down` 으로 `TargetIdx` 이동
- `space` 또는 `enter` 로 preview 요청
- `esc` 로 browse 복귀
- target 없음이면 blocked 상태 유지

이 모드는 browse 와 다르다. browse 는 그래프와 섹션을 탐색하는 모드이고, target 선택은 action preview 를 위한 별도 모드다.

### 4. outcome preview 모드 분리

`ModeOutcomePreview` 는 preview 결과를 보여주고, 실행 가능한 경우에만 실제 command 를 발동한다.

책임:

- `space` 또는 `enter` 로 execute
- `esc` 로 browse 또는 target pick 복귀
- `ActionPull` / `ActionAbort` / `ActionMerge` / `ActionRebase` / `ActionReset` 별 실행 분기

### 5. browse 모드 분리

browse 키 입력은 section별로 성격이 다르다.

권장 분기:

- Graph 섹션
  - merge / rebase / reset gate
  - lane 이동
  - go-top / go-bottom
  - new branch 진입
- Current / Remote 섹션
  - checkout
  - pull
  - push
  - branch creation
- Tags 섹션
  - 특별 action 없음

이 분리는 `navigation.go`와 `actions.go`가 이미 책임을 나누고 있으므로, key handler는 그 결과를 호출만 해야 한다.

`Browse`는 `Global`의 동의어가 아니다.
여기서 `global`은 `ModeBrowse` 안에서 섹션과 무관하게 동작하는 공통 hotkey 묶음을 뜻한다.

권장 순서:

1. mode 레벨 분기
2. section 레벨 분기
3. global hotkey 처리
4. section hotkey 처리

### 6. policy를 helper로 올리기

`key_handling.go`에서 다음 같은 policy를 직접 새로 만들지 않는다.

- local branch 판정
- graph lane 판정
- target 가능 여부
- pull 가능 여부
- reset 대상 가드

이런 판단은 이미 `actions.go`, `navigation.go`, `graph_rules.go`, `target_items.go`에서 책임을 갖고 있으므로, key handling은 그 결과를 사용만 해야 한다.

## BEFORE

현재 구조는 하나의 함수에 모든 상태와 키가 몰려 있다.

```go
func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.branchOpen {
		switch msg.String() {
		case "esc":
			...
		case "enter":
			...
		}
	}
	if m.status.Mode == state.ModeConfirm {
		switch msg.String() {
		case "y", "enter":
			...
		case "m":
			...
		}
	}
	switch msg.String() {
	case "m":
		...
	case "r":
		...
	case "s":
		...
	}
}
```

이 구조는 빠르게 작성하기는 쉽지만, 기능이 조금만 늘어도 분기 충돌이 생기기 쉽다.

## AFTER

`handleKeyMsg`는 상태별 라우터만 남긴다.

```go
func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.branchOpen {
		return m.handleBranchOpenKey(msg)
	}
	if m.status.Mode == state.ModeConfirm {
		return m.handleConfirmKey(msg)
	}
	return m.handleBrowseKey(msg)
}
```

그리고 각 상태별 함수가 더 작은 helper를 호출한다.

```go
func (m model) handleBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "m":
		return m.handleBrowseGraphMerge(msg)
	case "r":
		return m.handleBrowseGraphRebase(msg)
	case "s":
		return m.handleBrowseGraphReset(msg)
	case "p":
		return m.handleBrowsePull(msg)
	default:
		return m, nil
	}
}
```

## 테스트

리팩토링 전후 동작을 다음 기준으로 고정한다.

### branch 모달

- `esc` 가 branch modal 을 닫는지
- `enter` 가 branch 생성 command 를 반환하는지
- `backspace` 가 draft 를 한 글자 지우는지
- 일반 문자 입력이 draft 에 누적되는지

### target 선택 모드

- `up/down` 이 target index 를 바꾸는지
- `space` / `enter` 가 preview 로 넘어가는지
- `esc` 가 browse 로 돌아가는지
- target 이 없을 때 blocked 상태가 유지되는지

### confirm 모달

- pull confirm 이 fast-forward 여부에 따라 다른 command 를 반환하는지
- force push confirm 이 올바른 command 를 반환하는지
- reset / merge / rebase confirm 이 selected target 을 유지하는지
- abort confirm 이 merge/rebase in-progress 상태에서만 실행되는지

### outcome preview

- `space` / `enter` 가 execute 를 트리거하는지
- `esc` 가 preview 를 닫는지
- `ActionPull` 과 `ActionAbort` 의 복귀 동작이 다른지
- `ActionMerge` / `ActionRebase` / `ActionReset` 이 target 을 유지한 채 실행되는지

### browse 모드

- Graph 섹션에서 merge/rebase/reset gate가 유지되는지
- Current / Remote 섹션에서 checkout 동작이 유지되는지
- `p` 가 pull precheck 또는 status update 를 유지하는지
- `n` 이 branch creation 진입을 유지하는지
- `tab` / `shift+tab` 이 섹션 전환을 유지하는지
- `f`, `P`, `a`, `g`, `G`, `H`, `ctrl+u`, `ctrl+d`, `space` 가 기존처럼 동작하는지

### 경계 케이스

- detached HEAD 에서 merge/rebase가 막히는지
- remote target 이 checkout fallback 을 유지하는지
- pull preview / reset preview 흐름이 그대로 유지되는지

## 구현 순서

1. `handleBranchOpenKey`, `handleTargetPickKey`, `handleConfirmKey`, `handleOutcomePreviewKey`, `handleBrowseKey` 를 먼저 분리한다.
2. confirm 흐름에서 pull / push / reset / merge / rebase / abort 를 액션별 helper로 나눈다.
3. target pick / outcome preview 의 복귀 규칙을 고정한다.
4. browse 흐름에서 Graph / Current / Remote / Tags 를 나눈다.
5. key handling 안에 남아 있는 policy 계산을 `actions.go` / `navigation.go` / `target_items.go`로 이동시킨다.
6. 테스트를 추가하고 기존 테스트가 모두 통과하는지 확인한다.
7. `go vet ./...` 와 `go test ./...` 로 최종 검증한다.

## 완료 기준

- `handleKeyMsg`는 상태별 라우터 역할만 한다.
- branch / target pick / confirm / outcome preview / browse 흐름이 분리된다.
- key handling 안에 독립적인 policy 판정이 거의 남지 않는다.
- `actions.go`, `navigation.go`, `target_items.go`와 규칙이 중복되지 않는다.
- 기존 키 동작과 UX가 유지된다.

## 비고

- 이 단계는 `commands.go` 리팩토링보다 먼저 해도 되고, 같이 해도 된다.
- 다만 `key_handling.go`에서 policy를 새로 만들기 시작하면, 이전에 정리한 helper 분리가 다시 무너진다.
- 따라서 이 문서의 핵심은 “키 입력 라우팅만 남기고, 의사결정은 기존 helper를 재사용한다”는 점이다.
