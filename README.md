# V / v-code

Go로 구현하는 터미널 코딩 에이전트다. 제품 요구사항은 [SYSTEM.md](SYSTEM.md), 개발 규칙은 [AGENTS.md](AGENTS.md)를 따른다. 실행 파일은 `v` 하나이며, Linux·macOS용 흑백 TUI에서 모델 연결, 파일 도구, 승인, Plan과 대화 복구를 제공한다.

## 직접 빌드하기

지원 환경은 **Linux 또는 macOS의 amd64·arm64**다. Git과 **Go 1.27.2**를 준비한다. Linux 빌드에는 Bubblewrap을 실행 파일에 포함하기 위해 **실행 중인 Docker 엔진과 BuildKit**도 필요하다. macOS 빌드에는 Docker가 필요하지 않다.

### 1. 소스 받기

```sh
git clone https://github.com/keonho-kim/v-code.git
cd v-code
go version
go mod download
```

비공개 저장소라면 GitHub 접근 권한이 필요하다. 이후 명령은 모두 `go.mod`가 있는 저장소 루트에서 실행한다.

### 2. 운영체제에 맞게 빌드

**macOS — Apple Silicon·Intel**

```sh
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -o bin/v ./cmd/v
```

현재 Mac의 CPU에 맞는 `bin/v`가 생성된다.

**Linux — amd64·arm64**

```sh
docker version
sh scripts/build-bwrap.sh "$(go env GOHOSTARCH)"
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -o bin/v ./cmd/v
```

첫 명령으로 Docker 엔진 연결을 확인하고, 다음 스크립트로 현재 CPU용 Bubblewrap을 준비한다. 이후 Go 빌드가 helper를 `v` 안에 포함한다. helper 준비 없이 Go 빌드만 하면 프로젝트 Scope의 셸 실행을 사용할 수 없다. 생성된 helper와 `bin/`은 Git에서 제외된다.

### 3. 실행

대화형 터미널에서 실행한다.

```sh
./bin/v
./bin/v /path/to/project
./bin/v resume
```

빌드한 바이너리 실행에는 Go나 Docker가 필요하지 않다. 모델 서버는 별도로 연결하며, Linux의 프로젝트 셸 실행에는 호스트의 사용자·마운트 네임스페이스 허용이 필요하다.

### 4. PATH에 설치하기 — 선택 사항

```sh
mkdir -p ~/.local/bin
install -m 755 bin/v ~/.local/bin/v
export PATH="$HOME/.local/bin:$PATH"
v
```

새 터미널에서도 `v`를 쓰려면 사용하는 셸의 설정 파일에 `~/.local/bin`을 PATH에 추가한다.

## 처음 실행하기

첫 실행에서 사용자 홈의 `~/.v`에 기본 TOML 설정, 세션 DB 디렉터리, 로그 디렉터리를 준비한다. 기존 설정과 대화는 보존한다. 기본 정책은 `directory + approve`이며, 일반 시작은 해당 프로젝트의 최근 세션을 복구한다. 새 세션을 만들려면 `config.toml`의 `[session] restore_last = false`를 설정한다. `v resume`은 프로젝트와 관계없이 저장된 대화를 명시적으로 선택한다.

처음에는 `!model`로 Provider, Endpoint, API Key, 모델을 선택한다. 주소 필드는 실제 편집 커서와 가로 스크롤을 표시한다. `Ctrl+U`는 필드 전체를 지우고, 빈 주소·잘못된 URL은 다음 단계로 진행하지 않는다. 입력창에 `/v1`을 자동으로 재삽입하지 않는다; OpenAI 호환 API 요청의 `/v1` 경로 조립은 HTTP 어댑터에서 별도로 수행한다. Ollama, vLLM, SGLang, LM Studio, LiteLLM Proxy와 OpenAI 호환 서버의 실제 모델 목록을 조회한다. 조회 API가 없으면 Model ID를 직접 입력할 수 있다. 모델을 적용하기 전에 부작용 없는 텍스트 기반 도구 선택·단일 인자 응답 진단을 수행한다. Provider의 native Tool Calling 스키마나 Tool Parser 설정은 요구하지 않는다.

선택한 모델의 식별 근거를 Provider별로 조회한다. `!model` 상단에는 서빙 별칭 오른쪽에 단일 원본 식별명을 흐린 글씨로 표시한다. 복수 대상·미식별·불완전한 결과에는 식별명을 표시하지 않는다. `!model → 모델 정보`에서 원본 선언·모델군·복수 대상·식별 불가, 적용 프로파일과 출처를 확인하고 `PgUp/PgDn`으로 스크롤한다. 서버가 검토된 원본 ID를 제공하면 임의 서빙 별칭에도 자동 매칭하며 실제 요청의 별칭은 유지한다. 여러 LiteLLM upstream·불완전한 정보에는 자동 특화를 적용하지 않는다. 메타데이터 조회가 불가능해도 대화 진단을 진행한다. 식별과 Thinking 제어 검증은 별개다. [Provider별 조회와 제한](docs/model-profiles.md#provider별-원본-식별)을 참고한다.

서버가 SSE·NDJSON 스트림을 보내면 응답 본문과 Thinking을 도착하는 대로 갱신한다. 내부 `DONE`·`WAIT` 헤더, 도구 선택·인자 응답, 검증 전 완료 선언과 Plan 초안은 스트림 본문에 노출하지 않는다. 완료된 응답은 한 번만 기록하며 취소 시 마스킹된 부분 응답을 보존한다. 서버가 단일 JSON 응답을 보내면 완료 후 표시한다.

`!model`의 Provider 목록에서 **LiteLLM**을 선택할 수 있다. 기본 주소는 `http://localhost:4000/v1`이며 원하는 Proxy 주소로 바꿀 수 있다. API Key에는 해당 Proxy의 virtual key를 입력한다(인증 없는 Proxy는 비워 둔다). `/v1/models`가 반환한 모델 별칭을 그대로 선택하고 `/v1/chat/completions`로 스트리밍 요청을 보낸다. [LiteLLM Proxy 공식 문서](https://docs.litellm.ai/docs/proxy/docker_quick_start)의 HTTP API를 사용하며 별도 SDK를 설치하지 않는다. LiteLLM의 현재 `reasoning_content` 출력은 별도 Thinking으로 읽지만, 라우팅된 모델의 ON/OFF 제어를 추측하지 않아 토글은 미지원으로 표시한다. [추론 응답 규약](https://docs.litellm.ai/docs/reasoning_content)을 따른다. 실제 Proxy 배포와 연결 모델의 응답 품질은 별도 환경 검증 대상이다.

API Key는 OS 자격 증명 저장소에만 보관한다. Linux는 동작하는 D-Bus Secret Service, macOS는 Keychain이 필요하다. 저장소가 없거나 잠겨 있으면 연결 오류를 표시하며 평문 파일로 대체하지 않는다. 인증이 필요 없는 로컬 서버는 Key를 비워 둔다.

## TUI 사용

대화는 `> 사용자 입력`, 빈 줄, `Agent`, 응답 순서로 누적된다. 응답과 다음 요청 사이에는 빈 줄 두 개를 두고, 장식용 ASCII 구분자 대신 왼쪽의 `>`·`Agent` 표시와 여백으로 구분한다. 새 입력창과 모델·정책 상태줄은 기록 바로 아래에 놓이고, 기록이 화면을 채우면 하단에서 입력한다. 상단 고정 입력 영역과 `Model answer; no tool execution.` 완료 문구는 표시하지 않는다. 도구·Thinking 기록은 해당 대화 안에서 순서대로 표시하며 이어하기에도 같은 형식을 사용한다. 검증 상태는 내부에 보존하며 별도 완료 안내 문구를 붙이지 않는다. `!`·`@` 후보는 새 입력창 아래에 표시한다.

상단 오른쪽에는 `INPUT 12345 OUTPUT 6789`처럼 현재 대화의 누적 모델 사용량을 숫자로 표시한다. 스크롤·메뉴·파일 화면에서도 고정되고, 입력창은 대화 아래에 유지된다. vLLM·SGLang에는 `include_usage`와 `continuous_usage_stats`를 요청하고, LiteLLM·일반 호환 API에는 `include_usage`를 요청한다. LM Studio Responses의 실제 usage를 읽는다. Ollama는 native `prompt_eval_count`·`eval_count`를 읽는다. 중간 usage는 호출 전체의 최신값으로 교체하고 완료 시 한 번만 합산한다.

사용량은 Provider가 실제 전송한 수치만 반영한다. 수신 전에는 기존 숫자를 유지하며, 한 번도 수신하지 않았다면 `INPUT 0 OUTPUT 0`이다. 응답 텍스트나 Thinking 길이로 추정하지 않는다. usage를 전혀 받지 않은 호출은 사용량 기록을 만들지 않고, 수신 후 추가 텍스트가 오거나 호출이 중단되면 마지막으로 수신한 값만 한 번 기록한다. 서버가 보고하지 않은 사용량은 합계에 포함되지 않는다. 이전 추정 필드가 들어 있는 사용량 저장 형식은 명시적으로 거부하며 자동 변환·삭제하지 않는다. 출력의 reasoning 세부 수치를 completion 합계에 중복 가산하지 않는다. 일반 대화, 도구 결정·인자, Plan, Compaction 호출을 포함하고 연결·설정 진단은 제외한다. 사용량은 호출별 SQLite 이벤트로 저장하여 이어하기에서 복구한다. 집계 기록이 생성되기 전 과거 호출은 소급 계산하지 않는다. Compaction의 256,000 토큰 발동 누적량과 별개다. 전체 숫자가 들어가지 않는 좁은 창에서는 통계를 생략한다.

사용량 API 근거: [Ollama](https://docs.ollama.com/api/usage), [LiteLLM](https://docs.litellm.ai/docs/completion/usage), [SGLang StreamOptions](https://github.com/sgl-project/sglang/blob/main/python/sglang/srt/entrypoints/openai/protocol.py), [vLLM 스트림 설정](https://github.com/vllm-project/vllm/blob/main/vllm/entrypoints/anthropic/serving.py).


| 입력·키 | 동작 |
| --- | --- |
| 자연어 + `Enter` | 현재 모델에 작업 요청 |
| `Alt+Enter` | 줄바꿈, 입력창은 최대 5행 |
| `!`, `!mo` | 입력창 아래에 여섯 명령의 후보·설명 표시; 접두사로 필터링 |
| 명령 후보의 `↑` / `↓`, `Enter` / `Tab` | 후보 이동; `Enter`는 선택 명령 즉시 실행, `Tab`은 삽입만 |
| 후보 목록의 `Esc` | 입력을 유지하며 목록만 닫기 |
| `@auth`, `@src/au` | 파일·디렉터리 이름 또는 경로 후보 검색 |
| 후보 목록의 `↑` / `↓`, `Enter` | 선택한 경로를 `[이름](경로)`로 삽입; 메시지는 전송하지 않음 |
| `!dir [path]` | Column View 파일 탐색 |
| 탐색기의 방향키, `Space`, `Tab` | 폴더 탐색, 참조 선택·해제, 입력 포커스 전환 |
| `!model` | 모델 연결·선택과 지원되는 Thinking ON/OFF·모델별 강도 |
| `!permission`, `!scope` | 정책 선택 후 현재 세션 또는 전역 적용 선택 |
| `!plan` | 읽기·검색만 허용하는 계획 모드 |
| 승인 대기 Plan에서 빈 입력의 `Enter` | 승인·수정 요청 메뉴 다시 열기 |
| 승인 화면의 `PageUp` / `PageDown` | 긴 변경 제안 탐색 |
| `Ctrl+C`, `!quit` + `Enter` | 모든 화면에서 현재 작업을 취소·정리하고 앱 종료 |
| `:q` | 기능 화면 복귀 또는 세션 종료; 미승인 Plan 이탈은 거부 |
| 마우스 휠, `PageUp` / `PageDown` | 대화 출력 탐색; 과거 기록은 DB에서 페이지로 조회 |

화면 하단에는 nano 형태의 문맥별 키맵을 최대 두 행 표시한다. 키는 굵게, 설명은 낮춘 농도로 표시하며 배경색·역상은 사용하지 않는다. 채팅·명령·참조 후보, 파일 탐색·입력 포커스, 모델·설정·연결 필드, 승인·이어하기 화면마다 실제 주요 동작을 안내한다. 키맵 높이를 먼저 확보해 입력창·후보·본문과 겹치지 않고, 좁은 너비에서는 두 행 안에서 주요 키부터 표시한다. 높이가 너무 작으면 본문을 우선한다.

승인 diff의 추가·삭제 줄을 제외한 UI는 터미널 기본 전경·배경을 유지하고 배경색과 역상을 사용하지 않는다. 선택 항목은 굵은 글자, 나머지 후보와 Thinking 출력은 낮춘 글자 농도로 구분한다. 후보는 입력의 첫 명령 토큰을 작성할 때만 나타나며 자연어 본문·명령 인자·붙여넣기 중에는 열리지 않는다. 좁은 화면에서는 목록 높이를 제한하고 선택 항목을 따라 스크롤한다.

도구 기록은 다음처럼 왼쪽 들여쓰기로 이름·실제 인자·실제 출력을 구분한다. 이름은 두 칸, 인자·출력은 네 칸 들여쓰고 긴 줄의 감싼 행에도 들여쓰기를 유지한다. 실행 중에는 인자와 `Running…`을 임시로 표시하고, 완료되면 실제 결과 기록 하나로 교체한다. 실패·거부·중단도 같은 블록에 표시하며, 성공 요약만으로 실제 출력을 대신하지 않는다. 결과는 기존 도구·이벤트·뷰포트 크기 한도를 적용하며 비밀값을 제거한다.

```text
  execute_command

    command: go test ./...

    Output
    exit_code: 0
    stderr:
    stdout: ok example/package
```

명령 후보는 `Enter` 한 번으로 선택한 명령을 실행하며 `!q`의 `!quit` 선택도 즉시 종료한다. `Tab`은 명령을 입력창에 삽입만 한다. 메뉴·연결 필드·파일 탐색 화면과 이어하기 선택에서도 사용할 수 있다. 종료는 승인을 대신하지 않으며 미승인 Plan과 중단된 작업은 보존한다. worker를 취소·합류하고 모델 연결을 정리한 뒤 터미널을 복원한다. 외부 SIGINT도 동일하게 정상 종료한다.

macOS에서는 터미널이 전달한 `Cmd+C`(Meta+C)도 앱 종료로 처리한다. 기본 복사 단축키로 터미널이 소비하면 앱이 받을 수 없으므로, 해당 터미널에서 `Cmd+C`를 Ctrl+C 바이트(`0x03`) 전달로 매핑해야 한다. 터미널 설정을 자동 변경하지 않는다. `Ctrl+C`는 macOS에서도 지원한다.

`!dir`에서는 탐색 포커스에서도 `:q`를 바로 입력할 수 있다. 하단에 입력 중인 명령이 보이며 `Enter`로 채팅 화면에 복귀한다. `Backspace`로 오타를 수정하고 `Esc`로 명령 입력만 취소한다. 입력 포커스에서도 `:q` + `Enter`를 사용한다. 탐색 포커스의 복귀 명령은 채팅 초안·선택 파일·작업·Plan을 보존하며, 붙여넣기로 실행되지 않는다.

Thinking은 기본 OFF다. 활성화되면 모델명 뒤에 `Thinking`을 표시하고 명시한 강도가 있으면 `Thinking MEDIUM`처럼 함께 표시한다. `!model` 메뉴는 `Thinking ON / OFF` 한 행으로 표시하고 선택값은 굵게, 다른 값은 낮춘 농도로 구분한다. `Thinking effort` 행에서는 모델·Provider에 문서화된 강도만 선택한다. 프로파일의 `DEFAULT`는 강도 필드를 보내지 않는 선택이며, 선택한 강도의 짧은 진단과 저장이 성공한 뒤 적용한다. 같은 모델·주소·프로파일의 세션 선택은 복구하고 모델별 TOML 기본값보다 우선한다. OFF에서는 강도 표시와 요청 필드를 생략하되 선호값은 유지한다. 해당 ON/OFF 행으로 `↑`/`↓` 이동 후 `←`는 ON, `→`는 OFF를 선택하며 `Enter`로 세션·전역 적용 위치를 고른다. 지원하지 않는 제어는 `(unavailable)`로 표시한다. Ollama는 `/api/show`의 boolean·문자열 지원값을 확인하고, vLLM·SGLang은 모델별 프로파일에 기록한 템플릿 제어만 사용한다. 프로파일 대상은 2026년 3월 이후 출시, 전체 300B 이하인 검토된 모델이다. 연결 진단에서 OFF 요청에 reasoning이 반환되면 중단하며, ON/OFF에 동일한 질문을 보내 별도 reasoning의 발생·억제를 확인한 경우만 토글을 활성화한다. 미식별 서빙 별칭은 `~/.v/config.toml`의 주소·Provider·모델별 매핑으로 지정할 수 있다. [모델 목록·프로파일·별칭 설정](docs/model-profiles.md)을 참고한다. 일반 응답과 별도 Thinking 스트림은 구분하여 처리한다. 글자 농도의 실제 표현은 터미널의 bold/dim 지원과 테마에 따른다.

상태줄의 Scope는 `프로젝트` 또는 `전체 시스템`으로 표시한다. 기본 승인 정책은 문구를 생략하고 `allow-all`일 때만 `자동 승인`을 표시한다. 기본 상태줄은 `no-model │ 프로젝트`이며 설정 파일과 정책 값은 그대로 유지한다.

Plan 승인 전에는 변경과 명령 실행을 거부한다. 승인한 revision의 단계를 순서대로 실행하고, `approve` 정책이면 각 실제 변경 제안도 승인받는다. 변경 뒤에는 승인된 검증 명령의 실제 성공 결과를 완료 근거로 요구한다. 읽기만 한 작업은 읽기 검증을 사용할 수 있다. 검증 결과는 해당 검사가 성공했다는 증거이며, 임의 목표의 의미적 완성을 자동 증명하지 않는다. 검증을 할 수 없으면 사용자 검토·추가 정보를 요청한다. 승인 거부는 즉시 작업을 중단하고, 실패한 검사는 수정 후 다시 검증할 수 있다. 미승인 Plan과 중단된 Tool Call은 재시작 후 복구하며, 부작용을 자동 재실행하지 않는다. 같은 세션을 두 프로세스에서 동시에 열 수 없다.

파일 생성·수정·삭제의 승인 화면은 줄 단위 unified diff를 표시한다. `-`는 삭제, `+`는 추가, `@@`는 줄 범위이며 변경 주변 3줄만 문맥으로 남긴다. 떨어진 변경 구간은 별도 hunk로 표시하고 PageUp/PageDown으로 탐색한다. 추가 줄은 파스텔 초록색(`#DCFCE7`), 삭제 줄은 파스텔 빨강색(`#FEE2E2`) 배경과 짙은 회색(`#242424`) 글자로 화면 끝까지 강조하며, 긴 줄의 감싼 행에도 동일하게 적용한다. 헤더·문맥·승인 메뉴·명령 미리보기는 기본 배경을 유지한다. 마지막 줄바꿈 유무와 CRLF의 `\r`도 표시한다. 비교는 준비된 원본·실제 실행할 고정 내용으로 생성하며 비밀값은 제거한다. 비교 작업 한도(편집 거리 256·비교 작업 1,000,000 또는 65,536 초과 개행)를 넘으면 전체 제거·추가 형태의 확장 diff로 전환해 변경을 생략하지 않는다. 바이너리는 크기·SHA-256으로 표시하고 디렉터리 삭제·이동·명령은 해당 작업 대상으로 확인한다. 부분 승인 기능은 제공하지 않는다.

`directory` Scope에서는 파일 도구와 셸 작업을 프로젝트 루트로 제한한다. 셸은 macOS Seatbelt 또는 Linux Bubblewrap으로 격리하며, 외부 파일 접근 실패를 에이전트에게 반환한다. OS·설치된 도구 실행에 필요한 경로는 읽기 전용으로, 실행별 임시 디렉터리는 작업용으로 허용한다. 프로젝트 밖 사용자 파일·다른 프로젝트·V 상태에는 접근하지 못한다. 이 Scope의 셸에서는 네트워크도 허용하지 않는다. 격리가 불가능하면 실행을 거부하며 제한 없는 실행으로 재시도하지 않는다. `system` Scope에서는 기존 OS 사용자 권한으로 실행하며 승인·출력 한도·취소 정리를 적용한다.

셸은 전용 도구로 처리하지 못하는 빌드·테스트 등의 작업에 사용한다. 직접 호출한 `rm`·`rmdir`·`unlink`는 `delete_file`, 파일 읽기·목록·검색·이동·폴더 생성 등은 해당 전용 도구를 사용하도록 오류를 반환한다. POSIX Shell AST로 직접 호출과 알려진 래퍼를 확인하며, 이 분류는 임의 스크립트 내부 동작을 증명하거나 OS 격리를 대신하지 않는다. `approve`는 실제 셸 명령문에 대한 승인을 요구하고 `allow-all`은 개별 승인만 생략한다.

`execute_command`의 모델 입력은 `command` 하나다. 명령별 시간 인자와 기본 120초·최대 600초 제한은 없다. 전체 작업 예산(`[loop].max_seconds`, 기본 4시간)과 사용자 취소는 계속 적용한다. 시간 인자를 없애도 Scope·Permission·Plan 검사는 유지하며, 자동 승인도 Scope를 넓히지 않는다. 다른 도구에는 시간 인자가 없고 `read_file`·`list_directory`의 `offset`·`limit`은 조회 위치와 분량을 정한다.

도구 실행 실패는 `code`·`detail`·`next`와 수집한 출력을 도구 결과로 저장해 다음 모델 문맥에 전달한다. Shell의 비정상 종료에는 표준 출력·오류·종료 코드가 포함된다. 정책 차단·사용자 승인 거절·취소·저장 실패로 작업이 중단되면 다음 모델 호출을 강행하지 않는다.

## 프로젝트 지침과 긴 작업 루프

각 사용자 작업 시작 시 세션의 프로젝트 루트 `AGENTS.md`를 읽어 시스템 프롬프트에 포함한다. 개인 홈이나 부모 디렉터리의 지침을 자동 수집하지 않는다. 파일이 없으면 기본 런타임 지침을 사용한다. 32 KiB를 넘는 파일, 잘못된 인코딩, 일반 파일이 아닌 경로, 프로젝트 밖을 가리키는 링크는 오류로 중단한다. 비밀값을 제거한 지침 스냅샷과 해시를 같은 작업·승인된 Plan의 모든 단계에서 유지하고, 다음 사용자 작업에서 다시 읽는다. 지침 내용은 실행 정책을 우회할 수 없다.

Ralph 방식으로 한 반복에 하나의 동작을 수행한다. Compaction 전에는 SQLite의 사용자 요청·Agent 응답·확정된 도구 호출·실제 결과 전체를 시간순으로 모델에 전달한다. 최근 두 개·1,000개 제한이나 메시지별 4 KiB 잘림을 모델 문맥에 적용하지 않는다. Compaction 후에는 요약과 요약 범위 이후의 전체 대화를 전달하고, 요약에 포함된 현재 질문도 독립된 사용자 메시지로 유지한다. 도구 기록은 native Tool Calling 대신 비실행 텍스트 데이터로 투영하며, 현재 목표·프로젝트 지침·현재 단계 질문을 별도로 전달한다. TUI의 입력 회상·스크롤 범위는 모델 문맥을 제한하지 않는다. 이전 작업의 임시 스트림은 새 작업 시작 시 초기화한다.

이 순차 도구 호출 방식의 이름은 **Progressive Field Completion(PFC)**다. LSP와 JSON 스키마 기반 모델 도구 호출은 도입하지 않는다.

모델과 도구의 대화 예시:

```text
모델: TOOL edit_file
런타임: 어떤 파일인가요? 경로만 답하세요.
모델: src/auth.go
런타임: read_file에서 받은 시작 줄 표식은?
모델: 41#9a72d30f
런타임: 같은 읽기 결과의 끝 줄 표식은?
모델: 43#73cf285a
런타임: 대체 텍스트를 정확히 답하세요.
모델: ...새 텍스트...
런타임: 모든 인자 검증 → self-grounding 요청
모델: Evidence / Action / Verify의 짧은 근거 항목
런타임: 근거 검증·저장 → 변경 제안 고정 → Conclave 심사 → 사용자 승인 정책 → 실행 → 실제 결과 기록
```

문자열은 한 인자의 리터럴 값으로 받고, 코드·내용의 공백을 임의로 정리하지 않는다. 경로의 JSON 포장과 모호한 숫자는 거부한다. 각 인자에 타입별 시스템 규약을 추가하고 이 단계에는 다른 도구 목록을 보내지 않는다. 첫 경로에는 빈 인자 JSON을 제시하지 않는다. 오류가 나면 거부된 응답을 데이터로 보여주며 총 3회 안에서 교정한다. 한 줄 경로의 끝 LF/CRLF는 허용하고 내부 줄바꿈·JSON 래퍼는 거부한다. 교정 실패 시 해당 도구는 준비·승인·실행하지 않는다. 기존 사용자 프롬프트를 덮어쓰지 않아도 이 런타임 규약은 적용된다. 인사·일반 질문은 프로젝트 근거가 필요하지 않으면 도구 없이 바로 답할 수 있다. 모델이 전체 도구 JSON을 만들거나 native Tool Call을 반환하면 실행하지 않는다. 내부 실행·감사 기록의 JSON은 검증된 인자로 런타임이 구성하며, Provider HTTP 요청 자체는 각 서버의 JSON API를 사용한다.

모델의 결정 표기는 `TOOL`, `CHECK`, `DONE`, `WAIT`, `CONTINUE`다. 이 표기는 모델 프로토콜이며 사용자용 내부 명령 여섯 개는 바뀌지 않는다. `CHECK`의 파일 변경은 금지하며 Shell 검증도 기존 Scope·Permission을 적용한다. 변경 이후 읽기 확인만으로 실행 완료를 선언하지 않는다. 근거 없는 DONE, 반복되는 관측, 동일 부작용 재제안, 호출·시간 한도 소진은 성공과 구분한다.

모든 모델 주도 도구 호출은 준비 전에 **self-grounding**을 생성한다. 2KiB 이내의 세 항목 `Evidence`(관찰 근거·미확인 사항), `Action`(이번 작업), `Verify`(확인 방법)를 요구한다. 호출 ID·도구·인자 SHA-256에 묶어 저장하며, 누락·잘못된 형식·저장 실패 시 도구를 준비하지 않는다. 기록은 실행 전 문맥과 이후 원본 이력에 포함되고, 압축 후에도 별도 ledger가 원문 근거를 보존한다. 모델이 작성한 근거는 실제 관찰의 진실성이나 실행 성공을 보증하지 않는다. 자유로운 장문 추론·승인 대체물이 아니다.

일반 요청은 내부 Goal과 순차 Task로 분석한다. 작업 Task는 실제 실행·최신 CHECK 근거, 직접 답변 Task는 답변 완료가 필요하다. 원래 요청과 하위 작업을 함께 문맥에 유지하고, 전체 Task와 최종 보고가 끝나야 Goal을 완료한다. 기존 Plan은 승인된 스텝만 실행한다. 분석·작업·심사·보고는 같은 전체 실행 예산을 공유한다.

**Conclave**는 고정된 파일 변경을 사용자 승인 전에 심사한다. 추가+삭제 합계 1–75줄은 3개 중 2개, 76–150줄은 6개 중 4개, 151–225줄은 9개 중 6개, 그 이상은 12개 중 8개의 동의가 필요하다. 같은 선택 모델을 병렬 호출하며 현재 Task·원래 요청·Monitor 관측·후보·self-grounding을 제공한다. 거절에는 사유가 필요하다. 한 심사라도 재시도 이후 오류·시간 초과·잘못된 응답이면 실행을 차단한다. 별도 부결 횟수 제한 없이 기존 전체 예산 안에서 다시 생성·심사한다. 통과해도 Scope와 사용자 승인·파일 버전 검사는 그대로 적용한다.

Monitor worker는 제안, 부결, 승인, 실제 실행과 결과를 구분해 영속화하며 다음 심사는 최근 64개 관측을 받는다. 모델이 일지를 꾸며서 작성하지 않는다. Goal/Task 상태와 심사 사유는 압축의 별도 근거 ledger에도 보존한다. 중단된 상태는 재시작 후 재검토하며 변경을 자동 재실행하지 않는다. UI에는 도구 옆 `Conclave Pass/Fail`과 부결 안내 한 문장만 표시한다.

빈 파일·빈 폴더(`create_directory`) 생성과 내용 변화가 없는 후보는 심사를 생략하고 사용자 승인 정책만 적용한다. 기존 파일을 빈 내용으로 덮거나 텍스트 파일을 삭제하면 삭제 줄 수를 심사한다. 정확한 diff 계산·전체 심사 문맥이 한도를 넘거나 바이너리 변경이면 실행하지 않는다. 파일 이동·빈 폴더 삭제에는 내용 diff가 없다. 임의 Shell 명령은 사전 파일 diff를 알 수 없어 Conclave의 파일 심사 대상이 아니다. 파일 편집은 전용 도구를 쓰도록 지시하며 Shell의 모든 부작용이 심사·격리된다고 보장하지 않는다.

`read_file`의 본문은 `41#9a72d30f | 코드`처럼 줄 번호·8자리 표식·원문을 함께 제공한다. `edit_file`은 `path` → `start_anchor` → `end_anchor` → `new_text`를 하나씩 받는다. 줄 번호는 에이전트의 위치 파악·사용자 설명용이며, 짧은 표식은 읽기 기록 식별용이다. 내부에서는 수정 범위 전체의 원문·SHA-256과 그 앞의 읽기 위치를 검증한다. 시작·끝만 같고 중간이 바뀐 경우, 앞부분 변경으로 줄의 의미가 달라질 수 있는 경우 모두 재조회를 요구한다. 대상 이후의 변경은 보존한다. 이전 `old_text` 인자는 받지 않는다.

두 표식은 같은 파일의 같은 읽기 결과에서 가져와야 한다. 표식은 파일·읽기 범위·줄 번호·원문에 묶이며, 범위나 내용이 바뀌면 달라진다. 같은 근거를 다시 읽으면 같은 표식을 유지해 무진전 감지를 보존한다. 읽기 기록은 Executor 소유 메모리에 최대 64개·약 4MiB를 유지한다. 재시작·기록 만료·충돌·서로 다른 읽기 결과 혼합은 `anchor_unavailable`, 원문이나 위치 변경은 `file_changed`로 반환한다. 부분 줄·가려진 내용에는 편집 표식을 주지 않는다. byte offset 기반 조회와 UTF-8 경계·정확한 `NextOffset`은 유지한다. CRLF 범위의 교체에는 기존 CRLF를 보존한다.

편집 결과는 수정 전후 줄 범위를 반환하며 실제 파일 변경 기록·압축 ledger에도 남긴다. 삭제는 삭제 전 범위와 `Removed`를 반환한다. 줄 번호는 실행 당시의 위치이므로 이후 변경으로 줄이 이동했다면 다시 읽고 보고한다. PFC 표식 형식 오류와 `output_truncated`는 최대 3회 교정한 뒤 구체적인 실패를 다음 결정에 전달한다. 프로세스의 정상 종료·비정상 종료·작업 예산 초과·중단은 구분하며 수집한 출력을 보존한다.

큰 프로세스 출력은 stdout·stderr 각각 앞 32KiB와 뒤 32KiB를 남기고 중간 생략을 표시한다. 실패 진단은 stderr 우선으로 마지막 4KiB 이내를 별도 보존하여 압축 후에도 파일·위치·실제 오류 메시지를 확인할 수 있다. 전체 로그가 필요하면 승인된 명령에서 프로젝트 파일로 저장한 뒤 `read_file` 범위로 재조회한다. 자동으로 전체 로그를 보관하는 기능은 아니다.

CHECK 목록은 작업의 변경 revision에 묶인다. 한 검사가 실패한 상태에서 다른 검사만 통과해도 DONE을 허용하지 않는다. 새로운 변경 가능성이 생기면 이전 검사도 재검증 대상이다. 명령 검사는 실제 `exited` 상태와 종료 코드 0이 필요하며, 누락된 종료 상태를 성공으로 간주하지 않는다. 실패한 검사는 같은 명령을 통과해야 해소된다. 서로 다른 검사는 작업당 최대 32개이며 초과 시 누락하지 않고 작업 분리를 요구한다. 모델이 선택한 검사가 사용자 목표를 충분히 검증하는지까지 자동 증명하지는 않는다.

기본 3회 무진전이면 즉시 종료하는 대신 재읽기·범위 축소·다른 가설·WAIT를 안내하는 복구 단계를 최대 2회 제공한다. 기존 반복·호출·시간 예산은 늘리지 않는다. 같은 관측과 알려진 파일 내용 상태로의 순환을 감지한다. 파일 추가나 외부 변경이 관측되면 상태 비교 기준을 갱신한다. 이미 실행을 시작한 동일 부작용 제안은 실패 여부와 관계없이 다시 실행하지 않는다. 되돌리기는 새 근거·정책·승인을 거치는 별도 변경이다.

수정 후 같은 검사를 다시 실행하는 것은 새 변경 revision의 검증으로 취급한다. 파일 버전 검사가 실제 변경 전에 거부한 경우에는 새 근거와 승인으로 같은 편집을 다시 준비할 수 있다. 다음 사용자 작업에는 이전 미완료 체크포인트의 목표·단계·검사·관측을 문맥 데이터로 제공하고, 미실행 인자 초안은 제외한다. 파일·환경 상태를 다시 확인하며 이전 명령이나 변경을 자동 재실행하지 않는다.

모델 특화는 `provider/profile`의 모델군별 파일이 소유한다. 서빙 이름·서버 원본 메타데이터의 정확한 카탈로그 일치나 명시한 매핑만 적용하며, 모델명에 `qwen` 등의 문자열이 포함된다고 제어를 추측하지 않는다. 현재 vLLM은 `reasoning`, SGLang·LiteLLM은 각 API의 `reasoning_content` 필드를 읽는다. Provider 간 필드 대체나 구버전 호환 처리는 두지 않는다. GLM-4.5-Air는 2025년 모델로 새 프로파일 범위에 포함되지 않으며 기존 vLLM 자동 특화도 제거했다. 일반 대화 연결은 사용할 수 있다. 모델의 공식 제어와 실제 서버의 template·parser 설치 상태는 연결 진단 및 별도 환경 검증 대상으로 구분한다.

기본 제한은 다음과 같다. `[loop]`에서 모델과 환경에 맞게 조절할 수 있다.

```toml
[loop]
max_iterations = 128
max_model_calls = 1024
max_seconds = 14400
max_model_seconds = 600
max_context_bytes = 4194304
max_argument_bytes = 65536
stall_iterations = 3
```

`max_model_seconds`는 모델 호출의 전체 시간이 아니라 첫 응답·다음 네트워크 데이터를 기다리는 무수신 제한이다. 계속 수신 중인 호출을 고정 시간으로 끊지 않는다. `max_seconds`는 다음 모델 호출·도구 작업을 시작하기 전에 검사하며 진행 중인 모델 수신은 완료될 때까지 허용한다. 도구 실행과 승인 대기는 남은 작업 시간 안에서 수행한다. Plan 단계마다 호출·반복·전체 시간 예산을 초기화하지 않는다. 인자 수집 중에는 실행 준비와 승인을 하지 않으며, 중단된 초안은 체크포인트로만 복구한다. 재시작 후 모델 호출·변경을 자동 재개하지 않는다. `Ctrl+C` 또는 `!quit`으로 현재 작업을 취소·정리하고 앱을 종료한다.

30B급 모델을 목표로 결정 문법과 한 응답의 책임을 줄인 설계다. 바이트 제한은 모델별 정확한 토큰 수를 보장하지 않으며 실제 모델의 문맥 길이에 맞게 낮출 수 있다. 실제 30B 모델의 성공률·지연·장시간 수렴은 아직 측정하지 않았다. 반복의 참고 원리는 [Geoffrey Huntley의 Ralph 설명](https://ghuntley.com/ralph/)이며, V의 유한 예산·승인·중단 규칙은 이 제품의 구현 결정이다.

모델 문맥 바이트 한도의 기본값은 4 MiB이며, TOML에 명시한 `max_context_bytes` 값은 그대로 적용한다. 이 한도와 Provider 요청·모델의 실제 context 한도는 Compaction 토큰 임계값과 별개다. 한도를 초과하면 대화를 생략하거나 임의 압축하지 않고 오류로 중단한다.

## 연결 저장·서버 한도·native reasoning

연결 진단과 SQLite 저장이 성공한 뒤 활성 클라이언트를 교체한다. 실패 시 후보를 닫고 기존 연결·설정을 유지한다. API Key는 모델 진단 후 OS 키 저장소의 새 참조에 저장하며, 선택 저장 실패 시 새 키를 정리한다. 기존 대화가 참조하는 키를 덮어쓰지 않는다. Thinking 변경도 저장 후 적용한다. 전역 TOML 저장 후 세션 저장이 실패하면 전역값 복원을 시도하고 복원 실패도 알린다.

선택 시 서버 한도를 다시 조회하여 저장하고 `!model → 모델 정보`에 표시한다. vLLM·SGLang은 `max_model_len`, Ollama는 `/api/ps`의 로드된 컨텍스트, LM Studio는 선택 인스턴스의 `config.context_length`를 사용한다. 모델의 이론상 최대 길이를 현재 서버 용량으로 대체하지 않는다. LiteLLM은 같은 별칭의 모든 배포가 값을 제공할 때 입력·출력 한도 각각의 최솟값을 사용한다. 한도를 확인할 수 없으면 미제공 또는 조회 실패 사유를 표시한다.

출력 요청은 확인된 출력·컨텍스트 상한 이내로 제한한다. 준비된 vLLM 모델의 컨텍스트가 알려져 있으면 `/tokenize`에 전체 역할별 메시지와 동일한 템플릿 제어값을 보내 입력 토큰을 계산하고 남은 용량만 출력에 배정한다. 용량이 소진되면 이력을 보존한 채 중단한다. 토큰 계산은 생성 usage가 아니므로 INPUT/OUTPUT에 더하지 않는다. 다른 Provider의 입력 토큰은 추정하지 않는다. LM Studio에는 `truncation: disabled`를 요청한다. Ollama에는 확인된 현재 컨텍스트를 `num_ctx`로 고정하지만, 공개 Chat API에 입력 사전 토큰 계산·잘림 거부 기능이 없어 서버 내부의 이력 잘림까지 검출·방지할 수는 없다. 컨텍스트 미제공·서버 설정 변경·라우팅 변경도 서버의 최종 검증 대상이다.

Ollama의 문자열 `think` 값은 서버가 광고한 대소문자 그대로 보낸다. LM Studio는 전체 역할별 이력을 전달하는 `/v1/responses`를 사용하며 `store: false`로 원격 대화 저장을 끈다. reasoning 선택지는 native 모델 목록에서 확인한다. 공개 강도는 `reasoning.effort`로 보내고, binary OFF/ON은 `none`/`medium` 후보를 동일 질문으로 대조 진단한 뒤 토글을 제공한다. binary 모델에 강도 목록을 임의로 추가하지 않는다. native `DEFAULT`는 ON을 표현할 수 있는 서버 기본 강도, 없으면 첫 번째 광고 강도를 사용한다. OFF를 제공하지 않는 모델은 검증된 ON을 굵게 표시하고 `OFF unavailable`을 안내하면서 지원 강도는 선택할 수 있다. 짧은 진단은 reasoning 분리와 ON/OFF 동작을 검사하며 각 강도의 내부 연산량까지 측정하지는 않는다. LM Studio의 실제 사용량은 Responses의 `usage.input_tokens`·`output_tokens`에서 읽는다.

API 근거: [Ollama thinking](https://docs.ollama.com/capabilities/thinking), [Ollama 실행 모델](https://docs.ollama.com/api/ps), [LM Studio 모델 정보](https://lmstudio.ai/docs/developer/rest/list), [LM Studio Responses](https://lmstudio.ai/docs/developer/openai-compat/responses), [vLLM 토큰 계산 계약](https://github.com/vllm-project/vllm/blob/main/vllm/entrypoints/serve/tokenize/protocol.py).

## Provider 스트림과 복구

SSE·NDJSON의 누적 전송량·누적 텍스트·단일 프레임 1MiB 제한을 제거했다. 긴 스트림은 프레임을 순차 처리하며 한 프레임 크기에 비례하는 메모리가 필요하다. 비스트리밍 JSON 4MiB, Agent 단계별 응답·인자·요약 및 저장 한도는 각 소유자의 검증 규칙으로 적용한다.

HTTP의 3분 전체 타이머와 모델 호출·연결 진단의 고정 종료 타이머를 제거했다. 모델 생성은 `[loop].max_model_seconds`의 무수신 제한(기본 600초), 메타데이터·연결 진단은 기본 3분의 무수신 제한을 사용한다. 수신 데이터가 있으면 다음 읽기에서 새로 기다리고, 소비자가 데이터를 처리하는 동안에는 네트워크 대기 타이머를 멈춘다. DNS·연결·TLS 초기화 제한과 사용자 취소는 별개다.

네트워크·HTTP·스트림 오류는 기존 연결을 정리하고 **최대 3회 시도(재시도 2회)**한 후 실패 원인을 반환한다. 기본 대기는 250ms·500ms이며 `Retry-After`는 최대 30초까지 반영한다. 메타데이터는 응답 읽기·JSON 디코딩까지 재시도하고, 이전 실패의 일부 필드를 재사용하지 않는다. 재시도도 Agent의 모델 호출 예산에 포함한다. 사용자 취소, 로컬 요청·Scope·승인·인자·저장 검증 실패와 호출 예산 소진은 즉시 중단한다.

부분 응답에서 재시도하면 응답·Thinking·Compaction 초안을 초기화한 뒤 같은 요청을 다시 생성한다. 이미 실행한 도구를 재실행하지 않는다. 토큰은 각 시도에서 실제 수신한 usage만 기록한다. 출력 한도 등의 종료 오류에서도 이어지는 usage를 수신한 뒤 복구·오류 처리를 진행한다. 최종 오류에는 시도 횟수, HTTP 상태, 제공된 오류 코드·메시지를 표시하며 API Key와 터미널 제어 문자를 제거한다. 선택적 모델 식별 실패는 원인을 `!model → 모델 정보`에 보존하고 일반 연결 진단을 계속한다.

가상 시계로 장시간 수신·무수신 복구·소비자 처리 지연·재시도 대기 취소를 검증했다. 로컬 모의 서버로 8MiB 초과 전송, 4MiB 초과 디코딩, 부분 응답 초기화, 오류 후 usage 수신과 최종 오류 마스킹을 검증했다. 실제 서버·프록시에서 장시간 연결하는 검증은 별도로 필요하다.

## 역할별 추론 설정

추론 인자는 Provider가 아니라 호출 역할이 결정한다. 새 `~/.v/config.toml`에는 아래 기본값이 모두 기록된다. 기존 파일의 사용자 값은 덮어쓰지 않으며, 생략한 항목은 현재 기본값을 사용한다. 설정을 저장할 때는 적용된 역할별 값도 기록한다. TOML 직접 수정은 앱을 다시 시작하면 적용된다.

| 역할 키 | 용도 | temperature | max_output_tokens |
| --- | --- | ---: | ---: |
| `agent` | 메인 에이전트의 행동 선택·직접 응답 | 0.7 | 2048 |
| `plan` | 승인받을 Plan 초안·수정 | 0.7 | 2048 |
| `goal` | 내부 Goal·Task 분해 | 0.2 | 2048 |
| `pfc_value` | 경로·줄 표식·숫자·검색 패턴 | 0.2 | 2048 |
| `pfc_content` | 코드 본문·교체 본문·셸 명령 | 0.7 | 8192 |
| `grounding` | Evidence·Action·Verify 기록 | 0.2 | 2048 |
| `conclave` | 동의표 또는 거절표와 구체적인 사유 | 0.4 | 2048 |
| `compaction` | 대화 압축·근거 보존 | 0.2 | 4096 |
| `report` | 여러 Task의 최종 사용자 응답 요약 | 0.2 | 2048 |
| `diagnostic` | 연결·PFC·Thinking·effort 진단 | 0.2 | 512 |

예를 들어 다음 항목을 변경한다. Provider·서빙 별칭·모델 프로파일에 따라 이 값을 바꾸지 않는다.

```toml
[inference.agent]
temperature = 0.7
max_output_tokens = 2048

[inference.conclave]
temperature = 0.4
max_output_tokens = 2048

[inference.compaction]
temperature = 0.2
max_output_tokens = 4096
```

`temperature`는 유한한 0~2, `max_output_tokens`는 1~65536이다. 명시한 `temperature = 0`은 생략과 구분하여 전송한다. Thinking과 effort는 기존의 검증된 사용자 선택을 계승한다. 진단은 기능 확인을 위해 ON/OFF를 명시적으로 대조한다. 출력 토큰 한도는 모델의 reasoning 사용분도 고려해야 하며, 기존 바이트 한도·서빙 한도·전체 실행 예산을 늘리지 않는다. Provider는 확인된 출력·컨텍스트 용량 안에서 요청 한도를 제한할 수 있다.

`top_p`, `top_k`, `min_p`, penalty, seed는 현재 임의로 설정하지 않는다. 재시도에서도 선택된 역할 인자를 그대로 사용한다. 서버가 인자를 거절하면 기존 제한된 재시도 이후 오류를 반환하며 조용히 삭제하거나 다른 값으로 바꾸지 않는다. 요청 수락만으로 서버 내부 적용까지 증명하지는 않는다.

Conclave는 한 호출에서 `1` 또는 `0` 다음 줄의 영어 거절 사유를 반환한다. 이유 없는 `0`은 오류이며 실행을 차단한다. 거절 사유만을 위한 추가 호출은 만들지 않는다. Goal·Self-grounding의 설명은 영어로, PFC는 요청한 literal 값으로 반환한다. 사용자 요청·프로젝트 지침·경로·코드·인용한 근거는 번역하지 않는다. Plan은 사용자 승인 대상이므로 사용자 언어를 유지한다. Monitor·Scope·승인·검증 상태 전이는 코드가 처리하며 새 LLM 호출을 추가하지 않는다.

역할 선택은 `agent/inference`, API 형식 변환은 `provider/protocol`이 담당한다. Ollama에는 `options.temperature`, 호환 Chat/LM Studio Responses에는 `temperature`를 전송한다. API 계약 근거: [Ollama Chat](https://docs.ollama.com/api/chat), [LM Studio Responses](https://lmstudio.ai/docs/developer/openai-compat/responses).

## 프롬프트 파일과 Compaction

Bootstrap은 다음 Markdown 프롬프트를 누락된 경우에만 생성한다. 디렉터리는 `0700`, 파일은 `0600`이며 기존 수정본은 덮어쓰지 않는다. 다음 사용자 작업에서 다시 읽는다.

```text
~/.v/prompts/
  system.md       기본 시스템 프롬프트 + 프로젝트 AGENTS.md 결합
  compaction.md   항목 중심 대화 요약 프롬프트
  goal.md         영어 Goal·Task 분해 지침
  grounding.md    영어 Evidence·Action·Verify 지침
  conclave.md     영어 심사·거절 사유 지침
  pfc.md          literal 필드 수집 지침
```

원본은 저장소의 `internal/agent/prompt/defaults/`에 포함한다. 내부 역할은 각 전용 프롬프트와 프로젝트 제약을 결합하며, 메인 에이전트의 도구 선택·출력 프로토콜 전체를 심사 프롬프트에 넣지 않는다. 내부 응답 형식·언어 지침은 프로젝트의 응답 표현 선호보다 우선하지만 Scope·승인 제약을 대체하지 않는다. `system.md`를 수정해도 Scope·Permission·승인 검사는 코드에서 계속 적용한다.

기본 프롬프트는 이 저장소 AGENTS.md의 언어 공통 원칙을 담는다. 수정 대상·호출자·계약·테스트를 먼저 검토하고, 배치와 디자인 패턴을 판단한 뒤 구현·검증·단순화한다. 대상 언어의 native 문법·기본 기능·표준 라이브러리·관용적 방식을 우선하며 언어별 예시는 넣지 않는다. 저장소 전용 Go 구조와 명령은 공통 프롬프트에 복사하지 않는다. 이미 생성된 `~/.v/prompts/system.md`는 업데이트 시에도 자동으로 덮어쓰지 않는다.

기본 발동 조건은 마지막 성공한 압축 이후 **256,000 누적 토큰**이다. `~/.v/config.toml`에서 변경할 수 있다.

```toml
[compaction]
threshold_tokens = 256000
keep_recent_bytes = 16384
max_summary_bytes = 16384
max_input_bytes = 65536
```

누적량은 새 사용자 입력, 모델 출력(인자 응답·Thinking 포함), 도구 결과를 기준으로 한다. 반복 전송한 시스템 지침·AGENTS.md와 압축 모델의 응답은 다시 합산하지 않는다. 모델 출력은 Provider 사용량이 있으면 해당 output token 수를 사용하고, 그 밖의 부분은 UTF-8 바이트 수의 보수적인 추정치를 사용한다. 추정 여부를 기록한다. 이 누적량은 청구용 input+output 합계나 현재 문맥 토큰 수와 다르다.

다음 결정 문맥을 구성하기 전에 임계값을 확인한다. 인자 수집·self-grounding·승인 대기·도구 실행 중에는 압축하지 않는다. 마지막 응답이 임계값을 넘기면 다음 문맥이 필요한 시점에 실행한다. `keep_recent_bytes`는 최근 원문 메시지의 보존 예산이며 토큰 추정값이 아니다. 메시지를 쪼개지 않으므로 마지막 메시지 하나만으로 예산을 넘을 수 있다. 기본 16KiB에 해당하는 최근 메시지를 남기고 오래된 prefix만 요약한다. 0이면 최근 구간 보존을 끈다. 요약할 오래된 구간이 없으면 기존 메모리를 유지한다. 작은 소스 조각을 순서대로 병합하며 압축 호출도 Ralph의 모델 호출·시간 예산에 포함한다.

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
cmd/v/                       구체 어댑터 조립·프로세스·터미널 수명
internal/
  agent/
    model/                   생성·메시지·스트림 계약과 메시지 검증
    tool/                    호출·승인·실행 결과·실제 작업 기록
    session/                 저장 상태·프롬프트 준비와 제출 작업 실행
    run/                     Ralph 순서·예산·승인·Plan 실행 조율
    inference/               역할별 temperature·출력 토큰 한도
    context/                 전체 대화·현재 사실·요약의 모델 문맥 조립
    progress/                파일 상태 순환 감지
    compact/                 누적량 기반 요약과 근거 보존
    pfc/                     Progressive Field Completion 파싱
    goal/                    Goal·순차 Task 상태 전이
    monitor/                 실행 사실 관측 worker
    conclave/                병렬 심사·정족수·피드백
    grounding/               정확한 호출에 묶인 self-grounding
    verification/            변경 revision별 검사 상태
    plan/                    Plan 상태 전이·승인 단계 식별
    prompt/                  프로젝트 지침과 단계별 프롬프트
      defaults/              bootstrap용 Markdown 프롬프트
  commands/                  여섯 내부 명령 파서와 후보 카탈로그
    startup/                 실행 인수·프로젝트 경로 해석
    settings/                정책·Thinking 저장과 실패 보상
  config/                    TOML 기본값·저장·우선순위
    bootstrap/               초기 설정·프롬프트 조립
    prompts/                 사용자 Markdown 파일 생성·읽기
    statefs/                 private 파일 검증·원자적 게시
    credentials/             OS 자격 증명과 비밀값 제거
  policy/                    Scope·Permission 결정
    shell/                   전용 도구로 처리할 셸 명령 판별
  provider/
    connection/              Provider·Endpoint·모델 선택 값
    profile/                 모델별 검토된 제어 규칙
    identity/                원본 식별 근거와 저장소 ID 해석
    selection/               모델 선택·Thinking 조합 검증
    discovery/               모델 목록·원본·기능·서빙 한도 조회
    diagnostic/              생성기를 통한 PFC·Thinking 진단
    protocol/                요청·SSE·NDJSON·응답 형식 변환
    transport/               HTTP·인증·idle 감시·재시도
    httpclient/              선택 모델의 생성 요청 조율
    setup/                   발견·진단·연결 저장과 소유권 이전
  storage/
    session/                 저장 매체와 무관한 세션 식별·메타데이터
    history/                 순서 있는 대화 기록·압축 자료 투영
    recovery/                중단된 작업의 복구 절차와 저장 계약
    sqlite/                  세션 DB 파일 탐색·SQL·트랜잭션·잠금
  tools/
    result/                  도구 관측 결과 인코딩
    local/                   Scope·승인·후보 준비·실행 직전 재검사
      file/                  범위 읽기·줄 표시·원자적 쓰기
      search/                파일명·본문 검색
      anchors/               읽기 스냅샷·부분 수정 검증
      diff/                  실제 변경량·승인 미리보기
      process/               프로세스·출력·취소·그룹 정리
        sandbox/             OS별 프로젝트 격리
          bundle/            Linux 내장 helper 검증·추출
  tui/
    session/                 이벤트 루프·화면 상태·worker 수명
    startup/                 이어하기 목록과 프로젝트 재검증
    modelmenu/               모델 식별·Thinking 메뉴 표시값
    transcript/              메시지·도구 이벤트의 출력 조립
    view/                    표시값을 받는 렌더링
    input/                   Unicode 입력·Markdown 참조
    browser/                 Column View 상태
    viewport/                출력 투영·읽기 위치
```

SQLite는 `~/.v/sessions/<세션 ID>.db`에 **세션당 한 파일**로 저장한다. 메시지·도구 이벤트·Goal/Task·Conclave·Plan·Compaction·사용량·모델 선택·정책 설정은 모두 해당 DB에 들어간다. 세션 ID와 목록은 `sessions/`의 `session_*.db` 파일명에서 찾으며 별도 SQLite 인덱스나 중앙 DB를 만들지 않는다. 목록의 제목·프로젝트·갱신 시각은 각 DB에서 읽기 전용으로 조회하고 즉시 닫는다. `v resume`과 `restore_last` 모두 이 목록을 사용한다.

세션 디렉터리는 `0700`, DB는 `0600`이다. 새 DB는 임시 파일에 스키마와 단일 세션 메타데이터를 완성하고 닫은 뒤 원자적으로 공개한다. 실행 중에는 같은 세션의 `.db-wal`·`.db-shm`, 소유권 잠금용 `.db.lock`이 생길 수 있으며 목록에서 제외한다. 파일 하나를 옮길 때는 해당 세션을 종료해 WAL 내용이 DB에 반영된 뒤 옮긴다. 파일명과 내부 ID 불일치·심볼릭 링크·손상 DB는 명시적 오류로 처리한다.

`Current session` 정책 변경은 현재 세션에만 저장한다. 같은 프로젝트라도 새 세션은 전역 기본값을 사용하고, 기존 세션을 이어갈 때는 그 세션의 재정의가 복구된다. 전역 기본값은 계속 `config.toml`이 소유한다.

하위호환은 유지하지 않는다. 세션 DB의 현재 스키마는 version 5이며 이전 버전·미버전 DB는 자동 변환·삭제·재생성하지 않는다. 기존 통합 `~/.v/v.db`는 그대로 보존하지만 새 목록에는 포함하거나 자동 분리하지 않는다.

저장 기능은 SQLite와 분리한다. `storage/session`은 세션 값과 식별 규칙, `storage/history`는 순서 있는 기록과 압축 자료 변환, `storage/recovery`는 중단 복구를 소유한다. SQLite는 이 계약을 구현하는 저장 어댑터로서 DB 파일·SQL·트랜잭션·잠금만 담당한다. TUI와 작업 유스케이스는 필요한 저장 인터페이스에 의존하며, `cmd/v`에서 SQLite를 연결한다. 현재 DB 스키마와 JSON 기록 형식은 이 책임 분리로 변경하지 않는다.

`tui/session`은 활성 화면 상태와 worker의 시작·취소·합류를 소유한다. 작업 준비는 `agent/session`, 정책·Thinking 저장과 실패 보상은 `commands/settings`, 연결 저장과 자격 증명 보상은 `provider/setup`이 담당한다. 모델 선택 검증은 `provider/selection`, 메뉴 표시값은 `tui/modelmenu`가 맡는다. 화면 입력·승인·출력 상태는 이벤트 루프에서만 변경한다.

Provider는 전송(`transport`), 형식 변환(`protocol`), 메타데이터 조회(`discovery`), 실제 생성 진단(`diagnostic`), 생성 요청 조율(`httpclient`)로 나눈다. HTTP 본문·재시도·idle 감시의 수명은 전송 계층이, 연결 준비와 생성 클라이언트 이전은 `setup`이 소유한다. 모델별 특화는 `profile`에 유지한다. 모델 목록·출처·제어 범위는 [모델 프로파일 문서](docs/model-profiles.md)에 기록한다.

`agent/run`은 PFC → self-grounding → 후보 준비 → Conclave → 사용자 승인 → 실행 순서와 공유 예산을 유지한다. 컨텍스트 조립은 `agent/context`, 파일 상태 순환 감지는 `agent/progress`, 실행 사실은 `agent/tool`, 승인 단계 식별은 `agent/plan`이 소유한다. Compaction은 이 사실을 소비하며 실행 기록의 타입을 대신 소유하지 않는다. `tools/local`의 Scope·승인·버전 재검사를 통과한 작업만 `file`, `search`, `process` 어댑터로 전달한다.

각 패키지는 한 기능을, 하위 패키지는 독립적인 하위 기능을, 파일은 그 안의 한 책임을 맡는다. 작은 함수와 명시적인 상태 전이를 우선하며, 순환 의존·이전 위치의 전달 패키지·호환 별칭·사용되지 않는 추상화는 두지 않는다. 단일 Go 모듈과 `cmd/v` 실행 파일을 유지한다.

## 의존성과 검증 범위

Linux에 필요한 Bubblewrap 준비 절차는 위의 [직접 빌드하기](#직접-빌드하기)를 따른다. 고정된 helper 소스·체크섬·라이선스는 `scripts/bubblewrap.Dockerfile`과 `scripts/notices/`에서 관리한다.

직접 의존성은 `go.mod`에 정확한 버전으로 고정한다. 유지되는 공개 버전과 라이선스, 도입 목적을 확인했으며, 불필요한 웹·위젯 프레임워크는 사용하지 않는다. 2026-10-13에 직접 의존성과 빌드·테스트에 사용하는 간접 의존성을 최신 안정 버전으로 확인·갱신했다. 자동 업데이트는 아니며 이후 변경도 정확한 버전으로 고정하고 검증한다.

| 모듈 | 버전 | 목적·라이선스 |
| --- | --- | --- |
| `github.com/gdamore/tcell/v3` | 3.6.0 | 터미널 셀·입력·마우스·시뮬레이션, Apache-2.0 |
| `github.com/rivo/uniseg` | 0.4.7 | 결합문자와 CJK 셀 폭, MIT |
| `github.com/pelletier/go-toml/v2` | 2.4.3 | 설정 파싱, MIT |
| `github.com/zalando/go-keyring` | 0.2.8 | Keychain·Secret Service, MIT; 일부 소스 Apache-2.0 |
| `golang.org/x/term` | 0.47.0 | 대화형 TTY 확인, BSD-3-Clause |
| `modernc.org/sqlite` | 1.60.1 | CGO 없는 SQLite, BSD-3-Clause 및 포함 구성요소 라이선스 |
| `mvdan.cc/sh/v3` | 3.14.1 | POSIX Shell AST 파서, BSD-3-Clause; 셸 실행기 대신 syntax 패키지만 사용 |

tcell v3의 `EventQ`, 문자열 키 입력, `Put`/`Get`, 스타일 getter를 직접 사용한다. 가상 터미널 테스트는 v3의 `vt.MockTerm`을 사용하며, 이 테스트 API는 upstream에서 안정성을 보장하지 않으므로 버전 갱신 시 함께 검증한다. 이벤트 전달용 별도 goroutine과 v2 호환 래퍼는 두지 않는다. v3가 추가한 간접 의존성은 `clipperhouse/displaywidth`·`uax29/v2`로 Unicode 셀 폭·문자 경계를 처리하며 MIT 라이선스다.

SQLite 공식 지침에 따라 `modernc.org/libc`는 드라이버 `v1.60.1`이 사용하는 **v1.77.1**과 정확히 맞춘다. 단독 최신 버전으로 올리지 않는다. [드라이버의 libc 호환성 지침](https://pkg.go.dev/modernc.org/sqlite@v1.60.1#hdr-Fragile_modernc_org_libc_dependency)을 따른다. `go list -m -u all`에는 upstream 자체의 개발 도구·테스트 모듈도 나타나므로 V의 실제 빌드·테스트 의존성과 구분한다. 이 모듈들을 최신으로 표시하기 위해 사용하지 않는 직접 의존성을 추가하지 않는다.

SQLite는 Go 변환 런타임을 포함하므로 실행 파일 크기와 컴파일 비용이 있다. 현재 macOS arm64 기본 빌드의 실행 파일은 약 18.4 MiB다. 이번 변경 전 빌드 18,890,370바이트에서 19,330,466바이트로 증가했으며, 셸 파서와 줄 표식·격리 실행 코드가 함께 포함된 값이다. Linux 바이너리는 준비된 helper도 포함한다. CGO 없이 Linux amd64·arm64와 macOS amd64 교차 빌드를 확인했다. 빌드 성공을 실제 OS의 터미널·보안 동작 검증으로 간주하지 않는다. 자격 증명 저장소는 플랫폼 서비스와 시스템 도구에 의존한다.

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

현재 검증은 Go 1.27.2, macOS arm64에서 수행한다. 파일 정책·승인·Plan·HTTP·SQLite·입력·화면·단계별 도구·Ralph 예산·정체·체크포인트·Compaction 임계값과 문맥 연결은 자동 테스트를 사용하며, 실제 모델 서버 대신 `httptest`를 사용한다. 실 TTY 시작·파일 탐색·이어하기·프롬프트 bootstrap·종료 복원도 확인한다. 실제 모델의 요약 품질, Provider별 서버 버전, Linux 네이티브 TTY·Secret Service, 밝은·어두운 터미널별 시인성과 SSH는 별도 환경 검증이 남아 있다. Windows 네이티브 지원은 제공하지 않는다.

## 메모리·할당 점검

화면 기록을 제거할 때 배열 뒷부분의 문자열 참조, 폴더 열을 닫을 때 파일 목록 참조, 200개를 넘은 입력 기록의 앞부분 참조를 명시적으로 지운다. 자동완성 worker는 완료 시에도 context를 취소한다. 취소 후 worker join, HTTP body·idle connection 정리, SQLite 단일 연결, 프로세스 그룹 종료 경로를 점검했다. 전체 시스템의 누수가 없다는 보증이나 장시간 실서버 부하 검증은 아니다.

`go test ./internal/tui/viewport -run '^$' -bench BenchmarkHistoryLines -benchmem -benchtime=200ms`로 기록 200개(본문 약 69KiB), 폭 80칸을 재배치했다. Go 1.27.2, macOS arm64, M1 Max에서 문자열 연결을 `strings.Builder`로 바꾼 전후 결과는 다음과 같다. 이 수치는 해당 합성 작업의 측정값이며 실제 LLM 처리량은 아니다.

| 항목 | 변경 전 | 변경 후 |
| --- | ---: | ---: |
| 시간/회 | 8.06ms | 6.40ms |
| 할당 바이트/회 | 3,197,211 | 211,360 |
| 할당 횟수/회 | 70,011 | 1,611 |

남은 최적화 후보는 스트림 이벤트마다 전체 보관 기록을 다시 줄바꿈하는 비용과 누적 응답을 다시 검사·마스킹하는 비용이다. 캐시·렌더링 병합은 실제 긴 대화 프로파일링을 거쳐 검토한다. 24시간 연속 실행의 heap/RSS 추적과 모든 실제 Provider 조합 검증은 아직 수행하지 않았다.
