#!/bin/sh
# Purpose: Build the pinned browser sources into compressed executable embed inputs.
# Pattern: Reproducible source-build entry point.
# Usage: sh scripts/build-ui.sh before building cmd/meca.
# Related: webui/package.json, webui/bun.lock, webui/scripts/package.ts.
set -eu
meca_ui_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
if ! command -v bun >/dev/null 2>&1; then
    printf '소스 빌드에는 Bun 1.4.4 이상이 필요합니다. 배포 바이너리 실행에는 필요하지 않습니다.\n' >&2
    exit 1
fi
bun install --cwd "$meca_ui_root/webui" --frozen-lockfile --ignore-scripts
bun run --cwd "$meca_ui_root/webui" build
