#!/bin/sh
# Run only after acquiring the shared build lease and passing capacity admission.
set -eu
if [ "${PROSODY_BUILD_ADMITTED:-}" != "1" ]; then
  echo 'Acquire the build lease and pass capacity admission before setting PROSODY_BUILD_ADMITTED=1.' >&2
  exit 2
fi
cd "$(dirname "$0")/.."
mkdir -p .local
fixture_dir=$(mktemp -d "$PWD/.local/pg-test.XXXXXX")
started=0
cleanup() {
  if [ "$started" = 1 ]; then pg_ctl -D "$fixture_dir/data" -m fast -w stop >/dev/null; fi
  # Retain log/data for diagnosis; no unrelated cluster is touched.
}
trap cleanup EXIT HUP INT TERM
initdb -D "$fixture_dir/data" --auth-local=trust --auth-host=reject --no-locale -E UTF8 >"$fixture_dir/init.log"
pg_ctl -D "$fixture_dir/data" -l "$fixture_dir/server.log" -o "-c listen_addresses='' -k $fixture_dir -p 55439 -c shared_buffers=16MB -c max_connections=10" -w start >/dev/null
started=1
createdb -h "$fixture_dir" -p 55439 prosody_test
export PROSODY_TEST_DATABASE_URL="host=$fixture_dir port=55439 dbname=prosody_test sslmode=disable"
export GOMAXPROCS=2 GOCACHE="$PWD/.local/gocache"
go test -p 1 ./internal/store -count=1 -timeout=90s
