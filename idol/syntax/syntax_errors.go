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

package syntax

import (
	"fmt"
	"math"
	"unicode/utf8"
)

type Error struct {
	code    uint32
	message string
	span    Span
}

var _ error = (*Error)(nil)

func (err *Error) Error() string {
	return fmt.Sprintf("E%d: %s", err.code, err.message)
}

func (err *Error) Code() uint32 {
	return err.code
}

func (err *Error) Message() string {
	return err.message
}

func (err *Error) Span() Span {
	return err.span
}

func errSourceTooLong(srcLen int) error {
	lenUint32 := uint32(math.MaxUint32)
	if uint64(srcLen) < math.MaxUint32 {
		lenUint32 = uint32(srcLen)
	}
	return &Error{
		code: 1000,
		message: fmt.Sprintf(
			"Source file size (%d bytes) exceeds maximum (%d bytes)",
			srcLen, maxSrcLen,
		),
		span: Span{0, lenUint32},
	}
}

func errInvalidUtf8(src []byte) error {
	var off uint32
	for len(src) > 0 {
		r, size := utf8.DecodeRune(src)
		if r == utf8.RuneError {
			break
		}
		off += uint32(size)
		src = src[size:]
	}
	return &Error{
		code:    1001,
		message: "Source file contains invalid UTF-8",
		span:    Span{off, 1},
	}
}

func errUnexpectedCharacter(start uint32, r rune) error {
	return &Error{
		code:    1002,
		message: fmt.Sprintf("Unexpected character '%s' (U+%04X)", string(r), r),
		span:    Span{start, uint32(utf8.RuneLen(r))},
	}
}

func errForbiddenControlCharacter(start uint32, c byte) error {
	return &Error{
		code:    1003,
		message: fmt.Sprintf("Forbidden control character U+%04X", c),
		span:    Span{start, 1},
	}
}

func errTokenTooLong(start uint32, tokenLen int) error {
	lenUint32 := uint32(math.MaxUint32)
	if uint64(tokenLen) < math.MaxUint32 {
		lenUint32 = uint32(tokenLen)
	}
	return &Error{
		code: 1004,
		message: fmt.Sprintf(
			"Token size (%d bytes) exceeds maximum (%d bytes)",
			tokenLen, maxTokenLen,
		),
		span: Span{start, lenUint32},
	}
}

func errIntLitInvalid(start uint32, token []byte) error {
	tokenLen := uint32(math.MaxUint32)
	if uint64(len(token)) < math.MaxUint32 {
		tokenLen = uint32(len(token))
	}
	return &Error{
		code:    1005,
		message: fmt.Sprintf("Invalid integer literal %q", token),
		span:    Span{start, tokenLen},
	}
}

func errTextLitUnterminated(start, tokenLen uint32) error {
	return &Error{
		code:    1006,
		message: "Unterminated text literal",
		span:    Span{start, tokenLen},
	}
}

func errTextLitContainsNewline(start, newlineLen uint32) error {
	return &Error{
		code:    1007,
		message: "Text literal contains unescaped newline",
		span:    Span{start, newlineLen},
	}
}

func errIdentInvalid(start uint32, token []byte) error {
	tokenLen := uint32(math.MaxUint32)
	if uint64(len(token)) < math.MaxUint32 {
		tokenLen = uint32(len(token))
	}
	return &Error{
		code:    1008,
		message: fmt.Sprintf("Invalid identifier %q", token),
		span:    Span{start, tokenLen},
	}
}

func errExpectedSigil(
	wantKind TokenKind,
	gotKind TokenKind,
	gotToken string,
	span Span,
) error {
	var code uint32
	var want string
	switch wantKind {
	case T_AT:
		code = 2000
		want = "@"
	case T_COLON:
		code = 2001
		want = ":"
	case T_DOT:
		code = 2002
		want = "."
	case T_EQ:
		code = 2003
		want = "="
	case T_OPEN_CURL:
		code = 2004
		want = "{"
	case T_CLOSE_CURL:
		code = 2005
		want = "}"
	case T_OPEN_PAREN:
		code = 2006
		want = "("
	case T_CLOSE_PAREN:
		code = 2007
		want = ")"
	case T_OPEN_SQUARE:
		code = 2008
		want = "["
	case T_CLOSE_SQUARE:
		code = 2009
		want = "]"
	default:
		panic("unreachable")
	}
	return &Error{
		code:    code,
		message: fmt.Sprintf("Expected sigil '%s', got (%s %q)", want, gotKind, gotToken),
		span:    span,
	}
}

func errExpectedIntLit(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2010,
		message: fmt.Sprintf("Expected integer literal, got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errExpectedTextLit(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2011,
		message: fmt.Sprintf("Expected text literal, got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errExpectedIdent(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2012,
		message: fmt.Sprintf("Expected identifier, got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errExpectedKeywordAs(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2013,
		message: fmt.Sprintf("Expected keyword 'as', got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errExpectedKeywordNamespace(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2014,
		message: fmt.Sprintf("Expected keyword 'namespace', got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errExpectedDeclaration(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2015,
		message: fmt.Sprintf("Expected declaration keyword, got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errUnknownDeclaration(token string, span Span) error {
	return &Error{
		code:    2016,
		message: fmt.Sprintf("Unknown declaration keyword %q", token),
		span:    span,
	}
}

func errUnknownDecorator(token string, span Span) error {
	return &Error{
		code:    2017,
		message: fmt.Sprintf("Unknown decorator keyword %q", token),
		span:    span,
	}
}

func errExpectedTypeName(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2018,
		message: fmt.Sprintf("Expected type name, got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errExpectedValueName(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2026,
		message: fmt.Sprintf("Expected value name, got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errExpectedExportName(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2027,
		message: fmt.Sprintf("Expected export name, got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errExpectedConstValue(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2019,
		message: fmt.Sprintf("Expected const value, got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errExpectedOptionName(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2020,
		message: fmt.Sprintf("Expected option name, got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errExpectedOptionValue(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2021,
		message: fmt.Sprintf("Expected option value, got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}

func errIntLitTooPositive(token string, start uint32) error {
	return &Error{
		code: 2022,
		message: fmt.Sprintf(
			"Integer literal too positive (must be <= %d)",
			uint64(math.MaxUint64),
		),
		span: Span{
			start: start,
			len:   uint32(len(token)),
		},
	}
}

func errIntLitTooNegative(token string, start uint32) error {
	return &Error{
		code: 2023,
		message: fmt.Sprintf(
			"Integer literal too negative (must be >= %d)",
			int64(math.MinInt64),
		),
		span: Span{
			start: start,
			len:   uint32(len(token)),
		},
	}
}

func errTextLitInvalid(start uint32, token string) error {
	return &Error{
		code:    2024,
		message: fmt.Sprintf("Invalid text literal %q", token),
		span: Span{
			start: start,
			len:   uint32(len(token)),
		},
	}
}

func errExpectedProtocolItem(gotKind TokenKind, gotToken string, span Span) error {
	return &Error{
		code:    2025,
		message: fmt.Sprintf("Expected protocol item, got (%s %q)", gotKind, gotToken),
		span:    span,
	}
}
