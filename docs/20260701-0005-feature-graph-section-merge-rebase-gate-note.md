# Graph Section Merge/Rebase Gate Note

## 배경

`Graph` 섹션의 `merge` / `rebase` 는 현재 포인터가 로컬 lane 이면 활성처럼 보이고, 아니면 비활성처럼 보인다.
이 기준은 사용자에게 “이 포인터가 로컬 브랜치인가”만 알려 준다. 실제로 `merge` / `rebase` 가 의미 있는지까지는 보지 못한다.

현재 코드에서 이 판정은 `isLocalGraphPointer()` 에 묶여 있다.

- `internal/app/view_sections.go`
- `internal/app/key_handling_browse.go`
- `internal/app/graph_rules.go`

즉, 지금 gate 는 **그래프 토폴로지**가 아니라 **로컬 포인터 여부**에 가깝다.

## 문제 정의

이번에 바꾸려는 조건은 “로컬 브랜치에 서 있으면 된다”가 아니다.
원하는 건 다음에 가깝다.

1. 기본 상태는 비활성이다.
2. 활성 전환 조건은 단순 local lane 여부가 아니다.
3. 선택된 커밋이 `HEAD` 와 관계를 가질 때, 그 관계가 “이미 포함됨”이나 “fast-forward 가능” 같은 단순 케이스가 아니라, **양쪽이 각각 고유 커밋을 가진 분기 상태**여야 한다.

말을 줄이면, `Graph` 에서 `merge` / `rebase` 를 살릴 이유는 **실제로 갈라진 두 히스토리를 다루는 경우**다.

## 정리해야 할 판단 기준

`git rev-list --left-right --count HEAD...target` 로 얻는 두 값이 중요하다.

- `currentOnly` 는 `HEAD` 쪽 고유 커밋 수
- `targetOnly` 는 target 쪽 고유 커밋 수

이 값으로 토폴로지를 나누면 다음과 같다.

| 상태 | 의미 | Graph merge/rebase gate |
| --- | --- | --- |
| `currentOnly == 0 && targetOnly == 0` | 같은 커밋 | 비활성 |
| `currentOnly == 0 && targetOnly > 0` | target 이 HEAD 의 앞쪽, fast-forward 가능 | 비활성 유지 권장 |
| `currentOnly > 0 && targetOnly == 0` | target 이 이미 history 안에 있음 | 비활성 유지 권장 |
| `currentOnly > 0 && targetOnly > 0` | 서로 분기된 상태 | 활성 후보 |

여기서 핵심은 “HEAD 보다 앞서나가지 않은 커밋” 같은 서술보다, **양쪽 고유 커밋이 동시에 존재하는가**로 보는 편이 훨씬 덜 애매하다는 점이다.

## 권장 해석

Graph 섹션의 `merge` / `rebase` 는 다음 조건을 모두 만족할 때만 활성화하는 편이 낫다.

- 현재 포인터가 로컬 브랜치 기준이다.
- 선택 대상이 HEAD 와 같은 커밋이 아니다.
- 선택 대상이 HEAD 의 ancestor 도 아니고, HEAD 가 target 의 ancestor 도 아니다.
- 다시 말해 `currentOnly > 0 && targetOnly > 0` 이다.

이렇게 하면 다음이 분리된다.

- `fast-forward` 계열은 pull 또는 별도 preview 흐름이 담당
- `already in history` 계열은 비활성
- 진짜 merge/rebase 가 필요한 분기 상태만 Graph 에서 활성

## 왜 이 해석이 맞는가

`Graph` 섹션은 “현재 브랜치에서 무엇을 할 수 있는가”를 보여 주는 자리다.
여기서 merge/rebase 를 로컬 lane 여부만으로 켜 버리면, 다음 상태가 섞인다.

- 단순히 로컬 브랜치 위에 있는가
- 실제로 히스토리가 분기되었는가
- 그 대상이 fast-forward 인가
- 이미 포함된 커밋인가

사용자가 원하는 건 이 네 가지를 한 줄로 뭉개는 게 아니라, **서로 다른 Git 상태를 다른 액션으로 나누는 것**이다.

## 주의할 점

- `graph` 상의 이전 커밋이라는 이유만으로 활성화하면 안 된다.
  - ancestor 는 이미 포함된 히스토리일 수 있다.
- `merge` 와 `rebase` 를 같은 gate 로 묶되, 실행 시점의 세부 의미는 preview 단계에서 분리해야 한다.
- UI 문구는 “local lane only” 보다 “diverged branch only” 쪽이 더 정확할 수 있다.

## 이미 지나간 커밋의 활성 여부

지나온 커밋도 활성 후보가 될 수 있다.
현재 구현은 `isLocalGraphPointer()` 가 로컬 브랜치의 현재 tip 만 보는 게 아니라, 그 브랜치가 지나온 lane 위의 커밋도 local 로 간주한다.

즉,

- local 브랜치 tip
- local 브랜치의 과거 커밋

둘 다 Graph 섹션에서 `merge` / `rebase` 활성 후보가 될 수 있다.

## 그 상태에서의 실행 의미

선택한 대상이 이미 `HEAD` 의 history 안에 있는 경우, `previewSelection()` 은 `git rev-list --left-right --count HEAD...target` 결과를 보고 분기 여부를 계산한다.

- `targetOnly == 0` 이면 target 은 이미 포함된 커밋이다.
- 이 경우 `merge` 는 사실상 이미 반영된 상태로 끝날 가능성이 높다.
- `rebase` 는 현재 브랜치의 뒤쪽 커밋을 그 지점 위로 다시 쌓는 동작이 된다.

즉, 이미 지나간 커밋을 고르면 결과가 다음처럼 갈린다.

- `merge`: 대체로 no-op 또는 already up to date 계열
- `rebase`: 현재 브랜치의 이후 커밋을 다시 재배열하는 작업

## 결론

`Graph` 섹션의 merge/rebase 활성 조건은 단순 로컬 lane 판정이 아니라, **`HEAD` 와 target 이 서로 고유 커밋을 가진 분기 상태인지**로 재정의하는 게 맞다.
즉, 실질 조건은 `currentOnly > 0 && targetOnly > 0` 이다.

이 기준으로 바꾸면, fast-forward 나 이미 포함된 커밋을 잘못 활성화하는 일을 줄일 수 있다.
