"""Snapshot rules for incremental deploys."""

load("@io_bazel_rules_docker//container:providers.bzl", "BundleInfo", "ImageInfo")
load("@bazel_skylib//lib:shell.bzl", "shell")

SNAPSHOTS_ATTRS = {
    "_snapshots": attr.label(
        executable = True,
        default = Label("@snapshots-bin//:snapshots"),
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
        # Simplified handling for image bundles: use only the digests of the Docker manifest files.
        # We tried with `blobsum` previous to `manifest_digest`,
        # but for images created with container_run_and_commit and install_pkgs the blobsum doesn't seem to change, so this didn't work.
        for _, data in bundle_info.container_images.items():
            print(data["manifest_digest"])
            inputs.append(data["manifest"])

    args.add_all(inputs)

    ctx.actions.run(
        outputs = [tracker_file],
        inputs = inputs,
        executable = ctx.executable._snapshots,
        arguments = [args],
        progress_message = "Creating tracker",
        mnemonic = "ChangeTracker",
    )

    return OutputGroupInfo(change_track_files = depset([tracker_file]))

def _change_tracker_impl(ctx):
    track_files = []
    bundle_infos = []
    for dep in ctx.attr.deps:
        if BundleInfo in dep:
            # Handle BundleInfos separately
            bundle_infos.append(dep[BundleInfo])
        elif ImageInfo in dep:
            # When passing a container_image as a dependency, use the Docker manifest digest
            # for tracking
            track_files.extend(dep[ImageInfo].container_parts["manifest_digest"])
        else:
            # Handle other targets by just adding all the files
            track_files.extend(dep.files.to_list())

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
    attrs = dict(SNAPSHOTS_ATTRS, **{
        "run": attr.label_list(
            doc = "List of executable targets to run when there are changes to deps",
        ),
        "deps": attr.label_list(allow_files = True),
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

def _snapshots_runner_impl(ctx):
    args = []
    args.extend(["--storage-url", ctx.attr.storage])

    out_file = ctx.actions.declare_file(ctx.label.name + ".bash")
    substitutions = {
        "@@ARGS@@": shell.array_literal(args),
        "@@SNAPSHOTS@@": ctx.executable.snapshots.short_path,
    }
    ctx.actions.expand_template(
        template = ctx.file._template,
        output = out_file,
        substitutions = substitutions,
        is_executable = True,
    )
    runfiles = ctx.runfiles(files = [
        ctx.executable.snapshots,
    ]).merge(
        ctx.attr.snapshots[DefaultInfo].default_runfiles,
    )
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
        "snapshots": attr.label(
            default = "@snapshots-bin//:snapshots",
            cfg = "host",
            executable = True,
            allow_single_file = True,
        ),
        "storage": attr.string(
            doc = "Full URL of the bucket",
        ),
        "_template": attr.label(
            default = "//snapshots:runner.tmpl.sh",
            allow_single_file = True,
        ),
    },
    executable = True,
)

def snapshots(name, **kwargs):
    runner_name = "{name}-runner".format(name = name)
    _snapshots_runner(
        name = runner_name,
        tags = ["manual"],
        **kwargs,
    )
    native.sh_binary(
        name = name,
        srcs = [runner_name],
        tags = ["manual"],
    )
