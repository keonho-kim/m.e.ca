# 모델 연결과 추론 설정

[← README](../README.md) · [사용법](usage.md) · [설정](configuration.md)

## 연결과 기본값

!model에서 Provider·주소·API 키·서빙 모델 ID를 선택한다. 실제 요청에는 선택한 ID를 그대로 사용한다. 모델명·출시일·파라미터 수에 따른 등록 목록이나 수동 프로파일 매핑은 사용하지 않는다.

설정 화면은 저장된 Thinking·강도 또는 서버가 공개한 기본값을 표시한다. 확인된 기본값은 구체적인 ON/OFF·강도로 표시하고, 공개되지 않은 값만 DEFAULT로 둔다. DEFAULT는 think·reasoning·reasoning_effort·chat_template_kwargs의 해당 제어 필드 생략을 뜻한다. 역할별 temperature·출력 예산·컨텍스트 예산은 [설정 문서](configuration.md)의 독립된 설정이다.

연결 준비는 메타데이터를 조회하고 클라이언트를 구성한다. 모델 응답 생성은 !model → 연결 진단 또는 meca --diagnose에서 사용자가 요청할 때 수행한다. 메타데이터 조회 실패는 원인을 남기고 일반 연결을 준비하며 사용자 취소는 즉시 전파한다.

## Thinking과 추론 강도

!model은 연결·추론 설정과 적용 범위를 같은 화면에서 편집한다. ↑↓로 행 이동, →로 입력 또는 다음 값 선택, ←로 이전 값 선택을 한다. Enter의 저장 확인에서 y는 저장 후 채팅, n은 취소 후 채팅, Esc는 편집 화면으로 복귀한다. 세션·전역 설정과 인증정보는 y 전까지 저장하지 않는다. 전역 저장은 config.toml의 model 연결과 Thinking·강도 선택을 함께 보관한다.

Thinking effort의 DEFAULT는 강도 필드 생략이다. 서버가 제공한 문자열은 대소문자를 포함해 그대로 전송한다. OFF에서도 선호 강도를 보존하지만 실제 요청에서는 생략한다. 강도를 고를 때 상태가 미지정이면 ON을 명시한다. ON 전용 모델에는 OFF 선택을 제공하지 않는다.

| Provider | 선택지의 근거 | 요청 형식 |
| --- | --- | --- |
| Ollama | /api/show의 thinking.values·default | think boolean 또는 선언된 강도 문자열 |
| LM Studio | /api/v1/models의 reasoning.allowed_options·default | Responses의 reasoning.effort |
| vLLM | Provider API의 boolean·effort 요청 규약 | chat_template_kwargs의 enable_thinking·thinking, ON에서 최상위 reasoning_effort |
| SGLang | Provider API의 boolean·effort 요청 규약 | chat_template_kwargs의 enable_thinking·thinking·reasoning_effort |
| OpenRouter | /api/v1/models의 reasoning·supported_efforts·mandatory | reasoning.enabled와 ON에서 reasoning.effort |
| LiteLLM | 현재 어댑터에 제어 규약이 없는 경우 서버 기본값 | 추론 제어 인자 생략; 수신된 reasoning은 별도로 표시 |

vLLM·SGLang의 강도 후보는 minimal·low·medium·high·xhigh·max이다. 이는 API 요청 구문이며 모든 모델이 각 강도를 구현한다는 선언이 아니다. 같은 모델명이어도 서버 버전·template·parser·라우팅 설정에 따라 적용 결과가 달라질 수 있다. 모델 고유 boolean이나 강도를 다른 값으로 번역하는 규칙은 두지 않는다.

Ollama·LM Studio에서는 서버가 공개한 선택지를 우선한다. OFF를 제공하지 않으면 ON 전용으로 표시한다. 추론 출력 수신 자체는 ON/OFF 제어 지원의 증거가 아니다.

OpenRouter는 API Key를 Bearer 인증으로 보내며 기본 주소는 https://openrouter.ai/api/v1이다. 모델이 공개한 추론 선택지를 사용하고 mandatory=true이면 OFF를 차단한다. supported_efforts가 null이면 게이트웨이의 강도를 제공하고 필드가 생략되면 강도 선택을 제공하지 않는다. 모델 응답의 reasoning_details는 서명·암호화 데이터를 포함해 순서대로 보존하고 동일 연결·모델의 후속 도구 대화에 재전송한다. 중단된 응답의 상태는 재사용하지 않는다. 표시 가능한 추론만 Thinking으로 보여주며 동일한 reasoning 문자열을 중복 표시하지 않는다. 사용량은 서버가 실제 반환한 prompt_tokens·completion_tokens를 사용한다. [OpenRouter 추론 API](https://openrouter.ai/docs/guides/best-practices/reasoning-tokens), [모델 API](https://openrouter.ai/docs/api/api-reference/models/list-all-models-and-their-properties), [스트리밍 API](https://openrouter.ai/docs/api_reference/streaming).

API 근거: [vLLM reasoning](https://docs.vllm.ai/en/stable/features/reasoning_outputs/), [vLLM 요청 형식](https://github.com/vllm-project/vllm/blob/main/vllm/entrypoints/openai/chat_completion/protocol.py), [SGLang 요청 형식](https://github.com/sgl-project/sglang/blob/main/python/sglang/srt/entrypoints/openai/protocol.py), [Ollama Thinking](https://docs.ollama.com/capabilities/thinking), [LM Studio Responses](https://lmstudio.ai/docs/developer/openai-compat/responses).

## 연결 진단

진단은 일반 응답의 내용·완료와 가능한 ON/OFF 제어를 검사한다. ON에서는 분리된 추론과 완료된 답변이 필요하며, 같은 추론 문제의 OFF 요청에서는 추론이 없어야 한다. 서버가 제어를 무시하거나 요청이 실패하면 구체적인 오류를 반환한다. 명시적으로 선택한 강도도 별도 요청으로 확인한다.

강도 진단은 요청 수락·분리된 추론·답변 완료를 확인한다. low와 high의 실제 추론량 차이나 품질 차이를 인증하는 성능 측정은 아니다. 진단 성공은 native 도구 실행 능력·장시간 완수율·실제 가중치 동일성의 증명도 아니다. 진단은 기존 설정을 자동으로 바꾸지 않는다.

LM Studio는 Responses에 전체 역할별 메시지를 보내고 store:false·truncation:disabled를 사용한다. 원격 이전 응답 ID를 재사용하지 않는다. 수신된 추론은 제어 가능 여부와 독립적으로 화면·대화 기록에 보존한다.

## 서버 원본 표시

!model은 서빙 별칭 오른쪽에 서버가 선언한 단일 원본명을 흐리게 표시한다. 모델 정보에서는 실제 요청 ID, 식별 근거와 출처, 서버 한도와 제어 옵션을 확인한다. 같은 화면에서 모델 정보를 펼치고 PgUp/PgDn으로 스크롤하며 Esc로 접는다. Enter는 다른 항목과 동일하게 저장 확인이다.

| Provider | 조회 | 표시 근거 |
| --- | --- | --- |
| vLLM | /v1/models | 선택한 id의 root·parent·owned_by |
| OpenRouter | /api/v1/models | 선택한 id의 canonical_slug·name |
| SGLang | /model_info | 일치하는 served_model_name의 model_path·아키텍처·tokenizer 경로 |
| Ollama | /api/show | 원본 저장소·모델군·아키텍처·파라미터·양자화·manifest digest·Thinking 값 |
| LM Studio | /api/v1/models | key 또는 loaded_instances.id 일치, publisher·아키텍처·양자화·규모 |
| LiteLLM | /model/info, 불가하면 /model_group/info | 선택한 model_name의 upstream·Provider 선언 |

서버 원본 선언·모델군 정보·복수 모델/동적 라우팅·원본 식별 불가를 구분한다. 단일 저장소 ID·정확한 저장소 URL·캐시 경로는 표시용으로 정규화한다. 일반 로컬 파일명으로 체크포인트를 추측하지 않으며 표시된 원본은 추론 옵션 선택에 관여하지 않는다.

조회는 구성한 서버와 같은 origin에서 수행한다. 반환된 URL이나 경로를 따라가지 않고 인증·프롬프트·Modelfile·template 원문은 수집하지 않는다. 알려진 비밀값과 터미널 제어 문자는 제외한다. 식별 정보는 세션 선택 메타데이터로 저장한다.

메타데이터는 무수신 3분 제한과 최대 3회 시도를 적용한다. 응답은 16MiB, 목록은 10,000개, 보존 근거는 64개·값당 2,048바이트로 제한한다. 복구 후에도 조회할 수 없으면 모델 정보에 원인을 표시한다.

## 서빙 한도

선택 시 실제 서버 한도를 조회하고 모델 정보에 표시한다. vLLM·SGLang은 max_model_len, Ollama는 /api/ps의 로드된 컨텍스트, LM Studio는 선택 인스턴스의 config.context_length를 사용한다. LiteLLM은 같은 별칭의 모든 배포가 공개한 입력·출력 한도 각각의 최솟값을 사용한다. OpenRouter는 context_length·top_provider.context_length의 양수 최솟값과 top_provider.max_completion_tokens를 적용한다. 모델의 이론상 최대값과 현재 로드된 길이를 구분한다.

출력 예산은 설정과 확인된 서버 상한을 함께 적용한다. vLLM은 현재 context를 알면 /tokenize에 같은 템플릿·메시지·도구 정의를 보내 남은 공간을 계산한다. 이 값은 용량 확인에만 쓰며 UI usage가 아니다. 다른 Provider의 입력 토큰을 추정하지 않는다.

Ollama에는 확인된 context를 num_ctx로 지정하지만 공개 Chat API만으로 서버 내부 이력 잘림을 검출·방지할 수는 없다. 미공개 한도·실행 중 서버 설정 변경·라우팅 변경은 서버의 최종 검증 대상이다.

## Native tool calling

MECA는 Chat Completions의 tools·tool_calls, Ollama Chat의 객체형 인자, LM Studio Responses의 function_call·function_call_output을 사용한다. 일반 답변의 JSON은 실행하지 않는다. 서버가 도구를 지원하지 않거나 parser 설정이 틀리면 native 호출을 생성할 수 없다.

vLLM은 자동 도구 선택에 --enable-auto-tool-choice와 모델에 맞는 --tool-call-parser가 필요하다. Reasoning parser는 별도 설정이며 서빙 모델의 공식 권장 구성을 따른다. 서빙 별칭만으로 parser를 추측하지 않는다. [vLLM Tool Calling](https://docs.vllm.ai/en/latest/features/tool_calling/), [Ollama Tool Calling](https://docs.ollama.com/capabilities/tool-calling).
