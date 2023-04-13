SnaptoolInfo = provider(
    doc = "Information about how to invoke the snaptool executable.",
    fields = {
        "binary": "Executable snaptool binary",
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
            doc = "A hermetically downloaded (or built) executable target for the target platform.",
            mandatory = True,
            allow_single_file = True,
            executable = True,
            cfg = "exec",
        ),
    },
    doc = "Defines a snaptool toolchain. See: https://docs.bazel.build/versions/main/toolchains.html#defining-toolchains.",
)
