# Task 4.1 구현 상세 계획: ANSI 우선 터미널 색상 호환성

작성일: 2026-07-31
브랜치: develop
대상: Task 4.1 터미널 색상 호환성
상태: 구현 완료, 수동 terminal 검증 대기

## 1. 목표와 확정 정책

현재 UI는 ANSI 256색(205, 226, 81), RGB(#00b7eb, #ff5ca8, #9D00FF), reverse, bold를 혼용한다. 흰색·베이지색 배경에서는 밝은 yellow/pink가 읽히지 않는다는 외부 피드백이 있었다.

이번 작업은 RGB 테마나 배경 감지를 추가하지 않는다. 사용자의 터미널 팔레트를 존중하는 ANSI-first 정책으로 기존 렌더링을 통일한다.

정책:

1. semantic style에는 lipgloss.ANSIColor(0..15)만 사용한다.
2. lipgloss.Color("16..255")와 lipgloss.Color("#...")를 제거한다.
3. section/graph hover는 ANSI yellow + bold foreground를 사용하고, 검색 focus와 popup 선택처럼 실제 focus가 필요한 상태만 Reverse(true).Bold(true)를 사용한다.
4. dirty, conflict, tag, stash, remote는 색상 외 marker/label/bold를 유지한다. tag 이름은 underline/link 효과 없이 ANSI magenta foreground로 표시한다.
5. Ascii, ANSI, ANSI256, TrueColor 프로파일을 테스트한다.
6. 레이아웃·키 바인딩·상태 전이는 변경하지 않는다.
7. 이번 구현의 파일 범위는 `theme.go`, shell, graph, graph search, 관련 테스트와 정책 문서로 제한한다. 개별 popup renderer의 기존 색상 이전은 후속 작업으로 분리한다.

## 1.1 사용자 정보 계층과 강조 우선순위

ANSI 색상 변경은 레이아웃을 바꾸지 않지만, 한 화면에서 사용자가 읽는 순서를 보존해야 한다. 구현자는 아래 우선순위를 기준으로 semantic style을 적용한다.

| 우선순위 | 사용자가 먼저 읽는 정보 | 대표 표시 | 허용 강조 |
|---|---|---|---|
| 1 | 현재 위치와 현재 작업 대상 | `CURRENT`, head marker, focused row | reverse + bold 또는 bold + ANSI semantic color |
| 2 | 위험하거나 작업을 막는 상태 | conflict, dirty, blocked, error | visible label/marker + bold + ANSI red 계열 |
| 3 | 사용자가 선택하거나 이동할 수 있는 대상 | target branch, selected stash/tag, search focus | section/graph hover는 ANSI yellow + bold foreground, search focus와 popup 선택은 reverse + bold |
| 4 | 구조와 관계 | graph connector, branch, remote/local provenance | ANSI blue/green/magenta + marker/label |
| 5 | 보조 설명과 조작법 | popup description, help, disabled action | terminal 기본 전경색; 색상 슬롯 없음 |

### 강조 충돌 규칙

1. 같은 문자열에 `selected/focus`와 `warning/error`가 동시에 적용되면 reverse/bold를 유지하고, visible label 또는 marker로 위험 상태를 보존한다. 색상 하나를 추가해 강조를 겹치지 않는다.
2. `CURRENT`와 `TARGET`이 동시에 표시되면 위치 marker와 텍스트 label을 우선하고, 두 상태 모두 같은 ANSI 색상으로 합치지 않는다.
3. graph connector에는 semantic 색상을 적용하지 않고, commit marker와 상태 label에만 semantic style을 적용한다. 연결선이 강조되어 본문보다 먼저 읽히면 안 된다.
4. popup header와 primary action은 body/help보다 먼저 읽혀야 하므로 bold를 유지한다. help와 disabled 텍스트는 bright black을 사용하지 않고 terminal 기본 전경색을 따른다.
5. 한 문자열에 foreground/background 색상을 동시에 지정하지 않는다. 배경색 사용은 이 작업의 ANSI-first 범위에서 금지한다.

### 화면별 3초 스캔 기준

| 화면 영역 | 첫 번째로 보여야 하는 것 | 두 번째 | 세 번째 | 실패 기준 |
|---|---|---|---|---|
| graph | 현재 commit/branch 위치 | conflict/dirty/target marker | branch/tag/hash metadata | connector나 muted help가 현재 위치보다 먼저 눈에 띔 |
| section list | 현재 section과 선택 row | action 가능 여부와 상태 label | provenance/hash/count | 색상 제거 시 선택 row와 일반 row가 구분되지 않음 |
| popup | popup 목적 또는 header | 현재 선택/오류/primary action | 설명과 help | header·오류·action이 body/help와 같은 강도로 보임 |
| search | 검색어와 focus match | 일반 match | 검색 결과 없음 또는 조작법 | 일반 match와 focus match가 같은 attribute로 출력됨 |

## 1.2 기능별 사용자 상태 계약

아래 표는 Task 4.1에서 색상만 변경해도 반드시 보존해야 하는 사용자-visible 상태다. 새 상태 전이나 문구를 추가하지 않고, 현재 renderer가 출력하는 문구와 marker를 기준으로 한다.

| 기능 | Loading | Empty | Error/Blocked | Success | Partial |
|---|---|---|---|---|---|
| graph/section | 기존 loading 문구와 `ok`/`warn` 상태 label을 유지한다. | `(empty)` 또는 `(no targets)`를 `muted`로 표시하고, 가능한 다음 action을 함께 둔다. | `(conflict)`, `(dirty)`, `Blocked` 같은 visible label을 `errorStyle`/`warn`과 함께 유지한다. | `Browse`, `Reset`, `Cherry-pick` 등 현재 action label과 current/target marker를 유지한다. | 로드된 항목은 정상 표시하고, 누락된 관계나 원격 상태는 `(unknown)`, `(no-up)` 같은 label로 구분한다. |
| search | 검색 중인 현재 popup/header와 입력값을 유지한다. | 결과가 없으면 빈 graph처럼 보이게 하지 말고 기존 no-result 문구와 `esc`/입력 조작법을 유지한다. | 잘못된 입력이나 검색 실패는 오류 문구를 `errorStyle`로 표시하고 입력을 지우지 않는다. | focus match는 `reverse + bold`, 일반 match는 `underline + bold`로 구분한다. | 검색 결과가 일부만 표시되면 match text는 유지하고, 잘린 결과는 기존 truncation 규칙을 따른다. |
| stash popup | `Loading` 상태 label과 popup header를 먼저 읽히게 한다. | `(no stash entries)`와 `esc: dismiss`를 유지한다. | stash message 오류와 pop 차단 경고를 `errorStyle`/`warn`으로 표시한다. | 선택한 stash와 `enter: jump`/`enter: pop` action을 유지한다. | 일부 stash만 표시될 때 `...` marker와 선택 상태를 유지한다. |
| tag/provenance | tag 동기화 중 상태 label과 현재 대상 정보를 유지한다. | `No local tags found.`와 `Press F to sync tag provenance.`를 유지한다. | `(origin)`, `(local)`, `(unknown)` 및 충돌/동기화 오류 label을 색상 없이도 읽을 수 있게 한다. | tag 이름은 underline/link 효과 없이 ANSI magenta foreground, provenance는 label로 표시한다. | local/remote tag 집합이 일부만 준비된 경우 provenance label을 숨기지 않고 현재 목록을 그대로 표시한다. |
| confirm/target-pick/branch-input popup | popup 목적과 현재 입력/선택값을 유지한다. | 선택 가능한 대상이 없으면 `(no selection)` 또는 `(no targets)`와 취소 조작법을 표시한다. | 차단 이유나 입력 오류를 popup header보다 낮지 않은 강도로 표시한다. | primary action은 `enter`와 선택 대상, 취소는 `esc`로 명확히 유지한다. | 긴 목록은 기존 `...`/truncation을 유지하고 현재 선택 row가 사라지지 않게 한다. |

### 상태 표현의 공통 불변식

1. `Ascii` 프로파일에서는 색상 escape sequence가 없어도 위 표의 visible text, marker, `bold`가 제공하는 의미를 잃지 않는다.
2. `ANSI`, `ANSI256`, `TrueColor` 프로파일에서는 semantic style이 RGB escape sequence로 변환되지 않는다.
3. 빈 상태는 빈 문자열이나 테두리만 출력하지 않는다. 상태 문구와 가능한 다음 action을 함께 출력한다.
4. 오류/차단 상태는 색상만으로 표현하지 않는다. 오류 문구, marker, 또는 `(conflict)`/`Blocked` 같은 label 중 하나 이상을 반드시 유지한다.
5. partial 상태는 정상적으로 로드된 항목을 숨기지 않고, 누락·미확정 상태를 `(unknown)`, `(no-up)`, `...` 같은 기존 신호로 표시한다.

## 1.3 사용자 여정 storyboard

Task 4.1은 새 화면이나 새 흐름을 만들지 않는다. 아래 storyboard는 기존 Graphkeeper 흐름에서 색상 호환성이 사용자의 판단을 어떻게 지원해야 하는지 정의한다.

| 시간/단계 | 사용자가 하는 일 | 사용자가 느껴야 하는 것 | 설계가 제공해야 하는 신호 | 검증 기준 |
|---|---|---|---|---|
| 첫 5초: 방향 파악 | 화면에 진입해 graph와 section list를 훑는다. | “현재 위치와 주요 대상이 바로 보인다.” | current/head marker는 1순위, target/selected는 2순위, connector와 help는 배경으로 유지한다. | bright/dark terminal 모두에서 current와 target label을 먼저 식별할 수 있다. |
| 5분: 상태 확인 | dirty/conflict/remote/tag provenance를 확인하고 popup 또는 search를 연다. | “무엇이 위험하고 어떤 대상이 작업 가능한지 믿을 수 있다.” | 위험 상태는 red 색상과 `(dirty)`, `(conflict)`, `Blocked` 같은 label을 함께 사용하고, focus는 reverse/bold로 구분한다. | Ascii profile에서도 위험 label과 focus row가 유지된다. |
| 5분: action 선택 | stash jump, reset, cherry-pick, branch 선택 중 하나를 고른다. | “다음 action과 취소 방법을 실수 없이 안다.” | popup header/primary action이 body/help보다 강하고, `enter`/`esc` 조작법이 항상 보인다. | loading/empty/error/success/partial 상태에서 action 또는 취소 경로가 사라지지 않는다. |
| 반복 사용: 신뢰 형성 | 여러 repository와 서로 다른 터미널 profile에서 같은 흐름을 반복한다. | “내 터미널 색상 설정이 존중되고 상태 의미가 바뀌지 않는다.” | ANSI 0..15 delegation을 사용하고, 색상 외 marker/label/attribute를 의미의 안전장치로 유지한다. | ANSI/ANSI256/TrueColor 출력에 RGB sequence가 없고, semantic visible text가 동일하다. |

### 여정 실패 조건

- 사용자가 current와 target을 구분하려고 색상표를 외워야 하면 실패다.
- 흰색·베이지색 배경에서 yellow/magenta가 보이지 않아도 `(dirty)`, `(conflict)`, `(origin)`, `(local)` 같은 텍스트 신호가 남아 있어야 한다.
- popup에서 오류가 발생했을 때 오류 문구가 muted help보다 약하게 보이면 실패다.
- 검색 focus와 일반 match가 같은 attribute로 출력되거나, 검색 결과 없음 상태에 다음 action이 없으면 실패다.

## 1.4 DESIGN.md와 ANSI semantic token의 관계

`DESIGN.md`의 색상은 제품 의미 토큰의 기준이며, Task 4.1은 그 의미를 삭제하지 않는다. 다만 terminal UI에서는 앱이 RGB 값을 소유하지 않고 ANSI 기본색과 terminal attribute를 출력한다. 최종 색상은 사용자의 terminal palette가 결정한다.

| DESIGN.md 의미 토큰 | Task 4.1 semantic style | 기본 표현 | 색상 외 안전장치 |
|---|---|---|---|
| graph/navigation/info | `sectionTitle`, `contextValue`, `remoteColor`, `reviewTarget` | ANSI blue 계열 | graph 위치, branch/provenance label, marker |
| warning/stash accent | `warn`, `stashMark`, `accent` | ANSI yellow 슬롯. terminal palette가 주황색에 가깝게 매핑할 수 있음 | `(dirty)`, `(no-up)`, stash marker, warning 문구 |
| error/conflict | `errorStyle`, `conflictColor`, `conflictMark` | ANSI red 계열 | `(conflict)`, `Blocked`, 오류 문구 |
| success/current | `ok`, `headMark`, `reviewCurrent` | ANSI green 계열 또는 bold | `CURRENT`, head marker, success action label |
| selection/focus | `branchMark`, `pointerMark`, `highlight`, `searchFocusMark` | ANSI yellow + bold for section/graph hover; reverse + bold for focus/selection | hover는 foreground, focus/selection은 reverse + bold |
| tag/provenance | `tagColor`, `tagOverlapColor` | ANSI magenta 계열 | `(origin)`, `(local)`, tag 이름 |
| muted/help/disabled | `muted`, `popupHelp`, `reviewFooter`, `disabled` | terminal 기본 전경색; ANSI 색상 슬롯 없음 | 문구와 배치, action label |

매핑 규칙:

1. 위 표의 의미 토큰 이름과 사용자-visible label은 `DESIGN.md`의 의미를 따른다.
2. ANSI 번호는 고정 화면색이 아니라 terminal palette 슬롯을 가리킨다. `ANSI blue`가 특정 RGB 값과 같다고 가정하지 않는다.
3. 3-bit ANSI에는 고정 orange 슬롯이 없으므로 warning은 ANSI yellow 슬롯에 위임한다. 사용자의 palette가 이를 주황색에 가깝게 표시할 수 있지만, 앱은 orange RGB 값을 강제하지 않는다.
4. warning과 error가 사용자의 palette에서 비슷하게 보일 수 있으므로 두 상태를 색상만으로 구분하지 않는다.
5. `DESIGN.md`의 RGB hex 값은 terminal renderer에 직접 복사하지 않는다. 문서의 브랜드/preview 색상 기준과 TUI 출력 정책을 분리한다.
6. 향후 사용자가 직접 theme를 설정하는 기능을 추가할 때는 Task 4.1의 ANSI-first 정책을 우회하지 말고 별도 설계 과제로 다룬다.

## 1.5 터미널 너비와 접근성 기준

Task 4.1은 반응형 레이아웃을 새로 설계하지 않는다. 대신 기존 width/truncation 규칙이 색상 변경 후에도 핵심 정보를 보존하는지 확인한다.

| Terminal width | 반드시 보이는 정보 | 줄여도 되는 정보 | 검증 |
|---:|---|---|---|
| 80열 이상 | popup header, current/target label, primary action, `enter`/`esc` help, 오류/충돌 label | 긴 hash의 표시 길이, 보조 설명 | 대표 popup과 graph renderer의 visible width 및 핵심 문자열 검사 |
| 60–79열 | popup 목적, 선택 row, primary action, 취소 키, 상태 marker | description, provenance 보조 문구, hash 길이 | `fitVisibleWidth`/기존 truncation 결과가 border를 넘지 않는지 검사 |
| 40–59열 | 선택/focus row, 오류·충돌 label, `enter`/`esc` | help 상세 문구, remote/tag metadata | popup body가 잘리지 않고 선택 row가 유지되는지 수동 검사 |
| 40열 미만 | 기존 최소 너비 동작을 따름. 핵심 오류/선택 label을 빈 문자열로 만들지 않는다. | layout에 의해 이미 생략되는 보조 정보 | 새 레이아웃을 추가하지 않고 현재 renderer의 최소 폭 동작을 기록 |

### 키보드와 무색 출력 불변식

1. 모든 popup에서 `enter`는 primary action, `esc`는 cancel/dismiss라는 기존 의미를 유지한다.
2. focus row는 색상 없이도 `reverse + bold`, selection marker, 또는 visible label 중 하나 이상으로 현재 대상을 식별할 수 있어야 한다.
3. warning/error/conflict는 foreground 색상이 제거되어도 오류 문구, `(dirty)`, `(conflict)`, `Blocked` 같은 텍스트 신호가 남아야 한다.
4. underline은 search match에만 사용하는 보조 신호다. tag 이름은 underline/link 효과 없이 foreground 색상과 텍스트 label로 구분하며, underline이 지원되지 않는 terminal에서도 search query match text를 제거하지 않는다.
5. 색상 대비를 수치로 가정하지 않는다. terminal palette를 직접 제어하지 않으므로 수동 검증은 dark graphite, white, beige 계열 terminal에서 semantic label과 attribute의 식별 여부를 확인한다.
6. 색상으로만 상태를 구분하는 새 style, background color, blinking attribute는 추가하지 않는다.

수동 검증 시나리오:

    # 각 터미널에서 실행
    graphkeeper

    # 확인할 조건
    # 1. 80/60/40열에서 graph, search, target-pick, confirm popup을 연다.
    # 2. current, target, dirty, conflict, empty, error 상태를 각각 만든다.
    # 3. ANSI palette가 dark, white, beige인 환경에서 label과 focus를 읽는다.
    # 4. Ascii/NO_COLOR 환경에서 색상 없이도 같은 상태를 판별한다.

## 2. 변경 범위

생성:
- internal/app/theme.go

수정:
- internal/app/view_shell.go
- internal/app/graph_render.go
- internal/app/graph_search_render.go
- internal/app/model_test.go
- docs/highlighting-color-map.md
- docs/decisions.md

수정 금지:
- .taskmaster/tasks/tasks.json
- cmd/graphkeeper/**, internal/git/**, internal/graph/**, internal/state/**
- 현재 작업 트리의 기존 사용자 변경

직접 색상 생성 위치 조사:

    rg -n 'lipgloss\.Color\(|Foreground\(lipgloss|Background\(lipgloss' internal/app --glob '*.go'

view_shell.go의 전역 style, shell에서 소유한 popup border/body/help, graph_render.go의 handshake, graph_search_render.go의 match/focus가 변경 대상이다. 개별 popup 파일의 직접 색상 생성은 이번 범위에서 건드리지 않는다.

## 3. internal/app/theme.go 구현

Lip Gloss/Termenv는 숫자 0..15를 ANSI 기본색으로, 16..255를 ANSI 256색으로 해석한다. lipgloss.ANSIColor는 truecolor profile에서도 RGB로 확장되지 않는다.

### 3.1 상수와 helper

    package app

    import "github.com/charmbracelet/lipgloss"

    const (
        ansiBlack         uint = 0
        ansiRed           uint = 1
        ansiGreen         uint = 2
        ansiYellow        uint = 3
        ansiBlue          uint = 4
        ansiMagenta       uint = 5
        ansiCyan          uint = 6
        ansiWhite         uint = 7
        ansiBrightBlack   uint = 8
        ansiBrightRed     uint = 9
        ansiBrightGreen   uint = 10
        ansiBrightYellow  uint = 11
        ansiBrightBlue    uint = 12
        ansiBrightMagenta uint = 13
        ansiBrightCyan    uint = 14
        ansiBrightWhite   uint = 15
    )

    func ansiStyle(color uint) lipgloss.Style {
        return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(color))
    }

    func ansiBoldStyle(color uint) lipgloss.Style {
        return ansiStyle(color).Bold(true)
    }

ANSIColor 번호는 0–15 범위를 벗어나면 안 된다.

    rg -o 'lipgloss\.ANSIColor\([0-9]+\)' internal/app --glob '*.go' \
      | sed -E 's/.*ANSIColor\(([0-9]+)\).*/\1/' \
      | awk '$1 < 0 || $1 > 15 { print; invalid=1 } END { exit invalid }'

### 3.2 semantic style

view_shell.go의 기존 전역 style 선언을 삭제하고 아래 style을 theme.go에 둔다.

    var (
        border = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
        baseBox = lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.ANSIColor(ansiBrightBlack)).
            Padding(0, 1)
        activeBox = lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.ANSIColor(ansiMagenta)).
            Padding(0, 1)

        title = lipgloss.NewStyle().Bold(true)
        sectionTitle = ansiBoldStyle(ansiBlue)
        contextValue = ansiStyle(ansiBlue)
        hotkey = ansiBoldStyle(ansiMagenta)
        muted = lipgloss.NewStyle()
        accent = ansiBoldStyle(ansiYellow)
        warn = ansiBoldStyle(ansiYellow)
        ok = ansiBoldStyle(ansiGreen)
        disabled = lipgloss.NewStyle()

        headMark = ansiBoldStyle(ansiGreen)
        branchMark = ansiBoldStyle(ansiYellow)
        pointerMark = ansiBoldStyle(ansiYellow)
        dirtyMark = ansiBoldStyle(ansiRed)
        conflictColor = ansiBoldStyle(ansiRed)
        conflictMark = ansiBoldStyle(ansiRed)
        stashMark = ansiBoldStyle(ansiYellow)
        remoteColor = ansiStyle(ansiBlue)
        tagColor = ansiStyle(ansiMagenta)
        tagOverlapColor = ansiBoldStyle(ansiMagenta)
        highlight = lipgloss.NewStyle().Reverse(true).Bold(true)

        reviewCurrent = headMark
        reviewTarget = ansiBoldStyle(ansiBlue)
        reviewBase = ansiBoldStyle(ansiBrightBlack)
        reviewHash = lipgloss.NewStyle().Bold(true)
        reviewBranch = lipgloss.NewStyle()
        reviewMark = ansiStyle(ansiBrightBlack)
        reviewCount = lipgloss.NewStyle().Bold(true)
        reviewFooter = lipgloss.NewStyle()

        popupBorder = lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.ANSIColor(ansiMagenta))
        popupBody = lipgloss.NewStyle()
        popupHelp = lipgloss.NewStyle()
        popupHeader = lipgloss.NewStyle().Bold(true)
        errorStyle = ansiBoldStyle(ansiRed)
        handshakeMark = lipgloss.NewStyle().Reverse(true).Bold(true)
    )

tagColor는 기존 RGB 보라색의 단순 치환이지만 underline/link 효과는 제거한다. ANSI magenta foreground로 표시하고, tag overlap만 bold를 추가한다. section/graph hover와 handshake는 고정 배경색 대신 terminal foreground attribute를 사용한다. 검색 focus와 popup 선택은 focus 상태이므로 reverse/bold를 유지한다.

## 4. call site 이전

### 4.1 popup renderer

view_shell.go가 소유한 직접 style 생성과 공통 shell style 사용 지점을 아래 형태로 바꾼다.

기존:

    descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
    helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
    popupBox := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("205")).
        Padding(1, 2).
        Width(width)

변경:

    descStyle := popupBody
    helpStyle := popupHelp
    popupBox := popupBorder.
        Padding(1, 2).
        Width(width)

각 함수의 기존 Width, Padding, Align 값은 유지한다. 적용 함수는 shell의 renderConfirmPopup, renderReviewPopup, renderResetModePopup, renderLoadingPopup, renderTargetPickPopup, renderBranchInputPopup, renderGraphSearchPopup이다. `view_alert.go`, `stash_popup.go`, `tagging.go`, `cherry_pick_view.go`, `hidden_hotkeys.go`는 이번 구현에서 제외한다.

오류와 경고는 다음처럼 바꾼다.

    // 기존
    lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(message)

    // 변경
    errorStyle.Render(message) // 오류
    warn.Render(message)  // 경고

### 4.2 graph handshake

graph_render.go의 세 곳에 반복되는 아래 style을 제거한다.

    pinkBg := lipgloss.NewStyle().
        Background(lipgloss.Color("162")).
        Foreground(lipgloss.Color("255")).
        Bold(true)

동일한 세 곳을 다음으로 바꾼다.

    graphCell = strings.ReplaceAll(graphCell, "*", handshakeMark.Render("*"))

isHandshake 조건, graph cell padding, VIRTUAL_CONFLICT_HASH 예외는 변경하지 않는다.

### 4.3 graph search

theme.go에 다음 style을 추가하고, graph_search_render.go에서는 이 style을 사용만 한다.

    var (
        searchMatchMark = lipgloss.NewStyle().Underline(true).Bold(true)
        searchFocusMark = lipgloss.NewStyle().Reverse(true).Bold(true)
    )

highlightSearchText의 정규식과 literal fallback은 변경하지 않는다. 일반 match는 underline/bold, focus는 reverse/bold로 구분한다.

## 5. 프로파일 테스트

### 5.1 helper

internal/app/model_test.go의 forceTrueColorProfile을 아래 helper로 교체한다.

    func withColorProfile(t *testing.T, profile termenv.Profile) {
        t.Helper()
        previous := lipgloss.ColorProfile()
        lipgloss.SetColorProfile(profile)
        t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
    }

    func allColorProfiles() []struct {
        name string
        profile termenv.Profile
    } {
        return []struct {
            name string
            profile termenv.Profile
        }{
            {"ascii", termenv.Ascii},
            {"ansi", termenv.ANSI},
            {"ansi256", termenv.ANSI256},
            {"truecolor", termenv.TrueColor},
        }
    }

전역 profile을 바꾸므로 이 테스트에는 t.Parallel()을 사용하지 않는다.

### 5.2 ANSI 색상 유지

    func TestSemanticWarningUsesANSIColorAcrossProfiles(t *testing.T) {
        for _, tc := range allColorProfiles() {
            t.Run(tc.name, func(t *testing.T) {
                withColorProfile(t, tc.profile)
                got := warn.Render("warning")
                if tc.profile == termenv.Ascii {
                    if got != "warning" {
                        t.Fatalf("got %q", got)
                    }
                    return
                }
                if strings.Contains(got, "38;2;") {
                    t.Fatalf("semantic warning must not become RGB: %q", got)
                }
                if !strings.Contains(got, "31") ||
                    ansi.Strip(got) != "warning" {
                    t.Fatalf("unexpected warning style: %q", got)
                }
            })
        }
    }

### 5.3 검색과 handshake

    func TestSearchHighlightUsesTerminalAttributes(t *testing.T) {
        withColorProfile(t, termenv.TrueColor)
        match := highlightSearchText("Add branch", "branch", false)
        focus := highlightSearchText("Add branch", "branch", true)

        for name, got := range map[string]string{
            "match": match,
            "focus": focus,
        } {
            if strings.Contains(got, "38;2;") ||
                strings.Contains(got, "48;2;") {
                t.Fatalf("%s uses RGB: %q", name, got)
            }
            if ansi.Strip(got) != "Add branch" {
                t.Fatalf("%s changed text", name)
            }
        }
        if !strings.Contains(match, "4") &&
            !strings.Contains(match, "1") {
            t.Fatalf("match lacks underline/bold")
        }
        if !strings.Contains(focus, "7") {
            t.Fatalf("focus lacks reverse")
        }
    }

    func TestHandshakeUsesTerminalAttributes(t *testing.T) {
        withColorProfile(t, termenv.TrueColor)
        got := handshakeMark.Render("*")
        if strings.Contains(got, "38;2;") ||
            strings.Contains(got, "48;2;") {
            t.Fatalf("RGB: %q", got)
        }
        if !strings.Contains(got, "7") || ansi.Strip(got) != "*" {
            t.Fatalf("bad handshake: %q", got)
        }
    }

전체 escape sequence를 비교하지 않는다. Lip Gloss 버전에 따라 attribute 순서가 달라질 수 있으므로 RGB sequence 부재, 필요한 attribute, visible text만 검사한다.

### 5.4 색상이 없어도 상태가 남는지 검증

    func TestCriticalStatesRetainVisibleLabelsWithoutColor(t *testing.T) {
        withColorProfile(t, termenv.Ascii)
        tests := []struct{ name, got, want string }{
            {"dirty", dirtyMark.Render("(dirty)"), "(dirty)"},
            {"conflict", conflictMark.Render("(conflict)"), "(conflict)"},
            {"stash", stashMark.Render("*"), "*"},
            {"origin", renderTagProvenanceStateLabel(true, true, true), "(origin)"},
            {"local", renderTagProvenanceStateLabel(true, true, false), "(local)"},
        }
        for _, tc := range tests {
            t.Run(tc.name, func(t *testing.T) {
                if ansi.Strip(tc.got) != tc.want {
                    t.Fatalf("got %q, want %q", ansi.Strip(tc.got), tc.want)
                }
            })
        }
    }

기존 38;2;..., 38;5;... assertion은 특정 색상값 대신 ANSI attribute, visible label, semantic distinction을 검사하도록 바꾼다. forceTrueColorProfile 호출은 withColorProfile(t, termenv.TrueColor)로 교체한다.

### 5.5 보조 스타일이 terminal 기본 전경색을 사용하는지 검증

muted/help/disabled/footer는 ANSI 색상 슬롯을 사용하지 않는다는 계약을 profile matrix로 직접 검증한다.

    func TestSecondaryStylesUseTerminalForeground(t *testing.T) {
        styles := []struct {
            name  string
            style lipgloss.Style
        }{
            {"muted", muted},
            {"popup-help", popupHelp},
            {"disabled", disabled},
            {"review-footer", reviewFooter},
        }

        for _, profileCase := range allColorProfiles() {
            t.Run(profileCase.name, func(t *testing.T) {
                withColorProfile(t, profileCase.profile)
                for _, styleCase := range styles {
                    t.Run(styleCase.name, func(t *testing.T) {
                        got := styleCase.style.Render("help text")
                        if ansi.Strip(got) != "help text" {
                            t.Fatalf("visible text changed: %q", got)
                        }
                        if strings.Contains(got, "38;") ||
                            strings.Contains(got, "48;") ||
                            strings.Contains(got, "38;2;") ||
                            strings.Contains(got, "48;2;") {
                            t.Fatalf("secondary style must use terminal foreground: %q", got)
                        }
                    })
                }
            })
        }
    }

이 테스트는 “escape sequence가 아예 없음”을 profile 구현 세부사항으로 강제하지 않는다. 다만 foreground/background 색상 지정이 없고 visible text가 보존되는지 확인한다. `bold`/`underline` 같은 별도 attribute를 추가할 경우에는 해당 semantic style의 목적에 맞는 attribute assertion을 별도로 둔다.

### 5.6 NO_COLOR 실행 조건 smoke test

현재 코드베이스에는 `NO_COLOR`를 직접 읽어 Lip Gloss profile을 설정하는 startup 로직이 없다. Task 4.1에서는 새 환경 변수 처리기를 추가하지 않는다. 대신 실제 실행 조건에서 핵심 visible state가 유지되는지 별도 smoke test로 확인한다.

    NO_COLOR=1 go test ./internal/app \
      -run 'TestSecondaryStylesUseTerminalForeground|TestCriticalStatesRetainVisibleLabelsWithoutColor|TestSearchHighlightUsesTerminalAttributes|TestHandshakeUsesTerminalAttributes' \
      -count=1

검증 기준:

1. 테스트 프로세스가 `NO_COLOR=1` 환경에서 시작된다.
2. current/target, dirty/conflict, search focus, handshake marker의 visible text가 유지된다.
3. `38;2;`, `48;2;` RGB sequence가 출력되지 않는다.
4. 이 smoke test는 startup에서 `NO_COLOR`를 해석하는 새 기능을 요구하지 않는다. 실제 색상 비활성화 정책이 필요해지면 별도 task로 분리한다.

## 6. 문서 동기화

docs/highlighting-color-map.md를 색상 번호 목록이 아닌 semantic token 목록으로 바꾼다.

    ## selected / hover

    - section/graph hover 색상: ANSI yellow (3)
    - section/graph hover 스타일: bold foreground
    - search focus/popup 선택 스타일: reverse + bold
    - 사용처: section row, graph target, target picker, stash list
    - 이유: hover는 배경을 칠하지 않고 terminal foreground를 사용하며, 실제 focus/선택만 reverse를 유지한다.

    ## tag

    - 색상: ANSI magenta (5)
    - 스타일: foreground only; overlap은 bold
    - 사용처: graph marker, tag section, detail tag list
    - 보조 신호: (local), (origin), tag 이름

docs/decisions.md에 다음을 추가한다.

    ## 2026-07-31: 터미널 색상은 ANSI 우선 semantic palette를 따른다

    - semantic style에는 ANSI 기본색 0–15만 사용한다.
    - RGB와 ANSI 256색을 semantic style에 혼용하지 않는다.
    - section/graph hover는 ANSI yellow + bold foreground를 사용하고, 검색 focus와 popup 선택은 reverse/bold를 유지한다.
    - dirty, conflict, tag provenance, stash는 visible label 또는 marker를 유지한다.
    - Task 4.1에서는 배경 감지와 사용자 테마 설정을 추가하지 않는다.

## 7. 구현 순서

1. theme.go를 추가하고 view_shell.go의 전역 style을 이동한다.
2. popup 파일의 직접 색상 생성을 공통 style로 교체한다.
3. graph handshake를 handshakeMark로 교체한다.
4. graph search를 underline/reverse style로 교체한다.
5. 프로파일 helper와 새 테스트를 추가한다.
6. 기존 truecolor 전용 assertion을 수정한다.
7. 색상 map과 decision 문서를 갱신한다.
8. 전체 검증을 실행한다.

각 논리 단위마다 실행:

    go test ./internal/app

## 8. 검증 명령

    rg -n 'lipgloss\.Color\("(1[6-9]|[2-9][0-9]|1[0-9]{2}|2[0-4][0-9])"\)|lipgloss\.Color\("#' internal/app/theme.go internal/app/view_shell.go internal/app/graph_render.go internal/app/graph_search_render.go
    rg -n '38;2;|48;2;' internal/app/theme.go internal/app/view_shell.go internal/app/graph_render.go internal/app/graph_search_render.go
    gofmt -w internal/app/theme.go internal/app/view_shell.go internal/app/graph_render.go internal/app/graph_search_render.go internal/app/model_test.go
    go test ./internal/app
    ./scripts/test
    ./scripts/check
    git diff --check

첫 번째 검색은 semantic style의 RGB/ANSI256 사용이 없어야 함을 확인한다. 두 번째 검색은 코드에 RGB escape sequence를 직접 적지 않았는지 확인한다.

수동으로 어두운 터미널과 밝은 터미널에서 Graph pointer, graph의 stash/tag marker, Local/Remote/Tags selected row, provenance, dirty/conflict, search match/focus, shell의 confirm/review/loading/target-pick popup을 확인한다. Ascii profile에서도 (dirty), (conflict), (local), (origin), *, ⬆, ⬇, Blocked가 남아 기능 사용이 가능해야 한다. 개별 alert/stash/tag/cherry-pick/hidden-hotkeys popup은 후속 범위다.

## 9. 실패 처리

- 색상 profile 테스트가 간섭하면 t.Parallel()을 제거하고 helper의 t.Cleanup 복원을 확인한다.
- 밝은 배경에서 ANSI 색상이 약하면 RGB를 추가하지 말고 bold, underline, reverse, marker, label 순서로 보강한다.
- exact escape sequence가 깨지면 역사적 색상값 assertion을 제거하고 visible text, RGB 부재, attribute, 상태 구분을 검사한다.
- popup 폭·padding·정렬이 변하면 색상 변경과 레이아웃 변경이 섞인 것이므로 기존 값을 복원한다.

## 10. 완료 조건

- [ ] theme.go에 semantic style이 중앙화됨
- [ ] semantic style에서 RGB와 ANSI 256색 하드코딩 제거
- [ ] shell popup, handshake, search가 공통 정책 사용
- [ ] selected 상태가 reverse/bold 또는 동등한 비색상 신호 사용
- [ ] critical state에 visible marker/label 또는 bold/underline 존재
- [ ] Ascii/ANSI/ANSI256/TrueColor 테스트 추가
- [ ] truecolor 전용 assertion 수정
- [ ] 어두운·밝은 터미널 수동 검증 완료
- [ ] highlighting-color-map.md와 decisions.md 갱신
- [ ] scripts/check, scripts/test, git diff --check 통과
- [ ] .taskmaster/tasks/tasks.json과 기존 사용자 변경을 건드리지 않음

## 11. 구현 시작 명령

    git status --short
    rg -n 'lipgloss\.Color\(|Foreground\(lipgloss|Background\(lipgloss' internal/app/theme.go internal/app/view_shell.go internal/app/graph_render.go internal/app/graph_search_render.go
    sed -n '1,80p' internal/app/view_shell.go
    sed -n '1,150p' internal/app/graph_render.go
    sed -n '1,80p' internal/app/graph_search_render.go

첫 구현은 internal/app/theme.go 추가다. 이후 3장의 semantic style을 기준으로 call site를 이전하고 5장의 프로파일 테스트를 추가한다.

## 12. What already exists

- `internal/app/view_shell.go`에 전역 style 정의가 이미 있어 `theme.go`로 이동할 수 있다.
- `internal/app/model_test.go`에 `forceTrueColorProfile` helper와 stash, confirm, tag popup 테스트가 이미 있다. 이번 범위에서는 helper와 graph/shell 관련 테스트만 확장하고, 개별 popup 테스트는 후속 범위의 기준으로 재사용한다.
- `internal/app/graph_render.go`에는 stash/tag/HEAD/conflict marker 렌더링 경로가 이미 있다. Task 4.1은 해당 상태 계산을 바꾸지 않고 style만 교체한다.
- `internal/app/graph_search_render.go`에는 검색 매칭 알고리즘과 literal fallback이 이미 있다. Task 4.1은 query 처리나 매칭 결과를 변경하지 않는다.
- `docs/highlighting-color-map.md`가 현재 색상 사용처를 목록화하고 있어 semantic token 문서의 출발점으로 재사용한다.
- Lip Gloss의 `ColorProfile`과 Termenv의 `Ascii`, `ANSI`, `ANSI256`, `TrueColor`가 이미 의존성에 포함되어 있다.

## 13. NOT in scope

- 터미널 배경 자동 감지: ANSI 위임 정책으로 결정했으며, 감지 실패와 multiplexer fallback을 별도 과제로 남긴다.
- RGB 전용 또는 ANSI+RGB 선택 테마: 사용자 설정과 테마 배포가 필요해 Task 4.1 범위를 넘는다.
- Renderer 주입형 Theme 객체: 현재 package-level style을 중앙화하는 것으로 충분하다.
- 레이아웃·popup 폭·padding·정렬 변경: 색상 변경과 레이아웃 변경을 섞지 않는다.
- 키 바인딩, 상태 전이, Git 명령, 네트워크 동작 변경: 이 작업의 사용자 행동 계약에 영향이 없다.
- 실제 terminal background를 자동으로 판정하는 E2E harness: 수동 dark/light 검증으로 분리한다.
- `view_alert.go`, `stash_popup.go`, `tagging.go`, `cherry_pick_view.go`, `hidden_hotkeys.go`의 직접 색상 이전: 이번 diff의 회귀 표면을 줄이기 위해 shell/graph/search 핵심 경로와 분리한다.

## 14. Engineering review decisions

### Architecture

구조적 blocking issue 없음. 기존 `internal/app` 렌더링 경계를 재사용하고 새 서비스나 런타임 계층을 추가하지 않는다. 범위 축소 후 구현 파일 5개와 문서 2개만 수정하며, 새 클래스·서비스 없이 기존 직접 style 생성의 이전이다.

범위 challenge 결정: 전체 12개 파일 대신 핵심 5개 구현 파일과 문서 2개만 이번 task에 포함한다. 개별 popup 파일의 직접 색상 이전은 후속 task로 분리하고, 이번 검증 명령도 in-scope 파일만 대상으로 한다.

### Code quality

1. `error` style 변수는 Go의 predeclared identifier와 혼동되므로 `errorStyle`로 명명한다.
2. `searchMatchMark`와 `searchFocusMark`도 `theme.go`로 이동한다. 검색 semantic style만 `graph_search_render.go`에 남겨 중앙화 원칙을 깨지 않는다.
3. `muted`, `popupHelp`, `disabled`, `reviewFooter`는 ANSI bright black을 사용하지 않고 terminal 기본 전경색을 따른다. 보조 정보의 약화는 문구·배치·기존 marker로 표현한다.
4. 색상 잔존 검색 명령은 scope reduction과 일치하도록 `theme.go`, shell, graph, graph search, 관련 테스트만 검사한다.

구현 예시:

    errorStyle = ansiBoldStyle(ansiRed)
    searchMatchMark = lipgloss.NewStyle().Underline(true).Bold(true)
    searchFocusMark = lipgloss.NewStyle().Reverse(true).Bold(true)

### Test review

1. 범위에 포함된 semantic style과 graph/shell/search call site를 Ascii/ANSI/ANSI256/TrueColor profile matrix로 검증한다.
2. `model_test.go`의 기존 graph, shell overlay, search 테스트를 table-driven profile 테스트로 확장하고 개별 popup 테스트는 중복 수정하지 않는다.
3. 범위에 포함된 loading, target pick, search, graph handshake renderer에 대표 출력 smoke test를 추가한다. 개별 alert/stash/tag/cherry-pick/hidden-hotkeys renderer는 후속 범위다.
4. `NO_COLOR=1` 실행 조건에서 핵심 visible state와 RGB sequence 부재를 확인한다. startup의 `NO_COLOR` 해석 기능은 추가하지 않는다.

### Design review

초기 디자인 완성도는 6/10이었다. 아래 보완 후 8/10으로 평가한다.

1. 정보 계층을 추가해 current/target, 위험 상태, 선택/focus, 구조, 보조 설명의 읽기 순서를 고정한다.
2. graph, section, search, shell popup의 loading/empty/error/success/partial 사용자 상태 계약을 추가한다. 개별 popup 파일은 후속 범위다.
3. 첫 5초 방향 파악, 5분 상태 확인과 action 선택, 반복 사용 신뢰 형성의 사용자 여정을 추가한다.
4. `DESIGN.md` 의미 토큰과 ANSI 슬롯의 책임을 분리하고, fixed orange RGB를 사용하지 않는 규칙을 추가한다.
5. 80/60/40열 terminal과 Ascii/NO_COLOR의 핵심 정보 보존 기준을 추가한다.
6. warning은 ANSI yellow 슬롯, error/conflict는 ANSI red 슬롯으로 분리하며, marker/label을 색상 외 안전장치로 유지한다.

## 15. Test coverage diagram

    semantic token matrix
      ├── warning / error / ok / muted      ──> profile matrix + visible text
      ├── selected / branch / pointer       ──> ANSI yellow + bold foreground + row text
      ├── current / remote / tag / stash    ──> profile matrix + marker/label (in-scope graph paths)
      ├── muted / popupHelp / disabled      ──> profile matrix + no foreground/background color
      ├── review styles                     ──> review popup output
      └── shell popup styles                ──> shell popup output

    renderer call sites
      ├── view_shell popups                ──> existing tests + representative smoke tests
      ├── shell confirm/review/loading      ──> existing + representative output tests
      ├── graph handshake                  ──> reverse + no RGB sequence
      └── graph search                     ──> underline/reverse + visible query text

    environment smoke
      └── NO_COLOR=1                       ──> visible labels + no RGB sequence

    profile matrix
      ├── Ascii       ──> no escape, labels and markers remain
      ├── ANSI        ──> ANSI 0..15 output
      ├── ANSI256     ──> ANSIColor remains 4-bit, no RGB output
      └── TrueColor   ──> ANSIColor remains 4-bit, no RGB output

각 branch는 다음을 모두 확인한다.

- visible text가 사라지지 않는다.
- `38;2;`와 `48;2;` RGB sequence가 나오지 않는다.
- section/graph hover는 ANSI yellow + bold foreground, 검색 focus와 popup 선택은 reverse/bold, tag는 ANSI magenta foreground, 오류는 ANSI red/bold 같은 의도한 attribute를 가진다.
- popup의 기존 폭·padding·정렬이 유지된다.

## 16. Failure modes

| 실패 상황 | 테스트 | 오류 처리 | 사용자 결과 |
|---|---|---|---|
| ANSI profile이 RGB sequence를 출력 | profile matrix | 테스트 실패 | CI에서 즉시 확인 |
| Ascii profile에서 label이 색상과 함께 사라짐 | visible label test | 렌더러 수정 필요 | 상태가 텍스트로 남아야 함 |
| 범위 내 shell popup이 공통 style을 사용하지 않음 | renderer smoke test | 테스트 실패 | shell popup만 옛 색상으로 보이는 회귀 방지 |
| profile 전역 상태가 테스트 사이에 남음 | helper cleanup test 실행 | `t.Cleanup` 복원 | 후속 테스트 오염 방지 |
| ANSI 색상이 밝은 배경에서 약함 | dark/light 수동 검증 | bold/underline/marker 보강 | critical state를 색상 외 신호로 확인 |
| popup style 이전 중 폭·정렬 변경 | 기존 width/height 테스트 | 기존 값 복원 | 레이아웃 회귀 방지 |
| warning과 error가 같은 의미로 보임 | warning/error semantic assertion | warning은 ANSI yellow 슬롯, error/conflict는 ANSI red 슬롯으로 고정 | 주의와 차단 상태를 구분 |
| 좁은 terminal에서 핵심 action이 잘림 | 80/60/40열 renderer 검사 | 기존 truncation과 핵심 label 보존 | 사용자가 다음 action과 취소 키를 확인 |
| NO_COLOR 실행에서 핵심 상태가 사라짐 | `NO_COLOR=1` smoke test | visible label/marker 유지 | 색상 없이도 작업 흐름 유지 |

이번 작업에서 silent failure가 되는 critical path는 없다. 자동 테스트는 escape sequence와 visible text를 확인하고, 실제 terminal palette 차이는 수동 검증에서 확인한다.

## 17. 병렬화 전략

순차 구현이다. 모든 구현 단계가 `internal/app`의 동일한 package-level style과 테스트 파일을 공유하므로 독립 worktree로 나눌 실익이 없다.

    theme.go style 정의
        ↓
    call site 이전
        ↓
    기존 테스트 profile matrix 확장
        ↓
    문서 갱신과 전체 검증

`docs/highlighting-color-map.md`와 `docs/decisions.md`는 코드 변경과 독립적으로 작성할 수 있지만, 최종 token 이름과 매핑이 확정된 뒤 갱신해야 하므로 별도 병렬 lane으로 분리하지 않는다.

## 18. Implementation Tasks

리뷰에서 승인된 구현 작업이다. 이 목록은 구현을 수행하지 않으며, 다음 구현자가 실행할 작업만 정의한다.

- [x] **T1 (P1, human: ~30분 / CC: ~5분)** — semantic theme 추가 — `internal/app/theme.go`를 만들고 ANSI 0..15 token과 `errorStyle`, popup style, search style을 중앙화한다. `warn`은 ANSI yellow 슬롯, `errorStyle`/`conflictMark`는 ANSI red 슬롯을 사용한다. 검증: `go test ./internal/app`.
  - Surfaced by: Code quality review와 Pass 7 — warning/error semantic token을 고정해야 함.
  - Files: `internal/app/theme.go`.
  - Verify: `go test ./internal/app`.
- [x] **T2 (P1, human: ~35분 / CC: ~7분)** — 핵심 renderer call site 이전 — shell popup, handshake, search의 직접 RGB/ANSI256 style 생성을 제거하고 정보 계층의 충돌 규칙을 적용한다. 개별 popup renderer는 제외한다. 검증: 색상 검색 명령과 기존 renderer 테스트.
  - Surfaced by: Pass 1 Information Architecture — current/target, 위험 상태, focus의 우선순위 유지.
  - Files: `internal/app/view_shell.go`, `internal/app/graph_render.go`, `internal/app/graph_search_render.go`.
  - Verify: `rg -n 'lipgloss\\.Color\\(|38;2;|48;2;' internal/app/theme.go internal/app/view_shell.go internal/app/graph_render.go internal/app/graph_search_render.go`.
- [x] **T3 (P1, human: ~40분 / CC: ~8분)** — 핵심 profile matrix 테스트 — 범위 내 semantic token을 Ascii/ANSI/ANSI256/TrueColor에서 검증하고 RGB sequence 부재, visible label, warning/error 분리, muted/help/disabled/reviewFooter의 기본 전경색 계약을 확인한다. 검증: `go test ./internal/app -run 'Color|Search|Handshake|Popup|Graph'`.
  - Surfaced by: Pass 2 Interaction State Coverage와 Pass 7 — 프로파일별 상태 의미 보존.
  - Files: `internal/app/model_test.go`, `internal/app/graph_search_test.go`.
  - Verify: profile matrix, `ansi.Strip` visible text assertion, `38;`/`48;` foreground/background 부재 assertion.
- [x] **T4 (P1, human: ~25분 / CC: ~5분)** — 핵심 renderer smoke test — shell의 confirm/review/loading/target-pick과 graph/search 출력에서 80/60/40열의 header, focus, primary action, `enter`/`esc`가 유지되는지 확인한다. 개별 popup은 후속 범위다.
  - Surfaced by: Pass 2, Pass 6 — 상태 계약과 terminal 너비/키보드 접근성.
  - Files: `internal/app/model_test.go`, `internal/app/graph_search_test.go`.
  - Verify: visible text, popup width, primary/cancel key assertion.
- [x] **T5 (P2, human: ~20분 / CC: ~4분)** — 문서 동기화 — `docs/highlighting-color-map.md`를 semantic token 기준으로 갱신하고 `docs/decisions.md`에 ANSI-first, DESIGN.md 관계, fixed orange 금지 결정을 추가한다.
  - Surfaced by: Pass 5 Design System Alignment — 의미 토큰과 ANSI 출력 정책의 분리.
  - Files: `docs/highlighting-color-map.md`, `docs/decisions.md`.
  - Verify: 문서의 token/ANSI 슬롯/label 규칙과 구현의 semantic style 이름 대조.
- [ ] **T6 (P1, human: ~35분 / CC: ~6분)** — 전체 검증 — dark/light/beige terminal 수동 검증, 80/60/40열 시나리오, `NO_COLOR=1` smoke test, `./scripts/test`, `./scripts/check`, `git diff --check`를 실행한다.
  - Surfaced by: Pass 3 User Journey와 Pass 6 Responsive & Accessibility — 첫 5초 방향 파악과 반복 사용 신뢰 검증.
  - Files: 없음. 구현 후 검증만 수행.
  - Verify: 수동 시나리오와 `./scripts/test`, `./scripts/check`, `git diff --check`.

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| Eng Review | `/plan-eng-review` | Architecture, code quality, tests, performance | 2 | ISSUES_RESOLVED | 4 issues, 0 critical gaps |
| CEO Review | — | Not run | 0 | — | Not needed for this scoped fix |
| Design Review | `/plan-design-review` | Terminal UI hierarchy, states, accessibility | 1 | ISSUES_RESOLVED | score: 6/10 → 8/10, 7 decisions |
| Outside Voice | — | Not run | 0 | — | Not requested |

- **VERDICT:** ENG + DESIGN CLEARED — implementation may proceed after the approved scope and test decisions are applied.
- **Scope:** reduced from 12 files to 5 implementation files + 2 policy documents; excluded popup renderers are explicitly deferred.
- **Architecture:** no new service, screen, layout, runtime dependency, I/O, or concurrency path.
- **Code quality:** muted/help/disabled use terminal default foreground; verification commands match the reduced scope.
- **Tests:** 5/5 planned path families covered; profile matrix, visible labels, width/action checks, and `NO_COLOR=1` smoke test are specified.
- **Performance:** no issues found; styles are package-level and rendering adds no I/O or high-complexity work.
- **Parallelization:** sequential implementation; shared `internal/app` module.
- **Mockups:** not generated because the gstack designer is unavailable and this is a terminal-native UI plan.
- **TODOS.md:** no deferred design TODO proposed; no `TODOS.md` exists in the repository.

NO UNRESOLVED DECISIONS
