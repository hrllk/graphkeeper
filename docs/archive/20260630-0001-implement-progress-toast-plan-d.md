# Confirm 실행 Toast 계획서

## 목적

이 문서는 `confirm` 창에서 사용자가 승인을 누른 뒤 실제 Git 작업이 실행되는 동안,
짧고 일관된 `toast`로 "작업 중이니 잠시 기다려달라"는 경험을 제공하기 위한 계획서다.

핵심 목표는 다음과 같다.

1. `confirm` 승인 직후 실행 중임을 알리는 공통 `toast`를 둔다.
2. 작업별로 제각각 다른 로딩 문구를 정리한다.
3. `pull`, `fetch`, `push`, `merge`, `rebase`가 같은 기다림 경험을 따르도록 한다.
4. `toast`는 짧게, 결과는 별도 완료 상태로 분리한다.

## 참조 문서

이 계획은 다음 문서를 기준으로 정리한다.

- `docs/product-prd.md`
- `docs/roadmap.md`
- `docs/model-refactor-plan.md`
- `docs/archive/20260625-0015-feature-pull-reset-ux-implementation-plan.md`
- `docs/archive/20260625-0016-feature-reset-stash-plan.md`
- `docs/archive/20260625-0018-refactor-alert-messages-english-concise-plan.md`
- `docs/structure.md`

## 현재 관찰된 구현 상태

현재 코드에는 작업별 진행 상태가 이미 존재하지만, 공통 추상화는 아직 약하다.

- `fetch`, `pull`, `push`는 각자 별도 command와 update 흐름을 가진다.
- `pull`은 `fetch -> analysis -> confirm -> execute` 흐름을 탄다.
- `push`는 `fetch before push` 이후 confirm 또는 execute로 간다.
- `merge`, `rebase`는 target pick / preview / confirm 흐름을 가진다.
- `state.Status`에는 `ModeLoading`, `ModeConfirm`, `ModeOutcomePreview`, `ModeBlocked`가 이미 있다.
- `internal/app/view_shell.go`는 modal popup 형태로 진행 문구를 보여줄 수 있다.
- `internal/app/update_fetch.go`는 async 결과를 받아 상태를 갱신하는 중심 지점이다.

즉, 현재 문제는 진행 상태가 없는 것이 아니라, `confirm` 승인 후 보여줄 짧은 실행 안내가 작업별로 제각각이라는 점이다.

## 핵심 판단

### 1. toast는 confirm 승인 직후의 실행 안내여야 한다

이 문서에서 말하는 `toast`는 작업 완료 배너가 아니다.

다음 정보를 한 곳에서 다루는 공통 상태로 본다.

- 사용자가 승인했는지
- 지금 실제 작업이 실행 중인지
- 잠시 기다려야 하는지
- 완료되면 어떤 상태로 돌아갈지

즉, `toast`는 `confirm`과 `execute` 사이에 들어가는 아주 짧은 안내 계층이다.

### 2. progress bar는 이번 범위에 넣지 않는다

- 현재 원하는 경험은 길게 진행률을 보여주는 것이 아니라, 사용자가 승인한 작업이 실제로 동작 중임을 짧게 알려주는 것이다.
- 따라서 progress bar는 이번 계획의 대상이 아니다.
- 필요한 경우 나중에 별도 문서로 분리한다.

### 3. 작업별 예외는 toast 문구에서만 최소화한다

`pull`, `fetch`, `push`, `merge`, `rebase` 각각의 특수 처리를 화면 코드에 흩뿌리지 않는다.

대신 공통 규칙을 먼저 고정한다.

- 승인 직후에는 `Loading` 계열 상태로 들어간다.
- toast는 `작업명 + 잠시 기다려달라` 정도로 짧게 유지한다.
- 결과가 오면 `Browse`, `Blocked`, `OutcomePreview`, `Confirm` 중 하나로 수렴한다.
- 실패 메시지는 작업별로 달라도 되지만, toast 자체의 표시 방식은 같아야 한다.

## 범위

### 포함

- `fetch`
- `pull`
- `push`
- `force push`
- `merge`
- `rebase`
- 위 작업들의 confirm 승인 직후 실행 안내
- 공통 toast 문구
- 상태 전이와 렌더링 경계 정리

### 제외

- Git command 자체의 의미 변경
- merge/rebase 알고리즘 변경
- 새 Git 동작 추가
- stash/reset 세부 UX 재설계
- 별도 notifications framework 도입
- progress bar 도입

## 상태 모델 방향

현재 `state.Status`는 이미 UI 상태를 담고 있다. 이번 작업은 새 시스템을 크게 만드는 것보다, 공통 진행 상태를 흡수할 수 있게 정리하는 쪽이 맞다.

권장 방향은 다음과 같다.

### 1. 공통 toast 상태를 명시한다

`ModeLoading` 하나만으로 충분하다. 새 `ToastState` 구조체는 만들지 말고, 기존 `state.Status.Title/Message/Detail` 조합으로 공통 toast 를 표현한다.

예시:

```go
func loadingToast(title, detail string) state.Status {
	return state.New().WithLoading(title).WithDetail(detail)
}
```

### 2. 상태 전이용 helper를 하나의 경계로 둔다

작업별 코드가 `Loading`, `Confirm`, `Blocked`를 직접 조립하지 않도록 한다.

예시 책임 분리:

- `progress.go`: 공통 toast 상태 생성
- `messages.go`: async message 타입
- `update_fetch.go`: 작업 결과 수신
- `view_shell.go`: toast 렌더링

추천 helper 이름:

- `loadingToast(action state.Action) state.Status`
- `confirmToast(action state.Action, title, detail string) state.Status`
- `finishActionStatus(rs git.Status) state.Status`

### 3. 상태는 작업 이름 중심으로 통일한다

작업별 문구를 매번 새로 만들지 말고, 작업 이름과 상태 단계를 조합한다.

예시:

- `Fetching... Please wait.`
- `Pulling... Please wait.`
- `Pushing... Please wait.`
- `Merging... Please wait.`
- `Rebasing... Please wait.`

## UX 원칙

### 1. 화면 문구는 짧아야 한다

긴 설명은 modal confirm에서만 허용한다.

toast는 아래 수준을 넘지 않는 것이 좋다.

- 현재 작업
- 잠시 기다려달라는 짧은 안내
- 필요한 경우 한 줄의 보조 설명

### 2. 진행 중 상태는 사용자 조작을 과하게 막지 않는다

진행 중이라고 해서 전체 UI를 불필요하게 잠그지 않는다.

권장 원칙:

- 실제 위험이 있는 입력만 막는다
- 나머지 탐색은 가능하면 유지한다
- 작업이 끝나면 이전 browsing context 로 복귀한다

### 3. 공통 문구는 영어로 맞춘다

기존 문서와 현재 구현 경향을 고려하면, 사용자 노출 문구는 영어로 통일하는 편이 낫다.

예시:

- `Fetching... Please wait.`
- `Pulling... Please wait.`
- `Pushing... Please wait.`
- `Merging... Please wait.`
- `Rebasing... Please wait.`

## 구현 방향

### fetch

- 승인 직후 `Fetching... Please wait.` toast를 보여준다.
- fetch 성공 시에는 현재 상태를 갱신하고 toast를 정리한다.
- fetch 실패 시에는 공통 blocked state 로 수렴한다.

### pull

- 승인 직후 `Pulling... Please wait.` toast를 보여준다.
- `fetch before pull` 단계와 `pull analysis` 단계는 기존 흐름을 유지한다.
- fast-forward 여부는 progress 결과에 따라 분기한다.
- merge/rebase 선택은 `pull` 전용 confirm으로 남기되 표시 방식은 공통화한다.

### push

- 승인 직후 `Pushing... Please wait.` toast를 보여준다.
- `fetch before push`와 실제 push는 기존 흐름을 유지하되, 사용자에게는 하나의 실행 안내로 보이게 한다.
- no-upstream, non-fast-forward 같은 실패는 동일한 표기 규칙으로 보여준다.

### merge / rebase

- 승인 직후 `Merging... Please wait.` 또는 `Rebasing... Please wait.` toast를 보여준다.
- target preview와 실제 실행 전환은 기존 흐름을 유지한다.
- confirm에서는 상세를 보여주고, execute 단계에서는 짧은 toast만 보여준다.

### 상태 전이 규칙

실제 구현은 아래 규칙으로 단순화한다.

1. confirm 승인 직후 `WithLoading(...)` 으로 전환한다.
2. 실제 Git 명령이 끝나면 `Status()` 를 다시 읽는다.
3. 결과가 성공이면 `deriveStatus(rs)` 로 복귀한다.
4. 결과가 실패이면 `WithError(...)` 또는 `WithBlocked(...)` 로 수렴한다.

```go
m.status = state.New().WithLoading("Pushing...")
return m, executePush(m.repo, branch, m.commitLimit)
```

## 권장 파일 경계

현재 구조를 크게 흔들지 않으면서 정리하는 방향이 좋다.

권장 분리 후보는 다음과 같다.

- `internal/app/progress.go`
- `internal/app/view_progress.go`
- `internal/app/messages.go`

이미 존재하는 파일이 과밀하면 그때만 분리한다.

무조건 파일을 늘리는 것이 목표가 아니라, 공통 흐름이 드러나게 만드는 것이 목표다.

## 구현 순서

1. 현재 작업별 `Loading` 문구를 전수 조사한다.
2. `fetch / pull / push / merge / rebase`의 승인 후 실행 문구를 표준화한다.
3. 공통 toast 상태 타입과 helper를 만든다.
4. `update_fetch.go`와 `update_execute.go`의 분산 처리를 하나의 정책으로 묶는다.
5. `view_shell.go`에 공통 렌더링 경계를 추가한다.
6. `state.Status`와 modal confirm 문구를 정리한다.
7. 관련 테스트를 추가한다.

## 테스트 전략

문자열 정리와 상태 전이는 회귀가 쉽게 생기므로, 테스트를 함께 고정해야 한다.

우선 확인할 항목은 다음과 같다.

- `Loading` 상태가 작업별로 올바르게 설정되는지
- `confirm` 승인 직후 toast가 표시되는지
- 실패 시 `Blocked`로 일관되게 전환되는지
- `pull`의 `fetch -> analysis -> confirm` 흐름이 유지되는지
- `push`의 upstream / non-fast-forward 처리 경로가 깨지지 않는지
- `merge` / `rebase`의 preview 문구가 공통 규칙을 따르는지

권장 테스트 범위:

- `internal/app/actions_test.go`
- `internal/app/preview_test.go`
- `internal/app/key_handling_test.go`
- `internal/app/model_test.go`
- 공통 toast helper를 추가하면 전용 테스트 파일

최소 테스트 케이스:

- `confirm` 승인 직후 `ModeLoading` 으로 바뀌는가
- `fetch`/`push` 성공 후 browse 상태로 복귀하는가
- 실패 시 `WithBlocked`/`WithError` 로 수렴하는가
- `pull` 에서 `fetch -> analysis -> confirm -> execute` 순서가 유지되는가
- `merge`/`rebase` preview 문구가 기존 상세와 충돌하지 않는가

## 검증

```sh
go test ./internal/app
go test ./...
go build ./cmd/graphkeeper
```

## 비고

- 이 문서는 구현 세부를 고정하는 문서가 아니라, `confirm` 승인 후 실행 중이라는 짧은 경험을 공통화하는 문서다.
- toast는 오래 머무는 상태가 아니라, 작업 시작을 알려주는 짧은 안내여야 한다.
- 실제 구현 시에는 `docs/model-refactor-plan.md`와 함께 보면서 책임 경계를 맞추는 것이 좋다.
