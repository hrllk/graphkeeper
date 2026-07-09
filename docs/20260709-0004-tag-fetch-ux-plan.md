# Graph Tag Fetch UX Plan

## 목적

Tag row 는 로컬 tag ref 를 바로 렌더링해야 한다.
`origin` 존재 여부는 별도 provenance metadata 로 다루고, fetch 후에는 다시 그릴 수 있어야 한다.

이 문서는 tag list 의 표시 상태와 fetch UX 를 분리해서 정리한다.

## 왜 이 문서가 필요한가

`refs/tags/*` 만으로는 tag 이름과 tag object 정보를 읽을 수 있다.
하지만 `origin 에도 있는 tag 인지` 는 Git ref 자체에 저장되지 않는다.

즉, 아래 두 가지는 다르다.

1. tag row 를 그릴 수 있는가
2. 그 tag 가 origin 에도 있었는지 복원할 수 있는가

첫 번째는 가능하다.
두 번째는 별도 메타데이터 없이는 불가능하다.

## 사용자 목표

1. Tag 섹션은 로컬에 있는 tag 를 즉시 보여준다.
2. tag row 는 `name`, `commit hash`, `subject`, `age` 를 보여준다.
3. origin 존재 여부는 있으면 같이 보여주고, 없으면 숨기거나 unknown 으로 처리한다.
4. restart 후에도 local tag 는 다시 즉시 렌더링되어야 한다.
5. fetch UX 는 한 번에 무엇을 가져오는지 분명해야 한다.

## 범위

### 포함

- local tag rendering
- tag provenance metadata
- origin presence cache
- fetch chooser UX
- tag fetch progress overlay
- restart 후 provenance 복원
- 관련 tests

### 제외

- tag rename
- tag delete
- annotated tag edit
- graph layout 전면 개편

## 현재 상태

현재 구현은 tag row 를 그릴 수는 있지만 provenance 는 안정적으로 보존하지 않는다.

- `internal/git/repo.go` 는 local tag object 를 읽는다.
- `internal/app/view_sections.go` 는 tag row 를 렌더링한다.
- `internal/app/tag_cache.go` 는 process memory 안에서만 tag entries 를 유지한다.
- `origin` 존재 여부는 fetch 시점의 별도 조회 결과에 의존한다.

그래서 restart 후에는 local tag 의 기본 정보는 다시 그릴 수 있지만, origin 여부는 다시 계산하지 않으면 잃는다.

## 핵심 결정

### 1. Tag row 의 canonical source 는 local ref 다

Tag row 에 필요한 기본 정보는 로컬 repo 만으로 만든다.

```go
type TagEntry struct {
	Name        string
	CommitHash  string
	Subject     string
	RelativeAge string
	CommitUnix  int64
}
```

이 데이터는 `refs/tags/*` 와 local object DB 만 있으면 다시 만들 수 있다.

즉, local tag 목록은 startup 시점에도 바로 렌더링 가능하다.
문제는 `origin` provenance 뿐이다.

### 2. origin 여부는 별도 provenance cache 로 저장한다

`origin` 존재 여부는 `TagEntry` 와 섞지 않는다.
이 값은 cache 로 따로 저장한다.

```go
type TagProvenance struct {
	Name      string
	OnOrigin  bool
	FetchedAt time.Time
}

type TagSnapshot struct {
	Entries     []TagEntry
	Provenance  map[string]TagProvenance
	LoadedAt    time.Time
}
```

추천 저장소는 repo-local metadata 파일이다.

```go
// .graphkeeper/tag-snapshot.json
{
  "loaded_at": "2026-07-09T12:34:56+09:00",
  "tags": [
    {
      "name": "v1.0.0",
      "commit_hash": "abc1234",
      "subject": "initial release",
      "relative_age": "2 days ago",
      "on_origin": true,
      "fetched_at": "2026-07-09T12:34:56+09:00"
    }
  ]
}
```

이 파일이 있으면 reboot 후에도 origin 여부를 다시 그릴 수 있다.
없으면 local tag 정보만 그린다.

### 3. fetch UX 는 source 를 먼저 묻고, 결과는 overlay toast 로 보여준다

Fetch 는 하나의 버튼으로 뭉개지지 않아야 한다.

권장 흐름:

```text
f -> fetch chooser
  -> branch fetch
  -> tag fetch
  -> fetch all

F -> tag fetch quick path
```

각 경로는 progress overlay 를 따로 띄운다.

```text
Fetching branches...
Fetching tags...
Fetching sources...
```

### 4. Tag row 에서 보여줄 상태는 세 가지다

```go
type TagOriginState string

const (
	TagOriginKnown   TagOriginState = "known"
	TagOriginMissing TagOriginState = "missing"
	TagOriginUnknown TagOriginState = "unknown"
)
```

표시 규칙:

- `known`  -> `(origin)`
- `missing` -> `(no-up)`
- `unknown` -> `(unknown)` 또는 badge 숨김

핵심은 `unknown` 과 `missing` 을 구분하는 것이다.
둘을 섞으면 restart 후에 거짓 확정이 생긴다.

최초 진입 시에는 local tag 가 즉시 보이고, provenance 가 아직 없으면 `unknown` 이다.
fetch 후에만 `known` / `missing` 을 채운다.

### 5. restart 후 온전한 재구성을 목표로 한다

이상적인 목표는 다음이다.

- local tag data 는 항상 재구성 가능
- provenance cache 가 있으면 origin 표시도 재구성 가능
- cache 가 없으면 origin 여부는 unknown 으로 시작

이 방식이면 tag fetch 후 reboot 해도 동일한 최종 화면을 다시 그릴 수 있다.

## 대안 비교

### A. ref-only rendering

장점:

- 구현이 가장 단순하다
- local tag 정보는 바로 보인다

단점:

- origin 여부를 restart 후 복원할 수 없다
- `no-up` 를 tag 에 적용하는 것은 정확하지 않다

### B. provenance cache + progressive fetch

장점:

- restart 후에도 origin 여부를 복원할 수 있다
- local rendering 과 remote provenance 를 분리할 수 있다
- fetch UX 를 명확하게 만들 수 있다

단점:

- metadata 파일과 refresh 경로가 하나 더 생긴다
- test surface 가 늘어난다

### C. startup 에서 매번 remote lookup

장점:

- provenance cache 를 따로 설계하지 않아도 된다

단점:

- startup 이 느려진다
- 네트워크 실패가 UI 전체를 흔든다
- 이번 repo 의 목표와 반대다

## 추천

### 추천안: B. provenance cache + progressive fetch

이게 이상적이다.

- tag row 는 local data 로 즉시 그린다.
- origin 여부는 cache 가 있으면 붙인다.
- fetch 는 사용자가 선택한다.
- restart 후에도 local tag 는 다시 즉시 그린다.
- provenance cache 가 있으면 origin 여부까지 복원한다.

`no-up` 를 tag 에도 쓰고 싶다면, 그건 `origin 존재 여부가 cached and confirmed` 인 경우에만 허용한다.
confirmed 가 아니면 `unknown` 으로 남기는 편이 맞다.

## BEFORE

현재는 tag 표시와 provenance 가 분리돼 있지 않다.

```go
func buildTagSectionTargets(rs git.Status) []state.TargetItem {
	if len(rs.TagEntries) > 0 {
		items := make([]state.TargetItem, 0, len(rs.TagEntries))
		for _, tag := range rs.TagEntries {
			items = append(items, state.TargetItem{
				Kind:        state.TargetKindTag,
				Name:        tag.Name,
				Ref:         tag.Name,
				CommitHash:  tag.CommitHash,
				Subject:     tag.Subject,
				RelativeAge: tag.RelativeAge,
				OnOrigin:    tag.OnOrigin,
			})
		}
		return items
	}
	return nil
}
```

```go
func formatTargetItem(t state.TargetItem) string {
	switch t.Kind {
	case state.TargetKindTag:
		source := warn.Render("(no-up)")
		if t.OnOrigin {
			source = remoteColor.Render("(origin)")
		}
		return fmt.Sprintf("%-24s  %-28s  %-10s  %s",
			t.Name,
			compactTitleText(t.Subject),
			compactWhenText(t.RelativeAge),
			source,
		)
	default:
		return t.Name
	}
}
```

## AFTER

### Git layer

```go
func (r *Repo) TagEntries(ctx context.Context) ([]TagEntry, error) {
	names, err := r.gitLines(ctx, "for-each-ref", "--format=%(refname:short)", "refs/tags")
	if err != nil {
		return nil, err
	}

	originTags, err := r.originTagSet(ctx)
	if err != nil {
		originTags = map[string]bool{}
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
		entries = append(entries, TagEntry{
			Name:        strings.TrimSpace(name),
			CommitHash:  strings.TrimSpace(target),
			Subject:     strings.TrimSpace(subject),
			RelativeAge: strings.TrimSpace(relativeAge),
		})
	}

	return entries, nil
}
```

```go
func (r *Repo) FetchTags(ctx context.Context) error {
	_, err := r.git(ctx, "fetch", "origin", "--tags", "--prune")
	return err
}
```

### Snapshot cache

```go
func (m *model) storeTagSnapshot(snapshot TagSnapshot) {
	if len(snapshot.Entries) == 0 {
		return
	}
	m.tagSnapshot = snapshot
	_ = writeTagSnapshot(m.repoStatus.Root, snapshot)
}

func (m model) withCachedTagSnapshot(status git.Status) git.Status {
	if len(status.TagEntries) > 0 {
		return status
	}
	if len(m.tagSnapshot.Entries) == 0 {
		return status
	}
	status.TagEntries = append([]git.TagEntry(nil), m.tagSnapshot.Entries...)
	return status
}
```

### Render states

```go
func renderTagOriginBadge(state TagOriginState) string {
	switch state {
	case TagOriginKnown:
		return remoteColor.Render("(origin)")
	case TagOriginMissing:
		return warn.Render("(no-up)")
	default:
		return muted.Render("(unknown)")
	}
}
```

```go
func formatTagRow(t state.TargetItem) string {
	badge := renderTagOriginBadge(t.TagOriginState)
	return fmt.Sprintf("%-24s  %-28s  %-10s  %s",
		t.Name,
		compactTitleText(t.Subject),
		compactWhenText(t.RelativeAge),
		badge,
	)
}
```

### Fetch chooser

```go
func handleGlobalFetchKey(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "f":
		m.status = state.New().WithConfirm(state.ActionFetch, "Fetch sources?", "Choose branch fetch or tag fetch.")
		m.status.Title = "Fetch sources?"
		m.status.Detail = "f: branches  •  F: tags  •  esc: cancel"
		return m, nil
	case "F":
		m.status = loadingToast("Fetching tags...")
		return m, fetchTagsRepoState(m.repo, m.commitLimit)
	}
	return m, nil
}
```

### Test matrix

```go
func TestTagSnapshotPersistsAcrossRestart(t *testing.T)
func TestTagRowShowsUnknownWhenNoProvenanceCache(t *testing.T)
func TestTagFetchChooserRoutesBranchAndTagFetchSeparately(t *testing.T)
func TestTagFetchProgressToastIsSpecific(t *testing.T)
func TestTagListReloadAfterFetchDoesNotLoseRows(t *testing.T)
```

## Tests

- local tag rendering snapshot test
- cache persistence test
- restart restore test
- fetch chooser key handling test
- tag fetch overlay test
- missing provenance fallback test

## Verification

```sh
go test ./internal/app ./internal/git
go test ./...
```

## 결론

`no-up` 를 tag 에 그대로 복제하는 건 ref 구조상 정확하지 않다.
이상적인 방법은 provenance 를 별도 cache 로 두고, local tag rendering 과 remote provenance 를 분리하는 것이다.

최종 시나리오는 다음처럼 정리한다.

1. 최초 진입 -> local tag 는 즉시 렌더링, origin 여부는 unknown / 미표시
2. Fetching Tags -> provenance 갱신
3. Restart -> 1번으로 다시 시작, local tag 는 다시 즉시 렌더링

이 문서의 구현 목표는 다음 한 줄로 정리된다.

> Tag 는 항상 그릴 수 있어야 하고, origin 여부는 알 때만 말해야 한다.
