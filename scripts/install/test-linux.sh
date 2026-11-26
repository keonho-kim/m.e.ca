#!/bin/sh
# Purpose: Require real Linux mount and non-root permission coverage for uninstall.
# Pattern: Isolated executable integration runner using disposable containers.
# Usage: sh scripts/install/test-linux.sh from a host with Docker.
# Related: internal/commands/install/uninstall_linux_test.go, .github/workflows/release.yml.
set -eu

meca_install_test_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd -P)
meca_install_test_dir=$(mktemp -d "${TMPDIR:-/tmp}/meca-install-tests.XXXXXXXX")
trap 'rm -rf -- "$meca_install_test_dir"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP
meca_install_test_arch=$(docker version --format '{{.Server.Arch}}')
meca_install_test_image=debian@sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc
cd -- "$meca_install_test_root"
GOOS=linux GOARCH="$meca_install_test_arch" CGO_ENABLED=0 go test -c -tags=godbus_no_tcp \
    -o "$meca_install_test_dir/install.test" ./internal/commands/install
chmod 0755 "$meca_install_test_dir/install.test"

run_uninstall_cases() {
    meca_uninstall_cases=$1
    shift
    if ! docker run --rm --network none --read-only --cap-drop ALL \
        --security-opt no-new-privileges --tmpfs /tmp:rw,mode=1777 \
        --mount "type=bind,src=$meca_install_test_dir/install.test,dst=/install.test,readonly" \
        "$@" "$meca_install_test_image" /install.test -test.run "$meca_uninstall_cases" -test.v \
        > "$meca_install_test_dir/result.log" 2>&1; then
        cat "$meca_install_test_dir/result.log" >&2
        return 1
    fi
    cat "$meca_install_test_dir/result.log"
    if ! grep -q '^--- PASS: TestUninstall' "$meca_install_test_dir/result.log" || \
        grep -Eq '^[[:space:]]*--- SKIP:' "$meca_install_test_dir/result.log"; then
        printf 'Required uninstall tests did not run: %s\n' "$meca_uninstall_cases" >&2
        return 1
    fi
}

run_uninstall_cases '^TestUninstall(MountedStateAllowsReinstall|NestedMountRecovery|ReadOnlyStateRecovery)$' \
    --user 0:0 --cap-add SYS_ADMIN
run_uninstall_cases '^TestUninstallPermissionRecovery$' --user 1000:1000
