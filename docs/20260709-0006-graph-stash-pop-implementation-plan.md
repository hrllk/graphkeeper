# Graph Stash Pop Implementation Plan

## 목적

`Graph` 섹션의 `HEAD` 포인터에서 stash pop 을 시작하고, stash 개수에 따라 picker 또는 confirm 으로 분기하는 실제 구현 순서를 정리한다.

이 문서는 설계 문서가 아니라 실행 문서다.
즉, "무엇을 만들 것인가"보다 "어떤 파일을 어떤 순서로 바꿀 것인가"에 집중한다.

## 기준 동작

### 활성 조건

- `Graph` 포커스가 `HEAD` 여야 한다.
- `stashesForCommit(HEAD)` 결과가 1개 이상이어야 한다.
- 조건을 만족하지 않으면 pop 액션을 비활성화한다.

### 분기 규칙

- stash 0개: pop 진입점 비노출 또는 disabled
- stash 1개: confirm modal 로 즉시 진입
- stash 여러 개: picker 열기 -> stash 선택 -> confirm

### hotkey

- `Graph` pop hotkey 는 `o` 를 사용한다.
- 이유:
  - `p` 는 현재 Graph / Current 에서 pull 로 이미 쓰인다.
  - `P` 는 전역 fetch-for-push 에 쓰인다.
  - `o` 는 현재 충돌이 없다.

## 구현 범위

### 수정 대상

- `internal/app/key_handling_browse.go`
- `internal/app/view_sections.go`
- `internal/app/model.go`
- `internal/app/stash_popup.go`
- `internal/app/commands.go`
- `internal/app/view_detail.go`
- `internal/app/view_graph.go`
- `internal/app/stash_state.go`
- `internal/app/model_test.go`
- 필요 시 `internal/git/repo.go`

### 새로 필요한 상태

- `graphStashPopOpen`
- `graphStashPopCursor`
- `graphStashPopTargetHash`
- `graphStashPopEntries`
- `graphStashPopConfirmOpen`
- `graphStashPopSelected`

이미 있는 `stashPopupOpen` 은 전역 stash list 용으로 유지하고, Graph pop 플로우와 섞지 않는다.

## 상태 전이

### 1. Graph key handler

`internal/app/key_handling_browse.go` 의 `sectionGraph` 분기에서 `o` 를 받는다.

- 현재 포커스가 `HEAD` 가 아니면 disabled 상태로 반환
- `HEAD` 에 stash 가 없으면 disabled 상태로 반환
- stash 1개면 confirm 상태로 이동
- stash 여러 개면 picker 상태로 이동

### 2. picker

picker 는 별도 popup 이다.

권장 구현은 기존 stash popup 과 비슷한 렌더러를 재사용하되, 데이터 소스와 action 을 분리하는 것이다.

- 데이터 소스: `m.stashesForCommit(m.repoStatus.Head)`
- 선택 결과: `graphStashPopSelected`
- 종료 조건: `enter` 선택, `esc` dismiss

### 3. confirm

confirm 은 실제 pop 직전의 최종 확인이다.

- 선택된 stash 의 `Ref`, 7자리 hash, subject 를 보여 준다.
- `enter` 로 pop 실행
- `esc` 로 dismiss

### 4. execution

`commands.go` 에 pop executor 를 추가한다.

권장 형태:

```go
func executeStashPop(repo *git.Repo, limit int, entry git.StashEntry) tea.Cmd
```

실행 흐름은 다음과 같다.

1. `git stash pop` 실행
2. `Status()` 재조회
3. 필요하면 `Stashes()` 재조회
4. 성공 / 실패 / conflict 를 `executedMsg` 또는 전용 msg 로 반환

## 파일별 작업 순서

### 1. `internal/app/model.go`

- graph pop 전용 상태 필드를 추가한다.
- 생성자 `New()` 에 기본값을 넣는다.
- 전역 stash popup 과 graph pop popup 이 동시에 열리지 않도록 우선순위를 정한다.

### 2. `internal/app/key_handling_browse.go`

- `sectionGraph` 의 `o` 키를 pop 진입점으로 연결한다.
- 활성 조건을 계산하는 helper 를 추가하거나 기존 helper 를 재사용한다.
- `head` 기준 stash 존재 여부를 먼저 판정한다.

### 3. `internal/app/view_sections.go`

- `Graph Actions` 도움말에 `o: pop` 을 추가한다.
- 조건 불충족 시 disabled 문구를 보여 준다.
- picker/confirm 모드에서는 `enter` / `esc` 도움말만 노출한다.

### 4. `internal/app/stash_popup.go`

- 현재 전역 stash list popup 과 분리해서 graph pop 전용 popup 렌더러를 둔다.
- 하나의 렌더러를 재사용하더라도 title, body, footer 를 분기한다.
- picker 버전은 선택 리스트를, confirm 버전은 선택 stash 상세를 중심으로 보여 준다.

### 5. `internal/app/commands.go`

- `executeStashPop` 을 추가한다.
- pop 이후 refresh 를 수행하고, stale/conflict/error 를 상태로 돌린다.
- 필요하면 pop 결과와 stash 재조회 결과를 함께 묶는다.

### 6. `internal/app/view_detail.go`

- Graph focus 가 `HEAD` 일 때 stash summary 와 pop 가능 여부가 같이 읽히도록 문구를 맞춘다.
- action help 와 detail panel 이 서로 다른 말을 하지 않게 한다.

### 7. `internal/app/view_graph.go`

- stash count 표시가 `HEAD` pop 조건과 충돌하지 않는지 확인한다.
- focus 가 `HEAD` 일 때만 pop 액션이 의미를 갖는다는 점을 유지한다.

### 8. `internal/app/stash_state.go`

- `stashesForCommit(HEAD)` 가 pop 대상 판단에 그대로 쓰일 수 있는지 확인한다.
- 필요하면 `HEAD` 전용 helper 를 추가하되, 그룹 로직은 바꾸지 않는다.

### 9. `internal/app/model_test.go`

- 활성 조건 테스트
- stash 1개 / 여러 개 분기 테스트
- help text 테스트
- enter / esc 동작 테스트
- pop 후 refresh 테스트

## UI 계약

### Picker

- title: `Pop stash`
- body: `Select a stash to pop.`
- list: `Ref`, 7자리 hash, truncated subject
- footer: `enter: choose  •  esc: dismiss`

### Confirm

- title: `Pop stash?`
- body: 선택된 stash 의 식별 정보와 pop 경고
- footer: `enter: pop  •  esc: dismiss`

### Disabled help

- `HEAD` 가 아니면 `o: pop` 을 disabled 로 표시
- stash 가 없으면 `o: pop` 을 숨기거나 disabled 로 표시
- 둘 다 동일한 이유 텍스트로 보여 준다

## 테스트 우선순위

1. `HEAD` 전용 활성 조건
2. stash 1개 / 여러 개 분기
3. Graph Actions 도움말 노출
4. picker -> confirm 전이
5. confirm -> execute -> refresh
6. conflict / stale / missing stash 실패 메시지

## 구현 순서

1. `model` 에 graph pop 상태를 추가한다.
2. `o` 키와 help text 를 연결한다.
3. picker / confirm 렌더러를 추가한다.
4. pop executor 를 추가한다.
5. 상태 전이와 refresh 를 연결한다.
6. 테스트를 추가한다.
7. 필요하면 문구와 정렬을 조정한다.

## 완료 기준

- `Graph` 에서 `HEAD` + stash 조건일 때만 pop 진입점이 보인다.
- stash 1개면 confirm 으로, 여러 개면 picker 로 분기한다.
- pop 후 repo state 가 다시 갱신된다.
- 기존 전역 stash list popup 과 충돌하지 않는다.
