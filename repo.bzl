"""Repositories for bazel-snapshots."""

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:utils.bzl", "maybe")

# Pointing to the latest released binaries.
URLS = {
    "darwin_amd64": ["DARWIN_AMD64_URL"],
    "darwin_arm64": ["DARWIN_ARM64_URL"],
    "linux_amd64": ["LINUX_AMD64_URL"],
    "linux_arm64": ["LINUX_ARM64_URL"],
}

# sha256 sums of the binaries above
SHA256S = {
    "darwin_amd64": "DARWIN_AMD64_SHA256",
    "darwin_arm64": "DARWIN_ARM64_SHA256",
    "linux_amd64": "LINUX_AMD64_SHA256",
    "linux_arm64": "LINUX_ARM64_SHA256",
}

def _detect_host_platform(ctx):
    if ctx.os.name == "linux":
        goos, goarch = "linux", "amd64"
        res = ctx.execute(["uname", "-m"])
        if res.return_code == 0:
            uname = res.stdout.strip()
            if uname == "aarch64":
                goarch = "arm64"

    elif ctx.os.name == "mac os x":
        goos, goarch = "darwin", "amd64"

        res = ctx.execute(["uname", "-m"])
        if res.return_code == 0:
            uname = res.stdout.strip()
            if uname == "arm64":
                goarch = "arm64"

    else:
        fail("Unsupported operating system: " + ctx.os.name)

    return goos, goarch

def _get_url(ctx, goos, goarch):
    key = "{goos}_{goarch}".format(goos = goos, goarch = goarch)
    return ctx.attr.urls[key], ctx.attr.sha256s[key]

def _snapshots_binaries(ctx):
    if ctx.attr.from_source:
        ctx.file("BUILD", 'alias(name="snapshots", actual="@com_cognitedata_bazel_snapshots//snapshots/go/cmd/snapshots", visibility=["//visibility:public"])')
        return

    goos, goarch = _detect_host_platform(ctx)
    url, sha256 = _get_url(ctx, goos, goarch)

    ctx.download(
        url,
        output="snapshots",
        sha256 = sha256,
        executable = True,
    )
    ctx.file("BUILD", 'exports_files(["snapshots"])')

snapshots_binaries = repository_rule(
    implementation = _snapshots_binaries,
    attrs = {
        "from_source": attr.bool(),
        "urls": attr.string_list_dict(),
        "sha256s": attr.string_dict(),
    }
)

def snapshots_repos(name = "snapshots", from_source = False, urls = URLS, sha256s = SHA256S):
    """Fetches the necessary repositories for bazel-snapshots.

    Args:
      name: unique name (defaults to "snapshots")
      from_source: if True, will not fetch binaries and instead build the
        snapshots tool from source.
      urls: dict with platforms as keys, list of URLs as values
      sha256s: dict with platforms as keys, sha256 sum of files from urls as values
    """
    snapshots_binaries(
        name = "{name}-bin".format(name = name),
        from_source = from_source,
        urls = urls,
        sha256s = sha256s
    )

    maybe(
        http_archive,
        name = "bazel_skylib",
        # 1.1.1, latest as of 2022-01-24
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.1.1/bazel-skylib-1.1.1.tar.gz",
            "https://github.com/bazelbuild/bazel-skylib/releases/download/1.1.1/bazel-skylib-1.1.1.tar.gz",
        ],
        sha256 = "c6966ec828da198c5d9adbaa94c05e3a1c7f21bd012a0b29ba8ddbccb2c93b0d",
        strip_prefix = "",
    )

    maybe(
        http_archive,
        name = "io_bazel_rules_docker",
        sha256 = "85ffff62a4c22a74dbd98d05da6cf40f497344b3dbf1e1ab0a37ab2a1a6ca014",
        strip_prefix = "rules_docker-0.23.0",
        urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.23.0/rules_docker-v0.23.0.tar.gz"],
    )
