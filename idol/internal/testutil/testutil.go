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

package testutil

import (
	"encoding/json"
	"io/fs"
	"os"
	"strings"
	"testing"

	"go.idol-lang.org/idol/syntax"
)

var diagnosticsPaths string

func TestdataFS() (fs.FS, error) {
	const syntaxErrorsJSON string = "/diagnostics/syntax_errors.json"
	var testdataRoot string
	for _, path := range strings.Split(diagnosticsPaths, " ") {
		if root, ok := strings.CutSuffix(path, syntaxErrorsJSON); ok {
			testdataRoot = root
			break
		}
	}
	if testdataRoot == "" {
		panic("can't find testdata (invalid `diagnosticsPaths`)")
	}
	return fs.Sub(os.DirFS(os.Getenv("RUNFILES_DIR")), testdataRoot)
}

func SpanOrDie(t *testing.T, spanIface interface{}) syntax.Span {
	span := spanIface.(map[string]interface{})
	start, err := span["start"].(json.Number).Int64()
	if err != nil {
		t.Fatal(err)
	}
	len, err := span["len"].(json.Number).Int64()
	if err != nil {
		t.Fatal(err)
	}
	return syntax.NewSpan(
		uint32(start),
		uint32(len),
	)
}
