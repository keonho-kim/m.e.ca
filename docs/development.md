# 개발 가이드

[문서 목록](README.md) · [제품 명세](../SYSTEM.md) · [기여 정책](../CONTRIBUTING.md)

소스 빌드는 제품 Go 모듈·개발 도구·WebUI의 고정 의존성을 사용한다. 사용자 설정은 [설정 가이드](configuration.md), 공개 wire 계약은 [서버 API](server.md)를 따른다.

## 책임과 의존성

제품은 하나의 Go 모듈과 `cmd/meca` 진입점을 사용한다. 도메인 판단과 네트워크·DB·터미널 표현을 분리하며, 상태와 자원을 소유한 모듈이 변경·취소·정리를 책임진다.

| 위치 | 책임 |
| --- | --- |
| `cmd/meca` | CLI 분기, 의존성 조립과 프로세스 수명 |
| `internal/agent` | 모델 요청, native 도구 계약, 문맥·압축, 실행 순서와 팀 협업 |
| `internal/commands` | 사용자 명령, 설정·대화·서버 작업의 사용 사례 |
| `internal/provider` | 모델 탐색·연결, Provider별 요청·스트림 변환 |
| `internal/tools` | 도구 카탈로그, 후보 준비, 실행 어댑터와 결과 수집 |
| `internal/tools/codemode` | 내장 TS 변환·JS 실행, Promise 호출 묶음과 JSON 경계 |
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

Code Mode는 `modernc.org/quickjs`와 esbuild의 Go API를 제품에 내장한다. 변환은 파일·모듈 해석 없이 수행하고, 실행기에는 기본 모듈 로더나 OS 도우미를 등록하지 않는다. 요청의 직렬 소유자가 완료 결과를 저장하고 JS를 재개하며, 작업자는 준비된 도구 실행만 담당한다. 수정 시 [코드 실행 계약](codemode.md)의 초기화 중 취소·UTF-8 반환 한도·개별 완료 후속 작업·활성 대상 충돌·승인 거부와 저장 실패 이후의 효과 차단을 검증한다.

SQLite의 payload codec 5는 Code Mode 결과에 포함된 기존 원문을 같은 DB의 codec 0·1·2 참조와 리터럴 조각으로 저장한다. 준비한 참조는 쓰기 트랜잭션에서 다시 확인하며 복원은 순환·중첩 참조와 다른 세션을 허용하지 않는다. 참조 탐색은 최대 8단계·128개 후보로 제한하고 한 복원에서 반복되는 원문은 마지막 사용까지 최대 2MiB를 재사용한다. 전체 JSON의 원래 크기와 해시를 확인한 뒤에만 반환한다.

### 소스 빌드

저장소 루트에서 실행한다. 제품·개발 도구는 Go 1.27.4 이상, full의 UI 빌드는 Bun 1.4.4 이상을 사용한다.

```sh
# tui
go build -tags=godbus_no_tcp,tui -trimpath -o bin/meca ./cmd/meca

# full
sh scripts/build-ui.sh
go build -tags=godbus_no_tcp -trimpath -o bin/meca ./cmd/meca
```

설치는 `./bin/meca install`을 사용한다. `godbus_no_tcp`는 Linux 데스크톱 폴더 선택기에 쓰는 D-Bus의 `tcp:`·`nonce-tcp:` 전송을 제외한다. API Key 저장은 로컬 파일을 사용하며 설치 명령이 키 디렉터리와 권한을 준비한다. 빌드 산출물·생성 자산·로컬 상태는 Git에 넣지 않는다.

### 배포 종류와 빌드 경계

`tui` 태그는 서버 조립부와 WebUI를 실행 파일에서 제외한다. 공통 Provider·도구·팀·스킬은 두 종류에 포함한다. 변경 시 tui의 의존성에 서버·WebUI가 포함되지 않는지도 확인한다.

```sh
GOFLAGS='-tags=tui,godbus_no_tcp' go test -short ./cmd/meca ./internal/commands/startup
GOFLAGS='-tags=tui,godbus_no_tcp' go test ./cmd/meca -run '^TestTUIExecutable$'
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
gofmt -l cmd internal scripts
go test ./...
go vet ./...
go tool -modfile=devtools/go.mod staticcheck ./...
go -C devtools run ./cmd/diagnosticscheck -C=.. ./...
go build ./...
go mod verify
```

의존성 변경은 `govulncheck ./...`도 확인한다. 작은 결정적 fixture·`t.TempDir`·`httptest`를 사용하고 실제 자격 증명·사용자 프로젝트·유료 모델에 의존하지 않는다. 실패·취소·저장 불확실·중복 요청을 성공 경로와 함께 검증한다.

`go test ./...`는 패키지 기능·API 계약과 실제 CLI의 파일 작업·내보내기·설치·업그레이드를 검증한다. 빠른 반복 검사용 `go test -short ./...`는 실행파일과 일부 설치 검사를 생략한다. 모델 도구 계약·CLI·설치·배포를 바꿀 때는 생략 없는 기본 명령과 `GOFLAGS=-tags=godbus_no_tcp,tui go test ./cmd/meca ./internal/commands/startup`을 실행한다. Provider 재시도 대기는 가상 시간으로 검증하고, 스트림의 실제 순서와 연결 수명은 로컬 HTTP 연결로 확인한다. Linux 마운트와 브라우저 시나리오는 아래의 별도 실행 절차를 따른다.

테스트는 현재 기능의 입력·출력·부작용·실패·보존 계약을 담당 패키지에서 검증한다. 삭제된 기능의 부재, 내부 계산식이나 고정된 항목 수만 확인하는 검사는 추가하지 않는다. 승인 거부 시 파일 보존, 제거 명령의 실제 정리, 취소 후 자원 해제는 기능 계약으로 검증한다.

모델을 포함한 실행 테스트는 `internal/agent/modeltest`의 Code Mode 호출 fixture를 공유하고, Provider 전송 형식 자체를 검사하는 테스트는 각 프로토콜의 직접 fixture를 사용한다. 공통 실행 의존성은 `*_fixture_test.go`, 측정 코드는 `*_bench_test.go`에 두며 설치·삭제 검증은 모델 요청 테스트와 독립적으로 실행한다. 프론트 단위 테스트는 대응하는 상태·API·이력 변환 모듈 옆에 두고, 실제 브라우저 흐름은 `webui/e2e`에서 검증한다.

서버 계약을 바꾸면 `api/meca/v1`의 원본과 Go·TypeScript 바인딩을 함께 갱신한다. 생성 명령은 [서버 계약](server.md#개발과-생성)을 따른다. 생성 코드를 직접 수정하지 않는다.

## 플랫폼 실행 검증

macOS·Linux와 WSL의 실제 동작, 교차 빌드와 모의 전송 검사는 구분한다. 플랫폼 경계 변경은 해당 OS에서 취소·오류·종료와 자원 반환을 확인한다.

- 터미널: 키 입력·IME·마우스·크기 변경·종료 후 터미널 복원.
- 클립보드: 이미지·파일 복사·빈 값·초과 크기·취소. 개인 클립보드 대신 격리된 fixture 사용.
- WSL: Windows 상호운용, WSLg 유무와 Windows PATH를 가져오지 않는 환경.
- 자격 증명: 로컬 키 DB의 0600 권한·소유권·심볼릭 링크 거부, 설치 시 실제 저장 검증, 오류 원인 전달, 재설치·동시 저장·키 교체·백업 복구·삭제.
- 설치·삭제: 일반 디렉터리·직접 마운트된 `~/.meca`·하위 마운트·읽기 전용·권한 복구 후 재시도. `sh scripts/install/test-linux.sh`는 Docker에서 마운트 검사와 비루트 검사에 필요한 권한을 각각 부여하고, 검사가 건너뛰어지면 실패한다.

full의 PTY·CLI 설치·업그레이드는 내장 UI 빌드 후 `GOFLAGS=-tags=godbus_no_tcp go test ./cmd/meca ./internal/commands/upgrade`로 확인한다. 소스 설치 스크립트만 검사할 때는 `go test ./scripts/install`을 사용한다.

실행할 수 없는 환경은 미검증으로 보고한다. 지원 조건은 [이미지 입력](images.md)과 [실행 경계](execution-boundary.md)를 따른다.

## 성능과 장시간 수명

기능에 적용한 기법과 담당 코드는 [기능별 최적화](optimization-strategies.md)를 따른다. 변경 시에는 자원을 확보하는 시점, 취소·실패 시 반환 경로, 재사용 중 원본을 다시 확인하는 조건을 담당 모듈에서 확인한다.

### 성능 변경 시 고려사항

입력 수용·실행·저장·표시 예산을 동일시하지 않는다. 캐시는 생성·무효화·회수 주체와 상한이 있어야 하며 원문을 여러 계층에 복제하지 않는다. 자원 절감과 재조회 비용을 함께 검토하고 최초 실행·재사용·종료를 모두 확인한다.

할당량·동시 보유 메모리·RSS·I/O·응답 지연을 구분한다. 개별 함수의 결과를 모델 TPS나 전체 작업 완료 시간으로 해석하지 않는다. 구체적인 입력 한도는 [설정](configuration.md#본문-한도), 전송·저장 한도는 [서버 계약](server.md#이벤트재접속수명)을 따른다.

### PGO 빌드 책임

`scripts/performance/pgo.sh`는 `engineworkload` 태그로 격리된 엔진 부하의 프로파일을 만들고 임시 자료를 정리한다. 여러 공급자의 부하 시나리오는 `go test -tags=godbus_no_tcp,engineworkload ./cmd/meca -run '^TestEngineWorkload$'`로 실행한다. 프로파일은 소스·도구 체인·플랫폼·부하에 종속된 빌드 입력이다. 대표성과 적용 효과를 확인한 경우에만 사용하며 생성물을 Git에 넣지 않는다.

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

### Astryx 검사

Core·Neutral·CLI는 같은 정식 버전으로 고정한다. 의존성을 바꿀 때 현재 설치 버전부터 `astryx upgrade --from <version> --path .`를 실행하고 `--apply` 결과·수동 조치·생성물을 검토한다. 적용 중인 수정과 갱신 절차는 [패치 계약](../webui/patches/README.md)을 따른다.

`doctor --built dist`는 배포 자산의 Astryx 사용과 CSS 포함을 검사한다. 소스의 스타일·아이콘 사용은 `doctor --built .`로 검사한다. 스타일·아이콘 경고는 공개 컴포넌트·토큰 사용과 브랜드·아이콘 등록의 실제 용도를 확인한다. 테마 경고는 선택자가 같은 요소에 함께 적용될 수 있는지와 실제 표시를 대조한다.

Vite의 청크 크기 안내는 압축 전 단일 JavaScript 청크를 대상으로 한다. 초기 gzip 예산과 함께 분석하고, 큰 의존성은 초기 화면에서의 필요성과 지연 로딩 경계를 기준으로 검토한다. 경고를 일괄 억제하지 않으며 빌드·타입·동작 실패를 해결한 뒤 전달한다.

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

`bun run --cwd webui test`는 응답 파싱과 상태 전이 같은 브라우저 독립 계약을 검사한다. Playwright는 내장 UI에 접속해 브라우저에서 확인할 수 있는 기능을 검사한다. 실행기는 Go fixture를 한 번 빌드하고, 테스트마다 새 임시 홈·모의 모델·서버를 시작한 뒤 종료한다.

```sh
sh scripts/build-ui.sh
bun run --cwd webui playwright install --no-shell chromium
bun run --cwd webui test:e2e:smoke
bun run --cwd webui test:e2e --project=chromium
```

릴리스는 `@smoke` 시나리오를 실행한다. 서버가 준비되지 않거나 필수 시나리오가 건너뛰어지면 실패한다. `bun run --cwd webui playwright install firefox webkit` 후 `test:e2e --project=firefox` 또는 `--project=webkit`으로 해당 브라우저 호환 검사를 실행한다. 브라우저가 실제로 보존한 페이지의 복원은 Chromium 시나리오에서 확인한다. HTTPS/HTTP2 실행은 `MECA_UI_TLS=1`과 `MECA_UI_CERTIFICATE`, `MECA_UI_KEY`의 테스트 인증서·키 절대 경로를 사용한다.

`bun run --cwd webui test:performance`는 실제 입력에서 INP·CLS·LCP를 측정하며 수집되지 않은 지표를 정상으로 간주하지 않는다. DOM·네트워크 읽기 상한은 기능 검사에서 확인한다. 실제 모델 연동은 `MECA_LMSTUDIO_URL`과 `MECA_LMSTUDIO_MODEL`을 설정한 뒤 `bun run --cwd webui test:live-provider`로 별도 실행한다.

Playwright CLI는 실제 화면·입력·네트워크 확인에 사용한다. Chromium·Firefox·WebKit, 좁은 화면·키보드·forced colors·축소 모션을 확인하며 실제 Safari IME·스크린리더 검증과 구분한다. HTTP/1·HTTPS/HTTP2와 bfcache 복원은 별도 계약으로 확인한다. 테스트 연결 파일·쿠키·trace·캡처·로그는 로컬 산출물이다.

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
