"""Repositories for bazel-snapshots."""

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:utils.bzl", "maybe")
load("//snapshots/private:toolchains_repo.bzl", "toolchains_repo")

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

SNAPTOOL_BUILD_TMPL = """\
load("@com_cognitedata_bazel_snapshots//snapshots:toolchain.bzl", "snaptool_toolchain")
snaptool_toolchain(
    name = "snaptool_toolchain",
    snaptool = "{binary}"
)
"""

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

def _get_url(ctx):
    goos, goarch = _detect_host_platform(ctx)
    key = "{goos}_{goarch}".format(goos = goos, goarch = goarch)
    return ctx.attr.urls[key], ctx.attr.sha256s[key]

def _snaptool_repo_impl(ctx):
    if ctx.attr.from_source:
        binary = "@com_cognitedata_bazel_snapshots//snapshots/go/cmd/snapshots"
    else:
        binary = "snapshots"
        url, sha256 = _get_url(ctx)
        ctx.download(
            url,
            output = binary,
            sha256 = sha256,
            executable = True,
        )

    ctx.file("BUILD", SNAPTOOL_BUILD_TMPL.format(binary = binary))

snaptool_repositories = repository_rule(
    implementation = _snaptool_repo_impl,
    attrs = {
        "from_source": attr.bool(
            default = True,
        ),
        "urls": attr.string_list_dict(
            default = URLS,
        ),
        "sha256s": attr.string_dict(
            default = SHA256S,
        ),
    },
)

def snapshots_register_toolchains(name, from_source = True):
    """Fetches the necessary repositories for bazel-snapshots.

    Args:
      from_source: if True, will not fetch binaries and instead build the snapshots tool from source.
    """
    snaptool_toolchain_name = "{name}_snaptool_toolchains".format(name = name)

    snaptool_repositories(
        name = "{name}_snaptool".format(name = name),
        from_source = from_source,
    )

    native.register_toolchains("@{repo}//:toolchain".format(repo = snaptool_toolchain_name))

    toolchains_repo(
        name = snaptool_toolchain_name,
        toolchain_type = "@com_cognitedata_bazel_snapshots//snapshots:snaptool_toolchain_type",
        toolchain = "@{name}_snaptool//:snaptool_toolchain".format(name = name),
    )
