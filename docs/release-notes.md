# M.E.CA — Minimal Essentials Coding Agent

Go 실행 파일 하나로 코딩 작업을 수행한다. macOS·Linux의 amd64·arm64를 지원하며 WSL은 Linux 배포본을 사용한다.

## 대화와 완료 통계

- 팀 발언과 기여 선언은 작성자·요약·담당 경로·계약을 가진 본문으로 표시한다.
- 응답·팀 기여·고정 알림·도구·완료 측정은 배타적인 API 본문을 사용한다. 통계 행에는 고정 상태와 검증한 숫자만 전달하며 혼합된 본문은 수신 단계에서 거절한다.
- 완료 시 공개한 저장 위치에 이력이 도달한 뒤 실시간 초안·활동을 해제한다. 팀 사건과 스킬 읽기는 저장 위치·원본 참조로 합쳐 중복 표시를 방지한다.
- 같은 커서의 재전송과 다른 서버 epoch의 지연 프레임을 델타 결합·캐시 갱신·후속 예약 처리 전에 걸러낸다. 종료 작업의 늦은 본문을 새 초안으로 만들지 않는다.
- README·운영 문서·기본 스킬 자료는 현재 구현과 자산을 기준으로 안내한다.

외부 API 클라이언트는 현재 `api/meca/v1/*.proto`로 재생성해야 한다. `Event`의 표시 본문은 `message`·`work`·`notice`·`tool`·`completion` 중 하나이고, `HistoryItem`은 `message`·`event`·`completion` 중 하나다. 내장 WebUI는 서버와 함께 배포한다.

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
- `meca resume`: 모든 프로젝트의 저장 대화에서 선택
- `meca --print`: 최종 답변 출력
- `meca --mode json`: 순서 있는 JSONL 이벤트 출력
- `meca --session ID`: 저장 대화 재개
- `meca --session ID --export`: 원본 이력 JSONL 내보내기
- full의 `meca --server --ui`: 내장 WebUI 기동
- full의 `meca --server --protocol sse|socket|grpc`: 자체 서버 API 기동

배포 실행에는 Go·Bun·Node·별도 프론트 서버가 필요 없다. full은 빌드된 정적 프론트 자산과 브라우저 라이선스 고지를 내장한다.

## 모델과 코딩 작업

- Ollama·vLLM·SGLang·LM Studio·LiteLLM·OpenRouter·OpenAI·Claude·Gemini 연결
- 스트리밍 답변·Thinking, native 도구 호출과 비전 모델의 이미지 입력
- 파일 탐색·생성·부분 수정·셸 실행과 장기 명령 조회·대기·취소
- 작업 중 방향 수정·후속 요청 예약과 중단된 대화 재개
- 실행 직전 대상 확인·승인 정책과 외부 전송의 직접 승인
- 입력·출력 토큰·TPS·소요 시간과 OpenRouter의 보고 비용 표시

TUI는 `Alt+V`, macOS는 `Ctrl+V`로도 이미지를 첨부한다. macOS·Linux는 별도 클립보드 도구 없이 읽으며 WSL은 Windows 기본 PowerShell 브리지를 사용한다. 이미지당 64MiB·64M픽셀, 요청당 최대 8개·합계 128MiB를 적용한다.

## 팀과 스킬

`!team`은 현재 대화의 팀 모드를 전환한다. 독립 세션 두 개로 시작하고 필요한 동료를 초대·복구해 동시에 최대 여덟 개가 협업한다. 공용 대화·담당 경로·도구 진행을 공유하며 개인 Thinking은 공개 스트림에 내보내지 않는다.

`!skills`는 스킬 목록·출처·설명·본문을 조회한다. `#스킬명`으로 지침을 지정하거나 에이전트가 필요한 스킬을 탐색한다. 프로젝트 `.agents/skills/`, 개인 `~/.meca/skills/`, 공유 `~/.agents/skills/` 순서로 선택한다. 기본 스킬은 `create-skill`·`ui-design`·`backend-design`이며 설치된 사용자 수정본을 보존한다.

## WebUI

`meca --server --ui`로 내장 React 화면을 제공한다. 브라우저 파비콘과 사이드바는 흑백 ME/CA 모노그램, 접속·로딩 화면은 가로형 MECA 워드마크를 사용한다. 기본 라이트와 다크·시스템 테마, 중성 glass 표면과 의미별 상태색을 제공한다. 테마의 배경과 글자색은 함께 전환하며 축소 모션 설정을 따른다.

- 작업공간별 대화 탐색·생성·삭제와 독립된 다중 세션 실행
- 로컬 OS 폴더 선택창과 원격 서버 폴더 탐색·생성
- 스트리밍 답변·Thinking·병렬 도구·승인과 전송·방향 수정·후속 예약
- 이미지 드롭·붙여넣기·썸네일, 스킬 탐색과 명령 후보
- 최대 2MiB UTF-8 파일 편집, 문법 강조·충돌 확인과 Markdown Preview/Raw
- 파일·폴더 참조·이름 변경·삭제와 폴더로 드래그해 이동
- 전역 모델·언어·실행 설정과 대화별 팀 설정

도구 결과는 이름 밑줄의 아코디언에서 펼치고 요청 인자는 별도 불투명 모달로 연다. 답변·Thinking·코드·표는 채팅 폭 안에서 줄바꿈한다. 완료 통계는 응답 본문과 분리한 입력·출력 토큰·TPS·시간·비용을 표시하며 실패·중단은 고정된 UI 문구로 구분한다. 긴 이력은 가상화하고 상세 원문은 필요한 페이지부터 읽는다.

## 서버

서버는 접수한 작업의 수명을 소유한다. 관찰 연결이 종료돼도 작업을 계속 실행하며, `--concurrency`로 모든 작업공간의 동시 사용자 작업 수를 지정한다. 세션별 쓰기 소유권, 접수 키 중복 방지, 승인·취소·방향 수정과 커서 재구독을 제공한다. 스트림은 5초 무출력 후 1초마다 heartbeat를 보낸다.

서버 시작마다 새 인증 토큰을 발급하며 리스너가 준비되면 stdout에 한 번 표시한다. WebUI는 이 토큰으로 로그인하고 HttpOnly 쿠키를 사용한다. TLS·허용 Host/Origin 검사와 쿠키 수명에 연결된 관찰 취소를 지원한다. 로그·이벤트·URL에는 토큰을 포함하지 않는다.

## 저장과 보안

대화·도구 결과·사용량은 로컬 SQLite에 보존한다. 큰 원문은 참조·압축하고 세션·작업별 데이터를 구분한다. 지원하지 않는 저장 형식과 저장 용량 초과는 원본을 보존한 채 오류를 반환한다.

API Key는 암호화해 저장하고 OS 자격 증명 저장소와 내부 주기적 키 교체를 사용한다. 비밀값은 프롬프트·로그·저장 이벤트에서 제외한다. 실행 종료·취소 시 조회·스트림·타이머·임시 파일·DB와 프로세스 자원을 정리하며 입력·출력·캐시·저장소에는 개별 자원 예산을 적용한다.

[설치와 실행](https://github.com/keonho-kim/m.e.ca#설치) · [Provider](https://github.com/keonho-kim/m.e.ca#지원-provider) · [서버 API](https://github.com/keonho-kim/m.e.ca/blob/main/docs/server.md) · [설정과 보관](https://github.com/keonho-kim/m.e.ca/blob/main/docs/configuration.md)
