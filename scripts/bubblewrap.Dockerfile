# Purpose: Build the pinned static Linux Bubblewrap executable and its license.
# Pattern: Reproducible native dependency build stage.
# Usage: scripts/build-bwrap.sh invokes this Dockerfile for one target architecture.
# Related: scripts/build-bwrap.sh, internal/tools/local/process/sandbox/bundle/assets/README.md.
FROM alpine@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412 AS build
RUN apk add --no-cache build-base=0.5-r3 meson=1.8.1-r0 libcap-dev=2.78-r0 libcap-static=2.78-r0 linux-headers=6.14.2-r0 gcc=14.2.0-r6 g++=14.2.0-r6 binutils=2.44-r3 musl=1.2.5-r12 musl-dev=1.2.5-r12 samurai=1.2-r7 pkgconf=2.4.3-r0 python3=3.12.15-r0
COPY notices /notices
ENV SOURCE_DATE_EPOCH=0
WORKDIR /build
RUN wget -q -O source.tar.gz https://codeload.github.com/containers/bubblewrap/tar.gz/refs/tags/v0.13.0 \
 && echo 'caecc52894a20ad9c932af8cb2a2c1bfd251b2d085f16d6c4d0424624c6c3cf9  source.tar.gz' | sha256sum -c - \
 && tar -xzf source.tar.gz
RUN meson setup out bubblewrap-0.13.0 --buildtype=release -Dprefer_static=true -Dc_link_args=-static -Dselinux=disabled -Dman=disabled -Dtests=false -Dbash_completion=disabled -Dzsh_completion=disabled \
 && meson compile -C out \
 && strip out/bwrap \
 && mkdir /export \
 && cp out/bwrap /export/bwrap \
 && cat bubblewrap-0.13.0/COPYING /notices/musl.txt /notices/libcap.txt > /export/COPYING \
 && cp source.tar.gz /export/source.tar.gz
FROM scratch
COPY --from=build /export/ /
