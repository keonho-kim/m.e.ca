# 개발 가이드

[← README](../README.md) · [제품 명세](../SYSTEM.md) · [서버 API](server.md)

## 책임과 의존성

스킬 문서 값과 파일시스템 어댑터는 다른 의존성 경계다. agent/tool과 agent/run은 skills/instruction의 순수 값만 참조하고, 요청 코어는 소비자 소유 SkillLoader 인터페이스로 본문을 읽는다. commands/agentrequest가 실제 skills.Reader를 연결하며 없는 리더는 nil 인터페이스로 유지한다. 메타데이터의 직렬화 크기는 문서 DTO에 넣지 않고 reader의 catalogEntry에 한 번 계산해 보관한다. 공개 문서 값은 크기 캐시와 별개로 전달한다.

서버의 호환 요청/응답 값은 response.go, 안정 식별자는 identity.go, 상태 조회는 response_state.go, 원문 읽기와 예산은 response_output.go가 소유한다. submission_session.go는 대화 획득·이어가기 경계를, submission_input.go는 외부 입력의 신뢰 수준·정규화·Chat의 대여와 Responses의 단일 보존을 소유한다. 같은 Service와 DB 트랜잭션·잠금·읽기 대여 수명은 유지한다.

processmemory/transient.go는 취소 가능한 바이트 입장을, reply.go는 누적 응답을 보유한 채 대기하지 않는 예약을 소유한다. 이미지 픽셀·인코딩 크기를 메모리 가중치로 바꾸는 규칙은 tools/local/imagefile/decode.go에서 계산하고 processmemory에는 바이트만 전달한다. 같은 예산·취소·반환 경계를 사용하며 추가 worker나 버퍼를 만들지 않는다.

제품은 단일 Go 모듈과 cmd/meca 합성 루트를 사용한다. 기능은 internal 아래에 두고 실제 소유 패키지를 직접 참조한다. 소비자 소유의 작은 I/O 계약과 함수·struct를 우선하며 패키지는 한 기능, 파일은 그 안의 한 책임을 담당한다.

| 위치 | 책임 |
| --- | --- |
| cmd/meca | CLI 분기·어댑터 조립·프로세스와 터미널 수명 |
| agent/model | 모델 메시지·스트림·출력 예산 계약 |
| agent/session·run | 요청 준비·실행 순서·승인 조율·공유 예산·실측 통계 집계 |
| agent/session/recovery | 중단된 요청·native 호출의 복구 정책과 원자적 복구 기록 |
| agent/team | 참여·초대·복구·공용 대화 전달·공유 요청 예산·최종 보고자 전이 |
| agent/team/identity·channel·source | 고정 이름 목록·공개 메시지와 문맥 투영·명시적 공용 출처 값·공개 로그 저장 계약 |
| agent/team/contribution | 현재 담당 영역의 원자적 교체·겹침 판정·관련 변경 순서·독립 스냅샷과 예산 내 현황 포맷 |
| commands/teamrequest | 요청 공통 자원·개인 세션 수명·직렬 승인·활성 명령 조회를 분리한 팀 조립 |
| agent/record | 실행 이벤트·체크포인트·상태·실측 처리량과 최종 통계·원자적 기록 단위와 검증 |
| agent/context·compact | 모델 문맥과 단일 호출 요약 |
| commands/headless | 같은 요청 경로의 text·JSONL 출력 어댑터 |
| commands/server | 프로세스 소유 요청·native 선택과 호환 작업 대화의 분리·승인·방향 수정·취소·이벤트 보관·대화별 원본 읽기 대여 |
| commands/conversation | 서버 대화·실행이 공유하는 모델 연결의 명시적 대여와 마지막 소유자의 정리 |
| commands/modelchange | 세션 연결 저장·불확실한 커밋 확인·독립된 기본값 저장의 부분 실패 정책 |
| serverapi · api/meca/v1 | 공통 외부 계약·투영·인증·TLS·SSE·WebSocket·Protobuf RPC |
| serverapi/admission | 입력 바이트·큰 원문 작업의 동시 입장·취소·해제 |
| serverapi/detail | 공개 필드만 가진 상세 JSON·마스킹·바이트 페이지·세션별 최근 문서와 private spool 수명 |
| serverapi/openai | 텍스트 호환 입력 경계·Chat/Responses DTO·append-only 출력 투영과 오류 |
| serverapi/eventstream | 5초 무출력 이후 1초 heartbeat 시계·SSE 기록·flush. 실행 이력과 독립 |
| storage/serverrequests | 프로젝트별 SQLite의 원문 없는 요청 접수·완료·키 바인딩·원본 범위·후속 head의 원자적 예약 |
| storage/capacity | 프로세스 공통 할당 상한·지속적인 예약 횟수·새 할당 전 여유 공간 검사 |
| storage/credentialstore | Provider/endpoint에 인증된 API Key 암호문·독립 난수 암호화 키의 OS 저장 계약·원자적 SQLite 저장과 복구 실패 보존 |
| commands/agentrequest | 재사용 Runtime과 요청별 Generator·입력·방향 수정 채널을 공통 요청으로 조립 |
| skills | SKILL.md·호출 정책 파일 I/O·메타데이터 snapshot과 크기 캐시·출처 조회·참고 파일 대여와 경로 등록 |
| skills/instruction | 파일 I/O와 독립된 문서·메타데이터·선택자·읽기 참조 값·이름 규칙·명시 호출 파싱과 읽기 의도 |
| config/bootstrap/defaults | create-skill·application-design의 독립된 본문·참고 지침·배포 고지 |
| diagnostics/catalog·fault | 중앙 메시지·코드·이벤트·레벨·복구 문구·원인 분류와 오류 값 |
| diagnostics | 등록된 사건·허용 필드·범위·출력 큐·health·종료 합류 |
| diagnostics/logfile | private 파일·회전·종료된 실행 보관 |
| agent/tool·work | 실행 사실·가벼운 파일 관측·종료 원인 |
| agent/prompt·inference | 공통 작업 지침·팀 보강 합성·역할별 추론 설정 |
| commands/startup·install·upgrade·settings | 시작 경로·사용자 설치와 삭제·검증한 최신 릴리스 업데이트·언어와 승인 모드 저장 |
| config/bootstrap·prompts·statefs·credentials | 초기화·Markdown 파일·private 상태 수명·OS 자격 증명 |
| redaction | 저장소·OS I/O와 독립된 비밀값 등록·revision·JSON과 스트림 마스킹 |
| processmemory | 현재 RSS 조회·동시 실행 문맥·응답·이미지 decode·payload 변환의 논리 예약과 반환 |
| provider/connection·identity·selection | 연결 값·서버 원본 표시·Provider 추론 옵션·선택 검증 |
| provider/discovery·diagnostic·setup | 메타데이터·실제 생성 진단·준비된 연결 반환. 영속 확정은 commands/modelchange가 담당 |
| provider/protocol | API 조립·실행 범위의 이력 인코딩 재사용·재읽기 가능한 본문·공통 스트림 프레이밍 |
| provider/protocol/chat·ollama·responses·claude·gemini | API별 요청 DTO·수신 형식·signed reasoning/도구 이력과 전용 상태 |
| provider/protocol/native | 공통 함수 스키마·인자 누적·완료 검사·관측 사용량 상태 |
| provider/transport | HTTP 연결·수신 감지·재시도 |
| provider/httpclient | 생성 요청·유효 문맥·실행별 인코더의 수명 조율 |
| policy·policy/command | 공통 보호 대상·파일 Scope와 execute_command의 선택적 AST 검사·값 상태·명령별 승인 근거 |
| policy/command/sedscript | sed 스크립트 주소·구문 해석과 파일 쓰기/중첩 실행 효과 추출 |
| policy/command/transfer | HTTP·원격 복사·원격 실행의 전송 인자와 표시용 목적지 해석 |
| tools/local/catalog | 불변 도구 설명·스키마·native JSON 인자 파싱 |
| tools/team | 팀 전용 도구 실행과 로컬 도구 어댑터·접근 예약·명령 수명 연결 |
| tools/team/catalog | 불변 팀 도구 설명·스키마·입력 파싱과 도구 분류 |
| tools/team/workspace | 경로·inode 충돌 배제·셸/파일 접근권·준비된 대상 확인 |
| tools/team/approval | 동결된 후보를 기준으로 요청 내 사용자 거절 기록 공유 |
| tools/local | 준비·승인 바인딩·실행·자원 수명을 연결하는 호스트 어댑터 |
| fileview | 도구 실행과 분리한 읽기 전용 탐색·참조 검증·미리보기 I/O, 공통 경로 정책을 적용한 루트 대여 |
| tools/local/filesystem | 실제 경로·프로젝트 identity·파일 도구의 private 경로 검사 |
| tools/local/file·textedit·textformat·diff | 파일 I/O·정확한 부분 수정·바이트 형식·변경 미리보기 |
| tools/local/process | 호스트 셸·명령 등록부·프로세스 그룹·출력·대기·취소 |
| storage/session·history | 세션 식별·순서 있는 이력·표시 필드 선택과 순수 복원 투영 |
| storage/content | 큰 JSON 값의 참조 분리·선택 복원·압축·내용 무결성 |
| storage/sqlite | DB 파일·SQL·트랜잭션·잠금·content 콜백 어댑터 |
| tui/session | 단일 이벤트 루프·기능별 상태 구성·공통 포커스 판정·화면 전환과 I/O 조율 |
| tui/composer·composer/queue | 본문·선택 경로를 묶은 요청 값·후속 요청 순서·예약 편집·초안 복구·큐 표시 캐시 |
| tui/settings/model | 모델 설정 초안·값 전환·입력 검증·검색·식별 정보·화면 투영 |
| tui/browser | 파일 열 탐색·선택 위치·순수 탐색 상태 |
| tui/browser/controller | 디렉터리·미리보기·배치·감시 worker의 취소·결과 전달·종료 합류 |
| tui/browser/readqueue | 물리 실행 슬롯과 최신 대기 요청 한 건의 순수 입장 상태 |
| tui/preview | Raw/Markdown 배치 계산·표시 모드·읽기 위치·최신 원문과 폭에 맞는 결과 수용 |
| tui/locale | 불변 번역 목록·설정 우선 언어 선택·시스템 로캘 해석 |
| tui/chat/team | 발신자별 스트림·도구 카드·사용량·참여/압축 상태의 순수 화면 투영 |
| tui/transcript·view | 메시지/도구 투영·공통 설정 화면 및 대화 렌더링 |
| tui/input·textlayout·selection | 편집 revision·grapheme 배치·원문에 연결된 선택 |
| tui/startup | 대화 선택 |
| tui/viewport | 이력과 현재 출력의 증분 줄 인덱스·가시 구간·스크롤 위치 |
| tui/markdown | 응답·Thinking·파일 뷰어의 Markdown 파싱·스타일·증분 표시 |
| searchmatch | 명령·모델 후보의 Unicode 축약 검색 |

tui/session의 app은 대화 표시(chatState), 입력·예약(requestState), 파일 탐색(fileBrowserState), 연결(connectionState), 작업 시계(workerActivity)를 값으로 구성한다. 본문·토큰, 입력 기록, 탐색·선택 결과, 클라이언트 교체와 타이머 전이는 해당 상태가 소유한다. 세션은 I/O 시작·결과 전달·포커스 전환을 조율한다. 키 입력·키맵·화면 렌더링은 focusKind의 같은 판정을 사용하고 전역 선택·종료 입력의 우선순위를 유지한다.

TUI의 활성 worker는 idle·agent·agent stopping·setup·setup stopping 중 한 상태이며 이벤트 루프만 전환한다. TUI는 저장·연결 설정·요청 준비를 직접 소유하지 않는다. worker는 결과 이벤트를 반환하고 화면 상태는 이벤트 루프에서만 바꾼다. 연결 클라이언트·HTTP body·goroutine·timer·DB·자식 프로세스에는 취소와 정리 소유자가 있어야 한다.

TUI·비대화형 실행은 commands/agentrequest.New에서 같은 추론·문맥·압축 설정과 선택된 세션의 정규화한 프로젝트 경로를 사용한다. 요청의 Thinking 선택은 값으로 복사한다. 실제 자원 생성과 종료는 각 진입점, 지침 읽기와 요청 실행은 agent/session이 소유한다.

외부 전송 판정은 AST 연결과 프로토콜 파서로 나누고 명령·파일·DNS·네트워크를 실행하지 않는 순수 함수로 유지한다. `UserRequired`는 일반 승인 필요 여부와 구분하며 All Mighty도 직접 승인을 생략하지 않는다. 승인 창·원본 재검사·배치 실행 전 승인·실제 승인 출처 기록을 공유한다. 정상 다운로드·직접 지정한 loopback HTTP·인용된 명령 본문은 별도로 확인하고, 임의 프로그램 내부 통신까지 제어한다고 설명하지 않는다.

팀 실행은 agent/team과 commands/teamrequest가 소유한다. agent/team/channel은 공용 기록 계약·문맥 투영·최대 128건/2MiB의 요청 범위 캐시를 소유하고, 실행 관측의 순수 문자열 생성은 observation_format.go에서 게시·중복 판정과 분리한다. 관측 문자열과 출력 지문은 공용 게시 잠금 밖에서 계산하고, 잠금 안에서는 명령 관측 중복을 재검사한 뒤 기록·진전·알림을 확정한다. 단일 요청은 팀 참여·공용 대화·팀 도구를 구성하지 않는다. 팀 요청은 선택적 수신 문맥과 전체 요청 예산·마감 시각을 연결하고 같은 agent/run을 독립적으로 사용한다. 공용 본문은 원본 참조, 개인 수신 위치는 개인 문맥과 원자적으로 저장한다. TUI의 팀 스트림은 발신자별 최신 스냅샷으로 전달하고 같은 이력 블록을 갱신해 변경되지 않은 앞부분의 배치를 보존한다. tui/chat/team/message.go는 응답·도구 카드의 발신자와 본문 ID를 함께 소유한다. transcript.TeamBody는 실시간 완료 발화와 저장 이력의 마스킹·본문 형식을 공유하며 기존 발화 갱신에는 이름표와 블록 배열을 만들지 않는다. 공백뿐인 응답은 표시를 만들지 않으며 viewport는 이름표와 본문의 보유·제거를 한 그룹으로 처리한다. 이름 아래 한 줄·본문 뒤 두 줄은 Block.SpaceAfter로 지정하며 가시 창에서만 빈 행을 만든다. 원본에 개행을 덧붙이거나 저장하지 않고 배치 캐시·완료 스트림 인계도 같은 간격 값을 사용한다. 원본 이력과 본문의 증분 Markdown은 별도로 유지한다.

현재 참여자의 공개 발언 여부는 Room의 수명 한정 상태다. 커밋한 일반 발언·discuss·담당 선언으로 갱신하고 재개 시 초기화하며 DB 행이나 대화 사본을 추가하지 않는다. tools/team/admission은 첫 로컬 작업 전에 이를 검사하고 미충족 시 실행하지 않은 도구 피드백을 반환한다. 명령 제어·대화 조회는 독립적으로 유지한다. agent/context는 출처가 표시된 peer 관측 본문을 읽기 쉬운 비권한 문맥으로 제공하고 일반 관측·사용자 요청·native 호출의 처리는 구분한다.

팀의 set_work는 자기 업무의 결과·정규화한 경로·선택적 공유 규약·관련 발언 참조를 준비하고 공개 기록을 저장한 뒤 현재 상태를 교체한다. 경로·규약의 지문과 설명 문구의 동일성은 구분하며 설명만 바뀐 공개 기록은 재추론 알림을 만들지 않는다. 공유 규약의 제거도 실제 변경으로 전달하고 직접 답글 알림은 유지한다. 도구 결과에는 이름·경로·공개 순서·revision을 반환하며 본인 문맥에 본문을 다시 추가하지 않는다. 경로 해제는 완료 증명이 아니며 같은 경로의 설명·규약은 활성 명령 중에도 갱신할 수 있다. tools/team은 파일 후보를 만들기 전과 실행 직전에 담당 영역을 검사하고 준비 단계의 경로 투영을 후보 안에서 재사용한다. 공유 예약의 파일 상태는 대기 후 재검사하고 같은 대기 호출의 배열을 재사용한다. CommandApproval의 일시적인 MutationTargets는 기존 AST가 관측한 경로만 전달하고 JSON·SQLite에 보관하지 않는다. 셸 실행기는 해당 경로를 종료 관측까지 보유하고, 조립 어댑터가 커밋 이후 Operation.AffectedPaths에 연결해 영향 가능 관측으로 공유한 뒤 참조를 해제한다.

discuss는 질문·답변·피드백에 사용하며 같은 사용자 대화의 reply_to와 실제 멤버 이름을 검증해 team_discussion으로 공개한다. 발신자와 BasedOn은 실행기가 정하고 본문·참조 메타데이터를 기존 이벤트 payload에 저장한다. 동료를 실행·복구하거나 답변을 기다리지 않으며 셸 예약과도 독립적이다. channel.Attention과 team의 경로 관련성 판단은 모두가 읽는 기록과 추가 판단을 알릴 기록을 구분한다. 답글 작성자는 공용 창의 메타데이터를 재사용하거나 SQLite에서 본문 복원 없이 확인한다. 본문은 기존 공유 참조를 사용하고 새 테이블·메시지별 goroutine·추가 요약 호출을 만들지 않는다. 성공한 discuss·set_work 이벤트의 memory_role은 공개 순서를 참조하고 memory_text에 원문을 복제하지 않는다. memory_sources는 해당 공개 원문을 배치 복원해 압축에 제공한다. 수신 페이지의 16MiB 예산에는 본문·공유 규약·경로를 함께 계산한다. 공용 기록 조회는 동시 동일 요청의 복원을 합치고 큰 결과는 실행 중에만 공유한다. 표시 어댑터는 reply_author·mentions를 자연스러운 수신 대상 표시에 사용하며 live·history가 같은 포맷을 따른다.

실제 변경·실패는 team_observation, 변경 경로나 실패가 없는 명령 상태는 team_status로 공개 로그에 투영한다. 원본 출력·diff·Thinking을 복제하지 않고 변경 경로·명령 상태·원본 참조만 공유한다. 개인 결과가 확정된 뒤 공용 원본 참조와 관측을 한 트랜잭션으로 저장한다. 실행 중인 명령 ID만 중복 판정에 보유하고 첫 종료 관측에서 해제한다. 완료 보관 결과의 재조회는 새 관측을 만들지 않는다. team_status는 공용 원본과 수신 위치를 보존하고 개인 이력에는 추가하지 않는다. 멤버별 모든 활성 명령의 ID·상태를 현재 관측에 투영한다. 활성 명령이 없으면 최신 완료 관측 한 건을 사용하며 새 변경·실패가 있으면 이전 일반 상태를 해제한다. 알림·도구 없는 응답의 추가 판단 사유에서 제외한다. 일반 진행 발언도 다음 정상 판단에 전달하며 단독으로 추가 추론을 요구하지 않는다. 사용자 지시·관련 업무 변경·실패와 자신에게 주의가 필요한 토론은 판단을 재개한다. 범위를 모르는 변경·공유 규약은 보수적으로 알린다. TUI는 두 자동 관측을 큐와 스트림 병합 경계에 넣지 않으며 JSONL과 저장 원문은 유지한다. 마지막 판단의 수신 위치보다 뒤에 다른 동료가 관련 경로를 변경했거나 영향을 주었을 수 있으면 오래된 후보를 거절한다. 독립 변경은 그대로 실행한다.

현재 담당 현황은 선택적 ContextObservations로 현재 대화를 함께 받아 요청에만 주입하며 개인 이력에 추가하지 않는다. 원문이 남아 있는 업무·의견은 참조 중심으로 투영하고 압축 뒤 원문이 빠지면 짧은 내용을 보충한다. 멤버별 투영은 현재 스냅샷과 수신 내용의 비트마스크가 같을 때 재사용한다. 현재 영역은 설명·규약 미리보기와 전체 내용 해시를 보유하며, 해시·미리보기 구성은 공유 잠금 밖에서 수행한다. Room은 멤버·가용 자리·담당 revision·최신 상태·최근 의견 순서의 같은 스냅샷을 사용해 불변 현황 하나를 재사용한다. 캐시가 비면 전용 구성 잠금으로 중복 포맷을 합치고 문자열 구성 중에는 공유 상태 잠금을 놓는다. 반복 주입 예산과 전체 선언·경로 보존 계약은 [팀 모드](team.md#자원-관리)를 따른다. agent/context는 마스킹·접두문 투영을 최대 256KiB로 재사용하며 원문·모델·비밀키 revision 변경 때 무효화한다. Inbox는 공개 대화와 같은 경계에서 현황을 캡처하고 BeforeDecision은 해당 수신 위치와 불변 현황을 고정한다. 압축 직후 수신에서도 같은 절차를 적용한다. 두 연결은 팀 조립에서만 제공한다. ContextBatch는 사용자 지시의 공개 순서 번호도 전달한다. execution은 새 공개 지시와 단일 모드 PollSteering 수신을 공통 instructionRevision으로 추적하고 생성 후·실행 묶음 사이·후보 준비와 승인 후에 비교한다. 지시 번호와 로컬 수신 횟수를 같은 값 공간으로 섞지 않으며 revision은 실행 메모리에만 둔다. 병렬 준비 작업자는 입력을 소비하지 않고 합류 후 소유자가 한 묶음을 검사한다. 공용 대화 본문은 모델 판단 경계에서 수신하며 준비·승인·실행 묶음 사이에는 도구 어댑터의 CheckUserDecision으로 최신 지시만 확인한다. 미완료 native 호출·결과 묶음 사이에 동료 대화를 삽입하지 않아 정상 페어와 인코딩 재사용을 유지한다. 자동 승인 묶음의 입력 조회는 파일 개수에 비례하지 않는다. 대화형 승인에서 새 지시를 받으면 후속 승인 요청도 중단한다. Peer는 DecisionUserSequence를 모델 판단 시점에 고정하고 파일 효과·팀 도구가 같은 검사를 적용한다. 저장된 수신 위치를 복원하면 그 위치에 포함된 현재 사용자 지시 번호도 복구한다. 퇴장은 아직 검토하지 않은 관련 알림을 확인하며 공개 게시와 상태 전이를 같은 직렬 경계에서 처리한다. 현재 영역은 활성 인원만 보유하고 관련 변경 인덱스는 모든 활성 판단이 소비한 순서까지 제거한다. 확인 지점이 전진할 때만 정리하고 새 변경이 없으면 전체 변경 검색을 생략한다. 퇴장·취소·실패는 영역·명령 상태 참조를 정리한다.

팀 입력은 SteeringReady 알림에서 PollSteering의 캡처된 요청을 소비한다. 입력 알림이 없는 비대화형 요청은 타이머를 만들지 않는다. 닫힌 알림 채널은 비활성화한다. 답변 경계의 재개 판단은 메모리의 멤버별 알림 순서와 DecisionCursor를 비교하고 SQL을 실행하지 않는다. 새 발언 수신과 추가 추론 여부를 구분하며 알림 채널은 용량 한 건으로 합친다. 전체 대기는 사용자 지시 번호와 최근 64개 작업·실행 근거 해시를 기준으로 실제 진행 경계당 한 번만 재판단하고 변화 없이 다시 전체 대기하면 정체를 반환한다. 활성 명령·미처리 알림은 전체 대기로 취급하지 않는다. 같은 실패 근거는 공개 원문을 남기고 반복 알림만 억제하며 실제·가능한 파일 효과는 재검토를 유지한다. 새 사용자 지시는 요청이 공유하는 세대 신호를 닫아 접근 대기를 중단한다. 다음 유효한 판단이 같은 신호를 재사용하며 도구별 감시 고루틴·타이머를 만들지 않는다. 새 개인 DB 생성은 멤버 수용 경계 안에서 수행하되 공용 게시 잠금은 잡지 않고, 초대 확정 전에 사용자 지시를 다시 검사한다. 미게시 새 등록의 폐기는 작업 이력이 없는 정확한 세션만 대상으로 하며 기존 세션과 게시된 등록은 보존한다.

팀의 숨긴 Thinking은 원문·실제 usage를 계속 소비하고 종료·취소 시 마스킹한 개인 기록과 정상 continuation을 보존한다. 표시용 증분 마스킹과 Thinking 전용 스트림·화면 이벤트는 생성하지 않는다. 일반 모드의 Thinking 표시 경로는 같은 루프의 기본 동작이다.

실행기는 실제 usage와 대응하는 단조 시계 경과 시간을 전달한다. 순간 TPS를 별도로 전달하거나 저장하지 않는다. agent/record/throughput.go의 작은 값 누산기가 단일·팀·완료 표시의 같은 가중 평균 계산을 소유한다. 화면은 완료 합계와 최신 진행 표본을 분리해 중복 집계를 피하고 새 요청·연결 채택 시 측정 상태를 비운다. 전송용 정밀 시간은 JSON에서 제외하고 DB에는 기존 generation의 밀리초 기록만 보관한다.

명령 등록부는 요청 수명(registry.go), 시작(command_start.go), 완료·출력 보관(command_finish.go), 조회·대기·취소(command_control.go)를 구분한다. 완료 신호는 프로세스 종료·결과 저장·캡처 정리가 끝난 뒤 발생하며 등록부의 잠금과 출력 보관 직렬화는 같은 소유자를 유지한다. 변경·퇴장·접근권 해제의 실행 중 여부 검사는 ActiveID로 완료 신호만 읽는다. 모델 호출 전 HasCommands는 실행 중 또는 미수집 완료 결과의 존재만 확인한다. AppendCompleted는 호출자가 제공한 작은 배열에 완료 ID·원명령·보관 오류만 채우고 기존 ID 순서를 유지하며 시간·출력 상태를 계산하지 않는다. 완료 수집은 최대 16건의 스택 버퍼를 사용하고 필요한 결과 배열만 할당한다. 보관 콜백에는 시작 시 원명령을 직접 전달해 전체 목록을 재조회하지 않으며 수집 시 마스킹한 명령은 결과와 메타데이터에서 공유한다. 이 경로는 상태 배열·정렬·출력 시각 조회·명령 문자열 마스킹을 수행하지 않으며 상세 목록은 실제 상태 표시와 결과 수집에서만 구성한다.

agent/run의 Run은 실행 준비(start.go), 대화 입장과 정리(conversation.go), 결정 반복(loop.go)을 조율한다. emitter.go는 요청 식별·시각·확정 원본 참조의 보강과 순서 있는 전달을 소유하고, 성공한 전달 뒤 logging.go가 중앙 진단을 기록한다. 스킬 탐색 지침 문구는 agent/prompt/skills.go가 소유하며 start.go는 도구가 제공되는 요청에만 합성한다. 종료는 명령을 합류한 뒤 결과를 확정하고 실행 범위 생성기를 마지막에 해제한다.

실행 사실과 파일 관측은 agent/tool, 실행 순서는 agent/run, 문맥 조립은 agent/context가 소유한다. 영속 실행 기록은 agent/record가 정의하므로 저장 어댑터·이력·TUI 투영은 실행 루프 구현을 참조하지 않는다. storage/content는 SQL 없이 분리·복원하며 SQLite가 같은 트랜잭션의 쓰기 콜백과 조회 후 복원 콜백을 제공한다. 승인 전경은 파일 뷰어·전송 대기보다 먼저 입력을 처리하고 결정을 마치면 보류한 파일 화면·초안을 복원한다. 팀 화면 컴포넌트는 저장소·명령 실행기·세션 이벤트 루프를 참조하지 않고 값과 표시용 viewport만 갱신한다. 세션은 모드 저장·입력 이력·시계·연결 상태의 부수 효과를 맡는다. 상세 도구 결과의 원본 식별과 블록 교체는 viewport가 소유한다. TUI 하위 컴포넌트는 session을 역참조하지 않으며 상태 변경은 단일 이벤트 루프에서 수행한다. Compaction은 사실을 소비해 요약하며 별도 실행 원장을 만들지 않는다. SQLite는 저장 기능 전체가 아니라 저장 계약의 구현 수단이다. SQLite의 팀 어댑터는 identity·channel 자료형을 사용하고 Room 실행 구현에 의존하지 않는다. SQLite Store는 공용 원본 참조와 실제 획득한 캐시 사용권을 구분한다. 실행용 개인 세션만 사용권을 획득·반환하며 과거 기록 조회 연결의 종료는 활성 캐시 수명을 바꾸지 않는다. 같은 DB의 원자적 journal 커밋·payload 복원은 같은 어댑터에 두며, 커밋 영수증·개인 원본 조회 핸들·참조 복원은 각각의 파일이 소유한다.

## 의존성과 배포

제품은 [go.mod](../go.mod)의 Go 1.27.2 이상, 개발 도구는 [devtools/go.mod](../devtools/go.mod)의 Go 1.27.3 이상을 사용한다. 프로젝트 라이선스는 [Apache License 2.0](../LICENSE), 저작권 고지는 [NOTICE](../NOTICE), 외부 구성요소는 [외부 소프트웨어 고지](../THIRD_PARTY_NOTICES.md)를 따른다. 관리자가 단독으로 개발·병합·배포하며 외부 코드 기여는 [기여 정책](../CONTRIBUTING.md)에 따라 받지 않는다.

배포 의존성을 갱신할 때 macOS·Linux의 amd64·arm64 대상으로 `go list -deps -json ./cmd/meca`를 확인하고, 선택된 모듈의 LICENSE·COPYING·NOTICE·AUTHORS·PATENTS와 추가 라이선스 고지를 함께 갱신한다. Go 런타임의 라이선스도 포함한다. 개발 도구와 테스트 전용 모듈은 제품 실행 파일의 의존성 목록과 구분한다.

| 의존성 | 용도 |
| --- | --- |
| tcell/v3·uniseg | 터미널·입력·마우스·문자 경계와 셀 폭 |
| goldmark/v2 | 응답·Thinking·파일 뷰어의 Markdown AST를 TUI span으로 변환 |
| go-toml/v2 | TOML 파싱·직렬화 |
| godbus/dbus/v5 | 취소 가능한 Linux Secret Service IPC; macOS Keychain은 표준 os/exec 사용 |
| modernc.org/sqlite | CGO 없는 SQLite |
| klauspost/compress/zstd | 세션의 대용량 payload를 Zstandard로 압축·복원 |
| bytedance/sonic | 정렬된 키·문자열 검사를 유지하며 모델 요청 JSON을 인코딩 |
| goccy/go-json | 저장 레코드의 JSON 직렬화·참조 분리·복원 |
| x/image | WebP·BMP·TIFF 디코딩; 표준 이미지 코덱과 함께 사용 |
| x/sys·x/term | OS 파일 어댑터와 터미널 확인 |
| mvdan.cc/sh/v3 | 실행 없는 셸 AST 분석과 정적 단어 확장 |
| fsnotify | 파일 탐색 화면의 활성 디렉터리 변경 알림 |
| connectrpc.com/connect/v2·protobuf | native gRPC·gRPC-Web·Connect와 공통 명령/이벤트 계약 |
| coder/websocket | 인증한 WebSocket 제어와 이벤트 구독 |

테스트 전용 의존성은 go.uber.org/goleak(고루틴 정리 검사), google/go-cmp(복잡한 값 비교), creack/pty(실행 파일의 터미널 통합 검사)다. 일반 실행 파일에서는 이 패키지들을 import하지 않는다.

CLI 분기는 flag, 병렬 도구 합류는 sync.WaitGroup, 재시도 대기는 time.Timer를 사용한다. 검색은 strings·unicode로 부분일치를 우선하고 결과가 없을 때 문자 순서를 유지하는 축약 일치를 찾는다. 메타데이터 재사용은 연결 준비가 소유한 마지막 GET 응답과 선택한 Ollama 모델의 typed 응답으로 한정한다. 이 사용처에는 별도 CLI·동시성·캐시 프레임워크를 두지 않는다.

Go 빌드만으로 MECA 실행 파일을 만든다. 제품 빌드·실행에 Docker·Bubblewrap·별도 샌드박스 설치를 요구하지 않는다. 빌드·설치·삭제 명령은 [README](../README.md#설치)를 따른다. 소스 설치 스크립트는 Go가 없을 때만 공식 SDK를 체크섬 검증 후 임시 설치하며 작업 후 정리한다. Go 최소 버전을 바꿀 때 `scripts/install/go.sha256`의 네 플랫폼 검증값도 공식 배포 정보와 함께 갱신한다. macOS·Linux 릴리스는 [Release 워크플로](../.github/workflows/release.yml)와 GoReleaser 설정으로 검사·빌드 후 자동 게시한다. 명시적 major·minor와 같은 날짜의 갱신 정책은 [릴리스 가이드](releases.md)가 소유한다.

공식 빌드·소스 설치는 `godbus_no_tcp` 태그로 DBus의 TCP·nonce-TCP 구현을 제외한다. 수동 실행 파일은 `go build -tags=godbus_no_tcp -trimpath -o bin/meca ./cmd/meca`로 만든다. 아래 개발·검증 명령은 같은 셸에서 먼저 `export GOFLAGS='-tags=godbus_no_tcp'`를 설정해 실행한다. 테스트 내부의 자식 `go build`에도 전달되며 전역 Go 설정은 바꾸지 않는다. GoReleaser는 환경변수와 별개로 빌드 태그를 명시한다. 지원 범위는 [Linux 자격 증명 전송](configuration.md#linux-자격-증명-전송)을 따른다.

### 개발 도구

검증·배포 도구는 [devtools/go.mod](../devtools/go.mod)의 별도 모듈에 버전을 고정한다. 제품의 go.mod·go.sum과 의존성 해석을 분리하며 MECA 바이너리에 포함하지 않는다. 도구 실행에는 Go 1.27.3 이상이 필요하다. 다음 명령은 저장소 루트에서 실행한다. 최초 실행 시 Go가 필요한 도구를 내려받고 빌드 캐시에 저장하므로 전역 설치는 필요 없다.

| 도구 | 역할 | 실행 |
| --- | --- | --- |
| diagnosticscheck | 중앙 진단 선언·오류 생성·임의 로거 우회 검사 | `go -C devtools run ./cmd/diagnosticscheck -C=.. ./...` |
| Staticcheck | Go 코드의 오류·API 오용·불필요한 연산 검사 | `go tool -modfile=devtools/go.mod staticcheck ./...` |
| govulncheck | 알려진 취약점과 실제 호출 경로 검사 | `go tool -modfile=devtools/go.mod govulncheck ./...` |
| deadcode | 실행 진입점에서 도달하지 않는 함수 탐색 | `go tool -modfile=devtools/go.mod deadcode ./cmd/meca` |
| benchstat | 반복 벤치마크의 시간·할당량·저장 크기 비교 | `go tool -modfile=devtools/go.mod benchstat before.txt after.txt` |
| GoReleaser | 배포 빌드·압축·체크섬·패키지 생성 도구 | `go tool -modfile=devtools/go.mod goreleaser --help` |
| nFPM | Linux 설치 패키지 생성 도구 | `go tool -modfile=devtools/go.mod nfpm --help` |

Staticcheck는 go vet와 함께 사용한다. deadcode는 삭제 후보를 제시하며 `go tool -modfile=devtools/go.mod deadcode -test ./...`로 모든 패키지의 테스트까지 함께 대조한다. 테스트 전용 함수·인터페이스 구현·운영체제별 빌드 조건을 확인한 뒤 정리하며 호출되지 않는다고 표시됐다는 이유만으로 자동 삭제하지 않는다.

다른 운영체제의 코드를 검사할 때는 먼저 호스트용 도구를 빌드한 뒤 분석 대상 환경을 지정한다. `GOOS=linux go tool ...`은 도구 자체도 Linux용으로 빌드하므로 macOS에서 실행할 수 없다. 다음 검사는 Linux 코드의 분석이며 Linux에서 실제 실행한 검증과 구분한다. 도구 버전을 갱신하면 호스트 도구도 다시 빌드한다.

```sh
go build -modfile=devtools/go.mod -o bin/devtools/ honnef.co/go/tools/cmd/staticcheck golang.org/x/vuln/cmd/govulncheck golang.org/x/tools/cmd/deadcode
GOOS=linux GOARCH=amd64 ./bin/devtools/staticcheck ./...
GOOS=linux GOARCH=amd64 ./bin/devtools/govulncheck ./...
GOOS=linux GOARCH=amd64 ./bin/devtools/deadcode -test ./...
```

govulncheck는 Go 취약점 데이터베이스를 조회한다. 보고서의 영향 버전·호출 경로·수정 버전을 확인하고, 취약점이 발견된 경우 의존성 변경과 관련 테스트를 함께 검토한다. 검사에 실패하거나 데이터베이스에 접근할 수 없었던 경우를 취약점 없음으로 처리하지 않는다.

GoReleaser는 [.goreleaser.yaml](../.goreleaser.yaml)의 macOS·Linux 아카이브를 생성한다. `check`와 게시 없는 `release --snapshot --clean`으로 로컬 검증하며 `main` 푸시는 자동 게시를 시작한다. 릴리스 준비·게시 경계는 `go test ./scripts/release`로 검사하고 워크플로 변경 시 Actions 문법도 확인한다. nFPM은 Linux 설치 패키지가 필요할 때 사용하는 도구로 등록되어 있다. 현재 배포 형식은 tar.gz이며 제품 실행·일반 빌드·테스트에는 이 도구들이 필요하지 않다.

버전 갱신은 `go get -modfile=devtools/go.mod -tool <패키지>@<버전>`으로 수행한다. 이후 `go -C devtools mod tidy`와 `go -C devtools mod verify`를 실행하고 두 개발 도구 파일의 변경을 검토한다. 도구 요구 버전이 제품의 의존성이나 최소 Go 버전을 바꾸지 않도록 유지한다.

## 변경 검증

작업 범위의 집중 검사를 먼저 실행하고 공유 계약이 바뀌면 전체 검사로 넓힌다. 동시성·취소·자원 수명 변경에는 레이스 검사를 포함한다.

```sh
export GOFLAGS='-tags=godbus_no_tcp'
gofmt -l cmd internal
go test ./...
go test -race ./...
go vet ./...
go tool -modfile=devtools/go.mod staticcheck ./...
go -C devtools run ./cmd/diagnosticscheck -C=.. ./...
go build ./...
go mod verify
```

순수 상태·파서·승인·저장·Provider 변환은 작은 fixture와 httptest로 검증한다. 파일 검사는 t.TempDir, TUI는 가상 터미널과 실제 키 입력을 사용한다. 파일 탐색 controller는 물리 I/O가 끝나기 전까지 슬롯을 유지하며 종료 시 자체 worker와 결과 전달을 합류하고 슬롯·sink·문맥 참조를 해제한다. 화면·입력·상태 검사는 `screenFixture`를 사용하고, 저장·복원·도구 실행 검사는 실제 세션 DB와 실행기를 준비하는 `fixture`를 사용한다. 화면 전용 fixture는 저장 이력을 모두 읽은 상태로 시작하며 DB·프롬프트·실행기를 생성하지 않는다. 모델 응답을 고정한 검사는 실제 모델의 완수율이나 추론 품질 검증과 구분한다.

에이전트 실행·HTTP 생성·TUI 테스트는 패키지 종료 시 goleak으로 남은 고루틴을 검사한다. 구조체 복원과 대화 투영은 go-cmp로 필드 차이를 확인한다. cmd/meca의 통합 검사는 필요한 경우에만 패키지당 실행 파일 하나를 빌드하고 패키지 종료 시 제거한다. 각 실행은 별도 HOME·프로젝트·서버를 사용한다. PTY 검사는 실제 실행 파일을 시작해 터미널 크기 변경, Ctrl+C·!quit 종료, 터미널 설정 복원과 DB 잠금 해제를 확인한다. 설치 PTY 검사는 PATH가 필요한 대화형 설치만 동일 PID의 로그인 셸로 전환하는지, 전환 전에 설치 잠금이 풀리는지와 --no-shell·비대화형 동작을 확인한다. 소스 설치 검사는 오프라인 SDK fixture로 빌드·임시 SDK 정리·호출 환경 복원·설치 순서를 확인한다. PTY의 소유 프로세스가 종료 후 상태를 검사하고, 실패·시간 초과에는 해당 프로세스 그룹과 읽기 작업을 정리한다. `go test -short`에서는 실행 파일을 빌드하는 통합 검사를 생략한다.

외부 전송 검사는 모의 실행 파일과 loopback HTTP만 사용한다. Default·All Mighty의 매 호출 직접 승인·거절·비대화형 승인 불가와 배치의 실행 전 중단·native 결과 짝·승인 출처를 확인한다. 실제 외부 업로드를 발생시키지 않는다. 확인 키는 소문자 n 무시·대문자 N 취소를 설정·파일 전송 화면에서 검증한다.

```sh
go test ./cmd/meca -run TestTerminalExecutable -v
go test ./cmd/meca -run TestInstallShellExecutable -v
go test ./internal/commands/install -run TestSourceInstaller -v
go test ./internal/tui/session -run TestFileWatch -v
```

문서 변경은 실제 코드·기본값·로컬 링크·참조한 테스트와 벤치마크 이름을 대조한다. 코드의 헤더·주석·테스트는 해당 파일의 책임과 실제 동작을 설명하도록 유지한다.

### 서버 계약 검증

```sh
GOFLAGS='-tags=godbus_no_tcp' go test -race ./internal/commands/server ./internal/serverapi/... ./internal/storage/serverrequests ./cmd/meca
GOFLAGS='-tags=godbus_no_tcp' go test ./internal/serverapi -run 'TestOpenAI|TestNativeHeartbeat|TestStreamingInputDeadline'
go test ./internal/commands/server ./internal/serverapi/detail -run '^$' -bench 'Benchmark(StreamProjection|ToolProjection|EventReplay|DetailPages)$' -benchmem
```

서버 검사는 모의 모델과 실제 SQLite·도구·TLS 연결을 사용한다. SSE·WebSocket·native gRPC·gRPC-Web·Connect에서 완료 전 스트리밍·중복 접수·승인·취소·재접속·저널 순서와 도구 효과를 확인한다. 실행 파일 검사는 TTY 없는 시작·모델 저장·세션 재열기·요청 실행·SIGTERM 종료와 팀 실행 조립을 확인한다. Proto 생성 명령·생성 파일 예외·네트워크 한도는 [서버 API](server.md)를 따른다. 실제 브라우저 앱·원격 프록시·실모델 품질 검증과는 구분한다.

구독 취소는 이미 읽은 묶음의 다음 이벤트와 최초 snapshot도 중단하는지 확인한다. 시작 중 토큰 파일 교체에도 인증과 마스킹이 같은 토큰을 사용하는지, 4KiB 오류의 JSON 이스케이프 후에도 완료 기록을 재시작 후 조회·중복 확인할 수 있는지 검증한다. 저장 한도를 넘는 완료 기록은 기존 기록을 유지한 채 거부해야 한다.

설정·세션 변경은 서비스가 하나의 변경 상태를 예약한 뒤 I/O를 수행하고 짧은 잠금 안에서 결과만 확정한다. 모델 준비·키링·DB·설정 파일·접수 조회·자원 정리와 reader 합류는 상태 잠금 밖에서 실행한다. 종료는 새 등록을 막은 뒤 접수·준비·설정 I/O와 실제 요청을 취소·합류한다. 변경 중 다른 변경과 새 원본 조회는 거부하되 state·이벤트·작업 상태 조회와 활성 요청 취소는 계속 처리한다. 설정의 현재 값은 서비스 한 곳이 소유하고 조립 콜백에는 불변 설정·모델 준비 기본값을 전달한다.

Start는 조회 뒤 변경 상태 예약에 실패하면 이미 확정된 동일 접수를 다시 확인하고 원래 작업을 반환한다. 현재 프로세스의 접수 인계 ID를 보관해 accepted를 재시작 후 interrupted와 구분하며, 접수와 실행 사이의 조회도 실제 소유권을 반영한다. 아직 확정되지 않은 접수나 다른 변경에는 conflict를 반환할 수 있다.

모델 준비 콜백은 클라이언트와 메타데이터만 반환한다. 서비스가 commands/modelchange.Commit으로 세션·키링·선택적 기본값 저장을 확정한 뒤 클라이언트를 인계받는다. 저장 오류는 취소와 독립된 제한 시간의 읽기로 확인하고 실제 선택이 일치하면 확정된 결과로 적용한다. 저장 여부를 확인하지 못하면 이전 모델도 해제하고 재연결을 요구한다. 기본값 저장만 실패한 경우의 세션 유지·자격 증명 보존은 같은 Commit 계약을 따른다. 세션 자원은 먼저 선택 상태에서 분리하고 읽기 context를 취소한 뒤 합류·정리한다. 실패해도 분리한 상태를 게시하며 새로 연 자원의 정리 오류도 합친다.

상세 조회는 commands/server.OpenDetail이 참조와 현재 승인을 검사하고 원본 읽기를 대여한다. API 어댑터는 대여를 반환하고, 세션 정리는 모든 대여가 끝난 뒤 API 캐시의 해당 세션과 실제 저장 자원을 닫는다. DTO에는 허용된 공개 필드만 옮기며 내부 구조체를 통째로 직렬화하지 않는다. 인증·Host·Origin·CORS는 security.go, 생성·입장 한도·요청 취소·종료 합류는 transport.go가 담당한다. 동일 WebSocket에서 응답·실행 이벤트·취소·unwatch·구독 교체를 함께 검증하고 서버 API 패키지 종료에서 goleak으로 worker 누수를 검사한다.

이벤트 조회는 보관된 연속 sequence로 시작 위치를 계산하고 필요한 페이지만 복사한다. 알림 채널은 실제 대기자가 생겼을 때 생성한다. 전송 투영의 마스킹은 구독자가 소유한 배열·선택 값만 수정하며 공통 보관 이벤트를 변경하지 않는다. 상세 조회 벤치마크는 1MiB 원문을 64KiB 페이지로 끝까지 읽는 동안의 시간·할당·원문 복원 횟수를 측정한다. 최근 상세 한 건의 재사용은 세션 수명과 마스킹 revision에 묶이며 전송 구독 수만큼 원문을 복제하지 않는다.

### 서버 데이터와 자원 소유권

신규 영속 할당은 request-owned Reservation을 받는다. 세션의 최종 link, 접수 트랜잭션 Commit, OS 키 Set 직전에 Publish로 예약을 확정한다. 그 전에 끝난 작업은 독립된 최대 5초 정리로 슬롯을 한 번 반환한다. 카운터 쓰기가 불확실하면 반환도 반복하지 않는다. 프로세스 중단·게시 시도 후 오류·은퇴 키의 예약은 자동 재조정하지 않는다. 할당 90%부터 중앙 storage_capacity_low에 count·limit을 남긴다. allocated를 임의로 삭제/재설정하거나 백업 키를 일괄 삭제하지 않는다. 임의 시점의 DB 백업 보존과 과거 OS 키의 자동 회수는 함께 보장할 수 없으므로 보존 정책 변경은 별도 운영 결정이다.

실행 식별자를 획득하지 않은 queued 자식의 확정 실패는 그 자식이 잡은 부모 head만 같은 트랜잭션에서 해제한다. 실행 식별자가 있거나 완료 기록이 불확실하면 점유를 보존한다. 동일 실패 저장을 다시 수행해도 이후 자식의 head를 지우지 않는다. Chat 외부 이력은 ClientContext로 요청에만 전달한다. Responses 외부 이력은 이어가기 복구를 위해 대화가 보존하며, 팀에서는 client_context 출처의 공용 관측으로 한 번 저장하고 개인 실행에는 인증된 원문 참조를 기록한다. 공용 입력 전의 초기 수신 커서 때문에 참조가 빠지지 않도록 초기 등록을 분리한다.

서버 조립부는 프로젝트별 독점 lease를 서비스 자원보다 먼저 확보하고 마지막에 해제한다. 전역 설정은 `config.Update`가 `config-write/`의 프로세스 공통 잠금 안에서 최신 파일을 읽고 항목별 함수를 적용한 뒤 원자적으로 저장한다. TUI·서버는 이 계약으로 언어·승인 모드·기본 모델을 갱신하며 반환된 설정을 채택한다.

`serverapi/admission`은 입력 바이트와 큰 읽기 작업의 입장·취소·해제를 소유한다. 입력 예약은 합계 32MiB다. HTTP·RPC·WebSocket의 일반 본문은 실제 읽은 크기에 따라 32KiB 단위로 늘리고, 압축 입력은 해제 전에 17MiB를 예약한 뒤 실제 RPC 명령 크기로 조정한다. 호환 요청은 접수가 끝나면 입력 예약을 반환한다. RPC watch는 4KiB를 검사한 뒤 예약을 반환하고, 소켓은 프레임 입력을 30초로 제한하며 처리한 프레임을 다음 읽기 전에 해제한다. 큰 원문 읽기는 API 전체에서 한 번에 하나만 허용하고 완료 JSON·SSE 전송까지 슬롯을 보유한다.

`history.ReadBounds`는 일반 이력 페이지의 32MiB 복원 예산·16KiB 표시 접두부와 상세/완료 원문의 512MiB 복원을 구분한다. SQLite는 선택한 payload의 원문·저장 바이트 수를 복원 전에 확인하며 팀 원본 참조도 같은 예산을 사용한다. 제한된 payload 조회는 크기 확인과 조건부 body 읽기를 한 SQL 문장으로 처리한다. 메시지 미리보기는 숨겨진 continuation을 복원하지 않는다. 큰 원문은 processmemory의 공유 payload 예약을 획득하고 완료 전송 또는 상세 마스킹·spool 쓰기가 끝날 때 반환한다. 캐시 적중 페이지는 원문을 다시 읽지 않는다. 초과는 원문을 보존한 resource_exhausted이며 export는 API 조회 예산을 사용하지 않는다. 상태·원문 조회와 슬롯 대기는 eventstream.Await가 합류 가능한 worker로 분리하며 작성자는 heartbeat와 전송 순서를 소유한다.

세션 목록의 최초 검색은 최대 101건의 표시 메타데이터 heap과 ID·시각 참조만 유지한다. 후속 페이지는 해당 ID의 DB만 읽으며 별도 영속 카탈로그나 대화 사본을 만들지 않는다. 정렬 참조는 2분 동안 최대 8건·합계 8MiB, 한 건은 2MiB로 제한하고 변경이 없는 순서는 공유한다. 만료는 다음 접근에서 정리하며 DB 핸들이나 정리 worker를 보유하지 않는다. 커서는 현재 프로세스·프로젝트·정렬 참조·위치에 묶이고 갱신되거나 사라진 세션은 건너뛴다. 새 조회는 현재 순서를 다시 확인한다. 입력/복원 한도는 최종 RSS나 OS 메모리 quota를 뜻하지 않는다.

`storage/capacity`는 private 할당 카운터와 독점 lease를 소유한다. 새 세션·접수·OS 암호화 키 할당 전에 카운터를 확정하며 불확실한 부작용은 예약을 반환하지 않는다. 프로젝트당 대화 4096개, 전체 서버 접수 100000개, 새 암호화 키 버전 65536개를 허용한다. 카운터 최초 생성 시 현재 DB·파일의 해당 항목 수를 포함하며 이미 보존된 과거 OS 키의 전체 개수를 추측하지 않는다. 자격 증명 DB의 현재 참조는 256개로 제한하고 SQLite 문장 하나가 경쟁 삽입에도 상한을 유지한다. 새 할당에는 여유 공간 1GiB를 요구한다. 기존 조회·중복 확인·완료 기록과 백업 키는 자동 삭제하지 않는다.

세션 DB의 512MiB 상한은 실제 파일 헤더의 페이지 크기에서 계산해 DSN에 넣으므로 연결이 교체되어도 유지된다. 이미 더 큰 파일은 기존 페이지 수 아래로 줄이지 않는다. WAL은 256페이지마다 자동 checkpoint를 시도하고 정리 후 journal 보관은 16MiB로 제한한다. 활성 읽기 중 WAL 성장과 외부 프로그램의 디스크 사용은 별도다.

storage/serverrequests는 프로젝트 scope별 SQLite 연결 하나를 소유한다. 접수·실행 경계·종료를 별도 트랜잭션으로 확정하며 native 작업 조회는 같은 응답 행을 복원한다. 이어가기 head와 자식 예약을 한 트랜잭션으로 묶고 확정된 종료 값은 재변경하지 않는다. 전체 입력 해시는 길이 구분 형식으로 계산하고 입력 본문은 저장하지 않는다. WAL·FULL 동기화와 즉시 쓰기 트랜잭션을 사용하며 schema·application ID·DB 소유 프로젝트 scope를 확인한다. 식별자는 DB 행에 한 번만 저장하고 호환 상태는 저장소 전용 메타데이터 형식으로 보관한다. 기존 JSON 접수는 자동 변환하지 않고 시작을 거부한다. 모델 선택을 새 DB에 저장한 뒤 준비된 연결의 대여를 얻고 대화·요청마다 자기 핸들을 해제한다.

선택된 결과는 reader 대여로 DB를 재사용하고 과거 결과는 읽기 전용 DB로 읽는다. 짧은 범위는 타임라인 인덱스, 긴 범위는 메시지 INTEGER PRIMARY KEY의 시작·끝 범위와 source 순서로 조회한다. 미완성 메시지는 payload 복원 전에 제외하고 공개 content만 읽되 원본 payload 길이·해시 검증은 유지한다. 실패한 Chat의 버릴 원문은 읽지 않고 cancel의 결과는 한 번만 읽는다.

비스트림·팀·중복 스트림은 작업 완료 신호로 기다린다. 활성 상태 조회는 context·worker 등록 없이 잠금 안에서 값만 복사한다. 이벤트 로그는 지연 할당한 순환 버퍼에 공개 투영만 보관하며 삭제 칸의 참조를 해제한다. JSON 마스킹은 문자열 토큰만 바꾸고 숫자·컨테이너를 다시 만들지 않는다. redaction.Registry는 불변 비밀값 목록의 snapshot을 얻은 뒤 긴 문자열 처리를 잠금 밖에서 수행한다. credentials.Vault가 이를 포함하고 OS 저장 I/O만 추가한다. Provider의 HTTP/1.1 유휴 연결은 전체·호스트당 8개, 30초까지 재사용한다.

서버의 입력 파싱·델타 작성·접수·조회·페이지 복원은 아래 벤치마크로 현재 환경에서 측정한다. 로컬 HTTP 부하는 모델 추론을 포함하지 않으며 B/op는 누적 할당량이다. 최고 힙·RSS·파일 동기화 지연·다중 구독자의 총 메모리는 별도로 측정한다.

```sh
GOFLAGS='-tags=godbus_no_tcp' go test -p 1 ./internal/serverapi/openai ./internal/commands/server ./internal/storage/sqlite ./internal/storage/serverrequests ./internal/serverapi/detail -run '^$' -bench 'Benchmark(ParseMessages|OutputAppend|Stream|SubmissionDigest|EventPublishSteady|ActiveResponseState|ResponseOutput|ResponseLifecycle|DetailPages)$' -benchmem -count=2
GOFLAGS='-tags=godbus_no_tcp' go test ./internal/provider/transport -run '^$' -bench BenchmarkIdlePool -benchmem -benchtime=30x
GOFLAGS='-tags=godbus_no_tcp' go test ./internal/serverapi/openai -run '^$' -fuzz FuzzParseJSON -fuzztime=5s -parallel=2
```

세션 이력과 요청 중복 방지 기록은 자동 만료하지 않는다. 동기화·원본 무결성·승인 검사는 성능 경로에서도 유지하며, 실행 동시 수용량과 네트워크 연결 한도는 [서버 API](server.md)가 소유한다.

메모리 경계와 느린 I/O의 heartbeat는 다음 집중 검사로 확인한다. B/op는 누적 할당이며 RSS와 구분한다. 스트림의 보류 공백은 연결당 64KiB까지만 보관하고 이를 넘으면 완성 원문 조회로 전환한다.

```sh
GOFLAGS='-tags=godbus_no_tcp' go test ./internal/storage/content ./internal/storage/sqlite -run '^$' -bench 'BenchmarkTextPrefix8MiB|BenchmarkServerPreview8MiB' -benchmem
GOFLAGS='-tags=godbus_no_tcp' go test -race ./internal/serverapi/... -run 'TestAwait|TestResultSlot|TestInputBudget|TestSocketReleases|TestWhitespaceBuffer'
```

파일 탐색은 nonblocking open 뒤 실제 파일 종류를 확인해 FIFO 대기와 종료 worker 정체를 막는다. 디렉터리 페이지는 선택한 항목만 복사하고 정책 필터에서 제거한 참조도 비운다. TCP 수용량과 HTTP/2 스트림 수용량은 HTTP 요청의 동시 처리 한도와 별도로 전송 경계에서 제한한다.

## 플랫폼 실행 검증

Linux·WSL·macOS에서 호스트 셸의 작업 디렉터리·환경·프로세스 정리와 필수 승인 거절을 검증한다. 임시 HOME에서 캐시 생성, 하드링크, 호스트 DNS 설정 파일, 자식 프로세스·로컬 HTTP, 시작 실패·비정상 종료·취소 후 부분 출력을 확인한다.

```sh
go test ./internal/tools/local/... ./internal/agent/run ./internal/commands/headless
```

실제 WSL에서는 일반 터미널과 execute_command에서 같은 도메인 조회·HTTPS 다운로드·패키지 설치를 대조한다. DNS 설정 파일 읽기와 숫자 IP의 로컬 HTTP 통과만으로 외부 DNS 성공을 판단하지 않는다. Linux Secret Service·실제 Provider·터미널 키 전달도 수행한 환경과 결과를 구분해 기록한다. 교차 빌드는 네이티브 실행 검증을 대신하지 않는다. [실행 경계](execution-boundary.md)의 호스트 권한 계약을 함께 확인한다.

Linux의 필수 DBus 회귀 검사는 실제 자격 증명 조회 경로를 새 테스트 프로세스에서 호출한다. 루프백 TCP·nonce-TCP 연결과 canary 전송이 없고 Unix 소켓 연결은 유지되는지 확인한다. 직접 어댑터의 Unix 제한은 태그 없는 빌드에도 적용하며 배포 태그 존재는 별도 빌드 메타데이터 검사로 확인한다. Unix 연결 검사만으로 키 저장·조회 성공을 판단하지 않는다.

```sh
GOFLAGS='-tags=godbus_no_tcp' go test ./internal/config/credentials -run TestCredentialDBusTransport -v
```

실제 Secret Service 검사는 별도의 Unix 세션 버스와 잠금 해제된 키 저장소를 준비한 Linux 환경에서 다음과 같이 실행한다. 현재 사용자 키 저장소 대신 격리된 서비스 환경을 사용한다. 임시 서비스 이름으로 저장·조회·삭제하고 정리하며, 기본 테스트에서는 이 opt-in 검사만 생략한다.

```sh
MECA_TEST_SECRET_SERVICE=1 GOFLAGS='-tags=godbus_no_tcp' go test ./internal/config/credentials -run TestCredentialSecretService -v
```

macOS Keychain은 별도 임의 서비스 이름으로만 검증하고 마지막에 해당 서비스의 키를 제거한다. 기본 테스트는 주입한 메모리 저장소·명령 경계로 실제 사용자 키를 건드리지 않는다.

```sh
MECA_TEST_KEYCHAIN=1 GOFLAGS='-tags=godbus_no_tcp' go test ./internal/config/credentials/... -run 'TestNativeEncryptedVault|TestKeychainNativeRoundTrip' -count=1 -v
GOFLAGS='-tags=godbus_no_tcp' go test -race ./internal/provider/... ./internal/config/credentials/... ./internal/storage/credentialstore ./internal/commands/install ./internal/agent/run ./internal/agent/context
```

### 자격 증명 구현과 검증

자동 교체 pass의 전체 I/O 예산은 2분이다. 시간 초과 시 마지막 시도 위치를 private cursor에 기록해 느린 앞 항목이 이후 키를 계속 막지 않게 한다. 자격 증명 작업의 실패는 중앙 diagnostics의 고정된 사건으로 보고하며 별도 진단 파일을 만들지 않는다. 진단 출력 장애는 해당 Sink가 보고하고, 교체 worker는 다음 확인을 계속한다. worker의 panic은 고정 오류로 반환하고 종료한다. 저장용 키 할당 카운터는 백업 키를 삭제하지 않는 보관 상한이며 한도에서 새 할당을 거절한다.

Vault의 평문 마스킹 목록은 최대 1024개, 개별 값은 2048바이트다. 새 값을 기억할 수 없으면 연결·저장을 거부하며 기존 값을 잊지 않는다. 완전한 값은 긴 일치가 우선하는 불변 matcher를 사용하고, 스트림은 제어문자 제거로 비밀값이 다시 이어지는 경우도 차단한다. 잘린 미리보기 끝의 자격 증명 접두부도 마스킹한다.

credentialstore의 Put·Delete는 취소 가능한 프로세스 공통 쓰기 lease를 공유한다. 참조 scope와 현재 credential 수를 OS 키 발급 전에 확인하며 최종 SQL의 원자적 조건도 유지한다. 확실히 거절할 저장은 키와 할당 슬롯을 만들지 않는다. maintenance는 기존 optimistic 교체를 유지한다.

credentialstore는 저장 버전마다 `crypto/rand`로 독립적인 256비트 키를 만들고 OS 저장·재조회를 확인한 뒤 AES-256-GCM 암호문을 SQLite에 확정한다. nonce는 매번 새로 생성하며 credential 참조·Provider/endpoint 식별자의 SHA-256·암호화 키 참조를 인증 데이터에 묶는다. SQLite에는 암호문만 전달하고 원자적 쓰기·`synchronous=FULL`을 유지한다. 설정·대화에는 참조만 기록한다.

프로세스는 자격 증명 유지 worker 하나를 시작하고 종료 시 취소·합류한다. 시작과 매시간 저장된 버전을 확인하며, 처음 관측한 버전의 시각부터 24시간이 지나면 API Key 값·연결 참조를 유지한 채 재암호화한다. 주기는 내부 상수이며 메뉴·CLI·API 옵션으로 노출하지 않는다. export와 유지보수 명령에는 worker를 시작하지 않는다. credentials.db 형식은 유지하고 private `credential-rotation/`의 버전별 시각을 원자적으로 기록해 재시작 후에도 주기를 유지한다.

교체 pass는 별도 디렉터리 lease로 프로세스 간 중복을 막는다. 32건씩 읽고 DB 조회를 닫은 뒤 OS I/O를 수행하며 항목별 30초 상한을 둔다. 현재 key_ref·scope·암호문이 관측한 값과 같을 때만 조건부 갱신해 동시 저장·삭제를 우선한다. 확인된 갱신 뒤에만 시각을 기록하고 일부 항목 실패는 다른 항목의 처리를 막지 않는다. 시계가 뒤로 이동하면 만료를 무기한 연장하지 않고 교체한다. 실패는 원본과 OS 키를 유지한 채 다음 pass에서 재시도하며 중앙 진단에는 고정된 상태·오류 코드만 남긴다.

조회는 누락된 키를 재생성하지 않고 변조·미지원 형식에 오류를 반환한다. 불확실한 저장과 DB 백업의 복구를 위해 암호화 키 버전을 보존한다. 비밀값은 완전한 텍스트·스트림 조각·중첩 JSON에서 검사하며 서명 상태를 변경해야 하면 저장·재전송을 거부한다. 실행 중 평문 문자열과 OS 계정·프로세스 메모리 침해까지 완전한 소거·유출 방지를 보장하지 않는다.

credentials/osstore는 명령·DBus 연결·session·prompt의 취소와 정리를 소유한다. Vault가 호출자 context와 30초 상한을 제공하고 credentialstore의 AES 키 저장·재조회·SQLite 쓰기까지 같은 context를 전달한다. 읽기 실패·불확실한 선택 저장에는 기존 credential을 삭제하지 않는다. TUI 복원도 credential 읽기를 연결 worker에 맡겨 이벤트 루프의 취소 처리를 막지 않는다. 제거는 private DB 삭제의 디렉터리 동기화 이후에만 OS 키를 지우고 홈 lease와 제거 기록으로 재시도·동시 시작을 조율한다.

프로토콜 회귀 검사는 원본 블록 순서·빈 signed 필드·지문 변경 시 thinking 제외·무인자 Gemini 호출·필수 함수명·거절 텍스트·부분 캐시 usage·추론 전용 usage·영구 종료의 재시도 억제를 확인한다. JSON 이스케이프와 중첩 JSON 문자열의 비밀값도 검사한다. Claude 지문은 요청 한 번이 소유한 누적 해시로 계산하고 앞선 전체 JSON을 매 상태마다 다시 만들지 않는다.

### Provider 요청과 메모리 소유권

도구 이력은 Claude·Gemini·Responses 변환에서 검사한 엔진 호출을 읽기 전용으로 빌린다. Chat JSON을 만든 뒤 다시 파싱하는 중간 사본을 만들지 않는다. Claude는 continuation과 prefix를 한 번 복원하고, 응답 바인딩에서는 이미 검사한 block JSON을 그대로 감싼다. 제외한 thinking의 슬라이스 꼬리도 비운다. OpenAI Decoder는 반복 전송된 최종 텍스트를 문자열로 다시 보관하지 않고 형식만 검사하며, 최종 output JSON의 소유권을 continuation에 넘긴다. 최종 output이 없는 경우에만 완료 항목을 순서대로 합친다. Gemini는 조각을 한 번 인코딩해 응답 전용 버퍼에 쌓고 완료 시 그 버퍼를 넘긴다. 각 Decoder와 native 호출 누적기는 완료·실패 전환에서 임시 참조를 해제한다.

Gemini의 continuation 본문·서명 인자·로컬 호출 ID는 현재 메시지와 대조한다. Encoder가 캐시를 사용하지 않는 Provider로 바뀌면 이전 캐시도 해제하고 Responses 항목 ID 추적은 도구 호출 상한에 묶는다. HTTP 재요청 클로저는 변환한 이미지/이력 DTO를 잡아 두지 않는다. 자동 추론 요약 옵션이 거절된 경우에만 원래 이력에서 다시 변환하며 토큰화·모델 실행을 추가하지 않는다. 이미지의 Base64 바이트만으로 요청 한도를 넘으면 변환 전에 거부한다. 실행기는 응답을 버퍼에 추가하기 전에 크기를 확인하며, 큰 공백을 제거하고 남은 작은 최종 답변은 원래 버퍼와 분리한다.

비밀값 검사는 전체 원문 검사와 JSON 문자열 순회를 구분한다. JSON 문법 검사를 중복 실행하지 않고 escape가 없는 문자열은 재파싱하지 않으며, JSON이 아닌 문자열의 재귀용 바이트 사본도 만들지 않는다. 중첩 JSON 문자열·객체 키·Unicode escape·터미널 제어문자 검사는 유지한다. Projection의 Provider/model 출처 무효화도 회귀 검사한다.

서명 이력·도구 인자·완료 응답·조각 누적·중첩 JSON 보호 검사는 아래 명령으로 측정한다. 같은 입력과 Go·OS·CPU에서 반복하며 누적 할당량, 동시에 보유한 힙과 RSS를 구분한다. 실모델 생성 지연과 동시 접속 부하는 이 벤치마크에 포함하지 않는다.

```sh
GOFLAGS='-tags=godbus_no_tcp' go test ./internal/provider/protocol ./internal/provider/protocol/responses ./internal/provider/protocol/gemini ./internal/config/credentials -run '^$' -bench 'BenchmarkClaudeSignedHistory|BenchmarkCloudToolHistory|BenchmarkCompletedResponse|BenchmarkContinuation|BenchmarkCheckProtected' -benchmem -benchtime=300ms -count=3
```

이벤트 링·서버 draft·상세 원본 spool·페이지 payload·팀 공유 텍스트·TUI 표시 캐시는 각 소유자의 용량·수명 경계를 따른다. 서명된 원문과 표시/실행용 값은 서로 다른 소비 계약이므로 한쪽을 제거하지 않는다. 보관한 암호화 키 버전과 알려진 비밀값은 각각 DB 백업 복원과 이후 출력의 마스킹에 필요하다. 메모리 회수는 소유권 해제에 따른다. 강제 GC·전역 응답 캐시·별도 DB 연결 풀은 사용하지 않는다. 실제 Provider의 대형 서명 응답, 다중 구독자의 대형 최종 응답, Linux 런타임의 장시간 RSS는 별도 부하 측정 대상이다.

## 성능과 장시간 수명

큰 중첩 JSON의 복원은 참조 경로 trie를 만들어 각 부모 객체/배열을 한 번 decode·encode한다. 같은 payload는 한 번 읽고 큰 정수를 RawMessage로 보존한다. 겹치거나 비정규적인 배열 경로는 거부한다. 문맥 복원 메타데이터는 최대 256개 payload ID씩 batch 조회하여 필드별 N+1 읽기를 없앤다. 512바이트 이상 Thinking의 이벤트/메시지 사본과 비동기 명령의 JSON 문자열/구조화 출력은 같은 canonical payload를 공유한다.

local.Executor는 같은 요청의 불변 인자를 이름·전체 본문 SHA-256으로 재사용한다. 네이티브 호출 ID는 매번 검사하고 경로·승인·실행 직전 상태는 캐시하지 않는다. 수정용 slice는 호출별로 복사한다. 보관은 8MiB FIFO이며 BeginCommands·StopCommands·Close에서 해제한다. 큰 체크포인트 목표문은 기존 payload를 참조한다. 현재 실행의 목표문 지문만 작은 단일 캐시에 유지하고, 트랜잭션 준비가 취소/실패했을 수 있으므로 DB 존재 여부를 재검사한다. 가변 파일 사실은 최신 run_states 행에 두어 오래된 상태 payload를 누적하지 않는다.

2026-11-08 macOS arm64·Apple M1 Max·Go 1.27.3에서 동일 fixture로 측정했다. 아래 값은 합성 hot path의 시간·누적 할당이며 전체 서버 지연이나 RSS 개선율을 의미하지 않는다. 변경 전 한 표본, 변경 후 세 표본을 사용했다.

| 벤치마크 | 변경 전 | 변경 후 |
| --- | --- | --- |
| BenchmarkRestoreArray, 32개 × 64KiB | 21.99ms · 175.35MB | 1.37~1.45ms · 9.08~10.09MB |
| BenchmarkRepeatedArguments, 약 480KiB JSON | 4.34ms · 3.59MB | 0.214~0.221ms · 0B |
| BenchmarkRunStatePreparation, 약 4.4MiB 목표문 | 6.05ms · 6.18MB | 2.10~2.16ms · 약 2.2~2.4KB |

각 측정은 해당 패키지에서 `go test -tags=godbus_no_tcp -run '^$' -bench '<벤치마크명>' -benchmem`으로 재현한다. 다른 테스트의 준비 비용과 구분해 단독 실행한다.

2026-11-08의 macOS arm64·Apple M1 Max·Go 1.27.3 작업 트리에서 `BenchmarkDetailPages`를 `-benchtime=500ms -count=6 -benchmem`으로 실행했다. 1MiB 공개 본문의 최초 준비와 전체 64KiB 페이지 조회는 5.16~5.37ms/op, 약 1.272MB/op·342 allocs/op였으며 원본은 매 회 한 번 읽었다. 합성 로컬 상세 경로의 측정값이며 모델 추론·운영 서버의 전체 지연을 뜻하지 않는다. 정규화 저장 회귀 검사는 동일한 87,395바이트 도구 JSON과 메시지 문자열이 payload 한 건을 공유하는 것도 확인한다.

동시 단일·팀 실행은 processmemory의 프로세스 공통 문맥 예약을 사용한다. 합계 1GiB에서 원본·투영·요청의 예상 바이트를 3배로 계산하고 복원·요청 구성 전에 확인한다. 저장소의 ContextMessages는 선택한 본문의 메타데이터를 먼저 읽고 페이지 전체를 예약한 뒤 복원한다. 비전 미지원 문맥은 이미지 payload를 예약·복원하지 않는다. 예약 실패는 기존 원본을 유지하고 반환하며 실행 종료에서 참조와 예약을 해제한다. 이는 문맥 처리의 논리 예산으로 전체 RSS·다른 프로세스·커널의 하드 상한은 아니다.

생성 응답 누적은 별도 ReplyLease에서 프로세스 합계 256MiB를 예약한다. raw 텍스트·Thinking·완료 continuation·도구 인자를 합산하고 증가 전에 검사한다. 누적한 응답을 잡은 채 대기하지 않으며 초과 시 해당 생성을 실패시킨다. 재시도는 누적 상태와 예약을 비우고 종료에서 반환한다. image decode의 가중 합계는 256MiB, 큰 payload 변환은 512MiB다. decode는 픽셀 수·입력 크기, 저장은 원문과 임시 표현의 예상 비용으로 예약한다. 단일 큰 변환의 가중치는 예산 전체로 한정해 직렬 수용한다. 취소는 대기를 해제한다. 외부 원문도 payload 예약을 공유하며 이는 OS RSS quota가 아니다.

skills.Reader·local.Executor는 생성 옵션으로 마스킹·정책·스킬 reader·실제 이미지 디코더 슬롯을 고정한다. TUI·팀은 명시적인 Skills 의존성을 전달하고 설정용 setter나 선택적 타입 단언으로 실행 중 의존성을 바꾸지 않는다. 메타데이터 snapshot과 문맥 내 보존 결과는 각각 한 곳에서 소유하며, 동일 ID·revision의 스킬을 반복 보관하지 않는다. 본문 미리보기는 256행 간격 희소 위치와 가시 구간만 계산하며 조회·줄 배치·자동완성 작업의 결과 세대를 구분한다.

스킬 메타데이터 snapshot은 최대 8건·합계 8MiB·2분이며 completion은 최신 snapshot을 짧게 재사용한다. path ID는 최근 사용 기준 128개·512KiB 안에서 교체한다. 요청의 instruction evidence는 합계 8MiB이며 최신 스킬 버전과 동일 파일 revision의 페이지를 보존한다. manual-only 허용은 명시된 ID의 불변 context 값으로 해당 요청에만 전달한다. 참고 파일 revision에는 inode·device·크기·mtime·ctime·마스킹 revision을 사용한다. 목록·참고 조회와 오류 문구는 중앙 catalog를 사용한다.

스킬 도구 전달은 TestSkillToolDelivery에서 실제 Runner가 생성한 모델 요청의 시스템 지침·세 도구·인자 설명을 확인하고, 지원하는 9개 Provider의 최초/재사용 전송 JSON까지 검사한다. TestTeamSkillDelivery는 팀 어댑터가 같은 선언을 보존하는지 확인한다. TestSkillToolsPageReferences는 실제 로컬 실행기로 목록 조회·경로 로드·반환 ID 사용·revision을 전달한 후속 페이지·EOF를 끝까지 수행한다. offset의 실행 검사와 JSON Schema 상한은 skills.MaxResourceFileBytes를 사용한다. 이 검증은 결정적인 scripted model을 사용하며 실제 모델의 스킬 선택·준수율 측정과 구분한다.

기본 스킬은 config/bootstrap의 embedded 파일 트리로 배포한다. 본문·참고 파일·라이선스를 같은 private 디렉터리 경계에서 누락한 경우에만 원자적으로 게시하며 사용자 수정 파일은 유지한다. 기존 파일은 embedded 바이트 복사 전에 확인한다. application-design은 과업·맥락·사용성에서 구성을 도출하고, Product Design의 시각적 근거·완전한 흐름·실제 렌더링/상호작용 검토와 anti-slop을 설계 전 과정에 적용한다. 64개 참고 문서는 설계 선택의 조건·장단점·검증을 소유하고, assets/examples와 동봉 DESIGN.md·PRODUCT.md는 가상 사례의 현재 구현만 기록한다. 대상 프로젝트의 요구·기존 시스템·사용자 요청이 우선하며 사례의 색·화면 수·파일 구조·프레임워크를 일반 규칙으로 전파하지 않는다. Archive 사례는 미니멀리즘·에디토리얼 구성과 역할별 포인트 색을 사용한다. 본문 → 분류 지도 → 선택 자료의 세 층이며 설명은 8KiB, assets/examples의 코드·미디어는 1MiB까지 허용하고 512KiB 이하를 권장한다. 여섯 앱의 하위 페이지 31개, 예약 대안 3개, 캔버스 1개와 모션 1개의 총 36개 HTML을 역할별 CSS·Vanilla JS·로컬 이미지/아이콘으로 구성한다. 결재·DB 워크벤치·DB 지식 대화·AI 오피스는 정적 디자인 셸로서 실제 DB·AI·승인·편집·내보내기·저장을 실행하지 않는다. 이전 JSON 검사·범용 작업 관리 예시의 경로와 전용 컨트롤러는 제거하고 직접 접근 지도를 새 폴더에 맞춘다. 공유 상호작용 스크립트는 메뉴·대화상자·취소 가능한 표현 모션을 소유하고, 실제 탭 화면만 탭 선택·deep link 스크립트를 불러온다. 문서 목차·근거 이동·상태 필터는 각 앱이, 데모 드로어는 모션 예제가 소유한다. 스타일·입력·대화상자·축소 모션을 실제 렌더링과 키보드 동작으로 검토한다. 예약은 공유 도메인이 초안·날짜/인원 검증·계산·저장을 소유하고 표현 컨트롤러가 화면별 구성을 맡는다. 캔버스는 페이지가 좌표·선택·배율·pointer capture를 소유하며 포인터 취소와 종료에서 정리하고 되돌리기는 이후 변경을 덮어쓰지 않는다. 같은 브리프·데이터·뷰포트에서 시각 차이와 실제 동작을 함께 검토한다. 브라우저 상태·입력·수집 크기에는 예시별 한도를 두고 서버 기능을 가장하지 않는다. 이 예산은 기본 번들에만 적용하며 엔진의 64KiB 텍스트 페이지와 구분한다. 자료는 앱별·공통·모션·참고 캡처의 하위 폴더로 구분하며 Markdown 지도가 직접 파일 경로를 소유한다. 갤러리 화면이나 폴더별 중간 인덱스는 배포하지 않는다. 서브커맨드나 외부 실행기는 추가하지 않는다.

Bootstrap 검사는 실제 skills.Reader의 목록·자동 로드·내부 링크 순회로 모든 설명·코드·미디어의 도달 가능성·두 링크 이하 깊이·8KiB 설명/1MiB 자료 예산·JPEG 메타데이터·텍스트 페이지 진행과 private 권한을 확인한다. 64KiB를 넘는 UTF-8 예시의 페이지 읽기, 동시 초기화·사용자 수정 보존·누락 복구와 분류별 디렉터리와 실제 에셋 하위 폴더의 파일/링크 거부도 확인한다. 기본 스킬 변경 시 예시의 내부 경로·외부 런타임 의존성·350줄/헤더·라이선스를 검토하고 실제 렌더링·주요 상호작용은 브라우저에서 별도 확인한다. 설치/정적 검사는 모델의 디자인 품질이나 접근성 인증을 뜻하지 않는다.

storage/content는 JSON 도구 출력과 그 JSON을 담은 메시지 문자열이 같은 payload를 참조하도록 원본과 문자열 표현을 구분한다. SQLite는 같은 물리 ID를 복원·무결성 검사하고 표현만 경계에서 재구성한다. 세션 schema 28이 이 계약을 소유한다. 병렬 호출 묶음의 제안은 한 journal 커밋, 실행 의도와 체크포인트는 효과 발생 전 한 커밋으로 저장하며 관찰·승인·결과 순서를 유지한다.

큰 명령 출력의 입장은 취소 가능한 FIFO이며 취소된 대기자를 제거한다. 저장용 CommandCapture도 문맥을 받아 읽기·크기·파일 변경을 확인한다. 종료 정리는 독립된 제한 시간으로 대기하고 입장 실패를 성공 완료로 처리하지 않는다. 미확인 효과는 자동 재실행하지 않는다. 상세 API는 공개 DTO를 증분 인코딩·마스킹하고 한 페이지와 작은 메모리 또는 경로 없는 spool을 보유한다. 디스크 보관 실패 시 전체 JSON 버퍼를 만들지 않고 현재 페이지를 수집하며 중앙 진단을 남긴다.

상단 메모리·작업 경과 시간은 TUI가 소유한 같은 시계를 사용하며 작업 중에는 250ms, 대기 중에는 1초 간격으로 전환한다. 경과 시간의 초가 바뀔 때만 명령 메타데이터를 조회한다. 메모리는 시작 시에도 조회하고 정수 연산으로 반올림한 표시값·문자열 한 건만 보유한다. MiB는 소수점 한 자리, GiB는 두 자리로 비교하고 표시가 같으면 문자열을 할당하거나 그리지 않는다. 메모리·압축 점만 바뀌면 마지막 전체 화면의 정제된 제목을 재사용해 상단 한 줄만 갱신한다. 본문·입력·선택·스크롤 배치를 유지하고 전체 화면 갱신이 대기 중이면 같은 프레임에 합친다. Linux·WSL은 `/proc/self/statm`의 resident 페이지 수와 OS 페이지 크기, macOS는 native proc_info의 현재 resident 크기를 사용한다. Linux 카운터는 [커널의 빠른 RSS 집계](https://man7.org/linux/man-pages/man5/proc_pid_statm.5.html)이며 상세 smaps 측정과 차이가 날 수 있다. macOS의 [proc_taskinfo](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/proc_info.h)는 현재 RSS를 제공한다. 최고 사용량·Go 힙·자식 프로세스를 대신 표시하지 않는다. 시계는 이벤트 루프 종료에서 정리하고 측정값을 SQLite나 모델 문맥에 누적하지 않는다.

긴 ASCII 문단은 한 셀당 한 바이트인 경우에만 전용 줄 바꿈을 사용하고 한글·이모지·탭은 grapheme 경로를 유지한다. 긴 렌더링 줄 안에도 256행 간격의 희소 위치를 기록해 가시 구간 가까이서 재개한다. 열린 최상위 코드 펜스는 완성된 줄을 재사용하며 닫힘·폭 변경·원문 교체는 정상 파싱으로 되돌린다. 참조 링크·목록·표의 전체 의미는 차등 렌더링 검사로 검증한다.

대화 표시의 파생 캐시는 가시 이력과 활성 Thinking·응답을 합쳐 16MiB 예산을 적용한다. 완료 이력과 오래된 활성 블록부터 파싱 캐시를 해제하고 원문·희소 스크롤 위치·복사 좌표는 유지한다. 입력 수용 한도와 표시 캐시 예산은 별도다.

파일 관측은 변경된 경로만 갱신하고 추가 경로만 정렬·병합한다. 변화 없는 조회 결과는 작업 상태를 재구성하거나 재검증하지 않는다. 로그의 실행 중 게시와 최종 확정은 명령별 잠금으로 직렬화하고 동일한 재조회 파일을 사용한다.

저장소의 압축본·문맥 점유 표본은 세션별 최신 한 건을 유지한다. 화면 표시 투영은 숨은 payload를 복원하기 전에 필요한 필드를 선택하고 중간 JSON 변환을 줄인다. 원본 문맥과 내보내기는 전체 기록을 복원한다.

입력은 grapheme 경계를 보존하고 붙여넣기를 묶어 처리한다. 커서 탐색은 약 256 rune 간격의 parser 상태 체크포인트에서 재개하며 편집 위치 이후의 인덱스만 폐기한다. 큰 삭제와 초안 교체는 불필요한 인덱스 용량을 해제한다. 입력 배치는 줄별 rune 시작 위치를 한 번 기록해 커서 줄을 이진 탐색하고 그 줄 안에서만 바이트 위치와 선택 범위를 계산한다. 출력 가능한 ASCII 폭은 바이트 길이로 계산하고 나머지는 Unicode 폭 규칙을 사용한다. 이미 순회 중인 grapheme의 폭은 iterator에서 재사용한다. 변경되지 않은 source/width 배치와 Markdown 투영을 재사용하고 연속 화면 갱신을 합치되 순서 있는 이벤트는 버리지 않는다. 요청 worker는 발신자별로 표시용 텍스트와 중간 usage의 최신 미소비 mailbox를 각각 소유한다. 같은 사용량·비용 표본은 다시 알리지 않으며 중간 usage는 텍스트 병합을 끊지 않는다. 재시도·완료·승인·최종 사용량 경계에서는 두 mailbox를 함께 분리한다. 소비 후 본문 참조를 해제하며 별도 전송 goroutine을 만들지 않는다. 기록·파일 열·선택·뷰포트 캐시를 해제할 때 참조도 정리한다.

대화 줄 인덱스는 변경되지 않은 이력의 접두사를 보존하고 추가 이력과 현재 출력만 배치한다. 폭·이력 소유 배열·prepend·잘라내기·도구 접기 상태 변경은 인덱스를 다시 구성한다. 가시 블록 목록만 비교해 화면 밖 파싱 참조를 해제한다. 자동 갱신은 60Hz 간격과 최근 그리기 비용의 3배 중 큰 시간을 기다리며 추가 대기는 최대 250ms다. 입력은 이전 프레임 시작 기준 다음 60Hz 시점에 표시할 수 있다. 느린 자동 갱신 직후의 입력은 한 번 즉시 표시하고 연속 입력 프레임은 60Hz 예산을 따른다. 입력 배치는 최대 32개이며 런타임 알림과 공정하게 처리하고 붙여넣기·종료 순서를 보존한다.

후속 요청 FIFO와 큐 편집은 TUI 이벤트 루프가 소유한다. 편집은 선택 위치와 작성 중인 Editor 값만 보관하며 큐 전체를 복제하지 않는다. 목록은 전체 항목을 번호순으로 표시하고 항목별 마스킹·줄 배치를 재사용한다. 각 요청은 예약 시 본문과 선택 경로를 값으로 캡처한다. 편집은 본문만 바꾸며 선택 변경은 이미 예약된 요청의 참조에 영향을 주지 않는다. 요청 worker가 실제 전달 전에 참조를 재확인하고, 초안 복구는 캡처한 경로도 보존한다. 원문 또는 마스킹 revision이 바뀐 항목만 표시 문자열을 갱신하며 선택 색상 전환은 본문 재배치를 유발하지 않는다. 화면 밖 항목 선택은 transcript의 위치 탐색을 사용한다. 출력 증가로 live block ID가 바뀌면 큐 기준 읽기 위치를 보정하고 창 크기 변경 시 선택 항목을 다시 보이게 한다. 사용자가 직접 과거 출력으로 이동한 위치는 유지한다. 편집 중에는 dequeue를 보류해 선택 위치를 유지하고, 현재 worker의 완료·실패를 편집 상태에 기록해 닫을 때 실행·복구를 이어간다. 미전달 방향 수정은 큐 편집 본문 대신 보관된 일반 초안에 복구한다. 소비·삭제·복구한 큐 항목과 표시 캐시의 제거된 항목은 참조를 지우고 빈 큐와 닫힌 편집 상태는 해제한다. 별도 worker나 모델 호출을 추가하지 않는다.

디렉터리 조회와 파일 미리보기는 각각 실행 1개·최신 대기 요청 1개를 가진다. 취소는 대기 요청을 제거하지만 실행 슬롯은 실제 I/O 완료 알림에서 해제한다. 이벤트 루프가 최신 요청만 시작하며 이전 revision의 결과는 적용하지 않는다. 읽기와 배치 계산은 별도 슬롯을 사용한다. Raw/Markdown의 최초·폭 변경·모드 전환 배치는 단일 계산 worker에서 준비하며 이벤트 루프는 준비된 가시 구간만 조회한다. 원문·폭·모드와 revision이 일치하는 결과만 수용하고 비활성 레이아웃은 해제한다. Markdown은 viewport의 파싱·희소 줄 배치와 스타일 렌더러, Raw는 줄 번호 인덱스를 사용한다. 전체화면은 숨은 목록·입력창 배치를 생략한다. Shift+R은 같은 읽기 본문의 표시 형식만 전환하고 비활성 레이아웃 캐시를 해제한다. 내용·폭 변경은 배치를 갱신하고 같은 보기의 파일 갱신은 모드와 읽기 위치를 보존한다. 종료는 대기를 비우고 실제 worker를 합류한 뒤 슬롯·파싱 참조를 해제한다.

파일 변경 감시는 활성 디렉터리 하나에 연결한다. 플랫폼별 등록 I/O도 worker에서 수행하며 실행 1개·최신 대기 1개로 제한한다. 이벤트 루프가 감시 채널과 150ms 단발 타이머를 소비하고, 등록 직후와 알림 유실 시 재조회한다. 경로 전환·취소·화면 닫기에서 감시와 타이머를 정리하며 늦게 도착한 감시 객체도 닫는다. 갱신은 파일 읽기 슬롯을 사용하고 새 폴링 루프를 만들지 않는다.

명령 실행기는 요청의 전체 마감 시각·취소 문맥을 한 번 바인딩하고 최대 16개 명령을 소유한다. 최초 30초 대기는 팀 경로 예약 대기를 포함하고 wait_command의 기본 180초와 함께 프로세스 수명과 분리한 관측 창이다. waiting·running·finalizing은 모두 활성 명령이다. 레지스트리 잠금에서 수용량·ID를 예약한 뒤 경로 대기와 OS 프로세스 시작은 잠금 밖에서 수행한다. Wait는 한 번만 실행하고, running 결과의 next와 미완료 명령 피드백은 공통 도구 관리 안내를 사용한다. TUI는 경과 시간 타이머에서 작은 메타데이터만 조회한다. 마지막 출력 시각은 캡처 파일 I/O와 분리한 짧은 잠금으로 읽고 갱신한다. stdout·stderr 각각 1MiB를 넘으면 앞·뒤 8MiB씩의 ring을 암호화한 임시 파일로 옮긴다. AES-GCM은 64KiB 블록마다 새로운 nonce를 사용하며 임시 키는 메모리에만 존재한다. 한 스트림은 한 개의 평문·암호문 블록을 보유한다. 명령 실행과 Wait는 병렬이며 큰 출력의 구체화·마스킹·보관·커밋용 원문 생성은 요청 공유 가중 64MiB 예산으로 조율한다. 작은 출력은 예산 대기를 생략한다. 완료 결과는 작은 발췌로 대기하며 CommandCapture로 기록할 원본 한 건을 읽고 SQLite 커밋 후 AcknowledgeCommand로 해제한다. 임시 캡처는 종료·오류·취소에서 닫고 제거하며 무결성·I/O 실패는 반환한다. 활성 명령이 있는 모델 요청은 I/O worker의 이벤트를 소유자에게 동기 전달하고, 소유자는 완료 결과를 수집한다. 모든 worker·타이머는 종료 시 정리하며 기록 실패를 성공으로 처리하지 않는다.

OpenRouter reasoning은 수신 시 현재 블록을 점진적으로 병합한다. 서명·종류·식별자 경계와 알 수 없는 필드를 보존하고 닫힌 블록만 유지하므로 완료 시 모든 조각을 다시 해석하지 않는다. 취소·실패한 요청의 미완성 블록은 후속 요청에 공개하지 않는다.

Provider 요청·native 호출 이력의 JSON 인코딩은 protocol의 불변 Sonic 설정을 사용한다. HTTP JSON 본문에서 불필요한 HTML 이스케이프는 생략하고 키 정렬·문자열 검사·RawMessage 검증을 유지한다. LM Studio·OpenAI는 Responses, Claude는 Messages, Gemini는 GenerateContent 형식으로 인코딩한다. 스트림과 OpenRouter continuation의 파싱·병합은 표준 encoding/json을 사용한다. 도구 인자의 중복 키·종류·알 수 없는 필드·후행 입력 검사도 별도 경계에서 유지한다.

하나의 요청 실행은 전용 Provider 인코더를 소유한다. Chat Completions·Ollama·Responses는 Provider 변환 전에 불변 문자열·이미지 digest와 native 내용 지문을 비교해 메시지 JSON을 재사용한다. Claude·Gemini는 별도 native 원본 그룹과 wire 접두사를 재사용한다. 원본 지문으로 검증을 재사용하고, 큰 인자·continuation은 캐시 사본으로 보관하지 않고 현재 요청에서 다시 연결한다. 원본 그룹의 메타데이터·이미지 변환은 전체 인코딩 예산의 절반 이내, 최종 전송 조각은 남은 예산 안에서 유지한다. 원본 그룹과 wire 조각은 수명을 분리해 작은 식별자가 전체 그룹을 붙잡지 않는다. Claude의 시스템 지침·도구 변경은 서명 접두사를 다시 계산하고 변경된 메시지 이후를 재구성한다. SHA-256의 요청 내부 체크포인트만 재사용하며 서명된 원문을 바꾸지 않는다. Gemini는 같은 접두사에 속한 호출 ID만 복원하고 변경된 호출 뒤의 결과를 다시 연결한다.

컨텍스트 투영이 소유권을 인수한 native 인자·추론 상태는 한 번 지문을 만들고 읽기 전용으로 전달한다. 값·버퍼 교체는 지문을 무효화하며 소유권 계약이 없는 입력은 매번 내용 해시로 검사한다. 도구 결과와 뒤따르는 이미지 메시지는 한 그룹으로 순서를 유지한다. 추론 설정·도구 정의·서버 한도는 매번 반영한다. Chat Completions·Ollama·Responses의 도구 선언은 원본 정의의 구조와 중첩 필드·허용값이 같을 때 불변 wire JSON을 재사용하고 vLLM 토큰화에도 같은 조각을 사용한다. 모델·Provider 변경과 실행 종료는 전체 캐시를 해제한다. 이력 캐시는 보수적 부대비용을 포함해 약 64KiB부터 사용하며 전체 보유 예산은 16MiB다. 도구 스냅샷은 정의 메타데이터를 포함해 256KiB 이내로 유지한다. 예산 초과 이력은 완전한 앞부분만 재사용하고 나머지는 해당 HTTP 본문만 소유한다. 캐시 한도는 입력 수용 한도를 제한하지 않는다. HTTP 본문은 불변 조각과 정확한 Content-Length·독립 재읽기 커서로 재시도·리다이렉트를 보존한다. 종료·실패·취소는 worker를 합류한 뒤 참조를 해제한다. vLLM의 정확한 토큰 수 사전 검사는 같은 메시지 인코딩을 재사용하는 별도 요청으로 유지한다. 실제 usage가 없으면 전송 바이트를, 그 값도 없으면 이미지 본문을 제외한 텍스트 크기를 압축용으로 사용하며 이를 위한 전체 요청 재직렬화나 표시용 추정 토큰은 만들지 않는다.

메시지·실행 이벤트·요약·대용량 레코드 참조의 저장 JSON은 go-json의 기본 Marshal·Unmarshal을 사용한다. 키 정렬·HTML 이스케이프·nil과 빈 값·64비트 숫자·native arguments의 내용·복원 값의 독립성을 검사한다. SHA-256 무결성·저장 형식·트랜잭션 경계는 storage/content와 SQLite의 소유권을 따른다. 반환 값이 입력 버퍼에 의존하는 무복사 옵션은 사용하지 않는다.

HTTP는 표준 net/http를 사용한다. 사용자 지정 DialContext에서도 TLS ALPN으로 HTTP/2를 협상하고 HTTP/1.1 서버도 지원한다. 연결·TLS handshake·유휴 수신 감지는 각 경계의 제한을 유지하며 전체 응답 시간 제한을 추가하지 않는다. 응답 body·요청 취소·연결 종료는 transport가 소유한 수명으로 정리한다.

로컬 도구 카탈로그는 초기화 후 변경하지 않는 내부 정의 하나를 소유한다. 인자 검사는 이를 재사용하며 외부 Definitions 반환값은 중첩 Fields·Values까지 복사해 변경이 내부 검증으로 전파되지 않게 한다.

응답별 마스킹은 불변 압축 접두사 트리에서 누적 스냅샷의 새 구간만 검사하고 비밀키 접두사·미완성 UTF-8을 보류한다. 전체 키 목록을 문자마다 순회하지 않으며 가장 긴 일치와 제어 문자 제거로 재구성되는 키도 처리한다. 치환이 없는 출력은 원문 문자열을 공유하고 비밀키·제어 문자·잘못된 UTF-8을 변환하는 순간부터 독립된 출력 버퍼를 만든다. 응답과 Thinking은 상태를 공유하지 않으며 재시도마다 초기화한다. 파일 미리보기와 대화 본문은 희소 위치 인덱스로 가시 구간만 구체화한다. 일반 본문은 256개 표시 줄마다 원문 위치를 기록해 긴 한 줄도 중간부터 재개한다. 응답·Thinking의 Markdown은 마지막 두 최상위 블록을 가변 구간으로 두고 완성된 앞부분의 파싱·줄 수를 재사용한다. 간격이 촘촘한 최상위 목록은 마지막 두 항목을 가변 구간으로 두고 앞선 항목도 재사용한다. 목록 간격·종류가 바뀌면 해당 목록 처음으로 되돌리며 번호 목록의 시작 번호를 보존한다. 줄 위치는 바뀐 접미사와 최대 256개 직전 행만 다시 계산하고 캐시 보유 비용은 렌더링 변경 시 갱신한다. 참조 정의 가능성이 있는 `]:` 포함 문서와 느슨한 목록은 필요한 전체 파싱을 유지한다. 화면 밖 과거 응답·Thinking은 파싱 결과·span을 해제하고 위치 인덱스만 남긴다. 완료 블록의 큰 행 배열은 하나의 텍스트와 32비트 스타일 경계 배열로 보관하며 화면에 필요한 행과 스타일만 구체화한다. 희소 스크롤 위치는 유지하고 행별 재사용 지표는 활성 스트림만 보유한다. 표시 캐시 합계가 16MiB를 넘으면 완료 블록을 먼저 작은 표현으로 전환하고 예산 안에서 줄어드는 경우만 유지한다. 담을 수 없는 결과나 화면 밖 캐시는 해제하며 원문·희소 인덱스로 다시 구성한다. 일반 스트리밍은 증분 처리 수명 동안 보유해 매 delta의 전체 재파싱을 피한다. 줄이 매우 많은 활성 본문은 압축 행 또는 희소 순회로 전환하고 행별 재사용 지표를 만들지 않는다. 경로 표시는 선택 revision·폭·언어·마스킹 revision으로 캐시하고 실제 표시 예산을 채우면 변환을 끝낸다. Compaction은 시간 순서 목록과 우선순위별 제거 대기열로 사용자 지시·결과·실패의 보존 순서를 유지한다.

채팅에서 숨길 끝 빈 줄은 Markdown 행을 만들기 전에 표시 경계로 줄이고 원본과 파일 미리보기는 보존한다. 원문 예산은 diff 범위 배열의 보유 용량도 포함하며 예산 초과 시 화면 밖의 재조회 가능한 상세 내용을 먼저 해제한다. 요약·원본 참조와 가시 상세 내용은 유지한다.

팀 명령 카드는 상태 헤더와 결과 본문을 따로 갱신한다. command_cards.go는 명령 상태 조회를 카드에 투영하고 tools.go는 실행·대기·취소 결과를 발신자와 명령 ID에 연결한다. 제어 호출의 임시 표시를 정리하고 기존 카드의 원본 참조를 최신 결과로 교체하며 종료 시 추적 항목을 해제한다. 시계 갱신은 긴 명령·출력 문자열을 다시 만들지 않고 원본 참조·상세·접힘 상태를 보존한다. 상태가 바뀔 때만 짧은 접힘 요약도 갱신한다. 자격 증명 revision이 바뀌면 표시 원문을 다시 마스킹하고 재조회 가능한 상세는 해제해 현재 마스크로 읽는다. 상세 조회는 한 건만 소유하며 접기·화면 전환·이력 재구성에서 취소 표시를 남긴다. 이전 worker의 응답을 받은 뒤 다음 조회를 시작하고 취소된 원문은 포맷·큐 보관을 생략한다. 재시도는 자기 미확정 초안만 폐기하고 해당 발신자의 응답 또는 실제 출력 토큰 수신으로 안내를 해제한다. 입력 토큰만 보고되거나 다른 발신자가 출력하면 해당 안내를 유지한다. 접힌 명령 실패는 Process.Diagnostic을 사용해 원본 출력의 추가 조회 없이 원인을 표시한다. 영속 도구 결과·사용량·작업 통계는 세션 수명으로 전달하며 새 실행·승인·초안은 요청 취소를 따른다. 화면에 숨기는 자동 관측은 큐에 넣거나 전달 오류를 만들지 않는다. 화면 테스트는 실제 앱과 같은 기본 접힘 상태에서 시작한다.

Markdown의 Render·Stream.Render는 뷰포트의 실제 본문 폭을 받는다. 문법 설정은 공유하고 reader·문맥·AST는 호출마다 분리한다. 같은 스타일의 연속 inline 텍스트는 필요한 경우에만 builder로 합쳐 긴 문단의 반복 복사를 피하며 명시적 줄바꿈 조립에 임시 분할 배열을 만들지 않는다. 표의 열 너비는 내용과 구분자·들여쓰기 예산으로 계산하고 셀별 styled Rows를 조립하므로 표 전체를 하나의 긴 줄로 줄바꿈하지 않는다. 같은 폭의 증분 파싱과 완료 블록 재사용은 유지하며 폭 변경 시 이전 표 배치와 재사용 줄 수를 초기화한다. 희소 인덱스의 줄 수 계산과 가시 구간 출력은 같은 폭의 배치 결과를 사용한다. 좁은 폭에서는 헤더·값을 세로로 구성해 열 구분자만 남거나 내용이 옆 열을 침범하는 상황을 피한다.

tui/viewport는 블록 인덱스 조율(block_layout.go), Markdown·일반 텍스트 인덱스 구성(markdown_index.go·literal_index.go), 가시 줄 추출(block_window.go)을 구분한다. 표시할 내용이 없는 Markdown은 여백을 추가하지 않으며 원문에 내용이 이어지면 다시 배치한다. View.Update는 현재 접힘 상태로 정규화한 본문·참조·스타일·스트림 상태·간격이 모두 같으면 기존 배치와 창을 유지한다. 스트림 종료나 실제 표시 변경은 정상적으로 갱신한다. Transcript.Window는 하나의 최종 행 배열에 블록별 가시 행을 직접 추가하며 diff·가운데 정렬은 추가한 구간에만 적용한다. Block.Window를 직접 사용하는 미리보기는 독립된 행 배열을 받는다. Markdown 준비·복원과 파생 메모리 계수는 cache_budget.go가 소유하며 최초 인덱스 처리와 캐시 복원에서 같은 계산을 사용한다. markdown/packed.go는 완료 행의 불변 배열을, markdown/styles.go는 일반·작은 배열의 스타일 경계 순회를 소유한다. viewport/markdown_cache.go는 실제 표시 예산에 맞는 표현을 선택한다. completion.go는 원문·폭·표시 속성이 일치하는 스트림의 행과 희소 인덱스를 완료 표현으로 인계하고 가변 체크포인트·행 지표를 해제한다. markdown/render_packed.go는 큰 완료 문서를 전체 행 배열 없이 수집한다.

완료 응답·파일 미리보기와 줄이 많은 활성 본문은 행을 생성하는 중간에 같은 16MiB 표시 예산을 검사한다. 코드 블록 어댑터는 Goldmark의 문법 판정을 재사용하고 파싱 context가 소유한 본문 버퍼와 작은 segment scratch만 유지해 줄마다 원문 segment를 쌓지 않는다. 목록의 첫 행 표식과 끝 공백은 공통 행 sink에서 보존한다. 같은 내용·스타일의 연속 행은 횟수로 압축하며 논리 행 번호·복사 좌표는 유지한다. 압축 표현도 담을 수 없으면 전체 행 배열로 재시도하지 않고 원문을 순회해 희소 인덱스와 가시 구간만 구성한다. 일반 활성 스트림은 기존 증분 경계를 사용한다. 128KiB 이상이며 최상위 블록이 32개 이상인 큰 문서는 같은 불변 AST를 최대 네 그룹으로 나눠 처리하고 원래 순서로 합친다. 표가 있는 문서는 셀 배치의 임시 메모리를 줄이도록 직렬 처리하며 병렬 그룹도 표시 예산을 나눠 사용하고 합류한다.

압축 행이 예산을 넘으면 같은 AST에서 블록별 재개 계획을 시도한다. 일반 블록은 작은 압축 행, 최상위 코드 블록은 정규화한 본문과 256행 간격 바이트 위치로 보관한다. 원문과 같은 본문은 원문을 빌리고 변환된 본문만 소유하며 AST와 전체 행 배열은 보관하지 않는다. 희소 위치의 해당 논리 행에서 바로 재개한다. 계획도 같은 표시 예산을 따르며 화면 밖 해제·원문 교체·폭 변경에서 정리하고 필요할 때 재구성한다. 계획을 만들 수 없는 문서는 기존 순회·literal 경로로 표시한다.

tools/local의 Prepare는 공통 인자 검사를 마친 뒤 파일·폴더·셸·명령 제어·이미지의 구체 실행을 구성한다. 각 실행은 필요한 후보·경로·환경만 소유하고 공통 operation은 승인과 일회 실행 진입을 담당한다. 변경 전 원본·경로 재검사와 명령 안전성 재검사는 각 구체 실행 경계에 둔다.

같은 파일의 여러 부분 수정은 첫 일치 이후를 정방향으로 검사해 겹치는 중복도 거절한다. 원본 줄 위치를 앞에서부터 누적하고 최종 크기의 결과 버퍼를 한 번 할당해 수정 범위를 합친다. 파일 원본·개행·EOF 계약은 유지한다.

파일 변경 미리보기와 정확한 추가·삭제 줄 수는 같은 분석에서 얻는다. 공통 앞뒤 구간은 bytes.Equal 블록 비교로 건너뛰고 미리보기에 필요한 인접 문맥만 매칭한다. 전체 생성·삭제는 trace와 줄 배열 없이 출력한다. 미리보기의 작업량 한도에 도달한 경우에는 완전한 diff를 출력하고 정확한 변경량은 별도로 계산한다. 원본 검사·해시·변경 후보 바인딩은 유지한다.

storage/history가 화면 표시 필드 선택과 복원된 값의 표시 투영을 소유하고 SQLite는 표시할 레코드 조회·본문 복원·페이지 수명을 담당한다.

공용 본문 저장 시 canonical JSON 문자열의 해시·길이는 공용 Store가 최대 128건·본문 2MiB 안에서 재사용한다. 마지막 개인 Store가 닫히면 원문 참조와 인덱스를 함께 해제한다. 마스킹으로 원문이 바뀐 수신 문맥은 로컬 식별자로 저장해 다른 원본 바이트에 대한 참조를 만들지 않는다. 개인 DB 준비는 공용 트랜잭션 밖에서 수행하고 이름·순서·세션 연결만 짧은 트랜잭션으로 등록한다. 실패·불확실 커밋은 등록 여부를 확인한 뒤 미등록 새 파일만 정리한다.

큰 JSON 값의 인코딩·복원과 무결성 검사는 storage/content가 소유하고 SQLite는 `payloads`에 해시·원본 길이·압축 형식·본문을 저장한다. 새 payload는 Zstandard의 빠른 단일 실행 설정과 128KiB 창을 사용하며 압축 결과가 원문보다 작을 때 저장한다. 압축기·복원기는 Store가 지연 생성해 잠금 안에서 재사용하고 Close에서 해제한다. 재사용 출력 버퍼는 최대 256KiB로 제한하며 호출자에게는 독립된 결과를 반환한다. 복원 전 프레임의 선언 길이를 확인하고 출력 용량·해시·JSON 검사를 적용한다. 원문·gzip·Zstandard는 명시한 codec 값으로 구분한다. 작은 레코드는 payload 참조가 필요 없으면 재파싱·재직렬화를 생략하고 내용에 임의의 표식을 삽입하지 않는다. 한 연결을 쓰므로 조회 페이지를 닫은 뒤 payload를 복원한다. 기록과 payload는 같은 트랜잭션으로 커밋한다. 메시지 묶음의 수정 시각과 최초 사용자 요약은 같은 트랜잭션에서 한 번 갱신하며 수신 커서도 함께 확정한다. `record.Entry` 그룹은 같은 확정 시점의 메시지·이벤트·체크포인트를 원자적으로 저장한다. 실행 전 시작 의도는 효과 발생 전에, 실제 실행 결과·native 응답·변경 후 체크포인트는 취소와 무관하게 5초 저장 문맥에서 함께 확정한다. 실행 시작과 결과의 두 경계를 합치지 않는다. 종류·본문 길이·요약 입력 투영을 저장 시 기록하며 원본 읽기·export·취소 후 복원 계약을 함께 검사한다.

SQLite journal은 입력 검증을 쓰기 연결 대기 전에 끝내고, 약 4MiB의 준비 예산 안에서 메시지·이벤트·체크포인트·payload의 직렬화·해시·압축을 트랜잭션 밖에서 수행한다. 준비 배열의 예약 용량과 payload 메타데이터도 계산한다. 마지막으로 준비 중인 한 레코드는 기존 수용 한도까지 허용하고, 예산 이후 항목은 같은 트랜잭션 안에서 이어 처리한다. 불변 payload의 사전 존재 조회는 압축 생략에만 사용하며 실제 쓰기 때 재확인한다. 동시 신규 payload는 INSERT OR IGNORE로 합치고 실제 payload·레코드·수신 커서·체크포인트는 원래 순서로 원자적으로 확정한다. 준비한 payload는 쓰기 후 참조를 해제한다.

텍스트 미리보기는 저장 크기와 원본 크기로 먼저 읽기 예산을 확인한다. 압축 해제 중 전체 길이·해시·JSON 문자열을 끝까지 검사하고 32KiB 읽기 스크래치와 표시 접두사만 구체화한다. Zstandard의 모든 프레임에 작은 복원 창 상한을 적용하며 더 큰 창이 필요한 값은 같은 원본 읽기 예산 안의 완전 복원으로 검증해 큰 history ring 할당을 피한다. 디코더 옵션은 호출 후 복원한다. 같은 페이지의 반복 접두사는 2MiB payload 캐시 예산을 공유하고 마지막 참조에서 해제한다. 원문·export·숨은 필드 제외·손상 검출·취소 계약은 유지한다.

run_states는 실행별 최신 체크포인트 한 건을 갱신하며 32KiB 초과 구조화 상태를 같은 Codec으로 압축한다. 최종 이벤트는 작은 상태·통계만 두고 Export는 해당 실행의 전체 상태를 복원한다. Timeline은 TUI 전용 투영으로 표시할 종류를 먼저 선택하고 숨은 native 인자·continuation·이미지를 복원하지 않는다. 화면에 표시하는 도구의 display_args만 경로·명령·ID를 저장하며 잘못된 표시 필드가 실패 기록 자체를 막지 않는다. 인자는 Sonic의 지역적인 무복사 탐색으로 선택하고 반환하는 문자열은 독립된 값으로 복사한다. 원본 ContextMessages·Export와 표시용 Timeline의 계약을 구분한다.

storage/sqlite의 페이지별 본문 복원은 최대 256개 항목의 마지막 참조 인덱스와 작은 읽기 캐시를 소유한다. 키는 Store 소유자와 DB 내부 payload ID의 조합이므로 서로 다른 개인 DB의 같은 숫자 ID가 섞이지 않는다. 인덱스 예산을 넘는 본문도 캐시 없이 복원한다. 표시할 필드만 선택하고 반복되는 본문은 복원 결과를 공유한다. 팀의 공용 본문은 일반 문맥과 압축 입력 모두 페이지 단위로 조회·검증하며 전체 메시지 JSON을 다시 만들지 않고 선택한 메타데이터와 결합한다. 이미 읽은 개인 도구 메타데이터는 재연결 없이 투영하고 실제 payload가 필요할 때만 해당 DB를 연다. 캐시는 페이지 밖으로 전달하거나 Store에 저장하지 않는다. 원본 내보내기·숨은 필드 제외·손상 검출·취소·큰 단일 본문 수용을 함께 검증한다.

압축 점유량 조회는 최근 실측 샘플·최신 요약 revision의 경계·이후 추가된 메시지 크기·현재 timeline을 한 SQL 문과 같은 읽기 스냅샷에서 계산한다. 원문 본문을 읽거나 추가 영속 카운터·캐시·인덱스를 만들지 않는다.

이미지·native 메시지의 바이트 예산은 전체 JSON 직렬화 없이 계산한다. 완료 명령의 취소 문맥은 결과 확정 시 해제하고 남은 자원은 요청 종료 시 정리한다.

컨텍스트 투영 캐시는 요청 실행이 소유하고 변경되지 않은 불변 메시지만 재사용한다. 요청 메시지 배열은 한 번 할당하고 각 요청이 독립적으로 소유한다. 변경된 접미사와 영향받는 native 그룹 경계만 다시 검사한다. 고정 시스템 지침·프로젝트 경로 투영과 압축 요약 투영은 각각 256KiB 이내에서 재사용하며 원본·모델·비밀키 revision 변경 시 무효화한다. 요약 제거·교체는 이전 요약 참조를 해제하고 큰 요약도 추가 캐시 없이 수용한다. revision 재검사는 지침과 전체 문맥에 함께 적용한다. Compaction이 이력을 교체하면 캐시를 해제한다. 전체 미압축 대화와 Provider 전송 본문은 유지한다. 요청 인코딩 재사용은 별도 Provider 인코더가 담당하며 네트워크 전송량·모델 토큰 수를 줄이지 않는다.

대화 선택은 디렉터리의 DB 파일이 식별 근거다. 최초 조회는 파일 16개 미만이면 읽기 작업자 한 개, 16개 이상이면 최대 두 개를 사용한다. 각 작업자는 private 메모리 SQLite 연결 하나를 재사용해 파일을 읽기 전용으로 attach/detach하며 조회 후 모두 닫는다. 파일 대기열과 최초 자료는 각각 16개로 제한하고 요약 배열 하나와 정렬 시각만 보유한다. 콜백·정렬은 호출자가 담당하며 오류·취소 시 읽기 작업자를 모두 합류한다. 별도 레지스트리·sidecar·상시 worker를 만들지 않는다. 페이지 이동은 선택 화면이 소유하는 스냅샷을 사용한다. 전체 정렬을 위해 최초 조회는 파일 수에 비례한다.

명령 완료는 실행기 알림으로 수집하고 스트림마다 상태를 조회하지 않는다. 팀의 셸은 에이전트별 최대 16개씩 병렬 실행하며 전역 실행 잠금을 두지 않는다. 명령 묶음의 후보·승인은 실행 전에 준비하되, 확인된 변경 경로의 예약은 각 실행 worker가 얻어 자기 준비 단계의 교착을 피한다. 명령 예약은 전용 파일 작업의 16개 수용량과 별도로 계산하며 실제 경로 충돌만 조율한다. 확인된 변경 경로가 없는 명령은 예약·추가 완료 대기 worker를 만들지 않는다. 경로 예약은 명령 종료·출력 보관·캡처 정리가 끝나는 전용 완료 신호로 해제하며 결과 수집 호출과 독립적이다. 신호마다 자기 예약만 해제하고 실행기 종료에서 모든 대기 worker를 합류한다. 알림 없는 실행기는 주기 조회를 사용한다. 모델 이벤트 확인 채널은 생성 요청 단위로 재사용하며 종료 직전에도 명령 결과를 확인한다.

실행 중 명령의 화면은 고정 명령 본문과 변하는 시간 머리말을 분리해 조립한다. 마스킹 revision이 같으면 들여쓰기·마스킹한 본문을 재사용하고 최대 256KiB 안에서 실제 표시 문자열의 부분 참조로 보유한다. 명령 완료·화면 종료에서 표시 참조를 해제한다. 활성 명령 원본 목록은 레지스트리 잠금 안에서 짧게 캡처하고 개별 상태 조회는 잠금 밖에서 수행하며 조회 계약은 실행기 종료와 동시 호출을 허용한다.

실행 중 명령 조회는 출력 본문을 복사하지 않고 상태와 시각을 확인한다. 스트림별 변경 번호 또는 마스킹 revision이 바뀐 경우만 본문을 복사·마스킹하고 큰 로그를 갱신한다. 캐시는 작은 발췌·경로·변경 번호만 유지하고 명령별 표시 계산을 독립적으로 수행한다. 정제 로그·결과의 게시·회수만 실행기 예산 잠금으로 직렬화하고 완성 파일을 원자적으로 교체한다. 보관은 256MiB·256건으로 제한하며 DB 확정 뒤 acknowledge한 결과만 회수한다. 미확정 자료는 공간이 없어도 버리지 않는다. 결과 수집·요청 종료·Executor 종료에서 캐시 참조를 해제한다.

### 책임 분리 성능 검증

2026-11-08 macOS arm64·Apple M1 Max·Go 1.27.3에서 동일한 변경 전 소스 복사본과 변경 후 작업 트리를 순차 실행했다. 각 benchmark는 `-tags=godbus_no_tcp -p 1 -run '^$' -benchtime=200ms -count=6 -benchmem`으로 측정하고 고정 benchstat로 비교했다. 비교 대상은 BenchmarkSkillListMetadata·BenchmarkActiveResponseState·BenchmarkDecode·BenchmarkEngineWorkload다. 전체 엔진은 loopback Provider·실제 도구·SQLite·SSE 투영을 포함한다.

| 측정 경로 | 변경 전 중앙값 | 변경 후 중앙값 | 비교 |
| --- | --- | --- | --- |
| 스킬 메타데이터 탐색 | 1.239ms · 359.3KiB | 1.228ms · 359.6KiB | 시간·할당량 유의한 차이 없음 |
| 활성 응답 상태 조회 | 23.84ns · 0B | 23.86ns · 0B | 시간 유의한 차이 없음·무할당 유지 |
| PNG decode | 2.702ms · 3.063MiB | 2.719ms · 3.063MiB | 시간·할당량 유의한 차이 없음 |
| BMP decode | 5.314ms · 3.852MiB | 5.318ms · 3.852MiB | 시간·할당량 유의한 차이 없음 |
| 엔진 OpenRouter workload | 67.15ms | 57.13ms | 낮은 지연 관측, p=0.009 |
| 엔진 Claude workload | 61.16ms | 56.29ms | 낮은 지연 관측, p=0.026 |
| 엔진 Gemini workload | 59.15ms | 56.14ms | 유의한 차이 없음, p=0.818 |

세 엔진 workload의 할당량·할당 횟수에도 유의한 증가가 없었다. 위 경로에서 책임 분리로 인한 성능 저하는 관측되지 않았다. 파일 I/O를 포함한 합성 결과이므로 운영 서버 지연·실모델 속도나 RSS 개선율로 일반화하지 않는다. 릴리스 검증은 같은 셸의 함수 호출로 수행하고 아카이브당 `go version -m`을 정확히 한 번 호출하는 계약을 fixture에서 확인한다. 검증 명령은 실제 태그·업로드를 수행하지 않는 테스트를 사용한다.

### 최적화 측정

변경한 경로의 벤치마크를 같은 입력·환경에서 반복 실행하고 시간과 호출당 할당량을 비교한다. 아래 명령에서 해당 패키지와 이름을 골라 사용한다.

| 경로 | 패키지 | 벤치마크 이름 |
| --- | --- | --- |
| 문맥 투영·사용량 대체 계산 | internal/agent/context · internal/agent/run | BenchmarkProjection · BenchmarkSummaryProjection · BenchmarkUsageFallback |
| 현재 프로세스 RSS·표시 정밀도 내 변화·상단 부분 갱신 | internal/processmemory · internal/tui/session | BenchmarkResidentBytes · BenchmarkUnchangedMemorySample · BenchmarkRoundedMemorySample · BenchmarkMemoryRefresh |
| 압축 자료 선택 | internal/agent/compact | BenchmarkSourceSelection |
| 반복 본문·팀 압축·과거 개인 기록 복원 | internal/storage/sqlite | BenchmarkPayloadPages · BenchmarkTeamSharedMemory · BenchmarkTeamHistoricalReaders |
| 공용 대화 수신 | internal/agent/team | BenchmarkPublicHistoryFanout |
| 현재 팀 현황·확인한 변경 검색과 정리 | internal/agent/team · internal/agent/team/contribution | BenchmarkTeamWorkContext · BenchmarkTeamCoveredContext · BenchmarkReviewedChanges |
| 실행 근거 해시·명령 메타데이터·빈 판단 경계 | internal/agent/team · internal/tools/local/process · internal/tools/local · internal/agent/run | BenchmarkOperationEvidence · BenchmarkCommandMetadata · BenchmarkEmptyCommandCollection · BenchmarkEmptyInstructionCheck |
| 공용 원본 참조·실행 관측의 트랜잭션 | internal/storage/sqlite | BenchmarkTeamFactCommit · BenchmarkTeamTextIdentity |
| 완료 Markdown 초기 배치·스크롤·경계 빈 줄 | internal/tui/viewport | BenchmarkMarkdownInitialLayout · BenchmarkCompletedMarkdownInitial · BenchmarkCompletedMarkdownScroll · BenchmarkTrailingRows |
| 경로 묶음·재귀 폴더 준비 | internal/tools/local | BenchmarkParallelPlanning · BenchmarkDirectoryPreparation |
| 긴 본문·변화 없는 파일 관측 | internal/tui/viewport · internal/agent/tool | BenchmarkLiveMarkdownShape · BenchmarkQueryFacts |
| 이력 줄 배치·입력 표시 | internal/tui/viewport · internal/tui/session | BenchmarkLiveTranscriptFrame · BenchmarkTerminalPacing · BenchmarkStreamDelivery · BenchmarkLiveCommandView |
| 팀 화면 반복·변경 갱신과 완료 발화 | internal/tui/viewport · internal/tui/chat/team | BenchmarkUnchangedTeamFrame · BenchmarkManyTeamCards · BenchmarkTeamCompletionExistingPair |
| Markdown·입력·셀 폭 | internal/tui/markdown · internal/tui/input · internal/tui/textlayout | BenchmarkStreamTail · BenchmarkParagraph · BenchmarkLongDraft · BenchmarkWidth |
| 스트림 마스킹·동일 파일 복수 수정 | internal/redaction · internal/tools/local/textedit | BenchmarkCleanStreamMask · BenchmarkBatchEdits |
| 요청·native·이미지 이력 인코딩 | internal/provider/protocol | BenchmarkConversationEncoding · BenchmarkColdConversationEncoding · BenchmarkPartialHistory · BenchmarkNativeHistoryEncoding · BenchmarkOwnedNativeHistory · BenchmarkCloudConversationEncoding · BenchmarkImageHistory · BenchmarkToolSchemaEncoding |
| Provider 변환·스트림 | internal/provider/protocol | BenchmarkProviderRequests · BenchmarkProviderStream · BenchmarkOpenRouterReasoningFragments · BenchmarkTextMessages · BenchmarkImageEncoding |
| 로컬 HTTP 왕복 | internal/provider/httpclient | BenchmarkGenerationEncoding |
| 도구 표시·diff·배치 준비 | internal/tui/transcript · internal/tools/local/diff · internal/tools/local · internal/tools/local/catalog | BenchmarkDisplayArguments · BenchmarkAnalysis · BenchmarkParallelPlanning · BenchmarkCandidatePreparation · BenchmarkExecutionSnapshot · BenchmarkCatalog |
| 작은 출력·대용량 출력 보관 | internal/tools/local/process | BenchmarkSmallCommandCapture · BenchmarkCappedOutput64MiB · BenchmarkSpillOutput64MiB |
| 저장·대화 목록·압축 점유량 | internal/storage/sqlite | BenchmarkOutcomeCommit · BenchmarkConversations · BenchmarkCancellableConversations · BenchmarkJournal · BenchmarkJournalTransaction · BenchmarkMemoryWatermark · BenchmarkSessionFootprint |
| 압축·저장 JSON | internal/storage/content | BenchmarkPayloadCodec · BenchmarkCompressionOptions · BenchmarkRecordContent |
| 원문 미리보기·희소 스크롤·SSE 묶음 | internal/storage/sqlite · internal/tui/viewport · internal/serverapi/openai | BenchmarkServerPreview8MiB · BenchmarkSparseMarkdownScroll · BenchmarkStreamBatch |
| 전체 엔진 부하 | cmd/meca | BenchmarkEngineWorkload |
| 이미지 읽기·이력·압축 경계 | internal/tools/local · internal/storage/sqlite | BenchmarkReadImageBytes · BenchmarkImageHistory · BenchmarkImageBoundary |

```sh
export GOFLAGS='-tags=godbus_no_tcp'
go test ./internal/provider/protocol -run '^$' -bench 'Benchmark(ConversationEncoding|ColdConversationEncoding|OwnedNativeHistory|ImageHistory)$' -benchmem -count=6
go test ./internal/tui/session -run '^$' -bench BenchmarkTerminalPacing -benchtime=1x -count=6
go test ./internal/tools/local/process -run '^$' -bench 'Benchmark(SmallCommandCapture|CappedOutput64MiB|SpillOutput64MiB)$' -benchmem -count=6
```

- Go·OS·CPU·소스 식별값·입력 크기·창 너비·명령을 함께 기록한다. 반복 결과는 `benchstat`으로 비교한다.
- 인코딩의 cache=false·cache=true는 같은 부하의 비캐시·캐시 경로다. ColdConversationEncoding은 재사용 상태가 없는 첫 요청이다. Native·이미지 캐시는 보유 메모리와 준비된 경로의 할당량을 따로 본다.
- TerminalPacing은 프레임당 12ms 지연의 가상 터미널과 29~61ms의 입력 간격을 사용한다. drawing-percent는 출력이 차지한 경과 시간이며 CPU 사용률이 아니다. echo-mean-us는 입력 표본 8개의 평균이고 echo-p95-us는 그중 최댓값이다.
- GenerationEncoding은 로컬 HTTP 서버까지 본문을 전송하고 스트림을 읽는다. 모델 추론은 수행하지 않는다. 화면·요청 준비 비용을 실제 모델 TPS나 전체 완수 시간으로 해석하지 않는다.
- 힙 할당량·동시에 보유한 힙·RSS·디스크 I/O는 별도로 측정한다. 출력 spill과 화면 밖 파싱 해제는 보유 메모리와 재조회 비용을 함께 비교한다. 장시간 실행·중지 후 자원 해제와 같은 서버의 동시 생성 부하는 별도로 확인한다.

증분 처리와 입력 경계는 퍼즈 검사로 확인한다. 같은 명령은 한 번에 퍼즈 대상 하나만 실행한다.

```sh
go test ./internal/tui/markdown -run '^$' -fuzz '^FuzzStream$' -fuzztime=10s
go test ./internal/tui/input -run '^$' -fuzz '^FuzzBoundaryEdits$' -fuzztime=10s
go test ./internal/tools/local/diff -run '^$' -fuzz '^FuzzAnalysisCounts$' -fuzztime=10s
go test ./internal/provider/protocol -run '^$' -fuzz '^FuzzWireJSONEncode$' -fuzztime=15s
go test ./internal/provider/protocol -run '^$' -fuzz '^FuzzEncoderRequest$' -fuzztime=10s
go test ./internal/provider/protocol -run '^$' -fuzz '^FuzzCloudResponse$' -fuzztime=10s
go test ./internal/storage/content -run '^$' -fuzz '^FuzzStoredJSON$' -fuzztime=15s
go test ./internal/tui/viewport -run TestDisplayResources -v
```

### 저장 구조 측정값

2026-11-07, Go 1.27.3·macOS arm64·Apple M1 Max의 로컬 임시 디렉터리에서 구현 전·후 작업 트리를 같은 입력으로 비교했다. 아래 값은 3회 측정의 중앙값이며 실제 모델 호출을 포함하지 않는다. 짧은 표본의 관측값으로 운영 환경의 지연·메모리 감소를 보장하지 않는다.

| 측정 | 변경 전 | 현재 |
| --- | ---: | ---: |
| 응답 접수→실행 경계→종료, ResponseLifecycle | 39.6ms | 13.9ms |
| 4×128KiB journal, 트랜잭션 밖 준비 | 1.291ms | 1.219ms |
| 위 journal의 쓰기 트랜잭션 구간 | 293µs | 230µs |
| 위 journal의 할당 횟수 | 468 | 459 |
| 메시지 1000건, 각 128바이트 본문 DB | 581632바이트 | 389120바이트 |
| 메시지 1000건, 각 48KiB 압축 본문 DB | 733184바이트 | 401408바이트 |
| 공용 원문 32건의 개인 압축 자료 복원 | 14.70ms | 14.74ms |

Journal·공용 복원은 `-benchmem -benchtime=500ms -count=3`, DB 크기는 `BenchmarkSessionFootprint -benchtime=1x -count=3`로 측정했다. 크기 비교는 DB를 닫아 WAL을 반영한 뒤 main 파일을 읽는다. 응답 저장의 이전 표본은 `-benchtime=100ms`, 현재 표본은 `500ms`이므로 동기화 지연의 변동도 함께 고려한다. Claude·Gemini의 같은 cached 인코딩 입력은 각각 약 0.42ms로 유지됐다.

공용 복원의 누적 할당량은 14.65MB→14.69MB이며 할당 횟수는 3905→4903이다. 출처 값과 접두부 무결성 복원 비용이 추가됐으므로 모든 경로의 할당 감소로 해석하지 않는다. DB 크기·공용 복원을 함께 3회 실행한 테스트 프로세스의 최고 RSS는 42.04MB→40.50MB로 관측했으며 `/usr/bin/time -l`의 프로세스 최고값 한 표본이다. 누수 판정·운영 한도와 구분하고 수명 검사는 레이스·취소·종료 합류 검사로 별도 확인한다.

### PGO 빌드 측정

[Go PGO](https://go.dev/doc/pgo)는 실제 호출 비중을 컴파일러에 전달하는 빌드 최적화다. `BenchmarkEngineWorkload`는 임시 프로젝트·실제 에이전트·도구·SQLite·세 Provider의 로컬 HTTP 응답·OpenAI SSE·증분 Markdown 렌더링을 하나의 요청으로 실행한다. 실제 API Key·사용자 상태·외부 모델은 사용하지 않는다. 프로파일 생성은 기존 PGO를 끄고 합성 모델 서버의 CPU 표본을 제외한다. 프로파일과 임시 테스트 실행 파일은 성공·실패 모두에서 정리하며 완성된 프로파일만 지정 경로에 원자적으로 게시한다. 프로파일은 해당 소스·Go·플랫폼·부하에 대한 빌드 입력이므로 비교 결과와 함께 관리한다.

```sh
sh scripts/performance/pgo.sh bin/pgo/default.pgo
go build -tags=godbus_no_tcp -trimpath -pgo=off -o bin/meca-baseline ./cmd/meca
go build -tags=godbus_no_tcp -trimpath -pgo="$PWD/bin/pgo/default.pgo" -o bin/meca-pgo ./cmd/meca
go test -tags=godbus_no_tcp -pgo=off ./cmd/meca -run '^$' -bench '^BenchmarkEngineWorkload$' -benchmem -count=6
go test -tags=godbus_no_tcp -pgo="$PWD/bin/pgo/default.pgo" ./cmd/meca -run '^$' -bench '^BenchmarkEngineWorkload$' -benchmem -count=6
```

환경의 영향을 줄이려면 두 빌드를 번갈아 실행하고 `benchstat`으로 비교한다. CPU 시간·할당량·벽시계 시간을 구분하며 합성 부하의 수치를 실제 모델 응답 속도로 해석하지 않는다. 단일 미세 벤치마크만으로 기본 빌드에 적용하지 않는다. 생성한 자료는 무시되는 `bin/` 안에 둔다.

## 요청 실행의 책임

Provider 수신은 바이트 idle과 생성 진전 idle을 분리한다. 요청별 ContentWatch는 텍스트·Thinking·새 도구 조각·continuation 진전을 관찰하고 소비자 콜백 동안 시간을 정지한다. keepalive·빈 조각·사용량만의 이벤트는 생성 진전으로 취급하지 않는다. 취소 cause를 protocol 경계에서 유지해 deadline이 단순 context canceled로 사라지지 않도록 한다. metadata 교환은 시도 전체에 DefaultIdleTimeout을 적용한다. 재시도는 모델 호출과 전체 작업 예산을 다시 확인한다. Thinking·중단 메시지의 취소 독립 flush와 recovery.Restore에는 5초 마감을 둔다.

OpenRouter의 빈 추론 조각과 직전과 동일한 opaque 조각은 진전으로 세지 않는다. opaque data는 문자열을 별도로 복제하지 않고 내용 유무와 지문을 확인한다. Responses에서는 새로 완료된 추론 항목을 한 번 진전으로 알리며 동일 완료 항목의 재전송은 시간을 늘리지 않는다. 역사적 reasoning을 요청으로 변환할 때에는 스트림 감시용 지문을 다시 계산하지 않는다.

단일 모드는 메인 에이전트 하나가, 팀 모드는 각 참여자가 독립된 문맥으로 같은 agent/run을 사용한다. agent/run은 실행 순서·예산·승인·명령 수명을 조율하고, 시작한 명령의 종료·취소 결과를 수집한 뒤 응답 경계에 도달한다. 단일 모드는 정상 응답으로 종료하며 팀은 agent/team의 대화·퇴장·최종 보고 판단을 거친다. 파일 검사·OS 자격 증명·세션 SQLite는 각 어댑터가 소유한다.

원본 출력은 실행 이벤트가 소유하고, 작은 체크포인트·TUI·모델 문맥은 각각 필요한 투영만 사용한다. TUI·CLI·서버는 agent/session과 팀 실행기를 공유하며 테스트가 별도 실행 루프를 만들지 않는다. cmd/meca의 실행 파일 통합 검사는 임시 HOME·프로젝트와 모의 Provider를 사용하며 실모델 E2E와 구분한다.

## 이미지 처리 책임

agent/media는 이미지 스냅샷의 값·수용 한도, provider/connection은 입력 지원 상태, provider/discovery는 서버 선언 해석, provider/diagnostic은 입력 수용 검사, provider/setup은 연결 확정을 소유한다. provider/protocol의 요청 DTO는 수신 청크와 분리하고 Chat Completions·Ollama·Responses·Claude·Gemini의 형식으로 이미지를 인코딩한다. commands/agentrequest와 agent/run은 연결에 맞게 도구 목록을 구성한다.

tools/local/imagefile은 파일 내용의 형식 판별·디코딩·대표 프레임 변환을 담당하고 local/image는 기존 경로 경계와 취소 가능한 단일 디코딩 수용량을 적용한다. local/image_read는 알려진 크기의 스냅샷 읽기와 크기 변경 검사를, imagefile/png는 PNG 청크 기반 애니메이션 판별을 담당한다. 이미지 본문은 model 메시지에만 보존하며 TUI 이벤트에는 작은 메타데이터를 전달한다.

storage/content·sqlite는 이미지 본문을 같은 세션의 payload에 저장한다. sqlite의 RecentInputs는 사용자 텍스트만, PendingTools는 미완료 호출의 인자만 복원한다. PendingTools의 식별자 조회와 인자 선택은 별도 파일이 담당하며 부분 완료된 배치도 원래 순서를 유지한다. 모델 문맥 조회는 이미지 포함 여부를 명시하고 텍스트 연결에서는 이미지 payload를 읽지 않는다. 요약·화면 조회도 이미지 바이너리를 제외한다. 이미지 미소비 그룹은 압축 경계를 제한한다. 형식·검사·용량 계약은 [이미지 입력](images.md)을 따른다.

셸 AST와 초기 환경은 승인 후보가 불변 값으로 소유하고 실행 전에는 새 파일시스템 관측으로 재분석한다. 분기는 초기 환경을 공유하며 변경된 변수·함수만 copy-on-write로 분리한다. 한 분석 안의 같은 경로 질의만 재사용하고 실행 시점에는 별칭과 inode를 다시 확인한다. 명령 전용 묶음은 승인 후 불변 인자를 다시 파싱하지 않으며 파일 묶음의 별칭 재계획은 유지한다.

명령의 경로 입장은 local.PrepareAdmitted의 명시적 콜백으로 준비하며 준비 후 런타임 타입 단언으로 주입하지 않는다. 예약 해제는 실제 subprocess 종료와 출력 정리를 소유하는 등록부에 연결한다. 공유 작업 공간은 할당 없는 경로 경계 비교로 충돌을 검사하고 실제 대기자만 알림 채널을 소유한다. 해제·취소는 관련 경로와 파일 슬롯 대기자에게만 전달하고 오래된 후보는 실행 전에 반환한다. 명령 정리는 프로세스 소유자가 경로 예약까지 합류하며 별도 완료 감시 goroutine을 두지 않는다. 팀의 상태 문맥은 모든 활성 명령의 작은 관측과 불변 담당 업무 투영을 분리한다. 공개 기록은 저장 순서를 유지하고 UI 전달은 기록 잠금 밖에서 호출자들이 순서대로 처리한다.
