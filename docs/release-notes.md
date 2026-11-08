# M.E.CA — Minimal Essentials Coding Agent

Go 실행 파일 하나로 TUI·CLI·서버에서 코딩 작업을 수행한다. 배포 대상은 macOS·Linux의 amd64·arm64다.

## 제공 기능

- 스킬 카탈로그의 독립 탐색, 최신 버전·참고 페이지의 요청 내 보존과 manual-only 읽기 범위 검증
- 참고 페이지 revision 계약, bounded 경로 ID 재사용과 조회 만료 복구
- Chat 외부 이력의 요청 내 공유, Responses의 지속 이력·팀 원문 참조와 실패한 후속 접수의 원자적 정리
- 중첩 JSON 복원·인자 재사용·payload metadata batch 조회·Thinking/명령 출력/체크포인트 목표문 중복 제거
- 응답·이미지 decode·저장 변환·큰 원문 읽기의 공유 메모리 입장 제어와 취소 정리
- 내용 없는 Provider 스트림 정체 감지, 저장 flush 마감, 미게시 예약 반환·용량 경고와 동일 자격 증명의 참조 재사용
- 공통 SKILL.md의 프로젝트·개인·공유 탐색, !skills 목록·미리보기와 #스킬명 지정
- list_skills·load_skill·read_skill_file과 현재 요청의 읽기 지침 보존
- Bootstrap의 개인 스킬 경로·표준 작성 지침 create-skill 생성과 기존 사용자 파일 보존
- APPLICATION DESIGN 기본 스킬: 단일 호출에서 데스크톱 WebApp/SaaS 성향별 페이지·큰 레이아웃·필수 영역과 한국형 데스크톱 UX 자료 탐색
- 글래스모피즘·미니멀리즘·퓨처리즘 테마 파일과 Impeccable 62개 항목을 분석한 문맥별 anti-slop 지침
- 레이아웃별 Skeleton·재조회·저장·스트리밍·실제 진행 표시와 Stepper·폼·테이블·overlay·알림의 선택·상태·접근성 지침
- 모달·드롭다운·캐러셀·아코디언·입력·플로팅 버튼 등의 시각/모션 변형과 Swipe·Blinds·Depth Stack 전환 레시피
- 기본 스킬의 최대 세 층 탐색·개별 지침 8KiB 예산과 실제 Reader를 통한 전체 참고 링크 검증
- 과업·맥락·사용성에 따른 패턴 조합/변형 판단과 설계 전 과정에 연결한 anti-slop 지침
- Skeleton Pulse/Shimmer·기본 Blur-up·실제 Step Status, Hamburger Morph·Side Drawer·Staggered Dropdown 조합과 색 선택의 8가지 맥락별 원칙
- 여섯 앱과 예약 대안·자료 검토 캔버스·모션의 총 36개 HTML. 결재 앱을 문서·결재선 중심으로 전면 재설계하고 JSON 검사 대신 연결·스키마·SQL·결과를 갖춘 DB 워크벤치 디자인으로 교체
- DB 근거 기반 지식 대화 5화면과 AI 문서·슬라이드 오피스 6화면 추가. 결재·DB·지식 대화·오피스는 정적 디자인 셸과 최소 탭·모달·탐색만 제공하며 실제 서비스·저장·편집·내보내기는 실행하지 않음
- 동봉 예시의 오피스 문서·슬라이드 작업면, Archive 미니멀리즘·에디토리얼 구성과 역할별 포인트 색, 앱별 글래스 표면으로 정체성을 구분. Querydesk 다크 테마, 과업별 탐색·탭, 취소 가능한 상태 모션과 접근성 대체 표면 보강
- application-design의 본문과 64개 참고 문서를 설계 판단·선택 조건·장단점 중심으로 정리. 가상 사례와 디자인 기록의 범위를 구분하고 대상 프로젝트의 요구·기존 시스템을 우선하도록 안내
- 기존 예시의 타이포·대비·입력 경계·한글 줄바꿈과 화면별 정보 위계 보강. 파일별 1MiB 허용·512KiB 이하 권장과 외부 런타임 의존성 없는 배포 유지
- 에셋을 앱별·공통·모션·참고 캡처 폴더로 분리하고 Markdown 직접 접근 안내로 관리. 갤러리·전용 축소 이미지·카탈로그 이동 UI 제거
- 타이포·형태·공간·밀도·이미지·전체 구성·상호작용·모션을 독립적으로 판단하는 지침과 의도한 표현을 보존하는 anti-slop 사례 추가. 실제 렌더링 화면을 선택적 참고 자료로 배포
- Product Design의 시각적 근거·완전한 사용 흐름·실제 렌더링/상호작용 검토와 Apple·Instagram/Meta·Google·Hallmark 예시의 선별한 원칙을 독립된 지침으로 반영
- 기본 스킬의 본문·참고 파일·라이선스 동봉, 누락 파일 복구와 사용자 수정본 보존
- 모든 native 전송의 스킬 조회 API·skill_loaded 메타데이터와 기존 이력 복원
- 작업 식별자·발생 시각·오류 코드 보강, bounded JSON 운영 로그와 private TUI 파일 회전

- Ollama·vLLM·SGLang·LM Studio·LiteLLM·OpenRouter·OpenAI·Claude·Gemini 연결
- 스트리밍 답변·Thinking, native 도구 호출, 이미지 입력이 확인된 모델의 `view_image`
- 파일 생성·정확한 부분 수정·셸 실행과 장기 명령 조회·대기·취소
- 로컬 대화·실행 결과·사용량 저장, 압축·재개·JSONL 내보내기
- 작업 중 방향 수정과 후속 요청 예약, 최대 8개 독립 세션의 팀 협업
- `--server --protocol sse|socket|grpc`의 세션·모델·작업·승인·취소 API
- `--server --openai`의 텍스트 Chat Completions·Responses·SSE·background 조회·취소
- 서버 스트림의 5초 무출력 후 1초 heartbeat, 커서 재접속과 요청 키 중복 방지
- API Key 암호화 보관·OS 자격 증명 저장소 연동과 내부 주기적 키 교체

## 안정성과 자원 관리

- 메시지·오류 코드·이벤트·레벨·복구 문구를 중앙 diagnostics에 모으고 raw 로거·임의 진단 선언을 개발 검사에서 차단한다.
- 출력 장애의 즉시 health 신호, 제한된 반복 경고·주기 집계, interrupt 후 writer 합류와 중단된 로그 회전 정리를 제공한다.
- 스킬 메타데이터 snapshot·반복 읽기 중복 제거·문맥 투영 재사용·같은 파일 대여·TUI 희소 미리보기와 닫기 시 취소를 적용한다.
- 입력 크기에 맞는 서버 메모리 예약, 동시 모델 문맥 예산, 취소 가능한 명령 출력 입장, 상세 JSON 증분 인코딩으로 불필요한 보관·재계산을 줄인다.
- 도구 출력의 문자열·JSON 저장 원본을 공유하고 호출 묶음의 journal 커밋을 합친다.
- Task의 code·persistence를 제공한다. 완료 저장을 읽기로 확인하고 불확실하면 새 접수를 막으며, 도구 효과나 내부 실패를 자동 재생하지 않는다.

- native 서버 요청과 OpenAI 호환 요청의 접수를 프로젝트별 SQLite에서 관리하고, 중복 요청과 응답 후속 연결을 트랜잭션으로 확정한다.
- 선택한 대화와 실행 중인 작업의 소유권을 구분하고, 취소·종료 시 모델 연결·조회 작업·명령 출력 자료를 정리한다. 모델 교체 결과가 불확실하면 기존 연결을 계속 사용하지 않는다.
- 세션 저장의 중복 식별자·인덱스와 반복 조회를 줄이고, 준비한 기록을 함께 저장한다. 팀 공개 문맥은 확정한 원본과 출처를 기준으로 복원한다.
- 서버 입력·조회·출력과 SQLite 연결·저장 용량을 제한하고, 긴 응답·셸 출력·TUI Markdown에서 불필요한 복사와 보관을 줄인다.

## 실행과 보관

기본 `meca`는 TUI다. `--print`는 최종 답변, `--mode json`은 순서 있는 이벤트를 출력한다. 서버는 TTY 없이 작업을 소유하며 관찰 연결 종료만으로 작업을 취소하지 않는다. native 서버 API의 직접 승인과 승인 채널이 없는 호환 API의 거절을 구분한다.

세션 DB는 schema 28을 사용한다. 미지원 DB는 목록에 사용 불가 사유를 표시하고 열기·내보내기를 거부하며 새 대화 시작은 유지한다. 서버 접수 DB는 user_version 2로 프로젝트별 SQLite에 저장하며 이전 SQLite 형식도 자동 변환하지 않는다. 이전 JSON 접수 파일이 있으면 원본을 보존하고 서버 시작을 거부한다. 자동 변환·삭제·재실행은 하지 않는다. 기본 프롬프트는 `~/.meca/prompts/single/`과 `team/`에 보관하며 install·upgrade는 배포 기본본으로 갱신한다. 대화 백업에는 사용자 DB와 팀 개인 DB, 자격 증명 백업에는 `credentials.db`와 OS 키 저장소를 함께 포함한다.

[설치와 실행](https://github.com/keonho-kim/m.e.ca#설치) · [Provider](https://github.com/keonho-kim/m.e.ca#지원-provider) · [서버 API](https://github.com/keonho-kim/m.e.ca/blob/main/docs/server.md) · [설정과 보관](https://github.com/keonho-kim/m.e.ca/blob/main/docs/configuration.md)
