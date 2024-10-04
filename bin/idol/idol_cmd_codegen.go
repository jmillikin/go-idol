// Copyright (c) 2024 John Millikin <john@john-millikin.com>
//
// Permission to use, copy, modify, and/or distribute this software for any
// purpose with or without fee is hereby granted.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH
// REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY
// AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
// INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
// LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR
// OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
// PERFORMANCE OF THIS SOFTWARE.
//
// SPDX-License-Identifier: 0BSD

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	wasm "github.com/tetratelabs/wazero"

	"go.idol-lang.org/idol"
	"go.idol-lang.org/idol/codegen_idl"
	"go.idol-lang.org/idol/schema_idl"
)

type cmdCodegen struct {
	outDir     string
	pluginPath string
}

func (*cmdCodegen) help() *commandHelp {
	return &commandHelp{
		usage: "codegen",
	}
}

func (cmd *cmdCodegen) flags(flags *pflag.FlagSet) {
	flags.StringVarP(&cmd.outDir, "output", "o", "", "(docs TODO)")
	flags.StringVar(&cmd.pluginPath, "plugin-path", "", "(docs TODO)")
}

func (cmd *cmdCodegen) run(ctx context.Context, argv []string) int {
	if len(argv) < 1 {
		//log.Fatalf("usage: %s IDOL_SCHEMA [deps...]", os.Args[0])
		panic("todo: message on not enough args")
		return 1
	}

	if cmd.outDir == "" {
		fmt.Fprintln(os.Stderr, "No output directory specified (set --output=)")
		return 1
	}

	type Schema = schema_idl.Schema
	type RequestBuilder = codegen_idl.CodegenRequest__Builder
	type Response = codegen_idl.CodegenResponse

	schemaPath := argv[0]
	schemaBuf, err := os.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	requestBuilder := &RequestBuilder{}

	// FIXME: include directly in the request, without decoding first?
	schema, err := idol.DecodeAs[Schema](&idol.DecodeCtx{}, schemaBuf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	requestBuilder.Schema.Set(idol.Clone(schema).Self())

	for _, depPath := range argv[1:] {
		depBuf, err := os.ReadFile(depPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		// FIXME: include directly in the request, without decoding first?
		dep, err := idol.DecodeAs[Schema](&idol.DecodeCtx{}, depBuf)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		requestBuilder.Dependencies.Add(idol.Clone(dep).Self())
	}

	requestBuf, err := idol.Encode(&idol.EncodeCtx{}, requestBuilder)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	runtimeConfig := wasm.NewRuntimeConfigInterpreter()
	runtimeConfig = runtimeConfig.WithMemoryLimitPages(16384)
	runtime := wasm.NewRuntimeWithConfig(ctx, runtimeConfig)
	defer runtime.Close(ctx)

	pluginPath, err := cmd.locatePlugin("go")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	pluginBin, err := os.ReadFile(pluginPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	pluginExe, err := runtime.CompileModule(ctx, pluginBin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	moduleConfig := wasm.NewModuleConfig()
	plugin, err := runtime.InstantiateModule(ctx, pluginExe, moduleConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	mem := plugin.Memory()

	wasmAlloc := plugin.ExportedFunction("idol_codegen_allocate")
	//wasmDealloc := plugin.ExportedFunction("idol_codegen_deallocate")
	wasmGenerate := plugin.ExportedFunction("idol_codegen_generate/go")

	results, err := wasmAlloc.Call(ctx, uint64(len(requestBuf)))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	requestPtr := results[0]

	mem.Write(uint32(requestPtr), requestBuf)

	results, err = wasmAlloc.Call(ctx, 4)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	responsePtrPtr := uint32(results[0])

	results, err = wasmGenerate.Call(ctx, requestPtr, uint64(responsePtrPtr))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rc := uint8(results[0])

	responsePtr, _ := mem.ReadUint32Le(responsePtrPtr)
	responseLen, ok := mem.ReadUint32Le(responsePtr)
	if !ok {
		fmt.Fprintln(os.Stderr, "Failed to read response message length")
		return 1
	}
	responseBuf, ok := mem.Read(responsePtr, responseLen)
	if !ok {
		fmt.Fprintln(os.Stderr, "Failed to read response message")
		return 1
	}

	response, err := idol.DecodeAs[Response](&idol.DecodeCtx{}, responseBuf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if rc != 0 {
		// TODO: Replace any control characters with U+FFFD.
		// TODO: trim newlines if present
		fmt.Fprintf(os.Stderr, "%s\n", response.Error())
		return 1
	}

	outputFiles := response.OutputFiles()
	if outputFiles.Len() == 0 {
		fmt.Fprintln(os.Stderr, "Plugin did not generate any output files")
		return 1
	}
	if err := os.MkdirAll(cmd.outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for _, outputFile := range outputFiles.Iter() {
		outPath, err := cmd.outPath(outputFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		content := outputFile.Content().Collect()
		if err := os.WriteFile(outPath, content, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	return 0
}

func (cmd *cmdCodegen) locatePlugin(language string) (string, error) {
	path := cmd.pluginPath
	if path == "" {
		path = os.Getenv("IDOL_CODEGEN_PLUGIN_PATH")
	}
	if path == "" {
		return "", fmt.Errorf("No plugin path set, use --plugin-path= or $IDOL_CODEGEN_PLUGIN_PATH")
	}
	basename := fmt.Sprintf("idol-codegen-%s.wasm", language)
	// FIXME
	for _, path := range strings.Split(path, ":") {
		pluginPath := filepath.Join(path, basename)
		if _, err := os.Stat(pluginPath); err == nil {
			return pluginPath, nil
		}
	}
	return "", fmt.Errorf(
		"Idol codegen plugin %s not found in plugin path",
		basename,
	)
}

func (cmd *cmdCodegen) outPath(file codegen_idl.OutputFile) (string, error) {
	parts := file.Path().Collect()
	if len(parts) == 0 {
		return "", fmt.Errorf("Invalid output path %#v: empty", parts)
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("Invalid output path %#v: bad path component %q", parts, part)
		}
		if part[0] == '/' || filepath.IsAbs(part) {
			return "", fmt.Errorf("Invalid output path %#v: absolute path component %q", parts, part)
		}
		if strings.Contains(part, "/") {
			return "", fmt.Errorf("Invalid output path %#v: component %q contains '/'", parts, part)
		}
	}
	return filepath.Join(append([]string{cmd.outDir}, parts...)...), nil
}
