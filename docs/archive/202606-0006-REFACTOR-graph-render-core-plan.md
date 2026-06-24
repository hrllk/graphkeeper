# `internal/app/graph_render.go` core 정리 계획서

## 목표

`graph_render.go`는 row 렌더링, connector 렌더링, compact 포맷을 담당하는 core로 줄인다.

현재 목표는 동작을 바꾸지 않고 책임을 더 선명하게 나누는 것이다.

## 범위

- 대상 파일
  - `internal/app/graph_render.go`
  - 필요 시 `internal/app/graph_render_connectors.go`
  - 필요 시 `internal/app/graph_render_format.go`
- 관련 테스트
  - `internal/app/model_test.go`
  - 필요 시 `internal/app/graph_render_test.go`
- 새 패키지 생성: 하지 않음

## 현재 문제

현재 파일에는 다음이 함께 있다.

- `renderGraphLine`
- `renderRawGraphLine`
- `graphLineCell`
- `highlightRawGraphPrefix`
- `renderGraphConnectorLines`
- `collapseConnectorLines`
- `parentShiftConnectorLines`
- `compactDecorationInfo`
- `compactWhenText`
- `compactTitleText`

이 구조에서는 row 렌더링 정책과 포맷 정책이 한 파일 안에서 섞여 보인다.

## 분리 방향

### `graph_render.go`에 남길 것

- `renderGraphLine`
- `renderRawGraphLine`

단, 이 둘은 orchestration wrapper 역할로 축소한다.

### `graph_render_connectors.go`

- `renderGraphConnectorLines`
- `collapseConnectorLines`
- `parentShiftConnectorLines`
- `renderGraphSpacer`

### `graph_render_format.go`

- `compactDecorationInfo`
- `formatCompactDecorations`
- `compactWhenText`
- `compactTitleText`
- `hasHeadDecoration`
- `padRight`

## 구현 순서

1. format helper를 먼저 분리한다.
2. connector helper를 분리한다.
3. `renderGraphLine` / `renderRawGraphLine`는 orchestration만 남기도록 정리한다.
4. `view_graph.go`가 새 구조를 호출하도록 맞춘다.
5. 렌더링 테스트를 graph 전용 파일로 옮긴다.

## 테스트 항목

1. 선택 row pointer 강조가 유지되는지
2. raw graph prefix가 유지되는지
3. virtual conflict row의 blank 처리와 색상 처리 유지되는지
4. connector collapse가 한 줄 혹은 progressive 형태로 유지되는지
5. `compactDecorationInfo`의 branch precedence가 유지되는지
6. `compactWhenText`와 `compactTitleText`의 축약 결과가 유지되는지

## 완료 기준

- row 렌더링과 포맷 로직이 분리된다.
- connector 로직이 별도 파일로 분리된다.
- 렌더링 결과가 기존과 같다.
- 테스트가 책임 경계와 맞는다.

## 비고

- 이 단계는 마지막에 수행하는 것이 안전하다.
- `view_graph.go`와 `navigation.go`의 경계가 먼저 정리되어야 renderer 정리가 덜 흔들린다.
