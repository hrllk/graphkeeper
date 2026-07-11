# 오버레이 토스트/컨펌 레이아웃 일관화

## 문제
현재 제품 안의 overlay popup 들은 공통 렌더링 인프라를 공유하지만, 실제 보이는 형태는 제각각이다.

- 어떤 popup 은 제목이 가운데 정렬되어 있고
- 어떤 popup 은 본문이 왼쪽으로 붙어 있고
- 어떤 popup 은 footer shortcut 이 눈에 잘 안 들어오고
- 어떤 popup 은 내용 밀도가 달라서 같은 제품 안에서 다른 앱처럼 느껴진다

이 작업의 목표는 기능을 바꾸는 것이 아니다. 사용자가 어떤 overlay 를 열어도 같은 규칙으로 읽히게 만드는 것이다.

## 목표
1. title header 는 항상 가운데 정렬한다.
2. footer shortcut 은 항상 가운데 정렬한다.
3. content section 은 popup 중앙축을 기준으로 배치한다. 이 말은 모든 텍스트를 무조건 가운데 정렬하라는 뜻이 아니라, 본문 덩어리의 기준축과 여백이 한쪽으로 쏠리지 않게 하라는 뜻이다.
4. toast, confirm, inspect 계열 overlay 가 같은 프레임 언어를 쓰게 만든다.

여기서 중요한 것은 "모든 popup 을 똑같이 보이게" 만드는 것이 아니다.
의미는 유지하되, 프레임과 정렬 규칙만 통일한다.

## 범위
다음 overlay 들을 우선 대상으로 본다.

- confirm popup
- reset mode popup
- loading toast
- alert popup
- review popup
- target pick popup
- stash message popup
- graph stash pop popup
- branch input popup
- graph search popup
- hidden hotkeys popup
- stash list popup
- tag popup
- cherry-pick popup

즉, `internal/app/view_shell.go` 에서 `overlayPopup(...)` 으로 얹는 모든 surface 와, 비슷한 title strip 프레임을 쓰는 popup 들을 같은 규칙으로 맞춘다.

## 레이아웃 원칙
### 1. Title header 중앙 정렬
- popup 상단 title 은 현재 `renderTitleStrip(...)` 계열 형식을 유지한다.
- 제목이 길어지면 잘리더라도 가운데 축은 유지한다.
- 제목은 좌우 균형이 맞아야 한다.

### 2. Footer shortcut 중앙 정렬
- `enter`, `esc`, `y/n`, `m/r` 같은 shortcut 안내는 footer 한 줄에서 중앙 정렬한다.
- footer 가 길어져도 좌우가 한쪽으로 치우치지 않게 한다.
- confirm 이든 toast 이든 shortcut 노출 방식은 같은 문법을 쓴다.

### 3. Content section 중앙 배치
- 본문은 popup 안에서 중앙 축을 기준으로 배치한다.
- 리스트형 content 는 항목 간 간격이 일정해야 한다.
- 설명형 content 는 한쪽으로 밀리지 않게 하고, 필요한 경우 줄바꿈과 폭 clamp 로 정리한다.

### 4. 프레임은 하나, 밀도는 두세 단계
- toast 는 짧고 얕게
- confirm 은 action 을 분명하게
- inspect 는 조금 더 넓고 읽기 쉽게

하지만 이 차이는 "정렬 규칙"을 흔들지 않는 선에서만 허용한다.

## 현재 코드에서 보이는 차이
다음 파일들이 각각 조금씩 다른 규칙을 가지고 있다.

- `internal/app/view_shell.go`
- `internal/app/view_layout.go`
- `internal/app/view_alert.go`
- `internal/app/stash_popup.go`
- `internal/app/hidden_hotkeys.go`
- `internal/app/cherry_pick_view.go`
- `internal/app/tagging.go`

지금 구조의 문제는 공통 helper 가 없어서가 아니다.
이미 `popupWidthForBody(...)`, `renderFloatingTitlePopup(...)`, `renderFloatingTitleFrame(...)`, `overlayPopup(...)` 같은 도구는 있다.
문제는 각 renderer 가 이 도구를 자기 방식대로 써서 결과가 조금씩 달라진다는 점이다.

## 추천 방향
### 추천안: 공통 오버레이 프레임 + 중앙 정렬 규칙

가장 좋은 방향은 overlay 를 세 가지 가족으로 정리하는 것이다.

- `toast`
- `confirm`
- `inspect`

각 가족은 폭, padding, title, footer, content 정렬 규칙을 공통으로 갖는다.
그리고 각 popup 은 content 만 다르게 넣는다.

이렇게 하면 다음 장점이 있다.

- 사용자 입장에서 overlay 가 한 제품처럼 느껴진다
- 코드에서는 "이 popup 은 어떤 가족인가" 만 보면 된다
- 앞으로 새로운 popup 을 추가할 때도 기준이 흔들리지 않는다

## 구현 접근
### 선택된 접근: Shared families

선택한 방향은 `toast`, `confirm`, `inspect` 세 가족으로 overlay 를 정리하는 것이다.

이 접근에서 중요한 점은 새 추상화를 크게 만드는 것이 아니다.
기존 helper 는 그대로 두고, 각 renderer 가 어떤 가족에 속하는지 먼저 정한다.
그 다음 각 가족별로 다음 규칙을 고정한다.

- title header 중앙 정렬
- footer shortcut 중앙 정렬
- content section 중앙 축 기준 배치
- 폭 clamp 와 padding 규칙 일관화

이 방식은 현재 코드와 잘 맞는다.
왜냐하면 `popupWidthForBody(...)`, `renderFloatingTitlePopup(...)`, `renderFloatingTitleFrame(...)`, `overlayPopup(...)` 가 이미 존재해서, 새 인프라를 만들지 않아도 되기 때문이다.

### 가족 분류 초안
- `toast`
  - loading popup
  - success / transient notice
  - alert popup
- `confirm`
  - confirm popup
  - branch delete / tag delete / stash pop confirm
  - fast-forward / pull confirm
  - reset mode popup
- `inspect`
  - review popup
  - stash list popup
  - hidden hotkeys popup
  - graph search popup
  - branch input popup
  - cherry-pick popup
  - target pick popup
  - stash message popup
  - graph stash pop popup
  - tag popup

이 분류는 첫 시도다.
구현하면서 한두 개는 이동할 수 있지만, 큰 축은 이대로 유지하는 편이 맞다.

특히 `tag popup` 은 `inspect` 로 둔다.
이 popup 은 사용자가 대상과 이름을 확인하면서 입력하는 흐름이어서, 단순 confirm 보다는 inspect 성격이 강하다.
또한 body 안에 별도 title 을 한 번 더 넣지 말고, popup title strip 하나로만 제목을 보여준다.

## 성공 기준
1. 어떤 overlay 를 열어도 title 은 가운데에 있다.
2. 어떤 confirm 을 열어도 footer shortcut 이 가운데에 있다.
3. content section 이 popup 내부에서 한쪽으로 쏠려 보이지 않는다.
4. narrow terminal 에서도 프레임이 깨지지 않는다.
5. 사용자가 overlay 를 바꿔도 "다른 UI" 처럼 느끼지 않는다.

## NOT in scope
- overlay 자체를 full screen 패널로 바꾸는 일
- toast/confirm 를 서로 다른 색 체계로 다시 정의하는 일
- 각 popup 에 완전히 새로운 레이아웃 컴포넌트를 도입하는 일
- 모바일 대응 또는 터치 대응
- 애니메이션 추가
- 입력/검증 로직 변경

이 작업은 프레임 통일이지 행동 변경이 아니다.

## What already exists
- `internal/app/view_layout.go` 의 `overlayPopup(...)` 가 body overlay 를 중앙에 얹는다.
- `internal/app/view_shell.go` 의 `popupWidthForBody(...)` 가 폭을 clamp 한다.
- `renderFloatingTitlePopup(...)` 와 `renderFloatingTitleFrame(...)` 이 title strip + body frame 을 이미 공유한다.
- `centerReviewFooterLine(...)` 와 `centerReviewLineInWidth(...)` 가 footer / 설명 텍스트의 중심 정렬 도구 역할을 한다.
- `docs/decisions.md` 에서도 popup overlay 는 이미 body 위에 얹는 것으로 정리돼 있다.

즉, 이번 일은 새 시스템 발명보다 기존 규칙을 한 방향으로 모으는 작업이다.

## 테스트 계획
이 작업은 렌더링 규칙을 바꾸는 일이므로, 기존 스냅샷과 문자열 검증을 기준으로 확인한다.

- `confirm popup` 의 title / body / footer 정렬이 유지되는지 본다.
- `reset mode popup` 이 confirm 계열 기준으로 정렬되는지 본다.
- `loading toast` 와 `alert popup` 이 같은 toast 문법을 공유하는지 본다.
- `review popup`, `target pick popup`, `branch input popup`, `graph search popup` 이 inspect 가족 기준으로 정렬되는지 본다.
- `hidden hotkeys popup` 과 `stash list popup` 이 가운데 축과 footer 정렬을 잃지 않는지 본다.
- `stash message popup`, `graph stash pop popup`, `cherry-pick popup`, `tag popup` 이 동일한 프레임 언어를 유지하는지 본다.
- `tag popup` 은 body 안에 중복 title 이 남지 않는지 확인한다.
- narrow terminal 에서도 overlay 가 깨지지 않는지 기존 overlay 스냅샷과 함께 확인한다.

## Dream state delta
### CURRENT STATE
overlay 는 같은 인프라를 쓰지만, 각 renderer 가 제각각 타협해서 결과가 들쑥날쑥하다.

### THIS PLAN
공통 프레임 규칙을 정하고, overlay 를 몇 개의 가족으로 분류해서 같은 정렬 언어를 쓰게 만든다.

### 12-MONTH IDEAL
새 overlay 를 추가할 때 “이건 toast 인가, confirm 인가, inspect 인가” 만 결정하면 된다.
화면을 보는 사람은 어떤 popup 이든 제목, 본문, shortcut 의 위치를 몸으로 안다.
UI 는 기능별로 달라도, 프레임 감각은 하나로 느껴진다.

## 다음 할 일
1. 현재 overlay renderer 를 전부 목록화한다.
2. 각 renderer 를 `toast`, `confirm`, `inspect` 중 하나로 분류한다.
3. 공통 프레임 규칙을 먼저 정한다.
4. 가장 단순한 `loading toast` 와 `confirm popup` 부터 맞춘다.
5. 나머지 popup 을 같은 규칙으로 옮긴다.
6. narrow terminal 에서 title, footer, content center 가 유지되는지 확인한다.
7. `docs/decisions.md` 에 overlay family 결정이 재사용 가능한 레이아웃 규칙으로 남도록 기록한다.

## 비고
이 문서의 핵심은 "예쁘게 만들자" 가 아니다.

핵심은 "같은 제품의 overlay 라면 같은 문법으로 읽히게 하자" 다.
그 기준이 있으면 toast, confirm, inspect 가 서로 다른 일을 하더라도 화면은 하나의 시스템처럼 보인다.
