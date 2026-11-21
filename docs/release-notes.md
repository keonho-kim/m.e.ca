# M.E.CA v0.0.20261121

- **파일 편집·쓰기와 명령 실행**은 `edit_file`, `write_file`, `execute_command`의 필수 인자 `reason`에 작업 목적을 담습니다. `수정 후 실패 테스트 재확인`처럼 최대 64자의 한 줄 구절을 사용합니다. 도구 스키마는 이 인자를 먼저 제시하며, 백그라운드 명령의 완료 기록에도 같은 이유가 남습니다. [셸과 파일 작업](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261121/docs/tools.md#셸과-파일-작업)
- **도구 인자 교정**은 연속 오류에서 최대 두 번 시도합니다. 인자가 잘못된 호출에는 교정할 항목을, 같은 묶음의 다른 호출에는 실행 보류 사유를 반환합니다. 검증에 실패한 묶음은 실행 전에 보류되며, 호출별 결과를 기록합니다. 오류가 계속되면 마지막 오류와 실행 기록을 남깁니다. [모델 스트림과 복구](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261121/docs/runtime.md#모델-스트림과-복구)
- **브라우저와 WebFetch의 관찰 원문**은 유효한 다음 판단을 저장할 때까지 유지합니다. 인자 교정, 문맥 압축, 대화 복원 뒤에도 이어서 읽을 수 있습니다. [관찰 기록과 복구](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261121/docs/runtime.md#모델-스트림과-복구)
- **도구 요청 준비**에서 로컬 도구의 인자 파싱 결과를 검증·계획·준비에 재사용합니다. Claude와 Gemini 요청에서도 변경되지 않은 도구 스키마의 직렬화 결과를 재사용합니다. [모델 문맥 최적화](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261121/docs/optimization-strategies.md#모델-문맥과-provider)
