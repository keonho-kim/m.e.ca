#!/bin/sh
# Purpose: Install MECA from source with an existing or temporary verified Go toolchain.
# Pattern: Thin command entry point.
# Usage: ./install.sh
# Related: scripts/install/go.sh, cmd/meca/install.go, internal/commands/install/install.go, README.md.

set -eu

if [ "$#" -ne 0 ]; then
    printf '사용법: ./install.sh\n' >&2
    exit 2
fi

meca_install_root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
(
    cd "$meca_install_root"
    meca_go_stage=
    trap '[ -z "$meca_go_stage" ] || rm -rf -- "$meca_go_stage"' 0
    trap 'exit 129' HUP
    trap 'exit 130' INT
    trap 'exit 143' TERM

    if command -v go >/dev/null 2>&1; then
        meca_go_binary=$(command -v go)
    else
        . "$meca_install_root/scripts/install/go.sh"
        meca_bootstrap_go
    fi

    mkdir -p bin
    CGO_ENABLED=0 "$meca_go_binary" build -tags=godbus_no_tcp -trimpath -o bin/meca ./cmd/meca
)

exec "$meca_install_root/bin/meca" install
