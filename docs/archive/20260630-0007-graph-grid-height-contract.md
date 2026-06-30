# Graph Grid Height Contract

## 목적

`Graph` 영역과 우측 레일을 Bootstrap 스타일의 shared-height grid row 로 취급하고,
각 cell 이 같은 outer height 를 공유하도록 하는 구현 계약을 고정한다.

이 문서는 레이아웃 비율 설명이 아니라, 실제 구현 시 필요한 height contract 만 담는다.

## 범위

- 포함
  - `internal/app/view_shell.go`
  - `internal/app/view_layout.go`
  - `internal/app/view_graph.go`
  - `internal/app/view_sections.go`
  - `internal/app/graph_render.go`
  - `internal/app/graph_render_format.go`
  - `internal/app/model_test.go`
- 제외
  - section navigation
  - graph DAG 규칙
  - reset / pull / popup UX

## 핵심 계약

### 1. 부모 row 가 outer height 를 정한다

- `Graph` 와 right rail 은 같은 main row 안에 들어간다.
- main row 가 최종 outer height 를 결정한다.
- 자식 cell 은 높이를 자기 마음대로 늘리지 않는다.
- 각 cell 은 전달받은 height 안에서만 렌더한다.

### 2. outer height 와 inner height 를 분리한다

- outer height 는 cell 전체 높이이다.
- inner height 는 `border` 와 `padding` 을 뺀 내용 높이이다.
- 구현은 `Height(n)` 을 outer 기준으로 사용하고, content 계산은 inner height 기준으로 한다.
- top title 1줄, content, border/padding 을 혼동하면 경계가 무너진다.
- 이 문서에서 고정하는 기본 셀 구조는 `title 1줄 + content + border 2줄 + padding 0` 이다.
- 따라서 기본 inner height 계산은 `innerHeight = outerHeight - 3` 이다.

권장 계산식:

```go
outerHeight := rowHeight
innerHeight := outerHeight - 3
```

- `Graph` box 와 right rail box 는 같은 outerHeight 를 받는다.
- right rail 의 `Local / Remote / Tags` 는 그 outerHeight 를 다시 나눈다.

### 3. right rail 은 nested grid 이다

- `Local / Remote / Tags` 는 right rail 내부의 3개 cell 이다.
- 기본 분할은 `1:1:1` 이다.
- 정수 remainder 는 마지막 cell 이 흡수한다.
- 마지막 cell 이 remainder 를 먹지 못하면 전체 rail 높이와 어긋난다.

권장 규칙:

- `splitThreeHeights(total)` 는 `a + b + c = total` 을 만족해야 한다.
- `c` 가 remainder absorber 이다.
- `a`, `b`, `c` 는 모두 0보다 커야 한다. 단, 극단적으로 작은 화면에서는 최소 보장을 우선한다.

### 4. wrap 은 허용하지 않는다

- shared-height grid 에서 한 cell 이 wrap 되면 행 전체 높이 계약이 깨진다.
- 따라서 `Graph` row 와 rail content 는 반드시 visible width 안에 들어가야 한다.
- content가 길면 wrap 대신 truncate / clip 이 발생해야 한다.
- ANSI 스타일이 있어도 visible width 기준으로 계산해야 한다.
- clip 책임은 `graph_render.go` 와 `view_graph.go` 에 둔다.
- `renderGraphLine()` 은 row 단위 clip 을 담당하고, `renderGraphContent()` 는 page header / column header / connector / row append 단계에서 width 를 다시 보정한다.
- `renderRightRail()` 은 각 box 내부 content 가 넘치지 않게 `splitThreeHeights()` 결과에 맞춰 렌더한다.
- `renderGraphLine()` 의 clip 순서는 `title -> when -> refs -> graphCell -> hash` 이다.
- `renderGraphContent()` 의 clip 순서는 `page header -> column header -> connector line -> graph row` 이다.
- 최종 절단은 visible width 를 다시 계산해서 target width 를 넘으면 `fitVisibleWidth()` 같은 helper 로 한 번 더 자른다.
- ANSI 스타일이 섞인 경우에도 최종 절단은 styled text 가 아닌 visible width 기준으로 수행한다.

### 5. 폭과 높이의 책임 경계를 분리한다

- `view_layout.go`
  - body width / height
  - header height
  - graph rail height
  - right rail split
- `view_shell.go`
  - row 조립
  - cell 배치
  - popup overlay
- `view_graph.go`
  - graph content height 적용
  - graph page / header / row / connector 렌더
- `graph_render.go`
  - row 문자열 생성
  - row width fit

## 구현 규칙

### `renderAppView(m model)`

- main row 높이를 먼저 계산한다.
- `Graph` box 와 right rail 에 동일한 outer height 를 준다.
- right rail 내부 3개 cell 의 합이 outer height 와 같아야 한다.
- outer height 계산을 view layer 에서 다시 하면 안 된다.

### `renderRightRail(width, height)`

- `height` 는 outer height 이다.
- `splitThreeHeights(height)` 로 3개 cell height 를 구한다.
- 마지막 cell 이 remainder 를 흡수한다.
- 각 cell 은 `Height(cellHeight)` 만 사용하고 내용을 넘치게 만들지 않는다.
- `Height(cellHeight)` 는 outer 기준이다.
- 각 cell 내부 content 는 `cellHeight - 3` 을 사용한다.
- remainder 는 마지막 cell 의 outer height 에 반영한다.

### `renderGraphContent(width, height)`

- `height` 는 graph content inner height 이다.
- page header, column header, row, connector line 까지 모두 이 height 안에 들어가야 한다.
- 한 line 이 width 를 넘으면 안 된다.
- wrap 이 아니라 clip 이 발생해야 한다.
- page header 와 column header 는 먼저 width 보정 대상이다.
- row 와 connector line 은 `renderGraphLine()` 및 `renderGraphConnectorLines()` 결과를 width 안으로 다시 보정한다.
- `fitBlockLines()` 는 height 만 맞추고, width clip 은 별도 helper 가 담당한다.

### `renderGraphLine(...)`

- 최종 row 문자열은 visible width 기준으로 한 줄이어야 한다.
- title 이 먼저 잘리고, 그 다음 when, 그 다음 refs 순서로 줄인다.
- `hash` 와 lane 정보는 가능한 한 유지한다.
- raw graph prefix 가 있으면 그 prefix 의 의미가 유지되도록 clip 한다.

## 테스트 계약

다음 테스트는 유지하거나 추가해야 한다.

```go
func TestGraphRailMatchesStackedSideRailHeight(t *testing.T)
func TestGraphContentMatchesStackedSideRailContent(t *testing.T)
func TestRenderGraphContentFixedHeight(t *testing.T)
func TestRenderGraphLineKeepsCollapsedCommitMarker(t *testing.T)
func TestRenderRightRailUsesRemainderOnLastCell(t *testing.T)
func TestRenderGraphLineNeverWraps(t *testing.T)
func TestRenderGraphLineClipsTitleBeforeGraphCell(t *testing.T)
func TestRenderGraphContentClipsHeadersBeforeRows(t *testing.T)
```

추가하면 좋은 검증:

- 좁은 width 에서도 wrap 이 생기지 않는지
- ANSI 포함 문자열의 visible width 가 target width 를 넘지 않는지
- right rail 3개 cell 의 합이 항상 outer height 와 같은지

## 수용 기준

1. `Graph` 와 right rail 은 같은 outer height 를 공유한다.
2. `Local / Remote / Tags` 는 nested grid 로서 right rail height 를 정확히 분할한다.
3. 마지막 right rail cell 이 remainder height 를 흡수한다.
4. `Graph` row 는 wrap 없이 1 logical line 으로 유지된다.
5. height 계약과 width 계약이 서로 독립적으로 무너지지 않는다.
