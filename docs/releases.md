# 릴리스

[← README](../README.md) · [개발 가이드](development.md)

## 버전 규칙

태그는 `v<major>.<minor>.<YYYYMMDD>` 형식을 사용한다. major·minor는 저장소 루트의 [.release-version](../.release-version)에 `0.0`처럼 직접 지정한다. 사용자가 명시적으로 요청한 경우에만 이 파일의 major·minor를 변경하며 자동 증가·태그에서의 추측·누락 시 기본값 대체는 하지 않는다.

자동 배포 날짜는 한국 시간(Asia/Seoul)의 실행 당일로 계산한다. 예를 들어 `.release-version`이 `0.0`이고 2026년 10월 26일 실행하면 `v0.0.20261026`이다. 같은 날짜의 후속 푸시는 **그 날짜의 태그와 배포 파일을 최신 검증 커밋으로 갱신**한다. 다른 날짜의 릴리스는 보존한다. 날짜를 명시한 별도 배포는 사용자의 지정값을 따르며 자동 배포는 항상 당일 날짜를 쓴다.

## 자동 배포

[Release 워크플로](../.github/workflows/release.yml)는 `main` 푸시 또는 `main`에서의 수동 실행으로 시작한다. 원본 저장소에서만 게시하며 동시에 한 실행만 진행한다. 실행 중인 배포를 새 푸시로 취소하지 않고 후속 실행을 대기시킨다.

1. 명시한 major·minor와 고정 GoReleaser 버전을 읽어 로컬 날짜 태그를 준비한다.
2. `go test ./...`, `go vet ./...`, `go mod verify`를 실행한다.
3. GoReleaser로 네 플랫폼을 빌드하고 SHA-256과 각 아카이브의 실행 파일·라이선스 고지를 검사한다.
4. 원격 `main`이 이미 더 최신 커밋이면 이번 게시를 생략한다. 준비 이후 태그가 다른 작업에 의해 바뀌었으면 오류로 중단한다.
5. 태그를 갱신하고 배포 파일과 체크섬을 게시한다. 새 릴리스는 초안에서 업로드를 마친 뒤 공개하고, 같은 날짜의 기존 릴리스는 파일을 교체한다.

빌드·검사·사전 확인 실패 시 원격 태그를 변경하지 않는다. 게시 중 업로드에 실패하면 작업을 실패로 표시한다. 이 경우 태그나 일부 파일이 갱신되었을 수 있으므로 Actions에서 실패 원인을 확인하고 같은 실행을 다시 시도한다. 모든 게시가 끝난 릴리스에는 네 아카이브와 `checksums.txt`가 있어야 한다.

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

## 다운로드와 설치

[최신 릴리스](https://github.com/keonho-kim/m.e.ca/releases/latest)에서 대상 아카이브와 `checksums.txt`를 같은 폴더에 받는다. 저장소가 비공개이면 접근 권한이 있는 GitHub 계정으로 로그인한다. OS·CPU를 감지해 최신 릴리스를 다운로드·검증·설치하는 명령은 [README의 설치 안내](../README.md#최신-릴리스로-설치하기)를 따른다.

```sh
# macOS: 받은 파일의 결과가 checksums.txt의 동일 파일 항목과 같은지 확인
shasum -a 256 meca_<version>_macos_arm64.tar.gz

# Linux
sha256sum meca_<version>_linux_amd64.tar.gz
```

`<version>`과 플랫폼 부분은 실제 파일명으로 바꾼다. 확인한 아카이브를 빈 폴더에 풀고 `./meca install`을 실행하면 `~/.local/bin/meca`, PATH 등록과 기본 설정이 준비된다. 삭제는 `meca uninstall`을 사용한다. 내려받은 아카이브와 압축을 푼 원본은 사용자가 보관하거나 지운다.

## 로컬 검증

저장소 루트에서 실행한다. 도구 버전은 [devtools/go.mod](../devtools/go.mod)에 고정되어 있다.

```sh
go tool -modfile=devtools/go.mod goreleaser check
go tool -modfile=devtools/go.mod goreleaser release --snapshot --clean
```

스냅샷은 GitHub에 게시하지 않으며 `dist/`에 검증용 배포 파일을 만든다. `--clean`은 기존 `dist/` 출력물을 지우므로 보관할 산출물은 먼저 다른 경로에 둔다. 일반 코드·보안 검사 외에 아카이브 내용·SHA-256·해당 호스트 실행 파일의 `--help`, 임시 HOME에서 install/uninstall을 확인한다. 교차 빌드 성공은 다른 OS에서 실제 실행한 결과와 구분한다.

## GitHub 게시

`main` 푸시 후 Actions의 Release 실행 결과를 확인한다. 수동으로 같은 절차를 다시 실행하려면 GitHub Actions 화면의 Run workflow에서 `main`을 고르거나 다음을 사용한다.

```sh
gh workflow run release.yml --ref main
```

GoReleaser 버전은 [devtools/go.mod](../devtools/go.mod)의 고정값을 사용한다. [.goreleaser.yaml](../.goreleaser.yaml)과 워크플로는 같은 [릴리스 안내](release-notes.md)를 사용한다. 대상 저장소는 `keonho-kim/m.e.ca`이며 게시 후 태그 커밋, 네 아카이브, 체크섬 파일을 확인한다. 인증정보를 설정·로그·셸 스크립트에 기록하지 않는다.
