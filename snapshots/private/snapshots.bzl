"""Snapshot rules for incremental deploys."""

load("@bazel_skylib//lib:shell.bzl", "shell")

def create_tracker_file(ctx, inputs, run = [], tags = [], suffix = ".tracker.json"):
    """Creates an output group with a tracker file.

    Equivalent to using
    the change_tracker rule. Useful for integrating in other rule sets.

    Args:
        ctx: context object
        inputs: files to track
        run: targets to execute when digest changes
        tags: tags for the tracker
        suffix: suffix to add to label to create filename
    Returns:
        OutputGroupInfo with change_track_files.
    """
    snaptool = ctx.toolchains["@com_cognitedata_bazel_snapshots//snapshots:snaptool_toolchain_type"]
    tracker_file = ctx.actions.declare_file("{name}{suffix}".format(name = ctx.label.name, suffix = suffix))

    args = ctx.actions.args()
    args.add("digest")
    args.add(tracker_file, format = "--out=%s")
    args.add_all(run, format_each = "--run=%s")
    args.add_all(tags, format_each = "--tag=%s")
    args.add_all(inputs)

    ctx.actions.run(
        outputs = [tracker_file],
        inputs = inputs,
        executable = snaptool.snaptool_info.binary,
        arguments = [args],
        progress_message = "Creating tracker",
        mnemonic = "ChangeTracker",
    )

    return OutputGroupInfo(change_track_files = depset([tracker_file]))

def _change_tracker_impl(ctx):
    track_files = []
    for dep in ctx.attr.deps:
        track_files.extend(dep.files.to_list())

    # unique track_files
    track_files = [x for i, x in enumerate(track_files) if i == track_files.index(x)]

    return [
        create_tracker_file(
            ctx,
            track_files,
            run = [target.label for target in ctx.attr.run],
            tags = ctx.attr.tracker_tags,
            suffix = ".json",
        ),
    ]

_change_tracker = rule(
    implementation = _change_tracker_impl,
    attrs = {
        "run": attr.label_list(
            doc = "List of executable targets to run when there are changes to deps",
        ),
        "deps": attr.label_list(allow_files = True),
        "tracker_tags": attr.string_list(
            doc = "Tags for the tracker",
        ),
    },
    toolchains = [
        "@com_cognitedata_bazel_snapshots//snapshots:snaptool_toolchain_type",
    ],
)

def change_tracker(name, **kwargs):
    _change_tracker(
        name = name,
        **kwargs
    )

def _snapshots_runner_impl(ctx):
    snaptool = ctx.toolchains["@com_cognitedata_bazel_snapshots//snapshots:snaptool_toolchain_type"]

    args = []
    args.extend(["--storage-url", ctx.attr.storage])

    out_file = ctx.actions.declare_file(ctx.label.name + ".bash")
    substitutions = {
        "@@ARGS@@": shell.array_literal(args),
        "@@SNAPSHOTS@@": snaptool.snaptool_info.binary.short_path,
    }
    ctx.actions.expand_template(
        template = ctx.file._template,
        output = out_file,
        substitutions = substitutions,
        is_executable = True,
    )
    runfiles = ctx.runfiles(files = [
        snaptool.snaptool_info.binary,
    ]).merge(snaptool.default.default_runfiles)
    return [
        DefaultInfo(
            files = depset([out_file]),
            runfiles = runfiles,
            executable = out_file,
        ),
    ]

_snapshots_runner = rule(
    implementation = _snapshots_runner_impl,
    attrs = {
        "storage": attr.string(
            doc = "Full URL of the bucket",
        ),
        "_template": attr.label(
            default = "runner.tmpl.sh",
            allow_single_file = True,
        ),
    },
    toolchains = [
        "@com_cognitedata_bazel_snapshots//snapshots:snaptool_toolchain_type",
    ],
    executable = True,
)

def snapshots(name, **kwargs):
    runner_name = "{name}-runner".format(name = name)
    _snapshots_runner(
        name = runner_name,
        tags = ["manual"],
        **kwargs
    )
    native.sh_binary(
        name = name,
        srcs = [runner_name],
        tags = ["manual"],
    )
