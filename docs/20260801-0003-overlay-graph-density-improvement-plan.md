# Graphkeeper Overlay·Graph 정보 밀도 개선 계획

## 1. 목표

Hidden Hotkeys overlay와 Graph의 정보 밀도를 조정해 저장소 관리자와 개발자가
메인 화면에서 핵심 조작법과 커밋 정보를 더 빨리 읽도록 한다.

이번 작업의 완료 기준은 다음과 같다.

- `?` overlay가 뒤의 Local·Remote 카드 스타일을 오염시키지 않는다.
- main footer에는 `q: quit`, overlay footer에는 `q: close`만 표시하고, `esc`는
  표시하지 않아도 기존 close/back 동작을 유지한다.
- Hotkey overlay(현재 Hidden Hotkeys overlay)의 최대 폭을 현재 대비 약 30% 줄인다.
- Global 핵심 키는 main 화면 최하단에서 확인할 수 있고, active section 명령은
  `?` Hidden Hotkeys overlay에서 확인하도록 유도한다.
- Graph는 commit hash 5자리, date 제거, author 복원, title 확장을 지원한다.
- Graph와 우측 rail의 하단은 동일한 outer height에 맞아야 한다.
- 좁은 터미널·ANSI/NO_COLOR·빈 데이터·overlay 동시 상태에서도 줄 폭과 키 동작이
  깨지지 않아야 하며, terminal width/height별 fallback과 예외 테스트를 완료한다.

## 2. 현재 구조와 문제 원인

| 요구 | 현재 코드 | 확인된 문제 |
|---|---|---|
| overlay border | `view_layout.go:overlayPopup`, `overlayLine` | base ANSI 스타일의 중간을 자른 뒤 popup을 합성해 우측 base border가 기본색으로 복귀할 수 있다. |
| popup footer | `hidden_hotkeys.go:hiddenHotkeyPopupFooter`, 각 popup renderer | popup마다 footer 문구가 다르고, Hidden Hotkeys는 scroll 조작까지 footer에 노출한다. |
| popup 폭 | `hiddenHotkeys.go:hiddenHotkeyPopupLayout`에서 `popupWidthForBody(..., 44, 72)` | 최대 72열이 본문을 과도하게 가린다. |
| main footer | `view_shell.go:renderAppView` | Graph와 rail만 렌더하고 main 하단의 공통 키 안내가 없다. |
| Global 중복 | `visibleHiddenHotkeySections`, `hiddenHotkeySections` | overlay가 Global + active section을 보여줘 main footer를 도입하면 중복이 된다. |
| Graph 열 | `graph_render_format.go`, `view_graph.go`, `graph_render.go` | hash 7열, date 7열, topology 약 30%, author는 폭 조건에 따라 숨겨진다. |

### ANSI border 현상 분석

`overlayLine`은 base 문자열을 좌측·popup·우측으로 나눠 합성한다. base의 ANSI
foreground가 우측 border까지 이어져야 하지만, popup이 중간에 reset sequence를
출력하면 우측 조각이 현재 스타일 없이 렌더링된다. 이때 Local·Remote card의
border가 터미널 기본색인 흰색처럼 보인다.

수정은 popup 폭이나 색상 자체를 바꾸는 것이 아니라, display-width 기준으로
base를 자를 때 우측 조각에 절단 지점의 활성 SGR 상태를 복원하는 방식으로 한다.
그래야 activeBox magenta, baseBox bright-black, popupBorder magenta가 각각
서로의 색상을 덮어쓰지 않는다.

## 3. 확정에 가까운 UX 계약

### 3.1 Overlay footer와 키 동작

- main footer의 표시 텍스트는 `q: quit`, overlay footer의 표시 텍스트는 `q: close`로
  통일한다. confirm의
  `y/n`, picker의 `enter`처럼 작업을 완료하는 조작은 footer가 아니라 popup body의
  본문 계약으로 유지한다. footer 축소가 실제 작업 키 안내를 삭제한다는 뜻은 아니다.
- `esc`는 footer에 표시하지 않지만, 현재 overlay를 닫거나 현재 단계로
  돌아가는 기존 동작을 유지한다.
- main Browse 화면에서 `q`는 애플리케이션을 종료한다.
- overlay 안에서 `q`는 현재 popup을 close/cancel/back 처리한다. overlay footer는
  실제 의미에 맞춰 `q: close`를 표시한다.
- `q`는 input 문자열에 들어가거나 confirm을 실행하지 않아야 하며, 각 modal의
  기존 `esc` semantics와 같은 close/back 경로를 공유한다.
- 단, Bubble Tea가 보내는 `ctrl+c` 종료 동작은 기존대로 유지한다.
- footer가 없는 loading/blocked 상태는 공통 footer renderer 적용 여부를 별도로
  확인하고, 실제로 footer를 가진 overlay만 문맥에 맞는 q 안내를 표시한다.

### 3.2 Main footer와 Hidden Hotkeys의 source of truth

키 정의를 두 군데에 복사하지 않는다. 다음처럼 공통 registry/helper를 둔다.

```text
hotkey registry
  ├─ globalHotkeyItems()
  ├─ sectionHotkeyItems(activeSection)
  ├─ renderMainHotkeyFooter(width)
  └─ renderHiddenHotkeysPopup(activeSection)  // Global 제외, active만
```

main footer는 한 줄을 목표로 하며 우선순위는 다음과 같다.

```text
Global: tab/shift+tab · j/k · q quit · ?: hotkeys
```

- `Global:`은 고정 핵심 키만 노출한다: section 이동, cursor 이동, 종료, 도움말.
- active section 키는 main footer에 추가하지 않는다. `?: hotkeys`를 상세 명령의
  단일 진입점으로 사용한다.
- Hidden Hotkeys overlay는 Global 섹션을 제거하고 active section만 표시한다.
- overlay를 연 상태에서는 main footer가 popup 뒤에 가려지므로 footer 정보가
  popup 안에서 중복되지 않는다.

### 3.3 Hidden Hotkeys 폭

현재 Hotkey overlay의 `max 72`열을 `max 50`열로 줄인다. 이는 72의 약 70%로
약 30% 축소다.

- wide: 50열
- normal: body width에서 `12`열 여백을 뺀 값과 50 중 작은 값
- narrow: 최소 32열까지 허용하고 그보다 작으면 body width에 맞춘다.
- footer가 단일 q 안내만 가지므로 긴 footer 때문에 content viewport가 불필요하게
  줄지 않는다.

### 3.4 Graph 열 재배치

사용자가 말한 “Graph 30% 축소”는 Graph pane이나 Graph 섹션 전체를 줄이는 것이
아니라 Graph 섹션 안의 `graph` 항목 열을 현재 폭에서 30% 줄이는 것으로
확정한다. Graph pane과 right rail의 비율은 유지하고, topology 최소폭 아래로
내려가지 않을 때만 축소를 제한한다.

| 열 | 현재 | 변경안 |
|---|---:|---:|
| commit | 7 visible chars + padding | 5 visible chars + padding |
| branches | 기존 유지 | 기존 유지 |
| state | 기존 5열 | 기존 유지 |
| graph/topology | Graph content width의 30% | 현재 계산폭의 70% (최소 topology 폭은 유지) |
| date | 7열 | 제거 |
| author | 폭이 좁으면 숨김 | medium 이상에서 7열 복원 |
| title | 남은 폭 | date 제거분 + topology 절감분을 우선 배정 |

폭 우선순위는 `commit → branches → state → topology(min) → author → title`로
고정한다. title은 마지막 남은 폭을 사용하되, author를 복원할 수 없는 폭에서는
author를 다시 생략하고 title을 살린다. hash를 5자리로 줄여도 selection,
search, lookup은 내부 전체 hash를 사용한다.

## 4. 구현 단위

### T1. ANSI-aware overlay compositor

대상: `internal/app/view_layout.go`, `view_overlays_test.go` 및 관련 test helper

- `overlayLine`이 ANSI escape sequence와 wide rune을 함께 처리하도록 보완한다.
- base의 좌측 조각과 우측 조각 모두 기존 foreground/background 상태를 보존한다.
- popup은 불투명 영역으로 처리해 뒤 카드의 문자가 보이지 않게 한다.
- Local·Remote active/inactive border, popup border, TrueColor/ANSI/Ascii/NO_COLOR
  조합을 검증한다.

### T2. 공통 popup footer와 종료 키 계약

대상: `view_shell.go`, popup renderer, `key_handling.go`, 관련 tests

- 공통 `renderPopupFooter()`를 추가하고 main footer는 `q: quit`, overlay footer는
  `q: close`로 표시한다.
- footer를 가진 confirm/review/picker/input/list/search popup을 모두 전수 확인한다.
- `esc`는 표시하지 않아도 기존 close/back/cancel semantics를 보존한다.
- main Browse 상태의 q 종료와 각 overlay 상태의 q close/cancel 테스트를 추가한다.

### T3. Main footer와 section hotkey registry

대상: `hidden_hotkeys.go`, `view_shell.go`, `view_layout.go`, `key_handling_browse.go`,
tests, README

- Global 핵심 키와 section 핵심 키를 shared data/helper로 분리한다.
- main body의 outer height에서 footer 한 줄을 예약하고 Graph·rail height를 동일하게
  유지한다.
- footer는 shell의 최하단에 Global 핵심만 렌더하며 terminal width 부족 시
  deterministic truncation을 적용한다.
- Hidden Hotkeys는 active section만 보여주고 Global 중복을 제거한다.
- `?` 입력 후 active section의 전체 명령을 overlay에서 확인할 수 있는지 검증한다.

### T4. Hidden Hotkeys 폭 축소

대상: `hidden_hotkeys.go`, `hidden_hotkeys_test.go`

- popup width constant를 50/32 정책으로 변경한다.
- Hotkey overlay를 80/100/140열과 12/20/40행에서 검증하고 popup width, viewport,
  title, footer를 확인한다.
- content가 폭을 넘지 않고 scroll offset이 마지막 줄까지 접근하는지 검증한다.

### T5. Graph 정보 밀도 조정

대상: `graph_render_format.go`, `graph_render.go`, `view_graph.go`, graph tests

- hash visible width를 5자리로 변경하되 full hash 내부 동작은 유지한다.
- date header/row를 제거하고 fixed-width 계산을 갱신한다.
- Graph 섹션 안의 `graph` 항목 열을 현재 계산폭의 70%로 낮추고 최소폭을 보장한다.
- author를 medium 이상에서 복원하고 narrow에서는 title 우선 fallback을 유지한다.
- normal/raw graph, connector, search highlight, selection, stash/tag state 색상과
  column alignment를 함께 검증한다.

### T6. 통합 QA·문서·Task Master 동기화

대상: README, `docs/decisions.md`, 관련 tests, `.taskmaster/tasks/tasks.json`

- 키가 main footer와 overlay에서 중복·누락되지 않는지 문서화한다.
- ANSI/NO_COLOR 및 terminal width/height matrix를 반드시 실행한다.
- `go test ./...`, `go test -race ./internal/app`, `scripts/check`, `scripts/build`
  를 실행한다.
- T1~T5를 Task Master subtask 단위로 상태 갱신한다.

## 5. 테스트 매트릭스

| 영역 | 정상 | 경계/예외 | 검증 |
|---|---|---|---|
| overlay 합성 | popup 중앙 표시 | base ANSI border, popup reset, wide rune, 좁은 폭 | unit + snapshot-like width/color |
| footer | main은 q quit, overlay는 q close | q 문맥 분기, esc close, confirm/input/list 상태 | key handling tests |
| main footer | Global 핵심 키 | 80/100/140열 truncation, empty repo | renderer tests |
| hidden overlay | active section만 표시 | Global 미표시, 32/50열, 12행, scroll max | hidden_hotkeys tests |
| Graph columns | normal/raw row | 5-char hash collision-looking values, empty author/title, search, selected | graph renderer tests |
| color | ANSI/TrueColor/Ascii/NO_COLOR | popup 뒤 rail border 유지, stash/tag/handshake | color matrix |
| layout | Graph/rail bottom align | terminal resize, low height, overlay open | shell layout tests |

## 6. 실패 모드와 복구

| 실패 | 사용자 증상 | 방어 |
|---|---|---|
| SGR 상태 손실 | Local/Remote border가 흰색 | overlay compositor ANSI state test |
| footer가 body를 밀어냄 | Graph/Tags bottom 불일치 | footer 예약 높이와 outer-height invariant |
| q 문맥이 뒤섞임 | main q가 종료되지 않거나 overlay q가 입력됨 | Browse/overlay별 q 분기 test |
| Global 키가 양쪽에 중복 | 화면 밀도 증가 | registry 기반 overlay/main 분리 test |
| popup 폭 축소로 줄 잘림 | hotkey 설명 손실 | visible-width truncation + scroll test |
| author 복원으로 title 소실 | commit subject 가독성 하락 | author threshold/title priority matrix |
| 5자리 hash 혼동 | 사용자가 다른 commit으로 오인 | full hash detail/selection/search 보존, 5자리 충돌 테스트 |

## 7. NOT in scope

- Graph 데이터 조회 방식, Git adapter, repository snapshot 계약 변경
- 새로운 hotkey 추가나 기존 hotkey 의미 변경
- Graph topology 알고리즘 변경
- 전체 테마 색상 재설계
- footer에 모든 section의 전체 hotkey를 상시 노출하는 dashboard화

## 8. 이미 존재하는 기능 활용

- `shellOverlayStack`와 `overlayPopup`을 overlay 합성 진입점으로 재사용한다.
- `hiddenHotkeySections`의 section별 action 정의를 registry 전환의 출발점으로 삼는다.
- `renderGraphHeader`, `graphRowFixedWidth`, `renderGraphTitleWithAuthor`의 폭 계약을
  하나의 column budget helper로 묶는다.
- 기존 `withColorProfile`, `ansi.Strip`, `lipgloss.Width` 기반 테스트를 확장한다.

## 9. 아키텍처 흐름

```text
key input
  ├─ main q ──────────▶ tea.Quit
  ├─ overlay q ───────▶ close/cancel/back
  ├─ esc ─────────────▶ active overlay close/back
  └─ ? ───────────────▶ hiddenHotkeysOpen

hotkey registry ──────┬─▶ main footer
                      └─▶ active-section overlay

renderAppView
  ├─ renderGraph + right rail
  ├─ renderMainHotkeyFooter
  └─ applyShellOverlays
       └─ overlayPopup
            └─ ANSI-preserving overlayLine

graph column budget
  └─ hash/branches/state/topology/author/title
```

## 10. 검토 결과

### CEO 검토

문제는 “더 많은 키를 보여주기”가 아니라 main 화면을 읽는 동안 필요한 키만
항상 보이고, 전체 목록은 `?`에서 확인되게 하는 것이다. Global 제거와 footer
추가는 서로 충돌하지 않는다. footer는 핵심 navigation affordance이고 overlay는
active section의 전체 action reference다.

10x 관점에서는 footer와 overlay가 서로 다른 키 목록을 갖지 않는 것이 가장
중요하다. Graph pane 전체를 축소하는 안은 topology를 훼손하므로 채택하지 않고,
내부 topology 열만 축소한다.

### Design 검토 점수

| 차원 | 점수 | 판단 |
|---|---:|---|
| 정보 계층 | 9/10 | main footer는 방향을 주고 overlay는 상세를 제공한다. |
| 상태 범위 | 8/10 | resize/empty/overlay를 계약화한다. |
| 사용자 흐름 | 9/10 | q 종료와 esc 복귀가 예측 가능하다. |
| 시각 일관성 | 8/10 | ANSI state 보존으로 카드 색상 깜빡임을 제거한다. |
| 반응형 폭 | 9/10 | footer/popup/Graph에 동일한 display-width 규칙을 적용한다. |
| 접근성 | 8/10 | 색상만이 아니라 텍스트 S/T와 key label을 유지한다. |
| 구현 명확성 | 9/10 | 폭 상수, 우선순위, fallback을 명시했다. |

Taste decision: main footer에는 Global 핵심만 넣고 active section 명령은 `?` overlay로
유도하는 안을 채택한다. 전체 action을 footer에 상시 노출하면 새로운 Actions panel이
되어 main 화면의 정보 밀도가 다시 높아지므로, 상세 명령은 overlay에서 제공한다.

### Eng 검토

- ANSI 문제는 색상값 변경으로 해결하지 않고 합성기의 상태 보존으로 해결한다.
- body height는 footer 예약 후 계산하며 Graph와 rail은 같은 outer height를 계속
  사용한다.
- q dispatch는 모든 modal state보다 앞서야 footer와 동작이 일치한다.
- Graph 열 폭 계산은 한 helper에서 수행해 header와 row가 어긋나지 않게 한다.
- full hash는 내부 모델과 Details에 남겨 5자리 표시가 데이터 식별자를 바꾸지
  않게 한다.

### DX 검토

처음 실행한 개발자는 main footer에서 `tab`, `j/k`, `q`, `?`를 즉시 발견하고,
상세 명령은 `?`로 들어가며
Global 중복 없이 현재 section에 집중한다. README의 Keyboard 섹션은 footer와
overlay 정책을 반영해야 한다.

### Codex 독립 검토 결과

Codex는 실제 `internal/app` 코드와 테스트를 read-only로 대조했다. Claude
subagent 도구는 현재 세션에 노출되지 않아 unavailable이다.

| 심각도 | 근거 | 계획 반영 |
|---|---|---|
| High | `view_layout.go:462-511`의 `overlayLine`이 우측 base 조각에 활성 SGR을 복원하지 않고 escape를 `m` 기준으로만 처리한다. | T1에 SGR 상태 추적, wide rune 원자 경계, ANSI/Ascii/NO_COLOR 테스트를 명시 |
| High | `key_handling.go:9-47`에서 modal dispatch가 q보다 먼저 실행된다. | T2에서 main Browse q는 종료하고 modal q는 기존 esc close/cancel 경로로 처리 |
| High | `view_shell.go:47-82`에 main footer와 body height 예약이 없다. | T3에서 footer 1행 예약 후 Graph/rail outer height 재계산 |
| High | `hidden_hotkeys.go:52-54,134-149`의 폭 계산이 scroll/layout에 중복되어 있다. | T4에서 단일 width-policy helper를 모든 경로에 적용 |
| High | `graph_render.go:31,95,60-63,134-141`과 `graph_render_format.go:21-23`에 hash/date 고정 폭이 여러 경로로 흩어져 있다. | T5에서 shared column budget으로 header/normal/raw/placeholder/connector를 함께 갱신 |
| Medium | 기존 `hidden_hotkeys_test.go`가 Global과 `esc: close`를 기대한다. | T6에서 active-only, overlay q close, width/resize/final-scroll 기대값으로 교체 |

Codex의 T1~T6 판정은 모두 `PARTIAL`이다. 현재 커밋에는 계획 대상 구현이
아직 없고, 기존 코드와 테스트가 변경 전 계약을 보여주기 때문이다.

## 11. Decision Audit Trail

| # | Phase | Decision | Classification | Principle | Rationale | Rejected |
|---:|---|---|---|---|---|---|
| 1 | CEO | Global은 overlay에서 제거하고 main footer에는 Global 핵심만 둔다 | User-confirmed | User direction | active 명령은 `?`로 유도해 main 밀도를 낮춘다 | active section 키를 footer에 추가 |
| 2 | CEO | Graph pane가 아니라 Graph 내부 graph 항목 열을 30% 줄인다 | User-confirmed | User direction | Graph/rail 균형을 유지하면서 title/author 공간을 확보한다 | Graph pane 전체 축소 |
| 3 | Design | footer는 Global 핵심만 표시하고 active section은 `?`로 유도한다 | User-confirmed | User direction | 상시 Actions panel 재도입을 막는다 | active section 핵심 키 상시 표시 |
| 4 | Eng | main q는 종료, overlay q는 close/cancel로 분기한다 | User-confirmed | User direction | 같은 키의 문맥별 의미를 실제 동작과 맞춘다 | 모든 상태에서 q 종료 |
| 5 | Eng | overlayLine에서 ANSI state를 복원한다 | Mechanical | P5/DRY | 색상 테마 변경 없이 원인에 직접 대응한다 | popup border 색상 변경 |
| 6 | Eng | hash 표시만 5자리로 줄이고 내부 full hash는 유지한다 | Mechanical | data safety | 식별·검색·selection 계약을 보존한다 | 모델 hash truncate |
| 7 | DX | README와 main footer를 같은 registry 정책으로 동기화한다 | Mechanical | P4 | 문서·화면의 키 불일치를 막는다 | 문서만 수동 수정 |

## 12. 구현 서브태스크 제안

Task Master의 `feature` task 아래에 다음 단위로 추가한다.

1. **ANSI-preserving overlay compositor**: overlay 합성 시 Local·Remote border 색상
   보존 및 색상/폭 회귀 테스트
2. **공통 overlay footer와 q/esc 키 계약**: main은 `q: quit`, overlay는 `q: close`,
   esc close/back 유지
3. **Main hotkey footer와 shared registry**: Global 핵심 키만 main 최하단에 표시하고
   active section 명령은 overlay에서 확인하도록 유도
4. **Hidden Hotkeys 폭 축소**: max 50/min 32 정책과 low-height/scroll/width 테스트
5. **Graph 정보 열 재배치**: hash 5자리, date 제거, graph 항목 70%, author/title budget
6. **통합 회귀·문서·Task Master 상태 동기화**: 전체 테스트와 README/decisions 갱신

## 13. 구현 전 최종 확인이 필요한 항목

- main에서는 q가 종료되고 overlay에서는 q가 close/cancel된다는 계약
- Graph 섹션 안의 `graph` 항목 열을 현재 폭의 70%로 조정한다는 계약
- main footer는 Global 핵심만, active section 키는 `?` overlay로 유도한다는 계약

## 14. GSTACK REVIEW REPORT

CEO: main footer와 active-only overlay의 조합으로 discoverability와 정보 밀도를
함께 개선한다. Graph 전체 pane 축소는 topology 손실 위험이 있어 제외했다.

Design: footer, popup, Graph columns의 우선순위와 좁은 폭 fallback을 명시했다.
ANSI 색상은 장식이 아니라 상태 정보이므로 compositor가 보존해야 한다.

Eng: overlay ANSI 상태, modal q dispatch, footer height reservation, graph column
budget, full hash 보존을 핵심 회귀 계약으로 고정했다.

DX: 첫 화면에서 global navigation과 active action을 찾을 수 있고, `?`는 중복 없는
현재 section reference로 남는다.

Codex dual voice: 실제 코드 read-only 검토를 완료했으며 위의 six findings를
계획과 테스트 계약에 반영했다. Claude subagent voice는 unavailable이다.

VERDICT: 구현 가능한 계획이며 사용자 승인 완료 상태다. Task Master의 feature
task 아래에 T1~T6 단위 서브태스크를 pending으로 등록했으며, 다음 별도 구현
명령에서 의존성 순서대로 진행한다.
