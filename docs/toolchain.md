<!-- Generated with Stardoc: http://skydoc.bazel.build -->



<a id="snaptool_toolchain"></a>

## snaptool_toolchain

<pre>
load("@com_cognitedata_bazel_snapshots//snapshots:toolchain.bzl", "snaptool_toolchain")

snaptool_toolchain(<a href="#snaptool_toolchain-name">name</a>, <a href="#snaptool_toolchain-snaptool">snaptool</a>)
</pre>

Defines a snaptool toolchain. See: https://docs.bazel.build/versions/main/toolchains.html#defining-toolchains.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="snaptool_toolchain-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="snaptool_toolchain-snaptool"></a>snaptool |  A hermetically downloaded (or built) executable target for the target platform.   | <a href="https://bazel.build/concepts/labels">Label</a> | required |  |


<a id="SnaptoolInfo"></a>

## SnaptoolInfo

<pre>
load("@com_cognitedata_bazel_snapshots//snapshots:toolchain.bzl", "SnaptoolInfo")

SnaptoolInfo(<a href="#SnaptoolInfo-binary">binary</a>)
</pre>

Information about how to invoke the snaptool executable.

**FIELDS**

| Name  | Description |
| :------------- | :------------- |
| <a id="SnaptoolInfo-binary"></a>binary |  Executable snaptool binary    |


