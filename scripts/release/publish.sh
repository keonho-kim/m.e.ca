#!/usr/bin/env bash
# Purpose: Verify archive build policy before publishing current main under a same-day tag lease.
# Pattern: Release publication boundary.
# Usage: GitHub Actions supplies its commit, repository, token and prepared version outputs.
# Related: scripts/release/prepare.sh, .github/workflows/release.yml, .goreleaser.yaml, docs/release-notes.md.
set -euo pipefail

: "${GITHUB_REPOSITORY:?Repository is required}"
: "${GITHUB_SHA:?Build commit is required}"
: "${GITHUB_OUTPUT:?Step output file is required}"
: "${GH_TOKEN:?GitHub authentication is required}"
: "${MECA_RELEASE_TAG:?Prepared release tag is required}"
if [[ ! ${MECA_PREVIOUS_TAG+x} || ! "$MECA_RELEASE_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]{8}$ ]]; then
    printf 'A valid prepared release tag and previous tag state are required.\n' >&2
    exit 1
fi
release_tag=$MECA_RELEASE_TAG
release_commit=$(git rev-parse HEAD)
if [[ "$release_commit" != "$GITHUB_SHA" || $(git rev-parse "$release_tag^{commit}") != "$GITHUB_SHA" ]]; then
    printf 'Prepared tag and checkout must match the built commit.\n' >&2
    exit 1
fi

release_refs=$(git -c 'credential.helper=!gh auth git-credential' ls-remote origin refs/heads/main "refs/tags/$release_tag")
release_main=$(awk '$2 == "refs/heads/main" { print $1 }' <<< "$release_refs")
release_previous=$(awk -v ref="refs/tags/$release_tag" '$2 == ref { print $1 }' <<< "$release_refs")
if [[ "$release_main" != "$GITHUB_SHA" ]]; then
    printf 'A newer main commit is available; skipping this release.\n'
    printf 'published=false\n' >> "$GITHUB_OUTPUT"
    exit 0
fi
if [[ "$release_previous" != "$MECA_PREVIOUS_TAG" ]]; then
    printf 'The release tag changed after preparation; rerun against its current state.\n' >&2
    exit 1
fi

release_version=${release_tag#v}
release_archives=(
    "meca_${release_version}_linux_amd64.tar.gz"
    "meca_${release_version}_linux_arm64.tar.gz"
    "meca_${release_version}_macos_amd64.tar.gz"
    "meca_${release_version}_macos_arm64.tar.gz"
)
if [[ $(awk '{ print $2 }' dist/checksums.txt | LC_ALL=C sort) != $(printf '%s\n' "${release_archives[@]}" | LC_ALL=C sort) ]]; then
    printf 'Release checksums must describe exactly the four expected archives.\n' >&2
    exit 1
fi
(cd dist && sha256sum --check checksums.txt)
mkdir -p .cache
release_stage=$(mktemp -d .cache/release-verify.XXXXXX)
trap 'rm -rf -- "$release_stage"' EXIT
release_members=$(printf '%s\n' LICENSE NOTICE THIRD_PARTY_NOTICES.md meca | LC_ALL=C sort)
release_assets=(dist/checksums.txt)
for release_archive in "${release_archives[@]}"; do
    if [[ $(tar -tzf "dist/$release_archive" | LC_ALL=C sort) != "$release_members" ]]; then
        printf 'Release archive is missing its binary or license notices: %s\n' "$release_archive" >&2
        exit 1
    fi
    tar -xOzf "dist/$release_archive" meca > "$release_stage/meca"
    if ! release_metadata=$(go version -m "$release_stage/meca"); then
        printf 'Cannot read Go build metadata: %s\n' "$release_archive" >&2
        exit 1
    fi
    if ! awk '
        $1 == "build" && $2 ~ /^-tags=/ {
            count = split(substr($2, 7), tags, ",")
            for (i = 1; i <= count; i++) if (tags[i] == "godbus_no_tcp") found = 1
        }
        END { exit !found }
    ' <<< "$release_metadata"; then
        printf 'Release binary requires the godbus_no_tcp build tag: %s\n' "$release_archive" >&2
        exit 1
    fi
    release_assets+=("dist/$release_archive")
done

release_error=$release_stage/api-error
release_exists=true
if release_immutable=$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/$release_tag" --jq '.immutable' 2>"$release_error"); then
    if [[ "$release_immutable" != false ]]; then
        printf 'Daily release updates require a mutable GitHub release.\n' >&2
        exit 1
    fi
elif grep -Fq '(HTTP 404)' "$release_error"; then
    release_exists=false
else
    cat "$release_error" >&2
    exit 1
fi

git -c 'credential.helper=!gh auth git-credential' push \
    "--force-with-lease=refs/tags/$release_tag:$release_previous" origin "refs/tags/$release_tag:refs/tags/$release_tag"
if [[ "$release_exists" == false ]]; then
    gh release create "$release_tag" --draft --verify-tag --title "M.E.CA $release_tag" --notes-file docs/release-notes.md
fi
gh release upload "$release_tag" "${release_assets[@]}" --clobber
gh release edit "$release_tag" --draft=false --latest --title "M.E.CA $release_tag" --notes-file docs/release-notes.md
printf 'published=true\n' >> "$GITHUB_OUTPUT"
