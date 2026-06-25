# `internal/app/navigation.go` 경계 분리 계획서

## 목표

`internal/app/navigation.go`의 책임을 셋으로 나눈다.

- `navigation_graph`
- `navigation_section`
- `state_browse`

설명은 짧게 유지하고, 코드 경계는 명확하게 남긴다.

## 범위

- 대상 파일
  - `internal/app/navigation.go`
- 후보 파일
  - `internal/app/navigation_graph.go`
  - `internal/app/navigation_section.go`
  - `internal/app/state_browse.go`
- 관련 파일
  - `internal/app/model.go`
  - `internal/app/view.go`
  - `internal/app/key_handling_browse.go`
  - `internal/app/target_items.go`
  - `internal/app/preview.go`
  - `internal/app/update.go`
- 관련 테스트
  - `internal/app/model_test.go`
  - `internal/app/key_handling_test.go`
  - `internal/app/target_items_test.go`
- 새 package 생성: 하지 않음

## 참조 문서

이 계획은 다음 문서의 분리 원칙을 이어받는다.

- `docs/archive/202606-0009-refactor-model-structure-plan.md`
- `docs/archive/202606-0010-refactor-model-messages-plan.md`
- `docs/archive/202606-0011-refactor-model-boundary-plan.md`
- `docs/model-refactor-plan.md`

## 현재 문제

`navigation.go`에는 그래프 이동, 섹션 이동, browse 상태 복원이 섞여 있다.
이 상태는 읽기 어렵고, 다음 수정에서 영향 범위를 넓힌다.
여기에 더해 `indexOf`, `lastIndexOf`, `pendingChildren`는 현재 참조가 없는 잔여 헬퍼다.

## 분리 원칙

- `navigation_graph.go`: graph row/lane, cursor, page 이동
- `navigation_section.go`: 섹션 순서, 섹션 전환, target 조회
- `state_browse.go`: browse state 동기화와 커서 복원

`navigation.go`는 엔트리 포인트만 남기고, 실제 계산은 옮긴다.

## 구현 포인트

1. graph 전용 함수와 section 전용 함수를 분리한다.
2. browse 상태 복원 로직을 `state_browse.go`로 이동한다.
3. 섹션 target 조회와 섹션 순서를 `navigation_section.go`로 이동한다.
4. graph 래퍼와 graph 커서 계산을 `navigation_graph.go`로 이동한다.
5. `indexOf`, `lastIndexOf`, `pendingChildren`는 사용처가 없으면 삭제한다.
6. `maybeLoadMoreGraph()`처럼 비활성화된 로직은 남기지 않는다.

## BEFORE

현재 파일은 다음을 한 번에 담고 있다.

```go
func graphNodes(rs git.Status) []graphNode
func graphRows(rs git.Status) []graphRow
func graphRowWidth(row graphRow) int
func findGraphRowByHash(rows []graphRow, hash string) int
func graphPageSize(m *model) int
func moveSelectableGraphPointer(current int, rows []graphRow, delta int) int
func nearestSelectableGraphRow(rows []graphRow, start, step int) int
func graphPointerLane(row graphRow) int
func currentGraphFocus(rs git.Status, cursor int) graphNode

func graphSectionOrder() []graphSection
func sectionName(section graphSection) string
func nextGraphSection(current graphSection) graphSection
func prevGraphSection(current graphSection) graphSection
func sectionTargets(rs git.Status, section graphSection) []state.TargetItem
func activeSectionTarget(m model) string

func syncBrowseState(m *model, rs git.Status)
func moveBrowseCursor(m model, delta int) model
func moveGraphLane(m model, delta int) model
func pageBrowseGraph(m model, pages int) model
func maybeLoadMoreGraph(m model) (model, tea.Cmd)
func moveGraphScroll(current, total, delta int) int
func clampScroll(current, total, page int) int
func moveGraphPointer(current, total, delta int) int
func moveLanePointer(current int, row graphRow, delta int) int
func clampLaneCursor(current int, row graphRow) int
func clampCursor(current, total int) int

func indexOf(values []string, target string) int
func lastIndexOf(values []laneRef, target string) int
func pendingChildren(children []string, current string) []string
```

문제는 함수 수가 아니라 책임 혼합이다.

## AFTER

### `navigation.go`

엔트리 포인트만 둔다.

```go
func syncBrowseState(m *model, rs git.Status) {
    syncBrowseStateFromGraph(m, rs)
    syncBrowseStateSectionCursors(m, rs)
    syncBrowseStateSelection(m, rs)
}

func moveBrowseCursor(m model, delta int) model {
    switch m.activeSection {
    case sectionGraph:
        return moveGraphBrowseCursor(m, delta)
    case sectionCurrent, sectionRemote, sectionTags:
        return moveSectionBrowseCursor(m, delta)
    default:
        return m
    }
}
```

### `navigation_graph.go`

graph 이동과 graph 래퍼만 둔다.

```go
func graphNodes(rs git.Status) []graphNode
func graphRows(rs git.Status) []graphRow
func graphRowWidth(row graphRow) int
func findGraphRowByHash(rows []graphRow, hash string) int
func graphPageSize(m *model) int
func moveSelectableGraphPointer(current int, rows []graphRow, delta int) int
func nearestSelectableGraphRow(rows []graphRow, start, step int) int
func graphPointerLane(row graphRow) int
func currentGraphFocus(rs git.Status, cursor int) graphNode

func moveGraphBrowseCursor(m model, delta int) model
func moveGraphLane(m model, delta int) model
func pageBrowseGraph(m model, pages int) model
func moveGraphScroll(current, total, delta int) int
func clampScroll(current, total, page int) int
func moveGraphPointer(current, total, delta int) int
func moveLanePointer(current int, row graphRow, delta int) int
func clampLaneCursor(current int, row graphRow) int
func clampCursor(current, total int) int
```

### `navigation_section.go`

섹션 규칙과 target 조회만 둔다.

```go
func graphSectionOrder() []graphSection
func sectionName(section graphSection) string
func nextGraphSection(current graphSection) graphSection
func prevGraphSection(current graphSection) graphSection
func sectionTargets(rs git.Status, section graphSection) []state.TargetItem
func activeSectionTarget(m model) string
func moveSectionBrowseCursor(m model, delta int) model
```

### `state_browse.go`

browse 상태 복원만 둔다.

```go
func syncBrowseState(m *model, rs git.Status)
func syncBrowseStateFromGraph(m *model, rs git.Status)
func syncBrowseStateSectionCursors(m *model, rs git.Status)
func syncBrowseStateSelection(m *model, rs git.Status)
```

## 우선순위

1. `syncBrowseState()`를 먼저 나눈다.
2. `moveBrowseCursor()`의 분기 책임을 나눈다.
3. 섹션 규칙과 target 조회를 분리한다.
4. graph 래퍼와 clamp 헬퍼를 옮긴다.

## 검증 기준

1. `navigation.go`는 진입점만 보인다.
2. `navigation_graph.go`, `navigation_section.go`, `state_browse.go`의 역할이 겹치지 않는다.
3. `key_handling_browse.go`는 기존 행동을 그대로 유지한다.
4. browse cursor와 section cursor 동작이 바뀌지 않는다.

## 비고

- 이 문서는 preview 분리 문서가 아니다.
- 1차 리팩토링에서는 경계만 분리하고 동작은 유지한다.
