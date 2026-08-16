# 조건부 저장소 상태 안내 추가

Subtask ID: `1.10`
상위 기능: `1` — 커밋 검사 및 유지보수자 판단 흐름
상태: 검토 완료 — 범위 축소
작성일: 2026-08-02

## 요약

Graph 제목 아래에 저장소 상태를 항상 표시하지 않고, 사용자의 판단이
필요한 예외 상황에서만 짧은 한 줄 안내를 표시한다.

```text
[1] Graph
State: diverged · main ↔ origin/main
────────────────────────────────────
* 91ab2c1  latest commit
```

정상 상태에서는 추가 안내를 표시하지 않는다.

```text
[1] Graph
────────────────────────────────────
* 91ab2c1  latest commit
```

표시 예시는 다음과 같다.

```text
State: detached HEAD · branch 선택 필요
State: no upstream · upstream 설정 필요
State: merge in progress · 충돌 해결 필요
State: dirty worktree · 작업 전 commit 또는 stash 필요
```

## 목표

Details 패널이나 작업 Preview를 열지 않아도, 즉시 주의해야 하는 저장소
상태를 Graph 영역에서 알아볼 수 있게 한다.

## 제품 결정

기존에 계획했던 항상 노출형 Unified Decision Card는 제거한다. 현재
Details, 작업 Preview, Review Popup, 실행 결과 화면에 이미 관련 정보가
존재하므로, 별도 카드를 추가하면 정보가 중복되고 Graph 표시 영역이
줄어든다.

대신 저장소에 주의가 필요한 경우에만 Graph 제목 아래에 조건부 상태 안내
한 줄을 표시한다. clean, synced, ahead-only, behind-only 같은 일반 상태는
기존 Graph와 Details 화면을 그대로 사용한다.

## 범위

- 저장소 상태의 기준은 기존 `git.Status`를 재사용한다.
- 예외 상태를 판별하는 순서를 정의한다.

  ```text
  로딩/오류
      ↓
  merge/rebase/cherry-pick 진행 중
      ↓
  detached HEAD / 저장소 없음 / upstream 없음 / remote 없음
      ↓
  diverged 또는 다음 작업을 막는 dirty worktree
      ↓
  안내 표시 안 함
  ```

- `[1] Graph` 제목 아래에 최대 한 줄의 안내를 표시한다.
- 상태명과 이미 확인된 경우에만 branch/upstream 이름을 함께 표시한다.
- 이 작업에서는 Git 명령을 추천하거나 자동 실행하지 않는다.
- 색상을 사용할 수 없어도 텍스트만으로 의미가 전달되어야 한다.
- 낮은 화면 높이에서는 Graph의 최소 한 행을 우선 보장한다.
- 안내 문구는 포커스, 스크롤, 키 바인딩, Overlay 우선순위에 영향을 주지
  않아야 한다.

## 범위에 포함하지 않는 내용

- 항상 표시되는 Unified Decision Card
- 다음 행동 추천, 판단 이유, 작업 영향 Preview
- 작업 완료 결과를 Graph 영역에 계속 남기는 기능
- 새로운 Git 명령이나 저장소 상태 의미 변경
- 자동 작업 실행, 새로운 Popup, 새로운 화면
- 기존 Details, Local, Remote, Tags 패널 교체

후속 작업 `1.11~1.14`는 Unified Decision Card가 항상 존재한다고 가정하지
않고, 기존 Details/Preview/Review/실행 결과 화면을 기준으로 다시 정리해야
한다.

## 재사용할 코드

- `internal/git/repo.go`: worktree, divergence, detached, upstream,
  진행 중인 Git 작업 상태
- `internal/app/status_helpers.go`: 저장소 상태를 UI 상태로 변환하는 로직
- `internal/app/view_shell.go`: Graph 제목과 Shell 레이아웃
- `internal/app/view_layout.go`: Graph 높이 계산
- `internal/app/view_sections.go`: 기존 상태 문구와 스타일 규칙
- `internal/app/view_detail.go`: 기존 branch, upstream, worktree 상세 표현

## 상태별 처리

| 입력 상태 | 표시 내용 | 기존 복구 흐름 |
|---|---|---|
| 로딩 중 | 기존 로딩 표시를 우선 사용 | 기존 로딩 흐름 유지 |
| 저장소 상태 조회 오류 | `Repository unavailable` 형태의 짧은 안내 | 기존 오류·새로고침 흐름 사용 |
| merge/rebase/cherry-pick 진행 중 | 진행 중인 작업 상태 | 기존 abort·충돌 해결 흐름 사용 |
| detached / upstream 없음 / remote 없음 | 빈 값 대신 명시적 상태명 표시 | 기존 branch/upstream 설정 흐름 사용 |
| diverged | 상태명과 확인된 branch/upstream 표시 | 기존 merge/rebase Preview 사용 |
| dirty worktree | 다음 작업을 막는 경우에만 표시 | 기존 commit/stash/clean 흐름 사용 |
| 빈 저장소 또는 정보 부족 | 선택 정보는 생략하고 빈 문구를 만들지 않음 | 기존 빈 상태 흐름 사용 |
| 좁거나 낮은 터미널 | 참조명 생략 후 필요하면 안내 자체 생략 | Graph 최소 한 행 유지 |

## 테스트

- 우선순위별 상태 판별을 순수 함수 단위로 테스트한다.
- 빈 상태, 누락된 branch/upstream, 조회 오류를 테스트한다.
- clean/synced/ahead/behind 상태에는 안내가 표시되지 않는지 확인한다.
- ANSI 색상 제거 후에도 상태명이 남는지 확인한다.
- 좁은 너비와 낮은 높이에서 Graph 첫 행이 사라지지 않는지 확인한다.
- 안내 표시가 포커스, 스크롤, 키 입력, Overlay 동작을 바꾸지 않는지
  확인한다.

## 구현 작업

1. `internal/app`에 순수 저장소 상태 안내 Projection을 추가한다.
2. Graph 제목 아래에 조건부 안내를 렌더링하고 높이 예산을 조정한다.
3. 일반 상태, 예외 상태, 빈 상태, 오류 상태, 터미널 경계 테스트를 추가한다.
4. `1.11~1.14` 계획문서에서 항상 존재하는 Decision Card 전제를 제거한다.

## 추후 검토 사항

- 조건부 안내를 출시한 뒤에도 사용자가 다음 행동을 놓치는지 확인한다.
- 추가 안내가 필요하다면 영구적인 Graph 배너가 아니라, 실제 판단이
  발생하는 Preview 또는 Review 화면에만 추가한다.
- 여러 화면에서 상태 모델을 공유해야 한다는 근거가 생긴 경우에만
  통합 Decision Model을 다시 검토한다.

## 검증 명령

```sh
go test ./internal/app ./internal/git
scripts/check
```

## CEO 검토 결과

### 결정

`SCOPE REDUCTION` 및 접근 방식 A를 승인한다. 필요한 최소 변경은 전체
저장소 정보를 보여주는 카드가 아니라, 예외 상태를 알려주는 안내 라인이다.

### 이미 존재하는 기능

- `internal/app/view_detail.go:38-110`에 커밋, branch/upstream, worktree,
  divergence 정보가 이미 있다.
- `internal/app/preview.go:23-43`에 merge/rebase 관계 판별과 커밋 수 표시가
  이미 있다.
- `internal/app/graph_action_review.go:13-58`에 merge/rebase 영향과 충돌
  가능성 설명이 이미 있다.
- `internal/app/execution_detail.go`와 lifecycle 흐름에 작업 완료 메시지가
  이미 있다.
- `internal/git/repo.go:19-60`에 안내에 필요한 저장소 상태 필드가 이미 있다.

### 제거한 위험

기존 계획은 모든 상태에서 최대 4줄의 Decision Card를 표시하려 했다.
이는 기존 화면과 정보를 중복하고, 판단이 필요하지 않은 순간에도 Graph
높이를 줄이는 문제가 있었다. 수정된 계획은 예외 상황에서만 안내하여
Graph 중심 UX와 기존 Preview/Review 흐름을 보존한다.

### 반드시 확인할 실패 상황

1. 오래된 새로고침 결과로 정상 상태를 잘못 표시하지 않아야 한다.
2. 진행 중인 merge/rebase/cherry-pick이 동기화 상태보다 우선해야 한다.
3. branch/upstream/remote 누락을 빈 문자열로 표시하지 않아야 한다.
4. 터미널 높이가 작아도 Graph의 모든 topology 행을 숨기지 않아야 한다.
5. 안내 문구가 포커스, 탐색, Git 실행에 영향을 주지 않아야 한다.

### 검토 상태

검토 중 구현 코드는 변경하지 않았다. 기존 사용자 변경사항인
`.taskmaster/tasks/tasks.json`, `docs/decisions.md`, 다른 초안 계획문서는
보존했다.
