#!/usr/bin/env bash

# Used for editor integration.
# See https://github.com/bazelbuild/rules_go/wiki/Editor-setup
exec bazel run -- @io_bazel_rules_go//go/tools/gopackagesdriver "${@}"
