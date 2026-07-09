# Stash Session List UI Plan

## 목적

세션 단위로 stash list 를 한눈에 볼 수 있는 전역 진입점과 overlay UI 를 정의한다.

이 문서는 `Graph` 의 row 하이라이트와 분리된, stash 를 찾고 읽는 중심 화면을 다룬다.
전역 hotkey로 목록을 열고, 같은 shell 안의 overlay popup에서 읽고 선택하는 흐름을 기준으로 한다.

## 사용자 목표

1. 사용자는 현재 세션에서 접근 가능한 stash 들을 전역 hotkey로 즉시 열 수 있어야 한다.
2. 사용자는 overlay popup 안에서 특정 stash 를 선택해서 다음 행동으로 이어갈 수 있어야 한다.
3. 이 UI 는 `Graph` 와 독립적으로 동작하되, 같은 shell/section 상태 위에서 뜬다.

## 범위

### 포함

- 세션 stash list view
- 전역 hotkey 진입점
- overlay popup 렌더링
- stash summary / subject / ref
- selection / focus 상태
- stash 선택 이후의 후속 액션 진입점

### 제외

- Graph row 하이라이트
- 실제 stash pop 실행
- new branch / checkout / pop 연쇄
- stash 생성 / edit / drop
- Graph 내부에서의 stash 진입 UI 추가

## 핵심 모델

- list 는 stash entry 단위로 평탄하게 보여준다.
- 항목은 최신 stash 부터 먼저 보여준다.
- 각 항목은 7자리 hash, `Ref`, `Subject` 를 함께 보여준다.
- `BaseHash` 는 내부 점프용 메타데이터로만 유지하고 화면에는 노출하지 않는다.

## UX 원칙

- session list 는 전역 hotkey로 열리는 focused inspector 로 유지한다.
- overlay list 는 한 줄씩 빠르게 스캔되는 flat list 로 유지한다.
- hash 는 amber accent 로, ref 는 보조 텍스트로, subject 는 본문 텍스트로 읽히게 한다.
- Graph 의 시각 신호와는 별도로, overlay list 는 텍스트 정보와 선택 상태를 담당한다.

## 진입 방식

- Global 에 stash list 를 여는 hotkey 를 둔다.
- hotkey 는 현재 section 상태를 유지한 채 overlay popup 을 띄운다.
- popup 에서는 list 읽기, selection 이동, 후속 액션 진입까지만 다룬다.
- 실행 동작인 `pop` 은 별도 문서에서 다룬다.

## 테스트

- 최신 stash 가 먼저 나오는지
- hash 7자리로 보이는지
- base prefix 가 화면에 남지 않는지
- selection 이동이 안정적인지
