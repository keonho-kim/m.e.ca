#!/usr/bin/env bash
# Purpose: Publish verified current-main artifacts under the existing same-day tag lease.
# Pattern: Release publication boundary.
# Usage: GitHub Actions supplies its commit, repository, token and prepared version outputs.
# Related: scripts/release/prepare.sh, scripts/release/artifacts.sh, .github/workflows/release.yml, docs/release-notes.md.
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

release_stage=
trap '[ -z "$release_stage" ] || rm -rf -- "$release_stage"' EXIT
release_script_dir=${BASH_SOURCE[0]%/*}
if [[ "$release_script_dir" == "${BASH_SOURCE[0]}" ]]; then release_script_dir=.; fi
source "$release_script_dir/artifacts.sh"
verify_release_artifacts "$release_tag"

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
# Remove only the former names for this same tag, after all replacement files exist.
release_names=$(gh api "repos/$GITHUB_REPOSITORY/releases/tags/$release_tag" --jq '.assets[].name')
for release_platform in linux_amd64 linux_arm64 macos_amd64 macos_arm64; do
    release_legacy="meca_${release_tag#v}_${release_platform}.tar.gz"
    if grep -Fxq "$release_legacy" <<< "$release_names"; then
        gh release delete-asset "$release_tag" "$release_legacy" --yes
    fi
done
gh release edit "$release_tag" --draft=false --latest --title "M.E.CA $release_tag" --notes-file docs/release-notes.md
printf 'published=true\n' >> "$GITHUB_OUTPUT"
