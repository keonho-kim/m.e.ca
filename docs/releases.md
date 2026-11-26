# 릴리스

[← README](../README.md) · [최신 릴리스](https://github.com/keonho-kim/m.e.ca/releases/latest) · [개발 가이드](development.md)

## 버전 규칙

태그는 `v<major>.<minor>.<YYYYMMDD>` 형식을 사용한다. major·minor는 사용자가 명시적으로 요청할 때 [.release-version](../.release-version)에서 변경한다. 자동 배포는 한국 시간(Asia/Seoul)의 실행 당일을 사용한다. 같은 날 다시 배포하면 당일 태그와 파일을 최신 검증 커밋 기준으로 갱신한다.

## 자동 배포

원본 저장소의 `main` 푸시 또는 수동 실행으로 시작한다. 후속 실행은 진행 중인 배포가 끝날 때까지 대기한다. [Release 워크플로](../.github/workflows/release.yml)

1. WebUI 빌드 전에 tui의 실행·설치·업데이트와 서버 의존성 제외를 확인한다.
2. full의 내장 UI를 빌드하고 프론트·Go 검사를 실행한다.
3. 두 종류의 아카이브를 빌드하고 SHA-256·파일 구성·실행 파일의 빌드 메타데이터를 검사한다. [아카이브 검증](../scripts/release/artifacts.sh)
4. 원격 `main`과 태그가 게시 조건을 만족하면 파일을 게시한다. 새 릴리스는 초안의 업로드가 완료되면 공개한다. 같은 날의 파일은 교체본 업로드를 확인한 후 정리한다. [게시 스크립트](../scripts/release/publish.sh)

사전 검사 실패는 원격 태그 변경 전에 중단된다. 게시 도중 실패하면 태그나 일부 파일이 바뀔 수 있으므로 Actions에서 원인을 확인한 뒤 다시 실행한다. 당일 갱신을 위해 수정 가능한 릴리스와 `GITHUB_TOKEN`의 `contents: write` 권한을 사용한다.

## 배포 파일

[.goreleaser.yaml](../.goreleaser.yaml)이 `meca_<version>_<variant>_<platform>.tar.gz` 여덟 개와 공용 `checksums.txt`를 생성한다.

| `<variant>` | 범위 |
| --- | --- |
| `tui` | TUI·CLI와 모든 Provider·도구·스킬·팀. 서버 어댑터와 WebUI 자산 제외 |
| `full` | tui의 기능과 SSE·WebSocket·gRPC 서버·내장 WebUI |

각 종류에 아래 네 플랫폼을 제공한다. 실행 파일 이름은 둘 다 `meca`이며 한 설치 위치와 상태 형식을 공유한다.

| 파일 끝부분 | 대상 |
| --- | --- |
| `macos_arm64.tar.gz` | Apple Silicon Mac |
| `macos_amd64.tar.gz` | Intel Mac |
| `linux_arm64.tar.gz` | Linux arm64 |
| `linux_amd64.tar.gz` | Linux·WSL x86-64 |

각 아카이브에는 `meca`와 라이선스 고지가 들어 있다. full은 `WEBUI_THIRD_PARTY_NOTICES.txt`도 포함한다. 모든 실행 파일은 `CGO_ENABLED=0`과 `godbus_no_tcp`로 빌드하며 tui에는 `tui` 빌드 태그를 추가한다. macOS 배포본에는 Apple Developer ID 서명·공증이 적용되어 있지 않다. [배포별 빌드 경계](development.md#배포-종류와-빌드-경계)

## 다운로드와 설치

운영체제와 CPU에 맞는 아카이브를 받아 압축을 푼 폴더에서 `./meca install`을 실행한다. `~/.local/bin/meca`와 PATH, 기본 설정 및 `~/.meca/credential-keys/`를 준비하고 실제 키 저장 어댑터로 저장·조회를 검증한다. API Key 저장에는 별도 OS 패키지 설치나 사용자 빌드 설정이 필요하지 않다. [최신 릴리스 다운로드](https://github.com/keonho-kim/m.e.ca/releases/latest)

대화형 설치에서 PATH를 새로 반영해야 하면 로그인 셸을 연다. 자동화에서는 `./meca install --no-shell`로 현재 셸을 유지할 수 있다. 브라우저 도구를 함께 설치하는 옵션은 도구 문서에서 확인한다. `--no-shell`과 브라우저 설치 옵션은 함께 사용할 수 있다. [브라우저 설치와 갱신](tools.md#설치와-갱신)

### 체크섬 확인 (선택)

파일의 무결성을 확인하려면 같은 릴리스의 `checksums.txt`도 다운로드한다. 압축을 풀기 전에 아래 명령의 결과를 `checksums.txt`의 동일 파일명 항목과 비교한다.

```sh
# macOS: 받은 파일의 결과가 checksums.txt의 동일 파일 항목과 같은지 확인
shasum -a 256 meca_<version>_tui_macos_arm64.tar.gz

# Linux
sha256sum meca_<version>_full_linux_amd64.tar.gz
```

`<version>`과 플랫폼 부분은 실제 파일명으로 바꾼다.

## 소스 설치와 삭제

저장소를 복제하고 `./install.sh`를 실행하면 full을 빌드해 설치한다. `./install.sh tui`는 TUI·CLI만 빌드한다. full 빌드는 Bun 1.4.4 이상이 필요하다. Go가 없으면 스크립트가 체크섬을 검증한 공식 SDK를 임시 사용한다. SDK 다운로드에는 `curl`·`tar`와 `sha256sum` 또는 `shasum`이 필요하다. [소스 빌드](development.md#소스-빌드)

```sh
git clone https://github.com/keonho-kim/m.e.ca.git
cd m.e.ca
./install.sh          # full
```

`meca uninstall`은 실행 파일·MECA의 PATH 등록·암호화 키를 포함한 `~/.meca` 전체를 제거한다. 실행 중인 MECA를 종료하고 필요한 데이터를 먼저 백업한다. 제거가 중단되면 `meca uninstall`을 다시 실행해 완료한다. [대화 복원](configuration.md#세션-저장과-복원) · [자격 증명 복구](configuration.md#api-key-암호화와-복구)

## 업데이트

실행 중인 MECA를 종료한 뒤 `meca upgrade`를 실행한다. 현재 tui/full 종류를 유지하며 `--tui` 또는 `--full`로 전환할 수 있다. 최신 정식 릴리스에서 맞는 아카이브를 받아 SHA-256을 검증한다. 같은 날짜의 배포라도 파일 변경을 적용한다. GitHub 인증은 선택 사항이며 필요할 경우 `GH_TOKEN`, `GITHUB_TOKEN`, 로그인된 `gh`의 인증을 순서대로 사용한다.

업데이트를 실행할 수 없거나 배포 파일을 찾지 못하면 최신 아카이브를 받아 압축을 푼 폴더에서 `./meca install --no-shell`을 실행한다. 설치와 업데이트는 설정·모델 연결·인증정보·대화를 보존하고 사용자 편집을 포함한 기본 프롬프트 파일을 배포본으로 갱신한다. [프롬프트 설정](configuration.md#프롬프트와-compaction)

OS 자격 증명 저장소를 사용하던 버전에서 로컬 암호화 저장 버전으로 업데이트할 때는 `!model` 또는 WebUI 설정 → 모델에서 API Key를 다시 입력한다. [API Key 저장과 복구](configuration.md#api-key-암호화와-복구)

## 로컬 검증

저장소 루트에서 실행한다. 도구 버전은 [devtools/go.mod](../devtools/go.mod)에 고정되어 있다.

```sh
go tool -modfile=devtools/go.mod goreleaser check
sh scripts/build-ui.sh
go tool -modfile=devtools/go.mod goreleaser release --snapshot --clean
```

스냅샷은 GitHub에 게시하지 않으며 `dist/`에 검증용 배포 파일을 만든다. `--clean`은 `dist/`의 출력물을 지우므로 보관할 산출물은 먼저 다른 경로에 둔다. 일반 코드·보안 검사 외에 아카이브 내용·SHA-256·해당 호스트 실행 파일의 `--help`, 임시 HOME에서 install/uninstall을 확인한다. 교차 빌드 성공은 다른 OS에서 실제 실행한 결과와 구분한다.

게시 경계의 실패·임시 파일 정리·교체 순서는 `go test ./scripts/release -run TestPublishRelease`로 확인한다. [검증 환경](development.md#변경-검증)

## GitHub 게시

`main` 푸시 후 Actions의 Release 실행 결과를 확인한다. 수동으로 같은 절차를 다시 실행하려면 GitHub Actions 화면의 Run workflow에서 `main`을 고르거나 다음을 사용한다.

```sh
gh workflow run release.yml --ref main
```

게시 설명은 [릴리스 노트](release-notes.md)에서 가져온다. 게시 후 대상 태그의 커밋과 tui/full 각 네 개의 아카이브, 공용 `checksums.txt`를 확인한다.
