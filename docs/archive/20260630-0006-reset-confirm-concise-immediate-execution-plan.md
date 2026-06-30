# Reset Confirm Concise Immediate Execution Plan

## 목적

이 문서는 `reset` 확인 흐름을 더 짧고 명확하게 바꾸기 위한 구현 계획서다.

이번 변경의 핵심은 두 가지다.

1. `reset confirm` 에서 `soft / mixed / hard` 3개 모드를 보여주되, 선택 후 `enter` 를 한 번 더 누르지 않아도 즉시 실행되게 한다.
2. `reset confirm` 에서 같은 레이블이 여러 위치에 반복 노출되는 문제를 정리해, 한 번만 읽어도 동작을 이해할 수 있게 만든다.

즉, 이번 작업은 reset 알고리즘 자체를 바꾸는 것이 아니라, `mode pick -> execute` 의 인터랙션과 문구 중복을 정리하는 작업이다.

## 참조 문서

이 문서는 최근 계획과 현재 구현 흐름을 기준으로 작성한다.

- `docs/archive/20260630-0004-implement-confirm-command-ux-plan.md`
- `docs/archive/20260630-0001-implement-progress-toast-plan-d.md`
- `docs/archive/20260630-0002-centered-layout-relayout-followup.md`
- `docs/archive/20260625-0016-feature-reset-stash-plan.md`
- `docs/archive/20260625-0018-refactor-alert-messages-english-concise-plan.md`
- `docs/structure.md`

## 현재 관찰된 구현 상태

현재 `reset` 흐름은 이미 `ModeResetModePick` 을 사용하고 있다.

관련 흐름은 다음과 같다.

- Graph 섹션에서 `s` 를 누르면 reset preview 경로를 탄다.
- preview 이후 `ModeResetModePick` 으로 들어간다.
- `handleResetModePickKey()` 에서 `s / m / h` 를 누르면 `ResetMode` 만 바뀌고, `enter` 또는 `space` 를 눌러야 실제 실행이 시작된다.
- 실행 시에는 `loadingToast(strings.Title(string(mode)) + " reset...")` 가 표시된다.
- `renderConfirmPopup()` 는 modal title, body detail, footer help text 를 따로 렌더링한다.
- `ModeResetModePick` 의 detail 문자열과 footer help text 에 같은 mode label 이 반복되면서 중복 노출이 생긴다.

현재 문제를 더 구체적으로 보면 다음과 같다.

1. 사용자는 `soft / mixed / hard` 를 읽고 한번 더 `enter` 를 눌러야 한다.
2. modal 본문과 footer, 그리고 일부 상태 텍스트가 같은 label 을 중복으로 보여준다.
3. reset confirm 이 “선택형”인지 “즉시 실행형”인지 시각적으로 애매하다.

## 핵심 판단

### 1. reset mode picker 는 사실상 실행 버튼이 아니라 즉시 실행 스위치여야 한다

`reset` 은 선택한 mode 가 곧 실행 의미를 갖는다.

따라서 이번 계획에서는 `s / m / h` 를 “모드 선택”이 아니라 “즉시 실행 트리거”로 본다.

권장 해석은 다음과 같다.

- `s`: soft reset 실행
- `m`: mixed reset 실행
- `h`: hard reset 실행

이 상태에서는 `enter` 가 더 이상 필수 입력이 아니어야 한다.

### 2. `reset confirm` 의 레이블은 한 군데만 보여야 한다

현재 중복은 크게 세 위치에서 생긴다.

- modal title
- modal body detail
- modal footer help text

reset mode picker 에서는 세 개를 모두 풍성하게 보여줄 필요가 없다.
특히 mode label (`soft / mixed / hard`) 은 한 번만 노출되어야 한다.

권장 원칙은 다음과 같다.

- modal title 은 컨텍스트만 보여준다
- modal body 는 짧은 설명만 보여준다
- mode 목록은 한 번만 보여준다
- footer help 는 `esc` 같은 취소 동작만 남기거나, 아예 비운다

### 3. reset 결과 toast 는 기존 공통 progress toast 를 그대로 쓴다

mode 선택 직후에는 바로 실행되므로, 실행 중 상태는 이미 존재하는 공통 progress toast 로 보여주면 된다.

예시 문구:

- `Soft reset...`
- `Mixed reset...`
- `Hard reset...`

이 toast 는 짧게 유지하고, 실행 완료 후에는 `deriveStatus()` 로 복귀한다.

## 목표 UX

### reset 흐름

1. Graph 섹션에서 reset target 을 고른다.
2. `reset mode` modal 이 열린다.
3. 사용자는 `s / m / h` 중 하나를 누른다.
4. 누른 즉시 해당 mode 로 reset 이 실행된다.
5. `enter` 는 더 이상 실행 트리거로 쓰지 않는다.
6. `esc` 는 modal 을 닫고 이전 browse 상태로 복귀시킨다.
7. 실행 중에는 짧은 progress toast 가 보인다.
8. 실행 완료 후 repo status 가 refresh 된다.

### 화면상 기대 모습

- modal 은 짧아야 한다
- mode 를 읽고 바로 선택할 수 있어야 한다
- 선택 이후 별도 확인 동작이 없어야 한다
- label 은 한 번만 보이고 반복되지 않아야 한다

## 범위

### 포함

- `internal/app/key_handling_reset.go`
- `internal/app/view_shell.go`
- `internal/app/view_sections.go`
- `internal/state/state.go`
- `internal/app/model_test.go`
- `internal/app/key_handling_test.go`
- `internal/app/progress_test.go`
- `internal/app/preview_test.go`

### 제외

- reset Git command 자체의 의미 변경
- soft / mixed / hard 의 Git 매핑 변경
- stash 기능 확장
- reset preview 알고리즘 변경
- 다른 confirm flow 의 전반적 구조 변경

## 상태 모델 방향

이번 작업은 새로운 상태 타입을 만들 필요가 없다.

현재 상태 모델을 유지하되, `ModeResetModePick` 의 의미만 더 명확히 하면 된다.

### `ModeResetModePick` 의 역할

`ModeResetModePick` 은 다음 의미를 가진다.

- reset target 은 이미 정해졌다
- 사용자는 mode 만 고르면 된다
- 선택 즉시 실행한다

따라서 이 모드는 더 이상 “다시 확인해서 실행하는 confirm” 이 아니라, “즉시 실행형 mode picker” 로 해석해야 한다.

### `ResetMode`

현재 `state.ResetMode` 는 그대로 쓴다.

- `ResetModeSoft`
- `ResetModeMixed`
- `ResetModeHard`

`ResetMode` 는 선택된 실행 모드를 기억하는 값이고, `enter` 의 대기 상태를 의미하지 않는다.

## 구현 계약

### 1. 키 입력 계약

`handleResetModePickKey()` 의 계약을 다음처럼 바꾼다.

- `s` 를 누르면 soft reset 을 즉시 실행한다
- `m` 을 누르면 mixed reset 을 즉시 실행한다
- `h` 를 누르면 hard reset 을 즉시 실행한다
- `esc` 는 취소하고 이전 상태로 복귀한다
- `enter` 는 reset 실행 트리거로 사용하지 않는다
- `enter` 예외나 fallback 은 두지 않는다

### 2. 상태 전이 계약

mode 선택 직후의 전이는 아래 순서를 따른다.

1. `ResetMode` 를 선택한다
2. `loadingToast("<Mode> reset...")` 로 전환한다
3. `executeReset()` 를 실행한다
4. 실행 결과가 오면 `deriveStatus()` 로 복귀한다

여기서 `loadingToast` 는 너무 길지 않아야 한다.

권장 문구:

- `Soft reset...`
- `Mixed reset...`
- `Hard reset...`

### 3. 문구 계약

reset mode picker 에서는 문구가 중복되지 않도록 다음처럼 분리한다.

- title: `Reset mode`
- message: `Choose a reset mode.`
- detail: 사용하지 않는다
- footer help: `esc: back`
- mode list: `s: soft  •  m: mixed  •  h: hard`

reset modal 의 최종 표준 배치는 아래 한 가지로 고정한다.

예시 구조:

```text
Reset mode
Choose a reset mode.

s: soft  •  m: mixed  •  h: hard

esc: back
```

중요한 점은 `s / m / h` 가 detail 과 footer, global panel 에 동시에 반복되지 않도록 하는 것이다.
mode 목록은 위 한 줄만 사용하고, 세 줄 버전은 사용하지 않는다.

## UI 배치 원칙

### reset modal

- modal 은 `ModeResetModePick` 전용으로 렌더링한다
- generic confirm 과 같은 footer 규칙을 그대로 재사용하지 않아도 된다
- reset modal 은 action confirm 보다 mode picker 성격이 강하므로, footer 문구를 최소화한다
- reset modal 에서는 detail 영역을 렌더링하지 않는다

### global / context panel

- modal 이 열려 있을 때 global panel 에서 동일한 shortcut 을 다시 반복하지 않는다
- status compact view 에서는 `Reset` 만 보여준다
- 사용자가 현재 modal 안에서 바로 판단할 수 있는 정보만 남긴다

## 파일별 구현 가이드

### `internal/app/key_handling_reset.go`

가장 중요한 변경 지점이다.

현재는 `s / m / h` 가 `ResetMode` 만 바꾸고, `enter` 또는 `space` 에서 실행한다.
이 로직을 다음처럼 바꾼다.

- `s / m / h` 입력 시 해당 mode 를 선택하면서 즉시 `executeReset()` 를 반환한다
- 선택 즉시 progress toast 로 넘어가야 한다
- `enter` 는 더 이상 reset execution path 가 아니다
- `esc` 는 기존처럼 취소한다

여기서 필요한 추가 정보는 `m.status.Selected` 의 target 이다.
이 값은 preview 단계에서 이미 선택되어 있으므로 그대로 사용한다.

### `internal/app/view_shell.go`

reset modal 렌더링은 여기서 조정한다.

핵심은 `ModeResetModePick` 일 때 generic confirm footer 를 그대로 쓰지 않는 것이다.

필요한 조정:

- reset mode picker 전용 render helper 분리
- helper 내부에서 mode label 을 한 번만 출력
- generic confirm 의 `y: yes / n: no` 문구는 reset mode picker 에서는 쓰지 않음
- footer 는 `esc: back` 만 남기거나, footer 영역을 비워도 된다
- global panel 에서는 `ModeResetModePick` 를 설명하는 반복 문구를 최소화하거나 숨긴다

### `internal/app/view_sections.go`

현재 상태를 compact 하게 보여주는 문구도 점검해야 한다.

특히 `renderStatusCompact()` 가 `ModeResetModePick` 에서 너무 많은 정보를 다시 말하지 않도록 해야 한다.

권장 방향:

- global panel 은 `Reset` 만 남긴다
- mode label 목록은 modal 안에서만 한 번 보여준다
- `soft / mixed / hard` 는 global panel 에서 반복하지 않는다

### `internal/state/state.go`

상태 타입을 새로 만들 필요는 없지만, `ModeResetModePick` 의 의미를 문서와 코드 주석으로 분명히 남기는 것이 좋다.

필요하면 다음 수준의 주석을 추가한다.

- `ModeResetModePick` 는 confirm 이후의 즉시 실행형 picker 다
- `enter` 를 요구하지 않는다

### `internal/app/progress.go`

reset 실행 시 progress toast 문구를 공통 helper 기준으로 유지한다.

이 문구는 길지 않아야 한다.
이미 있는 `loadingToast()` 패턴을 그대로 사용하되, `Reset` 에서만 짧은 동사형 문장을 넣는다.

## 테스트 전략

이 변경은 문자열, 키 입력, 상태 전이가 모두 바뀌므로 테스트를 함께 고정해야 한다.

### 1. key handling tests

검증해야 할 항목:

- `s` 입력 시 `ModeLoading` 으로 바로 전환되는지
- `m` 입력 시 `ModeLoading` 으로 바로 전환되는지
- `h` 입력 시 `ModeLoading` 으로 바로 전환되는지
- `enter` 가 reset execution 에 필요하지 않은지
- `esc` 가 이전 browse 상태로 돌아가는지

### 2. view tests

검증해야 할 항목:

- `ModeResetModePick` 에서 mode label 이 두 번 이상 중복되지 않는지
- modal body 와 footer 가 같은 shortcut 을 반복하지 않는지
- confirm popup 이 짧고 읽기 쉬운지

### 3. preview / status tests

검증해야 할 항목:

- reset preview 의 detail 이 mode label 을 과도하게 반복하지 않는지
- `Choose a reset mode.` 와 `Soft reset.` 같은 문구가 역할을 혼동하지 않는지
- execution toast 가 reset mode 에 맞는지

## 구현 순서

1. `key_handling_reset.go` 에서 `enter` 의존을 제거하고 mode key 즉시 실행으로 바꾼다.
2. `view_shell.go` 에서 `ModeResetModePick` 전용 modal render helper 를 분리한다.
3. `view_sections.go` 에서 global compact status 문구를 정리한다.
4. `state.Status` 의 `ModeResetModePick` 의미를 코드와 문서에 맞춘다.
5. 관련 테스트를 추가하거나 갱신한다.
6. `scripts/check` 로 회귀를 확인한다.

## 완료 기준

아래 조건을 모두 만족하면 완료로 본다.

- reset mode picker 에서 `s / m / h` 를 누르면 바로 실행된다
- `enter` 를 누르지 않아도 reset 이 수행된다
- modal 안에서 동일한 reset label 이 반복되지 않는다
- generic confirm 과 reset mode picker 의 역할이 분리되어 보인다
- 테스트가 모두 통과한다

## 결론

이번 변경의 핵심은 reset 을 더 간단한 “mode 선택 후 즉시 실행” 흐름으로 바꾸는 것이다.

- `reset confirm` 은 더 이상 두 번 확인하는 흐름이 아니다
- `soft / mixed / hard` 는 선택 옵션이자 즉시 실행 트리거다
- label 중복은 modal body / footer / compact status 의 역할을 분리해서 해결한다
- 문구는 짧게, 동작은 즉시, 취소는 `esc` 로만 남긴다
