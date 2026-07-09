# Stash Continue From Branch Plan

## 목적

선택된 stash 를 기준으로 new branch 를 만들고, checkout 한 뒤, 그 상태에서 작업을 이어가는 흐름을 정의한다.

이 문서는 pop 실행 자체가 아니라, pop 전에 branch context 를 명시하는 연쇄 흐름을 다룬다.

## 사용자 목표

1. 사용자는 stash 를 바로 풀기 전에 branch context 를 만들 수 있어야 한다.
2. 새 branch 에 checkout 된 뒤 작업을 이어갈 수 있어야 한다.
3. 이 흐름은 list UI 와 pop 실행과 분리되어야 한다.

## 범위

### 포함

- new branch 생성
- checkout
- 그 다음 pop 으로 이어지는 컨텍스트 전달
- pop only 가 아닌 branch-first 권장 흐름

### 제외

- stash list UI
- 실제 pop 실행 로직
- Graph row 하이라이트
- stash 생성 / edit / drop

## UX 원칙

- branch 생성은 stash 의 복구 맥락을 명시하는 안전장치다.
- checkout 은 사용자가 이어갈 공간을 만든다.
- pop 은 마지막 단계로 남겨서, branch context 가 먼저 고정되도록 한다.

## 테스트

- branch 생성 후 checkout 순서가 유지되는지
- branch-first 경로가 기본값인지
- pop only 가 명시적 대안으로만 남는지

