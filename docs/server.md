# 서버 API

[← README](../README.md) · [제품 명세](../SYSTEM.md)

```sh
meca --server                         # SSE, 127.0.0.1:8787
meca /path/to/project --server --protocol socket
meca /path/to/project --server --protocol grpc --listen 127.0.0.1:8788
meca --server --session session_XXXXXXXXXXXXXXXXXXXXXXXXXX
meca --server --protocol grpc --openai   # 같은 포트에 호환 HTTP API 추가
```

`socket`은 WebSocket이다. 선택한 native 전송 방식 하나를 열고 `--openai`로 호환 HTTP 경로를 함께 제공한다. 기본 `meca`는 TUI, `--print`·`--mode json`은 일회성 CLI이며 서버와 조합하지 않는다. 서버에는 TTY가 필요 없다. 데스크톱 앱·stdio·ACP·공개 Go 엔진 SDK는 제공하지 않는다.

| 시작 옵션 | 기본값·의미 |
| --- | --- |
| `--server` | 현재 프로젝트를 서비스 |
| `--protocol sse\|socket\|grpc` | 기본 `sse`; native 전송 하나 선택 |
| `--listen IP:port` | `127.0.0.1:8787` |
| `--session ID` | 현재 프로젝트의 지정 대화 선택 |
| `--token-file PATH` | 기본 `~/.meca/server.token` |
| `--tls-cert PATH`, `--tls-key PATH` | 함께 지정. loopback 외에는 필수 |
| `--allow-host authority` | 정확한 HTTP Host 추가. 반복 가능 |
| `--allow-origin scheme://host:port` | 브라우저 Origin 추가. 반복 가능 |
| `--openai` | 같은 포트에 텍스트 호환 HTTP API 활성화 |

프로세스는 시작할 때 지정한 프로젝트 하나를 소유한다. API는 해당 프로젝트의 대화를 조회·생성·열기·닫기 할 수 있으며, 동시에 선택한 대화 하나와 사용자 요청 하나만 실행한다. 팀 모드는 기존의 독립 에이전트와 내부 병렬 실행을 사용한다. 다른 TUI나 서버가 소유한 대화는 기존 세션 잠금으로 열기를 거부한다.

## 인증과 네트워크

기본 토큰 파일은 `~/.meca/server.token`이다. 처음 실행할 때 원자적으로 생성하며 현재 사용자 소유의 private 일반 파일만 허용한다. `--token-file PATH`로 다른 파일을 지정할 수 있다. 상위 디렉터리는 미리 만들어야 하며 파일에는 32~256바이트의 공백 없는 ASCII 토큰을 넣는다. 시작 시 한 번 읽은 토큰을 인증과 비밀값 마스킹에 함께 사용한다. 파일을 교체한 뒤에는 서버를 다시 시작해야 적용된다. 토큰 내용은 시작 메시지에 출력하지 않는다.

HTTP·SSE·RPC는 매 요청의 `Authorization: Bearer <token>` 헤더를 사용한다. URL·쿠키 인증은 지원하지 않는다. 브라우저 SSE 클라이언트는 Authorization을 넣을 수 있는 fetch 스트림을 사용한다. 기본 EventSource는 사용자 정의 Authorization 헤더를 제공하지 않는다.

WebSocket은 브라우저 API의 헤더 제약 때문에 연결 후 첫 텍스트 프레임으로 인증한다.

```json
{"authenticate":"<token>"}
```

5초 내 인증하지 않거나 인증 전에 명령을 보내면 연결을 닫는다. 성공 시 `{"reply":{"id":"authenticate","ok":{}}}`를 받는다. 연결당 인증은 한 번이며 구독·제어 프레임은 그 뒤에 보낸다.

기본 Host 허용 목록은 실제 listen 주소와, loopback 바인딩에서는 같은 포트의 localhost·127.0.0.1·[::1]이다. 프록시의 외부 Host는 `--allow-host meca.example`처럼 정확한 authority로 추가한다. 기본 포트를 생략하는 Host에는 포트 없이, 8787 등 명시적 포트를 보내는 Host에는 `meca.example:8787`을 사용한다. 다른 브라우저 origin은 `--allow-origin https://frontend.example`처럼 추가한다. 옵션은 반복할 수 있으며 와일드카드는 허용하지 않는다. 동일 origin은 허용한다. 명시적으로 허용한 origin만 CORS 응답을 받는다.

```sh
meca --server --protocol grpc --listen 0.0.0.0:8787 \
  --tls-cert /private/server.crt --tls-key /private/server.key \
  --allow-host meca.example:8787 --allow-origin https://frontend.example
```

Loopback 외 바인딩은 인증서와 개인키가 모두 필요하다. TLS는 Go 표준 라이브러리를 사용하며 최소 TLS 1.2다. loopback에서는 평문 HTTP를 사용할 수 있고 grpc 모드에는 평문 HTTP/2도 제공한다. HTTPS 역방향 프록시에서 loopback으로 연결하는 배치도 가능하다. 인증서 발급·CA 설치는 수행하지 않는다. 모델 API Key는 암호화해 보관하며 서버 인증 토큰과 구분한다. [암호화와 복구](configuration.md#api-key-암호화와-복구)를 따른다.

## 공통 명령

같은 정규화 프로젝트를 서비스하는 서버는 한 프로세스만 허용한다. 포트나 protocol이 달라도 두 번째 시작은 소유권 오류로 거절한다. `listSessions`의 `nextCursor`를 다음 요청의 `cursor`로 전달하고 빈 커서에서 끝낸다. 커서는 프로젝트와 정렬 경계에 묶이며, 새로 갱신된 대화는 목록을 처음부터 새로 조회해 확인한다.

외부 계약의 원본은 [api/meca/v1](../api/meca/v1/service.proto)의 Protobuf다. JSON은 Protobuf JSON 형식이며 필드명은 lowerCamelCase를 사용한다. `int64`·`uint64`는 JSON 문자열, `bytes`는 base64다. 요청은 Command의 operation 하나, 응답은 Reply의 result 하나를 담는다. 알 수 없는 필드와 중복 operation은 거부한다.

| operation | 입력과 결과 |
| --- | --- |
| `listSkills` | `cursor`, `limit` → `skills:{items,nextCursor,diagnostics,diagnosticsTruncated}`. 기본·최대 32건 |
| `getSkill` | `id` → `skill:{skill,revision,body}`. 본문 미리보기이며 모델 작업을 시작하지 않음 |
| `readSkillFile` | `id`, 스킬 루트 기준 `path`, `offset`, `limit`, 선택적 `revision` → `skillFile:{revision,text,mediaType,offset,nextOffset,totalBytes,binary}`. offset이 0보다 크면 첫 페이지의 revision 필수 |
| `listSessions` | 선택적 `cursor`, `limit` → 현재 프로젝트의 목록 페이지와 `nextCursor`. 기본 50건, 최대 100건. 미지원 DB는 `unavailable`에 사유를 표시 |
| `selectSession` | `{"create":true}` 또는 `{"sessionId":"…"}` → 상태 |
| `closeSession` | `{}` → 자원 해제. 작업 중에는 거부 |
| `state` | `{}` → 세션·작업·모델·설정·대기 승인·초안·이벤트 커서 |
| `history` | `before`, `limit` → 시간순 페이지와 다음 조회의 `before` |
| `detail` | `tool:{sessionId,sequence}`, `approvalId`, `historySequence` 중 하나와 `offset`, `limit` → JSON 바이트 페이지 |
| `setModel` | `provider`, `endpoint`, `model`, 선택적 `credential`, `thinking`, `reasoningEffort`, `remember` → 상태 |
| `setSettings` | `permission` 또는 `team` 하나 → 상태 |
| `start` | `sessionId`, `key`, `text` → 즉시 작업 ID와 상태 |
| `taskStatus` | `taskId` → 상태 |
| `approve` | `taskId`, `approvalId`, `allow` → 처리 결과 |
| `steer` | `taskId`, `text` → 다음 안전한 판단 경계에 전달 |
| `cancel` | `taskId` → 취소 요청. 실제 종료는 `request_finished`로 확인 |

`setModel`은 기존 Provider 연결 준비·검증·저장과 OS 자격 증명 저장을 사용한다. 같은 연결에서 credential을 생략하면 저장된 참조를 사용한다. `remember`는 새 세션의 기본 연결도 저장한다. 기본값 저장만 실패하면 세션의 모델은 유지하고 부분 완료를 오류 메시지로 알린다. 저장된 모델은 첫 요청의 접수 전에 연결을 준비하므로 Provider나 키 저장소가 사용 불가능해도 대화 이력을 열고 모델을 다시 설정할 수 있다. 설정 변경은 실행 중에 허용하지 않는다. permission은 `default` 또는 `all_mighty`이며 팀 모드는 선택한 세션에 저장한다.

세션·설정 변경 중에는 다른 변경·새 요청·history/detail 조회에 conflict를 반환한다. state·이벤트 구독·taskStatus는 준비 I/O를 기다리지 않는다. 세션 교체에서 기존 자원 정리가 실패하면 선택 세션은 비어 있는 상태로 확정하며, 커서도 바뀐다. 오류 응답 뒤에는 최신 snapshot/state를 확인한다. 이미 시작한 이력·상세 조회는 해당 세션이 닫히면 취소된다.

`id`는 응답 상관관계용이다. 실제 실행 중복 방지는 `start.key`가 담당한다. 호출자가 요청마다 안정된 키를 만들고 응답을 잃었을 때 같은 세션·키·본문으로 재전송한다. 동일 키의 다른 본문은 conflict이며 기존 작업을 바꾸지 않는다. key는 8~128바이트다. `~/.meca/server-requests/<project-scope>/requests.db`는 본문 대신 해시와 작업 식별·상태만 저장한다. project-scope는 정규화 프로젝트 경로의 SHA-256이며 프로젝트 간 접수·조회가 분리된다. 실행 전에 기록하고 완료 후 상태를 확정한다. 비정상 종료되어 접수만 남은 작업은 interrupted로 조회되며 자동 재실행하지 않는다. 다시 수행하려면 실제 부분 효과를 확인한 뒤 새 키로 요청한다.

Task는 `code`와 `persistence`도 제공한다. `pending`은 현재 프로세스가 접수·실행을 소유한 상태, `confirmed`는 종료 상태의 저장을 확인한 상태, `uncertain`은 종료 저장이나 효과를 확인할 수 없는 상태다. 완료 쓰기가 실패하면 취소와 독립된 읽기로 같은 결과가 저장되었는지 확인한다. 확인하지 못하면 `code=completion_not_saved`, `persistence=uncertain`을 반환하고 해당 프로세스의 새 작업 접수를 막는다. 실행·쓰기 자체를 자동 재시도하지 않는다. 프로세스가 소유하지 않는 미완료 접수는 `interrupted`·`uncertain`으로 조회한다.

전송 어댑터는 [중앙 오류 분류](diagnostics.md#중앙-소유권과-오류)를 공유한다. 용량·저장 오류를 취소로 덮지 않으며 observer deadline은 HTTP 504, 용량 초과는 429다. 이미 응답을 쓰기 시작했다면 두 번째 오류 JSON을 이어 쓰지 않는다. 소켓·SSE의 전달 실패와 출력 내용의 재작성은 다른 오류다.

접수 조회 뒤 다른 호출이 먼저 같은 키를 확정한 경우에도 원래 작업을 반환한다. 현재 프로세스가 접수 확정 후 실행 인계를 진행 중이면 accepted, 실행 중이면 running/canceling으로 조회되며, 주인 없는 이전 프로세스의 접수만 interrupted로 복원한다.

승인은 서버가 발급한 일회성 approval ID와 task ID에 묶는다. 재사용·다른 작업·이미 끝난 승인에는 conflict를 반환한다. 승인 뒤에도 기존 실행기의 대상·버전·사용자 지침 재검사가 적용된다. 승인 이벤트의 preview가 잘렸으면 `detail.approvalId`로 전체 preview와 명령 판정 근거를 확인한다. 거절·취소·실행기 마감은 대기 중인 승인을 끝낸다.

명령 오류는 공통 `Reply.error:{code,message}`다. HTTP 제어는 이에 맞는 4xx/5xx 상태를 함께 반환한다. RPC의 애플리케이션 오류도 같은 Reply에 담으며 인증·잘못된 프레이밍·스트림 종료 등의 전송 오류는 해당 RPC 상태를 사용한다. 클라이언트는 둘 다 확인해야 한다.

ID·모델 설정 형식 오류는 invalid_argument, 없는 세션·원문은 not_found다. 취소된 제어·조회는 canceled이며 HTTP에서는 409로 반환한다. 세션 변경으로 조회가 취소되면 최신 상태를 확인한 뒤 필요한 원문만 다시 요청한다.

스킬 조회는 세션 선택 전에도 가능하며 서버의 고정 프로젝트·실행 계정을 기준으로 한다. `getSkill`은 탐색 ID와 엔진에서 경로로 읽어 등록한 ID를 받고 임의 경로 입력은 받지 않는다. 참고 파일은 검증한 스킬 루트 안에서만 읽는다. 목록 페이지는 현재 파일을 다시 조회하므로 파일 변경 뒤에는 빈 cursor로 목록을 갱신한다. `start` 입력은 기존 sessionId·key·text이며 스킬은 현재 입력의 #이름 또는 에이전트의 읽기 도구로 지정한다. 형식과 한도는 [스킬 문서](skills.md)가 소유한다.

## SSE

- `POST /v1/command`: application/json Command → Reply
- `GET /v1/events`: SSE Notification. 선택적 Last-Event-ID 또는 cursor query로 재개한다. 둘 다 있으면 헤더가 우선한다.

```sh
curl --no-buffer -H "Authorization: Bearer $(cat ~/.meca/server.token)" \
  http://127.0.0.1:8787/v1/events
```

POST로 보낼 Command 예시:

```json
{"id":"new-session","selectSession":{"create":true}}
```

```json
{"id":"submit","start":{"sessionId":"<returned-session-id>","key":"<stable-request-key>","text":"README를 검토해줘"}}
```

SSE의 event는 snapshot 또는 event, id는 해당 Notification의 커서다. data 한 줄은 Notification JSON이다. 작업이 없어도 구독은 유지하며 [공통 heartbeat](#스트리밍과-heartbeat)로 유휴 연결을 확인한다.

## WebSocket

`/v1/socket`에 ws 또는 wss로 연결한다. 인증 뒤 명령과 구독을 같은 소켓으로 주고받는다. 명령에는 1~128바이트의 id가 필요하다. 응답과 이벤트는 섞일 수 있으므로 Notification의 oneof와 Reply.id로 구분한다.

```json
{"command":{"id":"read-state","state":{}}}
```

```json
{"watch":{"cursor":"<last-cursor-or-empty>"}}
```

```json
{"unwatch":{}}
```

한 연결에는 구독 하나만 둔다. watch를 다시 보내면 이전 구독을 정리하고 교체한다. 소켓을 끊거나 unwatch해도 실행 중인 작업을 취소하지 않는다.

## gRPC·gRPC-Web·Connect

grpc 모드는 [Connect-Go](https://connectrpc.com/docs/go/grpc-compatibility/) 한 구현으로 세 wire protocol을 받는다. native gRPC에는 HTTP/2가 필요하며 TLS와 loopback 평문 HTTP/2를 제공한다. 프록시에서도 HTTP/2와 스트리밍 응답·trailer가 유지되어야 한다.

- `/meca.v1.MecaService/Call`: Command → Reply
- `/meca.v1.MecaService/WatchEvents`: WatchRequest → server-streaming Notification

브라우저는 gRPC-Web 또는 Connect 클라이언트로 같은 schema를 사용한다. 실행 명령과 이벤트 관찰을 분리했으므로 browser client/bidirectional streaming에 의존하지 않는다. 생성된 Go 코드가 저장소에 포함되어 일반 go build·소스 설치에는 protoc가 필요 없다. 클라이언트는 `.proto`로 자신의 언어용 코드를 생성할 수 있다.

## 스트리밍과 heartbeat

모든 서버 스트림은 마지막 애플리케이션 출력 후 5초가 지나면 heartbeat를 보내고, 출력이 재개될 때까지 1초마다 반복한다. 데이터가 나오면 다시 5초를 기다린다. native·OpenAI 호환 SSE는 `: heartbeat` 주석과 빈 줄을 flush한다. WebSocket·gRPC·gRPC-Web·Connect는 `event.kind="heartbeat"`인 Notification을 보낸다. heartbeat에는 cursor가 없으며 이력·재생 로그·사용량에 넣지 않는다. 클라이언트는 heartbeat로 마지막 커서를 덮어쓰지 않는다. 구독 취소·완료·쓰기 실패 시 해당 관찰자의 대기와 타이머도 끝난다.

SSE 작성기는 이미 도착한 연속 델타의 flush를 묶는다. 첫 애플리케이션 출력과 첫 델타·제어 이벤트·완료·heartbeat는 즉시 전송하고, 나머지는 받은 묶음의 끝이나 누적 약 64KiB에서 flush한다. 별도의 대기 타이머·전송 큐를 만들지 않으며 이벤트 순서·커서·sequence·본문은 유지한다.

## 이벤트·재접속·수명

새 사건은 `occurredAt`과 가능한 `errorCode`를 포함한다. 순서는 시각 대신 커서를 따른다. `skill_loaded`의 `skill:{id,name,source,revision}`은 본문 읽기 성공을 나타내며 모델의 지침 준수나 작업 완료를 뜻하지 않는다. 본문은 이벤트에 반복해서 넣지 않는다. 재접속 시 기존 history/detail의 같은 메타데이터로 읽기 기록을 복원한다. 별도 활성 스킬 snapshot이나 활성화 명령은 없다. [운영 로그](diagnostics.md)는 동일한 작업 식별자로 장애 추적 정보를 제공한다.

커서는 프로세스 epoch와 단조 증가 sequence의 조합이다. 세션 선택·닫기·호환 작업의 대화 교체에서도 같은 커서를 이어간다. 재시작하면 새 epoch를 사용한다. 작업 ID는 내부 에이전트의 run ID와 별개다. 이벤트에는 실제 작업의 session·task·agent·run·generation 식별자를 제공한다. snapshot의 session·model·team은 native 선택, execution은 현재 작업의 session·model·team이다. 작업이 없으면 execution은 없다. 마지막 task가 호환 대화에 속하면 task.sessionId는 선택한 session.id와 다를 수 있다. assistant_delta는 추가할 본문·Thinking, draft_reset은 해당 발신자의 초안을 교체하라는 뜻이다. Provider 재시도도 reset을 발생시킨다. progress·response·thinking은 완성된 해당 내용이며 델타로 다시 덧붙이지 않는다. 도구 이벤트는 원본 참조를, usage와 최종 checkpoint는 실제 수신 사용량과 통계를 제공한다. request_finished에는 최종 status와 error가 있다.

첫 구독이나 만료된/다른 epoch 커서에는 현재 상태의 snapshot을 보낸다. 보관 중인 커서라면 그 이후 이벤트만 순서대로 전송한다. snapshot을 받으면 그 커서까지 반영된 상태로 바꾸고 저장 이력은 history로 별도 조회한다. 과거 이력은 페이지별 시간순이며 다음 요청에는 응답의 before를 전달하고 빈 페이지에서 끝낸다. history/detail 조회는 모델·도구를 실행하지 않는다.

긴 현재 초안과 history 표시 필드는 잘릴 수 있으며 truncated로 알린다. 완료된 본문은 historySequence, 도구 결과는 tool reference로 detail을 읽는다. detail의 data를 offset 순서로 합치면 JSON 원문이 된다. 공개 표시 필드만 반환하며 Provider native continuation·이미지 원본·capture·자격 증명은 반환하지 않는다.

상세 JSON은 API가 정의한 문서이며 내부 Go·DB 구조체 형식과 독립적이다. 필드명은 lowerCamelCase이고, 본문은 다음 형태 중 하나다.

| kind | 공개 본문 |
| --- | --- |
| `tool` | `event`: 완전한 표시 이벤트 |
| `message` | `sequence`, `message:{id,role,content,incomplete}` |
| `event` | `sequence`, `event`: 완전한 표시 이벤트 |
| `approval` | `approval:{preview,command}`. command는 directory와 findings의 command·reasons·targets |

event는 kind·agent·final·runId·text·thinking·callId·approvalSource·changePreview와, 존재하는 call·output·operation·reference·usage·statistics·team을 포함한다. call은 id·name·arguments, reference는 sessionId·sequence다. arguments와 output은 원래 JSON 값이며 byte 필드의 base64 문자열로 다시 감싸지 않는다. operation에는 실행 식별·상태·명령·결과·실패·프로세스·파일 변경의 공개 관측만 담는다. checkpoint·provider/model continuation·캡처·실행 판정용 대상 목록은 내보내지 않는다. sequence·토큰·시간·팀 순서와 revision의 64비트 정수는 JSON 문자열이다. 사용자 도구 결과 안의 숫자는 원래 JSON 표현을 유지한다.

```json
{"kind":"tool","event":{"kind":"tool_succeeded","call":{"id":"write","name":"write_file","arguments":{"path":"result.txt","content":"written"}},"output":{"ok":true}}}
```

같은 상세의 연속 페이지는 선택 세션에 보관한 최근 문서 한 건을 재사용한다. 공개 DTO를 증분 JSON으로 인코딩하고 문자열을 조각별로 마스킹하여 완성 JSON 전체를 메모리에 만들지 않는다. 64KiB 이하는 메모리, 큰 문서는 private 임시 파일에 두고 파일 경로를 즉시 제거한다. 다른 원문이나 마스킹 revision으로 바뀌면 이전 문서를 먼저 해제한다. 세션 교체·닫기·프로세스 종료에서도 자원을 해제한다. 끝난 승인은 보관 여부와 무관하게 조회를 거부한다. 임시 보관을 사용할 수 없으면 현재 페이지만 수집해 반환하고 다음 조회에서 다시 준비하며 `detail_cache_bypassed`로 진단한다. 파일 정리 실패는 오류로 반환한다. 첫 조회의 저장 원문 복원과 이스케이프된 JSON 값 해석 비용까지 페이지 크기로 제한되는 것은 아니다.

| 한도 | 소유 범위 |
| --- | --- |
| 제어 요청 17MiB, 요청 본문 16MiB | 전송 수용·사용자 요청 |
| 세션 목록 1~100건, 기본 50건 | `nextCursor`로 다음 페이지 조회 |
| 세션 목록 snapshot 2분, 최대 8건·합계 8MiB | ID·정렬 시각만 보관하며 한 snapshot은 최대 2MiB |
| 입력 예약 합계 32MiB | HTTP·RPC·WebSocket 본문 파싱. 일반 입력은 실제 읽는 크기에 따라 32KiB 단위로 예약. 압축 입력은 해제 전 일시적으로 17MiB를 예약하고 RPC 해석 후 실제 명령 크기로 조정. 초과는 실행 접수 전에 거절 |
| 이력 페이지 복원 32MiB, 상세·완료 원문 복원 512MiB | 원문 읽기는 동시 작업 1개. 완료 JSON·SSE의 출력 또는 상세 문서의 마스킹·임시 보관이 끝날 때까지 공유 메모리 예약 유지 |
| 이력 1~100건, 기본 32건 | history 페이지 |
| 상세 1~65536바이트, 기본 65536 | detail 응답 바이트 |
| 최근 상세 1건, 메모리 보관 최대 64KiB | 큰 상세는 경로 없는 임시 파일에서 페이지 단위로 조회 |
| 2048건 및 8MiB | 이벤트 보관. 먼저 도달한 한도로 만료 |
| 이벤트 본문·Thinking 각 16KiB | 단일 표시 이벤트. 큰 스트림 델타는 분할 |
| 발신자별 본문·Thinking 각 128KiB, 최대 16개 발신자 | 활성 초안 snapshot. 전체 완료 본문은 DB 조회 |
| 16건 및 합계 1MiB | 실행 중 방향 수정 대기열 |
| native 작업 오류 4KiB | SQLite 행에 종료 상태와 저장하며 읽기 전 필드 크기를 검사 |
| 호환 API 결과 메타데이터 64KiB | 오류·종료 사유와 원본 위치. 본문은 기존 DB에 보관 |
| TCP 연결 최대 64개 | HTTP 해석·TLS 처리 전에 적용하며 유휴 연결도 포함. 초과 연결은 즉시 닫음 |
| 동시 HTTP 요청 32개, 그중 이벤트/소켓 최대 16개 | HTTP/2도 연결별 최대 32개 스트림을 알림. 미인증 WebSocket 포함 |
| 헤더 5초·입력 본문/소켓 프레임 읽기 30초·쓰기 15초 | 느린 연결 점유 제한. 입력을 다 읽으면 읽기 마감 해제. WebSocket 인증은 5초이며 완료한 명령은 다음 프레임 대기 전에 해제 |

세션 목록의 첫 페이지는 전체 순서를 확인하고 다음 페이지는 저장한 ID·시각에 해당하는 DB만 읽는다. 순서 참조는 현재 프로세스와 프로젝트에 묶인다. 만료·재시작 후에는 빈 커서로 다시 조회한다. 읽는 동안 갱신되거나 사라진 세션은 건너뛰며 새 목록에서 확인한다. 보관 수용량 초과는 `resource_exhausted`로 반환한다.

원문 복원 한도는 선택한 공개 payload의 읽기 작업량이며 모델 실행의 문맥·출력 예산과 구분한다. 한도 초과는 원문을 자르거나 삭제하지 않고 `resource_exhausted`로 반환한다. 세션 export에는 이 API 조회 한도를 적용하지 않는다. 느린 원문 조회와 조회 슬롯 대기 중에도 SSE heartbeat를 유지한다.

프로젝트의 대화 할당은 4096개, 전체 서버 접수 기록은 100000개까지 허용한다. 세션 DB는 512MiB 페이지 상한을 적용하고, 이미 더 큰 DB는 기존 페이지를 보존하며 추가 성장을 막는다. WAL은 checkpoint 정책으로 정리하지만 활성 reader 동안에는 별도 공간을 사용할 수 있다. 새 저장 할당 전에는 파일시스템 여유 공간 1GiB를 확인한다. 게시 전에 확정적으로 실패한 예약은 한 번 반환한다. 게시·커밋을 시도했거나 중단 후 결과를 확인할 수 없는 예약은 유지한다. 한도의 90%부터 storage_capacity_low에 count·limit을 기록한다. 한도에 도달해도 기존 조회·중복 확인·작업 완료 기록은 유지하고 새 할당을 거절한다. 기존 DB·접수 기록을 자동 삭제하거나 불확실한 counter를 재설정하지 않는다. 이 검사는 다른 프로그램의 동시 쓰기까지 예약하는 OS 디스크 quota는 아니다.

재조회용 정제 명령 로그·결과는 실행기당 256MiB·256건까지 보관한다. 상한에 도달하면 원본을 DB에 확정한 오래된 결과부터 회수한다. 미확정 결과는 지우지 않으며 여유가 없으면 오류를 반환한다. 원자적 교체에는 현재 쓰는 한 파일 크기만큼 추가 공간을 사용할 수 있다. 활성 원문 캡처는 별도 스트림별 수집 한도를 따르며 새 블록으로 커질 때에도 여유 공간 1GiB를 검사한다. 임시 파일의 만료는 DB 원본과 export를 삭제하지 않는다.

실행 이벤트는 네트워크에 직접 쓰지 않는다. 느린 관찰자는 독립적으로 읽고 쓰기 실패는 해당 연결만 끝낸다. 연결이 끊겨도 수락한 요청은 서버 프로세스의 context로 계속된다. 명시적인 cancel이나 정상 서버 종료만 작업을 취소한다. 종료는 새 요청을 막고 모델·도구·승인 대기를 취소·합류한 뒤 executor·client·DB·잠금을 해제한다. 강제 종료 후에는 기존 복구 규칙을 사용하며 불확실한 도구 효과를 재생하지 않는다.

## OpenAI 호환 API

`--server --openai`로 활성화한다. `--protocol`의 native 전송 선택과 독립적이며 같은 포트·TLS·토큰·Host·Origin·연결 한도를 공유한다. WebSocket 모드에서도 이 HTTP 경로에는 Bearer 헤더가 필요하다. 기본 실행에는 호환 경로를 열지 않는다.

| 메서드·경로 | 지원 범위 |
| --- | --- |
| `GET /v1/models` · `/v1/models/meca` | 가상 모델 `meca` 하나 |
| `POST /v1/chat/completions` | 텍스트 messages, 일반 JSON 또는 SSE, `n=1`, `stream_options.include_usage` |
| `POST /v1/responses` | 텍스트 input·instructions, JSON·SSE, `background`, `store`, `previous_response_id` |
| `GET /v1/responses/{id}` | 보관한 응답의 현재 상태와 완료 원문 |
| `POST /v1/responses/{id}/cancel` | background 작업 취소. 이미 끝난 작업은 같은 최종 결과 |

이는 MECA 작업 실행기를 위한 텍스트 호환 범위다. 모델 ID `meca`는 운영자가 선택하거나 기본값으로 저장한 실제 모델·Thinking·추론 설정을 사용한다. 요청별 upstream 모델 교체나 별도 실행 루프를 만들지 않는다. 첫 요청 전에 TUI에서 모델을 기본값으로 저장하거나 native API로 세션과 모델을 선택한다.

Chat은 `system`·`developer`·`user`·`assistant`의 문자열 content 또는 `type:"text"` parts를 받는다. Responses는 문자열 input 또는 같은 역할의 message 배열과 `input_text` parts를 받으며 assistant의 `output_text` parts도 허용한다. 마지막 메시지는 비어 있지 않은 user 텍스트여야 한다. 메시지 순서와 사용자·assistant 문맥을 보존한다. 클라이언트 system·developer 텍스트는 역할을 표시한 낮은 신뢰도의 문맥 메모로 전달하며 MECA의 운영 지침·실행·승인 정책을 교체하지 않는다. 외부 assistant 메시지를 도구 실행 사실이나 승인으로 가져오지 않는다. 팀의 개인 실행에도 같은 입력 문맥을 전달한다.

호환 JSON 본문은 17MiB, 전체 텍스트는 16MiB, 메시지는 4096개, 배열 content parts는 요청 전체 16384개까지 받는다. 입력은 UTF-8·완전한 JSON을 검사하고 중복 필드·미지원 필드를 거부한다. 알려진 Content-Length는 한 번 할당하며, 중첩 값은 원본 JSON 범위를 읽고 오류 경로는 실패할 때만 만든다. 요청 취소도 대량 메시지·parts 파싱 중 확인한다.

외부 `tools`·`tool_choice`·도구 결과, 이미지·오디오, Structured Outputs, sampling·출력 토큰 제어와 그 밖의 지원하지 않는 필드는 `400`과 `error.param`으로 거부한다. 입력 텍스트 합계는 16MiB, JSON은 17MiB, 메시지는 최대 4096개다. query와 stream resume·`starting_after`는 지원하지 않는다. 기능을 요구하는 전체 OpenAI 도구 호출 클라이언트를 그대로 연결하는 계약은 아니다.

```sh
meca --server --openai
```

클라이언트는 base URL `http://127.0.0.1:8787/v1/`, API key는 MECA 서버 토큰, model은 `meca`로 설정한다. 공식 Go SDK의 기본 사용 예시는 다음과 같다. `MECA_TOKEN`에는 토큰 파일을 읽은 값을 전달한다.

```go
client := openai.NewClient(
    option.WithBaseURL("http://127.0.0.1:8787/v1/"),
    option.WithAPIKey(os.Getenv("MECA_TOKEN")),
    option.WithMaxRetries(0),
)
response, err := client.Responses.New(ctx, responses.ResponseNewParams{
    Model: "meca",
    Input: responses.ResponseNewParamsInputUnion{OfString: openai.String("이 프로젝트를 살펴봐")},
})
```

새 Chat 요청과 previous ID가 없는 Responses 요청은 접수 예약 안에서 새 작업 대화를 만든다. native 선택 대화는 그대로 유지하고 작업 대화 한 자리만 재사용·교체한다. 기존 대화의 이력에 전체 messages를 다시 덧붙이지 않는다. 현재 모델과 팀 모드는 이어받으며 여전히 동시에 작업 하나만 수락한다. 실행 중인 다른 작업이 있으면 `409`다. `previous_response_id`는 보관한 최신 completed 응답만 받으며 후속 응답이나 native 작업으로 대화가 바뀌었다면 거부한다. 실패하거나 불확실한 후속 접수도 자동 분기·재실행하지 않는다.

Chat의 재전송된 과거 messages는 요청 소유 문맥으로 공유하며 새 작업 DB나 팀 개인 DB에 다시 적재하지 않는다. 마지막 사용자 입력과 실제 엔진 실행은 저장한다. 이 외부 문맥은 요청 전체에 유지되므로 Provider 문맥에 들어가는 크기로 보내야 하며, 엔진의 대화 압축은 저장된 실행 이력을 대상으로 한다. Responses는 이어 읽기와 재시작 복구를 위해 초기 외부 이력을 대화에 한 번 저장한다. 팀에서는 외부 이력을 공용 원문으로 보관하고 개인 DB는 출처를 검증한 참조만 저장한다. 외부 이력을 실제 MECA 실행이나 운영자 지침으로 승격하지 않는다. 서로 다른 요청을 이력의 공통 접두부로 자동 합치지 않는다. stateless 호출의 대화 수는 계속 증가하므로 긴 대화는 previous_response_id를 사용하고 위의 저장 할당 한도를 운영 용량으로 고려한다.

실행 식별자를 획득하기 전에 실패한 후속 접수는 실패 기록과 부모의 해당 head 점유 해제를 한 트랜잭션으로 확정한다. 실행을 시작했거나 결과가 불확실한 자식은 head를 유지하며 자동 재실행하지 않는다.

`Idempotency-Key`는 MECA의 선택적 확장이다. 프로젝트·API 종류·키를 전체 정규화 입력 해시에 세션 생성 전에 바인딩한다. 같은 입력의 재시도는 같은 응답과 작업을 반환하며 다른 입력은 `409`다. stream 선택은 응답 표현이므로 키의 입력 해시에 포함하지 않는다. 키가 없으면 매 요청이 새 작업이다. `X-Request-ID`는 응답 ID, `X-Meca-Task-ID`는 접수한 작업 ID다. 수락했거나 접수를 예약한 응답은 `X-Should-Retry: false`를 보낸다. 네트워크 실패만으로 효과를 판단할 수 없으므로 부작용이 있는 호출에는 안정된 키를 사용하고 클라이언트 자동 재시도를 끈다.

연결 종료는 관찰만 끝낸다. Responses의 `background:true`는 접수 상태를 즉시 돌려주며 GET으로 조회하고 cancel로 중지할 수 있다. `store:false`는 호환 GET·후속 연결을 차단한다. 작업 중복 방지용 메타데이터와 MECA의 기존 운영 이력 보관 정책은 유지한다. 이 구현의 background는 조회와 취소가 필요하므로 `store:true`를 요구한다. 재시작 뒤 진행 상태는 실패·중단으로 조회하며 작업을 자동 재생하지 않는다.

요청 DB의 형식은 **user_version 2**이며 native·Chat·Responses 접수를 함께 저장한다. 길이로 구분한 전체 정규화 입력의 해시와 native 작업 상태·오류 코드를 응답 한 행에 보관한다. 후속 head 예약과 자식 행 삽입은 같은 트랜잭션으로 확정한다. 예약·실행 경계·완료를 각각 동기화하고 종료된 상태의 재변경을 거부한다. native task ID는 response ID와 같은 digest를 사용하며 native 작업 조회도 이 행을 읽는다. 이전 SQLite 형식 또는 JSON 접수 파일이 있으면 서버 시작을 거부하고 원본을 보존한다. 자동 변환·무시·삭제·재생하지 않으며 운영자가 기존 부분 효과와 보존 자료를 확인해야 한다. 대화 DB도 현재 schema 28만 열 수 있다.

일반 OpenAI 클라이언트에는 MECA 승인 입력 채널이 없으므로 기존 비대화형 승인 차단을 사용한다. 승인이 필요하면 `approval_required`로 끝내며 대기 승인을 남기거나 권한을 올리지 않는다. native API로 시작한 작업의 대화형 승인은 기존과 같다.

호환 출력은 저장한 완성 assistant 원문을 사용한다. 각 공개 발화의 바깥 공백을 제거하고 발화 사이에 빈 줄을 넣는다. 이벤트 미리보기·draft 한도에 맞춰 완료 본문을 자르지 않는다. 팀은 최종 보고만 출력하고 보고자가 정해질 때까지 본문을 보류한다. Thinking·내부 팀 대화·도구 인자를 텍스트나 OpenAI tool_calls로 노출하지 않는다.

Chat SSE는 `chat.completion.chunk`와 `[DONE]`, Responses SSE는 created·in_progress·output_item·content_part·output_text·completed 등의 typed event를 사용한다. 성공한 델타 연결 결과는 일반 응답 본문과 같다. 단일 모드는 본문을 증분 전송하되, 이미 노출한 초안을 재시도가 바꾸거나 관찰자가 로그를 놓치면 `stream_rewrite`·`stream_gap` 오류로 관찰을 끝낸다. 성공 완료를 보내지 않으며 작업은 계속된다. 잘린 commit 미리보기로 뒤의 바이트를 확인할 수 없으면 남은 본문을 완료 원문까지 보류한다. 같은 키의 스트림 재요청도 원문을 기다려 새 스트림으로 반환하며 스트림 재개로 취급하지 않는다. SSE heartbeat는 위와 같은 주석이므로 델타·usage·event sequence에 섞이지 않는다.

일반 델타는 연결이 소유한 typed DTO와 SSE 프레임 버퍼를 재사용한다. Responses의 네 terminal 이벤트는 한 번 이스케이프한 완료 본문을 공유하며 각 이벤트의 전체 본문을 유지한다. 큰 본문은 프레임 버퍼에 다시 복사하지 않는다. 완료 대기에는 progress 이벤트를 복사하지 않고 작업 완료 신호를 사용하며, 공개 출력과 무관한 이벤트는 문자열 마스킹 전에 제외한다.

usage는 재시도·압축·팀을 포함한 실제 보고 토큰의 작업 합계다. 한 호출이라도 사용량을 보고하지 않으면 전체 usage는 `null`이며 0으로 추정하지 않는다. 정상 완료는 Chat `stop`·Responses `completed`, MECA 작업 예산 소진은 Chat `length`·Responses `incomplete`와 `meca_budget_exhausted`로 구분한다. 이는 요청별 OpenAI max_tokens 지원을 뜻하지 않는다. 실패·취소를 성공 완료로 바꾸지 않는다.

형식 참고: [공식 SDK](https://developers.openai.com/api/docs/libraries), [Chat 스트림](https://developers.openai.com/api/reference/resources/chat/subresources/completions/streaming-events), [Responses 스트림](https://developers.openai.com/api/docs/guides/streaming-responses). 위 지원 범위와 MECA의 상태·정책 차이는 이 문서가 소유한다.

## 개발과 생성

`internal/commands/server`는 native 선택·호환 작업 대화·요청·호환 접수·대화별 원본 읽기 대여 수명, `internal/serverapi`는 외부 투영·전송, `internal/serverapi/openai`는 호환 입력·출력·스트림, `internal/serverapi/eventstream`은 공통 heartbeat 시계와 SSE 작성기, `internal/serverapi/detail`은 공개 상세 JSON·마스킹·페이지 캐시, `internal/storage/serverrequests`는 접수와 원본 위치 기록을 소유한다. cmd/meca가 저장소·Provider·local tool·team 자원을 조립하고 상세 캐시를 세션 정리에 연결한다. 기존 TUI 이벤트 루프와 agent/run 정책을 서버로 이동하지 않는다.

공식 Go SDK 호환 검사는 `devtools/compatibility`에 두고 SDK 버전은 `devtools/go.mod`에 고정한다. 제품 모듈과 실행 파일에는 SDK·Azure·AWS 의존성을 추가하지 않는다. `go test ./internal/serverapi -run TestOpenAIOfficialSDK`가 모의 엔진의 로컬 서버를 띄워 SDK로 모델 목록·Chat·Responses·SSE·조회·취소를 확인한다. 실제 OpenAI 계정·API key·유료 호출을 사용하지 않는다. `-short`는 이 외부 클라이언트 실행을 생략한다.

재생성에는 `protoc`와 devtools 모듈에 고정된 두 Go generator를 사용한다. 일반 제품 빌드·설치에는 생성 코드가 포함되므로 이 도구들이 필요 없다. Python·OpenSSL 실행 파일도 요구하지 않는다.

```sh
go build -modfile=devtools/go.mod -o bin/devtools/ \
  google.golang.org/protobuf/cmd/protoc-gen-go \
  connectrpc.com/connect/v2/cmd/protoc-gen-connect-go
PATH="$PWD/bin/devtools:$PATH" protoc -I api \
  --go_out=. --go_opt=module=github.com/keonho-kim/meca \
  --connect-go_out=. --connect-go_opt=module=github.com/keonho-kim/meca \
  api/meca/v1/*.proto
```

`internal/serverapi/gen`의 생성물은 descriptor·메시지·RPC 계약을 함께 재생성해야 한다. 수동 분할은 generator 출력과 descriptor 관계를 손상시키므로 생성 파일에 350줄 제한을 적용하지 않는다. 수정 원본은 `.proto`이며 생성물은 직접 고치지 않는다.
