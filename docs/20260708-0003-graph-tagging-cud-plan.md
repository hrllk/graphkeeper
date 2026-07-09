# Graph Tagging CUD Plan

## 목적

`Graph` 에서 focus 된 commit 에 lightweight tag 를 생성한다.

이 문서는 CUD 전용이다. tag list inspection 은 `0002`에서 다룬다.

## 사용자 목표

1. 사용자는 `Graph` 에서 현재 focus 된 commit 에 tag 를 붙일 수 있어야 한다.
2. tag 이름 입력은 짧고 명확한 popup 으로 처리해야 한다.
3. tag 생성 후에는 repo 상태를 다시 읽어서 최신 목록으로 돌아와야 한다.
4. 실패 메시지는 숨기지 말고 popup 안에서 바로 보여줘야 한다.

## 범위

### 포함

- `Graph` 에서 tag popup 열기
- tag name 입력
- `git tag` 실행
- 성공 후 repo refresh
- empty / duplicate / stale / git failure 처리
- 관련 tests

### 제외

- tag list inspection
- tag grouping
- graph marker rendering
- tag rename
- tag delete
- annotated tag message editing

## 현재 상태

현재 코드는 tag 를 읽는 기능만 있고, 생성 진입점은 없다.

- `internal/git/repo.go` 에 tag 생성 메서드가 없다.
- `internal/app/key_handling_browse.go` 에 `Graph` tag shortcut 이 없다.
- `internal/app/view_shell.go` 에 tag popup 이 없다.
- `internal/app/model.go` 에 tag draft / error 상태가 없다.

즉, CUD 는 아직 설계만 있고 경로가 없다.

## 핵심 결정

### 1. tag 생성은 lightweight tag 로 시작한다

첫 버전은 annotation 없이 commit 에 바로 붙는 lightweight tag 로 둔다.

이유는 간단하다.

- 입력이 최소다.
- `Graph` 에서 빠르게 붙이기 좋다.
- list/view 계약과도 충돌이 적다.

### 2. `t` shortcut 은 `Graph` 에서만 활성화한다

tag 생성은 탐색 중인 commit 에 대한 액션이다.

그래서 전역 hotkey 가 아니라 `Graph` focus 에 종속된 shortcut 으로 둔다.

### 3. popup 은 이름 입력 + 대상 commit 확인 + error 메시지만 보여준다

popup 은 길면 안 된다.

필요한 정보는 세 개다.

- tag name draft
- target commit hash
- validation / git error

### 4. 성공하면 popup 을 닫고 repo 를 다시 읽는다

생성 후 중간 상태를 캐시하지 않는다.

`git tag` 실행 뒤 `Status()` 를 다시 읽어서 같은 commit 위치를 유지한다.

## BEFORE

현재는 생성 경로가 없다.

```go
type model struct {
    // ...
    // tag popup state 없음
}
```

```go
func handleGraphKey(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "t":
        // no-op
    }
    return m, nil
}
```

```go
func (r *Repo) CreateTag(ctx context.Context, name, target string) error {
    return nil
}
```

## AFTER

### Git layer

```go
package git

func (r *Repo) CreateTag(ctx context.Context, name, target string) error {
    name = strings.TrimSpace(name)
    target = strings.TrimSpace(target)

    if name == "" {
        return fmt.Errorf("tag name is required")
    }
    if target == "" {
        return fmt.Errorf("tag target is required")
    }

    _, err := r.git(ctx, "tag", "--", name, target)
    if err != nil {
        return err
    }
    return nil
}
```

### Model state

```go
type model struct {
    // ...
    tagPopupOpen   bool
    tagPopupDraft  string
    tagPopupError  string
    tagPopupTarget string
}
```

```go
func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    if m.tagPopupOpen {
        return m.handleTagPopupKey(msg)
    }
    if m.stashPopupOpen {
        return m.handleStashPopupKey(msg)
    }
    if m.branchOpen {
        return m.handleBranchOpenKey(msg)
    }
    if m.graphSearchOpen {
        return m.handleGraphSearchKey(msg)
    }
    switch m.status.Mode {
    case state.ModeTargetPick:
        return m.handleTargetPickKey(msg)
    case state.ModeConfirm:
        return m.handleConfirmKey(msg)
    case state.ModeReview:
        return m.handleReviewKey(msg)
    case state.ModeResetModePick:
        return m.handleResetModePickKey(msg)
    case state.ModeOutcomePreview:
        return m.handleOutcomePreviewKey(msg)
    case state.ModeBlocked:
        return m.handleBlockedKey(msg)
    case state.ModeBrowse:
        return m.handleBrowseKey(msg)
    default:
        return m, nil
    }
}
```

### Graph shortcut

```go
func (m model) handleGraphKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "t":
        focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
        if focus.Hash == "" {
            return m, nil
        }
        m.tagPopupOpen = true
        m.tagPopupDraft = ""
        m.tagPopupError = ""
        m.tagPopupTarget = focus.Hash
        return m, nil
    }
    return m, nil
}
```

### Popup input handling

```go
func (m model) handleTagPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.Type {
    case tea.KeyEsc:
        m.tagPopupOpen = false
        m.tagPopupDraft = ""
        m.tagPopupError = ""
        m.tagPopupTarget = ""
        return m, nil

    case tea.KeyEnter:
        name := strings.TrimSpace(m.tagPopupDraft)
        if name == "" {
            m.tagPopupError = "tag name is required"
            return m, nil
        }
        if m.tagPopupTarget == "" {
            m.tagPopupError = "target commit is missing"
            return m, nil
        }
        m.status = loadingToast("Tagging commit...")
        return m, executeCreateTag(m.repo, name, m.tagPopupTarget, graphPageSize(&m))

    case tea.KeyBackspace, tea.KeyDelete:
        if len(m.tagPopupDraft) > 0 {
            m.tagPopupDraft = m.tagPopupDraft[:len(m.tagPopupDraft)-1]
        }
        return m, nil
    }

    if r := msg.Runes; len(r) == 1 {
        m.tagPopupDraft += string(r[0])
    }
    return m, nil
}
```

### Tag command

```go
type tagCreatedMsg struct {
    Target string
    Status git.Status
    Err    error
}

type tagToastDoneMsg struct{}

func executeCreateTag(repo *git.Repo, name, target string, limit int) tea.Cmd {
    return func() tea.Msg {
        if err := repo.CreateTag(context.Background(), name, target); err != nil {
            return tagCreatedMsg{Target: target, Err: err}
        }
        status, err := repo.Status(context.Background(), limit)
        return tagCreatedMsg{Target: target, Status: status, Err: err}
    }
}
```

### Success / failure apply

```go
func (m model) handleTagCreatedMsg(msg tagCreatedMsg) (model, tea.Cmd) {
    if msg.Err != nil {
        m.tagPopupError = msg.Err.Error()
        return m, nil
    }

    m.repoStatus = msg.Status
    m.tagPopupOpen = false
    m.tagPopupDraft = ""
    m.tagPopupError = ""

    rows := graph.Rows(m.repoStatus)
    if row := graph.FindRowByHash(rows, msg.Target); row >= 0 {
        m.activeSection = sectionGraph
        m.sectionCursor[sectionGraph] = row
        m.graphLaneCursor = graph.PointerLane(rows[row])
        m.graphScroll = clampScroll(row, len(rows), graphPageSizeForRows(&m, rows, row, graphContentHeightForModel(&m)))
    }
    m.status = loadingToast("Tag created.")
    return m, tea.Tick(900*time.Millisecond, func(time.Time) tea.Msg {
        return tagToastDoneMsg{}
    })
}
```

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tagCreatedMsg:
        return m.handleTagCreatedMsg(msg)
    case tagToastDoneMsg:
        m.status = deriveStatus(m.repoStatus)
        return m, nil
    case tea.WindowSizeMsg:
        return handleWindowSize(m, msg)
    case loadedMsg, refreshedMsg, tickMsg:
        return handleLifecycleUpdate(m, msg)
    case stashLoadedMsg:
        return handleStashUpdate(m, msg)
    case fetchedMsg, preparedMsg, pullCheckedMsg, previewMsg, graphActionCheckMsg, pushFetchedMsg, pullFetchedMsg, pullPreviewReadyMsg, pullToastDoneMsg, branchToastDoneMsg:
        return handleFetchUpdate(m, msg)
    case executedMsg:
        return handleExecutedUpdate(m, msg)
    case createdBranchMsg:
        return handleBranchUpdate(m, msg)
    case tea.KeyMsg:
        return m.handleKeyMsg(msg)
    default:
        return m, nil
    }
}
```

### Popup render

```go
func renderTagPopup(m model, bodyWidth, bodyHeight int) string {
    popupWidth := popupWidthForBody(bodyWidth, 36, 56)
    popupBox := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("205")).
        Padding(1, 2).
        Width(popupWidth).
        Align(lipgloss.Left)

    lines := []string{
        title.Render("Create tag"),
        "",
        fmt.Sprintf("target: %s", shorten(m.tagPopupTarget, 8)),
        fmt.Sprintf("name: %s", m.tagPopupDraft),
    }
    if m.tagPopupError != "" {
        lines = append(lines, "")
        lines = append(lines, warn.Render(m.tagPopupError))
    }
    lines = append(lines, "")
    lines = append(lines, "enter: create  •  esc: cancel")
    return renderFloatingTitlePopup(popupBox, "Tag commit", strings.Join(lines, "\n"), popupWidth)
}
```

```go
func renderAppView(m model) string {
    // ... existing shell layout ...
    centeredBody := applyOuterMargins(body, bodyWidth, bodyHeight, hMargin, topMargin, max(bottomMargin-1, 0))

    if m.status.Mode == state.ModeTargetPick {
        centeredBody = overlayPopup(centeredBody, renderTargetPickPopup(m, bodyWidth))
    }
    if m.branchOpen {
        centeredBody = overlayPopup(centeredBody, renderBranchInputPopup(m, bodyWidth))
    }
    if m.stashPopupOpen {
        centeredBody = overlayPopup(centeredBody, renderStashPopup(m, bodyWidth, bodyHeight))
    }
    if m.tagPopupOpen {
        centeredBody = overlayPopup(centeredBody, renderTagPopup(m, bodyWidth, bodyHeight))
    }
    if m.graphSearchOpen {
        centeredBody = overlayPopup(centeredBody, renderGraphSearchPopup(m, bodyWidth))
    }
    if m.status.Mode == state.ModeLoading && !m.branchOpen && !m.tagPopupOpen {
        centeredBody = overlayPopup(centeredBody, renderLoadingPopup(m, bodyWidth))
    }
    if m.status.Mode == state.ModeBlocked && !m.branchOpen && !m.tagPopupOpen {
        centeredBody = overlayPopup(centeredBody, renderAlertPopup(blockedAlertContent(m.status), bodyWidth))
    }
    return centeredBody
}
```

## Tests

```go
func TestOpenTagPopupFromGraphFocus(t *testing.T)
func TestTagPopupRejectsEmptyName(t *testing.T)
func TestTagPopupRejectsMissingTarget(t *testing.T)
func TestCreateTagExecutesGitTagAndRefreshesStatus(t *testing.T)
func TestCreateTagFailureStaysInPopup(t *testing.T)
func TestTagPopupEscClosesWithoutSideEffects(t *testing.T)
```

검증 포인트는 다음이다.

- `Graph` 에서만 `t` 가 tag popup 을 여는지
- empty tag name 을 즉시 막는지
- `git tag` 실패 메시지가 popup 에 남는지
- 성공 후 repo refresh 가 일어나는지
- popup 이 닫힌 뒤 focus 가 같은 commit 에 남는지

## Verification

```sh
go test ./internal/app ./internal/git
```

## Notes

- create 는 read/list 와 분리해야 한다.
- 성공 후에는 저장된 draft 를 남기지 않는다.
- tag 생성 실패는 silent fallback 없이 그대로 보여준다.
