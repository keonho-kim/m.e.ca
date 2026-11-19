# M.E.CA v0.0.20261119

- **All Mighty**에서 외부 전송, 프로젝트 밖 삭제, 브라우저 클릭·입력 등 승인 대상 도구 호출을 자동 승인합니다. 기본 모드에서는 사용자 승인을 요청합니다. 두 모드에서 보호 경로와 실행 대상을 검사합니다.
- `./meca install --ext browser`로 본체와 브라우저 도구를 한 번에 설치합니다. 브라우저와 `agent-browser`가 준비된 환경에서 솔로·팀 에이전트에 브라우저 도구와 사용 지침을 제공합니다.
- `browser_action`의 `open`은 기본 모드와 All Mighty에서 HTTP(S) 주소를 승인 없이 엽니다.
- 설치 셸 테스트의 네 케이스에 각각 실행 시간 제한을 부여하고 타임아웃 메시지를 명확히 했습니다.
- README에서 설치·업데이트 명령을 하나씩 복사할 수 있게 정리했습니다. 서버·스킬 안내를 현재 동작에 맞추고 기능별 최적화 문서를 구현에 연결했습니다.

[설치와 업데이트](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261119/README.md#설치) · [승인 모드](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261119/docs/execution-boundary.md#자동-승인과-파일-도구) · [브라우저 도구](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261119/docs/browser.md)
