# T6 — neutral startup 경로가 stash·tag 를 로드하지 않는다

상태: **진단 완료 / 구현 대기.** 기준 커밋 `c865821`.
근거 항목: task 9.6 (T6). "worked at tag d5257bb, broken on main."

## 증상의 정확한 위치

`Update` 는 startup·refresh 를 **두 벌** 갖고 있다.

| 경로 | 메시지 | stash | tag |
|---|---|---|---|
| legacy | `loadedMsg` / `refreshedMsg` | `loadStashState` 호출 (`update_lifecycle.go:99`, `:139`) | `loadLocalTagStatus` 를 명령 안에서 호출 (`commands.go:31`, `:111`) |
| **neutral** | `loadedSnapshotMsg` / `refreshedSnapshotMsg` | **없음** | **없음** |

`loadedSnapshotMsg` 핸들러(`update_lifecycle.go:41-60`)는 `return m, nil` 로
끝난다. 명령을 하나도 내지 않는다.

**그리고 프로덕션은 neutral 경로를 탄다.** `refreshedMsg` 분기의 첫 줄이
`if m.repositoryRead != nil { return m, nil }` 이고, 프로덕션은
`repositoryRead` 를 주입한다. 즉 legacy 분기의 `loadStashState` 는
**주입이 없는 구성에서만** 실행된다.

`composition_wiring_test.go:153` 과 `repository_read_test.go:243` 이
`tagEntries == nil && stashEntries == nil` 을 단언하고 있다. 두 테스트가
**결함을 계약으로 고정**하고 있는 상태다(그래서 T7/T15 가 따로 있다).

## 무엇을 해야 하나

1. **neutral 경로가 stash·tag 를 배치한다.** `loadedSnapshotMsg` 와
   `refreshedSnapshotMsg` 가 `tea.Batch(..., loadStashState(m.repo))` 를
   반환하고, tag 상태도 같은 방식으로 실린다.
2. **`stashLoadedMsg` 에 epoch 를 배선한다.** 확인함 — `messages.go:20-23` 의
   `stashLoadedMsg` 는 `entries` 와 `err` 뿐이고, 형제 메시지
   (`refreshedMsg`, `refreshedSnapshotMsg`)는 전부 epoch 를 갖는다.
   **매초 로딩이 붙으면 stale 경합이 실사용 빈도로 올라간다** — 지금은 startup
   에서만 도니까 드러나지 않을 뿐이다. (eng-review 결정 7 + E4)

## 비용은 이미 측정됐다

C3(9.36)이 이 질문에 답해 뒀다. tag 500 + stash 50 에서 refresh 1회가
**194.3ms**, tag/stash 없는 저장소가 **184.9ms** — 즉 tag·stash 로딩이 더하는
비용은 **약 9ms(5%)** 이고 호출 수는 규모와 무관하다.
`docs/20260910-0003-c3-refresh-tick-cost.md` 참조. **T6 의 성능 우려는 해소됐다.**

## 순서

`docs/20260910-0005-t6-t7-t13-t15-ordering.md` 를 따른다.
T6 → T7 → T13 → T15. T7 은 T6 없이 착수할 수 없다.

## 착수하지 않은 이유

이 문서를 쓴 시점(2026-09-10 06:5x)에 야간 작업 요약까지 남은 시간이
한 시간 남짓이다. T6 은 startup 과 메시지 수명주기를 건드리고 epoch 배선을
포함하므로, 반쯤 하다 남기는 것보다 진단을 정확히 남기는 편이 낫다.
위 1·2 가 그대로 착수 지시다.
