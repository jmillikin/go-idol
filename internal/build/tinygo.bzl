load("@rules_go//go:def.bzl", "GoArchive", "GoSource")

_TINYGO_URL = "https://github.com/tinygo-org/tinygo/releases/download/"

_TINYGO_FILES = {
    "0.33.0": {
        "v0.33.0/tinygo0.33.0.darwin-amd64.tar.gz": "b4c8263185929c211f48ddbe00c155c1ea899857a2f231289d880a700bfa2264",
        "v0.33.0/tinygo0.33.0.darwin-arm64.tar.gz": "6e116cb29ce9a3387783186d7067280f08577abeba47360bdc85770f669848ce",
        "v0.33.0/tinygo0.33.0.linux-amd64.tar.gz": "a529eff745a9ecb78f0c086492ddc2645a53e0a37fa99e750d2d0a785a42ba91",
        "v0.33.0/tinygo0.33.0.linux-arm.tar.gz": "02e00e03501cdc5ec44612d37fe5ed8c74c349b2d986157a99a535a406ca439e",
        "v0.33.0/tinygo0.33.0.linux-arm64.tar.gz": "0b59b0910db468d4a255cbc452c5f9e740903c670486f2865120a415c558ea5d",
        "v0.33.0/tinygo0.33.0.windows-amd64.zip": "38fbe0c07fb5e0bc689278751a5ef23fd8cdefe86b70aeb00794e82006c7ded7",
    },
}

_BINARYEN_URL = "https://github.com/WebAssembly/binaryen/releases/download/"

_BINARYEN_FILES = {
    "119": {
        "version_119/binaryen-version_119-aarch64-linux.tar.gz": "537b0c137afcde45ea42df72e46fc19738c60af8ca78be9319967eefb6f8bcf6",
        "version_119/binaryen-version_119-arm64-macos.tar.gz": "c12dffafb3e3274026268e90577bd86d98186f7be32457618672f8ca437d8d53",
        "version_119/binaryen-version_119-x86_64-linux.tar.gz": "716bcf9f5f36a6f466239fbb09a925eeaf54c46411ccefac979ec649e7c06d2d",
        "version_119/binaryen-version_119-x86_64-macos.tar.gz": "eed19e583d7bc2a7482bced15ec3fa56a87811b310f37241b5016a66fc95cc4b",
        "version_119/binaryen-version_119-x86_64-windows.tar.gz": "e76eb6852208bc0044ad7015cfcaa351ef0992e18b966a83eddd5f6253fdbefc",
    },
}

_TINYGO_REPOSITORY_BUILD = """
exports_files(["bin/tinygo"])

filegroup(
    name = "all_files",
    srcs = glob(
        include = [
            "src/**/*",
            "targets/wasm*.json",
        ],
        exclude = [
            "src/device/**/*",
            "src/examples/**/*",
            "src/internal/wasi/**/*",
            "src/machine/**/*",
        ],
    ) + glob([
        "src/device/arm/**/*",
    ]),
    visibility = ["@go-idol//:__subpackages__"],
)
"""

def _tinygo_repository(ctx):
    version = ctx.attr.version
    platform = ctx.attr.platform

    ext = "tar.gz"
    if "windows" in platform:
        ext = "zip"

    checksums = _TINYGO_FILES[version]
    filename = "v{0}/tinygo{0}.{1}.{2}".format(version, platform, ext)

    ctx.download_and_extract(
        url = [_TINYGO_URL + filename],
        sha256 = checksums[filename],
        stripPrefix = "tinygo",
    )

    ctx.file("BUILD.bazel", _TINYGO_REPOSITORY_BUILD)

tinygo_repository = repository_rule(
    implementation = _tinygo_repository,
    attrs = {
        "version": attr.string(),
        "platform": attr.string(),
    },
)

_BINARYEN_REPOSITORY_BUILD = """
exports_files(["bin/wasm-opt"])
"""

def _binaryen_repository(ctx):
    version = ctx.attr.version
    platform = ctx.attr.platform

    checksums = _BINARYEN_FILES[version]
    filename = "version_{0}/binaryen-version_{0}-{1}.tar.gz".format(version, platform)

    ctx.download_and_extract(
        url = [_BINARYEN_URL + filename],
        sha256 = checksums[filename],
        stripPrefix = "binaryen-version_{}".format(version),
    )

    ctx.file("BUILD.bazel", _BINARYEN_REPOSITORY_BUILD)

binaryen_repository = repository_rule(
    implementation = _binaryen_repository,
    attrs = {
        "version": attr.string(),
        "platform": attr.string(),
    },
)

def _copy_to(tinygo_srcroot, src):
    filename = src.short_path

    return tinygo_srcroot + filename

def _idol_codegen_go_wasm(ctx):
    go = ctx.toolchains["@rules_go//go:toolchain"]
    go_tool = go.sdk.go

    tinygo_srcroot = ctx.attr.name + ".d/go.idol-lang.org/"

    idol_codegen_go_gomod = ctx.actions.declare_file(
        tinygo_srcroot + "bin/idol-codegen-go/go.mod",
    )
    ctx.actions.write(
        output = idol_codegen_go_gomod,
        content = "\n".join([
            "module go.idol-lang.org/bin/idol-codegen-go",
            "go 1.23.1",
            "require go.idol-lang.org/idol v0.0.0",
            "replace go.idol-lang.org/idol v0.0.0 => ../../idol",
            "",
        ]),
    )

    idol_gomod = ctx.actions.declare_file(
        tinygo_srcroot + "idol/go.mod",
    )
    ctx.actions.write(
        output = idol_gomod,
        content = "\n".join([
            "module go.idol-lang.org/idol",
            "go 1.23.1",
            "",
        ]),
    )

    inputs = [idol_codegen_go_gomod, idol_gomod]

    srcs = []
    for src in ctx.files.srcs:
        srcs.append(src)
    for dep in ctx.attr.deps:
        if GoArchive in dep:
            for src in dep[GoArchive].data.srcs:
                srcs.append(src)
            for tdep in dep[GoArchive].transitive.to_list():
                for src in tdep.srcs:
                    srcs.append(src)
        elif GoSource in dep:
            for src in dep[GoSource].srcs:
                srcs.append(src)

    for src in depset(srcs).to_list():
        copied = ctx.actions.declare_file(_copy_to(tinygo_srcroot, src))
        ctx.actions.expand_template(template = src, output = copied)
        inputs.append(copied)

    if ctx.attr.name.endswith("_wasm"):
        out_basename = ctx.attr.name[:-5]
    else:
        out_basename = ctx.attr.name
    out = ctx.actions.declare_file(out_basename + ".wasm")
    ctx.actions.run(
        outputs = [out],
        inputs = inputs + ctx.files._tinygo_files,
        executable = ctx.executable._tinygo_build,
        arguments = [
            "-tinygo=" + ctx.executable._tinygo.path,
            "-output=" + out.path,
            "-go-sdk-bin=" + go_tool.dirname,
            "-chdir=" + idol_codegen_go_gomod.dirname,
            "-wasm-opt=" + ctx.executable._wasm_opt.path,
            "--",
            "-target=wasm-unknown",
            "-no-debug",
            "-opt=s",
            "go.idol-lang.org/bin/idol-codegen-go",
        ],
        tools = [
            ctx.executable._tinygo,
            ctx.executable._wasm_opt,
            go_tool,
        ],
        mnemonic = "TinyGo",
        toolchain = "@rules_go//go:toolchain",
    )
    return DefaultInfo(files = depset([out]))

idol_codegen_go_wasm = rule(
    implementation = _idol_codegen_go_wasm,
    attrs = {
        "srcs": attr.label_list(
            allow_files = [".go"],
        ),
        "deps": attr.label_list(
            providers = [GoSource, GoArchive],
        ),
        "_tinygo": attr.label(
            default = "//internal/build:tinygo",
            executable = True,
            allow_single_file = True,
            cfg = "host",
        ),
        "_tinygo_files": attr.label(
            default = "//internal/build:tinygo_files",
            allow_files = True,
        ),
        "_tinygo_build": attr.label(
            default = "//internal/build:tinygo_build",
            executable = True,
            allow_single_file = True,
            cfg = "host",
        ),
        "_wasm_opt": attr.label(
            default = "//internal/build:wasm-opt",
            executable = True,
            allow_single_file = True,
            cfg = "host",
        ),
    },
    toolchains = [
        "@rules_go//go:toolchain",
    ],
)
