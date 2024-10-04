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

package syntax_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"testing"

	"go.idol-lang.org/idol/internal/testutil"
	"go.idol-lang.org/idol/syntax"
)

var (
	testdata     fs.FS
	syntaxErrors map[string]*testutil.SyntaxError
)

func init() {
	var err error
	testdata, err = testutil.TestdataFS()
	if err != nil {
		panic(err)
	}
	syntaxErrors, err = testutil.LoadSyntaxErrors(testdata)
	if err != nil {
		panic(err)
	}
}

func specTest(t *testing.T, testName string) {
	t.Parallel()

	srcPath := fmt.Sprintf("syntax/%s/%s.idol", testName, testName)
	src, err := fs.ReadFile(testdata, srcPath)
	testutil.AssertNoError(t, err)

	expectOK := fmt.Sprintf("syntax/%s/expect_ok.json", testName)
	expectErr := fmt.Sprintf("syntax/%s/expect_err.json", testName)

	if _, err := fs.Stat(testdata, expectErr); err == nil {
		testExpectErr(t, srcPath, src, expectErr)
	} else {
		testExpectOK(t, srcPath, src, expectOK)
	}
}

func testExpectOK(t *testing.T, srcPath string, src []byte, expectPath string) {
	expectJSON, err := fs.ReadFile(testdata, expectPath)
	testutil.AssertNoError(t, err)
	expectJSON = bytes.Trim(expectJSON, "\n")

	schema, err := syntax.Parse(src)
	testutil.AssertNoError(t, err)

	gotJSON := string(testutil.DumpJSON(schema))
	testutil.ExpectNoDiff(t, string(expectJSON), gotJSON)
}

func testExpectErr(t *testing.T, srcPath string, src []byte, expectPath string) {
	expectJSON, err := fs.ReadFile(testdata, expectPath)
	testutil.AssertNoError(t, err)

	test := make(map[string]interface{})
	decoder := json.NewDecoder(bytes.NewReader(expectJSON))
	decoder.UseNumber()
	testutil.AssertNoError(t, decoder.Decode(&test))

	errorName := test["error"].(string)
	expectErr, ok := syntaxErrors[errorName]
	if !ok {
		t.Fatalf("unknown parse error name %q", errorName)
	}

	_, err = syntax.Parse(src)
	testutil.AssertError(t, err)

	parseErr := err.(*syntax.Error)
	testutil.ExpectEq(t, expectErr.Code(), parseErr.Code())
	if pattern := expectErr.MessagePattern(); pattern != nil {
		testutil.ExpectMatch(t, pattern, parseErr.Message())
	} else if message := expectErr.Message(); message != "" {
		testutil.ExpectEq(t, message, parseErr.Message())
	}

	expectSpan := testutil.SpanOrDie(t, test["error_span"])
	testutil.ExpectEq(t, expectSpan, parseErr.Span())
}

func TestSyntax(t *testing.T) {
	t.Parallel()

	testDirs, err := fs.ReadDir(testdata, "syntax")
	testutil.AssertNoError(t, err)

	for _, testDir := range testDirs {
		if testDir.IsDir() {
			testName := testDir.Name()
			t.Run(testName, func(t *testing.T) {
				specTest(t, testName)
			})
		}
	}
}
