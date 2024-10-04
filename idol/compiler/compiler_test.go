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

package compiler_test

import (
	"fmt"
	"io/fs"
	"iter"
	"strings"
	"testing"

	"go.idol-lang.org/idol/compiler"
	"go.idol-lang.org/idol/encoding/idoltext"
	"go.idol-lang.org/idol/internal/testutil"
	"go.idol-lang.org/idol/schema_idl"
	"go.idol-lang.org/idol/syntax"
)

var (
	testdata       fs.FS
	schemaErrors   map[string]*testutil.SchemaError
	schemaWarnings map[string]*testutil.SchemaWarning
)

func init() {
	var err error
	testdata, err = testutil.TestdataFS()
	if err != nil {
		panic(err)
	}
	schemaErrors, err = testutil.LoadSchemaErrors(testdata)
	if err != nil {
		panic(err)
	}
	schemaWarnings, err = testutil.LoadSchemaWarnings(testdata)
	if err != nil {
		panic(err)
	}
}

func specTest(t *testing.T, testName string) {
	t.Parallel()

	expectOK := fmt.Sprintf("schema/%s/expect_ok.txt", testName)
	expectErr := fmt.Sprintf("schema/%s/expect_err.json", testName)

	if _, err := fs.Stat(testdata, expectErr); err == nil {
		testExpectErr(t, testName, expectErr)
	} else {
		testExpectOK(t, testName, expectOK)
	}
}

func testExpectOK(t *testing.T, testName string, expectOK string) {
	expectText, err := fs.ReadFile(testdata, expectOK)
	testutil.AssertNoError(t, err)

	var expectWarnings []*testutil.ExpectedWarning
	expectWarnPath := fmt.Sprintf("schema/%s/expect_warn.json", testName)
	if _, err := fs.Stat(testdata, expectWarnPath); err == nil {
		expectWarnings = testutil.LoadExpectedWarnings(
			t, schemaWarnings, testdata, expectWarnPath,
		)
	}

	result := compileTestInputs(t, testName)
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			testutil.ExpectNoError(t, err)
		}
		t.FailNow()
	}

	for warn, expectWarn := range zip(result.Warnings, expectWarnings) {
		if warn == nil {
			warnName := expectWarn.Message
			if warnName == "" {
				warnName = expectWarn.Key
			}
			t.Errorf(
				"expected schema warning %q (code %d)",
				warnName,
				expectWarn.Code,
			)
			continue
		}
		if expectWarn == nil {
			t.Errorf(
				"unexpected schema warning %q (code %d)",
				warn.Message(),
				warn.Code(),
			)
			continue
		}
		testutil.ExpectEq(t, expectWarn.Code, warn.Code())
		if expectWarn.Pattern != nil {
			testutil.ExpectMatch(t, expectWarn.Pattern, warn.Message())
		} else if expectWarn.Message != "" {
			testutil.ExpectEq(t, expectWarn.Message, warn.Message())
		}
		testutil.ExpectEq(t, expectWarn.Span, warn.Span())
	}

	expectBinPath := fmt.Sprintf("schema/%s/expect_ok.idolbin", testName)
	if _, err := fs.Stat(testdata, expectBinPath); err == nil {
		expectBin, err := fs.ReadFile(testdata, expectBinPath)
		testutil.AssertNoError(t, err)
		schemaData, err := result.EncodedSchema()
		testutil.AssertNoError(t, err)
		testutil.ExpectBytesEq(t, expectBin, []byte(schemaData))
	}

	schema, err := result.Schema()
	testutil.AssertNoError(t, err)

	gotText := string(idoltext.Encode(schema))
	testutil.ExpectNoDiff(t, string(expectText), gotText)
}

func testExpectErr(t *testing.T, testName string, expectErrPath string) {
	expectErrors := testutil.LoadExpectedErrors(
		t, schemaErrors, testdata, expectErrPath,
	)
	if len(expectErrors) == 0 {
		t.Fatalf("len(expectErrors) == 0")
	}

	result := compileTestInputs(t, testName)
	for err, expectErr := range zip(result.Errors, expectErrors) {
		if err == nil {
			errName := expectErr.Message
			if errName == "" {
				errName = expectErr.Key
			}
			t.Errorf(
				"expected schema error %q (code %d)",
				errName,
				expectErr.Code,
			)
			continue
		}
		if expectErr == nil {
			t.Errorf(
				"unexpected schema error %q (code %d)",
				err.Message(),
				err.Code(),
			)
			continue
		}
		testutil.ExpectEq(t, expectErr.Code, err.Code())
		if expectErr.Pattern != nil {
			testutil.ExpectMatch(t, expectErr.Pattern, err.Message())
		} else if expectErr.Message != "" {
			testutil.ExpectEq(t, expectErr.Message, err.Message())
		}
		testutil.ExpectEq(t, expectErr.Span, err.Span())
	}
}

func compileTestInputs(t *testing.T, testName string) compiler.CompileResult {
	testSrcs, err := fs.ReadDir(testdata, fmt.Sprintf("schema/%s", testName))
	testutil.AssertNoError(t, err)

	srcPath := fmt.Sprintf("schema/%s/%s.idol", testName, testName)
	src, err := fs.ReadFile(testdata, srcPath)
	testutil.AssertNoError(t, err)

	var deps []schema_idl.Schema
	for _, fileEntry := range testSrcs {
		fileName := fileEntry.Name()
		if !strings.HasSuffix(fileName, ".idol") {
			continue
		}
		if fileName == testName+".idol" {
			continue
		}

		filePath := fmt.Sprintf("schema/%s/%s", testName, fileName)
		fileContent, err := fs.ReadFile(testdata, filePath)
		testutil.AssertNoError(t, err)

		parsedSchema, err := syntax.Parse(fileContent)
		testutil.AssertNoError(t, err)
		result := compiler.Compile(parsedSchema)
		if len(result.Errors) > 0 {
			for _, err := range result.Errors {
				testutil.ExpectNoError(t, err)
			}
			t.FailNow()
		}

		schema, err := result.Schema()
		testutil.AssertNoError(t, err)
		deps = append(deps, schema)
	}

	var compileOpts []compiler.CompileOption
	if len(deps) > 0 {
		mergedDeps, err := compiler.Merge(deps)
		testutil.AssertNoError(t, err)
		compileOpts = append(compileOpts, compiler.WithDependencies(mergedDeps))
	}

	parsedSchema, err := syntax.Parse(src)
	testutil.AssertNoError(t, err)
	return compiler.Compile(parsedSchema, compileOpts...)
}

func TestSchema(t *testing.T) {
	t.Parallel()

	testDirs, err := fs.ReadDir(testdata, "schema")
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

func zip[X any, Y any](xs []*X, ys []*Y) iter.Seq2[*X, *Y] {
	maxLen := max(len(xs), len(ys))
	return func(yield func(x *X, y *Y) bool) {
		for ii := 0; ii < maxLen; ii++ {
			var ok bool
			if ii >= len(xs) {
				ok = yield(nil, ys[ii])
			} else if ii >= len(ys) {
				ok = yield(xs[ii], nil)
			} else {
				ok = yield(xs[ii], ys[ii])
			}
			if !ok {
				return
			}
		}
	}
}
