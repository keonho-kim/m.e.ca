# 기본 스킬 안내

[← README](../README.md) · [스킬 사용·추가·형식](skills.md)

tui와 full에 같은 기본 스킬을 제공한다. 최초 초기화와 설치 시 `~/.meca/skills/`에 누락한 파일을 생성하고 기존 사용자 수정본은 보존한다. 스킬은 에이전트가 읽는 지침이며 별도 앱이나 실행 모드가 아니다.

| 호출 | 용도 | 배포 원본 |
| --- | --- | --- |
| `#create-skill` | 스킬 작성·수정과 참고 자료 구성 | [SKILL.md](../internal/config/bootstrap/defaults/create-skill/SKILL.md) |
| `#ui-design` | WebApp/SaaS 화면·사용성·표현·브라우저 검증 | [SKILL.md](../internal/config/bootstrap/defaults/ui-design/SKILL.md) |
| `#backend-design` | TDD와 자원·DB·API·진단·보안·검색 설계 | [SKILL.md](../internal/config/bootstrap/defaults/backend-design/SKILL.md) |

호출 뒤에 원하는 작업을 자연어로 적는다. 서브커맨드나 필수 인자는 없으며 에이전트가 프로젝트와 요청을 읽고 필요한 참고 파일을 선택한다. `!skills`에서 설치된 스킬의 설명·출처·본문을 확인할 수 있다.

## 스킬 작성

`#create-skill`은 SKILL.md의 이름·설명·본문, `references/`·`scripts/`·`assets/`의 역할과 검증 방법을 안내한다. 기본 작성 위치는 프로젝트 `.agents/skills/`다. 파일 배치·이름·호출 정책은 [스킬 추가와 형식](skills.md#스킬-추가하기)을 따른다.

## UI 설계

`#ui-design`은 데스크톱 WebApp/SaaS의 과업과 정보 관계에서 페이지 지도와 큰 레이아웃을 도출하도록 안내한다. 앱 셸·검색/결과·목록/상세·대시보드·편집·협업·문서·예약/주문·공개 페이지의 필수·조건부 영역을 다룬다.

- 한국어·혼합문자·IME와 데스크톱 업무 화면의 정보 밀도를 고려한다.
- 글래스모피즘·미니멀리즘·퓨처리즘, 타이포·공간·색의 역할과 대비를 상황에 맞게 선택한다.
- 모달·팝오버·알림·드롭다운·탭·아코디언·폼·Stepper의 기본 동작과 시각·모션 변형을 비교한다.
- Skeleton Pulse/Shimmer, 이미지 Blur-up, 실제 진행·Step Status·스트리밍을 구분하고 빈 상태·오류·취소를 함께 설계한다.
- 불필요한 카드·KPI·장식·근거 없는 성공 표시를 피하고 실제 내용과 작업 흐름에 맞는 구조·문구·강조를 사용한다.
- 키보드·포커스·접근성·빠른 재조작·축소 모션과 상태 수명을 확인한다.

[참고 지도](../internal/config/bootstrap/defaults/ui-design/SKILL.md)는 필요한 지침과 앱별 HTML/CSS/Vanilla JS 예시를 연결한다. 예시는 구성과 조작을 참고하는 자료이며 대상 프로젝트의 팔레트·화면 수·기술 구성을 지정하지 않는다. 결재·DB 워크벤치·지식 대화·오피스 예시는 정적 디자인 화면으로 실제 DB·AI·승인·문서 저장 서비스를 제공하지 않는다.

구현 확인에는 **Playwright CLI (`playwright-cli`)**를 사용한다. [브라우저 검증 지침](../internal/config/bootstrap/defaults/ui-design/references/browser-tools.md)은 실제 화면·상호작용·콘솔·네트워크 확인과 선택적인 레이아웃·모션 검사를 안내한다. 검증 도구는 개발 환경에서 실행하며 MECA나 생성 앱의 런타임 의존성이 아니다.

## 백엔드 설계

`#backend-design`은 설계 단계부터 CPU·메모리·저장공간·I/O·네트워크·응답 지연과 정확성·복구 비용을 함께 검토하도록 안내한다.

- 테스트 가능한 구현은 **Red → Green → Refactor의 TDD**를 기본으로 한다. 요구·불변식에서 동작 테스트를 만들고 의도한 실패를 확인한 뒤 최소 구현과 리팩토링을 반복한다.
- SQLite·PostgreSQL·Redis를 부하·일관성·운영 비용에 맞춰 선택하고 스키마·인덱스·트랜잭션·캐시 수명을 연결한다.
- API 명령·조회·멱등성·스트림·작업·웹훅·테넌트 격리의 실행과 실패 경계를 정한다.
- 중앙 로거·오류 코드·메시지·이벤트·필드 정책과 각 도메인의 실패 처리를 구분한다.
- 암호화·키 수명·검색·벡터·메모리·I/O·과부하·복구 기법의 이점과 비용을 비교한다.

[실전 구성 지도](../internal/config/bootstrap/defaults/backend-design/references/compositions.md)는 기능 모듈·트랜잭션 명령·조회 모델·워크플로·파이프라인·외부 연동·질의/규칙 처리의 책임과 자원 수명을 연결한다. [설계 비교 사례](../internal/config/bootstrap/defaults/backend-design/references/design-studies.md)의 수치는 명시한 가정이며 실측 성능이 아니다. 신규 설계는 부하 가정과 비용 모델로, 기존 구현은 프로파일·부하 시험으로 선택을 검증한다.

## 자료 사용

참고 자료는 **본문 → 지도 → 선택 자료**의 최대 세 층으로 구성하며 필요한 파일만 읽는다. 기본 스킬의 설명 Markdown은 파일별 8KiB, 코드·미디어는 1MiB 이하이며 512KiB 이하를 권장한다. 이 작성 기준과 엔진의 참고 파일 페이지 한도는 구분한다.

외부 자료의 출처와 배포 조건은 각 스킬에 동봉한 고지·출처 문서를 따른다. 기본 스킬을 읽어도 외부 서비스·DB·개발 도구를 자동 설치하거나 실행하지 않는다. 실제 프로젝트의 요구·기존 시스템·보안 정책을 우선한다.
