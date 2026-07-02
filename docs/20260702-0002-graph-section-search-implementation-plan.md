# Graph Section Search Implementation Plan

## 목적

`Graph` 섹션에서 commit title, commit hash, branch name 을 빠르게 찾는 검색 기능을 추가한다.

이 문서의 목표는 구현 기준을 단순하게 고정하는 것이다.

1. 검색 대상은 `Graph` row 중심이다.
2. 검색어는 title, hash, branch decoration 에 모두 매칭된다.
3. 결과 선택 시 해당 row 로 즉시 이동한다.
4. 검색은 화면 상태를 바꾸지 않고, cursor / scroll 만 바꾼다.

## 사용 시나리오

- 긴 graph 에서 특정 commit title 을 빠르게 찾는다.
- hash 일부를 입력해서 정확한 commit 을 찾는다.
- branch name 을 입력해서 해당 branch 가 붙은 commit 으로 이동한다.
- `Graph` 바깥의 section navigation 은 유지한다.

## 범위

### 포함

- `Graph` 검색 input
- title / hash / branch 매칭
- result cycle
- graph cursor / scroll jump
- 관련 tests

### 제외

- Git ref rename
- search index persistence
- full-text search engine 도입
- server-side search

## 핵심 계약

### 1. 검색은 `Graph` 섹션에서만 열린다

`Graph` 섹션에서만 검색 overlay 를 열고, 다른 section 에서는 기존 navigation 을 유지한다.

권장 shortcut 은 `/` 이다.

```go
case "/":
	if m.activeSection != sectionGraph || m.status.Mode != state.ModeBrowse {
		return m, nil
	}
	m.graphSearchOpen = true
	m.graphSearchDraft = ""
	m.graphSearchIndex = buildGraphSearchIndex(m.repoStatus)
	m.graphSearchCursor = 0
	m.status = loadingToast("Search graph...")
	return m, nil
```

### 2. 검색 대상은 graph row 와 branch decoration 이다

`Graph` row 에서 다음을 검색한다.

- `row.Commit.Subject`
- `row.Commit.Hash`
- `row.Commit.Decorations`

branch name 은 decoration 에 포함된 값으로 본다.

예를 들어 다음과 같은 decoration 은 모두 검색 대상이다.

- `HEAD -> feature/login`
- `feature/login`
- `origin/feature/login`
- `tag: v1.2.0`

### 3. 매칭 우선순위는 hash > branch > title 이다

검색어가 commit hash prefix 로 보이면 hash 를 먼저 맞춘다.
그렇지 않으면 branch decoration, 그 다음 title subject 로 매칭한다.

짧은 입력에서도 commit hash 가 가장 강한 신호가 되도록 한다.

## 구현 파일

- `internal/app/model.go`
- `internal/app/key_handling.go`
- `internal/app/key_handling_graph_search.go`  // 새 파일 권장
- `internal/app/view_sections.go`
- `internal/app/view_detail.go` 또는 `internal/app/view_shell.go`
- `internal/app/navigation_graph.go`
- `internal/app/model_test.go`
- `internal/app/key_handling_test.go`

## 모델 추가

검색은 branch input 과 별도 상태로 두는 편이 낫다.

```go
type model struct {
	// existing fields...
	graphSearchOpen   bool
	graphSearchDraft  string
	graphSearchIndex  []graphSearchEntry
	graphSearchCursor int
}

type graphSearchEntry struct {
	Hash     string
	Title    string
	Branches []string
	Row      int
	Score    int
}
```

의도는 단순하다.

- `graphSearchOpen`: overlay 표시 여부
- `graphSearchDraft`: 입력 문자열
- `graphSearchIndex`: 현재 repoStatus 기준 검색 후보
- `graphSearchCursor`: 결과 목록에서 선택된 항목

`Row` 는 `graph.Rows()` 기준 row index 이다.

## 인덱스 생성

인덱스는 별도 저장소가 아니라 현재 repo status 로부터 매번 만든다.

```go
func buildGraphSearchIndex(rs git.Status) []graphSearchEntry {
	rows := graph.Rows(rs)
	index := make([]graphSearchEntry, 0, len(rows))

	for i, row := range rows {
		if row.Commit.Hash == "" || row.Commit.Hash == "VIRTUAL_CONFLICT_HASH" {
			continue
		}

		entry := graphSearchEntry{
			Hash:  row.Commit.Hash,
			Title: row.Commit.Subject,
			Row:   i,
		}

		for _, dec := range row.Commit.Decorations {
			dec = strings.TrimSpace(dec)
			switch {
			case dec == "":
				continue
			case strings.HasPrefix(dec, "HEAD -> "):
				entry.Branches = append(entry.Branches, strings.TrimPrefix(dec, "HEAD -> "))
			case strings.HasPrefix(dec, "tag: "):
				continue
			default:
				entry.Branches = append(entry.Branches, strings.TrimPrefix(dec, "origin/"))
				entry.Branches = append(entry.Branches, dec)
			}
		}

		entry.Score = 0
		index = append(index, entry)
	}
	return index
}
```

주의점:

- branch decoration 은 중복으로 들어갈 수 있다.
- `origin/feature/x` 와 `feature/x` 를 둘 다 후보로 두는 편이 찾기 쉽다.
- 결과 표시는 중복 제거 후 보여도 되고, 매칭 단계에서만 중복을 허용해도 된다.

## 매칭 규칙

검색은 prefix + contains 혼합으로 두되, score 기반 정렬로 결과를 고정한다.

```go
func scoreGraphSearchEntry(entry graphSearchEntry, q string) (int, bool) {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return 0, false
	}

	hash := strings.ToLower(entry.Hash)
	title := strings.ToLower(entry.Title)
	branches := make([]string, 0, len(entry.Branches))
	for _, b := range entry.Branches {
		branches = append(branches, strings.ToLower(b))
	}

	switch {
	case strings.HasPrefix(hash, q):
		return 3000 + len(hash) - len(q), true
	case anyContains(branches, q):
		return 2000, true
	case strings.Contains(title, q):
		return 1000, true
	default:
		return 0, false
	}
}
```

추천 우선순위:

1. exact hash prefix
2. branch name prefix / contains
3. title contains

`anyContains()` 는 아주 얇은 helper 로 충분하다.

```go
func anyContains(items []string, q string) bool {
	for _, item := range items {
		if strings.Contains(item, q) {
			return true
		}
	}
	return false
}
```

## 검색 실행

검색은 `enter` 시점에 현재 선택 결과로 점프한다.

```go
func applyGraphSearchSelection(m model) model {
	rows := graph.Rows(m.repoStatus)
	if len(rows) == 0 {
		return m
	}

	matched := graphSearchMatches(m.graphSearchIndex, m.graphSearchDraft)
	if len(matched) == 0 {
		m.status = state.New().WithBlocked(state.BlockTargetEmpty, "No graph match.", "Try a different query.")
		return m
	}

	entry := matched[m.graphSearchCursor%len(matched)]
	m.graphSearchOpen = false
	m.graphSearchDraft = ""
	m.graphSearchIndex = nil
	m.graphSearchCursor = 0
	m.activeSection = sectionGraph
	m.sectionCursor[sectionGraph] = entry.Row

	page := graphPageSizeForRows(&m, rows, entry.Row, graphContentHeightForModel(&m))
	m.graphScroll = clampScroll(entry.Row, len(rows), page)
	m.graphLaneCursor = graph.PointerLane(rows[entry.Row])
	return m
}
```

이때 `graphSearchMatches()` 는 score 내림차순으로 정렬한다.

```go
func graphSearchMatches(index []graphSearchEntry, q string) []graphSearchEntry {
	matches := make([]graphSearchEntry, 0, len(index))
	for _, entry := range index {
		if score, ok := scoreGraphSearchEntry(entry, q); ok {
			entry.Score = score
			matches = append(matches, entry)
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		if matches[i].Row != matches[j].Row {
			return matches[i].Row < matches[j].Row
		}
		return matches[i].Hash < matches[j].Hash
	})
	return matches
}
```

## key handling

검색 overlay 는 기존 branch input 과 같은 방식으로 별도 handler 로 분리한다.

```go
func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.branchOpen {
		return m.handleBranchOpenKey(msg)
	}
	if m.graphSearchOpen {
		return m.handleGraphSearchKey(msg)
	}
	switch m.status.Mode {
	case state.ModeTargetPick:
		return m.handleTargetPickKey(msg)
	// existing cases...
	default:
		return m, nil
	}
}
```

핵심 키 계약은 다음과 같다.

- `esc`: 닫기
- `enter`: 현재 result 로 점프
- `backspace`: query 삭제
- printable rune: query 추가
- `up/down` 또는 `j/k`: result cursor 이동

예시 구현:

```go
func (m model) handleGraphSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.graphSearchOpen = false
		m.graphSearchDraft = ""
		m.graphSearchIndex = nil
		m.graphSearchCursor = 0
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	case "enter":
		return applyGraphSearchSelection(m), nil
	case "up", "k":
		if m.graphSearchCursor > 0 {
			m.graphSearchCursor--
		}
		return m, nil
	case "down", "j":
		matches := graphSearchMatches(m.graphSearchIndex, m.graphSearchDraft)
		if len(matches) > 0 {
			m.graphSearchCursor = min(m.graphSearchCursor+1, len(matches)-1)
		}
		return m, nil
	case "backspace":
		if len(m.graphSearchDraft) > 0 {
			runes := []rune(m.graphSearchDraft)
			m.graphSearchDraft = string(runes[:len(runes)-1])
			m.graphSearchCursor = 0
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.graphSearchDraft += string(msg.Runes)
			m.graphSearchCursor = 0
			return m, nil
		}
	}
	return m, nil
}
```

## 화면 렌더링

검색은 `Graph` content 위에 얹는 얇은 overlay 로 충분하다.

권장 표시는 다음 수준이다.

```text
Search graph: feature/login
1. a1b2c3d feature/login
2. d4e5f6a fix login redirect
3. 9f8e7d6 origin/feature/login
```

렌더링은 기존 graph panel 위 또는 context/detail panel 내부에 두어도 된다.
중요한 것은 검색 중에도 graph 본문은 그대로 유지하고, 검색창만 덧씌우는 것이다.

```go
func (m model) renderGraphSearchOverlay(width int) []string {
	if !m.graphSearchOpen {
		return nil
	}

	matches := graphSearchMatches(m.graphSearchIndex, m.graphSearchDraft)
	lines := []string{
		title.Render("Search"),
		"query: " + m.graphSearchDraft,
	}
	for i, match := range matches {
		prefix := "  "
		if i == m.graphSearchCursor {
			prefix = "> "
		}
		lines = append(lines, prefix+shorten(match.Hash, 7)+" "+shorten(match.Title, max(width-12, 0)))
	}
	return lines
}
```

## 상세 동작

### 1. query 가 비어 있으면 최근 graph 순서를 그대로 보여준다

빈 query 는 전체 목록을 보여 주거나, 최소한 현재 `HEAD` 와 가까운 순서를 유지한다.

### 2. hash prefix 가 7자 이상이면 hash 검색을 우선한다

hash 는 보통 `7`자 이상에서 식별 가능성이 높다.
짧은 값도 허용하되, score 는 hash 우선으로 잡는다.

### 3. branch name 은 decoration 기반으로만 찾는다

검색 결과에 branch list 를 따로 만들지 말고, commit 이 실제로 붙어 있는 decoration 을 기준으로 한다.

### 4. 검색 결과 점프는 scroll/lane 동기화까지 포함한다

cursor 만 바꾸면 사용자가 위치를 잃는다.
반드시 `graphScroll` 과 `graphLaneCursor` 를 같이 갱신한다.

## 테스트

다음 테스트를 추가하면 충분하다.

```go
func TestGraphSearchMatchesHashTitleAndBranch(t *testing.T)
func TestGraphSearchSelectionMovesCursorScrollAndLane(t *testing.T)
func TestGraphSearchEscRestoresBrowseState(t *testing.T)
func TestGraphSearchIsGraphSectionOnly(t *testing.T)
```

각 테스트에서 확인할 내용은 명확하다.

- hash prefix 가 title 보다 우선하는지
- branch decoration 이 title 보다 우선하는지
- enter 시 `sectionCursor[sectionGraph]`, `graphScroll`, `graphLaneCursor` 가 동기화되는지
- `esc` 시 검색 상태가 완전히 초기화되는지
- Graph 바깥에서는 shortcut 이 무시되는지

## 구현 순서

1. `model` 에 search 상태를 추가한다.
2. `graph.Rows()` 기반 검색 인덱스를 만든다.
3. `handleKeyMsg()` 에 search overlay 분기를 추가한다.
4. 검색 입력 handler 를 분리한다.
5. 검색 결과 점프 helper 를 추가한다.
6. overlay 렌더를 추가한다.
7. 검색 관련 테스트를 추가한다.

## 완료 기준

- Graph 에서 `/` 로 검색을 열 수 있다.
- title, hash, branch name 으로 commit 을 찾을 수 있다.
- 결과를 선택하면 해당 row 로 즉시 이동한다.
- 검색 중에도 기존 graph navigation 은 망가지지 않는다.
- 구현 기준이 테스트로 고정된다.
