# 다음 행동 추천 추가

Subtask ID: `1.11`
상위 기능: `1` — 커밋 검사 및 유지보수자 판단 흐름
상태: 초안
작성일: 2026-08-02

## 목표

저장소 상태를 사용자가 이해하기 쉬운 다음 행동으로 연결한다. 사용자가
ahead, behind, diverged 상태를 Git 명령으로 직접 해석하지 않아도 되게 한다.

## 사용자 결과

사용자는 별도 Git 문서를 찾거나 pull, merge, rebase, push, stash 중 무엇을
선택할지 추측하지 않고 “다음에 무엇을 해야 하는가?”에 답할 수 있다.

## 범위

- 저장소 상태에서 행동으로 연결되는 결정 규칙을 정의한다.
- pull, push, merge, rebase, stash, checkout, no-action 상태를 다룬다.
- 주요 추천 행동과 보조 행동을 구분한다.
- 추천 결과는 기존 Details, Preview, Review 화면에서 표시한다.
- 기존 키 입력 흐름으로 추천 행동을 실행할 수 있게 한다.
- 상태가 모호하거나 사용자 입력이 필요한 경우 추천하지 않는다.

## 재사용할 코드

- `internal/app/preview.go`: merge/rebase 판단 로직
- `internal/app/graph_action_review.go`: 기존 Review 및 확인 상태
- `internal/app/key_handling_browse.go`: 기존 행동 진입점
- `internal/app/view_projection.go`: 화면 Projection 경계

## 완료 기준

- diverged branch에서 대상이 명확한 merge 또는 rebase를 추천한다.
- behind-only branch에서 pull 또는 fast-forward를 추천한다.
- upstream이 있는 ahead-only branch에서 push를 추천한다.
- dirty worktree에서는 필요한 경우 commit 또는 stash를 먼저 안내한다.
- upstream 없음, detached HEAD, remote 없음은 잘못된 추천 대신 설명 상태를
  표시한다.
- 추천 규칙을 렌더링과 분리된 순수 함수로 만들고 표 기반 테스트를 작성한다.
- reason 또는 impact 정보가 없어도 없는 내용을 만들어내지 않고 추천만 표시할
  수 있다.

## 범위에 포함하지 않는 내용

- 추천 행동 자동 실행
- 사용 이력 또는 기계학습 기반 추천 순위
- 추천 이유 설명의 공통 문구 정리. 별도 `1.12`에서 다룬다.
- 항상 표시되는 Unified Decision Card

## 의존성

- `1.10`의 조건부 저장소 상태 안내와 상태 분류 결과
- 기존 merge/rebase 및 pull 사전 검사 동작 유지

## 검증

```sh
go test ./internal/app ./internal/git
scripts/check
```
