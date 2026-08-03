# Git 작업 영향 Preview 추가

Subtask ID: `1.13`
상위 기능: `1` — 커밋 검사 및 유지보수자 판단 흐름
상태: 초안
작성일: 2026-08-02

## 목표

사용자가 Git 작업을 확인하기 전에 예상 결과와 위험을 알 수 있게 한다.
특히 history를 다시 쓰거나 저장소 상태를 제거하는 작업을 명확히 설명한다.

## 사용자 결과

사용자는 확인 키를 누르기 전에 정확한 대상, 예상 history 변화, 충돌 또는
데이터 손실 가능성을 확인할 수 있다.

## 범위

- 기존 outcome, review, confirm 문구에 대상 ref와 commit을 포함한다.
- merge commit 생성, rebase replay, fast-forward 이동, push/pull 방향을
  설명한다.
- reset, clean, branch 삭제, tag 삭제, force push를 위험 또는 파괴 가능
  작업으로 표시한다.
- 사전 분석에서 확인 가능한 경우 충돌 가능성을 표시한다.
- 기존 Preview와 Review 화면에 영향 요약을 제공한다.
- 기존 Popup 계열과 키 계약을 재사용한다.
- 실행 전에 repository epoch과 선택된 대상을 다시 확인한다.

## 재사용할 코드

- `internal/app/preview.go`: 작업 분석
- `internal/app/graph_action_review.go`: merge/rebase Review Popup
- `internal/app/view_shell.go`: Overlay 구성
- `internal/app/commands.go`: 비동기 작업 실행
- `internal/app/update_lifecycle.go`: repository epoch 보호

## 완료 기준

- 모든 변경 작업 Preview에 정확한 작업과 대상이 표시된다.
- merge와 rebase Preview에 history 변화가 표시된다.
- reset, clean, delete, force push는 명시적인 위험 문구와 확인을 요구한다.
- 오래된 대상이 Preview를 통해 실행되지 않는다.
- Preview 실패 시 실행을 차단하고 복구 경로를 표시한다.
- 기존 `q`, `esc`, `y`, `n` 및 작업별 키 계약을 유지한다.

## 범위에 포함하지 않는 내용

- 완료된 Git 작업의 undo 또는 자동 rollback
- Graphkeeper 내부 conflict resolution
- 새로운 Popup framework
- 항상 표시되는 Unified Decision Card

## 의존성

- `1.12`의 branch 판단 맥락 문구
- 기존 epoch 및 stale-state 보호

## 검증

```sh
go test ./internal/app ./internal/git
scripts/check
```
