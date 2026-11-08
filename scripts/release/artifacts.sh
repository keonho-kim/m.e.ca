#!/usr/bin/env bash
# Purpose: Verify local release archives, checksums, notices and required Go build settings.
# Pattern: Sourced offline artifact gate.
# Usage: publish.sh supplies a validated tag; the caller cleans release_stage and consumes release_assets after success.
# Related: scripts/release/publish.sh, scripts/release/publish_test.go, .goreleaser.yaml.

verify_release_artifacts() {
    local release_tag=$1
    local release_version=${release_tag#v}
    local release_archive release_metadata release_members
    local -a release_archives
    release_archives=(
        "meca_${release_version}_linux_amd64.tar.gz"
        "meca_${release_version}_linux_arm64.tar.gz"
        "meca_${release_version}_macos_amd64.tar.gz"
        "meca_${release_version}_macos_arm64.tar.gz"
    )
    if [[ $(awk '{ print $2 }' dist/checksums.txt | LC_ALL=C sort) != $(printf '%s\n' "${release_archives[@]}" | LC_ALL=C sort) ]]; then
        printf 'Release checksums must describe exactly the four expected archives.\n' >&2
        return 1
    fi
    (cd dist && sha256sum --check checksums.txt) || return
    mkdir -p .cache || return
    release_stage=$(mktemp -d .cache/release-verify.XXXXXX) || return
    release_members=$(printf '%s\n' LICENSE NOTICE THIRD_PARTY_NOTICES.md meca | LC_ALL=C sort)
    release_assets=(dist/checksums.txt)
    for release_archive in "${release_archives[@]}"; do
        if [[ $(tar -tzf "dist/$release_archive" | LC_ALL=C sort) != "$release_members" ]]; then
            printf 'Release archive is missing its binary or license notices: %s\n' "$release_archive" >&2
            return 1
        fi
        tar -xOzf "dist/$release_archive" meca > "$release_stage/meca" || return
        if ! release_metadata=$(go version -m "$release_stage/meca"); then
            printf 'Cannot read Go build metadata: %s\n' "$release_archive" >&2
            return 1
        fi
        if ! awk '
            $1 == "build" && $2 ~ /^-tags=/ {
                count = split(substr($2, 7), tags, ",")
                for (i = 1; i <= count; i++) if (tags[i] == "godbus_no_tcp") found = 1
            }
            END { exit !found }
        ' <<< "$release_metadata"; then
            printf 'Release binary requires the godbus_no_tcp build tag: %s\n' "$release_archive" >&2
            return 1
        fi
        release_assets+=("dist/$release_archive")
    done
}
