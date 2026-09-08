# Graph 날짜 노출과 두 커밋 구간 경로 하이라이팅 — 구현 계획

상태: REVIEWED — 확정 2건, 잠정 5건, **미결 4건 (다른 디바이스에서 이어서 결정 필요)**
출처: `/office-hours` 세션 2026-09-07 (Builder 모드, open source)
리뷰: `/plan-ceo-review` 2026-09-08 (HOLD SCOPE) — 자기모순 1건, CRITICAL GAP 1건 포함 8건 반영
기준 커밋: `8ba3d58`
설계 문서: `~/.gstack/projects/hrllk-graphkeeper/hrk-main-design-20260907-224500.md`

이 문서는 세 요청을 다룬다.

1. graph에 날짜(yymmdd) 노출
2. Commit Inspector 날짜 상세 표시
3. graph에서 두 커밋을 선택해 그 사이 구간을 하이라이팅

---

## TOC

- [요청 문구와 코드 사실의 차이](#요청-문구와-코드-사실의-차이)
- [확정된 결정](#확정된-결정)
- [잠정 결정 (되돌릴 수 있는 기본값)](#잠정-결정-되돌릴-수-있는-기본값)
- [미결 — 내일 이어서](#미결--내일-이어서)
- [작업 항목](#작업-항목)
- [테스트 전략](#테스트-전략)
- [기존 task와의 관계](#기존-task와의-관계)
- [리스크](#리스크)

---

## 요청 문구와 코드 사실의 차이

계획을 세우기 전에 확인한 사실이다. 요청 문구를 그대로 구현하면 안 되는 지점이
두 곳 있다.

### R1. "graph yymmdd **복구**"는 복구가 아니다

복구할 대상이 요청 문구와 다르다.

- `397b2ea`(task 1.8 "Graph 정보 열 재배치")가 폭 7의 `date` 컬럼을 제거했다.
  그 커밋의 taskmaster details는 명시적으로 "date를 제거하며"라고 적혀 있고,
  status: done이다. 의도된 결정이었고 사고가 아니다.
- 제거된 컬럼의 **내용은 `compactWhenText(row.Commit.RelativeAge)`**였다. 즉
  `3d`, `2mo` 같은 **상대 시간**이다. 헤더 라벨만 `date`였다.
- 이 저장소의 추적된 히스토리에 **절대 yymmdd가 존재한 적이 없다.** graph log
  포맷(`internal/git/repo_parse.go:186`)은 `%ar`(상대)만 가져오며, 그 이전
  버전(`be862b0` 이전)은 날짜 필드를 아예 요청하지 않았다.

따라서 이 항목은 "복구"가 아니라 서로 다른 두 작업이다.

- (a) 새 데이터 소스 추가: graph log 포맷에 절대 날짜 필드를 넣는다.
- (b) 삭제된 UI 결정의 되돌림: 제거된 컬럼 자리를 다시 만든다.

(a)는 충돌이 없다. (b)는 아래 R2와 정면으로 충돌한다.

### R2. 날짜 컬럼은 pending task 6.4와 정반대 방향이다

**정정 (2026-09-08 CEO review, 코덱스 아웃사이드 보이스 교차확인).** 이 문서의 초안은
80컬럼에서 Graph 패널 내부폭을 56, title 가용폭을 11자로 적었다. **틀렸다.** 56은
2026-08-28 감사 리포트 시점의 값이고, 그 뒤 `2d9c4c4`/`bab11fb`가 분할을 바꿨다.
현재 측정값은 코드에 테스트로 박혀 있다.

`shell_width_test.go:113-118` `TestGraphRailSplitPerWidth`가 고정한 실제 분할:

| 터미널 | graph 패널 | 렌더러가 받는 content (`view_shell.go:56` `graphWidth-4`) |
|---|---|---|
| 40 | 20 | 16 |
| 60 | 31 | 27 |
| 80 | **43** | **39** |
| 140 | 77 | 73 |

같은 테스트 위 주석이 직접 말한다: "width 80 moves from 56/20 to 54/22".

이제 `graph_render_format.go:24` `graphRowFixedWidth`를 그 content 폭으로 계산한다.

```
graphCommitWidth(6) + 1 + graphBranchFieldWidth(14) + 1
  + graphStatusWidth(5) + 1 + graphTopologyWidth(content) + 1
```

| 터미널 | content | `graphTopologyWidth` | 고정폭 합계 | **title 가용폭** |
|---|---|---|---|---|
| 40 | 16 | 12 | 41 | **-25** |
| 60 | 27 | 12 | 41 | **-14** |
| 80 | 39 | 12 | 41 | **-2** |
| 140 | 73 | 14 | 43 | +30 |

**즉 80컬럼 이하에서는 고정 컬럼만으로 이미 Graph 패널 폭을 초과한다.** title에
할당할 폭이 음수이고, 행은 `fitVisibleWidth`가 잘라낸다. 이것이 감사 리포트 D-004의
"40컬럼에서 커밋 메시지가 단 하나도 렌더되지 않아 화면이 해시·`-`·`*`만 남는다"의
정확한 원인이다.

`yymmdd`(6칸) + 구분자(1칸)를 넣으면 80컬럼에서 초과폭이 **-2에서 -9로** 커진다.
초안이 적은 "title 4자"는 존재하지 않는다. 4자를 줄 폭 자체가 없다.

이 정정은 D5를 뒤집지 않는다. **더 강하게 만든다** — 날짜 컬럼은 "빠듯하다"가 아니라
80컬럼 이하에서 **불가능**하고, task 6.4는 폭 재배분이 아니라 **현재 오버플로 수정**이
선행 조건이다. 6.2("80컬럼 미만 레이아웃 overflow 수정")가 pending인 이유가 이것이다.

이것이 문제인 이유는 세 곳에 이미 기록되어 있다.

- pending task **6.4**의 헌장 자체가 "빈 컬럼이 22칸을 점유하는 동안 커밋 제목이
  11자로 잘리는 정보 배분을 바로잡는다"이다. 날짜 컬럼 추가는 이 헌장을 악화시킨다.
- `docs/reddit-feedback-task-list.md`에 "전체 브랜치 이름과 subject를 20자 정도로
  강제 truncation하면 정보가 사라진다"가 공개 피드백으로 남아 있다.
- 6.4의 완료 기준이 "100컬럼에서 title 가용폭이 80컬럼 대비 증가"다. 날짜 컬럼을
  먼저 넣으면 이 기준을 만족시키기가 더 어려워진다.

### R2-bis. 날짜 컬럼은 `docs/decisions.md`에 기록된 결정과도 어긋난다

초안이 놓친 것이다. `docs/decisions.md` 2026-08-01 "Graph is the primary full-height
surface and Details heads the right rail"에 이렇게 적혀 있다.

> Give the graph title/subject **the remaining width** after the five-character hash,
> branches, state, narrower `graph` topology, and restored author metadata.

title이 **나머지 전부**를 받는다는 것이 기록된 결정이고, 그 열거 목록에 날짜는 없다.
같은 파일 2026-07-10에는 더 일반적인 원칙이 있다.

> Keep tag presence in the Graph row as **a compact marker, not a separate column**.

즉 이 저장소는 "graph 행에는 컬럼을 늘리지 말고 마커로 해결한다"를 이미 결정해 두었다.

이 사실의 효과는 두 방향이다.

- **D5 1단(Details 패널)을 강화한다.** 컬럼을 늘리지 않고 기존 상세 면을 쓰는 것이
  기록된 원칙과 같은 방향이다.
- **D5 2단(6.4의 컬럼 재도입)을 약화한다.** 이건 pending task와의 충돌이 아니라
  **기록된 결정의 반전**이다. 따라서 T9는 코드만 바꾸는 것으로 끝나지 않는다.

**T9의 필수 조건:** `docs/decisions.md`에 반전 항목을 새로 적는다. 적지 않으면 문서와
코드가 어긋난 상태로 남고, 이 저장소는 이미 그 실패를 경험했다 — `TODOS.md`의
"Success Criteria가 T7과 모순된다" 항목이 같은 종류다.

### R3. Commit Inspector에는 날짜 필드가 아예 없다

`internal/commitinspector/contract.go:93` `CommitSnapshot`에 날짜 필드가 없다.
데이터가 없어서 표시가 없는 것이며, 표시만 고치면 되는 문제가 아니다. 6개 계층을
관통해야 한다.

| # | 파일 | 지금 상태 |
|---|---|---|
| 1 | `internal/git/repo.go:110` `CommitInspection` | `Hash, Subject, Author, Message, Parent` — 날짜 없음 |
| 2 | `internal/git/repo_exec.go:162` | `show -s --format=%H%x00%P%x00%an%x00%ae%x00%s` + `SplitN(meta, "\x00", 5)` |
| 3 | `internal/commitinspector/contract.go:93` `CommitSnapshot` | 날짜 없음 |
| 4 | `internal/adapter/out/commitinspector/reader.go:86` | 필드 단위 수동 매핑 |
| 5 | `internal/app/commit_inspector_screen.go:56` `screenAuthorLine` | author 행에 `FROM <parent>`를 덧붙임 |
| 6 | 테스트 | `commit_inspector_screen_test.go` 등 |

**정정 (2026-09-08 CEO review).** 이 문서의 초안은 `internal/app/commit_inspector.go:122`
`renderCommitInspectorPopup`을 "같은 렌더를 하는 두 번째 경로"로 적었다. 틀렸다.
호출자를 전부 열거하면 다음이다.

```
internal/app/commit_inspector_test.go:74   m.renderCommitInspectorPopup(80, 20)
internal/app/commit_inspector_test.go:98   m.renderCommitInspectorPopup(60, 20)
```

**프로덕션 호출자가 0개다.** `view_shell.go:49`는 `renderCommitInspectorScreen`만
호출한다. 즉 이것은 살아 있는 두 번째 경로가 아니라 **테스트가 고정해 둔 죽은
렌더러**다. 실제 위험은 원래 적은 것과 다르다.

- T5가 살아 있는 경로에만 날짜를 넣으면 죽은 경로의 테스트는 계속 통과하므로
  아무도 불일치를 눈치채지 못한다. 이건 문제가 아니다.
- 문제는 그 반대다. "일관성을 위해 두 경로를 모두 고쳐라"는 지시를 따르면
  **죽은 코드를 유지보수하게 된다.**

따라서 T5는 `renderCommitInspectorScreen`만 고친다. 죽은 렌더러는 별도로 처리한다
(`TODOS.md` 참조).

`SplitN(..., 5)`와 `len(parts) != 5` 검사가 있으므로 포맷 문자열만 바꾸면 즉시
`invalid commit metadata`로 실패한다. 두 값을 함께 바꿔야 한다.

### R4. 구간 하이라이팅은 그래프 행 렌더러가 두 개다

`internal/app/graph_render.go`에 `renderGraphLineWithSearch`(:55)와
`renderRawGraphLineWithSearch`(:100)가 있고, 전자가 `row.Graph != ""`일 때 후자로
위임한다. 실제 저장소에서는 `--graph` 출력이 항상 있으므로 **raw 경로가 실사용
경로**다. 두 함수 모두 손대야 하며, 한쪽만 고치면 테스트는 통과하고 실행 화면은
안 바뀌는 형태의 버그가 된다.

이미 존재하는 재사용 지점:

- `internal/app/view_projection.go:30` `GraphProjection.Handshake map[string]bool` —
  해시 집합을 렌더러까지 나르는 관습이 이미 있다. 구간 멤버십도 같은 모양이면 된다.
- `internal/git/repo_exec.go:829` `Divergence(ctx, left, right)` —
  `rev-list --left-right --count left...right`. rev-list 호출 관습이 이미 있다.
- `theme.go` `searchMatchMark`(underline+bold), `searchFocusMark`(reverse+bold) —
  색에 의존하지 않는 하이라이트 토큰이 이미 정의되어 있다.

---

## 확정된 결정

### D4 (확정) — 구간의 정의는 `--ancestry-path A..B`

두 커밋 사이 구간은 **A에서 B로 실제로 이어지는 조상 경로 상의 커밋**으로 정의한다.
`git rev-list --ancestry-path A..B`.

근거:

- 요청 문구가 "두커밋 **길** 확인"이었다. 집합이 아니라 경로다. `--ancestry-path`가
  Git에서 정확히 그 대상이다.
- graph는 `--all --topo-order`(`repo_parse.go:177-186`)로 모든 브랜치를 한 화면에
  섞는다. topo-order는 자식이 부모보다 위에 온다는 것만 보장하고, **화면에서 붙어
  있는 두 행의 조상 관계는 보장하지 않는다.** 따라서 화면상 행 범위를 구간이라고
  부르면 도구가 사용자에게 거짓을 가르친다. 이 제품은 설계 문서
  (`hrk-develop-design-20260710-140155.md`)에 "부사수에게 형상관리를 교육"을 목적으로
  적어 두었으므로, 오해를 가르치는 것이 가장 나쁜 실패다.
- **결과가 비어 있는 것 자체가 정보다.** 조상 경로가 없으면 = 두 커밋이 분기했다
  = fast-forward 불가. 이 판정은 같은 설계 문서의 "graph를 이해하고 그 안에서 FF가
  가능한지 판별"과 동일한 질문이다. 즉 이 기능은 새 기능이 아니라 이미 적어둔 제품
  목적의 직접 구현이다.

기각한 대안:

- `A..B`(단방향 도달 가능 전집합): 브랜치가 여럿이면 100여 행이 통째로
  하이라이트되어 하이라이트가 정보를 잃는다.
- `A...B` + merge base(분기 전체): 마커가 3종(A쪽/B쪽/base) 필요하고, NO_COLOR에서
  거터 없이 세 종을 구분해야 해 pending task 6.3의 제약과 어긋난다. 다만 교육
  가치는 가장 높으므로 **후속 확장 후보로 남긴다** (아래 O4).
- 화면상 행 범위: git 상의 사실이 아니다. 기각.

### D5 (확정) — 날짜는 Details 패널에 먼저, 컬럼은 6.4에 편입

2단으로 나눈다.

- **지금:** `internal/app/view_detail.go`의 graph 섹션(:27-38)에 `date:` 행을 만든다.
  현재 이 섹션은 `focus` / `branches` / `stashes` / `tags`만 있고 날짜가 없다.
  같은 파일의 tags 섹션(:78)에는 이미 `age:`가 있으므로 **비대칭을 메우는 변경**이며,
  key/value 관습(`view_detail.go:110`)이 이미 있어 폭 예산 비용이 0이다.
- **6.4에서:** 컬럼 예산 재설계가 폭을 회수한 뒤, 그 예산 안에서 yymmdd 컬럼을
  재도입한다. 6.4의 완료 기준에 날짜 컬럼을 포함시킨다.

근거: R2의 숫자 때문에 지금 컬럼을 넣으면 6.4를 집는 사람이 즉시 되돌려야 하는
변경이 된다. 새 모순을 만들지 않는 유일한 순서다.

**이 결정의 대가를 명시한다:** 이번 작업에서 graph 행에는 날짜가 보이지 않는다.
행을 훑으며 시간 흐름을 읽는 것은 6.4 이후에 가능해진다. 커서를 움직여 Details를
보는 방식만 이번에 생긴다.

---

## 잠정 결정 (되돌릴 수 있는 기본값)

미결이 아니다. 방어 가능한 기본값을 이미 골라 두었으니 그대로 진행하고, 이견이
있으면 바꾼다. 되돌리는 비용이 낮은 것만 여기에 둔다.

### P1 — 날짜 형식은 `YYMMDD` 고정폭, Details/Inspector는 전체 형식

- graph 컬럼(6.4에서): `YYMMDD` 6칸 고정. 요청 문구 그대로.
- Details 패널: `YYYY-MM-DD HH:MM` + 상대 시간. 예: `date: 2026-09-07 22:35 (3h)`.
  폭이 부족하면 상대 시간을 먼저 버린다.
- Commit Inspector: 오프셋을 포함한 전체 형식. 예:
  `2026-09-07 22:35:42 +0900`.

근거: "날짜 상세히"라는 요청은 Inspector에 걸려 있다. graph는 훑는 면이고 Inspector는
확정하는 면이므로 상세도를 다르게 두는 것이 맞다.

### P2 — 타임존은 커밋에 기록된 원본 오프셋을 쓴다

`%ad`/`%cd`를 `--date=iso-strict`로 받아 커밋 작성자의 원래 오프셋을 그대로 보여준다.
로컬 타임존으로 변환하지 않는다.

근거: 다른 타임존의 협업자가 언제 커밋했는지가 리뷰에서 알고 싶은 값이다. 로컬로
변환하면 "이 사람 새벽 3시에 커밋했네"라는 정보가 사라진다. 되돌리는 비용은
포맷 인자 하나다.

### P3 — 구간 선택 키는 `v`, 해제는 `v` 또는 `esc`

지금 바인딩된 키를 전부 열거한 결과 사용 중인 키는 다음이다.

```
? / 1 2 3 4 a backspace c ctrl+c ctrl+d ctrl+u d D down enter esc f F g G h H
j k l left m n N o p P q r right s S shift+tab space t tab up x y
```

`v`는 비어 있고, vim의 visual mode와 의미가 정확히 같다. anchor를 세우고 커서를
옮기는 동작이 vim visual 선택과 동일하므로 학습 비용이 0에 가깝다.

되돌리는 비용: 키 문자 하나와 `hidden_hotkeys.go` 항목 하나.

### P4 — 방향을 자동 판정한다. "빈 결과 = 분기"를 그대로 쓰면 거짓을 가르친다

**이것이 CEO review가 찾은 가장 중요한 모순이다.**

D4를 고른 근거는 "결과가 비어 있는 것 자체가 정보다 — 조상 관계가 없다 = 분기했다 =
FF 불가"였다. 그런데 `--ancestry-path A..B`가 비어서 돌아오는 경우는 **셋**이고, 셋 중
분기는 하나뿐이다.

| 경우 | `--ancestry-path A..B` | 실제 의미 |
|---|---|---|
| A와 B가 진짜 분기했다 | 빈 결과 | 분기. FF 불가. |
| **사용자가 방향을 거꾸로 골랐다** (B가 A의 조상) | 빈 결과 | 경로가 있다. 방향만 반대. |
| **A와 B가 같은 커밋이다** | 빈 결과 | 같은 지점. 분기도 경로도 아니다. |

즉 초안의 성공 기준("경로가 없으면 `no ancestry path (diverged)`를 명시한다")을 그대로
구현하면, 사용자가 **최신 커밋을 먼저 찍고 오래된 커밋을 나중에 찍었을 때 "분기했다"고
표시한다.** 그 둘은 조상 관계가 멀쩡히 있는데도 그렇다.

D4를 고른 이유가 "화면상 행 범위는 사용자에게 거짓을 가르치므로 기각"이었으므로,
이 상태로 두면 **기각한 대안과 같은 죄를 D4가 짓는다.** 자기모순이다.

**해소:** 방향을 자동 판정한다.

```
anchor == cursor                      -> "same commit"
ancestry-path(anchor..cursor) 비었나?
  아니오                              -> "path: n commits (anchor -> cursor)"
  예   -> ancestry-path(cursor..anchor) 비었나?
           아니오                     -> "path: n commits (cursor -> anchor)"
           예                         -> "diverged (no ancestry path)"
조회 실패                             -> "unavailable"
```

최악의 경우 git 호출 2회다. 첫 호출이 비지 않으면 1회로 끝난다.

근거: 이 처리를 넣어야 "diverged"라는 단어가 신뢰할 수 있는 판정이 된다. 넣지 않으면
그 단어가 세 가지를 뭉개므로, D4가 사겠다고 한 교육 가치를 못 산다. 되돌리는 비용은
분기문 하나다.

**성공 기준을 4상태로 고친다:** `same commit` / `path: n commits (방향)` /
`diverged (no ancestry path)` / `unavailable`. 3상태로 적은 초안을 이 4상태가 대체한다.

### P5 — 조회 상태는 epoch로 무효화한다. anchor는 refresh를 넘어 살아남지 않는다

초안 T4는 3상태를 요구했지만 **repository refresh와의 관계를 적지 않았다.** 이건 gap이다.

`m.repositoryEpoch`(`model.go:29`)는 fetch/pull/mutating action에서 올라가고, graph rows는
그때 재구성된다. anchor 해시와 캐시된 경로 멤버 집합을 그냥 두면, refresh 뒤에도
하이라이트가 남아 **이제는 다른 의미인 행을 계속 강조한다.**

이 저장소는 이미 같은 문제를 Inspector에서 풀어 두었다.

- `commit_inspector_helpers.go:46` `m.commitInspectorStale = true`
- `key_handling_browse.go:215,220` 열 때 `commitInspectorStale = false`,
  `commitInspectorEpoch = m.repositoryEpoch`
- `update_lifecycle.go:42,62,77` epoch가 다른 결과를 폐기

**요구사항:** range 상태도 `graphRangeEpoch`를 들고, epoch가 어긋나면 하이라이트를
지우거나 stale로 표시한다. 조용히 유지하는 선택지는 없다.

관련 prior learning: `adapter-error-results-discarded-by-identity-guard`(10/10) — identity
가드가 조용히 결과를 버려 Inspector가 영원히 Loading에 걸린 사고다. 같은 계층의 문제다.

---

## 미결 — 내일 이어서

각 항목은 답을 고르면 바로 구현으로 갈 수 있는 형태로 적었다. **A/B/C 중 문자로
답하면 된다.** 권고안이 붙어 있으므로 이견이 없으면 권고안을 고르면 된다.

### O1 (blocking, 항목 2) — Inspector 헤더에 날짜 행을 어떻게 넣을까

**문제:** Inspector 헤더는 task 1.21에서 **4행 고정**(commit/message/author/path)으로
계약이 박혀 있다. 여기에 날짜를 넣는 방법이 세 가지이고, 셋 다 pending task 6.8과
겹친다. 6.8은 이미 이 헤더를 문제로 지목했다 — 40컬럼에서
`author: hrllk <heykia3@protonmail.c…`로 이메일이 잘리고, `FROM <full hash>`가
짝(TO)도 없이 얹혀서 가장 먼저 잘린다고 적혀 있다.

- **A) 5행으로 늘리고 `date:` 행을 독립시킨다 (권고)**
  6.8이 요구하는 key/value 정합성과 같은 방향이다. 헤더 높이 계약이 4→5로 바뀌므로
  `commit_inspector_screen_test.go`의 높이 매트릭스(12/20/30)를 다시 확인해야 한다.
  높이 12에서 diff에 남는 줄이 1줄 줄어든다.
- **B) author 행의 `FROM <parent>` 자리를 날짜로 바꾼다**
  높이 계약을 안 건드린다. 6.8이 이미 "FROM은 TO와 짝을 맞추거나 `parent:` 라벨로
  바꿔라"라고 지적했으므로, 그 자리를 비우는 것은 6.8과 같은 방향이다. 다만 parent
  해시를 헤더에서 잃으므로 6.8의 `parent:` 행 신설과 묶어야 한다.
- **C) 데이터 계층(계층 1~4)만 지금 하고 표시는 6.8로 넘긴다**
  이번에 사용자에게 보이는 변화가 없다. 요청을 절반만 이행한다.

권고 근거: A가 요청을 온전히 이행하면서 6.8과 같은 방향으로 간다. 대가는 높이 12
화면에서 diff 1줄이다.

### O2 (blocking, 항목 2) — author date와 commit date 중 무엇을 보여줄까

`git show`는 두 날짜를 따로 갖는다. rebase나 cherry-pick을 하면 두 값이 갈린다.

- **A) 둘이 다를 때만 둘 다 보여준다 (권고)**
  평소에는 한 줄, rebase/cherry-pick된 커밋에서만 두 줄. "이 커밋은 옮겨졌다"가
  화면에서 드러나므로 교육 목적에 맞는다.
- **B) author date만**
  가장 단순하고 `git log` 기본값과 같다. 옮겨진 커밋을 구분할 수 없다.
- **C) 항상 둘 다**
  일관되지만 대부분의 커밋에서 같은 값을 두 번 쓴다.

권고 근거: A는 정보가 있을 때만 줄을 쓴다. 다만 O1-A와 겹치면 헤더가 6행까지 갈 수
있으므로 **O1과 함께 결정해야 한다.**

### O3 (blocking, 항목 3) — 구간을 하이라이팅한 다음 무엇을 할 수 있어야 하나

이게 이 기능의 실제 제품 결정이다. 하이라이팅 자체는 20분이지만, 그 다음이 이
기능의 가치를 정한다.

- **A) 표시 + 요약만 (권고, 이번 범위)**
  하이라이트 + Details에 `range: <n> commits on path` 표시. 경로가 없으면
  `no ancestry path (diverged)`를 명시한다. FF 가능 여부 판정이 여기서 나온다.
- **B) A + 구간 diff를 Inspector로 열기**
  `enter`로 `git diff A..B`를 Inspector에 띄운다. Inspector는 지금 단일 커밋
  스냅샷 계약(`CommitRequest`는 `Commit` 하나)이므로 계약 확장이 필요하다.
  bounded streaming 예산도 다시 봐야 한다.
- **C) A + 구간을 대상으로 하는 액션 (cherry-pick range, rebase --onto)**
  실행 계열로 넘어간다. 이 저장소는 확인 모달·게이트·abort 흐름 계약이 이미
  무거우므로(task 7이 아직 in-progress) 범위가 크게 늘어난다.

권고 근거: A가 D4의 근거("빈 결과 자체가 정보")를 그대로 실현하며 계약 확장이 없다.
B는 다음 슬라이스로, C는 그 다음으로 남긴다.

### O4 (non-blocking) — 분기 시각화(`A...B` + merge base)를 후속으로 둘까

D4에서 기각했지만 교육 가치는 가장 높았던 대안이다. 지금 결정하지 않아도 되지만,
O3-A를 하면서 마커 어휘를 1종으로 고정해 버리면 나중에 3종으로 늘리는 비용이
생긴다. 미리 자리를 남길지만 정하면 된다.

- **A) 마커 어휘를 확장 가능하게 설계하되 이번엔 1종만 쓴다 (권고)**
- **B) 이번엔 1종으로 못 박고, 필요해지면 그때 재설계한다**

---

## 작업 항목

O1/O2/O3가 미결이므로 **T1~T4는 미결과 무관하게 지금 착수 가능**하고, T5~T9는
해당 미결이 풀린 뒤에 착수한다.

### 지금 착수 가능

**T1 — graph log 포맷에 절대 날짜를 추가한다**

`internal/git/repo_parse.go:177` `graphLogArgs`의 포맷 문자열에 날짜 필드를 넣고
`GraphCommit`(`internal/git/repo.go:78`)에 필드를 추가한다.

**손대야 하는 지점 전체 목록** (초안은 아래 3, 5를 빠뜨렸다 — 코덱스 교차확인):

1. `repo_parse.go:186` 포맷 문자열 `%x00%H%x1f%P%x1f%ar%x1f%an%x1f%D%x1f%s`.
   **절대 날짜를 얻는 방법을 함께 정한다.** `graphLogArgs`에는 `--date` 옵션이 없으므로
   `%ad`/`%cd`를 쓰려면 `--date=` 인자를 새로 추가해야 한다. `%as`/`%cs`(short date,
   `YYYY-MM-DD`)는 `--date` 없이도 동작하므로 `yymmdd`만 필요하면 이쪽이 인자 추가가
   없다. **P1은 형식을 정했지만 어느 placeholder를 쓸지는 정하지 않았다** — 구현 시
   `%cs`를 기본으로 하고, P2의 오프셋이 필요한 Inspector에서만 `--date=iso-strict`를
   쓴다.
2. `repo_parse.go:132` `strings.SplitN(..., "\x1f", 6)`과 `len(parts) < 6` 검사.
   **둘 중 하나만 바꾸면 모든 커밋이 조용히 스킵된다** (`continue`).
3. **`repo_parse.go:137-145`의 위치 인덱스.** 파서가 `parts[2]`=age, `parts[3]`=author,
   `parts[4]`=decorations, `parts[5]`=subject로 **하드코딩**되어 있다. 날짜를 중간에
   끼우면 이 인덱스가 전부 밀려서 필드가 서로 뒤바뀌거나 subject가 사라진다.
   **새 필드는 맨 끝에 붙이는 것이 가장 안전하다** (`...%x1f%s%x1f%cs`).
4. `internal/graph/graph.go`의 `Commit`(:19)과 `Node`(:35) 필드 추가.
5. **`internal/graph/graph.go`의 `Node` 재조립 두 곳.** `Nodes`(:72-84)와
   `rowsFromGraph`(:174-180)가 `Node`를 필드 단위로 다시 만든다. 여기를 안 올리면
   구조체에 필드가 있어도 값이 **조용히 zero value로 떨어진다.** T2의 Details 경로가
   이 rows에 의존한다(`navigation_graph.go:102-133`).
6. `internal/app/startup_projection.go:28`, `navigation_graph.go:128`,
   `cherry_pick.go:31`의 필드 복사.

완료 기준: 새 필드가 파서 테스트에서 검증되고, **`graph.Nodes`와 `rowsFromGraph`를
통과한 뒤에도 값이 남아 있음을 단언한다**(5번 누락을 잡는 유일한 테스트). 필드 추가
후에도 기존 graph 행 렌더 결과가 **바이트 단위로 동일**하다(아직 표시하지 않으므로).
`go test ./...`, `go build ./cmd/graphkeeper`, `scripts/check` CLEAN.

**T2 — Details 패널 graph 섹션에 `date:` 행을 만든다** (D5 전반부)

`internal/app/view_detail.go:27-38`. `focus` 다음 줄에 P1 형식으로 넣는다.

완료 기준: graph 섹션 커서를 옮기면 `date:`가 따라 바뀐다. 폭 40/60/80에서 이 행이
Details 박스를 넘치지 않는다. tags 섹션의 `age:`와 어휘가 충돌하지 않는다
(둘 다 남길지, `date:`로 통일할지는 6.6의 어휘 통일 범위와 겹치므로 여기서는
graph 섹션만 손댄다).

**T3 — Commit Inspector 데이터 계층에 날짜를 통과시킨다** (O1/O2와 무관한 부분)

R3 표의 계층 1~4. 표시(계층 5)는 O1 결정 후.

- `internal/git/repo.go:110` `CommitInspection`에 날짜 필드 추가
- `internal/git/repo_exec.go:162` 포맷 문자열 + **`SplitN(meta, "\x00", 5)`와
  `len(parts) != 5`를 함께 올린다**. 안 올리면 즉시 `invalid commit metadata`.
- `internal/commitinspector/contract.go:93` `CommitSnapshot`에 필드 추가
- `internal/adapter/out/commitinspector/reader.go:86` 매핑 추가

완료 기준: adapter 테스트가 날짜 값이 계층 4까지 도달함을 단언한다. 이 단계에서
화면 변화는 없다.

**T4 — 구간 상태와 조회를 만든다** (O3와 무관한 부분)

- `internal/git`에 `AncestryPath(ctx, from, to) ([]string, error)` 추가.
  `rev-list --ancestry-path from..to`.
  - **빈 ref 가드를 반드시 넣는다.** `Divergence`(`repo_exec.go:830-832`)가
    `if left == "" || right == ""`로 명시적으로 막는 이유가 있다. git에서 `..B`는
    `HEAD..B`로 해석되므로, anchor가 빈 문자열이면 조용히 **HEAD를 기준으로 한 전혀
    다른 질문**에 답한다. 오류도 안 난다. 같은 가드를 복사한다.
  - `VIRTUAL_CONFLICT_HASH` 행은 anchor로 받지 않는다. `graph_search.go:23`이 이미
    같은 이유로 이 해시를 인덱스에서 제외한다. 그 관습을 따른다.
- `internal/app` 모델에 anchor 상태 추가. `navigationState`(`model.go:114`)에
  `graphRangeAnchor string` + **`graphRangeEpoch uint64`** (P5).
- `GraphProjection`(`view_projection.go:28`)에 `RangeMembers map[string]bool`을
  추가한다. 같은 구조체의 `Handshake map[string]bool`(:30)과 **정확히 같은 관습**을
  쓴다. 새 관습을 만들지 않는다.
  - **구조체에 필드를 더하는 것만으로는 화면에 아무 일도 안 일어난다** (초안이 이걸
    "충분하다"는 뜻으로 읽히게 적었다 — 코덱스가 지목한 최대 오해). 세 곳을 함께
    올려야 한다.
    1. `view_projection.go:84`의 **수동 조립부**에 `RangeMembers:`를 채운다.
       이 구조체는 리터럴 한 줄로 손으로 조립된다.
    2. `view_graph.go:52-58`에서 행별 멤버십을 **렌더러 인자로 넘긴다.**
       `stashCount`, `isHandshake`가 이미 같은 방식으로 맵에서 꺼내 스칼라로
       전달된다(`p.StashCounts[hash]`, `p.Handshake[hash]`). 같은 형태로 넘긴다.
    3. `graph_render.go`의 두 렌더러 시그니처가 늘어난다(T6).
  - 렌더러 인자가 이미 10개다. 세 번째 행별 스칼라를 더하는 것은 냄새이지만, 새
    관습을 만드는 것보다 낫다. 정리는 이 작업 범위가 아니다.
- **조회 결과 전달은 message의 `err` 필드로 한다.** 초안은
  `LocalBranchesKnown`/`Fresh`/`Error`(`repository_read.go:42-43`)를 복사하라고 적었으나
  그 관용구는 **refresh로 재생성되는 repository projection 상태**용이다. ancestry 조회는
  사용자 키 입력으로 촉발되는 **일회성 비동기 질의**이므로, 맞는 짝은
  `checkGraphActionTarget`(`commands.go:536-548`)이 `graphActionCheckMsg{..., err: err}`로
  오류를 실어 보내는 관용구다. 저장소에 있는 두 관용구 중 후자를 쓴다.
  - epoch 가드는 별개다. P5대로 `update_lifecycle.go:42,62,77`의 폐기 패턴을 따른다.

완료 기준: P4의 **4상태**(`same commit` / `path: n (방향)` / `diverged` / `unavailable`)가
각각 단위 테스트로 단언된다. 3상태로는 미완이다 — 방향 역전과 동일 커밋이 "분기"로
읽히면 P4가 지적한 자기모순이 그대로 남는다. `git stash list` 실패를 "없음"으로
표시했던 `update_stash.go` 사고(`24e6e63`에서 수정)와 같은 실수를 반복하지 않기 위한
조건이다.

### 미결 해소 후

**T5 — Inspector 헤더에 날짜를 표시한다** (O1, O2 필요)

계층 5. **`commit_inspector_screen.go`의 `renderCommitInspectorScreen` 경로만 고친다.**
`commit_inspector.go:122` `renderCommitInspectorPopup`은 프로덕션 호출자가 없는 죽은
렌더러이므로 손대지 않는다 (위 "정정" 참조). 초안의 "두 경로를 모두 고친다"는 지시는
철회한다.

**T6 — 구간 하이라이트를 렌더한다** (O3 필요)

`graph_render.go`의 `renderGraphLineWithSearch`(:55)와
`renderRawGraphLineWithSearch`(:100) **둘 다**. raw 경로가 실사용 경로다.

- 색에 의존하지 않는 신호를 반드시 함께 넣는다. `theme.go`의
  `searchMatchMark`(underline+bold)를 재사용하거나, `df7ed32`에서 도입한 거터
  마커 패턴을 따른다. pending task 6.3과 `highlighting-color-map.md`의
  "NO_COLOR=1에서 visible label과 marker를 보존한다" 정책이 이를 요구한다.
- anchor 커밋 자체와 경로 멤버를 구분한다. anchor는 이미 커서/HEAD 마커와 겹칠 수
  있으므로 우선순위를 정해야 한다.

**T7 — 구간 요약을 Details에 표시한다** (O3 필요)

T4의 **4상태**(P4)를 그대로 노출한다.

```
range: same commit
range: 7 commits (4d8fcbc -> b2fd316)
range: diverged (no ancestry path)
range: unavailable
```

방향을 함께 보여주는 것이 P4의 핵심이다. 방향 없이 개수만 보여주면 사용자가 어느
쪽이 조상인지 알 수 없고, 그걸 알려주는 것이 이 기능의 교육 가치다.

**T8 — 키와 문서를 동기화한다**

`v` 바인딩을 `hidden_hotkeys.go`의 **graph 섹션(:231-249)에만** 등록한다.

**정정 (코덱스 교차확인).** 초안은 "메인 footer에도 등록한다"고 적었다. 그건 기록된
정책 위반이다. `docs/decisions.md:8-9`가 이렇게 못 박았다.

> Keep only Global core navigation keys in the main footer. Active-section actions
> remain discoverable through the `?` overlay.

코드도 그대로다 — footer는 `globalHotkeyItems`(`hidden_hotkeys.go:33-50`)만 소스로
쓰고, Graph 액션은 섹션 로컬(`:225-250`)이다. `v`는 graph 섹션 전용 액션이므로
**`?` 오버레이에만** 들어간다. footer에 넣으면 T8이 스스로 정책을 깬다.

**이 항목은 pending task 8과 직접 충돌한다.** task 8은 "동작하는 전역 키 10개가
푸터에도 오버레이에도 없다 — 설계 결정 필요"이며 미결 상태다. 문서화되지 않은 키를
11개로 늘리지 않으려면, `v`는 **등록과 같은 커밋에서** 구현한다. 키를 먼저 넣고
문서화를 나중에 하는 순서를 금지한다.

**T9 — 6.4에 날짜 컬럼을 편입한다** (D5 후반부)

task 6.4의 details와 완료 기준에 yymmdd 컬럼을 추가한다. 6.4의 기존 완료 기준
("100컬럼에서 title 가용폭이 80컬럼 대비 증가")은 **유지한다** — 즉 날짜 컬럼은
회수한 폭 안에서 해결하고, title 가용폭을 희생해서 얻지 않는다.

**필수 조건 (R2-bis).** `docs/decisions.md`에 반전 항목을 함께 적는다. 지금 기록된
2026-08-01 결정("title은 나머지 폭 전부를 받는다", 그 열거 목록에 날짜 없음)과
2026-07-10 원칙("graph 행은 컬럼이 아니라 compact marker로 해결한다")을 명시적으로
갱신하지 않으면, 코드와 결정 기록이 어긋난 채로 남는다. 코드 변경과 같은 커밋에서
적는다.

---

## 테스트 전략

**Prior learning (`view-never-called-in-tests`, confidence 9/10, 2026-08-26):**
이 저장소의 테스트 파일 64개 중 `model.View()`를 호출하는 것은 0개다. 렌더 테스트는
존재하지만 모델을 손으로 조립하므로, **Update가 바꾼 상태를 렌더러가 무시하는
종류의 버그를 구조적으로 잡을 수 없다.**

세 요청은 전부 "상태를 바꾸고 그 결과가 화면에 나타나는지"가 본질이므로 이 공백에
정면으로 걸린다. 특히 T6의 위험이 크다 — 두 렌더러 중 테스트가 닿는 쪽만 고치면
테스트는 초록이고 실행 화면은 안 바뀐다.

따라서 다음을 요구한다.

1. **T6에는 raw 경로 테스트를 필수로 넣는다.** `row.Graph != ""`인 fixture로
   `renderRawGraphLineWithSearch`를 직접 검증한다. non-raw 경로만 검증하는 테스트는
   이 작업의 완료 근거로 인정하지 않는다.
2. **NO_COLOR 단언을 넣는다.** 하이라이트가 색 없이도 식별되는지 검증한다.
   `df7ed32`가 커서에 대해 같은 것을 했으므로 그 테스트를 참고한다.
3. **T4의 4상태를 각각 단언한다** (P4). 정방향 경로 있음 / **역방향 경로 있음** /
   **동일 커밋** / 진짜 분기 / 조회 실패. 3상태만 단언하는 테스트는 P4가 지적한
   자기모순을 통과시킨다.
4. **T1의 파서 필드 수 변경에는 회귀 단언을 넣는다.** 필드 수가 어긋나면
   `continue`로 조용히 스킵되므로, "커밋 수가 0이 아니다"를 단언하는 테스트가
   없으면 이 실패는 빈 그래프로만 드러난다.
5. 폭 매트릭스 40/60/80과 높이 12/20/30을 T2, T5, T7에 적용한다.
6. **epoch 무효화를 단언한다** (P5). anchor를 세운 뒤 `repositoryEpoch`를 올리고,
   하이라이트가 지워지거나 stale로 표시되는지 검증한다. 조용히 남아 있으면 실패다.
7. **빈 ref 가드를 단언한다.** `AncestryPath(ctx, "", "abc")`가 오류를 반환하는지
   검증한다. 이 테스트가 없으면 `..B` = `HEAD..B` 해석이 조용히 다른 답을 준다.
8. **`VIRTUAL_CONFLICT_HASH`를 anchor로 시도하는 테스트를 넣는다.** 거부되거나
   무시되어야 하며, git 서브프로세스에 도달하면 안 된다.

`scripts/check`(test + vet + build + gofmt/diff-check) CLEAN을 각 T의 완료 조건에
포함한다.

---

## 기존 task와의 관계

| 기존 task | 상태 | 이 계획과의 관계 |
|---|---|---|
| **1.8** Graph 정보 열 재배치 | done | 이 계획의 T9가 1.8의 결정을 **부분적으로 되돌린다.** 1.8은 date 컬럼을 의도적으로 제거했다. 되돌리는 근거는 "제거된 것은 상대 시간이고 지금 필요한 것은 절대 날짜"이며, 1.8이 얻은 title 폭은 6.4의 재설계로 보전한다. |
| **6.4** Graph row 컬럼 예산 재설계 | pending | **T9가 6.4에 편입된다.** 순서 역전 금지: 6.4 이전에 날짜 컬럼을 넣으면 6.4의 완료 기준을 스스로 어긋나게 만든다. |
| **6.8** 카피·정보 우선순위 polish | pending | **O1이 6.8과 같은 헤더를 건드린다.** 6.8은 이미 author 행의 이메일 잘림과 짝 없는 `FROM`을 지목했다. T5는 6.8과 같은 방향으로만 진행한다. |
| **6.3** 색에 의존하지 않는 selection·focus 신호 | pending | **T6이 6.3의 제약을 상속한다.** 구간 하이라이트를 색으로만 표현하면 6.3을 집는 사람이 다시 고쳐야 한다. |
| **6.6** 축약·footer·선 문자 어휘 통일 | pending | T2의 `date:` 라벨과 tags의 `age:` 라벨 이원화가 6.6의 어휘 통일 범위에 들어간다. 이 계획에서는 통일하지 않고 6.6에 남긴다. |
| **task 8** 전역 키 10개가 문서에 없음 | pending, 설계 결정 필요 | **T8이 이 문제를 11개로 늘릴 수 있다.** `v`는 등록과 같은 커밋에서만 구현한다. |
| **task 7** merge/rebase 게이트 | in-progress | O3-C(구간 액션)를 고르면 이 미결 게이트 계약과 얽힌다. O3-A를 권고하는 이유 중 하나다. |

---

## 리스크

1. **T1의 필드 수 불일치가 조용히 실패한다.** `repo_parse.go:132`는 필드가 부족하면
   `continue`한다. 포맷과 `SplitN` 인자와 `len(parts)` 검사, 셋 중 하나를 놓치면
   그래프가 빈 화면이 되고 오류 메시지는 없다. 테스트 전략 4번이 이 리스크에 대한
   대응이다.

2. **T3의 `SplitN(..., 5)`는 즉시 깨진다.** T1과 달리 이쪽은 오류를 반환하므로
   발견은 빠르지만, 포맷만 바꾸고 커밋하면 Inspector 전체가 죽는다.

3. **T6의 렌더러 이중화.** 두 함수 중 하나만 고치면 테스트 초록 + 실행 화면 무변화.
   테스트 전략 1번이 대응이다.

4. **`--ancestry-path` 성능.** 큰 저장소에서 anchor를 세운 뒤 커서를 움직일 때마다
   rev-list를 돌리면 느려진다. 커서 이동마다 조회하지 않고 **두 번째 선택이 확정된
   시점에만** 조회하는 설계가 필요하다. O3-A의 상호작용 설계에서 이를 명시할 것.

5. **anchor 상태와 기존 마커의 우선순위 미정.** HEAD 마커, 커서 포인터, 검색 매치,
   stash/tag 마커가 이미 같은 행에 겹칠 수 있다. 구간 마커가 다섯 번째로 들어온다.
   T6에서 우선순위 표를 먼저 적어야 한다.

6. **날짜 데이터가 늘면 graph log 출력이 커진다.** `commitLimit`이 0(무제한,
   `update_execute.go:306`)이므로 큰 저장소에서 필드 하나가 전체 커밋 수만큼 늘어난다.
   실측 필요.

---

---

## CEO review 산출물 (2026-09-08, HOLD SCOPE)

### 이미 존재하는 것 (재사용 대상)

| 하위 문제 | 이미 있는 코드 | 계획이 재사용하나 |
|---|---|---|
| rev-list 호출 관습 | `internal/git/repo_exec.go:829` `Divergence` + 빈 ref 가드 :830 | 예 (T4) |
| 해시 집합을 렌더러까지 전달 | `view_projection.go:30` `GraphProjection.Handshake map[string]bool` | 예 (T4) |
| 색 없는 하이라이트 토큰 | `theme.go` `searchMatchMark`(underline+bold), `searchFocusMark`(reverse+bold) | 예 (T6) |
| 거터 마커 패턴 | `df7ed32` NO_COLOR 커서 수정 | 예 (T6) |
| 일회성 비동기 질의 + 오류 전달 | `commands.go:536-548` `graphActionCheckMsg{err}` | 예 (T4, 리뷰에서 교정) |
| epoch 기반 stale 무효화 | `commit_inspector_helpers.go:46`, `key_handling_browse.go:215,220`, `update_lifecycle.go:42,62,77` | 예 (P5, 리뷰에서 추가) |
| Details key/value 렌더 | `view_detail.go:110` | 예 (T2, T7) |
| 합성 행 제외 관습 | `graph_search.go:23` `VIRTUAL_CONFLICT_HASH` 스킵 | 예 (T4, 리뷰에서 추가) |
| 3상태 프로젝션 관용구 | `repository_read.go:42-43` | **아니오 — 짝이 아니다** (T4 교정 참조) |

재구축하는 것은 없다. 새 의존성도 없다.

### 오류·구제 레지스트리

| 코드패스 | 무엇이 잘못될 수 있나 | 오류 종류 |
|---|---|---|
| `git.AncestryPath` | anchor/cursor가 빈 문자열 | 명시적 오류 (가드) |
| | anchor 커밋이 사라짐 (브랜치 삭제, gc) | git exit != 0 |
| | `VIRTUAL_CONFLICT_HASH`가 ref로 전달됨 | git exit != 0 (호출 전 차단) |
| | context 취소 (다음 키 입력) | `context.Canceled` |
| | git 실행 불가 | exec 오류 |
| `graphLogArgs` 파싱 (T1) | 필드 수 불일치 | **없음 — 조용히 `continue`** |
| `InspectCommit` (T3) | `SplitN` 개수 불일치 | `invalid commit metadata` |
| Details 렌더 (T2, T7) | 날짜가 빈 문자열 | 없음 (표시 문제) |

| 오류 | 처리하나 | 처리 동작 | 사용자가 보는 것 |
|---|---|---|---|
| 빈 ref | Y | 조회 전 거부 | anchor 미설정 상태 유지 |
| anchor 소멸 | Y | msg의 `err`로 전달 | `range: unavailable` |
| VIRTUAL_CONFLICT_HASH | Y | anchor 후보에서 제외 | anchor가 안 세워짐 |
| `context.Canceled` | Y | 결과 폐기, 상태 불변 | 변화 없음 (투명) |
| git 실행 불가 | Y | msg의 `err` | `range: unavailable` |
| **T1 필드 수 불일치** | **N — GAP** | — | **빈 그래프, 오류 없음** |
| epoch 불일치 | Y (P5) | 하이라이트 폐기 | 하이라이트 사라짐 |

### 실패 모드 레지스트리

| 코드패스 | 실패 모드 | 처리 | 테스트 | 사용자가 보는 것 | 로그 |
|---|---|---|---|---|---|
| `AncestryPath` | 빈 ref → `HEAD..B`로 오해석 | Y | 전략 7 | 정상 (차단됨) | Y |
| `AncestryPath` | 역방향 선택 → "분기"로 오표시 | Y (P4) | 전략 3 | 방향 표시된 경로 | — |
| `AncestryPath` | 동일 커밋 → "분기"로 오표시 | Y (P4) | 전략 3 | `same commit` | — |
| range 상태 | refresh 후에도 하이라이트 잔존 | Y (P5) | 전략 6 | 하이라이트 사라짐 | Y |
| `graphLogArgs` 파싱 | 필드 수 불일치 → 전 커밋 스킵 | **N** | 전략 4 | **빈 그래프** | **N** |
| T6 렌더 | raw 경로 미수정 → 화면 무변화 | Y | 전략 1 | 정상 | — |
| T6 렌더 | 색 전용 하이라이트 | Y | 전략 2 | NO_COLOR에서도 식별 | — |

**CRITICAL GAP 1건:** `graphLogArgs` 파싱 필드 수 불일치. 오류를 만들지 않고
`continue`하므로 코드가 잘못돼도 조용하다. 완화는 테스트뿐이므로 전략 4번은
선택이 아니다.

### 다이어그램

**구간 판정 흐름 (P4)**

```
  v 누름 (anchor 없음)              v 누름 (anchor 있음)
        │                                   │
        ▼                                   ▼
  anchor = 커서 해시              cursor == anchor ?
  (VIRTUAL_CONFLICT_HASH               │        │
   이면 거부)                        예        아니오
        │                             ▼         ▼
        ▼                      "same commit"   AncestryPath(anchor..cursor)
  하이라이트 없음, anchor 마커만                    │
                                    ┌──────────────┼──────────────┐
                                  err            비었음         결과 n개
                                    ▼              ▼               ▼
                            "unavailable"   AncestryPath      "path: n
                                            (cursor..anchor)   (anchor->cursor)"
                                                 │                  │
                                        ┌────────┼────────┐         ▼
                                      err      비었음    n개    RangeMembers 채움
                                        ▼        ▼        ▼
                              "unavailable"  "diverged"  "path: n
                                                        (cursor->anchor)"
```

**range 상태 생애 (P5)**

```
   [없음] ──v──▶ [anchor만] ──v──▶ [range 활성] ──esc/v──▶ [없음]
      ▲              │                  │
      │              │                  │ repositoryEpoch 증가
      └──────────────┴──────────────────┘  (fetch/pull/mutating action)
                  전부 폐기 — 조용히 유지하지 않는다
```

**날짜 데이터 흐름 (T1, T3)**

```
  git log --format=...%<date> ──▶ SplitN(N+1) ──▶ GraphCommit.Date
        │                              │                  │
        ▼                              ▼                  ▼
   [필드 누락?]                  [개수 불일치?]      [빈 문자열?]
   git이 빈 값             ⚠ continue = 전 커밋 스킵    "-" 표시
                              (CRITICAL GAP)

  git show --format=...%ad%cd ──▶ SplitN(7) ──▶ CommitInspection ──▶ CommitSnapshot
                                      │
                                      ▼
                              [개수 != 7?] → invalid commit metadata (즉시 발견)
```

### NOT in scope

| 고려했으나 제외 | 이유 |
|---|---|
| `A...B` + merge base 분기 시각화 | 마커 3종이 필요해 task 6.3의 NO_COLOR 제약과 어긋남. O4에서 자리만 남길지 결정. |
| 구간 diff를 Inspector로 열기 | `CommitRequest`가 단일 커밋 계약. 계약 확장 + bounded streaming 예산 재검토 필요. O3-B. |
| 구간 대상 액션 (cherry-pick range, rebase --onto) | task 7(게이트)이 아직 in-progress. 실행 계열은 그 뒤. O3-C. |
| graph 행 yymmdd 컬럼 (지금) | R2 폭 예산, R2-bis 기록된 결정. 6.4로 이월(T9). |
| tags `age:` / graph `date:` 어휘 통일 | task 6.6의 어휘 통일 범위. 여기서 세 번째 어휘를 만들지 않는 것까지만 지킨다. |
| 죽은 렌더러 `renderCommitInspectorPopup` 삭제 | 이 작업의 범위가 아니다. `TODOS.md`로 이월. |
| `inspectorStatus`(`commit_inspector.go:456`) 삭제 | 호출자 0개. 무관한 정리. `TODOS.md`로 이월. |
| 로컬 타임존 변환 옵션 | P2가 원본 오프셋으로 확정. 요청이 오면 그때. |

### Dream state delta

```
  현재 상태                      이 계획                     12개월 이상적 상태
  ─────────                     ─────────                    ──────────────────
  커밋이 언제인지         ──▶   Details에 날짜,        ──▶   graph를 훑으며 시간·
  알 수 없다                    Inspector에 상세             구조·분기를 한 번에
                                                              읽는다
  두 지점의 관계를        ──▶   두 점을 찍으면 경로와   ──▶  분기 지점과 양쪽
  말로 설명해야 한다            방향과 분기 여부를            고유 커밋까지 한
                                도구가 답한다                 화면에서 (O4/Approach B)
  컬럼 예산이 깨져 있다   ──▶   손대지 않는다           ──▶  6.4가 연속 성장으로
                                (악화시키지 않음)             재설계 완료
```

이 계획은 12개월 이상적 상태로 **가는** 방향이다. 컬럼 예산을 악화시키지 않기로 한
D5가 그 이유다. 만약 D5를 반대로 골랐다면 6.4를 더 어렵게 만들어 이상적 상태에서
멀어졌을 것이다.

### 관측성

이 저장소는 `internal/telemetry`와 `internal/events` 싱크를 갖고 있고, 이미
**싱크에만 보내고 UI에는 거짓을 표시한 사고**를 겪었다 — `update_stash.go:16-19`가
`git stash list` 실패를 싱크로만 보내고 `(no stash entries)`를 표시했다(`24e6e63`에서
수정, `TODOS.md`에 기록).

**요구사항:** `AncestryPath` 실패는 이벤트 싱크와 **UI 양쪽**에 도달한다.
`range: unavailable`이 UI 몫이고, 원인 문자열이 싱크 몫이다. 한쪽만 하면 같은 사고다.

### 배포

새 산출물 없음. 마이그레이션 없음. 단일 바이너리 빌드 변화 없음. 롤백은
`git revert` 1회. feature flag 불필요 — 각 T가 독립적으로 revert 가능하다. 이슈 없음.

### 장기 궤적

- **되돌리기 쉬움: 4/5.** T1~T4는 순수 추가. T9(컬럼)만 결정 기록을 건드리므로 3/5.
- **부채:** T9가 6.4에 의존을 만든다. T9 본문과 6.4 details 양쪽에 기록해야 유실되지
  않는다. 계획이 그걸 요구한다.
- **12개월 뒤 읽는 사람:** P4의 4상태표와 R2-bis의 결정 기록 인용이 "왜 이렇게
  했나"를 스스로 설명한다.

### 설계·UX

UI 범위 있음. 남은 문제:

- **마커 우선순위 표가 아직 없다.** 한 행에 HEAD 마커, 커서 포인터, 검색 매치,
  stash/tag 마커가 이미 겹칠 수 있고 range 마커가 다섯 번째다. T6은 우선순위 표를
  선행 조건으로 요구한다(리스크 5).
- **T6 착수 전에 `/plan-design-review`를 돌린다.** `TODOS.md`에 이미 같은 요구가
  Phase 4에 대해 적혀 있다. 같은 이유가 여기에도 적용된다.

### 아웃사이드 보이스 (코덱스, 독립 검증)

`codex exec` (codex-cli 0.153.3, reasoning effort high, read-only)로 이 계획의 사실
주장을 코드에 대조했다. 12건을 냈고, 그중 **10건을 수용**했다. 두 모델이 독립적으로
같은 결론에 도달한 항목이 하나 있다 — D4의 방향 오분류(P4). 교차 합의가 있으므로
그 항목의 신뢰도가 가장 높다.

수용한 것:

| 코덱스 지적 | 처분 |
|---|---|
| 80컬럼 폭 전제가 낡았다 (56/11자 → 실제 43/39, title 음수) | **수용, R2 전면 교체.** 가장 중요한 사실 정정. |
| `graph.Nodes`(:72-84)와 `rowsFromGraph`(:174-180)의 Node 재조립 누락 | **수용, T1에 5번으로 추가.** 값이 조용히 zero value로 떨어지는 경로. |
| 파서 위치 인덱스 `parts[2:6]` 하드코딩 | **수용, T1에 3번으로 추가.** 필드를 끝에 붙이도록 지시 변경. |
| `graphLogArgs`에 `--date` 옵션이 없다 | **수용, T1에 1번으로 반영.** `%cs` 기본 + Inspector만 `--date=iso-strict`. |
| Inspector 새 필드 개수가 미지정 (5→6인지 7인지) | **수용.** O2와 묶여 있으므로 O2 답이 개수를 정한다. |
| Inspector "두 렌더 경로"가 부정확 | **수용 — 자체 리뷰에서도 동일 결론.** T5 정정. |
| **D4가 역방향 선택을 분기로 오분류** | **수용 — 자체 리뷰에서도 동일 결론(P4). 교차 합의.** |
| T8이 footer 키 정책(`decisions.md:8-9`)과 충돌 | **수용, T8 정정.** `?` 오버레이 전용으로 변경. |
| `RangeMembers` 추가만으로는 하이라이트가 안 나온다 (projection 수동 조립 + 렌더러 배관) | **수용, T4에 3단계 배관 명시.** 코덱스가 "최대 오해"로 지목. |
| bool 맵 하나로 3상태를 표현할 수 없다 | **수용 — P4/P5로 이미 4상태 + epoch로 해소.** |

기각한 것과 이유:

| 코덱스 지적 | 기각 이유 |
|---|---|
| "task 6.8 기록은 `내용 미공유`뿐이므로 author 이메일 잘림 주장은 근거 없다" | 코덱스가 **스텁 항목**을 읽었다. `tasks.json:1551-1558`은 task **10** 아래의 의존 그래프 플레이스홀더다. 실제 6.8은 task **6**의 subtask이고 D-014/016/018 전문을 담고 있다. 계획의 주장은 근거가 있다. |
| "task 6.4에는 완료 기준이 없다" | 같은 원인. 실제 6.4 subtask에 "완료 기준: 100컬럼에서 title 가용폭이 80컬럼 대비 증가하고..."가 있다. |

**다만 이 기각이 새 문제를 드러냈다 (신규 발견).** `.taskmaster/tasks/tasks.json`에
task 6.1~6.8이 **두 번** 표현되어 있다.

- task **6**의 subtasks: 실제 내용 (감사 리포트 인용, 완료 기준 포함)
- task **10**의 subtasks 9개: `taskId: "6.4"`, `title: "6.4 (내용 미공유)"`,
  `description: "원 우선순위 목록의 의존 그래프에만 등장하고 내용이 공유되지 않았다."`

즉 같은 작업이 두 id로 존재하고 한쪽은 빈 껍데기다. 독립 리뷰어(사람이든 모델이든)가
스텁을 먼저 읽으면 **"근거 없는 주장"이라는 반대 결론에 도달한다.** 실제로 이번에
그렇게 됐다. task 9에도 같은 형태의 스텁이 47개(`T1`~`T33`, `C1`~`C5`, `E1`~`E4`,
`L1`~`L5`) 있다.

이건 이 계획의 범위가 아니지만 `REPO_MODE: solo`이므로 짚어 둔다. `TODOS.md`로 이월.

## Implementation Tasks

이 리뷰의 findings에서 나온 것만 적는다.

- [ ] **C1 (P1, human: ~30min / CC: ~5min)** — plan — P4 4상태 판정을 T4/T7 계약으로 확정
  - Surfaced by: Section 4 — `--ancestry-path`가 비는 경우가 3가지인데 계획이 전부 "diverged"로 표시
  - Files: 이 문서 (반영 완료)
  - Verify: T4 완료 기준이 4상태를 요구하는지 확인
- [ ] **C2 (P1, human: ~1h / CC: ~10min)** — app — range 상태에 epoch 가드 추가
  - Surfaced by: Section 1 — `repositoryEpoch` 증가 후 하이라이트가 조용히 잔존
  - Files: `internal/app/model.go`, `internal/app/view_projection.go`, `internal/app/update_lifecycle.go`
  - Verify: 테스트 전략 6번
- [ ] **C3 (P1, human: ~15min / CC: ~3min)** — git — `AncestryPath`에 빈 ref 가드
  - Surfaced by: Section 3 — `..B`가 `HEAD..B`로 조용히 해석됨
  - Files: `internal/git/repo_exec.go`
  - Verify: 테스트 전략 7번
- [ ] **C4 (P2, human: ~10min / CC: ~2min)** — git — anchor 후보에서 `VIRTUAL_CONFLICT_HASH` 제외
  - Surfaced by: Section 4 — 합성 행이 ref로 전달될 수 있음
  - Files: `internal/app` anchor 설정 지점
  - Verify: 테스트 전략 8번
- [ ] **C5 (P2, human: ~20min / CC: ~5min)** — docs — T9에 `docs/decisions.md` 반전 기록 요구
  - Surfaced by: Section 10 — 2026-08-01/2026-07-10 결정과 코드가 어긋날 위험
  - Files: 이 문서 (반영 완료), 이후 `docs/decisions.md`
  - Verify: T9 필수 조건 문단 존재
- [ ] **C6 (P2, human: ~5min / CC: ~1min)** — plan — T5의 "두 경로 모두 수정" 지시 철회
  - Surfaced by: Section 5 — `renderCommitInspectorPopup` 프로덕션 호출자 0개
  - Files: 이 문서 (반영 완료)
  - Verify: T5가 `renderCommitInspectorScreen`만 지목
- [ ] **C7 (P3, human: ~30min / CC: ~5min)** — app — 죽은 Inspector 렌더러와 `inspectorStatus` 정리
  - Surfaced by: Section 5 — 테스트만 붙은 죽은 코드
  - Files: `internal/app/commit_inspector.go:122,456`, `internal/app/commit_inspector_test.go:74,98`
  - Verify: `scripts/check` CLEAN, 별도 커밋
- [ ] **C8 (P3, human: ~15min / CC: ~3min)** — plan — T1의 필드 수 CRITICAL GAP에 회귀 테스트 명시
  - Surfaced by: Section 2 — 유일한 무음 실패 경로
  - Files: 이 문서 (반영 완료)
  - Verify: 테스트 전략 4번
- [ ] **C9 (P1, human: ~30min / CC: ~5min)** — plan — 80컬럼 폭 전제를 실측값으로 교체
  - Surfaced by: 아웃사이드 보이스 — 초안의 56/11자는 `2d9c4c4` 이전 값
  - Files: 이 문서 (반영 완료), 근거는 `internal/app/shell_width_test.go:113-118`
  - Verify: R2 표가 40/60/80/140의 title 가용폭을 음수 포함해 적고 있는지
- [ ] **C10 (P1, human: ~20min / CC: ~5min)** — plan — T1에 `graph.Nodes`/`rowsFromGraph` 재조립 추가
  - Surfaced by: 아웃사이드 보이스 — 값이 zero value로 조용히 소실
  - Files: 이 문서 (반영 완료), 이후 `internal/graph/graph.go:72-84,174-180`
  - Verify: T1 완료 기준이 rows 통과 후 값 보존을 단언하는지
- [ ] **C11 (P1, human: ~15min / CC: ~3min)** — plan — T4에 projection 조립 + 렌더러 배관 3단계 명시
  - Surfaced by: 아웃사이드 보이스 — 구조체 필드만 더하면 화면 무변화
  - Files: 이 문서 (반영 완료), 이후 `view_projection.go:84`, `view_graph.go:52-58`
  - Verify: T4에 3단계 목록 존재
- [ ] **C12 (P2, human: ~10min / CC: ~2min)** — plan — T8을 `?` 오버레이 전용으로 정정
  - Surfaced by: 아웃사이드 보이스 — `decisions.md:8-9` footer 정책 위반
  - Files: 이 문서 (반영 완료)
  - Verify: T8이 footer 등록을 요구하지 않는지
- [ ] **C13 (P3, human: ~1h / CC: ~15min)** — taskmaster — task 6.x/9.x 스텁 중복 제거
  - Surfaced by: 아웃사이드 보이스 기각 분석 — 스텁을 읽은 리뷰어가 반대 결론에 도달
  - Files: `.taskmaster/tasks/tasks.json` (task 10의 subtasks 9개, task 9의 47개)
  - Verify: 하나의 taskId가 한 곳에만 존재

---

## 다음 행동

다른 디바이스에서 이어서 할 때의 순서다.

1. 이 문서의 **미결 O1, O2, O3**에 문자로 답한다. O4는 O3와 함께 답하면 좋다.
   각 미결에 권고안이 붙어 있으므로 이견이 없으면 권고안을 고르면 된다.
2. **T1~T4는 미결과 무관하므로 답을 기다리지 않고 착수 가능하다.** 단 리뷰가 추가한
   C2(epoch 가드), C3(빈 ref 가드), C4(합성 행 제외)를 T4에 포함해야 한다.
3. 미결이 풀리면 `/plan-eng-review`로 T5~T9의 구현 순서와 테스트 경계를 확정한다.
   이 저장소에서 eng review가 유일한 ship 게이트다.
4. T6 착수 전에 `/plan-design-review`를 돌린다 (마커 우선순위 표가 없기 때문).
5. taskmaster에 등록한다. 등록 위치는 `feature`(task 1) 하위가 아니라 **새 top-level
   task**가 맞다 — 세 요청이 서로 독립적이고 6.4/6.8과 교차 의존을 갖기 때문이다.

### 이 세션에서 확정된 것 / 확정되지 않은 것 요약

| 항목 | 상태 | 어디서 |
|---|---|---|
| 구간의 정의 = `--ancestry-path` | **확정** | D4 |
| 날짜는 Details 먼저, 컬럼은 6.4로 | **확정** | D5 |
| 날짜 형식 (graph 6칸 / Details / Inspector) | 잠정 | P1 |
| 타임존 = 원본 오프셋 | 잠정 | P2 |
| 구간 선택 키 = `v` | 잠정 | P3 |
| 방향 자동 판정 + 4상태 | 잠정 (리뷰에서 추가) | P4 |
| epoch 무효화 | 잠정 (리뷰에서 추가) | P5 |
| Inspector 헤더 행 배치 | **미결 — blocking** | O1 |
| author date vs commit date | **미결 — blocking** | O2 |
| 하이라이팅 다음 행동 | **미결 — blocking** | O3 |
| 분기 시각화 후속 자리 | 미결 — non-blocking | O4 |

---

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 1 | ISSUES_OPEN | mode: HOLD_SCOPE, 13 findings applied, 1 critical gap, 4 unresolved decisions |
| Codex Review | `/codex review` | Independent 2nd opinion | 1 | ISSUES_OPEN | 12 findings, 10 accepted, 2 rejected (stub-record misread) |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 0 | — | not run — required before implementing T5-T9 |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | not run — required before T6 (marker priority table missing) |
| DX Review | `/plan-devex-review` | Developer experience gaps | 0 | — | not run |

**CODEX:** codex-cli 0.153.3, high effort, read-only. Corrected the plan's stalest fact: the 80-column graph content width is 39 (not 56), so title budget is -2 columns, not 11 characters. Also caught three silent-loss wiring omissions (`graph.Nodes`/`rowsFromGraph` reconstruction, positional parser indexes, projection-to-renderer plumbing) and one recorded-policy violation (T8 putting a section-local key in the global footer).

**CROSS-MODEL:** Both reviewers independently reached the same conclusion on the plan's core self-contradiction — D4's "empty result means diverged" misclassifies reversed selection and same-commit selection, which would make the tool teach the exact error D4 was chosen to avoid. That is the highest-confidence finding in this review and is resolved by P4's 4-state contract. The reviewers also independently agreed the Inspector "two render paths" claim was wrong. Divergence: codex read `.taskmaster` stub records and concluded two of the plan's task citations were unsupported; those citations check out against the real task-6 subtasks, and the disagreement itself surfaced a duplicate-record defect now tracked in `TODOS.md`.

**VERDICT:** CEO CLEARED (scope held, no expansions) — eng review required before implementation.

**UNRESOLVED DECISIONS:**
- O1 (blocking) — Inspector header date row placement: 5 rows / replace the `FROM` tail / defer display to task 6.8. Recommended: 5 rows.
- O2 (blocking) — author date vs commit date: both only when they differ / author only / always both. Recommended: both only when they differ. Determines the `SplitN` field count, so it must be answered with O1.
- O3 (blocking) — what the highlight enables: display + summary only / range diff in Inspector / range-targeted actions. Recommended: display + summary only.
- O4 (non-blocking) — reserve room for the `A...B` + merge-base branch view: keep the marker vocabulary extensible / commit to one marker. Recommended: keep it extensible.
