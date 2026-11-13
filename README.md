<div align="center">

<img src="webui/public/favicon.svg" alt="MECA" width="96" height="96">

[설치](#설치) · [Provider](#지원-provider) · [사용법](#사용법) · [팀 모드](#팀-모드) · [스킬](#스킬과-추가-방법) · [서버·WebUI](#서버) · [문서](#문서)

</div>

**M.E.CA — Minimal Essentials Coding Agent**는 TUI·CLI·내장 WebUI에서 사용하는 코딩 에이전트입니다. 선택한 모델로 코드를 작성·수정·검증하고 대화를 로컬에 저장합니다.

- 스트리밍 답변·Thinking과 파일·셸 도구
- 이미지 첨부, 파일 참조와 변경 내용 확인
- 대화 재개, 작업 중 방향 수정과 후속 요청 예약
- 같은 프로젝트에서 협업하는 팀 모드
- `SKILL.md` 기반 작업 지침과 에이전트의 스킬 탐색

## 설치

macOS와 Linux·WSL의 amd64·arm64를 지원합니다. 배포본은 Go·Bun·Node 설치 없이 실행할 수 있습니다.

| 종류 | 포함 기능 |
| --- | --- |
| `tui` | TUI·CLI, 모든 Provider·도구·팀·스킬 |
| `full` | tui의 모든 기능 + 서버 API·내장 WebUI |

### 최신 릴리스로 설치하기

1. [최신 릴리스](https://github.com/keonho-kim/m.e.ca/releases/latest)에서 종류·운영체제·CPU에 맞는 `.tar.gz`를 받습니다.
2. 압축을 풀고 해당 폴더의 터미널에서 설치합니다.

```sh
./meca install
```

설치 후 `meca`를 실행합니다.

### 업데이트

실행 중인 MECA를 종료한 뒤 사용합니다.

```sh
meca upgrade         # 현재 종류의 최신 버전
meca upgrade --tui   # tui 최신 버전으로 교체
meca upgrade --full  # full 최신 버전으로 교체
```

설정·대화는 유지하고 기본 프롬프트는 배포 기본본으로 갱신합니다. [파일 선택·체크섬·소스 설치·삭제 안내 →](docs/releases.md)

## 지원 Provider

- Ollama
- vLLM
- SGLang
- LM Studio
- LiteLLM
- OpenRouter
- OpenAI
- Claude
- Gemini

[모델 연결 가이드 →](docs/model-connections.md)

## 사용법

```sh
meca                  # 현재 프로젝트에서 시작
meca /path/to/project  # 지정한 프로젝트에서 시작
meca resume           # 저장된 대화에서 선택
```

`!model`로 모델을 연결한 뒤 자연어로 작업을 요청합니다. `!dir`에서 파일을 선택해 참조할 수 있습니다.

| TUI 명령 | 기능 |
| --- | --- |
| `!model` | 모델 연결·추론 설정 |
| `!language` | 화면 언어 |
| `!permission` | 승인 모드 |
| `!dir [path]` | 파일 탐색·참조 |
| `!new` | 새 대화 |
| `!quit` | 종료 |
| `!team` | 팀 모드 전환 |
| `!skills` | 스킬 탐색 |

TUI에서 `Enter`는 전송, `Shift+Enter`는 줄바꿈, `Esc`는 작업 중지입니다. 응답 중 `Enter`로 방향을 수정하고 `Tab`으로 후속 요청을 예약합니다. 이미지 첨부는 `Alt+V`, macOS에서는 `Ctrl+V`도 지원합니다.

터미널 화면 없이 실행하려면 다음을 사용합니다.

```sh
meca --print "이 변경 사항을 검토해줘"
meca --mode json "요청한 파일을 만들어줘"
```

명령은 호스트 사용자 권한으로 실행하며 프로젝트 폴더는 OS 격리 경계가 아닙니다. [사용 가이드](docs/usage.md) · [승인과 실행 범위](docs/execution-boundary.md)

## 팀 모드

현재 대화에서 `!team`을 켜면 최대 8개 에이전트가 같은 프로젝트의 작업을 나눠 진행합니다.

```text
!team
API 구현과 테스트를 나눠서 진행하고 결과를 함께 검토해줘.
```

WebUI에서는 컴포저의 팀 버튼이나 설정에서 전환합니다. [팀 모드 안내 →](docs/team.md)

## 스킬과 추가 방법

스킬은 에이전트가 읽는 `SKILL.md` 작업 지침입니다. `!skills`로 탐색하고 `#스킬명`으로 호출합니다. 에이전트가 필요한 스킬을 스스로 찾아 읽을 수도 있습니다.

```text
#ui-design 업무 대시보드의 화면과 사용 흐름을 설계해줘.
#backend-design API와 데이터 저장 구조를 설계하고 구현해줘.
#create-skill 이 프로젝트의 코드 리뷰 스킬을 만들어줘.
```

기본 스킬은 `create-skill`·`ui-design`·`backend-design`입니다. WebUI에서는 컴포저의 `+` → 스킬 목록으로 접근합니다.

### 내 스킬 추가하기

프로젝트의 `.agents/skills/review/SKILL.md`를 다음처럼 만듭니다. 개인 스킬은 `~/.meca/skills/`에 같은 구조로 둡니다.

```markdown
---
name: review
description: 변경한 코드의 정확성과 회귀 위험을 검토할 때 사용한다.
---

# 코드 리뷰

변경 내용과 관련 테스트를 확인하고, 근거가 있는 문제를 파일 위치와 함께 설명한다.
```

`!skills`에서 확인한 뒤 `#review 변경 사항을 검토해줘`로 사용합니다. [경로·작성 형식](docs/skills.md) · [기본 스킬 안내](docs/builtin-skills.md)

## 서버

서버 API와 내장 WebUI는 **full 배포본**에서 사용할 수 있습니다.

### 내장 WebUI 기동

```sh
meca /path/to/project --server --ui
```

1. 브라우저에서 `http://127.0.0.1:8787/ui/`를 엽니다.
2. 서버 터미널에 표시된 `Server token: …`의 값을 입력합니다.
3. 작업공간과 대화를 선택하고 설정에서 모델을 연결합니다.

WebUI는 다중 작업공간·대화, 파일 편집, 팀·스킬·이미지와 라이트·다크 테마를 제공합니다. 후속 요청 예약은 `Ctrl/Cmd+Enter`를 사용합니다. [WebUI 사용 안내 →](docs/webui-design.md)

### API 서버 기동

```sh
meca --server --protocol sse
meca --server --protocol socket
meca --server --protocol grpc
```

`--concurrency N`으로 동시에 실행할 사용자 작업 수를 설정합니다. 기본값은 1이며 WebUI와 함께 사용할 수 있습니다. [서버 옵션·인증·API 계약 →](docs/server.md)

## 문서

[전체 문서 목록](docs/README.md)에서 사용·연동·개발 안내를 찾을 수 있습니다.

| 문서 | 내용 |
| --- | --- |
| [사용 가이드](docs/usage.md) | 키보드·파일·이미지·예약·사용량 |
| [모델 연결](docs/model-connections.md) | Provider 연결과 추론 설정 |
| [팀 모드](docs/team.md) · [스킬](docs/skills.md) | 협업과 작업 지침 |
| [WebUI](docs/webui-design.md) · [연동 API](docs/webui-api.md) | 화면 사용과 클라이언트 연동 |
| [설정과 대화 보존](docs/configuration.md) | 설정·프롬프트·저장·복구 |
| [설치와 배포](docs/releases.md) | 배포 파일·업데이트·소스 설치·삭제 |
| [개발 가이드](docs/development.md) · [제품 명세](SYSTEM.md) | 책임·구조·개발 기준 |

## 개발

소스 빌드에는 Go **1.27.4 이상**, full의 UI 빌드에는 Bun **1.4.4 이상**이 필요합니다. [소스 빌드](docs/development.md#소스-빌드) · [개발·검증 안내](docs/development.md)

## 기여 정책과 라이선스

개인이 단독 개발·관리하며 외부 코드 기여는 받지 않습니다. 버그 제보와 기능 제안은 [GitHub Issues](https://github.com/keonho-kim/m.e.ca/issues)로 받습니다. [기여 정책](CONTRIBUTING.md) · [보안 제보](SECURITY.md)

[Apache License 2.0](LICENSE) · [저작권 고지](NOTICE) · [외부 소프트웨어 고지](THIRD_PARTY_NOTICES.md)
