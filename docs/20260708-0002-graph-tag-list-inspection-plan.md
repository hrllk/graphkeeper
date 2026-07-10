# Graph Tag List Inspection Plan

## 목적

`Graph` 와 `Tags` 섹션에서 tag 를 읽고, 각 row 에 tag name / commit hash / subject / age 를 모두 보여준다.

이 문서는 읽기 전용이다. tag 생성, 이름 변경, 삭제, popup 입력은 `0003`에서 다룬다.

## 사용자 목표

1. 사용자는 현재 repo 의 tag 를 한 화면에서 바로 읽을 수 있어야 한다.
2. 각 row 에는 tag name, target commit hash, commit subject, relative age 가 함께 보여야 한다.
3. `Tags` 섹션에서 선택한 row 는 해당 commit 의 `Graph` row 로 연결되어야 한다.
4. `Graph` row 는 긴 tag 목록을 다 펼치지 말고 compact marker 만 보여야 한다.

## 범위

### 포함

- tag ref 읽기
- `Tags` 섹션 inspector
- tag row selection -> graph focus sync
- `Graph` row compact tag marker
- 관련 tests

### 제외

- tag 생성
- tag rename
- tag delete
- tag popup 입력
- annotated tag 메타데이터 편집

## 현재 상태

현재 코드는 tag 를 읽고는 있지만, read model 이 너무 얇다.

- `internal/git/repo.go` 는 `refs/tags` 이름만 읽는다.
- `internal/app/view_sections.go` 의 `Tags` 섹션은 raw name list 만 보여준다.
- `internal/app/target_items.go` 는 tag 를 `TargetItem{Name, Ref}` 로만 노출한다.
- `internal/app/graph_render_format.go` 와 `internal/app/graph_search.go` 는 `tag: ` decoration 을 대부분 건너뛴다.

즉, tag 는 존재하지만 읽기 contract 가 없다.

## 핵심 결정

### 1. `Tags` 섹션은 raw ref list 가 아니라 row-oriented inspector 다

목록은 tag 이름만 나열하지 않는다. 각 row 는 한 tag 를 대표한다.

각 row 는 최소한 다음을 포함해야 한다.

- tag name
- target commit hash
- commit subject
- relative age

### 2. `Graph` row 는 compact marker 만 가진다

row 안에 tag 이름을 전부 펼치지 않는다.

- tag 가 있으면 compact marker 를 보여준다
- tag 가 여러 개면 count 를 함께 보여줄 수 있다

자세한 이름 목록은 `Tags` 섹션만 담당한다.

### 3. stash 와 tag 가 같은 point 에 겹치면 색을 섞지 않는다

같은 commit 에 stash 와 tag 가 동시에 있으면, 두 상태를 한 색으로 뭉개지 않는다.

- stash 는 기존 orange 계열 accent 를 유지한다.
- tag 는 graph 전용 secondary accent 를 유지한다.
- 둘이 겹치면 `#A14743` 단색 combined badge 로 보여준다.
- commit hash 자체에는 상태색을 입히지 않는다.

이유는 단순하다. hash 는 식별자고, 색은 상태여야 한다. 둘을 섞으면 읽는 속도가 떨어진다.

### 4. stash 와 tag overlap 은 `#A14743` 단색 badge 로 본다

별도 SVG 시안은 쓰지 않는다.

- stash 와 tag 가 같은 commit 에 겹치면 `#A14743` 단색 badge 를 사용한다.
- split, dual tone, blended color 는 쓰지 않는다.
- commit hash 는 계속 중립색으로 유지한다.
- badge 내부 텍스트는 `S/T` 또는 `ST` 같은 짧은 표기로 충분하다.
- 공간이 부족하면 badge 폭을 키우는 대신 tag count 나 detail panel 쪽으로 정보를 넘긴다.

핵심은 하나다. 겹침 상태를 구조 분리로 보여주는 대신, 하나의 충돌색으로 읽히게 한다.

### 5. tag target 은 peeled commit hash 를 기준으로 잡는다

이 문서의 read model 은 annotated tag 도 commit 으로 풀어서 읽는다.

그래서 group key 는 tag ref 가 아니라 실제 commit hash 여야 한다.

### 6. `Tags` 섹션에서 `enter` 하면 `Graph` focus 로 연결한다

읽기 화면은 끝이 아니라 탐색 시작점이다.

사용자가 tag row 를 선택하고 `enter` 하면 같은 commit 이 `Graph` 에서 focus 되어야 한다.

```go
func (m model) handleBrowseSectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "enter":
        if m.activeSection != sectionTags {
            return m, nil
        }
        item := activeSectionTargetItem(m)
        if item.Ref == "" {
            m.status = state.New().WithBlocked(state.BlockTargetEmpty, "No tag selected.", "Move to a tag row.")
            return m, nil
        }
        rows := graph.Rows(m.repoStatus)
        row := graph.FindRowByHash(rows, item.Ref)
        if row < 0 {
            m.status = state.New().WithBlocked(state.BlockUnknown, "Tag target is missing.", "Refresh the repo and try again.")
            return m, nil
        }
        m.activeSection = sectionGraph
        m.sectionCursor[sectionGraph] = row
        m.graphLaneCursor = graph.PointerLane(rows[row])
        m.graphScroll = clampScroll(row, len(rows), graphPageSizeForRows(&m, rows, row, graphContentHeightForModel(&m)))
        return m, nil
    }
    return m, nil
}
```

## BEFORE

현재 구조는 이름만 읽는다.

```go
type Status struct {
    // ...
    Tags []string
}
```

```go
func buildTagSectionTargets(rs git.Status) []state.TargetItem {
    items := make([]state.TargetItem, 0, len(rs.Tags))
    for _, name := range rs.Tags {
        items = append(items, state.TargetItem{
            Kind: state.TargetKindTag,
            Name: name,
            Ref:  name,
        })
    }
    return items
}
```

```go
func formatTargetItem(t state.TargetItem) string {
    switch t.Kind {
    case state.TargetKindTag:
        return "tag    " + t.Name
    default:
        return t.Name
    }
}
```

## AFTER

read model 을 row-oriented list 로 바꾼다.

```go
package git

type TagEntry struct {
    Name        string
    CommitHash   string
    Subject      string
    RelativeAge  string
    CommitUnix   int64
}

type Status struct {
    // ...
    Tags []TagEntry
}
```

```go
func (r *Repo) Tags(ctx context.Context) ([]TagEntry, error) {
    names, err := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/tags")
    if err != nil {
        return nil, err
    }

    entries := make([]TagEntry, 0, len(names))
    for _, name := range names {
        target, err := r.git(ctx, "rev-parse", "--verify", name+"^{commit}")
        if err != nil {
            return nil, err
        }
        target = strings.TrimSpace(target)

        subject, err := r.git(ctx, "show", "-s", "--format=%s", target)
        if err != nil {
            return nil, err
        }
        relativeAge, err := r.git(ctx, "show", "-s", "--format=%cr", target)
        if err != nil {
            return nil, err
        }
        commitUnixText, err := r.git(ctx, "show", "-s", "--format=%ct", target)
        if err != nil {
            return nil, err
        }
        commitUnix, err := strconv.ParseInt(strings.TrimSpace(commitUnixText), 10, 64)
        if err != nil {
            return nil, err
        }
        entries = append(entries, TagEntry{
            Name:        name,
            CommitHash:  target,
            Subject:     strings.TrimSpace(subject),
            RelativeAge: strings.TrimSpace(relativeAge),
            CommitUnix:  commitUnix,
        })
    }

    sort.SliceStable(entries, func(i, j int) bool {
        if entries[i].CommitUnix != entries[j].CommitUnix {
            return entries[i].CommitUnix > entries[j].CommitUnix
        }
        return entries[i].Name < entries[j].Name
    })
    return entries, nil
}
```

```go
func buildTagSectionRows(tags []git.TagEntry) []tagSectionRow {
    rows := make([]tagSectionRow, 0, len(tags))
    for _, tag := range tags {
        rows = append(rows, tagSectionRow{
            Name:        tag.Name,
            Hash:        tag.CommitHash,
            Subject:     tag.Subject,
            RelativeAge: tag.RelativeAge,
        })
    }
    return rows
}
```

```go
func renderTagSectionRow(row tagSectionRow, selected bool, width int) string {
    label := fmt.Sprintf("%-24s  %s  %-28s  %s",
        row.Name,
        shorten(row.Hash, 7),
        compactTitleText(row.Subject),
        compactWhenText(row.RelativeAge),
    )
    label = fitVisibleWidth(label, width-2)
    if selected {
        return highlight.Render(">" + label)
    }
    return " " + label
}
```

```go
type graphMarkerKind string

const (
    graphMarkerNone     graphMarkerKind = "none"
    graphMarkerTag      graphMarkerKind = "tag"
    graphMarkerStash    graphMarkerKind = "stash"
    graphMarkerCombined graphMarkerKind = "combined"
)

type graphMarkerSpec struct {
    Kind        graphMarkerKind
    TagCount    int
    HasStash    bool
    LeftLabel   string
    RightLabel  string
}

func graphMarkerForCommit(hash string, tags []git.TagEntry, hasStash bool) graphMarkerSpec {
    tagCount := 0
    for _, tag := range tags {
        if tag.CommitHash == hash {
            tagCount++
        }
    }

    switch {
    case tagCount == 0 && !hasStash:
        return graphMarkerSpec{Kind: graphMarkerNone}
    case tagCount > 0 && !hasStash:
        return graphMarkerSpec{Kind: graphMarkerTag, TagCount: tagCount, LeftLabel: "T"}
    case tagCount == 0 && hasStash:
        return graphMarkerSpec{Kind: graphMarkerStash, HasStash: true, LeftLabel: "S"}
    default:
        return graphMarkerSpec{
            Kind:       graphMarkerCombined,
            TagCount:   tagCount,
            HasStash:   true,
            LeftLabel:  "S",
            RightLabel: "T",
        }
    }
}

func renderGraphMarker(spec graphMarkerSpec) string {
    switch spec.Kind {
    case graphMarkerNone:
        return ""
    case graphMarkerTag:
        return tagAccent.Render("[" + spec.LeftLabel + "]")
    case graphMarkerStash:
        return stashAccent.Render("[" + spec.LeftLabel + "]")
    case graphMarkerCombined:
        return stashAccent.Render("[" + spec.LeftLabel + "]") +
            tagAccent.Render("[" + spec.RightLabel + "]")
    default:
        return ""
    }
}
```

```go
func (m model) renderContextInfoLines(width int) []string {
    switch m.activeSection {
    case sectionTags:
        rows := tagSectionRows(m.repoStatus.Tags)
        if len(rows) == 0 {
            return []string{muted.Render("  (empty)")}
        }
        cursor := clampCursor(m.sectionCursor[sectionTags], len(rows))
        row := rows[cursor]
        return []string{
            fmt.Sprintf("target: %s", shorten(row.Hash, 7)),
            fmt.Sprintf("tag: %s", row.Name),
            fmt.Sprintf("subject: %s", shorten(row.Subject, max(width-10, 0))),
            fmt.Sprintf("age: %s", row.RelativeAge),
        }
    default:
        return nil
    }
}
```

## 테스트

```go
func TestTagsResolveToCommitHash(t *testing.T)
func TestTagsSortNewestFirst(t *testing.T)
func TestTagsSectionShowsTargetSubjectAndAge(t *testing.T)
func TestGraphShowsCompactTagMarkerForTaggedCommit(t *testing.T)
func TestGraphShowsCombinedMarkerWhenStashAndTagShareCommit(t *testing.T)
func TestTagSelectionJumpsToGraphCommit(t *testing.T)
```

검증 포인트는 다음이다.

- tag 가 commit hash 로 풀리는지
- tag 없는 commit 에 marker 가 안 붙는지
- stash 와 tag 가 같은 point 에 있을 때 합성 marker 가 나오는지
- `Tags` 에서 선택한 row 가 `Graph` focus 로 연결되는지

## Verification

```sh
go test ./internal/git ./internal/app
```

## Notes

- read model 과 graph marker 는 같은 source of truth 를 써야 한다.
- grouping key 는 tag ref 가 아니라 target commit hash 여야 한다.
- 생성/삭제/이름변경은 `0003`로 넘긴다.
