# Deferred Commit Inspector work

이 항목들은 현재 `feature` 아래의 유일한 `commit detail` subtask 범위를 넓히지
않는 후속 TODO다. Taskmaster sibling subtask로 등록하지 않는다.

- [ ] 실제 Tree-sitter grammar/query bundle과 언어별 라이선스·artifact 크기 정책을
      별도 dependency slice로 결정한다.
- [ ] merge commit parent selector와 combined diff를 first-parent MVP 이후의 별도
      history-inspection 확장으로 설계한다.

- [ ] Git 상태 필드별 조회 오류를 보존하고 `No remote`/`No upstream` 오인 표시를
      막는 상태 신뢰도 정책을 별도 설계한다. 현재 `internal/git/repo.go`는 head,
      upstream, remote, worktree 조회 오류를 개별적으로 무시하므로, 명령 실패와
      실제 설정 부재를 구분할 수 없다. 저장소 상태 계약과 파싱 테스트를 먼저
      정의한 뒤 구현한다.
