# `internal/app/model.go` 리팩토링 작업 노트

## Goal

`internal/app/model.go`를 작게 유지하기 위한 현재 작업 기준을 정리한다.
이 문서는 `pull / reset / stash` UX 문서와 실제 코드 경계를 함께 맞추는
기준 문서다.

## Current direction

`model.go` 관련 작업은 아래 아카이브 문서들을 기준으로 진행한다.

- `docs/archive/20260625-0015-feature-pull-reset-ux-implementation-plan.md`
- `docs/archive/20260625-0016-feature-reset-stash-plan.md`
- `docs/archive/202606-0008-refactor-model-shrink-plan.md`
- `docs/archive/202606-0009-refactor-model-structure-plan.md`
- `docs/archive/202606-0010-refactor-model-messages-plan.md`
- `docs/archive/202606-0011-refactor-model-boundary-plan.md`

## What to keep in mind

- graph lazy load는 과거 전제다. 더 이상 설계 기준으로 사용하지 않는다.
- `model.go`는 Bubble Tea 실행 컨테이너로 유지한다.
- `internal/state/state.go`는 사용자에게 보여줄 UI 상태 모델이다.
- `message type`, `helper`, `state boundary`는 서로 다른 파일 책임으로 분리한다.
- `pull` / `reset` / `stash` 흐름은 `browse`, `confirm`, `preview`, `execute` 경계가 섞이지 않게 정리한다.

## Next step

상세 구현은 다음 순서로 진행한다.

1. `pull` 진입 조건과 비활성 메시지를 현재 UX 문서와 맞춘다.
2. `reset` preview를 hard reset 기준으로 유지할지, mode 선택으로 확장할지 확정한다.
3. `stash`와 worktree 상태가 들어갈 경우 `state.Status`에 어떤 최소 상태만 둘지 정리한다.
4. `model.go`에서 메시지 타입과 preview 헬퍼를 분리한다.
5. `go test ./...`로 관련 행동이 유지되는지 확인한다.
