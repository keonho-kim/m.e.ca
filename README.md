<div align="center">

<img src="webui/public/favicon.svg" alt="MECA" width="96" height="96">

[설치](#설치) · [Provider](#지원-provider) · [사용법](#사용법) · [팀 모드](#팀-모드) · [스킬](#스킬과-추가-방법) · [서버·WebUI](#서버) · [문서](#문서)

</div>

**M.E.CA — Minimal Essentials Coding Agent**는 TUI·CLI·내장 WebUI에서 사용하는 코딩 에이전트입니다. 선택한 모델로 코드를 수정·검증하고 대화를 로컬에 저장합니다. 파일·이미지 참조, 실행 중 방향 수정과 후속 요청 예약, 팀 협업을 지원합니다.

## 설치

macOS와 Linux·WSL의 amd64·arm64를 지원합니다. 배포본은 Go·Bun·Node 설치 없이 실행할 수 있습니다.

| 종류 | 포함 기능 |
| --- | --- |
| `tui` | TUI·CLI, 모든 Provider·도구·팀·스킬 |
| `full` | tui의 모든 기능 + 서버 API·내장 WebUI |

종류와 플랫폼에 맞는 파일을 받아 압축을 푼 폴더에서 아래 명령 중 하나를 실행합니다. [최신 릴리스 다운로드](https://github.com/keonho-kim/m.e.ca/releases/latest)

```sh
./meca install  # 기본 설치
```

```sh
./meca install --ext browser  # MECA 본체와 브라우저 도구 설치
```

```sh
./meca install --no-shell  # 로그인 셸 전환 없이 설치
```

Chrome·Chromium과 `agent-browser` 도우미가 준비된 환경에서 에이전트가 브라우저 도구를 사용합니다. [설치 옵션](docs/releases.md#다운로드와-설치) · [브라우저 도구](docs/browser.md)

### 업데이트

실행 중인 MECA를 종료한 뒤 아래 명령 중 하나를 실행합니다. [업데이트 안내](docs/releases.md#업데이트)

```sh
meca upgrade  # 현재 배포 종류로 업데이트
```

```sh
meca upgrade --tui  # 최신 tui로 업데이트
```

```sh
meca upgrade --full  # 최신 full로 업데이트
```

## 지원 Provider

Ollama · vLLM · SGLang · LM Studio · LiteLLM · OpenRouter · OpenAI · Claude · Gemini

[모델 연결 가이드](docs/model-connections.md)

## 사용법

```sh
meca                   # 현재 프로젝트에서 시작
meca /path/to/project  # 지정한 프로젝트에서 시작
meca resume            # 저장된 대화 선택
meca --print "이 변경 사항을 검토해줘"
meca --mode json "요청한 파일을 만들어줘"
```

`!model`로 모델을 연결하고 `!dir`에서 파일을 참조합니다. TUI에서 응답 중 `Enter`는 방향 수정, `Tab`은 후속 요청 예약, `Esc`는 작업 중지입니다. [명령과 키보드](docs/usage.md) · [이미지 첨부](docs/images.md) · [실행과 승인](docs/execution-boundary.md)

## 팀 모드

현재 대화에서 `!team`을 켜면 최대 8개 에이전트가 같은 프로젝트에서 협업합니다. WebUI에서는 입력창의 팀 버튼이나 설정에서 전환합니다. [팀 모드](docs/team.md)

## 스킬과 추가 방법

스킬은 에이전트가 읽는 `SKILL.md` 작업 지침입니다. `!skills`로 찾아보고 `#스킬명`으로 호출합니다. 기본으로 `create-skill`·`ui-design`·`backend-design`을 제공합니다.

프로젝트의 `.agents/skills/review/SKILL.md`를 다음처럼 만듭니다. 개인 스킬은 `~/.meca/skills/`에 같은 구조로 둡니다.

```markdown
---
name: review
description: 코드 변경의 정확성과 회귀 위험을 검토한다.
---

# 코드 리뷰

변경과 관련 테스트를 검토하고, 발견한 문제의 근거와 파일 위치를 기록한다.
```

`#review 변경 사항을 검토해줘`처럼 호출합니다. [스킬 작성법](docs/skills.md) · [기본 스킬](docs/builtin-skills.md)

## 서버

**full 배포본**에서 실행합니다.

```sh
meca /path/to/project --server --ui  # 내장 WebUI
meca --server --protocol sse        # API 서버
```

WebUI는 기본 주소 `http://127.0.0.1:8787/ui/`에서 열고 서버 터미널에 표시된 토큰으로 접속합니다. API 전송은 `sse`·`socket`·`grpc` 중 선택하며 `--concurrency N`으로 동시 작업 수를 정합니다. [WebUI 사용법](docs/webui-design.md) · [서버 API](docs/server.md)

## 문서

[전체 문서](docs/README.md) · [설정과 백업](docs/configuration.md) · [제품 명세](SYSTEM.md) · [기능별 최적화](docs/optimization.md)

## 개발

소스 빌드는 Go **1.27.4 이상**, full의 UI 빌드는 Bun **1.4.4 이상**을 사용합니다. [개발 가이드](docs/development.md)

## 기여 정책과 라이선스

버그와 기능 제안은 [GitHub Issues](https://github.com/keonho-kim/m.e.ca/issues)에서 받습니다. [기여 정책](CONTRIBUTING.md) · [보안 제보](SECURITY.md)

[Apache License 2.0](LICENSE) · [저작권 고지](NOTICE) · [외부 소프트웨어 고지](THIRD_PARTY_NOTICES.md)
