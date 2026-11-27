# 기본 스킬 안내

[문서 목록](README.md) · [스킬 사용과 형식](skills.md)

tui와 full은 다음 기본 스킬을 함께 제공한다.

| 호출 | 용도 | 배포 원본 |
| --- | --- | --- |
| `#development` | 개발 방향·설계 패턴·알고리즘·자료구조·최적화와 앱·UI·백엔드 구현 및 검증 | [SKILL.md](../internal/config/bootstrap/defaults/development/SKILL.md) |
| `#create-skill` | 스킬 작성·수정과 참고 자료 구성 | [SKILL.md](../internal/config/bootstrap/defaults/create-skill/SKILL.md) |

## 배포와 초기화

일반 시작은 `~/.meca/skills/`의 누락된 배포 파일만 생성하고 기존 내용을 유지한다. 설치·업그레이드는 배포본에 포함된 기본 스킬 본문·참고 자료·스크립트·에셋을 현재 배포 내용으로 갱신한다. 배포 파일을 직접 수정한 내용도 이때 교체된다. 내용과 권한이 동일한 파일은 다시 쓰지 않는다.

배포 파일 목록 밖의 개인 스킬과 추가 자료는 유지한다. 생성 디렉터리는 0700, 새 파일과 갱신 파일은 0600이다. 관리 디렉터리는 실제 디렉터리여야 하며, 갱신 대상 파일이 심볼릭 링크나 비정규 파일이면 실패한다. 파일별 교체는 원자적으로 수행한다. [초기화 구현](../internal/config/bootstrap/skills.go) · [갱신 구현](../internal/config/bootstrap/refresh_skills.go)

개발 자료의 배포 경로는 `~/.meca/skills/development/`다. 기본 시스템 지침은 `meca:development`로 자료를 선택한다.

## 개발 방향과 참고 자료

`#development`는 사용자가 원하는 도움에 필요한 공학적 판단을 지원한다. 질문이나 작업에 맞는 증거를 선택하고, 책임·상태·효과·완료 조건과 관련된 가이드를 골라 읽는다. 지식을 설명하거나 비교하는 일, 사용자가 직접 적용할 방향을 제시하는 일, 구현을 맡아 수행하는 일의 범위는 대화에서 이해한 요청에 따른다. 싱글과 팀이 같은 본문을 사용하며, 공동 작업이 필요할 때 팀 참고 자료를 읽는다.

| 설계할 내용 | 담당 자료 |
| --- | --- |
| 책임 분리, 생성·구조·행동 패턴, 상태와 자원 소유권 | [설계 패턴](../internal/config/bootstrap/defaults/development/references/patterns.md) |
| DP·그래프·범위 질의·문자열 처리, 자료구조 선택, 측정과 최적화 | [알고리즘과 최적화](../internal/config/bootstrap/defaults/development/references/algorithms.md) |
| 앱별 사용자 흐름·레코드·권한·수락 조건 | [애플리케이션 흐름](../internal/config/bootstrap/defaults/development/references/applications.md) |
| 화면·컴포넌트·상호작용·접근성·모션과 브라우저 검증 | [UI 설계](../internal/config/bootstrap/defaults/development/references/ui.md) |
| DB·트랜잭션·API·진단·보안·검색·성능·복구 | [백엔드 설계](../internal/config/bootstrap/defaults/development/references/backend.md) |
| 팀의 도메인 경계·Spec 합의·통합 검증 | [팀 개발](../internal/config/bootstrap/defaults/development/references/team-workflow.md) |
| 요구 환경·기능·운영·수명·전달 경로의 검증 | [검증과 운영 가능성](../internal/config/bootstrap/defaults/development/references/serviceability.md) |

예약 기능이라면 앱 가이드에서 예약 식별과 정원 규칙을 정하고, UI 가이드에서 대기·확정·충돌 상태를, 백엔드 가이드에서 동시 요청·재시도·저장을 연결한다. 가용 시간 조회 비용이 크면 같은 정원·최신성 조건을 유지하면서 알고리즘과 자료구조를 선택한다. 마지막으로 사용자가 시작한 요청부터 표시되는 결과까지 검증한다.

앱 자료는 메신저·게시판·오프라인 기록·AI 작업·예약 등 18개 유형의 가이드·Do & Don’t·구현 Q&A를 포함한다. UI 자료는 컴포넌트·레이아웃·시각 체계·모션과 로컬 예제를 제공한다. 백엔드 자료는 트랜잭션 명령·조회 모델·워크플로 등 실제 구성과 장애·동시성 검증을 다룬다. 공통 개발 절차와 TDD는 스킬 본문에서 관리한다.

알고리즘 자료는 반복 ID 조회, 스냅샷 변경 반영, 만료 작업, 최근 구간 집계, 대용량 정렬, 의존 작업 실행, 자원 배정, 예산 내 선택, 가변 높이 목록, 증분 파싱 등 16개 실무 사례를 제공한다. 각 사례는 적용할 상황, 구성 방법, 상태 소유권·갱신 규칙, 비용과 검증할 동작을 연결한다. [실무 사례 선택](../internal/config/bootstrap/defaults/development/references/algorithms.md#start-from-the-observed-work)

## 시스템 지침과 읽기

기본 시스템 지침은 대화에서 요청한 결과와 범위를 이해하고 필요한 전문 지식을 선택하도록 안내한다. 본문은 `codemode` 안에서 `tools.load_skill({id: "meca:development"})`로 읽고, `tools.read_skill_file`에는 반환된 같은 ID와 스킬 루트 기준 참고 경로를 사용한다. 지침을 받은 뒤 필요한 작업을 결정하며 스킬의 전체 절차를 자동 실행하지 않는다. 명시한 `#development`는 첫 모델 판단 전에 읽으므로 제공된 본문을 재사용한다. 자동 선택과 지침 적용은 모델이 수행한다. [읽기와 문맥 계약](skills.md#사용자-호출과-호스트-함수)

개발 참고 자료는 `meca develop`으로 검색한다. 에이전트는 `tools.execute_command`로 검색하고 반환된 경로를 같은 스킬 ID의 `tools.read_skill_file`로 읽는다. [CLI 사용법과 실행 조건](cli.md#개발-자료-검색)

## 스킬 작성

`#create-skill 프로젝트 코드 리뷰용 스킬을 만들어줘`처럼 스킬의 작성·수정 작업에 적용한다. 이 스킬은 Agent Skills 표준에 맞춘 지침·최소 예시·참고 자료의 구성과 검증을 안내한다. 사용자가 경로를 지정하지 않으면 프로젝트 `.agents/skills/`에 작성하도록 안내한다. [배포 지침](../internal/config/bootstrap/defaults/create-skill/SKILL.md) · [파일 형식](skills.md#파일-형식)

## 배포 자료

개발 자료는 **본문 → 분야 가이드 → 관련 지도 → 선택 자료**의 최대 네 층으로 탐색한다. 각 링크는 구체적인 판단에 필요한 자료를 가리키며, 같은 요청에서는 읽은 내용을 재사용한다. 기본 스킬의 설명 Markdown은 파일별 8KiB, 코드·미디어는 1MiB 이하이며 512KiB 이하를 권장한다. 이 작성 기준과 엔진의 참고 파일 페이지 한도는 구분한다.

로컬 화면 예제·진단 스크립트와 라이선스 고지는 통합 스킬에 함께 배포한다. 브라우저의 설치와 제공 조건은 코어 도구가 소유한다. [브라우저 검증 지침](../internal/config/bootstrap/defaults/development/references/ui/browser-tools.md) · [출처와 고지](../internal/config/bootstrap/defaults/development/NOTICE.md)
