# M.E.CA v0.0.20261119

- 브라우저와 `agent-browser`가 준비된 요청에만 솔로·팀 에이전트에 `browser_action`·`browser_inspect`와 사용 지침을 제공합니다.
- `./meca install --ext browser`는 MECA 본체·PATH·기본 설정을 설치한 뒤 브라우저 의존성을 준비하고 실행을 확인합니다. 브라우저 준비에 실패해도 설치된 본체는 사용할 수 있습니다.
- `browser_action`의 `open`은 기본 모드와 All Mighty에서 HTTP(S) 주소를 승인 없이 엽니다. 외부 페이지에서 클릭·입력 등 상호작용을 할 때는 직접 승인이 필요합니다.
- README를 간소화하고 설치·업데이트, 스킬 탐색과 페이지 조회, 서버·WebUI 연동 문서를 현재 동작에 맞췄습니다. 기능별 최적화와 구현 코드를 연결했습니다.

[브라우저 도구 사용법](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261119/docs/browser.md) · [기능별 최적화](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261119/docs/optimization.md)
