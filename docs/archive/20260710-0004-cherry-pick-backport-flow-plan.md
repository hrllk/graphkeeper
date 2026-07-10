# CEO 계획: cherry-pick 백포트 흐름
2026-07-10 /plan-ceo-review로 생성
브랜치: develop | 모드: SELECTIVE EXPANSION
저장소: git@github.com:hrllk/graphkeeper.git

## 비전

### 10배 관점
Graphkeeper는 단순히 cherry-pick을 실행하는 도구가 아니라, release line에 fix를 옮길 대상을 읽고, 여러 커밋을 정확한 순서대로 전파하고, 어디까지 반영됐는지 즉시 보이게 하는 도구가 되어야 한다.

### 이상적 상태
사용자는 반영할 branch에 먼저 올라간 뒤 Graph 섹션을 연다. 같은 graph 위에 떠 있는 overlay 팝업에서 source commit들을 체크박스로 고르고, 선택된 commit이 어떤 순서로 적용되는지 확인한 뒤 실행한다. 도구는 destination을 숨기지 않고, 중간 실패도 침묵시키지 않는다.

## 범위 결정

| # | 제안 | 공수 | 결정 | 이유 |
|---|------|------|------|------|
| 1 | cherry-pick destination은 현재 checkout 된 HEAD branch로 고정한다 | S | 수용 | git 의미와 맞고, 사용자가 "어디로 보낼지"를 다시 선택하지 않아도 된다 |
| 2 | Graph 섹션에서 동일 graph 기반 OverlayPopup을 연다 | M | 수용 | 선택 맥락을 유지해야 실수가 줄고, 교육용 설명도 자연스럽게 붙는다 |
| 3 | OverlayPopup에서 N개의 source commit을 체크박스로 다중 선택한다 | M | 수용 | cherry-pick의 본질이 여러 커밋의 순차 전파이므로 UI도 그 모델을 따라야 한다 |
| 4 | 실행 전에는 선택/해제/재정렬 가능한 FIFO queue로 관리하고, 실행 후에는 sequencer를 고정한다 | M | 수용 | 사용자가 담은 순서가 곧 적용 순서여야 backport 의도가 유지되고, 실행 중에는 흐름이 흔들리면 안 된다 |
| 5 | destination branch 선택기를 따로 두는 설계 | M | 보류 | 현재 요구와 충돌한다. destination은 현재 HEAD로 고정하는 편이 더 명확하다 |

## 수용한 범위

- destination은 현재 checkout 된 branch로 고정한다.
- Graph 섹션에서 같은 graph를 배경으로 overlay picker를 연다.
- overlay 안에서 source commit들을 체크박스로 복수 선택한다.
- 실행 전에는 source commit을 선택/해제/재정렬할 수 있다.
- 실행이 시작되면 FIFO queue는 고정되고, 선택 상태를 더 이상 바꾸지 않는다.
- 실행 중에는 선택 개수, destination branch, 마지막 적용 commit, 다음 적용 commit을 보이게 한다.
- 실패나 중단은 어디까지 적용됐는지를 남기고, abort 상태를 사용자에게 보여준다.
- 충돌 해소 UI는 제공하지 않고, 사용자는 외부 도구나 다른 환경에서 충돌을 해결한 뒤 처음부터 다시 시도한다.
- dirty worktree, detached HEAD, 빈 선택은 명시적으로 막는다.
- 기존 merge/rebase review 패턴은 재사용하되, cherry-pick 전용 문구와 상태를 추가한다.

## TODO로 미룰 것

- destination branch 선택기 UI
- backport 추천 엔진
- 부사수 교육용 시나리오 모드
- cherry-pick 이력 요약 패널

## 이미 존재하는 것

- merge/rebase preview와 review 상태 흐름이 이미 있다.
- Graph target picker와 overlay confirmation 패턴이 이미 있다.
- `git cherry-pick`은 여러 commit을 한 번에 받을 수 있고, `--no-commit`도 지원한다.

## 꿈의 상태와의 차이

현재는 "merge/rebase가 가능한가"를 잘 보여주는 도구다.
이 계획이 끝나면 "어떤 fix를 어떤 release line에 어떤 순서로 backport할지"를 읽고 실행하는 도구가 된다.
12개월 뒤의 이상형은 Graphkeeper가 release owner의 backport copilot처럼 동작하는 상태다.

## 제약

- conflict resolution UI는 하지 않는다.
- destination은 현재 checkout 된 branch로 고정한다.
- 체크된 commit의 순서는 실행 시작 전까지만 바꿀 수 있고, 실행 시작 후에는 고정된다.
- Graph는 좁은 터미널에서도 읽혀야 한다.
- overlay는 같은 graph를 유지한 채 열려야 한다.
- 실패나 중단은 침묵하지 말고 눈에 보이는 상태로 남겨야 한다.
- merge commit은 기본 cherry-pick 대상에서 제외하고, 일반 commit만 허용한다.

## 열린 질문

- reorder는 check / uncheck만으로 제공한다.
- merge commit은 기본 대상에서 막고, 일반 commit만 허용할 것이다.

## 성공 기준

- 사용자는 현재 branch에 무엇을 backport하는지 헷갈리지 않는다.
- 실행 전에는 check / uncheck로 순서를 바꿀 수 있고, 실행 시작 후에는 그 순서대로 그대로 실행된다.
- 중간 실패 시, 어디까지 적용됐는지 즉시 알 수 있다.
- 좁은 terminal에서도 overlay와 graph가 함께 읽힌다.
- 실패 후에는 abort로만 빠져나올 수 있다.

## 다음 단계

1. Graph overlay picker의 상태 모델을 확정한다.
2. selected commit FIFO 정렬 및 reorder 규칙을 명시한다.
3. cherry-pick 실행 중 partial success와 rescue 상태를 정의한다.
4. `git cherry-pick` 다중 선택과 실패 복구에 대한 테스트를 적는다.

## 내가 본 당신의 사고 방식

- 당신은 `cherry-pick`을 단일 명령이 아니라 "release line으로 fix를 옮기는 흐름"으로 봤다.
- 당신은 destination과 source를 분리해서 생각했다.
- 당신은 UI보다 먼저 git 의미론을 맞추려고 했다.
- 당신은 체크박스 UX를 원했지만, 실행 순서는 절대 흐트러지면 안 된다는 점도 같이 잡았다.
