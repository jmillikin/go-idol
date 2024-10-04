load("@rules_go//go:def.bzl", "go_library")

go_library(
    name = "testutil",
    testonly = True,
    srcs = [
        "testutil.go",
        "testutil_assertions.go",
        "testutil_schema.go",
        "testutil_syntax.go",
    ],
    data = [
        "@idol//testdata:diagnostics",
    ],
    importpath = "go.idol-lang.org/idol/internal/testutil",
    visibility = [
        "//:__subpackages__",
    ],
    x_defs = {
        "diagnosticsPaths": "$(rlocationpaths @idol//testdata:diagnostics)",
    },
    deps = [
        "//idol/syntax",
        "@com_github_pmezard_go_difflib//difflib",
    ],
)
