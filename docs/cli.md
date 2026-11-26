# CLI

[문서 목록](README.md) · [TUI 사용](usage.md) · [설정](configuration.md) · [배포 파일](releases.md)

`meca`는 프로젝트 대화, 비대화형 작업, 개발 자료 검색과 설치 관리를 제공한다. tui·full에서 같은 명령을 사용하며 서버·내장 WebUI는 full에서 제공한다.

```sh
meca --help
```

## 프로젝트와 저장 대화

```sh
meca
meca /path/to/project
meca resume
meca /path/to/project --session SESSION_ID
```

경로를 생략하면 현재 디렉터리에서 시작한다. 프로젝트 경로는 실제 디렉터리로 확인하고 심볼릭 링크를 정규화한다. 일반 시작은 해당 프로젝트의 저장 대화를 보여주고, `resume`은 모든 프로젝트의 대화를 보여준다. `--session`은 선택한 프로젝트의 대화를 직접 연다.

모델은 TUI의 `!model` 또는 WebUI 설정에서 연결한다. 비대화형 실행은 저장된 세션 선택이나 전역 기본 모델을 사용한다. [모델 연결](model-connections.md) · [대화 보존](configuration.md#세션-저장과-복원)

## 비대화형 실행

```sh
meca /path/to/project --print "이 변경 사항을 검토해줘"
meca /path/to/project --mode json "요청한 파일을 만들어줘"
meca /path/to/project --session SESSION_ID --print "이어서 수정해줘"
git diff | meca --print "입력한 변경 사항을 검토해줘"
```

| 옵션 | 동작 |
| --- | --- |
| `--print` | 한 요청을 실행하고 최종 답변을 stdout에 출력 |
| `--mode json` | 순서 있는 실행 이벤트를 stdout에 JSONL로 출력 |
| `--session ID` | 저장 대화와 모델 선택을 복원해 요청 실행 |

명령 뒤의 텍스트와 파이프 stdin을 요청에 포함한다. 대화·도구 결과는 같은 세션 저장 경로를 사용한다. 기본 승인 모드에서 사용자 확인이 필요한 작업은 비대화형 실행에서 차단한다. 승인 모드는 저장된 설정을 따른다. [실행과 승인](execution-boundary.md#자동-승인과-파일-도구)

### 대화 내보내기

```sh
meca /path/to/project --session SESSION_ID --export > conversation.jsonl
```

선택한 세션의 시간순 원본 기록을 JSONL로 출력한다. `--session`이 필요하며 요청 텍스트와 `--diagnose`를 함께 지정할 수 없다. 긴 도구 출력의 수집 원본도 저장 기록을 통해 내보낸다. [이벤트 형식](diagnostics.md)

### 모델 연결 진단

```sh
meca /path/to/project --diagnose
meca /path/to/project --session SESSION_ID --diagnose
```

선택된 모델의 응답과 추론 동작을 진단한다. 요청 텍스트를 함께 지정할 수 없다. 진단은 실제 모델 연결을 사용하며 결과와 한도는 Provider의 응답에 따른다. [연결과 기본값](model-connections.md#연결과-기본값)

## 개발 자료 검색

```sh
meca develop "delayed task queue"
meca develop "incremental parsing"
meca develop --help
```

설치된 `development` 스킬의 참고 자료를 검색한다. 검색어는 하나의 문자열로 인용하거나 여러 단어로 전달할 수 있다. 가이드에서 사용하는 구체적인 영어 키워드 2~5개를 사용한다.

공백으로 구분한 키워드를 대소문자 구분 없이 문자 그대로 찾는다. 일치 단어 수를 우선하고 제목·경로의 일치 여부로 순위를 정한다. 파일별로 가장 관련된 절의 경로·줄 번호·제목과 일치 단어 수를 최대 10건 반환한다. 사용자 편집 내용은 다음 검색에 반영된다.

```text
references/algorithms/scheduling-and-streams.md:5: Delayed tasks and expiring entries [3/3 terms]
```

실행 파일은 `~/.local/bin/meca`, 자료와 검색 스크립트는 `~/.meca/skills/development/`에 설치된다. PATH에 명령이 없으면 `~/.local/bin/meca develop "검색어"`로 호출한다. 검색 스크립트는 `/bin/sh`가 읽으며 `awk`·`find`·`sort`를 사용한다. 실행 계정에 스킬 디렉터리 접근과 파일 읽기 권한이 필요하다.

검색은 모델 연결 없이 동작한다. 검색 중에는 설치·제거를 막는 공유 잠금을 보유한다. 취소 시 검색 프로세스 그룹과 잠금을 정리한다. 스킬이 준비되지 않았으면 `meca install`로 설치한다.

에이전트는 `load_skill`로 개발 지침을 읽고 기존 `execute_command`에서 이 명령을 호출한다. 결과의 경로를 골라 같은 스킬 ID의 `read_skill_file`로 본문을 읽는다. 명령 실행에는 현재 승인 정책이 적용된다. [개발 스킬 구성](builtin-skills.md#개발-방향과-참고-자료) · [도구 계약](tools.md#셸과-파일-작업)

프로젝트·공유 스킬 복사본은 해당 스킬의 실제 경로에서 `sh <skill-root>/scripts/search.sh "keywords"`를 실행한다.

## 설치·업데이트·삭제

### 설치

받은 실행 파일이 있는 폴더에서 실행한다.

```sh
./meca install
./meca install --no-shell
./meca install --ext browser --no-shell
```

`install`은 실행 파일·PATH·기본 설정·프롬프트·스킬·로컬 암호화 키 저장소를 준비한다. 현재 계정의 `~/.local/bin/meca`에 설치하며 API Key 저장 어댑터의 생성·저장·조회와 권한을 검사한다.

| 옵션 | 동작 |
| --- | --- |
| `--no-shell` | 설치 후 로그인 셸을 자동 시작하는 동작을 생략 |
| `--ext browser` | 기본 설치 후 브라우저 도구와 실행 환경 준비 |

대화형 설치에서 PATH 적용이 필요하면 로그인 셸을 연다. 자동화·비대화형 설치에서는 표시된 안내에 따라 새 터미널을 사용하거나 `exec "$SHELL" -l`로 반영한다. 브라우저 준비가 실패하면 오류와 완료된 기본 설치 상태를 확인한다. [브라우저 설치와 갱신](tools.md#설치와-갱신)

설치와 업그레이드는 설정·모델 연결·자격 증명·대화를 보존한다. 기본 프롬프트는 배포본으로 갱신하고, 스킬은 누락 파일을 생성하며 사용자 파일을 보존한다. [프롬프트 설정](configuration.md#프롬프트와-compaction) · [기본 스킬 배포](builtin-skills.md#배포와-초기화)

### 업데이트

```sh
meca upgrade
meca upgrade --tui
meca upgrade --full
```

실행 중인 MECA를 종료한 뒤 업데이트한다. 종류를 생략하면 현재 tui/full을 유지한다. 최신 정식 릴리스의 해당 플랫폼 아카이브를 받아 SHA-256을 검증하고 설치한다. 같은 날짜의 배포 파일도 현재 내용으로 적용한다.

GitHub 인증이 필요하면 `GH_TOKEN`, `GITHUB_TOKEN`, 로그인된 `gh`의 인증을 순서대로 사용한다. 업데이트를 실행할 수 없으면 맞는 아카이브를 받아 압축을 푼 폴더에서 `./meca install --no-shell`을 실행한다. [배포 파일](releases.md#배포-파일)

### 삭제

```sh
meca uninstall
```

설치 실행 파일·MECA의 PATH 등록·암호화 키를 포함한 `~/.meca`의 모든 데이터를 제거한다. 실행 중인 MECA를 종료하고 필요한 데이터를 먼저 백업한다. 다른 프로그램으로 교체된 실행 파일이나 수정된 PATH 등록은 오류로 알린다. [백업과 복구](configuration.md#api-key-암호화와-복구)

`~/.meca`가 볼륨 마운트 지점이면 내용 삭제를 확인하고 빈 디렉터리를 남긴다. 읽기 전용·권한·하위 마운트 때문에 제거가 중단되면 오류에 표시된 문제를 해결한 뒤 같은 명령을 다시 실행한다. 제거 중에는 일반 시작과 설치를 차단한다.

## 서버와 내장 WebUI

full 배포본에서 실행한다.

```sh
meca /path/to/project --server --ui
meca /path/to/project --server --protocol sse
meca /path/to/project --server --protocol socket
meca /path/to/project --server --protocol grpc
```

`--listen`, `--token-file`, TLS 인증서와 키, Host·Origin 허용 목록, `--concurrency` 등 연결·인증 옵션은 서버의 실행 계약을 따른다. [서버 기동과 옵션](server.md#기동) · [WebUI 접속](webui-design.md#사용-시작)

## 종료 상태

| 코드 | 의미 |
| --- | --- |
| `0` | 명령 완료 |
| `1` | 입력·설정·실행·정리 오류 |
| `130` | 취소로 종료 |

오류 안내는 stderr에 출력한다. `--mode json`과 `--export`의 stdout은 JSONL 소비에 사용한다. 취소와 함께 저장·정리 오류가 발생하면 해당 오류를 우선 보고한다. [오류 분류](diagnostics.md#중앙-소유권과-오류)
