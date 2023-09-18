# Releasing Bazel-snapshots

To release a new version of Bazel-snapshots, create a new git tag in following semver versioning format, eks. `vx.y.z`.
The v prefix is necessary for being able to import it from a go.mod.
When a tag of this format is pushed to Github the release action will run and generate a new draft Github Release.
The draft can be promoted to a proper release at your convenience after you've verified that everything looks right in the draft.
