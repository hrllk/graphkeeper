# Panel Title Overlap Implementation Plan

## 목적

현재 카드형 패널은 제목이 모두 카드 내부 첫 줄에 들어간다.
이 문서는 제목을 카드 내부가 아니라 상단 border 와 겹치게 보이도록 바꾸는 구현 계획이다.

핵심 목표는 다음과 같다.

1. 제목이 border 위에 얹힌 것처럼 보이게 한다.
2. 기존 패널의 width / height 계약은 최대한 유지한다.
3. 공통 helper 로 묶어서 모든 카드형 패널에 같은 규칙을 적용한다.
4. 기존 렌더링 로직과 데이터 로직은 건드리지 않는다.

## 현재 사실

현재 main shell 렌더링은 `internal/app/view_shell.go` 에 있다.

- `Global`, `Context`, `Graph` 는 `Render("Title\n" + content)` 방식이다.
- `Local`, `Remote`, `Tags` 도 같은 방식이다.
- confirm / loading / branch input / alert popup 도 같은 패턴이다.

현재 관련 코드 위치는 다음과 같다.

- `internal/app/view_shell.go`
- `internal/app/view_sections.go`
- `internal/app/view_detail.go`

현재 helper 는 title 을 card 내부 첫 줄로 넣는 방식이다.

```go
globalBox := baseBox.Width(globalWidth).Height(headerHeight).Render(
	"Global\n" + m.renderGlobalContent(max(globalWidth-4, 0), max(headerHeight-3, 0)),
)
```

```go
localBox := m.getBoxStyle(sectionCurrent).Width(width).Height(localHeight).Render(
	"Local\n" + m.renderSectionContent(sectionCurrent, max(width-4, 0), max(localHeight-3, 0)),
)
```

```go
return popupBox.Render(
	titleStyle.Render(popupTitle) + "\n\n" +
		descStyle.Render(m.status.Detail) + "\n\n" +
		helpStyle.Render(helpText),
)
```

즉, 제목을 border 위로 빼려면 `Render("Title\n...")` 를 그대로 유지할 수 없다.

## 범위

### 포함

- main shell 카드의 title layout 변경
- popup 카드의 title layout 변경
- 공통 title frame helper 추가
- 관련 테스트 업데이트

### 제외

- 상태 전이 변경
- action 의미 변경
- graph / navigation 규칙 변경
- inline 텍스트 heading 자체의 재설계

여기서 말하는 대상은 `Render("Title\n" + body)` 로 그려지는 카드형 패널이다.
`Details`, `Actions`, `Repo` 같이 일반 텍스트로 들어가는 섹션 라벨은 이번 계획의 1차 대상이 아니다.

## 구현 원칙

### 1. title 은 border line 에 들어가야 한다

새 helper 는 title 을 카드 첫 줄이 아니라 상단 border line 에 넣는다.

예상 모양은 다음과 같다.

```text
╭── Global ─────────────────────────╮
│ Mode: Browse | ...                │
│ Actions                           │
│ • tab: next section               │
╰────────────────────────────────────╯
```

중요한 점은 title 이 body 내부 한 줄을 차지하지 않는다는 것이다.

### 2. body height 는 기존보다 1줄 더 쓸 수 있어야 한다

현재는 title 이 body 첫 줄이므로 content height 계산에서 title 1줄을 빼고 있다.
title 을 border line 으로 옮기면, title 은 body 내부 줄을 소비하지 않는다.

그래서 각 call site 의 body height 계산은 `-3` 기준에서 `-2` 기준으로 바뀌어야 한다.

예시:

```go
// before
contentHeight := max(headerHeight-3, 0)

// after
contentHeight := max(headerHeight-2, 0)
```

이 규칙은 패널 내부에 padding 이 없는 카드에 우선 적용한다.
popup 계열은 padding 이 있지만, 동일하게 "title 1줄이 body 내부에서 빠진다"는 contract 를 따른다.

### 3. 공통 helper 는 모든 카드형 패널에서 재사용해야 한다

패널마다 따로 title-overlap 렌더링을 만들지 말고, 공통 helper 하나를 둔다.

권장 helper 형태:

```go
func renderFloatingTitleFrame(style lipgloss.Style, title string, body string, width, height int) string
func renderFloatingTitlePopup(style lipgloss.Style, title string, body string, width int) string
```

이 helper 는 다음을 책임진다.

- title strip 생성
- body card 생성
- title strip 과 body 를 위아래로 결합
- width / height clamp
- title truncation

## 구현 설계

### 1. 공통 title strip helper 를 만든다

상단 border line 에 title 을 넣는 helper 를 분리한다.

```go
func renderTitleStrip(style lipgloss.Style, title string, width int) string {
	border, borderTop, _, _, _ := style.GetBorder()
	if !borderTop || width <= 0 {
		return fitVisibleWidth(title, width)
	}

	title = strings.TrimSpace(title)
	if title == "" {
		title = " "
	}

	// title 이 너무 길면 border 를 깨지 않도록 먼저 자른다.
	available := width - lipgloss.Width(border.TopLeft) - lipgloss.Width(border.TopRight) - 4
	if available < 1 {
		available = 1
	}
	title = fitVisibleWidth(title, available)

	// border 상단은 title 을 중앙에 두고 양쪽을 border rune 으로 채운다.
	base := width - lipgloss.Width(border.TopLeft) - lipgloss.Width(border.TopRight)
	if base < 0 {
		base = 0
	}
	titleWidth := lipgloss.Width(title) + 2
	fillWidth := base - titleWidth
	if fillWidth < 0 {
		fillWidth = 0
	}
	leftFill := fillWidth / 2
	rightFill := fillWidth - leftFill

	return border.TopLeft +
		strings.Repeat(border.Top, leftFill) +
		" " + title + " " +
		strings.Repeat(border.Top, rightFill) +
		border.TopRight
}
```

이 helper 는 border rune 을 직접 조립하므로, `lipgloss` 에 title-on-border 전용 API 가 없어도 동작한다.

### 2. body renderer 는 top border 를 비워둔 스타일로 렌더한다

title strip 이 top border 를 담당하므로, body renderer 는 top border 를 비운 스타일을 사용한다.

고정 높이 패널은 다음처럼 렌더한다.

```go
func renderFloatingTitleFrame(style lipgloss.Style, title string, body string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	titleLine := renderTitleStrip(style, title, width)
	bodyStyle := style.BorderTop(false)
	bodyLineBudget := height - 1
	if bodyLineBudget < 1 {
		bodyLineBudget = 1
	}

	bodyBlock := bodyStyle.Height(bodyLineBudget).Width(width).Render(body)
	return titleLine + "\n" + bodyBlock
}
```

popup 같이 natural height 를 쓰는 패널은 별도 helper 로 처리한다.

```go
func renderFloatingTitlePopup(style lipgloss.Style, title string, body string, width int) string {
	if width <= 0 {
		return ""
	}

	titleLine := renderTitleStrip(style, title, width)
	bodyStyle := style.BorderTop(false)
	bodyBlock := bodyStyle.Width(width).Render(body)
	return titleLine + "\n" + bodyBlock
}
```

실제 구현에서는 `Width()` / `Height()` 호출이 기존 style contract 와 충돌하지 않도록, 현재 call site 가 넘기던 width / height 값을 그대로 받아서 내부에서만 조정한다.

### 3. call site 를 교체한다

`internal/app/view_shell.go` 의 다음 지점을 순차적으로 교체한다.

#### main shell cards

```go
globalBox := renderFloatingTitleFrame(
	baseBox.Width(globalWidth).Height(headerHeight),
	"Global",
	m.renderGlobalContent(max(globalWidth-4, 0), max(headerHeight-2, 0)),
	globalWidth,
	headerHeight,
)
```

```go
contextBox := renderFloatingTitleFrame(
	baseBox.Width(contextWidth).Height(headerHeight),
	"Context",
	m.renderContextContent(max(contextWidth-4, 0), max(headerHeight-2, 0)),
	contextWidth,
	headerHeight,
)
```

```go
graphBox := renderFloatingTitleFrame(
	m.getBoxStyle(sectionGraph).Width(graphWidth).Height(graphRailHeight),
	"Graph",
	m.renderGraphContent(max(graphWidth-4, 0), graphContentHeight),
	graphWidth,
	graphRailHeight,
)
```

#### right rail cards

```go
localBox := renderFloatingTitleFrame(
	m.getBoxStyle(sectionCurrent).Width(width).Height(localHeight),
	"Local",
	m.renderSectionContent(sectionCurrent, max(width-4, 0), max(localHeight-2, 0)),
	width,
	localHeight,
)
```

같은 방식으로 `Remote`, `Tags` 도 바꾼다.

#### popup cards

```go
return renderFloatingTitlePopup(
	popupBox,
	popupTitle,
	strings.Join([]string{
		descStyle.Render(m.status.Detail),
		helpStyle.Render(helpText),
	}, "\n\n"),
	popupWidthForBody(bodyWidth, 32, 54),
)
```

popup 은 현재 body 를 직접 `Render()` 에 넣고 있으므로, 먼저 body string 을 분리한 뒤 helper 로 넘기는 형태가 가장 단순하다.

### 4. title 이 body content 에 남지 않게 한다

title 을 바깥으로 빼면 기존 body builders 의 첫 줄 title 은 삭제해야 한다.

예시:

```go
// before
lines := []string{
	title.Render("Actions"),
	"• tab: next section",
}

// after
lines := []string{
	"• tab: next section",
}
```

이 작업이 필요한 파일은 다음과 같다.

- `internal/app/view_sections.go`
- `internal/app/view_detail.go`
- `internal/app/view_shell.go`

## 상세 적용 순서

### `internal/app/view_shell.go`

1. `renderFloatingTitleFrame` 와 `renderTitleStrip` 을 추가한다.
2. `Global`, `Context`, `Graph` call site 를 교체한다.
3. popup 렌더링을 body string + helper 조합으로 바꾼다.

### `internal/app/view_sections.go`

1. `renderGlobalContent()` 에서 내부 title line 을 제거한다.
2. `renderContextContent()` 의 내부 title line 을 제거한다.
3. `renderSplitColumns()` 에 넘기는 left / right lines 는 body 전용으로 만든다.

### `internal/app/view_detail.go`

1. `renderDetailContent()` 에서 `Repo`, `Actions` 같은 내부 title line 을 제거한다.
2. 이 파일이 outer card title 을 직접 그리지 않도록 정리한다.

## 테스트 계획

다음 테스트를 추가하거나 수정한다.

### 1. title 이 body 내부가 아니라 border line 에 있는지 확인

```go
func TestRenderAppViewPlacesPanelTitleOnBorder(t *testing.T) {
	m := model{
		width: 140,
		height: 60,
		status: state.New().WithBrowse(),
	}

	got := renderAppView(m)
	if strings.Contains(got, "\nGlobal\n") {
		t.Fatal("expected Global title to move out of body")
	}
	if !strings.Contains(got, "Global") {
		t.Fatal("expected Global title to remain visible")
	}
}
```

### 2. body height 가 1줄 늘어나는지 확인

```go
func TestFloatingTitleCardKeepsHeightContract(t *testing.T) {
	// title line 이 body 를 먹지 않는지 확인한다.
}
```

### 3. popup title 도 같은 helper 를 쓰는지 확인

popup title 은 `Confirm`, `Reset mode`, `Working...`, `Create branch`, `Alert` 의 visual 형태가 동일해야 한다.

### 4. 좁은 width 에서 title 이 truncate 되는지 확인

title 이 너무 길면 border 를 깨지 않고 잘려야 한다.

예상 검증:

- visible width 가 target width 를 넘지 않는다.
- ANSI 가 들어가도 줄 길이는 맞는다.
- `fitVisibleWidth()` 를 재사용한다.

## 완료 기준

1. main shell 카드의 title 이 border line 과 겹쳐 보인다.
2. popup 카드의 title 도 같은 규칙을 따른다.
3. title 이 body 첫 줄로 남지 않는다.
4. width / height 계약이 유지된다.
5. 관련 테스트가 통과한다.

## 메모

이 변경은 데이터 로직 변경이 아니라 렌더링 계약 변경이다.
따라서 구현 시에는 `state`, `actions`, `navigation` 쪽을 건드리지 말고 `view_*` 계층에만 국한하는 것이 맞다.
