load("@bazel_tools//tools/build_defs/repo:utils.bzl", "read_netrc", "use_netrc")

URLS = {
    "darwin_amd64": {
        "url": "https://cognite.jfrog.io/artifactory/internal-bins/cognite/bazel-snapshots/snapshots-darwin-amd64",
        "sha256": "fe8638e2ed906a5903b893c476a04867990d2f7869f062e4cc6e4eb841364828",
    },
    "linux_amd64": {
        "url": "https://cognite.jfrog.io/artifactory/internal-bins/cognite/bazel-snapshots/snapshots-linux-amd64",
        "sha256": "1a0d9166eb9b4028de72b105d2cbf77729ac313a3355f4189353a8c4d7a422a6",
    }
}

def _get_auth(ctx, urls):
    """Given the list of URLs obtain the correct auth dict."""
    if ctx.attr.netrc:
        netrc = read_netrc(ctx, ctx.attr.netrc)
        return use_netrc(netrc, urls, ctx.attr.auth_patterns)

    if "HOME" in ctx.os.environ and not ctx.os.name.startswith("windows"):
        netrcfile = "%s/.netrc" % (ctx.os.environ["HOME"])
        if ctx.execute(["test", "-f", netrcfile]).return_code == 0:
            netrc = read_netrc(ctx, netrcfile)
            return use_netrc(netrc, urls, ctx.attr.auth_patterns)

    if "USERPROFILE" in ctx.os.environ and ctx.os.name.startswith("windows"):
        netrcfile = "%s/.netrc" % (ctx.os.environ["USERPROFILE"])
        if ctx.path(netrcfile).exists:
            netrc = read_netrc(ctx, netrcfile)
            return use_netrc(netrc, urls, ctx.attr.auth_patterns)

    return {}

def _get_url(ctx):
    info = URLS["linux_amd64"]
    if ctx.os.name.startswith("mac"):
        info = URLS["darwin_amd64"]
    return (info["url"], info["sha256"])

def _snapshots_repos(ctx):
    if ctx.attr.from_source:
        ctx.file("BUILD", 'alias(name="snapshots", actual="@com_cognitedata_bazel_snapshots//snapshots/go/cmd/snapshots", visibility=["//visibility:public"])')
        return

    (url, sha256) = _get_url(ctx)

    auth = _get_auth(ctx, [url])

    ctx.download(
        url,
        output="snapshots",
        sha256 = sha256,
        executable = True,
        auth = auth,
    )
    ctx.file("BUILD", 'exports_files(["snapshots"])')

snapshots_repos = repository_rule(
    implementation = _snapshots_repos,
    attrs = {
        "from_source": attr.bool(),
        "netrc": attr.string(
            doc = "Location of the .netrc file to use for authentication",
        ),
        "auth_patterns": attr.string_dict(
            doc = "see docs for http_archive",
        ),
    }
)
