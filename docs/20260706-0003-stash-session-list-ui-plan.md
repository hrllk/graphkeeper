# Stash Session List UI Plan

## 목적

세션 단위로 stash list 를 한눈에 볼 수 있는 전역 진입점과 overlay UI 를 정의한다.

이 문서는 `Graph` 의 row 하이라이트와 분리된, stash 를 찾고 읽는 중심 화면을 다룬다.
전역 hotkey로 목록을 열고, 같은 shell 안의 overlay popup에서 읽고 선택하는 흐름을 기준으로 한다.

## 사용자 목표

1. 사용자는 현재 세션에서 접근 가능한 stash 들을 전역 hotkey로 즉시 열 수 있어야 한다.
2. stash 는 commit 중심으로 묶여 보여야 한다.
3. 사용자는 overlay popup 안에서 특정 stash 를 선택해서 다음 행동으로 이어갈 수 있어야 한다.
4. 이 UI 는 `Graph` 와 독립적으로 동작하되, 같은 shell/section 상태 위에서 뜬다.

## 범위

### 포함

- 세션 stash list view
- 전역 hotkey 진입점
- overlay popup 렌더링
- commit 기준 grouping
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

- list 는 `BaseHash` 기준으로 묶는다.
- 같은 base commit 을 가진 stash 들은 하나의 그룹으로 보인다.
- 그룹 안에서는 최신 stash 를 먼저 보여준다.
- 각 항목은 `Ref`, `Subject`, `BaseHash` 를 함께 보여준다.

## UX 원칙

- session list 는 전역 hotkey로 열리는 focused inspector 로 유지한다.
- 사용자가 commit 단위로 읽을 수 있게 그룹 헤더를 강하게 보여준다.
- Graph 의 시각 신호와는 별도로, overlay list 는 텍스트 정보와 선택 상태를 담당한다.

## 진입 방식

- Global 에 stash list 를 여는 hotkey 를 둔다.
- hotkey 는 현재 section 상태를 유지한 채 overlay popup 을 띄운다.
- popup 에서는 list 읽기, selection 이동, 후속 액션 진입까지만 다룬다.
- 실행 동작인 `pop` 은 별도 문서에서 다룬다.

## 테스트

- base commit 으로 grouping 되는지
- 최신 stash 가 먼저 나오는지
- base hash 가 없는 stash 는 제외되는지
- selection 이동이 안정적인지
