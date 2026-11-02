#!/bin/sh
# Run only after acquiring the shared build lease and passing capacity admission.
set -eu
if [ "${PROSODY_BUILD_ADMITTED:-}" != "1" ]; then
  echo 'Acquire the build lease and pass capacity admission before setting PROSODY_BUILD_ADMITTED=1.' >&2
  exit 2
fi
exec python3 "$(dirname "$0")/test-postgres.py"
