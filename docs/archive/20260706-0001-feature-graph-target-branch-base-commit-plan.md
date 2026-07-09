# Graph Section Target Branch Base Commit Plan

## 목적

`Graph` 섹션에서 현재 브랜치와 특정 target branch 사이의 분기점, 즉 `base commit` 을 찾는 기능을 정의한다.

이 기능의 목표는 단순히 “merge/rebase 를 할 수 있나”를 보여 주는 것을 넘어서, 사용자가 **어느 커밋에서 두 히스토리가 갈라졌는지**를 바로 확인하게 만드는 것이다.

## 문제 정의

지금 `Graph` 의 target 선택 흐름은 target 을 고르고 divergence 를 계산하는 데는 충분하지만, 분기점 자체는 화면에서 드러나지 않는다.

그래서 사용자는 다음 질문을 다시 머릿속에서 계산해야 한다.

1. current branch 와 target branch 는 어디서 갈라졌나
2. 그 분기점 이후에 양쪽이 각각 얼마나 커졌나
3. merge/rebase 전에 어떤 커밋을 기준으로 판단해야 하나

이 문서는 그 질문을 UI 와 preview 단계에서 한 번에 해결하는 방향을 잡는다.

## 사용 시나리오

- `Graph` 에서 현재 브랜치와 병합할 대상 브랜치를 고른다.
- target branch 와 current branch 의 `base commit` 을 즉시 확인한다.
- 분기점 이후 current branch 쪽 고유 커밋 수와 target branch 쪽 고유 커밋 수를 같이 본다.
- merge 전에 “진짜로 갈라진 브랜치인지”를 빠르게 판단한다.

## 범위

### 포함

- Graph target 선택 흐름
- current branch 와 target branch 의 `base commit` 계산
- divergence count 와 base commit 정보를 함께 보여 주는 preview
- base commit 으로 Graph row 를 이동하거나 강조하는 동작
- 관련 tests

### 제외

- Git history 재작성
- merge/rebase 실행 의미 변경
- server-side git 분석
- full text search 나 별도 index 도입

## 핵심 계약

### 1. base commit 은 current branch 와 target branch 의 merge-base 로 계산한다

이 기능에서 base commit 은 `HEAD` 와 target ref 사이의 `git merge-base` 결과로 정의한다.

가능하면 `target` 이 remote tracking branch 인 경우에도 동일한 규칙을 따른다. target 의 표현이 `origin/feature-x` 이든 `feature-x` 이든, 실제 계산은 ref resolution 이후의 커밋 기준으로 끝내야 한다.

### 2. preview 는 divergence 수치와 base commit 정보를 같이 보여 준다

기존 `HEAD...target` divergence 계산은 유지하되, 그 결과 옆에 base commit hash 와 subject 를 추가한다.

사용자가 봐야 하는 정보는 다음 셋이다.

- 분기점이 있는가
- 분기점이 있다면 어느 커밋인가
- 양쪽이 분기점 이후에 각각 몇 개의 고유 커밋을 가졌는가

### 3. Graph 에서 base commit 으로 바로 이동할 수 있어야 한다

base commit 을 찾는 기능은 텍스트 출력만으로 끝내지 않는다.

`Graph` 에서 target branch 를 고른 뒤:

- base commit row 를 강조하거나
- cursor 를 base commit row 로 이동시키거나
- 최소한 detail pane 에 base commit 을 고정 표시해야 한다

이 중 하나는 반드시 있어야 한다. 사용자가 분기점을 다시 찾지 않게 만드는 것이 핵심이다.

### 4. 분기점이 없는 경우도 명시적으로 다뤄야 한다

다음 경우는 별도 메시지가 필요하다.

- target 이 `HEAD` 와 같은 커밋인 경우
- target 이 `HEAD` 의 ancestor 인 경우
- `HEAD` 가 target 의 ancestor 인 경우

이때는 base commit 을 “없음”으로 처리하는 것이 아니라, 왜 분기점이 성립하지 않는지 보여 줘야 한다.

## 구현 방향

### 1. Git 레이어에 merge-base 조회를 추가한다

`internal/git` 에 current branch 와 target ref 사이의 merge-base 를 구하는 helper 를 둔다.

예시 형태:

```go
func (r *Repo) MergeBase(ctx context.Context, left, right string) (string, error)
```

이 helper 는 `Divergence()` 와 같은 수준의 저수준 Git wrapper 로 두는 편이 낫다.

### 2. target preview 에 base commit 을 포함한다

현재 preview 흐름은 `previewSelection()` -> `buildActionPreview()` 로 이어진다.
이 흐름에 merge-base 조회를 붙여서, preview 결과가 분기점 hash, subject, divergence count 를 함께 가지도록 만든다.

### 3. Graph selection 과 base commit lookup 을 분리한다

Graph row 선택은 화면 이동 책임이고, base commit 계산은 Git 책임이다.
둘을 한 함수에 몰아넣지 말고 다음처럼 분리한다.

- selection: 어떤 target 을 골랐는가
- lookup: 그 target 의 base commit 은 무엇인가
- rendering: 그 정보를 어디에 보여 줄 것인가

### 4. 기존 divergence 판단을 재사용한다

이 기능은 새로 “분기 여부”를 다시 정의하지 않는다.

이미 있는 `HEAD...target` divergence 계산을 재사용하고, 거기에 merge-base 정보를 덧붙인다.
즉, base commit 은 판단 근거를 보강하는 정보이고, 분기 여부는 기존 계약을 유지한다.

## 구현 파일

- `internal/git/repo_exec.go`
- `internal/app/preview.go`
- `internal/app/commands.go`
- `internal/app/key_handling_target.go`
- `internal/app/view_sections.go`
- `internal/app/model.go`
- `internal/app/model_test.go`
- `internal/app/preview_test.go`
- `internal/app/key_handling_test.go`

## 테스트 방향

아래 항목을 테스트로 고정한다.

1. `HEAD` 와 target 사이의 merge-base 가 올바르게 계산되는지 확인한다.
2. target 이 `HEAD` 와 같은 경우, base commit 이 명시적으로 비활성 처리되는지 확인한다.
3. target 이 ancestor / descendant 관계일 때 분기점 메시지가 혼동되지 않는지 확인한다.
4. preview 에 divergence count 와 base commit 정보가 같이 표시되는지 확인한다.
5. Graph 에서 base commit 으로 이동하거나 강조하는 동작이 cursor / scroll 과 함께 유지되는지 확인한다.
6. upstream branch 와 local branch 둘 다 동일한 규칙으로 동작하는지 확인한다.

## 완료 기준

- 사용자가 Graph 에서 target branch 를 고르면 base commit 이 바로 나온다.
- base commit 이 없는 상태는 이유까지 함께 보인다.
- divergence count 는 기존 기준과 일치한다.
- target 선택 후 다시 분기점을 찾기 위해 별도 계산을 반복할 필요가 없다.

## 메모

이 문서는 Graph 의 merge/rebase 판단을 바꾸려는 문서가 아니다.

핵심은 “현재 브랜치와 target branch 가 어디서 갈라졌는지”를 사용자가 바로 읽을 수 있게 만드는 것이다.
