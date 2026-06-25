# `internal/app` 알럿 메시지 영문화 및 간결화 계획서

> **For Hermes:** Use subagent-driven-development to implement this plan task-by-task.

**Goal:** `internal/app`에서 사용자에게 노출되는 알럿 메시지를 영어로 통일하고, 같은 의미를 더 짧고 일관된 문구로 정리한다.

**Architecture:** 상태 전이와 command flow는 그대로 두고, `state.Status`에 들어가는 `Message` / `Detail` / `Title` 문구만 정리한다. 문구 변경으로 동작 의미가 바뀌면 안 된다.

**Tech Stack:** Go, Bubble Tea, `internal/app`, `internal/state`, `go test`

---

### TOC

- ### Goal
- ### Scope
- ### Review Notes
- ### BEFORE
- ### AFTER
- ### Tests
- ### Verification
- ### Notes

---

### Goal

현재 UI 알럿 메시지는 영어와 한글이 섞여 있고, 일부 문구는 같은 의미를 반복해서 길다.

이 계획의 목표는 다음 두 가지다.
- 사용자에게 보이는 모든 알럿 메시지를 영어로 통일한다
- 같은 의미를 더 짧고 직접적인 문구로 줄인다

---

### Scope

- 대상 파일
  - `internal/state/state.go`
  - `internal/app/actions.go`
  - `internal/app/preview.go`
  - `internal/app/key_handling_branch.go`
  - `internal/app/key_handling_browse.go`
  - `internal/app/key_handling_confirm.go`
  - `internal/app/key_handling_outcome.go`
  - `internal/app/key_handling_reset.go`
  - `internal/app/key_handling_target.go`
  - `internal/app/update_branch.go`
  - `internal/app/update_execute.go`
  - `internal/app/update_fetch.go`
  - `internal/app/model.go`
- 관련 테스트
  - `internal/app/actions_test.go`
  - `internal/app/preview_test.go`
  - `internal/app/key_handling_test.go`
  - `internal/app/model_test.go`
  - 메시지 문자열을 직접 검증하는 추가 테스트
- 비범위
  - 상태 전이 로직 변경
  - command / update 구조 변경
  - 새 package 생성
  - 사용자 동작 플로우 변경

---

### Review Notes

- `docs/archive/202606-0002-refactor-commands-test-plan.md`의 테스트 우선 원칙을 따른다.
- 문구는 영어로 통일하되, 의미는 유지한다.
- 길고 설명적인 문장은 줄이고, 필요한 경우에만 detail을 남긴다.
- `Title`은 화면 레이블 역할만 하고, 설명은 `Message` / `Detail`로 분리한다.
- 상태값이나 telemetry event 이름은 바꾸지 않는다.
- `Merge/Rebase`, `Pull`, `Reset`, `Checkout`, `Fetch`, `Push` 같은 핵심 액션 용어는 일관된 영어 표현을 사용한다.

---

### BEFORE

현재 메시지는 한글과 영어가 섞여 있고, 일부는 같은 정보를 두 번 말한다.

```go
return state.New().WithOutcome(state.ActionMerge, "FF 가능. 포인터만 이동합니다.", "HEAD can move to "+target+". "+countDetail(currentOnly, targetOnly), true)
```

```go
status.Message = "Merge/Rebase in progress after conflict."
status.Detail = "Press enter to abort the in-progress merge/rebase."
```

```go
m.status.Message = "Pull stopped with conflicts."
m.status.Detail = "Press enter to abort the in-progress merge/rebase."
```

```go
titleMsg := "Push and Track Remote?"
detailMsg := fmt.Sprintf("There is no upstream configured for the current branch. Do you want to push and set upstream tracking to origin/%s?", branchName)
```

문제는 다음과 같다.
- 같은 화면 안에서 언어가 섞인다
- detail이 길어 읽기 부담이 크다
- 상태별 문구 스타일이 제각각이다

---

### AFTER

문구는 영어로 통일하고, 표현 규칙을 단순화한다.

권장 기준은 다음과 같다.
- title: 2-4단어의 짧은 레이블
- message: 한 문장으로 현재 상태를 말한다
- detail: 다음 행동이나 핵심 맥락만 남긴다
- 불필요한 반복 설명은 제거한다

예상 정리 방향은 다음과 같다.
- `FF 가능. 포인터만 이동합니다.` -> `Fast-forward available.`
- `대상은 이미 포함되어 있습니다.` -> `Target already included.`
- `FF 불가. merge commit이 필요합니다.` -> `Fast-forward unavailable.`
- `Choose soft, mixed, or hard reset.` -> `Choose reset mode.`
- `Running action...` -> `Running action...` 또는 더 짧은 동작형 문구로 정리
- `Pull completed successfully.` -> `Pull complete.`

핵심은 영어 번역이 아니라, 같은 정보를 더 짧고 일관되게 정리하는 것이다.

---

### Tests

문구 변경은 회귀가 눈에 잘 안 보이므로, 문자열을 직접 검증하는 테스트를 함께 고정한다.

우선순위가 높은 테스트는 다음과 같다.

```go
func TestActionPull(t *testing.T) { /* ... */ }
func TestActionPickTargets(t *testing.T) { /* ... */ }
func TestBuildMergePreview(t *testing.T) { /* ... */ }
func TestBuildRebasePreview(t *testing.T) { /* ... */ }
func TestBuildResetPreview(t *testing.T) { /* ... */ }
func TestKeyHandlingConfirmMessages(t *testing.T) { /* ... */ }
func TestExecuteResultMessages(t *testing.T) { /* ... */ }
```

실제 구현 시에는 다음을 확인해야 한다.
- 메시지 변경 후에도 상태 전이는 그대로인지
- `Message`와 `Detail`의 역할이 뒤섞이지 않는지
- 동일한 의미가 여러 파일에서 같은 톤으로 유지되는지

---

### Verification

```sh
go test ./internal/app
go test ./...
go build ./cmd/graphkeeper
```

---

### Notes

- 이 작업은 기능 추가가 아니라 문구 정리다.
- 동작 변경보다 UX 문구의 일관성이 우선이다.
- 문구를 바꿔도 기존 블록 사유, 실행 조건, telemetry 이벤트는 그대로 유지한다.
- `docs/archive/202606-0002-refactor-commands-test-plan.md`의 구조를 참고하되, 이번 계획은 메시지 정리에만 집중한다.
