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

	"github.com/spf13/pflag"

	"go.idol-lang.org/idol"
	"go.idol-lang.org/idol/compiler"
	"go.idol-lang.org/idol/encoding/idoltext"
	"go.idol-lang.org/idol/schema_idl"
	"go.idol-lang.org/idol/syntax"
)

type cmdCompile struct {
	outPath string
	format  string
}

func (*cmdCompile) help() *commandHelp {
	return &commandHelp{
		usage: "compile",
	}
}

func (cmd *cmdCompile) flags(flags *pflag.FlagSet) {
	flags.StringVarP(&cmd.outPath, "output", "o", "", "(docs TODO)")
	flags.StringVarP(&cmd.format, "format", "f", "", "(docs TODO)")
}

func (cmd *cmdCompile) run(ctx context.Context, argv []string) int {
	if len(argv) < 1 {
		// log.Fatalf("usage: %s IDOL_SCHEMA [deps...]", os.Args[0])
		panic("todo: message on not enough args")
		return 1
	}
	srcPath := argv[0]

	outputText := false
	switch cmd.format {
	case "":
		// FIXME: Select default format based on output path extension.
		//
		// ".txt", ".idoltext" -> "text"
		// ".bin", ".idolbin" -> "binary"
		//
		// Error out only if can't guess.
		fmt.Fprintln(os.Stderr, "No format selected (choose 'text' or 'binary')")
		return 1
	case "text", "idoltext":
		outputText = true
	case "bin", "binary", "idolbin":
	default:
		fmt.Fprintf(os.Stderr, "Unsupported output format %q\n", cmd.format)
		return 1
	}

	var deps []schema_idl.Schema
	for _, depPath := range argv[1:] {
		depBuf, err := os.ReadFile(depPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		// FIXME: include directly in the request, without decoding first?
		dep, err := idol.DecodeAs[schema_idl.Schema](&idol.DecodeCtx{}, depBuf)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		deps = append(deps, dep)
	}

	var opts []compiler.CompileOption
	if len(deps) > 0 {
		mergedDeps, err := compiler.Merge(deps)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		opts = append(opts, compiler.WithDependencies(mergedDeps))
	}

	if !filepath.IsAbs(srcPath) {
		opts = append(opts, compiler.WithSourcePath(splitPath(srcPath)))
	}

	src, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	parsed, err := syntax.Parse(src)
	if err != nil {
		// TODO: Map error span to line + column location
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// TODO:
	// - interleave by line number?
	// - Different colors for warnings vs errors?
	// - Map spans to line + column locations
	result := compiler.Compile(parsed, opts...)
	for _, warn := range result.Warnings {
		fmt.Fprintf(os.Stderr, "%v\n", warn)
	}
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		return 1
	}

	var output string
	if outputText {
		schema, err := result.Schema()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		output = idoltext.Encode(schema)
	} else {
		output, err = result.EncodedSchema()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	if cmd.outPath == "" {
		if _, err := os.Stdout.WriteString(output); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	openFlags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	fp, err := os.OpenFile(cmd.outPath, openFlags, 0o666)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, writeErr := fp.WriteString(output)
	closeErr := fp.Close()
	if writeErr != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if closeErr != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
