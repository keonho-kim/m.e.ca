# LAT 제거

macOS · Linux / amd64 · arm64

### 변경 사항

- LAT(Look At This) 기능을 제거했습니다. 사전 계획·비평·최종 검토를 위한 별도 세션 없이 메인 에이전트가 작업을 수행합니다.
- `!lat` 명령, LAT 진행 표시·응답 구분선, 전용 추론 설정과 프롬프트를 제거했습니다.
- LAT 전용 메모리·압축 작업·저장 테이블과 사용되지 않는 코드·테스트를 정리했습니다.
- 요청 사용량의 소유권을 메인 실행으로 단순화하고 불필요한 공유 상태·메모리 할당을 줄였습니다.
- 일반 대화 압축·도구 실행·스트리밍은 유지하며 관련 문서를 현재 동작에 맞췄습니다.

### 업데이트 안내

- `~/.meca/config.toml`에 `[lat]` 또는 `[lat_compaction]` 설정이 있다면 해당 섹션을 삭제한 뒤 설치·업그레이드를 진행하세요. 지원하지 않는 설정은 오류로 처리합니다.
- 세션 저장 형식은 **schema 24**입니다. 이전 형식의 세션은 자동 변환하지 않으므로 새 대화를 시작해야 합니다. 기존 DB 파일은 보존됩니다.
- 사용하는 기본 프롬프트는 `system.md`·`development.md`·`compaction.md` 세 파일입니다. 기존 LAT 프롬프트 파일은 사용하지 않으며 자동 삭제하지 않습니다.

[설치](https://github.com/keonho-kim/m.e.ca#설치) · [사용 가이드](https://github.com/keonho-kim/m.e.ca/blob/main/docs/usage.md) · [설정 가이드](https://github.com/keonho-kim/m.e.ca/blob/main/docs/configuration.md)
