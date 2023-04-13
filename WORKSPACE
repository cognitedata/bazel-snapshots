workspace(name = "com_cognitedata_bazel_snapshots")

load(":internal_deps.bzl", "snapshots_internal_deps")

# Fetch deps needed only locally for development
snapshots_internal_deps()

load("//snapshots:dependencies.bzl", "snapshots_deps")

# Fetch our "runtime" dependencies which users need as well
snapshots_deps()

load("//snapshots:repositories.bzl", "snapshots_register_toolchains")

snapshots_register_toolchains(
    name = "snapshots",
    from_source = True,
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

# gazelle:repository_macro snapshots/dependencies.bzl%go_dependencies
# gazelle:repository go_repository name=io_bazel_rules_go importpath=github.com/bazelbuild/rules_go
go_rules_dependencies()

go_register_toolchains(version = "1.20")

gazelle_dependencies()
