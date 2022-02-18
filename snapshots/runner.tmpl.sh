#!/usr/bin/env bash

set -euo pipefail

ARGS=@@ARGS@@
SNAPTOOL=@@SNAPTOOL@@

# args should go after the command to run
COMMAND="${1-}"

if [ $# -ne 0 ]; then
    shift
    ARGS+=("$@")
fi

"$SNAPTOOL" "$COMMAND" "${ARGS[@]}"
