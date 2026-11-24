# 개발 가이드

[문서 목록](README.md) · [제품 명세](../SYSTEM.md) · [기여 정책](../CONTRIBUTING.md)

이 문서는 소스 빌드, 모듈 책임, 변경 기준과 검증 절차를 다룬다. 사용자 설정은 [설정 가이드](configuration.md), 공개 wire 계약은 [서버 API](server.md)가 소유한다.

## 책임과 의존성

제품은 하나의 Go 모듈과 `cmd/meca` 진입점을 사용한다. 도메인 판단과 네트워크·DB·터미널 표현을 분리하며, 상태와 자원을 소유한 모듈이 변경·취소·정리를 책임진다.

| 위치 | 책임 |
| --- | --- |
| `cmd/meca` | CLI 분기, 의존성 조립과 프로세스 수명 |
| `internal/agent` | 모델 요청, native 도구 계약, 문맥·압축, 실행 순서와 팀 협업 |
| `internal/commands` | 사용자 명령, 설정·대화·서버 작업의 사용 사례 |
| `internal/provider` | 모델 탐색·연결, Provider별 요청·스트림 변환 |
| `internal/tools` | 도구 카탈로그, 후보 준비, 실행 어댑터와 결과 수집 |
| `internal/tools/browser` · `browser/runtime` | 설치 확인·준비, 브라우저 세션·관찰 페이지, 관리형 Chrome 갱신 |
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

제품 Go 버전·의존성은 [go.mod](../go.mod), 개발 도구는 [devtools/go.mod](../devtools/go.mod), WebUI는 [webui/package.json](../webui/package.json)과 `bun.lock`이 소유한다. 제품과 개발 도구의 의존성을 분리하고 직접 의존성 버전을 고정한다. 브라우저 도우미의 고정 버전·SHA-256과 플랫폼 경로는 `internal/tools/browser/runtime/assets.go`, Chrome Stable 조회·관리 설치본 선택은 같은 패키지의 `stable.go`·`managed.go`가 소유한다.

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

기능에 적용한 기법과 담당 코드는 [기능별 최적화](optimization-strategies.md)를 따른다. 변경 시에는 자원을 확보하는 시점, 취소·실패 시 반환 경로, 재사용 중 원본을 다시 확인하는 조건을 담당 모듈에서 확인한다.

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
bun run --cwd webui astryx doctor --built .
bun run --cwd webui bundle:budget
```

`verify`는 TypeScript·Biome·Oxlint·Knip·Steiger와 단위 테스트를 수행한다. 의존성 변경은 lockfile·라이선스·패치를 함께 확인한다.

`bundle:budget`은 진입 HTML에서 직접 로드·preload하는 자산의 gzip 합계를 **640KiB**, 전체 배포 자산의 gzip 합계를 **1536KiB**까지 허용한다. 실행 중 지연 로드한 화면의 전송량은 초기 합계와 구분한다. 예산 검사는 [스크립트](../webui/scripts/bundle-budget.ts)가 소유하며 `bun run --cwd webui analyze`로 초기·지연 청크의 구성을 확인한다.

TypeScript 7 환경에서 Astryx Doctor의 컴포넌트 props 검사는 개발 의존성 `@typescript/typescript6`의 컴파일러 API를 사용한다. 애플리케이션 타입 검사는 `typescript`에 고정한 버전으로 실행한다.

### Astryx 검사와 알려진 경고

Core·Neutral·CLI는 같은 정식 버전으로 고정한다. 의존성을 바꿀 때 이전 버전부터 `astryx upgrade --from <version> --path .`를 실행하고 `--apply` 결과·수동 조치·생성물을 검토한다. 현재 `0.7.17`의 유지보수 범위는 [패치 계약](../webui/patches/README.md)을 따른다.

`doctor --built dist`는 배포 자산의 Astryx 사용과 CSS 포함을 검사한다. 소스의 스타일·아이콘 사용까지 보려면 프로젝트 루트인 `--built .`도 검사한다. 0.7.17 기준으로 남아 있는 경고는 다음과 같이 구분한다.

- **스타일 154건·아이콘 43건:** CSS import·`className`·SVG·아이콘 라이브러리 사용에 대한 정적 검사다. 생성 테마 연결과 브랜드·아이콘 등록 코드도 검토 대상에 포함된다. 전부 화면 오류이거나 정당한 예외라는 뜻은 아니며, 각 항목을 공개 컴포넌트·토큰으로 대체할 수 있는지 별도로 분류해야 한다.
- **테마 선택자 2건:** `collapsible`의 `data-tone="tool"`·`"request"`와 `"thinking"`에 지정한 배경·여백이 동시에 적용된다고 예측한다. 같은 요소의 정확한 속성값은 상호 배타적이므로 현재 규칙에는 해당 교집합이 없다. 경고만 없애기 위해 불가능한 조합의 스타일을 추가하지 않는다. 선택자를 바꿀 때는 이 판단과 실제 브라우저 표시를 다시 확인한다.
- **Vite의 500kB 청크 안내:** 압축 전 단일 JavaScript 청크 크기에 대한 안내다. 초기 gzip 640KiB 검사와 따로 확인하며, 크기가 큰 의존성은 분석 결과와 실제 초기 화면의 필요성을 기준으로 분리한다.

경고를 일괄 억제하지 않는다. 빌드·타입·동작 실패와 디자인 사용 권고를 구분해 보고하고, 검사 버전이 바뀌면 이 기록도 재검토한다.

### 브라우저 성능·복구 책임

| 소유 모듈 | 범위 |
| --- | --- |
| `shared/api` | native HTTP, 응답 검사, 읽기 마감, 인증 수명과 큰 요청 인코딩 |
| `pages/workspace/api/observer` | 작업·파일 구독을 합친 WebSocket, 재접속과 페이지 복원 |
| `pages/workspace/model` | 접수 확인·후속 예약·입력 예산·파일 편집·저장 이력 인계 |
| `shared/lib/reload` · `shared/ui/recovery` | 미저장 변경 보호와 독립된 로딩 실패 복구 |
| `shared/ui/markdown` · `code-preview` | 페이지별 표현·증분 계산·줄바꿈 |
| `shared/lib/image-preview` | 원본과 분리된 파생 이미지·디코딩 예산·URL 수명 |

관찰 재연결·로딩 실패·인증 만료·페이지 복원에서는 작업 접수와 미저장 초안을 확인한다. 숨긴 화면의 승인·완료·예약 전달과 가상화한 행의 포커스·드래그 수명도 검증한다. [WebUI 구현 전략](optimization-strategies.md#webui-상태와-렌더링)

브라우저 검사는 사용자 상태를 사용하지 않는 opt-in fixture를 사용한다. 아래 서버 명령은 별도 터미널에서 실행한다.

```sh
mkdir -p .local/ui-check
MECA_UI_DESCRIPTOR="$PWD/.local/ui-check/connection.json" \
  go test -timeout=50m ./cmd/meca -run '^TestUIBrowserFixture$' -count=1 -v > .local/ui-check/server.log 2>&1
```

연결 파일이 준비된 뒤 다른 터미널에서 실행하고, 완료 후 fixture를 종료한다.

```sh
MECA_UI_DESCRIPTOR="$PWD/.local/ui-check/connection.json" bun run --cwd webui test:e2e --project=chromium
touch .local/ui-check/connection.json.stop
```

브라우저별로 새 fixture를 시작하고 `--project=firefox` 또는 `--project=webkit`으로 반복한다. Provider의 대기 신호와 승인 테스트 파일은 fixture 수명을 공유하므로 브라우저 간에 실행 상태를 재사용하지 않는다. 다음 실행에는 새 연결 파일 경로를 사용한다.

Playwright CLI는 실제 화면·입력·네트워크 확인에 사용한다. Chromium·Firefox·WebKit, 좁은 화면·키보드·forced colors·축소 모션을 확인하며 실제 Safari IME·스크린리더 검증과 구분한다. HTTP/1·HTTPS/HTTP2와 bfcache 복원은 별도 계약으로 확인한다. 테스트 연결 파일·쿠키·trace·캡처·로그는 로컬 산출물이다.

## 문서 작성과 갱신

README는 소개와 빠른 시작을, `SYSTEM.md`는 제품 요구와 모듈 경계를 담당한다. 에이전트의 판단·복구·기록은 런타임 문서에, 등록된 도구와 실행 어댑터는 도구 문서에, 승인 규칙은 실행 경계에 둔다. [에이전트 런타임](runtime.md) · [도구 목록과 연동](tools.md) · [호스트 실행과 승인](execution-boundary.md)

사용자 스킬의 탐색·작성·호출은 스킬 문서에, 배포 카탈로그와 기본 자료는 기본 스킬 문서에 둔다. 공통 서버 계약은 서버 API에, 화면 구현과 API의 연결은 WebUI 연동에 둔다. 최적화 문서는 적용한 기법과 구현 소유자를 가리킨다. 관련 문서에서 설명이 필요한 경우 담당 절을 연결한다. [문서 목록](README.md)

문서를 바꿀 때 소스·기본값·링크를 함께 확인한다. 릴리스 노트에는 직전 배포 이후 해당 버전의 변경을 기록한다.

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
