# `internal/app/graph_render.go` 리팩토링 계획서

## 목표

`internal/app/graph_render.go`에는 그래프 한 줄 렌더링, 커넥터 렌더링, compact decoration 포맷, 텍스트 축약 헬퍼가 한 파일에 함께 들어 있다.

이번 작업의 목표는 렌더링 결과는 바꾸지 않고, 책임을 더 작은 단위로 나눠서 테스트 가능한 형태로 정리하는 것이다.  
호출 지점은 `internal/app/view.go`와 기존 테스트를 그대로 유지한다.

이 문서는 동작 변경이 아니라 구조 개선을 위한 리팩토링 계획서다.

## 범위

- 코드 위치: `internal/app/graph_render.go`
- 관련 테스트: `internal/app/model_test.go`
- 필요 시 추가할 테스트 파일: `internal/app/graph_render_test.go`
- 패키지 경계: `internal/app` 안에서 유지
- 새 패키지 생성: 하지 않음
- 출력 계약: 현재 그래프 렌더링 동작 유지

## 검토 포인트

- `view.go`는 `renderGraphLine`과 `renderGraphConnectorLines`를 직접 호출하므로, 이 함수들의 시그니처는 유지하는 편이 안전하다.
- `renderGraphLine`에는 선택 상태 처리, virtual conflict 처리, raw graph prefix 처리, 텍스트 포맷팅이 섞여 있다.
- `renderRawGraphLine`은 포인터 강조와 HEAD decoration 처리 경로가 별도로 있어서, 더 잘게 나누기 전에 동작을 먼저 고정해야 한다.
- `renderGraphConnectorLines`는 안정 상태, collapse 상태, parent shift 상태를 모두 다루고 있다.
- `compactDecorationInfo`는 단순한 문자열 포맷이 아니라 branch 우선순위 규칙도 포함하므로 회귀 테스트가 필요하다.
- `compactWhenText`와 `compactTitleText`는 작지만 UI 계약을 직접 결정하므로 결과를 그대로 유지해야 한다.
- `padRight`는 styled text의 렌더된 너비를 기준으로 동작해야 한다.

## 추가 관심사 분리 검토

이번 파일만 나누는 것으로 끝내지 말고, 아래 파일들도 경계가 섞이는지 같이 확인해야 한다.

### `internal/app/view.go`

`renderGraphContent`는 단순한 호출 래퍼가 아니라 다음 책임을 함께 가진다.

- graph page 계산
- column width 계산
- header 렌더링
- row 렌더링
- connector 렌더링
- handshake 및 conflict 색상 처리

즉, `graph_render.go`를 정리한 뒤에도 `view.go`가 여전히 그래프 전용 조립 로직을 많이 갖고 있으면 `graph_view.go` 또는 `graph_content.go` 같은 별도 파일로 빼는 것이 자연스럽다.

### `internal/app/navigation.go`

`isLocalGraphPointer`는 graph row의 decoration 규칙을 다시 해석해서 현재 포인터가 local branch인지 판단한다.  
이 로직은 `compactDecorationInfo`가 렌더링하는 branch 규칙과 사실상 같은 도메인이다.

따라서 `graph_render.go`를 분리할 때 이 판단 규칙도 함께 검토해야 한다.

- 렌더링 규칙과 pointer 판정 규칙이 서로 어긋나지 않는지 확인
- 필요하면 branch decoration 판정을 공용 helper로 추출
- 최악의 경우 렌더링 파일과 navigation 파일이 서로 같은 문자열 규칙을 중복 구현하지 않도록 정리

### `internal/app/model_test.go`

현재 graph 렌더링 관련 테스트가 많이 섞여 있다.  
구현 시점에는 유지해도 되지만, 최종적으로는 graph 렌더링 전용 테스트만 `graph_render_test.go`로 옮기는 편이 더 읽기 쉽다.

## BEFORE

현재 파일은 여러 책임이 한 곳에 섞여 있다.

```go
func renderGraphLine(...)
func renderRawGraphLine(...)
func graphLineCell(...)
func highlightRawGraphPrefix(...)
func renderGraphConnectorLines(...)
func collapseConnectorLines(...)
func parentShiftConnectorLines(...)
func compactDecorationInfo(...)
func compactWhenText(...)
func compactTitleText(...)
```

문제는 단순히 파일이 크다는 점이 아니다.  
무엇이 렌더링 정책이고, 무엇이 포맷 정책인지 한눈에 구분하기 어렵다는 점이 더 크다.

## AFTER

현재의 진입 함수는 유지하되, 내부에서는 orchestration만 담당하도록 바꾼다.

제안하는 분리는 다음과 같다.

```go
func renderGraphLine(...)
func renderRawGraphLine(...)
```

이 두 함수는 view layer가 계속 사용하는 최상위 진입점으로 남긴다.

내부 렌더링 책임은 더 작은 헬퍼로 분리한다.

```go
func renderGraphCells(...)
func renderGraphMetadata(...)
func renderGraphLinePrefix(...)
func renderGraphText(...)
```

커넥터 렌더링은 별도 흐름으로 분리한다.

```go
func renderGraphConnectorLines(...)
func collapseConnectorLines(...)
func parentShiftConnectorLines(...)
func renderGraphSpacer(...)
```

포맷 관련 헬퍼는 한 묶음으로 정리한다.

```go
func formatCompactDecorations(...)
func compactDecorationInfo(...)
func compactWhenText(...)
func compactTitleText(...)
func hasHeadDecoration(...)
func padRight(...)
```

권장 파일 분리는 다음과 같다.

- `graph_render.go`: 진입점과 orchestration
- `graph_render_connectors.go`: 커넥터 생성 헬퍼
- `graph_render_format.go`: decoration 및 텍스트 포맷 헬퍼
- `graph_render_test.go`: 그래프 렌더링 중심 테스트

이 분리는 의도적으로 보수적으로 잡는다.  
목표는 추상화를 늘리는 것이 아니라 이해 비용을 줄이는 것이다.

## 테스트

현재 동작을 고정하기 위해 다음 테스트를 추가하거나 유지한다.

```go
func TestRenderGraphLineKeepsColumnOrder(t *testing.T)
func TestRenderGraphLineHighlightsSelectedPointer(t *testing.T)
func TestRenderGraphLineKeepsVirtualConflictMarkerBlank(t *testing.T)
func TestRenderGraphLinePreservesRawGraphPrefix(t *testing.T)
func TestRenderGraphLineHandlesHandshakeHighlighting(t *testing.T)
func TestRenderGraphConnectorLinesSkipsStableTransition(t *testing.T)
func TestRenderGraphConnectorLinesUsesSingleLineForTwoLaneCollapse(t *testing.T)
func TestRenderGraphConnectorLinesShowsProgressiveMultiLaneCollapse(t *testing.T)
func TestRenderGraphConnectorLinesShowsParentShiftWithoutFullCollapse(t *testing.T)
func TestCompactDecorationInfoKeepsBranchPrecedence(t *testing.T)
func TestCompactDecorationInfoKeepsOriginHeadVisible(t *testing.T)
func TestCompactWhenText(t *testing.T)
func TestCompactTitleText(t *testing.T)
func TestHasHeadDecoration(t *testing.T)
```

이미 `model_test.go` 안에서 이 동작들 중 일부가 커버되고 있다.  
리팩토링 과정에서는 그 테스트들을 그대로 두거나, 그래프 전용 케이스만 `graph_render_test.go`로 옮길 수 있다.

반드시 유지해야 할 동작은 다음과 같다.

1. 선택된 row는 기존과 같은 column 위치에 pointer marker를 표시해야 한다.
2. HEAD decoration은 branch 강조보다 우선해야 한다.
3. virtual conflict row는 hash와 refs column을 비워둔 상태를 유지해야 한다.
4. `row.Graph`가 있으면 raw graph prefix는 그대로 보여야 한다.
5. 안정 상태의 connector 출력은 지금처럼 짧고 간단해야 한다.
6. multi-lane convergence는 progressive collapse 형태를 유지해야 한다.
7. parent shift connector는 중간 vertical context를 유지해야 한다.
8. compact decoration text는 기존 10자 제한을 유지해야 한다.
9. relative time과 commit title 축약 결과도 기존 경계를 그대로 유지해야 한다.

## 검증

가장 좁은 범위부터 확인한다.

```sh
go test ./internal/app -run 'TestRenderGraph|TestCompact|TestHasHeadDecoration'
```

분리가 끝나면 저장소 전체 체크도 수행한다.

```sh
scripts/check
```

만약 테스트를 새 파일로 옮겼다면, 파일 이동 이후에 다시 한 번 위의 집중 테스트를 실행해 연결 상태를 확인한다.

## 비고

- 이번 리팩토링에서는 렌더링 텍스트 계약을 바꾸지 않는다.
- 그래프 렌더링 로직을 `view.go`로 옮기지 않는다.
- `graph_render.go`가 다시 커질 때까지는 새 패키지를 만들지 않는다.
- 헬퍼가 다시 서로 섞이기 시작하면, 추상화 레이어를 하나 더 얹기보다 파일 분리를 먼저 고려한다.

## 최종 검수

구현 전에 문서 기준으로 다음을 다시 확인한다.

1. `renderGraphLine`와 `renderGraphConnectorLines`의 외부 호출 시그니처를 바꾸지 않는다.
2. `view.go`에서 graph 렌더링 조립 책임이 과도하게 남아 있지 않은지 확인한다.
3. `navigation.go`와 문자열 규칙이 중복되지 않도록 한다.
4. `compactDecorationInfo`와 `isLocalGraphPointer`가 같은 도메인 규칙을 서로 다르게 해석하지 않는지 확인한다.
5. row rendering, connector rendering, text formatting을 서로 다른 파일로 나눈다.
6. 테스트는 `model_test.go`에 그대로 둘지, `graph_render_test.go`로 옮길지 한 번에 결정한다.
7. 렌더링 결과는 기존 스냅샷 수준으로 유지한다.
8. `scripts/check` 전에 `go test ./internal/app -run 'TestRenderGraph|TestCompact|TestHasHeadDecoration'`를 우선 실행한다.

## 열려 있는 결정

1. `compactDecorationInfo`를 `graph_render.go`에 그대로 둘지, 바로 `graph_render_format.go`로 옮길지
2. `model_test.go`에 있는 그래프 렌더링 검증을 지금 `graph_render_test.go`로 옮길지, 아니면 구현이 끝난 뒤 옮길지
3. `renderGraphLine` 아래에 더 작은 orchestration 함수를 둘지, 아니면 현재 함수명을 최상위 진입점으로 계속 유지할지
