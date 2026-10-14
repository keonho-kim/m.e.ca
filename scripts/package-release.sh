#!/bin/sh
# Purpose: Package one already-built V target with dependency notices and required sandbox source.
# Pattern: Release archive assembly.
# Usage: sh scripts/package-release.sh v0.0.0-beta linux amd64 after building dist/build/linux-amd64/v.
# Related: .github/workflows/build.yml, scripts/build-bwrap.sh, README.md.
set -eu
if [ "$#" -ne 3 ]; then
  echo 'usage: scripts/package-release.sh <version> <linux|darwin> <amd64|arm64>' >&2
  exit 2
fi
version=$1
target_os=$2
target_arch=$3
case "$version" in ''|*[!A-Za-z0-9._-]*) echo 'invalid release version' >&2; exit 2 ;; esac
case "$target_os-$target_arch" in linux-amd64|linux-arm64|darwin-amd64|darwin-arm64) ;; *) echo 'unsupported release target' >&2; exit 2 ;; esac
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
binary="dist/build/$target_os-$target_arch/v"
test -f "$binary"
archive_name="v-code_${version}_${target_os}_${target_arch}"
staging=$(mktemp -d)
trap 'rm -rf -- "$staging"' EXIT HUP INT TERM
package="$staging/$archive_name"
mkdir -p "$package/licenses/go" "$package/licenses/modules"
cp "$binary" "$package/v"
chmod 755 "$package/v"
cp README.md "$package/README.md"
for notice in LICENSE LICENSE.md COPYING NOTICE; do
  if [ -f "$notice" ]; then cp "$notice" "$package/"; fi
done
go_root=$(go env GOROOT)
go_license="$go_root/LICENSE"
if [ ! -f "$go_license" ]; then go_license="$(dirname -- "$go_root")/LICENSE"; fi
cp "$go_license" "$package/licenses/go/LICENSE"
for source in "$go_root"/src/vendor/golang.org/x/*; do
  if [ -f "$source/LICENSE" ]; then
    destination="$package/licenses/go/$(basename -- "$source")"
    mkdir -p "$destination"
    cp "$source/LICENSE" "$destination/LICENSE"
  fi
done
go version -m "$binary" > "$package/BUILD.txt"
awk -v os="$target_os" -v arch="$target_arch" '$1 == "build" && $2 == "GOOS=" os {o=1} $1 == "build" && $2 == "GOARCH=" arch {a=1} $1 == "build" && $2 == "CGO_ENABLED=0" {c=1} END {exit !(o && a && c)}' "$package/BUILD.txt"
revision=$(git rev-parse HEAD)
printf '%s\n' "$revision" > "$package/COMMIT"
awk -v revision="$revision" '$1 == "build" && $2 == "vcs.revision=" revision {found=1} END {exit !found}' "$package/BUILD.txt"
if [ "${GITHUB_ACTIONS:-false}" = true ]; then
  awk '$1 == "build" && $2 == "vcs.modified=false" {found=1} END {exit !found}' "$package/BUILD.txt"
fi
GOOS="$target_os" GOARCH="$target_arch" CGO_ENABLED=0 go list -deps -f '{{with .Module}}{{if not .Main}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}{{end}}' ./cmd/v > "$staging/modules-unsorted"
LC_ALL=C sort -u "$staging/modules-unsorted" > "$staging/modules"
: > "$package/MODULES.txt"
while IFS='|' read -r module module_version source; do
  if [ -z "$module" ]; then continue; fi
  test -d "$source"
  printf '%s %s\n' "$module" "$module_version" >> "$package/MODULES.txt"
  name=$(printf '%s' "$module" | tr '/' '_')
  destination="$package/licenses/modules/$name"
  mkdir -p "$destination"
  found=false
  for notice in "$source"/LICENSE* "$source"/COPYING* "$source"/NOTICE* "$source"/COPYRIGHT* "$source"/AUTHORS* "$source"/LICENSE-MIT "$source"/license*; do
    if [ -f "$notice" ]; then cp "$notice" "$destination/"; found=true; fi
  done
  if [ "$found" != true ]; then
    echo "missing dependency license: $module" >&2
    exit 1
  fi
done < "$staging/modules"
test -s "$package/MODULES.txt"
if [ "$target_os" = linux ]; then
  assets="internal/tools/local/process/sandbox/bundle/assets/$target_arch"
  test -s "$assets/bwrap"
  test -s "$assets/COPYING"
  test -s dist/bubblewrap-source/bubblewrap-0.13.0.tar.gz
  mkdir -p "$package/licenses/bubblewrap/build"
  cp "$assets/COPYING" "$package/licenses/bubblewrap/COPYING"
  cp dist/bubblewrap-source/bubblewrap-0.13.0.tar.gz "$package/licenses/bubblewrap/"
  cp scripts/bubblewrap.Dockerfile scripts/build-bwrap.sh "$package/licenses/bubblewrap/build/"
  cp -R scripts/notices "$package/licenses/bubblewrap/build/"
fi
mkdir -p dist/release
COPYFILE_DISABLE=1 tar -czf "dist/release/$archive_name.tar.gz" -C "$staging" "$archive_name"
(cd dist/release && shasum -a 256 "$archive_name.tar.gz" > "$archive_name.tar.gz.sha256")
