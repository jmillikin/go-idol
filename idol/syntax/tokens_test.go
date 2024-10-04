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

type strToken struct {
	kind string
	content string
}

func specTest(t *testing.T, testName string) {
	t.Parallel()

	testsPath := fmt.Sprintf("tokens/%s.json", testName)
	t.Logf("reading test cases from %q", "spec/testdata/"+testsPath)

	testsJSON, err := fs.ReadFile(testdata, testsPath)
	testutil.AssertNoError(t, err)

	tests := make(map[string][]map[string]interface{})
	decoder := json.NewDecoder(bytes.NewReader(testsJSON))
	decoder.UseNumber()
	testutil.AssertNoError(t, decoder.Decode(&tests))

	for ii, test := range tests["expect_ok"] {
		src := test["source"].(string)
		tokensIfaces := test["tokens"].([]interface{})
		tokens := make([]strToken, 0, len(tokensIfaces))
		for _, iface := range tokensIfaces {
			raw := iface.([]interface{})
			tokens = append(tokens, strToken{
				kind: raw[0].(string),
				content: raw[1].(string),
			})
		}
		t.Run(fmt.Sprintf("expect_ok/%d", ii), func(t *testing.T) {
			testExpectOK(t, src, tokens)
		})
	}

	for ii, test := range tests["expect_err"] {
		t.Run(fmt.Sprintf("expect_err/%d", ii), func(t *testing.T) {
			testExpectErr(t, test)
		})
	}
}

func testExpectOK(t *testing.T, src string, want []strToken) {
	t.Logf("source: %q", src)

	tokens, err := syntax.NewTokens([]byte(src))
	testutil.AssertNoError(t, err)

	var got []strToken
	for {
		var token syntax.Token
		testutil.AssertNoError(t, tokens.Next(&token))
		if token.Kind == syntax.T_EOF {
			break
		}
		got = append(got, strToken{
			kind: token.Kind.String(),
			content: string(src[:token.Len]),
		})
		src = src[token.Len:]
	}

	testutil.ExpectSliceEq(t, want, got)
}

func testExpectErr(t *testing.T, test map[string]interface{}) {
	src := test["source"].(string)
	t.Logf("source: %q", src)

	errorName := test["error"].(string)
	expectErr, ok := syntaxErrors[errorName]
	if !ok {
		t.Fatalf("unknown parse error name %q", errorName)
	}

	tokens, err := syntax.NewTokens([]byte(src))
	testutil.AssertNoError(t, err)

	for {
		var token syntax.Token
		err = tokens.Next(&token)
		if err != nil || token.Kind == syntax.T_EOF {
			break
		}
	}
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

func TestComments(t *testing.T) {
	specTest(t, "comments")
}

func TestIdents(t *testing.T) {
	specTest(t, "idents")
}

func TestIntLiterals(t *testing.T) {
	specTest(t, "int_literals")
}

func TestMiscErrors(t *testing.T) {
	specTest(t, "misc_errors")
}

func TestNewlines(t *testing.T) {
	specTest(t, "newlines")
}

func TestSigils(t *testing.T) {
	specTest(t, "sigils")
}

func TestSpaces(t *testing.T) {
	specTest(t, "spaces")
}

func TestTextLiterals(t *testing.T) {
	specTest(t, "text_literals")
}

func TestTokenKindStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind syntax.TokenKind
		want string
	}{
		{syntax.T_EOF, "EOF"},
		{syntax.T_SPACE, "SPACE"},
		{syntax.T_NEWLINE, "NEWLINE"},
		{syntax.T_COMMENT, "COMMENT"},
		{syntax.T_AT, "AT"},
		{syntax.T_COLON, "COLON"},
		{syntax.T_DOT, "DOT"},
		{syntax.T_EQ, "EQ"},
		{syntax.T_OPEN_CURL, "OPEN_CURL"},
		{syntax.T_CLOSE_CURL, "CLOSE_CURL"},
		{syntax.T_OPEN_PAREN, "OPEN_PAREN"},
		{syntax.T_CLOSE_PAREN, "CLOSE_PAREN"},
		{syntax.T_OPEN_SQUARE, "OPEN_SQUARE"},
		{syntax.T_CLOSE_SQUARE, "CLOSE_SQUARE"},
		{syntax.T_INT_LIT, "INT_LIT"},
		{syntax.T_BIN_INT_LIT, "BIN_INT_LIT"},
		{syntax.T_OCT_INT_LIT, "OCT_INT_LIT"},
		{syntax.T_DEC_INT_LIT, "DEC_INT_LIT"},
		{syntax.T_HEX_INT_LIT, "HEX_INT_LIT"},
		{syntax.T_TEXT_LIT, "TEXT_LIT"},
		{syntax.T_IDENT, "IDENT"},
		{syntax.TokenKind(255), "TokenKind(255)"},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			testutil.ExpectEq(t, test.want, test.kind.String())
		})
	}
}
