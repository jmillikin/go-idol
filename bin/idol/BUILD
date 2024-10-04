load("@rules_go//go:def.bzl", "go_binary")

go_binary(
    name = "idol",
    srcs = [
        "idol.go",
        "idol_cmd_codegen.go",
        "idol_cmd_compile.go",
        "idol_cmd_format.go",
        "idol_util.go",
    ],
    out = select({
        "@platforms//os:windows": "idol.exe",
        "//conditions:default": "idol",
    }),
    deps = [
        "//idol",
        "//idol/codegen_idl",
        "//idol/compiler",
        "//idol/encoding/idoltext",
        "//idol/schema_idl",
        "//idol/syntax",
        "@com_github_spf13_cobra//:cobra",
        "@com_github_spf13_pflag//:pflag",
        "@com_github_tetratelabs_wazero//:wazero",
    ],
)
