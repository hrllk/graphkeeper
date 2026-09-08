# Deferred Commit Inspector work

이 항목들은 현재 `feature` 아래의 유일한 `commit detail` subtask 범위를 넓히지
않는 후속 TODO다. Taskmaster sibling subtask로 등록하지 않는다.

- [ ] 실제 Tree-sitter grammar/query bundle과 언어별 라이선스·artifact 크기 정책을
      별도 dependency slice로 결정한다.
      선행 결정 필요: syntax highlighting 자체가 아직 제품 범위에 없고 `go.mod`에
      관련 의존성도 없다. highlighting을 도입할지 먼저 정한 뒤에 착수한다.
- [ ] merge commit parent selector와 combined diff를 first-parent MVP 이후의 별도
      history-inspection 확장으로 설계한다.

- [ ] `git.Status`의 남은 오류 무시 지점을 정리한다. (2026-08-28 재확인)
      head/upstream/remote 조회 오류는 이미 `TrackingError`/`TrackingFresh`로
      전파된다. 남은 문제는 다음 네 곳이다.
      - `noRemote`가 `remote == ""`만 보고 `remoteErr`를 무시하므로 `git remote`
        실패가 `No remote`로 오인 표시된다.
      - `worktreeDirty, _ :=` 실패 시 dirty 상태가 clean으로 표시된다.
      - `remoteBranches, _ :=`와 `lastFetchAt, _ :=`도 오류를 버린다.
      저장소 상태 계약과 파싱 테스트를 먼저 정의한 뒤 구현한다.

      범위 축소 (2026-08-26, /plan-eng-review): tag와 stash 부분은
      `hrk-main-design-20260826-145518.md` 작업에서 닫는다. 그 작업이 `TagsKnown`/
      `StashKnown` + `*Error` 3상태를 도입하며, 정답 관용구는 이미
      `internal/app/repository_read.go`의 `LocalBranchesKnown`/`LocalBranchesFresh`/
      `LocalBranchesError`에 있다. 같은 패턴을 나머지 네 필드에 복사하면 된다.
      이 항목이 실제 사용자 대면 버그로 터진 사례: `git stash list` 실패 시
      `internal/app/update_stash.go:16-19`가 오류를 telemetry sink로만 보내고
      `(no stash entries)`를 표시했다 — 성공하고 0개인 경우와 구분 불가.

- [ ] `internal/app/repository_read.go:39` `RepositoryProjection`을 관심사별로
      분리한다. 현재 30개 이상 필드가 한 구조체에 있다: 식별(Root/Branch/Head/
      Upstream/Remote/DefaultBranch), 브랜치 목록(Branches/LocalBranches/
      LocalBranchesKnown/Fresh/Error/RemoteBranches/BranchUpstreams), 추적
      (Tracking/TrackingKnown/TrackingFresh), 워크트리(WorktreeDirty/Detached/
      EmptyRepo), 원격 부재(NoRemote/NoUpstream/UpstreamGone), 진행 중 작업
      (Merge/Rebase/CherryPickInProgress/ConflictTarget), 신선도(LastFetchAt/
      RemoteSyncSummary). `Graph`는 이미 `ReadSnapshot`에서 형제 필드로 분리돼
      있어 일관성이 없다.

      선행 조건: 위 design doc의 Phase 3(계약 통합) 완료 후. 지금 나누면 통합
      diff와 섞여 리뷰가 불가능해진다. `docs/roadmap.md`의 "대칭을 위해 패키지를
      만들지 말라"와 긴장하는 지점이므로, 나눌 근거를 먼저 적을 것.

- [ ] Phase 4(UI 폴리시) 착수 **전에** `/gstack-plan-design-review`를 돌린다.
      T1(빈 상태 문구), T20(header `commit:` + 100 cell), T21(`unsupported height`
      문구 + Inspector 중앙 정렬 폭), T18(`?` 오버레이 목록)은 사용자가 직접 보는
      면이고 `DESIGN.md`가 지배한다. 계획에는 "사용자 대면 문구로 바꿔라"까지만
      있고 어떤 문구인지는 없다. 구현 중에 정하면 `DESIGN.md`와 어그러진다.
      (/plan-eng-review D20에서 A를 고른 대가로 남긴 알림)

- [ ] `hrk-main-design-20260826-145518.md`의 Success Criteria가 excluded tag/stash
      단언을 "**gone**, not inverted — deleted"라고 쓰는데, 같은 문서의 T7은
      "invert", T15는 "delete once the dual path is gone"이다. T7 → T15 순서로
      읽으면 말이 되지만 Success Criteria 문장이 과장돼 있다.

      위험: T7을 집는 사람이 Success Criteria를 먼저 읽으면 단언을 지금 지워버리고,
      그러면 Phase 3 전까지 듀얼 패스를 지키는 가드가 사라진다.

      고치는 법: Success Criteria를 "T7이 반전하고, 듀얼 패스가 사라지는 T15에서
      삭제한다"로 바꾼다. 발견: 2026-08-28 /plan-ceo-review (HOLD SCOPE).

- [ ] Commit Inspector의 죽은 렌더러를 정리한다. (2026-09-08 /plan-ceo-review C7)
      `internal/app/commit_inspector.go:122` `renderCommitInspectorPopup`은 프로덕션
      호출자가 0개다. `view_shell.go:49`는 `renderCommitInspectorScreen`만 호출한다.
      호출자는 `commit_inspector_test.go:74`와 `:98` 둘뿐이므로, 테스트가 죽은 코드를
      살아 있는 것처럼 고정하고 있다.

      위험: "일관성을 위해 두 렌더 경로를 모두 고쳐라"라는 지시를 따르는 사람이
      죽은 코드를 유지보수한다. `20260907-0001` 계획 초안이 실제로 그 지시를 적었고
      리뷰에서 철회했다.

      같은 파일 `:456` `inspectorStatus`는 호출자가 0개다(테스트 포함). 함께 정리한다.

      순서: 렌더러와 그 테스트를 같은 커밋에서 지운다. 테스트만 남기면 컴파일이
      깨지고, 렌더러만 남기면 다음 사람이 다시 속는다.

- [ ] `.taskmaster/tasks/tasks.json`의 스텁 중복을 제거한다. (2026-09-08 /plan-ceo-review C13)
      task 6.1~6.8이 두 번 표현되어 있다. task 6의 subtasks에는 실제 내용(감사 리포트
      인용과 완료 기준)이 있고, task 10의 subtasks 9개는 같은 taskId를 빈 껍데기로
      갖는다: `taskId: "6.4"`, `title: "6.4 (내용 미공유)"`,
      `description: "원 우선순위 목록의 의존 그래프에만 등장하고 내용이 공유되지 않았다."`

      실제로 터진 사례: 2026-09-08 아웃사이드 보이스(codex)가 스텁을 먼저 읽고
      "task 6.4에는 완료 기준이 없고 6.8은 author 이메일 잘림을 지적하지 않는다"는
      반대 결론을 냈다. 두 지적 모두 실제 항목을 보면 사실이 아니다. 독립 리뷰가
      빈 껍데기를 근거로 계획을 반박하게 만드는 구조다.

      task 9에도 같은 형태의 스텁이 47개 있다 (T1~T33, C1~C5, E1~E4, L1~L5).

      고치는 법: 하나의 taskId가 한 곳에만 존재하게 한다. 의존 그래프가 필요하면
      스텁을 복제하지 말고 dependencies 필드로 참조한다.
