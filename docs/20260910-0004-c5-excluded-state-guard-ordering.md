# C5 — "excluded tag/stash state" 가드를 언제 바꾸고 언제 지우나

상태: **정리 완료.** 기준 커밋 `2c235d6`.
근거 항목: task 9.38 (C5).

## 무엇이 어긋나 보였나

세 곳이 같은 두 단언에 대해 서로 다른 말을 한다.

| 출처 | 말하는 것 |
|---|---|
| 설계 문서 Success Criteria | 두 단언은 **삭제**된다. 반전이 아니다 |
| T7 (9.7, eng-review 로 재정의됨) | 반전이 아니라 **메시지 계약 단언으로 교체**한다 |
| T15 (9.15) | 이중 경로가 사라진 **뒤에** 삭제한다 |

대상은 두 곳이다.

- `internal/app/composition_wiring_test.go:153`
  — `got.tagEntries != nil || got.stashEntries != nil || got.tagSyncAttempted`
- `internal/app/repository_read_test.go:243`
  — `got.tagEntries != nil || got.tagSyncAttempted || got.stashEntries != nil`

## 실은 모순이 아니라 시점이 다르다

셋은 같은 하나의 순서를 각자 다른 지점에서 기술한다.

```
T7    지금.  이중 경로가 아직 있다  → 단언을 메시지 계약으로 바꾼다
T13   그 다음.                      → startup 이중 경로를 제거한다
T15   그 뒤.  가드할 대상이 없다     → 단언을 삭제한다
Success Criteria = T15 이후의 최종 상태
```

**Success Criteria 는 완료 상태를 적은 것이지 T7 의 지시가 아니다.**

## 이 문서가 막는 실수

C5 가 지목한 위험은 구체적이다 — **T7 을 하러 온 사람이 Success Criteria 를 먼저
읽고 가드를 지금 지워 버리는 것.** 그러면 T13 이 아직 남긴 이중 경로를 아무것도
지키지 않는 구간이 생긴다.

**T7 에서 지우지 않는다. T13 이 끝나기 전까지 가드는 남는다.**

## T7 이 반전이 아닌 이유 (eng-review 결정 4 재확인)

`loadStashState` 는 `tea.Cmd` 다. `Update(loadedSnapshotMsg)` 직후에는 명령이
아직 실행되지 않았으므로 `stashEntries` 는 **정상 동작에서도 nil** 이다.
단언을 문자 그대로 반전하면 올바른 코드에 대해 실패하는 테스트가 된다.
게다가 `composition_wiring_test.go:146` 의 `cmd != nil` 가드가 :153 보다 먼저
깨진다. 따라서 T7 은 "상태가 채워졌는가"가 아니라 **"올바른 메시지가 나갔는가"**
를 단언해야 한다.

## 곁다리: 이미 충족된 기준 하나

Success Criteria 의 다음 항목은 **2026-09-10 에 충족됐다.**

> `renderCommitInspectorPopup` 과 `renderInspectorBody` 는 삭제된다.
> Inspector 렌더러는 정확히 하나다.

task 12.1 과 그 마무리에서 둘 다 지웠고, 그 아래 고아가 된 diff 경로
(`commitInspectorUnifiedLines` 등)까지 함께 정리했다. 커밋 `0c69508`, `b376781`,
`2c235d6`.
