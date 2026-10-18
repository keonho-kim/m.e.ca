# 설정과 대화 보존

[← README](../README.md)

전역 설정은 ~/.v/config.toml, 대화와 세션별 선택은 프로젝트별 세션 DB, API Key는 OS 자격 증명 저장소에 보관한다.

## 모델 연결 재사용

!model의 연결 진단과 선택이 성공하면 세션 DB에 선택을 저장하고 마지막으로 직접 선택한 연결을 config.toml의 [model]에 기록한다. 저장 값은 provider·endpoint·name(서빙 ID)·credential_ref다. API Key 본문은 macOS Keychain 또는 Linux Secret Service에만 저장한다.

기존 세션의 선택이 우선이며 선택이 없는 새 세션은 전역 [model]을 사용한다. 복원 시 모델 식별·기능을 다시 확인한다. 전역 기본 연결 저장에 실패해도 이미 저장된 현재 세션 연결은 유지하고 오류를 알린다. Thinking은 세션 선택 또는 전역 기본값을 따르고, 별칭 식별은 정확한 provider/endpoint/model 바인딩을 사용한다. [모델 프로파일](model-profiles.md)을 참고한다.

## 메인 에이전트와 보조 역할의 추론 설정

메인 에이전트는 최상위 **[agent]**, 요약과 연결 진단은 [compaction]·[diagnotics]에서 설정한다. Provider별 기본값을 따로 선택하지 않는다. Bootstrap은 기본값을 기록하며 생략한 필드는 현재 기본값을 사용한다. 명시한 값은 자동 덮어쓰지 않으며 직접 수정은 재시작 후 적용한다.

```toml
context_length = 262144
thinking = false

[agent]
thinking = "inherit"
temperature = 1.0
max_output_tokens = 0

[compaction]
thinking = "off_if_supported"
temperature = 0.2
max_output_tokens = 8192
threshold_tokens = 200000
keep_recent_bytes = 16384
max_summary_bytes = 16777216
max_input_bytes = 16777216

[diagnotics]
thinking = "inherit"
temperature = 0.2
max_output_tokens = 12800
```

| 역할 | 용도 | temperature | 기본 출력 상한 |
| --- | --- | ---: | ---: |
| agent | 직접 도구 실행·대화·최종 응답 | 1.0 | context의 75% |
| compaction | 단일 호출 대화 요약 | 0.2 | 8192 |
| diagnotics | 모델 연결·Thinking·effort 진단 | 0.2 | 12800 |

temperature는 유한한 0~2이며 명시한 0도 전송한다. max_output_tokens는 0~1073741824다. 0은 유효 context length의 75%를 내림 계산하는 자동 모드이며 양수는 수동 상한이다. 서버가 공개한 context·출력 한도와 남은 입력 공간도 적용한다. 서버가 제시한 한도와 설정한 상한을 함께 지키며 모델 출력 한도 초과를 명시적으로 보고한다.

메인 에이전트는 사용자의 Thinking/effort를 계승한다. Compaction은 OFF가 검증된 조합에서만 Thinking을 끄며, 지원이 확인되지 않으면 사용자 선택을 유지한다. 연결 진단은 기능 확인을 위해 ON/OFF·effort를 대조한다. 출력 상한과 전체 실행 예산은 별도다. Ollama에는 options.temperature, 호환 Chat·LM Studio Responses에는 temperature를 전송한다.

## Permission과 실행 예산

config나 메뉴에 Scope 선택은 없다. 셸과 전용 파일 도구 모두 [공통 파일 경계](execution-boundary.md)를 적용한다. 기본 Permission은 allow-all이며 개별 승인 질문을 생략한다. OS 권한 상승이나 파일 버전 검사 우회를 의미하지 않는다. !permission으로 세션 또는 전역 승인 정책을 바꾼다.

```toml
[defaults]
permission = "allow-all"

[loop]
max_iterations = 128
max_model_calls = 1024
max_seconds = 14400
max_model_seconds = 600
max_context_bytes = 268435456
max_argument_bytes = 268435456
stall_iterations = 3
```

max_iterations는 native 호출 시도 수이고 재시도·압축은 같은 요청의 모델·시간 예산을 사용한다. 수신 중인 모델 응답은 전체 시간이나 무수신 제한 때문에 중간에 끊지 않고 다음 작업 전에 전체 시간을 확인한다. 시간·호출 예산 또는 정체로 중단하면 실제 수행 결과와 이유를 기록한다.

## 추가 쓰기 경로

프로젝트와 V 소유 임시 공간 이외의 쓰기는 ~/.v/config.toml에서 기존 디렉터리를 명시한다. 예를 들어 패키지 관리자가 사용하는 외부 캐시 경로가 필요할 때 해당 경로만 추가한다.

```toml
[execution]
writable_paths = []
```

경로는 절대 경로이며 시작 시 실제 디렉터리와 identity를 고정한다. 수정 후 재시작한다. 루트·홈 전체·시스템 경로·V 상태·보호한 비밀 경로와 그 상위 경로는 예외로 허용하지 않는다. 일반 캐시 하위 디렉터리는 직접 지정할 수 있다. 모델 도구·프로젝트 문서로 이 경계를 확대하지 않는다. 동일한 경계는 allow-all에서도 유지한다.

HOME·PATH와 기존 개발 설정을 재사용한다. 언어별 기본 캐시 경로를 자동으로 허용하거나 XDG·npm·Go·uv 설정을 덮어쓰지 않는다. 부모가 전달한 임의 환경변수의 비밀 여부까지 자동 식별하는 계약은 아니다. 자세한 보호 범위와 플랫폼 제약은 [실행 경계](execution-boundary.md)를 따른다.

## 프롬프트와 Compaction

Bootstrap은 다음 파일을 누락한 경우에만 생성한다. 디렉터리는 0700, 파일은 0600이다. 사용자 수정본은 보존하고 다음 요청에서 다시 읽는다.

```text
~/.v/prompts/
  system.md       직접 작업·응답·간결한 의사결정
  development.md  변경 전 검토·책임·배치·개발 원칙
  compaction.md   항목 중심 계속하기 요약
```

메인 에이전트는 system.md·development.md와 선택적인 프로젝트 루트 AGENTS.md를 한 요청의 스냅샷으로 받는다. 프로젝트 지침이 없거나 비어 있으면 빈 블록을 추가하지 않는다. AGENTS.md와 각 프롬프트의 본문 수용 한도는 16MiB다. 지침은 영어로 작성하며 최신 사용자 요청의 언어로 답하고 요약하도록 안내한다. 사용자·프로젝트 원문은 번역하지 않는다.

기본 개발 원칙은 수정할 소스·호출자·계약·테스트를 먼저 검토하고 책임·배치·가장 단순한 설계를 판단한 뒤 구현·검증하도록 한다. 대상 언어의 native 기능·표준 라이브러리·프로젝트 관례를 우선한다. 프롬프트 수정과 관계없이 Permission·Scope·버전 검사는 실행기가 소유한다.

추론 인자와 압축 발동·용량 설정은 위 예시처럼 하나의 [compaction] 섹션에 함께 기록한다.

발동량은 마지막 성공한 압축 이후 메인 에이전트의 누적 INPUT+OUTPUT이다. 같은 문맥을 재전송한 입력도 포함한다. 각 시도의 마지막 usage를 한 번 확정하고 재시도·취소의 수신량도 보존한다. 압축 자체와 연결 진단은 발동량에서 제외한다.

usage가 없는 호출은 요청 메시지·도구 정의와 수신 응답·Thinking·인자의 UTF-8 바이트 합을 3으로 나눠 올림한다. 200000은 약 600000바이트(586KiB)다. 응답을 전혀 받지 못한 실패는 더하지 않는다. 이 근삿값은 발동 계산에만 쓰며 UI INPUT/OUTPUT·TPS에는 실제 수신값만 사용한다.

압축은 다음 결정의 안전한 경계에서 수행한다. 현재 요청·이전 메모·실제 결과를 우선해 하나의 입력으로 한 번 요약한다. 기본 출력은 8192토큰이다. 최신 요청은 압축 뒤에도 별도로 제공한다. 짧은 항목 중심 형식·목표 분량은 지침이며 형식에 따른 통과/실패 판정은 없다.

keep_recent_bytes는 최근 원문 선택 목표다. native 호출·결과 묶음과 최신 수정 실패 복구에 필요한 조회는 쪼개지 않는다. 최근 꼬리가 충분하면 큰 과거 묶음은 요약 prefix로 보낸다. 0은 최근 원문 보존을 끈다. 원본 기록은 DB에 남고 요약할 prefix가 없으면 기존 메모를 유지한다. 명시적인 context 한도 오류에는 안전한 압축 후 모델 요청만 한 번 재시도하며 도구는 재생하지 않는다.

## 세션 저장과 복원

세션 DB는 ~/.v/sessions/<project-key>/<session-id>.db다. 정규화된 프로젝트 경로의 basename과 SHA-256 앞 32자리로 프로젝트 폴더를 만들고 해당 디렉터리의 DB 파일명에서 ID를 찾는다. 각 DB는 정확히 한 세션을 소유하며 별도 ID 레지스트리는 없다.

현재 형식은 **schema 17**이다. 대화·Thinking·도구 결과·실제 명령과 종료 코드·파일 버전·사용량·체크포인트·요약을 하나의 흐름에 저장한다. 원본 실행 기록과 압축 메모의 책임은 분리한다. 현재 파일 관측은 후속 요청에도 이어가며 오래된 명령 결과와 구별한다. 미지원 DB는 원본을 보존하고 오류를 반환한다.

!new는 현재 작업과 DB 자원을 정리한 뒤 새 파일을 만들며 전역 모델 연결을 유지한다. 일반 시작은 현재 프로젝트 대화를, v resume은 모든 프로젝트 대화를 나열한다. 새 대화가 맨 위에서 기본 선택된다. 복원은 중단된 native 기록을 짝지어 보존하며 부작용을 자동 재생하지 않는다.

.db.lock은 열린 핸들에 연결된 프로세스 잠금 파일이다. 파일이 남아 있다는 사실만으로 실행 중인 프로세스가 있다고 판단하지 않는다. !quit·Ctrl+C는 worker·자식 프로세스·클라이언트를 정리한 뒤 잠금과 DB를 닫는다.

## 본문 한도

TOML 파서는 알 수 없는 키·잘못된 타입을 거절하고 각 설정 소유자가 값의 범위를 검사한다. 사용자 수정 프롬프트는 기본값으로 자동 교체하지 않으므로 원하는 지침은 해당 Markdown 파일에서 직접 관리한다.

reason은 32MiB UTF-8, 프로젝트 지침·프롬프트·압축 입력·요약은 각각 16MiB다. 모델 응답·실행 문맥·인자는 기본 256MiB, 중첩 JSON 저장 이벤트는 512MiB 수용 한도를 둔다. 이는 미리 할당하는 크기나 모델에 매번 보내는 양이 아니다. 경로·ID·열거값·배열 개수와 화면 발췌 예산은 별도다. 실제 모델 context·출력 한도와 바이트 수용 상한을 혼동하지 않는다.
