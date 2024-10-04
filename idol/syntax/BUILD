load("@rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "syntax",
    srcs = [
        "syntax.go",
        "syntax_errors.go",
        "syntax_nodes.go",
        "syntax_tokens.go",
    ],
    importpath = "go.idol-lang.org/idol/syntax",
    visibility = ["//visibility:public"],
)

go_test(
    name = "syntax_test",
    size = "small",
    srcs = ["syntax_test.go"],
    data = [
        "@idol//testdata:diagnostics",
        "@idol//testdata:syntax",
    ],
    rundir = ".",
    deps = [
        ":syntax",
        "//idol/internal/testutil",
    ],
)

go_test(
    name = "tokens_test",
    size = "small",
    srcs = ["tokens_test.go"],
    data = [
        "@idol//testdata:diagnostics",
        "@idol//testdata:tokens",
    ],
    rundir = ".",
    deps = [
        ":syntax",
        "//idol/internal/testutil",
    ],
)
