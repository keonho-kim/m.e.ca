# WebUI 연결 가이드

[서버 API](server.md) · [제품 명세](../SYSTEM.md)

WebUI는 full 배포본의 MECA 자체 API를 사용한다. HTTP JSON·SSE는 모든 서버 protocol에서 제공하며 WebSocket 또는 gRPC-Web·Connect도 같은 명령을 사용한다. `meca --server --ui`로 내장 화면을 켜면 정적 자산과 브라우저 접속용 인증도 제공한다. tui에는 서버·내장 화면이 없다. 서버 토큰을 HTML·URL·로그에 포함하지 않는다.

구현된 화면과 상태 소유·디자인은 [WebUI 구조](webui-design.md)에 정리한다. 내장 UI는 동일 출처 fetch와 EventSource를 사용한다.

사용자가 내장 화면만 쓸 때는 API를 직접 호출할 필요가 없다. [서버 기동](server.md#내장-화면으로-시작하기) 후 토큰으로 접속하고 모델을 설정한다. 커스텀 클라이언트에서 팀은 `setSettings`의 `sessionId`·`team`, 스킬은 `listSkills`·`getSkill`·`readSkillFile`, 명시 호출은 `start.text`의 `#스킬명`으로 연동한다. 스킬을 별도로 활성화하는 API는 없다. 팀의 공개 응답·도구·승인은 스트리밍하며 팀 Thinking은 개인 이력으로 유지한다.

## 화면과 API의 대응

| 화면 기능 | 서버 계약 |
| --- | --- |
| 작업공간 목록·폴더 선택·등록 | `listWorkspaces`, `browseFolders`, `addWorkspace` |
| 명령 후보·해석 | `commandCatalog`, `parseInput`; 브라우저에서 !new·!quit는 거절 |
| 시작·지원 기능·한도 확인 | `capabilities` |
| 전역 모델 설정 | `state:{defaultModel:true}`, `providers`, `discoverModels`, `diagnoseModel`, `setModel:{defaultModel:true,...}` |
| 대화 목록·새 대화 | `listSessions`, `selectSession:{create:true}` |
| 대화 열람 | `state:{sessionId}`, `history:{sessionId}`, `detail:{sessionId,...}` |
| 대화 삭제 | `deleteSession:{sessionId}`. 활성 작업을 거절하며 대화 DB·해당 팀 내부 데이터만 삭제 |
| 대화 자원 닫기 | `closeSession:{sessionId}`. 기록 삭제가 아님 |
| 권한·팀·언어 설정 | `setSettings`. 한 번에 항목 하나 |
| 파일 목록·미리보기·감시 | `directory`, `previewFile`, `/v1/files/events` / RPC `WatchFiles` |
| 전체 문서 편집 | `editorRead`, `editorSave`; 전체 UTF-8·revision 검사와 원자적 교체 |
| 선택 참조 보존·전송 | `fileSelections`, `selectFile`, `start.paths`, `steer.paths` |
| 이미지 붙여넣기·파일 첨부 | HTTP `/v1/attachments`, `start.attachmentIds` |
| 저장 첨부 표시 | `history.items[].images`, `historyImage`, 인증된 이미지 GET |
| 실행·추가 지시·취소 | `start`, `steer`, `cancel` |
| 승인 대화상자 | snapshot/event의 approval, `detail:{approvalId}`, `approve` |
| 단일·팀 스트리밍 | SSE / WatchEvents / socket watch |
| 컴포저 상태 | `resources`의 서버 RSS, `sessionUsage:{sessionId}`의 누적 토큰, 스트림 실행 상태 |
| 작업 완료 확인 | `taskStatus` |
| 재접속·긴 실행 중 본문 | snapshot/state 페이지, `draft` |
| 사용량·시간·처리량 표시 | execution의 startedAt·statistics와 usage 이벤트 |
| 스킬 탐색 | `listSkills`(최대 32), `suggestSkills`, `getSkill`, `readSkillFile` |

탭·선택 메시지·접기·스크롤·입력 초안·클립보드·예약 입력 편집은 클라이언트가 소유한다. TUI의 예약 입력도 화면 수명의 큐다. 브라우저 종료 후에도 실행해야 하는 요청은 `start`로 접수한 작업이어야 한다. 관찰을 중단해도 이미 접수한 작업은 서버에서 계속된다.

파일 선택 시 `editorRead`로 전체 편집 문서를 한 번 읽으며 편집 가능한 파일에 `previewFile`을 중복 요청하지 않는다. 편집 불가 응답일 때만 `previewFile`의 읽기 전용 페이지를 사용한다. Preview/Raw와 미리보기 본문 클릭은 클라이언트 표시 상태이며 Markdown 미리보기는 현재 초안을 사용한다. 전환 자체로 서버에 저장하지 않으며 `editorSave`의 revision 검사와 저장 확인을 유지한다.

파일 이름 변경·이동·삭제는 `mutateFile`을 사용한다. 목록에서 받은 부모 디렉터리 `revision`과 참조를 갱신할 대화의 `sessionId`를 함께 보낸다. 서버는 작업공간 경로와 보호 항목을 확인하고 참조 변경을 한 DB 트랜잭션으로 저장한다. 응답의 `path`는 이동 결과, `paths`는 해당 대화의 확정 참조 목록이다. `referencesReady`가 참일 때 전송 입력에 반영한다. `errorCode`·`errorMessage`가 있으면 부분 완료 여부를 확인하며 파일 효과를 무조건 재실행하지 않는다. 파일시스템과 DB는 하나의 원자적 트랜잭션이 아니며, 참조 저장을 확인하지 못하면 전송을 막고 경로를 다시 확인한다. 같은 이름의 별도 대상은 덮어쓰지 않고 대소문자만 바꾸는 이름 변경은 지원한다. 폴더 삭제는 하위 항목을 포함하고 항목 사이에서 취소를 확인하므로 중단 시 일부 항목이 이미 삭제되었을 수 있다.

`sessionUsage`는 저장된 사용량 이벤트의 누적 합계다. 사용량 이벤트·요청 완료·재연결 snapshot에서 다시 조회하며 대화 삭제 시 해당 캐시도 제거한다. 전역 모델 폼은 전역 기본 모델만 조회하고 변경되지 않은 값은 저장하지 않는다.

## 대상과 동시성

`listWorkspaces`의 id와 대화 생성 응답의 `session.id`를 화면에 보관한다. 프로젝트별 명령에는 최상위 `workspaceId`를 지정하며 생략 시 시작 프로젝트를 사용한다. 잘못된 workspaceId를 다른 프로젝트로 대체하지 않는다. `--concurrency`는 모든 프로젝트를 합친 프로세스 상한이다. 대화별 모델·팀 변경과 세션 닫기에는 `sessionId`를 명시한다. 전역 기본 모델의 조회·저장은 `defaultModel:true`로 요청한다. state에 taskId도 지정할 때는 같은 세션의 작업이어야 한다. 권한·언어는 전역 설정이다. 전역 `selectSession`을 탭 전환마다 호출하지 않는다. 미선택 대화의 읽기와 작업 시작도 sessionId로 직접 수행한다.

```json
{"id":"models","discoverModels":{"provider":"ollama"}}
```

```json
{"id":"model-save","setModel":{"defaultModel":true,"provider":"ollama","model":"model-id"}}
```

```json
{"id":"start","start":{"sessionId":"session_XXXXXXXXXXXXXXXXXXXXXXXXXX","key":"client-generated-stable-key","text":"선택한 파일을 검토해 주세요","paths":["src/main.go"]}}
```

응답을 잃으면 `lookupRequest:{sessionId,key}`로 기존 작업부터 조회한다. 같은 논리 요청을 재시도할 때 key와 입력을 그대로 보낸다. 첨부를 다시 업로드한 새 ID는 새 입력이므로 새 key를 사용한다. 만료·해제된 업로드 ID라도 이미 접수된 동일 요청의 중복 확인은 가능하다. 모델 작업과 파일 효과를 새 key로 무조건 재실행하지 않는다.

## 상태와 스트림 복원

1. 최초 snapshot으로 현재 커서를 확보한다. `nextTask`가 있으면 `state.afterTask`로 후속 페이지를 읽는다. 이 페이지들은 조회 시점의 상태이며 하나의 원자적인 전체 작업 목록은 아니다. 중간 이벤트를 같은 커서 구독에서 함께 반영하고 task ID로 병합한다.
2. task별 화면은 `state.taskId`와 스트림의 taskId 필터를 사용한다. 다른 작업의 draft·승인·모델이 섞이지 않는다.
3. 스트림 순서는 cursor와 작업별 sequence를 따른다. heartbeat는 커서를 갱신하지 않는다. source는 taskId·agent·generation으로 식별한다.
4. snapshot의 draft는 작은 접두부다. `textBytes`·`thinkingBytes`가 실제 수신 길이를 알리고, `truncated`면 `draft`로 원문을 읽는다. offset·limit은 소스 바이트 기준이며 기본/최대 페이지는 64KiB다. draft 첫 페이지의 revision도 후속 요청에 넣는다. 마스킹으로 text 길이가 달라질 수 있으므로 text 길이가 아닌 nextOffset으로 진행한다. UTF-8 경계를 자르지 않으므로 다음 offset을 그대로 사용한다.
5. draft 페이지와 스냅샷의 generation이 달라지면 기존 조립 내용을 폐기한다. reset은 새 generation의 시작이며 같은 팀원이 퇴장 후 복귀해도 이전 generation을 재사용하지 않는다. draft 조회가 작업 종료와 겹치면 taskStatus와 history/detail로 전환한다.
6. 완료된 대화는 history로 읽는다. 메시지 sequence와 도구 reference가 원문 조회 키다. 확정 이력과 진행 초안을 중복 표시하지 않는다.

`usage`·`usage_progress`·`request_finished` 이벤트의 statistics는 서버가 집계한 최신 측정이다. 브라우저는 완료 뒤 live 사건을 정리해도 마지막 통계를 별도 작은 값으로 유지한다.

`activity`는 현재 도구·팀·진행·재시도의 제한된 복원 상태다. activityTruncated이면 해당 task의 state로 좁혀 조회한다. 확정된 사건 전체는 history/detail이 소유한다. `execution.statistics`는 완료 사용량과 현재 보고된 진행 표본을 합산하므로 같은 usage를 다시 더하지 않는다. 비용 미보고와 0을 구분한다.

## 파일·이미지 페이지

파일 목록은 반환된 nextOffset과 more를 사용한다. 파일 미리보기 첫 요청은 offset 0이며 revision을 받는다. 다음 페이지에 같은 path와 revision을 전달한다. revision은 원문과 마스킹 기준을 함께 고정하는 불투명 토큰이다. 캐시는 원문이 바뀌어도 해당 revision의 불변 미리보기를 반환할 수 있다. 캐시가 없고 현재 파일이 달라졌거나 마스킹 기준이 바뀌었으면 conflict이며 첫 페이지부터 새로 읽는다. 파일 접근 정책은 캐시 적중에서도 다시 검사한다.

이미지는 제출 직전에 업로드하고 받은 ID를 요청에 넣는다. 실행 대기 중인 예약 입력은 브라우저의 원본 File/Blob을 보유하고 제출 시 업로드해 임시 ID 만료를 피한다. 업로드의 expires_at과 lifetime_seconds는 임시 참조의 수명이다. 실행이 빌린 바이트나 이미 저장된 첨부의 수명을 바꾸지 않는다. HTTP GET `/v1/attachments/{id}`는 메타데이터만 반환한다.

저장 첨부는 history의 이미지 index로 GET `/v1/workspaces/{workspaceId}/sessions/{sessionId}/images/{sequence}/{index}`를 호출한다. 응답은 실제 이미지 MIME의 바이너리이고 인증이 필요하다. 메타데이터는 픽셀을 포함하지 않으며 알려지지 않은 bytes 값은 0이다.

## 승인과 운영 상태

approval의 expiresAt으로 만료 안내를 표시한다. `approve`는 taskId와 approvalId에 바인딩되며 이미 결정·만료된 승인은 conflict다. `approval_closed.status`는 approved·denied·expired·canceled·failed 중 하나다. 승인 응답 성공을 파일 효과 완료로 해석하지 않는다.

큰 원문 읽기는 자원 부족 시 즉시 resource_exhausted를 반환한다. 프론트는 사용자가 보고 있는 읽기 하나를 우선하고, 화면 전환 시 이전 요청을 취소하며 무제한 재시도 큐를 만들지 않는다. 작업 제어 REST는 일반 HTTP 요청과 별도 입장을 사용한다. 내장 UI는 `/v1/workspaces/{workspaceId}/tasks/{taskId}/cancel`과 `/approvals/{approvalId}`를 사용한다.

capabilities의 requestRecords·requestCapacity는 프로세스 간 보수적인 접수 기록 할당량이다. 90%부터 운영 로그에 용량 안내가 기록되며 한도에 도달하면 새 할당만 거절한다. 기존 작업 조회·중복 확인·완료 기록은 유지한다. 운영 백업은 세션·접수 DB·필요한 복구 자원을 함께 보존해야 하며 counter나 중복 방지 기록을 임의로 삭제하지 않는다.

상세 원문 detail도 offset 0에서 받은 revision을 후속 페이지에 넣는다. 서버 재시작·source 변경·비밀값 마스킹 revision 변경은 conflict이며 기존 페이지를 폐기하고 다시 읽는다. 원문 페이지를 서로 다른 revision으로 합치지 않는다.

detail 응답은 UTF-8 JSON 원문 조각인 `text`와 바이트 기준 `offset`·`nextOffset`·`totalBytes`를 사용한다. 이전 Base64 `data` 필드는 제거됐다. `nextOffset`은 완전한 UTF-8 문자 끝을 가리키며 요청한 limit 안에서 진행 가능한 문자가 없으면 invalid_argument다. 클라이언트는 문자열 길이로 offset을 계산하지 않는다.

디렉터리 목록은 첫 응답의 `revision`을 후속 `directory` 요청에 전달한다. 정렬된 목록이 바뀌면 conflict이며 첫 페이지에서 다시 시작한다. 목록 캐시가 적중해도 현재 정책·경로와 각 표시 항목을 확인한다. 파일 감시의 첫 refresh 뒤 조회를 시작하면 최초 목록을 중복 조회하지 않는다.

SSE의 `workspace_id`·`session_id`는 표시할 본문 범위를 정한다. 빈 session_id를 명시하면 본문을 구독하지 않는다. 작업 시작·완료·승인·사용량·설정·삭제·generation 전이는 전역으로 수신한다. 선택 범위의 본문을 생략한 구간은 `event: cursor`와 SSE id로 재접속 위치를 알리며 도메인 사건·heartbeat가 아니다. 브라우저 EventSource가 id를 보관하고, 앱은 이 프레임을 본문으로 렌더링하지 않는다.

저장 이미지 GET에 `?thumbnail=1`을 붙이면 최대 변 256px의 PNG를 반환한다. 원본과 동일한 인증·작업공간·세션 읽기 경계를 사용하며 Cache-Control은 no-store다. 원본을 요청할 때는 query를 생략한다. capabilities의 `promptBytes`와 `commandBytes`는 단일 UTF-8 본문과 JSON 전송 한도이며 브라우저의 전체 초안 보관 예산과 다르다.

저장한 참조를 해제할 때는 fileSelections가 반환한 경로를 selected:false로 보낸다. 실제 파일이 사라졌더라도 저장된 선택은 제거할 수 있다.

## 내장 브라우저 인증

`--ui`에서 GET `/ui/`는 정적 접속 화면이다. 서버 토큰을 `POST /v1/ui/login`의 Bearer 헤더로 한 번 전달하면 8시간 HttpOnly·SameSite=Strict 쿠키를 발급한다. TLS 연결에서는 Secure도 설정한다. 쿠키는 `/v1`에만 전송하며 서버는 최대 64개 세션의 해시·만료만 보관한다. `POST /v1/ui/logout`은 해당 쿠키를 폐기한다.

GET `/v1/ui/session`은 `{"authenticated":true|false}`만 반환하는 읽기 전용 접속 상태 조회다. 인증 전에도 Host·Origin 검사를 적용하며 보호된 명령을 실행하지 않는다. 첫 접속의 쿠키 복원과 로그인 후 쿠키 수신 확인에 사용한다. 쿠키 이름은 접속 주소별로 구분하므로 같은 호스트의 다른 포트에서 실행한 서버끼리 덮어쓰지 않는다. 서버 재시작·만료 뒤에는 다시 로그인한다.

쿠키를 사용하는 POST·DELETE는 정확한 동일 Origin과 `X-Meca-UI: 1`이 필요하다. 기본 Host·Origin·TLS 경계는 그대로 적용한다. 로그인 토큰을 URL·localStorage·HTML에 넣지 않는다. 다른 클라이언트의 Bearer 인증은 계속 사용할 수 있다.

업로드·해제와 기본 작업 제어/이미지 HTTP에 `X-Meca-Workspace`를 지정할 수 있다. 이미지·작업 전용 namespace URL은 경로의 작업공간을 사용한다. 파일 감시는 GET `/v1/files/events?workspace_id=...&path=...`이며 change 이벤트의 refresh가 true이면 해당 폴더 목록을 다시 읽는다. 감시는 재귀 트리 저장이나 파일 본문 스트림이 아니다.

## 완료·예약·편집

`request_finished.outcome`과 저장 이력의 `HistoryItem.outcome`은 실제 작업 결과다. task.status는 접수·종료 저장 상태를 뜻하므로 answered를 성공으로 간주하지 않는다. Task의 outcome은 현재 런타임에서 확인한 경우에만 제공하며 이전 영속 접수 행에는 없을 수 있다. 재연결 후 성공을 확인하지 못하면 후속 예약을 멈춘다.

`editorRead:{path}`는 전체 문서·revision·editable·reason을 반환한다. `editorSave:{path,text,revision}`는 전체 텍스트를 받아 현재 파일을 다시 확인한다. UTF-8 일반 파일 최대 2MiB만 편집하며 마스킹된 내용·경로, 바이너리·잘못된 UTF-8은 저장하지 않는다. 브라우저도 `capabilities.editorBytes`의 같은 UTF-8 바이트 한도를 입력·붙여넣기 전에 적용한다. 저장 응답은 본문을 다시 적재하지 않고 새 revision을 반환한다. 같은 본문은 파일을 다시 쓰지 않는다. 미리보기 페이지를 편집 원본으로 저장하지 않는다.
