# `internal/app/view.go` / `internal/app/view_graph.go` 분리 계획서

## 목표

`view.go`는 전체 화면 조립만 담당하고, 그래프 pane 조립은 `view_graph.go`로 분리한다.

이 단계의 목표는 화면 레이아웃과 그래프 렌더링 조립을 분리해서, `view.go`를 읽었을 때 전체 프레임만 보이게 만드는 것이다.

## 범위

- 대상 파일
  - `internal/app/view.go`
  - `internal/app/view_graph.go` 신설
- 관련 테스트
  - `internal/app/model_test.go`
  - 필요 시 `internal/app/view_graph_test.go`
- 새 패키지 생성: 하지 않음

## 현재 문제

`view.go`는 현재 다음을 함께 들고 있다.

- 전체 화면 레이아웃 계산
- 상단/하단 pane 배치
- 그래프 pane 조립
- detail pane 조립
- popup overlay
- footer

이 중 그래프 pane 조립은 독립적인 책임이므로 먼저 분리하는 편이 낫다.

## 분리 대상

### `view.go`에 남길 것

- `View()` 최상위 프레임
- 전체 너비/높이 계산
- top/bottom pane 조립
- popup overlay
- footer

### `view_graph.go`로 옮길 것

- `renderGraphContent`
- graph page 헤더
- graph row 반복
- graph connector 반복
- handshake / conflict 색상 처리

### 상황에 따라 검토할 것

- `renderStatusCompact`
- `renderTargets`
- `renderDetailContent`

이 셋은 이번 단계의 필수 분리 대상은 아니다.  
다만 `view.go`가 다시 비대해지면 다음 단계에서 `view_detail.go` 또는 `view_sections.go`로 나누는 후보가 된다.

## 이상적인 구조

```go
// view.go
func (m model) View() string
func (m model) getBoxStyle(section graphSection) lipgloss.Style
func (m model) renderDetailContent(width, height int) string
func renderStatusCompact(s state.Status) string
func renderTargets(s state.Status) string

// view_graph.go
func (m model) renderGraphContent(width, height int) string
```

`View()`는 프레임만 담당하고, 그래프 pane의 세부 조립은 `view_graph.go`가 맡는다.

## 구현 순서

1. `view_graph.go`를 새로 만든다.
2. `renderGraphContent`를 이동한다.
3. 그래프 전용 helper를 같이 옮긴다.
4. `view.go`에서 graph 관련 import와 호출을 정리한다.
5. `go test ./internal/app -run TestRenderGraphContentFixedHeight` 같은 좁은 테스트로 확인한다.

## 테스트 항목

1. graph pane 높이가 고정되어 있는지
2. `renderGraphContent`가 기존과 같은 줄 수를 유지하는지
3. graph header와 row, connector 순서가 유지되는지
4. `view.go`가 그래프 조립 디테일을 더 이상 직접 가지지 않는지

## 완료 기준

- `view.go`를 열었을 때 전체 화면 조립만 보인다.
- 그래프 pane 조립은 `view_graph.go`에 있다.
- 렌더링 결과가 기존과 같다.

## 비고

- 이 단계는 구조 분리의 시작점이다.
- 화면 전체 프레임과 그래프 pane 조립을 먼저 끊어야 이후 navigation/helper 분리와 renderer 정리가 쉬워진다.
