# 도구 목록과 코어 연동

[문서 목록](README.md) · [에이전트 작동 방식](runtime.md) · [호스트 실행과 승인](execution-boundary.md)

MECA는 TUI·CLI·서버에서 같은 native 도구 계약을 사용한다. 모델의 호출은 인자·승인·현재 대상을 검사한 뒤 실행하고 결과를 대화에 기록한다.

## 도구 목록

공통 카탈로그는 14종이다. 요청을 시작할 때 모델과 실행 환경을 확인해 사용할 도구를 선택한다. 팀 모드는 공통 도구에 협업 도구 6종을 더한다. [공통 카탈로그](../internal/tools/local/catalog/) · [팀 카탈로그](../internal/tools/team/catalog/catalog.go)

| 도구 | 역할과 사용 조건 |
| --- | --- |
| `execute_command` | `command`에 담긴 명령을 호스트 셸에서 실행한다. 파일 조회·검색·빌드·테스트·패키지 작업과 이동·삭제에 사용한다. |
| `list_command_runs` | 현재 실행기가 소유한 명령의 상태·진행 시간·미수집 결과를 조회한다. |
| `wait_command` | `command_id`로 완료를 기다리거나 현재 상태와 출력 발췌를 가져온다. |
| `cancel_command` | `command_id`로 실행한 프로세스 그룹을 취소하고 관찰한 종료 상태와 부분 출력을 받는다. |
| `create_directory` | `paths`에 지정한 디렉터리와 빠진 상위 폴더를 만든다. |
| `write_file` | 전체 UTF-8 본문으로 파일을 만들거나 교체한다. 빈 본문도 허용한다. |
| `edit_file` | 한 파일의 `edits`를 같은 원본에 대조해 부분 변경한다. 각 `old_text`는 정확히 한 번 일치해야 한다. |
| `view_image` | 래스터 이미지의 현재 원본과 메타데이터를 읽는다. 이미지 입력이 확인된 연결에서 제공한다. [지원 형식과 한도](images.md#view_image) |
| `list_skills` | 스킬 이름·설명·출처·환경 요구사항과 목록 커서를 가져온다. |
| `load_skill` | ID 또는 경로로 `SKILL.md`를 읽어 지침과 읽기 버전을 가져온다. |
| `read_skill_file` | 검증한 스킬 루트 안에서 참고 자료를 읽는다. [스킬 읽기 계약](skills.md#사용자-호출과-모델-도구) |
| `browser_action` | 전용 브라우저에서 페이지를 열고 이동·클릭·입력 등을 수행한다. Chrome·Chromium과 `agent-browser`가 필요하다. |
| `browser_inspect` | 열린 브라우저의 스냅샷·본문·오류·스크린샷을 얻거나 보관한 관찰을 이어 읽는다. `browser_action`과 같은 실행 환경을 사용한다. |
| `web_fetch` | HTTP(S) URL의 HTML·Markdown·텍스트 문서 한 건을 읽는다. 설치된 Chrome·Chromium과 MECA의 문서 전송 어댑터를 사용한다. |
| `add_team_member` | 독립 작업의 공용 `domain` 키와 `message`를 전달해 한 명을 초대한다. `work_ref`로 기존 공용 제안을 참조하고 `member_name`으로 종료한 세션을 복구할 수 있다. [초대 기준](team.md#시작과-참여) |
| `list_team_members` | 참여 상태, 초대·담당 도메인과 공용 제안, 작업 경로·기여 요약을 조회한다. |
| `read_team_messages` | 팀 공용 대화를 `after` 순번 이후부터 페이지로 읽는다. |
| `set_work` | 합의한 기여, 파일 경로와 현재 맡은 `domains`를 기록한다. 빈 `paths`로 파일 범위를, 빈 `domains`로 업무 담당을 해제한다. |
| `discuss` | 제안·질문·답변·검토 결과를 팀 공용 대화에 게시한다. `reply_to`와 `mentions`로 관련 발언과 동료를 지정한다. |
| `leave_team` | 팀에 작업 결과를 남기고 현재 참여자 세션을 종료한다. 마지막 참가자는 사용자에게 직접 결과를 보고한다. |

명령·파일 도구의 실행 범위는 승인 정책을 따르고, 팀의 파일 변경에는 공용 경로 조율도 적용한다. 사용자가 화면에서 입력하는 `!명령`과 서버 관리 명령은 각각의 사용자·API 계약에 정의한다. [승인 정책](execution-boundary.md) · [팀 모드](team.md) · [사용자 명령](usage.md#내부-명령과-키보드) · [서버 명령](server.md#공통-명령)

## 코어 등록과 실행

1. `cmd/meca`가 프로젝트·사용자 홈·자격 증명 마스킹·스킬 Reader와 WebFetch 클라이언트를 로컬 실행기에 연결한다. 팀 실행기는 참여자마다 로컬 실행기를 만들고 협업 도구와 공유 프로젝트 경계를 붙인다. [로컬 실행기](../internal/tools/local/executor.go) · [팀 실행기](../internal/tools/team/executor.go)
2. `internal/commands/agentrequest`가 설정·선택한 모델·세션·실행기를 `internal/agent/session`에 넘긴다. 실행 시작 시 `Definitions()`로 도구 설명·인자 스키마·효과를 선택하고, 모델의 이미지 지원과 설치된 외부 런타임에 맞춰 도구와 지침을 구성한다. [요청 조립](../internal/commands/agentrequest/request.go) · [도구 선택](../internal/agent/run/start.go) · [요청 지침](../internal/agent/context/request.go)
3. `internal/provider/protocol`이 정의를 Provider의 native 형식으로 인코딩하고, 응답 스트림에서 완료된 호출의 ID·이름·인자를 조립한다. 실행 입력은 native 호출이며 일반 답변의 JSON·셸 텍스트는 답변 본문으로 처리한다. 형식이 깨진 호출은 모델 판단에서 교정하고, 인자나 대상을 검증할 수 없는 호출은 실패 결과로 연결한다. [Provider별 형식](model-connections.md#native-tool-calling) · [복구와 교정](runtime.md#모델-스트림과-복구)
4. `internal/agent/run`이 단일 호출 또는 호출 묶음을 처리한다. `Prepare`는 카탈로그 스키마로 인자를 해석하고 대상·현재 상태·승인 조건을 묶어 실행 후보를 만든다. 정책과 승인 결과를 실행 경계에서 적용하고 변경 대상과 승인한 원본을 다시 확인한다. 독립 호출에는 병렬 실행을 허용하고 결과는 원래 호출에 연결한다. [도구 계약](../internal/agent/tool/) · [후보 준비](../internal/tools/local/prepare.go) · [실행과 승인](../internal/agent/run/execute.go) · [호출 묶음](../internal/agent/run/batch.go)
5. 실행 결과는 모델 호출 ID에 연결해 저장하고 다음 판단에 전달한다. 도구 실패는 `code`·`detail`·`next`와 수집한 출력을 전달한다. 실행기는 호출마다 별도 실행 ID를 부여해 승인과 표시 사건을 같은 실행에 연결한다. 작업을 마칠 때 소유한 명령과 관찰을 회수한다. [실행 ID](../internal/agent/run/execution_identity.go) · [결과 반영](../internal/agent/run/action_result.go) · [요청 종료](../internal/agent/run/conversation.go)

## 도구 추가와 검증

기존 도구를 변경하거나 새 도구를 추가할 때는 설명·인자·효과를 해당 카탈로그에 정의하고, 같은 스키마를 모델 요청과 입력 파싱에 사용한다. 로컬 도구의 실행 책임은 `internal/tools/local` 또는 해당 어댑터에 두고 `Prepare`에서 후보를 연결한다. 협업 동작은 팀 카탈로그와 팀 실행기가 소유한다. Provider별 형식 변화는 Provider 어댑터에 둔다. [로컬 카탈로그](../internal/tools/local/catalog/catalog.go) · [로컬 인자 파싱](../internal/tools/local/catalog/arguments.go) · [팀 실행기](../internal/tools/team/executor.go)

검증은 실제 요청에서의 제공 여부, 잘못된 인자 거절, 승인 이후 대상 변경, 호출과 결과의 대응, 실패·취소 시 소유 자원 회수를 다룬다. 도구의 외부 효과와 데이터 전송은 실행 경계의 규칙을 적용하고 이 문서의 목록과 사용 조건을 함께 갱신한다. [변경 검증](development.md#변경-검증) · [실행과 승인](execution-boundary.md)

## 셸과 파일 작업

`edit_file`, `write_file`, `execute_command`는 필수 인자 `reason`에 지금 이 작업을 하는 목적을 한 구절로 담는다. 예를 들어 `수정 후 실패 테스트 재확인`처럼 쓴다. 화면에 보이는 문자를 포함하는 최대 64자의 한 줄이며, 여러 구간을 편집해도 호출당 하나다. 도구 스키마는 `reason`을 먼저 제시하고, 실제 호출은 JSON 키 순서에 관계없이 처리한다. 백그라운드 명령의 완료 기록에서도 같은 호출의 이유를 확인할 수 있다.

`execute_command`의 기본 관측 창은 30초다. 그 안에 종료하면 결과를 돌려주고, 진행 중이면 `command_id`와 현재 상태·출력을 돌려준다. `wait_command`의 기본 조회 대기는 180초다. 만료되면 같은 ID로 조회를 계속할 수 있고, 실제 취소는 `cancel_command`를 사용한다. 에이전트당 대기·실행·마무리 중인 명령은 합계 16개다. 실행기가 프로세스 그룹과 출력 파이프를 소유하며 최종 보고 전에 활성 명령과 미수집 결과를 확인한다. 분리된 daemon의 수명은 호스트 관리 범위다. [명령 인자](../internal/tools/local/catalog/commands.go) · [명령 등록부](../internal/tools/local/process/registry.go)

명령 결과는 시작 여부·stdout·stderr·종료 상태를 기록한다. 긴 출력은 발췌와 현재 실행기에서 사용할 재조회 로그 경로를 제공하고, 정제한 원본은 세션 기록에 보존한다. 취소하거나 실패하면 확인한 부분 출력과 효과를 남긴다. 로그 파일은 실행기가 닫힐 때 회수한다. [출력 처리](usage.md#긴-출력) · [실행 사실과 저장](runtime.md#출력과-저장)

개발 스킬의 참고 자료를 찾을 때는 `execute_command`에서 `meca develop`을 호출하고 반환된 경로를 `read_skill_file`로 읽는다. [개발 자료 검색](cli.md#개발-자료-검색)

상대 파일 경로는 세션의 `project_root`에서 해석한다. `write_file`은 전체 본문을 저장하고 `edit_file`은 겹치지 않는 모든 변경을 원본에 검증한 뒤 적용한다. 두 도구의 기본 `format_policy=preserve`는 기존 줄바꿈·EOF 형식을 보존하며, 의도적으로 형식을 바꾸려면 `literal`을 사용한다. 파일 이동·삭제의 정책과 셸의 환경은 [호스트 실행과 승인](execution-boundary.md)을 따른다.

## 브라우저 연동

로컬 실행기의 `browser.Session`은 설치 상태를 확인해 호출 카탈로그에 브라우저 기능을 노출한다. `browser_action`과 `browser_inspect`는 구조화한 요청을 `agent-browser`의 명령 인자로 전달한다. `web_fetch`는 Go HTTP 전송부와 독립 Chromium의 DevTools 파이프를 사용한다. MECA가 실행되는 호스트가 두 경로를 소유하며, 서버 모드에서는 서버 호스트에서 접속하고 렌더링한다. 내장 WebUI는 사용자 입력과 관찰을 전달하는 화면 어댑터다. [가용성 검사](../internal/tools/browser/runtime/availability.go) · [도우미 호출](../internal/tools/browser/capture.go) · [문서 렌더링](../internal/tools/browser/webfetch.go)

조작 세션은 첫 `open`에서 시작한다. 세션마다 실행 경로·전용 이름·임시 설정과 소켓을 갖고, 팀 참여자는 독립 세션을 사용한다. `close`는 조작 세션과 해당 관찰을 정리한다. 작업 종료·취소 시 실행기는 조작 세션과 남은 WebFetch 결과를 함께 회수한다. 사용자 브라우저에서 열어 둔 탭이나 개인 프로필은 연동 인자로 받지 않는다. [세션 시작과 종료](../internal/tools/browser/session.go) · [관찰 수명](../internal/tools/local/observations.go)

조작 세션과 WebFetch의 HTML 렌더링은 프로세스당 기본 3개의 브라우저 입장 슬롯을 공유한다. `MECA_BROWSER_MAX_SESSIONS`에 양의 정수를 지정해 바꿀 수 있다. macOS 메모리 압박이 경고·위험 상태이거나 Linux·WSL 가용량과 cgroup v2를 반영한 잔여량이 512MiB 미만이면 새 브라우저를 열지 않는다. 측정 실패도 오류로 전달한다. 슬롯은 브라우저 시작을 제어하며 프로세스 RSS의 측정값은 별도다. [입장 관리](../internal/tools/browser/admission.go) · [자원 관리 구현](optimization-strategies.md#브라우저-관찰과-세션)

### 설치와 갱신

배포 아카이브를 푼 디렉터리에서 본체와 브라우저 도구를 함께 설치한다.

```sh
./meca install --ext browser
```

기존 설치에서는 MECA를 종료한 뒤 `meca install --ext browser`를 실행한다. 원격 서버나 컨테이너에서는 실제 MECA 실행 계정으로 설치한다. 확장 설치는 `~/.meca/browser/runtime/`에 `agent-browser`를 준비하고 기존 Chrome·Chromium 또는 관리형 Chrome for Testing Stable을 선택한다. 빈 페이지의 열기·스냅샷·종료를 확인하고 실행 경로를 출력한다. 설치가 실패하면 표시된 원인을 해결하고 다시 실행할 수 있다. [브라우저 설치 구현](../internal/tools/browser/runtime/install.go)

자체 브라우저의 실행 파일은 `MECA_BROWSER_EXECUTABLE`에 절대 경로로 지정한다. 지원 아키텍처는 macOS·Linux·WSL의 amd64·arm64이며 Linux·WSL에는 Chrome 시스템 라이브러리가 필요하다. Docker에서는 `--init`처럼 자식 프로세스를 회수하는 init을 사용한다. 도우미의 실행 설정에 따라 Chrome 샌드박스가 비활성화될 수 있다. [설치와 제거](releases.md#다운로드와-설치) · [호스트 권한](execution-boundary.md)

관리형 Chrome을 사용하는 새 조작 세션은 마지막 확인에서 24시간이 지났으면 Stable을 확인한다. 새 실행 파일을 검증한 뒤 사용하며 실패 시 기존 설치본을 선택한다. 사용 중인 버전은 각 세션이 반환할 때까지 보존하고 현재·직전 버전을 유지한다. WebFetch는 이미 설치된 브라우저 경로를 확보해 사용한다. [관리 설치본](../internal/tools/browser/runtime/managed.go) · [설치 경로 선택](../internal/tools/browser/runtime/install.go)

관리 설치 시 배포된 도우미와 Chrome의 고지를 함께 보관한다. [외부 소프트웨어 고지](../THIRD_PARTY_NOTICES.md)

### 브라우저 조작

`browser_action`은 열기, 탐색 기록 이동, 새로고침, 클릭, 입력, 키 입력, 호버, 선택·체크, 스크롤, 요소 대기, 뷰포트 변경과 닫기를 지원한다. 조작 대상은 현재 스냅샷의 요소 참조나 확인한 선택자로 지정한다. URL은 HTTP(S)로 제한하며 URL에 포함한 인증정보를 거절한다. 인자는 도구의 구조화된 필드로 전달한다. [조작 스키마](../internal/tools/local/catalog/browser.go) · [조작 승인](execution-boundary.md#브라우저-조작-승인)

`browser_inspect`의 `snapshot`은 요소 참조와 본문·상태를 반환한다. 기본 깊이는 5이며 `selector`로 범위를 좁힐 수 있다. `text`는 페이지나 영역의 텍스트를, `errors`는 브라우저 오류를, `screenshot`은 현재 뷰포트의 PNG를 제공한다. 스크린샷은 모델의 이미지 입력 지원이 필요하며 최대 4,096,000픽셀이다. 크기·픽셀 수 검사 후 메모리 예산 안에서 디코딩하고 모델 호출 동안 원본을 유지한다. [관찰 구현](../internal/tools/browser/inspect.go) · [이미지 자원과 저장](images.md#자원과-저장)

### 웹 문서 조회

사용자 요청에 URL을 넣으면 `web_fetch`로 문서를 읽을 수 있다. 첫 호출은 `url`을 받고, 쿼리와 fragment를 제외한 최종 주소·제목·본문 첫 페이지를 반환한다. HTML은 전용 Chromium에서 JavaScript를 실행해 본문 DOM을 고른 뒤 제목·목록·표·코드를 Markdown으로 변환한다. 일반 텍스트와 Markdown은 바로 읽는다. [문서 읽기](../internal/tools/browser/webfetch/document.go) · [도구 스키마](../internal/tools/local/catalog/browser.go)

공개 주소의 조회는 작업에 필요한 URL을 사용한다. 로컬·사설망 주소는 사용자가 현재 요청에 넣은 URL과 일치해야 한다. DNS로 검증한 IP로만 접속하며 메타데이터·링크 로컬·특수 목적 주소와 공개·사설 IP가 섞인 응답을 거절한다. 공개 URL은 공개 주소로 리다이렉트하고, 내부 URL은 같은 출처에서 리다이렉트한다. HTTPS의 호스트와 인증서 체인은 MECA 실행 호스트의 신뢰 저장소로 오프라인에서 확인한다. [목적지 검증](../internal/tools/browser/webfetch/destination.go) · [네트워크 사용 원칙](../SYSTEM.md#네트워크-사용과-데이터-전송)

Go 전송부가 원본 HTTP 응답을 읽고 Chromium의 DevTools 요청을 가로챈다. 최종 문서와 같은 출처의 스크립트·스타일·Fetch/XHR에 필요한 GET·HEAD만 검증한 연결로 가져온다. 독립 프로필을 사용하고 직접 접속은 거절 프록시로 보낸다. 개인 세션의 쿠키·인증 헤더·Referer와 다른 출처 리소스·쓰기 요청은 전달하지 않는다. 문서 추출과 페이지 구성은 MECA 안에서 수행한다. [Chromium 리소스 처리](../internal/tools/browser/chromium/resources.go) · [전송 구현](../internal/tools/browser/webfetch/client.go)

압축 해제 문서·렌더링 HTML·변환 본문은 각각 최대 2MiB다. 렌더링 리소스는 최대 48건, 응답 합계 4MiB로 제한한다. 한 요청에 이어 읽기 중인 문서는 최대 8개다. [WebFetch 자원 한도](optimization-strategies.md#웹-문서-조회)

### 관찰 결과와 이어 읽기

같은 출처의 동일한 브라우저 관찰은 `unchanged=true`와 기존 결과 참조를 돌려준다. 새 텍스트 관찰과 새 웹 문서의 첫 페이지는 기본 2KiB, 이어 읽기는 기본 8KiB다. 페이지당 최대 16KiB를 요청할 수 있으며 WebFetch는 남은 모델 문맥과 같은 응답에서 요청한 페이지 수에 맞춰 줄인다. 조작 세션의 관찰 원문 하나는 최대 4MiB이며 세션당 최대 16개·32MiB를 보관한다. [관찰 저장](../internal/tools/browser/paging.go) · [WebFetch 페이지 예산](../internal/agent/run/webfetch_budget.go)

`next_offset`이 있으면 같은 결과의 `result_id`와 `revision`을 넘기고 `offset`에 그 값을 사용한다. 현재 커서로 읽으면 다음 위치를 갱신하며, 다 읽은 임시 블록은 반환한다. 지난 위치나 건너뛴 위치는 오류로 처리한다. 조작 관찰은 `close`·용량 회수·작업 종료 시, 문서 커서는 해당 결과 회수나 작업 종료 시 만료된다.

본문은 외부 입력으로 취급하고, 스크린샷 사용과 공유는 사용자 작업 범위를 따른다. 완료된 모델 판단에서 읽은 페이지와 스크린샷은 다음 모델 문맥에서 열람 표시로 바뀌며 전달한 원문은 이력에서 조회할 수 있다. 압축은 대기 중인 관찰과 연결된 호출을 보존하고, 실행기가 보유한 활성 커서를 요약과 함께 전달한다. [문맥과 압축](runtime.md#문맥압축재개) · [관찰 문맥 처리](../internal/agent/run/observation_pages.go)
