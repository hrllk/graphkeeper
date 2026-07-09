# Graph Stash Pop Plan

## 목적

`Graph` 섹션에서 stash 가 걸린 `HEAD` 포인터를 대상으로 `stash pop` 을 실행하는 흐름을 정의한다.

이 문서는 `Graph Actions` 에서 제공되는 pop 진입점, 활성 조건, 선택 UI, confirm 흐름만 다룬다.
실제 pop 실행과 refresh 는 별도 실행 계층으로 남겨 둔다.

## 사용자 목표

1. 사용자는 `Graph` 에서 stash 가 붙은 `HEAD` 를 바로 pop 대상으로 볼 수 있어야 한다.
2. 동일한 commit 에 stash 가 여러 개면, 사용자가 먼저 어떤 stash 를 pop 할지 고를 수 있어야 한다.
3. stash 가 하나뿐이면, 사용자는 confirm 을 통해 실제 pop 여부를 한 번 더 확인할 수 있어야 한다.
4. pop 은 `Graph` 문맥 안에서 끝나야 하고, 다른 섹션의 stash list 흐름과 섞이면 안 된다.

## 범위

### 포함

- `Graph Actions` 의 pop hotkey 노출
- pop 활성 조건 판정
- stash 1개일 때의 confirm modal
- stash 여러 개일 때의 overlay picker
- 선택된 stash 를 실제 pop 실행기로 넘기는 계약
- 성공 / 실패 / stale 상태에 대한 후속 표시

### 제외

- 전역 stash list popup 재설계
- stash list 의 flat render 규칙 변경
- stash 생성 / edit / drop
- branch continuation 흐름
- Graph 밖에서의 pop 진입점 추가

## 핵심 전제

`Graph` 에서의 pop 은 아무 stash 나 대상으로 삼지 않는다.

이 기능은 다음 두 조건이 동시에 만족될 때만 활성화한다.

1. 현재 포커스가 `HEAD` 포인터여야 한다.
2. 그 `HEAD` 에 연결된 stash 가 하나 이상 있어야 한다.

즉, `Graph` 에서 stash 존재를 읽는 것과, 그 stash 를 실제로 pop 하는 것은 다른 수준의 동작이다.
이 문서는 후자를 다룬다.

## 활성 조건

### 1. 포커스 조건

- `Graph` 에서 현재 포커스된 row 가 `HEAD` 이어야 한다.
- 단순히 stash 가 붙은 commit 이라는 이유만으로는 활성화하지 않는다.
- `HEAD` 가 아닌 과거 commit 에 붙은 stash 는 이 진입점의 대상이 아니다.

### 2. stash 조건

- `stashesForCommit(HEAD)` 결과가 비어 있으면 비활성화한다.
- stash 가 하나 이상 있으면 `Graph Actions` 에 pop 항목을 보여준다.

### 3. 토폴로지 조건

- `Graph` pop 은 branch 라우팅이나 lane 이동과 무관하게, 현재 `HEAD` 상태만 본다.
- 즉, pop 가능 여부는 local lane 판정보다 더 좁게 잡는다.

## UX 흐름

### stash 가 0개일 때

- pop hotkey 를 노출하지 않거나, 비활성 상태로만 보여 준다.
- 사용자는 아무런 실행 동작을 시작할 수 없다.

### stash 가 1개일 때

- 바로 confirm modal 로 들어간다.
- confirm 에는 대상 stash 의 식별 정보가 보여야 한다.
- 사용자가 `enter` 로 확정해야 실제 pop 이 진행된다.
- `esc` 는 dismiss 로 끝난다.

### stash 가 여러 개일 때

- 먼저 overlay picker 를 띄운다.
- picker 는 같은 `HEAD` 에 연결된 stash 들만 보여 준다.
- 사용자는 여기서 pop 할 stash 를 고른다.
- 선택 후에는 공통 confirm 단계로 넘겨 실제 pop 여부를 한 번 더 확인한다.

## 표시 원칙

- picker 는 `Graph` 맥락에서 읽히는 좁고 빠른 선택 UI 여야 한다.
- 각 항목은 `Ref`, 7자리 hash, truncated subject 정도만 보여도 충분하다.
- `BaseHash` 나 내부 연결 정보는 선택 근거로만 유지하고, UI 에는 노출하지 않는다.
- `Graph Actions` 의 도움말은 활성 여부에 맞게만 갱신한다.

## Graph Actions 노출

- `Graph` 섹션의 액션 도움말에 pop hotkey 를 추가한다.
- 활성 조건이 맞지 않으면 disabled 상태로 보이거나 아예 숨긴다.
- hotkey 는 기존 `Graph` 단축키와 충돌하지 않아야 한다.
- 이 문서에서는 키를 특정하지 않고, 구현 시점에 가장 덜 충돌하는 키를 선택한다.

## 실행 계약

- picker 에서 선택된 stash 는 실제 pop executor 로 전달한다.
- confirm 은 선택된 stash 에 대해서만 열린다.
- pop 이 끝나면 repo state 를 다시 읽어 `Graph` 와 stash summary 가 최신 상태가 되도록 한다.
- stale stash, conflict, 이미 사라진 stash 는 실패 이유를 사용자에게 드러내야 한다.

## 테스트

- `HEAD` 가 아니면 pop 이 비활성화되는지
- `HEAD` 에 stash 가 없으면 pop 이 비활성화되는지
- stash 가 1개면 picker 없이 confirm 으로 가는지
- stash 가 여러 개면 picker 가 열리는지
- picker 에서 고른 대상이 confirm 에 정확히 이어지는지
- `enter` 는 실제 pop 경로만 실행하는지
- `esc` 는 picker 와 confirm 모두에서 dismiss 로 끝나는지
- pop 후 `Graph` 와 stash 상태가 갱신되는지

## 구현 순서

1. `Graph` focus 에서 `HEAD + stash` 활성 조건을 계산한다.
2. `Graph Actions` 에 pop 진입점을 연결한다.
3. stash 1개 / 다중 stash 분기용 UI 를 만든다.
4. confirm 이후 실제 pop executor 와 refresh 를 연결한다.
5. 관련 테스트를 추가한다.

## 완료 기준

- `Graph` 에서 `HEAD` + stash 조건일 때만 pop 이 보인다.
- stash 1개와 여러 개 케이스가 각각 기대한 UI 로 분기된다.
- 사용자는 실수로 다른 commit 의 stash 를 pop 하지 않는다.
- pop 이후 상태가 다시 동기화된다.
