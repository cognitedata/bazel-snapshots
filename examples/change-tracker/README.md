# Change tracker
This directory contains an example of how to use the ```change_tracker``` rule.
The rule will generate a JSON file containing and output like this:
```
{"digest":"6244db3ebf208f21526de3e458d396e3802eb6903ff8d5911a0a90bb21837a2c","run":["//:deploy"],"tags":["deployable"]}
```

The digest is a hash of the ```deps``` attribute and run is an array containing the targets
specified in the ```run``` atribute. 

The tags comes from the ```tracker_tags``` attribute of the rule and can contain an arbitrary list of tags. These tags can be used for querying with jq or some 
other JSON tool after running the diff command ```bazel run snapshots -- diff --format=json --stderr-pretty $(pwd)/old_tracker.json```. 
This command will output a list of changes in the format below.
```
[
  {
    "digest": "6244db3ebf208f21526de3e458d396e3802eb6903ff8d5911a0a90bb21837a2c",
    "run": [
      "//:deploy"
    ],
    "tags": [
      "deployable"
    ],
    "label": "//:artifact_change_tracker",
    "change": "changed"
  }
]
```

When runnig ```bazel run snapshots -- collect``` the snapshots tool run a query against the output group with 
name ```change_track_files``` which is the output group created by ```change_tracker```. It will then aggregate all 
the files into a single JSON file containg a list of digests and run targets. The ouput will look like the JSON object below.

```
{
  "labels": {
    "//:artifact_change_tracker": {
      "digest": "6244db3ebf208f21526de3e458d396e3802eb6903ff8d5911a0a90bb21837a2c",
      "run": [
        "//:deploy"
      ],
      "tags": [
        "deployable"
      ]
    }
  }
}
```
