#!/bin/sh
# Purpose: Download and verify a temporary official Go toolchain when Go is absent.
# Pattern: Installation bootstrap adapter.
# Usage: Sourced by install.sh; its owner removes meca_go_stage on every exit.
# Related: install.sh, scripts/install/go.sha256, go.mod.

meca_bootstrap_go() {
    case $(uname -s) in
        Darwin) meca_go_os=darwin ;;
        Linux) meca_go_os=linux ;;
        *) printf '자동 설치는 macOS와 Linux를 지원합니다.\n' >&2; return 1 ;;
    esac
    case $(uname -m) in
        arm64|aarch64) meca_go_arch=arm64 ;;
        x86_64|amd64) meca_go_arch=amd64 ;;
        *) printf '자동 설치는 arm64와 amd64를 지원합니다.\n' >&2; return 1 ;;
    esac
    for meca_go_command in curl tar awk mktemp; do
        if ! command -v "$meca_go_command" >/dev/null 2>&1; then
            printf 'Go 자동 설치에 %s 명령이 필요합니다.\n' "$meca_go_command" >&2
            return 1
        fi
    done
    if command -v sha256sum >/dev/null 2>&1; then
        meca_go_hash=sha256sum
    elif command -v shasum >/dev/null 2>&1; then
        meca_go_hash=shasum
    else
        printf '다운로드 검증에 sha256sum 또는 shasum이 필요합니다.\n' >&2
        return 1
    fi
    meca_go_version=$(awk '$1 == "go" { print $2; exit }' go.mod)
    meca_go_archive="go${meca_go_version}.${meca_go_os}-${meca_go_arch}.tar.gz"
    meca_go_expected=$(awk -v file="$meca_go_archive" '$2 == file { print $1; exit }' scripts/install/go.sha256)
    if [ ${#meca_go_expected} -ne 64 ]; then
        printf 'Go %s의 검증값이 없습니다. scripts/install/go.sha256을 확인하세요.\n' "$meca_go_version" >&2
        return 1
    fi
    mkdir -p .cache
    meca_go_stage=$(mktemp -d "$meca_install_root/.cache/go-install.XXXXXX")
    printf 'Go %s를 빌드용으로 임시 설치합니다 (%s/%s).\n' "$meca_go_version" "$meca_go_os" "$meca_go_arch"
    curl --fail --location --silent --show-error --proto '=https' --proto-redir '=https' \
        --retry 3 --connect-timeout 20 \
        "https://go.dev/dl/$meca_go_archive" -o "$meca_go_stage/go.tar.gz"
    if [ "$meca_go_hash" = sha256sum ]; then
        meca_go_actual=$(sha256sum "$meca_go_stage/go.tar.gz")
    else
        meca_go_actual=$(shasum -a 256 "$meca_go_stage/go.tar.gz")
    fi
    meca_go_actual=${meca_go_actual%% *}
    if [ "$meca_go_actual" != "$meca_go_expected" ]; then
        printf 'Go 다운로드의 SHA-256이 일치하지 않습니다. 설치를 중단합니다.\n' >&2
        return 1
    fi
    tar -xzf "$meca_go_stage/go.tar.gz" -C "$meca_go_stage"
    meca_go_binary="$meca_go_stage/go/bin/go"
    # An inherited GOROOT belongs to another installation, not this temporary SDK.
    unset GOROOT
}
