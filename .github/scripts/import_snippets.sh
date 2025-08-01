#!/usr/bin/env bash

# Borrowed from rules_python

set -o errexit -o nounset -o pipefail

[[ -z "$1" ]] && { echo "Repository URL is empty"; exit 1; }

TARFILE="$1"

# Set by GH actions, see
# https://docs.github.com/en/actions/learn-github-actions/environment-variables#default-environment-variables
TAG=${GITHUB_REF_NAME}
VERSION=${TAG//v/}

# https://github.com/bazelbuild/bazel/issues/17124
INTEGRITY=$(openssl dgst -sha256 -binary "$TARFILE" | openssl base64 -A | sed 's/^/sha256-/')

cat << EOF
Add to your \`MODULE.bazel\` file:

\`\`\`starlark
bazel_dep(name = "com_cognitedata_bazel_snapshots", version = "${VERSION}")

archive_override(
    module_name = "com_cognitedata_bazel_snapshots",
    integrity = "${INTEGRITY}",
    urls = ["https://github.com/cognitedata/bazel-snapshots/releases/download/${TAG}/snapshots-${TAG}.tar"],
)

\`\`\`
EOF
