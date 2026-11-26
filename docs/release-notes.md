# M.E.CA v0.0.20261127

- **개발 스킬 통합.** `#development`에서 설계 방향·디자인 패턴·앱 흐름·UI·백엔드·검증 가이드를 함께 사용할 수 있습니다. 싱글·팀이 공통 지침을 읽고, 팀 구현에서는 도메인 책임과 Spec 합의 자료를 추가로 활용합니다. [개발 스킬](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/builtin-skills.md)
- **개발 자료 검색.** `meca develop "검색어"`가 설치된 참고 자료의 관련 경로·줄 번호·절 제목을 최대 10건 반환합니다. 검색은 모델 연결 없이 실행되며, 에이전트도 기존 명령 도구에서 호출해 필요한 자료를 이어 읽을 수 있습니다. [CLI 검색](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/cli.md#개발-자료-검색)
- **실무 알고리즘·자료구조 가이드.** 반복 조회·변경 반영·만료 작업·대용량 처리·의존성·자원 배정·구간 집계·증분 파싱의 16개 사례를 제공합니다. 적용 조건과 구성 방법, 상태 소유권, 비용과 검증할 동작을 연결합니다. [실무 사례](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/internal/config/bootstrap/defaults/development/references/algorithms.md)
- **컨테이너 설치 제거와 복구.** `~/.meca`가 볼륨 마운트 지점이면 내부 데이터를 정리하고 빈 디렉터리를 남겨 제거를 완료합니다. 권한·읽기 전용 파일시스템·하위 마운트 오류에는 대상 경로와 복구 방법을 표시하며, 문제 해결 후 같은 명령으로 제거를 이어갈 수 있습니다. [삭제와 복구](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/cli.md#삭제)
- **서버 종료 처리.** 첫 HTTP 요청을 보내지 않은 연결을 종료 시 정리해 대기 연결로 인한 종료 지연을 줄입니다. [서버 실행](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261127/docs/server.md)
