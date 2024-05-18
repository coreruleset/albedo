group "default" {
    targets = [
        "default",
    ]
}

target "default" {
    context="."
    platforms = ["linux/amd64", "linux/arm64/v8", "linux/arm/v7", "linux/i386"]
    labels = {
        "org.opencontainers.image.source" = "https://github.com/coreruleset/albedo"
    }
}
