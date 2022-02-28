# Snapshots

Snapshots is a mechanism for doing _incremental deploys_ with Bazel. It can be used to find out which targets have changed between two versions of a Bazel workspace.
For instance, a continuous deployment (CD) mechanism can make snapshots of what has been deployed, thus only deploy what's necessary.

More generally, it can be used to implement side effects for changes to targets in Bazel workspaces.

## Overview

Snapshots consists of the following parts:

 * A rule `change_tracker`: Used to create an arbitrary _change tracker_ for some Bazel target.
 * A skylark function `create_tracker_file`: Used to integrate with existing rules, so that they output change trackers in addition to their existing output.
 * A tool `snapshots`: A CLI which is used to create, push, tag and create diffs between different snapshots.

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

