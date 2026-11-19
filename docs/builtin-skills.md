# 기본 스킬 안내

[← README](../README.md) · [스킬 사용·추가·형식](skills.md)

tui와 full에 같은 기본 스킬을 제공한다. 최초 초기화와 설치 시 `~/.meca/skills/`에 누락한 파일을 생성하고 기존 사용자 수정본은 보존한다. 스킬은 에이전트가 읽는 지침이며 별도 앱이나 실행 모드가 아니다.

| 호출 | 용도 | 배포 원본 |
| --- | --- | --- |
| `#create-skill` | 스킬 작성·수정과 참고 자료 구성 | [SKILL.md](../internal/config/bootstrap/defaults/create-skill/SKILL.md) |
| `#ui-design` | WebApp/SaaS 컴포넌트 선택·상태·상호작용·모션 설계와 검증 | [SKILL.md](../internal/config/bootstrap/defaults/ui-design/SKILL.md) |
| `#backend-design` | TDD와 자원·DB·API·진단·보안·검색 설계 | [SKILL.md](../internal/config/bootstrap/defaults/backend-design/SKILL.md) |
| `#application-best-dev-practices` | 앱 유형과 기능을 조합한 구현 흐름·권한·상태·실패 복구 | [SKILL.md](../internal/config/bootstrap/defaults/application-best-dev-practices/SKILL.md) |

호출 뒤에 원하는 작업을 자연어로 적는다. 서브커맨드나 필수 인자는 없으며 에이전트가 프로젝트와 요청을 읽고 필요한 참고 파일을 선택한다. `!skills`에서 설치된 스킬의 설명·출처·본문을 확인할 수 있다.

## 스킬 작성

`#create-skill`은 SKILL.md의 이름·설명·본문, `references/`·`scripts/`·`assets/`의 역할과 검증 방법을 안내한다. 기본 작성 위치는 프로젝트 `.agents/skills/`다. 파일 배치·이름·호출 정책은 [스킬 추가와 형식](skills.md#스킬-추가하기)을 따른다.

## UI 설계

`#ui-design`은 사용자가 하려는 일, 계속 보여야 할 정보, 값이 확정되는 시점에서 컴포넌트와 동작을 도출한다. 판단이 어려운 선택에는 적합한 조건과 실제 확인 방법을 제시한다.

- **컴포넌트 선택**에서 브레드크럼·페이지네이션·햄버거 메뉴, 스플릿 버튼·미트볼 메뉴·컨텍스트 메뉴, 모달·드로어·스크림 등 비슷한 패턴의 쓰임과 구성 요소를 비교한다. 상세 지침에서 키보드 이동, 포커스 복귀, 확정·취소, 로딩·오류 상태를 확인한다. [컴포넌트 선택표](../internal/config/bootstrap/defaults/ui-design/references/components/index.md)
- **모션 설계**에서 포커스·선택·열기·닫기·재정렬·화면 전환에 필요한 움직임을 고른다. 빠른 재조작과 중단을 다루고, 축소 모션에서도 입력·포커스·최종 상태를 확인한다. [모션 지도](../internal/config/bootstrap/defaults/ui-design/references/motion.md)
- **직접 조작**에서 목록·칸반·파일·캔버스의 드래그 앤 드롭, 우클릭과 길게 누르기로 여는 메뉴·패널, 크기 조절과 스와이프를 다룬다. 버튼과 키보드 대체 경로, 취소, 포커스 복귀, 저장 실패 후 복구를 함께 정한다. [조작 패턴](../internal/config/bootstrap/defaults/ui-design/SKILL.md#read-what-changes-the-decision)
- **상태 피드백**에서 토스트·인라인 알림·배너 알림과 스켈레톤·진행 표시·빈 상태를 구분한다. 모든 토스트에는 즉시 닫을 수 있는 `×` 버튼을 둔다. [알림 지침](../internal/config/bootstrap/defaults/ui-design/references/components/notifications.md) · [로딩과 진행](../internal/config/bootstrap/defaults/ui-design/references/loading-and-progress.md)

화면 전체를 다룰 때는 레이아웃·시각 체계와 HTML/CSS/Vanilla JS 예제를 필요한 만큼 읽는다. 예제는 각 화면의 구성과 조작을 검토하는 자료다. 대상 프로젝트의 요구와 디자인 시스템을 적용한다. [참고 지도](../internal/config/bootstrap/defaults/ui-design/SKILL.md)

사용 가능한 경우 MECA 호스트에서 `browser_action`과 `browser_inspect`로 실제 화면과 조작을 확인한다. 추가 진단이 필요하거나 내장 도구를 사용할 수 없으면 Playwright CLI(`playwright-cli`)를 사용한다. 화면·콘솔·네트워크 확인과 레이아웃·모션 검사 방법은 [스킬의 브라우저 검증 지침](../internal/config/bootstrap/defaults/ui-design/references/browser-tools.md), 내장 도구 준비는 [브라우저 도구](browser.md)를 따른다.

## 백엔드 설계

`#backend-design`은 설계 단계부터 CPU·메모리·저장공간·I/O·네트워크·응답 지연과 정확성·복구 비용을 함께 검토하도록 안내한다.

- 테스트 가능한 구현은 **Red → Green → Refactor의 TDD**를 기본으로 한다. 요구·불변식에서 동작 테스트를 만들고 의도한 실패를 확인한 뒤 최소 구현과 리팩토링을 반복한다.
- SQLite·PostgreSQL·Redis를 부하·일관성·운영 비용에 맞춰 선택하고 스키마·인덱스·트랜잭션·캐시 수명을 연결한다.
- API 명령·조회·멱등성·스트림·작업·웹훅·테넌트 격리의 실행과 실패 경계를 정한다.
- 중앙 로거·오류 코드·메시지·이벤트·필드 정책과 각 도메인의 실패 처리를 구분한다.
- 암호화·키 수명·검색·벡터·메모리·I/O·과부하·복구 기법의 이점과 비용을 비교한다.

[실전 구성 지도](../internal/config/bootstrap/defaults/backend-design/references/compositions.md)는 기능 모듈·트랜잭션 명령·조회 모델·워크플로·파이프라인·외부 연동·질의/규칙 처리의 책임과 자원 수명을 연결한다. [설계 비교 사례](../internal/config/bootstrap/defaults/backend-design/references/design-studies.md)의 수치는 명시한 가정이며 실측 성능이 아니다. 신규 설계는 부하 가정과 비용 모델로, 기존 구현은 프로파일·부하 시험으로 선택을 검증한다.

## 애플리케이션 구현

`#application-best-dev-practices`는 사용자가 시작한 일이 화면·API·저장을 거쳐 어떤 결과로 끝나는지 설계하고 검증하도록 안내한다. 메신저·게시판·오프라인 현장 기록·AI 작업·예약 등 18개 앱 유형마다 **앱 가이드**, **Do & Don’t**, **구현 Q&A**가 있다. [앱 유형과 활용 사례](../internal/config/bootstrap/defaults/application-best-dev-practices/SKILL.md)

주된 사용자 과업에 맞는 가이드를 읽고, 다른 유형에서 필요한 기능을 함께 적용한다. 예를 들어 소모임 앱은 게시판의 대화, 캘린더의 일정, 예약의 정원 확정 규칙을 조합한다. 대표 서비스가 제공하는 기능을 참고하고, 결합한 기능의 정보 공개 범위, 변경 권한, 확정 시점과 복구 동작을 대상 앱에서 정한다.

## 자료 사용

참고 자료는 **본문 → 지도 → 선택 자료**의 최대 세 층으로 구성하며 필요한 파일만 읽는다. 기본 스킬의 설명 Markdown은 파일별 8KiB, 코드·미디어는 1MiB 이하이며 512KiB 이하를 권장한다. 이 작성 기준과 엔진의 참고 파일 페이지 한도는 구분한다.

외부 자료의 출처와 배포 조건은 각 스킬에 동봉한 고지·출처 문서를 따른다. 기본 스킬을 읽어도 외부 서비스·DB·개발 도구를 자동 설치하거나 실행하지 않는다. 실제 프로젝트의 요구·기존 시스템·보안 정책을 우선한다.
