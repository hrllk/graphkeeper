# internal/app/commands.go 테스트 보강 계획

> **For Hermes:** Use subagent-driven-development to implement this plan task-by-task.

**Goal:** `internal/app/commands.go`의 현재 구조는 유지하고, command workflow의 동작을 테스트로 고정한다.

**Architecture:** `internal/app` 안에서만 정리한다. `commands.go`는 현재 위치를 유지한다. 리팩토링보다 테스트 보강이 우선이다.

**Tech Stack:** Go, Bubble Tea (`tea.Cmd`), `internal/git`, `internal/state`, `go test`, `go build`

---

### TOC

- ### Goal
- ### Scope
- ### Review Notes
- ### BEFORE
- ### AFTER
- ### Tests
- ### Verification
- ### Notes

---

### Goal

`commands.go`는 파일 분리나 패키지 분리를 할 정도로 비대하지 않다.  
대신 command workflow는 상태 전이와 Git side effect를 직접 다루므로, 회귀를 막는 테스트가 필요하다.

이 계획의 목표는 다음 두 가지다.
- `commands.go`의 현재 구조를 유지한다
- workflow별 동작을 테스트로 고정한다

---

### Scope

- 코드 위치: `internal/app/commands.go`
- 테스트 위치: `internal/app/commands_test.go`
- 기본 원칙: 구조 변경 없이 테스트를 먼저 고정한다
- 새 package 생성은 하지 않는다
- 파일 분리도 이번 범위에는 포함하지 않는다

---

### Review Notes

- `commands.go`는 유지한다.
- 리팩토링보다 command 동작 보장이 우선이다.
- `repo.Status()` 실패는 기존처럼 에러로 전달해야 한다.
- `loadPullPreviewCommits`의 `HEAD..@{upstream}` / `HEAD...@{upstream}` 차이는 반드시 유지한다.
- `executeCheckout`의 remote branch fallback은 반드시 유지한다.
- `pullCheck`는 단순 wrapper가 아니라 pull 정책이므로 테스트가 필요하다.
- `executeAbort`는 merge/rebase 상태를 구분하는 규칙이 있으므로 테스트가 필요하다.
- `executeFetchForPush`와 `executeFetchForPull`은 status 재조회와 fetch error 전달을 함께 검증해야 한다.

---

### BEFORE

현재 `commands.go`는 command workflow가 여러 개 있고, 각각이 직접 Git side effect와 상태 재조회를 처리한다.

```go
func executePush(repo *git.Repo, branch string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Push(context.Background(), branch, false, false)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionPush, target: branch, err: statusErr}
		}
		return executedMsg{action: state.ActionPush, target: branch, status: status, err: err}
	}
}

func executeCheckout(repo *git.Repo, target string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Run("switch", target)
		if err != nil && strings.Contains(target, "/") {
			localName := target[strings.Index(target, "/")+1:]
			_, err = repo.Run("switch", "--track", "-c", localName, target)
		}
		if err != nil {
			return executedMsg{action: state.ActionCheckout, target: target, err: err}
		}
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionCheckout, target: target, err: statusErr}
		}
		return executedMsg{action: state.ActionCheckout, target: target, status: status}
	}
}
```

```go
func executeFetchForPush(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		err := repo.Fetch(context.Background())
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return pushFetchedMsg{err: statusErr}
		}
		return pushFetchedMsg{status: status, err: err}
	}
}

func loadPullPreviewCommits(repo *git.Repo, isFF bool) tea.Cmd {
	return func() tea.Msg {
		var arg string
		if isFF {
			arg = "HEAD..@{upstream}"
		} else {
			arg = "HEAD...@{upstream}"
		}
		out, err := repo.Run("rev-list", arg)
		if err != nil {
			return pullPreviewReadyMsg{err: err, isFF: isFF}
		}
		lines := strings.Split(out, "\n")
		commits := make([]string, 0, len(lines))
		for _, line := range lines {
			hash := strings.TrimSpace(line)
			if hash != "" {
				commits = append(commits, hash)
			}
		}
		if isFF {
			headOut, headErr := repo.Run("rev-parse", "HEAD")
			if headErr == nil && strings.TrimSpace(headOut) != "" {
				commits = append(commits, strings.TrimSpace(headOut))
			}
		}
		return pullPreviewReadyMsg{commits: commits, isFF: isFF}
	}
}
```

### AFTER

구조는 유지하고, 테스트로 동작을 고정한다.  
추가 helper는 만들 수 있지만, 파일 분리는 하지 않는다.

```go
func executePush(repo *git.Repo, branch string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Push(context.Background(), branch, false, false)
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionPush, target: branch, err: statusErr}
		}
		return executedMsg{action: state.ActionPush, target: branch, status: status, err: err}
	}
}

func executeCheckout(repo *git.Repo, target string, limit int) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Run("switch", target)
		if err != nil && strings.Contains(target, "/") {
			localName := target[strings.Index(target, "/")+1:]
			_, err = repo.Run("switch", "--track", "-c", localName, target)
		}
		if err != nil {
			return executedMsg{action: state.ActionCheckout, target: target, err: err}
		}
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return executedMsg{action: state.ActionCheckout, target: target, err: statusErr}
		}
		return executedMsg{action: state.ActionCheckout, target: target, status: status}
	}
}
```

```go
func executeFetchForPush(repo *git.Repo, limit int) tea.Cmd {
	return func() tea.Msg {
		err := repo.Fetch(context.Background())
		status, statusErr := repo.Status(context.Background(), limit)
		if statusErr != nil {
			return pushFetchedMsg{err: statusErr}
		}
		return pushFetchedMsg{status: status, err: err}
	}
}

func loadPullPreviewCommits(repo *git.Repo, isFF bool) tea.Cmd {
	return func() tea.Msg {
		var arg string
		if isFF {
			arg = "HEAD..@{upstream}"
		} else {
			arg = "HEAD...@{upstream}"
		}
		out, err := repo.Run("rev-list", arg)
		if err != nil {
			return pullPreviewReadyMsg{err: err, isFF: isFF}
		}
		lines := strings.Split(out, "\n")
		commits := make([]string, 0, len(lines))
		for _, line := range lines {
			hash := strings.TrimSpace(line)
			if hash != "" {
				commits = append(commits, hash)
			}
		}
		if isFF {
			headOut, headErr := repo.Run("rev-parse", "HEAD")
			if headErr == nil && strings.TrimSpace(headOut) != "" {
				commits = append(commits, strings.TrimSpace(headOut))
			}
		}
		return pullPreviewReadyMsg{commits: commits, isFF: isFF}
	}
}
```

### Tests

리팩토링이 아니라 동작 고정이 목적이므로, 다음 케이스를 먼저 테스트한다.

```go
func TestLoadRepoState(t *testing.T) { /* ... */ }
func TestRefreshRepoState(t *testing.T) { /* ... */ }
func TestFetchRepoState(t *testing.T) { /* ... */ }
func TestPrepareAction(t *testing.T) { /* ... */ }
func TestPullCheck(t *testing.T) { /* ... */ }
func TestExecutePullVariants(t *testing.T) { /* ... */ }
func TestExecuteAbortKeepsMergeAndRebaseSplit(t *testing.T) { /* ... */ }
func TestExecutePushVariants(t *testing.T) { /* ... */ }
func TestExecutePushSetUpstream(t *testing.T) { /* ... */ }
func TestExecuteCheckoutKeepsRemoteFallback(t *testing.T) { /* ... */ }
func TestExecuteAction(t *testing.T) { /* ... */ }
func TestCreateBranch(t *testing.T) { /* ... */ }
func TestExecuteFetchForPush(t *testing.T) { /* ... */ }
func TestExecuteFetchForPull(t *testing.T) { /* ... */ }
func TestLoadPullPreviewCommitsUsesCorrectRange(t *testing.T) { /* ... */ }
```

테스트 위치는 `internal/app/commands_test.go`를 기본으로 둔다.

---

### Verification

```sh
go test ./internal/app
go test ./...
go build ./cmd/graphkeeper
```

---

### Notes

- `commands.go`는 유지한다.
- 구조 분리보다 command workflow의 회귀 방지가 목적이다.
- `repo.Status()` 실패는 숨기지 않고 기존처럼 에러로 전달한다.
- `executeCheckout`의 remote fallback은 반드시 유지한다.
- `loadPullPreviewCommits`의 rev-list 범위는 그대로 둔다.
- `pullCheck`, `executeAbort`, `executeFetchForPush`, `executeFetchForPull`는 테스트 우선순위가 높다.
- 테스트는 코드 옆 `*_test.go`로 둔다.
