# Graph Lane Rendering Plan

## Purpose

Checkout 이후 graph를 현재 branch 기준으로 다시 렌더링할 때 lane이 흔들리거나, 부모-자식 연결이 누락되거나, 불필요한 `|`가 남는 문제를 해결한다.

이번 작업의 목표는 단순히 화면 출력만 맞추는 것이 아니라, lane 배치 규칙을 명시해서 checkout, refresh, fetch 이후에도 같은 입력이면 같은 graph가 나오도록 만드는 것이다.

## Current Behavior

현재 데이터 흐름은 다음과 같다.

1. `internal/git/repo.go`에서 `git log --topo-order`로 commit 목록을 가져온다.
2. `internal/app/model.go`의 `graphRows`가 commit을 위에서 아래로 순회한다.
3. `active []laneRef`를 증분 갱신하면서 각 row의 `Before`, `After`, `Lane`을 만든다.
4. `internal/app/view.go`가 `Before`, `After`, `Lane`만 보고 `*`, `|`, `/`를 렌더링한다.

현재 구조의 문제는 lane 정렬과 commit topology 처리가 같은 루프 안에서 섞여 있다는 점이다. checkout branch를 앞으로 당기는 순간 sibling lane들의 상대 위치와 merge-base collapse 규칙이 같이 흔들린다.

## Fixed Requirements

1. Checkout 진행 시 current branch의 local lane은 첫 번째 lane으로 고정한다.
2. 1번 반영 후 나머지 local branch lane도 재배치한다.
3. 재배치 시 sort 규칙이 필요하다. 단순 alphabetical보다 topo 정보를 우선 검토한다.
4. Remote lane은 local branch lane을 보조하는 성격으로 둔다. current branch의 remote lane이 local lane보다 앞서면 안 된다.
5. Parent-child 관계는 lane 재배치 이후에도 끊기면 안 된다.
6. 동일 입력 graph는 매번 동일하게 렌더링되어야 한다.

## Proposed Direction

표시되는 commit 순서는 계속 `git log --topo-order`를 따른다. 대신 lane 번호는 현재처럼 매 row에서 즉흥적으로 밀고 당기는 방식이 아니라, checkout branch 기준으로 별도 lane assignment를 계산한 뒤 graph row 생성에 반영한다.

즉, 알고리즘을 두 단계로 분리한다.

1. `topo-order commit list -> branch family priority 계산`
2. `branch family priority -> row별 lane state 계산`

이렇게 분리하면 checkout이 바뀌어도 commit 표시 순서는 안정적이고, lane priority만 다시 계산된다.

## Lane Concepts

`laneRef`는 현재처럼 유지하되 의미를 더 엄격히 둔다.

```go
type laneRef struct {
    Hash   string
    Family string
    Side   laneSide // local, remote, other
}
```

정의:

- `Family`: branch 이름 기준의 묶음. `tmp3`와 `origin/tmp3`는 같은 family다.
- `Side`: 같은 family 안에서 local/remote/other를 구분한다.
- `Hash`: 현재 row 시점에서 해당 lane이 향하고 있는 commit hash다.

## Lane Sort Rule

후보 규칙은 다음 순서로 적용한다.

1. Current branch local lane
2. Current branch remote lane
3. Topology상 current branch와 가까운 local branch lane
4. 나머지 local branch lane
5. 각 local branch에 대응하는 remote lane
6. orphan/unknown lane

동일 그룹 안에서는 다음 우선순위를 적용한다.

1. merge-base가 current branch HEAD에 가까운 branch
2. topo-order에서 decoration이 먼저 등장한 branch
3. branch name lexicographic order

이 규칙의 의도는 current branch를 왼쪽에 고정하되, 나머지 branch도 graph topology와 관련성이 높은 순서로 배치하는 것이다.

## Topology Distance

topology 기반 정렬을 위해 branch별 distance를 계산한다.

입력:

- current branch HEAD
- local branch tip hashes
- commit parent map
- topo-order commit index

계산:

1. current HEAD에서 first-parent chain을 따라 내려가며 `currentDepth[hash]`를 만든다.
2. 각 branch tip에서 parent 방향으로 내려가며 current chain과 만나는 첫 commit을 찾는다.
3. 만난 commit의 currentDepth가 작을수록 current branch와 가깝다고 본다.
4. 만나지 못한 branch는 topo-order decoration index로 fallback한다.

검토 필요:

- first-parent distance만 쓸지, 전체 parent BFS distance를 쓸지 결정해야 한다.
- TUI graph는 사람 눈에는 first-parent 기준이 더 예측 가능하지만, merge-heavy repository에서는 BFS가 더 정확할 수 있다.

초기 제안:

- 1차 구현은 first-parent distance를 사용한다.
- 문제가 확인되면 BFS distance를 추가한다.

## Rendering Algorithm Proposal

### Step 1. Build graph metadata

`graphRows` 시작 전에 다음 메타데이터를 만든다.

- `childrenByHash`
- `parentByHash`
- `branchTips`
- `decorationByHash`
- `familyPriority`

### Step 2. Seed lanes

초기 lane은 current branch를 반드시 포함한다.

현재 구현은 current branch의 remote tip이 없으면 seed를 만들지 않는 문제가 있다. checkout branch local lane은 remote 존재 여부와 무관하게 첫 lane에 있어야 한다.

초기 lane:

1. current branch local. remote tip이 없거나 local과 remote가 같은 commit이어도 반드시 seed한다.
2. current branch remote, 존재하고 local과 hash가 다를 때
3. topo priority에 따른 다른 local branch tip

### Step 3. Process commits in topo-order

각 commit row에서:

1. 현재 commit hash와 일치하는 active lane을 찾는다.
2. 없으면 해당 commit decoration 또는 fallback family로 lane을 만든다.
3. row의 display lane은 current branch family가 있으면 그 lane을 우선한다.
4. commit 처리 후 matched lane은 parent hash로 전진한다.
5. 여러 lane이 같은 parent hash로 모이면 collapse candidate로 표시하되, 바로 무조건 삭제하지 않는다.

### Step 4. Compact lanes safely

lane compaction은 별도 함수로 분리한다.

삭제 가능한 lane:

- 같은 `Hash`, 같은 `Family`, 같은 `Side`가 중복된 lane
- commit decorations와 branch tip으로 더 이상 구분할 필요가 없는 `other` lane

삭제하면 안 되는 lane:

- 같은 hash라도 `Family`가 다른 local branch lane
- local과 remote가 diverged 상태를 표현 중인 lane
- 아직 화면 아래에서 decoration이 등장할 branch family lane

### Step 5. Render connectors

connector 렌더링은 lane assignment가 안정된 뒤 최소 규칙만 적용한다.

초기 구현 범위:

- stable transition: connector 없음
- collapse transition: 한 줄 connector
- merge parent expansion: 한 줄 connector

주의:

- lane shift 전체를 connector로 그리려 하면 줄이 폭발하기 쉽다.
- connector는 graph state를 보완하는 역할이지, 잘못된 lane state를 숨기는 방식으로 쓰면 안 된다.

## Implementation Tasks

1. `graphRows` 내부의 lane priority 계산을 별도 함수로 분리한다.
2. `initialGraphLanes`가 remote tip 존재 여부와 무관하게 current branch local lane을 seed하도록 수정한다.
3. `prioritizeLaneRefs`를 단순 current-first partition에서 `familyPriority` 기반 sort로 교체한다.
4. `advanceGraphLanes`는 lane 전진만 담당하게 유지하고, lane 정렬/compaction은 별도 함수에서 처리한다.
5. `compactLaneRefs`를 추가해 중복 lane 제거 규칙을 명시한다.
6. `renderGraphConnectorLines`는 최소 connector만 유지한다.
7. checkout 후 `syncBrowseState`가 기존 row hash를 찾되, lane cursor는 새 row의 display lane으로 재설정하는지 확인한다.

## Test Plan

필수 테스트:

1. current branch local lane is first
2. current branch remote lane never precedes local lane
3. checkout from `tmp3` to `tmp1` reorders lanes deterministically
4. local sibling lanes keep topology-based order after current branch promotion
5. `origin/tmp3` and `tmp3` diverged lanes remain separate until merge-base
6. `37f0954 -> efb164e` parent relation remains visible in row state
7. stale far-right `|` does not remain after lanes collapse at common ancestor
8. no multi-line connector explosion for repeated single-parent commits
9. graph page size remains layout-based and does not shrink unexpectedly

샘플 fixture:

```text
1507a22 tmp3 local
dee56f4
7d23746 origin/tmp3
37f0954 -> efb164e
efb164e
a458b4b
b219ab5 tmp2
5df093e tmp1 current
a39d548 main, develop
3999588 origin/main
...
5525707 common ancestor
```

이 fixture는 반드시 unit test로 고정한다.

## Open Decisions

아래는 구현 전에 논의가 필요하다.

1. 나머지 local branch 정렬에서 first-parent distance를 사용할지, 전체 parent BFS distance를 사용할지
2. remote lane을 local branch 바로 옆에 붙일지, 모든 local lane 뒤에 둘지
3. 같은 common ancestor로 수렴한 branch lane을 언제 collapse할지
4. connector를 최소 표현으로 둘지, lane shift까지 표현할지

제안 기본값:

1. first-parent distance
2. local 바로 뒤에 해당 remote 배치
3. 동일 hash로 수렴했고 아래쪽에 해당 family decoration이 더 없으면 collapse
4. connector는 최소 표현 유지

## Risk

가장 큰 위험은 lane sort를 매 row마다 강하게 적용해서 parent-child 시각 연결이 끊기는 것이다. 따라서 lane sort는 row 시작 전에 무조건 적용하는 방식보다, commit 처리 후 compaction 단계에서 제한적으로 적용해야 한다.

두 번째 위험은 connector로 문제를 덮는 것이다. connector가 많아지면 TUI에서는 화면이 쉽게 망가지므로, graph state가 올바른지 먼저 검증해야 한다.

## Recommended Implementation Order

1. 현재 상태를 유지한 채 fixture test를 먼저 추가한다.
2. `initialGraphLanes`의 current branch seed 문제만 수정한다.
3. `familyPriority` 계산 함수를 추가하고 테스트한다.
4. `prioritizeLaneRefs`를 priority 기반으로 교체한다.
5. `compactLaneRefs`를 추가한다.
6. fixture graph rendering을 snapshot 또는 string assertion으로 검증한다.
7. 실제 checkout flow에서 `tmp1`, `tmp2`, `tmp3` 전환을 수동 확인한다.

## Final Review Before Implementation

결론: 이 방향으로 구현하면 checkout 이후 graph를 다시 그릴 때 발생한 lane 흔들림 문제를 해결할 수 있다. 다만 아래 보완 조건을 먼저 반영해야 한다. 보완 없이 바로 구현하면 같은 종류의 문제가 다시 발생할 가능성이 높다.

### Scenario Compliance Check

요구 시나리오:

1. lazy fetch로 최초 렌더링 시 40 line을 rendering하고, current branch 첫 행/lane 고정 후 나머지는 topo 방식으로 배치한다.
2. 40 line 이상 탐색할 때마다 lazy fetch를 진행한다.
3. 중간에 특정 branch로 checkout하면 다시 1번부터 수행한다.

현재 코드 기준 판단:

1. 부분 충족이다.

   `commitLimit` 기본값은 40이라 최초 repository status는 40 commits 기준으로 읽는다. 하지만 화면에 실제로 보이는 row 수는 `graphPageSize`가 terminal height로 계산하므로 40줄 전체가 한 번에 렌더링되는 구조는 아니다.

   또한 commit list는 `git log --topo-order`를 사용하므로 commit 표시 순서는 topo-order다. 하지만 lane 배치는 topo 기반 정렬이 아니라 `prioritizeLaneRefs`의 current-family 우선 partition에 가깝다. current branch 첫 lane 고정도 remote tip이 없는 경우 `initialGraphLanes`가 seed를 만들지 않아 보장되지 않는다.

2. 부분 충족이다.

   graph에서 아래로 이동할 때 cursor가 `commitLimit - 10` 근처에 도달하고 현재 rows 수가 `commitLimit`과 같으면 `commitLimit += 40` 후 repository state를 다시 읽는다. 다만 이것은 network `fetch`가 아니라 local `git log` limit을 늘리는 lazy load에 가깝다.

   `down/j` 이동에는 적용되어 있지만 `ctrl+d`, `G` 같은 이동 경로에도 동일하게 적용할지는 구현 시 확인이 필요하다.

3. 미충족이다.

   checkout 후 graph cursor와 scroll은 상단으로 돌리지만, `commitLimit`은 기존 값을 유지한다. 즉 이미 120 commits까지 lazy load한 상태에서 checkout하면 다시 40부터 시작하지 않고 120 기준으로 다시 읽는다.

   요구대로라면 checkout 성공 시 `commitLimit`을 기본값 40으로 reset하고, 그 limit으로 status를 다시 읽어야 한다. 단, terminal size 때문에 `WindowSizeMsg`가 `height * 2`로 즉시 limit을 키우는 현재 로직과 충돌할 수 있으므로 `initialCommitLimit`과 `viewportPrefetchLimit`의 정책을 분리해야 한다.

구현 기준:

- "fetch"라는 용어는 network fetch와 혼동된다. 여기서는 `lazy graph load`로 명명한다.
- 최초 graph data load 단위는 40 commits로 고정한다.
- 실제 화면 표시 row 수는 viewport에 맡긴다.
- checkout 성공 시 graph load limit, cursor, scroll, lane cursor를 초기 상태로 되돌린다.
- 이후 scroll 탐색 시 40 commits 단위로 추가 로드한다.

### Required Amendments

1. lane sort는 매 row 시작 시 무조건 적용하지 않는다.

   `active` lane을 매 row마다 강제 정렬하면 parent-child 연결이 끊긴다. 정렬은 다음 시점에만 적용한다.

   - 초기 lane seed 생성 시
   - checkout branch가 바뀐 뒤 graph를 처음 재계산할 때
   - commit 처리 후 lane compaction 단계에서 안전하다고 판단될 때

2. `familyPriority`는 branch family 단위로 고정한다.

   row를 내려가며 priority가 계속 바뀌면 같은 branch가 화면 중간에서 좌우로 이동한다. checkout 직후 `graphRows` 호출 1회 안에서는 `familyPriority`가 불변이어야 한다.

3. lane collapse는 `visibleFamiliesBelow` 또는 `remainingDecorations` 기준이 필요하다.

   같은 hash로 모였다는 이유만으로 lane을 접으면 `tmp2`, `tmp3`, `origin/tmp3` 같은 sibling lane이 사라진다. 반대로 접지 않으면 common ancestor 아래에서 우측 `|`가 계속 남는다.

   따라서 각 row index 기준으로 아래쪽에 남아 있는 branch decoration/family를 미리 계산한다.

   - 아래쪽에서 다시 표시될 family면 유지한다.
   - 아래쪽에서 다시 표시되지 않는 family이고 같은 hash로 수렴했으면 collapse 가능하다.
   - local/remote diverged 상태는 merge-base 이전까지 유지한다.

4. `other` lane은 family lane보다 짧게 살아야 한다.

   fallback으로 만든 `other` lane이 오래 유지되면 의미 없는 `|`가 생긴다. `other` lane은 commit parent 연결을 임시로 보여주는 용도로만 사용하고, branch/tag decoration이 없는 common path에서는 우선 collapse 대상이 되어야 한다.

5. connector 렌더링은 lane state 검증 이후에만 확장한다.

   connector를 먼저 늘리면 TUI 화면이 쉽게 깨진다. 구현 1차 범위에서는 다음만 허용한다.

   - stable transition: 추가 connector 없음
   - 2-lane collapse: 한 줄 `| /`
   - 3개 이상 collapse: 한 줄만 출력

   lane shift 전체를 `/`, `\`로 표현하는 작업은 별도 후속 작업으로 둔다.

### Implementation Go/No-Go

Go 조건:

- fixture test를 먼저 작성한다.
- `familyPriority` 계산 테스트가 통과한다.
- `compactLaneRefs` 테스트가 통과한다.
- connector 줄 수가 row당 1줄을 넘지 않는 테스트가 있다.

No-Go 조건:

- snapshot 없이 실제 렌더링 로직부터 수정하는 경우
- connector를 늘려서 parent-child 누락을 덮으려는 경우
- 같은 hash 수렴만 보고 local branch lane을 즉시 삭제하는 경우
- row마다 full sort를 수행하는 경우

### Final Implementation Shape

구현은 아래 형태가 가장 안전하다.

```go
metadata := buildGraphMetadata(commits, rs)
priority := buildFamilyPriority(metadata, rs.Branch)
active := initialGraphLanes(commits, rs, priority)

for rowIndex, commit := range commits {
    active = seedLaneForCommit(active, commit, metadata, priority)
    matches := laneMatches(active, commit.Hash)
    before := clone(active)
    after := advanceGraphLanes(before, matches, commit, rs.Branch)
    after = compactLaneRefs(after, rowIndex, metadata, priority)
    after = orderLaneRefs(after, priority)
    rows = append(rows, graphRow{Before: before, After: after, ...})
    active = after
}
```

주의할 점은 `orderLaneRefs`가 단순 sort가 아니라는 점이다. 이미 연결 중인 lane의 위치를 가능한 보존하고, 새로 생긴 lane 또는 collapse 이후 lane에 대해서만 priority를 반영해야 한다.

### Expected Outcome

위 보완안을 포함해 구현하면 다음 문제를 해결할 수 있다.

- checkout 후 current branch가 첫 lane에 고정되지 않는 문제
- checkout 전 branch lane이 첫 lane에 남는 문제
- `tmp3`, `origin/tmp3`의 merge-base 관계가 끊기는 문제
- common ancestor 아래에서 뜬금없는 우측 `|`가 남는 문제
- connector가 여러 줄로 폭발하는 문제

단, "git native `--graph`와 완전히 동일한 ASCII 모양"은 목표가 아니다. 목표는 현재 TUI 구조 안에서 deterministic하고 읽을 수 있는 branch-lane graph를 만드는 것이다.
