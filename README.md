<div align="center">

<img src="docs/assets/meca-wordmark.png" alt="M.E.CA — Minimal Essentials Coding Agent" width="320">

*A coding agent with just the essentials.*

[설치](#설치) · [사용법](#사용법) · [Program Philosophy](#program-philosophy) · [설정](#설정) · [문서](#문서)

</div>

**M.E.CA — Minimal Essentials Coding Agent**는 필요한 기능만 담은 터미널 코딩 에이전트입니다. 원하는 모델을 연결해 코드를 읽고, 수정하고, 검증하며 같은 대화에서 작업 내역과 파일 변경을 확인합니다.

실행 명령은 `meca`이며 설정과 대화는 `~/.meca`에 보관합니다.

## 주요 기능

- **모델 선택** — Ollama, vLLM, SGLang, LM Studio, LiteLLM, OpenRouter 연결.
- **변경 검토** — 초록색·빨간색으로 구분하는 줄 단위 diff와 위험 작업의 필수 사용자 승인.
- **직접 작업과 종료** — 메인 에이전트가 파일·셸 도구로 직접 수행하고 실제 결과를 확인. 예산·오류 중단 시에도 수행 결과와 미완료 사항을 보고.
- **작업 이어가기** — 로컬 세션 저장과 대화 압축, `meca resume`으로 이전 작업 재개.
- **터미널 UI** — 스트리밍 응답, `!dir` 파일 참조, `!` 명령 검색, 마우스 스크롤과 문맥별 키 안내.
- **사용량 확인** — 실제 입력·출력 토큰과 TPS, OpenRouter의 실제 보고 비용 표시.

## 설치

### 최신 릴리스로 설치하기

[최신 릴리스 다운로드](https://github.com/keonho-kim/m.e.ca/releases/latest)에서 설치할 수 있습니다. Linux·WSL과 macOS의 amd64·arm64를 지원하며, 릴리스 바이너리는 Go 설치 없이 동작합니다.

최신 릴리스의 아카이브와 `checksums.txt`를 내려받습니다. Apple Silicon은 `macos_arm64`, Intel Mac은 `macos_amd64`, Linux·WSL은 `uname -m`이 `x86_64`이면 `linux_amd64`, `aarch64`이면 `linux_arm64`를 선택합니다. SHA-256 확인 후 빈 폴더에 압축을 풀고 그 안에서 `./meca install`을 실행합니다. 검증 명령은 [릴리스 가이드](docs/releases.md#다운로드와-설치)를 참고하세요.

설치하면 `~/.local/bin/meca`, `~/.meca`의 기본 설정·프롬프트와 필요한 셸 PATH 등록이 준비됩니다. 새 터미널과 재부팅 후에도 `meca` 명령을 사용할 수 있습니다.

`./meca install`은 대화형 터미널에서 PATH 반영이 필요하면 설치 프로세스를 `exec "$SHELL" -l`로 로그인 셸에 넘깁니다. 이 셸에서 `exit`하면 설치를 실행했던 기존 셸로 돌아갑니다. PATH가 이미 반영되어 있거나 입력·출력이 터미널이 아니면 전환하지 않습니다. 자동화나 수동 전환을 원할 때는 `./meca install --no-shell`을 사용하세요.

업데이트는 실행 중인 MECA를 모두 종료한 뒤 실행합니다.

```sh
meca upgrade
```

OS·CPU에 맞는 최신 GitHub 릴리스를 내려받아 SHA-256을 확인하고 설치합니다. 같은 날짜의 릴리스도 실제 바이너리를 비교하며, 설정·모델 연결·대화는 유지합니다. 설치·업데이트 시 세 프롬프트는 사용자 수정 여부와 관계없이 배포 기본본으로 갱신합니다. 비공개 저장소는 `GH_TOKEN`, `GITHUB_TOKEN`, 기존 `gh` 로그인 순서로 인증하며 공개 저장소는 인증 없이 사용할 수 있습니다. Go나 GitHub CLI 설치는 필수가 아닙니다. [업데이트 동작 →](docs/releases.md#업데이트)

`main`에 푸시하면 검사·빌드 후 릴리스를 자동으로 갱신합니다. major·minor는 [.release-version](.release-version)에 직접 지정하며, 날짜만 한국 시간 기준으로 바뀝니다. 같은 날짜의 추가 푸시는 그날의 태그와 배포 파일을 갱신합니다.

### 소스에서 설치하기

소스 설치에는 **Git**이 필요합니다. `install.sh`는 설치된 Go를 사용하며, Go가 없으면 공식 Go 1.27.2 배포본을 SHA-256으로 검증해 빌드용으로 임시 설치합니다. 자동 다운로드에는 curl·tar와 sha256sum 또는 shasum이 필요합니다. Linux와 macOS의 amd64·arm64를 지원합니다.

```sh
git clone https://github.com/keonho-kim/m.e.ca.git
cd m.e.ca
```

이후 명령은 저장소 루트에서 실행합니다.

```sh
./install.sh
```

현재 OS·CPU에 맞춰 `bin/meca`를 빌드하고 `~/.local/bin/meca`에 설치합니다. `~/.meca`와 기본 설정·프롬프트를 생성하고, 필요한 PATH 등록을 zsh·bash·sh 설정에 추가합니다. 기존 설정은 유지하며, 대화형 터미널에서는 위와 같은 로그인 셸 전환으로 PATH를 바로 반영합니다.

자동으로 받은 Go는 저장소의 `.cache` 아래에서 이번 빌드에만 사용하고 설치·셸 전환 전에 정리합니다. 빌드 실패 시에도 정리합니다. Go의 모듈·빌드 캐시는 일반 Go 설정을 따르며 시스템 Go 설치는 유지합니다. 설치와 새 셸은 `install.sh`를 실행한 작업 폴더와 환경을 이어받습니다.

### 빌드만 하기

Go 1.27.2 이상을 설치한 환경에서 실행합니다.

```sh
go build -trimpath -o bin/meca ./cmd/meca
```

`go build`는 컴파일만 합니다. 초기화·설치·PATH 등록은 `./install.sh` 또는 빌드한 바이너리의 `./bin/meca install`로 실행합니다.

### 실행

```sh
meca                  # 현재 프로젝트에서 시작
meca /path/to/project  # 지정한 프로젝트에서 시작
meca resume           # 이전 대화를 선택해 이어가기
```

### 터미널 없이 실행하기

```sh
meca --print "README를 검토해줘"
meca /path/to/project --mode json "요청한 파일을 만들어줘"
git diff | meca --print "이 변경을 검토해줘"
meca --session SESSION_ID --print "이어서 진행해줘"
meca --session SESSION_ID --export > session.jsonl
meca --diagnose
```

먼저 `!model`에서 기본 연결을 저장합니다. print는 최종 답변, JSON 모드는 순서 있는 이벤트를 stdout에 기록하며 같은 에이전트 루프·세션·파일 보호 정책을 사용합니다. 비대화형 실행의 필수 승인 요청은 명시적으로 차단합니다. 세션 ID는 첫 JSON `session` 이벤트의 `session_id` 또는 `~/.meca/sessions/<project-key>/`의 DB 파일명으로 확인합니다. Export는 원본 세션 이력을 JSONL로 출력합니다.

### 완전히 삭제하기

```sh
meca uninstall
```

설치된 `~/.local/bin/meca`, MECA가 추가한 PATH 블록, `~/.meca` 전체(설정·프롬프트·대화·로그)와 MECA가 저장한 OS 인증정보를 삭제합니다. 실행 중인 에이전트를 모두 종료한 뒤 수행합니다. 기존 셸 설정과 사용자 프로젝트는 유지하며, 삭제에 실패하면 원인을 반환합니다. 소스 저장소와 공용 Go·패키지 관리자 캐시는 삭제하지 않습니다.

## 사용법

### 모델 연결

모델 서버는 native tool calling을 지원해야 합니다. vLLM 등에서는 모델에 맞는 Tool Parser 설정도 필요합니다. [서버 설정 안내 →](docs/model-connections.md#native-tool-calling)

1. `!model`의 한 화면에서 Provider·Base URL·API Key·모델·추론 설정을 편집합니다. `↑↓`로 항목 이동, `→`로 값이 표시된 자리에서 입력·값 변경, `Enter`로 저장 확인을 열고 `y`로 저장합니다. `n`은 취소 후 채팅, `Esc`는 설정 편집을 계속합니다.
2. 자연어로 작업을 요청합니다. 참조할 파일은 `!dir`에서 선택할 수 있습니다.
3. 변경을 diff로 확인합니다. 일반 작업은 자동 승인하며, 위험 작업은 실행 전에 별도 승인을 받습니다. 분석만 원하면 파일을 바꾸지 말고 분석해 달라고 요청합니다.

모델명 행에서 `→`로 편집을 시작하면 목록을 자동 조회합니다. 입력한 글자로 부분 검색한 뒤 `↑↓`로 선택하고 `Tab`으로 채웁니다. 편집을 마친 뒤 다시 `→`로 들어가면 목록을 갱신하며, 타이핑 중에는 받은 목록으로 검색합니다. 목록 조회가 안 되는 서버에서도 모델 ID를 직접 입력할 수 있습니다. `Enter`는 전체 설정의 저장 확인입니다.

OpenRouter는 Provider를 `OpenRouter`, Base URL을 기본값 `https://openrouter.ai/api/v1`로 두고 OpenRouter API Key를 입력하면 됩니다. [OpenRouter 연결 안내 →](docs/model-connections.md#openrouter-연결)

| Provider | 기본 Base URL |
| --- | --- |
| Ollama | `http://localhost:11434` |
| vLLM | `http://localhost:8000/v1` |
| SGLang | `http://localhost:8000/v1` |
| LM Studio | `http://localhost:1234` |
| LiteLLM | `http://localhost:4000/v1` |
| OpenRouter | `https://openrouter.ai/api/v1` |

### 대화와 파일 작업

| 명령 | 기능 |
| --- | --- |
| `!model` | 모델 연결, 추론 켜기·끄기와 지원되는 추론 강도 선택 |
| `!language` | 화면 언어 설정 · 시스템 기본값 / 한국어 / English · 저장 확인 |
| `!dir [path]` | 파일·디렉터리 탐색 |
| `!new` | 같은 프로젝트에서 새 세션 시작 · 전역 모델 연결 유지 |
| `!quit` | 현재 작업을 취소·정리하고 종료 |

`!`를 입력하면 명령 후보가 나타나며 입력한 글자로 검색하고 `↑↓`·`Enter`로 실행합니다. 명령·모델 검색은 부분일치를 우선하고 결과가 없으면 `mdl` → `model` 같은 축약 검색을 사용합니다. 키맵은 화면 하단에 최대 두 줄로 표시하고, 모델명은 그 바로 위에 둡니다.

파일 조회·검색은 `execute_command`의 `cat`, `sed -n`, `rg` 등을 사용합니다. 삭제·이동은 같은 셸의 `rm`·`mv`로 수행합니다. 생성·본문 수정에는 원본 검사와 diff를 제공하는 `write_file`·`edit_file`, 폴더 생성에는 누락된 부모도 함께 만드는 `create_directory`를 사용합니다.

| 조작 | 동작 |
| --- | --- |
| `Enter` / `Shift+Enter` | 전송 / 줄바꿈 |
| 입력창 `↑↓` / `Ctrl/Cmd+↑↓` | 줄 이동 / 이전 입력 탐색 |
| 작업 중 `Enter` / `Tab` | 다음 판단에 방향 수정 / 개수 제한 없이 후속 요청 예약 |
| `Shift+←` / 큐 편집 중 `Shift+←/→` | 마지막 예약 열기 / 이전·다음 예약 탐색 |
| 큐 편집 중 `Enter`·`Tab` / `Esc` / `Ctrl+D` | 수정 저장 / 편집 취소 / 선택한 예약 삭제 |
| `Esc` | 열린 UI·편집 닫기 · 일반 채팅에서는 작업 중지 |
| `Ctrl+O` | 도구 결과 전체 접기·펼치기 · 기본 펼침 |
| 선택 후 `Ctrl/Cmd+C` | 선택한 원문 복사 · 선택 없는 `Ctrl+C`는 종료 |

입력창에서 선택한 내용은 Backspace로 삭제합니다. 긴 대화는 오른쪽 스크롤바로 탐색하며, 위에서는 읽던 위치를 유지하고 하단에서는 새 출력을 따라갑니다. 키 전달은 터미널 설정에 따라 달라질 수 있습니다. [전체 키맵과 조작 →](docs/usage.md)

작업 중 요청을 작성하고 `Tab`을 누르면 대기열 끝에 추가됩니다. 현재 작업이 끝나면 예약 순서대로 하나씩 실행하며, 화면에는 대기 건수와 다음 요청을 표시합니다. `Esc` 중지·작업 실패 시 남은 요청은 작성 중인 입력 앞에 순서대로 복구합니다. 대기열은 현재 실행 중인 세션에서 관리하며 실행된 요청부터 대화 DB에 기록합니다.

채팅에서 `Shift+←`로 마지막 예약을 열고 `Shift+←/→`로 이동해 기존 입력창에서 수정합니다. 입력창 위에 `예약 편집 2 / 5`처럼 위치를 표시하며, 마지막 예약 다음으로 이동하면 원래 작성 중인 입력으로 돌아갑니다. `Enter`·`Tab`은 같은 큐 위치에 저장하고 복귀하며, `Esc`는 원문을 유지한 채 편집만 취소합니다. `Ctrl+D`는 선택한 예약 자체를 삭제합니다. 미저장 수정은 저장·취소 후 이동할 수 있습니다. 편집 중에도 현재 작업은 계속하며 다음 예약 시작만 보류합니다. 편집 종료 후에는 작업의 성공·실패에 따라 실행을 재개하거나 남은 요청을 초안으로 복구합니다.

### 사용량

상단에는 실제 입력·출력 토큰과 TPS를 표시합니다. OpenRouter에서는 `INPUT · OUTPUT · TOTAL COST · TPS` 순서로 세션의 보고 비용 합계를 USD로 함께 표시합니다. 한국어 UI에서는 `입력 · 출력 · 총 비용 · 초당 토큰`입니다. 비용 미수신은 `—`, 실제 무료 호출은 `$0`이며, 토큰 단가로 추정하지 않습니다.

작업이 끝나면 마지막 응답 아래에 소요 시간·이번 작업의 실제 INPUT/OUTPUT·평균 TPS를 흐린 글씨로 가운데 표시합니다. OpenRouter 비용을 수신한 작업에는 OUTPUT과 TPS 사이에 해당 작업의 비용도 표시합니다. 세션 사용량과 작업 요약은 대화를 다시 열어도 복원됩니다. [집계 범위와 표시 기준 →](docs/usage.md#화면과-사용량)

### 파일 참조와 화면 설정

`!dir`에서 현재 보고 있는 폴더의 파일이 바뀌면 목록과 열린 미리보기를 자동으로 갱신합니다. 작성 중인 요청과 선택은 유지하며 화면을 닫으면 감시도 종료합니다.

`!model`과 `!language`는 한 화면에서 값을 편집하고 `Enter`로 저장을 확인합니다. 확인 중 `y`는 저장 후 채팅 복귀, `n`은 변경을 버리고 복귀, `Esc`는 설정 편집을 계속합니다. `!new`와 `!quit`는 즉시 실행합니다.

`!dir`에서 Space로 선택한 경로는 입력창 위에 표시됩니다. `Shift+Tab`으로 입력창에 이동해 요청을 작성하고 `Enter`로 전송 확인을 엽니다. `y`는 채팅 복귀 후 전송, `n`·`Esc`는 입력을 유지합니다. 탐색·입력 중 `Esc`는 요청을 보내지 않고 바로 채팅으로 돌아갑니다. 미리보기의 `Esc`는 미리보기만 닫으며 `:q`는 파일 탐색 입력에서 일반 텍스트입니다.

화면 언어는 `!language`의 `←→`로 선택하거나 `~/.meca/config.toml`의 `language`에 `auto`·`ko`·`en`을 지정합니다. 기본값 `auto`는 시스템 로캘을 따릅니다. 설정 화면에서 저장하면 즉시 적용하며 직접 파일을 수정하면 재시작 후 반영합니다. 설정값·모델 ID·명령·출력 원문과 사용자·모델의 대화 본문은 유지합니다. [언어 선택 기준](docs/usage.md#화면-언어)을 참고하세요.

[자세한 사용법과 키맵 →](docs/usage.md) · [모델 연결·추론 설정 →](docs/model-connections.md)

### 실행 범위

일반 작업은 자동 승인합니다. 프로젝트 내부의 확정된 하위 파일·폴더 삭제와 내부 경로 간 이동도 자동 승인합니다. 외부·불확실한 대상의 삭제·이동, 파괴적·원격 Git 작업과 프로젝트 밖 직접 변경은 실행 전에 확인합니다. 프로젝트 루트 자체·상위 경로의 삭제·이동, 보호 경로 변경과 권한 상승은 차단합니다.

셸은 호스트의 HOME·PATH·DNS·개발 도구·캐시와 실행 사용자의 권한을 사용합니다. 프로젝트는 기본 작업 디렉터리이며 OS 격리 경계가 아닙니다. [실행 경계와 승인 정책 →](docs/execution-boundary.md)

## Program Philosophy

M.E.CA는 미니멀함과 가벼움을 설계의 기준으로 삼습니다. 적은 준비로 시작하고, 작업 흐름을 유지하며, 실제 결과를 확인하는 경험에 집중합니다.

- **Only the essentials.** 모델 연결, 파일·셸 도구, 이어갈 수 있는 대화. 메인 에이전트 하나가 요청을 받아 직접 작업합니다.
- **Lightweight by construction.** 하나의 Go 실행 파일로 시작하고, 사용자의 터미널과 설치된 개발 도구를 활용합니다. 기능과 설정은 코딩 작업에 필요한 만큼만 둡니다.
- **Stay in your flow.** 요청·응답·변경 확인을 터미널 안에서 이어갑니다. 사용자는 진행 중 방향을 바꾸거나 작업을 멈추고 재개할 수 있습니다.
- **Private by design.** **Your code. Your model. Your control.** 연결할 모델과 작업에 사용할 데이터를 사용자가 선택합니다.

세션·설정·프롬프트는 `~/.meca`에 저장하며 별도 MECA 계정이나 중앙 대화 서버를 사용하지 않습니다. API Key는 macOS Keychain 또는 Linux Secret Service에 보관하고 설정·DB에는 참조만 기록합니다. 마지막 모델 연결은 새 세션에서 재사용합니다.

추론까지 로컬에서 처리하려면 같은 기기의 Ollama·LM Studio 등으로 연결하세요. 원격 모델을 사용하면 프롬프트·대화·작업에 포함된 코드와 도구 결과가 해당 서버로 전송됩니다. OpenRouter에서는 실제 추론 공급자까지 각 서비스의 데이터 처리 정책이 적용됩니다. 셸 명령과 패키지 설치도 호스트의 네트워크를 사용할 수 있습니다. [데이터 보존과 설정 →](docs/configuration.md)

## 설정

첫 실행에서 `~/.meca`와 기본 파일을 자동 생성합니다. 기존 파일은 보존합니다.

```text
~/.meca/
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

세션 저장 형식은 schema 20입니다. 큰 본문은 같은 DB 안에서 중복 제거·압축하며 원문 조회와 내보내기에서 복원합니다. 미지원 DB는 자동 변환하거나 삭제하지 않고 원본을 보존해 오류를 안내합니다. 프롬프트 편집은 일반 실행 중 유지하며 설치·업데이트 시 배포 기본본으로 교체합니다. [설정과 프롬프트](docs/configuration.md)를 참고하세요.

## 개발

단일 Go 모듈이며 진입점은 [`cmd/meca`](cmd/meca)입니다. 주요 의존성은 mvdan/sh(셸 구문 분석), goldmark, tcell, go-toml, modernc SQLite, go-keyring, fsnotify(파일 변경 알림), uniseg, x/sys, x/term입니다. 테스트에는 goleak·go-cmp·creack/pty를 사용합니다. 버전은 [`go.mod`](go.mod)에 고정하며, SQLite는 CGO 없이 빌드합니다.

```sh
gofmt -l cmd internal
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
```

지원 도구 체인은 Go 1.27.2입니다. Linux·WSL·macOS의 호스트 셸·네트워크 검증은 [개발 가이드](docs/development.md#플랫폼-실행-검증)를 따릅니다.

실제 모델의 작업·요약 품질, Linux Secret Service·배포판별 호스트 권한과 실행 환경, WSL·SSH·터미널별 키·마우스 전달과 실제 입력 지연은 추가 검증 대상입니다. 지원 운영체제는 Linux와 macOS입니다.

## 기여 정책

M.E.CA는 개인이 단독 개발·관리하며 외부 코드 기여는 받지 않습니다. 버그 제보는 환영합니다.

버그와 기능 제안은 [GitHub Issues](https://github.com/keonho-kim/m.e.ca/issues)로 받습니다. 자세한 제보 방법과 코드 기여 정책은 [CONTRIBUTING.md](CONTRIBUTING.md)를 참고하세요.

## 라이선스

M.E.CA는 [Apache License 2.0](LICENSE)을 따릅니다. 라이선스 조건을 지키면 사용·수정·재배포할 수 있습니다. 원본 저장소의 개발·병합·배포는 관리자가 결정합니다.

프로젝트 저작권 고지는 [NOTICE](NOTICE), 외부 구성요소의 라이선스는 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)를 참고하세요.

## 문서

| 문서 | 내용 |
| --- | --- |
| [사용 가이드](docs/usage.md) | 모델 연결, 키맵, 승인, 스트림과 오류 복구 |
| [설정과 대화 보존](docs/configuration.md) | 역할별 인자, 프롬프트, Compaction, 세션 저장 |
| [에이전트 작동 방식](docs/runtime.md) | 단일 에이전트, native tools, 결과 확인, 종료 |
| [모델 연결](docs/model-connections.md) | OpenRouter 연결·실제 비용, 서버 원본 표시, 서빙 한도, Thinking 설정과 진단 |
| [실행 경계](docs/execution-boundary.md) | 호스트 개발 환경, 승인·파일 검사와 실행 권한 |
| [개발 가이드](docs/development.md) | 패키지 구성, 의존성·라이선스, 검증·성능 기록 |
| [릴리스 가이드](docs/releases.md) | macOS·Linux 배포 파일, 체크섬, 날짜 버전과 게시 절차 |
| [제품 명세](SYSTEM.md) | 요구사항과 아키텍처 |

에이전트 응답과 Thinking은 흑백 Markdown으로 표시하며 Thinking은 흐린 글씨를 유지합니다. 물결표 취소선은 사용하지 않습니다. 상세 표시 규칙은 [사용법](docs/usage.md)을 참고하세요.
