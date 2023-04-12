load("@io_bazel_rules_docker//container:providers.bzl", "BundleInfo", "ImageInfo")

def _docker_image_id_impl(ctx):
    files = []
    image = ctx.attr.image

    if not BundleInfo in image and not ImageInfo in image:
        fail("`image` must contain BundleInfo or ImageInfo")

    if BundleInfo in image:
        for _, data in image[BundleInfo].container_images.items():
            if data["manifest_digest"] != None:
                files.append(data["manifest_digest"])
            files.extend(data["blobsum"])

    if ImageInfo in image:
        container_parts = image[ImageInfo].container_parts
        if container_parts["manifest_digest"] != None:
            files.append(container_parts["manifest_digest"])
        files.extend(container_parts["blobsum"])

    return DefaultInfo(
        files = depset(files),
    )

docker_image_id = rule(
    implementation = _docker_image_id_impl,
    attrs = {
        "image": attr.label(
            doc = "Image or image bundle to process",
            allow_files = True,
        ),
    },
)
