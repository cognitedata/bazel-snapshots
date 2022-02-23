#!/usr/bin/env bash

set -euo pipefail

ARGS=@@ARGS@@
SNAPSHOTS=@@SNAPSHOTS@@

# args should go after the command to run
COMMAND="${1-}"

if [ $# -ne 0 ]; then
    shift
    ARGS+=("$@")
fi

"$SNAPSHOTS" "$COMMAND" "${ARGS[@]}"
