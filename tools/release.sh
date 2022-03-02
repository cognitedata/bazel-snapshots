#!/usr/bin/env bash

set -e

[[ -z "$1" ]] && { echo "Version is empty"; exit 1; }

[[ -z "$2" ]] && { echo "Output path is empty"; exit 1; }

VERSION="$1"
OUT_DIR="$2/bazel-snapshots-$VERSION"

mkdir -p "$OUT_DIR"

cp "snapshots/go/cmd/snapshots/snapshots-darwin-amd64_/snapshots-darwin-amd64" "$OUT_DIR/snapshots-darwin-amd64"
cp "snapshots/go/cmd/snapshots/snapshots-linux-amd64_/snapshots-linux-amd64" "$OUT_DIR/snapshots-linux-amd64"

shasum -a 256 "$OUT_DIR"/*
