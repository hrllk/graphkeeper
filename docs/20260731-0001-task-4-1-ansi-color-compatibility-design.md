# 설계: Task 4.1 ANSI 우선 터미널 색상 호환성

office-hours 작성일: 2026-07-31  
브랜치: develop  
상태: DRAFT  
Taskmaster: 4.1  

## 문제 정의

외부 피드백에서 터미널 색상 호환성 문제가 제기되었다.

> 터미널의 색상 스킴을 완전히 존중해서 3-bit ANSI 색상을 사용하거나, 모든 색상을 RGB로 사용해야 한다. 흰색이나 베이지색 같은 비검정 배경을 사용하는 사람에게는 현재의 밝은 노란색과 분홍색이 읽기 어렵다.

현재 UI는 ANSI 256색 값(`205`, `226`, `81`), RGB 값(`#00b7eb`, `#ff5ca8`, `#9D00FF`, `#A14743`), `reverse`와 `bold` 같은 터미널 속성을 함께 사용한다. 이 혼용 때문에 같은 의미의 상태가 터미널 색상 프로파일에 따라 다르게 보일 수 있다.

## 수요 근거

- 피드백은 흰색·베이지색 배경에서 밝은 노란색과 분홍색이 읽히지 않는 구체적인 실패 사례를 제시했다.
- 피드백은 두 가지 일관된 정책을 제시한다. 터미널 팔레트를 따르거나, 앱이 전체 RGB 팔레트를 소유해야 한다.
- 현재의 ANSI 256색·RGB 혼용은 두 정책 어느 쪽에도 해당하지 않는다.
- 이 문제는 저장소의 `docs/reddit-feedback-task-list.md`와 Taskmaster task 4.1에 이미 기록되어 있다.

## 현재 상태와 기존 해결 방식

- 스타일은 대부분 `internal/app/view_shell.go`에 선언되어 있지만, 팝업·검색·그래프 렌더러에서도 색상을 직접 생성한다.
- `docs/highlighting-color-map.md`가 하드코딩된 색상 목록을 관리하지만, 실제 구현의 단일 기준은 아니다.
- 기존 색상 테스트는 Lip Gloss를 truecolor 모드로 고정한다. ANSI, ANSI 256색, truecolor를 함께 검증하는 프로파일 매트릭스는 없다.
- 기준 테스트는 `./scripts/test`로 통과한다.
- 현재 작업 트리에는 이 작업과 무관한 사용자 변경이 있다. Task 4.1은 해당 변경을 되돌리거나 함께 커밋하지 않아야 한다.

## 대상 사용자와 가장 작은 유효 범위

대상 사용자는 흰색·베이지색을 포함한 사용자 정의 터미널 팔레트를 사용하는 터미널 사용자다.

가장 작은 유효 범위는 다음과 같다.

1. 모든 의미 있는 UI 상태를 중앙화된 ANSI 우선 팔레트로 표현한다.
2. ANSI, ANSI 256색, truecolor 출력 프로파일에서 같은 상태의 의미가 유지되는지 테스트한다.
3. 밝은 배경에서도 중요한 상태가 색상 외의 표시와 함께 식별되는지 확인한다.

## 확정한 제약

1. 기본 렌더링은 사용자의 ANSI 팔레트를 존중한다.
2. 의미 기반 스타일에는 하드코딩된 RGB 값 대신 ANSI 호환 색상을 사용한다.
3. 중요한 상태를 색상 하나에만 맡기지 않는다. 기존 marker, label, bold, reverse, 위치 정보도 상태 표현의 일부로 유지한다.
4. Task 4.1에서는 터미널 배경 감지나 새로운 사용자 설정을 추가하지 않는다.
5. 레이아웃, 네비게이션, 동작 변경은 포함하지 않는다.
6. 기존 tag provenance와 stash/tag 겹침 상태의 의미는 유지한다.

## 전제

1. 주요 결함은 두 번째 light theme가 없는 것이 아니라 색상 정책이 혼용된 것이다.
2. 신뢰할 수 있는 배경 정보 없이 고정 RGB 팔레트를 선택하는 것보다, ANSI 팔레트를 존중하는 방식이 임의의 사용자 배경에 더 안전하다.
3. 현재 렌더러는 공통 semantic style 뒤로 점진적으로 이전할 수 있다.
4. ANSI 출력 테스트로 escape sequence와 프로파일 변환은 검증할 수 있지만, 최종 가독성은 어두운 터미널과 밝은 터미널에서 수동 확인해야 한다.

## 검토한 접근

### 접근 A: 의미 기반 ANSI 팔레트 중앙화

`current`, `warning`, `remote`, `tag`, `muted`, `selected`, `conflict` 같은 의미 기반 색상을 중앙화한다. 모든 렌더러의 직접적인 색상 생성을 공통 스타일 사용으로 바꾸고, 프로파일별 테스트와 색상 맵 문서를 함께 갱신한다.

예상 규모: 중간. 위험: 낮음~중간. 기존 전역 스타일 정의와 Lip Gloss 프로파일 변환을 재사용한다.

### 접근 B: Renderer 주입형 Theme 구조

Theme 값 또는 renderer 객체를 도입하고 model과 view 함수에 전달한다. 향후 RGB 테마를 넣기 좋은 구조지만, 현재 색상 문제를 해결하기 위해 많은 함수 경계를 변경하게 된다.

예상 규모: 큼. 위험: 중간~높음. Lip Gloss는 재사용하지만, 피드백 범위를 넘어선 렌더링 리팩터링이 된다.

### 접근 C: ANSI 팔레트와 비색상 상태 신호 강화

ANSI 팔레트를 적용하면서 dirty, conflict, origin, tag 상태를 marker, label, bold, 위치 정보로도 강화한다. 색상을 구분하기 어려운 환경에는 유리하지만, compact graph와 section UI의 밀도가 높아질 수 있다.

예상 규모: 중간~큼. 위험: 중간. 기존 marker와 label을 재사용하지만 화면 밀도에 대한 추가 제품 결정이 필요하다.

## 권장 접근

접근 A를 선택한다. 기존의 비색상 상태 신호는 안전장치로 유지하되, Task 4.1에 배경 감지, 테마 설정, Renderer 주입 구조는 추가하지 않는다.

구현 순서는 다음과 같다.

1. 의미 기반 ANSI 팔레트를 정의하고 각 토큰의 의미를 문서화한다.
2. shell, graph, search, popup, tag, review 렌더러의 직접적인 `lipgloss.Color(...)` 사용을 공통 semantic style로 이전한다.
3. 남은 RGB 색상은 제거하거나 명시적으로 분류한다. 문서화된 사유가 없는 RGB 값은 semantic status style에 남기지 않는다.
4. ANSI, ANSI 256색, truecolor 프로파일 테스트를 추가한다. truecolor escape sequence만 비교하지 말고, 프로파일별로 의미 구분이 유지되는지 검증한다.
5. 선택, 경고, 충돌, tag provenance, stash, 검색 상태를 어두운 터미널과 밝은 터미널에서 수동 확인한다.
6. `docs/highlighting-color-map.md`를 갱신하고 정책을 `docs/decisions.md`에 기록한다.

## 미결정 사항

- 일반적인 밝은·어두운 터미널 팔레트에서 각 semantic state를 구분하기 좋은 ANSI 기본 색상 조합은 무엇인가?
- ANSI 우선 정책에서 검색 선택 상태를 배경색으로 계속 표시할 것인가, 아니면 reverse/bold 중심으로 바꿀 것인가?
- tag와 stash 색상을 반드시 서로 다른 색으로 유지해야 하는가, 아니면 색상이 비슷하게 매핑되는 환경에서는 marker/label 차이로 충분한가?

## 성공 기준

- semantic status style에 ANSI 256색과 RGB 정책이 혼용되지 않는다.
- 모든 사용자 노출 상태 스타일에 하나의 문서화된 semantic owner가 있다.
- ANSI, ANSI 256색, truecolor 렌더링 프로파일을 테스트한다.
- current, warning, conflict, remote, local tag, origin tag, stash, selected 상태가 색상 외의 표시 또는 텍스트 label로도 구분된다.
- 어두운 터미널과 밝은 터미널에서 중요한 상태가 사라지지 않는다.
- `./scripts/test`와 관련된 가장 좁은 검사 명령이 통과한다.
- 기존의 무관한 작업 트리 변경을 건드리지 않는다.

## 배포 계획

배포 변경은 필요하지 않다. 기존 graphkeeper 바이너리와 release 흐름에 포함되는 TUI 렌더링 변경이다.

## 의존성

- 프로젝트가 이미 사용하는 Lip Gloss 색상 프로파일 동작
- `internal/app/model_test.go`와 관련 app 테스트
- 기존 색상 목록 문서 `docs/highlighting-color-map.md`
- Taskmaster상 선행 의존성 없음

## 다음 과제

구현 전에 어두운 터미널과 밝은 터미널에서 현재 동작을 실행하고, 선택·경고·충돌·local/origin tag·stash·검색 상태를 캡처한다. 이 결과를 task 리뷰에 첨부한다. 그러면 escape sequence 테스트만으로 판단하지 않고 실제 before/after 기준으로 색상 매핑을 검증할 수 있다.

## 이번 논의에서 확인한 점

- 일반적인 색상 팔레트 정리가 아니라 외부 사용자의 구체적인 실패 사례를 기준으로 범위를 좁혔다.
- 터미널 배경을 앱이 통제할 수 없다는 제약을 반영해 ANSI 우선 정책을 선택했다.
- Task 4.1을 전체 테마 시스템으로 확장하지 않고 색상 정책과 검증에 한정했다.

## 검토 상태

독립 reviewer Agent 도구가 현재 세션에 노출되지 않아 adversarial review는 실행하지 못했다. 문서는 저장소 상태, 기존 색상 맵, 관련 피드백, 기준 테스트 결과를 바탕으로 자체 점검했다.
