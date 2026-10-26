# 개발 가이드

[← README](../README.md) · [제품 명세](../SYSTEM.md)

## 책임과 의존성

제품은 단일 Go 모듈과 cmd/meca 합성 루트를 사용한다. 기능은 internal 아래에 두고 실제 소유 패키지를 직접 참조한다. 소비자 소유의 작은 I/O 계약과 함수·struct를 우선하며 패키지는 한 기능, 파일은 그 안의 한 책임을 담당한다.

| 위치 | 책임 |
| --- | --- |
| cmd/meca | CLI 분기·어댑터 조립·프로세스와 터미널 수명 |
| agent/model | 모델 메시지·스트림·출력 예산 계약 |
| agent/session·run | 요청 준비·실행 순서·승인 조율·공유 예산·요청별 실측 통계 |
| agent/context·compact | 모델 문맥과 단일 호출 요약 |
| commands/headless | 같은 요청 경로의 text·JSONL 출력 어댑터 |
| commands/agentrequest | TUI·비대화형 실행의 설정·모델·세션을 공통 요청으로 조립 |
| agent/tool·work | 실행 사실·가벼운 파일 관측·종료 원인 |
| agent/prompt·inference | 작업 지침·역할별 추론 설정 |
| commands/startup·install·settings | 시작 경로·사용자 설치와 삭제·언어 설정 저장 |
| config/bootstrap·prompts·statefs·credentials | 초기화·Markdown 파일·private 상태 수명·OS 자격 증명 |
| provider/connection·identity·selection | 연결 값·서버 원본 표시·Provider 추론 옵션·선택 검증 |
| provider/discovery·diagnostic·setup | 메타데이터·실제 생성 진단·연결 저장과 소유권 이전 |
| provider/protocol·transport·httpclient | wire 변환·HTTP/스트림/재시도·생성 요청 조율 |
| policy·policy/command | 공통 보호 대상·파일 Scope와 execute_command의 선택적 AST 검사·값 상태·명령별 승인 근거 |
| tools/local/filesystem | 실제 경로·프로젝트 identity·파일 도구의 private 경로 검사 |
| tools/local/file·textedit·textformat·diff | 파일 I/O·정확한 부분 수정·바이트 형식·변경 미리보기 |
| tools/local/process | 호스트 셸·프로세스 그룹·출력·취소 |
| storage/session·history·recovery | 세션 식별·순서 있는 이력·중단 복구의 저장 계약 |
| storage/sqlite | DB 파일·SQL·트랜잭션·잠금 어댑터 |
| tui/session | 이벤트 루프·활성 화면·모델 및 언어 설정 초안·worker 수명 |
| tui/locale | 불변 번역 목록·설정 우선 언어 선택·시스템 로캘 해석 |
| tui/modelmenu·transcript·view | 모델 표시값·메시지/도구 투영·공통 설정 화면 및 대화 렌더링 |
| tui/input·textlayout·selection | 편집 revision·grapheme 배치·원문에 연결된 선택 |
| tui/startup·browser·viewport·markdown | 대화 선택·파일 탐색·출력 위치·응답 Markdown 표시 |
| searchmatch | 명령·모델 후보의 Unicode 축약 검색 |

TUI의 활성 worker는 idle·agent·agent stopping·setup·setup stopping 중 한 상태이며 이벤트 루프만 전환한다. TUI는 저장·연결 설정·요청 준비를 직접 소유하지 않는다. worker는 결과 이벤트를 반환하고 화면 상태는 이벤트 루프에서만 바꾼다. 연결 클라이언트·HTTP body·goroutine·timer·DB·자식 프로세스에는 취소와 정리 소유자가 있어야 한다.

TUI·비대화형 실행은 commands/agentrequest.New에서 같은 추론·문맥·압축 설정과 선택된 세션의 정규화한 프로젝트 경로를 사용한다. 요청의 Thinking 선택은 값으로 복사한다. 실제 자원 생성과 종료는 각 진입점, 지침 읽기와 요청 실행은 agent/session이 소유한다.

실행 사실과 파일 관측은 agent/tool, 실행 순서는 agent/run, 문맥 조립은 agent/context가 소유한다. Compaction은 사실을 소비해 요약하며 별도 실행 원장을 만들지 않는다. SQLite는 저장 기능 전체가 아니라 저장 계약의 구현 수단이다.

## 의존성과 배포

정확한 도구 체인과 버전은 [go.mod](../go.mod)를 따른다. 프로젝트 라이선스는 [Apache License 2.0](../LICENSE), 저작권 고지는 [NOTICE](../NOTICE), 외부 구성요소는 [외부 소프트웨어 고지](../THIRD_PARTY_NOTICES.md)를 따른다. 관리자가 단독으로 개발·병합·배포하며 외부 코드 기여는 [기여 정책](../CONTRIBUTING.md)에 따라 받지 않는다.

배포 의존성을 갱신할 때 macOS·Linux의 amd64·arm64 대상으로 `go list -deps -json ./cmd/meca`를 확인하고, 선택된 모듈의 LICENSE·COPYING·NOTICE·AUTHORS·PATENTS와 추가 라이선스 고지를 함께 갱신한다. Go 런타임의 라이선스도 포함한다. 개발 도구와 테스트 전용 모듈은 제품 실행 파일의 의존성 목록과 구분한다.

| 의존성 | 용도 |
| --- | --- |
| tcell/v3·uniseg | 터미널·입력·마우스·문자 경계와 셀 폭 |
| goldmark/v2 | 응답 Markdown AST를 TUI span으로 변환 |
| go-toml/v2 | TOML 파싱·직렬화 |
| go-keyring | Keychain·Secret Service |
| modernc.org/sqlite | CGO 없는 SQLite |
| x/sys·x/term | OS 파일 어댑터와 터미널 확인 |
| mvdan.cc/sh/v3 | 실행 없는 셸 AST 분석과 정적 단어 확장 |
| fsnotify | 파일 탐색 화면의 활성 디렉터리 변경 알림 |

테스트 전용 의존성은 go.uber.org/goleak(고루틴 정리 검사), google/go-cmp(복잡한 값 비교), creack/pty(실행 파일의 터미널 통합 검사)다. 일반 실행 파일에서는 이 패키지들을 import하지 않는다.

CLI 분기는 flag, 병렬 도구 합류는 sync.WaitGroup, 재시도 대기는 time.Timer를 사용한다. 검색은 strings·unicode로 부분일치를 우선하고 결과가 없을 때 문자 순서를 유지하는 축약 일치를 찾는다. 메타데이터 재사용은 연결 준비가 소유한 마지막 GET 응답으로 한정한다. 이 사용처에는 별도 CLI·동시성·캐시 프레임워크를 두지 않는다.

Go 빌드만으로 MECA 실행 파일을 만든다. 제품 빌드·실행에 Docker·Bubblewrap·별도 샌드박스 설치를 요구하지 않는다. 빌드·설치·삭제 명령은 [README](../README.md#직접-빌드하기)를 따른다. 소스 설치 스크립트는 Go가 없을 때만 공식 SDK를 체크섬 검증 후 임시 설치하며 작업 후 정리한다. Go 최소 버전을 바꿀 때 `scripts/install/go.sha256`의 네 플랫폼 검증값도 공식 배포 정보와 함께 갱신한다. macOS·Linux 릴리스는 [릴리스 가이드](releases.md)와 GoReleaser 설정을 사용하며 CI·자동 게시 파이프라인은 구성하지 않는다.

### 개발 도구

검증·배포 도구는 [devtools/go.mod](../devtools/go.mod)의 별도 모듈에 버전을 고정한다. 제품의 go.mod·go.sum과 의존성 해석을 분리하며 MECA 바이너리에 포함하지 않는다. 도구 실행에는 Go 1.27.3 이상이 필요하다. 다음 명령은 저장소 루트에서 실행한다. 최초 실행 시 Go가 필요한 도구를 내려받고 빌드 캐시에 저장하므로 전역 설치는 필요 없다.

| 도구 | 역할 | 실행 |
| --- | --- | --- |
| Staticcheck | Go 코드의 오류·API 오용·불필요한 연산 검사 | `go tool -modfile=devtools/go.mod staticcheck ./...` |
| govulncheck | 알려진 취약점과 실제 호출 경로 검사 | `go tool -modfile=devtools/go.mod govulncheck ./...` |
| deadcode | 실행 진입점에서 도달하지 않는 함수 탐색 | `go tool -modfile=devtools/go.mod deadcode ./cmd/meca` |
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

GoReleaser는 [.goreleaser.yaml](../.goreleaser.yaml)의 macOS·Linux 아카이브를 생성한다. `check`와 게시 없는 `release --snapshot --clean`으로 먼저 검증하고 사용자 요청에 따라 게시한다. nFPM은 Linux 설치 패키지가 필요할 때 사용하는 도구로 등록되어 있다. 현재 배포 형식은 tar.gz이며 제품 실행·일반 빌드·테스트에는 이 도구들이 필요하지 않다.

버전 갱신은 `go get -modfile=devtools/go.mod -tool <패키지>@<버전>`으로 수행한다. 이후 `go -C devtools mod tidy`와 `go -C devtools mod verify`를 실행하고 두 개발 도구 파일의 변경을 검토한다. 도구 요구 버전이 제품의 의존성이나 최소 Go 버전을 바꾸지 않도록 유지한다.

## 변경 검증

다음은 실행 지침이며 이 문서 편집 시의 통과 기록이 아니다. 작업 범위에 맞는 집중 검사 후 공유 계약 변경은 전체 검사로 넓힌다.

```sh
gofmt -l cmd internal
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
```

순수 상태·파서·승인·저장·Provider 변환은 작은 fixture와 httptest로 검증한다. 파일 검사는 t.TempDir, TUI는 저장소의 가상 터미널과 실제 키 입력을 사용한다. 모델 응답을 고정한 검사는 실제 모델의 완수율이나 추론 품질 검증과 구분한다.

에이전트 실행·HTTP 생성·TUI 테스트는 패키지 종료 시 goleak으로 남은 고루틴을 검사한다. 구조체 복원과 대화 투영은 go-cmp로 필드 차이를 확인한다. PTY 통합 검사는 임시 HOME·프로젝트에서 실제 실행 파일을 시작해 터미널 크기 변경, Ctrl+C·!quit 종료, 터미널 설정 복원과 DB 잠금 해제를 확인한다. PTY의 소유 프로세스가 종료 후 상태를 검사하고, 실패·시간 초과에는 해당 프로세스 그룹과 읽기 작업을 정리한다. `go test -short`에서는 실행 파일을 빌드하는 통합 검사를 생략한다.

```sh
go test ./cmd/meca -run TestTerminalExecutable -v
go test ./internal/tui/session -run TestFileWatch -v
```

문서 변경은 실제 코드·기본값·링크·diff를 확인한다. 문서 검증은 변경된 설명과 참조 범위에 맞춰 수행한다. 코드의 헤더·주석·테스트는 해당 파일의 책임과 실제 동작을 설명하도록 유지한다.

## 플랫폼 실행 검증

Linux·WSL·macOS에서 호스트 셸의 작업 디렉터리·환경·프로세스 정리와 필수 승인 거절을 검증한다. 임시 HOME에서 캐시 생성, 기존 하드링크, 호스트 DNS 설정 파일, 자식 프로세스·로컬 HTTP, 시작 실패·비정상 종료·취소 후 부분 출력을 확인한다.

```sh
go test ./internal/tools/local/... ./internal/agent/run ./internal/commands/headless
```

실제 uv 설치 경로 검사는 다음의 opt-in 진입점을 사용한다. MECA_TEST_UV_PYTHON_ROOT로 관리 Python 트리를 지정할 수 있다. 이는 시험 fixture의 입력이며 제품의 언어별 실행 분기가 아니다. fixture는 사용자 캐시 쓰기 예외나 링크 모드 변경 없이 임시 HOME을 사용한다.

```sh
MECA_TEST_UV_BINARY=/path/to/uv go test ./internal/tools/local/process -run TestUVPackageProject -v
```

실제 WSL에서는 일반 터미널과 execute_command에서 같은 도메인 조회·HTTPS 다운로드·패키지 설치를 대조한다. DNS 설정 파일 읽기와 숫자 IP의 로컬 HTTP 통과만으로 외부 DNS 성공을 판단하지 않는다. Linux Secret Service·실제 Provider·터미널 키 전달도 수행한 환경과 결과를 구분해 기록한다. 교차 빌드는 네이티브 실행 검증을 대신하지 않는다. [실행 경계](execution-boundary.md)의 호스트 권한 계약을 함께 확인한다.

## 성능과 장시간 수명

입력은 grapheme 경계를 보존하고 붙여넣기를 묶어 처리한다. 변경되지 않은 source/width 배치와 Markdown 투영을 재사용하고 연속 화면 갱신을 합치되 순서 있는 이벤트는 버리지 않는다. 요청 worker가 표시용 최신 스트림 mailbox를 소유하고 재시도·완료·승인·사용량 경계에서 이전 mailbox를 분리한다. 소비 후 본문 참조를 해제하며 별도 전송 goroutine을 만들지 않는다. 기록·파일 열·선택·뷰포트 캐시를 해제할 때 참조도 정리한다.

디렉터리 조회와 파일 미리보기는 각각 실행 1개·최신 대기 요청 1개를 가진다. 취소는 대기 요청을 제거하지만 실행 슬롯은 실제 I/O 완료 알림에서 해제한다. 이벤트 루프가 최신 요청만 시작하며 이전 revision의 결과는 적용하지 않는다. 종료는 대기를 비우고 실제 worker를 합류한 뒤 슬롯 참조를 해제한다.

파일 변경 감시는 활성 디렉터리 하나에 연결한다. 플랫폼별 등록 I/O도 worker에서 수행하며 실행 1개·최신 대기 1개로 제한한다. 이벤트 루프가 감시 채널과 150ms 단발 타이머를 소비하고, 등록 직후와 알림 유실 시 재조회한다. 경로 전환·취소·화면 닫기에서 감시와 타이머를 정리하며 늦게 도착한 감시 객체도 닫는다. 갱신은 기존 파일 읽기 슬롯을 사용하고 새 폴링 루프를 만들지 않는다.

로컬 도구 카탈로그는 초기화 후 변경하지 않는 내부 정의 하나를 소유한다. 인자 검사는 이를 재사용하며 외부 Definitions 반환값은 중첩 Fields·Values까지 복사해 변경이 내부 검증으로 전파되지 않게 한다.

응답별 마스킹은 누적 스냅샷에서 새 구간만 검사하고 비밀키 접두사·미완성 UTF-8을 보류한다. 응답과 Thinking은 상태를 공유하지 않으며 재시도마다 초기화한다. 파일 미리보기와 대화 본문은 희소 위치 인덱스로 가시 구간만 구체화한다. 일반 본문은 256개 표시 줄마다 원문 위치를 기록해 긴 한 줄도 중간부터 재개한다. Markdown은 마지막 두 최상위 블록을 가변 구간으로 두고 완성된 앞부분의 파싱·줄 수를 재사용한다. 참조 정의 가능성이 있는 `]:` 포함 문서는 전체 파싱을 유지한다. 화면 밖 과거 응답은 파싱 결과·span을 해제하고 위치 인덱스만 남기며 현재 응답은 증분 처리 수명 동안 보유한다. 경로 표시는 선택 revision·폭·언어·마스킹 revision으로 캐시하고 실제 표시 예산을 채우면 변환을 끝낸다. Compaction은 시간 순서 목록과 우선순위별 제거 대기열로 사용자 지시·결과·실패의 보존 순서를 유지한다.

SQLite의 큰 JSON 값은 `payloads`에서 해시·원본 길이·압축 형식을 소유한다. 작은 레코드는 별도 참조 목록을 갖고 내용에 임의의 표식을 삽입하지 않는다. 한 연결을 쓰므로 조회 페이지를 닫은 뒤 payload를 복원한다. 기록과 payload는 같은 트랜잭션으로 커밋한다. `run.JournalRecord` 그룹은 같은 확정 시점의 메시지·이벤트·체크포인트를 원자적으로 저장한다. 실행 전 시작 의도는 효과 발생 전에, 실제 실행 결과·native 응답·변경 후 체크포인트는 취소와 무관하게 5초 저장 문맥에서 함께 확정한다. 실행 시작과 결과의 두 경계를 합치지 않는다. 종류·본문 길이·요약 입력 투영을 저장 시 기록하며 원본 읽기·export·취소 후 복원 계약을 함께 검사한다.

압축 점유량 조회는 최근 실측 샘플·최신 요약 revision의 경계·이후 추가된 메시지 크기·현재 timeline을 한 SQL 문과 같은 읽기 스냅샷에서 계산한다. 원문 본문을 읽거나 추가 영속 카운터·캐시·인덱스를 만들지 않는다.

컨텍스트 투영 캐시는 요청 실행이 소유하고 변경되지 않은 불변 메시지만 재사용한다. native 도구 결과 짝이 완성되거나 메시지·모델·비밀키 revision이 바뀌면 다시 투영한다. Compaction이 이력을 교체하면 캐시를 해제한다. 전체 미압축 대화와 Provider 전송 본문은 유지하며 API 요청 직렬화·전송을 위한 실제 바이트 복사는 필요하다.

대화 선택은 디렉터리의 DB 파일이 식별 근거다. 최초 조회는 private 메모리 SQLite 연결 하나에 파일을 읽기 전용으로 하나씩 attach/detach하고 요약 배열 하나와 정렬 시각만 보유한다. 별도 레지스트리·sidecar·상시 worker를 만들지 않으며 정상·오류·취소 시 연결을 닫는다. 페이지 이동은 선택 화면이 소유하는 스냅샷을 사용한다. 전체 정렬을 위해 최초 조회는 파일 수에 비례한다.

추가 성능·수명 검사는 다음을 사용한다. 수치는 실행한 환경·입력·측정 종류를 함께 기록하고 힙 할당량을 RSS와 혼동하지 않는다.

```sh
go test ./internal/agent/context -run '^$' -bench BenchmarkProjection -benchmem
go test ./internal/tui/markdown -run '^$' -bench BenchmarkStreamTail -benchmem
go test ./internal/tools/local -run '^$' -bench BenchmarkParallelPlanning -benchmem
go test ./internal/tools/local -run '^$' -bench BenchmarkCatalog -benchmem
go test ./internal/tui/session -run '^$' -bench BenchmarkStreamDelivery -benchmem
go test ./internal/storage/sqlite -run '^$' -bench BenchmarkOutcomeCommit -benchtime=100x -benchmem
go test ./internal/storage/sqlite -run '^$' -bench BenchmarkMemoryWatermark -benchmem
go test ./internal/storage/sqlite -run '^$' -bench 'Benchmark(Conversations|Journal)' -benchmem
go test ./internal/tui/viewport -run TestDisplayResources -v
go test ./internal/tui/markdown -run '^$' -fuzz '^FuzzStream$' -fuzztime=10s
```

```sh
go test ./internal/config/credentials ./internal/tui/viewport ./internal/tui/view ./internal/tui/browser ./internal/storage/sqlite ./internal/tools/local/process -run '^$' -bench . -benchmem
```

벤치마크에는 Go·OS·CPU·데이터 크기·창 너비·명령과 소스 식별값을 함께 기록한다. 합성 줄 배치 수치를 모델 TPS나 전체 작업 시간으로 해석하지 않는다. 장시간 heap/RSS·취소 후 자원·실제 스트림의 프로파일링은 별도 확인한다.

## 기본 루프의 책임

[Pi의 기본 루프](https://github.com/earendil-works/pi/blob/main/packages/agent/src/agent-loop.ts)의 단일 에이전트·native 도구·실제 대화 중심 문맥 구성을 참고한다. MECA의 Go 단일 바이너리·호스트 실행·파일 검사·필수 승인·OS 자격 증명·세션 SQLite는 현재 제품 계약이다. 동기적인 부가 평가·별도 실행 원장·명령 재호출 추정 차단 없이 모델의 정상 응답으로 종료한다.

원본 출력은 실행 이벤트가 소유하고, 작은 체크포인트·TUI·모델 문맥은 각각 필요한 투영만 사용한다. 비대화형 실행과 TUI는 agent/session을 공유하며 테스트가 별도 실행 루프를 만들지 않는다. cmd/meca의 실행 파일 통합 검사는 임시 HOME·프로젝트와 모의 Provider를 사용하며 실모델 E2E와 구분한다.
