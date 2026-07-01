# Checkout / Branch Create Graph Focus Plan

## 목적

`Graph` 또는 `Local` 섹션에서 `checkout` 또는 `new branch` 를 실행한 뒤, 사용자의 시선을 즉시 새 기준점으로 이동시키는 UX 로 정리한다.

이번 계획의 핵심은 다음 두 가지다.

1. 실행 성공 후 포커스를 `Graph` 섹션으로 일관되게 이동한다.
2. 포커스 이동 시 단순히 첫 번째 row 로 가지 않고, 실제 `HEAD` 가 가리키는 row 를 선택한다.

즉, 이번 작업은 실행 성공 후 "어디에 남을 것인가"를 명확히 고정하는 UX 정리 작업이다.

## 참조 문서

- `docs/archive/20260630-0008-branch-create-name-input-immediate-checkout-plan.md`
- `docs/archive/20260701-0002-local-pane-split-layout-plan.md`
- `docs/archive/20260701-0004-local-stash-cleanup-plan.md`
- `docs/20260701-0005-feature-graph-section-merge-rebase-gate-note.md`
- `docs/structure.md`

## 현재 관찰된 구현 상태

현재 구현은 실행 성공 후 상태 갱신은 되지만, 포커스 복귀 정책이 액션마다 일관되지 않다.

- `checkout` 성공 시 `commitLimit = 0` 으로 초기화한다.
- 다만 `Graph` cursor 는 무조건 첫 번째 row 로 이동한다.
- `checkout` 성공 시 `activeSection` 을 명시적으로 `Graph` 로 옮기지 않는다.
- `new branch` 성공 시 상태만 갱신하고 toast 를 띄운다.
- `new branch` 성공 시에도 `activeSection` 을 `Graph` 로 옮기지 않는다.
- `Graph` 에는 이미 `H` 키로 `HEAD` row 로 점프하는 동작이 존재한다.

즉, 필요한 기반은 이미 있지만, 성공 후 공통 복귀 정책으로 묶여 있지 않다.

## 핵심 판단

### 1. 성공 후 기준 화면은 Graph 로 고정한다

`checkout` 과 `new branch` 는 결국 브랜치 기준점이 바뀌는 동작이다.

실행 직후 사용자가 가장 확인하고 싶은 것은 아래 둘이다.

- 새 `HEAD` 가 어디로 이동했는가
- 그래프 상에서 어떤 commit 이 현재 기준점인가

이 정보는 `Graph` 섹션이 가장 명확하게 보여준다.

따라서 성공 후 기본 포커스는 `Graph` 로 고정하는 것이 맞다.

### 2. row 0 복귀보다 HEAD 복귀가 더 정확하다

현재 `checkout` 성공 시 row 0 으로 이동하는 방식은 구현은 단순하지만 의미가 약하다.

- row 0 이 항상 사용자가 확인해야 하는 row 라는 보장이 없다.
- commit 정렬이나 graph 구성 방식이 바뀌면 의도와 다른 위치가 될 수 있다.
- 이미 코드에는 `HEAD` hash 로 row 를 찾는 helper 가 존재한다.

따라서 복귀 기준은 "첫 row" 가 아니라 "`repoStatus.Head` row" 여야 한다.

### 3. branch create 는 checkout 과 같은 복귀 계약을 써야 한다

현재 branch create 는 `git switch -c` 로 동작한다.

즉, branch create 성공은 곧 새 branch checkout 성공과 동일한 의미다.

따라서 아래 UX 는 별도로 갈라지면 안 된다.

- Graph 에서 checkout 성공 후 복귀
- Local 에서 checkout 성공 후 복귀
- Graph 에서 branch create 성공 후 복귀
- Local 에서 branch create 성공 후 복귀

모두 같은 helper / 같은 계약으로 묶는 것이 맞다.

### 4. HEAD row 가 없을 때는 안전한 fallback 이 필요하다

정상 상태에서는 `repoStatus.Head` 가 graph row 에 존재해야 한다.
하지만 비정상 상태나 일시적 상태에서는 못 찾을 가능성도 있다.

따라서 fallback 은 다음 순서가 적절하다.

1. `HEAD` row 를 찾으면 그 row 로 이동
2. 못 찾으면 기존 row 0 으로 이동
3. row 자체가 없으면 cursor / scroll 을 0 으로 정리

즉, 목표 UX 는 `HEAD` 우선이지만, 실패 시 기존 동작보다 불안정해지면 안 된다.

## 목표 UX

### checkout 성공 후

1. 사용자가 `Graph` 또는 `Local` 에서 checkout 을 실행한다.
2. 성공 후 그래프 데이터가 새 상태로 갱신된다.
3. 포커스가 `Graph` 섹션으로 이동한다.
4. 선택 row 가 현재 `HEAD` commit row 로 이동한다.
5. lane cursor 도 해당 row 의 pointer lane 으로 맞춰진다.

### branch create 성공 후

1. 사용자가 `Graph` 또는 `Local` 에서 branch create 를 실행한다.
2. 성공 후 새 branch 로 checkout 된 상태가 반영된다.
3. toast 는 유지하되 포커스 기준은 `Graph` 로 이동한다.
4. 선택 row 는 새 `HEAD` row 로 이동한다.

## 범위

### 포함

- `internal/app/update_execute.go`
- `internal/app/update_branch.go`
- `internal/app/navigation_graph.go`
- `internal/app/model_test.go`
- `internal/app/key_handling_test.go`

### 제외

- checkout confirm 문구 변경
- branch create input UX 변경
- graph 렌더링 규칙 변경
- `HEAD` jump 단축키 자체의 재설계
- 성공 toast 문구 변경

## 구현 방향

### 1. Graph 복귀 helper 도입

성공 후 복귀 정책은 공통 helper 로 묶는 것이 맞다.

권장 책임은 다음과 같다.

- `activeSection = sectionGraph`
- `commitLimit = 0` 이 필요한 경우 함께 초기화
- `HEAD` hash 로 graph row 조회
- `sectionCursor[sectionGraph]` 설정
- `graphScroll` 을 page 크기에 맞게 보정
- `graphLaneCursor` 를 선택 row 기준으로 동기화

helper 이름 예시는 다음 수준이면 충분하다.

- `focusGraphHead`
- `focusGraphHeadRow`
- `resetGraphToHead`

핵심은 "Graph + HEAD + cursor/scroll/lane 동기화" 책임이 한 군데에 있어야 한다는 점이다.

### 2. checkout 성공 경로를 helper 기준으로 교체

현재 `checkout` 성공 경로는 row 0 기준 초기화 로직을 직접 갖고 있다.

이 부분은 다음처럼 바꿔야 한다.

- 기존 row 0 고정 로직 제거
- 상태 갱신 이후 공통 helper 호출
- `activeSection` 을 명시적으로 `Graph` 로 전환

주의점은 `syncBrowseState` 가 cursor 에 영향을 주는지 확인하는 것이다.
cursor 세팅이 덮어써지지 않도록 호출 순서를 고정해야 한다.

권장 순서는 아래 두 가지 중 하나다.

1. `syncBrowseState` 후 `focusGraphHead`
2. 또는 `syncBrowseState` 내부 계약을 확인한 뒤 안전한 순서로 helper 호출

이번 구현에서는 테스트로 호출 순서를 잠그는 것이 중요하다.

### 3. branch create 성공 경로도 동일 helper 사용

`createdBranchMsg` 성공 시에도 동일 helper 를 호출한다.

이 경로는 현재 toast 중심으로만 끝나므로, 다음이 추가되어야 한다.

- `repoStatus` 반영
- browse state 동기화
- `Graph` 포커스 이동
- `HEAD` row 선택
- 기존 success toast 유지

즉, 화면 모드는 toast 를 유지하더라도 포커스 기준 데이터는 이미 `Graph` 기준으로 정렬돼 있어야 한다.

### 4. HEAD row 진입 가능 여부를 명시적으로 검증

사용자 요구는 "그래프 섹션 포커스"에서 끝나지 않고, "HEAD 진입 가능 여부 확인"까지 포함한다.

여기서 진입 가능 여부는 아래를 뜻한다.

- 현재 선택 row 가 실제 `HEAD` row 여야 한다
- lane cursor 가 해당 row 의 유효 lane 을 가리켜야 한다
- 이후 browse 조작이 바로 이어져야 한다

즉, 단순히 `activeSection` 만 바꾸는 것으로는 충분하지 않다.

## 테스트 방향

### 1. checkout 성공 테스트 보강

기존 `TestCheckoutResetsGraphLoadState` 성격의 테스트를 다음 계약으로 갱신한다.

- `commitLimit = 0`
- `activeSection == sectionGraph`
- `sectionCursor[sectionGraph] == HEAD row index`
- `graphScroll == clampScroll(HEAD row, total, page)`
- `graphLaneCursor == graph.PointerLane(rows[HEAD row])`

현재 테스트가 row 0 을 기대하고 있다면, 새 계약에 맞게 수정해야 한다.

### 2. branch create 성공 테스트 추가

branch create 성공 후 다음을 검증하는 테스트가 필요하다.

- `activeSection == sectionGraph`
- `repoStatus.Head` 가 새 branch head 로 반영됨
- `sectionCursor[sectionGraph]` 가 해당 `HEAD` row 를 가리킴
- `graphLaneCursor` 가 유효 lane 으로 동기화됨
- success toast 는 기존처럼 유지됨

### 3. fallback 테스트 추가

가능하면 아래 edge case 도 잠그는 편이 좋다.

- `HEAD` 를 row 에서 못 찾을 때 row 0 fallback
- graph row 가 비어 있을 때 panic 없이 0 상태 유지

이 테스트는 helper 단위 또는 update 경로 단위 중 더 작은 단위로 두는 것이 적절하다.

## 구현 순서

1. 현재 checkout / branch create 성공 경로를 공통 관점으로 정리한다.
2. `Graph` `HEAD` 복귀 helper 를 도입한다.
3. `checkout` 성공 경로를 helper 기반으로 교체한다.
4. `branch create` 성공 경로를 helper 기반으로 교체한다.
5. 관련 테스트를 새 계약에 맞게 수정 / 추가한다.
6. `scripts/test` 또는 최소 관련 패키지 테스트로 검증한다.
7. 마지막에 `scripts/check` 로 회귀를 확인한다.

## 완료 기준

다음을 모두 만족하면 완료로 본다.

- `Graph` / `Local` 어디서 checkout 해도 성공 후 `Graph` 로 포커스가 이동한다.
- 선택 row 는 row 0 이 아니라 현재 `HEAD` row 다.
- branch create 성공 후에도 동일 규칙이 적용된다.
- success toast 동작은 깨지지 않는다.
- 관련 테스트가 추가되거나 갱신되고 모두 통과한다.

## 리스크 및 확인 포인트

### 1. syncBrowseState 와 cursor 복원 순서

`syncBrowseState` 가 일부 cursor 상태를 정리할 수 있으므로, helper 호출 순서를 잘못 두면 기대 위치가 다시 덮일 수 있다.

이 부분은 구현 시 가장 먼저 좁게 테스트해야 한다.

### 2. graph page size 의존성

`graphScroll` 은 `graphPageSize` 에 의존한다.
테스트에서 viewport 관련 기본값이 다르면 scroll 기대값이 흔들릴 수 있으므로, 테스트 모델 크기를 명시하거나 scroll 검증을 안정적인 값으로 고정해야 한다.

### 3. toast 와 browse focus 의 공존

branch create 성공 직후는 toast 상태다.
이때도 내부 포커스는 이미 `Graph` 에 맞춰져 있어야 하며, toast 종료 후 사용자가 곧바로 `HEAD` 기준 탐색을 이어갈 수 있어야 한다.
