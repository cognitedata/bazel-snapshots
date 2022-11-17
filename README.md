# Bazel Snapshots

Bazel Snapshots is a tool for finding the changed targets between two versions in a Bazel project.
It can be used to implement _incremental deployment_ – only re-deploying things that have changed – or to implement any other side effect of making a change which affects an output, such as sending notifications or interacting with pull requests.

Bazel Snapshots works by creating digests of outputs and recording them to files, which can be compared later.
By comparing two snapshots, we get a JSON structure containing the changed outputs, together with the metadata.
Implementing specific side-effects, such as deploying, is left for other tools.

Bazel Snapshots also has built-in support for storing snapshots and references to them remotely, so that they can be easily accessed and interacted with.

The way Bazel Snapshots works is in contrast to other approaches with similar goals, such as [https://github.com/Tinder/bazel-diff](bazel-diff), which analyses Bazel's graphs.
In short, Bazel Snapshots discovers which outputs have actually changed, whereas Bazel graph analysis methods discover which outputs could be affected by some change.
The main advantage with our approach is less over-reporting and more explicit control.

## Installation

### Use Pre-Built Binaries (recommended)

Add Bazel Snapshots to your `WORKSPACE` file.
See [Releases](https://github.com/cognitedata/bazel-snapshots/releases) for the specific snippet for the latest release.

```skylark
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
http_archive(
    name = "com_cognitedata_bazel_snapshots",
    sha256 = "...",
    url = "https://github.com/cognitedata/bazel-snapshots/releases/download/<VERSION>/snapshots-<VERSION>.tar",
)

load("@com_cognitedata_bazel_snapshots//:repo.bzl", "snapshots_repos")
snapshots_repos()
```

_NOTE:_ If you're using [rules_docker](https://github.com/bazelbuild/rules_docker), put `snapshots_repos()` later in the `WORKSPACE` file to avoid overriding.

Add the following to your _root_ `BUILD` file:

```
load("@com_cognitedata_bazel_snapshots//snapshots:snapshots.bzl", "snapshots")

snapshots(name = "snapshots")
```

You should now be able to run the Snapshots tool via Bazel:

```sh
$ bazel run snapshots
usage: snapshots <command> [args...]
# ...
```

### Build Binaries From Source

Requires rules_go and gazelle.
See [example](/examples/build-from-source).

## Getting Started

In order to use Bazel Snapshots, we first have to define trackers for the things we are interested in detecting changes on.

### Using The change_tracker Rule

Example: [change-tracker](/examples/change-tracker).

The `change_tracker` rule is a stand-alone rule defining a tracker.
You can use it to create trackers for existing targets.

```skylark
load("@com_cognitedata_bazel_snapshots//snapshots:snapshots.bzl", "snapshots", "change_tracker")

# A change_tracker
change_tracker(
    name = "my-change-tracker",
    deps = [
        # list of outputs and source files to track (required)
        "my-file.txt",
    ],
    run = [
        # list of executable targets to run when the tracked files have
        # changed (optional).
        # bazel-snapshots will not run these automatically; this only provides
        # hints to other tooling.
        "//:notify-slack",
    ],
    tracker_tags = [
        # list of "tags" for the tracker, useful for other tooling.
        "textfiles",
    ],
)

```

### Integrating With Other Rules

Example: [integrate-with-other-rules](/examples/integrate-with-other-rules/)

The `create_tracker_file()` Skylark function can be used to create a `OutputGroupInfo` which can be returned from any Bazel rule.
This technique can be used to create "transparent" support for Bazel Snapshots without using macros.
The tracker files can still be built separately using `bazel build //some:label --output_groups=change_track_files`.


### Remote Storage

So far, only Google Cloud Storage is supported for remote storage.
To start using a remote storage backend, add a `storage` attribute to `snapshots`
in your root BUILD file:

```skylark
snapshots(
    name = "snapshots",
    storage = "gcs://some-bucket/workspace-name",
)
```

Google Cloud Storage optionally takes `credential` and `project_id` query parameters in the storage URL.
If not set, the default credentials will be used and the project ID will be inferred.

Backend | Docs | Notes
---|---|---
Google Cloud Storage | [gcs](https://beyondstorage.io/docs/go-storage/services/gcs#storager) | `credential` and `project_id` defaults to `env`

Bazel Snapshots will create the following structure in the remote storage:

```
/
└── <workspace name>
    ├── snapshots
    │   ├── b1d4a4f.json  # snapshot files go here
    │   ├── abcd123.json  # (typically named by git commit)
    │   └── ...
    └── tags
        └── deployed      # a tag called "deployed"
```

_Snapshot files_ are JSON files containing the digests for all trackers in the Bazel project.
_Tag files_ emulate git tags, and can be referred to by name.
A tag file only contains the name of some snapshot file.

With remote storage, you can use these commands of the Snapshot tool:

 * `get`: get a snapshot from remote storage
 * `push`: push a snapshot to remote storage
 * `tag`: tag a remote snapshot

Usage example:

```sh
$ SNAPSHOT_NAME="$(git rev-parse --short HEAD)"

# Create a snapshot
$ bazel run snapshots -- collect --out "$SNAPSHOT_NAME.json"
snapshots: wrote file to /some-path/bcb0283.json

# Push the snapshot
$ bazel run snapshots -- push --name="$SNAPSHOT_NAME" --snapshot-path="$SNAPSHOT_NAME.json"

# Tag the snapshot
$ bazel run snapshots -- tag --name "$SNAPSHOT_NAME" latest
snapshots: tagged snapshot bcb0283 as latest: infrastructure/tags/latest

# Get or diff against the snapshot by name
$ bazel run snapshots -- get latest
$ bazel run snaptool -- diff latest
```


### Using in Continous Deployment Jobs

A minimal setup would have a deployment process (CD) which collects a snapshot and compares it with some already-known snapshot in order to find out which targets need to be re-deployed.
Re-deploying is often done by `bazel run`-ing some target, but the CD process could also determine this by itself.

Assuming there already exists some _tag_ called `deployed`, referring to some _snapshot_ representing the last set of deployed targets, we can use the `diff` command to both collect a snapshot and diff against the tag:

```sh
# Collect all trackers and diff against the snapshot tagged 'deployed'.
# Also output the collected snapshot to a file 'snapshot.json' and
# pretty-print a table of the detected changes to stderr.
$ bazel run snapshots -- diff --out snapshot.json --format=json --stderr-pretty deployed
```

The above command prints a JSON structure showing which targets have changed, along with their "run" labels and tags.
It's up to the CD process to interpret there results and run the necessary commands.

At the end of the CD process, we can push the snapshot we collected earlier and tag it as `deployed`, so that it will be used to diff against in the next CD process.

```sh
# Push the snapshot to remote storage
$ bazel run snapshots -- push --snapshot-path=snapshot.json

# Tag it as 'deployed'
bazel run snapshots -- tag deployed
```

## How It Works

Bazel Snapshots tracks Bazel targets (build artifacts, outputs) by creating a _digest_ of the output files.
This digest, together with some metadata such as a _label_ and _tags_ represents a _tracker_.
The data for all trackers in the Bazel project is collected together in a file called a _snapshot_, typically named after a code revision (e.g. a git revision).
Two snapshots can be _diff_-ed to find out which trackers have changed between the two snapshots.

Bazel Snapshots consists of the following parts:

 * A rule `change_tracker`: Used to create an arbitrary _change tracker_ for some Bazel target. This is a thin wrapper around the `create_tracker_file` function.
 * A skylark function `create_tracker_file`: Used to integrate with other rules, so that they output change trackers in addition to their primary output.
 * A tool `snapshots`: A CLI which is used to create, store, tag and create diffs between different snapshots.

### Change Trackers

You can specifically build the change trackers and see their contents using Bazel's `--output_groups` option:

```sh
$ bazel build //path/to:my-tracker --output_groups=change_track_files
Target //path/to:my-tracker up-to-date:
  bazel-bin/path/to/my-tracker.json
```

This can be useful for debugging purposes, i.e. if the digest isn't being changed as expected.
A tracker will typically look something like this:

```json
{
    "digest": "deadbeef",  // sha256 of the tracked files
    "run": ["//path/to:deploy-my-target"],
    "tags": ["notify-slack"],
}
```


