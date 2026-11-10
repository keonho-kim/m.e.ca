# 서버 API

[← README](../README.md) · [제품 명세](../SYSTEM.md)

서버와 내장 WebUI는 `full` 배포본에서 제공한다. `tui`은 TUI·CLI만 포함하며 서버 옵션은 full 설치 안내와 함께 거절한다. `meca --help` 첫 줄로 종류를 확인한다. MECA 자체 API만 제공한다. 모델 연결용 Provider와 서버 외부 계약은 분리하며, 클라이언트는 아래의 세션·작업·승인·이벤트 계약을 사용한다.

## 기동

```sh
meca --server                         # SSE, 127.0.0.1:8787
meca --server --ui                    # 내장 React 작업공간 /ui/
meca --server --concurrency 4          # 서로 다른 세션의 작업을 최대 4개 실행
meca /path/to/project --server --protocol socket
meca /path/to/project --server --protocol grpc --listen 127.0.0.1:8788
meca --server --session session_XXXXXXXXXXXXXXXXXXXXXXXXXX
```

`socket`은 WebSocket이며 grpc는 gRPC·gRPC-Web·Connect다. 선택한 전송 방식과 공통 HTTP 제어·이벤트·첨부·작업 승인 경로를 같은 인증 경계에서 제공한다. 서버에는 TTY가 필요 없다.

### 내장 화면으로 시작하기

1. `meca /path/to/project --server --ui`로 기동한다. `--ui`만 지정하면 오류이며, 기본 수신 주소는 `127.0.0.1:8787`이다.
2. `http://127.0.0.1:8787/ui/`에 접속하고 기본 토큰 파일 `~/.meca/server.token`의 값을 연결 화면에 입력한다. `--token-file`을 지정했다면 해당 파일을 사용한다.
3. 작업공간·대화를 만들고 설정 → 모델에서 연결을 준비한 뒤 요청한다. 팀·스킬·파일 참조·이미지와 작업 승인·중지는 같은 API를 사용한다.

로컬 loopback의 대화형 출력 터미널에서는 브라우저를 자동으로 연다. 원격·비대화형 실행은 출력된 URL에 직접 접속하며, 다른 컴퓨터에서 사용할 때는 TLS·Host·Origin을 아래 인증 규칙에 맞게 설정한다. 화면 사용은 [WebUI 구조와 동작](webui-design.md), 클라이언트 구현은 [연동 계약](webui-api.md)을 따른다.


| 시작 옵션 | 기본값·의미 |
| --- | --- |
| `--server` | 시작 프로젝트와 등록한 작업공간을 서비스 |
| `--ui` | 내장 WebUI·브라우저 접속 인증 제공. --server에서만 사용 |
| `--protocol sse\|socket\|grpc` | 기본 `sse`; native 전송 하나 선택 |
| `--listen IP:port` | `127.0.0.1:8787` |
| `--session ID` | 현재 프로젝트의 지정 대화 선택 |
| `--token-file PATH` | 기본 `~/.meca/server.token` |
| `--tls-cert PATH`, `--tls-key PATH` | 함께 지정. loopback 외에는 필수 |
| `--allow-host authority` | 정확한 HTTP Host 추가. 반복 가능 |
| `--allow-origin scheme://host:port` | 브라우저 Origin 추가. 반복 가능 |
| `--concurrency N` | 동시 작업 수. 기본 1, 양의 정수. 제품 고정 상한 없음 |

프로세스는 시작 프로젝트와 등록한 작업공간, 프로젝트별 native 선택 대화와 작업별 실행 대화를 소유한다. 모든 작업공간이 같은 실행 슬롯·첨부·이벤트·효과 게이트를 사용한다. 서로 다른 세션의 요청은 `--concurrency`까지 실행하며 같은 세션에는 한 작업만 허용한다. 용량이 차면 새 접수를 `429 resource_exhausted`로 거절하고 이미 접수한 같은 키는 원래 작업을 반환한다. 팀 에이전트는 각 작업의 기존 실행 루프를 사용한다. 다른 TUI나 서버가 소유한 대화는 세션 잠금으로 열기를 거부한다. 프로세스의 변경·임의 셸 실행은 준비된 효과와 살아 있는 하위 프로세스가 소유한 게이트로 직렬화한다. 효과와 프로세스가 모두 끝나면 작업의 다음 모델 응답을 기다리는 동안에도 다른 작업이 접근할 수 있다. 같은 팀의 내부 협업 규칙은 유지하며 이 게이트가 OS 샌드박스를 제공하는 것은 아니다.

설정값은 동시 작업의 상한이며 그 수만큼 자원을 미리 할당하지 않는다. 64·128 등 더 큰 값을 사용할 수 있다. 활성 작업 수와 HTTP 요청·연결 수는 서로 다른 한도다. 메모리·저장·디코딩 예산은 작업 수 설정과 독립적으로 적용한다. snapshot의 `concurrency`는 `int64`이며 Protobuf JSON에서는 문자열로 표현한다.

## 인증과 네트워크

기본 토큰 파일은 `~/.meca/server.token`이다. 처음 실행할 때 원자적으로 생성하며 현재 사용자 소유의 private 일반 파일만 허용한다. `--token-file PATH`로 다른 파일을 지정할 수 있다. 상위 디렉터리는 미리 만들어야 하며 파일에는 32~256바이트의 공백 없는 ASCII 토큰을 넣는다. 시작 시 한 번 읽은 토큰을 인증과 비밀값 마스킹에 함께 사용한다. 파일을 교체한 뒤에는 서버를 다시 시작해야 적용된다. 토큰 내용은 시작 메시지에 출력하지 않는다.

외부 HTTP·SSE·RPC 클라이언트는 `Authorization: Bearer <token>` 헤더를 사용한다. URL 토큰은 받지 않는다. `--ui`에서는 POST /v1/ui/login에 같은 토큰을 제출해 만료하는 HttpOnly·SameSite=Strict 쿠키를 발급한다. 쿠키를 쓰는 변경 요청에는 동일 Origin과 X-Meca-UI: 1이 필요하다. 내장 EventSource·저장 이미지도 같은 쿠키를 사용한다. [브라우저 인증 상세](webui-api.md#내장-브라우저-인증)를 따른다.

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
| `listWorkspaces` | `{}` → `workspaces:{items,defaultId}`. id·path·name·activeTasks·pendingApprovals·error |
| `addWorkspace` | `path` → 작업공간 등록·전역 설정 저장 후 목록. 실제 경로 기준 중복 제거 |
| `browseFolders` | `path`, `offset` → 서버 폴더 선택용 directory. 한 번에 200개 항목을 읽고 폴더만 반환 |
| `deleteSession` | `sessionId` → 대화와 소유 팀 내부 파일 삭제. 활성 작업·다른 소유자 잠금 거절 |
| `commandCatalog` | `{}` → 브라우저에서 지원하는 여섯 명령·설명 |
| `parseInput` | `text` → 공통 파서의 action·argument. !new·!quit·:q 거절 |
| `suggestSkills` | `prefix` → 전체 메타데이터에서 최대 32개 이름-prefix 후보. 본문은 읽지 않음 |
| `lookupRequest` | `sessionId`, `key` → 기존 접수 조회. 새 작업·효과를 실행하지 않음 |
| `diagnoseModel` | `sessionId`, provider·endpoint·credential·model → 연결·응답 진단. 설정과 키를 저장하지 않음 |
| `editorRead` | `path` → 전체 UTF-8 text·revision·editable·reason. 편집 상한 1MiB |
| `editorSave` | `path`, `text`, `revision` → 현재 원본 재검사와 원자적 교체, 본문을 제외한 새 revision |
| `listSkills` | `cursor`, `limit` → `skills:{items,nextCursor,diagnostics,diagnosticsTruncated}`. 기본·최대 32건 |
| `getSkill` | `id` → `skill:{skill,revision,body}`. 본문 미리보기이며 모델 작업을 시작하지 않음 |
| `readSkillFile` | `id`, 스킬 루트 기준 `path`, `offset`, `limit`, 선택적 `revision` → `skillFile:{revision,text,mediaType,offset,nextOffset,totalBytes,binary}`. offset이 0보다 크면 첫 페이지의 revision 필수 |
| `listSessions` | 선택적 `cursor`, `limit` → 현재 프로젝트의 목록 페이지와 `nextCursor`. 기본 50건, 최대 100건. 미지원 DB는 `unavailable`에 사유를 표시 |
| `selectSession` | `{"create":true}` 또는 `{"sessionId":"…"}` → 상태 |
| `closeSession` | `sessionId` → 해당 세션 자원 해제. 작업 중에는 거부하고 이미 닫힌 자원은 성공 처리. DB는 삭제하지 않음 |
| `state` | 선택적 `sessionId`, `taskId`, `afterTask`, `limit` → 상태 페이지. 기본 32건·최대 100건, `nextTask`·`totalTasks` 제공 |
| `history` | `before`, `limit`, 선택적 `sessionId` → 현재 프로젝트의 지정 대화 시간순 페이지. 메모리에 없는 대화는 읽기 전용으로 열며 생략 시 native 선택 |
| `detail` | 원문 source 하나와 `offset`, `limit`, `revision`, 선택적 `sessionId` → UTF-8 JSON 원문 조각 `text`와 바이트 기준 nextOffset. 후속 페이지는 revision 필수 |
| `setModel` | `sessionId`, `provider`, `endpoint`, `model`, 선택적 `credential`, `thinking`, `reasoningEffort`, `remember` → 상태 |
| `setSettings` | `permission`, `team`, `language` 중 하나 → 상태. team은 `sessionId` 필수, language는 전역 UI 언어 |
| `start` | `sessionId`, `key`, `text`, 선택적 `attachmentIds`, `paths` → 즉시 작업 ID와 상태. 이미지 전용 요청 허용 |
| `capabilities` | `{}` → API 버전·기능 목록·이미지/상태/상세 한도·동시성·언어·접수 기록 할당량 |
| `providers` | `{}` → provider ID·이름·기본 endpoint |
| `discoverModels` | `sessionId`, `provider`, 선택적 endpoint·credential·model → 모델 목록 또는 해당 모델의 이미지·추론 옵션·context·진단 notice. 설정과 Key를 저장하지 않음 |
| `directory` | `path`, `offset`, `limit` → 파일 목록과 다음 offset. 기본 100건 |
| `previewFile` | `path`, `offset`, `limit`, `revision` → 마스킹한 UTF-8 파일 페이지. 기본/최대 64KiB. 후속 페이지에는 revision 필수 |
| `fileSelections` | `sessionId` → 저장된 참조 경로 |
| `selectFile` | `sessionId`, `path`, `selected` → 참조 선택 저장 |
| `historyImage` | `sessionId`, 메시지 `sequence`, 0부터 시작하는 `index` → 첨부 메타데이터. 픽셀은 별도 인증 GET |
| `draft` | `taskId`, `agent`, `generation`, `offset`, `limit`, 선택적 `thinking`, 후속 페이지의 `revision` → 실행 중 원문 페이지. generation 변경은 conflict |
| `commands` | `taskId` → 현재 실행 중인 단일/팀 명령의 ID·agent·상태·경과/마지막 출력 시간. 작업을 실행하지 않는 메타데이터 조회 |
| `resources` | `{}` → 현재 서버 프로세스 RSS 또는 읽기 오류 코드 |
| `listTasks` | 선택적 `sessionId`, `cursor`, `limit` → 영속 작업 목록. 기본 32·최대 100건, `nextCursor` |
| `taskStatus` | `taskId` → 상태 |
| `approve` | `taskId`, `approvalId`, `allow` → 처리 결과 |
| `steer` | `taskId`, `text`, 선택적 `paths` → 다음 안전한 판단 경계에 전달 |
| `cancel` | `taskId` → 취소 요청. 실제 종료는 `request_finished`로 확인 |

`setModel`은 기존 Provider 연결 준비·검증·저장과 OS 자격 증명 저장을 사용한다. 같은 연결에서 credential을 생략하면 저장된 참조를 사용한다. `remember`는 새 세션의 기본 연결도 저장한다. 기본값 저장만 실패하면 세션의 모델은 유지하고 부분 완료를 오류 메시지로 알린다. 저장된 모델은 첫 요청의 접수 전에 연결을 준비하므로 Provider나 키 저장소가 사용 불가능해도 대화 이력을 열고 모델을 다시 설정할 수 있다. 모델 변경은 해당 세션의 실행 중에 거절한다. 팀 모드 변경은 현재 실행이 캡처한 값을 바꾸지 않고 다음 요청부터 적용하며 전역 permission은 활성 작업이 하나라도 있으면 변경하지 않는다. permission은 `default` 또는 `all_mighty`이며 팀 모드는 명시한 세션에 저장한다. WebUI의 탭 선택은 클라이언트가 소유하며 조회·모델·팀·참조 변경·작업 실행을 위해 전역 선택을 바꿀 필요가 없다.

세션·설정 변경과 접수 준비는 직렬 예약하며 네트워크·DB I/O를 상태 잠금 밖에서 처리한다. state·이벤트·taskStatus·활성 작업 취소는 계속 처리한다. 명시적 sessionId의 history/detail은 읽기 대여로 수명을 보호한다. 조회와 실행은 같은 모델 선택 규칙을 사용한다. 대화에 저장한 선택이 우선하며 없으면 현재 기본 모델을 사용하고, 조회만으로 Provider에 연결하지 않는다. 세션 교체 정리가 실패하면 실제 선택 상태를 확인하기 위해 snapshot/state를 다시 조회한다. 닫히는 세션의 진행 중 읽기는 취소·합류한다.

`id`는 응답 상관관계용이다. 실제 실행 중복 방지는 `start.key`가 담당한다. 호출자가 요청마다 안정된 키를 만들고 응답을 잃었을 때 같은 세션·키·본문으로 재전송한다. 동일 키의 다른 본문은 conflict이며 기존 작업을 바꾸지 않는다. key는 8~128바이트다. `~/.meca/server-requests/<project-scope>/requests.db`는 본문 대신 해시와 작업 식별·상태만 저장한다. project-scope는 정규화 프로젝트 경로의 SHA-256이며 프로젝트 간 접수·조회가 분리된다. 실행 전에 기록하고 완료 후 상태를 확정한다. 비정상 종료되어 접수만 남은 작업은 interrupted로 조회되며 자동 재실행하지 않는다. 다시 수행하려면 실제 부분 효과를 확인한 뒤 새 키로 요청한다.

Task는 `code`와 `persistence`도 제공한다. `pending`은 현재 프로세스가 접수·실행을 소유한 상태, `confirmed`는 종료 상태의 저장을 확인한 상태, `uncertain`은 종료 저장이나 효과를 확인할 수 없는 상태다. 완료 쓰기가 실패하면 취소와 독립된 읽기로 같은 결과가 저장되었는지 확인한다. 확인하지 못하면 `code=completion_not_saved`, `persistence=uncertain`을 반환하고 해당 프로세스의 새 작업 접수를 막는다. 실행·쓰기 자체를 자동 재시도하지 않는다. 프로세스가 소유하지 않는 미완료 접수는 `interrupted`·`uncertain`으로 조회한다.

전송 어댑터는 [중앙 오류 분류](diagnostics.md#중앙-소유권과-오류)를 공유한다. 용량·저장 오류를 취소로 덮지 않으며 observer deadline은 HTTP 504, 용량 초과는 429다. 이미 응답을 쓰기 시작했다면 두 번째 오류 JSON을 이어 쓰지 않는다. 소켓·SSE의 전달 실패와 출력 내용의 재작성은 다른 오류다.

접수 조회 뒤 다른 호출이 먼저 같은 키를 확정한 경우에도 원래 작업을 반환한다. 현재 프로세스가 접수 확정 후 실행 인계를 진행 중이면 accepted, 실행 중이면 running/canceling으로 조회되며, 주인 없는 이전 프로세스의 접수만 interrupted로 복원한다.

승인은 서버가 발급한 일회성 approval ID와 task ID에 묶는다. 재사용·다른 작업·이미 끝난 승인에는 conflict를 반환한다. 승인 뒤에도 기존 실행기의 대상·버전·사용자 지침 재검사가 적용된다. preview가 잘렸으면 `detail.approvalId`로 전체 근거를 확인한다. 거절·취소·실행기 마감 또는 승인 대기 5분 경과는 해당 승인을 끝낸다.

SSE·socket·grpc 모두 `POST /v1/command`, `GET /v1/events`, `POST /v1/attachments`를 제공한다. `GET /v1/tasks/{taskId}`는 task Reply, `GET /v1/tasks/{taskId}/approvals`는 해당 작업의 승인만 담은 state Reply다. `POST /v1/tasks/{taskId}/approvals/{approvalId}`에 `{"allow":true}` 또는 `{"allow":false}`를 보내며 allow 생략·잘못된 타입을 거부한다. `POST /v1/tasks/{taskId}/cancel`은 취소를 요청한다. API HTTP에는 Bearer 또는 --ui의 브라우저 인증이 필요하다. /ui/ 정적 접속 화면에는 인증 정보를 포함하지 않는다. 이미지 업로드·해제 형식과 한도는 [이미지 입력](images.md#사용자-이미지-첨부)을 따른다.

명령 오류는 공통 `Reply.error:{code,message}`다. HTTP 제어는 이에 맞는 4xx/5xx 상태를 함께 반환한다. RPC의 애플리케이션 오류도 같은 Reply에 담으며 인증·잘못된 프레이밍·스트림 종료 등의 전송 오류는 해당 RPC 상태를 사용한다. 클라이언트는 둘 다 확인해야 한다.

ID·모델 설정 형식 오류는 invalid_argument, 없는 세션·원문은 not_found다. 취소된 제어·조회는 canceled이며 HTTP에서는 409로 반환한다. 세션 변경으로 조회가 취소되면 최신 상태를 확인한 뒤 필요한 원문만 다시 요청한다.

스킬 조회는 세션 선택 전에도 가능하며 명시한 작업공간의 프로젝트·실행 계정을 기준으로 한다. `getSkill`은 탐색 ID와 엔진에서 경로로 읽어 등록한 ID를 받고 임의 경로 입력은 받지 않는다. 참고 파일은 검증한 스킬 루트 안에서만 읽는다. 목록 페이지는 현재 파일을 다시 조회하므로 파일 변경 뒤에는 빈 cursor로 목록을 갱신한다. `start` 입력은 기존 sessionId·key·text이며 스킬은 현재 입력의 #이름 또는 에이전트의 읽기 도구로 지정한다. 형식과 한도는 [스킬 문서](skills.md)가 소유한다.

## SSE

- `POST /v1/command`: application/json Command → Reply
- `GET /v1/events`: SSE Notification. Last-Event-ID 또는 cursor로 재개하며 헤더가 우선한다. workspace_id·session_id로 본문 범위를 정할 수 있고 작업·승인·generation 같은 전역 전이는 계속 받는다. 범위를 생략한 클라이언트는 전체 사건을 받는다.

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

모든 서버 스트림은 마지막 애플리케이션 출력 후 5초가 지나면 heartbeat를 보내고, 출력이 재개될 때까지 1초마다 반복한다. 데이터가 나오면 다시 5초를 기다린다. SSE는 `: heartbeat` 주석과 빈 줄을 flush한다. WebSocket·gRPC·gRPC-Web·Connect는 `event.kind="heartbeat"`인 Notification을 보낸다. heartbeat에는 cursor가 없으며 이력·재생 로그·사용량에 넣지 않는다. 클라이언트는 heartbeat로 마지막 커서를 덮어쓰지 않는다. 구독 취소·완료·쓰기 실패 시 해당 관찰자의 대기와 타이머도 끝난다.

SSE 작성기는 이미 도착한 연속 델타의 flush를 묶는다. 첫 애플리케이션 출력과 첫 델타·제어 이벤트·완료·heartbeat는 즉시 전송하고, 나머지는 받은 묶음의 끝이나 누적 약 64KiB에서 flush한다. 별도의 대기 타이머·전송 큐를 만들지 않으며 이벤트 순서·커서·sequence·본문은 유지한다.

## 이벤트·재접속·수명

새 사건은 `occurredAt`과 가능한 `errorCode`를 포함한다. 순서는 시각 대신 커서를 따른다. `skill_loaded`의 `skill:{id,name,source,revision}`은 본문 읽기 성공을 나타내며 모델의 지침 준수나 작업 완료를 뜻하지 않는다. 본문은 이벤트에 반복해서 넣지 않는다. 재접속 시 기존 history/detail의 같은 메타데이터로 읽기 기록을 복원한다. 별도 활성 스킬 snapshot이나 활성화 명령은 없다. [운영 로그](diagnostics.md)는 동일한 작업 식별자로 장애 추적 정보를 제공한다.

커서는 프로세스 epoch와 단조 증가 번호의 조합이다. 재시작하면 새 epoch를 사용하며 선택 세션 변경은 커서를 초기화하지 않는다. 이벤트는 실제 session·task·agent·run·generation과 작업별 `sequence`를 제공한다. 기본 snapshot의 session·model·team은 native 선택이며 `tasks`·`executions`는 활성 작업의 첫 페이지다. taskId 필터가 있으면 해당 작업 대화의 상태만 구성한다. 후속 상태는 `state.afterTask=nextTask`로 조회한다. 단일 `task`·`execution`은 선택 대화의 작업 또는 대표 작업이며 execution에도 taskId가 있다. approvals·drafts도 taskId로 구분한다. 클라이언트는 task·agent·generation을 묶어 초안을 갱신한다. assistant_delta는 추가 본문, draft_reset은 해당 초안 교체다. progress·response·thinking은 완성된 내용이므로 델타로 다시 덧붙이지 않는다. 도구 참조·usage·최종 checkpoint·request_finished의 의미는 같다.

스트림 일시정지는 관찰만 멈춘다. SSE는 연결을 닫고 `Last-Event-ID` 또는 cursor로 재접속하며 `task_id` 쿼리로 작업을 한정한다. WebSocket은 `unwatch` 후 `watch:{cursor,taskId}`, RPC는 WatchEvents를 취소한 뒤 같은 `cursor`·`taskId`로 다시 연다. heartbeat로 커서를 갱신하지 않는다. 작업 실행 자체를 중지하려면 cancel을 사용한다. 작업 필터가 있으면 다른 작업의 활동이 해당 구독의 heartbeat 유예를 초기화하지 않는다.

첫 구독이나 만료된/다른 epoch 커서에는 현재 상태의 snapshot을 보낸다. 보관 중인 커서라면 그 이후 이벤트만 순서대로 전송한다. snapshot을 받으면 그 커서까지 반영된 상태로 바꾸고 저장 이력은 history로 별도 조회한다. 과거 이력은 페이지별 시간순이며 다음 요청에는 응답의 before를 전달하고 빈 페이지에서 끝낸다. history/detail 조회는 모델·도구를 실행하지 않는다.

현재 초안 snapshot은 출처별 본문·Thinking 각 4KiB, 전체 본문 합계 512KiB로 제한하며 truncated와 원문 textBytes·thinkingBytes로 알린다. 실행 중 원문은 draft로 페이지 조회하고, 완료 뒤에는 history/detail을 사용한다. draft의 같은 task·agent·generation만 합치며 재시도/reset 때 이전 페이지를 폐기한다. history 표시 필드도 잘릴 수 있다. 완료된 본문은 historySequence, 도구 결과는 tool reference로 detail을 읽는다. detail의 data를 같은 revision과 offset 순서로 합치면 JSON 원문이 된다. 서버·원문·마스킹 기준이 달라진 revision은 conflict다. history의 images에는 index·경로·MIME·크기·digest를 제공하며 pixels는 포함하지 않는다. bytes는 알려진 경우에만 양수다. 공개 표시 필드만 반환하며 Provider native continuation·이미지 원본·capture·자격 증명은 반환하지 않는다.

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
| 입력 예약 합계 32MiB | HTTP·RPC·WebSocket 입력을 실제 읽는 크기에 따라 32KiB 단위로 예약. 해석한 명령은 처리·응답까지 디코딩된 크기의 예약을 유지. 압축 RPC는 해제 전 일시적으로 17MiB를 예약한 뒤 실제 명령 크기로 조정. 초과는 실행 접수 전에 거절 |
| 이력 표시 32MiB, 상세 원문 복원 512MiB | 텍스트 미리보기는 작은 scratch로 전체 무결성을 검사하며 원문 크기만으로 거부하지 않음. 큰 복원은 동시 1건, 사용 중이면 즉시 resource_exhausted |
| 이력 1~100건, 기본 32건 | history 페이지 |
| 상세 1~65536바이트, 기본 65536 | detail 응답 바이트 |
| 상세·파일 미리보기 캐시 4건, 문서별 메모리 64KiB·압축 파일 합계 64MiB | 64KiB 독립 압축 블록에서 해당 페이지만 복원. 한도·여유 공간 부족 시 캐시 없이 페이지 수집 |
| 이미지 업로드 예약 합계 128MiB | 큰 이미지 업로드에 사용. 일반 제어 입력의 32MiB 예산과 분리 |
| 이미지 한 장 64MiB·64M픽셀, 요청당 8개·128MiB | 네이티브 제어는 업로드 ID를 참조. 디코더 공용 예산 768MiB |
| 프로세스 전체 임시 이미지 16개·256MiB·5분 | 불변 바이트를 대여하며 종료 시 만료 타이머도 취소·합류 |
| 2048건 및 8MiB | 모든 작업공간의 공통 이벤트 보관. 먼저 도달한 한도로 만료 |
| 이벤트 본문·Thinking 각 16KiB | 단일 표시 이벤트. 큰 스트림 델타는 분할 |
| 작업별 본문·Thinking 각 128KiB 메모리 접두부, 최대 16개 발신자 | 긴 실행 중 원문은 프로세스 합계 512MiB의 private 임시 파일에서 페이지 조회. reset·종료 때 해제 |
| 16건 및 합계 1MiB | 실행 중 방향 수정 대기열 |
| native 작업 오류 4KiB | SQLite 행에 종료 상태와 저장하며 읽기 전 필드 크기를 검사 |
| TCP 연결 최대 64개 | HTTP 해석·TLS 처리 전에 적용하며 유휴 연결도 포함. 초과 연결은 즉시 닫음 |
| 동시 일반 HTTP 요청 32개, 이벤트/파일 감시/소켓 합계 최대 16개, 작업 제어 전용 4개 | `/v1/tasks/`와 작업공간별 namespace의 승인·취소는 일반 요청과 별도 입장. HTTP/2 연결별 37개 스트림은 일반 32·제어 4·즉시 거절 여유 1개를 포함하며 TCP 한도와 별도 |
| 헤더 5초·입력 본문/소켓 프레임 읽기 30초·쓰기 15초 | 느린 연결 점유 제한. 입력을 다 읽으면 읽기 마감 해제. WebSocket 인증은 5초이며 완료한 명령은 다음 프레임 대기 전에 해제 |

세션 목록의 첫 페이지는 전체 순서를 확인하고 다음 페이지는 저장한 ID·시각에 해당하는 DB만 읽는다. 순서 참조는 현재 프로세스와 프로젝트에 묶인다. 만료·재시작 후에는 빈 커서로 다시 조회한다. 읽는 동안 갱신되거나 사라진 세션은 건너뛰며 새 목록에서 확인한다. 보관 수용량 초과는 `resource_exhausted`로 반환한다.

원문 복원 한도는 선택한 공개 payload의 읽기 작업량이며 모델 실행의 문맥·출력 예산과 구분한다. 한도 초과는 원문을 자르거나 삭제하지 않고 `resource_exhausted`로 반환한다. 세션 export에는 이 API 조회 한도를 적용하지 않는다. 원문 읽기 용량이 부족하면 대기열을 쌓지 않고 즉시 오류를 반환한다. 관찰 스트림의 heartbeat 수명은 원문 조회와 독립한다.

원문 조회와 journal 변환의 프로세스 공용 payload 예산은 합계 512MiB다. 조회는 원문을 보유한 채 추가 용량을 기다리지 않고 부족하면 `resource_exhausted`로 종료해 예약을 반환한다. 작은 완료 응답은 최대 허용량 전체를 선점하지 않는다. 이 예산은 논리적인 선택·변환 바이트이며 OS의 RSS quota가 아니다.

프로젝트의 대화 할당은 4096개, 전체 서버 접수 기록은 100000개까지 허용한다. 세션 DB는 512MiB 페이지 상한을 적용하고, 이미 더 큰 DB는 기존 페이지를 보존하며 추가 성장을 막는다. WAL은 checkpoint 정책으로 정리하지만 활성 reader 동안에는 별도 공간을 사용할 수 있다. 새 저장 할당 전에는 파일시스템 여유 공간 1GiB를 확인한다. 게시 전에 확정적으로 실패한 예약은 한 번 반환한다. 게시·커밋을 시도했거나 중단 후 결과를 확인할 수 없는 예약은 유지한다. 한도의 90%부터 storage_capacity_low에 count·limit을 기록한다. 한도에 도달해도 기존 조회·중복 확인·작업 완료 기록은 유지하고 새 할당을 거절한다. 기존 DB·접수 기록을 자동 삭제하거나 불확실한 counter를 재설정하지 않는다. 이 검사는 다른 프로그램의 동시 쓰기까지 예약하는 OS 디스크 quota는 아니다.

재조회용 정제 명령 로그·결과는 실행기당 256MiB·256건까지 보관한다. 상한에 도달하면 원본을 DB에 확정한 오래된 결과부터 회수한다. 미확정 결과는 지우지 않으며 여유가 없으면 오류를 반환한다. 원자적 교체에는 현재 쓰는 한 파일 크기만큼 추가 공간을 사용할 수 있다. 활성 원문 캡처는 별도 스트림별 수집 한도를 따르며 새 블록으로 커질 때에도 여유 공간 1GiB를 검사한다. 임시 파일의 만료는 DB 원본과 export를 삭제하지 않는다.

실행 이벤트는 네트워크에 직접 쓰지 않는다. 느린 관찰자는 독립적으로 읽고 쓰기 실패는 해당 연결만 끝낸다. 연결이 끊겨도 수락한 요청은 서버 프로세스의 context로 계속된다. 명시적인 cancel이나 정상 서버 종료만 작업을 취소한다. 종료는 새 요청을 막고 모델·도구·승인 대기를 취소·합류한 뒤 executor·client·DB·잠금을 해제한다. 강제 종료 후에는 기존 복구 규칙을 사용하며 불확실한 도구 효과를 재생하지 않는다.


서버–UI 조회는 실행 동시성과 별도 예산으로 제어한다. 원문 작업은 8단위의 가중 예산을 사용하며 작은 history·이미지 메타데이터는 1, 편집·미리보기·detail은 2, 이미지 복원·축소는 5를 사용한다. 이미지 메타데이터 예약은 픽셀 작업을 기다리기 전에 반환해 예약을 보유한 채 성장하는 교착을 피한다. 픽셀 디코딩과 원본 복원 바이트 예산도 계속 적용한다. HTTP JSON 응답은 최대 32MiB의 인코딩 예약을 전송 완료까지 보유하며 큰 응답만 협상된 gzip으로 압축한다. 메모리 부족 응답도 상관관계 ID를 제한하며 실행·저장을 재생하지 않는다.

디렉터리는 inode·변경 시각에 묶인 정렬 스냅샷을 최대 8개·4MiB로 공유하고, 후속 페이지는 revision을 검증한다. 대화 목록은 DB/WAL 변경 상태를 확인하는 4MiB·4,096행 메타데이터 인덱스를 사용한다. 정렬 커서는 최대 2분·128개·합계 8MiB이며 오래된 참조를 먼저 회수한다. 만료·회수된 커서는 목록 처음부터 다시 읽는다. native 접수 DB는 지원하는 현재 schema에서 `(session,id)` 파생 인덱스를 유지하며 세션 필터와 전체 목록 SQL을 분리한다.

인증된 저장 이미지의 `?thumbnail=1` 응답은 최대 변 256px의 PNG다. JPEG·PNG·WebP의 EXIF 방향을 반영하고 원본은 바꾸지 않는다. 파생 바이트는 최대 32개·4MiB로 보관하며 같은 원본의 동시 변환을 합친다. 캐시 적중에서도 현재 원본 메타데이터와 DB/WAL 상태, 작업공간·세션 접근을 확인하고 읽기 수명을 전송까지 대여한다. 응답은 원본과 같이 no-store다.

동시 관찰자가 있을 때만 최대 128개·1MiB의 정제 사건과 JSON을 공유한다. 마스킹 기준 변경이나 공유 관찰 종료 시 회수한다. SSE 본문 필터가 생략한 사건은 `event: cursor`와 id로 전송 위치를 알리며, 이 프레임은 도메인 사건이나 heartbeat가 아니다. heartbeat는 기존처럼 커서를 변경하지 않는다. detail은 Base64 data 대신 UTF-8 text를 반환하고 바이트 offset과 완전한 문자 경계를 사용한다.

## 개발과 생성

요청 DB는 **user_version 4**의 native 전용 형식이다. `requests` 행은 작업 ID·입력 digest·세션·상태·오류만 저장한다. native text·첨부 ID 순서·참조 경로를 접수 키에 바인딩한다. 같은 요청 재시도는 임시 첨부가 만료·해제됐거나 세션이 메모리에서 정리됐어도 기존 작업을 반환한다. 새 업로드 ID나 입력을 같은 키에 섞으면 conflict다. 결과 원문과 이미지 본문은 세션 DB가 소유한다. 버전 3과 이전 형식은 원본을 보존하고 열기를 거절하며 자동 변환·삭제·초기화하지 않는다. 모델 Provider 설정·API Key 저장 형식과 세션 DB schema 28은 이 요청 DB 변경과 별개다.

`internal/commands/server`는 자체 작업·선택/실행 세션·승인·이벤트 보관·원본 읽기 대여 수명, `internal/serverapi`는 인증·전송·공개 투영, `internal/serverapi/eventstream`은 heartbeat와 SSE 작성기, `internal/serverapi/detail`은 원본 마스킹·페이지 캐시, `internal/storage/serverrequests`는 native 접수·종료 기록을 소유한다. 실행·저장·표현의 책임을 합치지 않는다.

재생성에는 `protoc`와 devtools 모듈에 고정된 두 Go generator를 사용한다. 일반 제품 빌드·설치에는 생성 코드가 포함되므로 이 도구들이 필요 없다. Python·OpenSSL 실행 파일도 요구하지 않는다.

```sh
go build -modfile=devtools/go.mod -o bin/devtools/ \
  google.golang.org/protobuf/cmd/protoc-gen-go \
  connectrpc.com/connect/v2/cmd/protoc-gen-connect-go
PATH="$PWD/bin/devtools:$PATH" protoc -I api \
  --go_out=internal/serverapi/gen --go_opt=module=github.com/keonho-kim/meca/internal/serverapi/gen \
  --connect-go_out=internal/serverapi/gen --connect-go_opt=module=github.com/keonho-kim/meca/internal/serverapi/gen \
  api/meca/v1/*.proto
```

`internal/serverapi/gen`의 생성물은 descriptor·메시지·RPC 계약을 함께 재생성해야 한다. 수동 분할은 generator 출력과 descriptor 관계를 손상시키므로 생성 파일에 350줄 제한을 적용하지 않는다. 수정 원본은 `.proto`이며 생성물은 직접 고치지 않는다.

## WebUI 연결과 복원

`meca --server --ui`는 같은 서버에서 내장 UI와 인증 초기 화면을 제공한다. 최상위 Command.workspaceId로 작업공간을 지정하고 생략 시 시작 프로젝트를 사용한다. 이벤트·snapshot은 workspaceId를 포함한다. HTTP 이미지·작업 namespace와 파일 감시 계약은 [연결 가이드](webui-api.md)가 소유한다. 현재 API는 모델 탐색, 대화·작업 목록, 설정, 파일 참조, 이미지, 승인, 팀 스트림과 상태 복원을 제공한다. [클라이언트 연결 가이드](webui-api.md)를 따른다.

저장 이미지 GET은 `/v1/sessions/{sessionId}/images/{sequence}/{index}`이며 같은 Bearer·Host·Origin 검사를 적용한다. 해당 대화가 현재 선택되어 있을 필요는 없다. 업로드 metadata의 expires_at과 남은 lifetime_seconds는 임시 업로드 수명이며 저장된 이미지의 보존 수명과 다르다.

`execution.statistics`는 현재 작업의 완료 usage와 최신 진행 표본을 중복 없이 합산한다. startedAt·누적 시간·출력 토큰으로 처리량을 계산할 수 있다. 진행 usage 이벤트의 measuredMillis도 같은 측정 구간을 제공한다. snapshot.activity에는 현재 도구·팀·진행·재시도 표시를 제한된 크기로 제공하며 확정 원문은 history/detail에서 읽는다. approval.expiresAt과 approval_closed의 status(approved·denied·expired·canceled·failed)로 승인 UI를 정리한다.

모델·스킬·세션/작업 목록·파일 탐색은 별도 조회 입장 4건을 사용하며 여유가 없으면 즉시 resource_exhausted다. `commands`는 TUI와 같은 실행 자원의 동기화된 메타데이터를 읽고 `resources`는 같은 RSS 측정기를 사용한다. 명령 문자열이 길면 truncated로 알리고 짧은 접두부를 제공한다.

요청 DB 버전 4는 임시 첨부가 사라져도 중복 요청을 확인할 수 있는 native 참조 입력 식별 계약이다. 버전 3의 이전 입력 digest를 새 계약으로 해석하지 않으며 원본 DB를 보존한 채 거부한다.

## 동시 읽기와 세션 귀속

`state`에 sessionId와 taskId를 함께 전달하면 같은 대화에 속해야 하며 불일치는 invalid_argument다. 세션을 지정한 상태·설정 응답에는 그 대화의 작업만 포함한다. 팀 설정 변경 이벤트의 sessionId도 실제 변경 대상이다. taskId 한 건 조회는 전체 작업 목록을 정렬하거나 복사하지 않는다.

초안 generation은 같은 작업의 팀원 재참여에서도 증가한다. 퇴장 전 페이지의 generation으로 복귀한 팀원의 본문을 읽을 수 없으며 conflict로 거절한다. 임시 저장소 생성·쓰기 실패는 관련 파일과 예약을 바로 반환하고 원래 모델 작업의 수명을 취소하지 않는다.

`selectFile:{selected:false}`는 fileSelections에서 받은 저장 경로를 정확히 지정해 해제한다. 파일이 삭제·이동됐어도 해제할 수 있다. 이미 없는 선택 해제는 성공 처리한다. 없는 저장 이미지 sequence는 HTTP 404/not_found다.


## 작업공간 자원과 파일 감시

등록한 작업공간은 전역 설정의 workspaces 목록에 실제 절대 경로로 저장한다. 조회·실행 시 필요한 런타임만 열고, 기본 작업공간·활성 대여·실행·업로드·불확실한 저장 상태를 제외한 idle 네 개를 유지한다. 닫는 중 재개방을 기다리며 종료 호출은 모든 정리의 완료를 함께 확인한다.

GET `/v1/files/events`와 RPC `WatchFiles`는 활성 디렉터리 하나의 변경을 알린다. 파일 감시도 공통 관찰 스트림 한도를 사용하며 취소 시 watcher와 작업공간 대여를 반환한다. 작업의 request_finished.outcome과 HistoryItem.outcome은 실제 실행 결과이며 접수 DB의 answered 상태와 구분한다. 저장된 접수 행에는 outcome이 없을 수 있으므로 클라이언트가 성공을 추정하지 않는다.
