"""Our "development" dependencies
Users should *not* need to install these. If users see a load()
statement from these, that's a bug in our distribution.
"""

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:utils.bzl", "maybe")

def snapshots_internal_deps():
    maybe(
        http_archive,
        name = "io_bazel_rules_go",
        sha256 = "f74c98d6df55217a36859c74b460e774abc0410a47cc100d822be34d5f990f16",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.47.1/rules_go-v0.47.1.zip",
            "https://github.com/bazelbuild/rules_go/releases/download/v0.47.1/rules_go-v0.47.1.zip",
        ],
    )

    maybe(
        http_archive,
        name = "bazel_gazelle",
        sha256 = "2c7545cc914a411e5c15c3d55a0a5ed34fbff62def0e1f0ab6ed14bda222c3cc",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.39.0/bazel-gazelle-v0.39.0.tar.gz",
            "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.39.0/bazel-gazelle-v0.39.0.tar.gz",
        ],
    )
