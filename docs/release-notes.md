# M.E.CA v0.0.20261126

- **API Key 로컬 암호화 저장.** API Key를 `~/.meca/credentials.db`에 AES-256-GCM으로 암호화하고, 버전별 암호화 키를 `credential-keys/keys.db`에 모아 보관합니다. macOS·Linux의 데스크톱 자격 증명 서비스에 접근할 수 없는 SSH·컨테이너 환경에서도 같은 저장 방식을 사용합니다. 디렉터리는 0700, 키 DB는 0600 권한을 적용합니다. [저장과 복구](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261126/docs/configuration.md#api-key-암호화와-복구)
- **설치 시 저장 기능 검증.** `meca install`이 로컬 키 디렉터리를 준비하고 실제 키 저장 어댑터로 원자적 생성·저장·조회를 확인합니다. 파일시스템이 필요한 저장 동작을 지원하지 않으면 설치 단계에서 오류를 알립니다. [다운로드와 설치](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261126/docs/releases.md#다운로드와-설치)
- **오류 안내와 중단 복구.** 파일 권한·소유권 문제, 암호화 키 누락, 저장 실패를 구분해 복구 방법을 안내합니다. 키 DB 쓰기 도중 프로세스가 종료되면 다음 접근에서 SQLite 저널을 복구합니다. 키 교체 후에도 이전 키 레코드를 보관해 과거 DB 백업을 복호화할 수 있습니다. [백업과 권한](https://github.com/keonho-kim/m.e.ca/blob/v0.0.20261126/docs/configuration.md#api-key-암호화와-복구)

**업데이트 후 API Key를 다시 입력하세요.** OS 자격 증명 저장소를 사용하던 버전에서 업데이트하면 `!model` 또는 WebUI 설정 → 모델에서 API Key를 입력해 새 저장소에 등록해야 합니다. 백업할 때는 MECA를 종료하고 `credentials.db`와 `credential-keys/keys.db`를 함께 보존하세요. 상태 디렉터리 전체를 읽을 수 있는 사용자나 프로세스는 API Key를 복호화할 수 있습니다.

macOS arm64에서 전체 Go 테스트·관련 race 검사·정적 검사·빌드를 수행했습니다. Linux arm64 컨테이너에서 일반 사용자 권한으로 설치·재설치·API Key 저장·복구를 확인했습니다. WSL과 Linux 데스크톱의 직접 실행은 이번 확인 범위에 포함하지 않았습니다.
