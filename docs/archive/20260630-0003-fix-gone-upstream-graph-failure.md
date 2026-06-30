# Fix Graph Failure on Gone Upstreams

## 목적

`~/dotfiles` 같은 저장소에서 Graph 섹션이 비어 보이거나 멈춘 듯 보이는 문제를 고친다.

원인은 `gone` upstream 이 `git log` 입력에 그대로 들어가서 그래프 수집이 실패하는 것이다.

## 증상

- Graph 섹션이 렌더되지 않는다.
- 화면은 살아 있지만 그래프가 비어 보인다.
- 저장소에 `branch [origin/foo: gone]` 같은 브랜치가 있으면 재현된다.

## 원인

현재 흐름은 다음과 같다.

1. `branchMetadata()` 가 로컬 브랜치의 upstream 을 읽는다.
2. `graphRefs()` 가 upstream 을 그대로 `git log` 인자로 넣는다.
3. upstream 이 삭제된 상태면 `git log` 가 `fatal: ambiguous argument` 로 실패한다.
4. `Status()` 가 그래프를 못 만들고 빈 상태로 돌아간다.

문제 지점:

- [internal/git/repo_parse.go](/Users/hwiryungkim/task/sources/opensources/graphkeeper/internal/git/repo_parse.go)
- [internal/git/repo.go](/Users/hwiryungkim/task/sources/opensources/graphkeeper/internal/git/repo.go)

## 수정 방향

`gone` upstream 은 그래프 ref 후보에서 제외한다.

권장 순서:

1. `parseBranchMetadataLine()` 에서 `gone` upstream 을 빈 문자열로 정규화한다.
2. `graphRefs()` 에서 `git rev-parse --verify --quiet` 로 실제 존재하는 ref만 남긴다.
3. `Status()` 가 그래프 실패를 빈 그래프로 숨기지 말고, `ErrorMessage` 로 노출한다.

## 구현안

### 1) upstream 정규화

`parseBranchMetadataLine()` 은 `parseBranchUpstreamLine()` 과 같은 규칙을 써야 한다.

```go
func parseBranchMetadataLine(line string) (branchName, upstream string, tracking BranchTracking, ok bool) {
    parts := strings.SplitN(strings.TrimSpace(line), "|", 3)
    ...
    if len(parts) > 1 {
        upstream = strings.TrimSpace(parts[1])
        if strings.Contains(parts[2], "gone") {
            upstream = ""
        }
    }
    ...
}
```

더 깔끔한 선택은 `parseBranchUpstreamLine()` 과 공통 헬퍼로 빼는 것이다.

### 2) 존재하는 ref만 그래프에 사용

삭제된 upstream 이 다른 경로에서 다시 들어와도 그래프가 깨지지 않게 방어한다.

```go
func (r *Repo) graphRefExists(ctx context.Context, ref string) bool {
    _, err := r.git(ctx, "rev-parse", "--verify", "--quiet", ref)
    return err == nil
}
```

`graphRefs()` 에서 `add()` 하기 전에 확인한다.

### 3) 실패를 숨기지 않기

현재는 `isNoCommits()` 가 아니면 `Status()` 가 에러를 반환한다. 이 동작은 유지하되,
UI 에서 빈 그래프로 오해되지 않도록 상태 메시지를 남긴다.

```go
graphCommits, graphErr := r.graphCommits(ctx, localBranches, branchUpstreams, limit)
if graphErr != nil && !isNoCommits(graphErr) {
    return Status{ErrorMessage: graphErr.Error()}, graphErr
}
```

## 테스트

아래 케이스를 추가한다.

1. `parseBranchMetadataLine()` 가 `gone` upstream 을 빈 문자열로 만든다.
2. `graphRefs()` 가 존재하지 않는 upstream 을 제외한다.
3. `Status()` 가 stale upstream 이 있는 저장소에서 에러를 반환한다.

추천 테스트 입력:

- branch: `tmp3`
- upstream: `origin/tmp3`
- remote ref: 실제로는 없음

## 수동 검증

1. `~/dotfiles` 에서 실행한다.
2. Graph 섹션을 연다.
3. `origin/tmp3` 같은 stale upstream 이 있어도 UI 가 멈추지 않고, 최소한 에러를 보여야 한다.
4. 정상 저장소에서는 기존 그래프가 그대로 보여야 한다.

## 완료 기준

- stale upstream 이 있어도 Graph 섹션이 실패하지 않는다.
- `git log` 실패가 조용히 빈 화면으로 바뀌지 않는다.
- 기존 저장소의 그래프 렌더링은 회귀하지 않는다.
