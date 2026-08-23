# 아키텍처 경계 개선 마이그레이션 원장

이 문서는 `~/.gstack/graphkeeper/20260731-0003-modern-architecture-boundary-improvement-plan.md`의 구현 추적표다. 각 단계는 기존 동작을 유지하면서 한 개의 경계를 옮긴다.

| 단계 | 현재 위치 | 목표 경계 | 책임 소유자 | 호환 장치 | 제거 조건 | 상태 |
| --- | --- | --- | --- | --- | --- | --- |
| T1 | `model`이 런타임과 저장소 파생 상태를 함께 보유 | 기준선과 책임표 고정 | `internal/app` | 기존 파일 구조 유지 | T2 계약 테스트 통과 | 완료 |
| T2 | 주기 refresh와 작업 결과가 순서 없이 적용될 수 있음 | snapshot epoch으로 오래된 읽기 폐기 | `internal/app` lifecycle | `loadedMsg`/`refreshedMsg.epochSet` 호환 shim | 모든 load/refresh 경로가 epoch을 전달 | 완료 |
| T3 | 파괴적 작업이 호출 시점의 target에 의존 | 실행 직전 target 재검증과 작업 결과 계약 | `internal/app` action/workflow | 기존 `execute*` 함수 유지 | branch/tag/remote/commit 재검증 테스트 통과 | 완료 |
| T4 | 렌더러가 `model` 세부 필드에 직접 접근 | app read model → renderer projection | `internal/app` view | 기존 렌더 함수 유지 | projection이 모든 저장소 필드를 대체 | 1차 완료: Graph/Section/Context, popup 후속 |
| T5 | Git snapshot의 선택 필드 오류 의미가 불명확 | required/optional validity 명시 | `internal/app` adapter | 기존 `git.Status` 유지 | validity 테스트와 오류 표시 통과 | 1차 완료: 최소 adapter/fake, OperationResult 후속 |
| T6 | 큰 통합 테스트와 구조 문서가 현재 경계를 반영하지 못함 | slice별 fixture와 문서 동기화 | 각 패키지 | 기존 테스트 유지 | `scripts/eval` 통과 | 1차 완료: projection test/contributor 문서, test split 후속 |

## T1 기준선

- 변경 전 기준선: `scripts/test` 통과.
- public CLI, 단축키, Git mutation 순서는 이번 단계에서 변경하지 않는다.
- 새 경계는 먼저 메시지 계약과 테스트로 고정한 다음 구현한다.
- 구현 중 실패한 테스트는 기능 회귀인지, 의도한 계약 변경인지 분류하고 원장에 기록한다.

## T2 적용 규칙

`repositoryEpoch`는 사용자 입력으로 저장소 작업을 시작할 때 증가한다. 주기 refresh는 생성 시 epoch을 캡처한다. 응답 시 현재 epoch과 다르면 저장소/UI 상태에 적용하지 않는다. epoch이 없는 기존 테스트 메시지는 하위 호환을 위해 검증 대상에서 제외한다. 이 호환 장치는 모든 refresh 생성 경로가 epoch을 전달하게 된 뒤 제거한다.

```text
사용자 작업 시작
    └─ repositoryEpoch++
         ├─ 이전 refresh(epoch=old) 완료 → 폐기
         └─ 다음 refresh(epoch=current) 완료 → 적용
```

파괴적 작업의 target 재검증과 snapshot validity는 T3/T5에서 별도 계약으로 추가한다. 따라서 T2의 epoch은 안전성의 전체 보장이 아니라, 비동기 순서 역전으로 인한 stale overwrite를 막는 최소 경계다.
