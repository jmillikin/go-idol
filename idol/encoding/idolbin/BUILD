load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "idolbin",
    srcs = [
        "idolbin.go",
        "idolbin_message.go",
        "idolbin_message_field.go",
    ],
    importpath = "go.idol-lang.org/idol/encoding/idolbin",
    visibility = ["//visibility:public"],
    deps = [
        "//idol",
    ],
)

go_test(
    name = "idolbin_test",
    size = "small",
    srcs = ["idolbin_test.go"],
    rundir = ".",
    deps = [
        ":idolbin",
        "//idol/internal/testutil",
    ],
)
