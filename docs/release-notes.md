# M.E.CA v0.0.20261120

- **웹 문서 조회** 기능을 추가했습니다. `web_fetch`는 HTML을 전용 Chromium에서 렌더링하고 본문을 Markdown 페이지로 전달합니다. 호스트에 Chrome이나 Chromium이 준비되면 솔로·팀, TUI·서버·비대화형 실행에서 사용할 수 있습니다. 다음 페이지는 결과 ID와 커서로 이어 읽습니다.
- **자동 문맥 압축**은 유효 컨텍스트 창의 75%에 도달하면 압축 가능한 이전 이력을 요약합니다. vLLM에서 다음 요청의 토큰 수를 확인할 수 있으면 해당 수치를 사용합니다. 압축 요약의 기본 출력 상한은 유효 창의 7%입니다. 기존 `[compaction]` 설정에 `reserve_tokens`가 있으면 해당 줄을 제거하세요.
- **스트림 복구**는 답변이 시작된 뒤 발생한 일시적 연결 오류에도 작동합니다. 재시도 중 기존 답변을 유지하고 새 내용부터 이어 보여줍니다. 형식이 잘못된 도구 호출은 검증 오류와 함께 모델에 돌려줘 교정합니다. Chat Completions에서 `stop`이나 `tool_calls`를 받은 뒤 연결이 정상 종료되면 응답과 최종 사용량을 확정합니다.
- **문맥과 저장 처리**에서 인코딩된 모델 요청을 재사용하고, 메타데이터를 스트리밍으로 읽으며, 필요한 SQLite 본문과 웹 관찰 페이지만 가져옵니다. 작업 중 아직 읽지 않은 관찰은 압축 뒤에도 이어 볼 수 있습니다.
- **기본 스킬**에 `#application-best-dev-practices`를 추가했습니다. 18개 앱 유형의 구현 흐름, 권한, 상태, 실패 복구를 다룹니다. `#ui-design`에는 컴포넌트 선택, 직접 조작, 모션, 로딩·오류 피드백 지침을 보강했습니다.
- **설정 초기화**가 작업 취소를 반영하고 기본 스킬을 저장할 때 디렉터리와 파일의 대상을 검사합니다.

[웹 문서 조회](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261120/docs/web-fetch.md) · [문맥 압축](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261120/docs/configuration.md#프롬프트와-compaction) · [Provider 복구](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261120/docs/usage.md#provider-스트림과-복구) · [기본 스킬](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261120/docs/builtin-skills.md)
