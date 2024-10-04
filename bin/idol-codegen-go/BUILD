load("@rules_go//go:def.bzl", "go_binary")
load("//internal/build:tinygo.bzl", "idol_codegen_go_wasm")

idol_codegen_go_wasm(
    name = "idol-codegen-go_wasm",
    srcs = [
        "idol-codegen-go.go",
        "idol-codegen-go_wasm.go",
    ],
    deps = [
        "//idol",
        "//idol/codegen_idl",
        "//idol/schema_idl",
    ],
)

go_binary(
    name = "idol-codegen-go",
    srcs = [
        "idol-codegen-go.go",
        "idol-codegen-go_main.go",
    ],
    out = select({
        "@platforms//os:windows": "idol-codegen-go.exe",
        "//conditions:default": "idol-codegen-go",
    }),
    deps = [
        "//idol",
        "//idol/compiler",
        "//idol/schema_idl",
        "//idol/syntax",
    ],
)
