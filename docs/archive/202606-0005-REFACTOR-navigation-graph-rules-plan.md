# `internal/app/navigation.go` 중복 규칙 분리 계획서

## 목표

`navigation.go`는 cursor 이동만 담당하고, 렌더링과 공유하는 branch/decorations 판정 규칙은 별도 helper로 뺀다.

핵심은 `isLocalGraphPointer`가 렌더링 문자열 규칙을 다시 해석하지 않도록 만드는 것이다.

## 범위

- 대상 파일
  - `internal/app/navigation.go`
  - 필요 시 `internal/app/graph_rules.go` 또는 동등한 helper 파일
- 관련 테스트
  - `internal/app/model_test.go`
  - `internal/app/key_handling.go` 관련 테스트가 있으면 함께 확인
- 새 패키지 생성: 하지 않음

## 현재 문제

현재 `navigation.go`에는 다음이 함께 있다.

- graph cursor/page helper
- browse state sync
- section target 계산
- lane cursor clamp
- local pointer 판정

이 중 local pointer 판정은 렌더링 규칙과 같은 도메인을 공유한다.  
즉, navigation 파일에서 문자열 규칙을 다시 구현하는 구조는 유지보수에 불리하다.

## 분리 대상

### `navigation.go`에 남길 것

- cursor 이동
- page 이동
- focus 계산
- browse state 동기화
- section 대상 선택

### helper 파일로 뺄 것

- `isLocalGraphPointer`
- `compactDecorationInfo`와 공유하는 branch decoration 규칙
- HEAD / local / remote 판정의 공통 로직

## 권장 helper 책임

예상 helper 파일 이름은 `graph_rules.go` 또는 유사한 이름이 적절하다.

그 안에는 다음이 들어갈 수 있다.

- `hasLocalBranchDecoration`
- `hasHeadDecoration`
- `normalizeBranchDecoration`
- `isLocalGraphPointer`

중요한 점은 helper가 cursor 이동을 알면 안 된다는 것이다.  
이 파일은 오직 규칙 해석만 담당해야 한다.

## 이상적인 구조

```go
// navigation.go
func graphRows(rs git.Status) []graphRow
func graphPageSize(m *model) int
func moveSelectableGraphPointer(current int, rows []graphRow, delta int) int
func nearestSelectableGraphRow(rows []graphRow, start, step int) int
func currentGraphFocus(rs git.Status, cursor int) graphNode
func syncBrowseState(m *model, rs git.Status)
func moveBrowseCursor(m model, delta int) model
func moveGraphLane(m model, delta int) model
func pageBrowseGraph(m model, pages int) model

// graph_rules.go
func isLocalGraphPointer(rs git.Status, cursor int, laneCursor int) bool
```

## 구현 순서

1. `isLocalGraphPointer`를 helper로 이동할 후보로 분리한다.
2. 렌더링에서 쓰는 decoration 규칙과 같은 helper를 만든다.
3. `navigation.go`에서 helper를 호출하도록 바꾼다.
4. raw graph 모드와 legacy lane 모드 둘 다 유지되는지 확인한다.
5. `key_handling.go`의 merge/rebase gate가 같은 helper를 계속 쓰는지 확인한다.

## 테스트 항목

1. raw graph 모드에서 local branch 판정이 기존과 같은지
2. legacy lane 모드에서 lane side 판정이 유지되는지
3. HEAD decoration이 local 판정에 반영되는지
4. remote / tag decoration이 local 판정에 섞이지 않는지
5. 렌더링과 navigation이 같은 branch 규칙을 보는지

## 완료 기준

- navigation 파일에 렌더링 문자열 규칙이 남아 있지 않다.
- local pointer 판정이 공용 helper로 이동한다.
- 렌더링과 navigation이 같은 branch/decorations 규칙을 공유한다.

## 비고

- 이 단계는 단순 파일 정리가 아니라 규칙 중복 제거다.
- `compactDecorationInfo`와 판정 규칙을 따로 두면 결국 다시 어긋난다.
- 따라서 navigation 분리는 graph rendering 분리와 같이 봐야 한다.
