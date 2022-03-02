#!/usr/bin/env bash

# Borrowed from rules_python

set -o errexit -o nounset -o pipefail

[[ -z "$1" ]] && { echo "Repository URL is empty"; exit 1; }

TARFILE="$1"

# Set by GH actions, see
# https://docs.github.com/en/actions/learn-github-actions/environment-variables#default-environment-variables
TAG=${GITHUB_REF_NAME}
SHA=$(shasum -a 256 $TARFILE | awk '{print $1}')

cat << EOF
WORKSPACE setup:
\`\`\`starlark
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
http_archive(
    name = "com_cognitedata_bazel_snapshots",
    sha256 = "${SHA}",
    url = "https://github.com/cognitedata/bazel-snapshots/releases/download/${TAG}/snapshots-${TAG}.tar",
)

load("@com_cognitedata_bazel_snapshots//:repo.bzl", "snapshots_repos")
snapshots_repos()
\`\`\`
EOF
