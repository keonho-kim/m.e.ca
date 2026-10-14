# 개발 가이드

[← README](../README.md)

패키지 책임, 의존성·라이선스와 검증 절차를 정리합니다. 성능 수치는 아래에 기록된 환경과 변경 시점의 측정값입니다.

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
  commands/                  일곱 내부 명령 파서와 후보 카탈로그
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
    diagnostic/              생성기를 통한 일반 응답·Thinking 진단
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
          linux/             내부 실행 프로토콜·마운트·Landlock·seccomp
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

저장 기능은 SQLite와 분리한다. `storage/session`은 세션 값과 식별 규칙, `storage/history`는 순서 있는 기록과 압축 자료 변환, `storage/recovery`는 중단 복구를 소유한다. SQLite는 이 계약을 구현하는 저장 어댑터로서 DB 파일·SQL·트랜잭션·잠금만 담당한다. TUI와 작업 유스케이스는 필요한 저장 인터페이스에 의존하며, `cmd/v`에서 SQLite를 연결한다. 현재 DB 스키마와 JSON 기록 형식은 이 책임 분리로 변경하지 않는다.

`tui/session`은 활성 화면 상태와 worker의 시작·취소·합류를 소유한다. 작업 준비는 `agent/session`, 정책·Thinking 저장과 실패 보상은 `commands/settings`, 연결 저장과 자격 증명 보상은 `provider/setup`이 담당한다. 모델 선택 검증은 `provider/selection`, 메뉴 표시값은 `tui/modelmenu`가 맡는다. 화면 입력·승인·출력 상태는 이벤트 루프에서만 변경한다.

Provider는 전송(`transport`), 형식 변환(`protocol`), 메타데이터 조회(`discovery`), 실제 생성 진단(`diagnostic`), 생성 요청 조율(`httpclient`)로 나눈다. HTTP 본문·재시도·idle 감시의 수명은 전송 계층이, 연결 준비와 생성 클라이언트 이전은 `setup`이 소유한다. 모델별 특화는 `profile`에 유지한다. 모델 목록·출처·제어 범위는 [모델 프로파일 문서](model-profiles.md)에 기록한다.

`agent/run`은 PFC → self-grounding → 후보 준비 → Conclave → 사용자 승인 → 실행 순서와 공유 예산을 유지한다. 컨텍스트 조립은 `agent/context`, 파일 상태 순환 감지는 `agent/progress`, 실행 사실은 `agent/tool`, 승인 단계 식별은 `agent/plan`이 소유한다. Compaction은 이 사실을 소비하며 실행 기록의 타입을 대신 소유하지 않는다. `tools/local`의 Scope·승인·버전 재검사를 통과한 작업만 `file`, `search`, `process` 어댑터로 전달한다.

각 패키지는 한 기능을, 하위 패키지는 독립적인 하위 기능을, 파일은 그 안의 한 책임을 맡는다. 작은 함수와 명시적인 상태 전이를 우선하며, 순환 의존·이전 위치의 전달 패키지·호환 별칭·사용되지 않는 추상화는 두지 않는다. 단일 Go 모듈과 `cmd/v` 실행 파일을 유지한다.

## 의존성과 검증 범위

빌드는 README의 [직접 빌드하기](../README.md#직접-빌드하기)를 따른다. Docker·C helper 빌드·내장 실행 파일은 필요하지 않다. Linux 셸 격리는 기존 x/sys를 통해 커널 API를 직접 호출한다. Landlock ABI 3 이상, seccomp와 user/mount/PID/network/IPC/UTS 네임스페이스 등이 필요하다. `V_SANDBOX_REQUIRED=1 go test ./internal/tools/local/process/...`는 격리를 실제로 실행하며 커널 기능이 없으면 실패한다. 일반 테스트는 지원하지 않는 호스트에서 사유와 함께 네이티브 검증을 생략한다. 네임스페이스가 차단된 호스트에서는 `V_SANDBOX_DENIED=1`로 명령이 실행되지 않고 scope_unavailable이 반환되는지 검증할 수 있다.

직접 의존성은 `go.mod`에 정확한 버전으로 고정한다. 유지되는 공개 버전과 라이선스, 도입 목적을 확인했으며, 불필요한 웹·위젯 프레임워크는 사용하지 않는다. 2026-10-13에 직접 의존성과 빌드·테스트에 사용하는 간접 의존성을 최신 안정 버전으로 확인·갱신했다. 자동 업데이트는 아니며 이후 변경도 정확한 버전으로 고정하고 검증한다.

| 모듈 | 버전 | 목적·라이선스 |
| --- | --- | --- |
| `github.com/gdamore/tcell/v3` | 3.6.0 | 터미널 셀·입력·마우스·시뮬레이션, Apache-2.0 |
| `github.com/rivo/uniseg` | 0.4.7 | 결합문자와 CJK 셀 폭, MIT |
| `github.com/pelletier/go-toml/v2` | 2.4.3 | 설정 파싱, MIT |
| `github.com/zalando/go-keyring` | 0.2.8 | Keychain·Secret Service, MIT; 일부 소스 Apache-2.0 |
| `golang.org/x/term` | 0.47.0 | 대화형 TTY 확인, BSD-3-Clause |
| `golang.org/x/sys` | 0.49.0 | Linux 커널 격리와 시스템 호출, BSD-3-Clause |
| `modernc.org/sqlite` | 1.60.1 | CGO 없는 SQLite, BSD-3-Clause 및 포함 구성요소 라이선스 |
| `mvdan.cc/sh/v3` | 3.14.1 | POSIX Shell AST 파서, BSD-3-Clause; 셸 실행기 대신 syntax 패키지만 사용 |

tcell v3의 `EventQ`, 문자열 키 입력, `Put`/`Get`, 스타일 getter를 직접 사용한다. 가상 터미널 테스트는 v3의 `vt.MockTerm`을 사용하며, 이 테스트 API는 upstream에서 안정성을 보장하지 않으므로 버전 갱신 시 함께 검증한다. 이벤트 전달용 별도 goroutine과 v2 호환 래퍼는 두지 않는다. v3가 추가한 간접 의존성은 `clipperhouse/displaywidth`·`uax29/v2`로 Unicode 셀 폭·문자 경계를 처리하며 MIT 라이선스다.

SQLite 공식 지침에 따라 `modernc.org/libc`는 드라이버 `v1.60.1`이 사용하는 **v1.77.1**과 정확히 맞춘다. 단독 최신 버전으로 올리지 않는다. [드라이버의 libc 호환성 지침](https://pkg.go.dev/modernc.org/sqlite@v1.60.1#hdr-Fragile_modernc_org_libc_dependency)을 따른다. `go list -m -u all`에는 upstream 자체의 개발 도구·테스트 모듈도 나타나므로 V의 실제 빌드·테스트 의존성과 구분한다. 이 모듈들을 최신으로 표시하기 위해 사용하지 않는 직접 의존성을 추가하지 않는다.

SQLite는 Go 변환 런타임을 포함하므로 실행 파일 크기와 컴파일 비용이 있다. 현재 macOS arm64 기본 빌드의 실행 파일은 약 18.4 MiB다. 이번 변경 전 빌드 18,890,370바이트에서 19,330,466바이트로 증가했으며, 셸 파서와 줄 표식·격리 실행 코드가 함께 포함된 값이다. 이 수치는 내장 helper 제거 전 측정이다. 현재 Linux 바이너리는 내부 커널 실행기를 사용하며 외부 helper를 포함하지 않는다. CGO 없이 Linux amd64·arm64와 macOS amd64 교차 빌드를 확인했다. 빌드 성공을 실제 OS의 터미널·보안 동작 검증으로 간주하지 않는다. 자격 증명 저장소는 플랫폼 서비스와 시스템 도구에 의존한다.

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

### Linux 내부 실행기 검증

2026-10-14에 Linux 7.0.14 arm64의 Alpine 컨테이너에서 Bubblewrap 설치 없이 파일 격리, 외부 경로·심볼릭 링크·V 상태·메타데이터 접근 차단을 확인했다. 일반 사용자 UID 65534에서도 실행했다. TCP·Unix 소켓과 격리 변경 차단, 취소 시 분리된 자식 프로세스 정리, 실행 파일 경로가 교체된 뒤에도 기존 이미지로 재실행되는 동작을 검증했다. Landlock 호출 자체를 거부한 환경에서도 명령 실행 없이 실패를 반환했다. 호스트 네임스페이스 생성이 차단된 컨테이너에서는 프로젝트 명령이 실행되지 않고 실패를 반환하는 경로를 확인했다. 컨테이너는 개발 검증 환경이며 제품의 빌드·실행 의존성이 아니다. 배포판별 AppArmor·커널 설정과 Linux amd64 실기 검증은 남아 있다.

내부 실행기의 경계는 [Landlock](https://docs.kernel.org/userspace-api/landlock.html)과 [seccomp](https://docs.kernel.org/userspace-api/seccomp_filter.html)의 커널 계약을 따른다. Landlock만으로 파일 존재 확인과 프로젝트 내부 V 상태 숨김을 대체하지 않으므로, 별도 마운트 뷰와 함께 적용한다. 설치 의존성을 없애는 대신 이 커널 경계의 구현과 회귀 검증을 저장소에서 소유한다.

사용자 설치 uv 회귀 검증은 `V_TEST_UV_BINARY=/path/to/uv go test ./internal/tools/local/process -run TestUVPackageProject -v`로 실행한다. 관리 Python을 사용하는 환경은 `V_TEST_UV_PYTHON_ROOT`에 한 설치 트리를 지정하면 임시 홈에 복사하여 검증한다. 네트워크 없이 실제 uv 0.13.0의 `uv init --package --offline --vcs none example`을 Linux arm64(시스템 Python 3.12.15)와 macOS arm64(관리 Python 3.13.9)에서 확인했다. WSL 실기와 Linux amd64 런타임 검증은 별도다.
