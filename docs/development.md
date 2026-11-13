# 개발 가이드

[문서 목록](README.md) · [제품 명세](../SYSTEM.md) · [기여 정책](../CONTRIBUTING.md)

이 문서는 소스 빌드, 모듈 책임, 변경 기준과 검증 절차를 다룬다. 사용자 설정은 [설정 가이드](configuration.md), 공개 wire 계약은 [서버 API](server.md)가 소유한다.

## 책임과 의존성

제품은 하나의 Go 모듈과 `cmd/meca` 진입점을 사용한다. 도메인 판단과 네트워크·DB·터미널 표현을 분리하며, 상태와 자원을 소유한 모듈이 변경·취소·정리를 책임진다.

| 위치 | 책임 |
| --- | --- |
| `cmd/meca` | CLI 분기, 의존성 조립과 프로세스 수명 |
| `internal/agent` | 모델 요청, 문맥·압축, 실행 순서와 팀 협업 |
| `internal/commands` | 사용자 명령, 설정·대화·서버 작업의 사용 사례 |
| `internal/provider` | 모델 탐색·연결, Provider별 요청·스트림 변환 |
| `internal/tools` | 도구 계약, 후보 준비, 실행과 결과 수집 |
| `internal/policy` | 경로·명령·승인 판단 |
| `internal/storage` | 대화·접수 기록·자격 증명의 영속 저장 |
| `internal/config` | 설정·초기화·프롬프트·기본 스킬 |
| `internal/skills` | 스킬 탐색·읽기·출처·참고 자료 접근 |
| `internal/diagnostics` | 공통 오류 분류·메시지·운영 로그 |
| `internal/serverapi` · `api/meca/v1` | 인증·전송·공개 API 계약과 데이터 투영 |
| `internal/tui` | 터미널 이벤트 루프·입력·화면 상태 |
| `internal/webui` · `webui/` | 정적 자산 전송과 브라우저 앱 |
| `devtools/` | 제품에 포함하지 않는 개발 도구와 고정 의존성 |

생성은 합성 루트에 두고 소비자는 실제 책임을 소유한 패키지를 직접 사용한다. 인터페이스는 실제 I/O·교체·테스트 경계에 필요한 작은 계약으로 제한한다. SQL·전송 형식·터미널 프레임워크 타입을 도메인 판단에 섞지 않는다.

## 변경 기준

- 변경 전 호출부·계약·관련 테스트를 확인하고 책임·배치·상태 소유권·실패 경로를 정한다.
- 직접 함수와 언어 기본 기능을 우선한다. 추상화·캐시·동시성·새 의존성은 실제 요구와 소비자가 있을 때 도입한다.
- 외부 입력은 경계에서 해석한다. 승인한 대상과 실제 효과를 일치시키고, 불확실한 부작용은 자동 재실행하지 않는다.
- `context.Context`를 취소 가능한 I/O에 전달한다. goroutine·구독·타이머·HTTP body·DB·자식 프로세스에는 종료와 회수 경로를 둔다.
- 오류 코드·메시지·로그 정책은 `internal/diagnostics`가 소유한다. 계층마다 임의 오류 코드나 중복 로그를 만들지 않는다.
- 테스트 가능한 동작 변경은 실패를 재현하는 검사부터 추가한다. 문구·문서·기계적 변경은 관련 diff와 참조를 확인한다.
- 파일은 실제 책임으로 나누며, 직접 작성한 코드·스크립트는 350줄 이내로 유지한다. 주석을 지원하는 파일의 시작에는 `Purpose`·`Pattern`·`Usage`·`Related` 헤더를 둔다. 생성 코드·외부 소스의 예외는 해당 소유 문서에 명시한다.
- 비밀값과 사용자 원문을 테스트 산출물·로그·스크린샷·커밋에 넣지 않는다. 지원하지 않는 저장 형식은 원본을 보존하고 명시적으로 거절한다.

## 의존성과 배포

제품 Go 버전·의존성은 [go.mod](../go.mod), 개발 도구는 [devtools/go.mod](../devtools/go.mod), 브라우저는 [webui/package.json](../webui/package.json)과 `bun.lock`이 소유한다. 제품과 개발 도구의 의존성을 분리하고 직접 의존성 버전을 고정한다.

변경 시 유지보수 상태·라이선스·실행 비용·지원 플랫폼을 검토하고 관련 고지를 갱신한다. `third_party/clipboard`의 수정 범위와 배포 조건은 [의존성 안내](../third_party/clipboard/MECA.md)를 따른다. WebUI 패치는 [패치 계약](../webui/patches/README.md), 라이선스 입력은 [고지 안내](../webui/licenses/README.md)가 소유한다.

### 소스 빌드

저장소 루트에서 실행한다. 제품·개발 도구는 Go 1.27.4 이상, full의 UI 빌드는 Bun 1.4.4 이상을 사용한다.

```sh
# tui
go build -tags=godbus_no_tcp,tui -trimpath -o bin/meca ./cmd/meca

# full
sh scripts/build-ui.sh
go build -tags=godbus_no_tcp -trimpath -o bin/meca ./cmd/meca
```

설치는 `./bin/meca install`을 사용한다. `godbus_no_tcp`의 범위는 [Linux 자격 증명 전송](configuration.md#linux-자격-증명-전송)을 따른다. 빌드 산출물·생성 자산·로컬 상태는 Git에 넣지 않는다.

### 배포 종류와 빌드 경계

`tui` 태그는 서버 조립부와 WebUI를 실행 파일에서 제외한다. 공통 Provider·도구·팀·스킬은 두 종류에 포함한다. 변경 시 tui의 의존성에 서버·WebUI가 포함되지 않는지도 확인한다.

```sh
GOFLAGS='-tags=tui,godbus_no_tcp' go test ./cmd/meca ./internal/commands/startup ./internal/commands/upgrade ./internal/commands/install
go list -tags=tui,godbus_no_tcp -deps ./cmd/meca
```

버전 규칙·배포 검증·게시 절차는 [릴리스 가이드](releases.md)를 따른다. 로컬 검증과 GitHub 게시는 별개다.

### 개발 도구

저장소 루트의 `go tool -modfile=devtools/go.mod`로 고정 도구를 사용한다.

| 도구 | 범위 |
| --- | --- |
| Staticcheck · `go vet` | 코드 오류·API 오용·불필요한 연산 |
| govulncheck | 의존성 취약점과 실제 호출 경로 |
| deadcode | 실행·테스트 진입점과 플랫폼별 도달 가능성 |
| diagnosticscheck | 중앙 오류·로그 계약의 우회 |
| benchstat | 성능 변경 검증 |
| GoReleaser · nFPM | 배포 산출물과 패키지 구성 |

자동 삭제나 일괄 억제 대신 각 결과를 호출부·플랫폼·테스트와 대조한다. 다른 OS를 분석할 때는 호스트에서 실행할 도구를 먼저 빌드하고 분석 대상에만 `GOOS`·`GOARCH`를 적용한다.

## 변경 검증

변경 경로의 집중 검사를 먼저 실행하고 공통 계약 변경은 범위를 넓힌다. 동시성·취소·수명 변경에는 race 검사를 포함한다.

```sh
export GOFLAGS='-tags=godbus_no_tcp'
gofmt -l cmd internal
go test ./...
go vet ./...
go tool -modfile=devtools/go.mod staticcheck ./...
go -C devtools run ./cmd/diagnosticscheck -C=.. ./...
go build ./...
go mod verify
```

의존성 변경은 `govulncheck ./...`도 확인한다. 작은 결정적 fixture·`t.TempDir`·`httptest`를 사용하고 실제 자격 증명·사용자 프로젝트·유료 모델에 의존하지 않는다. 실패·취소·저장 불확실·중복 요청을 성공 경로와 함께 검증한다.

서버 계약을 바꾸면 `api/meca/v1`의 원본과 Go·TypeScript 바인딩을 함께 갱신한다. 생성 명령은 [서버 계약](server.md#개발과-생성)을 따른다. 생성 코드를 직접 수정하지 않는다.

## 플랫폼 실행 검증

macOS·Linux와 WSL의 실제 동작, 교차 빌드와 모의 전송 검사는 구분한다. 플랫폼 경계 변경은 해당 OS에서 취소·오류·종료와 자원 반환을 확인한다.

- 터미널: 키 입력·IME·마우스·크기 변경·종료 후 터미널 복원.
- 클립보드: 이미지·파일 복사·빈 값·초과 크기·취소. 개인 클립보드 대신 격리된 fixture 사용.
- WSL: Windows 상호운용, WSLg 유무와 Windows PATH를 가져오지 않는 환경.
- 자격 증명: macOS Keychain·Linux Secret Service의 저장·조회·실패·복구.

실행할 수 없는 환경은 미검증으로 보고한다. 지원 조건은 [이미지 입력](images.md)과 [실행 경계](execution-boundary.md)를 따른다.

## 성능과 장시간 수명

| 소유 모듈 | 개발 시 고려사항 |
| --- | --- |
| `internal/processmemory` | 입장 예산·취소·예약 반환, 실제 보유량과 논리 예산의 구분 |
| `internal/agent/context` · `compact` | 문맥 선택·압축과 원본 이력 보존 |
| `internal/provider/protocol` | 실행별 인코딩 재사용·모델 변경 시 무효화·완료 시 회수 |
| `internal/storage/content` · `sqlite` | 원문 중복 방지·트랜잭션·선택 복원·읽기 수명 |
| `internal/serverapi/admission` · `detail` | 제한된 입력·원문 조회, 페이지와 revision의 일치 |
| `internal/tui` | 이벤트 순서·입력 우선 처리·표시 예산·화면 밖 자원 회수 |
| `webui/src/pages/workspace` | 요청 접수·관찰·초안·화면 표시의 독립된 수명 |
| `webui/src/shared/ui` · `shared/lib` | bounded Markdown·코드·이미지 표시와 취소 가능한 처리 |

### 성능 변경 시 고려사항

입력 수용·실행·저장·표시 예산을 동일시하지 않는다. 캐시는 생성·무효화·회수 주체와 상한이 있어야 하며 원문을 여러 계층에 복제하지 않는다. 자원 절감과 재조회 비용을 함께 검토하고 최초 실행·재사용·종료를 모두 확인한다.

할당량·동시 보유 메모리·RSS·I/O·응답 지연을 구분한다. 개별 함수의 결과를 모델 TPS나 전체 작업 완료 시간으로 해석하지 않는다. 구체적인 입력 한도는 [설정](configuration.md#본문-한도), 전송·저장 한도는 [서버 계약](server.md#이벤트재접속수명)을 따른다.

### PGO 빌드 책임

`scripts/performance/pgo.sh`는 격리된 엔진 부하의 프로파일을 만들고 임시 자료를 정리한다. 프로파일은 소스·도구 체인·플랫폼·부하에 종속된 빌드 입력이다. 대표성과 적용 효과를 확인한 경우에만 사용하며 생성물을 Git에 넣지 않는다.

## 내장 WebUI

React·Astryx·TypeScript·Bun·Vite로 정적 SPA를 빌드한다. UI는 FSD 계층과 공개 API를 사용하고 서버 캐시와 브라우저 임시 상태를 분리한다. `internal/webui`는 빌드된 자산을 제공하며 배포 실행에는 JavaScript 서버가 필요 없다.

UI 표현은 Astryx를 사용하고 코드 편집기는 명시한 패치 계약을 따른다. 디자인 규칙은 [webui/DESIGN.md](../webui/DESIGN.md), 테마 원본은 `webui/src/app/styles/theme.ts`가 소유한다. 생성된 테마·아이콘·번들은 직접 편집하지 않는다.

```sh
bun run --cwd webui verify
bun run --cwd webui build
bun run --cwd webui astryx doctor --built dist
bun run --cwd webui bundle:budget
```

`verify`는 TypeScript·Biome·Oxlint·Knip·Steiger와 단위 테스트를 수행한다. 의존성 변경은 lockfile·라이선스·패치를 함께 확인한다.

### 브라우저 성능·복구 책임

| 소유 모듈 | 범위 |
| --- | --- |
| `shared/api` | native HTTP, 응답 검사, 읽기 마감, 인증 수명과 큰 요청 인코딩 |
| `pages/workspace/api/observer` | 작업·파일 구독을 합친 WebSocket, 재접속과 페이지 복원 |
| `pages/workspace/model` | 접수 확인·후속 예약·입력 예산·파일 편집·저장 이력 인계 |
| `shared/lib/reload` · `shared/ui/recovery` | 미저장 변경 보호와 독립된 로딩 실패 복구 |
| `shared/ui/markdown` · `code-preview` | 페이지별 표현·증분 계산·줄바꿈 |
| `shared/lib/image-preview` | 원본과 분리된 파생 이미지·디코딩 예산·URL 수명 |

관찰 재연결로 명령을 다시 제출하지 않는다. 숨긴 본문은 렌더링을 동결하되 승인·완료·예약은 계속 처리한다. 가상화는 포커스·드래그 중인 행을 보존하고, 로딩 실패·인증 만료·페이지 복원은 초안 보호와 함께 검증한다.

브라우저 검사는 사용자 상태를 사용하지 않는 opt-in fixture를 사용한다. 아래 서버 명령은 별도 터미널에서 실행한다.

```sh
mkdir -p .local/ui-check
MECA_UI_DESCRIPTOR="$PWD/.local/ui-check/connection.json" \
  go test -timeout=50m ./cmd/meca -run '^TestUIBrowserFixture$' -count=1 -v > .local/ui-check/server.log 2>&1
```

연결 파일이 준비된 뒤 다른 터미널에서 실행하고, 완료 후 fixture를 종료한다.

```sh
MECA_UI_DESCRIPTOR="$PWD/.local/ui-check/connection.json" bun run --cwd webui test:e2e
touch .local/ui-check/connection.json.stop
```

Playwright CLI는 실제 화면·입력·네트워크 확인에 사용한다. Chromium·Firefox·WebKit, 좁은 화면·키보드·forced colors·축소 모션을 확인하며 실제 Safari IME·스크린리더 검증과 구분한다. HTTP/1·HTTPS/HTTP2와 bfcache 복원은 별도 계약으로 확인한다. 테스트 연결 파일·쿠키·trace·캡처·로그는 로컬 산출물이다.

## 문서 작성과 갱신

공개 문서는 사용·연동·빌드·유지보수에 필요한 현재 계약을 설명한다. README는 소개와 빠른 시작, [문서 목록](README.md)은 독자별 진입점, 각 가이드는 해당 기능을 소유한다. Provider 주소·인증·생성 API는 모델 연결 문서에 둔다.

작업 일지·개인 환경·조사 과정·측정 결과·테스트 통과 기록·미구현 계획은 공개 문서에 누적하지 않는다. 함수별 구현 해설이나 소스에서 확인할 수 있는 내부 저장·암호화·캐시 구성의 반복도 피한다. 변경은 소유 문단에 반영하고 중복 설명 대신 링크한다. 릴리즈 노트에는 직전 배포 이후 해당 버전의 변경만 기록하고 기존 기능 소개나 다른 날짜의 변경을 반복하지 않는다.

필수 한도·오류·보안 경계·백업 조건은 사용자가 판단할 수 있도록 설명한다. 구현을 숨기는 것을 접근 통제로 간주하지 않는다. 문서 변경은 실제 코드·기본값·로컬 링크와 대조한다.

## 저장소에 포함하는 파일

| 분류 | Git 정책 |
| --- | --- |
| `README.md`, `SYSTEM.md`, `CONTRIBUTING.md`, `SECURITY.md`, `docs/` | 현재 사용·계약·개발 문서 유지 |
| `LICENSE`, `NOTICE`, `THIRD_PARTY_NOTICES.md`, 외부 소스 고지 | 배포에 필요한 라이선스·출처 유지 |
| `cmd/`, `internal/`, `api/`, `webui/src/`, 테스트와 fixture | 제품·검증 소스 유지 |
| `internal/serverapi/gen/`, `webui/src/shared/api/gen/` | 재현 가능한 빌드 입력인 API 바인딩 유지 |
| `internal/config/bootstrap/defaults/` | 배포되는 스킬·예시·미디어·고지 유지 |
| `webui/DESIGN.md`, `webui/patches/`, `webui/licenses/` | UI 제작·패치·고지 입력 유지 |
| Go/Bun 의존성 파일, 빌드 스크립트·워크플로 | 재현성과 배포에 필요한 구성 유지 |
| `bin/`, `dist/`, `webui/dist/`, 생성 UI 번들·테마·아이콘 | 빌드 산출물로 제외 |
| `.cache/`, `.local/`, 테스트 보고서·브라우저 인증 상태 | 로컬 작업 자료로 제외 |
| `AGENTS.md`, 로컬 에이전트·IDE·디자인 도구 상태 | 개인 환경으로 제외 |
| 환경변수·토큰·키·실행 DB·로그 | 비공개 실행 상태로 제외 |

의도한 fixture·스킬 에셋·API 바인딩을 생성물이라는 이유만으로 제외하지 않는다. `.gitignore`는 추적 중인 파일이나 Git 이력의 내용을 제거하지 않는다. 추적 제외가 필요하면 대상을 검토한 뒤 별도로 처리하며, 공개된 자격 증명은 발급처에서 폐기·교체한다. [Git ignore 규칙](https://git-scm.com/docs/gitignore)을 따른다.
