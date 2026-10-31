# 릴리스

[← README](../README.md) · [개발 가이드](development.md)

## 버전 규칙

태그는 `v<major>.<minor>.<YYYYMMDD>` 형식을 사용한다. major·minor는 저장소 루트의 [.release-version](../.release-version)에 `0.0`처럼 직접 지정한다. 사용자가 명시적으로 요청한 경우에만 이 파일의 major·minor를 변경하며 자동 증가·태그에서의 추측·누락 시 기본값 대체는 하지 않는다.

자동 배포 날짜는 한국 시간(Asia/Seoul)의 실행 당일로 계산한다. 같은 날짜의 후속 푸시는 **그 날짜의 태그와 배포 파일을 최신 검증 커밋으로 갱신**한다. 다른 날짜의 릴리스는 보존한다. 날짜를 명시한 별도 배포는 사용자의 지정값을 따르며 자동 배포는 항상 당일 날짜를 쓴다.

## 자동 배포

[Release 워크플로](../.github/workflows/release.yml)는 `main` 푸시 또는 `main`에서의 수동 실행으로 시작한다. 원본 저장소에서만 게시하며 동시에 한 실행만 진행한다. 실행 중인 배포를 새 푸시로 취소하지 않고 후속 실행을 대기시킨다.

1. 명시한 major·minor와 고정 GoReleaser 버전을 읽어 로컬 날짜 태그를 준비한다.
2. 검증 단계에 `GOFLAGS=-tags=godbus_no_tcp`를 설정하고 `go test ./...`, `go vet ./...`, `go mod verify`를 실행한다. 테스트의 자식 빌드에도 같은 태그를 전달한다.
3. GoReleaser의 명시적 `godbus_no_tcp` 태그로 네 플랫폼을 빌드한다. SHA-256·아카이브의 실행 파일·라이선스 고지를 검사하고 추출한 실행 파일의 `go version -m` 메타데이터에 정확한 태그 항목이 있는지 확인한다.
4. 원격 `main`이 이미 더 최신 커밋이면 이번 게시를 생략한다. 준비 이후 태그가 다른 작업에 의해 바뀌었으면 오류로 중단한다.
5. 태그를 갱신하고 배포 파일과 체크섬을 게시한다. 새 릴리스는 초안에서 업로드를 마친 뒤 공개하고, 같은 날짜에 게시된 릴리스는 파일을 교체한다.

빌드·검사·사전 확인 실패 시 원격 태그를 변경하지 않는다. 게시 중 업로드에 실패하면 작업을 실패로 표시한다. 이 경우 태그나 일부 파일이 갱신되었을 수 있으므로 Actions에서 실패 원인을 확인하고 같은 실행을 다시 시도한다. 모든 게시가 끝난 릴리스에는 네 아카이브와 `checksums.txt`가 있어야 한다.

게시 사전 검사는 타 플랫폼 실행 파일을 실행하지 않고 빌드 메타데이터만 읽는다. 메타데이터 판독 실패·태그 누락·비슷한 이름의 태그는 원격 태그 변경·업로드 전에 거절하며, 추출한 임시 파일은 성공·실패 모두 정리한다. 이 검사는 빌드 옵션 누락을 막으며 바이너리의 무결성 증명이나 의존성의 악성 행위 탐지를 대신하지 않는다.

기본 `GITHUB_TOKEN`의 `contents: write` 권한을 게시 작업에 부여하며 별도 PAT나 사용자 인증정보를 등록할 필요는 없다. 같은 날짜의 갱신을 위해 릴리스는 수정 가능해야 한다. GitHub의 immutable release가 적용되어 있으면 태그를 바꾸기 전에 오류로 중단한다.

## 배포 파일

[.goreleaser.yaml](../.goreleaser.yaml)이 네 가지 tar.gz와 `checksums.txt`를 생성한다.

| 파일 끝부분 | 대상 |
| --- | --- |
| `macos_arm64.tar.gz` | Apple Silicon Mac |
| `macos_amd64.tar.gz` | Intel Mac |
| `linux_arm64.tar.gz` | Linux arm64 |
| `linux_amd64.tar.gz` | Linux·WSL x86-64 |

각 아카이브에는 `meca` 실행 파일과 `LICENSE`·`NOTICE`·`THIRD_PARTY_NOTICES.md`가 들어 있다. M.E.CA는 Apache License 2.0을 따르며 외부 구성요소의 라이선스는 각 고지를 따른다. 실행·설치에는 Go나 개발 도구가 필요하지 않다. Linux 바이너리는 CGO 없이 빌드하며 macOS는 기본 OS 라이브러리를 사용한다. macOS 배포본에는 Apple Developer ID 서명·공증이 적용되어 있지 않다.

모든 배포본은 `godbus_no_tcp`를 적용한다. Linux DBus의 TCP·nonce-TCP 전송 구현을 제외하고 Unix 소켓과 macOS Keychain을 유지한다. 모델 API·셸의 네트워크 기능은 영향을 받지 않는다. 보장 범위와 수동 빌드 옵션은 [설정 가이드](configuration.md#linux-자격-증명-전송)를 따른다.

## 다운로드와 설치

[최신 릴리스](https://github.com/keonho-kim/m.e.ca/releases/latest)에서 OS·CPU에 맞는 아카이브를 내려받아 압축을 푼다. 압축을 푼 폴더에서 `./meca install`을 실행하면 된다. 공개 릴리스이므로 GitHub 로그인·토큰이 필요하지 않다. 파일 선택표와 터미널 명령은 [README의 설치 안내](../README.md#최신-릴리스로-설치하기)를 따른다.

설치하면 `~/.local/bin/meca`, PATH 등록과 기본 설정이 준비된다. 대화형 터미널에서 PATH 반영이 필요하면 설치 프로세스를 로그인 셸로 전환하며, 이 셸을 종료하면 설치를 호출한 셸로 돌아간다. PATH가 이미 반영되어 있거나 비대화형 실행이면 전환하지 않는다. 자동화에서는 `./meca install --no-shell`로 전환을 명시적으로 끌 수 있다. 삭제는 `meca uninstall`을 사용한다. 내려받은 아카이브와 압축을 푼 원본은 사용자가 보관하거나 지운다.

### 체크섬 확인 (선택)

파일의 무결성을 확인하려면 같은 릴리스의 `checksums.txt`도 다운로드한다. 압축을 풀기 전에 아래 명령의 결과를 `checksums.txt`의 동일 파일명 항목과 비교한다.

```sh
# macOS: 받은 파일의 결과가 checksums.txt의 동일 파일 항목과 같은지 확인
shasum -a 256 meca_<version>_macos_arm64.tar.gz

# Linux
sha256sum meca_<version>_linux_amd64.tar.gz
```

`<version>`과 플랫폼 부분은 실제 파일명으로 바꾼다.

## 소스 설치와 삭제

[README의 소스 설치](../README.md#소스에서-설치하기)는 저장소의 `install.sh`를 사용한다. 설치된 Go가 있으면 사용하고 없으면 go.mod의 최소 버전에 맞는 공식 macOS·Linux amd64·arm64 SDK를 고정 SHA-256으로 검증해 임시 사용한다. 임시 SDK는 저장소 `.cache`에 두고 빌드 성공·실패 모두 정리한다. 공용 Go 모듈·빌드 캐시와 시스템 Go는 유지한다.

소스 설치도 배포본과 같은 `godbus_no_tcp` 빌드 태그를 적용한다. 저장소 `bin/meca`를 만든 뒤 호출한 작업 폴더·환경에서 설치 명령을 실행한다. 설치 위치·기본 설정·PATH 등록과 로그인 셸 전환은 릴리스 바이너리의 install과 같다.

`meca uninstall`은 설치된 `~/.local/bin/meca`, MECA가 등록한 PATH 블록, `~/.meca` 전체와 저장한 OS 자격 증명을 제거한다. 실행 중인 에이전트를 종료한 뒤 사용한다. 사용자 프로젝트·다른 셸 설정·소스 저장소·공용 캐시는 유지하며 정리에 실패하면 원인을 반환한다.

## 업데이트

설치된 명령에서 `meca upgrade`를 실행하면 [GitHub Releases API](https://docs.github.com/en/rest/releases/releases#get-the-latest-release)로 최신 정식 릴리스를 조회한다. 현재 macOS·Linux의 amd64·arm64에 맞는 아카이브와 `checksums.txt`를 받아 SHA-256을 검증한다. 같은 날짜의 태그·배포 파일도 갱신할 수 있으므로 버전 문자열만으로 건너뛰지 않는다.

공개 릴리스는 인증 없이 조회·다운로드할 수 있다. 선택적으로 API 인증을 사용하는 경우 GH_TOKEN, GITHUB_TOKEN, 설치되어 있고 로그인된 gh의 github.com 토큰 순서로 선택한다. 토큰을 설정·세션·로그에 기록하지 않으며 다른 호스트로 리다이렉트할 때 인증 헤더를 제거한다. Go·gh·tar·curl은 업데이트의 필수 실행 의존성이 아니다.

임시 자료는 ~/.meca의 비공개 디렉터리에 저장하고 성공·오류·취소 시 정리한다. 메타데이터·체크섬은 각각 16MiB, 압축 파일·실행 파일은 각각 256MiB, 압축을 푼 전체 데이터는 512MiB로 제한한다. 파일명·SHA-256·실행 파일 엔트리를 확인한 뒤 검증된 바이너리의 install --no-shell을 실행한다. 아카이브의 링크·경로를 따라 외부 파일을 쓰지 않으며 다운로드·검증 실패 시 설치 상태를 변경하지 않는다. 일시적인 연결·읽기 오류와 HTTP 429·5xx는 최대 3회 시도하고 이후 원인을 반환한다.

설치는 실행 중인 MECA와 설치 작업의 충돌을 막는 전용 잠금과 원자적 바이너리 교체를 사용한다. 설정·모델 연결·인증정보·대화는 유지하고 system.md·development.md·compaction.md·lat.md·lat_compaction.md는 사용자 수정 여부와 관계없이 배포 기본본으로 교체한다. 바이너리가 이미 같아도 프롬프트 동기화는 수행하며 동일한 프롬프트 파일은 다시 쓰지 않는다. 일반 실행 중의 프롬프트 편집은 다음 설치·업데이트 전까지 유지한다.

## 로컬 검증

저장소 루트에서 실행한다. 도구 버전은 [devtools/go.mod](../devtools/go.mod)에 고정되어 있다.

```sh
go tool -modfile=devtools/go.mod goreleaser check
go tool -modfile=devtools/go.mod goreleaser release --snapshot --clean
```

스냅샷은 GitHub에 게시하지 않으며 `dist/`에 검증용 배포 파일을 만든다. `--clean`은 `dist/`의 출력물을 지우므로 보관할 산출물은 먼저 다른 경로에 둔다. 일반 코드·보안 검사 외에 아카이브 내용·SHA-256·해당 호스트 실행 파일의 `--help`, 임시 HOME에서 install/uninstall을 확인한다. 교차 빌드 성공은 다른 OS에서 실제 실행한 결과와 구분한다.

`go test ./scripts/release -run TestPublishRelease`는 모의 Git·GitHub 경계에서 네 아카이브의 태그 검사와 누락 시 게시 중단·임시 파일 정리를 확인한다. 실행 전 개발 가이드의 `GOFLAGS`를 설정한다. 실제 산출물의 빌드 메타데이터 검사는 게시 스크립트가 수행한다.

## GitHub 게시

`main` 푸시 후 Actions의 Release 실행 결과를 확인한다. 수동으로 같은 절차를 다시 실행하려면 GitHub Actions 화면의 Run workflow에서 `main`을 고르거나 다음을 사용한다.

```sh
gh workflow run release.yml --ref main
```

GoReleaser 버전은 [devtools/go.mod](../devtools/go.mod)의 고정값을 사용한다. [.goreleaser.yaml](../.goreleaser.yaml)과 워크플로는 같은 [릴리스 안내](release-notes.md)를 사용한다. 대상 저장소는 `keonho-kim/m.e.ca`이며 게시 후 태그 커밋, 네 아카이브, 체크섬 파일을 확인한다. 인증정보를 설정·로그·셸 스크립트에 기록하지 않는다.
