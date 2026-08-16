<!-- /autoplan restore point: /Users/hrk/.gstack/projects/hrllk-graphkeeper/develop-autoplan-restore-20260803-165252.md -->

# Graph 커밋 상세 inspector 계획

상태: 구현 진행 중
작성일: 2026-08-02

## 배경

Reddit 피드백 작업 목록의 “선택한 커밋의 전체 commit message와 diff를
읽기 전용 preview로 빠르게 열기” 항목을 Graph의 `enter` 동작으로 연결한다.
현재 `enter`와 기본 inspector는 이미 구현되어 있지만, 기존 계획에는 이
기능의 화면 계약이 별도로 기록되어 있지 않다.

관련 문서:

- `docs/reddit-feedback-task-list.md` 1장 Graph 정보 밀도
- `docs/20260801-0003-overlay-graph-density-improvement-plan.md`의 Graph·overlay
  밀도 조정 범위

## 목표 화면

선택된 실제 commit에서 기존 Graph 화면을 완전히 가리는 독립 Inspector screen을
연다. Inspector가 닫히면 기존 Graph 화면과 선택 위치로 돌아간다.

```text
+-------------------------------------------------------------+
| Commit <full hash>  <full message>                          |
+---------------------------+---------------------------------+
| Changed files       | from code         | to code           |
| M path/to/file      | old line          | new line          |
| A path/to/new       |                   | new line          |
| D path/to/old       | old line          |                  |
+---------------------------+---------------------------------+
| j/k select  tab/enter open diff  ctrl-u/d scroll  q close   |
+-------------------------------------------------------------+
```

- 왼쪽 pane은 변경 파일을 상태·경로와 함께 표시한다.
- 가운데 pane은 선택 파일의 `from` 코드를, 오른쪽 pane은 `to` 코드를 표시한다.
  추가 줄은 `from`이 비어 있고, 삭제 줄은 `to`가 비어 있으며, 수정 줄은 같은
  행 쌍으로 정렬한다. 실제 Git diff의 hunk header와 rename 경로도 보존한다.
- 제목 영역에는 subject만이 아니라 전체 commit message를 확인할 수 있는
  상세 영역 또는 스크롤 경로를 제공한다.
- message header는 기본적으로 hash·subject·author와 body 요약만 보여주며,
  별도 `m` 토글로 고정 높이 message viewport를 열고 전체 body를 스크롤한다.
  message viewport를 열고 닫아도 선택 파일, from/to 위치와 pane focus는
  보존한다.
- 폭별 responsive mode를 명시한다. 넓은 폭에서는 3열을 유지하고, 중간 폭에서는
  Changed files pane을 접을 수 있게 하며, 매우 좁은 폭에서는 files → from/to를
  순차 전환한다. 모든 모드에서 선택 파일·상태·현재 focus와 복귀 경로를 상단에
  고정하며, 의미 있는 내용을 조용히 잘라내지 않는다.

## 현재 구현과 갭

- `internal/app/key_handling_browse.go`는 Graph `enter`에서 inspector를 연다.
- `internal/git/repo_exec.go`의 `InspectCommit`은 metadata와 변경 파일을
  읽고, `CommitDiff`는 선택 파일의 patch를 읽는다.
- `internal/app/commit_inspector.go`는 현재 파일 목록과 diff를 세로로
  렌더링하며, 전체 commit message와 좌우 pane 계약은 아직 없다.

따라서 기존 구현을 완료로 표시하지 않고 이 문서를 후속 subtask의 기준으로
삼는다.

## 구현 단위

1. commit metadata에 subject·body를 포함한 전체 message와 parent/from·to 정보를
   보존하고 message viewport를 제공한다.
2. Phase 1에서는 `CommitInspectorReader` application port와 `git.Repo` adapter를
   분리한다. app renderer/state는 Git concrete type이나 subprocess argument를
   직접 참조하지 않고 metadata/file tree/diff window contract만 소비한다.
3. Inspector screen을 좌측 Changed files pane과 가운데/우측 from/to code pane의
   공통 폭 계약으로 구현한다.
4. 파일 선택, pane 전환, diff scroll, commit message scroll, `q`/`esc` close를
   유지하고 low-width/low-height fallback을 추가한다.
5. root commit, rename, binary, 빈 변경 목록, 긴 path/message/diff를 테스트한다.
6. `README.md`, `docs/decisions.md`, Task Master 상태를 구현 결과와 동기화한다.

## Phase plan

### Phase 1: Inspector screen + structured diff contract

- `CommitInspectorReader` port와 `git.Repo` adapter
- subject/body metadata, parent selection metadata
- hunk/old-new line/pair model
- Changed files tree와 독립 screen state
- fake reader contract tests와 root/rename/binary parser tests
- temporary Git repository integration fixture와 pure pairing fixture를 함께 사용한다.

### Phase 2: viewport + bounded cache

- bounded streaming reader/parser로 `git show` 출력을 소비하고 전체 diff를
  `gitRaw`/`strings.Split`로 materialize하지 않는다.
- diff window loading은 요청 window 주변의 line/byte cap을 적용하고, cap 초과 시
  partial state와 continuation hint를 반환한다.
- 각 diff window request는 취소 가능한 context를 가지며, 새 request·파일 전환·
  Inspector 종료 시 이전 `git show` 프로세스와 parser를 취소한다. 취소된 결과는
  사용자 오류가 아니라 stale/canceled 결과로 폐기한다.
- file tree/diff/message viewport 분리
- bounded cache와 repository/request epoch stale result 폐기

### Phase 3: semantic style + fallback

- A/M/D 및 +/- semantic style
- modified pair word-level spans
- highlighter registry interface, fake highlighter, plain/NO_COLOR fallback
- terminal matrix, diagnostics, README/decision synchronization

테스트는 parser 단위 테스트만으로 끝내지 않는다. 실제 temporary Git repository를
사용해 `InspectCommit`/reader adapter의 argument와 root·rename·`--` 경계를 검증하고,
순수 fixture로 malformed hunk·삭제/추가 run·word pairing을 검증한다. app에는
fake `CommitInspectorReader`를 주입해 `Enter → loading → file select → diff
ready → retry/error → Esc` 상태 전이를 순서대로 검증한다.

### Side-by-side pairing 규칙

unified diff의 연속된 삭제 줄과 추가 줄을 하나의 변경 run으로 묶는다. 같은
run에서는 old/new 줄을 원래 순서대로 deterministic하게 pair하고, 한쪽 줄이
남으면 반대쪽을 빈 칸으로 채운다. context 줄은 양쪽에 모두 복제한다.
pairing은 의미 추론이나 Tree-sitter parse 결과에 의존하지 않으며, word-level
diff는 이미 만들어진 modified pair 안에서만 계산한다.

### Style precedence

diff semantic style이 syntax style보다 우선한다. `A/M/D`, `+/-`, from/to 구분은
색상 없이도 표시되어야 하며, syntax highlighting은 foreground·underline 같은
보조 표현으로만 적용한다. `NO_COLOR`, 미지원 언어, parse 실패에서는 syntax를
제거하고 semantic marker와 plain code를 유지한다.

## 완료 기준

- Graph에서 `enter`로 선택 commit의 inspector가 열린다.
- 변경 파일은 왼쪽, 선택 파일의 from/to diff는 오른쪽에 표시된다.
- 전체 commit message와 긴 diff를 정해진 scroll 키로 끝까지 읽을 수 있다.
- `q`와 `esc`의 기존 overlay close/back 계약 및 Graph paging/search/key binding을
  변경하지 않는다.
- `go test ./...`, `scripts/check`와 좁은 터미널 회귀 테스트가 통과한다.

## Inspector 키 계약

| Key | Files tree | from/to code | Loading/error |
|---|---|---|---|
| `j/k`, `up/down` | 파일·directory 이동 | diff row/hunk 이동 | loading 중 새 요청 차단 |
| `h/l` | directory collapse/expand | horizontal offset 이동 | error 상태에서는 no-op |
| `Tab` | focus를 다음 pane으로 이동 | focus를 다음 pane으로 이동 | metadata가 준비된 뒤에만 |
| `Ctrl+U/D` | tree page scroll | diff page scroll | 현재 focus viewport만 변경 |
| `m` | message viewport 열기/닫기 | message viewport 열기/닫기 | metadata가 준비된 뒤에만 |
| `r` | file list 재조회 | 선택 diff 재조회 | 현재 오류/partial 결과 retry |
| `q` | Inspector close | Inspector close | close/cancel |
| `Esc` | Graph로 복귀 | Files tree로 back 또는 Graph 복귀 | error도 동일 |

Inspector가 열린 동안 Graph paging, section switching, mutation hotkey는 소비하고
Graph로 전파하지 않는다. loading 중 `j/k`, `Tab`, `r`의 중복 async 요청은 만들지
않으며, 완료된 결과는 현재 commit/file/selection token과 repository epoch를
검사한 뒤에만 상태에 반영한다.

Phase 1의 reader 요청은 `InspectorRequest{Commit, Parent, FileID, Window,
RepositoryEpoch, RequestID}`를 함께 전달한다. `RequestID`는 같은 repository
epoch 안에서 파일을 빠르게 바꿨다가 돌아오는 경우까지 구분하고, epoch는 Git
mutation/refresh 이후의 결과를 폐기하는 데 사용한다.

Reader port와 DTO는 app이 소유하는 `internal/app/commit_inspector_contract.go`
같은 경계에 둔다. `internal/git`은 Git output을 이 contract로 변환하는 adapter로
남기며, 동일한 의미의 `CommitInspection`/`CommitDiff` 타입을 app과 git 양쪽에
복제하지 않는다.

## CEO 리뷰 확정 사항

2026-08-02 `/plan-ceo-review` 결과를 반영한다.

- `overlayPopup` 확장이 아니라 독립 `Inspector` screen state로 구현한다.
- Git 오류는 metadata/file/diff pane 단위로 유지하고 retry 경로를 제공한다.
- 저장소 경로·diff·파일 내용은 untrusted input으로 취급한다. Git argument
  array와 `--` separator, byte/line limit, parser cancellation, plain fallback을
  적용한다.
- raw unified diff 문자열을 renderer가 직접 해석하지 않는다. hunk, old/new
  line, line kind, line number, pair identity를 포함한 구조화된 contract를 둔다.
- file tree viewport와 diff viewport를 분리하고, bounded cache는
  `commit/parent/file/window`와 repository epoch를 기준으로 stale 결과를 폐기한다.
- Tree-sitter 실제 grammar/query 번들은 P2 후속 subtask로 연기한다. 이번 작업은
  highlighter registry interface와 fake/plain fallback만 포함한다.
- merge commit parent 선택 UI는 P2 후속 subtask로 연기한다. MVP는 first-parent
  기준을 화면에 명시한다.
- responsive breakpoint는 renderer contract와 snapshot/matrix test로 고정한다:
  3열 wide mode, 접을 수 있는 files pane의 compact mode, files/from-to 순차
  전환의 narrow mode. 각 mode는 path/status/selected-file identity와 `from/to`
  방향을 색상 없이도 보존해야 한다.
- focus 표현도 renderer contract로 고정한다. active pane은 제목/경계 변화,
  selected file은 고정 cursor marker, focused diff row는 line marker 또는
  reverse/underline을 사용하며, 색상은 이 의미를 대체하지 않는다.
- header/footer에는 `A added · M modified · D deleted`와 `+ added line ·
  - removed line · space context` legend를 제공한다. 파일 상태 marker와 line
  kind marker는 서로 다른 위치/문자열을 사용하고, from/to에 대응 내용이 없을
  때는 빈 문자열 대신 `—` 또는 `not present` placeholder를 표시한다.
- Changed files는 directory node와 file node를 구분하는 tree contract를 둔다.
  directory는 collapse/expand 상태를 비색상 marker로 보여주고, rename은
  `old/path → new/path`로 표시한다. 좁은 mode에서도 선택 file의 repository-
  relative full path는 header/status line에 고정해 동일 basename의 구분을
  보존한다.
- from/to code pane은 양쪽 모두 `kind marker | line number | code`의 고정 열을
  사용한다. line-number 폭은 현재 window의 최대 자릿수로 계산하고, 반대편에
  내용이 없는 row도 marker·번호 열과 `—` placeholder를 유지해 code 시작 위치와
  pair alignment가 흔들리지 않게 한다.

### Information hierarchy

사용자의 5초 scan 순서는 `commit identity → selected file → from/to change`로
고정한다. Header는 hash·subject·author와 현재 repository-relative path를 가장
먼저 식별하게 하고, Changed files는 선택 대상을 즉시 바꾸는 navigation surface,
from/to는 선택 파일의 변경을 읽는 primary work surface로 구분한다. Wide mode에서는
이 세 영역을 동시에 보여주되 from/to code의 대비와 active pane을 가장 강하게
표현하고, compact/narrow mode에서는 identity와 selected-file context를 고정한
채로 files navigation과 code comparison을 순차 전환한다. Message body와 legend,
retry hint는 primary diff를 가리지 않는 secondary layer로 둔다.

저높이 우선순위도 동일하게 고정한다. `(40,12)`와 `(60,12)`에서는 `COMMIT`/parent
또는 root context, selected repository-relative path, `FROM`/`TO` 방향, 최소 1행
이상의 diff viewport, `q`/`Esc`를 유지한다. full message body는 자동으로 펼치지 않고
`m`으로 여는 secondary viewport로 남기며, 공간이 부족하면 legend·subject 길이·
footer 설명 순서로 줄인다. selected path와 from/to 방향은 어떤 높이에서도 생략하지
않는다.
- privacy-safe local diagnostics만 기록하며 commit hash, path, message, diff
  content는 로그에 넣지 않는다.

## Autoplan corrective contract

현재 구현은 `shellOverlayStack → overlayPopup → renderCommitInspectorPopup`의
borderless vertical diff 경로에 머물러 있다. 따라서 아래 항목은 “후속 polish”가
아니라 Inspector MVP의 launch gate다.

### Screen boundary and identity band

- `renderAppView`는 `commitInspectorOpen`일 때 Graph shell을 먼저 대체하고,
  Inspector 전용 body bounds와 공통 border를 렌더링한다. Inspector는
  `shellOverlayStack`에서 제거하고 `overlayPopup`은 기존 modal에만 남긴다.
- Header는 `COMMIT <full hash>`, subject, author, `PARENT <short hash>` 또는
  `ROOT COMMIT`, 현재 repository-relative selected path, `READ-ONLY HISTORY`
  context를 고정한다. code pane의 방향은 항상 `FROM <parent>`와 `TO <commit>`로
  표시한다.
- 화면을 닫으면 Graph의 commit cursor와 scroll을 그대로 복원한다. renderer test는
  Inspector open 상태에서 Graph body가 출력되지 않고 Inspector frame이 body bounds를
  채우는 것을 검증한다.

### Message contract

`CommitInspection`은 `MessageSubject`, `MessageBody`, `MessageRaw` 또는 동등한
구조를 제공한다. Git metadata 요청은 subject만 읽지 않고 전체 message를 보존하며,
빈 줄·trailers·마지막 newline·embedded newline을 유지한다. invalid UTF-8은
replacement policy를 적용하고 terminal control sequence는 표시용 text에서 제거한다.
`m`은 fixed-height message viewport를 열고 닫으며 현재 file/diff focus를 변경하지
않는다. multiline body, trailers, empty body, long message를 fixture로 검증한다.

### Pane state matrix

| Surface | loading | empty | ready | partial | error/canceled |
|---|---|---|---|---|---|
| metadata/header | identity placeholder + cancel | `COMMIT` context + no metadata | full identity band | identity 유지 + partial hint | cause + `r` retry + `Esc` |
| Changed files | tree skeleton, Graph context 유지 | `No changed files` + `q/Esc` | tree navigation | loaded files + continuation hint | files pane error + retry |
| message | subject/author placeholder | `No commit message body` | summary/full viewport | body + truncation hint | message error, diff focus 보존 |
| from/to diff | pane labels + selected path 유지 | `No textual changes` 또는 binary state | paired rows | loaded rows + `Ctrl+D` continuation | diff pane error + retry, files 사용 가능 |

Empty state는 정상 결과와 실패를 구분할 수 있어야 한다. Changed files가 비어 있으면
`No changed files`와 commit identity를 유지하고 `Esc back`/`q close`를 제공한다.
선택 파일에 텍스트 변경이 없으면 `No textual changes`와 현재 path를 표시하며, binary
파일은 `Binary file`, submodule은 `Submodule change`로 구분한다. files pane에서
`j/k`로 다음 파일을 선택할 수 있는 경우에만 그 hint를 추가하고, 불가능한 경우에는
불필요한 안내를 표시하지 않는다. 이 상태들은 오류가 아니므로 error 색상이나 retry
문구를 사용하지 않는다.

`metadata/files/diff/message`는 각각 `idle/loading/ready/partial/error/canceled`
상태를 갖는다. 한 pane의 실패가 다른 ready pane을 가리지 않으며, canceled는
사용자 오류가 아니라 stale request 폐기로 표시하지 않는다.

### Responsive acceptance matrix

터미널 폭/높이는 `(40,12)`, `(40,20)`, `(40,30)`, `(60,12)`, `(60,20)`,
`(60,30)`, `(80,12)`, `(80,20)`, `(80,30)`을 최소 검증 집합으로 한다.
80열은 files/from/to wide mode, 60열은 접을 수 있는 files pane compact mode,
40열은 files와 from/to를 순차 전환하는 narrow mode다. 각 mode에서 full hash가
잘려도 selected path, `FROM/TO` 방향, 현재 pane, retry/close action은 사라지지
않는다. resize 중 loading/message viewport/diff partial 상태도 snapshot 대상이다.

### Terminal accessibility invariants

색상은 어떤 의미도 단독으로 전달하지 않는다. selected file/row는 cursor marker와
visible label을 함께 사용하고, active pane은 pane title과 border 또는 고정 focus
label을 함께 사용한다. line kind는 `+`/`-`/`space`, direction은 `FROM`/`TO`,
error는 원인 text와 `r retry` 또는 `Esc back` action을 함께 표시한다. 이 규칙은
ANSI, ANSI256, TrueColor, 밝은 terminal palette, `NO_COLOR`에서 동일해야 하며,
snapshot은 색상 제거 후에도 각 invariant가 남아 있는지 검사한다.

### Input and semantic acceptance

- `Tab/Shift+Tab`은 files → from/to → message를 순환한다.
- narrow mode에서는 한 번에 하나의 pane만 렌더링하고 별도 pane toggle key를
  추가하지 않는다. `Tab/Shift+Tab`이 files → from/to → message를 순환하며,
  header에 `FILES`, `FROM/TO`, `MESSAGE` 중 현재 pane을 짧게 표시한다.
- `q`는 항상 Inspector를 닫고, `Esc`는 diff/message focus에서 한 단계 back한 뒤
  files focus에서 Graph로 복귀한다.
- tree collapse/expand와 code horizontal scroll은 서로 다른 key 또는 명확한
  focus-dependent contract를 사용한다. footer는 현재 pane, back, retry,
  message toggle 중 현재 상태에 필요한 핵심 키만 짧게 노출한다. 기본 footer는
  `Tab focus · m message · q close · Esc back`처럼 유지하고, loading/ready에서는
  불필요한 `r`을 숨기며 error/partial일 때만 `r retry`를 추가한다. A/M/D와
  `+/-/space` legend는 고정 footer를 채우지 않고 header의 짧은 legend 또는 `?`
  도움말에서 제공한다. 최종적으로 기본 footer는 조작 키만 노출하고, 전체 A/M/D/R
  및 `+/-/space` legend와 pane별 상세 키는 `?` 도움말에서 제공한다.
- `A/M/D/R`, `+/-/space`, selected file, active pane, `FROM/TO`는 NO_COLOR에서도
  문자열/marker로 구분된다. wide-rune path, tab, control character, invalid UTF-8,
  long unbroken path는 visible-width와 pane alignment를 보존한다.

### Git basis contract

MVP는 first-parent basis를 `CommitInspection` snapshot에 고정한다. changed-file
listing과 from/to patch가 같은 parent/commit basis를 사용하며, merge commit header에
그 parent를 표시한다. rename은 display path가 아니라 stable `FileID`, `OldPath`,
`Path`를 reader에 전달한다. root/merge/rename/binary/empty cases는 user-visible
behavior table과 temporary Git integration fixture로 검증한다.

### Graphkeeper differentiation gate

Inspector는 generic diff viewer가 아니라 Graph에서 선택한 commit의 history reader다.
5초 안에 `selected commit → parent basis → changed file → from/to change`가 읽혀야
하며, syntax highlighting 없이도 이 맥락을 전달해야 한다. word-level diff와
highlighter registry는 이 gate를 통과한 뒤의 P2 enhancement로 유지한다.

## Autoplan decision audit trail

| # | Phase | Decision | Classification | Principle | Rationale | Rejected |
|---|---|---|---|---|---|---|
| 1 | CEO | 독립 screen/border를 launch gate로 승격 | Mechanical | completeness | 현재 overlay는 계획의 핵심 화면을 제공하지 않음 | polish-only overlay |
| 2 | CEO | stale token/cancellation을 Phase 1로 이동 | Mechanical | explicit over clever | 잘못된 diff가 보이는 MVP는 신뢰할 수 없음 | Phase 2 defer |
| 3 | CEO | first-parent basis를 file list와 diff에 공유 | Mechanical | DRY/pragmatic | merge에서 두 화면이 다른 변경을 보이는 위험 제거 | implicit Parents[0] |
| 4 | Design | pane 상태 matrix와 responsive snapshots 추가 | Mechanical | completeness | 상태·폭별 사용자 결과를 구현자에게 위임하지 않음 | generic 40/60/80 test |
| 5 | Design | Graph provenance를 identity band에 유지 | Taste | explicit over clever | generic diff viewer와 Graphkeeper history reader를 구분 | subject/hash-only header |
| 6 | Design | word-level/highlighter는 primary gate 뒤로 유지 | Taste | pragmatic | 현재 핵심 결함인 from/to와 상태를 먼저 완성 | syntax-first MVP |

## Engineering review 결과

2026-08-03 `/plan-eng-review` 결과를 반영했다. 현재 구현은 아래 P0를 해결하기 전에는
완료로 표시할 수 없다.

- 독립 screen 경계가 없고 Graph 위에 borderless popup을 합성한다.
- async message에 commit/file/request identity와 cancellation이 없어 A→B→A 또는
  close→reopen 순서에서 stale 결과가 현재 선택을 덮을 수 있다.
- changed-file listing과 diff가 서로 다른 Git basis를 사용할 수 있다. listing과
  patch 모두 동일한 first-parent snapshot을 소비해야 한다.
- metadata가 subject만 읽고 `Runner.RunContext`가 raw payload를 trim하므로 전체
  message, trailers, 마지막 newline을 보존할 수 없다.
- 대용량 diff가 `gitRaw`와 `strings.Split`로 전체 materialize된 뒤에야 limit을
  적용하므로 bounded streaming을 MVP gate로 이동한다.

P1로 app-owned reader port/DTO, pane별 상태, display-width layout, hostile input
sanitization과 fake-reader 테스트를 고정한다. P2는 key ambiguity가 없는 상태에서만
word-level diff와 실제 highlighter를 진행한다.

### Architecture dependency graph

```text
Graph Enter
    │  commit + repositoryEpoch
    ▼
Inspector screen/state ── cancel/request token ──┐
    │                                             │
    ├─ Header/message state                         │ stale discard
    ├─ Changed-files tree state                     │
    └─ From/to viewport state ◄── reader port ◄────┘
                                      │
                                      ▼
                           git adapter + bounded stream
                                      │
                         first-parent CommitSnapshot
                                      │
                         hunk/line-pair/binary contract
```

`internal/app`은 reader port와 화면 DTO를 소유하고, `internal/git`은 동일한
commit/parent/file/window 요청을 Git 명령으로 변환한다. renderer는 raw unified diff를
해석하지 않으며, 모든 결과는 현재 commit, stable FileID, request ID, repository epoch를
검사한 뒤에만 반영한다.

`ParentSelection`은 metadata 단계에서 한 번 resolve한다. root commit만
`git diff-tree --root ... <commit>`을 사용하고, normal/merge commit의 changed-file
listing과 selected-file patch는 모두 동일한 명시적 `<parent> <commit>` 범위를
사용한다. listing이 commit만 전달받아 Git의 implicit merge diff semantics에
의존하는 경로는 허용하지 않는다. E3 fixture는 normal/first-parent merge에서
listing과 patch의 parent argument가 동일함을 검사한다.

### Engineering test plan artifact

세부 테스트 매트릭스는
`/Users/hrk/.gstack/projects/hrllk-graphkeeper/develop-test-plan-20260803.md`에
기록한다. 최소 흐름은 다음과 같다.

```text
Enter → metadata/files loading → ready|empty|error
                   │
                   ├→ select file → from/to loading → ready|partial|error → r retry
                   ├→ m message viewport → preserve file/diff focus
                   ├→ A→B→A delayed result → stale discard
                   └→ q/Esc close → cancel → Graph cursor/scroll restore

width×height: (40|60|80) × (12|20|30)
Git fixtures: root / first-parent merge / rename / binary / empty / hostile paths
render fixtures: NO_COLOR / wide rune / invalid UTF-8 / malformed hunk / large stream
```

`go test ./...`와 `scripts/check`는 구현 단계의 필수 gate다. 현재 리뷰 환경에서는
Eng voice가 Go build cache 권한 문제로 테스트를 실행하지 못했으므로, 계획 승인과
구현 완료를 혼동하지 않는다.

## Developer-experience review 결과

2026-08-03 `/plan-devex-review` 결과를 반영했다. first-use journey를 다음 계약으로
고정한다.

| Journey | Contract | Acceptance evidence |
|---|---|---|
| 발견 | Graph action help, `?`, README에 `enter: inspect commit` 노출 | help/README assertions |
| 진입 | Enter 즉시 identity band와 files/from/to 상태 표시 | loading snapshot |
| 탐색 | footer에 `j/k`, `Tab`, `Enter`, `m`, `r`, `q`, `Esc` 표시 | focus별 footer snapshot |
| 복구 | `r`은 실패 pane만 재시도하고 ready pane·선택을 보존 | pane transition test |
| 종료 | `q`는 어느 pane에서도 즉시 close, `Esc`는 한 단계 back | key handling test |
| 좁은 화면 | files와 from/to를 순차 전환하되 path, FROM/TO, focus, close 유지 | 40-column snapshots |
| 특수 commit | root/merge/rename/binary/empty를 명시적 상태로 설명 | temporary Git fixtures |
| stale 결과 | canceled/mismatched 결과는 조용히 폐기하고 현재 선택을 변경하지 않음 | delayed fake-reader test |

DX 리뷰에서 발견된 P0는 retry와 structured from/to contract를 MVP gate로 승격한다.
`CommitDiff.Lines` 같은 raw string만으로는 사용자가 양쪽 변경을 안정적으로 읽을 수
없다. P1은 discoverability, `q`/`Esc` 의미, narrow/NO_COLOR, 특수 commit fixture와
문서 동기화다.

## Autoplan completion scorecard

점수는 원래 계획의 실행 가능성을 리뷰 전/보정 후로 기록한다. 보정 후 점수는 계획
문서의 계약과 검증 경로가 구현자에게 충분히 명시되었는지를 뜻하며, 구현 완료를
뜻하지 않는다.

| Dimension | Before | After | Reason for remaining gap |
|---|---:|---:|---|
| CEO / product outcome | 6/10 | 9/10 | 실제 구현은 아직 partial |
| Design / hierarchy & states | 5/10 | 9/10 | snapshots must be implemented |
| Engineering / architecture | 4/10 | 8/10 | bounded reader and port remain code work |
| DX / discoverability & recovery | 4/10 | 8/10 | README/help sync remains implementation work |
| Testability / operational safety | 3/10 | 8/10 | environment could not run Go build in review |

Autoplan voices: CEO `codex-only`, Design `codex-only`, Eng `codex-only`, DX
`codex-only`; Claude voice는 이 환경에 CLI/agent가 없어 실행하지 못했다. Mechanical
결정은 completeness와 explicit-over-clever 원칙으로 자동 확정했고, taste 결정은
Graph provenance 유지와 syntax-first defer로 pragmatic하게 확정했다.

### Dual-voice status

Claude subagent/CLI가 제공되지 않아 각 phase는 Codex 단일 voice로 실행했다. 따라서
아래 `N/A`는 합의가 아니라 missing voice를 뜻한다.

| Review | Claude | Codex | Decision |
|---|---|---|---|
| CEO: product outcome/scope | N/A | P0 boundary, stale, basis, message gaps | corrective contract에 반영 |
| Design: hierarchy/states | N/A | P0 screen/from-to, P1 responsive/focus gaps | pane matrix와 identity band에 반영 |
| Eng: architecture/tests/performance/security | N/A | P0 five risks, P1 coupling/state/input/test gaps | T1/T2/T3/T4와 test artifact에 반영 |
| DX: first use/recovery/docs | N/A | retry, q/Esc, discoverability, fixture gaps | DX contract와 T6에 반영 |

### DX scorecard and empathy note

| Dimension | Score | Contract |
|---|---:|---|
| first-use discoverability | 8/10 | Enter/help/footer/README에 inspect 노출 |
| time-to-first-success | 8/10 | identity → file → from/to를 5초 scan |
| feedback/loading | 8/10 | pane별 skeleton과 선택 context 유지 |
| error recovery | 8/10 | affected pane만 `r` retry, ready pane 보존 |
| keyboard discoverability | 8/10 | Tab, m, r, q, Esc를 footer에 고정 |
| narrow terminal | 8/10 | sequential mode에서도 path/방향/focus 보존 |
| special commits | 8/10 | root/merge/rename/binary/empty 명시 상태 |
| docs/test fixtures | 7/10 | 구현 후 README/help와 temporary Git matrix 동기화 |

사용자는 Graph에서 판단한 커밋을 빠르게 검증하려는 개발자다. 현재 borderless
popup과 사라지는 from/to 결과는 “내가 선택한 변경이 맞는가”를 확인하기 전에
신뢰를 잃게 만든다. 따라서 첫 성공은 syntax 색상이 아니라 commit identity,
parent basis, selected path, 양쪽 변경을 안정적으로 읽고 다시 Graph로 돌아오는
것으로 정의한다.

TTHW는 현재 `Enter → incomplete popup → unclear recovery`에서 목표
`Enter → identity → file → from/to → q/Esc return`으로 줄인다. DX checklist는
`enter` 발견, loading/empty/error, pane-local retry, q/Esc, 40열, NO_COLOR,
특수 commit, stale discard와 README/`?`/footer 동기화를 모두 포함한다.

## Cross-phase themes

1. **신뢰성 우선:** stale result, shared Git basis, raw message 보존은 화면 polish보다
   먼저 완료한다.
2. **Graph provenance 유지:** Inspector는 generic diff viewer가 아니라 선택한
   commit의 parent-to-commit history reader다.
3. **상태를 숨기지 않기:** loading/partial/error/canceled를 pane 단위로 표현하고,
   실패하지 않은 pane의 탐색을 막지 않는다.
4. **표현은 보조:** Tree-sitter와 색상은 semantic marker, 방향, line number를
   대체하지 않으며 NO_COLOR에서도 동일한 정보 구조를 유지한다.
5. **검증 가능한 범위:** 구현은 main `feature`와 단 하나의 `commit detail` subtask에서
   진행하고, 실제 Tree-sitter bundle과 merge parent selector는 `TODOS.md`로 연기한다.

### User journey storyboard

| Step | User does | User should feel | Plan support |
|---|---|---|---|
| 1. Graph scan | 커밋 row의 topology·subject·author를 확인한다 | 내가 조사할 대상을 정확히 골랐다는 확신 | Graph cursor와 기존 selection 보존 |
| 2. Enter | 선택 커밋에서 `Enter`를 누른다 | 화면이 바뀌어도 같은 commit을 보고 있다는 안정감 | 독립 border, hash/parent/ROOT identity band |
| 3. File select | Changed files tree에서 파일을 고른다 | 변경 범위를 빠르게 좁힌다는 통제감 | A/M/D/R marker, selected path 고정, pane-local loading |
| 4. Change verify | from/to와 line marker를 비교한다 | 무엇이 어디서 어떻게 바뀌었는지 확인했다는 신뢰 | paired rows, `FROM`/`TO`, NO_COLOR semantic marker, partial hint |
| 5. Return/recover | `q` 또는 `Esc`로 돌아가거나 `r`로 재시도한다 | 실패해도 원래 Graph 판단을 잃지 않는 안전감 | cursor/scroll 복원, ready pane 보존, 짧은 context-sensitive footer |

시간축 기준으로 첫 5초에는 identity·selected path·from/to 방향을, 5분 사용에서는
file tree·message·diff paging을, 장기 사용에서는 Graph cursor 복원과 first-parent
basis를 통해 반복 가능한 history reading 습관을 지원한다.

## Engineering review decisions

2026-08-03 `/plan-eng-review`를 확정했다. 스코프는 vertical slice로 줄였고, 다음
결정은 사용자 승인 후 이 계획의 구현 gate로 고정한다.

### Scope decision

1차 slice는 independent bordered Inspector, Changed files tree, first-parent basis,
기본 from/to pairing, request identity/cancellation, bounded streaming/cache와
visible tree projection을 포함한다. Tree-sitter grammar/query, word-level diff,
merge parent selector/combined diff는 `TODOS.md`에 유지한다.

### What already exists

| Existing flow | Reuse | Required change |
|---|---|---|
| Graph `Enter` commit hash intent | `internal/app/key_handling_browse.go` | Inspector request identity 추가 |
| Bubble Tea model/update | `internal/app/model.go`, `update.go` | `InspectorState`와 pane별 result로 교체 |
| shell layout/border | `view_shell.go`, `view_layout.go` | Inspector open 시 Graph branch, shared visible-width helper 재사용 |
| Git runner/context | `internal/git/repo_exec.go` | bounded reader와 immutable parent basis 추가 |
| NUL path parser | `parseCommitDiffFiles` | rename/copy/binary/mode metadata contract로 확장 |
| repository epoch | `model.go`, `update_lifecycle.go` | Inspector request identity와 함께 검사 |

새로운 state machine package나 외부 diff renderer는 만들지 않는다.

### Design system alignment

Inspector는 `popupBorder`, `baseBox`, `activeBox`의 기존 border·ANSI·`NO_COLOR`
의미를 기본값으로 재사용한다. 새 스타일은 색상 자체가 아니라 역할 이름으로만
추가하며, 예를 들면 `inspectorFrame`은 독립 screen 경계, `inspectorPaneActive`는
현재 focus pane, `inspectorSelectedRow`는 파일/line 선택을 담당한다. 기존 semantic
color map과 `docs/highlighting-color-map.md`의 fallback을 바꾸지 않고, focus와
상태는 marker·label·border 변화로도 식별 가능해야 한다.

### Architecture and data flow

```text
Graph Enter
    │ commit + repository epoch
    ▼
InspectorState ── request token/cancel ─────────────┐
    │                                               │ stale discard
    ├─ identity/message                              │
    ├─ visible file-tree projection                  │
    └─ from/to paired window ◄── app reader port ◄──┘
                                      │
                                      ▼
                           git adapter / bounded stream
                                      │
                           immutable first-parent basis
```

`internal/app`은 DTO, focus, pane state와 projection을 소유한다. `internal/git`은
Git argument array, NUL parser, hunk pairing, bounded reader를 소유한다. 동일한
의미의 model을 양쪽 package에 복제하지 않는다.

### Raw Git output boundary

기존 `Runner.RunContext`의 trim 반환 계약은 다른 Git 호출자의 호환성을 위해
유지한다. Inspector adapter에는 `RunRawContext` 또는 동등한 raw/stream 전용 경계를
추가해 metadata message의 줄바꿈·trailer·마지막 newline과 diff parser 입력을
trim 없이 소비한다. bounded diff는 이 raw 경계에서 line/byte cap을 적용하며,
renderer가 문자열을 다시 복원하려 하지 않는다.

### Request cancellation ownership

`InspectorState`는 현재 metadata/diff request의 child context와 cancel handle을
소유한다. 새 commit request, file/window 전환, retry, Inspector close가 발생하면
app state가 이전 cancel을 먼저 호출하고 request identity를 증가시킨 뒤 새 child
context를 reader port에 전달한다. reader adapter는 이 context를 `exec.CommandContext`
까지 연결하고, 결과는 request/commit/file/epoch 검증을 통과한 경우에만 반영한다.

```text
InspectorState.cancelCurrent()
          │
          ├─ child context canceled → git child process exits
          └─ request identity++    → late result discarded
                                      │
                                      └─ next request starts only after cancel call
```

취소는 사용자 오류 화면으로 표시하지 않는다. 테스트는 close/file switch/retry
각 경로에서 cancel 호출, child process 종료, 늦은 결과 폐기를 각각 검증한다.

### Failure modes

| Failure | Test | Handling | User result |
|---|---|---|---|
| stale A→B→A result | fake reader state test | request/file/epoch mismatch discard | 현재 선택 유지 |
| close during Git process | stream cancellation test | context cancel + child wait | 조용히 Graph 복귀 |
| Git timeout/exit | pane transition test | pane error + retry | ready pane 계속 사용 |
| malformed NUL/hunk | parser fixture | parse error, no guessed pair | 원인과 retry 표시 |
| 1 MiB/2,000 line cap | bounded stream test | partial + continuation | 일부 표시임을 명시 |
| ESC/invalid UTF-8/path | hostile input test | sanitize before projection | terminal corruption 없음 |
| width/height overflow | renderer snapshots | visible-width clamp/fallback | path/focus/close 보존 |

### Test coverage diagram

```text
CODE PATHS                                      USER FLOWS
Enter → metadata loading                       Enter selected commit
  ├─ ready → tree projection                   → identity + files visible
  ├─ empty → empty state                       → select file → from/to
  └─ error → retry/Esc                          → q/Esc → Graph restore
file select → bounded diff window              rapid A→B→A → stale discard
  ├─ ready → paired rows                        close during load → cancel
  ├─ partial → continuation                     NO_COLOR/hostile path → safe text
  └─ error → pane retry                          large diff → partial hint
Graph restore → cursor/scroll regression        Graph keys → unchanged behavior
```

Coverage gates are app fake-reader flows, temporary Git integration fixtures, renderer
matrix snapshots, hostile-input tests, bounded-stream tests, and existing Graph
regression tests. No E2E browser suite applies to this terminal UI.

### Parallelization strategy

| Step | Module | Depends on |
|---|---|---|
| A. app contract/state | `internal/app` | — |
| B. Git basis/parser/stream | `internal/git` | — |
| C. screen/tree/from-to renderer | `internal/app` | A |
| D. integration/regression tests | `internal/app`, `internal/git` | A+B+C |

Lanes A와 B는 서로 다른 package에서 병렬 착수할 수 있다. C는 A 이후, D는 A/B/C
병합 이후 순차 진행한다. A와 C가 모두 `internal/app`을 수정하므로 같은 worktree에서
동시에 진행하지 않는다.

### Implementation tasks from Eng review

- [ ] **E1 (P1, human: ~1–2d / CC: ~20–40min)** — app-owned `InspectorState`, reader port, request identity와 cancellation 구현
  - Surfaced by: Architecture 1/4, Code Quality 1/3
  - Files: `internal/app/model.go`, `internal/app/update.go`, `internal/app/commit_inspector.go`, `internal/app/messages.go`
  - Verify: fake-reader ready/error/retry, A→B→A stale, close/file-switch/retry cancel과 child 종료, q/Esc
- [ ] **E2 (P1, human: ~1d / CC: ~15–30min)** — Graph shell branch와 visible tree/from-to projection 구현
  - Surfaced by: Architecture 2, Code Quality 2/5, Performance 2
  - Files: `internal/app/view_shell.go`, `internal/app/view_overlays.go`, `internal/app/commit_inspector.go`, `internal/app/graph_render_format.go`
  - Verify: Graph body absence, frame bounds, 40/60/80 matrix, visible tree projection
- [ ] **E3 (P1, human: ~1–2d / CC: ~20–40min)** — immutable first-parent basis, app adapter mapping, deterministic pairing 구현
  - Surfaced by: Architecture 3/5, Code Quality 4
  - Files: `internal/app/commit_inspector_contract.go`, `internal/git/repo.go`, `internal/git/repo_exec.go`
  - Verify: root/normal/merge/rename fixture, same parent for listing/diff, paired rows
- [ ] **E4 (P1, human: ~1–2d / CC: ~20–40min)** — bounded stream와 epoch-aware window cache 구현
  - Surfaced by: Architecture 6, Performance 1
  - Files: `internal/git/repo_exec.go`, `internal/app/commit_inspector.go`, `internal/app/model.go`
  - Verify: 1 MiB/2,000-line cap, cache hit/miss, continuation, child cancellation
- [ ] **E5 (P1, human: ~1–2d / CC: ~20–40min)** — app/Git/renderer/regression test matrix 작성
  - Surfaced by: Test review 1–6
  - Files: `internal/app/commit_inspector_contract_test.go`, `internal/app/commit_inspector_state_test.go`, `internal/app/commit_inspector_view_test.go`, `internal/git/commit_inspector_integration_test.go`, `internal/git/commit_inspector_stream_test.go`
  - Verify: `go test ./...`, `scripts/check`, hostile input, Graph navigation regression

## NOT in scope

- 실제 Tree-sitter grammar/query 번들: 지원 언어·라이선스·cgo·artifact 크기 정책이
  별도 결정이어야 한다.
- merge parent selector와 combined diff: MVP first-parent 표시로 충분하며 별도
  history inspection 기능으로 분리한다.
- blame, symbol navigation, AST semantic diff, staging/editing: read-only commit
  inspection의 핵심 결과를 흐린다.

## What already exists

- `internal/app/key_handling_browse.go`: Graph `enter`와 선택 commit hash 진입점
- `internal/app/view_overlays.go`: 기존 overlay stack. 새 Inspector screen으로
  전환하되 다른 popup 계약은 재사용한다.
- `internal/git/repo_exec.go`: metadata, NUL-delimited changed-file parsing,
  selected-file diff 조회의 출발점
- `internal/app/model.go`와 `update.go`: Bubble Tea model/message/update 경계.
  기존 bool/flat scroll 상태는 새 Inspector state로 교체한다.
- `internal/app/theme.go`, `docs/highlighting-color-map.md`: ANSI/NO_COLOR
  semantic style 정책과 fallback 원칙

## Failure Modes Registry

이 표의 상태는 계획과 검증을 구분한다. `Verified`는 구현과 테스트가 실제로
통과한 뒤에만 사용할 수 있다.

| Codepath | Failure mode | Implementation | Verification | User sees |
|---|---|---|---|---|
| metadata read | missing/stale commit | Partial | Planned | error screen + Esc |
| file listing | Git exit/malformed NUL | Partial | Existing parser test + planned contract test | files pane error + retry |
| diff window | stale file/subprocess error | Planned | Planned | diff pane error + retry |
| diff reader | byte/line limit | Planned | Planned | partial marker + hint |
| pairing | malformed hunk | Planned | Planned | parse error, no silent pair |
| highlighter | unsupported/parse failure | Planned | Planned | plain code fallback |
| async result | old epoch/selection | Planned | Planned | stale result discarded |

## Implementation Tasks

- [ ] **T1 (P0, human: ~2–3d / CC: ~30–60min)** — Inspector screen boundary와 state —
  Graph shell 대체, 공통 border, identity band, pane state, request token/cancellation,
  key dispatch, loading/error/retry, Esc 복귀 구현
  - Surfaced by: Architecture/Error review
  - Files: `internal/app/model.go`, `internal/app/update.go`, `internal/app/key_handling.go`, `internal/app/view_shell.go`, `internal/app/view_overlays.go`, `internal/app/commit_inspector.go`
  - Verify: Graph body 부재·frame bounds·delayed A→B→A stale result·close cancellation·pane state tests
- [ ] **T2 (P0, human: ~2–3d / CC: ~30–60min)** — Message와 Git basis contract —
  full message, shared first-parent snapshot, stable FileID/OldPath/Path, root/merge/rename/binary parser 구현
  - Surfaced by: Data flow review
  - Files: `internal/git/repo.go`, `internal/git/repo_exec.go`, `internal/git/repo_test.go`
  - Verify: multiline message/trailers, root/merge/rename/binary/empty temporary Git fixture matrix
- [ ] **T3 (P0, human: ~2–3d / CC: ~30–60min)** — Changed files tree와 from/to layout
  - Surfaced by: UX review
  - Files: `internal/app/commit_inspector.go`, `internal/app/view_layout.go`, `internal/app/theme.go`, 관련 tests
  - Verify: 3-column/compact/narrow snapshots, fixed marker-number-code columns, focus/footer/NO_COLOR/wide-rune alignment
- [ ] **T4 (P0, human: ~1–2d / CC: ~20–40min)** — Lazy diff viewport와 bounded cache
  - Surfaced by: Performance review
  - Files: Inspector state/reader/cache 구현 파일, `internal/git/repo_exec.go`, 관련 tests
  - Verify: large diff에서도 전체 출력 materialization이 없는 bounded stream,
    partial window/continuation, byte·line cap, request cancellation, epoch invalidation
- [ ] **T5 (P2, human: ~1–2d / CC: ~20–40min)** — semantic style와 word-level diff
  - Surfaced by: Code quality/UX review
  - Files: Inspector renderer/highlighter 구현 파일, `docs/highlighting-color-map.md`, tests
  - Verify: A/M/D, +/- text fallback, ANSI/NO_COLOR, fake highlighter fallback
- [ ] **T6 (P1, human: ~2–3d / CC: ~30–60min)** — contract matrix·문서·diagnostics
  - Surfaced by: Test/Observability review
  - Files: `internal/app/*_test.go`, `internal/git/*_test.go`, `README.md`, `docs/decisions.md`, `internal/app/view_sections.go`, `internal/app/hidden_hotkeys.go`
  - Verify: Graph help/README `enter` discoverability, `q`/`Esc` contract, `go test ./...`, `scripts/check`, low-size and error matrix

### Design review additions

- [ ] **D1 (P1, human: ~2h / CC: ~15min)** — low-height hierarchy와 contextual empty state 구현
  - Surfaced by: Pass 1 Information Architecture, Pass 2 Interaction State Coverage
  - Files: `internal/app/commit_inspector.go`, `internal/app/view_layout.go`, `internal/app/commit_inspector_view_test.go`
  - Verify: `(40,12)`/`(60,12)`에서 identity·selected path·FROM/TO·최소 diff·q/Esc 유지; no-changes/binary/submodule 상태가 오류와 구분됨
- [ ] **D2 (P1, human: ~2h / CC: ~15min)** — compact focus model과 terminal accessibility invariant 고정
  - Surfaced by: Pass 5 Design System Alignment, Pass 6 Responsive & Accessibility, Pass 7 narrow transition
  - Files: `internal/app/theme.go`, `internal/app/view_shell.go`, `internal/app/commit_inspector.go`, `internal/app/commit_inspector_view_test.go`
  - Verify: 기존 `popupBorder`/`baseBox`/`activeBox` 재사용, Tab-only narrow transition, NO_COLOR에서도 marker·label·border·FROM/TO 식별
- [ ] **D3 (P2, human: ~1h / CC: ~10min)** — concise footer와 `?` help legend·journey copy 동기화
  - Surfaced by: Pass 3 User Journey, Pass 4 AI Slop Risk, Pass 7 legend decision
  - Files: `internal/app/hidden_hotkeys.go`, `internal/app/view_sections.go`, `README.md`, `docs/decisions.md`
  - Verify: ready/loading/error별 footer가 필요한 키만 노출하고, `?`에서 A/M/D/R·`+/-/space`와 pane 키를 확인할 수 있음

## Dream state delta

```text
현재                         이번 계획                         12개월 후
Graph row + 짧은 subject  →  Enter 기반 read-only       →   Graph history reader
세로형 patch 문자열           changed-files/from/to 화면       diff·blame·parent 비교
외부 도구로 상세 확인         구조화된 diff contract             여러 inspection mode
```

이번 작업은 Graphkeeper를 staging 도구로 바꾸지 않고, “그래프에서 판단한 뒤
커밋의 실제 변경을 즉시 검증하는 history reader” 방향으로 이동시킨다.

## Error & Rescue Registry

| Codepath | Exception/error class | Rescue action | User impact | Test |
|---|---|---|---|---|
| metadata read | `exec.ExitError`, `context.Canceled`, invalid commit | Inspector error state, Esc recovery | commit을 읽지 못한 이유 표시 | planned |
| file listing | `exec.ExitError`, malformed NUL record | files pane error, retry, malformed record visible | 파일 목록을 완전한 것으로 오인하지 않음 | planned |
| diff window | stale file, `exec.ExitError`, canceled command | diff pane error, selection 유지, retry; canceled request는 stale로 폐기 | 다른 파일로 튕기지 않음 | planned |
| bounded reader | byte/line limit exceeded | partial state와 continuation hint | 일부만 보인다는 사실을 인지 | planned |
| pair parser | malformed hunk/header | parse error row, no guessed pairing | 잘못된 from/to 비교 방지 | planned |
| optional highlighter | unsupported language, parse error, invalid UTF-8 | plain code fallback, semantic markers 유지 | 코드 읽기는 계속 가능 | planned |
| async result | old repository epoch/selection token | discard stale message | 이전 commit/file 결과가 덮어쓰지 않음 | planned |

## Rollback flow

```text
Inspector regression 발견
        │
        ├─ parser/renderer test failure → 해당 commit revert
        ├─ runtime crash/lockup         → 기능 commit revert, 기존 overlay 복구
        └─ style/readability issue      → semantic style 또는 layout commit만 revert
```

DB migration과 remote state 변경이 없으므로 rollback은 Git revert로 끝난다. 실제
Tree-sitter grammar는 별도 dependency slice로 분리해 이 Inspector MVP의 rollback
범위를 넓히지 않는다.

## Stale diagram audit

- 목표 화면 diagram: 독립 screen과 3열 from/to 구조로 갱신됨
- side-by-side pairing diagram: deterministic deletion/addition run 규칙과 일치함
- Dream state diagram: 현재 계획의 12개월 방향과 일치함

## Engineering review completion summary

- Scope was reduced to one vertical slice: independent bordered Inspector screen,
  changed-files tree, first-parent from/to diff, request identity/cancellation,
  bounded streaming/cache, and visible tree projection.
- Six architecture, five code-quality, six test, and two performance findings were
  accepted as implementation decisions (19/19 review decisions resolved).
- Actual Tree-sitter grammar/query integration and merge parent selection/combined
  diff remain explicit TODOs, not hidden implementation assumptions.
- No critical gaps remain for the reduced slice. E1–E5 are the locked execution tasks;
  E1 and E2 may begin in parallel with the Git contract work in E3/E4, followed by E5.
- The review artifacts are recorded in the external project workspace, including the
  test plan and Taskmaster-compatible execution records. No GitHub issue or remote
  mutation was created.

Inline implementation comments should document the state transition in
`internal/app/model.go`, request/result ownership in `internal/app/commit_inspector.go`,
and fixture-to-assertion flow in the Inspector and Git test files.

## Design review completion summary

| Pass | Before | After | Result |
|---|---:|---:|---|
| Information Architecture | 9/10 | 10/10 | low-height diff-first hierarchy fixed |
| Interaction State Coverage | 8/10 | 10/10 | contextual empty/binary/submodule states fixed |
| User Journey & Emotional Arc | 7/10 | 10/10 | five-step history-reader storyboard added |
| AI Slop Risk | 9/10 | 9/10 | no generic UI patterns found; visual mockup unavailable |
| Design System Alignment | 8/10 | 10/10 | existing border/focus token reuse fixed |
| Responsive & Accessibility | 8/10 | 10/10 | Tab-only narrow mode and non-color invariants fixed |
| Unresolved Design Decisions | 7/10 | 10/10 | narrow transition and legend placement resolved |

- System audit: `DESIGN.md` exists; this is a terminal APP UI, not a marketing page.
- Approved mockups: 0 generated, 0 approved. The installed designer could not run because
  no OpenAI API key was configured; review continued from plan, design system, and code.
- Design decisions added: 9 (low-height hierarchy, empty states, journey, token reuse,
  accessibility invariants, Tab-only narrow mode, concise footer, help-only legend, and
  two deferred TODO confirmations).
- Deferred design decisions: 2, both already tracked in `TODOS.md`: Tree-sitter bundle
  policy and merge parent selector/combined diff.
- Design tasks D1–D3 are ready to execute. No unresolved design decisions remain.

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|---|---|---|---:|---|---|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 3 | CLEAN | launch gates and vertical-slice scope resolved |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | — | not run in this review |
| Eng Review | `/plan-eng-review` | Architecture, data flow, tests, performance | 7 | CLEAN | 19 findings accepted; E1–E5 locked; no critical gaps |
| Design Review | `/plan-design-review` | UI/UX gaps | 5 | CLEAN | 8/10 → 9/10; 9 decisions; 0 unresolved; D1–D3 added |
| DX Review | `/plan-devex-review` | Developer experience gaps | 3 | CLEAN | discoverability, retry, q/Esc, special commit and narrow contracts resolved |

**VERDICT:** CEO, Design, Eng, and DX reviews are clear. The plan is design-complete
for implementation of the reduced slice; the feature remains `in-progress` until the
engineering and design implementation tasks plus their verification gates are completed.

NO UNRESOLVED DECISIONS
