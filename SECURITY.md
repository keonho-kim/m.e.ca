# 보안 문제 제보

취약점의 악용 방법, 인증정보 또는 비공개 데이터를 공개 Issue에 포함하지 마세요.

## 비공개 제보

[저장소 Security → Advisories](https://github.com/keonho-kim/m.e.ca/security/advisories)에서 **Report a vulnerability**가 표시되면 해당 기능으로 제보합니다. 표시되지 않으면 [Issues](https://github.com/keonho-kim/m.e.ca/issues)에 상세 내용 없이 비공개 연락 방법만 요청하세요. [GitHub의 비공개 제보 안내](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/report-privately)를 참고할 수 있습니다.

비공개 제보에는 사용 버전·배포 종류(tui/full)·운영체제, 영향받는 기능과 최소 재현 절차를 포함합니다. 실제 API Key·서버 토큰·개인 대화·사용자 프로젝트 대신 비식별 예시를 사용하세요.

일반 오류와 기능 제안은 [기여 정책](CONTRIBUTING.md)을 따릅니다.

## 제품의 보안 범위

MECA는 호스트 사용자 권한으로 명령을 실행합니다. 프로젝트 폴더는 OS 격리 경계가 아니며 서버 인증도 명령을 샌드박스에 넣지 않습니다. [실행·승인 경계](docs/execution-boundary.md)와 [서버 인증](docs/server.md#인증과-네트워크)을 확인하세요.

자격 증명의 저장·백업·삭제 조건은 [설정 가이드](docs/configuration.md#api-key-암호화와-복구)가 소유합니다. 비밀값을 Git이나 공개 제보에 넣지 않습니다.
