# 개발 가이드

[← README](../README.md) · [제품 명세](../SYSTEM.md) · [작업 규칙](../AGENTS.md)

## 책임과 의존성

단일 Go 모듈과 cmd/v 합성 루트를 사용한다. 기능은 internal 아래에 두고 실제 소유 패키지를 직접 참조한다. 소비자 소유의 작은 I/O 계약과 함수·struct를 우선하며 패키지는 한 기능, 파일은 그 안의 한 책임을 담당한다.

| 위치 | 책임 |
| --- | --- |
| cmd/v | CLI 분기·어댑터 조립·프로세스와 터미널 수명 |
| agent/model | 모델 메시지·스트림·출력 예산 계약 |
| agent/session·run | 요청 준비·실행 순서·승인 조율·공유 예산·요청별 실측 통계 |
| agent/context·compact | 모델 문맥과 단일 호출 요약 |
| commands/headless | 같은 요청 경로의 text·JSONL 출력 어댑터 |
| agent/tool·work | 실행 사실·가벼운 파일 관측·종료 원인 |
| agent/prompt·inference | 작업 지침·역할별 추론 설정 |
| commands/startup·install·settings | 시작 경로·사용자 설치와 삭제·Thinking·언어 설정 저장 |
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
| tui/session | 이벤트 루프·활성 화면·worker 수명 |
| tui/locale | 불변 번역 목록·설정 우선 언어 선택·시스템 로캘 해석 |
| tui/modelmenu·transcript·view | 모델 표시값·메시지/도구 투영·화면 렌더링 |
| tui/input·textlayout·selection | 편집 revision·grapheme 배치·원문에 연결된 선택 |
| tui/startup·browser·viewport·markdown | 대화 선택·파일 탐색·출력 위치·응답 Markdown 표시 |

TUI는 저장·연결 설정·요청 준비를 직접 소유하지 않는다. worker는 결과 이벤트를 반환하고 화면 상태는 이벤트 루프에서만 바꾼다. 연결 클라이언트·HTTP body·goroutine·timer·DB·자식 프로세스에는 취소와 정리 소유자가 있어야 한다.

실행 사실과 파일 관측은 agent/tool, 실행 순서는 agent/run, 문맥 조립은 agent/context가 소유한다. Compaction은 사실을 소비해 요약하며 별도 실행 원장을 만들지 않는다. SQLite는 저장 기능 전체가 아니라 저장 계약의 구현 수단이다.

## 의존성과 배포

정확한 도구 체인과 버전은 [go.mod](../go.mod), 재배포 라이선스는 [외부 소프트웨어 고지](../THIRD_PARTY_NOTICES.md)를 따른다.

| 의존성 | 용도 |
| --- | --- |
| tcell/v3·uniseg | 터미널·입력·마우스·문자 경계와 셀 폭 |
| goldmark/v2 | 응답 Markdown AST를 TUI span으로 변환 |
| go-toml/v2 | TOML 파싱·직렬화 |
| go-keyring | Keychain·Secret Service |
| modernc.org/sqlite | CGO 없는 SQLite |
| x/sys·x/term | OS 파일 어댑터와 터미널 확인 |
| mvdan.cc/sh/v3 | 실행 없는 셸 AST 분석과 정적 단어 확장 |

Go 빌드만으로 V 실행 파일을 만든다. 제품 빌드·실행에 Docker·Bubblewrap·별도 샌드박스 설치를 요구하지 않는다. 빌드·설치·삭제 명령은 [README](../README.md#직접-빌드하기)를 따른다. CI·자동 릴리스 파이프라인은 구성하지 않는다.

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

문서 변경은 실제 코드·기본값·링크·diff를 확인한다. 문서만 정리하기 위해 workspace 프로젝트를 생성하거나 설치·삭제·모델 시험을 실행하지 않는다. 변경한 코드의 헤더·크기·테스트 규칙은 AGENTS.md를 따른다.

## 플랫폼 실행 검증

Linux·WSL·macOS에서 호스트 셸의 작업 디렉터리·환경·프로세스 정리와 필수 승인 거절을 검증한다. 임시 HOME에서 캐시 생성, 기존 하드링크, 호스트 DNS 설정 파일, 자식 프로세스·로컬 HTTP, 시작 실패·비정상 종료·취소 후 부분 출력을 확인한다.

```sh
go test ./internal/tools/local/... ./internal/agent/run ./internal/commands/headless
```

실제 uv 설치 경로 검사는 다음의 opt-in 진입점을 사용한다. V_TEST_UV_PYTHON_ROOT로 관리 Python 트리를 지정할 수 있다. 이는 시험 fixture의 입력이며 제품의 언어별 실행 분기가 아니다. fixture는 사용자 캐시 쓰기 예외나 링크 모드 변경 없이 임시 HOME을 사용한다.

```sh
V_TEST_UV_BINARY=/path/to/uv go test ./internal/tools/local/process -run TestUVPackageProject -v
```

실제 WSL에서는 일반 터미널과 execute_command에서 같은 도메인 조회·HTTPS 다운로드·패키지 설치를 대조한다. DNS 설정 파일 읽기와 숫자 IP의 로컬 HTTP 통과만으로 외부 DNS 성공을 판단하지 않는다. Linux Secret Service·실제 Provider·터미널 키 전달도 수행한 환경과 결과를 구분해 기록한다. 교차 빌드는 네이티브 실행 검증을 대신하지 않는다. [실행 경계](execution-boundary.md)의 호스트 권한 계약을 함께 확인한다.

## 성능과 장시간 수명

입력은 grapheme 경계를 보존하고 붙여넣기를 묶어 처리한다. 변경되지 않은 source/width 배치와 Markdown 투영을 재사용하고 연속 화면 갱신을 합치되 순서 있는 이벤트는 버리지 않는다. 기록·파일 열·선택·뷰포트 캐시를 해제할 때 참조도 정리한다.

```sh
go test ./internal/tui/viewport -run '^$' -bench . -benchmem
```

벤치마크에는 Go·OS·CPU·데이터 크기·창 너비·명령과 소스 식별값을 함께 기록한다. 합성 줄 배치 수치를 모델 TPS나 전체 작업 시간으로 해석하지 않는다. 장시간 heap/RSS·취소 후 자원·실제 스트림의 프로파일링은 별도 확인한다.

## 기본 루프의 책임

[Pi의 기본 루프](https://github.com/earendil-works/pi/blob/main/packages/agent/src/agent-loop.ts)의 단일 에이전트·native 도구·실제 대화 중심 문맥 구성을 참고한다. V의 Go 단일 바이너리·호스트 실행·파일 검사·필수 승인·OS 자격 증명·세션 SQLite는 현재 제품 계약이다. 동기적인 부가 평가·별도 실행 원장·명령 재호출 추정 차단 없이 모델의 정상 응답으로 종료한다.

원본 출력은 실행 이벤트가 소유하고, 작은 체크포인트·TUI·모델 문맥은 각각 필요한 투영만 사용한다. 비대화형 실행과 TUI는 agent/session을 공유하며 테스트가 별도 실행 루프를 만들지 않는다. cmd/v의 실행 파일 통합 검사는 임시 HOME·프로젝트와 모의 Provider를 사용하며 실모델 E2E와 구분한다.
