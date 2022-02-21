workspace(name = "com_cognitedata_bazel_snapshots")

# Tell Gazelle to use @io_bazel_rules_docker as the external repository for rules_docker go packages
# gazelle:repository go_repository name=io_bazel_rules_docker importpath=github.com/bazelbuild/rules_docker
# gazelle:repository go_repository name=io_bazel_rules_go importpath=github.com/bazelbuild/rules_go

load(":repositories.bzl", "repositories")

repositories()

load(":deps.bzl", "dependencies")

dependencies()
