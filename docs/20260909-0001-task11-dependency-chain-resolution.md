# Task 11 의존 사슬 해소

상태: REVIEWED — ceo-review 1회 (HOLD SCOPE), 3건 반영
목적: task 11(날짜·road)이 pending design task와 얽힌 사슬을 풀어, 어느 subtask가
**진짜로 막혀 있고** 어느 것이 **제약만 물려받는지** 구분한다.
기준 커밋: `eaff738`
관련: `docs/20260907-0001-...-plan.md`(왜·무엇을), `docs/20260908-0001-...-spec.md`(계약)

이 문서는 계획도 spec도 아니다. **의존 관계만** 다룬다. 구현 지시는 없다.

---

## 왜 이 문서가 필요한가

taskmaster에 기록된 의존을 그대로 따르면 task 11의 절반이 4단 사슬 뒤에 갇힌다.

```
6.2 (80컬럼 overflow)
  └─▶ 6.6 (어휘 통일)          deps=[2]
        └─▶ 6.7 (팝업 affordance)  deps=[6]
              └─▶ 6.8 (카피 우선순위)  deps=[7]
                    └─▶ 11.5 (Inspector 날짜 표시)  ← 같은 헤더를 건드린다
```

11.5를 6.8 뒤로 두면 **Inspector 날짜는 4단 사슬이 다 끝난 뒤**에야 가능하다. 6.2는
레이아웃 overflow 수정이고 6.7은 팝업 입력 affordance다. 둘 다 Inspector 헤더에
날짜를 넣는 것과 아무 관계가 없다.

즉 이 사슬은 **파일이 겹친다는 사실**을 의존으로 오인한 결과다. 풀어야 한다.

---

## 핵심 구분: 의존 vs 제약 상속

두 관계를 섞으면 안 된다.

| | 의존 (dependency) | 제약 상속 (constraint inheritance) |
|---|---|---|
| 뜻 | A가 끝나야 B를 **시작**할 수 있다 | B가 A의 **규칙을 지켜야** 한다 |
| 이유 | A가 B의 입력을 만든다 | 같은 면·같은 계약을 공유한다 |
| 어기면 | B가 착수 불가 | B가 나중에 되돌려진다 |
| taskmaster 표기 | `dependencies` 필드 | `details`에 문장으로 |

**같은 파일을 건드리는 것은 의존이 아니다.** 두 작업이 그 파일의 서로 다른 결정을
소유하면 순서가 자유롭다. 의존이 되는 것은 한쪽이 다른 쪽의 **결정 결과를 입력으로
쓸 때**뿐이다.

---

## 해소 결과

### R-1. 11.5 (Inspector 날짜) — 6.8에 의존하지 **않는다**

**얽힘의 실체:** 둘 다 `commit_inspector_screen.go`의 헤더를 건드린다.

**경계가 이미 그려져 있다.** spec C-12가 소유권을 나눠 놓았다.

| 결정 | 소유자 |
|---|---|
| `date:` / `committed:` 행 추가 | **11.5** |
| author 행의 `FROM <parent>` 꼬리 제거 | **11.5** |
| chrome 행 수를 헤더 길이에서 파생 (`inspectorBodyRows`) | **11.5** |
| `parent:` 행 신설 여부와 위치 | **6.8** |
| 40컬럼에서 short hash를 쓸지, 이메일을 뺄지 | **6.8** |
| `COMMIT` 대문자 표제 vs 소문자 key/value 통일 | **6.8** |

11.5는 6.8의 결정을 **입력으로 쓰지 않는다.** 6.8이 나중에 `parent:` 행을 만들 때
11.5가 비워 둔 자리를 쓰면 되고, 그 자리를 비우는 것이 11.5의 일이다.

**방향이 오히려 반대다.** 11.5가 먼저 가면 6.8이 더 쉬워진다 — `FROM` 꼬리가 이미
사라져 있고 chrome 계산이 이미 파생식으로 바뀌어 있으므로, 6.8은 카피만 정리하면
된다. 11.5를 6.8 뒤로 두면 6.8이 `FROM` 꼬리 문제를 먼저 만나 같은 일을 두 번 한다.

**정정 (ceo-review C-1, C-2). 위 표에 두 가지 오류가 있었다.**

**C-1 — 역방향 의존을 걸면 이 문서가 자기 정의를 위반한다.** 초안은 "6.8에
`dependencies: ['11.5']`를 추가한다"로 끝났고 근거는 "11.5가 먼저 가면 6.8이 쉬워진다"
였다. 그건 **효율 선호**이고 시작 차단이 아니다. 이 문서가 위에서 스스로 정의했다 —
"의존: A가 끝나야 B를 **시작**할 수 있다". 6.8은 11.5 없이 시작할 수 있고, 다만
`FROM` 꼬리 작업을 다시 하게 된다.

하드 의존으로 기록하면 **디자인 백로그가 기능 배포에 묶인다.** 11.5가 미뤄지거나
취소되면 6.8이 이유 없이 막힌다. 그래서 `dependencies`에서 뺀다. 순서 선호는
6.8의 `details`에 문장으로 남긴다 — 이 문서가 정한 표기 규칙 그대로다.

**C-2 — `FROM` 꼬리 제거는 6.8에서 11.5로 옮기는 것이며, 이건 범위 이전이다.**
초안의 소유권 표는 그것을 11.5 몫으로 적었지만, 6.8의 기록(D-018)이 이미 소유하고
있다.

> D-018: commit_inspector_screen.go:64가 author 행 뒤에 '  FROM ' + snapshot.Parent를
> 덧붙인다. 대응하는 TO가 없고 (...) **TO와 짝을 맞추거나 parent: 라벨로 바꿔** 다른
> header 행과 같은 key/value 처리를 준다.

즉 깔끔한 분할이 아니라 **6.8의 헌장 일부를 11.5가 가져가는 것**이다. 11.5가 가져가야
하는 이유는 있다 — 헤더가 이미 넘치는 상태에서 `date:` 행을 얹으려면 꼬리를 먼저
떼야 한다. 하지만 그걸 명시하지 않으면 나중에 6.8을 집는 사람이 자기 헌장의 일부가
이미 처리된 것을 발견하고 자기가 잘못 읽었는지 의심한다.

따라서 **6.8의 details에 이전 사실을 적는다.** 6.8에 남는 것: `parent:` 행을 만들지
말지의 결정, `COMMIT` 대문자 표제와 소문자 key/value 통일, 40컬럼 카피 우선순위.

**결론:** `11.5 deps = ['11.3']` 유지. 6.8에 하드 의존을 **걸지 않는다.**
6.8 `details`에 (a) 11.5 뒤에 가는 것이 싸다는 순서 선호와 (b) `FROM` 꼬리 제거가
11.5로 이전됐다는 사실을 적는다.
제약 상속: 11.5는 6.8이 지목한 문제를 악화시키지 않는다 — 헤더에 잘릴 것을 더 얹지
않고, `FROM` 꼬리를 뺀다.

### R-2. 11.6 (road 하이라이트) — 6.3에 의존하지 **않는다**

**얽힘의 실체:** 둘 다 graph 행의 시각 신호를 건드린다.

- **6.3**이 소유: 커서 포인터와 섹션 포커스를 색 없이 식별 가능하게 만든다.
  `theme.go:47` `pointerMark`가 색만 있고 글리프도 reverse도 없는 문제.
- **11.6**이 소유: road anchor와 경로 멤버 마커를 추가한다.

**11.6은 6.3의 결정을 입력으로 쓰지 않는다.** 다만 6.3이 정한 **정책**을 지켜야 한다 —
`highlighting-color-map.md` Policy의 "NO_COLOR=1에서 visible label과 marker를
보존한다". 그건 6.3이 만드는 게 아니라 이미 문서에 있는 규칙이다.

**부분적으로 이미 해소됐다.** `df7ed32` "fix(graph): make the cursor visible under
NO_COLOR"가 커서 신호를 이미 넣었다. 즉 6.3의 커서 부분은 사실상 착수됐고 11.6은
그 패턴을 재사용한다.

**정정 (2026-09-09 검증).** 이 문단의 초안은 `df7ed32`가 "거터 마커"를 넣었다고
적었다. **틀렸다.** 거터는 `094ca87`이 제거하면서 `TestRenderGraphContentOmitsSelectionArrow`와
`TestRenderGraphContentStartsAtLeftEdge`로 잠갔다. `df7ed32`가 실제로 한 것은
`renderGraphHashField`(`graph_render.go:44-53`)에서 `noColorEnabled()` 분기에
`\x1b[7m`을 **직접 쓰는** 것이다. lipgloss 스타일은 NO_COLOR에서 attribute까지
전부 사라지므로 쓸 수 없다.

이 정정이 spec C-10을 바꿨다(E-7). 의존 결론 자체는 바뀌지 않는다.

**결론:** `11.6 deps = ['11.4']` 유지. 6.3 의존 추가하지 않는다.
제약 상속: NO_COLOR에서 anchor·멤버가 식별되어야 하며, **lipgloss 스타일이 아니라
`noColorEnabled()` 분기의 직접 escape**로 표현해야 한다. 거터는 금지(잠긴 테스트).
spec C-10 참조.

### R-3. 11.8 (`w` 키 등록) — task 8에 의존하지 **않는다**

**얽힘의 실체:** 둘 다 키 발견성을 다룬다.

**task 8의 문제는 Global 그룹이다.** task 8 본문이 이렇게 적고 있다.

> `? 오버레이`: 활성 섹션 하나만 렌더. Global 섹션은 존재하지만 절대 안 보임.

**코드로 확인했다 (ceo-review).** `hiddenHotkeySections`(`hidden_hotkeys.go:205-223`)의
Global 섹션은 `active:` 필드가 **아예 없어** 항상 `false`이고,
`visibleHiddenHotkeySections`(`:137-146`)가 `section.active`만 통과시키므로 **영구히
필터링된다.** 그게 task 8의 증상이다. 반면 Graph 섹션(`:224-226`)은
`active: m.activeSection == sectionGraph`이므로 graph가 활성일 때 렌더된다.

즉 **섹션 로컬 키는 `?`에서 정상으로 보인다.** 안 보이는 것은 Global 그룹이다.
`w`는 graph 섹션 전용 액션이므로 `hidden_hotkeys.go:231-249`의 graph 그룹에
넣으면 그 자리에서 보인다. task 8의 미결(Global을 어디에 노출할지)과 무관하다.

한 가지 한계: `w`는 graph가 활성 섹션일 때만 `?`에 보인다. Tags 섹션에서 `?`를 눌러도
안 보인다. 그건 섹션 인식 오버레이의 기존 설계이며 이 작업이 만드는 공백이 아니다.

**초안이 여기서 틀렸다가 고쳐졌다.** spec 초안은 `w`를 메인 footer에도 등록하라고
적었고, 그러면 `decisions.md:8-9`의 "footer에는 Global core 키만" 정책을 깨면서
task 8의 미결 영역으로 들어갔다. eng-review E-5/코덱스가 잡아 footer 등록을
철회했으므로, 지금 spec대로면 task 8과 겹치지 않는다.

**결론:** `11.8 deps = ['11.4']` 유지. task 8 의존 추가하지 않는다.
제약 상속: footer에 넣지 않는다. `?` 오버레이 graph 그룹에만 넣는다. 키 구현과
등록은 같은 커밋.

### R-4. 11.7 (`range:` 요약) — 6.6에 의존하지 **않는다**. 어휘 하나만 예약한다

**얽힘의 실체:** 11.2가 Details에 `date:`를 넣고, 같은 패널 tags 섹션에는 이미
`age:`가 있다. 6.6이 어휘 통일을 소유한다.

**11.2/11.7은 6.6의 결정을 입력으로 쓰지 않는다.** 다만 **세 번째 어휘를 만들면**
6.6의 일이 늘어난다. spec C-11이 이미 그 선을 그었다 — "tags 섹션의 `age:`는
건드리지 않는다. 세 번째 어휘를 만들지 않는 것까지만 이 spec이 지킨다."

**결론:** 의존 추가하지 않는다. 제약 상속: `date:`와 `range:` 두 라벨만 쓰고
`when:` / `time:` / `path:` 같은 변종을 만들지 않는다.
**6.6에 `date:`/`age:` 이원화를 정리 대상으로 명시했다** (초안 시점에는 6.6 details에 없었다).

### R-5. 11.9 (yymmdd 컬럼) — 6.4에 **진짜로** 의존한다. 그리고 6.4는 6.2에 의존한다

여기만 진짜 의존이다.

`shell_width_test.go:113-118`과 `view_shell.go:56`으로 측정한 현재 title 가용폭:
40컬럼 **-25**, 60컬럼 **-14**, 80컬럼 **-2**, 140컬럼 +30.

80컬럼 이하에서 고정 컬럼만으로 이미 패널 폭을 초과한다. **날짜 컬럼을 넣을 폭이
음수다.** 6.4가 폭을 회수해야 물리적으로 가능해지고, 6.4는 `deps=[2]`로 6.2(overflow
수정)에 의존한다. 이건 파일 겹침이 아니라 실제 입력 의존이다.

**결론:** `11.9 deps = ['11.1', '6.4']`로 **확장**한다. 사용자 결정(2026-09-09)에
따라 11.9는 보류 상태를 유지한다.

---

## 해소 후 그래프

```
  [지금 착수 가능]                    [사슬 뒤]

  11.1 ──┬──▶ 11.2                    6.2
         ├──▶ 11.3 ──▶ 11.5            └─▶ 6.6 ─▶ 6.7 ─▶ 6.8
         └──▶ (11.9 는 6.4 대기)              ▲              ▲
                                              │              │
  11.4 ──┬──▶ 11.6                     (11.7이 어휘   (11.5 뒤가 싸다.
         ├──▶ 11.7                      예약만 함)     의존은 아님 — C-1)
         └──▶ 11.8
                                        6.2 ─▶ 6.4 ─▶ 11.9 (보류)
```

**막힌 것은 11.9 하나다.** 나머지 8개(11.1~11.8)는 전부 지금 착수 가능하다.
사슬을 풀기 전에는 11.5가 4단 뒤에 갇혀 있었다.

## 병렬 가능성

두 갈래가 서로 독립이다.

| 갈래 | subtask | 공유 파일 |
|---|---|---|
| **날짜** | 11.1 → 11.2, 11.3 → 11.5 | `repo_parse.go`, `repo.go`, `repo_exec.go`, `contract.go`, `reader.go`, `view_detail.go`, `commit_inspector_screen.go` |
| **road** | 11.4 → 11.6, 11.7, 11.8 | `model.go`, `view_projection.go`, `graph_render.go`, `view_graph.go`, `key_handling_browse.go`, `hidden_hotkeys.go`, `view_detail.go` |

**충돌 지점 하나:** `view_detail.go`. 11.2가 `date:` 행을, 11.7이 `range:` 행을
같은 함수(`renderContextInfoLines`)에 넣는다. spec C-11이 삽입 순서를
`focus: → date: → range: → focusParentLines`로 못 박았으므로 순서 충돌은 없지만,
**같은 함수를 동시에 편집하면 머지 충돌이 난다.** 11.2를 먼저 끝내고 11.7이 그 위에
올라가는 것이 싸다.

`internal/git/repo_exec.go`도 둘 다 건드리지만 서로 다른 함수다 — 날짜는
`InspectCommit`(:157), road는 새 `AncestryPath`. 충돌하지 않는다.

## taskmaster에 반영할 변경

| task | 지금 | 바꿀 것 | 이유 |
|---|---|---|---|
| 11.5 | `deps=['11.3']` | 유지 | R-1 — 6.8 의존 아님 |
| 11.6 | `deps=['11.4']` | 유지 | R-2 — 6.3 의존 아님 |
| 11.8 | `deps=['11.4']` | 유지 | R-3 — task 8 의존 아님 |
| 11.7 | `deps=['11.4']` | `['11.4','11.2']` 추가 | 병렬 절 — `view_detail.go` 같은 함수 머지 충돌 회피 |
| 11.9 | `deps=['11.1']` | `['11.1','6.4']` | R-5 — 진짜 의존 |
| 6.8 | `deps=[7]` | `['6.7']` 로만 정규화 (11.5 추가 **안 함**) | C-1 — 순서 선호는 의존이 아니다. details 에 문장으로 |
| 6.6 | `deps=[2]` | 유지, details에 `date:`/`age:` 이원화 추가 | R-4 |
| 6.x 전체 | `deps=[2]`, `[1]`, `[7]`, `[6]` | 점 표기(`6.2`)로 정규화 | 11.x는 `'11.1'` 점 표기인데 6.x는 벌거벗은 숫자다. 두 관습이 섞여 의존을 기계로 읽을 수 없다 |

## 이번 루프에서 건너뛴 전제

사용자 지시: "결정사항이 전체를 좌우하지 않는다면 그 전제만 건너뛰고 나머지 plan을
모두 작성해."

| 건너뛴 전제 | 왜 전체를 좌우하지 않나 | 누가 결정해야 하나 |
|---|---|---|
| 6.8의 `parent:` 행을 만들지, `FROM`/`TO` 짝을 맞출지 | 11.5는 자리를 비우기만 하므로 어느 쪽이든 성립 | 6.8 계획 (다음 루프) |
| 6.6이 `date:`와 `age:` 중 무엇으로 통일할지 | 11.2/11.7은 두 라벨만 쓰고 변종을 안 만들므로 어느 쪽이든 성립 | 6.6 계획 (다음 루프) |
| task 8의 Global 키 노출 방식 (a/b/c/d 4안) | `w`는 섹션 로컬이라 Global 결정과 무관 | task 8 계획 (다음 루프) |
| 6.4의 컬럼 예산 재배분 방식 | 11.9만 막고 11.1~11.8은 무관 | 6.4 계획 (6.2 이후) |

네 전제 모두 **11.1~11.8을 막지 않는다.** 그래서 건너뛰고 이 문서를 완성했다.

## 다음 루프에서 쓸 계획 순서

의존이 얕은 것부터, 그리고 task 11을 실제로 푸는 것부터.

1. **6.8 계획** — 11.5의 역방향 의존이 붙었으므로 경계를 문서로 확정해야 한다.
2. **6.6 계획** — `date:`/`age:` 이원화가 새로 들어왔다.
3. **task 8 계획** — 4안(a/b/c/d)이 미결이고 `w` 등록이 이 결정 없이 가능한지
   R-3이 주장했으므로, 그 주장을 task 8 계획이 확인해야 한다.
4. **6.2 계획** — 6.4/11.9 사슬의 뿌리. 측정값(-25/-14/-2)이 이미 있다.
5. **6.3 계획** — `df7ed32`가 커서 부분을 이미 했으므로 남은 범위를 재산정해야 한다.
6. **6.4 계획** — 6.2 이후. 11.9의 날짜 컬럼을 포함해 산정.

---

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 1 | CLEAR | mode: HOLD_SCOPE, 3 findings, 0 critical gaps, 0 unresolved |
| Codex Review | `/codex review` | Independent 2nd opinion | 0 | — | not run on this document; ran on the plan it derives from |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 1 | CLEAR | ran against the spec, not this document; no code contract here to review |
| Design Review | `/plan-design-review` | UI/UX gaps | 0 | — | not applicable — this document holds no UI decisions |
| DX Review | `/plan-devex-review` | Developer experience gaps | 0 | — | not run |

**CEO FINDINGS (all applied):**

- **C-1 (P1, confidence 10/10)** The document violated its own definition. It defines a dependency as "A가 끝나야 B를 시작할 수 있다", then concluded with `6.8 dependencies += ['11.5']` on the rationale that 11.5 going first makes 6.8 cheaper. That is an ordering preference, not a start-blocking relation: 6.8 can start without 11.5 and would merely redo the FROM-tail work. Recording it as a hard dependency couples the design backlog to feature delivery, so a slip in 11.5 blocks 6.8 for no real reason. Removed from `dependencies`; the preference now lives in 6.8's `details`, which is exactly the split this document prescribes.
- **C-2 (P1, confidence 10/10)** `.taskmaster/tasks/tasks.json` task 6.8, D-018 — the ownership table presented a clean split while actually moving work out of 6.8. D-018 already owns the FROM-tail fix verbatim: "TO와 짝을 맞추거나 parent: 라벨로 바꿔 다른 header 행과 같은 key/value 처리를 준다." 11.5 does need it first, because the header already overflows before a date row is added, but an unannounced transfer means whoever picks up 6.8 finds part of their charter done and cannot tell whether they misread it. Now recorded as an explicit transfer, with 6.8's remainder named: the `parent:` row decision, the case-convention cleanup, and the 40-column copy priority.
- **C-3 (P3, confidence 10/10)** R-4 said the 6.6 vocabulary note "지금은 6.6 details에 없다" while the same commit added it. Corrected to past tense.

**VERIFICATION (R-3 promoted from assertion to evidence):** the R-3 claim that section-local keys are visible in the `?` overlay was asserted from task 8's prose. Confirmed in code: `hiddenHotkeySections` (`internal/app/hidden_hotkeys.go:205-223`) gives the Global section no `active:` field at all, so it is permanently false, and `visibleHiddenHotkeySections` (`:137-146`) passes only `section.active` — which is precisely task 8's symptom. The Graph section (`:224-226`) sets `active: m.activeSection == sectionGraph` and renders normally. One limit now stated in the document: `w` appears in `?` only while graph is the active section, which is the existing section-aware design rather than a gap this work introduces.

**VERDICT:** CEO CLEARED — the five resolutions stand, with R-1's conclusion corrected from a hard reverse dependency to a details-level ordering preference plus an explicit scope transfer. Net effect on the graph is unchanged: only 11.9 is blocked, and 11.1 through 11.8 can all start now. The dependency fields are now internally consistent with the document's own definition, which was the one thing this review had to check and initially failed.

NO UNRESOLVED DECISIONS
