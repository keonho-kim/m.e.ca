# 모델 프로파일

검토 기준일: **2026-10-13**. 모델 특화는 [`internal/provider/profile`](../internal/provider/profile)에서 관리한다. HTTP 어댑터는 이 패키지의 결정만 요청 형식으로 옮긴다. 새 모델이 출시돼도 이름 패턴만으로 자동 지원을 확대하지 않는다.

## 대상과 책임

- 모델군: GLM, Qwen 및 확인된 파생형, Nemotron, DeepSeek, EXAONE, Gemma.
- 출시 기준: **2026-03-01 포함 이후**. 저장소 수정일·양자화 재게시일을 원래 모델의 출시일로 사용하지 않는다.
- 규모 기준: 공식 모델 카드가 보고한 **전체 아키텍처 파라미터 300B 이하**. MoE의 활성 규모나 양자화 파일 용량으로 판정하지 않는다.
- 출시 정보가 월 단위로만 확인되면 `YYYY-MM`으로 기록한다. 정확한 날짜를 만들어 넣지 않는다.
- 일반 대화 API 연결은 프로파일 등록 범위 밖의 모델에도 가능하다. 특화 요청은 검토된 프로파일에만 적용한다.
- `glm.go`, `qwen.go`, `nemotron.go`, `deepseek.go`, `exaone.go`, `gemma.go`는 각 모델군의 정확한 이름·출시·규모·공식 출처·제어 키를 소유한다.
- `profile.go`는 정확한 식별, 범위 판정, 명시적 재정의와 충돌 검사를 담당한다. 동적 플러그인·임의 요청 필드·Python 실행은 사용하지 않는다.
- 공통 Agent 프롬프트, 프로젝트 AGENTS.md, 단계별 리터럴 도구 프로토콜과 승인·Scope 규칙을 모델 프로파일로 변경하지 않는다.

## 등록된 프로파일

각 행의 이름은 `profile` 설정에 사용하는 ID다. 전체 규모는 공식 카드의 아키텍처 수치이며, 보조 모듈을 포함한 weight 집계와 차이가 있으면 새 변형을 별도로 검토한다. 이미지·음성 능력은 이 표가 관리하는 텍스트 Agent 요청에 추가되지 않는다.

| ID | 출시 | 전체 규모 | 문서상 템플릿 제어 | 출처 |
| --- | --- | --- | --- | --- |
| `qwen3.5-0.8b` | 2026-03-02 | 0.8B | `enable_thinking` | [카드](https://huggingface.co/Qwen/Qwen3.5-0.8B) |
| `qwen3.5-2b` | 2026-03-02 | 2B | `enable_thinking` | [카드](https://huggingface.co/Qwen/Qwen3.5-2B) |
| `qwen3.5-4b` | 2026-03-02 | 4B | `enable_thinking` | [카드](https://huggingface.co/Qwen/Qwen3.5-4B) |
| `qwen3.5-9b` | 2026-03-02 | 9B | `enable_thinking` | [카드](https://huggingface.co/Qwen/Qwen3.5-9B) |
| `qwen3.6-35b-a3b` | 2026-04-16 | 35B | `enable_thinking` | [카드](https://huggingface.co/Qwen/Qwen3.6-35B-A3B) |
| `qwen3.6-27b` | 2026-04-22 | 27B | `enable_thinking` | [카드](https://huggingface.co/Qwen/Qwen3.6-27B) |
| `qwen3.8-27b` | 2026-08-14 | 27B | `enable_thinking` | [카드](https://huggingface.co/Qwen/Qwen3.8-27B) |
| `ornith-1.0-9b` | 2026-06 | 9B | 미확인 | [카드](https://huggingface.co/ornith-ai/Ornith-1.0-9B) |
| `ornith-1.0-35b` | 2026-06 | 35B | 미확인 | [카드](https://huggingface.co/ornith-ai/Ornith-1.0-35B) |
| `ornith-1.5-9b` | 2026-08 | 9B | `enable_thinking` | [카드](https://huggingface.co/ornith-ai/Ornith-1.5-9B), [템플릿](https://huggingface.co/ornith-ai/Ornith-1.5-9B/blob/main/chat_template.jinja) |
| `ornith-1.5-35b-a3b` | 2026-08 | 35B | `enable_thinking` | [카드](https://huggingface.co/ornith-ai/Ornith-1.5-35B-A3B), [템플릿](https://huggingface.co/ornith-ai/Ornith-1.5-35B-A3B/blob/main/chat_template.jinja) |
| `nemotron-3-nano-4b` | 2026-03-16 | 3.97B | `enable_thinking` | [카드](https://huggingface.co/nvidia/NVIDIA-Nemotron-3-Nano-4B-BF16) |
| `nemotron-3-super-120b-a12b` | 2026-03-11 | 120B | `enable_thinking` | [카드](https://huggingface.co/nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-BF16) |
| `nemotron-3-nano-omni-30b-a3b` | 2026-04-28 | 31B | `enable_thinking` | [카드](https://huggingface.co/nvidia/Nemotron-3-Nano-Omni-30B-A3B-Reasoning-FP8) |
| `nemotron-3.5-lightning-30b-a3b` | 2026-08-11 | 30B | `enable_thinking` | [카드](https://huggingface.co/nvidia/NVIDIA-Nemotron-3.5-Lightning-30B-A3B-BF16) |
| `deepseek-v4-flash` | 2026-04-24 | 284B | vLLM만: `thinking`·`enable_thinking` | [카드](https://huggingface.co/deepseek-ai/DeepSeek-V4-Flash), [vLLM tokenizer](https://github.com/vllm-project/vllm/blob/main/vllm/tokenizers/deepseek_v4.py) |
| `exaone-4.5-33b` | 2026-04-09 | 33B | `enable_thinking` | [카드](https://huggingface.co/LGAI-EXAONE/EXAONE-4.5-33B) |
| `gemma-4-e2b-it` | 2026-04-02 | 5.1B | `enable_thinking` | [카드](https://huggingface.co/google/gemma-4-E2B-it) |
| `gemma-4-e4b-it` | 2026-04-02 | 8B | `enable_thinking` | [카드](https://huggingface.co/google/gemma-4-E4B-it) |
| `gemma-4-26b-a4b-it` | 2026-04-02 | 25.2B | `enable_thinking` | [카드](https://huggingface.co/google/gemma-4-26B-A4B-it) |
| `gemma-4-31b-it` | 2026-04-02 | 30.7B | `enable_thinking` | [카드](https://huggingface.co/google/gemma-4-31B-it) |
| `gemma-4-12b-it` | 2026-06-03 | 11.95B | `enable_thinking` | [카드](https://huggingface.co/google/gemma-4-12B-it) |

출시 근거: [Qwen 공식 이력](https://github.com/QwenLM/Qwen3.8), [Ornith 공식 발표](https://ornith.ai/), [DeepSeek V4 발표](https://api-docs.deepseek.com/news/news260424), [EXAONE 공식 이력](https://github.com/LG-AI-EXAONE/EXAONE-4.5), [Gemma 4 발표](https://blog.google/innovation-and-ai/technology/developers-tools/gemma-4/), [Gemma 4 12B 발표](https://blog.google/innovation-and-ai/technology/developers-tools/introducing-gemma-4-12b/). Nemotron은 각 카드의 Release Date를 사용한다.

- Gemma E2B/E4B는 effective 규모 대신 임베딩을 포함한 전체 수치를 사용한다.
- Qwen 0.8B/2B의 공식 카드는 Thinking 반복 위험을 설명한다. V의 유한 호출·시간·반복 예산을 유지하며 모델 크기를 근거로 예산을 해제하지 않는다.
- Nemotron Super의 공식 BF16·FP8·NVFP4, Lightning의 BF16·NVFP4 이름은 동일 프로파일에 명시적으로 등록했다. 파일명 접미사를 지워 임의 양자화를 자동 등록하지 않는다.
- Ornith 1.0의 ON/OFF는 문서상 확인되지 않아 템플릿 제어를 보내지 않는다. Qwen 파생형이라는 이유로 Ornith 1.5의 스위치를 복사하지 않는다.
- DeepSeek V4 Flash의 현재 vLLM tokenizer는 두 boolean 값을 OR한다. OFF 요청은 둘 다 false로 지정한다. SGLang의 동일한 제어 계약은 이번 검토에서 확정하지 않았다.

## 모델별 Thinking 강도

ON/OFF와 강도를 별도로 다룬다. 같은 API 필드가 수락된다고 모든 모델이 같은 강도 의미를 구현하는 것은 아니다. 현재 모델 카드·템플릿에서 확인한 값만 프로파일에 등록한다.

| 프로파일 | UI·TOML 값 | vLLM 요청 | SGLang 요청 | 근거 |
| --- | --- | --- | --- | --- |
| `qwen3.8-27b` | `low`, `medium`, `xhigh` | 최상위 `reasoning_effort` 문자열 | `chat_template_kwargs.reasoning_effort` 문자열 | [카드](https://huggingface.co/Qwen/Qwen3.8-27B), [템플릿](https://huggingface.co/Qwen/Qwen3.8-27B/blob/main/chat_template.jinja) |
| `nemotron-3-super-120b-a12b` | `low`, `full` | `chat_template_kwargs.low_effort`를 true/false로 지정 | 동일한 문서화된 템플릿 boolean | [카드](https://huggingface.co/nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-BF16), [템플릿](https://huggingface.co/nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-BF16/blob/main/chat_template.jinja) |
| `deepseek-v4-flash` | `high`, `max` | 최상위 `reasoning_effort` 문자열 | 미확인 | [카드](https://huggingface.co/deepseek-ai/DeepSeek-V4-Flash), [tokenizer](https://github.com/vllm-project/vllm/blob/main/vllm/tokenizers/deepseek_v4.py) |
| 나머지 등록 모델 | `DEFAULT` | 명시적 강도 없음 | 명시적 강도 없음 | 강도별 의미 미확인: ON/OFF 지원과 구분 |

- `DEFAULT`는 빈 값이며 명시적 강도 필드를 생략한다. Qwen의 Provider 기본 강도가 xhigh여도 V가 강도를 xhigh로 강제 지정한 것은 아니다.
- Nemotron의 `full`은 `low_effort=false`라는 UI 값이다. 서버에 `reasoning_effort="full"`을 보내지 않는다.
- Gemma의 vLLM API가 `reasoning_effort`로 Thinking을 켤 수 있다는 사실만으로 low/medium/high를 서로 다른 모델 강도로 등록하지 않는다.
- Ornith·다른 Qwen·Nemotron Lightning에는 다른 체크포인트의 강도를 복사하지 않는다. Lightning 카드와 템플릿에서는 Super의 `low_effort` 계약을 확인하지 못했다.
- LiteLLM·일반 OpenAI-compatible에는 검증되지 않은 강도를 보내지 않는다. Ollama·LM Studio는 native 메타데이터의 정확한 선택지를 별도로 검증한다. 이는 모델군 이름 추측이나 프로파일 카탈로그 확대가 아니다.
- SGLang의 최상위 정규화는 Thinking을 토글하므로, Qwen 템플릿의 강도 문자열을 `chat_template_kwargs`에 명시한다. [SGLang 프로토콜](https://github.com/sgl-project/sglang/blob/main/python/sglang/srt/entrypoints/openai/protocol.py)

### 선택·저장·복구

1. `!model`의 `Thinking effort` 행을 선택한다. 지원 모델에서 `DEFAULT`와 그 모델의 허용 값만 목록에 보인다.
2. `↑`/`↓`로 고르고 Enter로 진단·적용한다. OFF 상태에서도 선호 강도는 설정할 수 있으며 설정만으로 Thinking을 켜지 않는다.
3. 현재 서버에 선택한 강도로 짧은 ON 진단 요청을 보낸다. HTTP 수락·별도 추론·완료를 확인한 뒤 SQLite에 저장한다. 실패·취소·저장 실패에서는 기존 값이 남는다. DEFAULT는 명시적 강도 요청이 없어 추가 모델 호출을 하지 않는다.
4. 상태줄은 ON에서 `Thinking MEDIUM`처럼 명시한 강도를 표시한다. OFF는 Thinking·강도를 모두 생략한다.
5. 모델별 TOML 기본값보다 같은 Provider·주소·모델·프로파일에 저장된 세션 선택이 우선한다. 값이 없는 세션은 TOML 기본값을 상속하며, 명시적인 DEFAULT는 빈 문자열로 보존해 재시작 시에도 Provider 기본값을 유지한다.
6. 재시작 때 허용 값과 현재 서버의 진단을 다시 확인한다. 모델·프로파일을 바꾸면 이전 모델의 강도를 넘기지 않는다.

```toml
[[model_profiles]]
provider = "vllm"
endpoint = "http://localhost:8000/v1"
model = "team-coder"
profile = "qwen3.8-27b"
reasoning_effort = "medium"
```

- 모든 일반 생성·도구 인자 수집·검증·Compaction 요청에 선택한 강도를 전달한다. OFF에서는 `reasoning_effort`와 `low_effort`를 생략해 강도 필드가 Thinking을 다시 켜지 않도록 한다.
- 모델 출력의 품질·길이·지연만으로 강도의 의미가 적용됐다고 판정하지 않는다. 진단에 성공해도 서버가 옵션을 무시할 수 있으므로 실제 강도별 계산량·수렴은 환경 검증 대상이다.
- 강도를 높여도 V의 모델 호출·문맥·출력·시간·반복 예산을 자동 확대하지 않는다.

## 제외와 미확인

| 모델 | 처리·근거 |
| --- | --- |
| GLM-4.5-Air | 2025-07 모델: 출시 범위 밖. 이전 vLLM 자동 특화 제거. [카드](https://huggingface.co/zai-org/GLM-4.5-Air) |
| GLM-4.7-Flash | 2026-01 모델: 출시 범위 밖. [카드](https://huggingface.co/zai-org/GLM-4.7-Flash) |
| GLM-5.3-Flash | 전체 320B: 규모 범위 밖. GLM군 파일에 제외 기록을 보존한다. [카드](https://huggingface.co/zai-org/GLM-5.3-Flash-BF16) |
| Qwen3.5-122B-A10B | 2026-02-24 모델: 크기가 작아도 출시 범위 밖. [이력](https://github.com/QwenLM/Qwen3.8) |
| Ornith-1.5-397B | 전체 397B: 활성 규모가 작아도 범위 밖. [발표](https://ornith.ai/ornith_1_5.html) |
| 추가 checkpoint, 파생형, 양자화, draft·base·embedding 모델 | 이름·템플릿·출시·전체 규모의 검토 없이 추정 등록하지 않는다. 새 체크포인트 프로파일로 별도 검토한다. |

이번에 확인한 GLM 공개 텍스트 모델 중 두 조건을 만족하는 신규 모델은 없었다. GLM 프로파일을 강제로 켜기 위해 날짜·규모 제한을 우회하지 않는다. `generic` 일반 연결은 이 카탈로그의 모델 지원 인증이 아니다.

## 서빙 이름을 식별할 수 없을 때

연결 시 선택한 서빙 이름의 Provider 메타데이터를 조회한다. 정확한 원본 ID가 검토된 카탈로그와 일치하면 별칭에도 프로파일을 자동 적용한다. 원본을 확인할 수 없는 경우 서버 운영자가 실제 checkpoint와 사용 중인 template·reasoning parser를 확인한 뒤 `~/.v/config.toml`에 매핑을 추가한다.

```toml
[[model_profiles]]
provider = "vllm"
endpoint = "http://localhost:8000/v1"
model = "local-coder"
profile = "ornith-1.5-35b-a3b"
```

- `model`은 **실제 HTTP 요청에서 사용할 서빙 이름**이다. 원래 Hugging Face ID로 바꾸지 않는다.
- Provider·정규화된 API 주소·서빙 이름이 모두 맞는 경우에만 매핑을 적용한다. 같은 별칭이라도 다른 주소나 Provider로 적용하지 않는다.
- 설정을 저장하고 V를 다시 시작한 뒤 `!model`로 연결한다. `!model` 화면은 적용된 프로파일 ID를 표시하고 SQLite 선택 기록에도 저장한다.
- 정확한 공식 이름과 매핑이 서로 다른 체크포인트를 가리키면 오류다. 중복 매핑·범위 밖 프로파일·오탈자도 시작 시 거부한다.
- `profile = "generic"`은 자동 특화를 명시적으로 끈다. 사용자 정의 template로 원본과 제어 방식이 달라진 경우에 사용할 수 있다.
- 일반 별칭, 제3자 namespace, 확인되지 않은 파일명만으로 특화를 적용하지 않는다. 정확한 원본 근거나 명시적 매핑이 없으면 일반 프로파일을 사용한다.
- 재시작 때 저장된 프로파일·식별 기록을 지원 증거로 재사용하지 않는다. 현재 설정과 서버 메타데이터로 다시 판별하고 연결 진단을 반복한다.
- 모델의 자기소개나 한 번의 응답은 실제 checkpoint의 증명이 아니다. 진단은 제어 동작만 확인한다. 서버가 같은 별칭을 다른 가중치로 바꿨다면 운영자가 매핑도 다시 검토해야 한다.

## Provider별 원본 식별

`!model` 상단은 서빙 별칭 오른쪽에 단일 원본 식별명을 흐린 글씨로 표시한다. 원본 저장소 URL·캐시 경로는 기존 저장소 ID 해석을 사용하고, 복수 대상·미식별·불완전한 결과나 별칭과 동일한 이름은 생략한다.

`!model → 모델 정보`에서 요청 모델, 식별 상태, 프로파일 적용 근거, 서버 선언과 출처를 확인한다. `PgUp/PgDn`으로 스크롤하고 `Enter/Esc`로 모델 메뉴에 돌아간다. 화면은 흑백이며 실제 요청의 모델명은 서빙 별칭 그대로 유지한다.

| Provider | 조회 | 식별 근거 |
| --- | --- | --- |
| vLLM | `/v1/models` | 선택한 `id`의 `root`; `parent`가 있으면 어댑터 변경 가능성을 고려해 특화 보류 |
| SGLang | `/model_info` | 일치하는 `served_model_name`의 `model_path`, 모델군·아키텍처·tokenizer 경로 |
| Ollama | `/api/show` | 모델군, 파라미터·양자화, GGUF 아키텍처·원본 저장소·첫 부모 모델, cloud 원본 이름, 선택된 manifest digest, 광고된 Thinking 값 |
| LM Studio | `/api/v1/models` | `key` 또는 `loaded_instances.id` 일치, publisher·아키텍처·양자화·규모·광고된 reasoning 옵션 |
| LiteLLM | `/model/info` | 선택한 `model_name`의 모든 upstream 모델. 상세 조회 불가 시 `/model_group/info?model_group=…`에서 Provider 근거 확인 |
| 일반 OpenAI-compatible | `/v1/models` | 해당 서버가 제공하는 선택 모델의 `root`, `parent`, `owned_by` |

- 조회 결과는 **서버 원본 선언**, **모델군·아키텍처 정보만 확인**, **복수 모델/동적 라우팅**, **원본 식별 불가**로 표시한다. 프로파일 근거는 서버 원본·서빙 이름의 카탈로그 일치, 사용자 TOML 매핑, 특화 없음으로 별도 표시한다. 서버 선언은 실제 가중치 동일성의 보증이 아니다.
- 공식 저장소 ID, 정확한 `https://huggingface.co/<publisher>/<model>` URL, `models--<publisher>--<model>/snapshots/<revision>` 캐시 경로를 카탈로그와 비교한다. 일반 로컬 경로의 basename·GGUF 양자화 접미사·tokenizer·부모 모델만으로 정확한 checkpoint를 추측하지 않는다.
- LiteLLM의 문서화된 transport 접두어만 분리한다. 여러 upstream이나 auto-router가 확인되면 하나를 선택하지 않는다. 복수 대상·불완전한 근거에서는 일반 프로파일을 사용하고 특화 매핑은 거부한다. `generic` 명시 또는 단일 모델로 직접 연결하여 해결한다. 원본 근거와 매핑의 충돌도 거부한다.
- Thinking 광고 정보와 실제 ON/OFF·강도 제어 지원은 별개다. 식별 성공만으로 제어 계약을 활성화하지 않는다. Ollama·LM Studio의 native 선택지도 실제 reasoning 분리와 ON/OFF 대조 진단을 거친다. 날짜·전체 300B 제한도 유지한다.
- 메타데이터는 전체 경과시간 대신 무수신 3분 제한과 최대 3회 시도를 적용한다. 응답은 4MiB, 목록은 10,000개, 보존 근거는 64개·값당 2,048바이트로 제한한다. 미지원·권한 거부·유효하지 않은 응답·무수신은 복구 시도 후 정보 조회 불가로 처리하고 안전하게 정리한 원인을 정보 화면에 표시하며 일반 대화 진단은 계속한다. 사용자 취소는 즉시 전파한다.
- 조회는 구성한 서버와 같은 origin에서 수행한다. 반환된 URL·파일 경로를 따라가지 않고 `/server_info`·프롬프트·template·Modelfile·인증 필드·upstream API 주소는 수집하지 않는다. 알려진 API 키와 터미널 제어 문자를 포함한 근거는 제외한다. 식별 기록은 SQLite 선택에 저장하며 모델 대화 context에 넣지 않는다.

API 근거: [vLLM 모델 카드](https://docs.vllm.ai/en/stable/api/vllm/entrypoints/openai/models/serving/), [SGLang 모델 정보 구현](https://github.com/sgl-project/sglang/blob/main/python/sglang/srt/entrypoints/http_server.py), [Ollama 모델 상세](https://docs.ollama.com/api-reference/show-model-details), [GGUF 출처 메타데이터](https://github.com/ggml-org/ggml/blob/master/docs/gguf.md), [LM Studio 모델 목록](https://lmstudio.ai/docs/developer/rest/list), [LiteLLM 모델 관리](https://docs.litellm.ai/docs/proxy/model_management), [LiteLLM 모델 그룹 구현](https://github.com/BerriAI/litellm/blob/main/litellm/proxy/proxy_server.py).

Provider별 HTTP 계약은 로컬 모의 서버로 검증했다. 실제 서버 버전·권한·프록시·배포 구성을 연결한 검증은 별도로 필요하다.

## Provider 제어와 연결 진단

| Provider | ON/OFF 제어 후보 | 별도 추론 출력 |
| --- | --- | --- |
| Ollama | `/api/show.thinking.values`의 boolean·정확한 문자열을 `think`로 전달; OFF 미지원은 ON 전용 | `message.thinking` |
| vLLM | 검토된 프로파일의 `chat_template_kwargs` | 현재 `reasoning` |
| SGLang | 검토된 프로파일의 `chat_template_kwargs`; vLLM 전용 기록 제외 | 현재 `reasoning_content` |
| LiteLLM | upstream 제어·전달 계약 미검증: 토글 비활성 | 현재 `reasoning_content` |
| LM Studio | native 지원값을 Responses `reasoning.effort`로 전달; binary none/medium 후보의 대조 진단 | Responses reasoning text 이벤트 |
| 일반 OpenAI-compatible | boolean 제어 계약 미검증: 토글 비활성 | 기존 일반 응답 계약 |

공식 프로토콜 근거: [vLLM reasoning](https://docs.vllm.ai/en/stable/features/reasoning_outputs/), [SGLang reasoning](https://docs.sglang.io/docs/advanced_features/separate_reasoning), [Ollama Thinking](https://docs.ollama.com/capabilities/thinking).

1. 명시한 프로파일과 실제 wire model ID의 충돌을 검사한다.
2. 일반 인사 요청으로 비어 있지 않은 응답과 완료를 확인한다. 가능한 제어가 있으면 OFF를 사용한다. 특정 정답이나 PFC 응답 형식은 요구하지 않으며 로컬 도구를 실행하지 않는다.
3. OFF에서 별도 reasoning이 나오면 서버 설정 오류로 연결을 거부한다.
4. ON으로 짧은 계산 요청을 보내 별도 reasoning·답변·완료 이벤트를 확인하고, 같은 질문을 OFF로 보내 추론 출력 억제까지 확인해야 토글을 활성화한다. OFF 미지원 모델은 ON만 검증하며 토글을 제공하지 않는다. 별도 reasoning이 관측되지 않으면 제어는 unavailable이며 Provider 기본 생성 모드를 사용한다.
5. HTTP 오류·미완료·출력 한도 초과는 진단 실패로 보고하고 클라이언트를 설치하지 않는다. 성공한 클라이언트의 정리 책임만 세션에 이전한다.

LM Studio는 Responses에 매번 전체 역할별 메시지를 보내며 `store: false`와 `truncation: disabled`를 사용한다. 재시도·Compaction에서 원격 이전 응답 ID를 재사용하지 않는다. reasoning 선택지는 native 모델 목록에서 확인한다. native 제어 메타데이터의 조회·검증 실패는 연결 실패로 표시한다. 이는 선택적인 원본 식별 조회와 별개다. 서버 한도·기본 강도·미지원 OFF 처리와 보장 범위는 README의 연결 계약을 따른다. 실제 서버·모델별 호환성은 추가 실기 검증 대상이다.

연결 진단은 해당 서버에서의 짧은 동작 확인이다. 모든 모델군의 장시간 수렴·속도·정확도·문맥 길이·실제 서버 버전 조합을 검증했다는 의미로 사용하지 않는다. V의 문맥·출력·실행 예산을 모델 카드의 최대값으로 자동 늘리지 않는다.

## 유지보수

- 해당 모델군 파일에 정확한 checkpoint 이름, 출시 정보, 전체 규모, 출처와 확인된 제어를 함께 추가한다.
- 알려진 모델과 파생형의 제어를 혼동하지 않는다. boolean OFF가 없는 모델은 이 UI의 ON/OFF 지원으로 등록하지 않는다.
- 새 서빙 버전·template의 제어 의미가 바뀌면 그 계약을 직접 갱신한다. 오래된 wire 필드를 대체하는 호환 분기를 추가하지 않는다.
- 식별·범위·충돌 테스트, HTTP ON/OFF·강도 변환·OFF 강도 생략 및 reasoning 필드 테스트, 연결 setup·저장·TUI 테스트를 실행한다.
- 검토 기준일과 이 문서의 목록·출처를 함께 갱신한다. 자동 네트워크 모델 카탈로그 갱신이나 외부 telemetry는 추가하지 않는다.
