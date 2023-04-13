"""
Rule to make a deployable filegroup
"""

load("@com_cognitedata_bazel_snapshots//snapshots:defs.bzl", "create_tracker_file")

def _deployable_filegroup_impl(ctx):
    files = depset(ctx.files.srcs)
    runfiles = ctx.runfiles(files = ctx.files.srcs)

    deployment_script = ctx.actions.declare_file(ctx.label.name + "-deploy.bash")
    script_content = """#!/bin/bash
echo "deploying {paths}"
""".format(paths = " ".join([file.short_path for file in ctx.files.srcs]))

    ctx.actions.write(
        output = deployment_script,
        content = script_content,
        is_executable = True,
    )

    tracker = create_tracker_file(
        ctx,
        files,
        run = [ctx.label],
        tags = ["deployable_filegroup"],
    )

    return [
        DefaultInfo(
            executable = deployment_script,
            files = files,
            runfiles = runfiles,
        ),
        tracker,
    ]

deployable_filegroup = rule(
    implementation = _deployable_filegroup_impl,
    attrs = {
        "srcs": attr.label_list(allow_files = True),
    },
    toolchains = [
        "@com_cognitedata_bazel_snapshots//snapshots:snaptool_toolchain_type",
    ],
    executable = True,
)
