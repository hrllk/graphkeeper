# Tag Provenance Cache Implementation Plan

## 목적

`Tag` 섹션은 local tag 를 즉시 렌더링한다.
`origin` provenance 는 `graphkeeper` 가 fetch 를 수행한 뒤 별도 파일에 기록한다.

이 문서는 Git 동작을 바꾸는 계획이 아니라,
`graphkeeper` 내부 method -> hotkey -> fetch -> 결과 기록 -> file write
라는 앱 전용 흐름을 구현하는 계획이다.

## 핵심 결론

전략은 아래처럼 고정한다.

1. startup 은 local tag 만 먼저 읽는다.
2. snapshot 이 있으면 origin provenance 만 덮어쓴다.
3. `F` 는 `git fetch --tags` 와 `git ls-remote --tags origin` 을 함께 수행한다.
4. fetch 결과와 remote tags 정보를 합쳐 provenance snapshot 을 쓴다.
5. 다음 startup 때 그 파일을 읽어서 `origin` 표시를 복원한다.
6. `t` 는 local tag 생성, `P` 는 tag push, `F` 는 provenance sync 로 역할을 분리한다.
7. `t` 직후 provenance 는 `unknown` 이다.
8. `P` 성공 후 provenance 는 `synced` 이다.
9. `d` 는 local delete, `D` 는 remote delete 로 분리한다.

즉, raw `git fetch --tags` 를 가로채는 게 아니라,
`graphkeeper` 가 fetch 를 “대행”할 때만 provenance 를 갱신한다.

## 왜 이 문서가 필요한가

태그 자체는 local ref 와 commit object 만 있으면 렌더링할 수 있다.
하지만 `origin` 여부는 Git ref 가 자동으로 유지하지 않는다.

그래서 필요한 건 두 층이다.

1. local tag snapshot
2. origin provenance snapshot
3. sync summary state

이 셋을 분리하지 않으면 restart 후 상태가 흔들린다.

## 사용자 목표

1. 최초 진입 시 local tag 는 바로 보인다.
2. origin provenance 는 없으면 `unknown` 으로 시작한다.
3. `F` 또는 fetch hotkey 로 provenance 를 갱신한다.
4. 갱신 결과는 file 에 남아서 restart 후에도 복원된다.
5. raw git 명령을 직접 써도 앱은 synced / unknown 상태를 명확히 다룬다.
6. local tag 가 아예 없을 때만 empty state 를 보여준다.
7. summary state 는 `never synced` 와 `synced` 두 개만 쓴다.
8. `F` 는 tag 충돌을 자동 overwrite 하지 않고, 비파괴적으로 실패한다.

## 범위

### 포함

- local tag rendering
- tag provenance snapshot file
- fetch 후 provenance 계산
- startup 시 local tag read + snapshot overlay
- hotkey -> method -> write flow
- tag push / tag fetch conflict handling
- 관련 tests

### 제외

- tag rename
- tag delete
- Git hook 설치
- raw `git fetch` 가 모든 경우에 자동 반영되도록 만드는 것

## 현재 상태

현재 코드는 tag 를 한 덩어리로 다루고 있다.

- `internal/git/repo.go` 의 `Status()` 는 tag snapshot 을 비워서 반환한다.
- `internal/git/repo_exec.go` 의 `TagEntries()` 는 local tag 와 remote provenance 를 같이 계산한다.
- `internal/app/tag_cache.go` 는 process memory 캐시만 유지한다.
- `internal/app/view_sections.go` 는 tag row 를 렌더링한다.
- `internal/app/commands.go` 는 fetch 후 `TagEntries()` 를 다시 읽는다.

즉, fetch 직후 표시할 수는 있지만, restart 복원과 local-only startup 경로가 없다.

## 핵심 결정

### 1. local tag 와 origin provenance 를 분리한다

local tag 는 local ref/object 만으로 읽는다.
origin provenance 는 별도 snapshot 으로 저장한다.

```go
type TagEntry struct {
	Name        string
	CommitHash  string
	Subject     string
	RelativeAge string
	CommitUnix  int64

	// provenance 가 아직 확인되지 않았으면 false.
	OriginKnown bool
	// provenance 가 확인된 뒤에만 의미가 있다.
	OnOrigin bool
}
```

`OriginKnown` 이 핵심이다.
이 값이 `false` 이면 `OnOrigin=false` 여도 `(unknown)` 으로 남아야 한다.
그 상태는 아직 모른다는 뜻이어야 한다.

### 2. git layer 에는 두 개의 조회 경로를 둔다

`TagEntries()` 하나로 다 해결하지 말고, 역할을 나눈다.

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
		target, err := r.git(ctx, "rev-parse", "--verify", name+"^{commit}")
		if err != nil {
			continue
		}
		subject, err := r.git(ctx, "show", "-s", "--format=%s", target)
		if err != nil {
			continue
		}
		relativeAge, err := r.git(ctx, "show", "-s", "--format=%cr", target)
		if err != nil {
			continue
		}
		commitUnixText, err := r.git(ctx, "show", "-s", "--format=%ct", target)
		if err != nil {
			continue
		}
		commitUnix, err := strconv.ParseInt(strings.TrimSpace(commitUnixText), 10, 64)
		if err != nil {
			continue
		}

		entries = append(entries, TagEntry{
			Name:        strings.TrimSpace(name),
			CommitHash:  strings.TrimSpace(target),
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

func (r *Repo) OriginTagSet(ctx context.Context) (map[string]bool, error) {
	lines, err := r.gitLines(ctx, "ls-remote", "--tags", "origin")
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, nil
	}
	tags := make(map[string]bool, len(lines))
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ref := strings.TrimPrefix(parts[1], "refs/tags/")
		ref = strings.TrimSuffix(ref, "^{}")
		if ref == "" {
			continue
		}
		tags[ref] = true
	}
	return tags, nil
}
```

### 3. snapshot 파일은 summary 와 provenance 만 저장한다

local tag 정보는 startup 때마다 다시 읽는다.
snapshot 은 remote provenance 와 sync summary 만 저장한다.

```go
type TagSyncSummary string

const (
	TagSyncNeverSynced TagSyncSummary = "never_synced"
	TagSyncSynced      TagSyncSummary = "synced"
)

type TagSnapshot struct {
	LoadedAt   time.Time        `json:"loaded_at"`
	Summary    TagSyncSummary   `json:"summary"`
	OriginSeen map[string]bool  `json:"origin_seen"`
}

func tagSnapshotPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".git", "graphkeeper", "tag-provenance.json")
}
```

권장 위치는 아래 하나로 고정한다.

```text
.git/graphkeeper/tag-provenance.json
```

### 4. startup 은 local tag 를 즉시 렌더링하고 snapshot 이 있으면 overlay 한다

startup 때는 local tag 를 먼저 읽고, snapshot 이 있으면 provenance 만 덮어쓴다.

```go
func loadRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return loadedMsg{status: status, err: err}
		}

		tags, tagErr := repo.LocalTagEntries(context.Background())
		if tagErr == nil {
			status.TagEntries = tags
			status.TagEntriesLoaded = true
			status.Tags = make([]string, 0, len(tags))
			for _, entry := range tags {
				status.Tags = append(status.Tags, entry.Name)
			}
		}

		snapshot, snapErr := loadTagSnapshot(status.Root)
		if snapErr == nil {
			status = applyTagSnapshot(status, snapshot)
		} else if len(status.TagEntries) > 0 {
			status = markTagOriginUnknown(status)
		}

		return loadedMsg{status: status, err: nil}
	}
}
```

`loadRepoState` 는 remote fetch 를 하지 않는다.
local tag 가 있으면 바로 렌더링한다.

### 5. `F` 는 fetch + ls-remote + write 를 수행한다

`F` 는 앱이 직접 provenance snapshot 을 갱신하는 트리거다.
`t` 는 local tag 생성만 하고, `P` 는 선택된 tag 를 origin 에 push 하는 별도 액션이다.
`F` 는 tag list refresh 와 provenance sync 를 수행하지만, tag ref 충돌이 나면 덮어쓰지 않는다.

```go
func fetchTagsRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		if err := repo.FetchTags(context.Background()); err != nil {
			return fetchedMsg{err: err}
		}

		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return fetchedMsg{status: status, err: err}
		}

		tags, err := repo.LocalTagEntries(context.Background())
		if err != nil {
			return fetchedMsg{status: status, err: err}
		}
		remoteTags, err := repo.OriginTagSet(context.Background())
		if err != nil {
			return fetchedMsg{status: status, err: err}
		}

		status.TagEntries = tags
		status.TagEntriesLoaded = true
		status.Tags = make([]string, 0, len(tags))
		for _, entry := range tags {
			status.Tags = append(status.Tags, entry.Name)
		}

		snapshot := buildTagSnapshot(tags, remoteTags, TagSyncSynced)
		if err := writeTagSnapshot(status.Root, snapshot); err != nil {
			return fetchedMsg{status: status, err: err}
		}

		status = applyTagSnapshot(status, snapshot)
		return fetchedMsg{status: status, err: nil}
	}
}
```

### 5-1. tag push 는 `P` 로 분리한다

`P` 는 Tags 섹션에서 선택된 tag 를 origin 에 push 하는 명시적 액션이다.
push 가 성공하면 provenance snapshot 을 synced 로 다시 계산하거나, 곧바로 `ls-remote --tags origin` 을 다시 돌려 갱신한다.

### 5-2. `F` 는 tag 충돌을 자동 overwrite 하지 않는다

remote tag 가 local tag 와 이름 충돌을 일으키거나 내용이 달라 fetch 가 실패하면,
앱은 해당 충돌을 숨기지 않고 비파괴적으로 실패 처리한다.

```go
func fetchTagsRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		if err := repo.FetchTags(context.Background()); err != nil {
			return fetchedMsg{err: err}
		}
		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return fetchedMsg{status: status, err: err}
		}
		tags, err := repo.LocalTagEntries(context.Background())
		if err != nil {
			return fetchedMsg{status: status, err: err}
		}
		remoteTags, err := repo.OriginTagSet(context.Background())
		if err != nil {
			return fetchedMsg{status: status, err: err}
		}
		// overwrite 하지 않고, 최신 remote snapshot 을 기준으로만 다시 계산한다.
		snapshot := buildTagSnapshot(tags, remoteTags, TagSyncSynced)
		if err := writeTagSnapshot(status.Root, snapshot); err != nil {
			return fetchedMsg{status: status, err: err}
		}
		status = applyTagSnapshot(status, snapshot)
		return fetchedMsg{status: status, err: nil}
	}
}
```

### 5-3. 키맵 요약

```text
t : local tag 생성
P : selected tag push
F : tag provenance sync / fetch
d : local tag delete
D : remote tag delete
```

### 6. snapshot 적용은 local tag 를 지우지 않는다

snapshot 이 없거나 손상돼도 local tag 목록은 유지한다.

```go
func applyTagSnapshot(status git.Status, snapshot TagSnapshot) git.Status {
	for i := range status.TagEntries {
		entry := &status.TagEntries[i]
		onOrigin, ok := snapshot.OriginSeen[entry.Name]
		entry.OriginKnown = ok
		entry.OnOrigin = ok && onOrigin
	}
	return status
}

func markTagOriginUnknown(status git.Status) git.Status {
	for i := range status.TagEntries {
		status.TagEntries[i].OriginKnown = false
		status.TagEntries[i].OnOrigin = false
	}
	return status
}

func buildTagSnapshot(entries []git.TagEntry, remoteTags map[string]bool, summary TagSyncSummary) TagSnapshot {
	originSeen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		originSeen[entry.Name] = remoteTags[entry.Name]
	}
	return TagSnapshot{
		LoadedAt:   time.Now(),
		Summary:    summary,
		OriginSeen: originSeen,
	}
}
```

### 7. empty state 는 local tag 가 아예 없을 때만 쓴다

local tag 가 1개 이상 있으면 그 목록은 즉시 보여주고,
provenance 만 비어 있을 때는 unknown 으로 남긴다.

```text
No tag snapshot yet.
Fetch 를 통해 Sync를 진행해주세요.
```

### 8. summary state 는 two-state 로 단순화한다

persisted summary 값과 user-facing label 을 분리한다.

`TagSyncSummary` 의 persisted token 은 위의 `never_synced` / `synced` 를 그대로 쓰고,
UI 는 아래처럼 표시만 바꾼다.

```go
func renderTagSyncSummary(summary TagSyncSummary) string {
	switch summary {
	case TagSyncNeverSynced:
		return "never synced"
	case TagSyncSynced:
		return "synced"
	default:
		return "never synced"
	}
}
```

표시 규칙:

- `never synced` -> `Fetch 를 통해 Sync를 진행해주세요`
- `synced` -> `Tag provenance is synced. Press F to refresh.`

summary 는 마지막 동기화 상태를 말하고,
provenance 는 각 tag 의 origin 여부를 말한다.

## Render 규칙

Tag row 는 local tag 기준으로 먼저 그린다.
origin provenance 가 확인된 경우에만 badge 를 표시한다.

```go
func formatTargetItem(t state.TargetItem) string {
	switch t.Kind {
	case state.TargetKindTag:
		base := fmt.Sprintf("%-24s  %-28s  %-10s",
			t.Name,
			compactTitleText(t.Subject),
			compactWhenText(t.RelativeAge),
		)
		switch {
		case !t.OriginKnown:
			return base + "  " + muted.Render("(unknown)")
		case t.OnOrigin:
			return base + "  " + remoteColor.Render("(origin)")
		default:
			return base + "  " + warn.Render("(unknown)")
		}
	default:
		return t.Name
	}
}
```

`state.TargetItem` 에도 `OriginKnown bool` 을 추가해야 한다.
기존 `OnOrigin bool` 만으로는 unknown 과 missing 을 구분할 수 없다.
`d` 는 local ref 삭제이므로 row 자체가 사라지고, `D` 는 remote tag 삭제이므로 local tag 는 남아 `unknown` 으로 회귀한다.

## BEFORE

현재는 fetch 결과를 메모리에서만 유지한다.

```go
func (m *model) storeTagEntries(status git.Status) {
	if len(status.TagEntries) == 0 {
		return
	}
	m.tagEntries = append([]git.TagEntry(nil), status.TagEntries...)
}
```

```go
func fetchTagsRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		if err := repo.FetchTags(context.Background()); err != nil {
			return fetchedMsg{err: err}
		}
		status, err := repo.Status(context.Background(), limit)
		if err == nil {
			status = attachTagEntries(repo, status)
		}
		return fetchedMsg{status: status, err: err}
	}
}
```

이 구조는 restart 복원도, local-only startup 도 못 한다.

## AFTER

### startup flow

```go
func loadRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return loadedMsg{status: status, err: err}
		}

		tags, tagErr := repo.LocalTagEntries(context.Background())
		if tagErr == nil {
			status.TagEntries = tags
			status.TagEntriesLoaded = true
			status.Tags = make([]string, 0, len(tags))
			for _, entry := range tags {
				status.Tags = append(status.Tags, entry.Name)
			}
		}

		if snapshot, snapErr := loadTagSnapshot(status.Root); snapErr == nil {
			status = applyTagSnapshot(status, snapshot)
		} else if len(status.TagEntries) > 0 {
			status = markTagOriginUnknown(status)
		}

		return loadedMsg{status: status, err: nil}
	}
}
```

### fetch flow

```go
func fetchTagsRepoState(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		if err := repo.FetchTags(context.Background()); err != nil {
			return fetchedMsg{err: err}
		}

		status, err := repo.Status(context.Background(), limit)
		if err != nil {
			return fetchedMsg{status: status, err: err}
		}

		tags, err := repo.LocalTagEntries(context.Background())
		if err != nil {
			return fetchedMsg{status: status, err: err}
		}

		remoteTags, err := repo.OriginTagSet(context.Background())
		if err != nil {
			return fetchedMsg{status: status, err: err}
		}

		snapshot := buildTagSnapshot(tags, remoteTags, TagSyncSynced)
		if err := writeTagSnapshot(status.Root, snapshot); err != nil {
			return fetchedMsg{status: status, err: err}
		}

		status.TagEntries = tags
		status.TagEntriesLoaded = true
		status.Tags = make([]string, 0, len(tags))
		for _, entry := range tags {
			status.Tags = append(status.Tags, entry.Name)
		}
		status = applyTagSnapshot(status, snapshot)

		return fetchedMsg{status: status, err: nil}
	}
}
```

### snapshot writer

```go
func writeTagSnapshot(repoRoot string, snapshot TagSnapshot) error {
	path := tagSnapshotPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
```

## Tests

### Git layer tests

```go
func TestLocalTagEntries(t *testing.T)
func TestOriginTagSet(t *testing.T)
func TestWriteTagSnapshot(t *testing.T)
func TestLoadTagSnapshot(t *testing.T)
func TestBuildTagSnapshot(t *testing.T)
```

### App layer tests

```go
func TestStartupLoadsLocalTagsWithoutSnapshot(t *testing.T)
func TestStartupOverlaysSnapshotWhenPresent(t *testing.T)
func TestFetchTagsWritesSnapshot(t *testing.T)
func TestUnknownTagOriginDoesNotRenderAsNoUp(t *testing.T)
func TestTagFetchHotkeyShowsSpecificToast(t *testing.T)
```

### Assertion focus

- snapshot 이 없어도 local tag 가 보이는지
- snapshot 이 있으면 origin badge 만 복원되는지
- `OriginKnown=false` 인 경우 `(unknown)` 을 절대 쓰지 않는지
- `F` 후 snapshot 이 실제로 파일에 써지는지

## Verification

```sh
go test ./internal/app ./internal/git
go test ./...
```

## 결론

전략은 맞다.

- `graphkeeper` 안의 method 가 fetch 를 수행한다.
- hotkey 가 그 method 를 호출한다.
- startup 은 local tag 를 즉시 렌더링한다.
- `F` 가 `ls-remote --tags origin` 을 함께 수행한다.
- fetch 결과와 remote tags 정보를 provenance 파일로 쓴다.
- startup 때 그 파일을 읽는다.
- snapshot 이 없으면 empty state 에서 fetch 를 유도한다.
- `P` 는 tag push 전용 키로 분리한다.
- `F` 는 충돌을 덮어쓰지 않고 비파괴적으로 실패한다.
- `d` 와 `D` 는 local/remote delete 를 분리한다.

사용자 노출 summary 는 `never synced` 와 `synced` 두 개만 쓴다.

이 방식이면 raw `git fetch --tags` 를 커스터마이즈하지 않아도 된다.
앱이 자기 책임 범위 안에서 provenance 를 관리하면 된다.
