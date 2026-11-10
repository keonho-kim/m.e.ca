# M.E.CA — Minimal Essentials Coding Agent

Go 실행 파일 하나로 코딩 작업을 수행한다. macOS·Linux의 amd64·arm64를 지원한다.

## 이번 릴리스

- **WebUI**는 로그인 전에 브라우저 세션을 확인하고 접속 주소별로 쿠키를 관리한다. 서버가 브라우저를 자동으로 열지 못하면 stdout에 표시된 주소로 접속할 수 있다.
- **작업공간 선택 창**에서 폴더를 탐색·생성하고 선택할 수 있다. 오른쪽 파일 목록에서 문서를 선택하면 바로 편집할 수 있고, Markdown은 Preview/Raw로 현재 초안을 전환해 본다. 미리보기 본문을 클릭하면 편집으로 돌아간다. 긴 코드도 화면 폭에 맞춰 줄바꿈한다.
- **WebUI 컴포저**의 `+` 메뉴에서 이미지 파일 선택기와 스킬 목록을 연다. 드롭·붙여넣기로 추가한 이미지도 컴포저 위 캐러셀에 표시하며 각 썸네일의 오른쪽 위에서 제거한다. `Ctrl+Enter` 또는 `Cmd+Enter`로 후속 요청을 예약하고, 실행 중 일반 텍스트를 전송하면 진행 중인 작업의 방향을 수정한다.
- **작업 목록**에 대화 제목과 상태를 표시하고, 모델 설정에서 현재 서버 주소의 모델 목록을 조회한다. 대화·파일·스킬 탭과 프로젝트·대화 정보는 별도 행에 표시한다.
- **TUI 이미지 첨부**는 입력 영역에 파란색 `[IMAGE:1]` 형태로 표시한다. 본문 맨 앞에서 `Backspace`를 누르거나 `Alt+Backspace`를 누르면 마지막 첨부를 제거한다.
- **긴 TUI 초안**은 편집 지점 직전 줄부터 재배치한다. 편집 한도를 넘는 파일은 메타데이터로 판정하며, SQLite에서는 선택한 첨부만 복원한다. WebUI는 같은 소스의 스트림 델타를 묶고 재접속 스냅샷에 포함된 대기 델타를 제거한다. 접힌 Thinking과 닫힌 실행 정보는 열릴 때 본문을 준비하고, 새 텍스트가 500ms 동안 오지 않으면 문자 애니메이션을 종료한다.

## 배포

| 종류 | 포함 기능 |
| --- | --- |
| `tui` | TUI·CLI, 모든 Provider·도구·스킬·팀 모드 |
| `full` | tui의 모든 기능과 SSE·WebSocket·gRPC 서버·내장 WebUI |

파일명은 `meca_<version>_<tui|full>_<platform>.tar.gz`다. 각 종류에 네 플랫폼을 제공하며 여덟 아카이브와 공용 `checksums.txt`를 게시한다. `meca --help` 첫 줄로 종류를 확인한다.

```sh
meca upgrade --tui
meca upgrade --full
```

지정한 종류의 최신 정식 배포본으로 교체한다. 옵션 없는 `meca upgrade`는 현재 종류를 유지하며 설정·모델 연결·인증정보·대화를 보존한다. 기본 프롬프트는 배포 기본본으로 갱신한다.

## 실행

- `meca [path]`: 프로젝트 TUI
- `meca --print`: 최종 답변 출력
- `meca --mode json`: 순서 있는 JSONL 이벤트 출력
- `meca --session ID`: 저장 대화 재개
- `meca --session ID --export`: 원본 이력 JSONL 내보내기
- full의 `meca --server --ui`: 내장 WebUI 기동
- full의 `meca --server --protocol sse|socket|grpc`: 자체 서버 API 기동

배포 실행에는 Go·Bun·Node·별도 프론트 서버가 필요 없다. full은 빌드된 정적 프런트 자산과 브라우저 라이선스 고지를 포함하며 tui는 서버와 WebUI를 포함하지 않는다.

## 모델과 코딩 작업

- Ollama·vLLM·SGLang·LM Studio·LiteLLM·OpenRouter·OpenAI·Claude·Gemini 연결
- 스트리밍 답변·Thinking, native 도구 호출과 모델 기능에 따른 이미지 입력
- 파일 탐색·생성·정확한 부분 수정·셸 실행과 장기 명령 조회·대기·취소
- 작업 중 방향 수정·후속 요청 예약과 중단된 대화 재개
- 도구 실행의 승인 정책·실행 직전 대상 확인과 작업별 승인 결정

## 팀과 스킬

`!team`은 현재 대화의 팀 모드를 전환한다. 독립 세션 두 개로 시작하고 필요한 동료를 초대·복구해 동시에 최대 여덟 개가 협업한다. 공용 대화·담당 경로·도구 진행을 공유하며 개인 Thinking은 공개 스트림에 내보내지 않는다.

`!skills`는 스킬 목록·출처·설명·본문을 조회한다. `#스킬명`으로 지침을 지정하거나 에이전트가 필요한 스킬을 탐색한다. 프로젝트 `.agents/skills/`, 개인 `~/.meca/skills/`, 공유 `~/.agents/skills/`를 순서대로 사용한다. 기본 스킬은 `create-skill`·`ui-design`·`backend-design`이며 기존 사용자 파일을 보존한다.

## WebUI와 서버

WebUI는 작업공간별 대화 탐색·생성·삭제, 독립된 다중 세션 실행, 스트리밍 응답·도구 활동·승인, 모델·실행·일반 설정을 제공한다. 문법 강조 편집과 명령·스킬 후보를 지원하며 좁은 창에서는 파일·스킬 목록과 본문을 전환한다. 기본 라이트와 다크·시스템 테마, 흑백 중심의 표면과 의미별 상태색, 축소 모션을 사용한다.

서버는 작업 수명을 소유하며 관찰 연결 종료나 스트림 표시 일시정지로 작업을 취소하지 않는다. `--concurrency`는 프로세스 전체의 동시 사용자 작업 수를 지정한다. 세션별 쓰기 소유권, 접수 키 중복 방지, 승인·취소·방향 수정·후속 예약과 커서 재구독을 제공한다. 스트림은 5초 무출력 후 1초마다 heartbeat를 보낸다.

서버 토큰 인증·TLS·허용 Host/Origin을 지원하며 WebUI는 HttpOnly 쿠키를 사용한다. 리스너가 준비되면 현재 서버 토큰을 stdout에 한 번 표시하며 연결 화면에 이 값을 입력한다. 운영 로그·이벤트·URL에는 토큰을 포함하지 않는다. API는 MECA 자체 계약을 사용한다. 모델 연결용 OpenAI·OpenAI 호환 Provider는 별도로 지원한다.

## 저장과 보안

대화·도구 결과·사용량은 로컬 SQLite에 보존한다. 큰 원문은 참조·압축하고 세션·작업별 데이터를 구분한다. 지원하지 않는 저장 형식은 원본을 보존한 채 오류를 반환한다.

API Key는 암호화해 저장하고 OS 자격 증명 저장소와 내부 주기적 키 교체를 사용한다. 비밀값은 프롬프트·로그·저장 이벤트에서 제외한다. 실행 종료·취소 시 조회·스트림·타이머·임시 파일·DB와 프로세스 자원을 정리하며 서버 입력·출력·캐시·저장소에는 개별 자원 예산을 적용한다.

[설치와 실행](https://github.com/keonho-kim/m.e.ca#설치) · [Provider](https://github.com/keonho-kim/m.e.ca#지원-provider) · [서버 API](https://github.com/keonho-kim/m.e.ca/blob/main/docs/server.md) · [설정과 보관](https://github.com/keonho-kim/m.e.ca/blob/main/docs/configuration.md)
