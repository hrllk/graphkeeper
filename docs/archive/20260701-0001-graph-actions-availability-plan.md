# Graph Actions Availability Plan

## 목적

`Graph` 섹션의 `Actions` 영역이 선택된 포인터에 따라 너무 다르게 보이는 문제를 정리한다.

이번 계획의 핵심은 액션 목록 자체를 자주 바꾸는 것이 아니라, 같은 액션 집합을 유지한 채 `활성 / 비활성` 상태만 명확하게 보여주는 것이다.

이 계획은 우측 `Local` 패널의 2열 분리 이후에 적용하는 것이 더 안전하다. 먼저 정보와 Actions 의 표시 경계를 분리한 뒤, 그 다음에 Actions 의 가용성 표현을 고정한다.

사용자가 확인해야 하는 것은 "지금 이 액션을 실행할 수 있는가"이지, 포인터가 바뀔 때마다 완전히 다른 도움말 목록이 아니다.

## 증상

- Graph 섹션의 `Actions` 내용이 포인터나 란(lane) 상태에 따라 달라진다.
- 일부 액션은 보이고, 일부는 숨겨지거나 문구가 바뀌어서 같은 화면 안에서도 도움말 기준이 흔들린다.
- 사용자는 `merge`, `rebase`, `pull`, `reset`, `new branch` 같은 Graph 액션의 공통 구조를 한눈에 읽기 어렵다.

## 원인

현재 `renderActionHelpLines()` 는 Graph 섹션에서 상태를 보고 문구를 분기한다.

- local lane 이 아니면 `merge` / `rebase` 를 비활성 표시한다.
- `pullReady()` 가 아니면 `pull` 을 비활성 표시한다.
- `canCreateBranch()` 가 아니면 `new branch` 를 비활성 표시한다.
- `reset` / scroll / jump 류는 항상 보이지만, 전체 액션 구조는 일관되게 고정되어 있지 않다.

이 방식은 "실행 가능 여부"를 보여주려는 의도는 맞지만, 결과적으로 `Actions` 패널이 상태 설명판처럼 흔들리는 부작용이 있다.

## 목표 상태

Graph 섹션의 `Actions` 는 다음 원칙을 따른다.

1. 액션 목록은 가능한 한 고정한다.
2. 상태에 따라 달라지는 것은 `활성 / 비활성` 표현과 필요한 짧은 사유뿐이다.
3. 숨김보다 비활성 표현을 우선한다.
4. 사용자가 포인터를 조금 움직여도 `Actions` 의 구조가 바뀌지 않게 한다.

권장 표현 예시:

- `• m: merge` 또는 비활성 스타일 + `local lane only`
- `• r: rebase` 또는 비활성 스타일 + `local lane only`
- `• p: pull` 또는 비활성 스타일 + `no upstream`
- `• s: reset`
- `• n: new branch` 또는 비활성 스타일 + `dirty`

즉, 액션의 존재 자체는 유지하고, 가능한지 여부만 표현한다.

## 범위

### 포함

- Graph 섹션 `Actions` 의 고정 목록화
- 활성 / 비활성 표현 규칙 정리
- 비활성 사유 문구 통일
- Graph 이외 섹션과의 표현 일관성 점검
- 관련 테스트 추가

### 제외

- 실제 Git 동작 의미 변경
- Graph 선택 규칙 변경
- command 실행 flow 변경
- action shortcut 자체의 재배치

## 구현 방향

### 1. 액션 목록을 먼저 고정한다

Graph 섹션에서는 다음 항목을 기본 목록으로 유지한다.

- `merge`
- `rebase`
- `pull`
- `reset`
- `scroll`
- `jump to top / bottom`
- `jump to HEAD`
- `new branch`

이 목록은 포인터나 lane 상태에 따라 생성/삭제하지 않는다.

### 2. 상태 판정은 별도 helper 로 모은다

`renderActionHelpLines()` 안에서 직접 조건 분기를 계속 늘리지 말고, Graph 액션별 가능 여부를 반환하는 helper 를 둔다.

예시 방향:

```go
type actionAvailability struct {
    enabled bool
    reason  string
}

func graphActionAvailability(m model) map[string]actionAvailability
```

이렇게 하면 `renderActionHelpLines()` 는 "문구 조립"만 담당하고, "가능 여부 판단"은 다른 곳에서 책임지게 된다.

### 3. 숨김보다 비활성 표현을 우선한다

현재처럼 아예 안 보이는 액션이 생기면, 사용자 입장에서는 왜 사라졌는지 다시 추적해야 한다.

이번 정리는 다음 규칙을 따른다.

- 실행 불가 액션도 목록에는 남긴다.
- 단, `disabled` 스타일과 짧은 사유를 붙인다.
- 사유는 길게 설명하지 않는다.

권장 사유 예시:

- `local lane only`
- `current branch lane`
- `no upstream`
- `dirty`

### 4. 문구 길이를 통제한다

Graph 는 이미 정보 밀도가 높은 영역이므로, Actions 문구는 길어지면 안 된다.

권장 규칙:

- 액션 이름은 짧게 유지한다.
- 사유는 1개만 붙인다.
- 같은 줄에 너무 많은 보조 정보를 넣지 않는다.

## 테스트 방향

아래 항목을 테스트로 고정한다.

1. Graph 섹션의 Actions 목록이 포인터 이동만으로 구조적으로 바뀌지 않는지 확인한다.
2. `merge` / `rebase` 가 비활성인 경우에도 목록은 유지되는지 확인한다.
3. `pull` 이 불가한 상태에서도 항목 자체는 보이는지 확인한다.
4. `new branch` 가 dirty 상태에서 비활성 표현되는지 확인한다.
5. Graph 외 섹션의 Actions 는 기존 규칙대로 유지되는지 확인한다.

## 구현 순서

1. Graph 섹션의 현재 `Actions` 출력 패턴을 정리한다.
2. 액션별 가용성 판정을 helper 로 분리한다.
3. `renderActionHelpLines()` 를 고정 목록 기반으로 정리한다.
4. 비활성 사유 문구를 통일한다.
5. 관련 렌더 테스트를 추가한다.
6. 실제 화면에서 포인터 이동 시 `Actions` 가 흔들리지 않는지 확인한다.

## 완료 기준

- Graph 섹션에서 `Actions` 의 구조가 포인터 상태에 따라 바뀌지 않는다.
- 활성 / 비활성만 명확하게 바뀐다.
- 비활성 사유가 짧고 일관되게 보인다.
- 기존 shortcut 과 실행 흐름은 유지된다.

## 메모

이 문서는 Graph 액션의 "표현"을 안정화하는 계획이다.

실행 가능한 액션의 범위를 더 줄이거나 늘리는 결정은 별도 문서로 분리하는 편이 낫다.
