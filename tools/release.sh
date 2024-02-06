#!/usr/bin/env bash

display_usage() {
	echo -e "\nUsage: ./release.sh [repository_url] [tag] [output_path] \n"
}

# if less than three arguments supplied, display usage
if [  $# -le 2 ]
then
    display_usage
    exit 1
fi

set -e

REPOSITORY="$1"
TAG="$2"
OUT_DIR="$3"

TARGETS=("darwin-amd64" "darwin-arm64" "linux-amd64" "linux-arm64")

mkdir -p "$OUT_DIR/docker"
mkdir -p "$OUT_DIR/snapshots"

cp "$BUILD_WORKSPACE_DIRECTORY/snapshots/repositories.bzl" "$OUT_DIR/snapshots/repositories.bzl"

VERSION=${TAG//v/}
sed -i "3s/version = .*/version = \"$VERSION\",/g" MODULE.bazel

# create an archive with the relevant files
tar -cf "$OUT_DIR/snapshots-$TAG.tar" -C "$BUILD_WORKSPACE_DIRECTORY" go.mod go.sum docker snapshots snapshots/dependencies.bzl BUILD.bazel MODULE.bazel WORKSPACE WORKSPACE.bzlmod README.md LICENSE

for t in "${TARGETS[@]}";
do
    FILENAME="snapshots-$t"
    FOLDER="snapshots/go/cmd/snapshots/${FILENAME}_"

    # turn e.g. linux-amd64 into LINUX_AMD64
    PLACEHOLDER=$(echo "$t" | tr a-z- A-Z_)

    # copy binaries to output folder
    cp "${FOLDER}/$FILENAME" "$OUT_DIR/$FILENAME"

    # generate sha files
    (cd "$OUT_DIR" ; shasum -a 256 "$FILENAME" > "$FILENAME.sha256")

    # find sha sums
    FILE_SHA256=$(cut -d " " -f 1 "$OUT_DIR/$FILENAME.sha256")
    PLACEHOLDER_SHA="${PLACEHOLDER}_SHA256"

    # replace sha sums in snapshots/repositories.bzl
    sed -i_bak "s,$PLACEHOLDER_SHA,$FILE_SHA256,g" "$OUT_DIR/snapshots/repositories.bzl"

    # build download url
    FILE_URL="$REPOSITORY/releases/download/$TAG/$FILENAME"
    PLACEHOLDER_URL="${PLACEHOLDER}_URL"

    # replace urls in snapshots/repositories.bzl
    sed -i_bak "s,$PLACEHOLDER_URL,$FILE_URL,g" "$OUT_DIR/snapshots/repositories.bzl"

    echo -e "Created $FILENAME"
done

# add to the archive
tar --append -C "$OUT_DIR" --file="$OUT_DIR/snapshots-$TAG.tar" "snapshots/repositories.bzl"

# create sha file for archive
(cd "$OUT_DIR" ; shasum -a 256 "snapshots-$TAG.tar" > "snapshots-$TAG.tar.sha256")
