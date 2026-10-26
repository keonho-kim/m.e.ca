#!/usr/bin/env bash
# Purpose: Validate the explicit release series and prepare a local dated tag without changing GitHub.
# Pattern: Release preparation boundary.
# Usage: Run from the repository root; stdout supplies GitHub Actions step outputs.
# Related: .release-version, devtools/go.mod, .github/workflows/release.yml, scripts/release/publish.sh.
set -euo pipefail

release_series=$(<.release-version)
if [[ ! "$release_series" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
    printf '.release-version must explicitly contain major.minor, for example 0.0.\n' >&2
    exit 1
fi
release_day=$(TZ=Asia/Seoul date +%Y%m%d)
release_tag="v${release_series}.${release_day}"
release_tool=$(go list -m -modfile=devtools/go.mod -f '{{ .Version }}' github.com/goreleaser/goreleaser/v2)
if [[ ! "$release_tool" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]]; then
    printf 'Pin the GoReleaser version in devtools/go.mod before preparing a release.\n' >&2
    exit 1
fi

release_previous=$(git rev-parse --verify --quiet "refs/tags/$release_tag" || true)
git tag --force "$release_tag" HEAD >/dev/null
printf 'tag=%s\nprevious=%s\ngoreleaser=%s\n' "$release_tag" "$release_previous" "$release_tool"
