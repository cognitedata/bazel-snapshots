"""This module implements the Snaptool CLI toolchain."""

SnaptoolInfo = provider(
    doc = "Information about how to invoke the Snaptool executable.",
    fields = {
        "binary": "Executable Snaptool CLI binary",
    },
)

def _snaptool_toolchain_impl(ctx):
    binary = ctx.executable.snaptool
    template_variables = platform_common.TemplateVariableInfo({
        "SNAPTOOL_BIN": binary.path,
    })
    default = DefaultInfo(
        files = depset([binary]),
        runfiles = ctx.runfiles(files = [binary]),
    )
    snaptool_info = SnaptoolInfo(binary = binary)
    toolchain_info = platform_common.ToolchainInfo(
        snaptool_info = snaptool_info,
        template_variables = template_variables,
        default = default,
    )
    return [
        default,
        toolchain_info,
        template_variables,
    ]

snaptool_toolchain = rule(
    implementation = _snaptool_toolchain_impl,
    attrs = {
        "snaptool": attr.label(
            doc = "Snaptool CLI executable.",
            mandatory = True,
            allow_single_file = True,
            executable = True,
            cfg = "exec",
        ),
    },
    doc = "Defines a Snaptool CLI toolchain. See: https://docs.bazel.build/versions/main/toolchains.html#defining-toolchains.",
)

def snaptool_register_toolchains(name = ""):
    native.register_toolchains("//build/toolchains/snaptool:snaptool_toolchain")
