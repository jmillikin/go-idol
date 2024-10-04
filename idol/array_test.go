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

package idol_test

import (
	"iter"
	"testing"
	"unsafe"

	"go.idol-lang.org/idol"
	"go.idol-lang.org/idol/internal/testutil"
)

func castBuf[T any](buf []uint8) T {
	frozen := string(buf)
	return *(*T)(unsafe.Pointer(&frozen))
}

func collectSeq2[K, V any](seq iter.Seq2[K, V]) []V {
	vs := []V{}
	for _, v := range seq {
		vs = append(vs, v)
	}
	return vs
}

func TestBoolArray(t *testing.T) {
	t.Parallel()

	array := castBuf[idol.BoolArray]([]uint8{0, 1, 0, 1, 2})
	values := []bool{
		false,
		true,
		false,
		true,
		true,
	}
	arrayStr := "[.false, .true, .false, .true, .true]"

	testutil.ExpectEq(t, uint32(len(values)), array.Len())
	testutil.ExpectSliceEq(t, values, array.Collect())
	testutil.ExpectSliceEq(t, values, collectSeq2(array.Iter()))
	testutil.ExpectEq(t, arrayStr, array.String())

	for ii, value := range values {
		got, ok := array.Get(uint32(ii))
		if testutil.ExpectTrue(t, ok); ok {
			testutil.ExpectEq(t, value, got)
		}
	}

	{
		_, ok := array.Get(999)
		testutil.ExpectFalse(t, ok)
	}
}

func TestUint8Array(t *testing.T) {
	t.Parallel()

	array := castBuf[idol.Uint8Array]([]uint8{
		0x01, 0x02,
		0x03, 0x04,
		0x05, 0x06,
	})
	values := []uint8{
		0x01, 0x02,
		0x03, 0x04,
		0x05, 0x06,
	}
	arrayStr := "[1, 2, 3, 4, 5, 6]"

	testutil.ExpectEq(t, uint32(len(values)), array.Len())
	testutil.ExpectSliceEq(t, values, array.Collect())
	testutil.ExpectSliceEq(t, values, collectSeq2(array.Iter()))
	testutil.ExpectEq(t, arrayStr, array.String())

	for ii, value := range values {
		got, ok := array.Get(uint32(ii))
		if testutil.ExpectTrue(t, ok); ok {
			testutil.ExpectEq(t, value, got)
		}
	}

	{
		_, ok := array.Get(999)
		testutil.ExpectFalse(t, ok)
	}
}

func TestInt8Array(t *testing.T) {
	t.Parallel()

	array := castBuf[idol.Int8Array]([]uint8{
		0x01, 0xFE,
		0x03, 0xFC,
		0x05, 0xFA,
	})
	values := []int8{
		1, -2,
		3, -4,
		5, -6,
	}
	arrayStr := "[1, -2, 3, -4, 5, -6]"

	testutil.ExpectEq(t, uint32(len(values)), array.Len())
	testutil.ExpectSliceEq(t, values, array.Collect())
	testutil.ExpectSliceEq(t, values, collectSeq2(array.Iter()))
	testutil.ExpectEq(t, arrayStr, array.String())

	for ii, value := range values {
		got, ok := array.Get(uint32(ii))
		if testutil.ExpectTrue(t, ok); ok {
			testutil.ExpectEq(t, value, got)
		}
	}

	{
		_, ok := array.Get(999)
		testutil.ExpectFalse(t, ok)
	}
}

func TestUint16Array(t *testing.T) {
	t.Parallel()

	array := castBuf[idol.Uint16Array]([]uint8{
		0x01, 0x02,
		0x03, 0x04,
		0x05, 0x06,
	})
	values := []uint16{
		0x0201,
		0x0403,
		0x0605,
	}
	arrayStr := "[513, 1027, 1541]"

	testutil.ExpectEq(t, uint32(len(values)), array.Len())
	testutil.ExpectSliceEq(t, values, array.Collect())
	testutil.ExpectSliceEq(t, values, collectSeq2(array.Iter()))
	testutil.ExpectEq(t, arrayStr, array.String())

	for ii, value := range values {
		got, ok := array.Get(uint32(ii))
		if testutil.ExpectTrue(t, ok); ok {
			testutil.ExpectEq(t, value, got)
		}
	}

	{
		_, ok := array.Get(999)
		testutil.ExpectFalse(t, ok)
	}
}

func TestInt16Array(t *testing.T) {
	t.Parallel()

	array := castBuf[idol.Int16Array]([]uint8{
		0x01, 0x02,
		0xFD, 0xFB,
		0x05, 0x06,
	})
	values := []int16{
		0x0201,
		-0x0403,
		0x0605,
	}
	arrayStr := "[513, -1027, 1541]"

	testutil.ExpectEq(t, uint32(len(values)), array.Len())
	testutil.ExpectSliceEq(t, values, array.Collect())
	testutil.ExpectSliceEq(t, values, collectSeq2(array.Iter()))
	testutil.ExpectEq(t, arrayStr, array.String())

	for ii, value := range values {
		got, ok := array.Get(uint32(ii))
		if testutil.ExpectTrue(t, ok); ok {
			testutil.ExpectEq(t, value, got)
		}
	}

	{
		_, ok := array.Get(999)
		testutil.ExpectFalse(t, ok)
	}
}

func TestUint32Array(t *testing.T) {
	t.Parallel()

	array := castBuf[idol.Uint32Array]([]uint8{
		0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C,
	})
	values := []uint32{
		0x04030201,
		0x08070605,
		0x0C0B0A09,
	}
	arrayStr := "[67305985, 134678021, 202050057]"

	testutil.ExpectEq(t, uint32(len(values)), array.Len())
	testutil.ExpectSliceEq(t, values, array.Collect())
	testutil.ExpectSliceEq(t, values, collectSeq2(array.Iter()))
	testutil.ExpectEq(t, arrayStr, array.String())

	for ii, value := range values {
		got, ok := array.Get(uint32(ii))
		if testutil.ExpectTrue(t, ok); ok {
			testutil.ExpectEq(t, value, got)
		}
	}

	{
		_, ok := array.Get(999)
		testutil.ExpectFalse(t, ok)
	}
}

func TestInt32Array(t *testing.T) {
	t.Parallel()

	array := castBuf[idol.Int32Array]([]uint8{
		0x01, 0x02, 0x03, 0x04,
		0xFB, 0xF9, 0xF8, 0xF7,
		0x09, 0x0A, 0x0B, 0x0C,
	})
	values := []int32{
		0x04030201,
		-0x08070605,
		0x0C0B0A09,
	}
	arrayStr := "[67305985, -134678021, 202050057]"

	testutil.ExpectEq(t, uint32(len(values)), array.Len())
	testutil.ExpectSliceEq(t, values, array.Collect())
	testutil.ExpectSliceEq(t, values, collectSeq2(array.Iter()))
	testutil.ExpectEq(t, arrayStr, array.String())

	for ii, value := range values {
		got, ok := array.Get(uint32(ii))
		if testutil.ExpectTrue(t, ok); ok {
			testutil.ExpectEq(t, value, got)
		}
	}

	{
		_, ok := array.Get(999)
		testutil.ExpectFalse(t, ok)
	}
}

func TestUint64Array(t *testing.T) {
	t.Parallel()

	array := castBuf[idol.Uint64Array]([]uint8{
		0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C,
		0x0D, 0x0E, 0x0F, 0x10,
		0x11, 0x12, 0x13, 0x14,
		0x15, 0x16, 0x17, 0x18,
	})
	values := []uint64{
		0x0807060504030201,
		0x100F0E0D0C0B0A09,
		0x1817161514131211,
	}
	arrayStr := "[578437695752307201, 1157159078456920585, 1735880461161533969]"

	testutil.ExpectEq(t, uint32(len(values)), array.Len())
	testutil.ExpectSliceEq(t, values, array.Collect())
	testutil.ExpectSliceEq(t, values, collectSeq2(array.Iter()))
	testutil.ExpectEq(t, arrayStr, array.String())

	for ii, value := range values {
		got, ok := array.Get(uint32(ii))
		if testutil.ExpectTrue(t, ok); ok {
			testutil.ExpectEq(t, value, got)
		}
	}

	{
		_, ok := array.Get(999)
		testutil.ExpectFalse(t, ok)
	}
}

func TestInt64Array(t *testing.T) {
	t.Parallel()

	array := castBuf[idol.Int64Array]([]uint8{
		0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08,
		0xF7, 0xF5, 0xF4, 0xF3,
		0xF2, 0xF1, 0xF0, 0xEF,
		0x11, 0x12, 0x13, 0x14,
		0x15, 0x16, 0x17, 0x18,
	})
	values := []int64{
		0x0807060504030201,
		-0x100F0E0D0C0B0A09,
		0x1817161514131211,
	}
	arrayStr := "[578437695752307201, -1157159078456920585, 1735880461161533969]"

	testutil.ExpectEq(t, uint32(len(values)), array.Len())
	testutil.ExpectSliceEq(t, values, array.Collect())
	testutil.ExpectSliceEq(t, values, collectSeq2(array.Iter()))
	testutil.ExpectEq(t, arrayStr, array.String())

	for ii, value := range values {
		got, ok := array.Get(uint32(ii))
		if testutil.ExpectTrue(t, ok); ok {
			testutil.ExpectEq(t, value, got)
		}
	}

	{
		_, ok := array.Get(999)
		testutil.ExpectFalse(t, ok)
	}
}

func TestAscizArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		buf          []uint8
		expectValues []string
		expectStr    string
	}{
		{
			buf:          []uint8{},
			expectValues: []string{},
			expectStr:    `[]`,
		},
		{
			buf: []uint8{
				3, 0, 0, 0,
				6, 0, 0, 0,
				3, 0, 0, 0,
				7, 0, 0, 0,
				72, 101, 108, 108, 111, 0,
				44, 32, 0,
				119, 111, 114, 108, 100, 33, 0,
			},
			expectValues: []string{
				"Hello\x00",
				", \x00",
				"world!\x00",
			},
			expectStr: `["Hello", ", ", "world!"]`,
		},
		{
			buf: []uint8{
				4, 0, 0, 0,
				6, 0, 0, 0,
				0, 0, 0, 0,
				3, 0, 0, 0,
				7, 0, 0, 0,
				72, 101, 108, 108, 111, 0,
				44, 32, 0,
				119, 111, 114, 108, 100, 33, 0,
			},
			expectValues: []string{
				"Hello\x00",
				"\x00",
				", \x00",
				"world!\x00",
			},
			expectStr: `["Hello", "", ", ", "world!"]`,
		},
		{
			buf: []uint8{
				1, 0, 0, 0,
				20, 0, 0, 0,
				1, 32, 9, 32, 10, 32, 25, 32,
				34, 32, 92, 32, 127, 32, 194, 128,
				32, 195, 166, 0, 0, 0, 0, 0,
			},
			expectValues: []string{
				"\x01 \x09 \x0A \x19 \x22 \x5C \x7F \u0080 \u00E6\x00",
			},
			expectStr: `["\x01 \t \n \x19 \" \\ \x7F \xC2\x80 \xC3\xA6"]`,
		},
	}

	for _, test := range tests {
		array := castBuf[idol.AscizArray](test.buf)

		testutil.ExpectEq(t, uint32(len(test.expectValues)), array.Len())
		testutil.ExpectSliceEq(t, test.expectValues, array.Collect())
		testutil.ExpectSliceEq(t, test.expectValues, collectSeq2(array.Iter()))
		testutil.ExpectEq(t, test.expectStr, array.String())

		for ii, value := range test.expectValues {
			got, ok := array.Get(uint32(ii))
			if testutil.ExpectTrue(t, ok); ok {
				testutil.ExpectEq(t, value, got)
			}
		}

		{
			_, ok := array.Get(999)
			testutil.ExpectFalse(t, ok)
		}
	}
}

func TestTextArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		buf          []uint8
		expectValues []string
		expectStr    string
	}{
		{
			buf:          []uint8{},
			expectValues: []string{},
			expectStr:    `[]`,
		},
		{
			buf: []uint8{
				3, 0, 0, 0,
				6, 0, 0, 0,
				3, 0, 0, 0,
				7, 0, 0, 0,
				72, 101, 108, 108, 111, 0,
				44, 32, 0,
				119, 111, 114, 108, 100, 33, 0,
			},
			expectValues: []string{
				"Hello",
				", ",
				"world!",
			},
			expectStr: `["Hello", ", ", "world!"]`,
		},
		{
			buf: []uint8{
				4, 0, 0, 0,
				6, 0, 0, 0,
				0, 0, 0, 0,
				3, 0, 0, 0,
				7, 0, 0, 0,
				72, 101, 108, 108, 111, 0,
				44, 32, 0,
				119, 111, 114, 108, 100, 33, 0,
			},
			expectValues: []string{
				"Hello",
				"",
				", ",
				"world!",
			},
			expectStr: `["Hello", "", ", ", "world!"]`,
		},
		{
			buf: []uint8{
				1, 0, 0, 0,
				20, 0, 0, 0,
				1, 32, 9, 32, 10, 32, 25, 32,
				34, 32, 92, 32, 127, 32, 194, 128,
				32, 195, 166, 0, 0, 0, 0, 0,
			},
			expectValues: []string{
				"\x01 \x09 \x0A \x19 \x22 \x5C \x7F \u0080 \u00E6",
			},
			expectStr: `["\x01 \t \n \x19 \" \\ \x7F ` + "\u0080 \u00E6\"]",
		},
	}

	for _, test := range tests {
		array := castBuf[idol.TextArray](test.buf)

		testutil.ExpectEq(t, uint32(len(test.expectValues)), array.Len())
		testutil.ExpectSliceEq(t, test.expectValues, array.Collect())
		testutil.ExpectSliceEq(t, test.expectValues, collectSeq2(array.Iter()))
		testutil.ExpectEq(t, test.expectStr, array.String())

		for ii, value := range test.expectValues {
			got, ok := array.Get(uint32(ii))
			if testutil.ExpectTrue(t, ok); ok {
				testutil.ExpectEq(t, value, got)
			}
		}

		{
			_, ok := array.Get(999)
			testutil.ExpectFalse(t, ok)
		}
	}
}

func TestMessageArray(t *testing.T) {
	t.Parallel()

	type RawMessage string
	tests := []struct {
		buf          []uint8
		expectValues []RawMessage
		expectStr    string
	}{
		{
			buf:          []uint8{},
			expectValues: []RawMessage{},
			expectStr:    `[]`,
		},
		{
			buf: []uint8{
				4, 0, 0, 0,
				8, 0, 0, 0,
				8, 0, 0, 0,
				8, 0, 0, 0,
				8, 0, 0, 0,
				0, 0, 0, 0,
				0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41,
				0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42,
				0x43, 0x43, 0x43, 0x43, 0x43, 0x43, 0x43, 0x43,
				0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44,
			},
			expectValues: []RawMessage{
				"AAAAAAAA",
				"BBBBBBBB",
				"CCCCCCCC",
				"DDDDDDDD",
			},
			expectStr: `[{AAAAAAAA}, {BBBBBBBB}, {CCCCCCCC}, {DDDDDDDD}]`,
		},
		{
			buf: []uint8{
				5, 0, 0, 0,
				8, 0, 0, 0,
				8, 0, 0, 0,
				8, 0, 0, 0,
				0, 0, 0, 0,
				8, 0, 0, 0,
				0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41,
				0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42,
				0x43, 0x43, 0x43, 0x43, 0x43, 0x43, 0x43, 0x43,
				0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44, 0x44,
			},
			expectValues: []RawMessage{
				"AAAAAAAA",
				"BBBBBBBB",
				"CCCCCCCC",
				"",
				"DDDDDDDD",
			},
			expectStr: `[{AAAAAAAA}, {BBBBBBBB}, {CCCCCCCC}, {}, {DDDDDDDD}]`,
		},
	}

	for _, test := range tests {
		array := castBuf[idol.MessageArray[RawMessage]](test.buf)

		testutil.ExpectEq(t, uint32(len(test.expectValues)), array.Len())
		testutil.ExpectSliceEq(t, test.expectValues, array.Collect())
		testutil.ExpectSliceEq(t, test.expectValues, collectSeq2(array.Iter()))
		testutil.ExpectEq(t, test.expectStr, array.String())

		for ii, value := range test.expectValues {
			got, ok := array.Get(uint32(ii))
			if testutil.ExpectTrue(t, ok); ok {
				testutil.ExpectEq(t, value, got)
			}
		}

		{
			_, ok := array.Get(999)
			testutil.ExpectFalse(t, ok)
		}
	}
}
