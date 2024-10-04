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
	"log"
	"os"
	"path/filepath"
	"slices"

	"go.idol-lang.org/idol"
	"go.idol-lang.org/idol/compiler"
	"go.idol-lang.org/idol/schema_idl"
	"go.idol-lang.org/idol/syntax"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		log.Fatalf("usage: %s IDOL_SCHEMA", os.Args[0])
	}
	schemaPath := args[0]

	src, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatalf("ReadFile(%q): %v", schemaPath, err)
	}

	parsed, err := syntax.Parse(src)
	if err != nil {
		log.Fatalf("Parse(%q): %v", schemaPath, err)
	}

	var opts []compiler.CompileOption
	if !filepath.IsAbs(schemaPath) {
		opts = append(opts, compiler.WithSourcePath(splitPath(schemaPath)))
	}
	compiled := compiler.Compile(parsed, opts...)
	if len(compiled.Warnings) > 0 {
		for _, err := range compiled.Warnings {
			log.Printf("[WARN ] %v", err)
		}
	}
	if len(compiled.Errors) > 0 {
		for _, err := range compiled.Errors {
			log.Printf("[ERROR] %v", err)
		}
		os.Exit(1)
	}

	schemaData, err := compiled.EncodedSchema()
	if err != nil {
		log.Fatal(err)
	}

	schema, err := idol.DecodeAs[schema_idl.Schema](nil, []byte(schemaData))
	if err != nil {
		log.Fatal(err)
	}
	c := codegen{
		schema: schema,
	}
	if err := c.emitSchema(); err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(c.output); err != nil {
		log.Fatal(err)
	}
}

func splitPath(path string) []string {
	var out []string
	for {
		dir, file := filepath.Split(path)
		if dir == "" {
			out = append(out, file)
			slices.Reverse(out)
			return out
		}
		out = append(out, file)
		path = dir[:len(dir)-1]
	}
}
