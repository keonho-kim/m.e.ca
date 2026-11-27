<div align="center">

<img src="webui/public/favicon.svg" alt="MECA" width="96" height="96">

[설치](#설치) · [모델 연결](#지원-provider) · [사용법](#사용법) · [도구 실행](#도구-실행) · [팀 모드](#팀-모드) · [스킬](#스킬) · [서버·WebUI](#서버) · [문서](#문서)

</div>

**M.E.CA — Minimal Essentials Coding Agent**는 TUI·CLI·내장 WebUI에서 사용하는 코딩 에이전트입니다. 선택한 모델과 개발 방향을 논의하고, 코드의 설계·구현·검증을 요청할 수 있습니다. 파일·이미지 참조, 실행 중 방향 수정, 후속 요청 예약과 팀 협업을 지원합니다.

대화와 실행 기록은 로컬에 보관하며 모델 요청은 사용자가 선택한 Provider로 보냅니다. TUI·CLI·서버·WebUI는 같은 도구 실행·승인·저장 흐름을 사용합니다.

## 설치

macOS와 Linux·WSL의 amd64·arm64를 지원합니다. 배포본은 Go·Bun·Node 설치 없이 실행할 수 있습니다.

| 종류 | 포함 기능 |
| --- | --- |
| `tui` | TUI·CLI, 모든 Provider·도구·팀·스킬 |
| `full` | tui의 모든 기능 + 서버 API·내장 WebUI |

종류와 플랫폼에 맞는 파일을 받아 압축을 푼 폴더에서 설치합니다. [최신 릴리스 다운로드](https://github.com/keonho-kim/m.e.ca/releases/latest)

```sh
./meca install
```

실행 파일은 `~/.local/bin/meca`에 설치하고, `~/.meca/`에 설정·프롬프트·기본 스킬·로컬 암호화 키 저장소를 준비합니다. `--no-shell`은 설치 후 로그인 셸 전환을 생략하며, `--ext browser`는 브라우저 도우미와 실행 환경을 함께 준비합니다. [설치 옵션](docs/cli.md#설치) · [브라우저 설치](docs/tools.md#설치와-갱신) · [배포 파일과 소스 설치](docs/releases.md)

### 업데이트

실행 중인 MECA를 종료한 뒤 현재 배포 종류로 업데이트합니다.

```sh
meca upgrade
```

`--tui`·`--full`로 배포 종류를 선택할 수 있습니다. 설치·업데이트는 배포 기본 프롬프트와 스킬 파일을 갱신하며, 설정·모델 연결·자격 증명·대화와 사용자 추가 자료를 보존합니다. 유지할 개인 지침은 프로젝트 `AGENTS.md`나 별도 스킬에 둡니다. [업데이트](docs/cli.md#업데이트) · [삭제와 백업](docs/cli.md#삭제) · [릴리스 노트](docs/release-notes.md)

## 지원 Provider

Ollama · vLLM · SGLang · LM Studio · LiteLLM · OpenRouter · OpenAI · Claude · Gemini

TUI의 `!model` 또는 WebUI 설정에서 Provider·주소·모델을 선택합니다. 비대화형 실행은 저장된 세션의 모델 선택이나 전역 기본 모델을 사용합니다. 도구 작업에는 모델과 서버의 native tool calling 지원이 필요합니다. [모델 연결 가이드](docs/model-connections.md) · [이미지 지원](docs/images.md#지원-확인)

## 사용법

```sh
meca                   # 현재 프로젝트에서 시작
meca /path/to/project  # 지정한 프로젝트에서 시작
meca resume            # 저장된 대화 선택
meca --print "이 변경 사항을 검토해줘"
meca --mode json "요청한 파일을 만들어줘"
```

`!dir`에서 파일을 참조하고 지원되는 모델에는 이미지를 첨부할 수 있습니다. TUI에서 응답 중 `Enter`는 방향 수정, `Tab`은 후속 요청 예약, `Esc`는 작업 중지입니다. `--print`는 최종 답변을, `--mode json`은 순서 있는 실행 이벤트를 출력합니다. [CLI 명령과 옵션](docs/cli.md) · [TUI 사용법](docs/usage.md) · [이미지 첨부](docs/images.md)

## 도구 실행

모델은 `codemode`에 JavaScript·TypeScript 본문을 전달하고, 그 안의 `tools.*`로 명령·파일·이미지·스킬·브라우저·협업 기능을 사용합니다. 변환기와 실행기는 MECA 바이너리에 포함됩니다.

실제 호스트 호출마다 인자·대상·승인 정책을 확인합니다. 코드의 명시적 반환값을 모델에 전달하며, `undefined`로 정상 종료하면 크기를 제한한 성공 결과를 자동 전달하고 잘림을 표시합니다. 원래 작업을 다시 실행하지 않습니다. 스킬 지침과 이미지는 모델이 확인할 입력으로 전달합니다. [Code Mode](docs/codemode.md) · [호스트 함수 목록](docs/tools.md#도구-목록) · [실행과 승인](docs/execution-boundary.md)

## 팀 모드

현재 대화에서 `!team`을 켜거나 WebUI 입력창의 팀 버튼·설정에서 전환합니다. 팀 요청은 두 에이전트로 시작하고, 필요한 독립 작업과 실행 예산에 따라 동료를 초대합니다. 각 참여자는 개인 대화와 문맥을 유지하면서 같은 프로젝트·공용 대화·담당 범위를 공유합니다. [팀 모드](docs/team.md)

## 스킬

스킬은 에이전트가 읽는 `SKILL.md` 작업 지침입니다. `!skills`로 찾아보고 `#스킬명`으로 요청에 지정합니다. 모델은 현재 요청에 필요한 지침과 참고 자료를 읽어 적용합니다.

| 기본 스킬 | 용도 |
| --- | --- |
| `#development` | 설계 패턴·알고리즘·자료구조·최적화와 앱·UI·백엔드 구현·검증 |
| `#create-skill` | 스킬 작성·수정과 참고 자료 구성 |

설치된 개발 자료는 모델 연결 없이 검색할 수 있습니다.

```sh
meca develop "delayed task queue"
```

[스킬 사용과 작성](docs/skills.md) · [기본 스킬 구성](docs/builtin-skills.md) · [검색 명령](docs/cli.md#개발-자료-검색)

## 서버

**full 배포본**에서 실행합니다.

```sh
meca /path/to/project --server --ui  # 내장 WebUI
meca --server --protocol sse        # API 서버
```

WebUI는 기본 주소 `http://127.0.0.1:8787/ui/`에서 열고 서버 터미널에 표시된 토큰으로 접속합니다. API 전송은 `sse`·`socket`·`grpc` 중 선택하며 `--concurrency N`으로 동시 작업 수를 정합니다. [WebUI 사용법](docs/webui-design.md) · [서버 API](docs/server.md)

## 문서

[전체 문서 목록](docs/README.md)에서 목적에 맞는 가이드를 찾을 수 있습니다.

| 목적 | 문서 |
| --- | --- |
| 실행과 자동화 | [CLI](docs/cli.md) · [TUI 사용법](docs/usage.md) |
| 도구와 승인 | [실행 흐름](docs/runtime.md) · [Code Mode](docs/codemode.md) · [호스트 함수](docs/tools.md) · [승인 경계](docs/execution-boundary.md) |
| 설정과 데이터 보존 | [설정·백업·복구](docs/configuration.md) |
| 서버 연동과 운영 | [서버 API](docs/server.md) · [WebUI 연동](docs/webui-api.md) · [이벤트와 로그](docs/diagnostics.md) |
| 제품 구조와 개발 | [제품 명세](SYSTEM.md) · [개발 가이드](docs/development.md) · [기능별 최적화](docs/optimization-strategies.md) |

## 개발

소스 빌드는 Go **1.27.4 이상**, full의 UI 빌드는 Bun **1.4.4 이상**을 사용합니다. [개발 가이드](docs/development.md)

## 기여 정책과 라이선스

버그와 기능 제안은 [GitHub Issues](https://github.com/keonho-kim/m.e.ca/issues)에서 받습니다. [기여 정책](CONTRIBUTING.md) · [보안 제보](SECURITY.md)

[Apache License 2.0](LICENSE) · [저작권 고지](NOTICE) · [외부 소프트웨어 고지](THIRD_PARTY_NOTICES.md)
