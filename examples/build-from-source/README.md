# Using binaries

This example show how you can use the Snapshot tool by building it from source code. This is a little more involved
than using the prebuilt binaries, as more dependencies are required for the building. Building from sources depends on 
`gazelle`, `protobuf` and `rules_go`. `snapshots_deps()` and `snapshots_repos()` is used for setting up the depencies 
for building from sources and using the Snapshot tool. 
