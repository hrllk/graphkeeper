# `internal/app` 리팩토링 마스터 계획서

## 목적

이 문서는 `view`, `navigation`, `graph_render` 관련 리팩토링의 마스터 인덱스다.

문서만 보고도 바로 구현이 가능하도록 하기 위해, 실제 작업은 아래 3개 하위 문서로 분리한다.

1. `view.go`와 `view_graph.go` 분리
2. `navigation.go`의 중복 규칙 helper 분리
3. `graph_render.go` core 정리

## 하위 문서

- [202606-0004-REFACTOR-view-graph-structure-plan.md](./202606-0004-REFACTOR-view-graph-structure-plan.md)
- [202606-0005-REFACTOR-navigation-graph-rules-plan.md](./202606-0005-REFACTOR-navigation-graph-rules-plan.md)
- [202606-0006-REFACTOR-graph-render-core-plan.md](./202606-0006-REFACTOR-graph-render-core-plan.md)

## 적용 순서

1. `view` 구조 분리
2. `navigation` 규칙 helper 분리
3. `graph_render` core 정리
4. 테스트 재배치
5. `scripts/check` 검증

## 완료 기준

- 각 단계 문서만 읽어도 해당 단계 구현이 가능하다.
- 단계별 책임 경계가 겹치지 않는다.
- 렌더링 결과와 navigation 판단 결과가 기존과 동일하다.

## 비고

- `model.go`는 상태 정의 파일로 유지하는 쪽이 낫다.
- `view.go` / `view_graph.go` 분리는 가장 우선순위가 높다.
- 렌더링과 navigation이 같은 문자열 규칙을 중복 구현하지 않도록 한다.
