# Embedded Bubblewrap build inputs

Linux release preparation runs `sh scripts/build-bwrap.sh amd64` and/or `arm64` before `GOOS=linux GOARCH=<arch> go build ./cmd/v`. Generated architecture directories contain the static Bubblewrap 0.13.0 executable and its LGPL license. These artifacts are ignored by Git; source checkouts must prepare them before distributing Linux builds. A build without its matching helper returns an explicit isolation-unavailable error instead of silently running a project command unrestricted or asking an end user to install a package.

The runtime extracts only its architecture under the private V state directory, checks the executable against the embedded SHA-256, and retains the accompanying license. No runtime download occurs. Linux namespaces still depend on kernel and host policy support.

Source: https://github.com/containers/bubblewrap/tree/v0.13.0
Archive SHA-256: `caecc52894a20ad9c932af8cb2a2c1bfd251b2d085f16d6c4d0424624c6c3cf9`.
The build copies corresponding source to `dist/bubblewrap-source/` for release distribution. Bubblewrap is LGPL-2.0-or-later; the static build also uses musl (MIT) and libcap (BSD-3-Clause). Distribute their notices and the rebuild inputs with a release. The pinned build recipe documents its container digest, package versions and disabled optional features.
