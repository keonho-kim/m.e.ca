<div align="center">

# V

**원하는 모델 서버와 연결하는 터미널 코딩 에이전트.**

Go · Linux / macOS · amd64 / arm64

[빌드](#직접-빌드하기) · [사용법](#사용법) · [설정](#설정) · [문서](#문서)

</div>

V는 터미널에서 코드를 읽고, 수정하고, 검증하는 에이전트입니다. 실행 파일은 `v` 하나이며, 작업 내역과 파일 변경을 터미널에서 확인할 수 있습니다.

- **모델 선택** — Ollama, vLLM, SGLang, LM Studio, LiteLLM, OpenAI 호환 서버 연결.
- **변경 검토** — 줄 단위 diff 표시와 위험 작업의 필수 사용자 승인.
- **직접 작업과 종료** — 메인 에이전트가 파일·셸 도구로 직접 수행하고 실제 결과를 확인. 예산·오류 중단 시에도 수행 결과와 미완료 사항을 보고.
- **작업 이어가기** — 로컬 세션 저장과 대화 압축, `v resume`으로 이전 작업 재개.
- **터미널 UI** — 스트리밍 응답, `!dir` 파일 참조, `!` 명령 검색, 마우스 스크롤과 문맥별 키 안내.

## 직접 빌드하기

**Git과 Go 1.27.2**가 필요합니다. Linux와 macOS 모두 Go 명령만으로 빌드합니다.

```sh
git clone https://github.com/keonho-kim/v-code.git
cd v-code
```

이후 명령은 저장소 루트에서 실행합니다.

### 설치 — Linux / macOS

```sh
./install.sh
```

현재 OS·CPU에 맞춰 빌드하고 `~/.local/bin/v`에 설치합니다. `~/.v`와 기본 설정·프롬프트를 생성하고, 필요한 PATH 등록을 zsh·bash·sh 설정에 추가합니다. 기존 설정은 유지합니다. 새 터미널에서 `v`를 사용할 수 있으며, 현재 터미널에는 설치 완료 메시지의 명령으로 반영합니다.

### 빌드만 하기

```sh
go build -trimpath -o bin/v ./cmd/v
```

`go build`는 컴파일만 합니다. 초기화·설치·PATH 등록은 `./install.sh` 또는 빌드한 바이너리의 `./bin/v install`로 실행합니다.

### 실행

```sh
v                   # 현재 프로젝트에서 시작
v /path/to/project  # 지정한 프로젝트에서 시작
v resume            # 이전 대화를 선택해 이어가기
```

### 터미널 없이 실행하기

```sh
v --print "README를 검토해줘"
v /path/to/project --mode json "요청한 파일을 만들어줘"
git diff | v --print "이 변경을 검토해줘"
v --session SESSION_ID --print "이어서 진행해줘"
v --session SESSION_ID --export > session.jsonl
v --diagnose
```

먼저 `!model`에서 기본 연결을 저장합니다. print는 최종 답변, JSON 모드는 순서 있는 이벤트를 stdout에 기록하며 같은 에이전트 루프·세션·파일 보호 정책을 사용합니다. 비대화형 실행의 필수 승인 요청은 명시적으로 차단합니다. 세션 ID는 첫 JSON `session` 이벤트의 `session_id` 또는 `~/.v/sessions/<project-key>/`의 DB 파일명으로 확인합니다. Export는 원본 세션 이력을 JSONL로 출력합니다.

### 완전히 삭제하기

```sh
v uninstall
```

설치된 `~/.local/bin/v`, V가 추가한 PATH 블록, `~/.v` 전체(설정·프롬프트·대화·로그)와 V가 저장한 OS 인증정보를 삭제합니다. 실행 중인 V를 모두 종료한 뒤 수행합니다. 기존 셸 설정과 사용자 프로젝트는 유지하며, 삭제에 실패하면 원인을 반환합니다. 소스 저장소와 공용 Go·패키지 관리자 캐시는 삭제하지 않습니다.

## 사용법

파일 조회·검색은 `execute_command`의 `cat`, `sed -n`, `rg` 등을 사용하고, 삭제는 같은 셸에서 `rm` 등으로 수행하며, 소스 생성·본문 수정·이동에는 diff와 원본 버전 검사를 제공하는 전용 파일 도구를 권장합니다. 셸의 생성·수정·이동은 호스트 사용자 권한과 위험 작업의 필수 승인 정책을 따릅니다.

모델 서버는 native tool calling을 지원해야 합니다. vLLM 등에서는 모델에 맞는 Tool Parser 설정도 필요합니다. [서버 설정 안내 →](docs/model-connections.md#native-tool-calling)

1. `!model`에서 Provider·주소·모델을 선택합니다. 인증이 필요한 서버는 API Key도 입력합니다.
2. 자연어로 작업을 요청합니다. 참조할 파일은 `!dir`에서 선택할 수 있습니다.
3. 변경을 diff로 확인합니다. 일반 작업은 자동 승인하며, 위험 작업은 실행 전에 별도 승인을 받습니다. 분석만 원하면 파일을 바꾸지 말고 분석해 달라고 요청합니다.

| 명령 | 기능 |
| --- | --- |
| `!model` | 모델 연결, 추론 켜기·끄기와 지원되는 추론 강도 선택 |
| `!language` | 화면 언어 선택 · 시스템 기본값 / 한국어 / English |
| `!dir [path]` | 파일·디렉터리 탐색 |
| `!new` | 같은 프로젝트에서 새 세션 시작 · 전역 모델 연결 유지 |
| `!quit` | 현재 작업을 취소·정리하고 종료 |

`!`를 입력하면 명령 후보가 나타납니다. 파일 탐색·선택·미리보기는 `!dir`을 사용합니다. `↑`·`↓`로 이동하고 `Enter`로 선택합니다. 선택한 텍스트가 없을 때 `Ctrl+C`로도 종료할 수 있으며, 모델·활성 추론은 하단 키맵 바로 윗줄에, 주요 키는 맨 아랫줄에 고정됩니다.

일반 작업은 **자동 승인**으로 고정되며 추론은 기본적으로 꺼져 있습니다. 삭제·이동·파괴적인 Git 작업과 프로젝트 밖의 직접 변경은 별도 승인을 받으며, 보호 경로 변경과 권한 상승은 차단합니다. 셸은 기존 HOME·PATH·DNS·개발 도구·캐시를 호스트 사용자 권한으로 사용합니다. 프로젝트는 기본 작업 디렉터리이며 파일 접근을 격리하지 않습니다. 실행 범위와 승인 정책은 [실행 경계](docs/execution-boundary.md)를 참고하세요. API Key는 macOS Keychain 또는 Linux Secret Service에 저장하며 마지막 모델 연결은 새 세션에서 재사용합니다.

입력창의 `↑↓`는 줄 이동, `Ctrl/Cmd+↑↓`는 이전 입력 탐색입니다. 작업 중 Enter는 다음 판단에 방향 수정을 전달하고 Tab은 후속 요청을 예약합니다. 도구 결과는 기본으로 펼쳐 보이며 Ctrl+O로 전체를 접거나 펼칩니다. 접으면 도구·대상·실행 결과 요약을 남깁니다. 드래그 선택 후 `Ctrl/Cmd+C`로 복사하고, 입력창 선택은 Backspace로 삭제합니다. `F1`에서 현재 화면의 전체 키 안내를 확인합니다.

긴 대화는 오른쪽 스크롤바를 클릭·드래그해 탐색합니다. 위로 스크롤하면 읽는 위치를 유지하며, 하단으로 돌아오거나 새 요청을 보내면 새 출력을 따라갑니다. 압축 전후에도 이 선택을 유지합니다.

작업이 끝나면 마지막 응답 아래에 소요 시간·이번 작업의 실제 INPUT/OUTPUT·평균 TPS를 흐린 글씨로 가운데 표시합니다. 이 요약은 대화를 다시 열어도 보존됩니다.

화면 언어는 `!language`에서 선택하거나 `~/.v/config.toml`의 `language`에 `auto`·`ko`·`en`을 지정합니다. 기본값 `auto`는 시스템 로캘을 따릅니다. 메뉴 선택은 저장 후 즉시 적용하며 직접 파일을 수정하면 재시작 후 반영합니다. 설정값·모델 ID·명령·출력 원문과 사용자·모델의 대화 본문은 유지합니다. [언어 선택 기준](docs/usage.md#화면-언어)을 참고하세요.

[자세한 사용법과 키맵 →](docs/usage.md) · [모델 연결·추론 설정 →](docs/model-connections.md)

## 설정

첫 실행에서 `~/.v`와 기본 파일을 자동 생성합니다. 기존 파일은 보존합니다.

```text
~/.v/
├── config.toml     # 전역 설정과 역할별 추론 인자
├── prompts/        # 사용자 수정 가능한 Markdown 프롬프트
├── sessions/       # 세션별 SQLite 대화 기록
└── logs/
```

추론 설정은 Provider가 아닌 **역할별**로 적용합니다. 기본 temperature는 메인 에이전트 `1.0`, 요약·연결 진단 `0.2`입니다. 출력 토큰 기본 상한은 메인 에이전트가 유효 context length의 `75%`, Compaction이 `8,192`, 연결 진단이 `12,800`입니다.

```toml
language = "auto" # auto: 시스템 기본값, ko: 한국어, en: English

[agent]
temperature = 1.0
max_output_tokens = 0 # 유효 context length의 75% 자동 계산

[compaction]
temperature = 0.2
max_output_tokens = 8192
reserve_tokens = 16384
keep_recent_tokens = 20000

[diagnostics]
temperature = 0.2
max_output_tokens = 12800
```

[전체 역할별 설정·프롬프트·Compaction →](docs/configuration.md)

세션 저장 형식은 schema 19입니다. 미지원 DB는 원본을 보존하고 오류를 안내합니다. 사용자 프롬프트는 자동으로 덮어쓰지 않습니다. [설정과 프롬프트](docs/configuration.md)를 참고하세요.

## 개발

단일 Go 모듈이며 진입점은 [`cmd/v`](cmd/v)입니다. 주요 의존성은 mvdan/sh(셸 구문 분석), goldmark, tcell, go-toml, modernc SQLite, go-keyring, uniseg, x/sys, x/term입니다. 버전은 [`go.mod`](go.mod)에 고정하며, SQLite는 CGO 없이 빌드합니다.

```sh
gofmt -l cmd internal
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
```

지원 도구 체인은 Go 1.27.2입니다. Linux·WSL·macOS의 호스트 셸·네트워크 검증은 [개발 가이드](docs/development.md#플랫폼-실행-검증)를 따르며, 변경 시 [개발 규칙](AGENTS.md)을 적용합니다.

실제 모델의 작업·요약 품질, Linux Secret Service·배포판별 호스트 권한과 실행 환경, WSL·SSH·터미널별 키·마우스 전달과 실제 입력 지연은 추가 검증 대상입니다. 지원 운영체제는 Linux와 macOS입니다.

## 문서

| 문서 | 내용 |
| --- | --- |
| [사용 가이드](docs/usage.md) | 모델 연결, 키맵, 승인, 스트림과 오류 복구 |
| [설정과 대화 보존](docs/configuration.md) | 역할별 인자, 프롬프트, Compaction, 세션 저장 |
| [에이전트 작동 방식](docs/runtime.md) | 단일 에이전트, native tools, 결과 확인, 종료 |
| [모델 연결](docs/model-connections.md) | 서버 원본 표시, 서빙 한도, Thinking 설정과 진단 |
| [실행 경계](docs/execution-boundary.md) | 호스트 개발 환경, 승인·파일 검사와 실행 권한 |
| [개발 가이드](docs/development.md) | 패키지 구성, 의존성·라이선스, 검증·성능 기록 |
| [제품 명세](SYSTEM.md) | 요구사항과 아키텍처 |

에이전트 응답은 흑백 Markdown으로 표시하며 물결표 취소선은 사용하지 않습니다. 상세 표시 규칙은 [사용법](docs/usage.md), 재배포 시 포함할 라이선스는 [외부 소프트웨어 고지](THIRD_PARTY_NOTICES.md)를 참고하세요.
