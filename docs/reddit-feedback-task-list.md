# Reddit 피드백 작업 목록

상태: ACTIVE  
출처: graphkeeper Reddit 공개 글의 사용자 피드백  
작성일: 2026-07-26

## 현재 상태 감사 (2026-07-28)

피드백을 받은 뒤 현재 브랜치에서 확인된 반영 사항:

- [x] README에 AI-assisted development 공개 섹션이 있다.
- [x] README에 로컬 로그와 Git remote 통신을 구분해 설명한다.
- [x] 저장소에서 실행 바이너리를 추적하지 않으며 `.gitignore`에 빌드 산출물이 포함된다.
- [x] 그래프 검색, stash/tag/remote 상태, branch 조작 등 단순 그래프 표시를 넘어서는 maintainer workflow가 존재한다.
- [x] 긴 문자열을 다루는 일부 렌더링 폭 테스트가 존재한다.
- [x] `scripts/check`가 통과한다.

아직 “완료”로 볼 수 없는 항목:

- `cmd/graphkeeper`는 인자를 처리하지 않으므로 `--help`와 `--version`이 없다.
- GitHub Release 자동화와 Linux 실행 검증은 없다. Linux cross-build 자체는 가능하지만, 현재 환경은 macOS이므로 Linux 바이너리의 실제 실행은 CI 또는 Linux 환경에서 검증해야 한다.
- 커밋 전체 메시지와 diff를 Graph에서 inspect하는 흐름이 없다.
- 색상 토큰이 3-bit, 256색, truecolor를 혼용하고 밝은 배경 기준의 검증이 없다.
- 로그 파일이 `0644`로 생성되고, 로그 비활성화 옵션이나 전용 테스트가 없다.
- AGENTS에 언급된 `scripts/eval`, `scripts/doctor`, `scripts/hooks` 중 현재 저장소에는 `eval`, `doctor`가 없다. 이 저장소에서 정말 필요한 명령인지 결정한 뒤 추가하거나 지침을 정정해야 한다.

## 목적

현재 graphkeeper가 단순한 커밋 그래프 뷰어로 보이는 문제를 줄이고, 사용자가
브랜치 상태를 판단하는 데 필요한 정보와 신뢰를 제공한다.

이 문서는 구현 순서를 정하기 위한 개인 작업 문서다. 각 항목은 실제 구현 시
별도의 구현 계획 또는 GitHub Issue로 분리할 수 있다.

## 한눈에 보는 작업 순서

1. 배포·CLI 기본기와 신뢰성 문제를 먼저 해결한다.
2. Graph의 정보 밀도를 높여 기존 도구와의 차이를 만든다.
3. 색상 접근성과 Linux 실행을 검증한다.
4. README 데모와 공개 고지를 새 기능에 맞춰 개정한다.

---

## 1. Graph 정보 밀도와 제품 가치

Reddit 피드백:

- tig와 lazygit도 전체 그래프를 보여주므로 graphkeeper의 장점이 불분명하다.
- 전체 커밋 메시지와 diff를 빠르게 확인할 수 있어야 한다.
- 전체 브랜치 이름과 subject를 20자 정도로 강제 truncation하면 정보가 사라진다.

### 작업 목록

- [ ] graph
  - [ ] 전체 브랜치 이름을 확인할 수 있는 detail 또는 popup 경로 제공
  - [ ] Graph row의 짧은 라벨과 detail panel의 전체 ref 목록 역할을 명확히 분리
  - [ ] 긴 브랜치 이름을 무조건 20자 안팎에서 자르지 않도록 표시 폭 정책 재검토
  - [ ] 좁은 터미널에서는 전체 이름을 보조 패널 또는 overlay에서 확인할 수 있도록 함
  - [ ] 커밋 subject 표시 길이를 늘리고, 공간이 부족할 때의 우선순위 정의
  - [ ] 선택한 커밋의 전체 commit message 표시
  - [ ] 선택한 커밋의 diff를 읽기 전용 preview로 빠르게 열기
  - [ ] diff가 길거나 바이너리 파일을 포함할 때의 축약·스크롤·오류 상태 정의
  - [ ] 전체 graph를 보여주는 것 외에 graphkeeper가 해결하는 maintainer workflow를 문서화
  - [ ] tig/lazygit과의 기능 중복 및 차별점을 README에서 구체적인 사용 시나리오로 설명

### 완료 기준

- 선택한 커밋에서 subject, 전체 message, 변경 파일 또는 diff를 별도 명령 없이 확인할 수 있다.
- 긴 local/remote branch ref가 숨겨지지 않고 최소 한 번의 명시적인 inspect 동작으로 전체 표시된다.
- README가 “그래프를 보여준다”가 아니라 “브랜치 유지보수 판단을 어떻게 줄이는가”를 설명한다.
- 좁은 터미널과 긴 텍스트를 포함한 렌더링 테스트가 있다.

### 선행 작업

- Graph detail panel의 현재 데이터 모델과 Git command 경계를 먼저 확인한다.
- 기존 compact branch label 결정(`docs/decisions.md`)과 충돌하지 않도록 row/detail의 책임을 확정한다.

## 2. AI 사용과 개인정보 신뢰

Reddit 피드백:

- AI가 어느 정도 사용되었는지 공개해야 한다.
- 사용자가 실행하는 코드의 안전성을 판단할 수 있도록 명확한 고지가 필요하다.
- 로그를 “telemetry”라고 부르는 것이 적절하지 않을 수 있다.
- 프로그램이 외부로 데이터를 보내는지 README에 밝혀야 한다.

### 작업 목록

- [ ] 프로젝트의 AI 사용 범위 작성
  - [ ] 기획, 코드 작성, 테스트, 취약점 탐색, 문서 작성에서 AI가 사용된 범위를 구분
  - [ ] 사람이 검토·결정한 부분과 AI가 생성한 부분을 과장 없이 설명
  - [ ] `README.md`에 “AI-assisted development” 또는 동등한 공개 섹션 추가
- [ ] 로그 동작 감사
  - [ ] `internal/telemetry/telemetry.go`의 모든 호출 지점 목록화
  - [ ] 로그에 기록되는 필드와 민감정보 가능성 검토
  - [x] 현재 로그가 네트워크로 전송되지 않고 로컬 임시 디렉터리에만 쓰이는지 검증
  - [x] 외부 전송이 없다면 `telemetry`라는 이름과 README 설명을 실제 동작에 맞게 수정
  - [x] README에 외부 로그 전송이 없다는 점과 Git 원격 작업의 네트워크 통신을 구분해 명시
  - [x] 기록되는 이벤트 종류와 필드, 로컬 로그 삭제 방법을 README에 명시
  - [ ] 로그 파일 권한과 오류 처리 정책 검토

검증 결과: 로그는 `os.TempDir()` 아래의 `graphkeeper-events.jsonl`에 이벤트가
발생할 때만 생성된다. Linux에서는 보통 `/tmp/graphkeeper-events.jsonl`,
macOS에서는 보통 `$TMPDIR/graphkeeper-events.jsonl`이다. 로그 자체를 외부로
전송하는 코드는 확인되지 않았지만, 사용자가 `fetch`, `pull`, `push`를 실행하면
Git 명령이 설정된 원격 저장소와 통신할 수 있다.
- [ ] 보안·신뢰 문서화
  - [ ] 실행 파일을 받는 경로와 소스 검증 방법 안내
  - [ ] 알려진 제한사항과 alpha 상태를 유지하되, 안전성에 대한 모호한 표현 제거

### 완료 기준

- 사용자가 README만 읽고 AI 사용 범위, 로그의 로컬/원격 여부, 수집 필드, 끄는 방법을 알 수 있다.
- 구현 동작과 README의 “Local Diagnostic Logs” 설명이 서로 다르지 않다.
- 로그 관련 테스트 또는 검증 명령으로 네트워크 전송이 없음을 확인하거나, 전송이 있다면 그 계약을 검증한다.

## 3. README와 Demo

Reddit 피드백:

- README의 이미지/애니메이션이 너무 작아 실제 동작을 확인하기 어렵다.
- 기존 README는 graphkeeper의 차별점을 충분히 설명하지 못한다.

### 작업 목록

- [ ] graph
  - [ ] README demo 이미지를 실제 터미널 화면을 읽을 수 있는 크기로 교체
  - [ ] 정적 스크린샷만으로 부족하면 전체 화면 GIF 또는 짧은 동영상 추가
  - [ ] 데모에 branch 상태, commit detail, diff/inspect 흐름이 보이도록 구성
  - [ ] README 상단에 10초 안에 이해되는 핵심 workflow 예시 추가
  - [ ] tig/lazygit과 경쟁하거나 대체한다는 식의 과장 대신 사용 목적을 명시
  - [ ] 설치·실행·플랫폼 지원·로그 정책·AI 사용 고지를 상단에서 찾기 쉽게 배치
- [ ] 이미지의 접근성 설명과 대체 텍스트 작성
- [ ] README의 alpha 버전 설명을 실제 배포 방식과 일치시킴

### 완료 기준

- GitHub README에서 별도 다운로드 없이 주요 화면과 핵심 workflow를 식별할 수 있다.
- 데모 이미지의 텍스트가 일반적인 README 화면 크기에서 읽힌다.
- 설치 명령이 공식 배포 경로와 일치한다.

## 4. 바이너리 관리와 배포

Reddit 피드백:

- 바이너리를 Git에 커밋하지 말고 GitHub Releases를 사용해야 한다.
- 현재 바이너리가 Linux에서 실행되지 않는다.

### 작업 목록

- [ ] graph
  - [x] 저장소에 추적 중인 `graphkeeper` 바이너리 제거
  - [x] `.gitignore`에 빌드 산출물 패턴 추가
  - [ ] README의 로컬 빌드 예시와 릴리스 설치 예시를 분리
  - [ ] GitHub Releases용 아카이브 이름·구조·checksum 정책 결정
  - [ ] macOS/Linux 대상 빌드와 아키텍처 범위 결정
  - [ ] GitHub Actions에서 재현 가능한 release artifact 생성
  - [ ] 태그 생성 시 release를 만드는 배포 workflow 작성
  - [ ] 릴리스에 소스 커밋, 버전, OS, 아키텍처 정보를 표시
  - [ ] Linux에서 실제 실행 가능한 binary를 clean environment에서 검증
  - [ ] 현재 `go 1.25.6` 요구사항과 지원 Go 버전을 명시

### 완료 기준

- Git history의 기본 개발 흐름에 바이너리가 포함되지 않는다.
- GitHub Release에서 macOS와 Linux용 artifact를 다운로드할 수 있다.
- 지원하는 Linux 환경에서 `--help`와 일반 실행이 모두 동작한다.
- release artifact에 checksum과 버전 정보가 있다.

### 주의사항

이 항목은 기존 커밋에서 바이너리를 제거하는 정리와 향후 Release 배포를 분리한다.
필요하면 Git history 정리는 별도의 명시적인 결정으로 다룬다.

## 5. Go CLI 기본기

Reddit 피드백:

- Go의 CLI tooling을 사용해야 한다.
- 최소한 `--help` 옵션이 있어야 한다.

### 작업 목록

- [ ] graph
  - [ ] `flag` 또는 선택한 Go CLI parser로 `--help` 구현
  - [ ] `--version` 구현
  - [ ] 도움말에 사용법, 지원 옵션, 환경 요구사항, 종료 코드 설명
  - [ ] 잘못된 옵션은 도움말과 함께 비성공 종료 코드 반환
  - [ ] 저장소 밖에서 실행했을 때의 오류 메시지와 종료 코드를 정의
  - [ ] `go run ./cmd/graphkeeper -- --help`와 빌드 binary의 동작을 일치시킴
  - [ ] CLI parser 선택 이유와 사용법을 README에 반영

### 완료 기준

- `graphkeeper --help`가 Git 저장소가 없어도 동작한다.
- `graphkeeper --version`이 릴리스 버전과 일치한다.
- help/version/잘못된 옵션에 대한 테스트가 있다.

## 6. 터미널 색상과 접근성

Reddit 피드백:

- 터미널의 색상 스킴을 존중하거나 모든 색상을 RGB로 사용해야 한다.
- 흰색·베이지색 배경에서는 현재의 밝은 yellow/pink 색상이 읽기 어려울 수 있다.

### 작업 목록

- [ ] 현재 hardcoded ANSI 3-bit/256-color/truecolor 사용을 전수 조사
- [ ] 지원 목표 결정
  - [ ] 터미널 색상 프로파일을 따르는 ANSI 팔레트
  - [ ] truecolor 전용 테마
  - [ ] 기본 ANSI 테마 + truecolor 선택 테마
- [ ] 밝은 배경과 어두운 배경에서 읽을 수 있는 색상 토큰 재설계
- [ ] 색상만으로 상태를 구분하지 않도록 marker, label, bold, 위치 정보를 함께 사용
- [ ] `DESIGN.md` 및 `docs/highlighting-color-map.md`와 실제 색상 구현을 동기화
- [ ] `NO_COLOR` 등 비색상 환경 지원 여부 결정
- [ ] 16색, 256색, truecolor, 밝은 배경, 어두운 배경에서 스냅샷 또는 수동 검증

### 완료 기준

- branch, remote, tag, stash, conflict, warning 상태가 밝은 배경에서도 식별된다.
- 색상을 끄거나 제한한 터미널에서도 정보 손실이 기능 사용을 막지 않는다.
- 색상 정책이 문서와 코드에서 하나의 source of truth를 가진다.

## 7. 검증 계획

- [ ] `scripts/check` 통과
- [ ] `scripts/test` 통과
- [ ] `go build ./cmd/graphkeeper` 통과
- [ ] `graphkeeper --help`를 Git 저장소 밖에서 실행
- [ ] macOS와 Linux release artifact 실행
- [ ] 긴 branch name, 긴 subject, multiline commit message, 큰 diff 검증
- [ ] 빈 저장소·비 Git 디렉터리·네트워크 불가 상태 검증
- [ ] 로그 생성 위치와 로그 내용 확인
- [ ] 밝은 배경·어두운 배경·색상 제한 터미널에서 화면 확인
- [ ] README의 demo, 설치, 개인정보 고지를 새 릴리스 기준으로 확인

## 추천 구현 순서

### P0 — 설치·신뢰의 최소선

1. CLI `--help`/`--version`과 저장소 밖 오류 처리
2. 로그 권한·비활성화·오류 처리 정책 확정 및 테스트
3. GitHub Releases workflow와 checksum 정책 확정
4. Linux/macOS artifact를 clean environment에서 실행 검증

### P1 — 피드백의 핵심 제품 가치

5. Graph detail에서 전체 message와 diff inspect 제공
6. 전체 branch name과 subject 표시 정책 개선
7. README demo와 maintainer workflow 중심의 차별점 설명 개정

### P2 — 접근성과 운영 품질

8. 색상 토큰과 밝은 배경 접근성 개선
9. 긴 branch/subject/message/diff, 빈 저장소, 네트워크 불가 상태의 회귀 검증
10. 문서·릴리스·후속 피드백을 함께 점검하고 Reddit에 변경 사항 공개

## 보류할 결정

- tig/lazygit의 모든 기능을 따라갈지 여부: 현재는 full Git cockpit이 아니라 maintainer graph workflow를 강화하는 방향을 우선한다.
- diff viewer의 범위: 내장 full-screen viewer인지, `$GIT_PAGER`/외부 pager 연동인지 별도 결정한다.
- user-facing 문서에서는 `telemetry` 대신 `Local Diagnostic Logs`라는 명칭을 사용한다. 내부 패키지명 `internal/telemetry`는 별도 리팩터링 작업으로 남긴다.
- Git history에서 기존 binary까지 제거할지 여부: 현재 커밋에서 제거하는 것과 별도 판단한다.

## 다음 작업

가장 먼저 `--help`/`--version`, 로그 파일 권한·비활성화 정책, binary 배포를 작은 단위로 확정한다.
이 세 가지는 사용자가 프로그램을 설치하고 안전성을 판단하는 첫 경험을 결정하므로,
Graph UI 확장보다 먼저 처리한다.
