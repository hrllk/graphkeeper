# Branch 판단 맥락 설명

Subtask ID: `1.12`
상위 기능: `1` — 커밋 검사 및 유지보수자 판단 흐름
상태: 초안
작성일: 2026-08-02

## 목표

추천 행동이 나온 branch 관계를 설명해 사용자가 이유 없는 명령 추천으로
받아들이지 않게 한다.

## 사용자 결과

사용자는 현재 branch 이름, 대상 branch 이름, 커밋 수, 관계 설명을 통해
“Graphkeeper가 왜 이 행동을 추천하는가?”에 답할 수 있다.

## 범위

- already aligned, fast-forward available, target included, behind-only,
  ahead-only, diverged 상태의 표준 문구를 정의한다.
- 모든 merge/rebase 판단에서 현재 ref와 대상 ref를 표시한다.
- 확인 가능한 경우 local-only와 target-only 커밋 수를 표시한다.
- 핵심 설명은 기존 Preview와 Review 화면에서 재사용한다.
- 커밋 수나 merge-base 정보를 읽지 못해도 유용한 문구를 제공한다.
- 좁은 터미널에서도 문구를 유지하고 필요한 경우 상세 화면에서 확장한다.

## 재사용할 코드

- `internal/app/preview.go`: 기존 관계 분류
- `internal/app/graph_action_review.go`: 기존 Review 설명
- `internal/app/update_fetch.go`: pull 및 행동 사전 검사 문구
- `internal/app/execution_detail.go`: 작업 완료 설명

## 완료 기준

- 사용자가 fast-forward, already-contained, diverged 상태를 구분할 수 있다.
- merge와 rebase 설명에 현재 branch와 대상 branch가 포함된다.
- 커밋 수를 알 수 있으면 표시하고, 알 수 없으면 생략 사실을 명확히 한다.
- 동일한 관계는 화면이 달라도 동일한 핵심 문구를 사용한다.
- 분석이 실패했거나 오래된 경우 작업이 안전하다고 주장하지 않는다.
- 모든 관계와 데이터 누락 경로를 단위 테스트한다.

## 범위에 포함하지 않는 내용

- merge/rebase 가능성 판단 규칙 변경
- 전체 커밋 diff 검사. 기존 `1.1` Inspector가 담당한다.
- 자연어 생성 또는 외부 AI 호출
- 항상 표시되는 Unified Decision Card

## 의존성

- `1.10`의 상태 분류 결과
- `1.11`의 다음 행동 추천 규칙
- 기존 divergence 및 merge-base 명령 유지

## 검증

```sh
go test ./internal/app ./internal/git
scripts/check
```
