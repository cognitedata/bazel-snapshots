#!/usr/bin/env bash

display_usage() {
	echo -e "\nUsage: ./release.sh [repository_url] [version] [output_path] \n"
}

# if less than three arguments supplied, display usage
if [  $# -le 2 ]
then
    display_usage
    exit 1
fi

set -e

REPOSITORY="$1"
VERSION="$2"
OUT_DIR="$3"

TARGETS=("darwin-amd64" "darwin-arm64" "linux-amd64" "linux-arm64")

mkdir -p "$OUT_DIR"

cp "$BUILD_WORKSPACE_DIRECTORY/repo.bzl" "$OUT_DIR/repo.bzl"

# create an archive with the relevant files
tar -cf "$OUT_DIR/snapshots-$VERSION.tar" -C "$BUILD_WORKSPACE_DIRECTORY" snapshots deps.bzl BUILD.bazel WORKSPACE README.md LICENSE

for t in "${TARGETS[@]}";
do
    FILENAME="snapshots-$t"

    # turn e.g. linux-amd64 into LINUX_AMD64
    PLACEHOLDER=$(printf '%s\n' "$t" | awk '{ print toupper($0) }' | sed -e "s/-/_/g")

    # copy binaries to output folder
    cp "snapshots/go/cmd/snapshots/${FILENAME}_/$FILENAME" "$OUT_DIR/$FILENAME"

    # generate sha files
    (cd $OUT_DIR ; shasum -a 256 "$FILENAME" > "$FILENAME.sha256")

    # find shasums
    FILE_SHA256=$(cut -d " " -f 1 "$OUT_DIR/$FILENAME.sha256")

    # build download url
    FILE_URL="$REPOSITORY/releases/download/$VERSION/$FILENAME"

    # replace urls in repo.bzl
    sed -i_bak "s|${PLACEHOLDER}_URL|$FILE_URL|g" "$OUT_DIR/repo.bzl"

    # replace shasums in repo.bzl
    sed -i_bak "s|${PLACEHOLDER}_SHA256|$FILE_SHA256|g" "$OUT_DIR/repo.bzl"

    echo -e "Created $FILENAME"
done

# add to the archive
tar --append -C "$OUT_DIR" --file="$OUT_DIR/snapshots-$VERSION.tar" "repo.bzl"
