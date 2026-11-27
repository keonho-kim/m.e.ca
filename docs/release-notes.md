# M.E.CA v0.0.20261127

- **내장 Code Mode.** 모델이 `codemode`에 JavaScript·TypeScript 코드를 작성하고, 코드 안에서 도구 호출·반복·분기·결과 가공을 조합합니다. 실행기는 tui·full에 포함되어 Go·Node.js·TypeScript를 별도로 설치할 필요가 없습니다. [코드와 도구 계약](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/codemode.md)
- **실제 작업별 승인.** 스크립트 전체를 중복 승인하지 않고 내부 도구의 실제 인자·대상·셸 AST를 검사합니다. 승인 거부·취소·저장 실패 시 후속 작업을 중단하고, 이미 시작한 작업의 실제 결과를 기록합니다. [승인 경계](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/execution-boundary.md)
- **완료된 결과부터 후속 실행.** 독립 도구의 결과를 저장한 직후 해당 Promise를 재개합니다. 빠른 결과를 받은 후속 작업은 무관한 느린 호출을 기다리지 않으며, 진행 중인 작업과 파일·별칭·상하위 경로 충돌을 확인합니다. `Promise.all` 결과는 입력 순서를 유지합니다. [비동기 실행](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/codemode.md#승인과-실행)
- **메모리와 원문 저장 개선.** 불필요한 결과 문자열 복사와 도구 설명 재생성을 줄이고, 반환 JSON의 UTF-8 크기를 Go로 전달하기 전에 검사합니다. 반환값에 포함된 큰 도구 원문은 저장된 본문을 공유하며, 복원 시 길이·해시·JSON을 검증합니다. 초기화 중 취소와 불확실한 기록의 재시도도 보강했습니다. [자원과 기록](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/codemode.md#반환과-실행-기록)
- **코드와 내부 도구의 실행 표시.** TUI·WebUI에서 코드 실행과 내부 작업을 연결해 표시하고, 요청 원문·개별 결과를 확인할 수 있습니다. 완료 통계는 코드 실행 수와 실제 도구 실행 수를 구분합니다. [WebUI 사용](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/webui-design.md#작업-상태와-출력)
- **도구·저장 형식 변경.** 모델이 개별 도구를 직접 호출하는 계약을 `codemode` 단일 도구로 전환했습니다. 새 Code Mode 결과 저장 형식은 이전 실행 파일과 호환되지 않습니다.
- **개발 스킬 통합.** `#development`에서 설계 방향·디자인 패턴·앱 흐름·UI·백엔드·검증 가이드를 함께 사용할 수 있습니다. 싱글·팀이 공통 지침을 읽고, 팀 구현에서는 도메인 책임과 Spec 합의 자료를 추가로 활용합니다. [개발 스킬](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/builtin-skills.md)
- **개발 자료 검색.** `meca develop "검색어"`가 설치된 참고 자료의 관련 경로·줄 번호·절 제목을 최대 10건 반환합니다. 검색은 모델 연결 없이 실행되며, 에이전트도 기존 명령 도구에서 호출해 필요한 자료를 이어 읽을 수 있습니다. [CLI 검색](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/cli.md#개발-자료-검색)
- **실무 알고리즘·자료구조 가이드.** 반복 조회·변경 반영·만료 작업·대용량 처리·의존성·자원 배정·구간 집계·증분 파싱의 16개 사례를 제공합니다. 적용 조건과 구성 방법, 상태 소유권, 비용과 검증할 동작을 연결합니다. [실무 사례](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/internal/config/bootstrap/defaults/development/references/algorithms.md)
- **컨테이너 설치 제거와 복구.** `~/.meca`가 볼륨 마운트 지점이면 내부 데이터를 정리하고 빈 디렉터리를 남겨 제거를 완료합니다. 권한·읽기 전용 파일시스템·하위 마운트 오류에는 대상 경로와 복구 방법을 표시하며, 문제 해결 후 같은 명령으로 제거를 이어갈 수 있습니다. [삭제와 복구](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/cli.md#삭제)
- **서버 종료 처리.** 첫 HTTP 요청을 보내지 않은 연결을 종료 시 정리해 대기 연결로 인한 종료 지연을 줄입니다. [서버 실행](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/server.md)
