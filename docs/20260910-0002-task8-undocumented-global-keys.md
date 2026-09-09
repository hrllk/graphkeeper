# Task 8 — 동작하지만 어디에도 적히지 않은 전역 키

상태: **조사 완료 / 결정 대기.** 이 문서는 결정하지 않는다.
기준 커밋: `6800f16`
조사 근거: `handleBrowseGlobalKey` (`internal/app/key_handling_browse.go:57-214`)
대 `globalHotkeyItems()` (`internal/app/hidden_hotkeys.go:40`) 와
`renderMainHotkeyFooter` (같은 파일 `:50`).

---

## 실측

**전역 핸들러가 실제로 답하는 키**

```
ctrl+c  q  1  2  3  4  f  F  P  S  ?  tab  shift+tab
up/k  down/j  left/h  right/l  g  G  H  ctrl+u  ctrl+d
```

**푸터에 적힌 것** (= `globalHotkeyItems()`, 오버레이 Global 목록과 같은 출처)

```
tab  k/j  q  ?  ctrl+u/d
```

**오버레이 Global 이 추가로 적는 것**

```
gg  G
```

## 적히지 않은 채 동작하는 것

| 키 | 하는 일 | 왜 빠졌다고 볼 수 있나 |
|---|---|---|
| `ctrl+c` | 종료 | 터미널 관습. `q` 가 적혀 있으면 충분하다는 주장이 가능 |
| `1` `2` `3` `4` | 섹션 1~4 로 점프 | **네 개. 어디에도 없다.** 패널 제목이 `[1] Graph` 처럼 번호를 보여주므로 화면에 힌트는 있다 |
| `f` | 저장소 상태 새로고침 | 차단 메시지가 "Press f to refresh repository state." 로 **상황이 닥쳤을 때만** 알려준다 |
| `S` | stash 목록 팝업을 연다 | **어디에도 없다.** Local 섹션의 소문자 `s`(stash changes)와 짝을 이루는데 그 관계도 적혀 있지 않다 |
| `shift+tab` | 이전 섹션으로 | `tab` 만 적혀 있다 |
| `left/h` `right/l` | 가로 이동 | `k/j` 만 적혀 있다 |

`H`(jump to HEAD)와 `P`(push)는 전역 핸들러에 있지만 오버레이의 **섹션** 목록에
적혀 있으므로 "어디에도 없다"에 해당하지 않는다. 다만 전역인데 섹션별로 적히는
것은 그 자체로 일관성 문제다.

`F`(tag provenance 동기화)는 Tags 패널이 "Press F to sync tag provenance." 로
그 자리에서 알려준다 — `f` 와 같은 **상황 노출** 방식이다.

## 결정이 필요한 지점

이건 "빠진 키를 채운다"가 아니라 **어디까지 적을 것인가**의 문제다. 셋 중 하나다.

**A. 푸터는 그대로, 오버레이가 전부 적는다.**
`?` 오버레이가 "동작하는 모든 키"의 단일 출처가 된다. 푸터는 지금의 다섯 개를
유지한다. Global 섹션이 5 → 약 15행이 되지만, 6.7 에서 팝업이 콘텐츠 기준으로
크기를 잡게 됐으므로 폭 문제는 없다.

**B. 상황 노출을 규칙으로 삼는다.**
`f` 와 `F` 가 이미 하는 방식 — 그 키가 쓸모 있는 순간에 그 자리에서 알려준다.
이 경우 `1~4` 는 패널 번호가, `shift+tab` 은 `tab` 근처가 담당한다. 목록은
짧게 유지되고, 발견은 사용 시점으로 옮겨간다.

**C. 일부는 관습으로 두고 적지 않는다.**
`ctrl+c`, `left/h`·`right/l` 은 터미널·vim 관습이므로 적지 않는다고 **명시적으로**
결정한다. 지금과 결과는 같지만, "빠뜨린 것"과 "적지 않기로 한 것"이 구별된다.

DESIGN.md 는 "Decoration level: Minimal" 과 "A cell earns its columns" 를 말하고,
6.8 은 오버레이에서 독자가 행동할 수 없는 라벨을 걷어냈다. 그 방향은 A 보다
B·C 쪽에 가깝다. 다만 `1~4` 네 개는 어느 쪽 논리로도 변호하기 어렵다 —
번호가 화면에 있다는 것과 그 번호를 **누를 수 있다**는 것은 다른 정보다.

## 이번에 함께 고친 것

오버레이 Global 목록이 `ctrl+u/d` 를 **두 번** 싣고 있었다. `globalHotkeyItems()`
가 `ctrl + u/d` 로, 6.8 에서 그룹을 합칠 때 덧붙인 항목이 `ctrl+u/d` 로 —
같은 키가 철자만 다르게 두 행을 차지했다. 제거했고,
`TestHotkeySectionsListEachKeyOnce` 가 재발을 막는다.
