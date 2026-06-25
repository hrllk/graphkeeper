# `internal/app/update.go` 디스패치 분리 계획서

## 목표

`internal/app/update.go`의 큰 `switch`를 메시지 계열별로 나누어, `Update()`는 라우터만 남기고 실제 상태 전환은 작은 핸들러로 분리한다.

설명은 짧게 유지하고, 이벤트 처리 경계만 분명하게 만든다.

## 범위

- 대상 파일
  - `internal/app/update.go`
- 후보 파일
  - `internal/app/update_lifecycle.go`
  - `internal/app/update_fetch.go`
  - `internal/app/update_execute.go`
  - `internal/app/update_branch.go`
- 관련 파일
  - `internal/app/commands.go`
  - `internal/app/messages.go`
  - `internal/app/navigation.go`
  - `internal/app/actions.go`
  - `internal/app/key_handling_*.go`
  - `internal/app/view.go`
- 관련 테스트
  - `internal/app/model_test.go`
  - `internal/app/commands_test.go`
  - `internal/app/key_handling_test.go`
- 새 package 생성: 하지 않음

## 참조 문서

이 계획은 아래 문서들의 경계 분리 원칙을 이어받는다.

- `docs/archive/202606-0009-refactor-model-structure-plan.md`
- `docs/archive/202606-0010-refactor-model-messages-plan.md`
- `docs/archive/202606-0011-refactor-model-boundary-plan.md`
- `docs/archive/20260625-0012-refactor-navigation-boundary-plan.md`
- `docs/model-refactor-plan.md`

## 현재 문제

`update.go`는 현재 다음 책임을 한 파일에서 처리한다.

- window size 반영
- repo load / refresh 결과 반영
- fetch / prepare 결과 반영
- pull preview 및 pull integration 상태 반영
- executed action 결과 반영
- branch creation 결과 반영
- Bubble Tea key message 진입점

문제는 기능이 아니라 밀도다.
하나의 파일 안에 "패시브 상태 갱신", "비동기 fetch 결과", "실행 결과", "브랜치 생성"이 모두 들어 있어 상태 전이를 추적하기 어렵다.

## 분리 원칙

- `update.go`: `Update()` 진입점만 남긴다.
- `update_lifecycle.go`: window size, loaded, refreshed, tick 같은 생명주기성 갱신
- `update_fetch.go`: fetch / prepare / pull check / preview 계열 결과 처리
- `update_execute.go`: executed action 결과 처리
- `update_branch.go`: branch creation 결과 처리

핵심은 메시지 종류가 아니라 관심사 기준으로 나누는 것이다.

## 구현 포인트

1. `Update()`는 msg type에 따라 handler를 호출만 한다.
2. `loadedMsg`, `refreshedMsg`, `tickMsg`는 생명주기 계열로 묶는다.
3. `fetchedMsg`, `preparedMsg`, `pullCheckedMsg`, `previewMsg`, `pullPreviewReadyMsg`는 fetch/preview 계열로 묶는다.
4. `executedMsg`는 action 결과 처리에만 집중시킨다.
5. `createdBranchMsg`는 branch 생성 전용 경로로 분리한다.

## BEFORE

현재는 하나의 switch 안에 다음이 모두 들어 있다.

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		...
	case loadedMsg:
		...
	case tickMsg:
		...
	case refreshedMsg:
		...
	case fetchedMsg:
		...
	case preparedMsg:
		...
	case pullCheckedMsg:
		...
	case previewMsg:
		...
	case pushFetchedMsg:
		...
	case pullFetchedMsg:
		...
	case pullPreviewReadyMsg:
		...
	case executedMsg:
		...
	case createdBranchMsg:
		...
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}
```

문제는 switch 길이가 아니라, 상태 전이의 도메인이 서로 다른데도 한 곳에서 처리된다는 점이다.

## AFTER

`update.go`는 분기만 남긴다.

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return handleWindowSize(m, msg)
	case loadedMsg, refreshedMsg, tickMsg:
		return handleLifecycleUpdate(m, msg)
	case fetchedMsg, preparedMsg, pullCheckedMsg, previewMsg, pushFetchedMsg, pullFetchedMsg, pullPreviewReadyMsg:
		return handleFetchUpdate(m, msg)
	case executedMsg:
		return handleExecutedUpdate(m, msg)
	case createdBranchMsg:
		return handleBranchUpdate(m, msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	default:
		return m, nil
	}
}
```

각 핸들러는 더 작은 상태 전이만 담당한다.

```go
func handleLifecycleUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd)
func handleFetchUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd)
func handleExecutedUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd)
func handleBranchUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd)
```

## 테스트

현재 유지해야 할 테스트 범위는 다음과 같다.

```go
func TestFetchedMsgKeepsPassiveBrowseState(t *testing.T)
func TestCheckoutResetsGraphLoadState(t *testing.T)
func TestPushSetUpstreamTriggeredWhenNoUpstream(t *testing.T)
func TestPushNormalTriggeredWhenUpstreamExists(t *testing.T)
func TestOutcomePreviewEscapeRoutesByAction(t *testing.T)
```

리팩토링 시 추가로 확인할 케이스는 다음이다.

```go
func TestUpdateRoutesLifecycleMessages(t *testing.T)
func TestUpdateRoutesFetchMessages(t *testing.T)
func TestUpdateRoutesExecutedMessages(t *testing.T)
func TestUpdateRoutesBranchMessages(t *testing.T)
```

## 검증 기준

1. `Update()`가 라우터 역할만 한다.
2. 메시지 계열별 상태 전이가 각각의 파일에 모인다.
3. key handling과 command 실행 계약은 바뀌지 않는다.
4. 기존 tests는 의미를 유지한 채 통과한다.

## 비고

- 이 문서는 command 생성 쪽 분리 문서가 아니다.
- `messages.go`는 전달 계약으로 유지하고, `update.go`는 계약 소비자 역할로 줄인다.
- 상태 전이가 다시 커지면, switch를 늘리기보다 handler 파일을 먼저 늘린다.
