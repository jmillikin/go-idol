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

const (
	maxSrcLen   = 0x7FFFFFFF // (2**31)-1
	maxTokenLen = int(math.MaxUint16)

	tokenFlagTextHasNoEscapes uint8 = 0x01
)

type Token struct {
	Len   uint16
	Kind  TokenKind
	flags uint8
}

type TokenKind uint8

const (
	T_EOF TokenKind = iota

	T_SPACE
	T_NEWLINE
	T_COMMENT

	T_AT
	T_COLON
	T_DOT
	T_EQ

	T_OPEN_CURL
	T_CLOSE_CURL
	T_OPEN_PAREN
	T_CLOSE_PAREN
	T_OPEN_SQUARE
	T_CLOSE_SQUARE

	T_INT_LIT
	T_BIN_INT_LIT
	T_OCT_INT_LIT
	T_DEC_INT_LIT
	T_HEX_INT_LIT

	T_TEXT_LIT

	T_IDENT
)

func (k TokenKind) String() string {
	switch k {
	case T_EOF:
		return "EOF"
	case T_SPACE:
		return "SPACE"
	case T_NEWLINE:
		return "NEWLINE"
	case T_COMMENT:
		return "COMMENT"
	case T_AT:
		return "AT"
	case T_COLON:
		return "COLON"
	case T_DOT:
		return "DOT"
	case T_EQ:
		return "EQ"
	case T_OPEN_CURL:
		return "OPEN_CURL"
	case T_CLOSE_CURL:
		return "CLOSE_CURL"
	case T_OPEN_PAREN:
		return "OPEN_PAREN"
	case T_CLOSE_PAREN:
		return "CLOSE_PAREN"
	case T_OPEN_SQUARE:
		return "OPEN_SQUARE"
	case T_CLOSE_SQUARE:
		return "CLOSE_SQUARE"
	case T_INT_LIT:
		return "INT_LIT"
	case T_BIN_INT_LIT:
		return "BIN_INT_LIT"
	case T_OCT_INT_LIT:
		return "OCT_INT_LIT"
	case T_DEC_INT_LIT:
		return "DEC_INT_LIT"
	case T_HEX_INT_LIT:
		return "HEX_INT_LIT"
	case T_TEXT_LIT:
		return "TEXT_LIT"
	case T_IDENT:
		return "IDENT"
	default:
		return fmt.Sprintf("TokenKind(%d)", uint8(k))
	}
}

type Tokens struct {
	src    []byte
	offset uint32
}

func NewTokens(src []byte) (*Tokens, error) {
	if len(src) > maxSrcLen {
		return nil, errSourceTooLong(len(src))
	}
	if !utf8.Valid(src) {
		return nil, errInvalidUtf8(src)
	}
	return &Tokens{
		src: src,
	}, nil
}

func (t *Tokens) Next(token *Token) error {
	if len(t.src) == 0 {
		*token = Token{
			Kind: T_EOF,
		}
		return nil
	}

	c := t.src[0]
	var kind TokenKind
	switch c {
	case '\t', ' ':
		return t.nextSpace(token)
	case '\n':
		kind = T_NEWLINE
		goto len1
	case '@':
		kind = T_AT
		goto len1
	case ':':
		kind = T_COLON
		goto len1
	case '.':
		kind = T_DOT
		goto len1
	case '=':
		kind = T_EQ
		goto len1
	case '{':
		kind = T_OPEN_CURL
		goto len1
	case '}':
		kind = T_CLOSE_CURL
		goto len1
	case '(':
		kind = T_OPEN_PAREN
		goto len1
	case ')':
		kind = T_CLOSE_PAREN
		goto len1
	case '[':
		kind = T_OPEN_SQUARE
		goto len1
	case ']':
		kind = T_CLOSE_SQUARE
		goto len1
	case '#':
		return t.nextComment(token)
	case '"':
		return t.nextTextLit(token)
	case '\r':
		if len(t.src) < 2 || t.src[1] != '\n' {
			return errForbiddenControlCharacter(t.offset, c)
		}
		*token = Token{
			Kind: T_NEWLINE,
			Len:  2,
		}
		t.offset += 2
		t.src = t.src[2:]
		return nil
	default:
		goto big
	}

len1:
	*token = Token{
		Kind: kind,
		Len:  1,
	}
	t.offset += 1
	t.src = t.src[1:]
	return nil

big:
	if (c >= '0' && c <= '9') || c == '-' {
		return t.nextNumLit(token)
	}

	if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
		return t.nextIdent(token)
	}

	r, _ := utf8.DecodeRune(t.src)
	if r == '\u00A0' {
		return t.nextSpace(token)
	}

	if r < 0x20 || r == 0x7F {
		return errForbiddenControlCharacter(t.offset, c)
	}
	return errUnexpectedCharacter(t.offset, r)
}

func (t *Tokens) nextSpace(token *Token) error {
	src := t.src
	for {
		if src[0] == ' ' || src[0] == '\t' {
			src = src[1:]
		} else if r, runeLen := utf8.DecodeRune(src); r == '\u00A0' {
			src = src[runeLen:]
		} else {
			break
		}
		if len(src) == 0 {
			break
		}
	}
	tokenLen, err := t.checkTokenLen(len(t.src) - len(src))
	if err != nil {
		return err
	}
	*token = Token{
		Kind: T_SPACE,
		Len:  tokenLen,
	}
	t.offset += uint32(tokenLen)
	t.src = src
	return nil
}

func (t *Tokens) nextComment(token *Token) error {
	src := t.src
	for ii, c := range src {
		if c == '\n' || c == '\r' {
			src = src[:ii]
			break
		}
	}

	tokenLen := len(src)
	if tokenLen, err := t.checkTokenLen(tokenLen); err != nil {
		return err
	} else {
		*token = Token{
			Kind: T_COMMENT,
			Len:  tokenLen,
		}
	}
	t.offset += uint32(tokenLen)
	t.src = t.src[tokenLen:]
	return nil
}

func (t *Tokens) nextNumLit(token *Token) error {
	numSrc := t.src

	tokenLen := 0
	neg := false
	if numSrc[0] == '-' {
		if len(numSrc) == 1 {
			return errIntLitInvalid(t.offset, t.src[:1])
		}
		tokenLen += 1
		numSrc = numSrc[1:]
		neg = true
	}

	kind := T_INT_LIT
	invalid := false
	if numSrc[0] == '0' {
		if len(numSrc) == 1 {
			if neg {
				return errIntLitInvalid(t.offset, t.src[:2])
			}
			*token = Token{
				Kind: T_INT_LIT,
				Len:  1,
			}
			t.offset += 1
			t.src = t.src[1:]
			return nil
		}

		switch base := numSrc[1]; base {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			invalid = true
		case 'b':
			kind = T_BIN_INT_LIT
		case 'o':
			kind = T_OCT_INT_LIT
		case 'd':
			kind = T_DEC_INT_LIT
		case 'x':
			kind = T_HEX_INT_LIT
		default:
			invalid = true
		}
		if kind != T_INT_LIT {
			tokenLen += 2
			numSrc = numSrc[2:]
		}
	}

	switch kind {
	case T_INT_LIT, T_DEC_INT_LIT:
		for ii, c := range numSrc {
			if (c >= '0' && c <= '9') || c == '_' {
				continue
			}
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				invalid = true
				continue
			}
			numSrc = numSrc[:ii]
			break
		}
	case T_HEX_INT_LIT:
		for ii, c := range numSrc {
			if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f') || c == '_' {
				continue
			}
			if (c >= 'G' && c <= 'Z') || (c >= 'g' && c <= 'z') {
				invalid = true
				continue
			}
			numSrc = numSrc[:ii]
			break
		}
	case T_OCT_INT_LIT:
		for ii, c := range numSrc {
			if (c >= '0' && c <= '7') || c == '_' {
				continue
			}
			if c == '8' || c == '9' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				invalid = true
				continue
			}
			numSrc = numSrc[:ii]
			break
		}
	case T_BIN_INT_LIT:
		for ii, c := range numSrc {
			if c == '0' || c == '1' {
				continue
			}
			if (c >= '2' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_' {
				invalid = true
				continue
			}
			numSrc = numSrc[:ii]
			break
		}
	default:
		invalid = true
		for ii, c := range numSrc {
			if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'Z') || c == '_' {
				continue
			}
			numSrc = numSrc[:ii]
			break
		}
	}

	if len(numSrc) == 0 {
		invalid = true
	} else {
		tokenLen += len(numSrc)
		if tokenLen == 1 && numSrc[0] == '0' {
			invalid = false
		}
	}
	if invalid {
		return errIntLitInvalid(t.offset, t.src[:tokenLen])
	}

	if tokenLen, err := t.checkTokenLen(tokenLen); err != nil {
		return err
	} else {
		*token = Token{
			Kind: kind,
			Len:  tokenLen,
		}
	}
	t.offset += uint32(tokenLen)
	t.src = t.src[tokenLen:]
	return nil
}

func (t *Tokens) nextTextLit(token *Token) error {
	src := t.src
	escaped := false
	hasEscapes := false
	ok := false
	var flags uint8
	for ii, c := range t.src {
		if ii == 0 {
			continue
		}
		if escaped {
			escaped = false
			continue
		}
		if c == '"' {
			src = t.src[:ii+1]
			ok = true
			hasEscapes = true
			break
		}
		if (c <= 0x1F || c == 0x7F) && c != 0x09 {
			off := t.offset + uint32(ii)
			if c == 0x0A {
				return errTextLitContainsNewline(off, 1)
			}
			if c == 0x0D && ii+1 < len(t.src) && t.src[ii+1] == 0x0A {
				return errTextLitContainsNewline(off, 2)
			}
			return errForbiddenControlCharacter(off, c)
		}
		escaped = c == '\\'
	}
	if !ok {
		return errTextLitUnterminated(t.offset, uint32(len(src)))
	}

	if !hasEscapes {
		flags |= tokenFlagTextHasNoEscapes
	}

	tokenLen := len(src)
	if tokenLen, err := t.checkTokenLen(tokenLen); err != nil {
		return err
	} else {
		*token = Token{
			Kind:  T_TEXT_LIT,
			Len:   tokenLen,
			flags: flags,
		}
	}
	t.offset += uint32(tokenLen)
	t.src = t.src[tokenLen:]
	return nil
}

func (t *Tokens) nextIdent(token *Token) error {
	src := t.src
	underscore := false
	invalid := false
	for ii, c := range src {
		if ii == 0 {
			continue
		}
		if c == '_' {
			if underscore {
				invalid = true
			}
			underscore = true
			continue
		}
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			underscore = false
			continue
		}
		src = src[:ii]
		break
	}

	if underscore || invalid {
		return errIdentInvalid(t.offset, src)
	}

	tokenLen := len(src)
	if tokenLen, err := t.checkTokenLen(tokenLen); err != nil {
		return err
	} else {
		*token = Token{
			Kind: T_IDENT,
			Len:  tokenLen,
		}
	}
	t.offset += uint32(tokenLen)
	t.src = t.src[tokenLen:]
	return nil
}

func (t *Tokens) checkTokenLen(len int) (uint16, error) {
	if len > maxTokenLen {
		return 0, errTokenTooLong(t.offset, len)
	}
	return uint16(len), nil
}
