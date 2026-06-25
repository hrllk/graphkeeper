# `internal/app/view.go` / `internal/app/view_detail.go` 경계 분리 계획서

## 목표

`view.go`의 프레임 조립, 섹션 렌더링, 레이아웃 헬퍼, detail pane 렌더링 경계를 나눈다.

핵심은 `view.go`가 화면을 조립만 하고, 섹션/디테일의 내용과 레이아웃 계산은 별도 파일로 흘려보내는 것이다.

## 범위

- 대상 파일
  - `internal/app/view.go`
  - `internal/app/view_detail.go`
- 후보 파일
  - `internal/app/view_shell.go`
  - `internal/app/view_layout.go`
  - `internal/app/view_sections.go`
  - `internal/app/view_detail.go`
- 관련 파일
  - `internal/app/view_graph.go`
  - `internal/app/navigation_section.go`
  - `internal/app/navigation_graph.go`
  - `internal/app/graph_render.go`
  - `internal/app/graph_render_format.go`
- 관련 테스트
  - `internal/app/model_test.go`
  - `internal/app/preview_test.go`
  - `internal/app/execution_detail_test.go`
- 새 package 생성: 하지 않음

## 참조 문서

이 계획은 아래 문서들의 경계 분리 원칙을 이어받는다.

- `docs/archive/202606-0003-refactor-graph-render-test-plan.backup.md`
- `docs/archive/202606-0004-refactor-view-graph-structure-plan.md`
- `docs/archive/202606-0011-refactor-model-boundary-plan.md`
- `docs/archive/20260625-0012-refactor-navigation-boundary-plan.md`
- `docs/model-refactor-plan.md`

## 현재 문제

`view.go`는 단순한 entrypoint가 아니라 다음을 함께 가진다.

- 전체 frame 배치
- pane size 계산
- section box 렌더링
- graph / detail pane 조립
- popup overlay helper
- layout padding / width helper

`view_detail.go`는 작은 파일이지만, detail pane 안에서 repo summary, focus summary, action help를 한 번에 묶는다.

지금도 동작은 맞지만, 화면 조립과 내용 생성이 한 파일 단위로 너무 가까이 붙어 있다.

## 분리 원칙

- `view.go` 또는 `view_shell.go`: `View()` 진입점과 frame 조립
- `view_layout.go`: pane size, width/height, overlay helper
- `view_sections.go`: section box content, browse section summary
- `view_detail.go`: detail pane content

`view_detail.go`는 이미 detail pane 전용 파일이므로 유지하되, 내부가 다시 커지면 세부 블록으로 더 나눈다.

## 구현 포인트

1. `View()`는 화면 구조만 조립한다.
2. section list 렌더링은 `view_sections.go`로 이동한다.
3. layout 계산은 `view_layout.go`로 이동한다.
4. `renderDetailContent()`는 detail pane의 최상위 진입점만 유지한다.
5. detail 내부의 repo / focus / action block이 커지면 별도 helper로 나눈다.

## BEFORE

현재 `view.go`와 `view_detail.go`는 아래 책임을 함께 가진다.

```go
func (m model) View() string
func (m model) renderSectionContent(section graphSection, width, height int) string
func paneWidth(total int, ratio float64) int
func splitPaneWidths(total int) (int, int)
func splitDashboardHeights(total int) (int, int)
func splitPaneHeights(total int) (int, int)
func fitBlockLines(lines []string, height int) string
func overlayPopup(base string, popup string) string
func overlayLine(baseLine string, popupLine string, startX, popupW int) string

func renderStatusCompact(s state.Status) string
func renderTargets(s state.Status) string
func formatTargetItem(t state.TargetItem) string
func renderActionHelpLines(m model) []string

func (m model) renderDetailContent(width, height int) string
func focusParentLines(node graphNode, width int) []string
func focusBranchSummaryLines(node graphNode, width int) []string
```

문제는 단일 책임이 아니라, "frame", "layout", "section content", "detail content"가 한 흐름에 붙어 있다는 점이다.

## AFTER

`view.go`는 shell 역할로 줄인다.

```go
func (m model) View() string {
	return renderAppView(m)
}
```

레이아웃은 별도 파일로 옮긴다.

```go
func paneWidth(total int, ratio float64) int
func splitPaneWidths(total int) (int, int)
func splitDashboardHeights(total int) (int, int)
func splitPaneHeights(total int) (int, int)
func fitBlockLines(lines []string, height int) string
func overlayPopup(base string, popup string) string
func overlayLine(baseLine string, popupLine string, startX, popupW int) string
```

섹션 렌더링은 별도 파일로 둔다.

```go
func (m model) renderSectionContent(section graphSection, width, height int) string
func renderSectionBox(...)
func renderStatusCompact(s state.Status) string
func renderTargets(s state.Status) string
func formatTargetItem(t state.TargetItem) string
func renderActionHelpLines(m model) []string
```

`view_detail.go`는 detail pane 최상위 진입점만 유지한다.

```go
func (m model) renderDetailContent(width, height int) string
func renderDetailRepoLines(...)
func renderDetailFocusLines(...)
func renderDetailActionLines(...)
func focusParentLines(node graphNode, width int) []string
func focusBranchSummaryLines(node graphNode, width int) []string
```

## 테스트

현재 유지해야 할 동작은 다음과 같다.

```go
func TestRenderGraphContentFixedHeight(t *testing.T)
func TestRenderDetailContentFixedHeight(t *testing.T)
func TestRenderActionHelpLinesAreSectionSpecific(t *testing.T)
func TestFormatCompactDecorations(t *testing.T)
```

리팩토링 후에는 다음도 확인한다.

```go
func TestViewLayoutKeepsSectionsAligned(t *testing.T)
func TestViewDetailKeepsFocusAndActionBlocks(t *testing.T)
func TestOverlayPopupCentersWithoutDistortingBase(t *testing.T)
```

## 검증 기준

1. `View()`는 frame 조립만 담당한다.
2. layout helper와 content helper가 섞이지 않는다.
3. `view_detail.go`는 detail pane 전용 경계를 유지한다.
4. 렌더링 결과와 fixed-height 동작은 바뀌지 않는다.

## 비고

- `view_graph.go`는 이미 graph pane 전용 조립 경계로 작동하므로 이번 계획에서는 건드리지 않는다.
- `target_items.go`는 browse/action target 정책을 함께 담지만, 현재는 같은 도메인이라 유지해도 된다.
- detail pane이 다시 커질 때만 내부 helper를 더 나눈다.
