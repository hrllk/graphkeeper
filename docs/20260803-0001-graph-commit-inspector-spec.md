# Graph Commit Inspector 구현 스펙

상태: 확정
기반 계획: Graph Commit Inspector 계획 문서
Taskmaster: main `feature` 아래 `commit detail` subtask

## Context

Graphkeeper 사용자는 maintainer뿐 아니라 개발자까지 포함한다. 이들이 Graph에서
선택한 커밋을 보고 가장 먼저 확인하려는 것은 “이 커밋에서 어떤 내용이 변경되었는가?”
이다. fixture repository에서 `Enter` key event부터 identity와 첫 diff row가 포함된
ready frame까지 wall-clock 5초 이하를 목표로 한다. loading frame은 첫 Bubble Tea
update 안에 identity placeholder와 cancel action을 표시한다.

현재 Inspector는 Graph 위의 borderless popup에서 파일 목록과 세로 unified diff만
표시한다. from/to 구조가 없고, 파일을 빠르게 이동하면 이전 비동기 결과가 현재 선택을
덮을 수 있다. 이 스펙은 해당 partial 구현을 독립 read-only Inspector로 완성한다.

## Current State

| 영역 | 현재 상태 | 근거 |
|---|---|---|
| Screen | Graph 위 borderless popup | `internal/app/view_shell.go:47-83`, `internal/app/view_overlays.go:21-27` |
| File tree | 단순 파일 목록 | `internal/app/commit_inspector.go:115-124` |
| Diff | raw `Lines` 기반 세로 출력 | `internal/git/repo.go:106-110`, `internal/app/commit_inspector.go:125-140` |
| Message | subject만 조회 | `internal/git/repo_exec.go:151-165` |
| Async | request identity/cancel 없음 | `internal/app/commit_inspector.go:19-32`, `internal/app/update.go:19-45` |
| Git basis | file listing과 diff 기준 불일치 가능 | `internal/git/repo_exec.go:160`, `:169-180` |
| Large diff | 전체 stdout materialize 후 limit 적용 | `internal/git/repo_exec.go:180-200`, `:530-550` |

## User Outcome

Graph에서 커밋을 선택하고 `Enter`를 누르면 Graph가 완전히 가려진 bordered
Inspector가 열린다. 사용자는 5초 안에 commit identity, parent basis, selected file,
unified diff와 commit message를 확인하고 `q` 또는 `Esc`로 Graph의 기존 선택 위치로
돌아갈 수 있어야 한다.

## Screen Contract

```text
+-------------------------------------------------------------+
| commit: <full hash>                                         |
| message: <subject, max 100 visible cells>                   |
| author: <author/email>                                      |
| path: <selected repository-relative path>                   |
+---------------------------+---------------------------------+
| Changed files             | Diff                            |
| > M ../../model.go        | @@ -10 +10 @@                  |
|   M ../../update.go       |  10  10 │ context                |
+---------------------------+---------------------------------+
| q close  Esc back  ? help                                   |
+-------------------------------------------------------------+
```

- Inspector는 일반 `overlayPopup`이 아니라 독립 screen이다.
- Graph body는 Inspector가 열린 동안 렌더링하지 않는다.
- 공통 overlay와 동일한 border style을 사용한다.
- 닫으면 Graph cursor와 scroll을 보존한다.
- Header는 4행으로 고정하고 `commit:`, `message:`, `author:`, `path:`를 표시한다.
  `path:`에는 선택 파일의 repository-relative full path를 표시하며, parent/root와
  `READ-ONLY HISTORY` context는 author 또는 pane label의 보조 context로 표시한다.
- 본문만 viewport scroll 대상이며, 좌측은 현재 커밋의 변경 파일을 flat list로
  표시하고 우측은 old line / new line / marker / code 열의 unified diff를 표시한다.
- MVP footer에는 commit message 본문을 표시하지 않는다. header의 `message:` 행에
  subject를 최대 100 visible cells까지 표시하며, 전체 message/body/trailer viewport는
  후속 기능으로 남긴다.

## Data Contract

`internal/app`이 reader port와 DTO를 소유한다. `internal/git`은 Git adapter로
contract를 채운다.

### Implementation scope split

아래 계약 중 **MVP Required**만 이번 `commit detail` 구현의 완료 조건이다.
**Follow-up Contract**는 타입 호환성과 후속 작업의 경계를 미리 고정하지만, 이번
구현에서 값을 생성하거나 UI를 제공하지 않는다.

| Area | MVP Required | Follow-up Contract |
|---|---|---|
| commit identity | full hash, subject, author, resolved first-parent/root | alternate merge parent selection |
| changed files | stable ID, A/M/D/R/C, old/new path, flat sorted projection | richer mode metadata and future status extensions |
| diff | bounded hunk/paired rows, old/new line columns, binary/submodule state | `WordSpan`, word-level diff, syntax token spans |
| reader | `InspectCommit`, `LoadDiff`, request/epoch identity, cancellation | additional inspection modes such as blame/parent comparison |
| rendering | border, flat files/unified diff, fixed header, loading/empty/partial/error, NO_COLOR markers | actual Tree-sitter grammar/query bundle |

### CommitSnapshot

- `FullHash`
- `Subject`
- `MessageBody`
- `MessageRaw`
- `AuthorEmail`
- `Parent`
- `IsRoot`
- `Files []ChangedFile`

### ChangedFile

- `StableID`
- `Status`: `A`, `M`, `D`, `R`, `C`, `B`, `S`, `ModeOnly`
- `OldPath`
- `Path`
- `Additions`
- `Deletions`

`Status`는 단일 typed enum으로 관리한다. `B`는 binary, `S`는 submodule/gitlink,
`ModeOnly`는 content 없이 mode만 바뀐 상태다. binary 여부를 별도 bool 조합으로
판단하지 않으며, renderer는 status enum을 통해 `No textual diff`, `Submodule
change`, `Mode-only change`를 선택한다.

### DiffWindow

- `FileID string`
- `Hunks []DiffHunk`
- `HasMore bool`
- `PartialReason string`

구현할 app-owned 최소 타입은 다음과 같다.

```go
type CommitInspectorReader interface {
    InspectCommit(context.Context, CommitRequest) InspectorResult[CommitSnapshot]
    LoadDiff(context.Context, DiffRequest) InspectorResult[DiffWindow]
}

type CommitRequest struct {
    Commit, RequestID string
    RepositoryEpoch uint64
}

type DiffRequest struct {
    Commit, Parent, FileID, RequestID string
    RepositoryEpoch uint64
    Window DiffWindowRequest
}

type DiffWindowRequest struct {
    StartLine, MaxLines, MaxBytes int
}

type InspectorPaneState string
const (
    PaneIdle InspectorPaneState = "idle"
    PaneLoading InspectorPaneState = "loading"
    PaneReady InspectorPaneState = "ready"
    PanePartial InspectorPaneState = "partial"
    PaneError InspectorPaneState = "error"
    PaneCanceled InspectorPaneState = "canceled"
)

type InspectorError struct {
    Kind string // timeout, git_exit, parse, invalid_input, binary, canceled
    Message string
    Retryable bool
}

type InspectorResult[T any] struct {
    State InspectorPaneState
    Value T
    Error *InspectorError
    Commit, Parent, FileID, RequestID string
    RepositoryEpoch uint64
}

`InspectorError.Message`는 adapter가 Git stderr에서 추출한 짧은 사용자용 문구다.
UTF-8 replacement, terminal control sequence 제거, 최대 표시 길이 제한을 adapter
경계에서 적용한다. 원본 stderr는 `InspectorError`나 renderer contract에 전달하지
않는다. renderer는 마지막으로 pane width에 맞춰 visible-width clamp만 수행한다.

```go
type InspectorPaneSnapshot struct {
    State InspectorPaneState
    Error *InspectorError
    RecoveryHint string
    RequestID string
}

type InspectorFocus string
const (
    FocusFiles InspectorFocus = "files"
    FocusDiff InspectorFocus = "diff"
)

type InspectorState struct {
    Open bool
    Commit CommitSnapshot
    Metadata, Files, Diff InspectorPaneSnapshot
    Focus InspectorFocus
    SelectedFileID string
    PairedRowIndex, HorizontalOffset int
    Window DiffWindow
}
```

`InspectorState`의 pane snapshot은 사용자에게 보이는 상태의 source of truth다.
`Commit`과 `Window`는 성공한 pane value이며, error/partial/canceled 상태에서도
다른 ready pane의 value를 지우지 않는다. cancel handle과 child context는 runtime
소유물이므로 DTO에 직렬화하지 않고 app state의 request controller가 관리한다.

type DiffHunk struct {
    Header string
    OldStart, OldCount, NewStart, NewCount int
    Rows []PairedRow
}

type PairedRow struct {
    ID string
    Kind LineKind
    From CodeLine
    To CodeLine
    FromWordSpans, ToWordSpans []WordSpan
    FromPresent, ToPresent bool
}

type CodeLine struct {
    Number int
    Text string
}

type WordSpan struct {
    Start, End int
    Kind LineKind
}

type LineKind string
const (
    ContextLine LineKind = "context"
    AddedLine LineKind = "added"
    RemovedLine LineKind = "removed"
    ModifiedLine LineKind = "modified"
    BinaryLine LineKind = "binary"
)
```

`CommitSnapshot.Author`와 `Parent`는 UTF-8 sanitized string이다. `Parent`는 일반
커밋의 first parent full hash, root commit의 빈 문자열이다. adapter는 NUL-delimited
raw path bytes로 `ChangedFile.StableID`를 계산한다:
`sha256(Status + "\x00" + rawOldPath + "\x00" + rawPath)`의 lowercase hex 값이다.
`OldPath`와 `Path`는 화면 표시용 sanitized string이며, raw path byte와의 매핑은
adapter가 immutable snapshot 안에서 보존한다. app은 raw path를 재조합하거나
정규화하지 않고 `StableID`만 `LoadDiff`에 전달한다.
R/C status는 Git name-status의 `R<score>`/`C<score>`를 `R`/`C`로 normalize하고
score는 표시하지 않는다. numstat의 두 값이 `-`이거나 binary patch marker가 있으면
`B` status로 매핑한다. gitlink/submodule 변경은 `S`, content 없이 mode만 바뀐
경우는 `ModeOnly`로 매핑하며, renderer는 typed status에 따라 안내를 선택한다.

전체 commit metadata는 `InspectCommit` 한 번의 snapshot으로 가져온다. diff window는
같은 commit/parent/file identity로 continuation한다. MVP에서는 retry 전용 key를 두지
않으며, `q`로 Graph에 돌아갔다가 같은 commit을 다시 열어 재시도할 수 있다.

ChangedFile 목록은 renderer용 flat projection으로 직접 변환한다. directory
collapse/expand 상태는 보관하지 않으며, 목록은 repository-relative path 기준으로
정렬하고 파일명 중심의 `../../filename` 표기를 사용한다.

```go
type TreeNodeKind string
const ( DirectoryNode TreeNodeKind = "directory"; FileNode TreeNodeKind = "file" )
type FileTreeNode struct {
    ID, Name, Path string
    Kind TreeNodeKind
    FileID string
    Expanded bool
    Children []FileTreeNode
}
```

목록 항목은 name 오름차순으로 정렬한다. 별도 pane focus는 제공하지 않으며,
file selection은 `j/k`로 이동하고 selection 변경 즉시 해당 file의 Diff를 갱신한다.

모든 request/result는 `Commit`, `Parent`, `FileID`, `Window`, `RepositoryEpoch`,
`RequestID`를 포함한다. `RepositoryEpoch`는 app과 reader가 공유하는 `uint64`이며
`0`은 초기 epoch다. commit request에서는 `Parent`와 `FileID`를 빈 값으로 둔다.
현재 commit, file, request, epoch와 일치하지 않는 결과는
버린다. Inspector 종료, 파일 전환, 새 request, repository mutation 시 진행 중인
작업을 취소한다.

취소의 소유자는 app의 `InspectorState`다. state는 현재 metadata/diff child context와
cancel handle을 보관하고, close·file switch·retry·새 window request 전에 기존
cancel을 호출한 뒤 request identity를 증가시킨다. Git adapter는 전달받은 context를
`exec.CommandContext`까지 연결한다. request identity 검사는 늦은 결과를 폐기하고,
cancel 호출은 실제 child process 종료를 보장한다. canceled 결과는 사용자 error로
렌더링하지 않는다.

## Git Semantics

- 일반 커밋은 commit과 parent 사이를 표시한다.
- merge commit은 first parent를 사용하고 header에 `FROM <parent>`를 표시한다.
- root commit은 빈 tree를 기준으로 `FROM ROOT`를 표시한다.
- file listing과 diff 조회는 동일한 commit/parent snapshot을 사용한다.
- rename은 `StableID`, `OldPath`, `Path`를 모두 유지하고 `old/path → new/path`로
  표시한다.
- Git path argument 앞에는 `--`를 둔다.
- commit hash, path, message, diff는 untrusted input으로 취급한다.

adapter는 다음 basis 규칙을 사용한다.

| Case | File listing and patch basis |
|---|---|
| root | `git diff-tree --root` with commit as the right side |
| normal | both listing and patch use explicit first parent and commit |
| merge | both listing and patch use explicit first parent and commit, never combined diff in MVP |

listing은 `--name-status -z -M -C`, additions/deletions와 binary는 `--numstat`,
text patch는 `-p --full-index --no-ext-diff`를 사용한다. 모든 file path argument는
`--` 뒤에 둔다. rename/copy의 old/new path는 NUL record에서 함께 읽고, patch 조회도
`OldPath`와 `Path`를 contract에 전달한다.

root 이외의 listing도 commit만 전달해 Git의 implicit merge diff semantics에 의존하지
않는다. metadata 단계에서 resolve한 동일한 `ParentSelection`을 listing과 patch 양쪽
명령의 `<parent> <commit>` 범위에 사용한다.

`InspectCommit`이 parent를 한 번 resolve한 뒤 `CommitSnapshot.Parent`에 저장한다.
후속 `LoadDiff`는 caller가 새 parent를 계산하지 않고 snapshot의 parent만 받는다.
이렇게 listing과 patch가 같은 snapshot을 사용하며, repository epoch가 바뀌면 두
작업 모두 폐기한다.

기존 `Runner.RunContext`는 다른 Git 호출자의 trim 계약을 유지한다. Inspector
adapter는 raw stdout을 보존하는 `RunRawContext` 또는 동등한 stream API를 사용해
message와 diff parser 입력을 trim하지 않는다. message의 final newline과 diff의
bounded line/byte 처리는 이 raw 경계에서 수행한다.

## Unified Diff Contract

wide mode의 기본 폭은 Changed files 30%, diff 70%다. compact mode는 files 28%, diff
72%다. narrow mode에서도 files와 diff를 함께 표시한다.

diff pane은 `kind marker | line number | code` 고정 열을 사용한다.

폭이 부족한 경우 file tree path는 `../../<filename>` 형태로 앞쪽 directory를
축약하고 파일명을 항상 보존한다. 선택된 파일의 전체 repository-relative path는
header의 `path:` 행에 표시한다.
diff code는 marker와 line number 고정 열을 먼저 확보한 뒤 남은 폭을 사용하며, 초과한
내용은 visible-width 기준으로 `…`를 붙여 절단한다. 경로와 코드 모두 raw byte
길이가 아니라 terminal cell width를 기준으로 계산한다.

- 신규 파일: added marker와 `+` row로 표시
- 삭제 파일: removed marker와 `-` row로 표시
- 수정 파일: 연속 삭제/추가 run을 원래 순서대로 deterministic하게 paired unified rows로 표시
- context line: context marker와 line number를 표시
- binary/submodule: `No textual diff`
- line number와 marker 열은 모든 diff row에서 유지

Pairing은 Tree-sitter나 의미 추론에 의존하지 않는다. word-level diff는 이미 생성된
modified pair 안에서만 수행한다.

1차 slice에서는 `WordSpan`을 생성하지 않는다. `FromWordSpans`와 `ToWordSpans`는
후속 highlighter/word-diff slice를 위한 확장 필드이며 항상 nil이어도 된다.

pairing 알고리즘은 다음과 같다.

1. hunk header를 읽어 old/new line cursor를 초기화한다.
2. context row는 동일 text와 양쪽 line number를 갖는 `ContextLine`으로 만든다.
3. 연속된 removed rows와 이어지는 added rows를 하나의 run으로 수집한다.
4. 두 run을 index 순서로 zip한다. 양쪽 모두 있으면 `ModifiedLine`, old만 있으면
   `RemovedLine`, new만 있으면 `AddedLine`으로 만든다.
5. 한쪽이 없는 row는 `FromPresent` 또는 `ToPresent`를 false로 두고 line number는
   0, renderer는 `—`를 표시한다.
6. malformed hunk/header는 추측하지 않고 `parse` error로 반환한다.

`WordSpan`은 `ModifiedLine`에서만 채우며 byte offset이 아닌 sanitized code의
display-rune offset이다. syntax/highlighter가 없어도 PairedRow와 semantic marker는
항상 유지한다.

## Message Contract (Follow-up)

MVP header는 subject만 표시한다. 전체 commit message 보존과 body/trailer viewport는
후속 계약으로 남기며, MVP keymap에는 message 전용 key를 두지 않는다.

- multiline body
- blank lines
- trailers
- final newline
- embedded newline

invalid UTF-8은 replacement policy를 적용하고 terminal control sequence는 표시 전에
제거한다. 전체 message viewport를 추가할 때도 selected file과 diff 위치는 유지한다.

## State and Key Contract

metadata, files, diff는 각각 다음 상태를 가진다.

```text
idle / loading / ready / partial / error / canceled
```

Inspector는 files와 diff를 항상 함께 표시한다. 선택 파일은 files pane에서 `>`로
표시하고 diff는 선택 파일에 종속된다.

breakpoint는 `width >= 80`이면 wide, `60 <= width < 80`이면 compact,
`width < 60`이면 narrow다. narrow에서도 files와 diff를 함께 표시한다.

| Key | 동작 |
|---|---|
| `j/k` | Changed files에서 이전/다음 파일 선택 및 diff 갱신 |
| `Ctrl+U/D` | 현재 선택 파일의 diff viewport page scroll |
| `q` | Inspector close |
| `Esc` | Inspector close |
| `?` | 전체 Inspector key 도움말을 열고 닫음 |

Inspector가 열린 동안 Graph paging, section switching, mutation key는 Graph로
전파하지 않는다. loading 중 중복 page/navigation request는 만들지 않는다.

key dispatch는 Inspector를 global overlay q→Esc rewrite보다 먼저 처리한다. 따라서
Inspector handler는 원본 `q`와 `Esc`를 모두 즉시 Graph로 복귀시키며, 파일 이동과
diff scroll은 Inspector 내부에서 소비한다.
다른 popup의 기존 q→Esc 계약은 이 분기 밖에서 유지한다.

한 pane의 실패가 다른 ready pane을 가리지 않는다. canceled는 사용자 오류로
표시하지 않는다.

각 pane은 독립적으로 상태를 렌더링한다. Metadata error는 commit identity 영역에
오류를 표시하고, 파일 목록이 없는 경우에만 Files pane을 `No file list`로 표시한다.
Files가 ready인 상태에서 Diff가 empty/error/partial이면 header와 flat file list를 유지한 채
Diff 영역에만 `No textual changes`, 원인 또는 continuation을 contextual하게 표시한다.
오류가 난 request는 `q`로 닫고 Graph에서 다시 진입해 재시도한다. 한 pane의 오류가 이미 ready인 다른 pane의
value나 탐색 가능성을 대체하지 않는다.

우선순위는 항상 `q` close가 먼저이며, `q`와 `Esc`는 어느 상태에서도 즉시 Graph로
복귀한다.

footer에는 `q close`, `Esc back`, `? help`만 표시한다. 전체 key contract는 `?`
도움말 화면에서 확인할 수 있으며, 도움말을 닫아도 선택 파일과 scroll 위치를
보존한다. footer와 도움말은 narrow mode에서도 close/back action을 생략하지 않는다.

각 request에는 3초 deadline을 적용한다. deadline 초과는 해당 pane의 error이며,
`q`로 닫고 Graph에서 다시 열어 재시도할 수 있다. `context.Canceled`는 canceled 상태로만 기록하고 사용자
오류 문구로 표시하지 않는다. subprocess는 `exec.CommandContext`로 연결하며,
close/file change 시 child process가 종료되었음을 fake runner로 검증한다.

상태별 사용자 화면은 다음으로 고정한다.

| State | Required rendering |
|---|---|
| loading | identity/path 유지, skeleton 또는 `Loading`, cancel/close action |
| ready | 정상 content와 selected-file marker |
| partial | loaded rows, `partial`, continuation hint |
| error | 원인, 영향 pane, `Esc` |
| canceled | content mutation 없음, 사용자 error 문구 없음 |
| empty | `No changed files`, `No textual changes`, 또는 `No commit message body` |

metadata와 Files가 ready가 되면 첫 visible file을 flat list 정렬 순서에서 deterministic하게
자동 선택하고 해당 file의 첫 diff request를 즉시 시작한다. Diff pane은 같은 선택
파일의 loading/ready 상태를 함께 렌더링한다. 변경 파일이 없으면 자동 선택과 diff
request를 만들지 않고 contextual empty state를 표시한다.

loading 화면은 실제 코드로 오인될 수 있는 가짜 diff block을 생성하지 않는다. 고정된
pane header, identity/path, `Loading…`, 제한된 placeholder row와 cancel/close action만
표시하며, animation, gradient, 중첩 card, 장식용 큰 여백은 사용하지 않는다.

`DiffWindowRequest.StartLine`은 raw diff line이 아니라 paired row의 zero-based index다.
각 window는 hunk header를 포함한 첫 row에서 시작하고, continuation은 마지막 반환
paired row 다음 index부터 시작한다. adapter 내부 raw line offset은 외부 contract로
노출하지 않는다. parser는 window 시작에 필요한 hunk header와 old/new line cursor를
복원한 뒤 paired rows를 생성한다. hunk 내부 `Rows`를 canonical source로 두며,
동일한 paired row를 top-level에 중복 저장하는 계약은 두지 않는다. renderer/app projection이
`FlattenHunks(Hunks)`를 통해 현재 window의 평탄화 view를 생성한다.

## Special States

| Case | 표시 |
|---|---|
| root commit | `FROM ROOT` |
| merge commit | first-parent identity와 `FROM <parent>` |
| new file | from pane `—` |
| deleted file | to pane `—` |
| rename | `old/path → new/path` |
| binary/submodule | `No textual diff` |
| empty commit | `No changed files` |
| partial diff | loaded rows와 continuation hint |
| pane error | 원인, `Esc` |

## Responsive and Colorless Contract

Inspector frame과 pane은 기존 `popupBorder`, `baseBox`, `activeBox`, theme의 semantic
color 및 repository의 visible-width helper를 재사용한다. Inspector 전용 색상,
border 문자, spacing scale을 추가하지 않는다. pane 구분은 기존 frame/border와
header hierarchy와 selected-file marker로 표현한다.

최소 snapshot matrix는 `(40,12)`, `(40,20)`, `(40,30)`, `(60,12)`, `(60,20)`,
`(60,30)`, `(80,12)`, `(80,20)`, `(80,30)`이다.

`NO_COLOR=1`에서도 다음은 문자열 또는 marker로 구분된다.

- `A/M/D/R`
- `+`, `-`, context marker
- selected file
- `Diff`
- close action
- `ROOT COMMIT` 또는 parent identity

구체적으로 selected file에는 `>`, diff pane header에는 `Diff`를 항상 출력한다.
file status의 `A/M/D/R/C/B/S/ModeOnly`와 diff row의
`+`, `-`, context marker도 색상과 무관하게 유지하며, narrow mode에서도 이 marker
열을 제거하지 않는다.

wide rune, combining mark, tab, invalid UTF-8, terminal control sequence, long
unbroken path에서도 pane alignment와 visible width를 보존한다.

renderer 함수는 전달받은 `width`, `height`에 대해 outer frame의
`lipgloss.Width == width`, `lipgloss.Height == height`를 보장한다. width가 40보다
작으면 narrow contract를 유지한 채 footer와 close action을 보존하며, height가
12보다 작으면 message/legend를 접고 identity, selected path, pane, close action을
우선 표시한다.

각 diff request의 기본 cap은 `MaxLines=2000`, `MaxBytes=1 MiB`다. 둘 중 하나를
초과하면 이미 읽은 rows만 `partial`로 반환하고 `HasMore=true`로 표시한다. reader는
전체 stdout을 `bytes.Buffer`에 쌓지 않고 line/byte bounded reader를 사용한다.
continuation은 이전 window의 마지막 paired row 다음을 `StartLine`으로 요청한다.

continuation 성능을 위해 adapter는 `commit/parent/file/repositoryEpoch`를 cache key로
하는 bounded checkpoint cache를 둔다. checkpoint는 가장 가까운 parsed paired-row
boundary의 raw offset, hunk header, old/new line cursor만 보관하며 전체 rows나 stdout을
보관하지 않는다. cache는 최대 entry/byte 수를 고정하고 LRU 또는 동등한 eviction을
사용한다. 새 epoch·file 전환·Inspector close에서는 해당 key를 폐기하며, cache miss는
stream을 재실행하되 checkpoint가 있으면 그 위치에서 재개한다.

## Acceptance Criteria

1. Graph에서 커밋 선택 후 `Enter`를 누르면 Graph body가 사라지고 Inspector가 body
   bounds를 border로 채운다.
2. Header에 `commit:`, `message:`, `author:`, parent 또는 `ROOT COMMIT`가 표시된다.
3. Changed files가 왼쪽 tree pane에 표시된다.
4. wide mode에서 Changed files와 unified Diff가 동시에 표시된다.
5. 신규/삭제/수정 파일의 added/removed/context marker와 line number가 표시된다.
6. 수정 파일의 paired unified rows가 deterministic하게 정렬된다.
7. binary/submodule은 `No textual diff`를 표시한다.
8. merge commit은 first parent를 사용하고 header에 표시한다.
9. Header에 subject가 `message:` 행으로 표시된다. 전체 commit message/trailer viewport는 MVP 범위가 아니다.
10. delayed A→B→A 결과가 현재 선택을 덮지 않는다.
11. Inspector close 시 진행 중인 reader 작업이 취소된다.
12. diff 실패 시 files와 metadata는 계속 탐색 가능하고 `Esc` 또는 `q`로 Graph에 복귀할 수 있다.
13. `q`는 어느 pane에서도 즉시 Graph로 복귀한다.
14. `Esc`는 Diff에서 Files, Files에서 Graph로 복귀한다.
15. 40/60/80 폭과 12/20/30 높이 matrix가 통과한다.
16. `NO_COLOR=1`에서도 semantic marker와 방향 정보가 유지된다.
17. large diff는 전체 stdout을 materialize하지 않고 `MaxLines=2000` 또는
    `MaxBytes=1 MiB` cap 안에서 bounded stream/window로 처리한다.
18. `go test ./...`와 `scripts/check`가 통과한다.
19. Inspector 외부의 Graph paging, search, section switching, mutation 동작이
    변경되지 않는다.

## Testing Plan

### Pre-implementation baseline gate

Inspector 구현 전에 현재 기준점에서 `go test ./...`와 `scripts/check`를 실행하고,
실패한 test name·stack·commit을 baseline artifact에 기록한다. working tree에 이미
있는 변경은 보존하며, baseline 실패와 Inspector 신규 실패를 같은 green/red 판정에
섞지 않는다. baseline이 깨진 상태라면 구현 완료는 “Inspector 신규 경로 통과,
baseline failures unchanged”로만 표시하고, 최종 ship 전에는 baseline failure의
소유자와 해결 계획을 별도로 확정한다.

| Layer | 범위 | 최소 수량 |
|---|---|---:|
| Unit | subject/path parsing, hunk pairing, binary detection | 12 fixtures |
| App state | fake reader loading/ready/error/cancel/stale | 10 flows |
| Git integration | root, merge, rename, binary, empty, hostile path/message | 6 fixtures |
| Renderer snapshot | 9 폭/높이 조합과 NO_COLOR 변형 | 18 snapshots |
| Performance | bounded large diff reader | 1 regression |
| Regression | 기존 Graph navigation/key binding | 전체 기존 테스트 |

정확한 테스트 파일은 다음과 같다.

- `internal/app/commit_inspector_contract_test.go`: DTO, tree, pairing contract
- `internal/app/commit_inspector_state_test.go`: fake reader state/key transition,
  q/Esc dispatch, pane-local error/partial/canceled matrix와 ready pane 보존
- `internal/app/commit_inspector_view_test.go`: 18 renderer snapshots
- `internal/git/commit_inspector_integration_test.go`: temporary Git fixture,
  raw path StableID invariants, hostile rename mapping, display-only sanitization,
  A/M/D/R/C/B/S/ModeOnly mapping and visible-state table
- `internal/git/commit_inspector_stream_test.go`: byte/line cap과 child cancellation,
  trim/raw runner dual contract, full-message final newline 보존, hunk/context/
  deletion/addition/byte-cap continuation equivalence, hostile stderr error
  sanitization와 canceled 사용자 문구 부재

snapshot pass/fail 기준은 frame width/height, required marker 문자열, pane header,
selected path, footer action의 존재 여부다. A→B→A fixture는 delayed fake reader로
실행하고, B request 결과가 A를 덮지 않는 것을 assertion한다. large fixture는 10 MiB
이상 patch를 만들고 반환 payload가 1 MiB cap을 넘지 않으며 `HasMore=true`인지
검증한다.

`commit_inspector_stream_test.go`의 fake runner는 stdout을 한 번에 반환하는
`gitRaw` 경로가 호출되지 않았음을 확인하고, reader가 소비한 최대 bytes가
`MaxBytes + 4 KiB` 이하인지 기록한다. cancellation test는 context cancel 뒤
`cmd.Wait`가 반환되고 child process가 남지 않는 것을 assertion한다. terminal safety
test는 rendered output에 ESC/C0 control byte가 없음을 확인한다.

5초 기준은 고정 temporary repository fixture에서 `Enter` event timestamp와 first
ready frame timestamp의 차이로 측정한다. 10회 실행 중 p95가 5초 이하이고, 첫
loading frame은 Bubble Tea update 1회 이내에 출력되어야 통과한다.

## Rollback Plan

Inspector 변경을 기능 단위 커밋으로 분리한다. runtime crash, terminal corruption,
stale result, 잘못된 diff 표시가 발견되면 Inspector 관련 커밋만 revert하여 기존 Graph
동작으로 복구한다. DB, remote ref, working tree를 변경하지 않으므로 별도 migration
rollback은 없다.

## Effort Estimate

| 영역 | 예상 human effort |
|---|---:|
| Screen boundary, state, cancellation | 2~3일 |
| Git/message/basis contract | 2~3일 |
| grouped-files/unified-diff renderer | 2~3일 |
| bounded stream/cache | 1~2일 |
| semantic marker/highlighter fallback | 1~2일 |
| tests/docs/Taskmaster sync | 2~3일 |

## Files Reference

| 파일 | 변경 |
|---|---|
| `internal/app/model.go` | Inspector state, pane state, request identity |
| `internal/app/update.go` | result validation, stale discard, pane transitions |
| `internal/app/commit_inspector.go` | key handling, fixed-header screen renderer, unified diff viewport |
| `internal/app/view_shell.go` | Graph 대체 및 Inspector frame |
| `internal/app/view_overlays.go` | Inspector를 일반 overlay stack에서 제거 |
| `internal/app/commit_inspector_contract.go` | app-owned reader port와 DTO 신규 |
| `internal/git/repo.go` | snapshot, message, structured diff model |
| `internal/git/repo_exec.go` | full message, shared first-parent, bounded reader |
| `internal/app/commit_inspector_contract_test.go` | DTO, tree, pairing contract |
| `internal/app/commit_inspector_state_test.go` | fake-reader, key/state, stale/cancel |
| `internal/app/commit_inspector_view_test.go` | renderer snapshots, width/height, NO_COLOR |
| `internal/git/commit_inspector_integration_test.go` | temporary Git fixture |
| `internal/git/commit_inspector_stream_test.go` | byte/line cap과 process cancellation |
| `README.md` | Graph `enter`와 Inspector key 안내 |
| `docs/decisions.md` | 최종 구현 결정 동기화 |
| `.taskmaster/tasks/tasks.json` | `commit detail` subtask 상태 동기화 |

## Engineering review decisions (2026-08-06)

이번 engineering review에서 다음 결정을 확정했다. 이 목록은 구현자가 이전 초안의
구형 from/to footer, message pane, keymap을 다시 구현하지 않도록 하는 실행 기준이다.

- MVP는 bordered 독립 screen, 4행 고정 header, flat changed-files list, unified diff,
  `j/k`, `Ctrl+U/D`, `q/Esc/?`, semantic marker/color만 포함한다.
- metadata와 selected-file diff는 별도 pane state와 request/cancel lifecycle을 갖는다.
  loading 중에도 `q`, `Esc`, `?`는 소비하며 stale result는 버린다.
- metadata에서 parent를 한 번 확정하고 changed-files와 diff가 같은 explicit basis를
  사용한다. rename/copy는 `OldPath`, `Path`, stable ID와 실제 diff를 연결한다.
- app은 최소 `CommitInspectorReader` port와 DTO만 소비하고, Git subprocess와 parser는
  adapter에 둔다. renderer는 raw diff 문자열을 재파싱하지 않는다.
- status는 typed enum과 `Unknown/Unsupported`를 사용한다. metadata 조회는 batch화하고
  파일별 N+1 Git 호출을 만들지 않는다.
- byte/line cap 초과 시 child process를 종료하고 partial 결과를 반환한다. file list는
  immutable sorted projection으로 유지한다.
- `Tab`, `m`, `r`, `h/l`, `Enter`, message footer와 전체 message viewport는 MVP에서 제거한다.
  Tree-sitter grammar/query, word-level diff와 syntax token은 후속 TODO다.

### Data flow

```text
Graph Enter
   │ commit hash + repository epoch
   ▼
InspectorState ── CommitInspectorReader ── Git adapter
   │                    │                    ├─ resolve parent once
   │                    │                    ├─ batch metadata/path/status
   │                    │                    └─ bounded structured diff
   │                    ▼
   ├─ metadata/files snapshot ──┐
   └─ selected FileID + window ─┴─ request/epoch validation
                                  │ stale/canceled → discard
                                  ▼
                 tree projection + DiffHunk/PairedRow renderer
                                  │
                 bordered 4-line header / body / q-Esc-? footer
```

### Failure modes

| Failure | Test | Error handling | User-visible result |
|---|---|---|---|
| stale file response | fake reader state test | request/epoch discard | current selection remains |
| close during Git process | cancellation/stream test | context + child termination | immediate Graph return |
| metadata batch/parse failure | Git fixture + error test | typed sanitized error | files/header context preserved |
| oversized diff | bounded stream test | partial result + continuation hint | readable partial diff |
| malformed hunk | parser table test | parse error | diff pane error, tree remains usable |
| hostile path/message/ANSI | sanitization fixture | adapter sanitization | safe display, stable raw ID |
| narrow/wide terminal | renderer matrix | visible-width clamp | border/markers remain aligned |

현재 계획 기준 critical silent failure는 없다. 각 failure는 최소 하나의 테스트와
사용자에게 보이는 복구 또는 안전한 discard 경로를 가져야 한다.

## NOT in scope

- 실제 Tree-sitter grammar/query bundle
- merge parent selector와 combined diff
- blame
- symbol navigation
- AST semantic diff
- word-level diff와 실제 syntax highlighting
- staging/editing
- GitHub Issue 생성

## What already exists

- `internal/app/key_handling_browse.go`: Graph `Enter`와 선택 commit hash 진입점이
  부분 구현되어 있다.
- `internal/app/key_handling.go`, `view_shell.go`, `view_overlays.go`: Inspector를
  Graph shell보다 우선 렌더링하고 일반 overlay stack에서 분리하는 출발점이 있다.
- `internal/app/model.go`, `update.go`: request/epoch 검사와 단일 cancellation의
  부분 구현이 있다. 최신 `InspectorState`/pane state로 정리해야 한다.
- `internal/git/repo.go`, `repo_exec.go`: NUL path parsing, raw runner,
  `exec.CommandContext`, bounded reader와 paired-row parser의 부분 구현이 있다.
- `internal/app/commit_inspector_test.go`, `internal/git/commit_inspector_diff_test.go`:
  q/Esc, 기본 frame, paired row, terminal sanitization의 최소 테스트가 있다.
- 기존 Graph/overlay visible-width와 ANSI 정책은 재사용할 수 있지만, Inspector 전용
  semantic style와 terminal matrix 테스트는 추가해야 한다.

## Implementation Tasks

Synthesized from the 2026-08-06 Engineering review. 구현 전 각 task의 테스트를 먼저
작성하고, 소스 구현은 이 스펙의 MVP Required 범위만 대상으로 한다.

- [ ] **E1 (P1, human: ~1d / CC: ~15min)** — Inspector state/controller — `InspectorState`, metadata/files/diff pane state, cancellation, q/Esc/? dispatch, request/epoch validation 구현
  - Surfaced by: Architecture D5, Code Quality D9/D10, Test D15/D16
  - Files: `internal/app/model.go`, `internal/app/update.go`, `internal/app/messages.go`, `internal/app/key_handling.go`, `internal/app/commit_inspector.go`
  - Verify: fake-reader `Enter → loading → ready/error`, close while loading, stale A→B→A, Graph cursor preservation
- [ ] **E2 (P1, human: ~1d / CC: ~15min)** — reader boundary and Git basis — app-owned reader DTO, author email, one parent snapshot, explicit root/normal/merge basis, rename/copy path contract, typed status/error 구현
  - Surfaced by: Architecture D2/D4/D8, Code Quality D10/D11
  - Files: `internal/app/commit_inspector_contract.go`, `internal/git/repo.go`, `internal/git/repo_exec.go`
  - Verify: author email, parent arguments, `OldPath`/`Path`, stable ID, A/M/D/R/C/B/S/ModeOnly/Unknown, sanitized retryable errors
- [ ] **E3 (P1, human: ~1d / CC: ~15min)** — batch metadata and bounded structured diff — remove per-file Git N+1, parse `DiffHunk/PairedRow` once, preserve line numbers and partial state, terminate capped child process
  - Surfaced by: Architecture D6, Performance D21/D22/D23
  - Files: `internal/git/repo_exec.go`, `internal/app/commit_inspector_contract.go`, `internal/app/commit_inspector.go`
  - Verify: batch subprocess count, pairing table, cap/partial, child termination, adapter output consumed without renderer reparse
- [ ] **E4 (P1, human: ~1d / CC: ~15min)** — file tree projection and screen renderer — immutable tree index, visible projection, 4-line fixed header, shared border, unified diff, loading/empty/error states, visible-cell truncation 구현
  - Surfaced by: Architecture D3/D7, Code Quality D12
  - Files: `internal/app/model.go`, `internal/app/commit_inspector.go`, `internal/app/view_shell.go`, `internal/app/view_overlays.go`
  - Verify: grouped tree, selected FileID, root/rename labels, `(120,24)`, `(80,20)`, `(60,12)`, `(40,12)` dimensions
- [ ] **E5 (P1, human: ~0.5d / CC: ~10min)** — semantic style and keymap cleanup — A/M/D/R/C/B/S/ModeOnly colors, +/-/context markers, selected reverse/bold + `>`, NO_COLOR fallback, remove Tab/m/r/h/l/Enter/message footer
  - Surfaced by: Architecture D7, Code Quality D13/D14
  - Files: `internal/app/commit_inspector.go`, `internal/app/theme.go`, `internal/app/key_handling.go`
  - Verify: keymap matrix, help/footer contract, ANSI and `NO_COLOR`, Graph key propagation blocked
- [ ] **E6 (P1, human: ~1d / CC: ~15min)** — app state and renderer tests — fake reader transitions, stale/cancel, keymap, tree, semantic markers, narrow/low-height matrix
  - Surfaced by: Test D15/D16/D18/D20
  - Files: `internal/app/commit_inspector_contract_test.go`, `internal/app/commit_inspector_state_test.go`, `internal/app/commit_inspector_view_test.go`
  - Verify: baseline-separated `go test ./...`, exact frame dimensions, first diff ready, q/Esc/? and Ctrl+U/D behavior
- [ ] **E7 (P1, human: ~1d / CC: ~15min)** — Git integration and parser fixtures — temporary repository for root/normal/merge/rename/copy/binary/submodule/mode-only/hostile input and malformed hunk tables
  - Surfaced by: Test D17/D19
  - Files: `internal/git/commit_inspector_integration_test.go`, `internal/git/commit_inspector_stream_test.go`, `internal/git/commit_inspector_diff_test.go`
  - Verify: same parent basis, `--` path boundary, status mapping, malformed/uneven pairing, bounded cancellation
- [ ] **E8 (P2, human: ~0.5d / CC: ~5min)** — implementation handoff verification — update README, decisions, Taskmaster and run standard checks
  - Surfaced by: review completion and repository handoff requirements
  - Files: `README.md`, `docs/decisions.md`, `.taskmaster/tasks/tasks.json`
  - Verify: `go test ./...`, `scripts/check`, `git diff --check`, Taskmaster JSON validity

### Engineering review artifacts

- Test plan: `/Users/hrk/.gstack/projects/hrllk-graphkeeper/hrk-develop-eng-review-test-plan-20260803.md`
- Task aggregation JSONL: `/Users/hrk/.gstack/projects/hrllk-graphkeeper/tasks-eng-review-20260806-215442.jsonl`

## Engineering review completion summary (2026-08-06)

- Scope: 최소 MVP로 축소했고, Tree-sitter·word diff·전체 message viewport·고도화
  continuation cache는 후속 범위로 명시했다.
- Architecture: 7개 이슈. 모두 A 선택; parent basis, tree, reader boundary,
  pane state, structured diff, 최신 screen contract, rename diff를 확정했다.
- Code Quality: 6개 이슈. 모두 A 선택; state consolidation, typed/sanitized error,
  Unknown status, visible width, semantic style, keymap cleanup을 확정했다.
- Test: coverage diagram 작성, 6개 gap을 E6/E7에 반영했다.
- Performance: 4개 이슈. 모두 A 선택; batch metadata, child termination, single
  structured parse, visible tree projection을 확정했다.
- Failure modes: critical silent gap 0. Outside voice는 실행하지 않았다.
- Parallelization: 3 lanes. Lane A app state/tree/renderer, Lane B Git adapter/stream을
  병렬 시작하고, 둘을 합친 뒤 Lane C 통합·matrix 테스트를 순차 실행한다.
- Lake Score: 23/23 recommendations chose complete option.
- 구현 전 baseline gate를 유지한다. 기존 테스트 실패와 Inspector 신규 실패를 분리한다.

## Previous Design review completion summary (superseded)

이전 Design review는 from/to side-by-side, `Tab`, `m`, 전체 message viewport를
전제로 했다. 2026-08-06 Engineering review에서 확정한 4행 header, unified diff,
단순화 keymap과 충돌하므로 구현 기준으로 사용하지 않는다. 최신 시각 검수는 구현 후
별도 `/plan-design-review`에서 다시 진행한다.

## Related

- Graph Commit Inspector 계획 문서
- `TODOS.md`
- `.taskmaster/tasks/tasks.json`

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|---|---|---|---:|---|---|
| Eng Review | `/plan-eng-review` | Architecture, code quality, tests, performance | 2 | CLEAN | 23 findings resolved; 0 critical gaps; E1–E8 locked |
| Design Review | `/plan-design-review` | UI/UX validation | 1 | REVISION REQUIRED | Previous review is superseded by the unified-diff/header/footer UX revision |

**VERDICT:** ENG CLEARED — the minimum MVP is ready for implementation. The previous Design
review remains superseded by the latest fixed-header/unified-diff/keymap contract; run a
fresh Design review before treating visual acceptance as final.

NO UNRESOLVED DECISIONS
