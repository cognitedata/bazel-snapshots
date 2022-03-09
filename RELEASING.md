# Releasing Bazel-snapshots

To release a new version of Bazel-snapshots, create a new git tag in following semver versioning format, eks. `x.y.z`. When a tag of this format is pushed
to Github the release action will run and generate a new draft Github Release. The draft can be promoted to a proper release at your convenience after you've verified that everything looks right in the draft.
