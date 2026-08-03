# Git 작업 완료 결과 요약 표시

Subtask ID: `1.14`
상위 기능: `1` — 커밋 검사 및 유지보수자 판단 흐름
상태: 초안
작성일: 2026-08-02

## 목표

Git 작업 전후 상태를 비교해, 단순한 성공 또는 오류 Toast보다 실제 결과를
명확하게 보여준다.

## 사용자 결과

작업 후 사용자는 무엇이 바뀌었는지, 목표 상태에 도달했는지, 저장소가 여전히
막혀 있다면 무엇을 해야 하는지 알 수 있다.

## 범위

- 작업 시작 시 필요한 이전 상태를 저장한다.
- 기존 lifecycle 흐름을 사용해 작업 후 저장소 상태를 새로고침한다.
- 성공, 실패, 일부 새로고침, 오래된 결과를 구분한다.
- branch 정렬, 새 merge commit, 남은 divergence, dirty worktree,
  conflict 진행 상태 같은 의미 있는 변화를 요약한다.
- 기존 실행 결과와 새로고침 흐름에 결과 문구를 연결한다.
- 결과를 닫은 뒤 선택된 Graph 맥락을 유지한다.
- 새로고침 또는 실행 실패에 대해 재시도나 Graph 복귀 경로를 제공한다.

## 재사용할 코드

- `internal/app/commands.go`: 작업 실행 명령
- `internal/app/messages.go`: 실행 및 새로고침 메시지
- `internal/app/update_lifecycle.go`: 새로고침과 repository epoch 처리
- `internal/app/execution_detail.go`: 기존 작업 결과 문구
- `internal/app/view_overlays.go`: 기존 임시 결과 화면

## 완료 기준

- 성공한 작업이 단순 완료가 아니라 결과 저장소 상태를 보여준다.
- 실패한 작업이 오류를 보존하고 복구 가능한 상태로 돌아온다.
- 새로고침 실패와 Git 작업 실패를 구분한다.
- 오래된 비동기 결과가 최신 상태나 선택 맥락을 덮어쓰지 않는다.
- 일부 결과는 새로고침하지 못한 정보를 정확히 명시한다.
- 좁은 터미널과 `NO_COLOR`에서도 결과 문구가 읽힌다.

## 범위에 포함하지 않는 내용

- 영구 작업 기록
- 원격 분석 또는 telemetry 변경
- 파괴적 Git 작업의 자동 재시도
- 항상 표시되는 Unified Decision Card

## 의존성

- `1.10~1.13`의 상태, 설명, Preview 계약
- 기존 비동기 메시지와 repository epoch 계약

## 검증

```sh
go test ./internal/app ./internal/git
go test -race ./internal/app ./internal/git
scripts/check
```
