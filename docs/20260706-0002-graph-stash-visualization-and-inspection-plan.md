# Graph Stash Visualization Plan

## 목적

`Graph` 섹션에서 stash 존재 여부를 더 분명하게 보여주고, 해당 stash가 있는 commit 을 눈에 띄게 읽을 수 있게 한다.

현재 구현은 stash 데이터를 이미 읽고 있다.

- `git stash list` 결과를 `StashEntry` 로 파싱한다.
- `BaseHash` 기준으로 stash 를 commit 에 연결한다.
- `Graph` focus 와 detail panel 에 stash summary 를 노출할 수 있다.

이번 계획의 핵심은 이 기존 기반을 정리해서, 사용자가 `Graph` 에서 stash 를 “보게” 하고, focus 시 의미를 읽을 수 있게 만드는 것이다.

## 사용자 목표

1. `Graph` 에서 특정 commit 에 stash 가 있는지 즉시 알 수 있어야 한다.
2. stash 가 있는 commit 을 포커싱하면 detail panel 에 의미 있는 요약이 보여야 한다.
3. stash 표현은 `Graph` 섹션 안에만 머무른다.
4. stash 목록 UI, pop 흐름, branch continuation 은 `Graph` 와 분리된 별도 문서로 다룬다.
5. stash 목록은 Global hotkey 로 열고, overlay popup 에서 읽는다.

## 범위

### 포함

- `Graph` row 의 stash 표현
- `Graph` focus 시 stash summary 표시
- stash 관련 tests

### 제외

- stash list UI
- stash apply / pop / drop
- stash 생성 흐름 재설계
- stash rename / edit
- stash 를 전역 패널에 노출하는 방식

## 이미 존재하는 기반

현재 코드에는 다음이 이미 있다.

- `internal/git/repo_exec.go` 의 `Repo.Stashes()`
- `internal/git/repo.go` 의 `StashEntry{Ref, Hash, BaseHash, Subject}`
- `internal/app/update_stash.go` 의 stash 로드 처리
- `internal/app/stash_state.go` 의 `groupStashesByBase()` / `stashesForCommit()`
- `internal/app/stash_view.go` 의 `stashSummaryLines()`
- `internal/app/graph_render.go` 의 row highlight / stash highlight 패턴
- `internal/app/view_graph.go` 와 `internal/app/view_detail.go` 의 focus / detail 렌더링

즉, 이번 작업은 stash 데이터를 새로 발명하는 작업이 아니다.
이미 있는 `BaseHash` 연결을 UI 계약으로 끌어올리는 작업이다.

## 핵심 결정

### 1. 활성화 조건은 “HEAD 에 stash 가 있을 때만”이 아니라 “repo 에 stash 가 하나라도 있을 때”다

`stash` 는 특정 branch 의 상태가 아니라, stash 생성 시점의 `BaseHash` 를 가진 별도 복구 메타데이터다.

그래서 활성화 조건을 `HEAD` 기준으로 묶으면 정보가 사라진다.

올바른 조건은 다음이다.

- stash 목록이 비어 있지 않으면 `Graph` 에 stash 표현을 보여준다.
- 각 row 는 `BaseHash == row.Commit.Hash` 인 stash 를 가진 경우에만 강조한다.
- focus 된 commit 에 stash 가 있으면 detail panel 에 summary 를 표시한다.

### 2. stash 목록의 표현 단위는 raw list 가 아니라 `BaseHash` 기준의 commit-centered view 다

`Graph` 는 commit 그래프이므로 stash 도 commit 에 매달린 메타데이터로 보는 편이 맞다.

따라서 확인 UI 는 다음 규칙으로 표현한다.

- 1차 키는 `BaseHash` 이다.
- 같은 `BaseHash` 를 가진 stash 들은 한 그룹으로 묶는다.
- 그룹 내부는 stash 생성 시점의 최신 순서로 보여준다.
- 각 항목은 `Ref`, `Subject`, `BaseHash` 를 함께 보여준다.

이렇게 하면 사용자는 “어느 commit 에서 stash 되었는지”와 “그 commit 에 몇 개가 쌓였는지”를 동시에 읽을 수 있다.

### 3. stash 표현은 row 내의 보조 신호와 detail panel 의 요약으로 나눈다

Graph row 안에서 모든 stash 를 펼치면 너무 무겁다.

권장 UX:

- row: 짧은 stash token 또는 amber/orange 계열의 작은 badge
- detail panel: 현재 focus commit 의 stash 요약

이렇게 나누면 row 는 시각 신호만 담당하고, 상세 정보는 오른쪽 패널이 담당한다.

### 4. 목록 UI, pop 흐름, branch continuation 은 별도 문서로 분리한다

Graph 문서는 `stash 가 있는 commit 이 보인다`에 집중한다.
Graph 는 stash의 존재를 보여주고, 전역 hotkey 는 stash 목록 popup 을 여는 진입점만 제공한다.

이후 흐름은 아래 문서로 나눈다.

- session-wide stash list UI: `docs/20260706-0003-stash-session-list-ui-plan.md`
- selected stash 에 대한 pop 실행: `docs/20260706-0004-stash-pop-execution-plan.md`
- new branch -> checkout -> pop continuation: `docs/20260706-0005-stash-continue-from-branch-plan.md`

## UX 제안

### Graph row 표현

stash 가 있는 row 에는 다음 중 하나를 사용한다.

- branch column 옆 작은 `stash` badge
- branch column 안의 compact marker
- row 전체의 약한 amber accent

추천은 `branch column` 옆의 작은 marker 이다.

이유:

- graph topology 를 덮지 않는다.
- branch / head / stash 신호를 구분할 수 있다.
- 좁은 폭에서도 유지된다.

### Detail panel 표현

focus commit 에 stash 가 있으면 detail panel 에 다음을 보여준다.

- stash 개수
- 가장 최근 stash `Ref`
- 가장 최근 stash `Subject`

예시:

```text
stashes: 2
- stash@{0} - WIP on main: add search
- stash@{1} - On feature/login: cleanup
```

## 구현 순서

1. `Graph` row 렌더에 stash marker 를 넣는다.
2. focus commit 의 detail panel summary 를 정리한다.
3. stash 관련 tests 를 추가한다.

## 테스트

다음 테스트가 필요하다.

```go
func TestGraphShowsStashMarkerForBaseCommit(t *testing.T)
func TestGraphDetailShowsStashSummaryForFocusedCommit(t *testing.T)
```

각 테스트가 잡아야 하는 예외는 다음이다.

- stash 가 없을 때 marker 가 안 보이는지
- stash 가 다른 commit 에 있을 때 잘못된 row 를 강조하지 않는지
- stash 가 여러 개일 때 순서가 뒤집히지 않는지

## 예외 케이스

- stash 목록이 비어 있으면 아무 marker 도 보이지 않아야 한다.
- `BaseHash` 가 없는 stash 는 UI 에 연결하지 않는다.
- stash 가 현재 `HEAD` 와 다른 commit 에 붙어 있어도 그 commit 에서 보여야 한다.
- stash 가 여러 개 붙은 commit 은 count 와 summary 둘 다 필요하다.
- merge / rebase conflict 중에도 stash 표시 자체는 유지돼야 한다.

## 완료 기준

- `Graph` 에서 stash 가 있는 commit 을 즉시 알아볼 수 있다.
- focus 된 commit 의 stash summary 가 detail panel 에 보인다.
- stash 표시와 stash inspect 는 `Graph` 섹션 밖으로 새지 않는다.
- stash 관련 테스트가 통과한다.
