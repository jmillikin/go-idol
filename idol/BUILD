load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "idol",
    srcs = [
        "idol.go",
        "idol_array.go",
        "idol_errors.go",
        "idol_field_builders.go",
        "idol_message.go",
    ],
    importpath = "go.idol-lang.org/idol",
    visibility = ["//visibility:public"],
)

go_test(
    name = "array_test",
    size = "small",
    srcs = ["array_test.go"],
    rundir = ".",
    deps = [
        ":idol",
        "//idol/internal/testutil",
    ],
)

go_test(
    name = "message_test",
    size = "small",
    srcs = ["message_test.go"],
    rundir = ".",
    deps = [
        ":idol",
        "//idol/internal/testutil",
    ],
)
