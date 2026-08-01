# 기여자 workflow 추적 가이드

Graphkeeper의 Git 작업은 다음 흐름으로 추적한다. 구현자는 `commands.go`에 새 Git 작업을 바로 추가하지 않고, 먼저 contract와 테스트를 정한다.

```text
key: m
  → intent: ActionMerge(target)
  → workflow: Snapshot → ValidateTarget → Execute → Reload
  → result: operation result { action, target, epoch, phase, error, recovery hint }
  → message: action completed
  → state: state.Status
  → projection: ScreenProjection
  → view: Graph / section / popup renderer
  → tests: workflow contract + update transition + projection output
```

## 변경 위치

- 화면 입력: `internal/app/key_handling_*.go`
- 순수 정책: `internal/app/actions.go`, `target_items.go`, `preview.go`
- 저장소 경계: `internal/app/repository_contract.go`
- Git 구현: `internal/git/repo.go`, `repo_exec.go`, parser files
- 상태 전이: `internal/app/update_*.go`, `internal/state/state.go`
- 화면 입력 projection: `internal/app/view_projection.go`
- 화면 출력: `internal/app/view_*.go`, `graph_render*.go`

## 테스트 실행

빠른 경계 테스트:

```sh
go test ./internal/app -run 'Test(Workflow|Projection|Epoch)'
go test ./internal/git
```

전체 검증:

```sh
scripts/check
scripts/build /tmp/graphkeeper-architecture
```

workflow 테스트는 `fakeRepository`로 Git subprocess 없이 실행한다. 실제 parser와 runner 동작은 `internal/git`의 temporary Git fixture에서 검증한다.

## 새 workflow 추가 규칙

1. `targetRef`와 필요한 snapshot field를 먼저 정의한다.
2. `ValidateTarget`가 `Execute`보다 먼저 호출되는 테스트를 추가한다.
3. 실행 후 `Reload` 실패를 partial result로 보존한다.
4. 작업 시작 epoch과 결과 epoch이 다르면 stale 결과를 적용하지 않는다.
5. projection에 사용자에게 필요한 상태·원인·복구 방법을 추가한다.
6. migration ledger에 compatibility shim과 제거 조건을 기록한다.
