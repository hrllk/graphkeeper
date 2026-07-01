# Local Pane Split Layout Plan

## 목적

우측 `Local` 패널이 섹션 이동에 따라 설명과 액션을 함께 담는 구조를 유지하면서도, 내용이 길어지면 하단이 잘리는 문제를 해결한다.

이번 계획의 핵심은 우측 `Local` 패널 내부를 두 영역으로 나누는 것이다.

- 좌측: 섹션 기본 정보
- 우측: 해당 섹션에서 가능한 `Actions`

가운데에 border 또는 separator 를 두어, 정보와 조작 가능 항목이 서로 다른 책임을 가진다는 점을 화면에서 바로 읽을 수 있게 한다.

## 범위 확정

이 문서의 1차 구현 범위는 상단 우측 `Local` 패널이다.

- 1차 적용 대상: `renderContextContent()`
- 후속 정리 대상: `renderDetailContent()`

즉, 먼저 화면 상단의 섹션 설명 / 액션 영역을 안정화하고, 하단 `Detail` 패널은 같은 규칙을 따라가도록 별도 정리한다.

## 증상

- 섹션을 바꿀 때마다 우측 `Local` 패널에 보여야 하는 정보가 늘어난다.
- 현재는 설명, 상태 요약, 액션 안내가 한 흐름으로 쌓인다.
- 패널 높이가 고정되어 있어서 내용이 많아지면 아래쪽이 잘린다.
- 사용자는 정보와 액션을 같은 덩어리로 읽게 되어, 무엇이 상태이고 무엇이 조작 가능한 항목인지 즉시 구분하기 어렵다.

## 원인

현재 우측 `Local` 내용은 한 컬럼에 계속 append 하는 방식이다.

- `renderContextContent()` 는 섹션별 기본 정보와 `Actions` 를 같은 리스트에 넣는다.
- `renderDetailContent()` 도 동일하게 `Actions` 를 마지막에 붙인다.
- 마지막에 `fitBlockLines()` 가 적용되므로, 높이가 모자라면 아래쪽이 잘린다.

즉, 지금 구조는 "내용을 줄인다"는 관점에는 맞지만, "무엇을 먼저 보여줄지"를 제어하기에는 너무 단순하다.

## 목표 상태

우측 `Local` 패널은 다음 원칙을 따른다.

1. 좌측은 섹션의 기본 정보만 보여준다.
2. 우측은 `Actions` 만 보여준다.
3. 가운데 border 를 두어 시각적으로 분리한다.
4. 섹션이 바뀌어도 좌측 정보와 우측 액션의 역할이 흔들리지 않는다.
5. 내용이 많아질 때는 기본 정보를 우선 보존하고, 액션 목록은 고정된 규칙으로 정리한다.

권장 화면 의미:

- 좌측: `focus`, `branches`, `stashes`, `target`, `worktree`, `sync` 같은 상태 정보
- 우측: `merge`, `rebase`, `pull`, `reset`, `new branch` 같은 실행 가능 항목

이렇게 나누면 사용자는 "지금 무엇을 보고 있는지"와 "지금 무엇을 할 수 있는지"를 즉시 구분할 수 있다.

## 범위

### 포함

- 상단 우측 `Local` 패널 내부 레이아웃 분리
- 좌측 기본 정보 / 우측 Actions 2열 구성
- 가운데 border 또는 separator 추가
- 최소 폭 기준과 잘림 우선순위 정의
- 높이 부족 시 우선순위 정리
- `renderContextContent()` 의 역할 재정의
- 관련 렌더 테스트 추가

### 제외

- Graph 레이아웃 자체 변경
- 섹션 이동 규칙 변경
- 실제 Git command 의미 변경
- Actions 목록 재정의
- `renderDetailContent()` 의 즉시 구조 변경

## 구현 방향

### 1. Local 패널을 두 컬럼으로 분리한다

현재 단일 문자열 렌더링을 유지하지 말고, 우측 `Local` 패널 내부를 `left / right` 로 쪼갠다.

권장 구조:

- `left`: 현재 섹션의 기본 정보
- `right`: `Actions`
- 가운데: `│` 또는 rounded border 계열 separator

권장 폭 예산:

- 전체 폭이 충분하면 `left 60% / separator 1ch / right 39%` 정도로 시작한다.
- 최소 폭이 부족한 경우에는 `left` 를 우선 보존하고 `right` 를 줄인다.
- `right` 가 너무 좁아지면 액션 줄 수를 줄이는 대신 separator 를 유지한다.
- separator 까지 포함한 전체 폭이 읽기 어려운 수준이면 split 을 포기하지 말고, 액션을 짧게 축약한다.

예시 형태:

```text
Local
[ basic info      ]│[ Actions          ]
[ focus / target   ]│[ m: merge         ]
[ branch / sync    ]│[ r: rebase        ]
[ stash / worktree ]│[ p: pull          ]
```

### 2. 컬럼별 책임을 분리한다

좌측은 읽기 정보만 담당하고, 우측은 행동 안내만 담당하게 한다.

좌측 후보 정보:

- Graph 섹션: focus commit, parent, branch summary, stash summary
- Current 섹션: target, worktree, sync state
- Remote / Tags 섹션: target, item count

우측 후보 정보:

- 섹션별 가능한 action 목록
- 비활성 사유
- shortcut 안내

### 3. 분리용 helper 를 만든다

`renderContextContent()` 안에서 직접 문자열을 이어 붙이기보다, 좌측/우측 내용을 따로 만들고 마지막에 합친다.

예시 방향:

```go
func (m model) renderLocalInfoLines(width, height int) []string
func (m model) renderLocalActionLines(width, height int) []string
func renderSplitLocalPanel(left, right []string, width, height int) string
```

이렇게 하면 다음 장점이 있다.

- 좌측과 우측의 높이 예산을 따로 관리할 수 있다.
- Actions 가 늘어나도 기본 정보가 같이 밀려나지 않는다.
- 다른 패널에도 같은 분할 패턴을 재사용하기 쉽다.

### 4. 높이 예산을 우선순위로 나눈다

한 컬럼 안에서 모든 것을 보여주려 하지 말고, 우선순위를 정한다.

권장 우선순위:

1. 현재 섹션 이름과 핵심 상태
2. 좌측 기본 정보
3. 우측 Actions

특히 `Actions` 는 길어질 수 있으므로, 우측 컬럼에서는 고정된 라인 수를 목표로 한다.

권장 잘림 규칙:

- 좌측은 핵심 상태가 먼저 보이도록 한다.
- 우측은 액션 이름이 잘리기 전에 사유 문구를 먼저 줄인다.
- 두 컬럼 모두 부족하면 마지막 보조 정보부터 생략한다.
- `fitBlockLines()` 에 의존해 전체를 한 번에 자르지 말고, 컬럼별로 먼저 정리한 뒤 합친다.

### 5. `Detail` 패널과의 정합성을 정한다

상단 `Local` 패널만 새 구조로 바꾸면, 같은 정보가 하단 `Detail` 과 다르게 보일 수 있다.

권장 방향:

- 1차: 상단 우측 `Local` 패널에 split layout 적용
- 2차: `Detail` 패널도 같은 정보 분리 규칙을 따르도록 정리

즉, 화면 상단에서 먼저 구조를 고정하고, 하단은 그 규칙을 따라가도록 맞춘다.
이번 문서에서는 하단 `Detail` 패널을 바로 바꾸지 않는다.

## 테스트 방향

아래 항목을 테스트로 고정한다.

1. `Local` 패널이 좌측 정보 / 우측 Actions 로 나뉘는지 확인한다.
2. 내용이 많아도 좌측 핵심 정보가 먼저 보존되는지 확인한다.
3. Actions 가 길어져도 우측 컬럼 안에서만 잘리는지 확인한다.
4. 섹션을 바꿔도 separator 와 2열 구조가 유지되는지 확인한다.
5. 좁은 화면에서도 separator 와 우선순위가 깨지지 않는지 확인한다.

## 구현 순서

1. 현재 `renderContextContent()` 와 `renderDetailContent()` 의 책임을 분리한다.
2. 좌측 기본 정보 렌더링 helper 를 만든다.
3. 우측 Actions 렌더링 helper 를 만든다.
4. 두 컬럼을 합치는 split renderer 를 만든다.
5. separator 와 width/height 분배 규칙을 고정한다.
6. 상단 `Local` 패널에 먼저 적용한다.
7. 좁은 화면에서 좌측 정보가 유지되는지 검증한다.
8. 필요하면 후속 문서에서 `Detail` 패널에도 동일한 분리 규칙을 적용한다.
9. 렌더 테스트를 추가한다.

## 완료 기준

- 우측 `Local` 패널에서 정보와 Actions 가 시각적으로 분리된다.
- 내용이 늘어나도 기본 정보와 액션이 같은 덩어리로 잘리지 않는다.
- 가운데 border 가 역할 구분을 명확하게 만든다.
- 섹션이 바뀌어도 패널 구조가 흔들리지 않는다.
- 좁은 화면에서도 split 이 일관되게 유지된다.

## 메모

이 문서는 `Actions` 목록을 줄이거나 늘리는 계획이 아니다.

먼저 패널 구조를 분리해 정보와 행위의 경계를 고정한 다음, 이후에 `Actions` 안정화 작업을 진행하는 순서가 더 안전하다.
