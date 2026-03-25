<!-- Generated with Stardoc: http://skydoc.bazel.build -->



<a id="change_tracker"></a>

## change_tracker

<pre>
load("@com_cognitedata_bazel_snapshots//snapshots:defs.bzl", "change_tracker")

change_tracker(<a href="#change_tracker-name">name</a>, <a href="#change_tracker-kwargs">**kwargs</a>)
</pre>



**PARAMETERS**


| Name  | Description | Default Value |
| :------------- | :------------- | :------------- |
| <a id="change_tracker-name"></a>name |  <p align="center"> - </p>   |  none |
| <a id="change_tracker-kwargs"></a>kwargs |  <p align="center"> - </p>   |  none |


<a id="create_tracker_file"></a>

## create_tracker_file

<pre>
load("@com_cognitedata_bazel_snapshots//snapshots:defs.bzl", "create_tracker_file")

create_tracker_file(<a href="#create_tracker_file-ctx">ctx</a>, <a href="#create_tracker_file-inputs">inputs</a>, <a href="#create_tracker_file-run">run</a>, <a href="#create_tracker_file-tags">tags</a>, <a href="#create_tracker_file-suffix">suffix</a>)
</pre>

Creates an output group with a tracker file.

Equivalent to using
the change_tracker rule. Useful for integrating in other rule sets.


**PARAMETERS**


| Name  | Description | Default Value |
| :------------- | :------------- | :------------- |
| <a id="create_tracker_file-ctx"></a>ctx |  context object   |  none |
| <a id="create_tracker_file-inputs"></a>inputs |  files to track   |  none |
| <a id="create_tracker_file-run"></a>run |  targets to execute when digest changes   |  `[]` |
| <a id="create_tracker_file-tags"></a>tags |  tags for the tracker   |  `[]` |
| <a id="create_tracker_file-suffix"></a>suffix |  suffix to add to label to create filename   |  `".tracker.json"` |

**RETURNS**

OutputGroupInfo with change_track_files.


<a id="snapshots"></a>

## snapshots

<pre>
load("@com_cognitedata_bazel_snapshots//snapshots:defs.bzl", "snapshots")

snapshots(<a href="#snapshots-name">name</a>, <a href="#snapshots-kwargs">**kwargs</a>)
</pre>



**PARAMETERS**


| Name  | Description | Default Value |
| :------------- | :------------- | :------------- |
| <a id="snapshots-name"></a>name |  <p align="center"> - </p>   |  none |
| <a id="snapshots-kwargs"></a>kwargs |  <p align="center"> - </p>   |  none |


