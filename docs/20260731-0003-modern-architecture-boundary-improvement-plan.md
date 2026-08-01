<!-- /autoplan restore point: /Users/hrk/.gstack/projects/hrllk-graphkeeper/develop-autoplan-restore-20260731-182426.md -->

# Graphkeeper 점진적 구조개선 계획

상태: T4~T6 1차 구현 완료, 최종 리뷰 대기
브랜치: `develop`
작성일: 2026-07-31
선택한 방향: A. 점진적 경계 개선

## 1. 목표

Graphkeeper를 전면 재작성하지 않고, 2026년 기준으로 다음 책임이 서로 명확하게 분리된 구조로 이동한다.

```text
Bubble Tea 입력/수명주기
        ↓
애플리케이션 상태 전이와 workflow orchestration
        ↓
Git 읽기/쓰기 adapter 및 순수 도메인 판정
        ↓
Graphkeeper의 사용자-visible 상태와 렌더링
```

현재 기능의 의미는 보존한다.

- Graph / Current / Remote / Tags 탐색
- 키 바인딩과 popup 흐름
- Git fetch, pull, merge, rebase, reset, stash, tag, branch 작업
- Graph topology, provenance, conflict, ANSI 색상 표시
- 기존 CLI 동작과 `scripts/*` 검증 진입점

목표는 파일을 많이 만드는 것이 아니다. 신규 기여자가 `입력 → 상태 전이 → Git 작업 → 메시지 → 렌더링`의 흐름을 한 번에 따라갈 수 있게 만드는 것이다.

## 2. 현재 구조 진단

### 2.1 핵심 문제

| 문제 | 현재 증거 | 사용자/개발자 영향 |
|---|---|---|
| 실행 상태와 화면 상태가 한 모델에 밀집 | `internal/app/model.go`의 repo snapshot, cursor, popup, search, input 필드 혼재 | 한 기능 수정이 unrelated UI 상태를 건드릴 위험이 큼 |
| 명령 생성과 Git orchestration 결합 | `internal/app/commands.go`가 fetch/status/tag snapshot/write/execute를 직접 조합 | 비동기 실패와 재조회 정책을 테스트하기 어렵고 중복이 늘어남 |
| `git.Status`가 앱 전역 DTO처럼 사용 | `internal/git/repo.go`의 큰 Status와 `internal/app`의 직접 의존 | Git parser 변경이 렌더러와 workflow까지 전파됨 |
| 메시지 타입과 update routing 분산 | `messages.go`, `update_*.go`, `key_handling_*.go`에 workflow별 규칙 분산 | 이벤트가 어느 상태를 바꾸는지 추적 비용이 높음 |
| 렌더링과 정책 판정 경계가 흐림 | `graph_render*.go`, `view_*.go`, `target_items.go`, action helpers 혼재 | 같은 정책이 Graph와 section/popup에서 다르게 표현될 가능성이 있음 |
| 테스트가 대형 단일 파일에 집중 | `model_test.go` 3,744줄, `key_handling_test.go` 1,885줄 | 실패 원인 탐색과 신규 테스트 배치가 느림 |
| 과거 구조개선 계획이 누적 | `docs/archive/202606*`와 `docs/model-refactor-plan.md` | 무엇이 완료됐고 무엇이 아직 목표인지 판단하기 어려움 |

### 2.2 잘 작동하는 기존 경계

- `internal/git`: Git 명령 실행과 parsing을 소유한다.
- `internal/graph`: graph topology와 lane 계산을 순수 로직으로 유지한다.
- `internal/state`: 사용자-visible mode/action/block/target 상태를 표현한다.
- `internal/app/view_*`: 화면별 렌더링 파일 분할이 이미 시작되어 있다.
- `target_items.go`: browse와 action flow가 공유하는 target 정책을 한 곳에 모은다.
- `scripts/check`, `scripts/test`, `scripts/build`: 변경 후 검증 진입점이 고정되어 있다.

이 경계를 폐기하지 않고, 경계 사이의 계약을 명확히 하는 것이 기본 전략이다.

## 3. 확정된 전략과 대안

### A. 점진적 경계 개선 (선택)

작은 수직 단위마다 계약, 테스트, 이동을 함께 완료한다. 먼저 `model`/message/state를 정리하고, 그 다음 workflow orchestration, rendering, Git adapter 순서로 이동한다.

- 장점: 회귀 범위가 작고 각 단계가 revert 가능하다.
- 단점: 중간 단계에서 임시 compatibility shim이 잠시 존재한다.
- 규모: human 2–4일, CC 30–60분 단위 작업 여러 회.

### B. 전면 package 재작성

`internal/app`, `internal/domain`, `internal/infra/git`, `internal/ui`를 먼저 만들고 모든 코드를 한 번에 이동한다.

- 장점: 목표 구조가 빠르게 보인다.
- 단점: Bubble Tea 상태, Git DTO, 렌더링 테스트가 동시에 흔들려 회귀 원인 추적이 어렵다.
- 규모: human 1–2주, CC 2–4시간.

### C. UI 파일 분할만 수행

현재 `docs/20260731-0002-project-ui-readability-improvement-plan.md`처럼 layout과 Graph readability만 개선한다.

- 장점: 사용자 체감이 빠르고 구현 위험이 낮다.
- 단점: runtime model과 Git orchestration의 결합을 해결하지 못한다.

A를 선택한 이유: 구조의 핵심 위험은 파일 수가 아니라 책임 간 결합이다. 먼저 동작을 고정하고 경계를 이동해야 장기 구조와 단기 안정성을 함께 얻을 수 있다.

## 4. 목표 구조

```text
cmd/graphkeeper/
  main.go                         # CLI parsing, repo open, app bootstrap only

internal/app/
  app.go                          # Bubble Tea model construction and lifecycle
  runtime_state.go                 # cursor, viewport, popup/input runtime state
  screen_state.go                  # current screen/status projection
  messages.go                      # event types and payload contracts
  update.go                        # top-level event routing only
  update_lifecycle.go              # load/refresh state transitions
  update_actions.go                # action execution result transitions
  key_handling_*.go                # input-to-intent mapping only
  commands/                        # optional later seam; command builders after contracts settle
  view_*.go                        # screen composition and rendering
  policy/                          # only if shared pure policy exceeds app-local scope

internal/graph/
  topology and navigation rules    # no Bubble Tea or Git command dependency

internal/git/
  repo.go                          # public repository facade
  repo_exec.go                     # command runner and timeout boundary
  parse_*.go                        # Git output parsing
  snapshot.go                       # repository snapshot assembly, if proven useful

internal/state/
  screen.go                         # user-visible screen/action states
  target.go                         # target contracts
  status.go                         # immutable transition helpers or reducers

internal/telemetry/
  telemetry.go                      # local diagnostic side effects only
```

`commands/`, `policy/`, `snapshot.go` 같은 새 package/file은 이동 대상과 계약이 실제로 확인된 뒤 추가한다. 구조를 예쁘게 보이게 만들기 위해 빈 package를 먼저 만들지 않는다.

## 5. 책임과 의존성 규칙

### 5.1 허용 방향

```text
cmd → app → state
          ├→ graph
          ├→ git
          └→ telemetry (진단 경계)

graph → 표준 라이브러리만
state → 표준 라이브러리만
git → telemetry 및 표준 라이브러리
view → state/graph/app의 render input
```

### 5.2 금지 규칙

1. `internal/graph`와 `internal/state`는 Bubble Tea, Lip Gloss, `internal/git`를 import하지 않는다.
2. renderer는 Git 명령을 실행하지 않는다.
3. key handler는 Git 명령을 직접 실행하지 않고 intent 또는 `tea.Cmd` 경계를 호출한다.
4. `git.Status`를 화면 전용 필드처럼 확장하지 않는다. repository snapshot과 screen state의 책임을 구분한다.
5. `state.Status`에 terminal width, cursor, popup draft 같은 runtime UI 필드를 넣지 않는다.
6. `commands.go`에 새로운 기능을 계속 추가하지 않는다. workflow가 커지면 입력, 실행, 결과 반영을 별도 계약으로 나눈다.
7. 순환 import를 피하기 위해 consumer가 필요한 최소 interface를 consumer package에서 정의한다.

## 6. 점진적 구현 순서

### Phase 0. 현재 동작 고정

- `go test ./...`, `scripts/check`, `scripts/build` 기준 결과 기록
- Graph/section/popup의 대표 출력과 key flow를 golden-like string/visible-width 테스트로 고정
- `model_test.go`, `key_handling_test.go`의 영역별 책임을 목록화
- 기존 archive 계획 중 실제 미완료 항목을 현재 코드와 대조

완료 기준: 구조 이동 전 실패가 기능 회귀인지 baseline 차이인지 구분 가능해야 한다.

### Phase 1. runtime model / screen state / messages 분리

대상:

- `internal/app/model.go`
- `internal/app/messages.go`
- `internal/state/state.go`
- 관련 `update_*.go`, tests

작업:

1. `model`을 Bubble Tea runtime container로 정의한다.
2. repo snapshot, cursor/scroll, popup/input, async flags를 묶되 별도 타입으로 분리한다.
3. `state.Status`는 user-visible action state만 남긴다.
4. message payload가 raw repository state와 error를 어떻게 포함하는지 명시한다.
5. message type 이름을 `Loaded`, `RefreshCompleted`, `ActionCompleted`처럼 lifecycle 기준으로 통일한다.
6. 기존 message alias는 한 단계 동안 유지하고 테스트 통과 후 제거한다.

완료 기준: `model.go`에서 상태 전이 정책과 문자열 렌더링이 사라지고, `state`가 runtime UI 필드를 소유하지 않는다.

### Phase 2. workflow orchestration 경계

대상:

- `internal/app/commands.go`
- `internal/app/actions.go`
- `internal/app/update_fetch.go`
- `internal/app/update_execute.go`
- tag/stash/branch/cherry-pick workflow files

작업:

1. 각 workflow를 `load → validate → execute → reload → present result` 단계로 표로 만든다.
2. 중복되는 reload/tag snapshot/error mapping을 작은 순수 helper로 통합한다.
3. 명령 함수는 Git adapter 호출과 message 생성만 담당하게 한다.
4. 정책 판정(`canCreateBranch`, `pullReady`, target selection)은 pure function으로 유지한다.
5. Git mutation 직전 검증은 workflow 경계에서 한 번만 수행한다.
6. 실패 시 partial status와 사용자-facing recovery message를 보존한다.

완료 기준: 동일한 Git 작업의 재조회·오류 매핑이 여러 파일에 복제되지 않고, workflow별 happy/error/partial 테스트가 존재한다.

### Phase 3. Git adapter와 repository snapshot 계약

대상:

- `internal/git/repo.go`
- `internal/git/repo_exec.go`
- `internal/git/repo_parse.go`
- `internal/app`의 직접적인 `Repo` 사용 지점

작업:

1. app consumer가 필요로 하는 최소 repository interface를 app 경계에서 정의한다.
2. concrete `*git.Repo`는 adapter 구현으로 남긴다.
3. 읽기 snapshot과 mutation operation의 호출 계약을 구분한다.
4. `git.Status`에 Git parser 내부 임시값과 화면에 필요한 안정 필드를 혼합하지 않는다.
5. timeout, command error, partial parse 결과를 adapter boundary에서 표준 error/status로 매핑한다.
6. fake repository는 실제 Git command 대신 deterministic fixture로 workflow 테스트에 사용한다.

완료 기준: app workflow 테스트가 filesystem/Git subprocess 없이 실패·재시도·partial 상태를 검증할 수 있다. 실제 Git parser는 `internal/git` integration test로 별도 보호한다.

### Phase 4. rendering boundary와 layout contract

기존 UI 계획 `docs/20260731-0002-project-ui-readability-improvement-plan.md`를 이 구조개선의 후속 vertical slice로 연결한다.

작업:

- render input을 app runtime model에서 직접 읽는 대신 screen-specific projection으로 제한
- Graph header/row width budget 단일화
- Context/popup/frame spacing metrics 단일화
- ANSI semantic style과 visible marker를 renderer 계약에 포함
- render helper 테스트를 `model_test.go`에서 기능별 테스트 파일로 이동

완료 기준: renderer가 Git adapter나 mutation 함수를 참조하지 않고, 동일한 projection 입력은 동일한 visible output을 만든다.

### Phase 4 상세 구현 계획: T4 renderer projection

T4의 목적은 `model`을 작게 보이게 만드는 것이 아니라, 렌더러가 저장소 DTO와 실행 정책을 알지 못하도록 입력 경계를 고정하는 것이다. 기존 Graph-first industrial TUI의 출력과 단축키 의미는 유지한다.

#### T4.1 projection 타입과 소유 경계

새 파일은 실제 중복이 확인된 뒤 추가한다. 최초 구현에서는 `internal/app/view_projection.go`에 app 전용 projection 타입을 둔다. projection은 renderer가 필요한 값만 가진다.

```go
// view_projection.go
type ScreenProjection struct {
	Width       int
	Height      int
	Graph       GraphProjection
	Sections    SectionProjection
	Context     ContextProjection
	Overlay     OverlayProjection
	Global      GlobalProjection
}

type GraphProjection struct {
	Rows          []GraphRowProjection
	SelectedIndex int
	Scroll        int
	Search        SearchProjection
	Handshake     map[string]bool
}

type GraphRowProjection struct {
	Hash       string
	Decorations []string
	Graph      string
	Author     string
	Subject    string
	RelativeAge string
	StashCount int
	TagCount   int
	Virtual    bool
}

type SectionProjection struct {
	Current SectionListProjection
	Remote  SectionListProjection
	Tags    SectionListProjection
}

type SectionListProjection struct {
	Title       string
	Items       []SelectableItemProjection
	Cursor      int
	Overflow    int
	Active      bool
}

type SelectableItemProjection struct {
	Label       string
	Ref         string
	State       string
	Disabled    bool
	RecoveryHint string
}
```

`projectScreen(m model) ScreenProjection`만 `model`, `git.Status`, `state.Status`를 읽는다. `renderAppView`와 하위 `renderGraph*`, `renderSection*`, popup renderer는 projection만 인자로 받는다.

```go
func (m model) screenProjection() ScreenProjection {
	return ScreenProjection{
		Width:  m.width,
		Height: m.height,
		Graph: projectGraph(m.repoStatus, m.sectionCursor[sectionGraph], m.graphScroll, m.handshakeCommits, m.graphSearchQuery),
		Sections: projectSections(m.repoStatus, m.activeSection, m.sectionCursor),
		Context: projectContext(m.status, m.contextScroll),
		Overlay: projectOverlay(m),
		Global:  projectGlobal(m.status),
	}
}

func renderAppView(p ScreenProjection) string {
	// 이 함수와 하위 renderer는 git.Repo, git.Status, state.Action을 import하지 않는다.
	return renderShell(p)
}
```

초기 이행 중에는 `renderAppViewFromModel(m model)` compatibility shim을 둔다. shim은 `m.screenProjection()`을 한 번 호출한 뒤 새 renderer로 전달한다. 모든 호출자가 projection을 직접 사용하고 나면 shim 제거가 T4 완료 조건이다.

#### T4.2 상태와 오류 표현 계약

projection은 정상 상태만 표현하지 않는다. 다음 상태를 각각 명시적으로 표현한다.

| 상태 | projection 표현 | renderer 규칙 | 검증 |
|---|---|---|---|
| loading | `Loading=true`, 기존 snapshot 선택 | 현재 값과 loading 문구를 혼동하지 않음 | width/height별 출력 |
| empty | `Empty=true`, 빈 items | 빈 화면과 오류를 같은 문구로 표시하지 않음 | empty repo fixture |
| partial | `Degraded=true`, `RecoveryHint` | 유효한 필드는 유지하고 누락 필드를 `-` 또는 안내로 표시 | optional field 실패 |
| stale 폐기 | 화면 입력으로 전달하지 않음 | 이전 refresh 결과가 현재 projection을 변경하지 않음 | epoch 전환 테스트 |
| blocked/error | `ErrorMessage`, `RecoveryHint` | 문제·원인·복구 키를 함께 표시 | non-zero/timeout 테스트 |
| 좁은 터미널 | 폭 예산으로 축약 | hash, topology, 선택 상태, recovery hint 우선 보존 | 40/60/80열 |

`RecoveryHint`는 단순한 색상이나 아이콘으로 대체하지 않는다. 색상은 보조 수단이고 ANSI/NO_COLOR 환경에서도 문자열 의미가 남아야 한다.

#### T4.3 폭·색상·순수성 계약

```go
func renderProjection(p ScreenProjection) string
func graphRowFixedWidth(graphWidth int) int
func fitProjectionText(value string, width int) string
```

- `lipgloss.Width` 기준 visible width를 사용한다.
- 한 row의 폭 계산은 `graphRowFixedWidth`와 동일한 함수를 header/row/connector가 공유한다.
- 40열에서는 author를 생략할 수 있지만 hash, graph, 날짜 또는 상태, title의 최소 식별자는 보존한다.
- ANSI, ANSI256, TrueColor, `NO_COLOR`에서 semantic marker의 문자열과 visible width가 동일해야 한다.
- projection builder는 Git 명령, 파일 접근, telemetry side effect를 수행하지 않는다.
- 동일한 `ScreenProjection` 입력은 동일한 visible output을 반환해야 한다.

#### T4.4 T4 완료 게이트

1. `rg "git\\.Status|\\*git\\.Repo|repo\\.Run" internal/app/view_*.go internal/app/graph_render*.go` 결과가 0건이다.
2. renderer test가 `model`을 직접 생성하지 않고 projection fixture를 사용한다.
3. Graph/Context/Local/Remote/Tags/popup의 normal, empty, partial, error, narrow 출력이 고정된다.
4. 기존 `model_test.go`의 관련 테스트가 새 projection test로 이동한 뒤 compatibility shim을 제거한다.

### Phase 5 상세 구현 계획: T5 workflow adapter cleanup

T5는 T4 projection과 독립적인 Git 재작성 작업이 아니다. T2/T3에서 확정한 snapshot epoch·target validation 계약을 workflow 결과와 adapter 경계로 확장한다. `commands.go`를 먼저 파일로 쪼개지 않고, 한 workflow를 수직으로 이동한 뒤 반복을 제거한다.

#### T5.1 app 소유 최소 repository contract

`internal/app`의 workflow consumer가 필요한 최소 계약을 정의한다. 이 interface는 `git.Repo` 전체를 복제하지 않는다.

```go
// internal/app/repository_contract.go
type Repository interface {
	Snapshot(ctx context.Context, limit int) (RepositorySnapshot, error)
	ValidateTarget(ctx context.Context, target TargetRef) error
	Execute(ctx context.Context, operation Operation) (CommandResult, error)
	Reload(ctx context.Context, limit int) (RepositorySnapshot, error)
}

type TargetRef struct {
	Kind state.TargetKind
	Name string
	Hash string
}

type RepositorySnapshot struct {
	Epoch       uint64
	CapturedAt  time.Time
	Branch      string
	Head        string
	Graph       []GraphSnapshotRow
	Branches    []BranchSnapshot
	Tags        []TagSnapshot
	Worktree    WorktreeSnapshot
	Validity    SnapshotValidity
}

type SnapshotValidity struct {
	Graph    FieldValidity
	Branches FieldValidity
	Tags     FieldValidity
	Worktree FieldValidity
}

type FieldValidity struct {
	Available bool
	Degraded  bool
	Cause     string
}
```

`gitRepositoryAdapter`만 `*git.Repo`와 `git.Status`를 알고 app contract로 변환한다. `update_*`, renderer, state transition helper는 `RepositorySnapshot`과 `OperationResult`만 사용한다.

#### T5.2 workflow와 결과 계약

각 workflow는 아래 순서를 지키며, 단계 누락을 허용하지 않는다.

```text
intent
  → snapshot(epoch=current)
  → validate(target + current state)
  → execute(operation)
  → reload(snapshot)
  → OperationResult 생성
  → message 전달
  → state transition + projection
```

```go
type OperationResult struct {
	Operation    state.Action
	Target       TargetRef
	StartedEpoch uint64
	ResultEpoch  uint64
	Snapshot     RepositorySnapshot
	Command      CommandResult
	Phase        OperationPhase
	Err          error
	RecoveryHint string
}

type OperationPhase string

const (
	PhaseValidated OperationPhase = "validated"
	PhaseExecuted  OperationPhase = "executed"
	PhaseReloaded  OperationPhase = "reloaded"
	PhasePartial   OperationPhase = "partial"
)
```

- `StartedEpoch != ResultEpoch`이면 결과를 화면 상태에 적용하지 않고 최신 snapshot을 요청한다.
- Git command가 성공했지만 reload가 실패하면 `PhasePartial`을 유지하고 “작업은 실행됐지만 최신 상태를 읽지 못했다”를 표시한다.
- command non-zero/timeout은 `CommandResult`의 exit code와 stderr를 보존하되 사용자 문구에는 action, target, recovery hint를 포함한다.
- force-push, clean, hard reset, branch/tag/remote delete는 `ValidateTarget` 후 `Execute` 사이에 target을 다시 만들거나 바꿀 수 없다는 Git의 한계를 문서화한다. 확인 가능한 target ref/hash를 함께 전달하고, hash mismatch면 execute하지 않는다.

#### T5.3 수직 이행 순서

1. merge/rebase를 첫 workflow로 adapter에 연결한다.
2. reset/branch delete/tag delete/remote delete를 동일한 `TargetRef` 검증으로 이동한다.
3. stash/clean/pull/push/cherry-pick을 `OperationResult`에 연결한다.
4. tag snapshot attach, reload, error mapping 중복을 adapter/workflow helper로 이동한다.
5. 기존 `load*`, `execute*` 함수는 compatibility shim으로 유지하고 모든 호출자가 새 workflow를 사용하면 제거한다.

#### T5.4 fake repository 계약

```go
type FakeRepository struct {
	Snapshots  []RepositorySnapshot
	Validations []TargetRef
	Operations []Operation
	Results    []FakeResult
}

func (f *FakeRepository) Snapshot(context.Context, int) (RepositorySnapshot, error)
func (f *FakeRepository) ValidateTarget(context.Context, TargetRef) error
func (f *FakeRepository) Execute(context.Context, Operation) (CommandResult, error)
func (f *FakeRepository) Reload(context.Context, int) (RepositorySnapshot, error)
```

fixture는 호출 순서와 epoch을 검사한다. `Execute` 전에 `ValidateTarget`이 없으면 테스트가 실패해야 한다. filesystem/subprocess 없는 app workflow test와 실제 temporary Git repository를 사용하는 `internal/git` integration test를 분리한다.

#### T5.5 T5 완료 게이트

1. `internal/app` workflow test가 `exec.Command`, temporary Git repository, 실제 remote 없이 실행된다.
2. workflow별 happy/error/timeout/partial/stale target 결과가 `OperationResult`로 검증된다.
3. `commands.go`에는 새 workflow를 추가하지 않으며, 남은 함수는 compatibility shim 목록에 기록된다.
4. `internal/git` adapter test는 실제 Git command parsing과 timeout만 검증한다.
5. shim 제거 전 `rg`로 기존 함수 호출자가 0건인지 확인한다.

### Phase 6 상세 구현 계획: T6 테스트·문서 동기화

T6는 단순 파일 이동이 아니라 신규 기여자가 변경 위치와 검증 명령을 찾는 비용을 줄이는 마무리 slice다.

#### T6.1 테스트 분리 규칙

| 대상 | 새 파일 | fixture/검증 |
|---|---|---|
| runtime/epoch | `internal/app/runtime_state_test.go` | message transition, stale load/refresh |
| navigation/intent | `internal/app/navigation_test.go`, `intent_test.go` | pure model, no Git subprocess |
| projection/Graph | `internal/app/view_projection_test.go`, `graph_render_test.go` | projection fixture, visible width |
| sections/popup | `internal/app/section_render_test.go`, `popup_render_test.go` | normal/empty/error/overflow |
| workflow | `internal/app/workflow_test.go` | `FakeRepository`, operation order |
| Git adapter | `internal/git/adapter_test.go` 또는 기존 integration test | temporary Git fixture |

기존 대형 파일은 한 번에 삭제하지 않는다. 테스트를 새 파일로 복사하고 새 테스트를 먼저 통과시킨 뒤, 기존 중복 테스트를 단계적으로 제거한다. 테스트 이름은 `Test<Boundary>_<Scenario>` 형식으로 통일한다.

#### T6.2 contributor 문서

`docs/contributor-workflow.md`에 merge workflow 한 개를 다음 형식으로 기록한다.

```text
key: m
  → intent: ActionMerge(target)
  → workflow: Snapshot → ValidateTarget → Execute → Reload
  → result: OperationResult{Phase, Epoch, Err, RecoveryHint}
  → message: actionCompletedMsg
  → state: state.Status
  → projection: ScreenProjection
  → view: renderGraph/renderOverlay
  → tests: workflow_test + update_test + view_projection_test
```

문서에는 새 workflow를 추가할 때 수정할 파일, fake 사용법, focused test 명령, shim 제거 조건을 copy-paste 가능한 예제로 포함한다.

#### T6.3 문서 일관성 검사

- `docs/structure.md`: 실제 tree, 책임 map, import 방향을 갱신한다.
- migration ledger: current → target, owner, shim, removal gate, 상태를 갱신한다.
- `README.md`: 사용자 Quick Start를 오염시키지 않는 짧은 contributor 링크만 추가한다.
- plan 문서: 완료한 slice의 체크박스, commit, 검증 명령, 남은 risk를 갱신한다.

#### T6.4 T6 완료 게이트

1. 신규 기여자가 README → structure → contributor workflow 순서로 30분 안에 merge 흐름을 추적할 수 있다.
2. `go test ./internal/app -run 'Test(Workflow|Projection|Epoch)'`가 빠른 회귀 경로로 동작한다.
3. `go test ./internal/git`, `scripts/check`, `scripts/build /tmp/graphkeeper-architecture`가 통과한다.
4. migration ledger의 모든 `진행` 항목이 실제 파일·테스트·제거 조건과 연결된다.
5. 문서에 존재하지 않는 package/file을 목표 구조로 제시하지 않는다.

### T4~T6 의존성 및 롤백 순서

```text
T4 projection 계약
  ├─ T5 workflow result/adapter 계약
  └─ T6 projection/workflow fixture와 contributor trace
```

- T4 실패 시 renderer projection과 fixture만 revert한다. workflow adapter는 시작하지 않는다.
- T5 실패 시 adapter shim과 fake를 revert하고 T4 projection은 유지한다.
- T6 실패 시 테스트 파일 이동과 문서만 revert한다. production behavior는 변경하지 않는다.
- 각 단계는 독립 커밋하며, commit message에 `T4`, `T5`, `T6`와 검증 명령을 기록한다.

### Phase 6. 테스트와 문서 구조 정리

- `model_test.go`를 runtime, navigation, graph render, section render, workflow transition 테스트로 분리
- `key_handling_test.go`를 browse/global/popup/action intent 테스트로 분리
- architecture dependency check를 `scripts/check`에 추가할지 검토
- `docs/structure.md`, `docs/model-refactor-plan.md`, README contributor section 동기화
- 각 phase 종료 시 `git diff --check`, `go test ./...`, `scripts/check` 실행

완료 기준: 새 기능의 테스트 위치와 변경해야 할 경계가 문서와 실제 tree에서 일치한다.

## 7. 테스트 계약

```text
pure policy              → table-driven unit test
Git parser/runner        → internal/git integration test with fixture repo
workflow orchestration   → fake repository + message/result assertions
Bubble Tea update        → message-to-state transition test
renderer                 → projection-to-visible-output and width tests
CLI bootstrap            → cmd/graphkeeper tests
```

각 phase는 다음을 반드시 확인한다.

- 정상/빈 상태/오류/partial 상태
- detached HEAD, no upstream, no remote, empty repo
- merge/rebase/cherry-pick in progress
- command timeout와 Git non-zero exit
- ANSI/Ascii/NO_COLOR에서 visible label 보존
- 기존 key binding과 cursor/scroll 의미 보존

## 8. 명시적 비범위

- 전체 코드베이스 전면 재작성
- `internal/ui` 같은 빈 추상 package의 선제 도입
- Git library 교체
- Bubble Tea 교체
- 새로운 persistence/database/network service
- 사용자 theme 설정 기능
- 동작 변경을 구조개선으로 위장하는 action redesign
- 모든 popup의 색상 이전. 이는 Task 4.1의 후속 범위다.

## 9. 롤백과 안전장치

- phase별로 독립 커밋한다.
- 각 phase는 public behavior를 바꾸지 않는 것을 기본으로 한다.
- 계약 변경이 필요한 경우 compatibility adapter와 deprecation 기간을 둔다.
- baseline 테스트가 깨지면 다음 phase로 진행하지 않고 해당 phase를 revert한다.
- package 이동보다 순수 helper 추출을 먼저 수행해 diff를 작게 유지한다.

## 10. 완료 정의

1. 책임별 import 방향이 문서와 코드에서 일치한다.
2. `model`은 runtime container, `state.Status`는 user-visible state로 구분된다.
3. workflow orchestration과 Git adapter의 테스트 경계가 분리된다.
4. renderer는 projection 입력을 사용하고 Git mutation과 무관하다.
5. 기존 기능·키 바인딩·Git 상태 전이의 회귀가 없다.
6. `go test ./...`, `scripts/check`, `scripts/build`가 통과한다.
7. 새 기여자가 30분 안에 특정 workflow의 입력부터 출력까지 추적할 수 있다.

## 11. 구현 작업

- [x] T1. baseline 및 migration ledger 고정
- [x] T2. snapshot/operation epoch과 stale 결과 폐기
- [x] T3. destructive action 실행 직전 target 재검증
- [ ] T4. rendering projection 및 layout 계약 연결
- [ ] T5. workflow adapter와 deterministic fake 경계 도입
- [ ] T6. 테스트 파일과 구조 문서 동기화

## 12. 리뷰 기록

이 문서는 `$autoplan`의 CEO → Design → Eng → DX 리뷰 결과를 아래에 누적한다.

### 12.1 CEO 리뷰 반영

검토 근거: `internal/app/update_lifecycle.go`, `internal/app/commands.go`, `internal/app/model.go`, `internal/app/messages.go`, `internal/git/repo.go`와 실제 테스트 규모를 확인했다.

1. **높음: 비동기 결과 순서 역전**
   - 매초 refresh가 실행되고 mutation 결과에는 sequence/version 계약이 없다.
   - Phase 1에 `operationID` 또는 repository revision epoch를 추가하고, stale message 폐기 규칙과 out-of-order 테스트를 먼저 정의한다.
2. **높음: repository snapshot 유효성 계약 부족**
   - `git.Status`는 여러 명령 실패를 부분적으로 무시한다.
   - adapter 도입 전 required/optional field, snapshot timestamp/revision, partial error, displayable 여부를 표로 고정하고 silent degradation을 관측 가능하게 한다.
3. **높음: 사용자 가치 없는 장기 내부 리팩터링 위험**
   - Phase 1–5가 약 18,500줄 규모의 코드에 걸치며 완료 기준이 구조 중심이다.
   - T2–T5를 refresh 신뢰성, destructive action 안전성, rendering projection 같은 3개의 vertical slice로 재편하고 각 slice마다 사용자-visible 개선과 구조개선을 함께 ship한다.
4. **중간: destructive action 안전 계약 부족**
   - `executeCleanWorkingTree`, force-push, branch/tag 삭제는 실제 Git history 또는 작업 트리를 파괴할 수 있다.
   - Phase 2 이전에 confirmation, 실행 직전 target 재검증, stale target 방지, force-push/cleanup/branch delete/remote delete 회귀 테스트를 추가한다.
5. **중간: interface만 추가하면 Git DTO 결합이 남음**
   - app message가 raw `git.Status`를 운반한다.
   - Phase 1에서 application-facing read model과 mutation result를 정의하고, message가 parser-specific `git.Status` 대신 해당 계약을 운반하게 한다.

### 12.2 디자인 리뷰

Design scope: 있음. Graph, Context, section, popup, terminal width와 accessibility가 직접 영향을 받으므로 UI 계획으로 평가했다.

| 차원 | 점수 | 확인한 내용 | 결정 |
|---|---:|---|---|
| 정보 계층 | 8/10 | Graph-first, Context 보조, popup action이라는 기존 계층이 명확하다. | renderer projection에 primary identity, state label, recovery hint 우선순위를 명시한다. |
| 상태 범위 | 6/10 | loading/empty/error/partial은 목록에 있으나 stale snapshot과 mutation 중 refresh가 충분히 정의되지 않았다. | 상태 매트릭스에 stale/refreshing/conflict를 추가한다. |
| 사용자 여정 | 8/10 | maintainer가 Graph를 보고 action을 고르는 흐름은 보존된다. | 각 vertical slice가 scan → decide → recover 흐름을 유지하는지 검증한다. |
| 터미널 반응성 | 7/10 | ANSI width helper가 있으나 projection별 최소 폭 계약이 부족하다. | 40/60/80열, low-height, Ascii/NO_COLOR 기준을 renderer 계약으로 고정한다. |
| 접근성 | 7/10 | 키보드 중심이며 visible label을 유지한다. | 색상만으로 구분하지 않고 marker/label/focus attribute를 보존한다. |
| 디자인 시스템 | 8/10 | industrial, compact, graph-first 방향과 일치한다. | 새 card/legend/package abstraction을 추가하지 않는다. |
| 미해결 | 6/10 | screen projection의 정확한 필드와 stale 상태 표현이 남아 있다. | Phase 1의 계약 표에서 결정하고 구현자가 추측하지 않게 한다. |

Design 결정:

1. 구조개선은 visual redesign가 아니다. 기존 shell overlay precedence, title strip, section highlight, graph topology를 보존한다.
2. renderer는 `screen projection → visible output`의 순수 경계를 목표로 하되, projection 타입은 실제 중복이 확인된 뒤 최소 단위로 만든다.
3. loading, empty, error, success, partial, refreshing, stale를 surface별로 명시한다.

### 12.3 엔지니어링 리뷰

#### 목표 의존성 그래프

```text
tea.KeyMsg / tea.WindowSizeMsg
              │
              ▼
      key intent + runtime state
              │
              ▼
     workflow command boundary ──────┐
              │                       │
              ▼                       │
   repository read/mutation contract  │
              │                       │
              ▼                       │
   git.Repo adapter + parser tests    │
              │                       │
              ▼                       │
     application read model/result ◀─┘
              │
              ▼
       screen projection
              │
              ▼
        renderer / TUI output
```

핵심 engineering 결정:

- `git.Status`를 app message의 장기 계약으로 유지하지 않는다. `RepositorySnapshot`, `OperationResult`, `PartialError` 같은 application-facing 계약을 먼저 정의한다.
- 매초 refresh와 mutation 결과 사이에 `operationID` 또는 repository epoch를 넣고, 오래된 message는 상태를 덮어쓰지 못하게 한다.
- snapshot field를 required/optional/stale 허용 여부로 분류한다. optional field 실패는 visible degraded state와 local diagnostic event를 남긴다.
- force-push, hard reset, clean, branch/tag/remote delete는 실행 직전 target과 repository state를 재검증한다.
- interface는 큰 `Repo` 전체를 복제하지 않고 workflow consumer가 실제로 사용하는 최소 계약으로 만든다.
- 새 package는 import cycle이나 실제 경계가 확인된 경우에만 만든다. 파일 이동 자체를 완료 기준으로 삼지 않는다.

#### 테스트 흐름 지도

```text
NEW UX FLOWS
  refresh 중 mutation 완료       → stale result guard test + update transition test
  destructive action confirmation → target revalidation + cancel/confirm test
  partial repository snapshot     → degraded label + recovery hint test
  projection-based rendering     → visible text/width/profile test

NEW DATA FLOWS
  git commands → parser → snapshot validity → application read model → screen projection
  key intent → workflow validation → mutation → result/reload → state transition

NEW CODEPATHS
  operation epoch match/mismatch
  required/optional snapshot field failure
  fake repository success/error/partial/timeout
  renderer projection for normal/empty/error/stale states

NO NEW BACKGROUND JOBS
  existing periodic refresh remains; only ordering/cancellation semantics change

NO NEW EXTERNAL INTEGRATIONS
  Git subprocess remains the only external system
```

각 항목의 테스트는 pure policy unit, fake repository workflow test, real fixture Git integration test, Bubble Tea message transition test, projection renderer test의 피라미드로 배치한다. `model_test.go` 전체를 한 번에 옮기는 작업은 구현 가치가 없으므로 vertical slice에서 필요한 영역만 먼저 이동하고, 마지막에 남은 테스트를 정리한다.

#### Engineering failure modes

| 경로 | 실패 | 복구 | 테스트 | 사용자 결과 |
|---|---|---|---|---|
| refresh/mutation | 오래된 refresh가 최신 mutation을 덮음 | epoch mismatch 폐기 및 재조회 | out-of-order transition | 최신 상태 유지 |
| snapshot | required Git field 실패 | 작업 중단, error state | fake + fixture error | 원인과 재시도 방법 표시 |
| snapshot | optional field 실패 | stale/degraded flag 유지 | partial snapshot test | 나머지 정보는 보이고 누락 label 표시 |
| destructive action | target이 stale | 실행 직전 재검증 후 차단 | target changed test | 실행하지 않고 refresh 안내 |
| Git mutation | timeout/non-zero exit | error mapping과 상태 재조회 시도 | timeout/error test | 실패 원인과 복구 action 표시 |
| renderer | 좁은 terminal | required label 우선 truncation | width/profile matrix | topology와 action hint 보존 |

### 12.4 개발자 경험(DX) 리뷰

| 항목 | 현재 | 목표 | 계획 변경 |
|---|---|---|---|
| 30분 trace | 목표만 있고 예제 없음 | merge workflow 한 개를 입력부터 renderer까지 추적 | `docs/contributor-workflow.md`를 Phase 0 산출물로 추가 |
| 구조 문서 | 현재 tree만 기술 | phase별 current → target → owner → shim 제거 조건 | migration ledger를 Phase 0부터 갱신 |
| 테스트 실행 | 전체 suite 중심 | 빠른 package/filter 실행 가능 | fixture 위치와 `go test ./internal/app -run ...` 명령 문서화 |
| 오류 진단 | TUI 문구와 local JSONL이 분리 | operation/target/cause/recovery/diagnostic event 일치 | error contract와 troubleshooting 문서 추가 |
| 업그레이드 | compatibility shim 언급만 있음 | Go version, shim lifetime, removal gate 명시 | migration section과 README contributor link 추가 |

DX 결정:

1. Phase 0에서 contributor trace와 migration ledger를 작성한다. Phase 5까지 미루지 않는다.
2. `scripts/test`를 무조건 확장하지 않고, 먼저 직접 실행 가능한 빠른 Go 명령을 문서화한다. 반복되는 요구가 확인될 때만 filter passthrough를 추가한다.
3. README에는 구조개선의 목표와 contributor 문서 링크만 추가하고, 사용자 Quick Start를 구조 설명으로 오염시키지 않는다.

### 12.5 리뷰 후 확정 구현 순서

기존 Phase 1–5를 장기간 내부 이동으로 실행하지 않고 아래 vertical slice로 재배치한다.

1. **Slice A: refresh 신뢰성**
   - application-facing snapshot 계약, required/optional field, operation epoch를 정의한다.
   - refresh와 mutation의 out-of-order 결과를 차단한다.
   - 사용자-visible 개선: 오래된 상태가 최신 작업 결과를 덮어쓰지 않는다.
2. **Slice B: destructive action 안전성**
   - clean, hard reset, force-push, branch/tag/remote delete의 target 재검증과 confirmation 계약을 통일한다.
   - 사용자-visible 개선: stale target이 실행되지 않고 실패 이유와 복구 경로가 일관된다.
3. **Slice C: renderer projection과 contributor trace**
   - screen projection을 도입하고 Graph/Context/popup의 visible-width/state 계약을 고정한다.
   - `docs/contributor-workflow.md`와 migration ledger를 갱신한다.
   - 사용자-visible 개선: 기존 layout과 state label이 좁은 terminal에서도 안정적으로 유지된다.
4. **Slice D: workflow/adapter 정리**
   - Slice A–C에서 검증된 계약을 기준으로 commands/update/Git adapter 내부 중복을 제거한다.
   - fake repository와 fixture integration test를 완성한다.
5. **Slice E: 테스트와 문서 정리**
   - 남은 대형 test file을 기능별로 분리하고, 실제 current tree와 문서를 동기화한다.

각 slice는 독립 커밋, 독립 검증, 독립 rollback을 가진다. 사용자-visible 결과가 없는 파일 이동은 해당 slice의 필수 변경으로 인정하지 않는다.

## 13. CEO·디자인·엔지니어링·DX 종합 점수

| 리뷰 | 점수 | 판단 |
|---|---:|---|
| CEO / 전략 | 8/10 | 방향은 맞지만 비동기 순서, snapshot 계약, 파괴적 작업 안전성을 먼저 고정해야 한다. |
| 디자인 | 7/10 | 기존 Graph-first 방향과 일치하나 stale/partial/좁은 terminal 계약이 더 구체적이어야 한다. |
| 엔지니어링 | 7/10 | 단계적 경계는 타당하지만 raw `git.Status`와 큰 message payload를 먼저 분리해야 한다. |
| 개발자 경험(DX) | 6/10 | 현재 문서는 구현 방향은 설명하지만 신규 기여자의 30분 추적과 migration 경로가 부족했다. |

종합 판단: **조건부 승인 가능**. Slice A의 epoch/snapshot 계약과 Slice B의 destructive safety가 구현 전에 계획에 반영되었고, UI 구조는 기존 시각 언어를 보존하는 조건이다.

## 14. 범위에서 제외

- Bubble Tea 또는 Lip Gloss 교체: 현재 문제를 해결하지 않으며 migration risk가 크다.
- Git library 교체: parser/runner 경계를 먼저 고정한 뒤 별도 검토한다.
- 전면 package rewrite: 현재 방향 A와 충돌하고 rollback 비용이 크다.
- 새로운 network/database/persistence service: 구조개선의 사용자 가치와 무관하다.
- 영구 legend, onboarding mode, theme editor: 기존 industrial TUI의 밀도를 해친다.
- 전체 test file 일괄 분리: vertical slice 이후 남은 테스트에만 적용한다.
- 모든 popup 색상 migration: Task 4.1의 deferred scope로 유지한다.

## 15. 결정 감사 기록

| # | 단계 | 결정 | 분류 | 원칙 | 근거 | 제외한 선택 |
|---:|---|---|---|---|---|---|
| 1 | CEO | 점진적 경계 개선 A 채택 | 사용자 확인 | 완전성 + 실용성 | 사용자가 A를 선택했고 동작 보존 migration을 원한다. | 전면 rewrite, UI-only |
| 2 | CEO | 내부 리팩터링을 vertical slice로 재편 | 자동 결정 | 실행 편향 | 사용자 가치 없는 장기 이동을 줄이고 각 단계의 결과를 검증한다. | 5단계 일괄 내부 이동 |
| 3 | 엔지니어링 | operation epoch/stale result guard 선행 | 자동 결정 | 명시성 | 매초 refresh와 mutation 결과의 순서 역전 위험이 실제 코드에 있다. | 무조건 순서가 맞는다고 가정 |
| 4 | 엔지니어링 | raw `git.Status`를 장기 app 계약에서 제거 | 자동 결정 | 중복 제거 + 실용성 | interface만 추가해도 parser DTO 결합이 남기 때문이다. | 큰 Repo interface 복제 |
| 5 | 엔지니어링 | destructive action safety contract 선행 | 자동 결정 | 완전성 | clean/reset/force-push/delete는 실패 시 사용자 비용이 크다. | 정상 경로만 테스트 |
| 6 | 디자인 | 기존 Graph-first industrial UI 보존 | 자동 결정 | 명시성 | 구조개선이 visual redesign로 변질되지 않도록 한다. | 새 legend/card UI |
| 7 | DX | contributor trace와 migration ledger를 Phase 0로 이동 | 자동 결정 | 완전성 | 문서화를 마지막에 미루면 migration 중 구조가 다시 drift한다. | Phase 5에서 일괄 문서화 |

## 16. 구현 작업

- [ ] **T1 (P1, human: ~3h / CC: ~25min)** — Baseline and migration ledger — 현재 책임, import, message 흐름과 기존 테스트 baseline 기록
  - Surfaced by: CEO/DX — 구조 이동 전 회귀 원인과 current→target 경로가 문서화되지 않음
  - Files: `docs/structure.md`, `docs/model-refactor-plan.md`, `docs/contributor-workflow.md`, `internal/app/*_test.go`
  - Verify: `go test ./...`, `scripts/check`, 대표 merge workflow trace 확인
- [ ] **T2 (P1, human: ~1d / CC: ~45min)** — Snapshot and epoch contract — application read model, partial validity, operation epoch, stale message guard 도입
  - Surfaced by: CEO/Eng — refresh가 mutation 결과를 덮어쓸 수 있고 raw `git.Status`가 message를 관통함
  - Files: `internal/app/model.go`, `internal/app/messages.go`, `internal/app/update_lifecycle.go`, `internal/git/repo.go`, 관련 tests
  - Verify: out-of-order refresh/action, required/optional field, partial snapshot, timeout tests
- [ ] **T3 (P1, human: ~1d / CC: ~45min)** — Destructive action safety — 실행 직전 target/state 재검증과 공통 confirmation/error contract
  - Surfaced by: CEO/Eng — clean, hard reset, force-push, branch/tag/remote delete의 stale target 위험
  - Files: `internal/app/commands.go`, `internal/app/update_execute.go`, `internal/app/actions.go`, 관련 tests
  - Verify: target changed, cancel, confirm, non-zero exit, partial reload, mutation safety tests
- [ ] **T4 (P1, human: ~1d / CC: ~45min)** — Renderer projection — Graph/Context/popup이 screen projection만 소비하도록 전환
  - Surfaced by: Design/Eng — renderer가 runtime model과 Git-shaped DTO에 직접 결합됨
  - Files: `internal/app/view_*.go`, `internal/app/graph_render*.go`, `internal/state/*`, 관련 tests
  - Verify: 40/60/80열, low-height, Ascii/ANSI/ANSI256/TrueColor/NO_COLOR visible output
- [ ] **T5 (P2, human: ~1d / CC: ~30min)** — Workflow adapter cleanup — validated contract 기준 commands/update 중복 제거와 fake repository 완성
  - Surfaced by: Eng — commands.go의 reload/tag/error mapping 중복과 큰 concrete Repo 의존
  - Files: `internal/app/commands.go`, `internal/app/update_*.go`, `internal/git/*`, fixture tests
  - Verify: app workflow tests without subprocess, git fixture integration tests, `go test ./...`
- [ ] **T6 (P2, human: ~4h / CC: ~20min)** — Test and documentation migration — 기능별 테스트 파일, contributor guide, README/structure sync
  - Surfaced by: DX — 30분 trace, focused command, shim removal criteria가 부족함
  - Files: `docs/contributor-workflow.md`, `docs/structure.md`, `README.md`, `internal/app/*_test.go`
  - Verify: `go test ./internal/app -run TestMerge`, `go test ./internal/git`, `scripts/check`, `scripts/build /tmp/graphkeeper-architecture`

## 17. 오류 및 복구 레지스트리

| 경로 | 실패 | 복구 | 사용자 영향 | 진단 |
|---|---|---|---|---|
| refresh snapshot | required field/command failure | error state, retry/refresh hint | stale data is not presented as current | structured app event |
| refresh snapshot | optional field failure | degraded/stale flag, keep valid fields | partial state remains visible with label | field + error event |
| mutation result | stale epoch | discard and request latest snapshot | old result cannot overwrite current state | stale-message counter/log |
| destructive action | target changed | block before mutation, refresh target list | no unintended history/worktree mutation | target mismatch event |
| Git mutation | timeout/non-zero exit | map operation error, reload if safe | actionable failure message | action/target/error event |
| renderer | width/height too small | priority truncation | identity/topology/recovery hint remains | no external side effect |

## 18. 개발자 여정 지도

| 단계 | 기여자 행동 | 현재 마찰 | 계획 결과 |
|---|---|---|---|
| Discover | README와 structure 문서 읽기 | current tree와 target tree가 다름 | migration ledger와 contributor guide |
| Build | bootstrap/test 실행 | 전체 suite만 눈에 띔 | focused Go command와 scripts 기준 명시 |
| Trace | `m` merge 흐름 추적 | key/update/commands/Git가 분산 | 단일 end-to-end trace 문서 |
| Change | policy 또는 renderer 수정 | model_test 대형 파일 탐색 필요 | contract별 test 위치와 projection |
| Break | timeout/partial Git 결과 재현 | fake 경계 부족 | deterministic fake + fixture repo |
| Diagnose | TUI 오류와 local log 대조 | taxonomy가 암묵적 | error contract와 troubleshooting |
| Review | phase diff 검토 | 구조 이동만으로 가치 판단 어려움 | vertical slice와 user-visible acceptance |
| Upgrade | 다음 phase 이어가기 | shim 제거 기준 불명확 | migration ledger와 removal gate |
| Contribute | 새 기능 추가 | import 방향 추측 필요 | 의존성 규칙과 30분 trace |

## 19. 12개월 목표와 현재 계획의 차이

```text
현재: Bubble Tea model이 repo snapshot, runtime UI, popup, search를 보유
  ↓ Slice A/B: 최신 snapshot과 mutation 안전성을 계약으로 고정
  ↓ Slice C/D: projection, workflow, adapter의 실제 경계를 검증
  ↓ Slice E: 테스트/문서가 코드 경계를 반영
목표: 새로운 Git workflow를 추가할 때
      intent → contract → adapter → result → projection → renderer/test를
      30분 안에 추적하고, 기존 UI/상태 전이를 깨뜨리지 않음
```

## 20. 완료 요약

| 항목 | 결과 |
|---|---|
| 모드 | 선택적 확장, 사용자 선택 A 기반 |
| 시스템 감사 | `internal/app/model.go`, `commands.go`, update handler, `git.Status`, 대형 테스트 파일 확인 |
| CEO | 5개 발견사항 반영: epoch, snapshot 유효성, vertical slice, 파괴적 작업 안전성, application read model |
| 디자인 | 7/10. Graph-first industrial TUI, terminal width, visible label, stale/partial 상태 명시 |
| 엔지니어링 | 7/10. fake repository, adapter 계약, 오류/롤백/테스트 매트릭스 명시 |
| DX | 6/10. contributor trace, migration ledger, 빠른 테스트 명령, troubleshooting 경로 추가 |
| 새 연동 | 없음 |
| Critical gap | 없음. 구현 전 Slice A/B 계약이 선행되어야 함 |
| 사용자-visible 완료 기준 | refresh 순서, 파괴적 작업 안전성, projection width/state 보존 |
| 롤백 | slice별 독립 커밋과 단계별 revert |

## 21. 최종 제안

이 계획은 승인 가능하다. 단, 구현은 T1 baseline과 T2 epoch/snapshot 계약에서 시작해야 하며, T2가 완료되기 전에 package 이동이나 대규모 test 파일 분리를 시작하지 않는다. 새 package는 실제 import 경계가 드러난 뒤에만 만든다.

## GSTACK 리뷰 보고서

- `/autoplan` 파이프라인: CEO → 디자인 → 엔지니어링 → DX
- 전제/방향 gate: 사용자 선택 A로 통과
- 외부 검토: Codex CEO, 디자인, 엔지니어링, DX 리뷰 실행. 현재 Codex 환경에서 Claude subagent를 사용할 수 없어 Codex-only로 기록한다.
- 리뷰 결과: `DONE_WITH_CONCERNS`
- 우려사항: async epoch와 snapshot validity를 먼저 구현해야 하며, destructive action safety 계약 없이 후속 구조 이동을 진행하지 않는다.
- T1/T2 계약과 테스트가 추가된 후 구현-ready로 간주한다.

## 22. 사용자 승인

- 승인: A. 계획 승인
- 승인일: 2026-07-31
- 승인 범위: T1 baseline부터 T6 테스트·문서 정리까지의 점진적 경계 개선
- 구현 시작 조건: T1 baseline 기록과 T2 snapshot/operation epoch 계약을 먼저 완료한다.

## 23. 2026-08-01 최종 구현 검수

판정: **T1은 바로 구현 가능하며, 전체 T1~T6 일괄 구현은 아직 시작하지 않는다.**

검수 결과:

- 계획 문서가 실제 현재 코드의 핵심 결합 지점(`model.go`, `commands.go`, update handler, `git.Status`, 대형 테스트 파일)을 참조한다.
- T1 baseline과 migration ledger의 대상 파일·검증 명령·완료 기준이 정의되어 있다.
- T2는 operation epoch, snapshot validity, stale message 처리 규칙이 명시되어 있어 T1 이후 구현 가능하다.
- T3 이후 단계는 T2 계약과 테스트 결과에 의존하므로, T1/T2 검증 전 package 이동이나 대규모 테스트 분리를 시작하지 않는다.
- 구현 전 추가로 필요한 사용자 결정은 없다. 기존 승인 범위와 현재 Taskmaster 상태가 일치한다.

T1~T3 범위에서 NO UNRESOLVED DECISIONS

## 24. 2026-08-01 T4~T6 autoplan 상세 리뷰

### 24.1 리뷰 범위와 결론

이번 autoplan은 이미 구현된 T1~T3을 다시 구현하지 않고, 후속 T4~T6의 구현 가능성을 검토했다.

| 항목 | 결과 |
|---|---|
| UI scope | 있음. Graph, Context, section, popup renderer의 projection·width·상태 계약을 검토함 |
| DX scope | 있음. Graphkeeper는 기여자가 구조와 workflow를 추적해야 하는 개발자 도구임 |
| 기존 문서 상태 | 방향·완료 기준은 있음. 구체적인 소유 타입, adapter 결과, fake 호출 계약이 부족함 |
| 핵심 수정 | T4/T5 명칭과 순서를 정렬하고 T4 projection, T5 adapter/workflow, T6 fixture/document gate를 상세화함 |
| 구현 상태 | T4~T6 미구현. 이번 변경은 계획 문서만 갱신함 |

결론은 **상세화 필요**다. 기존 문서만으로도 방향은 이해할 수 있지만, 구현자는 projection 타입을 어디에 둘지, `git.Status`를 어떤 경계에서 변환할지, partial reload를 어떤 결과로 표현할지 임의로 결정해야 했다. 이 결정들을 본 문서의 T4~T6 상세 계약으로 고정했다.

### 24.2 CEO 리뷰

검토한 문제는 “구조를 나누는 것”이 사용자에게 실제로 어떤 안전성과 속도 개선을 주는지, T4~T6이 장기 rewrite로 변질되지 않는지였다.

- 이미 T1~T3에서 refresh stale overwrite와 destructive target 위험을 줄였으므로, T4~T6은 그 계약을 사용자가 체감하는 출력·기여자 경로·테스트 속도로 연결해야 한다.
- T4를 단순 파일 이동으로 진행하면 사용자 가치가 없다. projection은 좁은 terminal, partial/error 상태, ANSI/NO_COLOR에서 identity와 recovery hint를 보존해야 한다.
- T5를 `Repo` 전체 interface 복제로 시작하면 결합이 이름만 바뀐다. workflow consumer가 실제 사용하는 최소 snapshot/validate/execute/reload 계약만 둔다.
- T6에서 전체 대형 테스트 파일을 한 번에 분리하면 회귀 추적이 어려워진다. 각 slice의 fixture와 문서를 먼저 만들고, 중복 테스트 제거는 shim 제거 게이트 뒤로 미룬다.

선택: 기존 점진적 전략을 유지하고 T4→T5→T6 순서를 명시한다. 전면 package rewrite와 테스트 일괄 이동은 범위 밖으로 유지한다.

### 24.3 디자인 리뷰

| 차원 | 점수 | 판단 |
|---|---:|---|
| 정보 계층 | 8/10 | Graph-first hierarchy를 유지하고 projection에 identity/state/recovery 우선순위를 명시함 |
| 상태 완전성 | 8/10 | loading, empty, partial, stale, blocked/error를 별도 계약으로 추가함 |
| 좁은 terminal | 8/10 | 40/60/80열과 visible width 규칙, 최소 식별자 보존을 고정함 |
| 색상·접근성 | 8/10 | 색상을 보조 수단으로 제한하고 ANSI/NO_COLOR 문자열 의미를 유지함 |
| 일관성 | 7/10 | projection 이후 section/popup까지 동일 계약을 적용해야 함 |
| 구현 명확성 | 8/10 | 타입과 함수 예시를 추가했으나 실제 field mapping은 구현 slice에서 검증 필요 |
| 회귀 위험 | 7/10 | compatibility shim과 projection fixture로 낮추되 기존 golden 범위를 확인해야 함 |

디자인 결정은 새 UI를 만드는 것이 아니라 현재 industrial TUI의 의미를 보존하는 것이다. `S`, `T`, `S·T`, 선택 상태, 오류 복구 문구는 색상 없이도 읽혀야 하며, projection은 이 규칙을 테스트 가능한 입력으로 만든다.

### 24.4 엔지니어링 리뷰

목표 의존성은 다음과 같다.

```text
model + git.Status + state.Status
              │
              ▼
      projectScreen(model)
              │  ScreenProjection
              ▼
      renderGraph / renderSection / renderPopup

intent ─▶ workflow ─▶ Repository adapter ─▶ git.Repo
  │          │              │
  │          └─ OperationResult ──▶ message/update
  └─ TargetRef + epoch validation
```

핵심 엔지니어링 결정:

1. T4 projection은 `internal/app`이 소유하되 renderer가 `git.Status`를 import하지 않도록 한다.
2. T5 adapter는 app consumer가 필요한 최소 계약만 정의하고 `gitRepositoryAdapter`에서 `git.Status`를 변환한다.
3. operation 결과는 실행 전 epoch, 실행 후 epoch, phase, partial snapshot, recovery hint를 보유한다.
4. fake repository는 호출 순서와 target validation 선행을 검사한다.
5. T4 projection이 먼저 고정되어야 T5 workflow 결과를 화면에 연결할 수 있고, T6는 두 계약의 fixture를 기준으로 진행한다.

#### 테스트 흐름 지도

| 코드 경로 | 정상 | 오류/partial | stale/경계 | 테스트 위치 |
|---|---|---|---|---|
| model → projection | normal browse | loading/empty/error | stale snapshot 미적용 | `view_projection_test.go` |
| projection → Graph | graph row/header | narrow width | ANSI/NO_COLOR | `graph_render_test.go` |
| intent → workflow | merge/reset/delete | non-zero/timeout/reload failure | target mismatch/epoch mismatch | `workflow_test.go` |
| adapter → snapshot | complete fields | optional field degraded | required field invalid | `internal/git` integration |
| message → state | action complete | partial recovery | stale result discard | `update_*_test.go` |
| docs → contributor | merge trace | troubleshooting | shim removal | `docs/contributor-workflow.md` + focused commands |

### 24.5 DX 리뷰

현재 신규 기여자는 README와 structure를 읽은 뒤 `key_handling_* → update_* → commands.go → git.Repo → view_*`를 수동으로 따라가야 한다. T6의 contributor trace는 이 경로를 `intent → workflow → adapter → result → projection → renderer → tests` 한 줄로 제공한다.

| DX 항목 | 현재 | T4~T6 목표 |
|---|---:|---:|
| 특정 workflow 추적 시간 | 약 30분 이상 | 30분 이내 |
| focused test 발견 | 불명확 | `Test(Workflow|Projection|Epoch)` 명령 제공 |
| 실패 원인 파악 | TUI 문구와 Git stderr 분리 | operation/target/cause/recovery 계약 |
| 새 workflow 추가 위치 | commands.go 중심 추측 | contract/adapter/result/projection 목록 |
| shim 제거 판단 | 암묵적 | ledger의 호출자 0건 + 테스트 게이트 |

### 24.6 자동 결정과 남은 승인

| # | 영역 | 결정 | 분류 | 적용 원칙 |
|---:|---|---|---|---|
| 8 | T4 | projection은 app 소유, renderer는 projection만 소비 | 자동 결정 | 명시성 |
| 9 | T4 | loading/empty/partial/error/stale를 별도 상태로 테스트 | 자동 결정 | 완전성 |
| 10 | T5 | 전체 Repo interface 복제 대신 최소 snapshot/operation contract | 자동 결정 | 실용성 + DRY |
| 11 | T5 | merge/rebase부터 수직 이행 후 나머지 workflow 확장 | 자동 결정 | 실행 편향 |
| 12 | T6 | 대형 테스트 파일 일괄 이동 금지, fixture 우선 분리 | 자동 결정 | 회귀 최소화 |
| 13 | 범위 | T4~T6 구현은 별도 독립 커밋으로 진행 | 자동 결정 | rollback 가능성 |

남은 사람의 승인 지점은 하나다. 위의 projection/app contract/fake repository 방향을 T4~T6의 구현 기준으로 채택할지 확인해야 한다. 이 승인이 끝나면 T4 구현을 시작할 수 있고, 승인 전에는 production code를 변경하지 않는다.

## 25. T4~T6 상세화 최종 상태

- T4는 projection 타입, 상태 표현, 폭·색상·순수성, 완료 게이트를 정의했다.
- T5는 최소 repository contract, `OperationResult`, target/epoch 규칙, fake repository, 수직 이행 순서를 정의했다.
- T6는 테스트 파일 분리 규칙, contributor workflow 문서 형식, 문서 동기화, focused verification, shim 제거 게이트를 정의했다.
- T4~T6의 1차 구현이 반영되었다. Graph/Section/Context projection, 최소 repository adapter/fake 계약, projection 테스트와 contributor 문서가 추가되었다.
- 전체 popup projection 전환, 모든 workflow의 `OperationResult` 통합, 대형 테스트 파일의 기능별 이동은 후속 호환 shim 제거 slice로 남아 있다.

상태: **사용자 승인 완료, T4~T6 1차 구현 완료, 최종 리뷰 대기**

## 26. T4~T6 사용자 승인

- 승인: A. 상세 계획 승인
- 승인일: 2026-08-01
- 승인 범위: T4 renderer projection, T5 workflow/adapter contract, T6 테스트·문서 동기화
- 구현 순서: T4 → T5 → T6
- 구현 전제: production code는 승인된 contract와 완료 게이트를 기준으로 변경하며, 각 단계는 독립 커밋·검증·rollback을 갖는다.
