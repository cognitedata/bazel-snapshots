# Integrating with custom rules

It may be desirable to integrate snapshots with your custom rules, instead of using the `change_tracker` rule.
This directory contains an example of that. In the [deployable-filegroup](./rules/deployable-filegroup/) directory we define
a rule called `deployable_filegroup` which takes a single attribute, `srcs`, which is a list of files.
The rule creates an executable target which can be used to "deploy"(in this case just print the name) the list files provided in the
`srcs` attribute. In addition to this the rule implementation `_deployable_filegroup_impl` uses the `create_tracker_file`
to create a JSON file with the digest of `srcs`, and a reference to the exectuable target.


By doing it like this we can ensure that all targets generated with `deployable_filegroup` are tracked, and more importantly tracked with the same tracker tags and without any extra effort.

Running `bazel run snapshots -- diff --format=json --stderr-pretty $(pwd)/old_tracker.json` will show you how all the changes have the same tag, and
that they each have their own target in the run array.
