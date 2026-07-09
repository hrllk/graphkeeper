# Graph Tagging and Tag List Plan

## 목적

`Graph` 섹션에서 선택한 commit 에 tag 를 붙일 수 있게 하고, 이미 존재하는 tag 들을 목록으로도 읽을 수 있게 한다.

현재 코드는 tag 를 완전히 무시하지는 않는다.

- `internal/git/repo.go` 가 `refs/tags` 를 읽는다.
- `internal/app/view_sections.go` 의 `Tags` 섹션이 tag 이름을 보여준다.
- `internal/app/graph_render_format.go` 와 `internal/app/graph_search.go` 는 `tag: ` decoration 을 일부 경로에서 제외한다.

하지만 아직 다음 두 가지는 없다.

1. `Graph` 에서 현재 focus 된 commit 에 tag 를 생성하는 흐름
2. tag 를 commit 중심으로 읽는 목록 UI

이 문서는 그 두 가지를 하나의 feature 로 묶어서 정리한다.

## 사용자 목표

1. `Graph` 에서 현재 보고 있는 commit 에 tag 를 붙일 수 있어야 한다.
2. tag 를 이름만 보는 것이 아니라, 어느 commit 에 붙었는지 함께 볼 수 있어야 한다.
3. `Graph` topology 는 tag UI 때문에 흐려지지 않아야 한다.
4. tag 생성과 tag 목록 보기는 같은 데이터 source 를 공유해야 한다.

## 범위

### 포함

- `Graph` focus commit 에 tag 생성
- tag 이름 입력 popup
- tag 생성 후 repo refresh
- tag 목록 UI
- commit 중심 tag grouping
- `Graph` row 의 compact tag marker
- 관련 tests

### 제외

- tag rename
- tag delete
- tag message / annotated tag 의 고급 편집
- remote tag sync 정책 재설계
- tag 를 git ref 가 아닌 로컬 메타데이터로 저장하는 방식

## 현재 상태

현재 구현은 tag 를 읽을 수는 있지만, tag 를 commit 에 붙이는 액션은 없다.

- `Tags` 섹션은 단순히 `refs/tags` 이름 목록을 보여준다.
- `Graph` row 에서는 `tag: v1.0.0` decoration 이 검색에서만 살짝 고려되고, 시각 표시로는 거의 쓰이지 않는다.
- tag 를 선택해서 graph row 로 이동하는 통합 flow 도 없다.

즉, 이번 작업은 새 개념을 invent 하는 것이 아니라, 이미 있는 git tag 를 UI contract 로 끌어올리는 작업이다.

## 핵심 결정

### 1. "tagging" 은 git tag 생성으로 해석한다

이 문서에서 말하는 tagging 은 custom label 이 아니라 실제 git ref 를 만든다는 뜻으로 둔다.

첫 버전은 lightweight tag 를 기본값으로 둔다.

- commit 중심으로 바로 만들 수 있다.
- 추가 입력이 적다.
- Graph 에서 쓰는 흐름과 맞다.

annotated tag 는 다음 단계 확장으로 남긴다.

### 2. tag 목록은 commit 중심 list 로 본다

목록은 tag 이름만 나열하는 것이 아니라, 어느 commit 에 붙었는지 함께 보여야 한다.

권장 표현은 다음이다.

- tag name
- target commit hash
- commit subject
- relative age

같은 commit 에 여러 tag 가 있으면 한 그룹으로 묶는다.

이렇게 하면 사용자는 "이 tag 가 무엇인지"와 "어디에 붙어 있는지"를 한 화면에서 같이 읽을 수 있다.

### 3. `Tags` 섹션은 list view 로 확장한다

기존 `Tags` 섹션이 있으므로, 별도 화면을 새로 만드는 것보다 이 섹션을 tag list inspector 로 키우는 편이 낫다.

이 섹션은 다음 역할을 맡는다.

- 현재 repo 의 tag 목록을 보여준다.
- 선택된 tag 가 가리키는 commit 을 표시한다.
- 필요하면 graph row 와 서로 이동한다.

즉, `Tags` 섹션은 "raw ref list" 에서 "commit-centered list" 로 바뀐다.

### 4. `Graph` row 에는 compact tag marker 만 둔다

tag 가 많은 repo 에서 row 안에 tag 이름을 전부 펼치면 graph 가 무거워진다.

그래서 `Graph` row 에는 다음만 둔다.

- tag 존재 여부를 보여주는 compact marker
- 같은 commit 에 tag 가 여러 개일 때의 count

자세한 tag 이름 목록은 `Tags` 섹션이 담당한다.

### 5. tag 생성과 tag 목록은 같은 source of truth 를 쓴다

생성 액션은 `git tag` 를 실행하고, 목록은 다시 `refs/tags` 를 읽는다.

중간 상태를 따로 저장하지 않는다.

이렇게 해야 repo 가 다른 터미널이나 external command 에 의해 바뀌어도 UI 가 바로 따라간다.

## UX 제안

### Graph 에서 tag 생성

`Graph` 에서 commit 을 focus 한 상태로 tag 생성 shortcut 을 제공한다.

예시 흐름:

1. 사용자가 `Graph` 에서 commit 을 선택한다.
2. `t` 같은 shortcut 으로 tag popup 을 연다.
3. tag 이름을 입력한다.
4. enter 를 누르면 `git tag <name> <commit>` 를 실행한다.
5. 성공하면 repo state 를 refresh 하고 같은 commit 에 머문다.

실패 케이스는 명확해야 한다.

- empty tag name
- 이미 존재하는 tag name
- target commit 이 사라진 stale state
- git tag 실행 실패

### Tag list 보기

`Tags` 섹션은 다음 정보를 보여주는 list 로 바꾼다.

```text
v1.2.0   8f3c1ab  Merge release/1.2
v1.1.0   7ac9d10  Fix login redirect
v1.0.0   3b812f4  Initial release
```

같은 commit 에 여러 tag 가 있으면 group 으로 묶는다.

```text
v1.2.0, latest   8f3c1ab  Merge release/1.2
```

이 list 는 이름만 보는 뷰가 아니라, graph 를 읽는 보조 뷰다.

### Graph 와 list 의 연결

Graph row 에 tag marker 가 있으면, 사용자는 `Tags` 섹션으로 내려가서 같은 commit 의 tag 묶음을 읽을 수 있어야 한다.

반대로 `Tags` 섹션에서 tag 를 선택하면 해당 commit 이 `Graph` 에서 focus 되도록 맞춘다.

이 연결이 있어야 "tag 를 붙였다" 와 "tag 를 찾았다" 가 같은 작업 흐름 안에 들어온다.

## 구현 순서

1. `internal/git/repo.go` 에 tag 생성 command 를 추가한다.
2. tag 입력 popup 과 tag action 상태를 추가한다.
3. `Graph` row 에 compact tag marker 를 렌더링한다.
4. `Tags` 섹션을 commit-centered list 로 확장한다.
5. tag 선택 시 graph focus 동기화를 넣는다.
6. tests 를 보강한다.

## 구현 파일 후보

- `internal/git/repo.go`
- `internal/app/commands.go`
- `internal/app/key_handling_browse.go`
- `internal/app/view_graph.go`
- `internal/app/graph_render_format.go`
- `internal/app/view_sections.go`
- `internal/app/navigation.go`
- `internal/app/model.go`
- `internal/app/model_test.go`
- `internal/git/repo_test.go`

## 테스트

다음 테스트가 필요하다.

```go
func TestCreateTagOnFocusedGraphCommit(t *testing.T)
func TestTagListGroupsTagsByTargetCommit(t *testing.T)
func TestGraphRendersTagMarkerForTaggedCommit(t *testing.T)
func TestTagSelectionJumpsToGraphCommit(t *testing.T)
```

검증 포인트는 다음이다.

- 동일 commit 에 tag 가 여러 개면 하나의 group 으로 묶이는지
- tag 없는 commit 에 marker 가 안 붙는지
- tag 생성 후 refresh 시 목록이 최신인지
- Graph 와 Tags 섹션이 같은 ref data 를 보는지

## 완료 기준

- `Graph` 에서 선택한 commit 에 tag 를 붙일 수 있다.
- `Tags` 섹션에서 tag 를 목록으로 읽을 수 있다.
- Graph 에서는 compact marker 만 보이고, 자세한 정보는 list 에서 본다.
- tag 생성 후 상태가 repo 기준으로 다시 동기화된다.
- 관련 테스트가 통과한다.
