# Pull / Reset UI·UX 구현 문서

## 목적

이 문서는 `pull`과 `reset` 기능의 UI·UX를 먼저 확정하고, 내일 바로 구현을 이어갈 수 있도록 현재 의사결정과 남은 작업을 정리한다.

우선순위는 다음과 같다.

1. `pull`의 노출 위치와 활성 조건을 단순화한다.
2. `reset`은 현재 브랜치 기준의 `hard reset`만 먼저 제공한다.
3. `reset`은 타깃 선택 이후 confirm 단계에서 mode 선택과 preview를 함께 보여준다.
4. `merge` / `rebase`는 같은 선택 UI를 재사용하되 이번 문서의 범위에서는 부차적으로 둔다.

## 현재 관찰된 구현 상태

현재 코드에서는 다음 흐름이 이미 존재한다.

- `f`는 전역 fetch로 동작한다.
- `p`는 pull 진입점으로 존재한다.
- `m`, `r`, `s`는 Graph 섹션에서 merge / rebase / reset preview 용도로 사용된다.
- `space`는 Graph 섹션에서 checkout을 수행하지 않도록 막혀 있다.
- `Mode` 패널은 섹션별 액션을 보여주는 역할로 이미 사용 중이다.

즉, 기능 자체는 일부 연결되어 있지만, 사용자 입장에서는 아직 “어디에서 무엇을 눌러야 하는지”가 충분히 명확하지 않다.

## 핵심 판단

### 1. pull

`pull`은 그래프 포인터 탐색과 분리하되, 노출 지점은 `Local`과 `Graph` 둘 다 둔다.

이유:

- `pull`은 특정 커밋을 고르는 행위가 아니라 현재 브랜치 상태를 갱신하는 행위에 가깝다.
- Graph에서 임의 커밋을 대상으로 쓰게 만들면 사용자가 `fetch`, `merge`, `rebase`, `reset`과 혼동할 가능성이 높다.
- `Local`에서는 현재 브랜치 상태 갱신의 기본 진입점으로, `Graph`에서는 현재 브랜치 포인터가 포커스된 경우 보조 진입점으로 제공하는 편이 더 일관적이다.

권장 활성 조건:

- 현재 HEAD가 branch 상태일 것
- remote가 존재할 것
- upstream이 설정되어 있을 것
- 대상이 명확하지 않은 detached HEAD 상태는 비활성
- `Graph`에서는 현재 브랜치 포인터에만 pull을 노출할 것

권장 메시지:

- 활성 상태: `pull`
- 비활성 상태: `No upstream configured` 또는 `Detached HEAD`

### 2. reset

`reset`은 기본적으로 `hard reset`만 제공한다.

이유:

- soft / mixed / hard를 한 번에 넣으면 TUI에서 설명 비용이 커진다.
- 사용자는 `reset`을 실행했을 때 현재 브랜치 포인터와 작업 결과가 어떻게 바뀌는지 즉시 이해해야 한다.
- 안전성 측면에서 “무엇이 버려지는지”를 명확히 보여주는 편이 중요하다.
- 타깃 선택 이후 confirm 단계에서 `soft / mixed / hard`를 고르고 preview 정보를 함께 본 뒤 실행한다.

권장 표현:

- UI 라벨: `hard reset`
- 내부 액션: `ActionReset`
- 상태 설명: `reset will move current branch pointer to selected target`

## UI 배치 제안

### pull

권장 배치:

- `Local` 섹션의 기본 액션으로 노출
- `Graph` 섹션에서는 현재 브랜치 포인터에만 보조 진입점으로 노출

### reset

권장 배치:

- `Graph` 섹션의 액션으로 노출
- 대상은 그래프 포인터를 통해 선택
- `Current` 섹션에서는 실행하지 않음

이유:

- reset은 “현재 브랜치가 어디로 되돌아갈지”를 그래프 상에서 보여줄 수 있어야 한다.
- Graph에서 바로 타깃을 찍고 preview를 보여주는 흐름이 가장 자연스럽다.

## reset UX 상세안

### 대상 범위

허용 대상:

- local branch
- commit

제외 대상:

- remote branch
- origin 계열 참조

이유:

- reset은 현재 브랜치 포인터를 직접 움직이는 동작이므로, 원격 ref를 대상으로 삼는 것은 UX상 의미가 약하고 오해를 부른다.

### 실행 전 preview

사용자가 reset을 실행하려고 하면 아래 정보를 confirm 안에서 함께 보여준다.

- 현재 branch 이름
- 현재 HEAD commit
- 선택한 target commit
- target 선택 경로가 branch인지 commit인지
- reset 후 HEAD가 이동할 위치
- 사라질 가능성이 있는 커밋 범위
- 작업 트리 영향 경고
- reset mode 선택 UI(`soft / mixed / hard`)

### 그래프 예시 표기

confirm에서는 “현재 위치”와 “이동 후 위치”를 함께 보여줘야 한다.

예시 형식:

- `HEAD: main -> c1`
- `target: feature-x -> c0`
- `after reset: main -> c0`
- `commits between c1..c0 may be discarded`

실제 TUI에서는 아래 중 하나로 표현한다.

1. 현재 그래프를 유지한 채 `HEAD` 마커만 target으로 이동한 미리보기
2. 별도 preview 패널에서 `before / after`를 텍스트로 비교
3. reset 시 영향을 받는 commit 목록을 짧게 보여주는 축약표시

### 추천 방식

1번 + 2번 조합을 추천한다. 다만 구현상 preview를 별도 화면으로 강제하지 말고, confirm 패널 안에 같이 보여줘도 된다.

- 그래프 상에서는 포인터 이동을 시각화
- Mode 패널에서는 before / after를 텍스트로 설명

이 방식이 가장 직관적이고 구현 난이도도 과하지 않다.

## pull UX 상세안

### 동작 흐름

1. 사용자가 `pull`을 실행한다.
2. upstream / remote 상태를 확인한다.
3. 가능하면 fetch 후 pull 가능성을 판단한다.
4. fast-forward 가능하면 실행한다.
5. diverged 상태면 중단하고 원인을 보여준다.

### 권장 정책

- `pull`은 Graph 선택과 독립적으로 실행
- `pull`은 자동으로 target picker를 띄우지 않음
- target이 필요한 경우에는 현재 브랜치 upstream만 사용

### 실패 케이스

- detached HEAD
- upstream 없음
- remote 없음
- fetch 결과가 최신이 아님
- fast-forward 불가

## merge / rebase와의 관계

이번 문서의 범위는 pull/reset이지만, merge/rebase는 같은 선택 UI를 재사용하는 방향이 맞다.

권장 규칙:

- `merge` / `rebase`는 Graph 또는 Local에서 target을 고르게 한다
- 대상 활성화는 “같은 브랜치가 아닌 경우”로 제한한다
- 현재 브랜치와 동일 ref는 비활성 처리한다
- 충돌 해결 UI는 이번 단계에서 다루지 않는다

## 구현 순서

1. `pull`의 배치 위치를 `Local` 기본, `Graph` 보조 진입점으로 확정한다.
2. `pull`의 활성 조건과 비활성 메시지를 `Local` / `Graph` 기준으로 정리한다.
3. `reset`을 `hard reset`으로 명시하되, confirm 단계에서 `soft / mixed / hard` 선택을 노출한다.
4. `reset` 대상 허용 범위를 local branch / commit으로 제한한다.
5. `reset` preview 정보를 confirm 안에 포함시킨다.
6. `Mode` 패널의 액션 도움말을 pull/reset 기준으로 갱신한다.
7. 관련 테스트를 추가한다.

## Code Sketch

### `internal/state/state.go`

`pull/reset` 흐름을 UI 상태로 표시하려면 `Mode`와 `ResetMode`를 먼저 고정한다.

```go
type Mode string

const (
	ModeBrowse         Mode = "browse"
	ModePullCheck      Mode = "pull_check"
	ModeTargetPick     Mode = "target_pick"
	ModeOutcomePreview Mode = "outcome_preview"
	ModeResetModePick  Mode = "reset_mode_pick"
	ModeBlocked        Mode = "blocked"
	ModeLoading        Mode = "loading"
	ModeEmpty          Mode = "empty"
	ModeError          Mode = "error"
	ModeConfirm        Mode = "confirm"
)

type ResetMode string

const (
	ResetModeSoft  ResetMode = "soft"
	ResetModeMixed  ResetMode = "mixed"
	ResetModeHard   ResetMode = "hard"
)

type Status struct {
	Mode      Mode
	Action    Action
	Block     BlockReason
	Title     string
	Message   string
	Detail    string
	Targets   []TargetItem
	TargetIdx int
	Selected  string

	ResetMode  ResetMode
	CanExecute bool
}

func (s Status) WithResetModePick(action Action, message, detail string) Status {
	s.Mode = ModeResetModePick
	s.Action = action
	s.Block = BlockNone
	s.Title = "Reset"
	s.Message = message
	s.Detail = detail
	s.CanExecute = true
	if s.ResetMode == "" {
		s.ResetMode = ResetModeHard
	}
	return s
}
```

### `internal/app/key_handling.go`

새 모드가 들어오면 키 라우팅을 먼저 분기한다.

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

### `internal/app/key_handling_browse.go`

`pull`은 `Local`과 `Graph`에서 같은 helper를 타고, `reset`은 Graph 포커스에서 시작한다.

```go
func (m model) triggerPull() (tea.Model, tea.Cmd) {
	if pullReady(m.repoStatus) {
		m.status = state.New().WithLoading("Fetching upstream before pull...")
		return m, executeFetchForPull(m.repo, m.commitLimit)
	}
	m.status = actionPull(m.repoStatus)
	return m, nil
}

func (m model) handleBrowseGlobalKey(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch msg.String() {
	case "p":
		if m.activeSection == sectionCurrent {
			return true, m.triggerPull()
		}
		if m.activeSection == sectionGraph && isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
			return true, m.triggerPull()
		}
		return true, m, nil
	default:
		return false, m, nil
	}
}

func (m model) handleBrowseGraphKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s":
		focus := graph.CurrentFocus(m.repoStatus, m.sectionCursor[sectionGraph])
		if focus.Hash == "" || focus.Hash == "VIRTUAL_CONFLICT_HASH" {
			m.status = state.New().WithBlocked(state.BlockUnknown, "No reset target.", "Move the pointer onto a commit line.")
			return m, nil
		}

		currentOnly, targetOnly, err := m.repo.Divergence(context.Background(), "HEAD", focus.Hash)
		if err != nil {
			m.status = state.New().WithBlocked(state.BlockUnknown, "Reset preview failed.", err.Error())
			return m, nil
		}

		preview := buildResetPreview(focus.Hash, m.repoStatus, currentOnly, targetOnly)
		m.status = state.New().WithResetModePick(state.ActionReset, preview.Message, preview.Detail+
			"\n\ns: soft  •  m: mixed  •  h: hard  •  enter: execute  •  esc: back")
		m.status.Selected = focus.Hash
		m.status.ResetMode = state.ResetModeHard
		return m, nil
	default:
		return m, nil
	}
}
```

### `internal/app/key_handling_confirm.go`

`reset` 실행은 mode 선택 상태에서만 가능하게 둔다.

```go
func (m model) handleResetModePickKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s":
		m.status.ResetMode = state.ResetModeSoft
		m.status.Title = "Reset (" + string(m.status.ResetMode) + ")"
		return m, nil
	case "m":
		m.status.ResetMode = state.ResetModeMixed
		m.status.Title = "Reset (" + string(m.status.ResetMode) + ")"
		return m, nil
	case "h":
		m.status.ResetMode = state.ResetModeHard
		m.status.Title = "Reset (" + string(m.status.ResetMode) + ")"
		return m, nil
	case "enter":
		target := m.status.Selected
		mode := m.status.ResetMode
		m.status = state.New().WithLoading("Running " + string(mode) + " reset...")
		return m, executeReset(m.repo, target, mode, m.commitLimit)
	case "esc", "n":
		m.status = deriveStatus(m.repoStatus)
		return m, nil
	default:
		return m, nil
	}
}
```

### `internal/app/commands.go` and `internal/app/update_execute.go`

실행 커맨드는 reset mode를 그대로 Git command에 매핑하고, 완료 후 상태를 다시 읽어야 한다.

```go
type executedMsg struct {
	action    state.Action
	target    string
	resetMode state.ResetMode
	status    git.Status
	err       error
}

func executeReset(repo *git.Repo, target string, mode state.ResetMode, limit int) tea.Cmd {
	return func() tea.Msg {
		if target == "" {
			return executedMsg{action: state.ActionReset, resetMode: mode, err: fmt.Errorf("target is empty")}
		}

		args := []string{"reset", "--hard", target}
		switch mode {
		case state.ResetModeSoft:
			args = []string{"reset", "--soft", target}
		case state.ResetModeMixed:
			args = []string{"reset", "--mixed", target}
		case state.ResetModeHard:
			args = []string{"reset", "--hard", target}
		}

		_, err := repo.Run(args...)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionReset, target: target, resetMode: mode, err: statusErr}
		}
		return executedMsg{action: state.ActionReset, target: target, resetMode: mode, status: status, err: err}
	}
}
```

```go
func handleExecutedUpdate(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
	msg2, ok := msg.(executedMsg)
	if !ok {
		return m, nil
	}

	if msg2.action == state.ActionReset {
		rows := graph.Rows(msg2.status)
		if rowIdx := graph.FindRowByHash(rows, msg2.status.Head); rowIdx >= 0 {
			m.sectionCursor[sectionGraph] = rowIdx
			m.graphScroll = clampScroll(rowIdx, len(rows), graphPageSize(&m))
		}

		syncBrowseState(&m, msg2.status)
		m.status = deriveStatus(msg2.status)
		m.status.Message = fmt.Sprintf("%s reset completed to %s.", strings.ToUpper(string(msg2.resetMode)), shorten(msg2.target, 7))
		return m, nil
	}

	// existing branches keep their current behavior
	return m, nil
}
```

### `internal/app/view_sections.go`

`pull`과 `reset` 도움말은 섹션 기준으로 다르게 보여준다.

```go
case sectionCurrent:
	if pullReady(m.repoStatus) {
		lines = append(lines, "• p: pull           • P: push")
	} else {
		lines = append(lines, disabled.Render("• p: pull")+"   "+muted.Render("(no upstream / detached)"))
	}
case sectionGraph:
	if isLocalGraphPointer(m.repoStatus, m.sectionCursor[sectionGraph], m.graphLaneCursor) {
		lines = append(lines, "• p: pull           • s: reset")
	} else {
		lines = append(lines, disabled.Render("• p: pull")+"   "+disabled.Render("• s: reset")+" "+muted.Render("(current branch only)"))
	}
```

## Test Sketch

```go
func TestPullIsAvailableFromLocalAndGraph(t *testing.T)
func TestPullIsBlockedOutsideCurrentBranchContext(t *testing.T)
func TestResetOpensModePickWithPreview(t *testing.T)
func TestResetModePickExecutesSelectedMode(t *testing.T)
func TestResetConfirmRendersModeAndPreviewInDetailPane(t *testing.T)
```

## 테스트 항목

- `pull`이 detached HEAD에서 비활성인지
- `pull`이 upstream 없는 브랜치에서 비활성인지
- `pull`이 정상 상태에서 실행되는지
- `pull`이 Local과 Graph 섹션에서 올바르게 노출되는지
- `reset`이 Graph 섹션에서만 진입 가능한지
- `reset` 대상이 remote branch면 거부되는지
- `reset` confirm이 mode 선택과 preview 정보를 함께 보여주는지
- `reset` 실행 전 confirm 단계가 존재하는지
- `reset`이 hard reset으로만 동작하는지

## 내일 이어서 할 일

- [ ] `pull`의 Local / Graph 노출 문구를 확정한다
- [ ] `reset` confirm 문구를 확정한다
- [ ] `reset`의 before / after 표시 방식 결정
- [ ] target 선택기에서 remote branch 제외
- [ ] 테스트 케이스 추가
- [ ] 구현 후 실제 repo에서 흐름 검증

## 결론

현재 단계에서는 `pull`과 `reset`을 “서로 다른 성격의 작업”으로 분리하는 것이 가장 안전하다.

- `pull`은 Local과 Graph에서 현재 브랜치 기준으로 갱신하는 동작
- `reset`은 Graph에서 대상 선택 후 confirm 안에서 mode와 preview를 함께 보고 실행하는 hard reset

이렇게 고정하면 이후 `merge` / `rebase`도 같은 타깃 선택 구조 위에 얹을 수 있다.
