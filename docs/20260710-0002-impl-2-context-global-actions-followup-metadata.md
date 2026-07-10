# 20260710-0002 Impl 2

`docs/20260710-0001-context-global-actions-rebalance-plan.md` 의 후속 메타데이터를 구현하기 위한 문서다.

이 문서는 1차 rebalance 문서에서 의도적으로 분리한 두 영역만 다룬다.

- `Remote` 의 `last fetch` / `sync status`
- `Tags` 의 `tagger` / `tagged time` / `message`

## 왜 별도 문서인가

1차 문서는 화면 밀도와 hotkey 재배치에 집중한다.
이 문서는 그 위에 얹히는 정보 보강만 담당한다.

- `Remote` 는 현재 repo state 에서 이미 계산 가능한 부분과, fetch 시점의 로컬 메타데이터를 합쳐서 보여준다.
- `Tags` 는 현재 `TagEntry` 가 commit 기준 정보만 들고 있으므로, tag object 메타데이터를 추가로 읽어와야 한다.

## 구현 원칙

- 기존 동작은 유지한다.
- 새 메타데이터는 읽기 전용으로 추가한다.
- `TagProvenance` 계약은 건드리지 않는다.
- `Graph` / `Local` / `Remote` / `Tags` layout 계약은 1차 문서를 따른다.

## 파일별 수정 순서

1. [`internal/git/repo.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/git/repo.go ) 에 `Remote` / `TagEntry` 필드를 추가한다.
2. [`internal/git/repo_exec.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/git/repo_exec.go ) 에 fetch head 시간과 tag object 메타데이터 수집 로직을 추가한다.
3. [`internal/app/view_detail.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_detail.go ) 에 `Remote` / `Tags` 상세 라인을 추가한다.
4. [`internal/app/view_sections.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_sections.go ) 에 tag row 표시용 helper 를 정리한다.
5. [`internal/app/model_test.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/model_test.go ) 와 [`internal/git/tag_test.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/git/tag_test.go ) 에 새 필드를 검증하는 테스트를 추가한다.

## 1. `git.Status` 와 `git.TagEntry` 확장

### 1-1. `git.Status`

[`internal/git/repo.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/git/repo.go ) 에 다음 필드를 추가한다.

```go
type Status struct {
    Root                  string
    Branch                string
    Head                  string
    DefaultBranch         string
    Upstream              string
    Remote                string
    Detached              bool
    HasCommits            bool
    Graph                 []string
    GraphCommits          []GraphCommit
    Branches              []string
    LocalBranches         []string
    BranchUpstreams       map[string]string
    Tracking              map[string]BranchTracking
    RemoteBranches        []string
    Tags                  []string
    TagEntries            []TagEntry
    TagEntriesLoaded      bool
    TagProvenanceLoaded   bool
    TagSyncSummary        string
    LastFetchAt           time.Time
    RemoteSyncSummary     string
    Remotes               []string
    EmptyRepo             bool
    NoUpstream            bool
    NoRemote              bool
    WorktreeDirty         bool
    MergeInProgress       bool
    RebaseInProgress      bool
    ConflictTarget        string
    ConflictTargetSubject string
    ErrorMessage          string
    LoadingReason         string
}
```

### 1-2. `git.TagEntry`

같은 파일에 tag detail 용 필드를 추가한다.

```go
type TagEntry struct {
    Name        string
    CommitHash  string
    Subject     string
    RelativeAge string
    CommitUnix  int64
    Tagger      string
    TaggedAt    time.Time
    Message     string
    Annotated   bool
    OriginKnown bool
    OnOrigin    bool
}
```

### 의도

- `Subject` 는 기존처럼 tag 가 가리키는 commit subject 로 유지한다.
- `Message` 는 tag object 의 annotation message 로 추가한다.
- `Tagger` / `TaggedAt` 는 annotated tag 의 작성자 정보다.
- lightweight tag 는 `Annotated=false` 로 두고 `Tagger` / `Message` 를 빈 값으로 유지할 수 있다.

## 2. Remote last fetch

`last fetch` 는 앱이 따로 저장한 파일보다 `FETCH_HEAD` 의 mtime 을 읽는 편이 더 단순하고 안정적이다.

### 2-1. `Repo.Status` 에서 FETCH_HEAD 시간 읽기

[`internal/git/repo_exec.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/git/repo_exec.go ) 에 다음 helper 를 추가한다.

```go
func (r *Repo) fetchHeadModTime(ctx context.Context) (time.Time, bool) {
    path, err := r.git(ctx, "rev-parse", "--git-path", "FETCH_HEAD")
    if err != nil {
        return time.Time{}, false
    }
    path = strings.TrimSpace(path)
    if path == "" {
        return time.Time{}, false
    }
    if !filepath.IsAbs(path) {
        path = filepath.Join(r.root, path)
    }
    info, err := os.Stat(path)
    if err != nil {
        return time.Time{}, false
    }
    return info.ModTime(), true
}
```

그리고 `Repo.Status` 의 마지막에 연결한다.

```go
func (r *Repo) Status(ctx context.Context, limit int) (Status, error) {
    branch, err := r.currentBranch(ctx)
    if err != nil {
        return Status{}, err
    }
    head, _ := r.git(ctx, "rev-parse", "HEAD")
    upstream, _ := r.git(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
    remote, _ := r.git(ctx, "remote")
    branches, branchUpstreams, tracking := r.branchMetadata(ctx)
    localBranches := branches
    remoteBranches, _ := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/remotes")
    defaultBranch := r.defaultRemoteBranch(ctx)
    graphCommits, graphErr := r.graphCommits(ctx, localBranches, branchUpstreams, limit)
    if graphErr != nil && !isNoCommits(graphErr) {
        return Status{ErrorMessage: graphErr.Error()}, graphErr
    }
    worktreeDirty, _ := r.worktreeDirty(ctx)

    status := Status{
        Root:           r.root,
        Branch:         branch,
        Head:           head,
        DefaultBranch:  defaultBranch,
        Upstream:       upstream,
        Remote:         remote,
        GraphCommits:   graphCommits,
        Branches:       branches,
        LocalBranches:  localBranches,
        BranchUpstreams: branchUpstreams,
        Tracking:       tracking,
        RemoteBranches: remoteBranches,
        WorktreeDirty:  worktreeDirty,
    }
    if t, ok := r.fetchHeadModTime(ctx); ok {
        status.LastFetchAt = t
    }
    status.RemoteSyncSummary = remoteSyncSummary(status)
    return status, nil
}
```

### 2-2. sync summary helper

`internal/app` 쪽에 표시용 helper 를 둔다.

```go
func remoteSyncSummary(rs git.Status) string {
    switch {
    case rs.Root == "":
        return ""
    case rs.NoRemote:
        return "no remote"
    case rs.NoUpstream:
        return "no upstream"
    case rs.Detached:
        return "detached"
    }

    track := rs.Tracking[rs.Branch]
    switch {
    case track.Ahead == 0 && track.Behind == 0:
        return "synced"
    case track.Ahead > 0 && track.Behind == 0:
        return fmt.Sprintf("ahead %d", track.Ahead)
    case track.Ahead == 0 && track.Behind > 0:
        return fmt.Sprintf("behind %d", track.Behind)
    default:
        return fmt.Sprintf("diverged (%d ahead, %d behind)", track.Ahead, track.Behind)
    }
}
```

### 2-3. Remote detail rendering

[`internal/app/view_detail.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_detail.go ) 의 `sectionCurrent, sectionRemote, sectionTags` 분기에서 Remote 전용 detail line 을 추가한다.

```go
if m.activeSection == sectionRemote {
    if m.repoStatus.Upstream != "" {
        lines = append(lines, fmt.Sprintf("upstream: %s", shorten(m.repoStatus.Upstream, width-10)))
    } else {
        lines = append(lines, "upstream: -")
    }
    if m.repoStatus.DefaultBranch != "" {
        lines = append(lines, fmt.Sprintf("default: %s", shorten(m.repoStatus.DefaultBranch, width-9)))
    }
    if !m.repoStatus.LastFetchAt.IsZero() {
        lines = append(lines, fmt.Sprintf("last fetch: %s", compactWhenTime(m.repoStatus.LastFetchAt)))
    } else {
        lines = append(lines, "last fetch: -")
    }
    if m.repoStatus.RemoteSyncSummary != "" {
        lines = append(lines, fmt.Sprintf("sync: %s", m.repoStatus.RemoteSyncSummary))
    }
    lines = append(lines, fmt.Sprintf("branches: %d", len(m.repoStatus.RemoteBranches)))
}
```

### 2-4. 표시 규칙

- `last fetch` 는 없으면 `-` 로 보인다.
- `sync` 는 `synced`, `ahead`, `behind`, `diverged`, `no upstream`, `no remote` 중 하나다.
- `FETCH_HEAD` 가 없으면 최초 fetch 전 상태로 본다.

## 3. Tags metadata

현재 `TagEntries` 는 commit 중심 정보만 가지고 있으므로, tag object 를 추가로 읽어야 한다.

### 3-1. `LocalTagEntries` 재구성

[`internal/git/repo_exec.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/git/repo_exec.go ) 의 `LocalTagEntries` 를 아래 방식으로 바꾼다.

```go
func (r *Repo) LocalTagEntries(ctx context.Context) ([]TagEntry, error) {
    names, err := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/tags")
    if err != nil {
        return nil, err
    }
    if len(names) == 0 {
        return nil, nil
    }

    entries := make([]TagEntry, 0, len(names))
    for _, name := range names {
        name = strings.TrimSpace(name)
        if name == "" {
            continue
        }

        commitHash, err := r.git(ctx, "rev-parse", "--verify", name+"^{commit}")
        if err != nil {
            continue
        }
        commitHash = strings.TrimSpace(commitHash)

        commitSubject, err := r.git(ctx, "show", "-s", "--format=%s", commitHash)
        if err != nil {
            continue
        }
        commitAge, err := r.git(ctx, "show", "-s", "--format=%cr", commitHash)
        if err != nil {
            continue
        }
        commitUnixText, err := r.git(ctx, "show", "-s", "--format=%ct", commitHash)
        if err != nil {
            continue
        }
        commitUnix, err := strconv.ParseInt(strings.TrimSpace(commitUnixText), 10, 64)
        if err != nil {
            continue
        }

        taggerName, _ := r.git(ctx, "for-each-ref", "--format=%(taggername)", "refs/tags/"+name)
        taggerDate, _ := r.git(ctx, "for-each-ref", "--format=%(taggerdate:unix)", "refs/tags/"+name)
        message, _ := r.git(ctx, "for-each-ref", "--format=%(contents:subject)", "refs/tags/"+name)

        entry := TagEntry{
            Name:        name,
            CommitHash:  commitHash,
            Subject:     strings.TrimSpace(commitSubject),
            RelativeAge: strings.TrimSpace(commitAge),
            CommitUnix:  commitUnix,
            Tagger:      strings.TrimSpace(taggerName),
            Message:     strings.TrimSpace(message),
        }

        if ts := strings.TrimSpace(taggerDate); ts != "" {
            if unix, err := strconv.ParseInt(ts, 10, 64); err == nil {
                entry.TaggedAt = time.Unix(unix, 0)
                entry.Annotated = true
            }
        }

        if entry.Annotated && entry.Tagger == "" {
            entry.Tagger = "(unknown)"
        }
        if !entry.Annotated {
            entry.Tagger = ""
            entry.Message = ""
        }

        entries = append(entries, entry)
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

### 3-2. Lightweight tag fallback

`taggername` / `taggerdate` 가 비어 있으면 lightweight tag 로 본다.

```go
if !entry.Annotated {
    entry.Tagger = "lightweight"
    entry.TaggedAt = time.Unix(entry.CommitUnix, 0)
    entry.Message = entry.Subject
}
```

이 fallback 을 쓰면 detail panel 에서 빈 줄이 사라진다.

### 3-3. Tag detail rendering

[`internal/app/view_detail.go`]( /Users/hrk/task/sources/opensources/graphkeeper/internal/app/view_detail.go ) 에서 Tags section detail 을 아래처럼 바꾼다.

```go
if m.activeSection == sectionTags {
    items := sectionTargets(m.repoStatus, sectionTags)
    if len(items) == 0 {
        lines = append(lines, muted.Render("  (empty)"))
        return lines
    }
    cursor := m.sectionCursor[sectionTags]
    if cursor < 0 || cursor >= len(items) {
        cursor = 0
    }
    item := items[cursor]
    lines = append(lines, fmt.Sprintf("selected: %s", item.Name))
    lines = append(lines, fmt.Sprintf("commit: %s", shorten(item.CommitHash, 7)))
    if item.Tagger != "" {
        lines = append(lines, fmt.Sprintf("tagger: %s", item.Tagger))
    } else {
        lines = append(lines, "tagger: lightweight")
    }
    if !item.TaggedAt.IsZero() {
        lines = append(lines, fmt.Sprintf("tagged: %s", compactWhenTime(item.TaggedAt)))
    }
    if item.Message != "" {
        lines = append(lines, fmt.Sprintf("message: %s", shorten(item.Message, max(width-9, 0))))
    }
}
```

### 3-4. Tag list compact rendering

Tags section list 자체는 compact 하게 유지한다. `renderSectionContent` 에서는 길어진 메타데이터를 다 넣지 말고, 선택된 row 의 detail panel 에서만 펼친다.

```go
case state.TargetKindTag:
    parts := []string{
        tagColor.Render(shorten(t.CommitHash, 7)),
        shorten(t.Name, 8),
    }
    if t.RelativeAge != "" {
        parts = append(parts, compactWhenText(t.RelativeAge))
    }
    return fitVisibleWidth(strings.Join(parts, "  "), width)
```

## 4. `compactWhenTime` helper

detail panel 에서 `TaggedAt` 와 `LastFetchAt` 를 읽기 좋은 문자열로 줄이는 helper 가 필요하다.

```go
func compactWhenTime(t time.Time) string {
    if t.IsZero() {
        return "-"
    }
    return t.Format("2006-01-02 15:04")
}
```

만약 상대시간 표기를 선호하면 `time.Since` 기반 helper 로 바꿔도 되지만,
이 문서는 구현 단순성을 우선해서 절대시간 포맷을 기준으로 둔다.

## 5. Tests

### 5-1. Remote tests

```go
func TestRepoStatusExposesLastFetchAt(t *testing.T) {
    fixture := newCommandRepo(t)
    if err := fixture.repo.Fetch(context.Background()); err != nil {
        t.Fatalf("fetch failed: %v", err)
    }
    status, err := fixture.repo.Status(context.Background(), 40)
    if err != nil {
        t.Fatalf("status failed: %v", err)
    }
    if status.LastFetchAt.IsZero() {
        t.Fatal("expected last fetch timestamp after fetch")
    }
}

func TestRemoteSyncSummary(t *testing.T) {
    status := git.Status{
        Root:      "/repo",
        Branch:    "main",
        Upstream:  "origin/main",
        Tracking:  map[string]git.BranchTracking{"main": {Ahead: 1, Behind: 0}},
        Remote:    "origin",
        NoRemote:  false,
        NoUpstream: false,
    }
    if got := remoteSyncSummary(status); got != "ahead 1" {
        t.Fatalf("unexpected sync summary: %q", got)
    }
}
```

### 5-2. Tag tests

```go
func TestLocalTagEntriesIncludesTaggerMetadata(t *testing.T) {
    fixture := newCommandRepo(t)
    runGit(t, fixture.root, "tag", "-a", "v1.0.0", "-m", "release 1.0.0")

    entries, err := fixture.repo.LocalTagEntries(context.Background())
    if err != nil {
        t.Fatalf("LocalTagEntries failed: %v", err)
    }
    if len(entries) == 0 {
        t.Fatal("expected at least one tag entry")
    }
    if entries[0].Tagger == "" {
        t.Fatal("expected annotated tag to include tagger")
    }
    if entries[0].Message == "" {
        t.Fatal("expected annotated tag to include message")
    }
    if entries[0].TaggedAt.IsZero() {
        t.Fatal("expected annotated tag to include tagged time")
    }
}

func TestTagDetailPanelShowsAnnotatedFields(t *testing.T) {
    got := strings.Join(renderContextInfoLines(model{
        status:        state.New().WithBrowse(),
        activeSection: sectionTags,
        repoStatus: git.Status{
            TagEntries: []git.TagEntry{{
                Name:       "v1.0.0",
                CommitHash: "abc1234",
                Tagger:     "Test User",
                TaggedAt:   time.Now(),
                Message:    "release 1.0.0",
            }},
        },
        sectionCursor: map[graphSection]int{sectionTags: 0},
    }, 80), "\n")
    if !strings.Contains(got, "tagger: Test User") {
        t.Fatalf("expected tagger line, got %q", got)
    }
    if !strings.Contains(got, "message: release 1.0.0") {
        t.Fatalf("expected message line, got %q", got)
    }
}
```

## 6. Implementation notes

- `TagProvenance` 는 그대로 둔다. provenance snapshot 은 origin 존재 여부만 담당한다.
- `TagEntry.Message` 는 provenance 와 무관하다.
- `LastFetchAt` 는 fetch 직후만이 아니라 `repo.Status` 가 읽을 수 있으면 언제든 보여준다.
- `RemoteSyncSummary` 는 표시용 파생값이라 캐시하지 않아도 된다.

## 7. Final checklist

- `Remote` detail panel 에 `last fetch` 와 `sync` 가 보인다.
- `Tags` detail panel 에 `tagger` / `tagged` / `message` 가 보인다.
- lightweight tag 는 빈 메타데이터 때문에 화면이 깨지지 않는다.
- annotated tag 는 추가 정보가 꽉 채워진다.
- 1차 rebalance 문서와 충돌하지 않는다.

