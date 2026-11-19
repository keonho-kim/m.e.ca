# M.E.CA v0.0.20261119

- `./meca install --ext browser`로 MECA 본체·PATH·기본 설정을 설치한 뒤 브라우저 도구를 준비하고 실행을 확인합니다. 브라우저 준비에 실패해도 설치된 본체는 사용할 수 있습니다.
- `browser_action`의 `open`은 기본 모드와 All Mighty에서 HTTP(S) 주소를 승인 없이 엽니다. 외부 페이지에서 입력·클릭 등 상호작용을 할 때는 직접 승인이 필요합니다.
- 모델 문맥, SQLite, 브라우저, 서버 스트리밍, TUI·WebUI 등 12개 기능의 최적화를 정리하고 각각의 구현 전략과 담당 코드를 연결했습니다.

[브라우저 도구 사용법](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261119/docs/browser.md) · [기능별 최적화](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261119/docs/optimization.md)
