# 섹션별 hidden hotkey 표시 범위 개선

Task Master: `2.2 섹션별 간략 hotkey 목록 노출`

## Context

저장소 관리자가 GraphKeeper를 사용할 때 `?`를 누르면 현재 섹션과 관계없는
모든 섹션의 hotkey까지 한 popup에 표시된다. 목록이 길어 필요한 단축키를
찾는 데 시간이 걸리고, 현재 어떤 조작이 가능한지 한눈에 파악하기 어렵다.

공통 조작은 유지하되 현재 섹션의 hotkey만 함께 보여 popup을 짧고 실용적으로
만든다. 개발자 사용도 지원하지만 주 사용자는 저장소 관리자다.

## Current State

`internal/app/hidden_hotkeys.go:36`의 `renderHiddenHotkeysPopup`은
`hiddenHotkeySections(m)`가 반환하는 전체 목록을 렌더링한다.

현재 popup에는 다음 5개 영역이 모두 표시된다.

- Global
  - Common
  - Moved out
- Graph
- Local
- Remote
- Tags

`?` 입력은 `internal/app/key_handling_browse.go:102`에서
`hiddenHotkeysOpen`을 열고, popup 닫기는
`internal/app/hidden_hotkeys.go:26`의 `esc`/`?` 처리로 동작한다.
이 routing은 정상이며 변경하지 않는다.

## Proposed Change

`?` popup에 다음 순서로만 표시한다.

1. `Global`
   - `Common`
   - `Moved out`
2. 현재 `activeSection`
   - Graph 또는 Local 또는 Remote 또는 Tags

예시:

- Graph에서 `?` → Global + Graph
- Local에서 `?` → Global + Local
- Remote에서 `?` → Global + Remote
- Tags에서 `?` → Global + Tags

비활성 섹션은 표시하지 않는다. Global과 현재 섹션 사이는 빈 줄로 구분한다.
각 영역의 기존 group 순서와 설명은 유지한다.

### Implementation Details

변경 파일:

- `internal/app/hidden_hotkeys.go`
- `internal/app/model_test.go`

`hiddenHotkeySections(m)`는 hotkey 정의의 단일 출처이므로 유지한다.
별도의 hotkey 목록을 복사하지 않는다.

`renderHiddenHotkeysPopup`에서 전체 목록을 바로 순회하지 않고 다음 선택 규칙을
적용한다.

- title이 `Global`인 섹션은 항상 포함
- `section.active == true`인 현재 섹션 하나만 포함
- 그 외 섹션은 제외
- 결과 순서는 항상 Global → 현재 섹션
- 현재 섹션을 찾지 못하면 Global만 표시하고 전체 섹션으로 fallback하지 않음

기존 동작은 유지한다.

- `?`로 popup 열기
- `esc` 또는 `?`로 popup 닫기
- popup overlay precedence
- popup 제목과 `focus: <section>` 표시
- `Visible`, `Conditional`, `Common`, `Moved out` group 구조
- `fitVisibleWidth` 기반 줄 폭 제한
- Graph search의 `/` 동작
- popup 높이가 부족할 때는 `title → footer → focus → content` 우선순위를
  적용한다. title과 footer는 유지하고 focus/content는 단계적으로 줄어들 수
  있으며 content viewport는 0줄까지 허용한다.
- `↑/↓`·`j/k`는 한 줄, `ctrl+u/d`는 한 페이지 스크롤로 사용한다.
- footer에 `↑/↓ j/k: scroll · ctrl+u/d: page · esc: close`를 표시한다.
- popup이 열린 상태에서 terminal 크기가 바뀌면 offset을 reset하지 않고 새
  content viewport 범위로 clamp한다. 창이 커지면 현재 위치를 유지하고,
  창이 작아지면 마지막 유효 위치로 보정한다.

## Acceptance Criteria

1. Graph에서 `?`를 누르면 Global과 Graph hotkey만 표시된다.
2. Local에서 `?`를 누르면 Global과 Local hotkey만 표시된다.
3. Remote에서 `?`를 누르면 Global과 Remote hotkey만 표시된다.
4. Tags에서 `?`를 누르면 Global과 Tags hotkey만 표시된다.
5. 비활성 섹션 제목과 해당 섹션의 hotkey는 popup에 포함되지 않는다.
6. Global의 `Common`과 `Moved out`은 모든 섹션에서 표시된다.
7. Global과 현재 섹션은 빈 줄로 구분된다.
8. `esc`와 `?`로 popup을 닫는 기존 동작이 유지된다.
9. Graph의 `/` 검색 동작이 유지된다.
10. 좁은 터미널에서도 기존 popup 폭 제한 및 줄바꿈 규칙이 유지된다.
11. 좁은 터미널 높이에서도 title과 footer가 표시되고, 가능한 경우 focus와
    hotkey 내용이 스크롤된다.
12. `↑/↓`·`j/k` 한 줄 및 `ctrl+u/d` 페이지 스크롤이 offset 범위 안에서 동작한다.
13. popup 열린 상태의 terminal resize 후에도 content가 빈 상태가 되지 않는다.
14. 신규 테스트와 기존 `internal/app` 테스트가 통과한다.
15. `scripts/check`가 통과한다.

## Testing Plan

| Layer | What | Count |
|---|---|---:|
| Unit | Global + active section 선택 helper | +1 |
| Unit | Graph/Local/Remote/Tags별 포함·제외 검증 | +4 cases |
| Regression | popup title, focus, footer 중앙 정렬 | 기존 유지 |
| Regression | `?` 열기, `esc`/`?` 닫기, `/` 검색 | 기존 유지 또는 보강 |
| Interaction | 한 줄/페이지 스크롤 및 작은 높이 viewport | 신규 |
| Regression | popup 열린 상태 terminal resize 및 offset clamp | 신규 |

권장 테스트:

- `TestHiddenHotkeysPopupShowsGlobalAndActiveSection`
- `TestHiddenHotkeysPopupExcludesInactiveSections`
- `TestHiddenHotkeysPopupKeepsGlobalGroups`
- `TestQuestionMarkClosesHiddenHotkeys`

## Rollback Plan

구현 커밋을 revert하면 기존 전체 섹션 popup 렌더링으로 복구된다.
데이터 변경이나 migration은 없다.

## Effort Estimate

- `internal/app/hidden_hotkeys.go`: 선택 helper, 높이 viewport 및 렌더 순서 변경, 약 1.5~2.5시간
- `internal/app/model.go`, `internal/app/key_handling_browse.go`: popup scroll 상태/입력, 약 30~60분
- `internal/app/model_test.go`: 섹션별 렌더링/scroll 회귀 테스트, 약 1~2시간
- 검증 및 수동 확인: 약 30분
- 총 예상: 약 2.5~4시간

## Files Reference

| File | Change |
|---|---|
| `internal/app/hidden_hotkeys.go:36` | Global + active section 선택 및 높이 viewport 렌더링 |
| `internal/app/hidden_hotkeys.go:92` | 기존 hotkey catalog 재사용, 선택 helper 추가 |
| `internal/app/view_overlays.go` | `bodyHeight`를 hidden hotkey renderer에 전달 |
| `internal/app/key_handling_browse.go:102` | `?` 진입점 유지 및 popup scroll offset 초기화 |
| `internal/app/key_handling.go:25` | 변경하지 않음. popup 우선 처리 유지 |
| `internal/app/view_overlays.go:74` | 변경하지 않음. overlay 등록 유지 |
| `internal/app/model_test.go:1515` | 기존 전체 섹션 기대값을 새 표시 계약으로 수정 |
| `internal/app/model_test.go` | 섹션별 포함/제외 및 닫기 회귀 테스트 추가 |
| `internal/app/model.go` | `hiddenHotkeysScroll` offset 추가 |
| `internal/app/key_handling_browse.go` | popup open 시 offset 초기화 |

## Out of Scope

- hotkey 키 또는 설명 문구 자체 변경
- Global hotkey 재배치
- conditional hotkey의 실제 활성화 상태 판정 개선
- popup 스타일, 색상, 애니메이션 변경
- 섹션 전환 UX 변경
- `+N more` 형태의 축약 표시
- popup 높이 부족 시 scroll 없는 단순 축약 표시
- Remote/Tags 메타데이터 추가
- 구현 에이전트 자동 실행

## Related

- Task Master: `2.2 섹션별 간략 hotkey 목록 노출`
- 계획문서: 앞서 작성한 refactor section hotkey 목록 계획문서
