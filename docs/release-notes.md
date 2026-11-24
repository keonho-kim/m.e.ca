# M.E.CA v0.0.20261124

- **팀 최종 응답과 종료.** `leave_team`으로 동료의 퇴장이 확정된 뒤 남은 팀원의 최종 응답이 기다림에 묶이는 문제를 고쳤습니다. 응답을 먼저 게시하고 모든 팀원의 자원 정리를 기다려 요청을 종료합니다. 게시 뒤 정리가 실패하면 응답을 보존하고 실패 상태를 기록합니다. [팀 종료와 재개](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261124/docs/team.md#종료와-재개)
- **TUI 화면 갱신.** 팀원별 사용량과 모델 활동을 최신 상태로 합쳐 오래된 상태가 뒤늦게 표시되는 문제를 고쳤습니다. 상태만 바뀌는 경우 본문 전체를 다시 그리는 작업을 줄였으며, 취소와 저장이 겹쳐도 확정된 팀 응답을 표시합니다. [화면과 사용량](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261124/docs/usage.md#화면과-사용량)
- **WebUI 출력 복원.** 선택한 팀원이 사라지면 전체 보기로 돌아가 남은 응답을 표시합니다. 최종 응답과 서버가 확정한 본문을 이력 조회가 끝날 때까지 유지하고, 같은 서버·대화의 재연결에서도 확인 전의 완료 출력이 사라지지 않도록 보강했습니다. 복원 경로에도 초안 보유 한도를 적용합니다. [상태와 스트림 복원](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261124/docs/webui-api.md#상태와-스트림-복원)
- **도구 호출 전 추론 표시.** 응답이 native 도구 호출로 확정될 때 유효한 Thinking까지 빈 내용으로 덮어쓰던 문제를 고쳤습니다. [모델 스트림과 복구](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261124/docs/runtime.md#모델-스트림과-복구)
- **Astryx 0.7.17.** Core·Neutral·CLI를 함께 올리고 upstream의 입력·좁은 화면·클립보드·테마·Markdown 수정을 반영했습니다. MECA의 스트리밍·스크롤 패치를 새 버전에 맞춰 갱신하고 배포 라이선스 고지도 정리했습니다. [패치 계약](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261124/webui/patches/README.md)
- **번들과 검증.** 초기 gzip 허용 예산을 640KiB로 확대했습니다. 상속된 테마와 함께 로드하던 Neutral CSS를 제거해 비압축 자산 약 29.5KiB, 초기 gzip 804바이트를 줄였습니다. 로컬 빌드의 초기 gzip은 약 318.46KiB입니다. [WebUI 빌드와 검사](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261124/docs/development.md#내장-webui)

## 확인 범위와 남은 사항

macOS에서 Go 동작·race 검사와 Chromium·Firefox·WebKit의 관련 브라우저 검사를 수행했습니다. WSL 및 실기기 모바일에서의 직접 실행은 이번 검증에 포함하지 않았습니다. Astryx의 스타일·아이콘 권고와 상호 배타적인 테마 선택자 경고는 [개발 가이드](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261124/docs/development.md#astryx-검사와-알려진-경고)에 분류와 남은 검토 범위를 기록했습니다.
