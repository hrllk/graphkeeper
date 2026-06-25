# Reset / Stash Feature Plan

## 목적

이 문서는 `reset` 과 `stash` 를 Graph-first 워크플로우에 맞게 추가하기 위한 기능 계획서다.

이번 계획의 핵심은 다음이다.

1. Graph 섹션에서 특정 포인터를 기준으로 `reset` 을 시작한다.
2. `reset` 은 `soft / mixed / hard` 선택자를 제공한다.
3. `dirty / clean` 은 UI flow 가 아니라 작업 트리 상태로 별도 필드에 둔다.
4. `dirty` 상태에서만 `working tree clean` 과 `stash` 관련 행위를 허용한다.
5. `stash` 는 Graph 포인터 hover 시 해당 시점의 존재 여부를 보여주고, 포커스 시 상세 목록을 제공한다.

## 참조 문서

이 계획은 다음 아카이브 문서를 참고한다.

- `docs/archive/20260625-0015-feature-pull-reset-ux-implementation-plan.md`
- `docs/archive/202606-0004-refactor-view-graph-structure-plan.md`
- `docs/archive/20260625-0012-refactor-navigation-boundary-plan.md`
- `docs/archive/20260625-0013-refactor-update-dispatch-plan.md`
- `docs/archive/architecture.md`
- `docs/roadmap.md`

## 현재 관찰된 구현 상태

현재 코드에서는 다음 흐름이 이미 존재한다.

- `s` 는 Graph 섹션에서 `reset` 진입점으로 사용된다.
- `reset` 은 현재 `hard reset` 만 실행한다.
- `dirty worktree` 는 이미 공통 차단 사유로 사용된다.
- branch 생성은 dirty worktree 에서 차단된다.
- merge / rebase / pull / push 는 confirm 및 preview 흐름을 이미 갖고 있다.

즉, 이번 작업은 완전히 새로운 패턴을 만드는 것이 아니라, 기존 action / confirm / status 흐름에 `reset mode` 와 `working tree state` 를 추가하는 작업이다.

## 핵심 결정

### 1. reset 은 현재 브랜치 작업을 방해할 수 있는가를 기준으로 상태를 노출한다

`reset` 이후 UI 가 노출해야 하는 것은 원인이 아니라 결과다.

- `soft` 또는 `mixed` 이후에는 작업 트리가 정리되지 않았으므로 `dirty` 로 본다.
- `hard` 이후에는 작업 트리가 정리되므로 `clean` 으로 본다.
- 상세 상태는 이번 단계에서 분리하지 않는다.

권장 라벨:

- `dirty`
- `clean`

권장 설명:

- `dirty`: current branch 에 local changes 가 남아 있다
- `clean`: working tree clean

### 2. stash 는 Graph 포인터 hover 시 존재 여부를 보여준다

stash 는 별도의 브랜치 상태가 아니라 복구 메타데이터로 취급한다.

이번 단계에서는 다음만 먼저 노출한다.

- 해당 시점에 stash 가 있는지
- stash 개수
- 최근 stash 요약

stash 의 시각적 표현은 포인터 fg color/marker 스타일과 Mode 패널 목록을 함께 쓰는 방향으로 둔다.

### 3. restore 보다 working tree clean 이 정확한 표현이다

현재 시나리오에서 사용자가 원하는 것은 일반적인 restore 라기보다, 현재 작업물을 정리해 다음 작업을 가능하게 만드는 행위다.

따라서 UI 와 상태 설명은 다음 표현을 우선한다.

- `working tree clean`
- `stash`

`restore` 는 필요할 경우 별도 파일 복원 기능으로 분리한다.

## 상태 모델

### 작업 트리 상위 상태

이번 기능에서 관리할 상위 상태는 두 개면 충분하다.

```text
clean
dirty
```

`dirty` 는 다음 조건을 포괄한다.

- staged changes
- unstaged changes
- untracked files
- soft / mixed reset 이후 남은 변경사항

이번 단계에서는 세부 원인을 분리하지 않는다.

### reset 모드

`reset` 은 하나의 action 으로 유지하고, 선택자만 분리한다.

```go
type ResetMode string

const (
    ResetModeSoft  ResetMode = "soft"
    ResetModeMixed ResetMode = "mixed"
    ResetModeHard  ResetMode = "hard"
)
```

### 작업 트리 상태

```go
type WorktreeState string

const (
    WorktreeClean WorktreeState = "clean"
    WorktreeDirty WorktreeState = "dirty"
)
```

### 권장 상태 확장

`state.Status` 또는 그에 준하는 UI 상태에는 최소한 아래 두 가지가 추가되어야 한다.

- `ResetModePick` 계열 모드
- `OutcomePreview` 계열 모드

이 둘은 서로 다른 역할을 갖는다.

- `ResetModePick`: `soft / mixed / hard` 선택과 confirm 진입
- `OutcomePreview`: 선택된 reset 결과와 작업 트리 영향 설명

## UX 흐름

### reset 흐름

1. Graph 섹션에서 특정 포인터에 `s` 를 입력한다.
2. reset trigger 가 열린다.
3. confirm 에서 `soft / mixed / hard` 를 선택하고 preview 정보를 함께 확인한다.
4. 선택된 mode 로 reset 을 실행한다.
5. repo status 를 다시 읽는다.
6. `soft / mixed` 면 `dirty` 로 전환한다.
7. `hard` 면 `clean` 으로 유지한다.
8. `dirty` 인 경우에만 `working tree clean` 과 `stash` 를 활성화한다.

### stash 흐름

1. 사용자가 Graph 포인터를 hover 한다.
2. 해당 시점의 stash 존재 여부를 보여준다.
3. dirty 상태에서만 stash 생성 / 복구 관련 행동을 활성화한다.
4. stash 생성 후에는 working tree 를 clean 으로 되돌린다.
5. Graph 포커스가 해당 commit 에 도달하면 Mode 패널에서 stash 목록을 보여준다.

## Git command 매핑

### reset

| 사용자 선택 | Git command | 결과 | 상위 상태 |
|---|---|---|---|
| soft | `git reset --soft <target>` | HEAD 이동, index/worktree 유지 | dirty |
| mixed | `git reset --mixed <target>` | HEAD/index 이동, worktree 유지 | dirty |
| hard | `git reset --hard <target>` | HEAD/index/worktree 정리 | clean |

### working tree clean

`working tree clean` 은 복구 목적의 별도 명령이라기보다, 현재 작업물을 정리해 clean 상태로 만드는 동작으로 본다.

후보 구현은 다음 중 하나로 정리한다.

- 파일 복원 전용 command
- 현재 상태를 clean 으로 만드는 command
- stash 와 조합한 recovery flow

### stash

| 사용자 선택 | Git command | 결과 | 표시 가능 정보 |
|---|---|---|---|
| stash 생성 | `git stash push -m <msg>` | 현재 작업 보존 후 clean | 개수, 최신 메시지, 기준 branch |
| stash 목록 | `git stash list` | stash 존재 여부 확인 | 존재 여부, 개수 |
| stash 상세 | `git stash show -p stash@{n}` | 변경 내용 미리보기 | 선택적 |

## stash 추적 가능 범위

Git 은 stash 를 `.git` 내부의 ref / reflog 기반으로 관리한다.

따라서 다음은 추적 가능하다.

- 어떤 branch 에서 stash 가 생성되었는지
- stash 당시의 HEAD commit 이 무엇이었는지
- stash 메시지와 순서

이번 계획에서는 stash 가 생성된 정확한 시점과 기준 commit 을 최소 메타데이터로만 활용한다.

## UI 배치 원칙

### reset

- `reset` 진입점은 Graph 섹션에 둔다.
- reset mode 선택은 confirm 단계에서 제공한다.
- confirm 이후 추가 preview 가 필요한 경우 별도 status 로 분리한다.

### dirty

- dirty 는 현재 branch 의 플래그처럼 보이게 한다.
- 다만 의미는 branch 자체가 아니라 working tree 상태다.
- `main dirty` 또는 `local changes` 같은 축약 표현을 사용할 수 있다.
- 표시 형식은 브랜치 라벨 뒤에 보조 마커를 붙이는 방식이 적합하다.
- 예시: `l->branch-67b6 ⬇`
- 이 마커는 현재 브랜치에 local changes 가 남아 있음을 나타낸다.
- stash 가 연결된 포인터는 fg color 또는 보조 스타일로 약하게 강조한다.
- 강조의 기준은 stash 가 생성된 시점의 commit hash 와 현재 포커스한 commit hash 의 일치 여부다.

### stash

- stash 존재 여부는 Graph 포인터 hover 시 표시한다.
- stash 시각화는 후속 단계에서 더 다듬는다.
- 현재 단계에서는 메타 정보 요약과 상세 목록을 함께 제공한다.
- 포커스한 commit 에 연결된 stash 가 여러 개면 목록형으로 보여준다.

## 제약

1. 이번 단계에서는 staged / unstaged / untracked 를 분리하지 않는다.
2. `dirty` 와 `clean` 만 유지한다.
3. stash 시각화는 hover 기반 요약으로 시작한다.
4. Graph 내부에 stash 를 억지로 그리지 않는다.
5. `reset` 과 `stash` 는 같은 action 계열이지만, 상태 모델은 분리한다.
6. stash 표시는 Graph row 스타일과 Mode 상세 목록을 함께 사용한다.

## 구현 순서

1. `reset mode` enum 을 추가한다.
2. `WorktreeState` 상위 상태를 추가한다.
3. `git.Status` 에서 `dirty / clean` 을 계산하는 기준을 정한다.
4. `state.Status` 에 `ResetModePick` / `OutcomePreview` 계열 모드를 추가한다.
5. Graph 섹션 `reset` confirm 에 `soft / mixed / hard` 선택자를 붙인다.
6. reset 실행 후 status refresh 흐름을 연결한다.
7. `dirty` 상태에서만 `working tree clean` 과 `stash` 를 활성화한다.
8. stash 존재 여부를 Graph hover 경로에 연결한다.
9. 관련 테스트를 추가한다.

## Code Sketch

### `internal/state/state.go`

`dirty / clean` 은 UI flow가 아니라 worktree 상태이므로 별도 필드로 둔다.

```go
type WorktreeState string

const (
	WorktreeClean WorktreeState = "clean"
	WorktreeDirty WorktreeState = "dirty"
)

type Status struct {
	Mode       Mode
	Action     Action
	Block      BlockReason
	Title      string
	Message    string
	Detail     string
	Targets    []TargetItem
	TargetIdx  int
	Selected   string
	ResetMode  ResetMode
	Worktree   WorktreeState
	CanExecute bool
}
```

### `internal/git/repo.go`

stash는 reflog 기반이므로, `stash list`와 각 entry의 부모 commit을 함께 읽으면 된다.

```go
type StashEntry struct {
	Ref     string
	Hash    string
	Base    string
	Message string
}

func (r *Repo) Stashes(ctx context.Context) ([]StashEntry, error) {
	lines, err := r.gitLines(ctx, "stash", "list", "--format=%gd|%H|%gs")
	if err != nil {
		return nil, err
	}

	entries := make([]StashEntry, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}

		entry := StashEntry{
			Ref:     strings.TrimSpace(parts[0]),
			Hash:    strings.TrimSpace(parts[1]),
			Message: strings.TrimSpace(parts[2]),
		}

		if parents, err := r.git(ctx, "show", "-s", "--format=%P", entry.Ref); err == nil {
			fields := strings.Fields(strings.TrimSpace(parents))
			if len(fields) > 0 {
				entry.Base = fields[0]
			}
		}

		entries = append(entries, entry)
	}
	return entries, nil
}
```

### `internal/app/model.go`

stash 목록과 commit 기준 grouping을 모델이 들고 있으면, graph row와 mode detail이 같은 데이터를 공유할 수 있다.

```go
type model struct {
	repo              *git.Repo
	status            state.Status
	repoStatus        git.Status
	activeSection     graphSection
	sectionCursor     map[graphSection]int
	graphLaneCursor   int
	graphScroll       int
	awaitingGoTop     bool
	branchOpen        bool
	branchDraft       string
	branchBase        string
	width             int
	height            int
	commitLimit       int
	err               error
	handshakeCommits  map[string]bool
	pullIsFastForward bool

	stashEntries []git.StashEntry
	stashByBase  map[string][]git.StashEntry
}

func (m model) stashesForCommit(hash string) []git.StashEntry {
	if len(m.stashByBase) == 0 {
		return nil
	}
	return m.stashByBase[hash]
}
```

### `internal/app/commands.go`

stash는 `status` 갱신과 분리해서 읽어도 되고, refresh 흐름에 같이 붙여도 된다. 아래처럼 별도 command로 두면 모델이 단순해진다.

```go
type stashLoadedMsg struct {
	entries []git.StashEntry
	err     error
}

func loadStashState(repo *git.Repo) tea.Cmd {
	return func() tea.Msg {
		entries, err := repo.Stashes(context.Background())
		return stashLoadedMsg{entries: entries, err: err}
	}
}

func refreshRepoState(repo *git.Repo, limit int) tea.Cmd {
	return tea.Batch(loadRepoState(repo, limit), loadStashState(repo))
}
```

### `internal/app/update.go`

stash 정보를 받은 뒤에는 commit hash 기준으로 묶어서 graph/view에서 같이 쓴다.

```go
func handleStashUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	msg2, ok := msg.(stashLoadedMsg)
	if !ok {
		return m, nil
	}
	if msg2.err != nil {
		m.status = state.New().WithBlocked(state.BlockUnknown, "Stash scan failed.", msg2.err.Error())
		return m, nil
	}

	m.stashEntries = msg2.entries
	m.stashByBase = make(map[string][]git.StashEntry, len(msg2.entries))
	for _, entry := range msg2.entries {
		m.stashByBase[entry.Base] = append(m.stashByBase[entry.Base], entry)
	}
	return m, nil
}
```

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case stashLoadedMsg:
		return handleStashUpdate(m, msg)
	case loadedMsg, refreshedMsg, tickMsg:
		return handleLifecycleUpdate(m, msg)
	case fetchedMsg, preparedMsg, pullCheckedMsg, previewMsg, pushFetchedMsg, pullFetchedMsg, pullPreviewReadyMsg:
		return handleFetchUpdate(m, msg)
	case executedMsg:
		return handleExecutedUpdate(m, msg)
	case createdBranchMsg:
		return handleBranchUpdate(m, msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	default:
		return m, nil
	}
}
```

### `internal/app/view_graph.go` / `internal/app/graph_render.go`

포인터에 stash가 붙어 있으면 약한 fg color 강조를 주고, count를 표시한다.

```go
func (m model) renderGraphContent(width, height int) string {
	...
	for i := start; i < end; i++ {
		stashCount := len(m.stashesForCommit(rows[i].Commit.Hash))
		lineStr := renderGraphLine(rows[i], graphActive && i == m.sectionCursor[sectionGraph], graphActive, m.graphLaneCursor, m.repoStatus.LocalBranches, graphColWidth, isHandshake, stashCount)
		lines = append(lines, lineStr)
		...
	}
	return fitBlockLines(lines, height)
}

func renderGraphLine(row graphRow, selected bool, graphActive bool, laneCursor int, localBranches []string, graphColWidth int, isHandshake bool, stashCount int) string {
	...
	if stashCount > 0 && !selected {
		refs = stashMark.Render(refs)
		refs += " " + muted.Render(fmt.Sprintf("stash:%d", stashCount))
	}
	...
}
```

```go
var stashMark = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
```

### `internal/app/view_detail.go`

Mode 패널은 focus commit에 연결된 stash 목록을 펼쳐 보여준다.

```go
focus := currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])
if focus.Hash != "" {
	lines = append(lines, fmt.Sprintf("focus: %s", shorten(focus.Hash, max(width-7, 0))))
	lines = append(lines, focusParentLines(focus, width)...)
	if branchLines := focusBranchSummaryLines(focus, width); len(branchLines) > 0 {
		lines = append(lines, "branches:")
		lines = append(lines, branchLines...)
	}
	if stashes := m.stashesForCommit(focus.Hash); len(stashes) > 0 {
		lines = append(lines, "stash:")
		for _, entry := range stashes {
			lines = append(lines, fmt.Sprintf("  - %s %s", entry.Ref, shorten(entry.Message, max(width-6, 0))))
		}
	}
}
```

### `internal/app/key_handling.go`

stash는 읽기 전용 표시부터 먼저 붙이고, 이후 생성/복원은 별도 action으로 확장한다.

```go
func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.branchOpen {
		return m.handleBranchOpenKey(msg)
	}
	switch m.status.Mode {
	case state.ModeTargetPick:
		return m.handleTargetPickKey(msg)
	case state.ModeResetModePick:
		return m.handleResetModePickKey(msg)
	case state.ModeConfirm:
		return m.handleConfirmKey(msg)
	case state.ModeOutcomePreview:
		return m.handleOutcomePreviewKey(msg)
	case state.ModeBrowse:
		return m.handleBrowseKey(msg)
	default:
		return m, nil
	}
}
```

## Test Sketch

```go
func TestResetModePickShowsPreviewAndModeSelection(t *testing.T)
func TestPullIsExposedInLocalAndGraph(t *testing.T)
func TestStashListGroupsByBaseCommit(t *testing.T)
func TestGraphFocusShowsStashEntries(t *testing.T)
func TestWorktreeStateIsTrackedSeparatelyFromMode(t *testing.T)
```

## 테스트 항목

- soft reset 후 `dirty` 로 전환되는지
- mixed reset 후 `dirty` 로 전환되는지
- hard reset 후 `clean` 으로 유지되는지
- dirty 상태에서만 `working tree clean` 과 `stash` 가 활성화되는지
- Graph 포인터 hover 시 stash 존재 여부가 표시되는지
- reset confirm 에 soft / mixed / hard 선택자와 preview 정보가 함께 보이는지
- 특정 commit 포커스 시 Mode 패널에 stash 목록이 보이는지

## 내일 이어서 할 일

- [ ] 상태 enum 초안 확정
- [ ] reset mode 선택 UI 확정
- [ ] dirty / clean 표기 문구 확정
- [ ] stash hover 노출 포맷 초안 작성
- [ ] 테스트 항목 우선순위 정리

## 결론

이 기능은 `reset` 을 더 세분화하는 기능이 아니라, `working tree state` 를 더 정확히 보여주는 기능이다.

- reset 은 Graph 중심으로 시작한다.
- reset 결과는 `dirty / clean` 으로만 먼저 노출한다.
- stash 는 Graph hover 기반으로 존재 여부를 보여준다.
- `working tree clean` 이 이번 UX 에서 가장 정확한 복구 표현이다.
