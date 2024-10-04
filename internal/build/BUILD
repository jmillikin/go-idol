load("@rules_go//go:def.bzl", "go_binary")

config_setting(
    name = "aarch64-linux",
    constraint_values = [
        "@platforms//cpu:aarch64",
        "@platforms//os:linux",
    ],
)

config_setting(
    name = "aarch64-macos",
    constraint_values = [
        "@platforms//cpu:aarch64",
        "@platforms//os:macos",
    ],
)

config_setting(
    name = "x86_64-linux",
    constraint_values = [
        "@platforms//cpu:x86_64",
        "@platforms//os:linux",
    ],
)

config_setting(
    name = "x86_64-macos",
    constraint_values = [
        "@platforms//cpu:x86_64",
        "@platforms//os:macos",
    ],
)

config_setting(
    name = "x86_64-windows",
    constraint_values = [
        "@platforms//cpu:x86_64",
        "@platforms//os:windows",
    ],
)

alias(
    name = "tinygo",
    actual = select({
        ":aarch64-linux": "@tinygo0.33.0.linux-arm64//:bin/tinygo",
        ":aarch64-macos": "@tinygo0.33.0.darwin-arm64//:bin/tinygo",
        ":x86_64-linux": "@tinygo0.33.0.linux-amd64//:bin/tinygo",
        ":x86_64-macos": "@tinygo0.33.0.darwin-amd64//:bin/tinygo",
        ":x86_64-windows": "@tinygo0.33.0.windows-amd64//:bin/tinygo",
    }),
    visibility = ["//:__subpackages__"],
)

alias(
    name = "tinygo_files",
    actual = select({
        ":aarch64-linux": "@tinygo0.33.0.linux-arm64//:all_files",
        ":aarch64-macos": "@tinygo0.33.0.darwin-arm64//:all_files",
        ":x86_64-linux": "@tinygo0.33.0.linux-amd64//:all_files",
        ":x86_64-macos": "@tinygo0.33.0.darwin-amd64//:all_files",
        ":x86_64-windows": "@tinygo0.33.0.windows-amd64//:all_files",
    }),
    visibility = ["//:__subpackages__"],
)

alias(
    name = "wasm-opt",
    actual = select({
        ":aarch64-linux": "@binaryen-119-aarch64-linux//:bin/wasm-opt",
        ":aarch64-macos": "@binaryen-119-arm64-macos//:bin/wasm-opt",
        ":x86_64-linux": "@binaryen-119-x86_64-linux//:bin/wasm-opt",
        ":x86_64-macos": "@binaryen-119-x86_64-macos//:bin/wasm-opt",
        ":x86_64-windows": "@binaryen-119-x86_64-windows//:bin/wasm-opt",
    }),
    visibility = ["//:__subpackages__"],
)

go_binary(
    name = "tinygo_build",
    srcs = ["tinygo_build.go"],
    visibility = ["//:__subpackages__"],
)
