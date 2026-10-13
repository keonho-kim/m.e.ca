# V / v-code

Go로 구현하는 터미널 코딩 에이전트다. 제품 요구사항은 [SYSTEM.md](SYSTEM.md), 개발 규칙은 [AGENTS.md](AGENTS.md)를 따른다. 실행 파일은 `v` 하나이며, Linux·macOS용 흑백 TUI에서 모델 연결, 파일 도구, 승인, Plan과 대화 복구를 제공한다.

## 시작하기

Go **1.27.2**와 대화형 TTY가 필요하다.

```sh
go build -o bin/v ./cmd/v
./bin/v
./bin/v /path/to/project
./bin/v resume
```

첫 실행에서 사용자 홈의 `~/.v`에 기본 TOML 설정, SQLite DB, 로그 디렉터리를 준비한다. 기존 설정과 대화는 보존한다. 기본 정책은 `directory + approve`이며, 일반 시작은 해당 프로젝트의 최근 세션을 복구한다. 새 세션을 만들려면 `config.toml`의 `[session] restore_last = false`를 설정한다. `v resume`은 프로젝트와 관계없이 저장된 대화를 명시적으로 선택한다.

처음에는 `!model`로 Provider, Endpoint, API Key, 모델을 선택한다. 주소 필드는 실제 편집 커서와 가로 스크롤을 표시한다. `Ctrl+U`는 필드 전체를 지우고, 빈 주소·잘못된 URL은 다음 단계로 진행하지 않는다. 입력창에 `/v1`을 자동으로 재삽입하지 않는다; OpenAI 호환 API 요청의 `/v1` 경로 조립은 HTTP 어댑터에서 별도로 수행한다. Ollama, vLLM, SGLang, LM Studio와 OpenAI 호환 서버의 실제 모델 목록을 조회한다. 조회 API가 없으면 Model ID를 직접 입력할 수 있다. 모델을 적용하기 전에 부작용 없는 텍스트 기반 도구 선택·단일 인자 응답 진단을 수행한다. Provider의 native Tool Calling 스키마나 Tool Parser 설정은 요구하지 않는다.

API Key는 OS 자격 증명 저장소에만 보관한다. Linux는 동작하는 D-Bus Secret Service, macOS는 Keychain이 필요하다. 저장소가 없거나 잠겨 있으면 연결 오류를 표시하며 평문 파일로 대체하지 않는다. 인증이 필요 없는 로컬 서버는 Key를 비워 둔다.

## TUI 사용

| 입력·키 | 동작 |
| --- | --- |
| 자연어 + `Enter` | 현재 모델에 작업 요청 |
| `Alt+Enter` | 줄바꿈, 입력창은 최대 5행 |
| `!`, `!mo` | 입력창 아래에 다섯 명령의 후보·설명 표시; 접두사로 필터링 |
| 명령 후보의 `↑` / `↓`, `Enter` / `Tab` | 선택한 명령을 입력창에 삽입; 다음 `Enter`로 실행 |
| 후보 목록의 `Esc` | 입력을 유지하며 목록만 닫기 |
| `@auth`, `@src/au` | 파일·디렉터리 이름 또는 경로 후보 검색 |
| 후보 목록의 `↑` / `↓`, `Enter` | 선택한 경로를 `[이름](경로)`로 삽입; 메시지는 전송하지 않음 |
| `!dir [path]` | Column View 파일 탐색 |
| 탐색기의 방향키, `Space`, `Tab` | 폴더 탐색, 참조 선택·해제, 입력 포커스 전환 |
| `!model` | 모델 연결·선택과 지원되는 Thinking ON/OFF |
| `!permission`, `!scope` | 정책 선택 후 현재 프로젝트 또는 전역 적용 선택 |
| `!plan` | 읽기·검색만 허용하는 계획 모드 |
| 승인 대기 Plan에서 빈 입력의 `Enter` | 승인·수정 요청 메뉴 다시 열기 |
| 승인 화면의 `PageUp` / `PageDown` | 긴 변경 제안 탐색 |
| `Ctrl+C` | 현재 모델·도구 작업 취소 |
| `:q` | 기능 화면 복귀 또는 세션 종료; 미승인 Plan 이탈은 거부 |
| 마우스 휠, `PageUp` / `PageDown` | 대화 출력 탐색; 과거 기록은 DB에서 페이지로 조회 |

승인 diff의 추가·삭제 줄을 제외한 UI는 터미널 기본 전경·배경을 유지하고 배경색과 역상을 사용하지 않는다. 선택 항목은 굵은 글자, 나머지 후보와 Thinking 출력은 낮춘 글자 농도로 구분한다. 후보는 입력의 첫 명령 토큰을 작성할 때만 나타나며 자연어 본문·명령 인자·붙여넣기 중에는 열리지 않는다. 좁은 화면에서는 목록 높이를 제한하고 선택 항목을 따라 스크롤한다.

`!dir`에서는 탐색 포커스에서도 `:q`를 바로 입력할 수 있다. 하단에 입력 중인 명령이 보이며 `Enter`로 채팅 화면에 복귀한다. `Backspace`로 오타를 수정하고 `Esc`로 명령 입력만 취소한다. 입력 포커스에서도 `:q` + `Enter`를 사용한다. 탐색 포커스의 복귀 명령은 채팅 초안·선택 파일·작업·Plan을 보존하며, 붙여넣기로 실행되지 않는다.

Thinking은 기본 OFF다. 활성화되면 모델명 뒤에 `Thinking`을 표시한다. `!model` 메뉴는 `Thinking ON / OFF` 한 행으로 표시하고 선택값은 굵게, 다른 값은 낮춘 농도로 구분한다. 해당 행으로 `↑`/`↓` 이동 후 `←`는 ON, `→`는 OFF를 선택하며 `Enter`로 세션·전역 적용 위치를 고른다. 지원하지 않는 제어는 `(unavailable)`로 표시한다. Ollama는 `/api/show`의 boolean 정보를 확인하고, vLLM의 식별 가능한 GLM-4.5 계열에는 공식 `chat_template_kwargs.enable_thinking` 요청 형식을 사용한다. 임의 별칭·알 수 없는 모델에는 이 제어를 추측해 적용하지 않는다. 연결 진단에서 Thinking OFF 요청에 reasoning이 반환되면 서버 설정 오류로 중단한다. 일반 응답과 별도 Thinking 스트림은 구분하여 처리한다. 글자 농도의 실제 표현은 터미널의 bold/dim 지원과 테마에 따른다.

상태줄의 Scope는 `프로젝트` 또는 `전체 시스템`으로 표시한다. 기본 승인 정책은 문구를 생략하고 `allow-all`일 때만 `자동 승인`을 표시한다. 기본 상태줄은 `no-model │ 프로젝트`이며 설정 파일과 정책 값은 그대로 유지한다.

Plan 승인 전에는 변경과 명령 실행을 거부한다. 승인한 revision의 단계를 순서대로 실행하고, `approve` 정책이면 각 실제 변경 제안도 승인받는다. 변경 뒤에는 승인된 검증 명령의 실제 성공 결과를 완료 근거로 요구한다. 읽기만 한 작업은 읽기 검증을 사용할 수 있다. 검증 결과는 해당 검사가 성공했다는 증거이며, 임의 목표의 의미적 완성을 자동 증명하지 않는다. 검증을 할 수 없으면 사용자 검토·추가 정보를 요청한다. 승인 거부는 즉시 작업을 중단하고, 실패한 검사는 수정 후 다시 검증할 수 있다. 미승인 Plan과 중단된 Tool Call은 재시작 후 복구하며, 부작용을 자동 재실행하지 않는다. 같은 세션을 두 프로세스에서 동시에 열 수 없다.

파일 생성·수정·삭제의 승인 화면은 줄 단위 unified diff를 표시한다. `-`는 삭제, `+`는 추가, `@@`는 줄 범위이며 변경 주변 3줄만 문맥으로 남긴다. 떨어진 변경 구간은 별도 hunk로 표시하고 PageUp/PageDown으로 탐색한다. 추가 줄은 파스텔 초록색(`#DCFCE7`), 삭제 줄은 파스텔 빨강색(`#FEE2E2`) 배경과 짙은 회색(`#242424`) 글자로 화면 끝까지 강조하며, 긴 줄의 감싼 행에도 동일하게 적용한다. 헤더·문맥·승인 메뉴·명령 미리보기는 기본 배경을 유지한다. 마지막 줄바꿈 유무와 CRLF의 `\r`도 표시한다. 비교는 준비된 원본·실제 실행할 고정 내용으로 생성하며 비밀값은 제거한다. 비교 작업 한도(편집 거리 256·비교 작업 1,000,000 또는 65,536 초과 개행)를 넘으면 전체 제거·추가 형태의 확장 diff로 전환해 변경을 생략하지 않는다. 바이너리는 크기·SHA-256으로 표시하고 디렉터리 삭제·이동·명령은 해당 작업 대상으로 확인한다. 부분 승인 기능은 제공하지 않는다.

`directory` Scope에서는 파일 도구를 프로젝트 루트로 제한한다. 임의 Shell 명령의 파일시스템 격리를 아직 제공하지 않으므로 해당 Scope의 `execute_command`는 거부한다. `system` Scope의 명령은 OS 사용자 권한으로 실행되며, 승인 정책·출력 한도·취소 시 프로세스 그룹 정리를 적용한다.

## 프로젝트 지침과 긴 작업 루프

각 사용자 작업 시작 시 세션의 프로젝트 루트 `AGENTS.md`를 읽어 시스템 프롬프트에 포함한다. 개인 홈이나 부모 디렉터리의 지침을 자동 수집하지 않는다. 파일이 없으면 기본 런타임 지침을 사용한다. 32 KiB를 넘는 파일, 잘못된 인코딩, 일반 파일이 아닌 경로, 프로젝트 밖을 가리키는 링크는 오류로 중단한다. 비밀값을 제거한 지침 스냅샷과 해시를 같은 작업·승인된 Plan의 모든 단계에서 유지하고, 다음 사용자 작업에서 다시 읽는다. 지침 내용은 실행 정책을 우회할 수 없다.

Ralph 방식으로 한 반복에 하나의 동작을 수행한다. 매번 프로젝트 지침, 현재 목표, 최근 실제 도구 관측과 검증 근거를 넣어 문맥을 새로 만든다. 이전 사용자 대화는 짧은 참조만 포함하며, 전체 대화와 도구 기록은 SQLite에서 보존한다. 읽기 결과가 문맥 한도를 넘으면 잘림을 명시하고 더 작은 범위를 요청하게 한다. 현재 목표와 AGENTS 지침은 조용히 자르지 않는다.

모델과 도구의 대화 예시:

```text
모델: TOOL edit_file
런타임: 어떤 파일인가요? 경로만 답하세요.
모델: src/auth.go
런타임: 바꿀 기존 텍스트를 정확히 답하세요.
모델: ...기존 텍스트...
런타임: 대체 텍스트를 정확히 답하세요.
모델: ...새 텍스트...
런타임: 모든 인자 검증 → 변경 제안 고정 → 승인 → 실행 → 실제 결과 기록
```

문자열은 한 인자의 리터럴 값으로 받고, 코드·내용의 공백을 임의로 정리하지 않는다. 경로의 JSON 포장과 모호한 숫자는 거부한다. 각 인자에 타입별 시스템 규약을 추가하고 이 단계에는 다른 도구 목록을 보내지 않는다. 첫 경로에는 빈 인자 JSON을 제시하지 않는다. 오류가 나면 거부된 응답을 데이터로 보여주며 총 3회 안에서 교정한다. 한 줄 경로의 끝 LF/CRLF는 허용하고 내부 줄바꿈·JSON 래퍼는 거부한다. 교정 실패 시 해당 도구는 준비·승인·실행하지 않는다. 기존 사용자 프롬프트를 덮어쓰지 않아도 이 런타임 규약은 적용된다. 인사·일반 질문은 프로젝트 근거가 필요하지 않으면 도구 없이 바로 답할 수 있다. 모델이 전체 도구 JSON을 만들거나 native Tool Call을 반환하면 실행하지 않는다. 내부 실행·감사 기록의 JSON은 검증된 인자로 런타임이 구성하며, Provider HTTP 요청 자체는 각 서버의 JSON API를 사용한다.

모델의 결정 표기는 `TOOL`, `CHECK`, `DONE`, `WAIT`, `CONTINUE`다. 이 표기는 모델 프로토콜이며 사용자용 내부 명령 다섯 개는 바뀌지 않는다. `CHECK`의 파일 변경은 금지하며 Shell 검증도 기존 Scope·Permission을 적용한다. 변경 이후 읽기 확인만으로 실행 완료를 선언하지 않는다. 근거 없는 DONE, 반복되는 관측, 동일 부작용 재제안, 호출·시간 한도 소진은 성공과 구분한다.

vLLM GLM-4.5-Air의 별도 추론 출력은 서버의 `--reasoning-parser glm45` 설정이 필요하다. 현재 vLLM의 `reasoning` 필드를 읽으며, 구버전 `reasoning_content` 대체 처리는 제공하지 않는다. 요청 형식은 [GLM 공식 모델 카드](https://huggingface.co/zai-org/GLM-4.5-Air)와 [vLLM reasoning 문서](https://docs.vllm.ai/en/stable/features/reasoning_outputs/)를 따른다. 실제 서버 버전·사용자 정의 chat template에서의 의미적 지원은 별도 환경 검증 대상이다.

기본 제한은 다음과 같다. `[loop]`에서 모델과 환경에 맞게 조절할 수 있다.

```toml
[loop]
max_iterations = 128
max_model_calls = 1024
max_seconds = 14400
max_model_seconds = 600
max_context_bytes = 98304
max_argument_bytes = 65536
stall_iterations = 3
```

시간 한도에는 승인 대기 시간도 포함한다. Plan 단계마다 호출·반복 예산을 초기화하지 않는다. 인자 수집 중에는 실행 준비와 승인을 하지 않으며, 중단된 초안은 체크포인트로만 복구한다. 재시작 후 모델 호출·변경을 자동 재개하지 않는다. `Ctrl+C`로 언제든 현재 작업을 취소한다.

30B급 모델을 목표로 결정 문법과 한 응답의 책임을 줄인 설계다. 바이트 제한은 모델별 정확한 토큰 수를 보장하지 않으며 실제 모델의 문맥 길이에 맞게 낮출 수 있다. 실제 30B 모델의 성공률·지연·장시간 수렴은 아직 측정하지 않았다. 반복의 참고 원리는 [Geoffrey Huntley의 Ralph 설명](https://ghuntley.com/ralph/)이며, V의 유한 예산·승인·중단 규칙은 이 제품의 구현 결정이다.

## 프롬프트 파일과 Compaction

Bootstrap은 다음 Markdown 프롬프트를 누락된 경우에만 생성한다. 디렉터리는 `0700`, 파일은 `0600`이며 기존 수정본은 덮어쓰지 않는다. 다음 사용자 작업에서 다시 읽는다.

```text
~/.v/prompts/
  system.md       기본 시스템 프롬프트 + 프로젝트 AGENTS.md 결합
  compaction.md   항목 중심 대화 요약 프롬프트
```

원본은 저장소의 `internal/agent/prompt/defaults/`에 포함한다. `system.md`를 수정해도 Scope·Permission·승인 검사는 코드에서 계속 적용한다.

기본 프롬프트는 이 저장소 AGENTS.md의 언어 공통 원칙을 담는다. 수정 대상·호출자·계약·테스트를 먼저 검토하고, 배치와 디자인 패턴을 판단한 뒤 구현·검증·단순화한다. 대상 언어의 native 문법·기본 기능·표준 라이브러리·관용적 방식을 우선하며 언어별 예시는 넣지 않는다. 저장소 전용 Go 구조와 명령은 공통 프롬프트에 복사하지 않는다. 이미 생성된 `~/.v/prompts/system.md`는 업데이트 시에도 자동으로 덮어쓰지 않는다.

기본 발동 조건은 마지막 성공한 압축 이후 **256,000 누적 토큰**이다. `~/.v/config.toml`에서 변경할 수 있다.

```toml
[compaction]
threshold_tokens = 256000
max_summary_bytes = 16384
max_input_bytes = 65536
```

누적량은 새 사용자 입력, 모델 출력(인자 응답·Thinking 포함), 도구 결과를 기준으로 한다. 반복 전송한 시스템 지침·AGENTS.md와 압축 모델의 응답은 다시 합산하지 않는다. 모델 출력은 Provider 사용량이 있으면 해당 output token 수를 사용하고, 그 밖의 부분은 UTF-8 바이트 수의 보수적인 추정치를 사용한다. 추정 여부를 기록한다. 이 누적량은 청구용 input+output 합계나 현재 문맥 토큰 수와 다르다.

다음 결정 문맥을 구성하기 전에 임계값을 확인한다. 인자 수집·승인 대기·도구 실행 중에는 압축하지 않는다. 마지막 응답이 임계값을 넘기면 다음 문맥이 필요한 시점에 실행한다. 작은 소스 조각을 순서대로 읽고 이전 요약과 병합하므로 256,000토큰 내역을 한 요청에 넣지 않는다. 압축 호출도 Ralph의 모델 호출·시간 예산에 포함한다.

요약에는 다음 고정 섹션을 쓰며 본문은 글머리 항목으로 제한한다.

요약 프롬프트는 대화를 **frontier tree**로 분석한다. 원래 목표를 루트로, 하위 작업·Plan revision·스텝을 가지로 두고 실행·변경 파일·검증 근거를 해당 노드에 연결한다. 각 가지의 가장 깊은 미완료 노드와 부모 자체의 남은 요구를 frontier로 보존한다. 차단·중단·불명·미승인 노드도 포함하며, 의존 관계는 노드 참조로 표현한다. 완료한 가지는 근거와 변경 내역을 남겨 짧게 정리하고 회귀가 발견되면 다시 연다. 매 소스 조각을 병합할 때 다른 가지의 frontier를 유지하며, 관계가 불명확한 사실은 추측하지 않고 미배정 항목으로 남긴다. 아래 섹션 안의 중첩 항목과 동일한 노드 참조로 트리·현재 진행 지점을 이어간다. 이는 모델 분석 지침이며 트리 관계의 의미적 정확성을 런타임에서 강제하는 기능은 아니다.

- Goals: 원래 목표, 달성·부분 달성·미완료·불명 상태와 근거
- Task history: 수행 내역, 결과, 관측과 원본 참조
- Steps and changed files: Plan revision·스텝, 실제 변경 파일과 이동 대상, 미실행 의도 구분
- Verification: 검사 명령, 성공·실패·중단·미실행, 남은 불확실성
- Decisions and constraints: 사용자 결정·요구사항·거절한 접근·제약
- Pending work and blockers: 미완료 작업, 차단 원인, 필요한 정보
- Continuation: 현재 목표, 다음 작업, 미승인 계획과 중단 상태

모델 요약과 별도로 실제 task status, Plan·스텝, 변경 파일·작업, 검증 여부와 원본 ID를 구조화된 ledger로 보존한다. 이 ledger와 요약은 다음 모델 문맥의 `ConversationMemory`에 제공한다. 현재 목표·AGENTS.md·Scope·Permission·승인 상태와 인자 초안은 요약으로 대체하지 않는다. 요약의 완료 주장은 런타임 완료 판정이 아니다.

압축은 원본 메시지·도구 기록·파일 내용을 삭제하지 않는다. 성공 시에만 새 revision과 압축한 원본·사용량 범위를 원자적으로 확정한다. 취소, 잘못된 형식, 과대 요약, 호출 한도 또는 충돌이면 이전 요약과 원본을 유지한다. 원본 코드와 긴 출력은 기록에서 참조하고, 정보나 정확한 파일 목록을 버려야 한도에 들어가는 경우에는 조용히 생략하지 않고 오류로 중단한다. 모델 요약의 의미적 충실도는 실제 모델 평가가 추가로 필요하다.

## 패키지 구성

```text
cmd/v/                       실행 조립과 프로세스·터미널 수명
internal/
  agent/                     모델 메시지와 소비자 소유 도구 계약
    compact/                 누적량 기반 요약과 실제 근거 보존
    dialogue/                결정 문법과 리터럴 인자 파싱
    prompt/                  프로젝트 지침과 런타임 프롬프트
      defaults/              bootstrap에 포함할 Markdown 프롬프트
    plan/                    결정적인 Plan 상태 전이
    run/                     Ralph 반복, 인자 대화, 검증과 체크포인트
  commands/                  다섯 내부 명령 파서와 후보 카탈로그
    startup/                 실행 인수와 프로젝트 경로 해석
  config/                    TOML 기본값·저장·설정 우선순위
    bootstrap/               ~/.v 초기화와 초기 설정·프롬프트 조립
    prompts/                 사용자 소유 Markdown 파일 생성·읽기
    statefs/                 private 디렉터리 검증과 원자적 파일 게시
    credentials/             OS 자격 증명 및 비밀값 제거
  policy/                    Scope·Permission 결정
  provider/
    connection/              Provider·Endpoint 검증과 모델 발견 결과
    httpclient/              HTTP·NDJSON·SSE 어댑터
  storage/sqlite/            스키마 생성·기록·복구·세션 잠금
  tools/local/               scoped 파일·검색·승인된 실행 조립
    diff/                    고정된 파일 변경의 줄 단위 승인 미리보기
    process/                 OS 프로세스·출력 제한·취소·그룹 정리
  tui/
    session/                 활성 세션 이벤트 루프와 입력·화면 조정
    startup/                 이어하기 목록과 프로젝트 재검증
    view/                    Unicode 셀·상태줄·입력·글자 농도 렌더링
    input/                   Unicode 입력과 Markdown 참조
    browser/                 Column View 상태
    viewport/                출력 투영과 읽기 위치
```

하위호환은 유지하지 않는다. 새 DB는 현재 스키마(version 4)를 직접 생성하며, 현재 버전의 DB만 연다. 이전 스키마나 내용이 있는 미버전 DB는 자동 변환·토큰 보충 없이 시작을 거부하고 원본을 보존한다.

단일 Go 모듈을 유지하며 `cmd/v`가 구체 의존성을 조립한다. 하위 패키지는 책임·입출력 경계로 나누고, 이전 위치의 호환 래퍼나 빈 Go 패키지를 두지 않는다. `config/bootstrap`은 설정·프롬프트 생성의 조립만 담당하고, `statefs`는 두 파일 소유자가 공유하는 private I/O 경계다. 연결 설정은 명령 파서에 의존하지 않는다. `tools/local`이 Scope·승인을 검사한 뒤 `process`를 호출한다. `tui/session`은 가변 상태와 worker 수명을 소유하며 `view`에는 표시할 값만 전달한다. 세션 내부도 이벤트 반영·모델 연결·설정 적용을 별도 파일로 나눈다. Agent 상태 전이와 SQLite 트랜잭션은 응집된 기존 소유 패키지에 유지한다.

## 의존성과 검증 범위

직접 의존성은 `go.mod`에 정확한 버전으로 고정한다. 유지되는 공개 버전과 라이선스, 도입 목적을 확인했으며, 불필요한 웹·위젯 프레임워크는 사용하지 않는다. 2026-10-13에 직접 의존성과 빌드·테스트에 사용하는 간접 의존성을 최신 안정 버전으로 확인·갱신했다. 자동 업데이트는 아니며 이후 변경도 정확한 버전으로 고정하고 검증한다.

| 모듈 | 버전 | 목적·라이선스 |
| --- | --- | --- |
| `github.com/gdamore/tcell/v3` | 3.6.0 | 터미널 셀·입력·마우스·시뮬레이션, Apache-2.0 |
| `github.com/rivo/uniseg` | 0.4.7 | 결합문자와 CJK 셀 폭, MIT |
| `github.com/pelletier/go-toml/v2` | 2.4.3 | 설정 파싱, MIT |
| `github.com/zalando/go-keyring` | 0.2.8 | Keychain·Secret Service, MIT; 일부 소스 Apache-2.0 |
| `golang.org/x/term` | 0.47.0 | 대화형 TTY 확인, BSD-3-Clause |
| `modernc.org/sqlite` | 1.60.1 | CGO 없는 SQLite, BSD-3-Clause 및 포함 구성요소 라이선스 |

tcell v3의 `EventQ`, 문자열 키 입력, `Put`/`Get`, 스타일 getter를 직접 사용한다. 가상 터미널 테스트는 v3의 `vt.MockTerm`을 사용하며, 이 테스트 API는 upstream에서 안정성을 보장하지 않으므로 버전 갱신 시 함께 검증한다. 이벤트 전달용 별도 goroutine과 v2 호환 래퍼는 두지 않는다. v3가 추가한 간접 의존성은 `clipperhouse/displaywidth`·`uax29/v2`로 Unicode 셀 폭·문자 경계를 처리하며 MIT 라이선스다.

SQLite 공식 지침에 따라 `modernc.org/libc`는 드라이버 `v1.60.1`이 사용하는 **v1.77.1**과 정확히 맞춘다. 단독 최신 버전으로 올리지 않는다. [드라이버의 libc 호환성 지침](https://pkg.go.dev/modernc.org/sqlite@v1.60.1#hdr-Fragile_modernc_org_libc_dependency)을 따른다. `go list -m -u all`에는 upstream 자체의 개발 도구·테스트 모듈도 나타나므로 V의 실제 빌드·테스트 의존성과 구분한다. 이 모듈들을 최신으로 표시하기 위해 사용하지 않는 직접 의존성을 추가하지 않는다.

SQLite는 Go 변환 런타임을 포함하므로 실행 파일 크기와 컴파일 비용이 있다. 현재 macOS arm64 기본 빌드의 실행 파일은 약 17.4 MiB다. CGO 없이 Linux amd64·arm64와 macOS amd64 교차 빌드를 확인했다. 빌드 성공을 실제 OS의 터미널·보안 동작 검증으로 간주하지 않는다. 자격 증명 저장소는 플랫폼 서비스와 시스템 도구에 의존한다.

공식 자료: [tcell 3.6.0](https://pkg.go.dev/github.com/gdamore/tcell/v3@v3.6.0), [Ollama Thinking](https://docs.ollama.com/capabilities/thinking), [SQLite 드라이버](https://pkg.go.dev/modernc.org/sqlite@v1.60.1), [go-keyring 소스](https://github.com/zalando/go-keyring/tree/v0.2.8).

```sh
gofmt -l cmd internal
go test ./...
go vet ./...
go build ./...
go build -o /tmp/v-code-check ./cmd/v
go test -race ./...
go mod verify
```

현재 검증은 Go 1.27.2, macOS arm64에서 수행한다. 파일 정책·승인·Plan·HTTP·SQLite·입력·화면·단계별 도구·Ralph 예산·정체·체크포인트·Compaction 임계값과 문맥 연결은 자동 테스트를 사용하며, 실제 모델 서버 대신 `httptest`를 사용한다. 실 TTY 시작·파일 탐색·이어하기·프롬프트 bootstrap·종료 복원도 확인한다. 실제 모델의 요약 품질, Provider별 서버 버전, Linux 네이티브 TTY·Secret Service, 밝은·어두운 터미널별 시인성과 SSH는 별도 환경 검증이 남아 있다. Windows 네이티브 지원과 릴리스 자동화는 제공하지 않는다.
