# `internal/app/model.go` 리팩토링 작업 노트

## Goal

`internal/app/model.go`를 작게 유지하기 위한 현재 작업 기준을 정리한다.

## Current direction

`model.go` 관련 작업은 아래 아카이브 문서들을 기준으로 진행한다.

- `docs/archive/202606-0008-refactor-model-shrink-plan.md`
- `docs/archive/202606-0009-refactor-model-structure-plan.md`
- `docs/archive/202606-0010-refactor-model-messages-plan.md`
- `docs/archive/202606-0011-refactor-model-boundary-plan.md`

## What to keep in mind

- graph lazy load는 더 이상 전제로 두지 않는다.
- `model.go`는 앱 실행 상태 컨테이너로만 유지한다.
- message type, helper, state boundary는 분리한다.
- `internal/state/state.go`는 UI 상태 모델이고, `model.go`는 Bubble Tea 실행 컨테이너다.

## Next step

상세 구현은 0009에서 시작하고, 0010과 0011로 경계를 정리한다.
