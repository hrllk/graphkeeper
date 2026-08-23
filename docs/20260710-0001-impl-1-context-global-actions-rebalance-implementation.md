# 20260710-0001 Impl 1

`~/.gstack/graphkeeper/20260710-0001-context-global-actions-rebalance-plan.md` 만 보고도 바로 구현할 수 있게 정리한 실행 문서다.

이 문서는 아래 목표를 실제 코드 단위로 쪼갠다.

- `Global` / `Context` 분리
- `Graph` action 4개 기본 노출
- `?` 를 hidden hotkey drawer 로 전환
- `scroll`, `top`, `bottom` 을 Global 로 이동
- `Local` / `Remote` / `Tags` 의 action 밀도 정리

## 구현 범위

이 문서는 1차 구현만 다룬다.

- 포함
  - hotkey help copy 재배치
  - `?` drawer 추가
  - Graph search 와 `?` 분리
  - 섹션별 actions helper 분리
- 제외
  - `Remote` sync status 상태 수집
  - `Tags` tagger / tagged time 상태 수집
  - git 동작 변경

## 파일별 수정 순서

1. [`internal/app/model.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/model.go ) 에 drawer 상태를 추가한다.
2. [`internal/app/key_handling.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/key_handling.go ) 에 drawer 우선 처리기를 연결한다.
3. [`internal/app/key_handling_browse.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/key_handling_browse.go ) 에서 `?` 를 search 에서 분리한다.
4. [`internal/app/view_shell.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go ) 에 drawer overlay 를 추가한다.
5. [`internal/app/view_detail.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_detail.go ) 의 Global / Context copy 를 바꾼다.
6. [`internal/app/view_sections.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_sections.go ) 에서 섹션별 action helper 로 쪼갠다.

## 1. Model 상태 추가

`?` drawer 는 일반 modal 과 별개 상태로 둔다. 검색 팝업과 상태를 공유하지 않는다.

```go
type model struct {
    repo                 *git.Repo
    status               state.Status
    repoStatus           git.Status
    tagEntries           []git.TagEntry
    tagSyncAttempted     bool
    stashEntries         []git.StashEntry
    stashByBase          map[string][]git.StashEntry
    stashMessageOpen     bool
    stashMessageDraft    string
    stashMessageError    string
    stashPopupOpen       bool
    stashPopupCursor     int
    graphStashPopOpen    bool
    graphStashPopMode    graphStashPopMode
    graphStashPopCursor  int
    graphStashPopEntries []git.StashEntry
    tagPopupOpen         bool
    tagPopupDraft        string
    tagPopupError        string
    tagPopupTarget       string
    activeSection        graphSection
    sectionCursor        map[graphSection]int
    graphLaneCursor      int
    graphScroll          int
    graphSearchOpen      bool
    graphSearchDraft     string
    graphSearchQuery     string
    graphSearchIndex     []graphSearchEntry
    graphSearchCursor    int
    graphSearchError     string
    hiddenHotkeysOpen    bool
    awaitingGoTop        bool
    branchOpen           bool
    branchDraft          string
    branchBase           string
    branchError          string
    width                int
    height               int
    commitLimit          int
    err                  error
    handshakeCommits     map[string]bool
    pullIsFastForward    bool
}
```

### 이유

- `graphSearchOpen` 은 검색 전용 상태다.
- `hiddenHotkeysOpen` 은 help / overlay 전용 상태다.
- 둘을 분리해야 `?` 와 `/` 가 섞이지 않는다.

## 2. Key handling 연결

`?` 는 browse 모드에서만 drawer 를 연다. 다른 popup 이 열려 있을 때는 기존 popup 이 우선한다.

### 2-1. 최상위 분기

[`internal/app/key_handling.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/key_handling.go ) 에 drawer 우선 분기를 넣는다.

```go
func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    if m.graphStashPopOpen {
        return m.handleGraphStashPopKey(msg)
    }
    if m.stashMessageOpen {
        return m.handleStashMessageKey(msg)
    }
    if m.tagPopupOpen {
        return m.handleTagPopupKey(msg)
    }
    if m.stashPopupOpen {
        return m.handleStashPopupKey(msg)
    }
    if m.branchOpen {
        return m.handleBranchOpenKey(msg)
    }
    if m.hiddenHotkeysOpen {
        return m.handleHiddenHotkeysKey(msg)
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

### 2-2. Drawer key handler

새 파일을 만들거나 기존 browse key 파일에 붙인다.

```go
func (m model) handleHiddenHotkeysKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "esc", "?":
        m.hiddenHotkeysOpen = false
        return m, nil
    default:
        return m, nil
    }
}
```

### 2-3. Browse key 에서 `?` 분리

[`internal/app/key_handling_browse.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/key_handling_browse.go ) 의 Graph 분기에서 `?` 를 제거하고 `?` drawer 를 연다.

```go
func (m model) handleBrowseGlobalKey(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c", "q":
        return true, m, tea.Quit
    case "1":
        m = switchBrowseSection(m, sectionGraph)
        return true, m, nil
    case "2":
        m = switchBrowseSection(m, sectionCurrent)
        return true, m, nil
    case "3":
        m = switchBrowseSection(m, sectionRemote)
        return true, m, nil
    case "4":
        m = switchBrowseSection(m, sectionTags)
        return true, m, nil
    case "f":
        m.status = loadingToast("Fetching sources...")
        return true, m, fetchRepoState(m.repo, m.commitLimit)
    case "F":
        m.status = loadingToast("Fetching tags...")
        return true, m, fetchTagsRepoState(m.repo, m.commitLimit)
    case "S":
        m.stashPopupOpen = true
        return true, m, nil
    case "?":
        if m.status.Mode == state.ModeBrowse {
            m.hiddenHotkeysOpen = true
            return true, m, nil
        }
        return false, m, nil
    case "tab":
        m.activeSection = nextGraphSection(m.activeSection)
        return true, m, nil
    case "shift+tab":
        m.activeSection = prevGraphSection(m.activeSection)
        return true, m, nil
    case "up", "k":
        m = moveBrowseCursor(m, -1)
        return true, m, nil
    case "down", "j":
        m = moveBrowseCursor(m, 1)
        return true, m, nil
    case "g":
        if m.activeSection == sectionGraph {
            if m.awaitingGoTop {
                m.sectionCursor[sectionGraph] = 0
                m.graphScroll = 0
                rows := graph.Rows(m.repoStatus)
                if len(rows) > 0 {
                    m.graphLaneCursor = graph.PointerLane(rows[0])
                }
                m.awaitingGoTop = false
                return true, m, nil
            }
            m.awaitingGoTop = true
        }
        return true, m, nil
    case "G":
        if m.activeSection == sectionGraph {
            rows := graph.Rows(m.repoStatus)
            if len(rows) > 0 {
                last := len(rows) - 1
                m.sectionCursor[sectionGraph] = last
                m.graphScroll = clampScroll(last, len(rows), graphPageSize(&m))
                m.graphLaneCursor = graph.PointerLane(rows[last])
            }
            m.awaitingGoTop = false
        }
        return true, m, nil
    case "ctrl+u":
        if m.activeSection == sectionGraph {
            m = pageBrowseGraph(m, -1)
        }
        return true, m, nil
    case "ctrl+d":
        if m.activeSection == sectionGraph {
            m = pageBrowseGraph(m, 1)
        }
        return true, m, nil
    default:
        return false, m, nil
    }
}
```

Graph 전용 handler 에서는 `"/"` 만 search 로 남긴다.

```go
func (m model) handleBrowseGraphKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "esc":
        if strings.TrimSpace(m.graphSearchQuery) != "" || m.graphSearchOpen {
            m.graphSearchOpen = false
            m.graphSearchDraft = ""
            m.graphSearchQuery = ""
            m.graphSearchCursor = 0
            m.graphSearchError = ""
            return m, nil
        }
        return m, nil
    case "/":
        m.graphSearchOpen = true
        m.graphSearchDraft = m.graphSearchQuery
        m.graphSearchIndex = buildGraphSearchIndex(m.repoStatus)
        m.graphSearchError = ""
        return m, nil
    case "n":
        if strings.TrimSpace(m.graphSearchQuery) != "" {
            return applyGraphSearchRepeat(m, 1), nil
        }
        base := branchCreateBaseForActiveSection(m)
        m, _ = startBranchCreateInput(m, base)
        return m, nil
    case "N":
        if strings.TrimSpace(m.graphSearchQuery) != "" {
            return applyGraphSearchRepeat(m, -1), nil
        }
        return m, nil
    case "t":
        // existing tag popup flow stays the same
    case "o":
        // existing graph stash pop flow stays the same
    case "m":
        // existing merge flow stays the same
    case "r":
        // existing rebase flow stays the same
    case "s":
        // existing reset flow stays the same
    case "space", " ":
        // existing checkout flow stays the same
    case "d":
        // existing delete branch flow stays the same
    default:
        return m, nil
    }
}
```

### 핵심 포인트

- `?` 는 `handleBrowseGlobalKey` 로 끌어올린다.
- `/?` 를 같은 switch 에 두지 않는다.
- `graphSearchOpen` 은 검색 전용으로 유지한다.

## 3. Drawer 렌더링

[`internal/app/view_shell.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_shell.go ) 에서 다른 overlay 와 같은 층에 둔다.

### 3-1. overlay 추가

```go
func renderAppView(m model) string {
    // existing shell composition

    if m.hiddenHotkeysOpen {
        centeredBody = overlayPopup(centeredBody, renderHiddenHotkeysPopup(m, bodyWidth))
    }

    if m.status.Mode == state.ModeLoading && !m.branchOpen {
        centeredBody = overlayPopup(centeredBody, renderLoadingPopup(m, bodyWidth))
    }
    if m.status.Mode == state.ModeBlocked && !m.branchOpen {
        centeredBody = overlayPopup(centeredBody, renderAlertPopup(blockedAlertContent(m.status), bodyWidth))
    }

    shell := centeredBody + "\n"
    return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, shell)
}
```

### 3-2. drawer popup

```go
func renderHiddenHotkeysPopup(m model, bodyWidth int) string {
    descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
    helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
    popupWidth := popupWidthForBody(bodyWidth, 42, 72)
    popupBox := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("205")).
        Padding(1, 2).
        Width(popupWidth).
        Align(lipgloss.Left)

    lines := []string{
        descStyle.Render(sectionName(m.activeSection) + " hidden hotkeys"),
        "",
    }
    lines = append(lines, renderHiddenHotkeyGroup("Visible", hiddenVisibleHotkeys(m))...)
    lines = append(lines, "")
    lines = append(lines, renderHiddenHotkeyGroup("Conditional", hiddenConditionalHotkeys(m))...)
    lines = append(lines, "")
    lines = append(lines, renderHiddenHotkeyGroup("Hidden / moved out", hiddenMovedOutHotkeys(m))...)
    lines = append(lines, "")
    lines = append(lines, helpStyle.Render("esc: close"))

    return renderFloatingTitlePopup(
        popupBox,
        "Hidden Hotkeys",
        strings.Join(lines, "\n"),
        popupWidth,
    )
}
```

### 3-3. drawer group helper

```go
func renderHiddenHotkeyGroup(title string, items []string) []string {
    lines := []string{title + ":"}
    if len(items) == 0 {
        lines = append(lines, "  (none)")
        return lines
    }
    for _, item := range items {
        lines = append(lines, "  • "+item)
    }
    return lines
}
```

## 4. Drawer 내용 규칙

drawer 는 시각적으로 길게 설명하지 말고, 각 섹션의 "왜 안 보이는지"만 짧게 말한다.

### 4-1. Graph

```go
func hiddenVisibleHotkeys(m model) []string {
    switch m.activeSection {
    case sectionGraph:
        return []string{
            "m: merge",
            "r: rebase",
            "space: checkout",
            "H: jump to HEAD",
        }
    case sectionCurrent:
        return []string{
            "s: stash changes",
            "c: clean working tree",
            "space: checkout",
            "d: delete branch",
        }
    case sectionRemote:
        return []string{
            "space: checkout",
            "f: fetch",
            "p: pull",
            "d: delete branch",
        }
    case sectionTags:
        return []string{
            "enter: jump to graph",
            "d: delete tag",
        }
    default:
        return nil
    }
}

func hiddenConditionalHotkeys(m model) []string {
    switch m.activeSection {
    case sectionGraph:
        return []string{
            "s: reset",
            "d: delete branch",
            "p: pull",
            "P: push",
            "t: tag commit",
            "o: pop stash",
            "n: new branch or repeat search",
            "N: repeat search backward",
        }
    case sectionCurrent:
        return []string{
            "a: abort merge",
            "P: push",
        }
    case sectionRemote:
        return []string{
            "none",
        }
    case sectionTags:
        return []string{
            "none",
        }
    default:
        return nil
    }
}

func hiddenMovedOutHotkeys(m model) []string {
    return []string{
        "tab: next section",
        "shift+tab: previous section",
        "j/k: move",
        "f: fetch",
        "F: fetch tags",
        "S: stash list",
        "q: quit",
        "gg: top",
        "G: bottom",
        "ctrl+u/d: scroll",
    }
}
```

### 4-2. 주의점

- `Visible` 는 지금 섹션에서 즉시 읽히는 핵심만 둔다.
- `Conditional` 은 상태가 맞아야 의미가 있는 것만 둔다.
- `Hidden / moved out` 에는 Global 로 옮긴 조작을 넣는다.
- `n/N` 은 검색 반복과 브랜치 생성의 이중 의미가 있으므로 drawer 에 이유를 같이 적는다.

## 5. Global / Context copy 정리

[`internal/app/view_detail.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_detail.go ) 의 Global copy 는 현재 코드와 문서의 표현을 맞춘다.

```go
func (m model) renderGlobalContent(width, height int) string {
    if height <= 0 {
        return ""
    }
    lines := make([]string, 0, height)
    if m.branchOpen {
        lines = append(lines, "")
    } else if m.status.Mode == state.ModeLoading || m.status.Mode == state.ModeBlocked {
        lines = append(lines, "")
    } else {
        lines = append(lines, "Mode: "+renderStatusCompact(m.status))
    }
    lines = append(lines, "")
    lines = append(lines, title.Render("Actions"))
    lines = append(lines, "• tab: next section")
    lines = append(lines, "• shift+tab: previous section")
    lines = append(lines, "• j/k: move")
    lines = append(lines, "• f: fetch")
    lines = append(lines, "• F: fetch tags")
    lines = append(lines, "• S: stash list")
    lines = append(lines, "• q: quit")
    lines = append(lines, "• ?: show hidden hotkeys")
    lines = fitBlockWidth(lines, width)
    return fitBlockLines(lines, height)
}
```

Context 쪽은 섹션별 details + actions 를 유지하되, action helper 를 섹션별로 분리한다.

```go
func (m model) renderContextContent(width, height int) string {
    if height <= 0 {
        return ""
    }
    sectionTitle := sectionName(m.activeSection)
    leftLines := append([]string{title.Render(sectionTitle + " Details")}, m.renderContextInfoLines(width)...)
    rightLines := append([]string{title.Render(sectionTitle + " Actions")}, renderActionHelpLines(m)...)
    rightLines = indentLines(rightLines, 1)
    return renderSplitColumns(leftLines, rightLines, width, height)
}
```

`Context Details` 는 render 에서 바로 상태를 만들지 말고, 섹션별 snapshot 을 먼저 만든다.

snapshot 의 의도는 다음과 같다.

- `renderContextInfoLines` 는 단순 출력만 맡는다.
- `buildContextDetailsSnapshot` 류 helper 가 state 계산을 맡는다.
- 섹션별 detail field 는 각 섹션 전용 builder 로 채운다.
- 비어 있는 필드는 `-` 혹은 빈 문자열로 남겨도 된다.
- `GraphDetails` 는 `currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])` 를 사용한다.
- `TagDetails` 는 `m.sectionCursor[sectionTags]` 를 selected index 로 사용한다.

```go
type ContextDetailsSnapshot struct {
    Graph  GraphDetailsSnapshot
    Local  LocalDetailsSnapshot
    Remote RemoteDetailsSnapshot
    Tags   TagDetailsSnapshot
}

type GraphDetailsSnapshot struct {
    FocusHash   string
    ParentHash  string
    Branches    []string
    Stashes     []string
    Tags        []string
}

type LocalDetailsSnapshot struct {
    Target          string
    Upstream        string
    WorktreeState   string
    Ahead           int
    Behind          int
    DivergenceState string
}

type RemoteDetailsSnapshot struct {
    Target        string
    DefaultBranch string
    LastFetch     string
    BranchCount   int
}

type TagDetailsSnapshot struct {
    Name       string
    Hash       string
    Age        string
    Message    string
    Provenance string
}
```

스냅샷은 다음 상태 이름을 고정해서 쓴다.

```go
const (
    divergenceEqual     = "equal"
    divergenceAheadOnly = "aheadOnly"
    divergenceBehindOnly = "behindOnly"
    divergenceDiverged  = "diverged"

    tagProvenanceUnknown = "unknown"
    tagProvenanceLocal   = "local"
    tagProvenanceOrigin  = "origin"
)
```

핵심 규칙은 다음과 같다.

- `ahead` / `behind` 는 숫자만 계산한다.
- `divergenceState` 는 숫자를 묶는 판단 상태다.
- `renderContextInfoLines` 는 위 snapshot 만 읽고, repo 상태 계산을 직접 다시 하지 않는다.
- `TagDetailsSnapshot.Message` 는 commit subject 가 아니라 tag object message 다.
- `RemoteDetailsSnapshot.LastFetch` 는 값이 없으면 빈 문자열로 두고 render 에서 `-` 로 바꾼다.
- `RemoteDetailsSnapshot` 은 1차 표시 계약에서 `syncState` 를 드러내지 않는다.

의미와 예시:

- `ahead` 는 현재 branch 가 upstream 보다 앞선 커밋 수다. 예: `ahead: 2` 는 아직 upstream 에 안 올라간 커밋이 2개라는 뜻이다.
- `behind` 는 현재 branch 가 upstream 보다 뒤처진 커밋 수다. 예: `behind: 1` 은 upstream 에는 있지만 로컬에 없는 커밋이 1개 있다는 뜻이다.
- `divergenceState` 는 `ahead` 와 `behind` 를 묶은 상태다. 예: `ahead: 1`, `behind: 1` 이면 `diverged` 다.
- `provenance` 는 tag 가 어디서 왔는지 나타낸다. `local` 은 로컬에서만 보이는 태그, `origin` 은 origin 쪽에서도 확인된 태그, `unknown` 은 아직 출처를 확정하지 못한 상태다.

### builder 분해

구현은 다음 순서를 따른다.

```go
func buildContextDetailsSnapshot(rs git.Status, focus graphRow, tags []git.TagEntry, selectedTagIndex int) ContextDetailsSnapshot {
    return ContextDetailsSnapshot{
        Graph:  buildGraphDetailsSnapshot(rs, focus),
        Local:  buildLocalDetailsSnapshot(rs),
        Remote: buildRemoteDetailsSnapshot(rs),
        Tags:   buildTagDetailsSnapshot(tags, selectedTagIndex),
    }
}
```

```go
func buildGraphDetailsSnapshot(rs git.Status, focus graphRow) GraphDetailsSnapshot {
    return GraphDetailsSnapshot{
        FocusHash:  shortHash(focus.Hash),
        ParentHash: shortHash(focus.ParentHash),
        Branches:   renderBranchSummary(rs, focus),
        Stashes:    renderStashSummary(rs, focus),
        Tags:       renderTagSummary(rs, focus),
    }
}
```

```go
func buildLocalDetailsSnapshot(rs git.Status) LocalDetailsSnapshot {
    upstream := rs.Upstream
    if upstream == "" {
        upstream = "none"
    }
    divergence := divergenceEqual
    if track := rs.Tracking[rs.Branch]; track.Ahead > 0 || track.Behind > 0 {
        switch {
        case track.Ahead > 0 && track.Behind > 0:
            divergence = divergenceDiverged
        case track.Ahead > 0:
            divergence = divergenceAheadOnly
        default:
            divergence = divergenceBehindOnly
        }
    }
    return LocalDetailsSnapshot{
        Target:          rs.Branch,
        Upstream:        upstream,
        WorktreeState:   renderWorktreeState(rs),
        Ahead:           rs.Tracking[rs.Branch].Ahead,
        Behind:          rs.Tracking[rs.Branch].Behind,
        DivergenceState: divergence,
    }
}
```

```go
func buildRemoteDetailsSnapshot(rs git.Status) RemoteDetailsSnapshot {
    return RemoteDetailsSnapshot{
        Target:        rs.Remote,
        DefaultBranch: rs.DefaultBranch,
        LastFetch:     renderLastFetch(rs),
        BranchCount:   len(rs.RemoteBranches),
    }
}
```

```go
func buildTagDetailsSnapshot(tags []git.TagEntry, selectedTagIndex int) TagDetailsSnapshot {
    if len(tags) == 0 {
        return TagDetailsSnapshot{Provenance: tagProvenanceUnknown}
    }
    if selectedTagIndex < 0 || selectedTagIndex >= len(tags) {
        selectedTagIndex = 0
    }
    selected := tags[selectedTagIndex]
    return TagDetailsSnapshot{
        Name:       selected.Name,
        Hash:       shortHash(selected.CommitHash),
        Age:        selected.RelativeAge,
        Message:    selected.Message,
        Provenance: tagProvenanceForEntry(selected),
    }
}
```

이 시점의 목적은 완성형 UI 가 아니라, “무슨 값을 어디서 읽는지”를 고정하는 것이다.

## 6. Action helper 분리

현재 `renderActionHelpLines` 는 너무 많은 조건을 한 함수에 담고 있다.
이 문서대로라면 아래처럼 분리한다.

```go
func renderActionHelpLines(m model) []string {
    switch m.status.Mode {
    case state.ModeBrowse:
        switch m.activeSection {
        case sectionGraph:
            return renderGraphActionHelpLines(m)
        case sectionCurrent:
            return renderLocalActionHelpLines(m)
        case sectionRemote:
            return renderRemoteActionHelpLines(m)
        case sectionTags:
            return renderTagActionHelpLines(m)
        default:
            return []string{"• no section actions"}
        }
    case state.ModeTargetPick:
        if m.status.Action == state.ActionCheckout {
            return []string{"• enter: checkout", "• esc: back"}
        }
        return []string{"• enter: preview", "• esc: back"}
    case state.ModeResetModePick:
        return []string{"• s: soft  •  m: mixed  •  h: hard", "• esc: back"}
    case state.ModeReview:
        return []string{"• y: continue                    • n: cancel"}
    case state.ModeOutcomePreview:
        if m.status.CanExecute {
            return []string{"• enter: execute                    • esc: back"}
        }
        return []string{"• esc: back"}
    default:
        return []string{"• r: refresh"}
    }
}
```

### 6-1. Graph helper

Graph 는 기본 4개만 강하게 노출한다.

```go
func renderGraphActionHelpLines(m model) []string {
    lines := []string{
        "• m: merge",
        "• r: rebase",
        "• space: checkout",
        "• H: jump to HEAD",
    }

    // secondary actions are intentionally not shown in the main panel
    // but remain available through direct key handling and the hidden drawer.
    return lines
}
```

### 6-2. Local helper

```go
func renderLocalActionHelpLines(m model) []string {
    lines := make([]string, 0, 6)
    if m.activeSection != sectionCurrent {
        return []string{"• no section actions"}
    }
    if m.repoStatus.WorktreeDirty {
        lines = append(lines, "• s: stash changes")
        lines = append(lines, "• c: clean working tree")
    } else {
        lines = append(lines, disabled.Render("• s: stash changes")+" "+muted.Render("(dirty only)"))
        lines = append(lines, disabled.Render("• c: clean working tree")+" "+muted.Render("(dirty only)"))
    }
    if m.repoStatus.WorktreeDirty {
        lines = append(lines, disabled.Render("• space: checkout")+" "+muted.Render("(dirty)"))
    } else {
        lines = append(lines, "• space: checkout")
    }
    lines = append(lines, "• d: delete branch")
    if m.repoStatus.MergeInProgress {
        lines = append(lines, "• a: abort merge")
    } else {
        lines = append(lines, disabled.Render("• a: abort merge"))
    }
    lines = append(lines, "• n: new branch")
    return lines
}
```

### 6-3. Remote helper

```go
func renderRemoteActionHelpLines(m model) []string {
    if m.activeSection != sectionRemote {
        return []string{"• no section actions"}
    }
    return []string{
        "• space: checkout",
        "• f: fetch",
        "• p: pull",
        "• d: delete branch",
    }
}
```

### 6-4. Tags helper

```go
func renderTagActionHelpLines(m model) []string {
    if m.activeSection != sectionTags {
        return []string{"• no section actions"}
    }
    return []string{
        "• enter: jump to graph",
        "• d: delete tag",
    }
}
```

## 7. Minimal tests

이 변경은 help copy 와 hotkey routing 이 핵심이므로, 테스트는 가볍지만 명확해야 한다.

```go
func TestRenderGlobalContentShowsHiddenHotkeys(t *testing.T) {
    got := renderGlobalContent(80, 12)
    if !strings.Contains(got, "?: show hidden hotkeys") {
        t.Fatalf("expected global help to expose hidden hotkeys, got %q", got)
    }
}

func TestGraphHelpDoesNotShowQuestionMarkAsSearch(t *testing.T) {
    got := renderActionHelpLines(model{
        status:        state.New().WithBrowse(),
        activeSection: sectionGraph,
    })
    joined := strings.Join(got, " ")
    if strings.Contains(joined, "?: search") {
        t.Fatalf("expected graph help to stop advertising ? as search, got %q", joined)
    }
}

func TestQuestionMarkOpensHiddenHotkeysDrawer(t *testing.T) {
    m := model{
        status: state.New().WithBrowse(),
    }
    next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
    got := next.(model)
    if !got.hiddenHotkeysOpen {
        t.Fatal("expected ? to open the hidden hotkeys drawer")
    }
}

func TestSlashStillOpensGraphSearch(t *testing.T) {
    m := model{
        status:        state.New().WithBrowse(),
        activeSection: sectionGraph,
    }
    next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
    got := next.(model)
    if !got.graphSearchOpen {
        t.Fatal("expected / to keep opening graph search")
    }
}
```

## 8. 구현 체크리스트

- `?` 가 search 가 아니라 drawer 를 연다.
- drawer 는 `esc` 로 닫힌다.
- drawer 는 현재 섹션 기준 hotkey 만 보여준다.
- `Global` 에 `?` 가 표시된다.
- `Graph` help 에 `?` search 문구가 남지 않는다.
- `Graph` 기본 action 은 4개만 보인다.
- `scroll`, `top`, `bottom` 은 Graph main help 에서 사라진다.
- `Local` / `Remote` / `Tags` 는 섹션별 helper 로 분리된다.
