load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "codegen_idl",
    srcs = ["codegen_idl.go"],
    importpath = "go.idol-lang.org/idol/codegen_idl",
    visibility = ["//visibility:public"],
    deps = [
        "//idol",
        "//idol/schema_idl",
    ],
)
