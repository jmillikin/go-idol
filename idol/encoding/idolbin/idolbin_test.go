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

package idolbin_test

import (
	"iter"
	"testing"

	"go.idol-lang.org/idol/encoding/idolbin"
	"go.idol-lang.org/idol/internal/testutil"
)

func collectSeq2[K, V any](seq iter.Seq2[K, V]) []V {
	var vs []V
	for _, v := range seq {
		vs = append(vs, v)
	}
	return vs
}

func TestMessage_TextArray(t *testing.T) {
	messageBuf := string([]uint8{
		48, 0, 0, 0,
		0, 0, 1, 0,

		0, 0, 0, 0b11000000, 32, 0, 0, 0,

		3, 0, 0, 0,
		6, 0, 0, 0,
		3, 0, 0, 0,
		7, 0, 0, 0,
		72, 101, 108, 108, 111, 0,
		44, 32, 0,
		119, 111, 114, 108, 100, 33, 0,
	})

	message, err := idolbin.NewMessage(messageBuf)
	testutil.AssertNoError(t, err)

	values := []string{
		"Hello",
		", ",
		"world!",
	}

	field := message.Field(1)
	array, err := field.GetTextArray()
	testutil.AssertNoError(t, err)

	testutil.ExpectSliceEq(t, values, collectSeq2(array.Iter()))
}
