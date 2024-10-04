load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "compiler",
    srcs = [
        "compiler.go",
        "compiler_deps.go",
        "compiler_errors.go",
        "compiler_warnings.go",
    ],
    importpath = "go.idol-lang.org/idol/compiler",
    visibility = ["//visibility:public"],
    deps = [
        "//idol",
        "//idol/schema_idl",
        "//idol/syntax",
    ],
)

go_test(
    name = "compiler_test",
    size = "small",
    srcs = ["compiler_test.go"],
    data = [
        "@idol//testdata:diagnostics",
        "@idol//testdata:schema",
    ],
    rundir = ".",
    deps = [
        ":compiler",
        "//idol/encoding/idoltext",
        "//idol/internal/testutil",
        "//idol/schema_idl",
        "//idol/syntax",
    ],
)
