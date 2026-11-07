#!/bin/sh
# Purpose: Generate a synthetic whole-engine CPU profile as an explicit reproducible Go build input.
# Pattern: Offline benchmark orchestration with atomic publication and owned temporary resources.
# Usage: sh scripts/performance/pgo.sh bin/pgo/default.pgo; promote only after comparing -pgo=off builds.
# Related: cmd/meca/engine_workload_test.go, docs/development.md.
set -eu

if [ "$#" -ne 1 ] || [ -z "$1" ]; then
  printf 'Usage: sh scripts/performance/pgo.sh OUTPUT.pgo\n' >&2
  exit 2
fi

profile_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
case "$1" in
  /*) profile_target=$1 ;;
  *) profile_target=$profile_root/$1 ;;
esac
if [ -d "$profile_target" ]; then
  printf 'The profile destination must be a file.\n' >&2
  exit 2
fi
profile_parent=$(dirname -- "$profile_target")
mkdir -p -- "$profile_parent"
profile_temp=$(mktemp -d "$profile_parent/.meca-pgo.XXXXXXXX")
trap 'rm -rf -- "$profile_temp"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP

cd -- "$profile_root"
go test -trimpath -tags=godbus_no_tcp -pgo=off ./cmd/meca \
  -run '^$' -bench '^BenchmarkEngineWorkload$' \
  -benchtime="${MECA_PGO_BENCHTIME:-3s}" -count=1 \
  -cpuprofile "$profile_temp/workload.pprof" -o "$profile_temp/meca.test"

# The loopback model is an external service in production. Exclude its CPU
# samples while retaining the actual agent, storage, transport and rendering.
go tool pprof -proto -ignore='engineProvider|engineFrames|engineGeminiFrame' \
  "$profile_temp/meca.test" "$profile_temp/workload.pprof" > "$profile_temp/default.pgo"
go tool pprof -top -nodecount=20 "$profile_temp/default.pgo" >&2
chmod 0644 "$profile_temp/default.pgo"
mv -f -- "$profile_temp/default.pgo" "$profile_target"
printf 'PGO build input: %s\n' "$profile_target"
