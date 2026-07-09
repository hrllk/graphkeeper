# Stash Pop Execution Plan

## 목적

overlay stash list popup 에서 선택된 항목을 실제로 `stash pop` 하는 흐름을 정의한다.

이 문서는 list UI 와 분리되어 있으며, pop 실행과 실패 처리를 담당한다.
진입점은 `0003`의 전역 hotkey + overlay popup 이고, 이 문서는 그 안에서의 실행만 다룬다.

## 사용자 목표

1. 사용자는 선택한 stash 를 실제로 pop 할 수 있어야 한다.
2. pop 결과는 명확하게 성공 / 실패로 드러나야 한다.
3. pop 이후 stash list 와 detail 상태는 다시 동기화되어야 한다.
4. direct pop 은 가능하되, 위험한 경우에는 branch-first 를 강하게 안내해야 한다.

## 범위

### 포함

- 선택된 stash 에 대한 pop 실행
- pop 완료 후 refresh
- 충돌 / 실패 / stale state 처리
- 성공 / 실패 메시지
- `HEAD` 가 stash 의 `BaseHash` 보다 자식일 때 branch-first 안내

### 제외

- stash list UI 자체
- 전역 hotkey / overlay popup 렌더링
- branch create / checkout 흐름
- Graph 하이라이트
- stash 생성 / edit / drop

## 실행 원칙

- pop 은 선택된 stash 항목에 대해서만 일어난다.
- pop 이 실패하면 사용자는 왜 실패했는지 알아야 한다.
- pop 이 끝나면 stash source of truth 를 다시 읽는다.
- `HEAD` 가 `BaseHash` 의 descendant 라면, direct pop 은 허용하되 branch-first 가 더 안전한 경로라는 안내를 먼저 보여준다.
- direct pop 은 명시적 대안으로 남기고, 기본 권장 경로는 branch-first 다.

## 실패 케이스

- stash 가 이미 사라진 경우
- working tree conflict 가 난 경우
- 대상 stash 가 stale 하게 된 경우
- refresh 중 git 상태가 바뀐 경우

## 테스트

- 성공 시 refresh 되는지
- stale stash 가 안전하게 실패하는지
- conflict 메시지가 유지되는지
- pop 후 list 가 최신 상태로 돌아오는지
