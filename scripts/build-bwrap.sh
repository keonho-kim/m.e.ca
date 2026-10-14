#!/bin/sh
# Purpose: Prepare the Linux sandbox binary embedded into a target V build.
# Pattern: Native artifact build orchestration.
# Usage: sh scripts/build-bwrap.sh amd64 (or arm64), then GOOS=linux GOARCH=... go build ./cmd/v.
# Related: scripts/bubblewrap.Dockerfile, internal/tools/local/process/sandbox/bundle/assets/README.md.
set -eu
case "${1:-}" in
  amd64|arm64) target_arch=$1 ;;
  *) echo 'usage: scripts/build-bwrap.sh amd64|arm64' >&2; exit 2 ;;
esac
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
build_output=$(mktemp -d)
trap 'rm -rf -- "$build_output"' EXIT HUP INT TERM
docker build --platform "linux/$target_arch" --file "$project_root/scripts/bubblewrap.Dockerfile" --output "type=local,dest=$build_output" "$project_root/scripts"
target_dir="$project_root/internal/tools/local/process/sandbox/bundle/assets/$target_arch"
mkdir -p "$target_dir"
cp "$build_output/bwrap" "$target_dir/bwrap"
cp "$build_output/COPYING" "$target_dir/COPYING"
# The exact corresponding source accompanies release preparation without being embedded.
source_dir="$project_root/dist/bubblewrap-source"
mkdir -p "$source_dir"
cp "$build_output/source.tar.gz" "$source_dir/bubblewrap-0.13.0.tar.gz"
