# M.E.CA — Minimal Essentials Coding Agent

Go 실행 파일 하나로 TUI·CLI·서버에서 코딩 작업을 수행한다. 배포 대상은 macOS·Linux의 amd64·arm64다.

## 제공 기능

- Ollama·vLLM·SGLang·LM Studio·LiteLLM·OpenRouter·OpenAI·Claude·Gemini 연결
- 스트리밍 답변·Thinking, native 도구 호출, 이미지 입력이 확인된 모델의 `view_image`
- 파일 생성·정확한 부분 수정·셸 실행과 장기 명령 조회·대기·취소
- 로컬 대화·실행 결과·사용량 저장, 압축·재개·JSONL 내보내기
- 작업 중 방향 수정과 후속 요청 예약, 최대 8개 독립 세션의 팀 협업
- `--server --protocol sse|socket|grpc`의 세션·모델·작업·승인·취소 API
- `--server --openai`의 텍스트 Chat Completions·Responses·SSE·background 조회·취소
- 서버 스트림의 5초 무출력 후 1초 heartbeat, 커서 재접속과 요청 키 중복 방지
- API Key 암호화 보관·OS 자격 증명 저장소 연동과 내부 주기적 키 교체

## 안정성과 자원 관리

- native 서버 요청과 OpenAI 호환 요청의 접수를 프로젝트별 SQLite에서 관리하고, 중복 요청과 응답 후속 연결을 트랜잭션으로 확정한다.
- 선택한 대화와 실행 중인 작업의 소유권을 구분하고, 취소·종료 시 모델 연결·조회 작업·명령 출력 자료를 정리한다. 모델 교체 결과가 불확실하면 기존 연결을 계속 사용하지 않는다.
- 세션 저장의 중복 식별자·인덱스와 반복 조회를 줄이고, 준비한 기록을 함께 저장한다. 팀 공개 문맥은 확정한 원본과 출처를 기준으로 복원한다.
- 서버 입력·조회·출력과 SQLite 연결·저장 용량을 제한하고, 긴 응답·셸 출력·TUI Markdown에서 불필요한 복사와 보관을 줄인다.

## 실행과 보관

기본 `meca`는 TUI다. `--print`는 최종 답변, `--mode json`은 순서 있는 이벤트를 출력한다. 서버는 TTY 없이 작업을 소유하며 관찰 연결 종료만으로 작업을 취소하지 않는다. native 서버 API의 직접 승인과 승인 채널이 없는 호환 API의 거절을 구분한다.

세션 DB는 schema 27을 사용한다. 미지원 DB는 목록에 사용 불가 사유를 표시하고 열기·내보내기를 거부하며 새 대화 시작은 유지한다. 서버 접수는 프로젝트별 SQLite에 저장한다. 이전 JSON 접수 파일이 있으면 원본을 보존하고 서버 시작을 거부한다. 자동 변환·삭제·재실행은 하지 않는다. 기본 프롬프트는 `~/.meca/prompts/single/`과 `team/`에 보관하며 install·upgrade는 배포 기본본으로 갱신한다. 대화 백업에는 사용자 DB와 팀 개인 DB, 자격 증명 백업에는 `credentials.db`와 OS 키 저장소를 함께 포함한다.

[설치와 실행](https://github.com/keonho-kim/m.e.ca#설치) · [Provider](https://github.com/keonho-kim/m.e.ca#지원-provider) · [서버 API](https://github.com/keonho-kim/m.e.ca/blob/main/docs/server.md) · [설정과 보관](https://github.com/keonho-kim/m.e.ca/blob/main/docs/configuration.md)
