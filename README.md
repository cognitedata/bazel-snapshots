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

## Demo

TBW.

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

### Build binaries from source

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

TBW: create a snapshot, make a change, diff against previous snapshot

### Integrating With Other Rules

TBW
### Remote Storage

TBW
### Using in Continous Deployment Jobs

TBW



## How It Works

Bazel Snapshots works by tracking Bazel targets (build artifacts, outputs), by creating a _digest_ of the output files.
This digest, together with some metadata such as a _label_ and _tags_ represents a _tracker_.
The data for all trackers in the Bazel project is collected together in a file called a _snapshot_, typically named after a code revision (e.g. a git revision).
Two snapshots can be _diff_-ed to find out which trackers have changed between the two snapshots.

Bazel Snapshots consists of the following parts:

 * A rule `change_tracker`: Used to create an arbitrary _change tracker_ for some Bazel target. This is a thin wrapper around the `create_tracker_file` function.
 * A skylark function `create_tracker_file`: Used to integrate with other rules, so that they output change trackers in addition to their primary output.
 * A tool `snapshots`: A CLI which is used to create, store, tag and create diffs between different snapshots.

### Change Trackers

Change trackers can be created by any rule by using the `create_change_tracker` Skylark function, or using the `change_tracker` rule to create arbitrary change trackers based on other rules.
Change trackers consist of a set of output targets to track, as well as a list called `run` of executables to run when the output targets change, and a list of `tags` which can be used to separate change trackers into categories.

```py
load("//build/rules/snapshots:snapshots.bzl", "change_tracker")

change_tracker(
    name = "my-tracker",
    deps = [
        ":my-target",  # some output target (or source file)
    ],
    run = [":deploy-my-target"],  # executable to run when my-target changes
    tracker_tags = ["notify-slack"],
)
```

In the above example, `:deploy-my-target` can be set up to be executed whenever `my-tracker` has changed.
`my-tracker` will be considered changed whenever the SHA256 of `my-target` changes.
The tracker's tag `notify-slack` is metadata which can be used e.g. to perform other actions, such as sending Slack notifications.
The `run` and `tracker_tags` fields are optional.

Change Trackers can be built individually:

```sh
$ bazel build //path/to:my-tracker --output_groups=change_track_files
Target //path/to:my-tracker up-to-date:
  bazel-bin/path/to/my-tracker.json
```

The contents of a Change Tracker will look something like this:

```json
{
    "digest": "deadbeef",  # sha256 of the tracked files
    "run": ["//path/to:deploy-my-target"],
    "tags": ["notify-slack"],
}
```

### Collecting Snapshots

The `snapshots` CLI has a special command `collect`, which is used to collect all the individual Change Trackers.
This is done effectively by building the whole workspace using the `change_track_files` output groups, while also recording the actions performed.
The command will then collect all the individual Change Trackers and create a JSON file called a _Snapshot_.

The `snapshots` CLI is typically "installed" in a Bazel workspace so that it can be invoked with `bazel run snapshots -- <arguments>`.

A snapshot of the current state of the workspace can be collected:

```sh
$ bazel run snapshots -- collect > my-snapshot.json
```

The snapshots are the basis for comparing the state of the workspace at different points in time (that is, different git commits).
A snapshot can easily be pushed to a Storage Bucket, using snapshot's `push` command.
The snapshots tool also allows _tagging_ a Snapshot, so that it can be fetched by that tag later on.

### Performing a diff

In order to get the Change Trackers which have changed between two snapshots, the snapshots tool offers a command `diff`.
If given only one snapshot, the `diff` command will internally run `collect` and diff the given snapshot against the current state of the workspace.
The diff is effectively a list of Change Trackers, augmented with the _type_ of change they have seen: _added_, _removed_ or _changed_.

The diff can be provided as a "pretty" human-readable table, as a plain list of labels or in JSON format.
Using the JSON format allows the greatest flexibility in how the result is interpreted.

```groovy
def diffStr = sh('bazel run snapshots -- diff --format=json deployed', returnStdout: true)
def diff = readJSON text: diffStr

// Get the changes with 'notify-slack' tag
def notifiable = diff.findAll { change -> change.tags.any { tag -> tag == 'notify-slack' } }
```

The `run` field in the change tracker does _not_ cause actions to automatically be executed – it's up to some outside system to actually invoke the commands.

## Installation

Snapshots can be installed either using pre-built binaries (recommended) or by building them from source.
For installation examples, see:

 * [examples/use-binaries](Use pre-built binaries)
 * [examples/build-from-source](Build from source)

## Useful Commands

```sh
# Create a Snapshot
# --bazel_stderr routes Bazel's stderr to stderr, for debugging.
$ bazel run snapshots -- collect [--bazel_stderr]

# Get some Snapshot
# 'deployed' in the example is a specific tag.
$ bazel run snapshots -- get deployed

# Diff two snapshots
# Assumes there exists a snapshot named after the master branch HEAD
$ bazel run snapshots -- diff deployed $(git rev-parse master)
```

