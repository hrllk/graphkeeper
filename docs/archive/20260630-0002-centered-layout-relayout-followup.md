# Centered Layout Relayout Follow-up

## 목적

이 문서는 기존 레이아웃 리레이아웃 계획에서 아직 반영되지 않은 항목을 정리하고,
왜 화면이 기대한 대로 바뀌지 않았는지 원인을 분석하기 위한 후속 문서다.

참조한 원문은 다음이다.

- `docs/archive/20260625-0019-refactor-centered-layout-relayout-plan.md`
- `docs/archive/20260625-0017-refactor-graph-first-layout-and-mode-panels-plan.md`

## AS-IS

스크린샷 `Screenshot 2026-06-30 at 16.22.35.png` 기준의 기존 방향은 다음과 같다.

- 상단은 `Local / Remote / Tags` 가 먼저 노출된다.
- 하단은 `Graph / Mode` 로 나뉜다.
- 전체가 좌상단 기준으로 붙어 보이고, 시각적 중심이 없다.
- top margin 이 충분히 체감되지 않는다.
- `Global / Context` 같은 전역 헤더는 없다.
- 숫자키와 섹션 위계가 현재 목표와 맞지 않는다.

이 상태는 “layout이 살아는 있지만 제품 화면처럼 정리되지 않은 상태”다.

## TO-BE

스크린샷 `Screenshot 2026-06-30 at 16.22.53.png` 가 보여주려는 목표 방향은 다음이다.

- 화면 전체가 가운데 정렬되어야 한다.
- 상단에는 `Mode - Global` 과 `Mode - Local` 이 분리되어 보여야 한다.
- `Global / Context` 헤더는 3:7 비율이어야 한다.
- `Graph` 는 좌측의 가장 큰 작업면이 되어야 한다.
- 우측에는 `Local / Remote / Tags` 가 세로로 쌓여야 한다.
- `Graph` 와 우측 레일은 하나의 정렬 규칙 안에서 맞물려야 한다.
- top margin 은 실제로 체감되어야 하고, 레이아웃이 천장에 붙어 보이면 안 된다.

즉, TO-BE 는 단순히 박스를 더 예쁘게 만드는 것이 아니라,
`Global / Context / Graph / Local / Remote / Tags` 의 위계가 명확하게 보이는 상태다.

## 최종 구현 계약

이 문서를 보고 바로 구현할 수 있도록, 최종 배치 계약을 아래처럼 고정한다.

### 1. 전체 shell

- 화면 전체는 terminal 중앙에 배치한다.
- shell 바깥 여백은 좌우 10%, 상하 10% 를 기본으로 한다.
- top margin 은 실제 빈 공간으로 보여야 한다.
- footer 는 shell 내부 폭이 아니라 terminal 전체 폭 기준으로 정렬한다.

### 2. 헤더 영역

- 헤더는 `Mode - Global` 과 `Mode - Local` 두 박스로 나눈다.
- 폭 비율은 `3:7` 이다.
- 좌측 박스는 전역 상태, 우측 박스는 현재 포커스/컨텍스트를 보여준다.
- 헤더는 메인 그래프보다 작은 높이를 유지한다.

### 3. 메인 영역

- 메인 영역은 `Graph` 와 우측 레일로 나눈다.
- `Graph` 가 좌측의 주 작업면이다.
- 우측 레일은 `Local / Remote / Tags` 세 박스를 세로로 쌓는다.
- `Graph` 는 우측 레일의 전체 높이와 시각적으로 맞물려야 한다.
- 우측 레일의 마지막 박스가 remainder height 를 흡수한다.

### 4. 비율 규칙

- `Global / Context` 는 `3:7`.
- `Graph / right rail` 은 `대략 7:3`.
- `Local / Remote / Tags` 는 `1:1:1` 에 가까운 세로 분할.
- 비율 계산에서 정수 반올림이 필요하면, remainder 는 우측 또는 마지막 row 에 흡수한다.

### 5. 최소 크기

- shell body width 는 너무 좁아지지 않도록 최소 폭을 둔다.
- header 와 graph rail 은 각각 읽을 수 있는 최소 높이를 유지한다.
- 작은 화면에서는 비율을 유지하되, 최소 폭/높이를 먼저 보장한다.

### 6. 팝업

- confirm / reset popup 은 body 위에 오버레이한다.
- popup 폭은 body 폭을 넘지 않아야 한다.
- popup 은 중앙에 떠야 하고, 전체 shell 의 중앙 정렬을 깨지 않아야 한다.

## Code Sketch

이 섹션은 문서를 읽는 즉시 구현할 수 있도록, 실제 코드에 가깝게 적는다.

### `internal/app/view_layout.go`

레이아웃 수치는 여기서 고정한다.

```go
func layoutShellMargins(m model) (hMargin, topMargin, bottomMargin int) {
	hMargin = int(float64(m.width) * 0.10)
	topMargin = int(float64(m.height) * 0.10)
	bottomMargin = int(float64(m.height) * 0.10)

	if hMargin < 2 {
		hMargin = 2
	}
	if topMargin < 1 {
		topMargin = 1
	}
	if bottomMargin < 1 {
		bottomMargin = 1
	}

	// 너무 넓은 shell 을 막는다.
	if maxMargin := (m.width - 80) / 2; maxMargin >= 0 && hMargin > maxMargin {
		hMargin = maxMargin
	}

	// top margin 과 bottom margin 이 shell height 를 지나치게 잠식하지 않도록 제한한다.
	if maxTop := m.height - 20; maxTop >= 0 && topMargin > maxTop {
		topMargin = maxTop
	}
	if maxBottom := m.height - topMargin - 19; maxBottom >= 0 && bottomMargin > maxBottom {
		bottomMargin = maxBottom
	}
	return hMargin, topMargin, bottomMargin
}

func splitPaneWidths(total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}

	left := total * 3 / 10
	if left < 1 {
		left = 1
	}
	if left > total-1 {
		left = total - 1
	}
	right := total - left
	return left, right
}

func splitThreeHeights(total int) (int, int, int) {
	if total <= 0 {
		return 0, 0, 0
	}

	first := total / 3
	second := total / 3
	third := total - first - second

	if first == 0 {
		first = 1
	}
	if second == 0 && total > 1 {
		second = 1
	}
	if third == 0 && total > 2 {
		third = 1
	}

	for first+second+third > total {
		switch {
		case third > 1:
			third--
		case second > 1:
			second--
		case first > 1:
			first--
		default:
			return total, 0, 0
		}
	}

	if rem := total - (first + second + third); rem > 0 {
		third += rem
	}
	return first, second, third
}
```

### `internal/app/view_shell.go`

렌더링은 `header -> main -> footer` 순서로 고정한다.

```go
func renderAppView(m model) string {
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	bodyWidth, bodyHeight := layoutShellBodySize(m, hMargin, topMargin, bottomMargin)
	headerHeight := layoutHeaderHeight(bodyHeight)
	graphRailHeight := layoutGraphRailHeight(bodyHeight)

	globalWidth, contextWidth := splitPaneWidths(bodyWidth)
	headerRow := renderHeaderRow(m, globalWidth, contextWidth, headerHeight)
	mainRow := renderMainRow(m, bodyWidth, graphRailHeight)

	body := lipgloss.JoinVertical(lipgloss.Left, headerRow, mainRow)
	body = applyOuterMargins(body, bodyWidth, bodyHeight, hMargin, topMargin, bottomMargin)

	if m.status.Mode == state.ModeConfirm || m.status.Mode == state.ModeResetModePick {
		body = overlayPopup(body, renderModalPopup(m, bodyWidth))
	}

	footer := renderFooterLine(m)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, body+"\n"+footer+"\n")
}

func renderHeaderRow(m model, leftWidth, rightWidth, height int) string {
	globalBox := baseBox.Width(leftWidth).Height(height).Render(
		"Mode - Global\n" + m.renderGlobalContent(max(leftWidth-4, 0), max(height-3, 0)),
	)
	contextBox := baseBox.Width(rightWidth).Height(height).Render(
		"Mode - Local\n" + m.renderContextContent(max(rightWidth-4, 0), max(height-3, 0)),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, globalBox, contextBox)
}

func renderMainRow(m model, width, height int) string {
	graphWidth := int(float64(width) * 0.72)
	if graphWidth < 56 {
		graphWidth = 56
	}
	if graphWidth > width-18 {
		graphWidth = width - 18
	}
	if graphWidth < 0 {
		graphWidth = 0
	}

	rightWidth := width - graphWidth
	graphBox := m.getBoxStyle(sectionGraph).Width(graphWidth).Height(height).Render(
		"Graph\n" + m.renderGraphContent(max(graphWidth-4, 0), max(height-3, 0)),
	)
	rightRail := renderRightRail(m, rightWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, graphBox, rightRail)
}

func renderFooterLine(m model) string {
	footer := muted.Render("global: 1 local  •  2 remote  •  3 tags  •  4 graph  •  tab/shift+tab section  •  up/down/j/k move  •  f fetch  •  q quit")
	return lipgloss.Place(m.width, 1, lipgloss.Center, lipgloss.Center, footer)
}
```

### `internal/app/view_shell.go` continued

우측 레일과 popup 은 같은 파일에서 처리한다.

```go
func renderRightRail(m model, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	localHeight, remoteHeight, tagsHeight := splitThreeHeights(height)
	localBox := m.getBoxStyle(sectionCurrent).Width(width).Height(localHeight).Render(
		"Local\n" + m.renderSectionContent(sectionCurrent, max(width-4, 0), max(localHeight-3, 0)),
	)
	remoteBox := m.getBoxStyle(sectionRemote).Width(width).Height(remoteHeight).Render(
		"Remote\n" + m.renderSectionContent(sectionRemote, max(width-4, 0), max(remoteHeight-3, 0)),
	)
	tagsBox := m.getBoxStyle(sectionTags).Width(width).Height(tagsHeight).Render(
		"Tags\n" + m.renderSectionContent(sectionTags, max(width-4, 0), max(tagsHeight-3, 0)),
	)
	return lipgloss.JoinVertical(lipgloss.Left, localBox, remoteBox, tagsBox)
}

func renderModalPopup(m model, bodyWidth int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	popupWidth := bodyWidth - 12
	if popupWidth > 54 {
		popupWidth = 54
	}
	if popupWidth < 32 {
		popupWidth = 32
	}

	popupBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(popupWidth).
		Align(lipgloss.Center)

	popupTitle := m.status.Title
	if popupTitle == "" || popupTitle == "Confirm" {
		popupTitle = "Continue?"
	}

	helpText := "y: yes  •  n: no"
	if m.status.Action == state.ActionPull && !m.pullIsFastForward {
		helpText = "m: merge  •  r: rebase  •  esc: cancel"
	} else if m.status.Mode == state.ModeResetModePick {
		helpText = "s: soft  •  m: mixed  •  h: hard  •  enter: execute  •  esc: back"
	}

	return popupBox.Render(
		titleStyle.Render(popupTitle) + "\n\n" +
			descStyle.Render(m.status.Detail) + "\n\n" +
			helpStyle.Render(helpText),
	)
}

func applyOuterMargins(content string, totalWidth, totalHeight, hMargin, topMargin, bottomMargin int) string {
	lines := strings.Split(content, "\n")
	leftPad := strings.Repeat(" ", hMargin)
	rightPad := strings.Repeat(" ", hMargin)
	blank := strings.Repeat(" ", totalWidth)

	out := make([]string, 0, totalHeight+topMargin+bottomMargin)
	for i := 0; i < topMargin; i++ {
		out = append(out, blank)
	}
	for _, line := range lines {
		out = append(out, leftPad+line+rightPad)
	}
	for i := 0; i < bottomMargin; i++ {
		out = append(out, blank)
	}
	return strings.Join(out, "\n")
}
```

### `internal/app/view_layout.go` popup overlay

popup 은 화면 정렬을 깨지 않도록 별도 오버레이 규칙을 둔다.

```go
func overlayPopup(base string, popup string) string {
	baseLines := strings.Split(base, "\n")
	popupLines := strings.Split(popup, "\n")
	if len(baseLines) < len(popupLines) {
		return base
	}

	popupW := 0
	for _, line := range popupLines {
		if w := lipgloss.Width(line); w > popupW {
			popupW = w
		}
	}

	startY := (len(baseLines) - len(popupLines)) / 2
	for i, popupLine := range popupLines {
		y := startY + i
		if y >= len(baseLines) {
			break
		}

		baseLine := baseLines[y]
		startX := (lipgloss.Width(baseLine) - popupW) / 2
		if startX < 0 {
			startX = 0
		}
		baseLines[y] = overlayLine(baseLine, popupLine, startX, popupW)
	}
	return strings.Join(baseLines, "\n")
}
```

### `internal/app/model_test.go`

핵심은 비율과 중앙 정렬을 문자열 레벨에서 고정하는 것이다.

```go
func TestShellLayoutUsesTenPercentMargins(t *testing.T) {
	m := model{width: 140, height: 60}
	hMargin, topMargin, bottomMargin := layoutShellMargins(m)
	if hMargin != 14 {
		t.Fatalf("expected horizontal margin to use 10%% of width, got %d", hMargin)
	}
	if topMargin != 6 {
		t.Fatalf("expected top margin to use 10%% of height, got %d", topMargin)
	}
	if bottomMargin != 6 {
		t.Fatalf("expected bottom margin to use 10%% of height, got %d", bottomMargin)
	}
}

func TestSplitPaneWidthsUseThreeSevenRatio(t *testing.T) {
	left, right := splitPaneWidths(100)
	if left != 30 || right != 70 {
		t.Fatalf("expected 3:7 split, got %d and %d", left, right)
	}
}

func TestRenderAppViewPlacesFooterAcrossFullWidth(t *testing.T) {
	m := model{
		width:  140,
		height: 60,
		status: state.New().WithBrowse(),
	}
	got := renderAppView(m)
	lines := strings.Split(got, "\n")
	lastVisible := ""
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lastVisible = line
	}
	if lastVisible == "" {
		t.Fatal("expected rendered output to contain visible content")
	}
	if w := lipgloss.Width(lastVisible); w != m.width {
		t.Fatalf("expected footer line to be placed across full width, got %d want %d", w, m.width)
	}
}
```

### 구현 시 지켜야 할 순서

1. `view_layout.go` 에서 수치와 비율을 먼저 고정한다.
2. `view_shell.go` 에서 `header -> main -> footer` 순서로 재배치한다.
3. popup 이 body 를 넘지 않도록 clamp 한다.
4. 테스트로 `10% margin`, `3:7 split`, `full-width footer` 를 고정한다.

## CURRENT

현재 실제 렌더링은 TO-BE 와 다르게 몇 가지가 동시에 어긋나 있다.
이 섹션은 코드의 의도보다 실제 스크린샷 기준을 우선한다.

- 상단 `Mode - Global / Mode - Local` 헤더가 안정적으로 보이지 않는다.
- 화면이 가운데로 안정적으로 모이지 않고, 좌상단 쪽으로 붙어 보인다.
- top margin 이 거의 체감되지 않는다.
- `Global / Context` 헤더가 TO-BE 처럼 분리되지 않는다.
- footer 와 본문의 기준 폭이 섞여 보이면서 전체 균형이 무너진다.
- `Graph / Local / Remote / Tags` 만 따로 보이고, 상단 컨텍스트 계층이 사라진 느낌이다.

스크린샷 `Screenshot 2026-06-30 at 16.22.53.png` 에서 보이는 현재 상태를 요약하면:

- 좌측 `Graph` 는 크지만, 화면 전체를 안정적으로 잡아주지 못한다.
- 우측 `Local / Remote / Tags` 는 세로 레일 형태는 유지되지만 전체 중심선과 맞지 않는다.
- 하단 footer 가 본문과 다른 폭 기준으로 계산되어 화면 하단에서 따로 노는 느낌이 있다.
- 결과적으로 “3:7 + centered frame” 이 아니라 “상단 컨텍스트가 빠진 채 박스 몇 개가 나열된 상태”로 보인다.

## 남은 작업

- [ ] relayout 작업 진행
  - [ ] 가운데 정렬 안됨 (+ top margin 미반영)
  - [ ] Global, Context Width 비율 3:7 구성

## 현재 관찰된 구현 상태

현재 코드에는 레이아웃 관련 함수가 분리되어 있다.

- `internal/app/view_shell.go`
- `internal/app/view_layout.go`
- `internal/app/model_test.go`

하지만 실제 렌더링과 계획 문서의 요구사항 사이에는 차이가 남아 있다.

확인된 현재 상태:

- `layoutShellMargins` 는 10% 기준으로 바꿨지만, 최종 shell 이 화면 중앙에 고정되는지에 대한 검증이 약하다.
- `splitPaneWidths` 는 3:7 로 바꿨지만, 실제 렌더링에서 헤더 폭과 본문 폭이 따로 계산되면 체감 비율이 다시 흔들릴 수 있다.
- `renderAppView` 는 body 를 만든 뒤 다시 `Place` 하고 있지만, footer 와 body 가 같은 기준선으로 묶였는지 재검토가 필요하다.
- 테스트는 margin / ratio 를 일부 확인하지만, 스크린샷처럼 체감되는 전체 균형까지는 고정하지 못한다.

## 원인 분석

### 1. 3:7 비율이 코드에 아직 반영되지 않았다

원래 원인은 `splitPaneWidths` 의 50:50 구현이었다.

이제 비율 자체는 3:7 로 바뀌었지만, 문제는 거기서 끝나지 않는다.

- 헤더 비율은 바뀌어도
- 본문 폭과 footer 폭이 서로 다른 기준으로 렌더링되면
- 사용자 눈에는 여전히 “레이아웃이 일관되지 않다”로 보인다.

즉, 비율을 바꾸는 것만으로는 TO-BE 가 완성되지 않는다.

### 2. 중앙 정렬은 margin 계산만으로 완성되지 않는다

`applyOuterMargins` 는 내부 프레임에 여백을 더하는 함수다.

이 방식은 여백 문자열을 더하는 데는 충분하지만,
전체 프레임을 화면 중앙 기준으로 다시 놓는 최종 배치 단계는 아니다.

즉, 계산된 마진이 있어도 실제 시각 결과가 “가운데 정렬된 레이아웃”으로 느껴지지 않을 수 있다.

### 3. top margin 은 계산되지만, 화면 검증이 약하다

`layoutShellMargins` 에서 top margin 을 계산하더라도,
렌더링 결과를 정확히 검증하는 테스트가 부족하면
margin 값은 남아 있어도 사용자가 보기에는 반영이 약하게 느껴질 수 있다.

현재 테스트는 다음 수준에 머문다.

- 첫 번째 visible line 이 공백으로 시작하는지
- 마지막 visible line 이 공백으로 시작하는지

이 검증만으로는 다음을 보장하지 못한다.

- 전체 shell 이 실제로 중앙에 놓이는지
- top margin 이 계획한 비율만큼 보이는지
- `Global / Context` 가 정확히 3:7 인지

### 4. 계획과 구현이 두 문서로 나뉘어 있었고, 최종 결론이 코드에 고정되지 않았다

초기 계획은 `centered layout`, 이후 계획은 `graph-first layout / mode panel split` 에서 이어졌다.

이 과정에서 화면 구조는 바뀌었지만,
최종적으로 고정해야 할 비율과 중앙 배치 규칙이 코드 수준 테스트로 굳어지지 않았다.

결과적으로 문서상 의도는 남아 있으나, 구현은 부분 반영 상태로 남았다.

## 정리해야 할 구현 포인트

1. `Mode - Global / Mode - Local` 헤더를 `3:7` 으로 고정한다.
2. `Graph` 와 우측 레일을 같은 main frame 안에서 묶는다.
3. `Local / Remote / Tags` 를 같은 우측 레일 규칙으로 쌓는다.
4. `renderAppView` 의 전체 shell 중앙 정렬을 유지한다.
5. top margin 과 footer full-width 정렬을 test 로 고정한다.
6. `3:7` 비율과 centered frame 을 test 로 고정한다.

## 테스트 보강 방향

다음과 같은 테스트가 필요하다.

- `renderAppView` 가 terminal 중앙에 배치되는지
- `Global / Context` 폭 비율이 3:7 인지
- top margin 이 실제 렌더 문자열에서 체감되는지
- 전체 레이아웃이 좌상단에 붙지 않는지
- footer 가 본문과 같은 중심축을 공유하는지
- 작은 화면에서도 비율이 과도하게 깨지지 않는지
- popup 이 body 폭을 넘지 않는지
- `Graph / Local / Remote / Tags` 가 메인 레일 구조를 유지하는지

## 결론

이번 이슈는 단순한 스타일 수정이 아니라,
계획 문서에 적힌 레이아웃 규칙을 코드와 테스트에 끝까지 고정하지 못한 상태로 보는 것이 맞다.

우선순위는 다음과 같다.

1. 가운데 정렬과 top margin 을 화면 단에서 확정한다.
2. `Global / Context` 를 3:7 로 변경한다.
3. 테스트로 두 규칙을 고정한다.
4. 메인 레일과 footer/popup 의 기준 폭을 분리한다.
