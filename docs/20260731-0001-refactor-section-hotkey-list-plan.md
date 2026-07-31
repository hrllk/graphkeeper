# 섹션별 간략 hidden hotkey 목록 노출 계획

Task Master: 2.2 섹션별 간략 hotkey 목록 노출

## 최신 결정

2026-07-31 확정: hidden hotkey popup은 Global(Common + Moved out)과
현재 activeSection만 표시한다. 결과 순서는 항상 Global → activeSection이다.
2026-07-31 디자인 리뷰 확정: 좁은 터미널 높이에서도 두 영역을 유지하고,
`↑/↓`·`j/k` 한 줄 스크롤과 `ctrl+u/d` 페이지 스크롤을 제공한다.
이 문서의 초기 "현재 섹션만" 표현은 폐기하며, 상세 계약은
docs/20260731-0001-refactor-section-hotkey-list-spec.md를 따른다.

## 목표

현재 ?를 누르면 Global, Graph, Local, Remote, Tags의 hidden hotkey가 모두
표시된다. 저장소 관리자가 현재 섹션에서 사용할 항목을 찾기 어렵기 때문에,
공통 조작과 현재 섹션의 항목만 보여 popup을 짧게 만든다.

## 현재 구조

- internal/app/hidden_hotkeys.go: hiddenHotkeySections가 전체 catalog를 만들고
  renderHiddenHotkeysPopup이 전체 section을 순회한다.
- internal/app/key_handling_browse.go: handleBrowseGlobalKey의 ? case가
  hiddenHotkeysOpen을 연다.
- internal/app/key_handling.go: hiddenHotkeysOpen이면
  handleHiddenHotkeysKey를 우선 호출한다.
- internal/app/view_overlays.go: hidden-hotkeys overlay가 이미 등록되어 있다.
- internal/app/model.go: activeSection과 hiddenHotkeysOpen이 이미 존재한다.

새 overlay나 새 hotkey catalog는 만들지 않는다. 높이 대응을 위해
`hiddenHotkeysScroll` 상태 필드 하나만 추가한다.

## 표시 계약

| activeSection | 표시 순서 | 표시 그룹 |
| --- | --- | --- |
| sectionGraph | Global → Graph | Common, Moved out → Visible, Conditional |
| sectionCurrent | Global → Local | Common, Moved out → Visible, Conditional |
| sectionRemote | Global → Remote | Common, Moved out → Visible |
| sectionTags | Global → Tags | Common, Moved out → Visible |

- Global은 항상 표시한다.
- active=true인 Graph/Local/Remote/Tags 중 하나만 표시한다.
- 비활성 section은 표시하지 않는다.
- Global과 active section 사이에 빈 줄을 둔다.
- 기존 title, focus label, active title style, group 순서, footer, popup 폭,
  fitVisibleWidth, overlay precedence를 유지한다.
- active section이 없는 비정상 model에서는 Global만 표시한다.
- ? 열기, esc/? 닫기, Graph의 / search는 변경하지 않는다.
- popup 높이가 현재 body에 맞지 않으면 title/focus/footer를 고정하고 중간
  hotkey 내용만 viewport로 잘라낸다. `↑/↓`·`j/k`는 한 줄, `ctrl+u/d`는
  표시 가능한 내용 영역 기준 한 페이지씩 이동한다.
- footer는 `↑/↓ j/k: scroll · ctrl+u/d: page · esc: close`로 탐색 조작을
  안내한다.

## 구현 방법

### 1. popup scroll 상태와 입력

파일: internal/app/model.go, internal/app/key_handling_browse.go,
internal/app/hidden_hotkeys.go

`model`에 `hiddenHotkeysScroll int`을 추가한다. `?`로 popup을 열 때 0으로
초기화하고, popup이 열린 동안 `up`/`k`와 `down`/`j`는 한 줄,
`ctrl+u`와 `ctrl+d`는 viewport 높이만큼 이동한다. 모든 offset은 0과 마지막
viewport 사이로 clamp한다. 스크롤 입력은 Browse의 section 이동, Graph cursor
이동, 검색 입력으로 전달하지 않는다.

### 2. 표시 section 선택 helper

파일: internal/app/hidden_hotkeys.go

전체 정의인 hiddenHotkeySections(m)을 단일 출처로 유지한다. 다음 규칙의
visibleHiddenHotkeySections(m) helper를 추가한다.

    sections := hiddenHotkeySections(m)
    visible := make([]hiddenHotkeySection, 0, 2)
    for _, section := range sections {
        if section.title == "Global" || section.active {
            visible = append(visible, section)
        }
    }
    return visible

Global은 title로 선택하고 나머지는 기존 active flag로 선택한다. 정상 browse
상태에서는 결과가 Global과 현재 section 두 개이며, invalid active section에서는
Global 하나만 반환한다.

### 3. popup renderer 변경

renderHiddenHotkeysPopup에서 전체 catalog 대신
visibleHiddenHotkeySections(m)의 결과를 순회한다. signature는
`renderHiddenHotkeysPopup(m, bodyWidth, bodyHeight)`로 확장한다. renderer는
title strip, popup border/padding, title/focus/footer 고정 줄을 제외한
`contentViewportHeight`를 명시적으로 계산하고, section/group/item으로 만든
content lines에 `hiddenHotkeysScroll`을 적용한다. 높이가 부족하면 고정
우선순위는 `title → footer → focus → content`로 한다. content viewport는
0줄까지 줄어들 수 있지만 title과 footer는 항상 표시하고, content가 0줄이면
scroll offset은 0으로 clamp한다.

`shellOverlayStack`은 기존 `bodyHeight`를 renderer에 전달한다. 공통
`overlayPopup`은 수정하지 않는다. 따라서 popup 최종 높이는 항상 base body
높이 이하이며, 기존 overlay의 `baseH < popupH` 조기 반환을 피한다.

content line 수와 현재 viewport 높이로 계산하는
`clampHiddenHotkeyScroll(offset, totalLines, viewportHeight)` helper를
renderer와 `handleHiddenHotkeysKey`가 공유한다. terminal resize 자체는
`hiddenHotkeysScroll`을 직접 reset하지 않고, 다음 render/input 시점에 새
viewport 범위로 파생 clamp한다. 창이 커지면 가능한 한 현재 위치를 유지하고,
창이 작아지면 마지막 유효 위치로만 이동한다.

기존 renderHiddenHotkeySectionTitle, renderHiddenHotkeyGroupLines,
renderCenteredPopupLine, fitVisibleWidth, popupWidthForBody 호출은 유지한다.
렌더 대상만 줄이고 스타일과 줄바꿈 책임은 기존 helper에 둔다.

### 3. 변경하지 않는 영역

- activeSection 외의 model 상태는 추가하지 않는다. 단, popup 높이 대응에
  필요한 `hiddenHotkeysScroll` offset은 추가한다.
- handleHiddenHotkeysKey는 닫기뿐 아니라 popup scroll 입력도 처리한다.
- handleBrowseGlobalKey의 ? 진입점은 유지하되 열 때 scroll offset을 0으로
  초기화한다.
- shellOverlayStack의 등록과 순서
- activeSection 전환 규칙
- hotkey key/설명 문구
- conditional action 판정
- popup 스타일과 색상

## 테스트 계획

파일: internal/app/model_test.go

### Helper 테스트

TestVisibleHiddenHotkeySectionsReturnsGlobalAndActiveSection을 table-driven으로
추가한다.

- Graph, Local, Remote, Tags 각각의 결과 길이가 2인지 확인한다.
- 결과 순서가 Global → sectionName(activeSection)인지 확인한다.
- invalid graphSection에서는 결과 길이가 1이고 title이 Global인지 확인한다.

### Popup 포함/제외 테스트

TestHiddenHotkeysPopupShowsGlobalAndActiveSection을 table-driven으로 추가한다.
ANSI를 제거한 popup 문자열을 검사한다.

| section | 포함해야 함 | 제외해야 함 |
| --- | --- | --- |
| Graph | Global, Moved out, Graph, m: merge | Local, enter: jump to graph |
| Local | Global, Moved out, Local, s: stash changes | Remote, m: merge |
| Remote | Global, Moved out, Remote, space: checkout | Tags, s: stash changes |
| Tags | Global, Moved out, Tags, enter: jump to graph | Graph, s: stash changes |

각 케이스에서 Common 그룹과 Moved out 그룹이 포함되는지도 확인한다.

### 회귀 테스트

다음 테스트를 유지한다.

- TestHiddenHotkeysPopupCentersHeaderAndFooter
- TestRenderHiddenHotkeySectionTitleHighlightsActiveSection
- TestGraphSearchQuestionMarkOpensHiddenHotkeys
- popup `up/down`, `j/k`, `ctrl+u/d` scroll offset clamp 테스트
- 좁은 body height에서도 title/focus/footer가 유지되는 렌더 테스트
- `shellOverlayStack → overlayPopup` 경로에서 popup 최종 높이가 base body 이하이고
  실제 합성 결과에 title/footer가 포함되는 통합 테스트
- popup key handler table-driven 테스트: `up/down`, `j/k`, `ctrl+u/d`, `esc`, `?`,
  무시되는 입력 각각의 offset·open 상태·activeSection·cursor 불변 검증
- `?`로 popup을 다시 열 때 scroll offset이 0으로 reset되는 model update 테스트

기존에 닫기 동작 테스트가 없으면 TestQuestionMarkClosesHiddenHotkeys를
추가해 ? 입력 후 hiddenHotkeysOpen=false를 확인한다. esc도 동일하게 확인한다.

### 검증

    go test ./internal/app
    scripts/check

## 실패 모드

| 실패 | 테스트 | 사용자 영향 | 대응 |
| --- | --- | --- | --- |
| 전체 section 재노출 | 포함/제외 assertion | 긴 popup과 탐색 비용 | visible helper 결과만 순회 |
| Global 누락 | Global 포함 assertion | 공통 조작 발견성 저하 | title == Global 규칙 유지 |
| active section 누락 | helper 길이/title assertion | 현재 조작을 찾지 못함 | active flag 매핑 확인 |
| invalid section에서 전체 노출 | invalid helper test | 잘못된 hotkey 안내 | Global만 반환 |
| ?/esc 회귀 | key routing 회귀 test | popup이 열리거나 닫히지 않음 | key handling 미수정 |
| 좁은 폭 줄바꿈 회귀 | 기존 width/center test | terminal에서 text clipping | 기존 fit/width helper 재사용 |
| 좁은 높이에서 popup 미표시 | renderer + overlay integration test | title/footer 우선, focus/content 단계적 축소 | popup 최종 높이를 base body 이하로 clamp |
| popup 열린 상태의 terminal resize | resize/render 테스트 | 공유 clamp helper로 offset 파생 보정 | 빈 content 없이 현재 위치 유지 또는 마지막 위치로 이동 |
| popup 입력이 Browse로 누출 | key handler table-driven 테스트 | popup 우선 분기와 입력 소비 | section/cursor/search가 변하지 않음 |
| popup 재오픈 시 이전 위치 잔류 | open reset 테스트 | `?` open 시 offset 0 | 항상 첫 content부터 시작 |

## 구현 순서

1. hiddenHotkeysScroll 상태와 popup 입력 처리 추가
2. visibleHiddenHotkeySections helper 추가
3. shellOverlayStack에서 bodyHeight 전달, renderer의 외곽 높이 제외 계산 및 높이 viewport 추가
4. renderer와 key handler가 공유하는 scroll clamp helper 및 resize 테스트 추가
5. model_test.go에 section 포함/제외, scroll clamp, 입력 소비/open reset, 닫기 회귀 및 overlay 합성 테스트 추가
6. go test ./internal/app 실행
7. scripts/check 실행
8. Graph/Local/Remote/Tags 및 작은 terminal height에서 수동 확인

## 범위 제외

- hotkey key/설명 문구 재설계
- Global help panel 재배치
- conditional hotkey 실제 활성화 판정 개선
- popup 높이 부족 시 scroll 없는 단순 축약 표시
- popup 스타일, 색상, 애니메이션 변경
- section 전환 UX 변경
- Remote/Tags 메타데이터 추가
- 구현 agent 자동 실행

## What already exists

activeSection, hiddenHotkeySections catalog, popup renderer, overlay registration,
? key routing, section help tests가 이미 있다. 구현은 이 요소들을 재사용하고
새 상태나 새 서비스는 만들지 않는다.

## NOT in scope

이번 변경은 hidden hotkey popup의 표시 범위만 바꾼다. hotkey 정의, section 이동,
search, popup 스타일과 Global help 구조는 별도 작업이다.

## 아키텍처

    activeSection
          |
          v
    hiddenHotkeySections(m)       existing complete catalog
          |
          v
    visibleHiddenHotkeySections   Global + active filter
          |
          v
    renderHiddenHotkeysPopup      existing layout/group renderer
          |
          v
    shellOverlayStack -> hidden hotkeys overlay

## 병렬화

Sequential implementation, no parallelization opportunity. 두 변경 파일이 모두
동일한 hidden hotkey 동작에 관여하며 테스트는 구현 helper 계약에 의존한다.

## Implementation Tasks

- [ ] P1 visibleHiddenHotkeySections로 Global + active section 선택
- [ ] P1 popup 렌더 대상 교체
- [ ] P1 Graph/Local/Remote/Tags 포함/제외 테스트
- [ ] P2 invalid section과 ?/esc 닫기 회귀 테스트
- [ ] P2 scripts/check 및 수동 terminal 확인

## 결정 기록

| 결정 | 이유 |
| --- | --- |
| Global(Common + Moved out) 유지 | 모든 section에서 필요한 조작의 발견성을 보존한다. |
| active section 하나만 추가 | 비활성 section 정보 과밀을 제거한다. |
| 기존 catalog 재사용 | hotkey 정의 중복과 drift를 막는다. |
| 새 상태 최소화 | activeSection은 그대로 재사용하고, 높이 대응에 필요한 scroll offset만 추가한다. |
| 높이 책임은 renderer에 국한 | 공통 overlayPopup의 기존 배치 계약을 보존하고 hidden hotkey popup만 bodyHeight를 사용한다. |
| 최소 높이 우선순위 | title과 footer를 먼저 보장하고 focus/content를 단계적으로 줄여 작은 화면에서도 닫기 경로를 남긴다. |
| resize offset 보정 | popup 상태를 reset하지 않고 renderer/input 공통 clamp로 현재 위치를 보존한다. |
| invalid fallback은 Global만 | 잘못된 context에서 전체 hotkey를 잘못 안내하지 않는다. |
| popup scroll은 기존 TUI 관례 사용 | `↑/↓`·`j/k` 한 줄, `ctrl+u/d` 페이지로 학습 비용을 낮춘다. |

## Engineering Review Addendum

### Architecture review

One documentation-contract issue was found and resolved during review: the initial
plan said active section only while the approved spec said Global plus active
section. The plan now matches the approved spec. The implementation architecture
reuses the existing model.activeSection,
hiddenHotkeySections catalog, popup renderer, and shell overlay. It adds no
service, persistence, or distribution surface; the only new state is the local
hiddenHotkeysScroll offset required for the approved height-aware popup.

### Code quality review

No code-quality issues found. The planned helper keeps hotkey definitions in one
catalog and leaves key routing, styling, and width logic in their existing helpers.
The Global selection is intentionally explicit in the helper contract and is
covered by tests, while the active section continues to use the existing active
flag.

### Test review coverage diagram

    ? key in Browse
          |
          v
    hiddenHotkeysOpen=true, scroll=0
          |
          v
    handleKeyMsg priority
      | popup open                  | popup closed
      v                             v
    handleHiddenHotkeysKey        handleBrowseKey
      | esc/? close                | ? opens + reset
      | up/down,j/k line           | section/cursor/search
      | ctrl+u/d page
      | default consume
      v
    renderHiddenHotkeysPopup(bodyHeight)
          |
          v
    visible sections: Global + active
          |
          v
    content lines -> fixed-height viewport
      | total=0 / viewport=0       | total>viewport
      v                             v
    offset=0, title/footer         shared clamp(offset)
    remain visible                 + sliced content
          \_________________________/
                       |
                       v
           shellOverlayStack -> overlayPopup
             | popupH <= baseH      | popupH > baseH
             v                       v
          compose popup            prevented by renderer

User flows covered:

- Graph/Local/Remote/Tags + ?: Global and active section are present; inactive
  sections are absent.
- Invalid active section: Global remains and no unrelated section is shown.
- Popup opened + esc or ?: popup closes.
- Graph + /: existing search behavior remains unchanged.
- Narrow width: existing fitVisibleWidth and centered header/footer behavior remains.

Coverage status: 13 code branches and 11 user flows planned; all have unit or
regression coverage. No E2E test is needed because the change is a local
renderer/input interaction with no external integration, persistence, or
repository side effect.

### Failure modes

| Failure | Test | Error handling | User-visible result |
| --- | --- | --- | --- |
| Global omitted | Global inclusion assertion | Helper contract prevents it | Common controls disappear; detected by test |
| Wrong active section included | Four-section table test | Existing active flag mapping | User sees incorrect actions; detected by test |
| Invalid model exposes all sections | Invalid helper test | Global-only fallback | Safe, limited help instead of misleading full catalog |
| Empty group renders incorrectly | Existing group renderer plus empty-case assertion | `(none)` fallback | User sees explicit empty group |
| Popup close regression | esc/? key test | Existing handler remains unchanged | Popup can be closed by documented keys |
| Width regression | Existing centering/fit tests | Existing width helpers remain in path | Text stays bounded by popup width |
| Popup input leaks to Browse | Table-driven key handler test | `hiddenHotkeysOpen` priority consumes input | Background section/cursor/search stays unchanged |
| Reopen keeps stale offset | Open-reset model test | `?` sets offset to zero | Popup starts at first content line |
| Resize leaves stale offset | Resize/render clamp test | Shared clamp helper | Content remains visible at current/last valid position |
| Popup exceeds base height | Overlay integration test | Renderer computes bounded final height | Popup is actually composited on screen |

No critical silent failure gap found.

### Performance review

No performance issues found. The helper scans the existing fixed five-section
catalog and the renderer emits the same item lines for two sections instead of
five. There are no repository reads, network calls, allocations proportional to
repository size, or caches to add.

### Review completion

- Step 0: Scope Challenge — scope accepted as-is; five implementation/test files and no new services.
- Architecture Review: 3 issues found, all resolved during review.
- Code Quality Review: 0 issues found.
- Test Review: diagram produced, 2 test gaps found and added to the plan.
- Performance Review: 0 issues found.
- NOT in scope: written above.
- What already exists: written above.
- TODOS.md updates: 0 items proposed.
- Failure modes: 0 critical gaps flagged.
- Outside voice: skipped; Codex app-server could not initialize in this sandbox.
- Parallelization: 1 sequential lane, 0 parallel.
- Lake Score: 5/5 recommendations chose the complete option.
- Korean QA artifact: `/Users/hrk/.gstack/projects/hrllk-graphkeeper/hrk-develop-eng-review-test-plan-20260731-korean.md`.

## Design Review Addendum

### System audit

- UI scope: existing terminal hidden-hotkeys popup, its keyboard interactions, and
  its narrow-height rendering behavior.
- `DESIGN.md`: present. The review follows the industrial/utilitarian, compact but
  readable, minimal-chrome rules and reuses the existing popup vocabulary.
- Existing leverage: `popupWidthForBody`, `fitVisibleWidth`,
  `renderFloatingTitlePopup`, `overlayPopup`, existing section title styles, and
  existing `up/down`/`j/k` plus `ctrl+u/d` navigation conventions.
- Design mockups: not generated because the installed designer required an
  unavailable OpenAI API key; review used the real renderer architecture and
  existing TUI patterns instead.

### Pass results

| Pass | Initial | Final | Result |
| --- | ---: | ---: | --- |
| Information architecture | 8/10 | 9/10 | Global → active section order and fixed hierarchy are explicit. |
| Interaction states | 6/10 | 9/10 | Small-height viewport, fixed footer, clamp rules, and close states are defined. |
| User journey | 8/10 | 9/10 | `?` discoverability, context focus, scan, scroll, and close flow are covered. |
| AI slop risk | 10/10 | 10/10 | No generic cards, decoration, or marketing patterns; existing TUI language is retained. |
| Design system alignment | 9/10 | 9/10 | Existing colors, border, spacing, typography, and popup helpers are reused. |
| Responsive/accessibility | 4/10 | 9/10 | Height-aware scrolling and visible keyboard instructions resolve the main gap. |
| Unresolved decisions | 7/10 | 10/10 | Scroll granularity and key bindings were resolved as 2A. |

### Resolved design decision

The initial plan did not define behavior when the popup exceeded the terminal
height. Existing `overlayPopup` returns the base view when the popup is taller than
the base, which could make `?` appear to do nothing. The plan now adds a dedicated
content viewport: title, focus, and footer remain visible; the content scrolls with
`↑/↓` or `j/k` by one line and `ctrl+u/d` by one viewport page. The footer documents
these bindings, and the scroll offset is clamped.

### Pass findings

- Information architecture: no remaining issue; Global remains the persistent
  orientation layer and the active section is the task-specific layer.
- Interaction states: no remaining issue; normal, invalid-section, empty-group,
  close, and narrow-height states are specified.
- User journey: no remaining issue; the flow preserves instant discovery and adds
  an explicit recovery path for content that cannot fit vertically.
- AI slop risk: no issue; this is a utility popup with purposeful density.
- Design system alignment: no issue; no new visual vocabulary is introduced.
- Responsive/accessibility: resolved by the scroll decision; keyboard-only use is
  explicit and color is not the sole indicator of active section because of the
  `›` marker and focus label.
- Unresolved decisions: none.

### Implementation Tasks

- [ ] P1 Add `hiddenHotkeysScroll` state and reset it when `?` opens the popup.
- [ ] P1 Handle `up/down`, `j/k`, `ctrl+u/d`, `esc`, and `?` while popup is open.
- [ ] P1 Filter visible sections to Global + active section.
- [ ] P1 Render a height-aware content viewport with fixed title/focus/footer.
- [ ] P1 Add shared scroll clamp and resize coverage alongside section inclusion/exclusion tests.
- [ ] P1 Verify `shellOverlayStack → overlayPopup` keeps the small-height popup visible.
- [ ] P1 Add table-driven popup input-consumption and reopen-reset regression tests.
- [ ] P2 Run `go test ./internal/app`, `scripts/check`, and manual small-height TUI checks.

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 0 | — | Not run |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | — | Not run |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 2 | CLEAR | 5 issues resolved, 0 critical gaps |
| Design Review | `/plan-design-review` | UI/UX gaps | 1 | CLEAR | 8/10 → 9/10, 1 decision resolved |
| DX Review | `/plan-devex-review` | Developer experience gaps | 0 | — | Not run |

**VERDICT:** ENG + DESIGN CLEARED — ready to implement.

NO UNRESOLVED DECISIONS
