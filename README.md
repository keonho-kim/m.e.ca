<div align="center">

# V

**원하는 모델 서버와 연결하는 터미널 코딩 에이전트.**

Go · Linux / macOS · amd64 / arm64

[빌드](#직접-빌드하기) · [사용법](#사용법) · [설정](#설정) · [문서](#문서)

</div>

V는 터미널에서 코드를 읽고, 수정하고, 검증하는 에이전트입니다. 실행 파일은 `v` 하나이며, 변경 내용을 확인하고 승인하면서 작업할 수 있습니다.

- **모델 선택** — Ollama, vLLM, SGLang, LM Studio, LiteLLM, OpenAI 호환 서버 연결.
- **변경 검토** — 줄 단위 diff와 사용자 승인, 별도로 설정하는 파일 접근 범위.
- **작업 이어가기** — 로컬 세션 저장과 대화 압축, `v resume`으로 이전 작업 재개.
- **키보드 중심 UI** — 스트리밍 응답, `@` 파일 참조, `!` 명령 검색과 문맥별 키 안내.

## 직접 빌드하기

**Git과 Go 1.27.2**가 필요합니다. Linux에서는 내장 Bubblewrap을 준비할 **Docker 엔진과 BuildKit**도 필요합니다. macOS 빌드에는 Docker가 필요하지 않습니다.

```sh
git clone https://github.com/keonho-kim/v-code.git
cd v-code
go mod download
```

이후 명령은 저장소 루트에서 실행합니다. 비공개 저장소라면 GitHub 접근 권한이 필요합니다.

### macOS

```sh
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -o bin/v ./cmd/v
```

### Linux

```sh
docker version
sh scripts/build-bwrap.sh "$(go env GOHOSTARCH)"
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -o bin/v ./cmd/v
```

두 환경 모두 현재 CPU에 맞는 `bin/v`가 생성됩니다. Linux는 helper 준비가 필요하며, 프로젝트 셸 실행에는 호스트의 사용자·마운트 네임스페이스 지원이 필요합니다.

### 실행

대화형 터미널에서 실행합니다. 빌드한 바이너리 실행에는 Go나 Docker가 필요하지 않습니다.

```sh
./bin/v                   # 현재 프로젝트에서 시작
./bin/v /path/to/project  # 지정한 프로젝트에서 시작
./bin/v resume            # 이전 대화를 선택해 이어가기
```

<details>
<summary><strong>어디서든 v로 실행하기</strong></summary>

```sh
mkdir -p ~/.local/bin
install -m 755 bin/v ~/.local/bin/v
export PATH="$HOME/.local/bin:$PATH"
```

새 터미널에서도 사용하려면 셸 설정 파일에 `~/.local/bin`을 PATH로 추가합니다.

</details>

## 사용법

1. `!model`에서 Provider·주소·모델을 선택합니다. 인증이 필요한 서버는 API Key도 입력합니다.
2. 자연어로 작업을 요청합니다. `@파일명` 또는 `@경로`로 파일을 참조할 수 있습니다.
3. 제안된 변경을 diff로 확인하고 승인합니다. 먼저 계획을 검토하려면 `!plan`을 사용합니다.

| 명령 | 기능 |
| --- | --- |
| `!model` | 모델 연결, Thinking ON/OFF와 지원되는 추론 강도 선택 |
| `!plan` | 변경 전 계획 작성·검토 |
| `!dir [path]` | 파일·디렉터리 탐색 |
| `!scope` | 접근 범위를 프로젝트 또는 전체 시스템으로 설정 |
| `!permission` | 변경·명령 실행의 승인 정책 설정 |
| `!quit` | 현재 작업을 취소·정리하고 종료 |

`!`를 입력하면 명령 후보가, `@`를 입력하면 파일 후보가 나타납니다. `↑`·`↓`로 이동하고 `Enter`로 선택합니다. `Ctrl+C`로도 종료할 수 있으며, 나머지 키는 화면 하단에서 안내합니다.

기본값은 **프로젝트 범위 · 승인 필요 · Thinking OFF**입니다. 자동 승인을 선택해도 접근 범위가 넓어지지는 않습니다. API Key는 macOS Keychain 또는 Linux Secret Service에 저장합니다.

[자세한 사용법과 키맵 →](docs/usage.md) · [모델 식별·프로파일 →](docs/model-profiles.md)

## 설정

첫 실행에서 `~/.v`와 기본 파일을 자동 생성합니다. 기존 파일은 덮어쓰지 않습니다.

```text
~/.v/
├── config.toml     # 전역 설정과 역할별 추론 인자
├── prompts/        # 사용자 수정 가능한 Markdown 프롬프트
├── sessions/       # 세션별 SQLite 대화 기록
└── logs/
```

추론 설정은 Provider가 아닌 **역할별**로 적용합니다. 기본 temperature는 작업·코드 생성 `0.7`, Conclave 심사 `0.4`, 요약·단순 필드 수집 `0.2`입니다.

```toml
[inference.agent]
temperature = 0.7
max_output_tokens = 2048
```

[전체 역할별 설정·프롬프트·Compaction →](docs/configuration.md)

## 개발

단일 Go 모듈이며 진입점은 [`cmd/v`](cmd/v)입니다. 주요 의존성은 tcell, go-toml, modernc SQLite, go-keyring, uniseg, x/term, mvdan/sh입니다. 버전은 [`go.mod`](go.mod)에 고정하며, SQLite는 CGO 없이 빌드합니다.

```sh
gofmt -l cmd internal
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
```

Linux에서는 위 빌드 절차로 helper를 먼저 준비합니다. 지원 도구 체인은 Go 1.27.2이며, 변경 시 [개발 규칙](AGENTS.md)을 따릅니다.

실제 모델의 작업·요약 품질, Linux Secret Service·배포판별 격리 정책, SSH·터미널별 표현은 추가 검증 대상입니다. Windows는 지원하지 않습니다.

## 문서

| 문서 | 내용 |
| --- | --- |
| [사용 가이드](docs/usage.md) | 모델 연결, 키맵, 승인, 스트림과 오류 복구 |
| [설정과 대화 보존](docs/configuration.md) | 역할별 인자, 프롬프트, Compaction, 세션 저장 |
| [에이전트 작동 방식](docs/runtime.md) | Goal·Task, PFC, self-grounding, Conclave, 검증 루프 |
| [모델 프로파일](docs/model-profiles.md) | 모델 식별, 서빙 별칭, 모델별 Thinking 제어 |
| [개발 가이드](docs/development.md) | 패키지 구성, 의존성·라이선스, 검증·성능 기록 |
| [제품 명세](SYSTEM.md) | 요구사항과 아키텍처 |
