# Graphkeeper 메인 화면 가독성·레이아웃 개선 계획

상태: 구현 및 `$review` 완료
브랜치: `develop`
작성일: 2026-07-31

## 목표

Graphkeeper의 메인 화면을 더 쉽게 스캔할 수 있도록 개선한다. 화면을 더
화려하게 만드는 작업이 아니라, Graph 중심의 정보 구조를 유지하면서 제목,
상태 표시, 여백, 높이, 오버플로우 규칙을 명확하게 만든다.

주 사용자는 저장소 관리자와 릴리즈 담당자이며, 개발자는 두 번째 주요 사용자다.
따라서 릴리즈 기준점(tag), stash 상태, Graph topology를 빠르게 식별하는 것을
우선하고, 기존 전문 사용자의 paging·search·key binding 흐름은 보존한다.

이번 작업의 핵심 문제는 다음 네 가지다.

1. Graph 행의 커밋 제목 영역이 고정 폭 20자로 제한되어 유용한 제목이 잘린다.
2. stash point와 tag point가 같은 Graph node의 색상만 바꾸어 구분이 어렵다.
3. Context와 popup의 제목·본문·footer 간 여백이 호출부의 빈 줄에 흩어져 있다.
4. Graph와 Local/Remote/Tags rail의 실제 바닥선이 맞지 않고, 좁은 화면에서
   항목이 조용히 잘려 사용자가 전체 항목을 확인했는지 알 수 없다.

## 현재 코드 근거

- `internal/app/view_shell.go:47-105`가 전체 화면과 Graph/right rail을 조합한다.
- `internal/app/view_graph.go:8-53`이 Graph page/header/row를 조합한다.
- `internal/app/graph_render.go:10-14,141-168`에 `graphTitleWidthTarget = 20`과
  제목·author 폭 계산이 있다.
- `internal/app/graph_render.go:190-271`이 stash/tag 상태를 Graph node 색상으로
  표시한다.
- `internal/app/graph_render_connectors.go:93-95`에 connector용 고정 prefix가
  별도로 존재한다.
- `internal/app/view_layout.go:234-327`이 title strip, frame, popup을 렌더링한다.
- `internal/app/view_layout.go:304-316`은 frame의 `height`를 body 높이처럼
  사용하므로 title strip까지 포함한 실제 outer height와 어긋날 수 있다.
- `internal/app/view_detail.go:12-47,280-323`이 Context 내용을 만들며, 호출부에
  빈 줄이 흩어져 있다.
- `internal/app/view_sections.go:17-68`은 active section만 cursor 기준으로
  스크롤하고 inactive section의 초과 항목은 표시하지 않는다.
- `internal/app/model.go:23-28`에는 Graph scroll은 있지만 Context detail 전용
  viewport 상태는 없다.
- `internal/app/theme.go`는 현재 Task 4.1 범위의 ANSI semantic style을 중앙화한다.

## 추천 방향

세 가지 디자인 요청과 높이·항목 누락 문제는 별개가 아니라 공통 레이아웃
계약이 없는 데서 생긴 하나의 문제다. 다음 방향을 추천한다.

### 선택지

| 선택지 | 내용 | 규모 | 판단 |
|---|---|---:|---|
| A. 공통 레이아웃 + 의미 기반 marker + viewport | 폭·높이·여백을 공통 계약으로 만들고 S/T marker와 오버플로우를 추가 | M | **추천** |
| B. Graph만 부분 수정 | 제목 폭을 늘리고 기존 node 색상만 조정 | S | Context·popup·높이 문제가 남음 |
| C. 고정 범례와 상세 패널 추가 | stash/tag 설명용 범례와 별도 상세 영역 추가 | L | 화면이 복잡해지고 현재 TUI 방향과 충돌 |

A를 추천한다. Graph topology를 보존하면서 실제 원인을 해결하고, 이후 popup과
Context를 개선할 때도 같은 규칙을 재사용할 수 있다.

## 범위

### 포함

- Graph 제목 폭을 고정 20자가 아닌 남은 폭 기반으로 계산한다.
  - commit, branch, graph topology, marker, date를 먼저 보존한다.
  - author는 제목보다 우선순위가 낮으므로 좁은 화면에서 먼저 제거한다.
  - 남은 폭을 제목에 할당한다.
  - 최소 폭 이하에서는 제목을 생략할 수 있지만 제목 슬롯에 `…`을 남긴다.
- branches와 Graph topology 사이에 고정 state 영역을 추가한다.
  - `S` = stash point
  - `T` = tag point
  - `S·T` = stash와 tag가 모두 있는 commit
  - topology node인 `*`는 모든 row에서 그대로 유지한다.
  - marker는 branches와 topology 사이의 별도 고정 5칸 `state` 컬럼에 표시한다.
  - 상태가 없는 일반 row는 상태 컬럼을 공백으로 둔다.
  - `S·T`를 좁은 화면에서 `S`나 `T` 하나로 축약하지 않는다.
- 일반 row, raw graph row, connector row, 빈 connector row, header가 같은 column
  budget helper를 사용하도록 한다.
- Graph frame과 Local/Remote/Tags 전체 rail이 동일한 실제 outer height를 갖도록
  수정한다.
- frame의 `height`를 title strip, title/body gap, body를 모두 포함하는 outer height로
  정의한다.
- Context, frame, popup의 title/body/footer 간격을 공통 spacing metric으로 관리한다.
- 항목이 viewport를 초과하면 조용히 잘리지 않도록 한다.
  - Graph: 기존 `page x-y/N`과 Graph paging을 유지한다.
  - Graph page 정보 줄 오른쪽에 `S stash · T tag` 축약 범례를
    표시한다. 좁은 폭에서는 row의 topology·identity·state보다 먼저 축약하거나
    숨긴다.
  - active Local/Remote/Tags: cursor window로 전체 항목을 탐색한다.
  - inactive rail: 보이지 않는 항목 수를 `… +N`으로 표시한다.
  - Context detail: `ctrl+u`/`ctrl+d`로 상세 내용을 스크롤할 수 있게 한다.
  - section이나 focus가 바뀌면 해당 viewport offset을 초기화한다.
- 폭·높이 경계, state 조합, ANSI profile, paging, overflow 테스트를 추가한다.
- 색상·레이아웃 정책 문서와 README의 marker 설명을 갱신한다.

### 제외

- Graph/Local/Remote/Tags key binding의 의미 변경
- Graph topology renderer 또는 두 번째 graph model 도입
- `view_alert.go`, `stash_popup.go`, `tagging.go`, `cherry_pick_view.go`,
  `hidden_hotkeys.go`의 직접 색상 생성 전체 이전
- 별도 영역을 차지하는 영구 범례 패널, onboarding/tutorial mode, 사용자 테마 설정
- popup 전체 구조나 action 문구 재작성
- web preview 또는 screenshot 기반 visual regression 체계
- 대칭성을 이유로 한 `internal/ui` 신규 패키지

## 재사용할 기존 코드

| 문제 | 기존 코드 | 방침 |
|---|---|---|
| Shell geometry | `layoutShellMargins`, `layoutShellBodySize`, `renderAppView` | 기존 layout helper 확장 |
| Frame/popup | `renderTitleStrip`, `renderFloatingTitleFrame`, `renderFloatingTitlePopup` | 공통 spacing·outer height 계약 추가 |
| Graph 폭 | `renderGraphHeader`, `renderGraphTitleWithAuthor`, `fitVisibleWidth` | header와 row가 동일 helper 사용 |
| stash/tag 데이터 | `stashesForCommit`, `row.Commit.Tags` | 새 Git 조회나 상태 필드 추가 없음 |
| Rail 목록 | `sectionCursor`, `renderSectionContent` | 기존 cursor를 활용하고 작은 viewport offset만 추가 |
| ANSI style | `theme.go`, reverse/bold style | 색상보다 visible marker를 우선 |
| 테스트 | `internal/app/model_test.go`, graph 관련 테스트 | 별도 테스트 harness를 만들지 않음 |

## 디자인 검토

### 목표 상태

```text
현재 상태                         이번 계획                         목표 상태
고정 20자 제목             --->   남은 폭 기반 제목              ---> Graph topology,
색상 중심 stash/tag               S/T/S·T visible state column        branch, stash, tag,
호출부마다 다른 빈 줄             공통 spacing metric                 제목과 decision state가
Graph/rail 바닥선 불일치          동일 outer height                  한 번에 스캔되는 화면
초과 항목 조용한 누락             viewport + `… +N`
```

### 정보 우선순위

```text
Graph row
  1. topology + commit identity
  2. branch/ref + state column
  3. date + 가능한 가장 긴 subject

Context / Popup
  1. title
  2. 측정된 title/body gap
  3. 주요 내용
  4. section gap
  5. footer·복구 안내·overflow indicator
```

### 메인 화면 높이 계약

```text
main row outer height = H
  Graph frame       = title + gap + body = H
  Right rail        = H
    Local frame     = 할당 높이
    Remote frame    = 할당 높이
    Tags frame      = 할당 높이
    rail separator  = 남은 행을 정확히 소비

검증:
  lipgloss.Height(Graph frame) == lipgloss.Height(Right rail)
  Graph bottom border row == Tags bottom border row
```

현재 `renderFloatingTitleFrame`은 입력 높이를 body에 그대로 사용해 title strip이
추가될 수 있다. 구현 시 입력 높이는 outer height로 고정하고, title strip·gap을
먼저 차감한 뒤 body를 계산해야 한다. `graphContentHeightForModel`과
`graphPageSizeForRows`도 같은 차감 규칙을 사용해야 한다.

### 항목 누락 방지 계약

```text
items > viewport
       │
       ├─ active section  ─▶ cursor window + j/k + ctrl+u/d + `… +N`
       ├─ inactive rail   ─▶ 보이는 항목 + `… +N`
       ├─ Graph           ─▶ page x-y/N + 기존 Graph paging
       └─ Context detail  ─▶ detail viewport + ctrl+u/d + `… +N`
```

여기서 “누락없이 보여준다”는 모든 항목이 한 화면에 동시에 들어간다는 뜻이
아니다. 화면 밖 항목이 있다는 사실을 표시하고, 사용자가 정해진 키로 마지막
항목까지 도달할 수 있어야 한다는 뜻이다. 잘린 목록을 완전한 목록처럼 보이게
두지 않는다.

### 디자인 점수

| 항목 | 점수 | 10점이 되기 위해 필요한 것 |
|---|---:|---|
| 정보 구조 | 8/10 | Graph title/marker 우선순위 고정 |
| 상태 표현 | 7/10 | combined marker·overflow·low height 계약 |
| 사용자 흐름 | 8/10 | 밀도 높은 Graph에서 scan 비용 감소 |
| AI-style 위험 | 9/10 | 영구 범례·장식 카드 추가 금지 |
| 디자인 시스템 | 8/10 | spacing token 추가 |
| 반응형/접근성 | 7/10 | terminal 폭·높이별 우선순위 테스트 |

## 엔지니어링 검토

### 의존성 구조

```text
repoStatus + stashByBase
          │
          ├─ graphRows() ─▶ graphRow
          │                    ├─ graphMarkerState()
          │                    ├─ graphColumnBudget()
          │                    └─ renderGraphLine()
          │
          └─ renderAppView()
                ├─ layoutMetrics()
                ├─ renderFloatingTitleFrame()
                ├─ renderContextContent()
                └─ renderRightRail()
```

새 네트워크, 파일, Git mutation, persistence, async state는 없다. marker와
viewport는 기존 model 데이터에서 계산하는 화면 상태다.

### Graph 폭 계약

```text
rowWidth
  ├─ commit       8
  ├─ branch      14
  ├─ state        5 visible cells: `     ` / `S    ` / `T    ` / `S·T  `
  ├─ topology    기존 graph column
  ├─ date         7
  ├─ separator    고정 폭
  └─ title        나머지 전체

좁은 폭:
  1. author 제거
  2. state 5칸 유지 + topology의 `*` 유지
  3. ANSI-aware title truncate
  4. 최소 폭 아래에서는 title 생략 + `…`
  5. topology·commit identity·state marker는 유지
```

일반 row와 raw/connector/header 모두 동일한 column helper를 써야 한다. 그렇지
않으면 raw graph나 connector에서 marker만큼 날짜와 제목의 시작 위치가 밀린다.

### 에러·복구 레지스트리

렌더링 전용 변경이므로 외부 예외·retry 경로는 없다. 대신 다음 shadow path를
명시한다.

| 경로 | 문제 | 복구 | 사용자 표시 | 테스트 |
|---|---|---|---|---|
| marker 계산 | nil/empty stash map | `S` 없음 | 정상 Graph row | 필수 |
| marker 계산 | empty tag list | `T` 없음 | 정상 Graph row | 필수 |
| combined marker | stash+tag 동시 존재 | 항상 `S·T` | 두 상태 모두 확인 가능 | 필수 |
| 폭 계산 | 최소 폭 이하 | author 제거·title `…` | topology/hash/marker 유지 | 필수 |
| 높이 계산 | low-height terminal | gap 축소·body clip | title/footer 우선 | 필수 |
| overflow | viewport 초과 | offset + `… +N` | 전체 항목 탐색 가능 | 필수 |
| ANSI profile | style 제거 | visible token 유지 | 색상 없이도 의미 유지 | 필수 |
| conflict/connector | 가상 row에 marker 적용 | marker slot 비움 | 기존 conflict/topology 유지 | 필수 |

### 성능·유지보수

- marker 계산은 row당 O(1)이다.
- viewport 계산은 기존 list 길이에 대한 단순 clamp다.
- 새 cache나 feature flag는 필요하지 않다.
- app 상태와 Lip Gloss에 의존하므로 helper는 `internal/app`에 둔다.

## 테스트 계획

```text
코드 경로                                  사용자 흐름
graphMarkerState()                         terminal resize
  ├─ none                         [unit]    ├─ wide Graph row       [render]
  ├─ stash                       [unit]    ├─ medium Graph row     [render]
  ├─ tag                         [unit]    ├─ narrow Graph row     [render]
  └─ stash + tag                 [unit]    └─ Context/popup scan    [render]

graphColumnBudget()/title width              main-row geometry
  ├─ author 경계                  [unit]    ├─ Graph bottom == Tags bottom [render]
  ├─ minimum 폭                  [unit]    ├─ low-height rail split        [render]
  ├─ Unicode/ANSI title           [unit]    └─ page size == rendered rows   [unit]
  └─ raw/connector/header offset  [render]

overflow viewport                            color profile
  ├─ inactive `… +N`              [render]   ├─ Ascii/NO_COLOR       [matrix]
  ├─ active last item 도달        [update]   ├─ ANSI/ANSI256         [matrix]
  └─ Context ctrl+u/d              [update]   └─ TrueColor             [matrix]
```

필수 테스트:

- marker 네 가지 상태와 `S·T` 비축약을 검증한다.
- author가 들어가는 경계, author가 빠지는 경계, 최소 폭, 긴 Unicode 제목을
  검증한다.
- 일반 row/raw row/connector/empty connector/header의 column offset을 비교한다.
- Graph frame과 right rail의 실제 `lipgloss.Height`와 bottom border row를
  wide/medium/low height에서 비교한다.
- Graph page size가 실제 표시 가능한 row/connector 수와 일치하는지 검증한다.
- Context, Local, Remote, Tags가 `… +N`을 표시하고 마지막 항목까지 도달하는지
  검증한다.
- title/body/footer spacing이 Context, hidden hotkeys, confirm, search, input
  popup에서 동일한 규칙을 따르는지 검증한다.
- Ascii, ANSI, ANSI256, TrueColor에서 visible marker를 검증한다.
- `NO_COLOR=1`은 현재 startup에서 직접 해석하지 않으므로, “환경변수만으로
  색상이 꺼진다”고 주장하지 않는다. 실제 subprocess smoke test로 현재
  동작을 확인하거나, 별도 `NO_COLOR` startup 지원 task로 분리한다.
- 기존 graph paging, search, stash/tag detail, conflict rendering, key handling
  회귀 테스트를 유지한다.

### 사용자 성공 기준

구현 완료는 단위·렌더링 테스트 통과만으로 판단하지 않고, 다음 화면 기준을 모두
만족해야 한다.

- 일반적인 터미널 폭에서 Graph 제목이 기존 고정 폭보다 더 많은 내용을 보여준다.
- Graph와 Local/Remote/Tags rail의 bottom border가 같은 행에 놓인다.
- 색상이 꺼진 환경에서도 `S`, `T`, `S·T`로 stash/tag 상태를 구분할 수 있다.
- 좁은 화면에서도 `S·T`가 `S` 또는 `T`로 잘못 축약되지 않는다.
- 항목이 화면 밖으로 밀릴 때 paging, scroll, `… +N` 중 하나로 계속 접근할 수 있다.
- `?`를 열지 않아도 Graph page 줄에서 상태 범례를 확인할 수 있다.
- Graph navigation, search, conflict rendering, 기존 key binding의 의미가 변하지
  않는다.

수동 QA에서는 wide/medium/narrow terminal과 Ascii/ANSI/ANSI256/TrueColor profile을
각각 확인하고, 위 기준의 결과를 T7 검증 기록에 남긴다.

## DX 검토

Graphkeeper는 repository maintainer/release owner와 Git 형상을 배우는 개발자가
사용하는 CLI/TUI다.

| 단계 | 현재 마찰 | 개선 |
|---|---|---|
| 첫 실행 | Graph row가 dense함 | 제목이 남은 폭을 사용 |
| 형상 확인 | stash/tag 색상 의미가 약함 | topology `*`와 S/T/S·T를 함께 확인 |
| 상세 확인 | Context가 길면 일부가 잘림 | viewport와 `… +N` 제공 |
| 좁은 터미널 | bottom drift·silent clipping | outer height·overflow 계약 |
| 기여 | 폭 계산이 여러 곳에 분산 | 공통 helper와 테스트 계약 |

이 계획은 설치나 명령어를 바꾸지 않고 첫 화면의 인지 비용과 contributor의
예측 가능성을 개선한다. community/telemetry/release measurement는 별도 범위다.

## 구현 순서

1. `view_layout.go`에 spacing metric과 outer-height accounting을 만든다.
2. Graph header/normal/raw/connector row가 공통 column budget을 사용하게 한다.
3. author 제거 우선순위와 title `…` fallback을 구현한다.
4. 5칸 state column과 conflict/connector 비적용 규칙을 구현한다.
5. Graph/right rail의 실제 outer height와 bottom border를 일치시킨다.
6. Context detail과 rail overflow viewport, `… +N`, `ctrl+u/d`를 구현한다.
7. Context·popup의 여백을 공통 metric으로 치환한다.
8. 테스트와 문서를 갱신한 뒤 전체 검증을 수행한다.

모든 변경이 `internal/app`과 동일 테스트 파일에 수렴하므로 병렬 worktree보다
순차 구현이 적합하다.

## 실패 모드 레지스트리

| 경로 | 실패 | 테스트 | 사용자 결과 |
|---|---|---|---|
| Graph 폭 | title 폭 음수 | 있음 | `…` 또는 제목 생략, identity 유지 |
| marker | S·T 중 하나 유실 | 있음 | topology `*` 유지 + 5칸 `state` 유지 |
| raw/connector | column offset drift | 있음 | topology/date 정렬 유지 |
| frame 높이 | Graph/Tags bottom 불일치 | 있음 | 두 bottom border 동일 행 |
| paging | 실제 row 수와 page size 불일치 | 있음 | 마지막 row까지 이동 가능 |
| overflow | 항목 silent clipping | 있음 | `… +N`과 scroll로 전체 접근 |
| ANSI | style 제거로 의미 유실 | 있음 | visible marker 유지 |
| conflict | 가상 row marker 오염 | 있음 | 기존 conflict 표시 유지 |

Critical silent failure는 허용하지 않는다.

## 롤아웃·롤백

- 현재 Task 4.1 ANSI 작업을 먼저 깨끗한 경계에서 커밋한다.
- 이후 공통 layout → Graph marker/title → Graph/rail height → overflow/popup
  순서의 단계별 게이트를 통과하며 구현한다.
- migration, feature flag, compatibility shim은 필요하지 않다.
- 각 단계는 독립적으로 검토·검증 가능한 커밋으로 유지한다.
- 좁은 terminal 또는 색상 profile 회귀가 발견되면 해당 커밋을 `git revert`한다.
- 수동 확인 폭: 80/100/120/160 columns, low-height terminal, empty graph,
  conflict, stash/tag/both, Context overflow, `NO_COLOR=1`.

## 현재 backlog 기준 추천 우선순위

1. Task 4.1 터미널 색상 호환성 작업 완료 및 커밋
2. 본 계획의 메인 화면 가독성·레이아웃 구현
3. Task 3.1 로컬 로그 권한·비활성화·테스트
4. 커밋 전체 메시지와 diff inspect 기능
5. 표준 scripts, Linux 실행 검증, GitHub Release 정비

## 구현 작업

- [x] **T1 (P1, human 약 3시간 / CC 약 25분)** — Graph column budget을 header,
  normal/raw/connector row가 공유하도록 구현
  - 파일: `internal/app/view_graph.go`, `internal/app/graph_render.go`,
    `internal/app/graph_render_connectors.go`, `internal/app/view_layout.go`
  - 검증: 폭 경계·raw/connector 정렬·긴 제목 테스트
- [x] **T2 (P1, human 약 2시간 / CC 약 15분)** — topology `*`를 보존한 S/T/S·T 상태 컬럼 구현
  - 파일: `internal/app/graph_render.go`, `internal/app/theme.go`, 관련 테스트
  - 검증: 네 marker 상태와 color profile matrix
- [x] **T3 (P1, human 약 2시간 / CC 약 15분)** — Graph와 Tags bottom이 맞는
  outer-height/frame spacing 계약 구현
  - 파일: `internal/app/view_layout.go`, `internal/app/view_shell.go`, 관련 테스트
  - 검증: 실제 `lipgloss.Height`와 border row 비교
- [x] **T4 (P1, human 약 3시간 / CC 약 25분)** — rail·Context 항목을 silent
  clipping 없이 탐색하는 overflow viewport 구현
  - 파일: `internal/app/model.go`, navigation/key handling, `view_sections.go`,
    `view_detail.go`, 관련 테스트
  - 검증: inactive `… +N`, active 마지막 항목, Context `ctrl+u/d`
- [x] **T5 (P2, human 약 2시간 / CC 약 15분)** — Context·popup 공통 spacing metric 적용
  - 파일: `internal/app/view_layout.go`, `view_detail.go`, popup renderer, 테스트
  - 검증: title/body/footer 줄 수와 low-height 우선순위
- [x] **T6 (P2, human 약 1시간 / CC 약 10분)** — 색상/layout 정책과 marker 문서 갱신
  - 파일: `docs/highlighting-color-map.md`, `README.md`
  - 검증: 실제 출력·hotkey·marker 설명 일치
- [x] **T7 (P2, human 약 1시간 / CC 약 10분)** — 터미널 크기·profile QA matrix 수행
  - 검증: `go test ./...`, `scripts/check`, `scripts/build`, `git diff --check`

## 리뷰 요약

### CEO/전략

- Graphkeeper의 핵심 제품 가치는 Graph를 읽고 판단하는 것이므로, 이 작업은
  제품의 중심 surface를 직접 개선한다.
- 범위를 Git cockpit 전체로 넓히지 않고 Graph/Context/Popup의 가독성에 제한한다.
- 별도 범례 패널이나 새 화면을 추가하지 않고 Graph page 줄에만 축약 범례를 두어
  현재 산업적·간결한 방향을 유지한다.

### 디자인

- `DESIGN.md`의 compact, restrained, industrial TUI 방향과 일치한다.
- mockup binary가 없어 텍스트 layout contract, width/height matrix, renderer test를
  기준으로 검토했다.
- 색상만으로 의미를 전달하지 않고 branches와 topology 사이에 S/T/S·T visible state column을 추가한다.

### 엔지니어링

- 독립 Codex 리뷰에서 다음 위험을 확인했고 계획에 반영했다.
  - 최소 폭 계약 부재
  - topology `*`와 상태 컬럼 `S·T`의 폭·정렬 계약
  - raw/connector 경로의 별도 column 계산
  - spacing 변경과 paging 불일치
  - caller별 빈 줄 미정리
  - `NO_COLOR`를 실제 startup에서 해석하지 않는 점
  - conflict/connector row에 marker가 적용될 위험
- 새 외부 integration, persistence, Git mutation은 없다.

### DX

- maintainer와 Git을 배우는 개발자 모두에게 Graph 첫 스캔 비용을 낮춘다.
- 명령어·설치·key binding은 바꾸지 않는다.
- contributor가 width/height/overflow를 테스트 계약으로 이해할 수 있게 한다.

## GSTACK REVIEW REPORT

| 리뷰 | 트리거 | 목적 | 상태 | 결과 |
|---|---|---|---|---|
| CEO Review | `$autoplan` | 범위·전략 | CLEAR | 공통 레이아웃 기반 추천 |
| Codex Review | `$autoplan` | 독립 아키텍처 검토 | CLEAR | 8개 위험 검토, 계획에 반영 |
| Eng Review | `$autoplan` | 아키텍처·테스트 | CLEAR | 폭·높이·overflow 계약 필수 |
| Design Review | `$autoplan` | UI/UX | CLEAR | mockup 도구 없음, renderer 계약으로 대체 |
| DX Review | `$autoplan` | CLI/TUI 사용성 | CLEAR | 첫 화면 scanability 개선 |

VERDICT: CEO + Design + Eng + DX 검토와 구현 및 `$review`를 완료했다. UI 변경은
커밋 가능한 상태다.

### 구현 후 `$review` 결과

- 결과: **No issues found**
- 검증: `go test ./...`, `go test -race ./internal/app`, `scripts/check`,
  `scripts/build`, `git diff --check`
- 이번 보완 검증: hidden hotkey 팝업의 ANSI semantic color와 Global 중복 항목 제거

NO UNRESOLVED DECISIONS
