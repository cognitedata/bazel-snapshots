"""Defines the go dependencies for bazel-snapshots."""

load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_dependencies():
    """Go dependencies defined using Gazelle.

    Update using `bazel run gazelle-update-repos`.
    """

    go_repository(
        name = "com_github_spf13_pflag",
        build_file_proto_mode = "disable_global",
        importpath = "github.com/spf13/pflag",
        sum = "h1:iy+VFUOCP1a+8yFto/drg2CJ5u0yRoB7fZw3DKv/JXA=",
        version = "v1.0.5",
    )
