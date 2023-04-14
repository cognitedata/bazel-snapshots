TOOLCHAIN_TMPL = """\
toolchain(
    name = "toolchain",
    toolchain = "{toolchain}",
    toolchain_type = "{toolchain_type}",
)
"""

def _toolchains_repo_impl(repository_ctx):
    build_content = TOOLCHAIN_TMPL.format(
        name = repository_ctx.attr.name,
        toolchain_type = repository_ctx.attr.toolchain_type,
        toolchain = repository_ctx.attr.toolchain,
    )

    repository_ctx.file("BUILD.bazel", build_content)

toolchains_repo = repository_rule(
    _toolchains_repo_impl,
    doc = "Creates a repository with toolchain definitions for all known platforms which can be registered or selected.",
    attrs = {
        "toolchain": attr.string(doc = "Label of the toolchain. example; @snapshots_snaptool//:toolchain"),
        "toolchain_type": attr.string(doc = "Label of the toolchain_type. example; //snapshots:snaptool_toolchain_type"),
    },
)
