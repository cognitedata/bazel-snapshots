load("@com_cognitedata_bazel_snapshots//snapshots:dependencies.bzl", "go_dependencies")
load("@com_cognitedata_bazel_snapshots//snapshots:repositories.bzl", "snapshots_register_toolchains")

toolchains = tag_class(attrs = {
    "name": attr.string(
        doc = """\
    Base name for generated repositories, allowing more than one set of toolchains to be registered.
    Overriding the default is only permitted in the root module.
    """,
        default = "snapshots",
    ),
    "from_source": attr.bool(
        doc = "if True, will not fetch binaries and instead build the snapshots tool from source",
        default = True,
    ),
})

# buildifier: disable=unused-variable
def _snapshots_extension(module_ctx):
    go_dependencies()

    registrations = {}
    for mod in module_ctx.modules:
        for toolchains in mod.tags.toolchains:
            if toolchains.name != "snapshots" and not mod.is_root:
                fail("""\
                Only the root module may override the default name for the snapshots toolchains.
                This prevents conflicting registrations in the global namespace of external repos.
                """)

            if toolchains.name not in registrations:
                registrations[toolchains.name] = toolchains.from_source

    for name, from_source in registrations.items():
        snapshots_register_toolchains(name, from_source = from_source, register = False)

snapshots = module_extension(
    implementation = _snapshots_extension,
    tag_classes = {
        "toolchains": toolchains,
    },
)
