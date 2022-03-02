#!/usr/bin/env bash

set -e

[[ -z "$1" ]] && { echo "Repository URL is empty"; exit 1; }

[[ -z "$2" ]] && { echo "Version is empty"; exit 1; }

[[ -z "$3" ]] && { echo "Output path is empty"; exit 1; }

REPOSITORY="$1"
VERSION="$2"
OUT_DIR="$3"

DARWIN_AMD64_FILENAME="snapshots-darwin-amd64"
DARWIN_AMD64_URL="$REPOSITORY/releases/download/$VERSION/$DARWIN_AMD64_FILENAME"
ESCAPED_DARWIN_AMD64_URL=$(printf '%s\n' "$DARWIN_AMD64_URL" | sed -e 's/[\/&]/\\&/g')

LINUX_AMD64_FILENAME="snapshots-linux-amd64"
LINUX_AMD64_URL="$REPOSITORY/releases/download/$VERSION/$LINUX_AMD64_FILENAME"
ESCAPED_LINUX_AMD64_URL=$(printf '%s\n' "$LINUX_AMD64_URL" | sed -e 's/[\/&]/\\&/g')

mkdir -p "$OUT_DIR"

# create an archive with the relevant files
tar -cvf "$OUT_DIR/snapshots-$VERSION.tar" -C "$BUILD_WORKSPACE_DIRECTORY" snapshots deps.bzl BUILD.bazel WORKSPACE README.md LICENSE

# copy binaries to output folder
cp "snapshots/go/cmd/snapshots/snapshots-darwin-amd64_/$DARWIN_AMD64_FILENAME" "$OUT_DIR/$DARWIN_AMD64_FILENAME"
cp "snapshots/go/cmd/snapshots/snapshots-linux-amd64_/$LINUX_AMD64_FILENAME" "$OUT_DIR/$LINUX_LINUX_AMD64_FILENAME"

# generate sha files
(cd $OUT_DIR ; shasum -a 256 "$DARWIN_AMD64_FILENAME" > "$DARWIN_AMD64_FILENAME.sha256")
(cd $OUT_DIR ; shasum -a 256 "$LINUX_AMD64_FILENAME" > "$LINUX_AMD64_FILENAME.sha256")

# find shasums
DARWIN_AMD64_SHA256=$(shasum -a 256 "$OUT_DIR/$DARWIN_AMD64_FILENAME" | cut -d " " -f 1)
LINUX_AMD64_SHA256=$(shasum -a 256 "$OUT_DIR/$LINUX_AMD64_FILENAME" | cut -d " " -f 1)

# replace urls in repo.bzl
sed \
    "s/DARWIN_AMD64_URL/$ESCAPED_DARWIN_AMD64_URL/g; s/LINUX_AMD64_URL/$ESCAPED_LINUX_AMD64_URL/g" \
    "$BUILD_WORKSPACE_DIRECTORY/repo.bzl" \
    > "$OUT_DIR/repo.bzl"

# replace shasums in repo.bzl
sed -i \
    "s/DARWIN_AMD64_SHA256/$DARWIN_AMD64_SHA256/g; s/LINUX_AMD64_SHA256/$LINUX_AMD64_SHA256/g" \
    "$OUT_DIR/repo.bzl"

# add to the archive
tar --append -C "$OUT_DIR" --file="$OUT_DIR/snapshots-$VERSION.tar" "repo.bzl"
