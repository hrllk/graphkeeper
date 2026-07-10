# Context and Global Actions Rebalance Plan

## 목적

`Global` 과 `Context` 섹션의 표출 항목을 다시 나누고, `Graph` / `Local` / `Remote` / `Tags` 별로 보이는 hotkey 밀도를 조절한다.

이 문서는 제공한 시안을 기준으로 다음을 고정한다.

1. `Global` 은 섹션 공통 hotkey 와 전역 조작만 담는다.
2. `Context` 는 현재 섹션의 상태와 핵심 작업만 담는다.
3. `Graph` 의 hotkey 는 전부 상시 노출하지 않고, 기본 노출 + hidden drawer 로 나눈다.
4. `?` 는 더 이상 search 가 아니라, 숨겨진 hotkey 를 보여주는 진입점으로 쓴다.

## 배경

현재 `Graph` 섹션의 Actions 는 14개 수준으로 늘어났고, 2열 배치만으로는 화면이 금방 포화된다.
이 상태에서 모든 hotkey 를 같은 우선순위로 보여주면 다음 문제가 생긴다.

- 핵심 조작이 묻힌다.
- 섹션별 맥락이 약해진다.
- `Global` 과 `Context` 의 역할이 겹친다.
- 행 수가 늘면서 일부 항목은 잘리거나 시선 흐름이 끊긴다.

따라서 이번 작업은 "모든 hotkey 를 숨긴다"가 아니라, "지금 보일 것과 접어둘 것을 분리한다"가 핵심이다.

## 구현 전제

이 계획은 현재 코드 구조를 기준으로 다음 두 층으로 나눠서 본다.

### 즉시 구현 가능

- `Global` / `Context` 라벨과 카피 정리
- `Graph` action 우선순위 재배치
- `?` 를 search 대신 hidden drawer 진입점으로 전환
- `scroll`, `top`, `bottom` 의 Global 이관
- `Local` / `Remote` / `Tags` 의 기본 노출 밀도 정리

### 후속 보강 필요

- `Remote` 의 `last fetch` / `sync status`
- `Tags` 의 `tagger`
- `Tags` 의 `tagged time` / `message`

즉, 이번 문서의 목표는 시안과 같은 위계를 먼저 고정하고, 부족한 메타데이터는 후속 문서로 분리하는 것이다.

## 핵심 원칙

### 1. Global 은 공용 제어면이다

모든 섹션에서 의미가 같은 조작만 `Global` 에 둔다.

권장 대상:

- `tab` / `shift+tab`: section navigation
- `j/k`: move
- `f`: fetch
- `F`: fetch tags
- `S`: stash list
- `q`: quit
- `?`: hidden hotkey overlay

### 2. Context 는 현재 섹션의 상태면이다

`Context` 는 "이 섹션에서 지금 무엇을 할 수 있는가"만 보여준다.

`Context` 안에는 다음만 남긴다.

- 현재 focus 된 대상의 요약 정보
- 그 대상에 대해 바로 실행할 수 있는 핵심 action
- 나머지 action 은 `?` 오버레이로 확인 가능하다는 힌트

### 3. Graph 는 기본 4개 + hidden drawer 로 나눈다

`Graph` 는 정보량이 가장 많기 때문에, Actions 를 전부 같은 레벨로 보여주면 안 된다.

기본 노출 원칙:

- 상시 노출은 4개 내외로 제한한다.
- 상태 의존 action 은 조건이 맞을 때만 노출한다.
- 나머지는 `?` 오버레이로 접는다.

이 문서에서의 목표는 "무작정 숨김"이 아니라, "발견 가능한 접기"다.

## 범위

### 포함

- `Global` / `Context` 구획 재배치
- `Graph` Actions 우선순위 재정의
- `Graph` 전용 `?` search 제거
- `?` 를 hidden hotkey drawer 로 재정의
- `scroll`, `top`, `bottom` 을 `Global` 로 이관
- `Local` / `Remote` / `Tags` 섹션의 layout 정렬

### 제외

- Git 동작 자체의 변경
- stash pop / tag create / graph search 같은 기능 계약 변경
- section 전환 방식 변경
- render engine 전면 교체

## 화면 계약

### Global

`Global` 은 모든 섹션에서 공통으로 읽히는 제어면이다.

포함 항목:

- `tab`: next section
- `shift+tab`: previous section
- `j/k`: move
- `f`: fetch
- `F`: fetch tags
- `S`: stash list
- `q`: quit
- `?`: show hidden hotkeys

이중 `scroll`, `top`, `bottom` 관련 hotkey 는 `Graph Actions` 에 남기지 않고 `Global` 로 올린다.

### Context

`Context` 는 현재 섹션의 상태와 핵심 작업만 보여준다.

공통 규칙:

- 왼쪽은 details
- 오른쪽은 actions
- actions 는 기본 4개 중심
- 화면이 부족하면 `?` drawer 로 접는다

#### Context Details contract

`Context Details` 는 렌더링 코드에서 문자열 상태를 직접 조립하지 않고, 섹션별 스냅샷을 먼저 만든 뒤 그 스냅샷을 읽는 구조로 고정한다.

이 문서에서는 이를 2안으로 확정한다.

- 공통 진입점은 `ContextDetailsSnapshot` 이다.
- `ContextDetailsSnapshot` 은 섹션별 전용 구조를 담는 상위 snapshot 이다.
- 각 섹션은 전용 구조를 유지한다.
- 상태 판정은 별도 mapper 에서 하고, render 는 결과만 읽는다.

섹션별 필드는 다음을 기준으로 둔다.

- `GraphDetails`: `focus`, `parent`, `branches`, `stashes`, `tags`
- `LocalDetails`: `target`, `upstream`, `worktree`, `ahead`, `behind`, `divergenceState`
- `RemoteDetails`: `target`, `defaultBranch`, `lastFetch`, `branches`
- `TagDetails`: `name`, `hash`, `age`, `message`, `provenance`

상태 이름도 여기서 고정한다.

- `divergenceState`: `equal`, `aheadOnly`, `behindOnly`, `diverged`
- `provenance`: `unknown`, `local`, `origin`

의도:

- `ahead` / `behind` 는 숫자다.
- `divergenceState` 는 숫자를 묶는 판단 상태다.
- `provenance` 는 tag 의 출처 맥락이다.
- `renderContextInfoLines` 는 위 스냅샷만 읽고, repo 상태 계산을 직접 늘리지 않는다.

설명과 예시:

- `ahead` 는 현재 branch 가 upstream 보다 앞선 커밋 수다. 예: `ahead: 2` 는 아직 upstream 에 안 올라간 커밋이 2개 있다는 뜻이다.
- `behind` 는 현재 branch 가 upstream 보다 뒤처진 커밋 수다. 예: `behind: 1` 은 upstream 에는 있지만 로컬에 없는 커밋이 1개 있다는 뜻이다.
- `divergenceState` 는 `ahead` 와 `behind` 를 묶은 상태다. 예: `ahead: 1`, `behind: 1` 이면 `diverged` 다.
- `provenance` 는 tag 가 어디서 왔는지 나타낸다. `local` 은 로컬에서만 보이는 태그, `origin` 은 origin 쪽에서도 확인된 태그, `unknown` 은 아직 출처를 확정하지 못한 상태다.

## 섹션별 계획

### 1. Graph section

`Graph` 는 가장 강한 압축 규칙을 적용한다.

`Graph Details` 는 현재 graph cursor 가 가리키는 commit 을 기준으로 읽는다.

#### Graph Details

다음 정도만 유지한다.

- focus commit
- parent summary
- branch summary
- stash summary
- tag summary

표시 규칙:

- `focus` 와 `parent` hash 는 8자까지만 보여준다.
- `branches` 와 `stashes` 는 이름 목록으로 읽히되, 넘칠 경우 더보기 방식으로 숨긴다.
- `tags` 는 optional summary 로만 보여주고, 없으면 억지로 공간을 채우지 않는다.

예시:

- `focus: a1b2c3d4`
- `parent: 9f8e7d6c`
- `branches: HEAD, feature/login +2`
- `stashes: stash@{0}, stash@{1} +1`
- `tags: v1.2.0`

#### Graph Actions

`Graph Actions` 는 다음 두 층으로 나눈다.

#### Visible

- `m`: merge
- `r`: rebase
- `space`: checkout
- `H`: jump to HEAD

#### Conditional visible

현재 포커스와 repo state 에 따라 켜질 때만 보인다.

- `s`: reset
- `d`: delete branch
- `p`: pull
- `P`: push
- `t`: tag commit
- `o`: pop stash

#### Hidden / moved out

`Graph Actions` 에서 제거하고 `Global` 로 옮긴다.

- `gg`: top
- `G`: bottom
- `ctrl+u/d`: scroll

#### Search rule

- `Graph` 검색 진입은 `/` 로 유지한다.
- `?` 는 search 가 아니라 hidden hotkey drawer 이다.
- `Graph` 도움말에서 search 를 보여줄 때는 `/ : search` 만 남기고 `?` 는 제거한다.

### 2. Local section

`Local` 은 현재 checkout target 과 local branch 상태를 읽는 영역이다.

`Local Details` 는 현재 checkout target, 즉 현재 branch 기준으로 읽는다.

#### Local Details

시안 기준으로 다음을 보여준다.

- target
- upstream
- worktree state
- ahead / behind summary
- divergence state

표시 규칙:

- `upstream` 이 없으면 `none` 으로 보여준다.
- 연결된 upstream 이 있으면 `origin/...` 정보를 그대로 보여준다.
- `ahead` / `behind` 는 `HEAD` 와 upstream 의 차이를 숫자로 읽는다.
- `diverged` 는 `ahead > 0` 이고 `behind > 0` 일 때 사용한다.
- `target` 은 현재 checkout 대상이고, detached 상태면 `HEAD` 중심 표현으로 읽힌다.

예시:

- `target: feature/login`
- `upstream: origin/feature/login`
- `worktree: clean`
- `ahead: 2`
- `behind: 1`
- `divergenceState: diverged`

#### Local Actions

기본적으로 다음 우선순위를 따른다.

- `s`: stash changes
- `c`: clean working tree
- `space`: checkout
- `d`: delete branch
- `p`: pull
- `P`: push
- `a`: abort merge
- `n`: new branch

`Local` 은 `Graph` 보다 훨씬 좁게 읽혀야 하므로, 상태 의존 action 은 조건부로만 보이게 한다.

### 3. Remote section

`Remote` 는 upstream 상태와 remote branch 목록을 읽는 영역이다.

`Remote Details` 는 현재 repo 가 알고 있는 upstream / remote metadata 를 읽고, 선택 커서가 아니라 repo 단위 상태를 보여준다.

#### Remote Details

시안 기준으로 다음을 보여준다.

- upstream
- default branch
- last fetch
- branch count

표시 규칙:

- `last fetch` 는 fetch 시점이 없으면 `-` 로 둔다.
- `sync status` 는 현재 1차 표시 계약에서 제외한다.
- `branch count` 는 remote 브랜치 개수만 보여준다.

예시:

- `upstream: origin`
- `default branch: origin/main`
- `last fetch: 3m ago`
- `branch count: 8`

#### Remote Actions

핵심은 remote 대상 조작만 남기는 것이다.

- `space`: checkout
- `f`: fetch
- `p`: pull
- `d`: delete branch

필요한 경우 `push` 는 `Global` 이 아닌 `Local` / `Graph` 맥락과의 관계를 보고 노출한다.

### 4. Tags section

`Tags` 는 tag list inspector 로 유지하되, action 수는 적게 가져간다.

`Tags Details` 는 현재 tag cursor 가 가리키는 tag 를 기준으로 읽는다.
`GraphDetails` 는 `currentGraphFocus(m.repoStatus, m.sectionCursor[sectionGraph])` 를 기준으로 읽고,
`TagDetails` 는 `m.sectionCursor[sectionTags]` 가 가리키는 현재 tag 를 기준으로 읽는다.

#### Tags Details

시안 기준으로 다음을 보여준다.

- name
- hash
- age
- message
- provenance

표시 규칙:

- `name` 은 tag 이름이다.
- `hash` 는 tag가 가리키는 commit hash다.
- `age` 는 tag가 마지막으로 읽힌 상대 시점이다.
- `message` 는 commit subject 가 아니라 tag object message 다.
- `provenance` 는 `local`, `origin`, `unknown` 중 하나다.

예시:

- `name: v1.2.0`
- `hash: 68b6a97b`
- `age: 2w`
- `message: Release v1.2.0`
- `provenance: origin`

현재 코드에서는 `tagger` 와 일부 tag metadata 가 별도 필드로 유지되지 않으므로,
이 영역은 표시 가능한 정보부터 우선 정렬하고 메타데이터 보강은 후속으로 분리한다.

#### Tags Actions

- `enter`: jump to graph
- `d`: delete tag

## Hidden hotkey drawer

`?` 는 search 가 아니라 "숨겨진 hotkey 보기"로 작동한다.

이 drawer 의 역할은 다음과 같다.

- 현재 섹션의 숨겨진 hotkey 를 보여준다.
- 상태 의존 action 이 왜 안 보이는지 설명한다.
- `Global` 로 이관된 hotkey 도 함께 보여준다.

표시 방식은 overlay popup 이다.

- 기존 화면 위에 얹는 overlay 로 구현한다.
- `esc` 로 닫는다.
- search 팝업과 재사용하지 않는다.
- drawer 의 핵심 역할은 현재 섹션의 숨겨진 hotkey 와 이관된 hotkey 를 읽게 하는 것이다.

## 노출 전략

### 1. 기본은 4개

화면을 읽기 쉽게 유지하기 위해 각 Context action 영역은 기본 4개를 기준으로 둔다.

### 2. 상태 의존 action 은 조건이 맞을 때만 보인다

예를 들어 `Graph` 의 `o` 나 `d` 처럼 조건이 강한 action 은
"항상 자리만 차지하는 항목"이 아니라 "조건이 만족될 때만 나타나는 항목"이어야 한다.

### 3. `+N more` 는 쓰지 않는다

초기 아이디어로는 축약 라벨을 고려할 수 있었지만, 최종 계약에서는 채택하지 않는다.
이유는 `?` drawer 가 이미 overflow 발견 통로 역할을 수행하기 때문이다.

### 4. `?` 는 언제나 확장 통로다

전체 hotkey 가 많아질수록, `?` 는 가장 중요한 발견 진입점이 된다.

## 구현 순서

1. `Global` 과 `Context` 의 hotkey 책임을 재분리한다.
2. `Graph` Actions 에서 `scroll`, `top`, `bottom` 을 제거하고 `Global` 로 이동한다.
3. `Graph` 에서 `?` search 라벨을 제거하고 hidden hotkey drawer 로 바꾼다.
4. `Graph` / `Local` / `Remote` / `Tags` 별 기본 노출 hotkey 를 정리한다.
5. drawer overlay 를 연결한다.
6. 레이아웃과 텍스트가 잘리지 않는지 확인한다.

## 완료 기준

- `Graph` 에서 hotkey 가 14개여도 기본 영역은 과포화되지 않는다.
- `scroll`, `top`, `bottom` 은 `Global` 에서만 읽힌다.
- `?` 는 search 가 아니라 hidden hotkey 표시로 동작한다.
- 각 섹션은 제공한 시안과 같은 정보 밀도와 위계를 유지한다.
- `Context Actions` 가 늘어나도 화면이 깨지지 않는다.
- `Tags` / `Remote` 의 후속 메타데이터는 별도 문서로 분리한다.

## 구현 블루프린트

이 섹션은 실제 구현자가 바로 옮길 수 있게, 현재 코드와 목표 코드를 1:1로 맞춘다.

### 1. `internal/app/view_sections.go`

현재 액션 헬프는 한 함수에서 Graph, Current, Remote, Tags 를 모두 분기한다.
이 부분이 이번 변경의 중심이다.

```go
func renderActionHelpLines(m model) []string {
	switch m.status.Mode {
	case state.ModeBrowse:
		switch m.activeSection {
		case sectionGraph:
			return renderGraphActionHelpLines(m)
		case sectionCurrent:
			return renderLocalActionHelpLines(m)
		case sectionRemote:
			return renderRemoteActionHelpLines(m)
		case sectionTags:
			return renderTagActionHelpLines(m)
		default:
			return []string{"• no section actions"}
		}
	default:
		return renderModalHelpLines(m)
	}
}
```

Graph helper 는 기본 4개만 기본 렌더하고, 나머지는 drawer 로 넘긴다.

### 2. `internal/app/view_shell.go`

hidden drawer 는 기존 popup 오버레이 체계에 얹는다.
search popup 과 이름이나 동작을 공유하지 않는다.

### 3. `internal/app/key_handling_browse.go`

- `?` 는 graph search 진입이 아니라 drawer 토글이어야 한다.
- `/` 는 Graph search 용으로 유지한다.
- `esc` 는 drawer 를 닫는 동작으로 일관되게 처리한다.

### 4. `internal/app/view_detail.go`

- `Global` 은 공통 hotkey 만 보여준다.
- `Context` 는 현재 섹션의 세부정보와 핵심 action 만 보여준다.
- `Local` / `Remote` / `Tags` 의 action 개수는 과밀하지 않게 유지한다.

### 5. `internal/git/repo.go`

`Remote` / `Tags` 의 후속 메타데이터는 여기서 상태 필드를 확장하는 방향으로 처리한다.
이 문서의 1차 범위에는 포함하지 않는다.

## 문서 경계

이 문서는 현재 화면의 정보 밀도 재조정과 hotkey 재배치 계약을 고정한다.
아래는 별도 문서로 분리한다.

- `Remote` 의 `last fetch` / `sync status` 수집 방식
- `Tags` 의 `tagger` / `tagged time` / `message` 보강 방식
- hidden drawer 의 세부 스타일과 애니메이션
