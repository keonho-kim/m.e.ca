# 릴리스

[← README](../README.md) · [개발 가이드](development.md)

## 버전 규칙

태그는 `v<major>.<minor>.<YYYYMMDD>` 형식을 사용한다. 현재 접두사는 `v0.0`이며 예를 들어 2026년 10월 25일 배포는 `v0.0.20261025`다. 날짜 앞의 major·minor는 사용자가 명시적으로 요청한 경우에만 변경한다. 날짜를 명시하지 않으면 사용자 시간대의 배포 당일 날짜로 매번 갱신하며 이전 릴리스 날짜를 그대로 재사용하지 않는다. 같은 날짜의 태그가 이미 있으면 덮어쓰거나 이동하지 않고 새 버전 결정을 사용자에게 확인한다.

## 배포 파일

[.goreleaser.yaml](../.goreleaser.yaml)이 네 가지 tar.gz와 `checksums.txt`를 생성한다.

| 파일 끝부분 | 대상 |
| --- | --- |
| `macos_arm64.tar.gz` | Apple Silicon Mac |
| `macos_amd64.tar.gz` | Intel Mac |
| `linux_arm64.tar.gz` | Linux arm64 |
| `linux_amd64.tar.gz` | Linux·WSL x86-64 |

각 아카이브에는 `meca` 실행 파일과 `THIRD_PARTY_NOTICES.md`가 들어 있다. 실행·설치에는 Go나 개발 도구가 필요하지 않다. Linux 바이너리는 CGO 없이 빌드하며 macOS는 기본 OS 라이브러리를 사용한다. macOS 배포본에는 Apple Developer ID 서명·공증이 적용되어 있지 않다.

## 다운로드와 설치

[GitHub Releases](https://github.com/keonho-kim/m.e.ca/releases)에서 대상 아카이브와 `checksums.txt`를 같은 폴더에 받는다. 저장소가 비공개이면 접근 권한이 있는 GitHub 계정으로 로그인한다.

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

사용자가 게시를 요청한 경우 검증한 변경을 커밋하고 같은 커밋에 날짜 태그를 만든다. 커밋과 태그를 원격에 올린 후 해당 태그가 선택된 깨끗한 작업 트리에서 실행한다.

```sh
go tool -modfile=devtools/go.mod goreleaser release --clean
```

GitHub 저장소의 릴리스 작성 권한을 가진 `GITHUB_TOKEN`을 실행 환경으로 제공한다. 키는 설정·로그·셸 스크립트에 기록하지 않는다. 설정의 저장소는 `keonho-kim/m.e.ca`이며 게시 후 태그 커밋, 네 아카이브, 체크섬 파일을 확인한다. 배포는 명시적인 로컬 실행으로 수행하며 자동 CI는 구성하지 않는다.
