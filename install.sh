#!/bin/sh
# Purpose: Start MECA installation from its source checkout.
# Pattern: Thin command entry point.
# Usage: ./install.sh
# Related: cmd/meca/install.go, internal/commands/install/install.go, README.md.

set -eu

if [ "$#" -ne 0 ]; then
    printf '사용법: ./install.sh\n' >&2
    exit 2
fi

if ! command -v go >/dev/null 2>&1; then
    printf 'Go가 필요합니다. README의 지원 버전을 설치한 뒤 다시 실행하세요.\n' >&2
    exit 1
fi

meca_install_root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
cd "$meca_install_root"
exec go run ./cmd/meca install
