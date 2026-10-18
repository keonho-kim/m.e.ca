# 개발 가이드

[← README](../README.md) · [제품 명세](../SYSTEM.md) · [작업 규칙](../AGENTS.md)

## 책임과 의존성

단일 Go 모듈과 cmd/v 합성 루트를 사용한다. 기능은 internal 아래에 두고 실제 소유 패키지를 직접 참조한다. 소비자 소유의 작은 I/O 계약과 함수·struct를 우선하며 패키지는 한 기능, 파일은 그 안의 한 책임을 담당한다.

| 위치 | 책임 |
| --- | --- |
| cmd/v | CLI 분기·어댑터 조립·프로세스와 터미널 수명 |
| agent/model·tool | 모델 메시지·스트림·호출·승인·실제 실행 사실 |
| agent/session·run·work | 요청 준비·실행 순서·공유 예산·중단 원인 |
| agent/context·compact | 모델 문맥과 단일 호출 요약 |
| agent/progress·completion | 상태 순환 감지와 현재 파일·명령 결과 |
| agent/grounding·monitor | 정확한 인자에 연결된 reason과 실행 worker의 순서 있는 관측 |
| agent/prompt·inference·response | 작업 지침·역할별 추론 설정·모델 응답 해석 |
| commands/startup·install·settings | 시작 경로·사용자 설치와 삭제·정책/Thinking 저장 |
| config/bootstrap·prompts·statefs·credentials | 초기화·Markdown 파일·private 상태 수명·OS 자격 증명 |
| provider/connection·identity·profile·selection | 연결 값·원본 식별·검토된 모델 제어·선택 검증 |
| provider/discovery·diagnostic·setup | 메타데이터·실제 생성 진단·연결 저장과 소유권 이전 |
| provider/protocol·transport·httpclient | wire 변환·HTTP/스트림/재시도·생성 요청 조율 |
| policy·policy/shell | 접근·승인과 직접 셸 변경 분류 |
| tools/local/filesystem | 실제 경로·루트 identity·공통 파일 경계 검사 |
| tools/local/file·textedit·textformat·diff | 파일 I/O·정확한 부분 수정·바이트 형식·변경 미리보기 |
| tools/local/process·sandbox | 프로세스·출력·취소·OS 격리 |
| storage/session·history·recovery | 세션 식별·순서 있는 이력·중단 복구의 저장 계약 |
| storage/sqlite | DB 파일·SQL·트랜잭션·잠금 어댑터 |
| tui/session | 이벤트 루프·활성 화면·worker 수명 |
| tui/modelmenu·transcript·view | 모델 표시값·메시지/도구 투영·화면 렌더링 |
| tui/input·textlayout·selection | 편집 revision·grapheme 배치·원문에 연결된 선택 |
| tui/startup·browser·viewport·markdown | 대화 선택·파일 탐색·출력 위치·응답 Markdown 표시 |

TUI는 저장·연결 설정·요청 준비를 직접 소유하지 않는다. worker는 결과 이벤트를 반환하고 화면 상태는 이벤트 루프에서만 바꾼다. 연결 클라이언트·HTTP body·goroutine·timer·DB·자식 프로세스에는 취소와 정리 소유자가 있어야 한다.

실행 사실은 agent/tool, 현재 상태는 agent/completion, 문맥 조립은 agent/context가 소유한다. Compaction은 사실을 소비해 요약하며 별도 실행 원장을 만들지 않는다. SQLite는 저장 기능 전체가 아니라 저장 계약의 구현 수단이다.

## 의존성과 배포

정확한 도구 체인과 버전은 [go.mod](../go.mod), 재배포 라이선스는 [외부 소프트웨어 고지](../THIRD_PARTY_NOTICES.md)를 따른다.

| 의존성 | 용도 |
| --- | --- |
| tcell/v3·uniseg | 터미널·입력·마우스·문자 경계와 셀 폭 |
| goldmark/v2 | 응답 Markdown AST를 TUI span으로 변환 |
| go-toml/v2 | TOML 파싱·직렬화 |
| go-keyring | Keychain·Secret Service |
| modernc.org/sqlite | CGO 없는 SQLite |
| x/sys·x/term | OS 격리와 터미널 확인 |
| mvdan.cc/sh/v3/syntax | POSIX 명령 분석. 실행은 OS 셸 사용 |

Go 빌드만으로 V와 Linux 내부 실행기를 만든다. 제품 빌드·실행에 Docker·Bubblewrap·별도 샌드박스 설치를 요구하지 않는다. 빌드·설치·삭제 명령은 [README](../README.md#직접-빌드하기)를 따른다. CI·자동 릴리스 파이프라인은 구성하지 않는다.

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

순수 상태·파서·승인·저장·Provider 변환은 작은 fixture와 httptest로 검증한다. 파일 경계는 t.TempDir, TUI는 저장소의 가상 터미널과 실제 키 입력을 사용한다. 모델 응답을 고정한 검사는 실제 모델의 완수율이나 추론 품질 검증과 구분한다.

문서 변경은 실제 코드·기본값·링크·diff를 확인한다. 문서만 정리하기 위해 workspace 프로젝트를 생성하거나 설치·삭제·모델 시험을 실행하지 않는다. 변경한 코드의 헤더·크기·테스트 규칙은 AGENTS.md를 따른다.

## 플랫폼 실행 검증

모든 셸에 공통 파일 경계를 적용한다. 아래 검사는 실제 OS 강제와 준비 실패 계약을 확인하며, 파일 경계와 호스트 네트워크의 허용을 구분한다.

Linux 어댑터는 Landlock ABI 3 이상, seccomp, user/mount/PID/IPC/UTS namespace와 private proc 마운트를 요구한다. 네이티브 지원이 없으면 일반 검사는 사유와 함께 생략할 수 있다. 강제 실행과 명시적인 호스트 거절 경로는 각각 다음으로 검사한다.

```sh
V_SANDBOX_REQUIRED=1 go test ./internal/tools/local/process/...
V_SANDBOX_DENIED=1 go test ./internal/tools/local/process/...
```

mountinfo와 하위 마운트 처리, 외부 쓰기·symlink·V 상태 접근 차단, 사용자 설치 도구, 실행 파일 교체와 취소 후 자손 정리를 확인한다. 격리 준비 실패는 명령을 실행하지 않고 scope_unavailable을 반환해야 한다. WSL 커널 버전 문자열로 기능을 추정하지 않는다.

macOS는 선택된 개발 도구·인증서·실행 자원을 읽기 전용으로 검사하고, 프로젝트 외 파일과 private 상태 차단·프로세스 정리를 네이티브 환경에서 확인한다. 교차 빌드는 OS 보안·TTY·자격 증명 서비스의 실기 검증을 대신하지 않는다.

실제 uv 경로 회귀 검사는 아래 기존 진입점을 사용한다. 관리 Python 트리를 쓰면 V_TEST_UV_PYTHON_ROOT를 함께 지정할 수 있다. 이는 시험 fixture의 입력이며 제품의 언어별 실행 분기가 아니다.

```sh
V_TEST_UV_BINARY=/path/to/uv go test ./internal/tools/local/process -run TestUVPackageProject -v
```

배포판별 AppArmor·namespace 정책, WSL 장비, Linux Secret Service와 실제 Provider/모델 조합은 수행한 환경과 결과를 별도로 기록한다. [공통 실행 경계](execution-boundary.md)의 파일 보호와 호스트 네트워크 계약을 함께 확인한다.

## 성능과 장시간 수명

입력은 grapheme 경계를 보존하고 붙여넣기를 묶어 처리한다. 변경되지 않은 source/width 배치와 Markdown 투영을 재사용하고 연속 화면 갱신을 합치되 순서 있는 이벤트는 버리지 않는다. 기록·파일 열·선택·뷰포트 캐시를 해제할 때 참조도 정리한다.

```sh
go test ./internal/tui/viewport -run '^$' -bench . -benchmem
```

벤치마크에는 Go·OS·CPU·데이터 크기·창 너비·명령과 소스 식별값을 함께 기록한다. 합성 줄 배치 수치를 모델 TPS나 전체 작업 시간으로 해석하지 않는다. 장시간 heap/RSS·취소 후 자원·실제 스트림의 프로파일링은 별도 확인한다.
