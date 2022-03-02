#!/usr/bin/env bash

set -e

[[ -z "$1" ]] && { echo "Version is empty"; exit 1; }

[[ -z "$2" ]] && { echo "Output path is empty"; exit 1; }

VERSION="$1"
OUT_DIR="$2/bazel-snapshots-$VERSION"

mkdir -p "$OUT_DIR"

# create an archive with the relevant files
tar -cvf "$OUT_DIR/snapshots-$VERSION.tar" -C "$BUILD_WORKSPACE_DIRECTORY" snapshots deps.bzl BUILD.bazel WORKSPACE README.md LICENSE

# copy binaries to output folder
cp "snapshots/go/cmd/snapshots/snapshots-darwin-amd64_/snapshots-darwin-amd64" "$OUT_DIR/snapshots-darwin-amd64"
cp "snapshots/go/cmd/snapshots/snapshots-linux-amd64_/snapshots-linux-amd64" "$OUT_DIR/snapshots-linux-amd64"

# find shasums
DARWIN_AMD64_SHA256=$(shasum -a 256 "$OUT_DIR/snapshots-darwin-amd64" | cut -d " " -f 1)
LINUX_AMD64_SHA256=$(shasum -a 256 "$OUT_DIR/snapshots-linux-amd64" | cut -d " " -f 1)

# replace shasums in repo.bzl and add to the archive
sed \
    "s/DARWIN_AMD64_SHA256/$DARWIN_AMD64_SHA256/g; s/LINUX_AMD64_SHA256/$LINUX_AMD64_SHA256/g" \
    "$BUILD_WORKSPACE_DIRECTORY/repo.bzl" \
    > "$OUT_DIR/repo.bzl"
tar --append -C "$OUT_DIR" --file="$OUT_DIR/snapshots-$VERSION.tar" "repo.bzl"
