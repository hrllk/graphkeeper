<!-- /autoplan restore point: /private/tmp/graphkeeper-autoplan-restore-20260801-0002.md -->

# Graphkeeper 메인 화면 레이아웃 재조정 계획

상태: CEO 전제 확인 완료, Design/Eng/DX 리뷰 진행 중
브랜치: `develop`
작성일: 2026-08-01

## 1. 요청사항과 목표

이번 작업은 기존 Graph 중심 화면의 정보 구조를 단순화하고, Graph를 더 넓고
높게 읽을 수 있게 만드는 UI 재배치다.

사용자 요청:

1. 전체 레이아웃 재조정
2. 현재 상단의 `Context` 섹션 프레임 삭제
3. 기존 `Context Details` 정보만 상단 우측의 독립 패널로 이동
   - Local/Remote/Tags와 같은 title + body 레이아웃을 사용
   - Context Actions는 이 패널에 포함하지 않음
4. 상단 높이를 줄이고 Graph 섹션 높이를 확대
5. Global/section 단축키는 메인 화면에 상시 나열하지 않고 `?` Hidden Hotkeys
   오버레이에서 확인하도록 정리
6. Graph 제목이 더 많이 보이도록 폭 우선순위를 재조정
7. stash/tag가 있는 Graph point의 색상 누락 경로를 찾아 모든 렌더 경로에서
   동일하게 보이도록 수정

### 확정된 최종 화면 구조

```text
+------------------------------+-----------+
| Graph                        | Details   |
|                              +-----------+
|                              | Local     |
|                              +-----------+
|                              | Remote    |
|                              +-----------+
|                              | Tags      |
+------------------------------+-----------+
```

- `Global` 영역과 기존 `Context` 프레임, `Context Actions` 열은 모두 제거한다.
- 우측 상단 패널은 현재 선택 섹션에 따라 `Graph Details`,
  `Local Details`, `Remote Details`, `Tags Details`로 바꾼다.
- 기존 Global/section Actions 목록은 메인 화면에서 제거한다.
- 단축키는 `?` Hidden Hotkeys overlay에서만 제공한다. overlay에는
  `Global` 공통 키와 현재 active section 키만 표시한다.
- Graph와 우측 rail의 outer bottom border는 계속 같은 행에 맞춘다.

## 2. CEO 전제 확인

### 전제 A: Global과 Context Actions를 없애도 정보 손실은 없다

현재 상단에는 Global과 Context가 있고 Context는 Details와 Actions를 함께 표시한다.
이번 계획은 Global과 Context Actions를 모두 삭제하고 Details만 우측 상단
독립 패널로 옮긴다. Actions는 Hidden Hotkeys로 이동한다.
따라서 선택 커밋/브랜치/태그의 정보는 유지되지만, Context Actions를 메인 화면에서
항상 보는 흐름은 바뀐다.

판정: 사용자 확정. `?` 오버레이가 단축키 discoverability를 책임진다. 첫 실행
사용자가 이를 알 수 있도록 빈 화면이나 Details footer에 상시 한 줄 안내를 둘지
Design 리뷰에서 결정한다.

### 전제 B: Graph 높이 확대가 Context 정보보다 가치가 높다

Graph는 여러 commit row와 connector를 동시에 읽어야 하므로 세로 공간의
marginal value가 높다. Details는 선택된 항목 한 건의 요약이므로 컴팩트한 상단
패널로 축소해도 된다.

판정: 합리적이다. 다만 작은 터미널에서 Details가 제목만 남거나 Graph가
최소 높이를 침범하지 않도록 높이 경계를 명시해야 한다.

### 전제 C: Graph 제목을 위해 author를 먼저 줄여도 된다

현재 `renderGraphTitleWithAuthor`는 author 폭을 예약한 뒤 남은 폭을 title에
할당한다. 릴리즈/저장소 관리자 관점에서는 commit subject가 author보다 먼저
읽혀야 한다.

판정: 합리적이다. 넓은 터미널에서는 author를 유지하고, 좁은 터미널에서는
author를 숨기거나 1열로 축약하는 적응형 정책을 사용한다.

### 전제 D: 색상 누락은 데이터 누락이 아니라 렌더 경로 불일치다

stash/tag 상태 데이터는 이미 `stashesForCommit`과 `row.Commit.Tags`로
Graph row에 전달된다. normal row, raw row, connector/spacer, focus/handshake
상태가 서로 다른 색상 적용 규칙을 갖고 있어 누락이 생길 가능성이 높다.

판정: 구현 전 모든 Graph 렌더 경로를 상태 매핑표로 고정한다. 데이터 모델이나
Git 조회를 새로 만들지 않는다.

## 3. 기존 코드 레버리지

| 요구사항 | 기존 코드 | 변경 방향 |
|---|---|---|
| 상단/하단 재배치 | `internal/app/view_shell.go:40-111` | 상단을 Global + Details로 재구성하고 Graph rail 높이를 재계산 |
| Shell 높이 | `view_layout.go:16-82` | 상단 compact height와 Graph minimum/remaining height 계약 추가 |
| Details 이동 | `view_detail.go:12-160` | `renderContextInfoLines`를 Details panel renderer로 추출 |
| Context Actions 제거 | `view_detail.go:24-41`, `renderActionHelpLines` | 메인 shell에서 호출 제거, Hidden Hotkeys만 source of truth로 유지 |
| Hidden Hotkeys | `hidden_hotkeys.go:95-290` | Global 중복 제거 정책과 section mapping 유지·확장 |
| Graph 제목 | `graph_render.go:145-164`, `graph_render_format.go` | title-priority budget helper로 author/title 우선순위 변경 |
| Graph 색상 | `graph_render.go:166-269` | normal/raw/connector/focus 상태별 marker color contract 통합 |
| Right rail | `view_shell.go`의 `renderRightRail` | Details와 동일한 title/body frame 계약 재사용 |
| 테스트 | `internal/app/model_test.go`, `hidden_hotkeys_test.go` | layout snapshot-like line/width/color matrix 추가 |

## 4. 범위

### 포함

- Global 영역 제거
- Context 프레임과 Context Actions 제거
- 기존 Context Details를 상단 우측 `<Section> Details` 패널로 이동
- Details 패널의 active section별 내용, empty/loading/error, overflow 표시
- Graph rail 높이 확대를 위한 shell height budget 재배분
- Graph와 Local/Remote/Tags rail의 outer height 및 bottom border 정렬 유지
- `?` Hidden Hotkeys를 메인 단축키 안내의 단일 진입점으로 정리
- Graph title 폭 우선 정책 변경
- normal/raw graph row, connector, focus/handshake/conflict 조합의 stash/tag
  point 색상 계약과 회귀 테스트
- ANSI/NO_COLOR, narrow width/height, Unicode marker, paging/scroll 테스트
- `DESIGN.md`, `README.md`, `docs/decisions.md` 동기화

### 제외

- Graph topology 모델 교체
- Git 조회 방식, stash/tag persistence, repository state 계약 변경
- 기존 단축키 의미나 실행 순서 변경
- Context Details의 정보 내용 자체를 새로 설계하는 작업
- 별도 영구 Hotkey 패널 또는 onboarding 추가
- 터미널 외 web preview의 레이아웃 변경
- 색상 테마 설정 기능 추가

## 5. 핵심 레이아웃 계약

### 5.1 Outer height

```text
bodyHeight = terminalHeight - outer margins
graphHeight = bodyHeight
railHeight = bodyHeight

main row:
  Graph frame == graphHeight
  Right rail == railHeight
    Details + Local + Remote + Tags + internal separators == railHeight
```

권장 초기값:

- body height 12 미만: 기존 minimum을 유지하되 overflow 우선
- body height 20 미만: four right-rail cards에 최소 1행씩 배분하고 Graph를 우선
- body height 20 이상: Details/Local/Remote/Tags에 균등한 rail height 배분
- Graph content는 frame border/title/page/header를 차감한 뒤 계산
- Details가 최소 3 body rows를 확보하지 못하면 제목 + `… +N` 우선
- Graph는 가능한 높이를 모두 사용하되 최소 5 content rows를 보호

고정 숫자는 helper 하나에 두고, 모든 caller가 같은 outer-height 계약을 사용한다.

### 5.2 Top-right Details

- `renderContextInfoLines`의 정보 생성 로직을 재사용한다.
- shell에서는 `renderDetailsPanel`만 호출한다.
- title은 `Graph Details`, `Local Details`, `Remote Details`,
  `Tags Details` 중 하나다.
- Actions column은 제거한다.
- 정보가 없으면 `(empty)`, 로딩이면 기존 loading 상태, 오류면 기존 오류 상태를
  유지한다.
- 정보가 height를 넘으면 `… +N`을 표시하고 `ctrl+u/d`로 이동한다.
- section/focus 변경 시 Details scroll offset을 0으로 초기화한다.

### 5.3 Main Hotkey

- Global/section Actions를 메인 화면에 렌더하지 않는다.
- `?`를 누르면 Global 공통 항목 + active section 항목만 표시한다.
- `f/F/S`처럼 active section에 이미 설명되는 항목은 Global에서 반복하지 않는다.
- popup 내부의 `esc`, `?`, scroll 키 동작은 유지한다.
- `?` 오버레이가 닫힌 뒤 active section/cursor/scroll 상태는 변하지 않는다.

### 5.4 Graph title budget

우선순위는 다음과 같다.

1. commit hash, branches, state, topology, date
2. title subject
3. author

정책:

- wide: subject를 최대 폭으로 사용하고 author를 남은 폭에 배치
- medium: author를 축약하거나 제거하고 subject를 확장
- narrow: subject 최소 폭 12와 `…`를 보호하고 author를 제거
- title이 최소 폭도 확보할 수 없으면 title은 `…` 하나만 남기되 topology와
  state를 삭제하지 않는다.
- header와 row가 동일한 budget helper를 사용한다.
- raw/normal row의 fixed prefix는 동일해야 한다.

### 5.5 Graph point color contract

| 상태 | Graph point | State column | 범례 |
|---|---|---|---|
| none | 기본 topology 색상 | blank | 없음 |
| stash | ANSI yellow | `S` | `S stash` |
| tag | ANSI magenta | `T` | `T tag` |
| stash + tag | ANSI overlap/결정된 조합색 | `S·T` | 행에서는 조합을 직접 표시 |
| HEAD/focus only | 기존 HEAD/focus 우선순위 | blank | 없음 |
| conflict | conflict red가 최우선 | blank | 없음 |

상태 색상은 normal row, raw graph row, 그에 연결된 point render에서 같아야 한다.
connector 선은 point 상태를 상속하지 않으며, node point만 상태 색을 사용한다.
동시에 handshake/focus가 활성화되면 기존 interaction 표시가 의미를 잃지 않도록
우선순위를 테스트로 고정한다.

## 6. 구현 단계

- [ ] T1. layout budget을 Global + Details / Graph + rail 구조로 재구성
- [ ] T2. Context Details renderer를 상단 우측 panel로 이동하고 Actions 제거
- [ ] T3. Graph title priority budget 및 header/row 정렬 수정
- [ ] T4. Graph point stash/tag 색상 누락 경로 수정
- [ ] T5. Hidden Hotkeys를 main hotkey source of truth로 정리
- [ ] T6. narrow/height/ANSI/NO_COLOR/paging 회귀 테스트와 문서 동기화

## 7. 테스트 계획 초안

| 영역 | 케이스 |
|---|---|
| layout | body height 12/16/20/24/40, width 80/100/140, top/bottom height 합계 |
| borders | Graph bottom과 Tags bottom의 실제 line index 일치 |
| Details | Graph/Local/Remote/Tags/empty/loading/error, overflow, ctrl+u/d |
| hotkeys | Global은 상태만 표시, `?`는 active section만, popup close 후 상태 보존 |
| title | wide/medium/narrow, long author, long subject, Unicode subject |
| graph color | normal/raw, stash/tag/both, HEAD/focus/handshake/conflict 조합 |
| topology | `*`, `|`, `/`, `\\` 보존, connector와 status column 정렬 |
| color profiles | Ascii, ANSI, ANSI256, TrueColor, `NO_COLOR=1` |
| regression | `go test ./...`, `go test -race ./internal/app`, `scripts/check`, build |

## 8. CEO/Design/Eng/DX 리뷰 결과

이 섹션은 autoplan 각 phase 완료 시 실제 검토 결과로 채운다.

## 9. 디자이너 관점 우려사항과 대응 초안

| 우려 | 영향 | 계획상 대응 |
|---|---|---|
| Global 삭제로 처음 보는 사용자가 단축키를 모를 수 있음 | 초기 discoverability 저하 | `?`를 primary help entrypoint로 유지하고 Graph/Details의 빈 공간에 `?: show hotkeys`를 최소 안내로 둘지 Design 리뷰에서 확정 |
| 우측 4개 panel이 낮아져 Details/Tags 내용이 잘릴 수 있음 | 상태 정보가 제목만 남음 | 각 panel은 outer height를 동일하게 유지하되 body에 `… +N`, section별 scroll, minimum title/body 우선순위를 적용 |
| Graph가 넓어지며 우측 rail이 너무 좁아질 수 있음 | branch/tag label 가독성 저하 | Graph/rail 최소 폭 계약과 80/100/140 column matrix 테스트 |
| Graph point 색상만으로 의미가 전달되지 않을 수 있음 | ANSI/NO_COLOR 또는 색약 환경에서 stash/tag 식별 실패 | 기존 `state` 컬럼의 S/T/S·T를 source of truth로 유지하고 point 색상은 보조 표현 |
| Details 제목이 active section 변화 때 자주 바뀔 수 있음 | 위치 인식 비용 증가 | title 형식을 고정하고 active section marker만 색상/텍스트로 구분 |
| Hotkey overlay가 긴 목록으로 커질 수 있음 | 현재 작업 흐름을 가림 | Global + active section만 렌더하고 popup 자체 paging/scroll 유지 |

## 10. NOT in scope

리뷰 완료 후 확정한다.

## 11. Decision Audit Trail

| # | Phase | Decision | Classification | Principle | Rationale | Rejected |
|---|---|---|---|---|---|---|

## 12. GSTACK REVIEW REPORT

리뷰 완료 후 작성한다.

## 13. CEO 리뷰

### 13.1 핵심 판단

이번 작업의 실제 목표는 패널을 줄이는 것이 아니라 Graph topology를 읽는 시간을
늘리고, 선택 항목의 세부 정보와 조작법을 필요할 때만 꺼내는 것이다. Global과
Context Actions를 제거하는 것은 이 목표에 맞지만, `?`가 유일한 도움말 진입점이
되므로 overlay의 첫 화면과 닫기/스크롤 규칙이 제품 계약이 된다.

아무것도 하지 않으면 Graph는 현재보다 더 많은 row를 보여주지 못하고, Context의
Actions와 Details가 같은 공간을 경쟁한다. 반대로 우측 rail을 네 패널로 쪼개면
각 패널의 body가 낮아져 정보가 제목만 남을 수 있다. 따라서 성공 조건은 단순히
Graph가 커지는 것이 아니라 `Graph height 증가 + Details/rail 최소 가독성 유지`다.

### 13.2 10x 방향과 현실적 범위

12개월 이상적인 Graphkeeper는 Graph를 읽는 동안 선택한 commit의 provenance와
다음 행동을 한 번에 파악하고, 도움말은 작업 흐름을 가리지 않는 command palette
형태로 필요할 때만 호출하는 TUI다. 이번 계획은 그 방향으로 가는 첫 단계이며,
새 command palette나 renderer projection 전체 분리는 범위에 넣지 않는다.

### 13.3 구현 접근 대안

| 접근 | 설명 | 노력/위험 | 판단 |
|---|---|---|---|
| A. 기존 renderer 재배치 | `renderAppView`, Details renderer, right rail을 확장하고 기존 model을 재사용 | M / 중간 | **추천** |
| B. slot 기반 layout primitive | Graph/Details/rail을 slot과 공통 metric으로 추상화한 뒤 shell을 조합 | M-L / 중간 | 후속 TUI layout 기반으로 유용하지만 이번 변경에는 과함 |
| C. screen projection 전면 분리 | read model과 renderer를 분리하고 새 screen model을 도입 | L-XL / 높음 | 현재 사용자 목표보다 구조 변경이 큼 |

A를 선택한다. 이번 변경은 7개 내외 renderer/layout 파일의 책임 조정으로 해결할
수 있고, B/C는 현재 architecture migration T4와 충돌할 수 있다. 다만 A에서도
height/width metric과 Details projection은 작은 helper로 명시해 다음 구조 개선이
쉽게 이어지게 한다.

### 13.4 우려사항 피드백

| 질문 | 결론 |
|---|---|
| Global이 사라지면 hotkey는 어떻게 보나? | `?` Hidden Hotkeys overlay가 유일한 상세 목록이다. Global 공통 키와 active section 키만 표시하고, section 전환 후 overlay를 다시 열면 새 목록을 표시한다. |
| `?`를 모르는 첫 사용자는? | 메인 화면에 Actions 목록은 두지 않는다. 대신 Graph page info line에 폭이 허용될 때만 `?: hotkeys`라는 짧은 affordance를 표시하는 것을 권장한다. 좁은 터미널에서는 생략한다. |
| 우측 패널이 너무 낮아지면? | Details/Local/Remote/Tags는 같은 outer height budget을 사용하되, 제목은 항상 보존하고 body는 `… +N`과 기존 scroll로 접근한다. 단일 패널이 전체 rail을 독점하지 않는다. |
| Graph가 넓어지면 branch/tag 정보가 잘리지 않나? | Graph/rail 최소 폭 계약을 둔다. 80열에서는 title보다 hash/branch/state/topology를 우선하고, rail이 최소 폭 미만이면 rail label을 축약한다. |
| 색상이 빠진 point는 어떻게 보완하나? | `S/T/S·T` state column을 의미의 원본으로 유지하고 point 색상은 같은 상태 계산을 공유하는 보조 표현으로 고친다. `NO_COLOR`에서도 상태 텍스트는 남는다. |

### 13.5 CEO 결론

범위는 사용자 확정 구조를 유지한다. 추가로 영구 Actions panel, command palette,
새 Git 데이터 조회를 넣지 않는다. `?: hotkeys` 한 줄 hint는 discoverability와
subtraction 사이의 경계에 있는 디자인 선택이므로 최종 승인에서 확인한다.

## 14. Design 리뷰

### 14.1 디자인 기준

`DESIGN.md`의 industrial/utilitarian, compact, information-dense 방향에 맞춰
검토했다. 이 화면은 marketing page가 아니라 APP UI이므로 장식보다 hierarchy,
calm surface, minimal chrome을 우선한다. 기존 rounded frame과 ANSI semantic
color를 재사용하고 새 카드 장식이나 gradient를 추가하지 않는다.

초안 디자인 완성도는 7/10이다. 구조와 정보 우선순위는 명확하지만, 첫 사용자용
hotkey affordance, low-height rail, loading/empty/error의 패널별 표현이 구현
계약으로 더 구체화되어야 10/10이 된다.

시각 mockup은 gstack designer binary와 output 경로는 확인했지만, 현재 환경에
OpenAI API key가 없고 사용자 홈의 design artifact 디렉터리 쓰기도 차단되어
생성하지 못했다. 따라서 이번 검토의 시각 기준은 위 ASCII wireframe, 실제
lipgloss frame 계약, terminal width/height matrix로 고정한다.

### 14.2 정보 hierarchy

```text
1. Graph topology + commit identity + state
2. Graph title subject
3. active section Details
4. Local/Remote/Tags collection state
5. Hotkey help, only when requested or as one-line hint
```

Graph는 시선의 anchor이고 Details는 현재 focus를 설명한다. 우측 rail의 네 패널은
동일한 visual weight를 갖되 active section만 border/title accent로 표시한다.
Details가 Local보다 위에 있어 focus의 의미를 먼저 제공하지만, Details가 비어도
Local/Remote/Tags의 탐색을 막지 않는다.

### 14.3 상태 커버리지

| Panel | Loading | Empty | Error | Success | Partial/overflow |
|---|---|---|---|---|---|
| Graph | 기존 loading overlay | `(no graph to show yet)` | 기존 blocked/error | page/header/rows | Graph paging, `page x-y/N` |
| Details | `Loading...` 또는 빈 body | `(empty)` | 기존 detail error | selected detail rows | `… +N`, `ctrl+u/d` |
| Local/Remote/Tags | 기존 section loading 상태 | 기존 empty 문구 | 기존 status/blocked overlay | rows | active cursor window/inactive `… +N` |
| Hotkeys | popup open state | 항상 Global + active section | popup size fallback | listed actions | popup scroll/page |

### 14.4 Hotkey UX 권장안

- 메인 layout에는 Actions 목록을 다시 넣지 않는다.
- Graph page info line에 `?: hotkeys`를 width가 충분할 때만 표시한다. 이는 목록이
  아니라 affordance이며, 정보 밀도를 거의 늘리지 않는다.
- `?`를 누르면 `Hidden Hotkeys` popup의 header/focus/content/footer를 유지한다.
- popup content는 Global의 unique navigation/system 키와 active section의
  visible/conditional 키만 표시한다.
- popup을 닫아도 active section, cursor, Graph paging, Details scroll은 바뀌지 않는다.
- low-height에서는 footer를 먼저 유지하고 content를 줄이며, 마지막에는 title과
  `esc: close`만 남긴다.

### 14.5 디자이너 우려와 결정

| 우려 | 설계 결정 |
|---|---|
| 4등분 rail이 카드 모자이크처럼 보임 | rail 사이의 border는 유지하되 각 card의 padding/장식을 줄이고, 제목과 데이터만 남긴다 |
| Details가 좁은 우측 열에서 답답함 | Details는 한 열 layout을 유지하고 key/value를 줄바꿈하지 않도록 value를 truncate + scroll한다 |
| Global 삭제가 앱 위치를 숨김 | Graph title, active section title, `?: hotkeys` hint가 orientation을 담당한다 |
| 색상만으로 point를 구분 | state 텍스트 컬럼과 topology `*`를 항상 유지한다 |

### 14.6 반응형 규칙

- 140열 이상: Graph title을 최대화하고 author를 보조로 유지한다.
- 100~139열: author를 축약하고 title을 우선한다.
- 80~99열: author를 제거하고 hash/branch/state/topology/date/title 순서를 유지한다.
- 80열 미만: shell의 최소 폭 계약을 적용하되, state/topology와 Graph page 정보는
  유지하고 title/rail value를 먼저 truncate한다.
- 높이가 낮으면 Details/rail body를 줄이고 Graph의 page navigation을 보존한다.

## 15. Eng 리뷰

### 15.1 의존성 구조

```text
model.repoStatus + stashByBase
          │
          ├─ renderContextInfoLines() ─▶ renderDetailsPanel()
          ├─ graphRows() ─▶ graphRow / pointState
          └─ hiddenHotkeySections() ─▶ Hidden Hotkeys overlay
                                      │
                                      ▼
renderAppView()
  ├─ layoutMainRow() ─▶ Graph frame + right rail
  ├─ renderDetailsPanel() ─▶ Details frame
  ├─ renderSectionContent() ─▶ Local/Remote/Tags
  └─ applyShellOverlays() ─▶ hidden hotkeys and action overlays
```

새 async command, Git query, persistence, public CLI option은 없다. 위험한 결합은
두 가지다. 첫째, `renderContextInfoLines`를 이동하면서 기존 `contextScroll`
초기화 규칙을 잃을 수 있다. 둘째, graph point state 계산을 normal/raw 경로에
각각 복사하면 색상 누락이 반복된다. 따라서 Details data lines와 graph point
state를 각각 순수 helper로 둔다.

### 15.2 구현 규칙

1. `renderAppView`에서 top/header row 생성을 제거하고 main row 하나를 만든다.
2. `renderRightRail`은 Details, Local, Remote, Tags 네 프레임을
   `splitFourHeights`로 만든다. 합계는 rail outer height와 정확히 같아야 한다.
3. Details frame은 `renderContextInfoLines`를 호출하되 Actions column은 호출하지
   않는다. 함수명은 `renderDetailsContent`로 바꾸거나 호환 wrapper를 둔다.
4. `graphPointState(stashCount, tagCount, interaction)` 같은 순수 mapping helper를
   두고 normal/raw point renderer가 이를 공유한다.
5. connector/spacer는 point state를 받지 않고 status field는 blank로 유지한다.
6. graph title budget은 fixed prefix를 먼저 계산하고 author 제거 → title 확장 순서로
   동작한다. header/row/raw row의 prefix width를 같은 helper에서 계산한다.
7. `?` hint는 Graph page line의 available width가 충분할 때만 렌더한다.

### 15.3 실패 모드

| 실패 | 사용자 증상 | 방어 테스트/복구 |
|---|---|---|
| 네 frame 높이 합계 초과 | Tags bottom이 Graph bottom보다 아래로 내려감 | lipgloss.Height와 border row equality test |
| Details body height 음수 | panic 또는 제목/본문 소실 | 1~12 height matrix, minimum fallback |
| active section 변경 후 stale detail | 이전 section 정보가 남음 | section switch resets detail scroll, render test |
| overlay hint가 좁은 폭에서 row를 밀어냄 | Graph title/state clipping | available-width branch test |
| raw row만 point 색상 누락 | 같은 상태가 row 종류별로 다르게 보임 | normal/raw + stash/tag/both matrix |
| color profile에서 색상 제거 시 의미 손실 | NO_COLOR 사용자가 상태를 못 읽음 | ANSI/NO_COLOR text marker test |
| `?` popup에서 action 누락 | 사용자가 실행 키를 찾지 못함 | 각 section expected key set test |

### 15.4 테스트 diagram

```text
renderAppView
  ├─ layoutMainRow
  │   ├─ graphHeight/railHeight boundary       [unit: all heights]
  │   ├─ renderGraphContent                    [unit: page/empty/overflow]
  │   └─ renderRightRail
  │       ├─ renderDetailsPanel                [unit: section/state/scroll]
  │       ├─ renderSectionContent x3            [unit: empty/active/inactive]
  │       └─ splitFourHeights                   [unit: 0..N sum invariant]
  └─ applyShellOverlays
      └─ renderHiddenHotkeysPopup              [unit: section/close/scroll]

graphRow
  ├─ graphPointState
  │   ├─ none / stash / tag / both
  │   ├─ focus / handshake / conflict precedence
  │   └─ ANSI / NO_COLOR output
  ├─ normal row point rendering
  ├─ raw row point rendering
  └─ connector/status column alignment
```

### 15.5 테스트 계획 artifact

구현 전 별도 test plan artifact를 `~/.gstack/projects/hrllk-graphkeeper/`에
작성하려 했으나, 현재 실행 환경은 사용자 홈의 gstack 디렉터리 쓰기를 허용하지
않는다. 동일한 내용을 이 문서의 테스트 diagram과 테스트 계획 초안에 기록해
계획 자체가 독립적으로 실행 가능하도록 한다.

## 16. DX 리뷰

Graphkeeper는 개발자와 저장소 관리자가 사용하는 TUI이므로 DX scope를 적용했다.

### 개발자 journey

| 단계 | 개발자 행동 | 마찰 | 계획 대응 |
|---|---|---|---|
| 설치 | binary 실행 | 별도 없음 | 범위 밖 |
| 첫 진입 | 화면 구조 파악 | Global이 사라짐 | Graph/rail title과 `?: hotkeys` hint |
| Graph 읽기 | row/subject scan | title 폭 부족 | subject 우선 budget |
| focus 이동 | j/k/tab | Details가 stale할 수 있음 | offset reset 계약 |
| 상태 확인 | stash/tag point 확인 | 색상 누락/NO_COLOR | S/T state + color matrix |
| 명령 발견 | `?` 입력 | `?`를 모를 수 있음 | 짧은 hint와 README 설명 |
| 도움말 탐색 | popup scroll | 긴 목록 | Global + active section만 표시 |
| 좁은 터미널 | resize | rail clipping | min width/height fallback |
| 문제 복구 | esc/resize | overlay 상태 유실 | popup close 상태 보존 테스트 |

### DX scorecard

| 항목 | 점수 | 보완 |
|---|---:|---|
| 첫 진입 이해도 | 7/10 | `?: hotkeys` hint 필요 |
| 단축키 발견성 | 7/10 | overlay와 README 동기화 |
| 명령 일관성 | 9/10 | 기존 key binding 보존 |
| 오류/empty 이해도 | 8/10 | 네 rail 패널 low-height 문구 확정 |
| 문서 findability | 8/10 | README layout/help 설명 추가 |
| upgrade 안전성 | 9/10 | Git/data contract 변경 없음 |
| terminal portability | 8/10 | ANSI/NO_COLOR/width matrix |
| contributor clarity | 8/10 | metric/helper와 diagram 문서화 |

### DX 결론

Hotkey 목록을 상시 노출하지 않는 결정은 power user에게는 밀도를 낮추지만,
first-time user에게는 진입점을 숨긴다. 따라서 `?` overlay만으로 끝내지 않고
Graph page info line에 공간이 있을 때만 `?: hotkeys`를 표시하는 것이 추천안이다.
이 hint는 전체 목록을 상시 노출하지 않으므로 사용자 요청과 충돌하지 않는다.

## 17. 통합 구현 작업

- [x] **T1 (P1, human 2h / CC 15m)** — main row layout과 four-card right rail 구현
  - 파일: `view_shell.go`, `view_layout.go`
  - 검증: width/height matrix, Graph/Tags border equality
- [x] **T2 (P1, human 1.5h / CC 12m)** — Context Details를 Details panel로 이동하고 Actions 제거
  - 파일: `view_detail.go`, `view_shell.go`, navigation 관련 파일
  - 검증: Graph/Local/Remote/Tags detail, scroll reset, empty/error/loading
- [x] **T3 (P1, human 1.5h / CC 12m)** — title-priority graph column budget 구현
  - 파일: `graph_render.go`, `graph_render_format.go`, `view_graph.go`
  - 검증: 80/100/140열, raw/normal/header alignment
- [x] **T4 (P1, human 1.5h / CC 12m)** — 모든 Graph point render 경로의 stash/tag 색상 통합
  - 파일: `graph_render.go`, `graph_render_connectors.go`, `theme.go`
  - 검증: none/stash/tag/both/focus/handshake/conflict × color profile
- [x] **T5 (P2, human 45m / CC 5m)** — `?` hint와 Hidden Hotkeys 계약 동기화
  - 파일: `view_graph.go`, `hidden_hotkeys.go`, README
  - 검증: active section 목록, popup close/scroll, 좁은 폭 hint 생략
- [x] **T6 (P2, human 1h / CC 8m)** — 문서와 전체 회귀 검증
  - 파일: `docs/decisions.md`, `DESIGN.md` 또는 README, tests
  - 검증: `go test ./...`, `go test -race ./internal/app`, `scripts/check`, build

## 18. Decision Audit Trail

| # | Phase | Decision | Classification | Principle | Rationale | Rejected |
|---|---|---|---|---|---|---|
| 1 | CEO | Global/Context Actions 제거, Details를 rail 최상단으로 이동 | User-confirmed | User direction | 최종 레이아웃을 사용자가 확정 | 기존 상단 Global + Context |
| 2 | CEO | 기존 renderer 재배치 접근 선택 | Auto-decided | P5 explicit / P3 pragmatic | 새 screen model 없이 목표 달성 | slot abstraction, projection 전면 분리 |
| 3 | Design | Graph를 primary anchor로 두고 rail을 보조 정보로 유지 | Auto-decided | P1 completeness | Graph scan 비용을 가장 크게 낮춤 | 동일 weight dashboard mosaic |
| 4 | Design/DX | `?` overlay + 폭 허용 시 `?: hotkeys` hint | Taste decision | P1 discoverability + subtraction | 목록은 숨기되 진입점은 숨기지 않음 | hint 완전 제거 |
| 5 | Eng | point state 순수 helper 공유 | Auto-decided | P4 DRY | normal/raw 색상 누락 재발 방지 | 경로별 개별 색상 분기 |

## 19. GSTACK REVIEW REPORT

CEO: 사용자 확정 구조를 유지하면서 hotkey discoverability와 low-height rail을
보완했다.
Design: APP UI 기준 7/10에서 9/10으로 보완했다. rounded frame, ANSI palette,
subtraction 원칙을 유지한다.
Eng: four-height sum, Details state, title budget, point state matrix를 테스트
계약으로 고정했다.
DX: `?` overlay를 source of truth로 삼되 폭 허용 시 `?: hotkeys` hint를 둔다.
Codex independent voice: 실행 환경 권한 문제로 unavailable. 실제 코드 기반
검토는 본 리뷰에서 수행했다.

VERDICT: 구현 완료. 전체 테스트·레이스 테스트·정적 검사·빌드 검증을 통과했으며,
커밋 전 최종 diff review 대기 상태다.
