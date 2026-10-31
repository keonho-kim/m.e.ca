# 개발 가이드

[← README](../README.md) · [제품 명세](../SYSTEM.md)

## 책임과 의존성

제품은 단일 Go 모듈과 cmd/meca 합성 루트를 사용한다. 기능은 internal 아래에 두고 실제 소유 패키지를 직접 참조한다. 소비자 소유의 작은 I/O 계약과 함수·struct를 우선하며 패키지는 한 기능, 파일은 그 안의 한 책임을 담당한다.

| 위치 | 책임 |
| --- | --- |
| cmd/meca | CLI 분기·어댑터 조립·프로세스와 터미널 수명 |
| agent/model | 모델 메시지·스트림·출력 예산 계약 |
| agent/session·run | 요청 준비·실행 순서·승인 조율·공유 예산·실측 통계 집계 |
| agent/record | 실행 이벤트·체크포인트·상태·최종 통계·원자적 기록 단위와 검증 |
| agent/context·compact | 모델 문맥과 단일 호출 요약 |
| commands/headless | 같은 요청 경로의 text·JSONL 출력 어댑터 |
| commands/agentrequest | TUI·비대화형 실행의 설정·모델·세션을 공통 요청으로 조립 |
| agent/tool·work | 실행 사실·가벼운 파일 관측·종료 원인 |
| agent/prompt·inference | 작업 지침·역할별 추론 설정 |
| commands/startup·install·upgrade·settings | 시작 경로·사용자 설치와 삭제·검증한 최신 릴리스 업데이트·언어와 승인 모드 저장 |
| config/bootstrap·prompts·statefs·credentials | 초기화·Markdown 파일·private 상태 수명·OS 자격 증명 |
| provider/connection·identity·selection | 연결 값·서버 원본 표시·Provider 추론 옵션·선택 검증 |
| provider/discovery·diagnostic·setup | 메타데이터·실제 생성 진단·연결 저장과 소유권 이전 |
| provider/protocol | wire 변환·실행 범위의 이력 인코딩 재사용·재읽기 가능한 본문·스트림 해석 |
| provider/transport | HTTP 연결·수신 감지·재시도 |
| provider/httpclient | 생성 요청·유효 문맥·실행별 인코더의 수명 조율 |
| policy·policy/command | 공통 보호 대상·파일 Scope와 execute_command의 선택적 AST 검사·값 상태·명령별 승인 근거 |
| policy/command/transfer | HTTP·원격 복사·원격 실행의 전송 인자와 표시용 목적지 해석 |
| tools/local/catalog | 불변 도구 설명·스키마·native JSON 인자 파싱 |
| tools/local | 준비·승인 바인딩·실행·자원 수명을 연결하는 호스트 어댑터 |
| fileview | 도구 실행과 분리한 읽기 전용 탐색·참조 검증·미리보기 I/O, 공통 경로 정책을 적용한 루트 대여 |
| tools/local/filesystem | 실제 경로·프로젝트 identity·파일 도구의 private 경로 검사 |
| tools/local/file·textedit·textformat·diff | 파일 I/O·정확한 부분 수정·바이트 형식·변경 미리보기 |
| tools/local/process | 호스트 셸·명령 등록부·프로세스 그룹·출력·대기·취소 |
| storage/session·history·recovery | 세션 식별·순서 있는 이력·중단 복구의 저장 계약 |
| storage/content | 큰 JSON 값의 참조 분리·선택 복원·압축·내용 무결성 |
| storage/sqlite | DB 파일·SQL·트랜잭션·잠금·content 콜백 어댑터 |
| tui/session | 이벤트 루프·입력 우선 갱신·화면 전환·worker 수명과 하위 화면 컴포넌트 조율 |
| tui/composer·composer/queue | 본문·선택 경로를 묶은 요청 값·후속 요청 순서·예약 편집·초안 복구·큐 표시 캐시 |
| tui/settings/model | 모델 설정 초안·값 전환·입력 검증·검색·식별 정보·화면 투영 |
| tui/browser | 파일 열 탐색·선택 위치·순수 탐색 상태 |
| tui/browser/readqueue | 디렉터리·미리보기 읽기·배치 계산·감시 등록의 실행 슬롯과 최신 대기 요청 |
| tui/preview | Raw/Markdown 배치 계산·표시 모드·읽기 위치·최신 원문과 폭에 맞는 결과 수용 |
| tui/locale | 불변 번역 목록·설정 우선 언어 선택·시스템 로캘 해석 |
| tui/transcript·view | 메시지/도구 투영·공통 설정 화면 및 대화 렌더링 |
| tui/input·textlayout·selection | 편집 revision·grapheme 배치·원문에 연결된 선택 |
| tui/startup | 대화 선택 |
| tui/viewport | 이력과 현재 출력의 증분 줄 인덱스·가시 구간·스크롤 위치 |
| tui/markdown | 응답·Thinking·파일 뷰어의 Markdown 파싱·스타일·증분 표시 |
| searchmatch | 명령·모델 후보의 Unicode 축약 검색 |

TUI의 활성 worker는 idle·agent·agent stopping·setup·setup stopping 중 한 상태이며 이벤트 루프만 전환한다. TUI는 저장·연결 설정·요청 준비를 직접 소유하지 않는다. worker는 결과 이벤트를 반환하고 화면 상태는 이벤트 루프에서만 바꾼다. 연결 클라이언트·HTTP body·goroutine·timer·DB·자식 프로세스에는 취소와 정리 소유자가 있어야 한다.

TUI·비대화형 실행은 commands/agentrequest.New에서 같은 추론·문맥·압축 설정과 선택된 세션의 정규화한 프로젝트 경로를 사용한다. 요청의 Thinking 선택은 값으로 복사한다. 실제 자원 생성과 종료는 각 진입점, 지침 읽기와 요청 실행은 agent/session이 소유한다.

외부 전송 판정은 AST 연결과 프로토콜 파서로 나누고 명령·파일·DNS·네트워크를 실행하지 않는 순수 함수로 유지한다. `UserRequired`는 일반 승인 필요 여부와 구분하며 All Mighty도 직접 승인을 생략하지 않는다. 승인 창·원본 재검사·배치 실행 전 승인·실제 승인 출처 기록을 공유한다. 정상 다운로드·직접 지정한 loopback HTTP·인용된 명령 본문은 별도로 확인하고, 임의 프로그램 내부 통신까지 제어한다고 설명하지 않는다.

실행 사실과 파일 관측은 agent/tool, 실행 순서는 agent/run, 문맥 조립은 agent/context가 소유한다. 영속 실행 기록은 agent/record가 정의하므로 저장 어댑터·이력·TUI 투영은 실행 루프 구현을 참조하지 않는다. storage/content는 SQL 없이 분리·복원하며 SQLite가 같은 트랜잭션의 쓰기 콜백과 조회 후 복원 콜백을 제공한다. 승인 전경은 파일 뷰어·전송 대기보다 먼저 입력을 처리하고 결정을 마치면 보류한 파일 화면·초안을 복원한다. TUI 하위 컴포넌트는 session을 역참조하지 않으며 상태 변경은 단일 이벤트 루프에서 수행한다. Compaction은 사실을 소비해 요약하며 별도 실행 원장을 만들지 않는다. SQLite는 저장 기능 전체가 아니라 저장 계약의 구현 수단이다.

## 의존성과 배포

정확한 도구 체인과 버전은 [go.mod](../go.mod)를 따른다. 프로젝트 라이선스는 [Apache License 2.0](../LICENSE), 저작권 고지는 [NOTICE](../NOTICE), 외부 구성요소는 [외부 소프트웨어 고지](../THIRD_PARTY_NOTICES.md)를 따른다. 관리자가 단독으로 개발·병합·배포하며 외부 코드 기여는 [기여 정책](../CONTRIBUTING.md)에 따라 받지 않는다.

배포 의존성을 갱신할 때 macOS·Linux의 amd64·arm64 대상으로 `go list -deps -json ./cmd/meca`를 확인하고, 선택된 모듈의 LICENSE·COPYING·NOTICE·AUTHORS·PATENTS와 추가 라이선스 고지를 함께 갱신한다. Go 런타임의 라이선스도 포함한다. 개발 도구와 테스트 전용 모듈은 제품 실행 파일의 의존성 목록과 구분한다.

| 의존성 | 용도 |
| --- | --- |
| tcell/v3·uniseg | 터미널·입력·마우스·문자 경계와 셀 폭 |
| goldmark/v2 | 응답·Thinking·파일 뷰어의 Markdown AST를 TUI span으로 변환 |
| go-toml/v2 | TOML 파싱·직렬화 |
| go-keyring | Keychain·Secret Service |
| modernc.org/sqlite | CGO 없는 SQLite |
| klauspost/compress/zstd | 세션의 대용량 payload를 Zstandard로 압축·복원 |
| bytedance/sonic | 정렬된 키·문자열 검사를 유지하며 모델 요청 JSON을 인코딩 |
| goccy/go-json | 저장 레코드의 JSON 직렬화·참조 분리·복원 |
| x/image | WebP·BMP·TIFF 디코딩; 표준 이미지 코덱과 함께 사용 |
| x/sys·x/term | OS 파일 어댑터와 터미널 확인 |
| mvdan.cc/sh/v3 | 실행 없는 셸 AST 분석과 정적 단어 확장 |
| fsnotify | 파일 탐색 화면의 활성 디렉터리 변경 알림 |

테스트 전용 의존성은 go.uber.org/goleak(고루틴 정리 검사), google/go-cmp(복잡한 값 비교), creack/pty(실행 파일의 터미널 통합 검사)다. 일반 실행 파일에서는 이 패키지들을 import하지 않는다.

CLI 분기는 flag, 병렬 도구 합류는 sync.WaitGroup, 재시도 대기는 time.Timer를 사용한다. 검색은 strings·unicode로 부분일치를 우선하고 결과가 없을 때 문자 순서를 유지하는 축약 일치를 찾는다. 메타데이터 재사용은 연결 준비가 소유한 마지막 GET 응답과 선택한 Ollama 모델의 typed 응답으로 한정한다. 이 사용처에는 별도 CLI·동시성·캐시 프레임워크를 두지 않는다.

Go 빌드만으로 MECA 실행 파일을 만든다. 제품 빌드·실행에 Docker·Bubblewrap·별도 샌드박스 설치를 요구하지 않는다. 빌드·설치·삭제 명령은 [README](../README.md#설치)를 따른다. 소스 설치 스크립트는 Go가 없을 때만 공식 SDK를 체크섬 검증 후 임시 설치하며 작업 후 정리한다. Go 최소 버전을 바꿀 때 `scripts/install/go.sha256`의 네 플랫폼 검증값도 공식 배포 정보와 함께 갱신한다. macOS·Linux 릴리스는 [Release 워크플로](../.github/workflows/release.yml)와 GoReleaser 설정으로 검사·빌드 후 자동 게시한다. 명시적 major·minor와 같은 날짜의 갱신 정책은 [릴리스 가이드](releases.md)가 소유한다.

공식 빌드·소스 설치는 `godbus_no_tcp` 태그로 DBus의 TCP·nonce-TCP 구현을 제외한다. 수동 실행 파일은 `go build -tags=godbus_no_tcp -trimpath -o bin/meca ./cmd/meca`로 만든다. 아래 개발·검증 명령은 같은 셸에서 먼저 `export GOFLAGS='-tags=godbus_no_tcp'`를 설정해 실행한다. 테스트 내부의 자식 `go build`에도 전달되며 전역 Go 설정은 바꾸지 않는다. GoReleaser는 환경변수와 별개로 빌드 태그를 명시한다. 지원 범위는 [Linux 자격 증명 전송](configuration.md#linux-자격-증명-전송)을 따른다.

### 개발 도구

검증·배포 도구는 [devtools/go.mod](../devtools/go.mod)의 별도 모듈에 버전을 고정한다. 제품의 go.mod·go.sum과 의존성 해석을 분리하며 MECA 바이너리에 포함하지 않는다. 도구 실행에는 Go 1.27.3 이상이 필요하다. 다음 명령은 저장소 루트에서 실행한다. 최초 실행 시 Go가 필요한 도구를 내려받고 빌드 캐시에 저장하므로 전역 설치는 필요 없다.

| 도구 | 역할 | 실행 |
| --- | --- | --- |
| Staticcheck | Go 코드의 오류·API 오용·불필요한 연산 검사 | `go tool -modfile=devtools/go.mod staticcheck ./...` |
| govulncheck | 알려진 취약점과 실제 호출 경로 검사 | `go tool -modfile=devtools/go.mod govulncheck ./...` |
| deadcode | 실행 진입점에서 도달하지 않는 함수 탐색 | `go tool -modfile=devtools/go.mod deadcode ./cmd/meca` |
| benchstat | 반복 벤치마크의 시간·할당량·저장 크기 비교 | `go tool -modfile=devtools/go.mod benchstat before.txt after.txt` |
| GoReleaser | 배포 빌드·압축·체크섬·패키지 생성 도구 | `go tool -modfile=devtools/go.mod goreleaser --help` |
| nFPM | Linux 설치 패키지 생성 도구 | `go tool -modfile=devtools/go.mod nfpm --help` |

Staticcheck는 go vet와 함께 사용한다. deadcode는 삭제 후보를 제시하며 `go tool -modfile=devtools/go.mod deadcode -test ./...`로 모든 패키지의 테스트까지 함께 대조한다. 테스트 전용 함수·인터페이스 구현·운영체제별 빌드 조건을 확인한 뒤 정리하며 호출되지 않는다고 표시됐다는 이유만으로 자동 삭제하지 않는다.

다른 운영체제의 코드를 검사할 때는 먼저 호스트용 도구를 빌드한 뒤 분석 대상 환경을 지정한다. `GOOS=linux go tool ...`은 도구 자체도 Linux용으로 빌드하므로 macOS에서 실행할 수 없다. 다음 검사는 Linux 코드의 분석이며 Linux에서 실제 실행한 검증과 구분한다. 도구 버전을 갱신하면 호스트 도구도 다시 빌드한다.

```sh
go build -modfile=devtools/go.mod -o bin/devtools/ honnef.co/go/tools/cmd/staticcheck golang.org/x/vuln/cmd/govulncheck golang.org/x/tools/cmd/deadcode
GOOS=linux GOARCH=amd64 ./bin/devtools/staticcheck ./...
GOOS=linux GOARCH=amd64 ./bin/devtools/govulncheck ./...
GOOS=linux GOARCH=amd64 ./bin/devtools/deadcode -test ./...
```

govulncheck는 Go 취약점 데이터베이스를 조회한다. 보고서의 영향 버전·호출 경로·수정 버전을 확인하고, 취약점이 발견된 경우 의존성 변경과 관련 테스트를 함께 검토한다. 검사에 실패하거나 데이터베이스에 접근할 수 없었던 경우를 취약점 없음으로 처리하지 않는다.

GoReleaser는 [.goreleaser.yaml](../.goreleaser.yaml)의 macOS·Linux 아카이브를 생성한다. `check`와 게시 없는 `release --snapshot --clean`으로 로컬 검증하며 `main` 푸시는 자동 게시를 시작한다. 릴리스 준비·게시 경계는 `go test ./scripts/release`로 검사하고 워크플로 변경 시 Actions 문법도 확인한다. nFPM은 Linux 설치 패키지가 필요할 때 사용하는 도구로 등록되어 있다. 현재 배포 형식은 tar.gz이며 제품 실행·일반 빌드·테스트에는 이 도구들이 필요하지 않다.

버전 갱신은 `go get -modfile=devtools/go.mod -tool <패키지>@<버전>`으로 수행한다. 이후 `go -C devtools mod tidy`와 `go -C devtools mod verify`를 실행하고 두 개발 도구 파일의 변경을 검토한다. 도구 요구 버전이 제품의 의존성이나 최소 Go 버전을 바꾸지 않도록 유지한다.

## 변경 검증

다음은 실행 지침이며 이 문서 편집 시의 통과 기록이 아니다. 작업 범위에 맞는 집중 검사 후 공유 계약 변경은 전체 검사로 넓힌다.

```sh
export GOFLAGS='-tags=godbus_no_tcp'
gofmt -l cmd internal
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
```

순수 상태·파서·승인·저장·Provider 변환은 작은 fixture와 httptest로 검증한다. 파일 검사는 t.TempDir, TUI는 저장소의 가상 터미널과 실제 키 입력을 사용한다. 모델 응답을 고정한 검사는 실제 모델의 완수율이나 추론 품질 검증과 구분한다.

에이전트 실행·HTTP 생성·TUI 테스트는 패키지 종료 시 goleak으로 남은 고루틴을 검사한다. 구조체 복원과 대화 투영은 go-cmp로 필드 차이를 확인한다. PTY 통합 검사는 임시 HOME·프로젝트에서 실제 실행 파일을 시작해 터미널 크기 변경, Ctrl+C·!quit 종료, 터미널 설정 복원과 DB 잠금 해제를 확인한다. 설치 PTY 검사는 PATH가 필요한 대화형 설치만 동일 PID의 로그인 셸로 전환하는지, 전환 전에 설치 잠금이 풀리는지와 --no-shell·비대화형 동작을 확인한다. 소스 설치 검사는 오프라인 SDK fixture로 빌드·임시 SDK 정리·호출 환경 복원·설치 순서를 확인한다. PTY의 소유 프로세스가 종료 후 상태를 검사하고, 실패·시간 초과에는 해당 프로세스 그룹과 읽기 작업을 정리한다. `go test -short`에서는 실행 파일을 빌드하는 통합 검사를 생략한다.

외부 전송 검사는 모의 실행 파일과 loopback HTTP만 사용한다. Default·All Mighty의 매 호출 직접 승인·거절·비대화형 승인 불가와 배치의 실행 전 중단·native 결과 짝·승인 출처를 확인한다. 실제 외부 업로드를 발생시키지 않는다. 확인 키는 소문자 n 무시·대문자 N 취소를 설정·파일 전송 화면에서 검증한다.

```sh
go test ./cmd/meca -run TestTerminalExecutable -v
go test ./cmd/meca -run TestInstallShellExecutable -v
go test ./internal/commands/install -run TestSourceInstaller -v
go test ./internal/tui/session -run TestFileWatch -v
```

문서 변경은 실제 코드·기본값·링크·diff를 확인한다. 문서 검증은 변경된 설명과 참조 범위에 맞춰 수행한다. 코드의 헤더·주석·테스트는 해당 파일의 책임과 실제 동작을 설명하도록 유지한다.

## 플랫폼 실행 검증

Linux·WSL·macOS에서 호스트 셸의 작업 디렉터리·환경·프로세스 정리와 필수 승인 거절을 검증한다. 임시 HOME에서 캐시 생성, 하드링크, 호스트 DNS 설정 파일, 자식 프로세스·로컬 HTTP, 시작 실패·비정상 종료·취소 후 부분 출력을 확인한다.

```sh
go test ./internal/tools/local/... ./internal/agent/run ./internal/commands/headless
```

실제 uv 설치 경로 검사는 다음의 opt-in 진입점을 사용한다. MECA_TEST_UV_PYTHON_ROOT로 관리 Python 트리를 지정할 수 있다. 이는 시험 fixture의 입력이며 제품의 언어별 실행 분기가 아니다. fixture는 사용자 캐시 쓰기 예외나 링크 모드 변경 없이 임시 HOME을 사용한다.

```sh
MECA_TEST_UV_BINARY=/path/to/uv go test ./internal/tools/local/process -run TestUVPackageProject -v
```

실제 WSL에서는 일반 터미널과 execute_command에서 같은 도메인 조회·HTTPS 다운로드·패키지 설치를 대조한다. DNS 설정 파일 읽기와 숫자 IP의 로컬 HTTP 통과만으로 외부 DNS 성공을 판단하지 않는다. Linux Secret Service·실제 Provider·터미널 키 전달도 수행한 환경과 결과를 구분해 기록한다. 교차 빌드는 네이티브 실행 검증을 대신하지 않는다. [실행 경계](execution-boundary.md)의 호스트 권한 계약을 함께 확인한다.

Linux의 필수 DBus 회귀 검사는 실제 자격 증명 조회 경로를 새 테스트 프로세스에서 호출한다. 루프백 TCP·nonce-TCP 연결과 canary 전송이 없고 Unix 소켓 연결은 유지되는지 확인한다. `linux` 조건만 사용하므로 태그를 빠뜨린 Linux 테스트는 실패한다. Unix 연결 검사만으로 키 저장·조회 성공을 판단하지 않는다.

```sh
GOFLAGS='-tags=godbus_no_tcp' go test ./internal/config/credentials -run TestCredentialDBusTransport -v
```

실제 Secret Service 검사는 별도의 Unix 세션 버스와 잠금 해제된 키 저장소를 준비한 Linux 환경에서 다음과 같이 실행한다. 현재 사용자 키 저장소 대신 격리된 서비스 환경을 사용한다. 임시 서비스 이름으로 저장·조회·삭제하고 정리하며, 기본 테스트에서는 이 opt-in 검사만 생략한다.

```sh
MECA_TEST_SECRET_SERVICE=1 GOFLAGS='-tags=godbus_no_tcp' go test ./internal/config/credentials -run TestCredentialSecretService -v
```

## 성능과 장시간 수명

입력은 grapheme 경계를 보존하고 붙여넣기를 묶어 처리한다. 커서 탐색은 약 256 rune 간격의 parser 상태 체크포인트에서 재개하며 편집 위치 이후의 인덱스만 폐기한다. 큰 삭제와 초안 교체는 불필요한 인덱스 용량을 해제한다. 입력 배치는 줄별 rune 시작 위치를 한 번 기록해 커서 줄을 이진 탐색하고 그 줄 안에서만 바이트 위치와 선택 범위를 계산한다. 출력 가능한 ASCII 폭은 바이트 길이로 계산하고 나머지는 Unicode 폭 규칙을 사용한다. 이미 순회 중인 grapheme의 폭은 iterator에서 재사용한다. 변경되지 않은 source/width 배치와 Markdown 투영을 재사용하고 연속 화면 갱신을 합치되 순서 있는 이벤트는 버리지 않는다. 요청 worker가 표시용 최신 스트림 mailbox를 소유하고 재시도·완료·승인·사용량 경계에서 이전 mailbox를 분리한다. 소비 후 본문 참조를 해제하며 별도 전송 goroutine을 만들지 않는다. 기록·파일 열·선택·뷰포트 캐시를 해제할 때 참조도 정리한다.

대화 줄 인덱스는 변경되지 않은 이력의 접두사를 보존하고 추가 이력과 현재 출력만 배치한다. 폭·이력 소유 배열·prepend·잘라내기·도구 접기 상태 변경은 인덱스를 다시 구성한다. 가시 블록 목록만 비교해 화면 밖 파싱 참조를 해제한다. 자동 갱신은 60Hz 간격과 최근 그리기 비용의 3배 중 큰 시간을 기다리며 추가 대기는 최대 250ms다. 입력은 이전 프레임 시작 기준 다음 60Hz 시점에 표시할 수 있다. 느린 자동 갱신 직후의 입력은 한 번 즉시 표시하고 연속 입력 프레임은 60Hz 예산을 따른다. 입력 배치는 최대 32개이며 런타임 알림과 공정하게 처리하고 붙여넣기·종료 순서를 보존한다.

후속 요청 FIFO와 큐 편집은 TUI 이벤트 루프가 소유한다. 편집은 선택 위치와 작성 중인 Editor 값만 보관하며 큐 전체를 복제하지 않는다. 목록은 전체 항목을 번호순으로 표시하고 항목별 마스킹·줄 배치를 재사용한다. 각 요청은 예약 시 본문과 선택 경로를 값으로 캡처한다. 편집은 본문만 바꾸며 선택 변경은 이미 예약된 요청의 참조에 영향을 주지 않는다. 요청 worker가 실제 전달 전에 참조를 재확인하고, 초안 복구는 캡처한 경로도 보존한다. 원문 또는 마스킹 revision이 바뀐 항목만 표시 문자열을 갱신하며 선택 색상 전환은 본문 재배치를 유발하지 않는다. 화면 밖 항목 선택은 transcript의 위치 탐색을 사용한다. 출력 증가로 live block ID가 바뀌면 큐 기준 읽기 위치를 보정하고 창 크기 변경 시 선택 항목을 다시 보이게 한다. 사용자가 직접 과거 출력으로 이동한 위치는 유지한다. 편집 중에는 dequeue를 보류해 선택 위치를 유지하고, 현재 worker의 완료·실패를 편집 상태에 기록해 닫을 때 실행·복구를 이어간다. 미전달 방향 수정은 큐 편집 본문 대신 보관된 일반 초안에 복구한다. 소비·삭제·복구한 큐 항목과 표시 캐시의 제거된 항목은 참조를 지우고 빈 큐와 닫힌 편집 상태는 해제한다. 별도 worker나 모델 호출을 추가하지 않는다.

디렉터리 조회와 파일 미리보기는 각각 실행 1개·최신 대기 요청 1개를 가진다. 취소는 대기 요청을 제거하지만 실행 슬롯은 실제 I/O 완료 알림에서 해제한다. 이벤트 루프가 최신 요청만 시작하며 이전 revision의 결과는 적용하지 않는다. 읽기와 배치 계산은 별도 슬롯을 사용한다. Raw/Markdown의 최초·폭 변경·모드 전환 배치는 단일 계산 worker에서 준비하며 이벤트 루프는 준비된 가시 구간만 조회한다. 원문·폭·모드와 revision이 일치하는 결과만 수용하고 비활성 레이아웃은 해제한다. Markdown은 viewport의 파싱·희소 줄 배치와 스타일 렌더러, Raw는 줄 번호 인덱스를 사용한다. 전체화면은 숨은 목록·입력창 배치를 생략한다. Shift+R은 같은 읽기 본문의 표시 형식만 전환하고 비활성 레이아웃 캐시를 해제한다. 내용·폭 변경은 배치를 갱신하고 같은 보기의 파일 갱신은 모드와 읽기 위치를 보존한다. 종료는 대기를 비우고 실제 worker를 합류한 뒤 슬롯·파싱 참조를 해제한다.

파일 변경 감시는 활성 디렉터리 하나에 연결한다. 플랫폼별 등록 I/O도 worker에서 수행하며 실행 1개·최신 대기 1개로 제한한다. 이벤트 루프가 감시 채널과 150ms 단발 타이머를 소비하고, 등록 직후와 알림 유실 시 재조회한다. 경로 전환·취소·화면 닫기에서 감시와 타이머를 정리하며 늦게 도착한 감시 객체도 닫는다. 갱신은 파일 읽기 슬롯을 사용하고 새 폴링 루프를 만들지 않는다.

명령 실행기는 요청의 전체 마감 시각·취소 문맥을 한 번 바인딩하고 최대 4개 명령을 소유한다. 최초 30초 대기와 wait_command의 기본 180초 대기는 각각의 관측 창이며 프로세스 수명과 분리한다. Wait는 한 번만 실행하고, running 결과의 next와 미완료 명령 피드백은 공통 도구 관리 안내를 사용한다. TUI는 경과 시간 타이머에서 작은 메타데이터만 조회한다. 완료 출력은 정제한 실행기 소유 파일로 옮겨 메모리 버퍼를 해제하고 런타임 소유자가 SQLite 기록을 확정한 뒤 핸들을 제거한다. 활성 명령이 있는 모델 요청은 I/O worker의 이벤트를 소유자에게 동기 전달하고, 소유자는 완료 결과를 수집한다. 모든 worker·타이머는 종료 시 정리하며 기록 실패를 성공으로 처리하지 않는다.

OpenRouter reasoning은 수신 시 현재 블록을 점진적으로 병합한다. 서명·종류·식별자 경계와 알 수 없는 필드를 보존하고 닫힌 블록만 유지하므로 완료 시 모든 조각을 다시 해석하지 않는다. 취소·실패한 요청의 미완성 블록은 후속 요청에 공개하지 않는다.

Provider 요청·native 호출 이력의 JSON 인코딩은 protocol의 불변 Sonic 설정을 사용한다. HTTP JSON 본문에서 불필요한 HTML 이스케이프는 생략하고 키 정렬·문자열 검사·RawMessage 검증을 유지한다. LM Studio는 Responses 형식으로 직접 인코딩한다. 스트림과 OpenRouter continuation의 파싱·병합은 표준 encoding/json을 사용한다. 도구 인자의 중복 키·종류·알 수 없는 필드·후행 입력 검사도 별도 경계에서 유지한다.

하나의 요청 실행은 전용 Provider 인코더를 소유한다. 큰 텍스트 이력의 변하지 않은 메시지 JSON을 재사용하고 헤더·추론 설정·도구 선언은 매번 구성한다. 메시지 변경은 이후 접미사를 폐기하며 모델·Provider 변경이나 이력 축소는 전체 캐시를 해제한다. 캐시는 메시지별 보수적 부대비용을 포함해 약 64KiB부터 사용하고 보유 예산은 16MiB다. 작은 이력·이미지·예산 초과 이력은 캐시를 보유하지 않으며 입력 수용 한도는 유지한다. HTTP 본문은 불변 인코딩 조각을 읽고 정확한 Content-Length와 독립된 재읽기 커서를 제공해 재시도·리다이렉트를 보존한다. 실행 종료·실패·취소는 worker를 합류한 뒤 캐시 참조를 해제한다. vLLM의 정확한 토큰 수 사전 검사는 별도 요청으로 유지한다.

메시지·실행 이벤트·요약·대용량 레코드 참조의 저장 JSON은 go-json의 기본 Marshal·Unmarshal을 사용한다. 키 정렬·HTML 이스케이프·nil과 빈 값·64비트 숫자·native arguments의 내용·복원 값의 독립성을 검사한다. SHA-256 무결성·저장 형식·트랜잭션 경계는 storage/content와 SQLite의 소유권을 따른다. 반환 값이 입력 버퍼에 의존하는 무복사 옵션은 사용하지 않는다.

HTTP는 표준 net/http를 사용한다. 사용자 지정 DialContext에서도 TLS ALPN으로 HTTP/2를 협상하고 HTTP/1.1 서버도 지원한다. 연결·TLS handshake·유휴 수신 감지는 각 경계의 제한을 유지하며 전체 응답 시간 제한을 추가하지 않는다. 응답 body·요청 취소·연결 종료는 transport가 소유한 수명으로 정리한다.

로컬 도구 카탈로그는 초기화 후 변경하지 않는 내부 정의 하나를 소유한다. 인자 검사는 이를 재사용하며 외부 Definitions 반환값은 중첩 Fields·Values까지 복사해 변경이 내부 검증으로 전파되지 않게 한다.

응답별 마스킹은 누적 스냅샷에서 새 구간만 검사하고 비밀키 접두사·미완성 UTF-8을 보류한다. 응답과 Thinking은 상태를 공유하지 않으며 재시도마다 초기화한다. 파일 미리보기와 대화 본문은 희소 위치 인덱스로 가시 구간만 구체화한다. 일반 본문은 256개 표시 줄마다 원문 위치를 기록해 긴 한 줄도 중간부터 재개한다. 응답·Thinking의 Markdown은 마지막 두 최상위 블록을 가변 구간으로 두고 완성된 앞부분의 파싱·줄 수를 재사용한다. 참조 정의 가능성이 있는 `]:` 포함 문서는 전체 파싱을 유지한다. 화면 밖 과거 응답·Thinking은 파싱 결과·span을 해제하고 위치 인덱스만 남기며 현재 스트리밍은 증분 처리 수명 동안 보유한다. 경로 표시는 선택 revision·폭·언어·마스킹 revision으로 캐시하고 실제 표시 예산을 채우면 변환을 끝낸다. Compaction은 시간 순서 목록과 우선순위별 제거 대기열로 사용자 지시·결과·실패의 보존 순서를 유지한다.

Markdown의 Render·Stream.Render는 뷰포트의 실제 본문 폭을 받는다. 문법 설정은 공유하고 reader·문맥·AST는 호출마다 분리한다. 같은 스타일의 연속 inline 텍스트는 필요한 경우에만 builder로 합쳐 긴 문단의 반복 복사를 피하며 명시적 줄바꿈 조립에 임시 분할 배열을 만들지 않는다. 표의 열 너비는 내용과 구분자·들여쓰기 예산으로 계산하고 셀별 styled Rows를 조립하므로 표 전체를 하나의 긴 줄로 줄바꿈하지 않는다. 같은 폭의 증분 파싱과 완료 블록 재사용은 유지하며 폭 변경 시 이전 표 배치와 재사용 줄 수를 초기화한다. 희소 인덱스의 줄 수 계산과 가시 구간 출력은 같은 폭의 배치 결과를 사용한다. 좁은 폭에서는 헤더·값을 세로로 구성해 열 구분자만 남거나 내용이 옆 열을 침범하는 상황을 피한다.

파일 변경 미리보기와 정확한 추가·삭제 줄 수는 같은 분석에서 얻는다. 공통 앞뒤 구간은 bytes.Equal 블록 비교로 건너뛰고 미리보기에 필요한 인접 문맥만 매칭한다. 전체 생성·삭제는 trace와 줄 배열 없이 출력한다. 미리보기의 작업량 한도에 도달한 경우에는 완전한 diff를 출력하고 정확한 변경량은 별도로 계산한다. 원본 검사·해시·변경 후보 바인딩은 유지한다.

큰 JSON 값의 인코딩·복원과 무결성 검사는 storage/content가 소유하고 SQLite는 `payloads`에 해시·원본 길이·압축 형식·본문을 저장한다. 새 payload는 Zstandard의 빠른 단일 실행 설정과 128KiB 창을 사용하며 압축 결과가 원문보다 작을 때 저장한다. 압축기·복원기는 Store가 지연 생성해 잠금 안에서 재사용하고 Close에서 해제한다. 재사용 출력 버퍼는 최대 256KiB로 제한하며 호출자에게는 독립된 결과를 반환한다. 복원 전 프레임의 선언 길이를 확인하고 출력 용량·해시·JSON 검사를 적용한다. 원문·gzip·Zstandard는 명시한 codec 값으로 구분한다. 작은 레코드는 payload 참조가 필요 없으면 재파싱·재직렬화를 생략하고 내용에 임의의 표식을 삽입하지 않는다. 한 연결을 쓰므로 조회 페이지를 닫은 뒤 payload를 복원한다. 기록과 payload는 같은 트랜잭션으로 커밋한다. `record.Entry` 그룹은 같은 확정 시점의 메시지·이벤트·체크포인트를 원자적으로 저장한다. 실행 전 시작 의도는 효과 발생 전에, 실제 실행 결과·native 응답·변경 후 체크포인트는 취소와 무관하게 5초 저장 문맥에서 함께 확정한다. 실행 시작과 결과의 두 경계를 합치지 않는다. 종류·본문 길이·요약 입력 투영을 저장 시 기록하며 원본 읽기·export·취소 후 복원 계약을 함께 검사한다.

압축 점유량 조회는 최근 실측 샘플·최신 요약 revision의 경계·이후 추가된 메시지 크기·현재 timeline을 한 SQL 문과 같은 읽기 스냅샷에서 계산한다. 원문 본문을 읽거나 추가 영속 카운터·캐시·인덱스를 만들지 않는다.

컨텍스트 투영 캐시는 요청 실행이 소유하고 변경되지 않은 불변 메시지만 재사용한다. native 도구 결과 짝이 완성되거나 메시지·모델·비밀키 revision이 바뀌면 다시 투영한다. Compaction이 이력을 교체하면 캐시를 해제한다. 전체 미압축 대화와 Provider 전송 본문은 유지한다. 요청 인코딩 재사용은 별도 Provider 인코더가 담당하며 네트워크 전송량·모델 토큰 수를 줄이지 않는다.

대화 선택은 디렉터리의 DB 파일이 식별 근거다. 최초 조회는 private 메모리 SQLite 연결 하나에 파일을 읽기 전용으로 하나씩 attach/detach하고 요약 배열 하나와 정렬 시각만 보유한다. 별도 레지스트리·sidecar·상시 worker를 만들지 않으며 정상·오류·취소 시 연결을 닫는다. 페이지 이동은 선택 화면이 소유하는 스냅샷을 사용한다. 전체 정렬을 위해 최초 조회는 파일 수에 비례한다.

### 최적화 측정

2026-10-31, Go 1.27.3·macOS arm64·Apple M1 Max에서 측정했다. 같은 부하의 비교 기준과 적용 구현을 각각 반복 실행한 중앙값이다. 줄 배치는 5회, 터미널과 첫 요청 인코딩은 6회, HTTP는 8회 반복했다.

| 측정 | 부하 | 비교 기준 | 적용 구현 | 감소율 |
| --- | --- | --- | --- | --- |
| 화면 갱신 준비 | 이력 1,000개·80셀 폭·30줄 조회 | 19.829µs | 2.681µs | 86.5% |
| 평균 입력 표시 지연 | 프레임당 12ms 지연의 가상 터미널 | 30.54ms | 14.21ms | 53.5% |
| 자동 그리기 시간 점유 | 같은 가상 터미널·타이핑 없음 | 43.95% | 27.57% | 37.3% |
| 로컬 HTTP 요청 처리 | 고정된 8MiB 이력과 바뀌는 마지막 메시지 | 7.666ms | 1.504ms | 80.4% |

같은 HTTP 부하의 요청당 힙 할당량은 약 23,093KiB에서 126KiB로 줄었다. 짧은 2KiB HTTP 요청과 새 인코더의 첫 8MiB 인코딩은 처리 시간에 통계적으로 유의미한 차이가 없었다. 자동 그리기 점유는 CPU 사용률이 아니며 할당량은 RSS가 아니다. 화면·요청 준비·로컬 전송의 측정값을 모델 추론 TPS·원격 응답 시간·전체 작업 완수율로 해석하지 않는다.

캐시·비캐시 요청의 Provider별 JSON 값, native 호출·결과 순서, 변경·축소·모델 전환, 이미지·예산 초과, 본문 재읽기와 HTTP 재시도·리다이렉트, 완료·실패·취소 후 참조 해제를 검사했다. 전체 테스트, 관련 패키지 레이스 검사, go vet·Staticcheck·deadcode와 macOS·Linux의 amd64·arm64 교차 빌드를 수행했다. 네이티브 실행 확인은 macOS arm64의 도움말 호출이며 실제 Linux·WSL 터미널이나 실제 모델 추론의 성능 측정과 구분한다.

### 재현과 추가 검사

성능·수명 검사는 다음을 사용한다. 수치는 실행한 환경·입력·측정 종류를 함께 기록하고 힙 할당량을 RSS와 혼동하지 않는다. 요청 인코딩 벤치마크의 `cache=false`와 `cache=true`는 같은 부하의 비캐시·캐시 경로다. 새 인코더의 첫 요청은 ColdConversationEncoding으로 구분한다.

```sh
go test ./internal/agent/context -run '^$' -bench BenchmarkProjection -benchmem
go test ./internal/tui/viewport -run '^$' -bench BenchmarkLiveTranscriptFrame -benchmem
go test ./internal/tui/session -run '^$' -bench BenchmarkTerminalPacing -benchtime=1x -count=6
go test ./internal/provider/protocol -run '^$' -bench 'Benchmark(ConversationEncoding|ColdConversationEncoding)' -benchmem -count=6
go test ./internal/provider/httpclient -run '^$' -bench BenchmarkGenerationEncoding -benchmem -count=8
go test ./internal/tui/markdown -run '^$' -bench 'Benchmark(StreamTail|Paragraph)' -benchmem
go test ./internal/tui/input -run '^$' -bench BenchmarkLongDraft -benchmem
go test ./internal/tools/local/diff -run '^$' -bench BenchmarkAnalysis -benchmem
go test ./internal/tools/local -run '^$' -bench BenchmarkParallelPlanning -benchmem
go test ./internal/tools/local/catalog -run '^$' -bench BenchmarkCatalog -benchmem
go test ./internal/tui/session -run '^$' -bench BenchmarkStreamDelivery -benchmem
go test ./internal/storage/sqlite -run '^$' -bench BenchmarkOutcomeCommit -benchtime=100x -benchmem
go test ./internal/storage/content -run '^$' -bench 'Benchmark(PayloadCodec|CompressionOptions)' -benchmem
go test ./internal/provider/protocol -run '^$' -bench 'Benchmark(ProviderRequests|ProviderStream|OpenRouterReasoningFragments)' -benchmem
go test ./internal/provider/protocol -run '^$' -bench 'Benchmark(TextMessages|ImageEncoding)$' -benchmem
go test ./internal/tools/local -run '^$' -bench BenchmarkReadImageBytes -benchmem
go test ./internal/storage/sqlite -run '^$' -bench 'BenchmarkImage(History|Boundary)$' -benchmem
go test ./internal/storage/content -run '^$' -bench BenchmarkRecordContent -benchmem
go test ./internal/tui/textlayout -run '^$' -bench BenchmarkWidth -benchmem
go test ./internal/storage/sqlite -run '^$' -bench BenchmarkMemoryWatermark -benchmem
go test ./internal/storage/sqlite -run '^$' -bench 'Benchmark(Conversations|Journal)' -benchmem
go test ./internal/tui/viewport -run TestDisplayResources -v
go test ./internal/tui/markdown -run '^$' -fuzz '^FuzzStream$' -fuzztime=10s
go test ./internal/tui/input -run '^$' -fuzz '^FuzzBoundaryEdits$' -fuzztime=10s
go test ./internal/tools/local/diff -run '^$' -fuzz '^FuzzAnalysisCounts$' -fuzztime=10s
go test ./internal/provider/protocol -run '^$' -fuzz '^FuzzWireJSONEncode$' -fuzztime=15s
go test ./internal/provider/protocol -run '^$' -fuzz '^FuzzEncoderRequest$' -fuzztime=10s
go test ./internal/storage/content -run '^$' -fuzz '^FuzzStoredJSON$' -fuzztime=15s
```

TerminalPacing의 느린 화면은 프레임마다 12ms 지연을 주는 가상 터미널이다. 입력 간격은 29~61ms로 달리해 프레임 주기와의 고정된 정렬을 피한다. drawing-percent는 화면 출력이 차지한 경과 시간이며 CPU 사용률이 아니다. echo-mean-us는 입력 표본 8개의 평균이고 echo-p95-us는 그중 최댓값이다. GenerationEncoding은 로컬 HTTP 서버까지 본문을 전송하고 스트리밍 응답을 읽지만 실제 모델 추론은 수행하지 않는다.

```sh
go test ./internal/config/credentials ./internal/tui/viewport ./internal/tui/view ./internal/tui/preview ./internal/storage/sqlite ./internal/tools/local/process -run '^$' -bench . -benchmem
```

벤치마크에는 Go·OS·CPU·데이터 크기·창 너비·명령과 소스 식별값을 함께 기록한다. 합성 줄 배치 수치를 모델 TPS나 전체 작업 시간으로 해석하지 않는다. 장시간 heap/RSS·취소 후 자원·실제 스트림의 프로파일링은 별도 확인한다.

실행 중 명령 조회는 출력 본문을 복사하지 않고 상태와 시각을 확인한다. 스트림별 변경 번호가 바뀐 경우만 본문을 복사·마스킹하고 큰 로그를 갱신한다. 캐시는 작은 발췌·경로·변경 번호만 유지하고 명령별로 동기화해 독립 명령의 I/O를 직렬화하지 않는다. 결과 수집·요청 종료·Executor 종료에서 캐시 참조를 해제한다.

## 요청 실행의 책임

메인 에이전트 하나가 native 도구와 실제 대화 문맥을 사용해 요청을 수행한다. agent/run은 실행 순서·예산·승인·명령 수명을 조율하고, 시작한 명령의 종료·취소 결과를 수집한 뒤 모델의 정상 응답으로 종료한다. 파일 검사·OS 자격 증명·세션 SQLite는 각 어댑터가 소유한다.

원본 출력은 실행 이벤트가 소유하고, 작은 체크포인트·TUI·모델 문맥은 각각 필요한 투영만 사용한다. 비대화형 실행과 TUI는 agent/session을 공유하며 테스트가 별도 실행 루프를 만들지 않는다. cmd/meca의 실행 파일 통합 검사는 임시 HOME·프로젝트와 모의 Provider를 사용하며 실모델 E2E와 구분한다.

## 이미지 처리 책임

agent/media는 이미지 스냅샷의 값·수용 한도, provider/connection은 입력 지원 상태, provider/discovery는 서버 선언 해석, provider/diagnostic은 입력 수용 검사, provider/setup은 연결 확정을 소유한다. provider/protocol의 요청 DTO는 수신 청크와 분리하고 chat·Ollama·Responses 형식으로 이미지를 인코딩한다. commands/agentrequest와 agent/run은 연결에 맞게 도구 목록을 구성한다.

tools/local/imagefile은 파일 내용의 형식 판별·디코딩·대표 프레임 변환을 담당하고 local/image는 기존 경로 경계와 취소 가능한 단일 디코딩 수용량을 적용한다. local/image_read는 알려진 크기의 스냅샷 읽기와 크기 변경 검사를, imagefile/png는 PNG 청크 기반 애니메이션 판별을 담당한다. 이미지 본문은 model 메시지에만 보존하며 TUI 이벤트에는 작은 메타데이터를 전달한다.

storage/content·sqlite는 기존 payload 저장을 재사용한다. sqlite의 RecentInputs는 사용자 텍스트만, PendingTools는 미완료 호출의 인자만 복원한다. PendingTools의 식별자 조회와 인자 선택은 별도 파일이 담당하며 부분 완료된 배치도 원래 순서를 유지한다. 모델 문맥 조회는 이미지 포함 여부를 명시하고 텍스트 연결에서는 이미지 payload를 읽지 않는다. 요약·화면 조회도 이미지 바이너리를 제외한다. 이미지 미소비 그룹은 압축 경계를 제한한다. 형식·검사·용량 계약은 [이미지 입력](images.md)을 따른다.
