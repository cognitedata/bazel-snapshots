#!/usr/bin/env bash

set -euo pipefail

ARGS=@@ARGS@@
SNAPSHOTS=@@SNAPSHOTS@@

# args should go after the command to run
COMMAND=
if [[ $# -gt 0 && "$1" != -* ]]; then
    # If the first argument is not an option, treat it as the command
    COMMAND="$1"
    shift
fi

if [ $# -ne 0 ]; then
    ARGS+=("$@")
fi

"$SNAPSHOTS" "$COMMAND" "${ARGS[@]}"
