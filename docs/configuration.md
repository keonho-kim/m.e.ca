# 설정과 대화 보존

[← README](../README.md) · [모델 연결](model-connections.md) · [서버 API](server.md)

전역 설정은 ~/.meca/config.toml, 대화와 세션별 선택은 프로젝트별 세션 DB, API Key 암호문은 ~/.meca/credentials.db, 암호화 키는 OS 자격 증명 저장소에 보관한다.

tui·full은 이 설정·대화·스킬·인증정보를 공유한다. 종류는 `meca --help`에 표시하며 config.toml에서 바꾸지 않는다. 서버·WebUI와 그 인증·접수 파일은 full에서 해당 기능을 사용할 때 준비한다. `meca upgrade --tui` 또는 `meca upgrade --full`로 종류를 바꾸고, 옵션 없는 `meca upgrade`는 현재 종류를 유지한다.

| 저장 위치 | 현재 책임 |
| --- | --- |
| `config.toml` | 언어·승인 모드·모델 기본값·역할별 추론·실행/압축 예산 |
| `prompts/single/`, `prompts/team/` | 실행 프로필별 지침 |
| `sessions/` | 프로젝트별 사용자 대화와 팀 개인 DB |
| `credentials.db` | 암호화한 모델 API Key |
| `server.token` | 기본 서버 인증 토큰. 사용자 지정 경로는 `--token-file` |
| `server-requests/<project-scope>/requests.db` | 프로젝트별 native 작업 접수 digest·세션·종료 상태 |
| `logs/` | 로컬 진단 로그 |
| `skills/<name>/SKILL.md` | 개인 스킬. 프로젝트·공유 경로와의 우선순위는 [스킬 문서](skills.md) |

이 경로는 `~/.meca` 기준이며 기능을 사용할 때 필요한 파일을 만든다. 서버의 listen·protocol·TLS·Host·Origin은 config.toml에 저장하지 않고 시작 옵션으로 전달한다. 서버 토큰과 모델 API Key는 서로 다른 인증정보다.

일반 시작과 설치의 Bootstrap은 `skills/`에 `create-skill`·`ui-design`·`backend-design` 기본 스킬을 생성한다. 레이아웃·테마 등 중첩된 참고 파일·라이선스·고지도 함께 배포한다. 누락한 파일만 생성하고 이미 있는 스킬 파일은 재설치 때도 덮어쓰지 않는다. [기본 스킬과 초기화](skills.md#기본-스킬과-초기화)를 참고한다.

언어·승인 모드·기본 모델을 저장할 때는 프로세스 공통 잠금 안에서 최신 설정의 해당 항목만 바꾼다. 저장 상한과 할당 기록은 내부에서 관리하며, 한도에 도달해도 기존 대화·자격 증명·백업을 자동 삭제하지 않는다. [서버 한도](server.md)를 따른다.

로그 레벨은 시작 환경변수 `MECA_LOG_LEVEL`로 선택한다. TUI 파일 보관과 서버·CLI의 JSON stderr는 [이벤트와 운영 로그](diagnostics.md)를 따른다. 스킬은 기존 세션 기록에 보존한다. 지원하지 않는 저장 형식은 원본을 보존하고 열기를 거부한다. 자동 변환·삭제·재생성하지 않으며 서버 접수 DB를 제거해 이미 실행된 요청을 다시 접수하는 방식으로 해결하지 않는다.

## 화면 언어

최상위 `language`의 기본값은 `"auto"`다. `"ko"`·`"en"`은 시스템 로캘보다 우선하는 한국어·영어 설정이며 다른 값은 거절한다. !language에서 ←→로 선택한 뒤 Enter의 저장 확인에서 y를 누르면 전역 파일에 저장하고 현재 TUI에 즉시 적용한다. 저장 전에는 적용 중인 설정을 유지한다. 직접 파일을 수정하면 재시작 후 적용한다. 새 대화·이전 대화 선택 화면에도 이 값을 사용하며 모델 응답 언어는 사용자 요청을 따른다. 자동 선택 기준과 메뉴 조작은 [사용 가이드](usage.md#화면-언어)를 따른다.

## 모델 연결 재사용

!model의 저장 확인에서 y를 누르면 현재 대화의 선택을 저장한다. 저장 범위가 모든 대화의 기본값이면 config.toml의 [model]에도 provider·endpoint·name(서빙 ID)·credential_ref와 선택적 thinking·reasoning_effort를 기록한다. 추론 강도는 모델 연결에 묶인 선택이며 새 대화 연결 시 서버 메타데이터로 재검증한다. API Key 본문은 별도의 credentials.db에서만 암호화해 보관하고 config.toml·대화 DB에는 포함하지 않는다.

WebUI의 설정 → 모델은 대화를 열기 전에도 전역 `[model]`을 조회하고 저장한다. 저장된 세션의 선택이 우선이며 서버는 개별 모델을 지정하지 않은 대화의 다음 요청에 현재 전역 기본값을 적용한다. 복원 시 모델 식별·기능을 다시 확인한다. 모델 저장은 TUI 입력 루프 밖에서 수행한다. 취소했더라도 DB에서 저장 완료를 확인하면 해당 연결을 적용한다. 저장 오류는 실제 DB 값을 다시 읽어 판정하며, 확인할 수 없으면 현재 연결을 해제하고 재연결을 요구한다. 확실히 거부된 저장은 기존 연결을 유지한다. 전역 기본 연결 저장에 실패해도 이미 저장된 현재 세션 연결은 유지하고 오류를 알린다. Thinking은 세션 선택 또는 전역 기본값을 따르며 생략하면 서버 설정을 사용한다. 원본 식별은 서버가 제공한 정보를 표시한다. [모델 연결](model-connections.md)을 참고한다.

### API Key 암호화와 복구

저장한 API Key는 암호화해 보관하며 설정·대화 DB에 평문으로 기록하지 않는다. macOS Keychain 또는 Linux Secret Service가 필요하다. OS 저장소를 사용할 수 없으면 저장·조회 오류를 반환하고 기존 기록을 유지한다.

백업은 MECA를 종료한 뒤 `credentials.db`와 OS 자격 증명 저장소를 함께 보존한다. DB만 새 장치로 복사하면 사용할 수 없다. OS 저장소와 백업을 모두 잃으면 공급자에서 API Key를 다시 발급받아야 한다.

`meca uninstall`은 저장된 인증정보도 삭제한다. 정리가 실패하면 오류 안내에 따라 `meca uninstall`을 다시 실행한다. 정리가 끝날 때까지 일반 시작·설치는 차단된다.

### Linux 자격 증명 전송

공식 릴리스와 `install.sh`는 `godbus_no_tcp` 태그를 적용한다. Linux Secret Service의 DBus 연결에서 `tcp:`와 `nonce-tcp:` 전송 구현 및 nonce 파일 전송을 빌드에서 제외하고 `unix:` 연결을 유지한다. macOS Keychain과 모델 API·OpenRouter·셸 네트워크 동작은 유지한다.

보장 범위는 공식 MECA 바이너리의 DBus 의존성이 TCP 전송을 제공하지 않는다는 것이다. OS 전체의 외부 통신을 차단하거나 연결한 Unix 소켓의 서비스 신뢰성까지 보장하지 않는다. 태그 없는 일반 `go build`도 가능하므로 [수동 빌드](development.md#소스-빌드)에는 명시적 옵션이 필요하다. Linux에서 API Key를 저장하려면 접근 가능한 Unix 세션 버스와 Secret Service가 필요하며, 저장소를 사용할 수 없으면 오류를 반환한다.

Linux에서는 사용 중인 데스크톱 세션의 Unix DBus와 Secret Service에 접근할 수 있어야 한다. MECA가 세션 버스나 키 저장소를 자동 설치·기동하지 않는다.

## 메인 에이전트와 보조 역할의 추론 설정

메인 에이전트는 최상위 **[agent]**, 요약·연결 진단은 [compaction]·[diagnostics]에서 설정한다. 역할별 공통 설정을 Provider 어댑터가 각 API 형식으로 변환한다. Bootstrap은 기본값을 기록하며 생략한 필드는 현재 기본값을 사용한다. 명시한 값은 자동 덮어쓰지 않으며 직접 수정은 재시작 후 적용한다.

```toml
language = "auto"
permission = "default"
context_length = 262144
# thinking = true 또는 false를 지정하면 전역 기본값을 덮어쓴다. 생략 시 서버 기본값.

[agent]
thinking = "inherit"
temperature = 1.0
max_output_tokens = 0

[compaction]
thinking = "off_if_supported"
temperature = 0.2
max_output_tokens = 8192
reserve_tokens = 16384
keep_recent_tokens = 20000
max_summary_bytes = 16777216
max_input_bytes = 16777216

[diagnostics]
thinking = "inherit"
temperature = 0.2
max_output_tokens = 12800
```

| 역할 | 용도 | temperature | 기본 출력 상한 |
| --- | --- | ---: | ---: |
| agent | 직접 도구 실행·대화·최종 응답 | 1.0 | context의 75% |
| compaction | 단일 호출 대화 요약 | 0.2 | 8192 |
| diagnostics | 모델 연결·Thinking·effort 진단 | 0.2 | 12800 |

temperature는 유한한 0~2이며 지원되는 Provider에는 명시한 0도 전송한다. OpenAI·Claude는 모델마다 sampling 지원이 달라 이 값을 전송하지 않고 공급자 기본값을 사용한다. Gemini는 generationConfig.temperature를 사용한다. max_output_tokens는 0~1073741824다. agent·compaction·diagnostics의 0은 유효 context length의 75%를 내림 계산하는 자동 모드이며 양수는 수동 상한이다. 서버가 공개한 context·출력 한도와 남은 입력 공간도 적용한다. 서버가 제시한 한도와 설정한 상한을 함께 지키며 모델 출력 한도 초과를 명시적으로 보고한다.

메인 에이전트는 사용자의 Thinking/effort를 계승한다. Compaction은 서버 메타데이터 또는 Provider API에 OFF 요청 규약이 있는 연결에서 Thinking을 끄며, 제어 규약이 없으면 사용자 선택을 유지한다. 실제 OFF 동작은 명시적 연결 진단으로 확인한다. 연결 진단은 사용자가 !model의 연결 진단 또는 meca --diagnose를 선택했을 때 ON/OFF·effort를 대조한다. 출력 상한과 전체 실행 예산은 별도다. Ollama에는 options.temperature, 호환 Chat·LM Studio Responses에는 temperature를 전송한다.

## 자동 승인과 실행 예산

최상위 `permission`은 `"default"` 또는 `"all_mighty"`다. 기본 `default`는 일반 작업을 자동 실행하고 승인 대상 작업만 사용자에게 묻는다. `all_mighty`는 외부 전송을 제외한 승인 대상 작업을 사용자에게 묻지 않고 자동 승인한다. !permission에서 ←→로 선택하고 Enter 뒤 y/N/Esc로 저장·취소·계속 편집한다. 저장한 값은 이후 요청과 새 대화·재개·비대화형 실행에 동일하게 적용한다. 실행 중인 요청은 시작 시 선택한 정책을 유지하며 작업 중에는 모드를 변경하지 않는다. 모드별 선택을 세션 DB에 중복 저장하지 않는다. 승인 기록은 직접 사용자 승인과 All Mighty 결정을 구분한다.

외부 전송은 모든 모드에서 직접 승인이 필요하다. 명시적 업로드·원격 실행·전송 인자가 포함된 요청은 All Mighty에서도 매번 직접 승인받고, 비대화형 실행에서는 수행하지 않는다. 목적지 허용 목록을 별도 설정하지 않는다. 일반 다운로드·개발 명령은 호스트 환경을 사용하며 직접 지정한 loopback HTTP 테스트는 전송 승인 예외다. 검사 대상과 보장 범위는 [외부 전송 승인](execution-boundary.md#외부-전송-승인)을 따른다.

config나 메뉴에 Scope 선택은 없다. 셸과 자식 프로그램은 호스트 사용자 권한으로 실행한다. 두 승인 모드 모두 보호 경로 차단·자격 증명 보호·인자·파일 버전 검사와 실제 실행 실패 반환을 유지한다. 전용 파일 도구의 대상·버전 검사와 셸의 OS 권한 범위는 [실행 경계](execution-boundary.md)를 따른다.

```toml
[loop]
max_model_calls = 1024
max_seconds = 14400
max_model_seconds = 600
max_context_bytes = 268435456
max_argument_bytes = 268435456
```

도구 호출 수는 실행 기록에 남기며 별도 횟수 상한을 두지 않는다. 모델은 완료했거나 해결할 다음 행동이 없으면 미완료 결과·구체적인 장애·필요한 사용자 입력을 최종 답변으로 설명하고 종료한다. 모델 요청과 전체 시간 예산은 별도로 유지하며 재시도·압축도 같은 요청의 예산을 사용한다. 팀 모드에서는 모든 참여자가 같은 모델 요청 수·마감 시각을 공유한다.

max_model_seconds는 에이전트·압축 호출에서 데이터가 수신되지 않는 연속 대기 시간이다. 수신할 때마다 다시 세며 응답 전체의 시간 상한이나 셸 명령별 제한이 아니다. max_seconds는 전체 작업 예산이다. 메인·압축 응답은 다음 작업 경계에서 전체 시간 예산을 확인하며, 도구 실행에는 전체 작업의 마감 시각을 적용한다. 시간·모델 요청 예산 또는 오류로 중단하면 실제 수행 결과와 이유를 기록한다.

## 프롬프트와 Compaction

일반 시작은 누락된 기본 프롬프트만 생성한다. 각 실행 프로필의 지침은 다음 파일에서 편집할 수 있다.

```text
~/.meca/prompts/
  single/
    system.md
    development.md
    compaction.md
  team/
    system.md
    development.md
    compaction.md
```

`system.md`는 작업·응답 원칙, `development.md`는 개발 절차, `compaction.md`는 문맥 요약을 안내한다. 각 프로필은 독립된 파일을 사용한다. 프로젝트별 지침은 프로젝트 루트의 `AGENTS.md`에 둔다.

실행 중 수정한 프롬프트는 다음 요청에서 다시 읽는다. **install·upgrade는 사용자 수정 여부와 관계없이 배포 기본 프롬프트로 교체**하므로 필요한 수정본은 별도로 보관한다. 일반 실행은 수정본을 유지하고 기본 스킬의 사용자 수정본도 덮어쓰지 않는다.

Compaction은 현재 문맥 점유와 다음 응답에 필요한 여유를 기준으로 수행한다. `reserve_tokens`는 응답 여유, `keep_recent_tokens`는 최근 원문 보존, `max_input_bytes`·`max_summary_bytes`는 압축 입력·요약의 바이트 예산이다. 모델의 실제 context·출력 한도도 함께 적용한다.

압축은 최신 요청·실제 결과·도구 호출과 응답의 관계를 보존하며 원본 대화 DB를 삭제하지 않는다. 필수 내용이 예산에 들어가지 않거나 압축이 실패하면 원본을 유지하고 오류를 반환한다. 문맥 판단용 추정값과 화면에 표시하는 실제 토큰 사용량은 구분한다.

## 세션 저장과 복원

사용자 대화는 `~/.meca/sessions/<project-key>/<session-id>.db`에 저장한다. 팀 개인 기록은 해당 프로젝트의 `.team/<사용자 세션 ID>/` 아래에 보관하므로 백업에 이 하위 폴더도 포함한다.

대화·Thinking·도구 결과·사용량과 재개에 필요한 상태를 저장한다. 큰 원문은 압축·참조할 수 있으며 화면용 이력과 전체 원문 내보내기는 읽는 범위가 다르다. 파일을 직접 수정하거나 일부 DB만 분리해서 옮기지 않는다.

안전한 백업은 MECA를 종료한 뒤 관련 세션 폴더 전체를 복사하는 방식이다. 자격 증명은 [별도의 백업 조건](#api-key-암호화와-복구)을 따른다. 대화 삭제는 해당 대화와 소유한 팀 기록을 제거하며 프로젝트 작업 파일은 삭제하지 않는다.

중단된 요청은 저장 상태와 실제 부분 효과를 확인해 재개한다. 부작용을 자동 재실행하거나 저장 불확실 상태를 정상 완료로 바꾸지 않는다. 지원하지 않는 저장 형식은 원본을 보존하고 열기를 거절하며 자동 변환·삭제·초기화를 하지 않는다.

## 본문 한도

TOML 파서는 알 수 없는 키·잘못된 타입을 거절하고 각 설정 소유자가 값의 범위를 검사한다. 프롬프트 편집은 일반 실행 중 유지하며 install·upgrade를 실행하면 배포 기본본으로 갱신한다. 모델 연결과 나머지 설정·대화는 프롬프트 갱신으로 변경하지 않는다.

프로젝트 지침·프롬프트·압축 입력·요약은 각각 16MiB다. 모델 응답·실행 문맥·인자는 기본 256MiB, 중첩 JSON 저장 이벤트는 512MiB 수용 한도를 둔다. 이는 미리 할당하는 크기나 모델에 매번 보내는 양이 아니다. 경로·ID·열거값·배열 개수와 화면 발췌 예산은 별도다. 실제 모델 context·출력 한도와 바이트 수용 상한을 혼동하지 않는다.

모델 응답은 답변·Thinking·native 상태·완료 호출을 합산하고 추가 전에 검사한다. OpenAI·Claude·Gemini 프레임과 continuation에도 256MiB를 적용한다. 이미지 Base64만으로 요청 상한을 넘으면 변환 전에 거부한다. 서버의 native 제어 17MiB·입력 텍스트 16MiB와 이미지당 64MiB·64M픽셀은 이 모델/저장 상한과 별개다. [서버 계약](server.md#이벤트재접속수명)과 [이미지 입력](images.md)을 따른다.

## 이미지 입력 상태

이미지 지원 여부는 config.toml의 수동 모델 프로파일로 지정하지 않는다. 연결 때 메타데이터 또는 작은 이미지 요청으로 확인하고 세션의 모델 선택에 근거를 보관한다. 재연결은 저장된 판정만 믿지 않고 다시 확인한다. 상태·지원 형식·자원 한도는 [이미지 입력](images.md)을 따른다.
