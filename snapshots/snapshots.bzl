"""Snapshot rules for incremental deploys."""

# load("@io_bazel_rules_docker//container:providers.bzl", "BundleInfo")
load("@bazel_skylib//lib:shell.bzl", "shell")

SNAPTOOL_ATTRS = {
    "_snaptool": attr.label(
        executable = True,
        default = Label("//snapshots/go/cmd/snaptool"),
        cfg = "host",
    ),
}

def create_tracker_file(ctx, inputs, run = [], tags = [], bundle_infos = [], suffix = ".tracker.json"):
    """Creates an output group with a tracker file.

    Equivalent to using
    the change_tracker rule. Useful for integrating in other rule sets.

    Args:
        ctx: context object
        inputs: files to track
        run: targets to execute when digest changes
        tags: tags for the tracker
        bundle_infos: BundleInfo objects to extract digests from
        suffix: suffix to add to label to create filename
    Returns:
        OutputGroupInfo with change_track_files.
    """
    tracker_file = ctx.actions.declare_file("{name}{suffix}".format(name = ctx.label.name, suffix = suffix))

    args = ctx.actions.args()
    args.add("digest")
    args.add(tracker_file, format = "--out=%s")
    args.add_all(run, format_each = "--run=%s")
    args.add_all(tags, format_each = "--tag=%s")

    for bundle_info in bundle_infos:
        # Simplified handling for image bundles: use only the blobsum files.
        # for images created with container_run_and_commit the blobsum doesn't seem to change, so this doesn't work
        for _, data in bundle_info.container_images.items():
            inputs.extend(data["blobsum"])

    args.add_all(inputs)

    ctx.actions.run(
        outputs = [tracker_file],
        inputs = inputs,
        executable = ctx.executable._snaptool,
        arguments = [args],
        progress_message = "Creating tracker",
        mnemonic = "ChangeTracker",
    )

    return OutputGroupInfo(change_track_files = depset([tracker_file]))

def _change_tracker_impl(ctx):
    track_files = []
    bundle_infos = []
    for dep in ctx.attr.deps:
        track_files.extend(dep.files.to_list())
        # if BundleInfo in dep:
        #     # Handle BundleInfos separately
        #     bundle_infos.append(dep[BundleInfo])
        # else:
        #     # Handle other targets by just adding all the files
        #     track_files.extend(dep.files.to_list())

    # unique track_files
    track_files = [x for i, x in enumerate(track_files) if i == track_files.index(x)]

    return [
        create_tracker_file(
            ctx,
            track_files,
            run = [target.label for target in ctx.attr.run],
            tags = ctx.attr.tracker_tags,
            bundle_infos = bundle_infos,
            suffix = ".json",
        ),
    ]

_change_tracker = rule(
    implementation = _change_tracker_impl,
    attrs = dict(SNAPTOOL_ATTRS, **{
        "run": attr.label_list(
            doc = "List of executable targets to run when there are changes to deps",
        ),
        "deps": attr.label_list(),
        "tracker_tags": attr.string_list(
            doc = "Tags for the tracker",
        ),
    }),
)

def change_tracker(name, **kwargs):
    _change_tracker(
        name = name,
        **kwargs
    )

def _snaptool_runner_impl(ctx):
    args = []
    args.extend(["--gcs-bucket", ctx.attr.bucket])
    args.extend(["--workspace-name", ctx.workspace_name])

    out_file = ctx.actions.declare_file(ctx.label.name + ".bash")
    substitutions = {
        "@@ARGS@@": shell.array_literal(args),
        "@@SNAPTOOL@@": ctx.executable.snaptool.short_path,
    }
    ctx.actions.expand_template(
        template = ctx.file._template,
        output = out_file,
        substitutions = substitutions,
        is_executable = True,
    )
    runfiles = ctx.runfiles(files = [
        ctx.executable.snaptool,
    ]).merge(
        ctx.attr.snaptool[DefaultInfo].default_runfiles,
    )
    return [
        DefaultInfo(
            files = depset([out_file]),
            runfiles = runfiles,
            executable = out_file,
        ),
    ]

_snaptool_runner = rule(
    implementation = _snaptool_runner_impl,
    attrs = {
        "snaptool": attr.label(
            default = "//snapshots/go/cmd/snaptool",
            cfg = "host",
            executable = True,
        ),
        "bucket": attr.string(
            mandatory = True,
            doc = "Name of the bucket to use",
        ),
        "_template": attr.label(
            default = "//snapshots:runner.tmpl.sh",
            allow_single_file = True,
        ),
    },
    executable = True,
)

def snaptool(name, bucket):
    runner_name = "{name}-runner".format(name = name)
    _snaptool_runner(
        name = runner_name,
        bucket = bucket,
        tags = ["manual"],
    )
    native.sh_binary(
        name = name,
        srcs = [runner_name],
        tags = ["manual"],
    )
