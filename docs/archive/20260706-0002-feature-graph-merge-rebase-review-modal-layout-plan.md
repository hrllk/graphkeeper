# Graph Merge/Rebase Review Modal Layout Plan

## 목적

Graph 섹션에서 `merge` / `rebase` 를 누를 때, 분기된 대상에 한해 뜨는 중간 모달의 레이아웃을 개선한다.

현재 모달은 분기 정보, 그래프 요약, 다음 행동 안내가 한 박스 안에서 같은 비중으로 보인다. 그래서 사용자는 지금 이 창이 "검토 화면"인지 "실행 직전 confirm"인지 한 번에 읽기 어렵다.

핵심 목표는 다음과 같다.

1. 분기점 검토와 최종 실행 확인을 시각적으로 분리한다.
2. 모달에서 가장 먼저 보여야 하는 것은 목적지, base commit, currentOnly, targetOnly 이다.
3. 그래프는 본문을 장식하는 부속 정보가 아니라, 분기 판단을 돕는 시각적 증거로 보이게 한다.
4. fast-forward / ancestor 케이스는 지금처럼 짧게 막고, review 모달을 억지로 열지 않는다.

## 현재 디자인의 문제

현재 구현은 `ModeReview` 에서도 `ModeConfirm` 과 같은 popup 골격을 재사용한다.

- 제목이 크게 보이지 않는다.
- 본문 상단의 `target`, `base`, `currentOnly`, `targetOnly` 가 하나의 덩어리처럼 묶여 있어 우선순위가 약하다.
- 그래프 excerpt 가 본문 아래에 붙는 방식이라, 사용자는 먼저 읽어야 할 숫자와 시각 정보를 같이 분리해서 읽어야 한다.
- review 모달과 final confirm 모달의 경계가 약해서, "검토"와 "실행"이 같은 단계처럼 느껴진다.

즉, 기능은 맞지만 정보 구조가 약하다. 이건 confirm layout 이 아쉬운 이유다.

## What already exists

이 계획은 기존 흐름을 최대한 재사용한다.

- `internal/app/view_shell.go` 의 `renderConfirmPopup(...)`
- `internal/app/graph_action_review.go` 의 `buildGraphActionReviewDetail(...)`
- `internal/app/update_fetch.go` 의 `graphActionCheckMsg` 처리
- `internal/app/key_handling_review.go` 의 review 키 흐름

기존에 이미 좋은 부분도 있다.

- review 상태와 final confirm 상태를 분리할 수 있는 `ModeReview` 가 있다.
- merge-base 와 divergence 계산은 이미 Git 레이어에 들어가 있다.
- 분기점이 깊지 않을 때는 실제 graph row excerpt 를 보여줄 수 있다.

## 범위

### 포함

- review 모달의 내부 레이아웃 재구성
- final confirm 모달과 review 모달의 시각적 차이 강화
- graph excerpt 의 위치와 밀도 조정
- title / footer / body hierarchy 정리
- review 모달의 responsive width / alignment 규칙 정의

### 제외

- merge / rebase 실행 로직 변경
- fast-forward / ancestor block 규칙 변경
- Git 연산 추가
- Graph 본문 렌더링 전체 재설계

## 제안 UX

### 1. review 모달은 "분기점 확인" 화면으로 보이게 한다

review 모달은 단순 confirm처럼 보이면 안 된다. 사용자가 여기서 해야 하는 일은 yes/no 결정이 아니라, "지금 대상이 맞는가"를 확인하는 것이다.

권장 구조:

```text
╭───── Check Points for Merge or Rebase ─────╮
│ target    feature-branch  commit subject │
│ base      abc1234         merge base    │
│ currentOnly  12 commits                  │
│ targetOnly   5 commits                   │
│                                          │
│ * ... graph excerpt ...                  │
│ * ...                                    │
│                                          │
│ y: continue  •  n: cancel               │
╰──────────────────────────────────────────╯
```

레이아웃 메모:

- title 은 `Check Points for Merge or Rebase` 로 고정한다. 이 화면은 분기 상태를 읽는 검토 모달이지, 일반적인 상태 표시가 아니다.
- summary 행은 `current / target / base` 순서를 고정한다.
- `current` 행은 현재 브랜치의 HEAD 를 보여준다. `current` 는 현재 위치를, `target` 은 목적지를, `base` 는 분기점을 각각 한 번에 읽게 하는 역할을 분리한다.
- `target` 행에는 merge/rebase 맥락이 필요하면 보조 라벨로만 붙인다. 제목 자체를 바꾸기보다, 행동 맥락은 작고 명시적으로 넣는다.
- 각 행은 `{hash} ({branch mark}) {title}` 형태로 동일한 문법을 유지한다. 이렇게 해야 숫자와 그래프를 보기 전에 위치 관계가 먼저 읽힌다.

핵심은 숫자와 그래프가 먼저 읽히는 것이다. 설명 문구는 마지막에 둔다.

### Highlighting rules

review 모달의 highlighting 은 "누가 지금 중요한가"를 먼저 알려줘야 한다.

- `current` 는 가장 강하게 강조한다. 사용자는 지금 여기서 출발하므로, 현재 HEAD 는 가장 먼저 읽혀야 한다.
- `target` 은 다음 강도로 강조한다. 사용자가 고른 목적지라서 눈에 띄어야 하지만, `current` 보다 앞서면 안 된다.
- `base` 는 분기점으로서 기능만 분명하면 된다. 시각적으로는 다른 두 항목보다 한 단계 낮게 두고, 대신 `merge base` 라벨을 명확히 붙인다.
- `currentOnly` / `targetOnly` 숫자는 같은 시선에서 비교되도록 정렬하고, 숫자 자체는 bold 처리하되 주변 설명은 덜 튀게 둔다.
- graph excerpt 에서는 `current` tip, `target` tip, `base` 지점을 각각 다른 marker 로 표시한다. 색만 달라지는 방식은 피하고, `current` 는 가장 강한 marker, `base` 는 가장 약한 marker 로 둔다.
- selected graph row 는 reverse-video 보다 경계선 / bold / marker 우선으로 표현한다. 너무 강한 반전은 모달 전체를 무겁게 만들기 쉽다.
- 강조는 단일 색상 의존이 아니라 굵기, 간격, 라벨, marker 를 함께 써서 만든다. 사용자가 색각 차이가 있어도 위치 관계를 읽을 수 있어야 한다.

### 2. final confirm 은 더 작고 더 단순해야 한다

review 를 통과한 뒤의 final confirm 은 길면 안 된다.

권장 구조:

```text
╭──── Merge into current branch? ────╮
│ Merge feature-branch into main.    │
│ A merge commit will be created.    │
│                                    │
│ y: yes  •  n: no                   │
╰────────────────────────────────────╯
```

이 단계에서는 분기점 정보를 반복하지 않는다. 이미 review 에서 봤기 때문이다.

### 3. graph excerpt 는 충분히 깊지 않으면 실제 graph, 깊으면 축약형

그래프는 "예쁜 장식"이 아니라 분기 판단의 증거다.

- shallow case: 실제 graph row excerpt 를 그대로 보여준다.
- deep case: ladder형 축약 그래프를 사용한다.
- 너무 길면 전체를 다 보여주지 말고, merge-base 근처와 current / target tip 만 강조한다.
- 축약 상태가 들어가면 사용자가 그 사실을 알 수 있어야 한다. 단순히 행을 잘라내지 말고 `condensed` / `focus range` 같은 짧은 라벨을 붙여서, 지금 보고 있는 것이 전체 그래프가 아니라는 점을 숨기지 않는다.
- `current`, `target`, `base` 표시는 남기고 subject 만 먼저 줄인다. 역할 라벨이 사라지면 사용자는 어디가 기준인지 다시 해석해야 한다.
- 아주 좁은 폭에서는 그래프의 세부 선보다 `currentOnly` / `targetOnly` 와 base 라벨이 먼저 읽혀야 한다. 그래프가 먼저 튀어나와서 숫자를 밀어내면 검토 화면의 목적이 흐려진다.

### 4. fast-forward / ancestor 는 짧게 끝낸다

이 케이스들은 review 모달로 끌고 오지 않는다.

- fast-forward 가능: 짧은 안내 + 바로 막기
- target already included: 짧은 안내 + 바로 막기
- diverged case only: review 모달 진입

이 규칙을 지켜야 review 모달이 무의미하게 반복 노출되지 않는다.

### 5. 좁은 터미널에서는 요약 우선, 그래프는 후순위다

폭이 충분할 때만 지금의 3단 구성(요약 / 그래프 / footer)을 유지한다.

- 넓은 화면: current / target / base 3행을 먼저 읽고, 그 다음 graph excerpt 를 보여준다.
- 중간 폭: 요약은 유지하되 subject 길이를 줄이고, 그래프는 2~3행짜리 축약형으로 제한한다.
- 좁은 폭: 그래프보다 요약을 우선한다. 이때는 `condensed graph` 라벨을 보여주고, 실제 그래프 선은 최소한만 남긴다.
- 1줄씩 무리해서 쪼개는 대신, 덜 중요한 subject 를 먼저 줄인다. hash, role, count 는 끝까지 보존한다.

## 정보 구조

검토 화면에서 읽는 순서는 다음이어야 한다.

1. 현재 무엇을 하려는지
2. 어디가 목적지인지
3. 분기점이 어디인지
4. 어느 쪽이 currentOnly / targetOnly 인지
5. 그 다음에 그래프 증거
6. 마지막에 continue / cancel

ASCII 흐름:

```text
graph action
  -> divergence analysis
    -> diverged only
      -> review modal
        -> final confirm
          -> execute
```

## 구현 원칙

### 1. review 와 confirm 은 서로 다른 시각 언어를 써야 한다

같은 popup 스타일을 쓰더라도 인상은 달라야 한다.

- review: 왼쪽 정렬, 넓은 본문, 숫자와 그래프 우선
- confirm: 가운데 정렬, 좁은 본문, 실행 문구 우선

### 2. 숫자는 표처럼, 그래프는 증거처럼

`target`, `base`, `currentOnly`, `targetOnly` 는 설명 문장으로 섞지 말고, 라벨-값 형태로 분리한다.

### 3. footer 는 행동의 단계와 맞아야 한다

review 에서는 `y: continue • n: cancel`
final confirm 에서는 `y: yes • n: no`

지금처럼 공통 confirm footer 를 그대로 재사용하면 단계 구분이 흐려진다.

### 4. 모달 높이는 내용에 따라 달라져야 한다

review 모달은 그래프가 들어가면 높이가 늘어날 수 있다.
그 대신 width 는 충분히 넓게 잡고, padding 과 title strip 을 활용해서 답답함을 줄인다.

### 5. 색만 믿지 않는다

highlighting 은 색으로만 구분하면 안 된다.

- current / target / base 는 서로 다른 marker 문법을 써서 구분한다.
- selected row 는 reverse-video 보다 굵기, 테두리, 앞쪽 marker 를 우선한다.
- 중요한 행은 bold + spacing + label 조합으로 강조하고, 색은 보조 신호로만 쓴다.
- `base` 라인은 특히 약해지기 쉬우므로 `merge base` 라벨을 끝까지 유지한다.

## 상태별 표현

review 모달의 상태는 다음처럼 명시한다. 구현자가 별도의 추측을 하지 않도록, 폭과 데이터 가용성에 따라 화면이 어떻게 바뀌는지 적는다.

| 상태 | 사용자가 보는 것 | 설계 의도 |
|------|------------------|-----------|
| normal | current / target / base 요약 + graph excerpt + continue/cancel | 전체 판단을 가장 균형 있게 보여준다 |
| deep graph | condensed label + base 근처만 남긴 축약 그래프 | 너무 깊은 분기는 전체를 다 보여주지 않는다 |
| narrow width | 요약 우선, graph excerpt 축소, subject 더 강하게 절단 | 작은 터미널에서 정보 밀도를 유지한다 |
| graph unavailable | 숫자 요약 + fallback ladder diagram | 그래프가 없어도 검토는 가능해야 한다 |
| final confirm | 짧은 실행 문구 + y/n footer | review 와 실행을 다시 섞지 않는다 |

## 접근성 / 판독성

- `current`, `target`, `base` 는 색이 아니라 글자와 위치만으로도 구분돼야 한다.
- 커밋 subject 는 먼저 잘리고, hash 와 role label 은 나중에 잘린다.
- footer 의 `continue` / `cancel` 문구는 단순한 단축키 안내가 아니라 상태 전환의 의미를 설명해야 한다.
- 긴 subject 가 들어와도 줄바꿈이 summary 행을 분리하지 않도록, summary 는 가능한 한 한 줄 문법을 유지한다.

## NOT in scope

- merge/rebase 실행 명령의 semantics 변경
- Git divergence 계산 방식 변경
- branch picker UI 추가
- 별도 side panel 추가
- pull / push / reset confirm 의 재설계

## 리스크

1. review 와 confirm 이 여전히 같은 스타일이면, 사용자 입장에서는 두 단계가 다시 섞여 보인다.
2. 그래프를 너무 많이 보여주면 본문이 다시 뭉개진다.
3. final confirm 까지 review 정보를 반복하면 중복이 늘어난다.

## 구현 계획

1. `ModeReview` 전용 popup body 를 분리한다.
2. review 모달은 표 형태의 요약 영역과 graph excerpt 영역을 나눠 렌더한다.
3. final confirm 은 현재보다 더 짧고 좁게 유지한다.
4. review / confirm 의 footer 문구를 분리한다.
5. shallow / deep graph case 별 레이아웃을 테스트한다.
6. review 모달의 width / alignment 규칙을 확정한다.

## 검증

- diverged merge 대상에서 review 모달이 먼저 뜨는지
- review 모달에서 target / base / currentOnly / targetOnly 가 먼저 읽히는지
- deep graph 에서 축약형 excerpt 가 읽히는지
- 120 / 80 / 60 폭에서 summary 우선순위가 유지되는지
- graph unavailable 상태에서도 검토를 이어갈 수 있는지
- final confirm 이 review 보다 더 compact 한지
- fast-forward / ancestor 케이스가 review 를 거치지 않는지

## 결정 메모

- review 와 final confirm 을 분리한다.
- review 모달에서는 그래프를 증거로 다룬다.
- fast-forward / ancestor 케이스는 지금처럼 짧게 막는다.

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | not run | 0 |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | not run | 0 |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 1 | clear | 0 |
| Design Review | `/plan-design-review` | UI/UX gaps | 1 | clear | 0 |
| DX Review | `/plan-devex-review` | Developer experience gaps | 0 | not run | 0 |

- **VERDICT:** ENG + DESIGN CLEARED - ready to implement.
NO UNRESOLVED DECISIONS
