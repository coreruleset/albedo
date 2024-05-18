variable "GIT_TAG" {
    default = "v0.0.0"
}

variable "REPOS" {
    # List of repositories to tag
    default = [
        "ghcr.io/coreruleset/albedo",
    ]
}

function "version" {
    params = [git_tag]
    result = regex_replace(git_tag, "^v", "")
}

function "major" {
    params = [version]
    result = split(".", version)[0]
}

function "minor" {
    params = [version]
    result = join(".", slice(split(".", version),0,2))
}

function "patch" {
    params = [version]
    result = join(".", slice(split(".", version),0,3))
}

function "tag" {
    params = [tag]
    result = [for repo in REPOS : "${repo}:${tag}"]
}

group "default" {
    targets = [
        "default",
    ]
}

function "major" {
    params = [version]
    result = split(".", version)[0]
}

function "major_minor" {
    params = [version]
    result = join(".", slice(split(".", version),0,2))
}

function "major_minor_patch" {
    params = [version]
    result = join(".", slice(split(".", version),0,3))
}

function "tag" {
    params = [tag]
    result = [for repo in REPOS : "${repo}:${tag}"]
}

target "default" {
    context="."
    platforms = ["linux/amd64", "linux/arm64/v8", "linux/arm/v7", "linux/i386"]
    labels = {
        "org.opencontainers.image.source" = "https://github.com/coreruleset/albedo"
    }
    tags = concat(
        tag(major(version(GIT_TAG))),
        tag(major_minor(version(GIT_TAG))),
        tag(major_minor_patch(version(GIT_TAG)))
    )
}
