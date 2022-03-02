#!/usr/bin/env bash

# Borrowed from rules_python

set -o errexit -o nounset -o pipefail

# Set by GH actions, see
# https://docs.github.com/en/actions/learn-github-actions/environment-variables#default-environment-variables
TAG=${GITHUB_REF_NAME}
PREFIX="bazel-snapshots-${TAG:1}"
SHA=$(git archive --format=tar --prefix=${PREFIX}/ ${TAG} | gzip | shasum -a 256 | awk '{print $1}')

cat << EOF
WORKSPACE setup:
\`\`\`starlark
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
http_archive(
    name = "com_cognitedata_bazel_snapshots",
    sha256 = "${SHA}",
    strip_prefix = "${PREFIX}",
    url = "https://github.com/cognitedata/bazel-snapshots/archive/refs/tags/${TAG}.tar.gz",
)

load("@com_cognitedata_bazel_snapshots//:repo.bzl", "snapshots_repos")
snapshots_repos()
\`\`\`
EOF