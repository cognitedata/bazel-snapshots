"""Repositories for bazel-snapshots."""

URLS = {
    "darwin_amd64": {
        "url": "https://github.com/cognitedata/bazel-snapshots/releases/download/v0.0.1/snapshots-darwin-amd64",
        "sha256": "2bd4950e27f5204169cacb75c113162c9d3ffebd154ab1edae0719f62da41a99",
    },
    "linux_amd64": {
        "url": "https://github.com/cognitedata/bazel-snapshots/releases/download/v0.0.1/snapshots-linux-amd64",
        "sha256": "1a0d9166eb9b4028de72b105d2cbf77729ac313a3355f4189353a8c4d7a422a6",
    }
}

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

    ctx.download(
        url,
        output="snapshots",
        sha256 = sha256,
        executable = True,
    )
    ctx.file("BUILD", 'exports_files(["snapshots"])')

snapshots_repos = repository_rule(
    implementation = _snapshots_repos,
    attrs = {
        "from_source": attr.bool(),
    }
)
