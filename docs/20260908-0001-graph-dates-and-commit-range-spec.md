# Spec: Graph 날짜 노출과 두 커밋 구간 경로 하이라이팅

상태: REVIEWED — eng-review 1회 (HOLD SCOPE), 6건 반영. 즉시 구현 가능.
소유: 이 문서가 **타입 레벨 계약**을 소유한다. 왜·무엇을은 계획 문서가 소유한다.
계획: `docs/20260907-0001-graph-dates-and-commit-range-plan.md`
설계: `~/.gstack/projects/hrllk-graphkeeper/hrk-main-design-20260907-224500.md`
기준 커밋: `4695926`
Taskmaster: task 11 (subtask 11.1~11.9)

Flags: dedupe=OFF (백로그가 taskmaster이고 GitHub 이슈를 쓰지 않는다), gate=수동
(codex), audit=OFF, execute=OFF (`--file-only` 상당 — 저장소 문서로 남기고 이슈를
발행하지 않는다)

---

## Context

graphkeeper는 graph-first Git TUI다. 설계 문서에 제품 목적이 "도구는 판단을 돕고,
최종 판단은 사람이 한다", "부사수에게 형상관리하는 방법을 교육"으로 기록되어 있다.
현재 세 가지가 그 목적을 막는다.

1. 커밋이 **언제** 만들어졌는지 어디에도 없다. graph 행에 날짜 컬럼이 없고, graph
   섹션 Details 패널에도 없다. 같은 Details의 tags 섹션에는 `age:`가 있어 비대칭이다.
2. Commit Inspector에도 없다. 데이터 계층부터 없어서 표시만 고칠 수 없다.
3. 두 커밋의 관계를 도구에 물어볼 방법이 없다. "이 둘은 왜 fast-forward가 안 되나"를
   사람이 말로 설명해야 한다.

3번이 제품 목적과 직결된다. `git rev-list --ancestry-path A..B`가 비어서 돌아오면
그것이 곧 "조상 관계가 없다 = 분기했다 = FF 불가"라는 답이다.

## Current State

2026-09-08, 커밋 `4695926`에서 검증했다.

| 사실 | 근거 |
|---|---|
| graph log는 상대 시간만 가져온다 | `internal/git/repo_parse.go:186` `--format=%x00%H%x1f%P%x1f%ar%x1f%an%x1f%D%x1f%s` |
| graph 행에 날짜/age 컬럼이 없다 | `internal/app/graph_render.go:97` `hash + " " + refs + " " + status + " " + graphCell + " " + title` |
| 폭 7의 `date` 컬럼이 있었으나 제거됐다 | `397b2ea` (task 1.8, status done). 내용은 `compactWhenText(RelativeAge)` = `3d`, `2mo`. 절대 날짜는 이 저장소에 존재한 적 없다 |
| Details graph 섹션에 날짜가 없다 | `internal/app/view_detail.go:27-38` — `focus`/`branches`/`stashes`/`tags`만 |
| Details tags 섹션에는 `age:`가 있다 | `internal/app/view_detail.go:78` |
| `CommitSnapshot`에 날짜 필드가 없다 | `internal/commitinspector/contract.go:93-103` |
| `CommitInspection`에 날짜 필드가 없다 | `internal/git/repo.go:110-115` |
| Inspector 메타데이터는 5필드다 | `internal/git/repo_exec.go:162` `--format=%H%x00%P%x00%an%x00%ae%x00%s` + `SplitN(meta, "\x00", 5)` (:166), `len(parts) != 5` (:167) |
| graph 행 렌더러가 둘이고 raw가 실사용 경로다 | `graph_render.go:55` `renderGraphLineWithSearch` → `row.Graph != ""`이면 `:100` `renderRawGraphLineWithSearch`로 위임 |
| Inspector 렌더러는 하나만 살아 있다 | `view_shell.go:49`가 `renderCommitInspectorScreen`만 디스패치. `commit_inspector.go:122` `renderCommitInspectorPopup`은 호출자가 `commit_inspector_test.go:74`, `:98` 뿐 |
| anchor 개념이 없다 | `internal/app/model.go:114-122` `navigationState`에 range/anchor 필드 없음 |
| rev-list 호출 관습이 있다 | `internal/git/repo_exec.go:829` `Divergence`, 빈 ref 가드 :830-832 |
| 해시 집합 전달 관습이 있다 | `internal/app/view_projection.go:30` `Handshake map[string]bool` |
| epoch 무효화 관습이 있다 | `commit_inspector_helpers.go:46`, `key_handling_browse.go:215,220`, `update_lifecycle.go:42,62,77` |
| `v` 키가 비어 있다 | 바인딩된 33개: `? / 1 2 3 4 a backspace c ctrl+c ctrl+d ctrl+u d D down enter esc f F g G h H j k l left m n N o p P q r right s S shift+tab space t tab up x y` |
| **graph 행이 80컬럼 이하에서 이미 폭을 초과한다** | `shell_width_test.go:113-118`이 80→graph 43 고정, `view_shell.go:56`이 `graphWidth-4`=39를 넘김, `graphRowFixedWidth`=41 → title 가용폭 **-2**. 40컬럼 -25, 60컬럼 -14 |

### 실측으로 확인한 git 동작

```
$ git log -1 --format='cI=%cI%naI=%aI%ncs=%cs'
cI=2026-09-08T23:15:16+09:00      # --date 플래그 없이 원본 오프셋
aI=2026-09-08T23:15:16+09:00
cs=2026-09-08                     # 시각 손실

$ git rev-list --ancestry-path <HEAD~3>..<HEAD>
469592663ccb...   # == HEAD (to)
d104a4b6e425...
901b06acc2d2...   # HEAD~3 (from) 은 없음
```

- `A..B`는 **A를 제외하고 B를 포함**한다. 3커밋 구간에 3개가 나온다.
- 출력은 **최신 우선**이다. `result[0] == to`.
- **동일 커밋** `B..B` → 0개.
- **역방향** `B..A` (B가 A의 자손) → 0개.

마지막 두 줄이 이 spec의 핵심 설계 근거다. 빈 결과는 세 가지를 뭉갠다.

## Proposed Change

### 판정 흐름

```
  v 누름 ─┬─ anchor 없음 ──▶ anchor = 커서 해시 (합성 행이면 거부)
          │                   하이라이트 없음, anchor 마커만
          │
          └─ anchor 있음 ──▶ cursor == anchor ?
                              ├─ 예 ──▶ RangeSame
                              └─ 아니오 ──▶ AncestryPath(anchor, cursor)
                                            ├─ err ────────▶ RangeUnavailable
                                            ├─ len > 0 ────▶ RangeForward
                                            └─ len == 0 ──▶ AncestryPath(cursor, anchor)
                                                            ├─ err ────────▶ RangeUnavailable
                                                            ├─ len > 0 ────▶ RangeBackward
                                                            └─ len == 0 ──▶ RangeDiverged
```

git 호출은 최악 2회, 정방향이면 1회다.

### 상태 생애

```
   [Off] ──v──▶ [AnchorOnly] ──v──▶ [Active] ──esc/v──▶ [Off]
     ▲               │                  │
     │               │                  │ repositoryEpoch 증가
     └───────────────┴──────────────────┘   (fetch/pull/mutating action)
                  전부 [Off] 로. 조용히 유지하지 않는다.
```

---

## Implementation Details

**구현자가 내릴 설계 결정이 없어야 한다.** 아래가 계약이다.

### C-1. `internal/git` — `AncestryPath`

```go
// AncestryPath returns the commit hashes on the ancestry path from `from` to
// `to`, as `git rev-list --ancestry-path from..to` reports them.
//
// Range semantics follow git's `from..to` exactly:
//   - `from` is EXCLUDED from the result.
//   - `to` is INCLUDED in the result.
//   - Order is newest-first, so result[0] == to whenever the result is non-empty.
//
// A nil result with a nil error means there is no ancestry path in this
// direction. That is a valid answer, not a failure: it happens when the two
// commits have diverged, when `from` is a descendant of `to`, and when
// from == to. Callers MUST distinguish these by asking again in the other
// direction and by comparing the two refs first; this function does not.
// Callers MUST test emptiness with len(), never with != nil.
//
// Returns an error when either ref is empty, or when git fails.
func (r *Repo) AncestryPath(ctx context.Context, from, to string) ([]string, error)
```

구현 요구사항:

1. **빈 ref 가드가 첫 줄이다.** `Divergence`(`repo_exec.go:830-832`)와 동일하게
   `if from == "" || to == ""` 를 명시적으로 거부한다. git에서 `..B`는 `HEAD..B`로
   해석되므로 가드가 없으면 **오류 없이 전혀 다른 질문에 답한다.**
2. `r.git(ctx, "rev-list", "--ancestry-path", from+".."+to)` 로 호출한다.
   `Divergence`가 `left+"..."+right`를 만드는 것과 같은 관습이다.
3. 출력을 `strings.Fields`로 자른다. 빈 출력이면 `nil, nil`.
4. **상한을 두지 않는다.** 이 저장소는 이미 `commitLimit = 0`
   (`update_execute.go:306`)으로 무제한 `git log`를 돌린다. 상한을 두면 "n commits"가
   거짓이 되므로, 무제한이 기존 동작과 일관되고 정직하다.
5. `r.git(ctx, ...)`(`repo_exec.go:884`)을 쓰고 `exec.Command`를 직접 만들지 않는다.
   `r.git`이 repo 디렉터리 지정과 오류 래핑을 이미 소유하므로, 직접 만들면 그 둘을
   호출부에서 다시 구현하게 된다.

   **주의 (eng-review E-1 정정).** 이 문단의 초안은 "`internal/architecture` 가드가
   `os/exec`를 금지한다"를 근거로 적었다. **그 근거는 틀렸다.**
   `internal/architecture/guard_test.go:91-95`가 이유다.

   ```go
   // Existing legacy packages are scanned to keep the baseline observable, but
   // their known coupling is not treated as a new-boundary failure.
   for _, rel := range []string{"internal/app", "internal/git", "internal/graph", "internal/adapter"} {
       if _, err := ScanDirWithLedger(root, rel, forbiddenImportsForPackage(rel), entries); err != nil {
   ```

   `internal/git`에는 `len(violations) != 0` 단언이 **없다.** 위반 단언이 붙는 패키지는
   `guard_test.go:71`의 세 곳뿐이다 — `internal/architecture/testdata/extracted`,
   `internal/commitinspector`, `internal/events`. 그리고
   `internal/git/repo_exec.go:10`이 이미 `"os/exec"`를 import한다. 가드는 여기서
   아무것도 막지 않는다. 규칙 자체는 유지하되 근거를 일관성으로 바꾼다.

### C-2. `internal/git` — 날짜 필드

```go
type GraphCommit struct {
    Graph       string
    Hash        string
    Parents     []string
    RelativeAge string
    Author      string
    Decorations []string
    Subject     string
    Tags        []string
    CommitDate  string // NEW. strict ISO 8601 from %cI, e.g. "2026-09-08T23:15:16+09:00".
                       // "" means git did not report it. Never parsed in this layer.
}
```

```go
type CommitInspection struct {
    Hash, Subject, Author, Message, Parent string
    AuthorDate, CommitDate                 string // NEW. strict ISO 8601 from %aI / %cI.
    IsRoot                                 bool
    Parents                                []string
    Files                                  []CommitDiffFile
}
```

**타입은 `string`이고 `time.Time`이 아니다.** 근거 세 가지.

1. 이 계층은 날짜 산술을 하지 않는다. 표시만 한다.
2. 형제 필드 `RelativeAge string`과 일관된다.
3. `time.Time`으로 파싱하면 파싱 실패라는 **새 오류 경로**가 생기고, 그 경로가
   `repo_parse.go`의 무음 `continue` 안에 들어가 커밋을 조용히 버릴 수 있다.
   문자열로 나르면 그 위험이 없다.

`TagEntry.TaggedAt time.Time`(`repo.go:96`)이 있으나 그건 tag 정렬에 산술이 필요해서다.
여기는 필요 없다.

### C-3. `internal/git/repo_parse.go` — graph log 파서

포맷 문자열(`:186`)에 `%cI`를 **맨 끝에** 붙인다.

```
전: --format=%x00%H%x1f%P%x1f%ar%x1f%an%x1f%D%x1f%s
후: --format=%x00%H%x1f%P%x1f%ar%x1f%an%x1f%D%x1f%s%x1f%cI
```

**끝에 붙이는 것이 계약이다.** 파서(`:132-145`)가 위치 인덱스를 하드코딩한다 —
`parts[2]`=age, `parts[3]`=author, `parts[4]`=decorations, `parts[5]`=subject.
중간에 끼우면 필드가 서로 밀려 subject가 사라진다.

세 값을 **함께** 올린다. 하나라도 빠뜨리면 `len(parts) < N`이 걸려 `continue`로
**모든 커밋이 조용히 스킵되고 오류 메시지가 없다.**

| 위치 | 전 | 후 |
|---|---|---|
| `:186` 포맷 | 6 필드 | 7 필드 (`%cI` 추가) |
| `:132` `SplitN` | `6` | `7` |
| `:133` 검사 | `len(parts) < 6` | `len(parts) < 7` |
| `:145` 이후 | — | `entry.CommitDate = strings.TrimSpace(parts[6])` |

`--date` 인자를 추가하지 않는다. `%cI`는 플래그 없이 원본 오프셋을 준다 (실측 확인).

### C-4. `internal/graph` — Node 재조립

`internal/graph/graph.go`의 `Commit`(:19)과 `Node`(:35)에 `CommitDate string`을 더하고,
**Node를 필드 단위로 다시 만드는 두 곳을 반드시 함께 올린다.**

- `Nodes`(:72-84)
- `rowsFromGraph`(:174-180)

여기를 빠뜨리면 구조체에 필드가 있어도 값이 **zero value로 조용히 떨어진다.**
T2의 Details 경로가 이 rows에 의존한다(`navigation_graph.go:102-133`).

추가로 필드를 복사하는 곳: `internal/app/startup_projection.go:28`,
`navigation_graph.go:128`, `cherry_pick.go:31`.

### C-5. `internal/git/repo_exec.go` — Inspector 메타데이터

```
전: "show", "-s", "--format=%H%x00%P%x00%an%x00%ae%x00%s"          + SplitN(..., 5), len != 5
후: "show", "-s", "--format=%H%x00%P%x00%an%x00%ae%x00%s%x00%aI%x00%cI" + SplitN(..., 7), len != 7
```

`%aI`와 `%cI`를 **끝에** 붙이고 개수를 5→7로 올린다. 포맷만 바꾸면 `:167`이 즉시
`invalid commit metadata`를 반환해 Inspector 전체가 죽는다. C-3와 달리 이쪽은
오류를 내므로 발견은 빠르다.

`parts[5]` = AuthorDate, `parts[6]` = CommitDate.

### C-6. `internal/commitinspector/contract.go` — `CommitSnapshot`

```go
type CommitSnapshot struct {
    FullHash    string
    Subject     string
    AuthorName  string
    AuthorEmail string
    AuthorDate  string // NEW. strict ISO 8601, "" when unknown.
    CommitDate  string // NEW. strict ISO 8601, "" when unknown.
    MessageBody string
    Parent      string
    IsRoot      bool
    Files       []ChangedFile
}
```

`internal/adapter/out/commitinspector/reader.go:86`의 수동 매핑에 두 필드를 더한다.
이 파일은 필드를 하나씩 옮기므로 빠뜨리면 조용히 `""`가 된다.

### C-7. `internal/app` — range 상태

`navigationState`(`model.go:114-122`)에 더한다.

**하나의 구조체를 포인터로 들고, nil이 "anchor 없음"이다.**

```go
// graphRange is nil when there is no anchor. Non-nil means Anchor is set.
type graphRange struct {
    Anchor  string          // never "". Never a synthetic hash.
    Kind    graphRangeKind  // never rangeOff; the nil pointer IS "off".
    To      string          // endpoint the path runs toward, for the summary line.
    Members map[string]bool // nil unless Kind is rangeForward or rangeBackward.
    Count   int
    Epoch   uint64          // repositoryEpoch when the anchor was set.
    Err     string          // "" unless Kind == rangeUnavailable.
}
```

`navigationState`(`model.go:114-122`)에는 필드 **하나만** 더한다.

```go
graphRange *graphRange // nil = no anchor
```

```go
type graphRangeKind int

const (
    rangeAnchorOnly graphRangeKind = iota // anchor set, second commit not chosen yet
    rangeSame                             // anchor == cursor
    rangeForward                          // path anchor -> cursor
    rangeBackward                         // path cursor -> anchor
    rangeDiverged                         // both directions empty
    rangeUnavailable                      // query failed
)
```

**정정 (eng-review E-4).** 초안은 `graphRangeAnchor string`과 `graphRangeState`의
`rangeOff`를 **둘 다** 들고 있었다. 같은 사실("anchor가 있는가")을 두 곳에 저장하므로
서로 어긋날 수 있다 — `graphRangeAnchor == ""` 인데 `graphRangeState == rangeForward`
인 상태가 타입상 표현 가능하고, 그 상태에서 `AncestryPath`가 빈 ref로 호출되면
C-1의 가드가 오류를 내지만 사용자에게는 이유 없는 `unavailable`로 보인다.

포인터 nil을 유일한 "off" 표현으로 만들면 그 조합이 **타입상 표현 불가능**해진다.
`nil` 체크 한 번이 anchor 유무와 상태 유무를 동시에 답한다. 상태는 7개에서 6개로
줄지만 표현력은 같다.

**6개 상태가 계약이다.** 4개로 줄이면 `Same`이 `Diverged`로 읽혀 도구가 거짓을
가르치고, 방향 두 개를 하나로 합치면 어느 쪽이 조상인지 알려줄 수 없다.

### C-8. `internal/app/view_projection.go` — 프로젝션

`GraphProjection`(:28-34)에 더한다.

```go
RangeMembers map[string]bool // read-only view of the model's map. May be nil.
RangeAnchor  string          // "" when the model's graphRange is nil.
```

`:84`의 **수동 조립 리터럴에 반드시 채운다.** 구조체에 필드를 더하는 것만으로는
화면에 아무 일도 일어나지 않는다. `Handshake: m.handshakeCommits`와 같은 줄이다.

**소유권 계약.** 프로젝션은 모델 맵을 **참조로** 받는다(`Handshake`와 동일).
따라서:

- 렌더러는 이 맵을 **읽기만** 한다. 쓰기는 계약 위반이다.
- 모델은 range가 바뀔 때 맵을 **통째로 교체**한다. 제자리 수정(`delete`/`m[k]=v`)을
  하지 않는다. 그래야 렌더 중에 부분 갱신된 맵이 관측되지 않는다.

### C-9. `internal/app` — 조회 명령과 오류 전달

`commands.go`에 `checkGraphActionTarget`(:536-548)과 **같은 관용구**로 만든다.

```go
type graphRangeMsg struct {
    anchor, cursor string
    epoch          uint64
    kind           graphRangeKind
    members        map[string]bool
    count          int
    to             string
    err            error
}
```

`repository_read.go:42`의 `LocalBranchesKnown`/`Fresh`/`Error` 3상태 프로젝션 관용구를
쓰지 **않는다.** 그것은 refresh로 재생성되는 repository projection 상태용이고, 이건
사용자 키 입력으로 촉발되는 일회성 비동기 질의다. 저장소에 있는 두 관용구 중
message-`err` 쪽이 짝이다.

**epoch 가드.** `update.go`의 `graphRangeMsg` 핸들러는 첫 줄에서
`if msg.epoch != m.repositoryEpoch { return m, nil }` 로 결과를 폐기한다.
`update_lifecycle.go:42,62,77`이 같은 일을 한다.

**epoch 무효화.** `repositoryEpoch`가 오르는 지점에서 `m.graphRange = nil`로 만든다.
포인터 하나를 비우면 anchor·상태·멤버 집합이 함께 사라지므로 부분 초기화가 불가능하다. `commit_inspector_helpers.go:46`이
`commitInspectorStale = true`를 세우는 것과 같은 자리다. 조용히 유지하는 선택지는 없다.

### C-10. `internal/app/graph_render.go` — 하이라이트 렌더

두 렌더러 **모두** 고친다. `renderGraphLineWithSearch`(:55)와
`renderRawGraphLineWithSearch`(:100). raw가 실사용 경로이므로 non-raw만 고치면
테스트는 초록이고 화면은 안 바뀐다.

행별 신호는 `view_graph.go:52-58`에서 프로젝션 맵에서 꺼내 넘긴다. 다만 **스칼라를
더 붙이지 않고 구조체 하나로 묶는다.**

**정정 (eng-review E-2).** 초안은 "기존 `stashCount`/`isHandshake`와 같은 형태로
스칼라 인자를 추가"라고 적었다. 지금 호출부가 이렇다 (`view_graph.go:57`).

```go
lineStr := renderGraphLineWithSearch(rows[i], graphActive && i == p.Cursor, graphActive, p.LaneCursor, p.LocalBranchInventory, graphColWidth, width, isHandshake, stashCount, p.SearchQuery)
```

**이미 위치 인자가 10개다.** `graphActive`가 두 번 나오고, `graphColWidth, width`가
연속된 `int`이고, `isHandshake, stashCount`가 연속된 `bool, int`다. 여기에
`isRangeMember bool`과 `isRangeAnchor bool`을 더하면 **연속된 같은 타입 인자가 셋**이
되어 호출부에서 두 개를 뒤바꿔도 컴파일이 통과한다. anchor와 member는 시각적으로
비슷한 마커이므로 테스트가 우선순위를 정확히 단언하지 않으면 뒤바뀐 것도 통과한다.

"explicit over clever" 선호와 어긋나므로 대신 행별 신호를 묶는다.

```go
// graphRowMarks carries the per-row signals the row renderers need. It exists so
// the render call does not grow a third and fourth same-typed positional argument
// that a caller can silently transpose.
type graphRowMarks struct {
    Handshake   bool
    StashCount  int
    RangeMember bool
    RangeAnchor bool
}
```

두 렌더러의 `isHandshake bool, stashCount int` 두 인자를 `marks graphRowMarks` 하나로
**교체**한다. 인자 개수가 10개에서 9개로 줄고, 새 신호는 이름으로 지정되므로 전치가
불가능해진다. 이건 새 추상화가 아니라 같은 타입 인자 4개를 이름 있는 필드로 접는
것이다 — 최소 diff로 전치를 막는 방법이다.

호출부는 이렇게 된다.

```go
marks := graphRowMarks{
    Handshake:   rows[i].Commit.Hash != "" && p.Handshake[rows[i].Commit.Hash],
    StashCount:  p.StashCounts[rows[i].Commit.Hash],
    RangeMember: p.RangeMembers[rows[i].Commit.Hash],
    RangeAnchor: rows[i].Commit.Hash != "" && rows[i].Commit.Hash == p.RangeAnchor,
}
```

`renderGraphLine`/`renderRawGraphLine`(`graph_render.go:26,100`)의 얇은 래퍼 두 개도
같이 고친다.

**마커 우선순위 (높은 것이 이긴다).** 한 행에 여러 신호가 겹치므로 순서를 못 박는다.

| 순위 | 신호 | 표현 |
|---|---|---|
| 1 | 검색 포커스 | `searchFocusMark` (reverse+bold) |
| 2 | 커서 (선택 행) | 기존 거터 마커 (`df7ed32`) |
| 3 | **range anchor** | 거터에 `▏` + `searchMatchMark` |
| 4 | **range 멤버** | `searchMatchMark` (underline+bold) |
| 5 | HEAD | `headMark` |
| 6 | stash/tag | state 컬럼 `S`/`T`/`S·T` |

3과 4가 신규다. 새 색을 만들지 않고 `theme.go`의 기존 토큰을 재사용한다 —
`highlighting-color-map.md` Policy가 ANSI 0-15만 허용하고, task 6.5가 미사용 토큰
정리를 대기 중이므로 토큰을 늘리지 않는다.

**NO_COLOR 계약.** range 멤버는 `underline`이 색이 아니라 terminal attribute이므로
`NO_COLOR=1`에서도 남는다. anchor는 거터 글리프 `▏`를 쓰므로 색과 무관하다.
색 전용 표현은 계약 위반이다 — `highlighting-color-map.md` Policy와 task 6.3이 요구한다.

### C-11. `internal/app/view_detail.go` — Details 두 행

graph 섹션(`renderContextInfoLines`, `:21-38`)에 두 행을 넣는다. `view_detail.go:110`의
key/value 헬퍼를 쓴다.

**삽입 위치가 계약이다 (eng-review E-6).** 초안은 "`focus` 다음"이라고만 적었는데
`:29`에서 `focus:` 바로 다음이 이미 `focusParentLines(focus, width)`다. 어디에
끼우는지 정하지 않으면 구현자가 임의로 정한다. 순서는 이렇다.

```
focus:  4d8fcbcc        <- 기존 (:28)
date:   2026-09-08 23:15  <- 신규
range:  7 commits (...)   <- 신규, graphRange != nil 일 때만
<focusParentLines>      <- 기존 (:29)
branches: ...           <- 기존
stashes: ...            <- 기존
tags: ...               <- 기존
```

근거: `date:`는 focus 커밋의 identity에 속하므로 `focus:` 바로 아래가 맞다.
`range:`는 지금 진행 중인 상호작용이므로 낮은 높이에서 접히지 않게 위에 둔다.

**절단 걱정은 없다.** Inspector와 달리 Details는
`renderContextViewport(p.InfoLines, height, p.Scroll, width)`(`:17`)를 통과하는
**스크롤 뷰포트**다. 행이 늘면 `contextScroll`로 스크롤되며 조용히 잘리지 않는다.
대가는 낮은 높이에서 기존 `tags:`가 접히는 것이고, `ctrl+u/d`로 볼 수 있다.

```
focus: 4d8fcbcc
date:  2026-09-08 23:15
range: 7 commits (4d8fcbcc -> b2fd3168)
```

`date:` 렌더 규칙 — 입력은 C-2의 strict ISO 문자열이다.

| 조건 | 출력 |
|---|---|
| 정상 | `2026-09-08 23:15` (앞 16자) |
| 폭 부족 | `2026-09-08` (앞 10자) |
| `""` 또는 파싱 불가 | `-` |

`range:` 렌더 규칙 — C-7의 nil + 6상태를 그대로 노출한다.

| 상태 | 출력 |
|---|---|
| `graphRange == nil` | 행 자체를 그리지 않는다 |
| `rangeAnchorOnly` | `range: anchor set (4d8fcbcc)` |
| `rangeSame` | `range: same commit` |
| `rangeForward` | `range: 7 commits (4d8fcbcc -> b2fd3168)` |
| `rangeBackward` | `range: 7 commits (b2fd3168 -> 4d8fcbcc)` |
| `rangeDiverged` | `range: diverged (no ancestry path)` |
| `rangeUnavailable` | `range: unavailable` |

**방향 표시가 필수다.** 개수만 보여주면 사용자가 어느 쪽이 조상인지 알 수 없고,
그것을 알려주는 것이 이 기능의 목적이다.

`n commits`의 `n`은 `len(members)`이며 **C-1의 끝점 규칙에 따라 anchor를 제외하고
목표 커밋을 포함한다.** 위 예시의 7은 `4d8fcbcc`를 세지 않고 `b2fd3168`을 센다.

tags 섹션의 `age:`는 건드리지 않는다. `date:`와 `age:` 이원화는 task 6.6의 어휘
통일 범위다. **세 번째 어휘를 만들지 않는 것까지만** 이 spec이 지킨다.

### C-12. `internal/app/commit_inspector_screen.go` — 헤더

`renderCommitInspectorScreen`만 고친다. `commit_inspector.go:122`
`renderCommitInspectorPopup`은 프로덕션 호출자가 0개인 죽은 렌더러이므로 **손대지
않는다.**

헤더는 **5행 기본, author/committer 날짜가 다를 때만 6행**이다.

```
COMMIT 4d8fcbcc16d09a9d81...          (기존)
message: <subject>                     (기존)
author: hrllk <heykia3@protonmail.com> (기존, FROM 꼬리 제거)
date: 2026-09-08T23:15:16+09:00        (신규)
committed: 2026-09-08T23:20:01+09:00   (신규, AuthorDate != CommitDate 일 때만)
path: <selected file>                  (기존)
```

`FROM <parent>` 꼬리를 author 행에서 **떼어낸다**(`screen_go:56` `screenAuthorLine`).
task 6.8이 이미 "대응하는 TO가 없고 author와 이메일이 있는 행에 얹혀서 가장 먼저
잘린다"고 지목했다. parent를 헤더에 남길지는 task 6.8 범위이므로 이 spec은
**꼬리를 떼는 것까지만** 한다. `IsRoot`의 `ROOT COMMIT` 표기는 유지한다.

`date:`/`committed:` 렌더 규칙 — Inspector는 확정하는 면이므로 오프셋을 포함한
전체 형식을 쓴다. 폭이 부족하면 `2026-09-08 23:15`로 줄이고, 더 부족하면
`2026-09-08`로 줄인다. `""`면 행을 그리지 않는다.

**높이 계약이 4행에서 5~6행으로 바뀌고, 그 숫자가 코드에 하드코딩되어 있다.**

**정정 (eng-review E-3, P1).** 초안은 "테스트 매트릭스를 6행 기준으로 다시
통과시켜야 한다"까지만 적었다. 그것만으로는 구현자가 이 함수를 찾지 못한다.

`internal/app/commit_inspector_screen.go:105-107`:

```go
func inspectorBodyRows(height int) int {
	return max(max(height-2, 1)-7, 1)
}
```

`7`이 chrome 행 수를 하드코딩한다 — header 4행 + separator 1행(`:30`) + 패딩 1행 +
footer 1행. 헤더가 5~6행이 되면 이 함수는 **여전히 7을 뺀다.** 그러면
`screenBody`(`:31`)가 실제로 들어갈 수 있는 것보다 많은 행을 만들고,
`:41-43`이 뒤를 조용히 자른다.

```go
	for len(lines) < contentHeight-1 {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight-1 {
		lines = lines[:contentHeight-1]
	}
```

높이 12에서 `inspectorBodyRows(12)` = `max(10-7,1)` = **3**. 헤더를 2행 늘리면 실제
여유는 1행인데 함수는 3을 보고하므로 **diff 2줄이 오류도 표시도 없이 사라진다.**
`lines[:contentHeight-1]`은 슬라이스 자르기이므로 아무 신호가 없다.

따라서 C-12는 세 가지를 함께 한다.

1. **chrome 행 수를 헤더 실제 행 수에서 파생시킨다.** `7`을 상수로 두지 말고
   `len(headerLines) + 3`(separator + 패딩 + footer)으로 계산한다. 헤더가 조건부로
   5행/6행이므로 상수는 이제 정답이 될 수 없다.
2. `screenBody`에 넘기는 `inspectorBodyRows(height)` 호출부(`:31`)를 같이 고친다.
3. **높이 12에서 body 행 수를 단언하는 테스트를 5행 헤더와 6행 헤더 각각 넣는다.**
   기존 매트릭스는 "렌더가 터지지 않는다"만 보므로 조용한 절단을 잡지 못한다.

이 세 개를 안 하면 증상은 "낮은 높이에서 diff 마지막 줄이 안 보인다"이고, 원인은
`-7`이며, 그 사이에 아무 연결 고리가 없다.

### C-13. `internal/app` — 키

`v`를 graph 섹션 액션으로 바인딩한다. `key_handling_browse.go`의 graph 섹션 분기에
넣는다.

| 키 | 조건 | 동작 |
|---|---|---|
| `v` | graph 섹션, anchor 없음 | anchor = 커서 해시. 합성 해시면 무시 |
| `v` | graph 섹션, anchor 있음, 커서 != anchor | 조회 실행 |
| `v` | graph 섹션, anchor 있음, 커서 == anchor | `rangeSame` (조회 없음) |
| `esc` | `graphRange != nil` | `graphRange = nil` |

`hidden_hotkeys.go`의 **graph 섹션(:231-249)에만** 등록한다. 메인 footer에 넣지
않는다 — `docs/decisions.md:8-9`가 "Keep only Global core navigation keys in the main
footer"로 못 박았고, footer는 `globalHotkeyItems`(`hidden_hotkeys.go:33-50`)만 소스로
쓴다.

**키 구현과 `?` 등록은 같은 커밋이어야 한다.** task 8이 "동작하는 전역 키 10개가
푸터에도 오버레이에도 없다"를 미결로 들고 있으므로, 문서화 없는 키를 11개로 늘리지
않는다.

**`esc` 순서 (E-5 정정).** 초안은 `key_handling.go:66`을 근거로 들었으나 그 줄은
`handleOperationResultKey` 안이므로 **operation-result 모드**이고 browse 모드가 아니다.
정확한 구조는 이렇다.

- `key_handling.go:57`이 `state.ModeBrowse`에서만 `handleBrowseKey`로 보낸다.
- `esc`를 자체 처리하는 곳이 이미 여럿이다 — `confirm.go:53,65,75,85,94`,
  `commit_inspector.go:68`, `hidden_hotkeys.go:55`, `key_handling_cherry_pick.go:28`,
  `key_handling_blocked.go:12`.
- `independentOverlayOpen()`(`key_handling.go:77`)과 `overlayOpen()`(`:82`)이 게이트다.

따라서 range 해제는 **browse 모드의 오버레이 게이트를 통과한 뒤**에 놓는다. 오버레이나
confirm이 열려 있으면 그쪽이 먼저 `esc`를 소비하므로 task 1.23의 Esc-first close 계약이
유지된다. 결론은 초안과 같고 근거만 정확해졌다.

---

## Acceptance Criteria

1. `AncestryPath(ctx, "", "abc")`와 `AncestryPath(ctx, "abc", "")`가 오류를 반환한다.
2. `AncestryPath(ctx, A, B)`가 3커밋 구간에서 3개를 반환하고 `result[0] == B`이며
   A를 포함하지 않는다.
3. `AncestryPath(ctx, B, B)`가 `nil, nil`을 반환한다.
4. 역방향 `AncestryPath(ctx, B, A)`(B가 A의 자손)가 `nil, nil`을 반환한다.
5. 합성 해시 `VIRTUAL_CONFLICT_HASH`로 anchor를 세우려 하면 anchor가 세워지지 않고
   git 서브프로세스가 실행되지 않는다.
6. graph log 파서가 7필드를 읽고, 필드 수를 올린 뒤에도 반환 커밋 수가 0이 아니다.
7. `CommitDate` 값이 `graph.Nodes`와 `rowsFromGraph`를 통과한 뒤에도 남아 있다.
8. `CommitDate`/`AuthorDate` 값이 `CommitSnapshot`까지 도달한다 (adapter 계층 단언).
9. graph 섹션에서 커서를 옮기면 Details `date:` 행이 따라 바뀐다.
10. Details `date:`가 폭 40/60/80에서 Details 박스를 넘치지 않는다.
11. `v`를 두 번 눌러 확정하면 경로 커밋이 하이라이트되고 Details에
    `range: n commits (from -> to)`가 방향과 함께 나온다.
12. 역방향으로 고르면 `rangeBackward`로 표시되고 **`diverged`가 아니다.**
13. 같은 커밋을 두 번 고르면 `same commit`이고 **`diverged`가 아니다.**
14. 진짜 분기한 두 커밋에서만 `diverged (no ancestry path)`가 나온다.
15. 조회 실패가 `unavailable`로 나오고 `diverged`와 구분된다.
16. `NO_COLOR=1`에서 range 멤버와 anchor가 식별된다 (underline attribute + 거터 글리프).
17. **raw 렌더 경로**(`row.Graph != ""`)에서 하이라이트가 나온다.
18. anchor를 세운 뒤 `repositoryEpoch`가 오르면 하이라이트가 사라진다.
19. epoch가 어긋난 `graphRangeMsg`가 상태를 바꾸지 않는다.
20. Inspector 헤더가 `date:`를 표시하고, author/committer가 갈린 커밋에서만
    `committed:`를 추가로 표시한다.
21. Inspector 헤더 높이 계약이 12/20/30에서 깨지지 않는다.
22. author 행에서 `FROM <parent>` 꼬리가 사라지고 `ROOT COMMIT` 표기는 남는다.
23. `?` 오버레이 graph 섹션에 `v`가 보이고 메인 footer에는 없다.
24. 모달이 열린 상태에서 `esc`가 모달을 먼저 닫는다 (range 해제보다 우선).
25. `renderCommitInspectorPopup`과 그 테스트가 이 변경으로 수정되지 않는다.
26. graph 행에 날짜 컬럼이 **추가되지 않는다** (task 6.4 범위).
27. `scripts/check` (test + vet + build + gofmt/diff-check) CLEAN.
28. `internal/architecture` 가드가 통과한다. **단 이 가드는 `internal/git`의
    `os/exec` 사용을 검사하지 않는다** (E-1 참조) — 이 기준은 `internal/commitinspector`
    변경(C-6)이 금지 import를 들이지 않았음만 보장한다.
29. **높이 12에서 `inspectorBodyRows`가 보고하는 행 수와 실제 렌더된 body 행 수가
    같다.** 5행 헤더와 6행 헤더 각각. (E-3 — 조용한 절단 방지)
30. **`graphRowMarks`가 두 렌더러의 `isHandshake`/`stashCount` 인자를 대체하고,
    호출부 위치 인자가 늘지 않는다.** (E-2 — 인자 전치 방지)
31. **`anchor == ""`이면서 range 상태가 활성인 조합이 타입상 표현 불가능하다.**
    `graphRange` 포인터가 nil이면 anchor가 없다. (E-4 — 이중 진실 방지)

## Testing Plan

| Layer | What | Count |
|---|---|---|
| Unit (`internal/git`) | `AncestryPath` 빈 ref 가드, 끝점 규칙, 순서, 동일 커밋, 역방향 | +5 |
| Unit (`internal/git`) | graph log 7필드 파싱, 필드 수 회귀 (커밋 수 != 0) | +2 |
| Unit (`internal/git`) | Inspector 7필드 파싱, `%aI`/`%cI` 값 | +2 |
| Unit (`internal/graph`) | `Nodes`/`rowsFromGraph` 통과 후 `CommitDate` 보존 | +2 |
| Unit (`internal/app`) | 6상태 판정 각각 + nil, 합성 해시 거부 | +8 |
| Unit (`internal/app`) | epoch 폐기, epoch 무효화 | +2 |
| Render (`internal/app`) | Details `date:`/`range:` nil + 6상태, 폭 40/60/80 | +6 |
| Render (`internal/app`) | 하이라이트 **raw 경로**, non-raw 경로, 마커 우선순위 | +4 |
| Render (`internal/app`) | `NO_COLOR=1` marker 존재 | +2 |
| Render (`internal/app`) | Inspector 헤더 5행/6행, 높이 12/20/30 | +4 |
| Render (`internal/app`) | **`inspectorBodyRows` 보고값 == 실제 body 행 수** (5행/6행 헤더, 높이 12) | +2 |
| Integration (`internal/adapter`) | 날짜가 `CommitSnapshot`까지 도달 | +1 |
| Contract | `?` 오버레이에 `v` 존재, footer에 부재 | +2 |
| Contract | **`graphRange == nil`이 유일한 off 표현임을 단언** (anchor 있는 nil 불가) | +1 |

**Prior learning (`view-never-called-in-tests`, 9/10, 2026-08-26).** 이 저장소의 테스트
64개 중 `model.View()`를 호출하는 것은 0개다. 위 Render 계층 테스트는 모델을 손으로
조립하는 기존 방식(`commit_inspector_screen_test.go:24`)을 따르되, **최소 하나는
Update를 통과한 상태로 렌더까지 간다.** 그러지 않으면 "Update가 바꾼 상태를 렌더러가
무시하는" 종류의 버그를 구조적으로 못 잡는다.

**flakiness.** 시각·난수·외부 서비스·순서 의존 없음. git fixture는
`clone --local` 사본을 쓴다 (prior learning `mutating-action-masks-startup-load-bug`:
read-only 세션에서 0 키입력으로 단언해야 하는 항목은 9번과 18번이다).

## Root Cause Analysis

**날짜가 없는 이유.** 사고가 아니다. `397b2ea`(task 1.8)가 정보 밀도를 위해 7칸
`date` 컬럼을 의도적으로 제거했고 그 자리를 title+author에 줬다. 제거된 것은 상대
시간이었고 절대 날짜는 애초에 없었다. Inspector는 애초에 날짜를 요청하지 않았다.

**지금 날짜 컬럼을 되살릴 수 없는 이유.** 폭이 없다. 80컬럼 이하에서 고정 컬럼만으로
이미 패널 폭을 초과한다(title 가용폭 -2/-14/-25). 이건 이 spec이 만든 문제가 아니라
`2d9c4c4` 이후의 현재 상태이며 task 6.2가 대기 중이다. 그래서 날짜는 Details로 가고
컬럼은 6.4로 간다.

**구간 기능이 없었던 이유.** graph state에 anchor 개념이 없다. 그리고 `--ancestry-path`
없이 "두 커밋 사이"를 구현하면 화면 행 범위가 되는데, graph가 `--all --topo-order`라
그건 git 상의 사실이 아니다.

## Effort Estimate

| 컴포넌트 | human | CC |
|---|---|---|
| C-1 `AncestryPath` + 가드 + 테스트 5개 | ~1.5h | ~10min |
| C-2~C-4 graph 날짜 (필드·파서·재조립·복사 6곳) | ~2h | ~15min |
| C-5~C-6 Inspector 날짜 데이터 4계층 | ~1.5h | ~10min |
| C-7~C-9 range 상태·조회·epoch | ~3h | ~25min |
| C-10 하이라이트 렌더 (두 경로 + `graphRowMarks` 교체 + 우선순위 + NO_COLOR) | ~3.5h | ~30min |
| C-11 Details 두 행 | ~1.5h | ~10min |
| C-12 Inspector 헤더 + `inspectorBodyRows` chrome 파생 (E-3) | ~2.5h | ~20min |
| C-13 키 + `?` 등록 + esc 순서 | ~1h | ~10min |
| 테스트 43개 | ~4.5h | ~35min |
| 합계 | **~22h** | **~2.8h** |

## Rollback Plan

각 C가 독립적으로 revert 가능하다. 마이그레이션·상태 저장·새 산출물이 없고 단일
바이너리 빌드에 변화가 없다. feature flag 불필요.

되돌리기 어려운 것 하나: **C-12가 Inspector 헤더 높이 계약을 4→6행으로 바꾼다.**
revert하면 테스트 매트릭스도 함께 되돌려야 한다. C-12를 별도 커밋으로 둔다.

## Files Reference

| File | Change |
|---|---|
| `internal/git/repo_exec.go` | C-1 `AncestryPath` 신규. C-5 Inspector 포맷 5→7필드 |
| `internal/git/repo.go:78` | C-2 `GraphCommit.CommitDate` |
| `internal/git/repo.go:110` | C-2 `CommitInspection.AuthorDate/CommitDate` |
| `internal/git/repo_parse.go:186` | C-3 `%cI`를 끝에 |
| `internal/git/repo_parse.go:132-133` | C-3 `SplitN` 6→7, `len < 6`→`< 7` |
| `internal/git/repo_parse.go:145+` | C-3 `entry.CommitDate` 대입 |
| `internal/graph/graph.go:19,35` | C-4 `Commit`/`Node` 필드 |
| `internal/graph/graph.go:72-84` | C-4 `Nodes` 재조립 |
| `internal/graph/graph.go:174-180` | C-4 `rowsFromGraph` 재조립 |
| `internal/commitinspector/contract.go:93` | C-6 `CommitSnapshot` 두 필드 |
| `internal/adapter/out/commitinspector/reader.go:86` | C-6 수동 매핑 |
| `internal/app/model.go:114-122` | C-7 `graphRange *graphRange` 한 필드 + `graphRangeKind` 6상태 |
| `internal/app/view_projection.go:28-34` | C-8 `RangeMembers`/`RangeAnchor` |
| `internal/app/view_projection.go:84` | C-8 수동 조립 리터럴 |
| `internal/app/commands.go` | C-9 `graphRangeMsg` + 조회 cmd |
| `internal/app/update.go` | C-9 epoch 가드 핸들러 |
| `internal/app/update_lifecycle.go` | C-9 epoch 무효화 |
| `internal/app/graph_render.go:55,100` | C-10 두 렌더러 — `isHandshake`/`stashCount`를 `graphRowMarks`로 교체 |
| `internal/app/graph_render.go:26,100` | C-10 얇은 래퍼 두 개 시그니처 |
| `internal/app/view_graph.go:52-58` | C-10 `graphRowMarks` 조립 |
| `internal/app/view_detail.go:27-38` | C-11 `date:`/`range:` |
| `internal/app/commit_inspector_screen.go:56` | C-12 헤더 + `FROM` 꼬리 제거 |
| `internal/app/commit_inspector_screen.go:105-107` | **C-12 `inspectorBodyRows`의 하드코딩 `-7`을 헤더 행 수에서 파생 (E-3)** |
| `internal/app/commit_inspector_screen.go:31` | C-12 `inspectorBodyRows` 호출부 |
| `internal/app/key_handling_browse.go` | C-13 `v` 바인딩 |
| `internal/app/hidden_hotkeys.go:231-249` | C-13 `?` 등록 |
| `internal/app/startup_projection.go:28` | C-4 필드 복사 |
| `internal/app/navigation_graph.go:128` | C-4 필드 복사 |
| `internal/app/cherry_pick.go:31` | C-4 필드 복사 |

## Out of Scope

- **graph 행 날짜 컬럼** — task 6.4. 지금은 폭이 음수다.
- **`A...B` + merge base 분기 시각화** — 마커 3종이 필요해 task 6.3 NO_COLOR 제약과
  어긋난다. C-10의 우선순위 표가 확장 가능하게 설계되어 나중에 3종을 넣을 수 있다.
- **구간 diff를 Inspector로 열기** — `CommitRequest`가 단일 커밋 계약이다.
- **구간 대상 액션** (cherry-pick range, rebase --onto) — task 7 게이트가 in-progress.
- **`date:` / `age:` 어휘 통일** — task 6.6.
- **Inspector 헤더에 `parent:` 행 신설** — task 6.8. 이 spec은 `FROM` 꼬리를 떼는
  것까지만.
- **죽은 `renderCommitInspectorPopup` 삭제** — `TODOS.md`.
- **tasks.json 스텁 중복 정리** — `TODOS.md`.
- **로컬 타임존 변환** — 원본 오프셋으로 확정.

## Related

- 계획: `docs/20260907-0001-graph-dates-and-commit-range-plan.md`
- Taskmaster: task 11 (11.1~11.9)
- 선행/충돌: task 6.2 (80컬럼 overflow), 6.3 (색 비의존 신호), 6.4 (컬럼 예산),
  6.6 (어휘 통일), 6.8 (Inspector 카피), 8 (키 문서화 미결)
- 되돌리는 결정: `397b2ea` / task 1.8 (date 컬럼 제거) — 부분적으로만, 6.4에서
- 기록된 결정: `docs/decisions.md` 2026-08-01 (title이 나머지 폭), 2026-07-10
  (컬럼 대신 marker), 2026-08-01 (footer는 Global 키만)

---

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 1 | CLEAR | mode: HOLD_SCOPE, 13 findings applied to the plan, 1 critical gap |
| Codex Review | `/codex review` | Independent 2nd opinion | 1 | CLEAR | 12 findings on the plan, 10 accepted, 2 rejected (stub-record misread) |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 1 | CLEAR | mode: HOLD_SCOPE, 6 findings, 0 critical gaps, 0 unresolved |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | not run — C-10's marker priority table now pins the order this would have decided |
| DX Review | `/plan-devex-review` | Developer experience gaps | 0 | — | not run |

**ENG FINDINGS (all applied):**

- **E-1 (P2, confidence 10/10)** `internal/architecture/guard_test.go:91-95` — C-1's requirement was justified by a guard that does not apply. `internal/git` has no `len(violations) != 0` assertion, and `internal/git/repo_exec.go:10` already imports `os/exec`. Rule kept, justification corrected to consistency with `r.git`.
- **E-2 (P1, confidence 10/10)** `internal/app/view_graph.go:57` — the render call already passes 10 positional arguments with `graphColWidth, width` and `isHandshake, stashCount` adjacent and same-typed. Adding two more bools makes a silent transposition compile. Replaced with a named `graphRowMarks` struct, which also drops the call to 9 arguments.
- **E-3 (P1, confidence 10/10)** `internal/app/commit_inspector_screen.go:105-107` — `inspectorBodyRows` hard-codes `-7` chrome rows for a 4-line header. Adding header lines makes `screenBody` produce more rows than fit, and `:41-43` `lines[:contentHeight-1]` then truncates the diff pane with no signal. At height 12 the function reports 3 body rows while only 1 fits. C-12 now requires deriving the chrome count from the actual header length plus a test asserting reported == rendered.
- **E-4 (P2, confidence 9/10)** C-7 stored anchor presence twice (`graphRangeAnchor == ""` and `graphRangeState == rangeOff`), making `anchor == "" && state == rangeForward` representable. Collapsed into one `*graphRange` pointer where nil is the only "off", so the bad combination is unrepresentable. 7 states became 6 with no loss of expressiveness.
- **E-5 (P3, confidence 10/10)** `internal/app/key_handling.go:66` — the cited line is inside `handleOperationResultKey` (operation-result mode), not browse mode. Conclusion held; the citation now names `key_handling.go:57`, the overlay gates at `:77`/`:82`, and the nine handlers that consume `esc` first.
- **E-6 (P2, confidence 10/10)** `internal/app/view_detail.go:29` — "insert after `focus`" was ambiguous because `focusParentLines` already occupies that position. Insertion order is now pinned. Verified Details uses `renderContextViewport` (`:17`), a scrolling viewport, so unlike the Inspector it cannot silently truncate.

**CROSS-MODEL:** The codex pass ran against the plan, not this spec, so there is no per-finding overlap to report here. Its highest-value plan finding (the stale 80-column width premise) is carried into this spec's Current State table as a measured row citing `shell_width_test.go:113-118`. Its independent agreement with Claude on the empty-`ancestry-path` misclassification is what made P4's direction detection a settled decision rather than a proposal, and this spec encodes it as the 6-state contract.

**SELF-CORRECTION:** The E-4 edit initially left seven stale references to the replaced field names elsewhere in the document — the same "one fact in two places" defect class E-4 itself fixed. Caught on a follow-up grep and repaired before the report was written.

**VERDICT:** CEO + ENG CLEARED — ready to implement. Scope held at HOLD_SCOPE through both reviews; no expansions accepted. Test count rose 38 → 43 and effort ~20h → ~22h (human) / ~2.5h → ~2.8h (CC) from the review-added work. Recommend `/plan-design-review` before C-10 only if the marker priority table proves insufficient in practice; it now pins what that review would have decided.

NO UNRESOLVED DECISIONS
