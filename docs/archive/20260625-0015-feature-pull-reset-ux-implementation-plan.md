# Pull / Reset UI·UX 구현 문서

## 목적

이 문서는 `pull`과 `reset` 기능의 UI·UX를 먼저 확정하고, 내일 바로 구현을 이어갈 수 있도록 현재 의사결정과 남은 작업을 정리한다.

우선순위는 다음과 같다.

1. `pull`의 노출 위치와 활성 조건을 단순화한다.
2. `reset`은 현재 브랜치 기준의 `hard reset`만 먼저 제공한다.
3. `reset` 실행 전에는 반드시 대상과 결과를 미리 보여준다.
4. `merge` / `rebase`는 같은 선택 UI를 재사용하되 이번 문서의 범위에서는 부차적으로 둔다.

## 현재 관찰된 구현 상태

현재 코드에서는 다음 흐름이 이미 존재한다.

- `f`는 전역 fetch로 동작한다.
- `p`는 pull 진입점으로 존재한다.
- `m`, `r`, `s`는 Graph 섹션에서 merge / rebase / reset preview 용도로 사용된다.
- `space`는 Graph 섹션에서 checkout을 수행하지 않도록 막혀 있다.
- `Mode` 패널은 섹션별 액션을 보여주는 역할로 이미 사용 중이다.

즉, 기능 자체는 일부 연결되어 있지만, 사용자 입장에서는 아직 “어디에서 무엇을 눌러야 하는지”가 충분히 명확하지 않다.

## 핵심 판단

### 1. pull

`pull`은 그래프 포인터 탐색과 분리한다.

이유:

- `pull`은 특정 커밋을 고르는 행위가 아니라 현재 브랜치 상태를 갱신하는 행위에 가깝다.
- Graph에서 대상을 탐색하게 만들면 사용자가 `fetch`, `merge`, `rebase`, `reset`과 혼동할 가능성이 높다.
- 현재 브랜치의 upstream 상태만 알면 되므로, `Current` 또는 `Local` 섹션에서 제공하는 편이 더 일관적이다.

권장 활성 조건:

- 현재 HEAD가 branch 상태일 것
- remote가 존재할 것
- upstream이 설정되어 있을 것
- 대상이 명확하지 않은 detached HEAD 상태는 비활성

권장 메시지:

- 활성 상태: `pull`
- 비활성 상태: `No upstream configured` 또는 `Detached HEAD`

### 2. reset

`reset`은 기본적으로 `hard reset`만 제공한다.

이유:

- soft / mixed / hard를 한 번에 넣으면 TUI에서 설명 비용이 커진다.
- 사용자는 `reset`을 실행했을 때 현재 브랜치 포인터와 작업 결과가 어떻게 바뀌는지 즉시 이해해야 한다.
- 안전성 측면에서 “무엇이 버려지는지”를 명확히 보여주는 편이 중요하다.

권장 표현:

- UI 라벨: `hard reset`
- 내부 액션: `ActionReset`
- 상태 설명: `reset will move current branch pointer to selected target`

## UI 배치 제안

### pull

권장 배치:

- `Current` 섹션의 액션으로 노출
- 필요하다면 `Local` 섹션에도 동일 동작을 노출할 수 있으나, 한 곳에서만 시작하는 편이 더 단순하다

권장 선택:

- `Current` 섹션만 1차 배치

### reset

권장 배치:

- `Graph` 섹션의 액션으로 노출
- 대상은 그래프 포인터를 통해 선택
- `Current` 섹션에서는 실행하지 않음

이유:

- reset은 “현재 브랜치가 어디로 되돌아갈지”를 그래프 상에서 보여줄 수 있어야 한다.
- Graph에서 바로 타깃을 찍고 preview를 보여주는 흐름이 가장 자연스럽다.

## reset UX 상세안

### 대상 범위

허용 대상:

- local branch
- commit

제외 대상:

- remote branch
- origin 계열 참조

이유:

- reset은 현재 브랜치 포인터를 직접 움직이는 동작이므로, 원격 ref를 대상으로 삼는 것은 UX상 의미가 약하고 오해를 부른다.

### 실행 전 preview

사용자가 reset을 실행하려고 하면 아래 정보를 보여준다.

- 현재 branch 이름
- 현재 HEAD commit
- 선택한 target commit
- target 선택 경로가 branch인지 commit인지
- reset 후 HEAD가 이동할 위치
- 사라질 가능성이 있는 커밋 범위
- 작업 트리 영향 경고

### 그래프 예시 표기

preview에서는 “현재 위치”와 “이동 후 위치”를 함께 보여줘야 한다.

예시 형식:

- `HEAD: main -> c1`
- `target: feature-x -> c0`
- `after reset: main -> c0`
- `commits between c1..c0 may be discarded`

실제 TUI에서는 아래 중 하나로 표현한다.

1. 현재 그래프를 유지한 채 `HEAD` 마커만 target으로 이동한 미리보기
2. 별도 preview 패널에서 `before / after`를 텍스트로 비교
3. reset 시 영향을 받는 commit 목록을 짧게 보여주는 축약표시

### 추천 방식

1번 + 2번 조합을 추천한다.

- 그래프 상에서는 포인터 이동을 시각화
- Mode 패널에서는 before / after를 텍스트로 설명

이 방식이 가장 직관적이고 구현 난이도도 과하지 않다.

## pull UX 상세안

### 동작 흐름

1. 사용자가 `pull`을 실행한다.
2. upstream / remote 상태를 확인한다.
3. 가능하면 fetch 후 pull 가능성을 판단한다.
4. fast-forward 가능하면 실행한다.
5. diverged 상태면 중단하고 원인을 보여준다.

### 권장 정책

- `pull`은 Graph 선택과 독립적으로 실행
- `pull`은 자동으로 target picker를 띄우지 않음
- target이 필요한 경우에는 현재 브랜치 upstream만 사용

### 실패 케이스

- detached HEAD
- upstream 없음
- remote 없음
- fetch 결과가 최신이 아님
- fast-forward 불가

## merge / rebase와의 관계

이번 문서의 범위는 pull/reset이지만, merge/rebase는 같은 선택 UI를 재사용하는 방향이 맞다.

권장 규칙:

- `merge` / `rebase`는 Graph 또는 Local에서 target을 고르게 한다
- 대상 활성화는 “같은 브랜치가 아닌 경우”로 제한한다
- 현재 브랜치와 동일 ref는 비활성 처리한다
- 충돌 해결 UI는 이번 단계에서 다루지 않는다

## 구현 순서

1. `pull`의 배치 위치를 `Current` 섹션으로 확정한다.
2. `pull`의 활성 조건과 비활성 메시지를 정리한다.
3. `reset`을 `hard reset`으로 명시한다.
4. `reset` 대상 허용 범위를 local branch / commit으로 제한한다.
5. `reset preview` 화면에 before / after 정보를 추가한다.
6. `Mode` 패널의 액션 도움말을 pull/reset 기준으로 갱신한다.
7. 관련 테스트를 추가한다.

## 테스트 항목

- `pull`이 detached HEAD에서 비활성인지
- `pull`이 upstream 없는 브랜치에서 비활성인지
- `pull`이 정상 상태에서 실행되는지
- `reset`이 Graph 섹션에서만 진입 가능한지
- `reset` 대상이 remote branch면 거부되는지
- `reset` preview가 현재 HEAD와 target을 함께 보여주는지
- `reset` 실행 전 confirm 단계가 존재하는지
- `reset`이 hard reset으로만 동작하는지

## 내일 이어서 할 일

- [ ] `pull`의 실제 진입 위치를 정한다
- [ ] `reset` preview 문구를 확정한다
- [ ] `reset`의 before / after 표시 방식 결정
- [ ] target 선택기에서 remote branch 제외
- [ ] 테스트 케이스 추가
- [ ] 구현 후 실제 repo에서 흐름 검증

## 결론

현재 단계에서는 `pull`과 `reset`을 “서로 다른 성격의 작업”으로 분리하는 것이 가장 안전하다.

- `pull`은 현재 브랜치 기준의 상태 갱신
- `reset`은 Graph에서 대상 선택 후 실행하는 hard reset

이렇게 고정하면 이후 `merge` / `rebase`도 같은 타깃 선택 구조 위에 얹을 수 있다.
